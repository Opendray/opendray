# WeChat (个人微信)

**Mode:** push via WxPusher (https://wxpusher.zjiecode.com), no public URL needed
**Capabilities:** text · card (markdown)

> Personal WeChat has no official open API. opendray uses **WxPusher**,
> a free, account-safe push relay: each recipient subscribes to your
> *application* once via QR code in WeChat; opendray then pushes
> notifications by App Token. **Outbound only**: push services
> cannot relay user replies. For bidirectional personal WeChat use a
> bridge channel with a WeChaty / iPad-protocol adapter.

## 1. Create a WxPusher application

1. Visit https://wxpusher.zjiecode.com/admin → 登录 (WeChat scan).
2. **应用管理** → **创建应用** → fill name + description.
3. Copy the **APP_TOKEN** (starts with `AT_`).

## 2. Subscribe recipients

There are two ways to address recipients:

### A. By UID (specific people)

1. **用户管理** → 复制关注二维码.
2. Each recipient scans it in WeChat → follows your app's official
   account → returns a `UID_…` token visible in the user list.
3. Copy each UID and paste them (one per line) into opendray.

### B. By topic (broadcast channel)

1. **主题管理** → **新建主题** → returns a numeric *topicId*.
2. Share the topic's QR code; each subscriber scans → joins the
   topic.
3. Pasting the topicId in opendray sends to **everyone** subscribed.

You can mix both, and opendray pushes to the union of UIDs + topic
subscribers.

## 3. Configure in opendray

Channels → New channel → kind `WeChat (个人微信)`.

- **App token:** `AT_…` from step 1.
- **Recipient UIDs:** one per line (optional if topic IDs given).
- **Topic IDs:** one per line, numeric (optional if UIDs given).
- **Tap-through URL:** when set, tapping the WeChat notification
  opens this URL (e.g. opendray's admin UI).

At least one of UIDs / topic IDs is required.

## 4. Test

- Admin **Test** button pushes a one-line text. Check the recipient's
  WeChat for the notification.
- Trigger session.idle: a markdown message arrives with the **header
  title** + body. Buttons whose value is a clickable URL appear as
  inline links; `cmd:` buttons are dropped.

## Notes

- WxPusher's free tier limits message size to 40 KB and rate to ~5
  messages/sec. Plenty for opendray notifications.
- Notification banner preview is capped to ~20 chars. opendray uses
  the card header (or trims the body) for the preview.
- App Token is bearer-style: anyone with it can push to all your
  recipients. Rotate via WxPusher's app management page if leaked.
- WxPusher requires phone-side WeChat to be alive to deliver; offline
  recipients receive on next reconnect.
