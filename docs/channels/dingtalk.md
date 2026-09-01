# DingTalk (钉钉)

**Mode:** custom group robot (outbound only, no public URL needed)
**Capabilities:** text · card (markdown / actionCard), no callback buttons

> v1 supports outbound notifications only. To receive messages or
> handle interactive callbacks you need the app-platform credential
> set (corp + agent + secret), which is not yet implemented. Use a
> bridge channel for now if you need bidirectional DingTalk.

## 1. Add a custom robot

1. Open the target DingTalk group → ⋯ → **Group settings** →
   **Group bots** → **Add Robot** → **Custom**.
2. **Robot name:** e.g. `OpenDray`.
3. **Security settings:** pick at least one. Easiest is **Sign**:
   - Generates a `SEC...` secret.
   - opendray will append `&timestamp=...&sign=...` to every webhook
     call automatically.
4. Click **Done**. DingTalk reveals the **Webhook URL** like:
   ```
   https://oapi.dingtalk.com/robot/send?access_token=abc123...
   ```

Other security options:
- **Custom keyword:** every message must contain a fixed substring
  (e.g. `OpenDray`) or DingTalk drops it. Less convenient.
- **IP allow-list:** restrict by source IP. Useful when opendray runs
  on a fixed egress.

## 2. Configure in opendray

Channels → New channel → kind `DingTalk`.

- **Webhook URL:** from step 1.
- **Sign secret:** the `SEC...` value (only when *Sign* is selected as
  the security option).

Save with **Enabled = on**.

## 3. Test

- Admin **Test** button sends a plain-text message to the group.
- Trigger session.idle, and opendray sends an actionCard with title +
  markdown body. Buttons whose value is a clickable URL (e.g.
  `https://opendray.example/sessions/<id>`) appear as buttons; `cmd:`
  buttons are dropped silently because group-robot cards cannot fire
  callbacks.

## Notes

- The DingTalk webhook expects payloads under **20 KB** and is rate
  limited to ~20 messages/min. For higher throughput / interactive
  callbacks, switch to the app-platform setup (planned for a future
  release).
- The `Sign` security mode produces a per-request signature; the
  `timestamp` must be within 1 hour of DingTalk's clock, and opendray
  uses `time.Now().UnixMilli()` which is fine as long as the host
  clock is roughly in sync.
