# Multi-stage build for opendray v2.
#
# Stage 1 builds the React SPA via pnpm.
# Stage 2 builds the Go binary, embedding the SPA via go:embed.
# Stage 3 is the runtime — distroless, non-root, single binary.
#
# Build:
#   docker build -t opendray:dev .
#
# Run:
#   docker run --rm -p 8770:8770 \
#     -e OPENDRAY_DATABASE_URL='postgres://...' \
#     -e OPENDRAY_ADMIN_PASSWORD='...' \
#     -v /etc/opendray/config.toml:/etc/opendray/config.toml:ro \
#     opendray:dev serve -config /etc/opendray/config.toml

# ── Stage 1: web bundle ───────────────────────────────────────────────
FROM node:22-alpine AS web

# pnpm via corepack (ships with Node 22); pinned to v10 to match
# the packageManager field in package.json and the version action
# installs in CI.
RUN corepack enable && corepack prepare pnpm@10 --activate

WORKDIR /src

# Cache deps independently of source changes. The monorepo lockfile
# and workspace manifest live at the repo root (see #45). Bringing
# every workspace's package.json in first lets pnpm install resolve
# the full dependency graph from the lock without pulling in source.
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY app/web/package.json ./app/web/
COPY app/shared/package.json ./app/shared/
COPY app/shared-ui/package.json ./app/shared-ui/
RUN pnpm install --frozen-lockfile

# Bring source after lock install so changes here don't bust the
# dependency layer. Build the web workspace — Vite writes
# ../../internal/web/dist (the gobuild stage copies it from there).
COPY app ./app
RUN pnpm --filter web build

# ── Stage 2: go binary ────────────────────────────────────────────────
FROM golang:1.25-alpine AS gobuild

WORKDIR /src

# Cache modules independently of source.
COPY go.mod go.sum ./
RUN go mod download

# Bring in the rest of the source plus the dist/ from stage 1.
COPY . .
COPY --from=web /src/internal/web/dist ./internal/web/dist

ARG VERSION=0.0.0-docker
ARG COMMIT=unknown
ARG DATE=unknown

RUN CGO_ENABLED=0 go build \
    -trimpath \
    -ldflags="-s -w \
      -X github.com/opendray/opendray-v2/internal/version.Version=${VERSION} \
      -X github.com/opendray/opendray-v2/internal/version.Commit=${COMMIT} \
      -X github.com/opendray/opendray-v2/internal/version.Date=${DATE}" \
    -o /out/opendray \
    ./cmd/opendray

# ── Stage 3: runtime ──────────────────────────────────────────────────
# Distroless static-debian12 ships /etc/passwd with `nonroot:65532` and
# nothing else — no shell, no package manager, minimal CVE surface.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=gobuild /out/opendray /usr/local/bin/opendray

USER nonroot:nonroot
EXPOSE 8770

# Default to `serve`; operator overrides via `docker run … <subcommand>`.
ENTRYPOINT ["/usr/local/bin/opendray"]
CMD ["serve"]
