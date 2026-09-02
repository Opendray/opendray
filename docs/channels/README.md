# Channels: provisioning guides

opendray bundles native implementations for the most common messaging
platforms. Each guide below walks through obtaining credentials from
the platform and pasting them into the admin UI.

| Kind        | Inbound       | Outbound          | Buttons         | Public URL? |
|-------------|---------------|-------------------|-----------------|-------------|
| `telegram`  | long-poll     | REST              | inline keyboard | no          |
| `slack`     | Socket Mode   | Web API (blocks)  | block actions   | no          |
| `discord`   | Gateway WS    | REST (embeds)     | components      | no          |
| `feishu`    | webhook       | tenant-token API  | interactive card | **yes**    |
| `dingtalk`  | (none in v1)  | group-robot webhook | nav links only  | no         |
| `wecom`     | (none in v1)  | group-robot webhook | nav links only  | no         |
| `wechat`    | (none, push)  | WxPusher push     | nav links only  | no         |
| `bridge`    | WebSocket     | WebSocket         | adapter-defined | no          |

"Public URL" means the platform pushes events to opendray over HTTP and
opendray must be reachable from the internet (Cloudflare Tunnel / ngrok
/ public LB). The other kinds open an outbound connection from
opendray to the platform, so opendray can run behind a NAT.

Per-platform setup:

- [telegram.md](./telegram.md)
- [slack.md](./slack.md)
- [discord.md](./discord.md)
- [feishu.md](./feishu.md)
- [dingtalk.md](./dingtalk.md)
- [wecom.md](./wecom.md)
- [wechat.md](./wechat.md)

For custom platforms not listed here, see the bridge protocol:
[../bridge-protocol.md](../bridge-protocol.md).
