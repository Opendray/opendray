.PHONY: dev dev-backend dev-app build build-web build-apk run test vet clean \
        release-linux release package deploy ssh logs status

# Linux target for the LXC deployment.
LINUX_GOOS   ?= linux
LINUX_GOARCH ?= amd64
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
BUILD_SHA    := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
LDFLAGS      := -s -w -X main.version=$(VERSION) -X main.buildSha=$(BUILD_SHA)

# Deployment target — operator must override via .env or shell env.
# Example: NTC_DEPLOY_HOST=root@10.0.0.42 make deploy
DEPLOY_HOST  ?= $(NTC_DEPLOY_HOST)
DEPLOY_KEY   ?= $(HOME)/.ssh/ntc_deploy_key
DEPLOY_PATH  ?= /opt/ntc/releases

include .env
export

dev:
	@trap 'kill 0' EXIT; \
		go run ./cmd/ntc & \
		cd app && flutter run -d chrome & \
		wait

dev-backend:
	go run ./cmd/ntc

dev-app:
	cd app && flutter run -d chrome

build-web:
	cd app && flutter build web --release

build: build-web
	go build -o bin/ntc ./cmd/ntc

build-apk:
	cd app && flutter build apk --release

run:
	./bin/ntc

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
	  -o bin/ntc-$(LINUX_GOOS)-$(LINUX_GOARCH) ./cmd/ntc
	@ls -lh bin/ntc-$(LINUX_GOOS)-$(LINUX_GOARCH)

# Pack the release tarball: binary + plugins + deploy helpers.
release: release-linux
	@mkdir -p dist dist/.stage
	@cp bin/ntc-$(LINUX_GOOS)-$(LINUX_GOARCH) dist/.stage/ntc-$(LINUX_GOOS)-$(LINUX_GOARCH)
	@cp -r plugins dist/.stage/plugins
	@cp -r deploy dist/.stage/deploy
	@tar --exclude='._*' --exclude='.DS_Store' \
	    -czf dist/ntc-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz \
	    -C dist/.stage .
	@rm -rf dist/.stage
	@ls -lh dist/ntc-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz
	@echo "Built $(VERSION) ($(BUILD_SHA))"

# Push the tarball to the LXC and run upgrade.sh. Requires SSH key access.
deploy: release
	@echo "→ uploading dist/ntc-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz to $(DEPLOY_HOST)"
	scp -i $(DEPLOY_KEY) \
	    dist/ntc-$(VERSION)-$(LINUX_GOOS)-$(LINUX_GOARCH).tar.gz \
	    $(DEPLOY_HOST):/tmp/ntc-release.tar.gz
	@echo "→ running upgrade on $(DEPLOY_HOST)"
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST) \
	    "VERSION=$(VERSION) /opt/ntc/current/deploy/upgrade.sh /tmp/ntc-release.tar.gz"

ssh:
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST)

logs:
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST) journalctl -u ntc -f

status:
	ssh -i $(DEPLOY_KEY) $(DEPLOY_HOST) systemctl status ntc --no-pager
