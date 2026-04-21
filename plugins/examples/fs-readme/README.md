# fs-readme — OpenDray M3 reference host plugin

A Node sidecar that reads `${HOME}/README.md` via the capability-gated `fs/readFile` host call and returns the first 400 bytes. Exercises the full M3 host-form path: supervisor spawn + LSP-framed JSON-RPC + sidecar → host routing + `fs.read` capability grant.

Requires Node 20+ on the dev host's `PATH`. CI machines without Node should skip the fs-readme E2E (`exec.LookPath("node")` check — documented in T26).

## Try it

1. `OPENDRAY_ALLOW_LOCAL_PLUGINS=1 opendray` (dev mode; `${home}/**` grant expands to your real `$HOME`).
2. `opendray plugin install ./plugins/examples/fs-readme --yes`
3. Consent screen shows: `fs.read` on `${home}/**`.
4. The supervisor spawns `node sidecar.js` once the plugin activates on startup; `ps aux | grep sidecar.js` confirms it.
5. The summarise command lands as an invokable endpoint in T26 (command dispatcher doesn't yet recognise `run.kind: "host"`). Until then the plugin is E2E-tested by `TestE2E_FSReadmeFullLifecycle` — install → invoke via the bridge → receive first 400 bytes → kill PID → supervisor respawn.

## What this plugin proves

- A host-form plugin loads, validates, and spawns a Node sidecar under the Supervisor (M3 T14).
- Sidecar → host JSON-RPC flows through the stdio mux and HostRPCHandler (M3 T15–T17) — the sidecar never touches the filesystem directly.
- The `fs.read` capability gate enforces `${home}/**` identically for host-form and webview plugins — revoking the grant returns `EPERM` on the next call.
- The supervisor's `restart: "on-failure"` policy respawns the sidecar after `SIGKILL` within its backoff window.
- The manifest uses `${home}/**` (not `${workspace}/**`) because M3's PathVarResolver does not yet populate `${workspace}` — using an unresolved variable fails closed with EINVAL.

## Files

```
fs-readme/
├── manifest.json   — v1 host-form manifest, runtime "node", fs.read on ${home}/**
├── sidecar.js      — JSON-RPC 2.0 over LSP framing, implements "summarise"
└── README.md       — this file
```

## M2 vs M3

`plugins/examples/kanban` is the M2 reference: **webview** form, in-browser surface, talks only to `storage` and `events` over the bridge WebSocket. fs-readme is the M3 reference: **host** form, a long-lived Node process under the Supervisor's care, talks to `fs` over stdio JSON-RPC while reusing the same capability gate that fronts the bridge. Kanban proves "declarative contributes + webview SDK"; fs-readme proves "sidecar lifecycle + sidecar→host RPC + capability parity across transports".
