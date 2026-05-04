# Quickstart

End-to-end in 5 minutes with `curl`. By the end you'll have a
running Claude session that you spawned through the API.

## Prerequisites

- opendray running and reachable (default `http://127.0.0.1:8770`).
- Admin credentials (defaults: `admin` / `12345678` from
  `config.toml [admin]`).
- One installed agent provider — `shell` ships with opendray, no
  setup needed.

## Step 1 · Mint an admin token

```bash
ADMIN=$(curl -s -X POST http://127.0.0.1:8770/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"12345678"}' \
  | jq -r .token)
```

This token is short-lived (default 24h) and full-power. **Use it
once** to register your integration and then forget about it.

## Step 2 · Register a consumer-only integration

```bash
RESP=$(curl -s -X POST http://127.0.0.1:8770/api/v1/integrations \
  -H "Authorization: Bearer $ADMIN" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "my-app",
    "base_url": "",
    "route_prefix": "",
    "scopes": [
      "session:read",
      "session:create",
      "session:input",
      "event:subscribe:session.*"
    ],
    "version": "0.1.0"
  }')

echo "$RESP" | jq -r .api_key
```

Copy the printed `api_key` — opendray won't show it again.

```bash
KEY="odk_live_…the_key_you_just_copied"
```

## Step 3 · List sessions

```bash
curl -s http://127.0.0.1:8770/api/v1/sessions \
  -H "Authorization: Bearer $KEY" \
  | jq .sessions
```

200 → an array (possibly empty). 401 → key wrong / scopes wrong /
integration disabled.

## Step 4 · Spawn a shell session

```bash
SESSION=$(curl -s -X POST http://127.0.0.1:8770/api/v1/sessions \
  -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "demo",
    "provider_id": "shell",
    "cwd": "/tmp"
  }')

SESSION_ID=$(echo "$SESSION" | jq -r .id)
echo "spawned $SESSION_ID"
```

## Step 5 · Send input

```bash
curl -s -X POST \
  "http://127.0.0.1:8770/api/v1/sessions/$SESSION_ID/input" \
  -H "Authorization: Bearer $KEY" \
  -H 'Content-Type: application/json' \
  -d '{"data": "echo hello\n"}'
```

> **Important**: shell expects `\n`. **Claude expects `\r`** (raw
> mode). Wrong terminator = your prompt sits in the input box
> uncommitted.

## Step 6 · Read the output

```bash
curl -s "http://127.0.0.1:8770/api/v1/sessions/$SESSION_ID/buffer" \
  -H "Authorization: Bearer $KEY"
```

Returns the PTY ring buffer as raw bytes (ANSI-coloured shell
output). The last few lines should contain `hello`.

## Step 7 · Cleanup

```bash
curl -s -X DELETE \
  "http://127.0.0.1:8770/api/v1/sessions/$SESSION_ID" \
  -H "Authorization: Bearer $KEY" \
  -o /dev/null -w '%{http_code}\n'
# → 204
```

## What you just proved

- The same `api_key` authenticates **every** call from steps 3-7.
- You never used the admin token after step 2.
- Your scope set restricted what you could do — try a `provider:read`
  call and you'll get 403 because we didn't include that scope.

For the WebSocket event subscription side of the API, jump to
[Event subscriptions](#consuming-websocket-events).
