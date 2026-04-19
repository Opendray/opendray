# Stage 1: Build Flutter web bundle
FROM ghcr.io/cirruslabs/flutter:beta AS flutter-build
WORKDIR /src
COPY app/ app/
RUN cd app && flutter pub get && flutter build web --release

# Stage 2: Build Go binary
FROM golang:1.24-bookworm AS go-build
WORKDIR /src
COPY . .
COPY --from=flutter-build /src/app/build/web/ app/build/web/
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -trimpath -o /opendray ./cmd/opendray

# Stage 3: Minimal runtime
FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates && \
    rm -rf /var/lib/apt/lists/*
COPY --from=go-build /opendray /opendray
EXPOSE 8640
USER nobody
ENTRYPOINT ["/opendray"]
