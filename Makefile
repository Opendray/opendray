.PHONY: dev dev-backend dev-app build build-web build-apk run test vet clean \
        release-linux release-all release package deploy ssh logs status

# Linux target for the LXC deployment.
LINUX_GOOS   ?= linux
LINUX_GOARCH ?= amd64
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_SHA    := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS      := -s -w -X main.version=$(VERSION) -X main.buildSha=$(BUILD_SHA)

# Deployment target — operator must override via .env or shell env.
# Example: OPENDRAY_DEPLOY_HOST=root@10.0.0.42 make deploy
DEPLOY_HOST  ?= $(OPENDRAY_DEPLOY_HOST)
DEPLOY_KEY   ?= $(HOME)/.ssh/opendray_deploy_key
DEPLOY_PATH  ?= /opt/opendray/releases

include .env
export

dev:
	@trap 'kill 0' EXIT; \
		go run ./cmd/opendray & \
		cd app && flutter run -d chrome & \
		wait

dev-backend:
	go run ./cmd/opendray

dev-app:
	cd app && flutter run -d chrome

build-web:
	cd app && flutter build web --release

build: build-web
	go build -o bin/opendray ./cmd/opendray

build-apk:
	cd app && flutter build apk --release

run:
	./bin/opendray

test:
	go test -race ./...

vet:
	go vet ./...

clean:
	rm -rf bin/ app/build/ dist/

# ── Production deployment targets ──────────────────────────

# Cross-compile a stripped Linux/amd64 binary + embedded web dist.
release-linux: build-web
	@mkdir -p bin
	GOOS=$(LINUX_GOOS) GOARCH=$(LINUX_GOARCH) CGO_ENABLED=0 \
	  go build -ldflags='$(LDFLAGS)' -trimpath \
	  -o bin/opendray-$(LINUX_GOOS)-$(LINUX_GOARCH) ./cmd/opendray
	@ls -lh bin/opendray-$(LINUX_GOOS)-$(LINUX_GOARCH)

# Build binaries for darwin/linux × amd64/arm64 + SHA256 sums. Output
# lands in bin/ alongside a checksums file. Run `make release-all` before
# uploading artefacts to a GitHub release.
release-all: build-web
	@mkdir -p bin
	@for os in darwin linux; do for arch in amd64 arm64; do \
	  echo "→ building $$os/$$arch"; \
	  GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 \
	    go build -ldflags='$(LDFLAGS)' -trimpath \
	    -o bin/opendray-$$os-$$arch ./cmd/opendray; \
	done; done
	@cd bin && shasum -a 256 opendray-* > SHA256SUMS && cat SHA256SUMS

# Pack the release tarball: binary + plugins + deploy helpers.
release: release-linux
	@mkdir -p dist dist/.stage
	@cp bin/opendray-$(LINUX_GOOS)-$(LINUX_GOARCH) dist/.stage/opendray-$(LINUX_GOOS)-$(LINUX_GOARCH)
	@cp -r plugins dist/.stage/plugins
	@cp -r deploy dist/.stage/deploy
	@tar --exclude='._*' --exclude='.DS_Store' \
	    -czf dist/opendray-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz \
	    -C dist/.stage .
	@rm -rf dist/.stage
	@ls -lh dist/opendray-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz
	@echo "Built $(VERSION) ($(BUILD_SHA))"

# Push the tarball to the LXC and run upgrade.sh. Requires SSH key access.
deploy: release
	@echo "→ uploading dist/opendray-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz to $(DEPLOY_HOST)"
	scp -i $(DEPLOY_KEY) \
	    dist/opendray-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz \
	    $(DEPLOY_HOST):/tmp/opendray-release.tar.gz
	@echo "→ running upgrade on $(DEPLOY_HOST)"
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST) \
	    "VERSION=$(VERSION) /opt/opendray/current/deploy/upgrade.sh /tmp/opendray-release.tar.gz"

ssh:
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST)

logs:
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST) journalctl -u opendray -f

status:
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST) systemctl status opendray --no-pager
