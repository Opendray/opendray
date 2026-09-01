# WeCom / Enterprise WeChat (企业微信)

**Mode:** group robot webhook (outbound only, no public URL needed)
**Capabilities:** text · card (markdown), no callback buttons

> Only the group-robot webhook is supported in v1. The full
> app-platform path (corp_id + agent_id + secret + AES-encrypted
> callback) is not yet implemented. Use a bridge channel for
> bidirectional WeCom.

## 1. Add a group robot

1. Open the target group in WeCom → tap the group name to enter
   **Group settings**.
2. Scroll to **Group bots** → **Add Robot** → choose **Group robot
   (Webhook)**.
3. Give it a name (e.g. `OpenDray`) and confirm.
4. The group reveals a **Webhook URL** like:
   ```
   https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=abc-123-...
   ```
5. The `key=` query parameter is what opendray needs.

## 2. Configure in opendray

Channels → New channel → kind `WeCom`.

Either:

- **Webhook key:** the `key=` value alone (`abc-123-…`), **or**
- **Or full webhook URL:** paste the entire URL from step 1 if you
  prefer to skip extracting the key.

If both are set, the full URL wins.

Save with **Enabled = on**.

## 3. Test

- Admin **Test** button sends a plain text message to the group.
- Trigger session.idle, and a markdown message appears with the bold
  title + body. Buttons whose value is a clickable URL appear as
  inline links at the bottom; `cmd:` buttons are dropped (group
  robots cannot fire callbacks).

## Notes

- WeCom group robots are rate-limited to **20 messages/min per robot**.
  Bursty `session.*` notifications can hit this if you run many
  sessions concurrently. Consider tightening `notify_on` (e.g.
  `session.ended` only) or switch to the bridge.
- WeCom markdown only supports a small subset: `**bold**`, `>` quote,
  `# heading`, `[label](url)`, `<font color="info|warning|comment">`.
  Tables and code blocks render poorly.
- The webhook URL is a bearer credential. Anyone with it can post to
  the group. Do not commit it to source control.
