# OpenDray plugin marketplace

On-disk catalog backing `GET /api/marketplace/plugins` and the
`marketplace://` install source. The server loads `catalog.json` at
boot; package bundles live under `packages/<name>/<version>/`.

## Layout

```
plugins/marketplace/
  catalog.json          # list of Entry records (see plugin/marketplace/catalog.go)
  packages/
    <name>/
      <version>/
        manifest.json   # bundled plugin manifest (v1)
        …               # any other bundle assets (sidecar.js, ui/, …)
```

## Install flow

1. The Flutter Hub page fetches `GET /api/marketplace/plugins` and
   renders one card per `Entry`.
2. Tapping **Install** posts `{"src":"marketplace://<name>"}` to
   `POST /api/plugins/install`.
3. The gateway resolves the ref against this catalog, wraps the bundle
   dir in a `TrustedSource`, and hands it to the existing installer
   (stage → consent → confirm).

## Adding an entry

1. Create `packages/<name>/<version>/` and drop a fully-built v1
   plugin in it (same shape as `plugin/examples/*`).
2. Add an `Entry` object to `catalog.json`. `permissions` copied from
   the manifest lets the Hub card preview what consent the install
   will ask for.

The server re-reads `catalog.json` on boot — no hot reload yet.
