# Key rotation

opendray supports **immediate key rotation**: the operator clicks
Rotate in the UI (or hits `POST /integrations/{id}/rotate-key`),
opendray replaces the stored hash, and the old plaintext returns
`401 unauthorized` on the very next request.

A consumer that's running 24/7 has to deal with that gracefully.
This page covers the patterns.

## What rotate does on the server

```
1. Generate new plaintext token + bcrypt hash.
2. UPDATE integration SET api_key_hash = <new hash> WHERE id = ?
3. Clear the in-memory token cache.
4. Publish integration.key_rotated event.
5. Return new plaintext to caller (admin in the UI).
```

The OLD plaintext is now permanently lost — neither opendray nor
the operator can recover it. Your consumer is the only place a
working credential exists once it's saved.

## Three patterns for handling rotate

### Pattern 1 · Reload from secret store

The simplest. Your app loads the key on every startup from a
secret manager. When 401 happens during runtime, you signal "I
need a restart" and let your operator (or a process supervisor)
re-deploy with the new key in place.

```ts
async function callOpendray(path: string) {
  const res = await fetch(path, {
    headers: { Authorization: `Bearer ${apiKey}` },
  })
  if (res.status === 401) {
    process.exit(75) // EX_TEMPFAIL — let supervisor restart
  }
  return res
}
```

Use when: small CLI tools, cronjobs, anything started fresh per
run.

### Pattern 2 · Re-fetch from secret store on 401

Like Pattern 1 but no restart — the consumer assumes the secret
store has the latest key and refreshes the in-process cache.

```ts
async function callOpendray(path: string) {
  let res = await tryWith(apiKey, path)
  if (res.status === 401) {
    apiKey = await secretStore.getLatest('opendray-key')
    res = await tryWith(apiKey, path)
  }
  return res
}
```

The operator's rotation flow is now: rotate in opendray UI →
update secret in vault → consumer auto-recovers on next 401.
Two manual steps but no service restart required.

Use when: long-running services with access to a programmatic
secret store.

### Pattern 3 · Self-rotate (admin-credential needed)

Your app holds admin credentials and rotates **for itself** when
it sees 401, then writes the new key locally. This is what
[demo-client](#consuming-typescript-sdk) does:

```ts
// 401 detected → log in as admin → rotate-key → write state
const admin = new OpendrayClient({ base })
await admin.login(adminUser, adminPassword)
const { api_key } = await admin.rotateKey(integrationId)
saveState({ ...state, api_key, registered_at: new Date().toISOString() })
```

Use when: development / single-machine setups where storing admin
credentials in your app is acceptable. **Don't** ship this in
multi-user production — admin credentials in an integration's
threat surface defeats the point of integrations.

## Don't proactively rotate

Cron-based "rotate every 30 days for hygiene" loops are a common
anti-pattern with opendray. Reasons:

1. opendray has no way to push the new key to your consumer. Every
   rotation is a manual sync step.
2. Forgetting that step = downtime.
3. opendray's threat model is "API key under operator control",
   not "API key as time-bound session token". TTL doesn't add
   security here.

Rotate **on incident** — suspected leak, departing operator,
audit log shows weird traffic. Otherwise leave keys alone.

## Don't share keys across apps

Each consumer gets its own integration row + key. Reasons:

1. Scope sets can differ — a dashboard doesn't need
   `session:input`, a bot does.
2. Call-log attribution depends on per-integration keys — sharing
   means you can't tell which app made a particular call.
3. Rotation kills every consumer using a shared key.

If you have N apps, register N integrations.

## Race during rotate

Between the moment the operator clicks Rotate and the moment your
consumer picks up the new key, every in-flight request fails 401.
There's no overlap window. Plan for ≤ a couple of seconds of
errors during a rotate event and have your retry layer absorb it
without paging anyone.
