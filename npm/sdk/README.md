# @opendray/sdk

TypeScript / JavaScript client for the [opendray](https://opendray.dev)
gateway: REST + WebSocket access to sessions, providers, channels,
integrations, and the event stream.

## Install

```sh
npm install @opendray/sdk
```

Works in Node 18+, Deno, Bun, and the browser. On Node <22 you'll need
to provide a `WebSocket` implementation (e.g. `ws`) to the WS helpers.

## Quick start

```ts
import { Client, subscribeEvents } from "@opendray/sdk";

const client = new Client({
  baseUrl: "https://opendray.example.com",
  token: process.env.OPENDRAY_TOKEN!, // admin token OR `odk_live_…` integration key
});

// REST
const sessions = await client.listSessions();
const session = await client.createSession({ provider: "claude", cwd: "/tmp" });
await client.sendInput(session.id, { data: "ls -la\n" });

// WebSocket events
for await (const frame of subscribeEvents(client, { topics: ["session.*"] })) {
  console.log(frame.topic, frame.data);
}
```

## API surface

| Group            | Methods |
| ---------------- | ------- |
| Integrations     | `registerIntegration`, `listIntegrations`, `getIntegration`, `updateIntegration`, `deleteIntegration`, `rotateIntegrationKey` (admin only) |
| Sessions         | `createSession`, `listSessions`, `getSession`, `deleteSession`, `sendInput`, `resizeSession`, `getSessionBuffer` |
| Providers        | `listProviders`, `setProviderConfig` |
| Channels         | `listChannels` |
| Event stream     | `subscribeEvents(client, { topics })` — async iterable, auto-reconnect |
| Session stream   | `streamSession(client, id, { onOutput, onEnded })` — bidirectional terminal WS |

Errors thrown by REST calls are `OpendrayError` instances with `status` and `body`.

See the [integration guide](https://github.com/Opendray/opendray/blob/main/docs/integration-guide.md)
for the full surface this SDK wraps.

## Versioning

This SDK ships in lockstep with the gateway: `@opendray/sdk@X.Y.Z`
matches gateway `vX.Y.Z`. If the wire types haven't changed between
two adjacent gateway releases, the SDK still gets a patch bump for
consistency.

## License

Apache-2.0
