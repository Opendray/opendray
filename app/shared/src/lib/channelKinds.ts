// Catalogue of all channel kinds opendray ships natively. Drives the
// admin UI's New Channel dialog (form fields, hints, defaults) and the
// per-card display (which credential field to mask and show as
// "token: 1234…abcd").
//
// `bridge` is intentionally absent — it has its own dedicated UI
// (token generator, capability multiselect, post-create adapter
// setup dialog) wired in Channels.tsx.

export type FieldType = 'text' | 'password' | 'textarea'

export interface KindField {
  name: string
  label: string
  type: FieldType
  required?: boolean
  placeholder?: string
  hint?: string
  /** When true and not provided by the user, the field is omitted from
   * the submitted config (so server defaults apply). */
  optional?: boolean
}

export interface KindDef {
  kind: string
  label: string
  emoji: string
  /**
   * Optional simple-icons / inline-brand key. When set, the admin
   * UI renders the official brand mark via <BrandIcon>; emoji is
   * used as fallback (and in compact list views where the SVG is
   * overkill).
   */
  iconKey?: string
  description: string
  /**
   * Fields the operator fills in. Order matches the form display.
   */
  fields: KindField[]
  /**
   * Fields whose presence in `config` should be rendered on the
   * channel card as a masked "token preview" line. First match wins.
   */
  tokenFields?: string[]
  /**
   * When true the channel needs a publicly-reachable webhook URL the
   * operator has to paste into the platform's admin console. The card
   * surfaces it via "Webhook URL" with a copy button.
   */
  webhookBased?: boolean
  /**
   * Optional after-create hint shown as a callout in the create
   * dialog. Useful for "remember to paste this URL into Feishu".
   */
  afterCreateHint?: string
}

export const KIND_DEFS: KindDef[] = [
  {
    kind: 'telegram',
    label: 'Telegram',
    emoji: '✈️',
    iconKey: 'telegram',
    description:
      'Bot via @BotFather. opendray long-polls getUpdates and sends via REST. Buttons + reply_to_message work natively.',
    tokenFields: ['bot_token'],
    fields: [
      {
        name: 'bot_token',
        label: 'Bot token',
        type: 'password',
        required: true,
        placeholder: '123456:ABC-DEF...',
        hint: 'From @BotFather. Stored in channel config; admin-only API.',
      },
      {
        name: 'chat_id',
        label: 'Default chat ID',
        type: 'text',
        placeholder: '42 (optional — used for outbound when no ReplyCtx)',
        optional: true,
      },
    ],
  },
  {
    kind: 'slack',
    label: 'Slack',
    emoji: '💬',
    iconKey: 'slack',
    description:
      'Socket Mode — no public webhook needed. Requires both a bot OAuth token (xoxb-) and an app-level token (xapp-) with connections:write.',
    tokenFields: ['bot_token'],
    fields: [
      {
        name: 'bot_token',
        label: 'Bot token (xoxb-…)',
        type: 'password',
        required: true,
        placeholder: 'xoxb-...',
        hint: 'OAuth & Permissions → Bot User OAuth Token. Needs chat:write.',
      },
      {
        name: 'app_token',
        label: 'App-level token (xapp-…)',
        type: 'password',
        required: true,
        placeholder: 'xapp-...',
        hint: 'Settings → Basic Information → App-Level Tokens. Scope: connections:write.',
      },
      {
        name: 'channel_id',
        label: 'Default channel ID',
        type: 'text',
        placeholder: 'C0123ABC456 (optional)',
        optional: true,
      },
    ],
  },
  {
    kind: 'discord',
    label: 'Discord',
    emoji: '🎮',
    iconKey: 'discord',
    description:
      'Bot via Discord Developer Portal with MESSAGE CONTENT INTENT enabled. Connects to Gateway WS — no public URL required.',
    tokenFields: ['bot_token'],
    fields: [
      {
        name: 'bot_token',
        label: 'Bot token',
        type: 'password',
        required: true,
        placeholder: 'Bot token from Discord Developer Portal',
        hint: 'Application → Bot → Reset Token. Invite bot with send_messages + embed_links.',
      },
      {
        name: 'channel_id',
        label: 'Default channel ID',
        type: 'text',
        placeholder: '123456789012345678 (right-click channel → Copy ID)',
        optional: true,
      },
    ],
  },
  {
    kind: 'feishu',
    label: 'Feishu (飞书)',
    emoji: '🐦',
    iconKey: 'feishu',
    description:
      'App-level credentials. Uses event subscription webhook for inbound. Public webhook URL is generated below — paste it into the Feishu dev console.',
    tokenFields: ['app_secret'],
    webhookBased: true,
    afterCreateHint:
      'Open the webhook URL on the channel card and paste it into Feishu Open Platform → Event Subscriptions → Request URL.',
    fields: [
      {
        name: 'app_id',
        label: 'App ID',
        type: 'text',
        required: true,
        placeholder: 'cli_a1b2c3d4...',
      },
      {
        name: 'app_secret',
        label: 'App secret',
        type: 'password',
        required: true,
        placeholder: 'Application credential secret',
      },
      {
        name: 'verification_token',
        label: 'Verification token (optional)',
        type: 'password',
        optional: true,
        hint: 'From Event Subscriptions → Verification Token. When set, opendray rejects webhooks with a different token.',
      },
      {
        name: 'chat_id',
        label: 'Default chat ID (oc_…)',
        type: 'text',
        placeholder: 'oc_xxxxxxxxxx (optional)',
        optional: true,
      },
    ],
  },
  {
    kind: 'dingtalk',
    label: 'DingTalk (钉钉)',
    emoji: '📞',
    iconKey: 'dingtalk',
    description:
      'Custom group robot. Outbound only (text + markdown + actionCard). Group chat → Robots → Add → Sign mode → copy webhook + secret.',
    tokenFields: ['secret', 'webhook_url'],
    fields: [
      {
        name: 'webhook_url',
        label: 'Webhook URL',
        type: 'password',
        required: true,
        placeholder: 'https://oapi.dingtalk.com/robot/send?access_token=...',
      },
      {
        name: 'secret',
        label: 'Sign secret (optional)',
        type: 'password',
        optional: true,
        placeholder: 'SEC...',
        hint: 'When the robot is set to "Sign" security mode, copy the secret here. opendray adds the timestamp + sign query params automatically.',
      },
    ],
  },
  {
    kind: 'wecom',
    label: 'WeCom (企业微信)',
    emoji: '🏢',
    iconKey: 'wecom',
    description:
      'Group robot webhook. Outbound only (text + markdown). Group settings → Group robots → Add → copy webhook URL.',
    tokenFields: ['webhook_key', 'webhook_url'],
    fields: [
      {
        name: 'webhook_key',
        label: 'Webhook key',
        type: 'password',
        required: true,
        placeholder: 'The "key=" query value from the robot webhook URL',
        hint: 'Or paste the whole webhook URL into the field below — either is enough.',
      },
      {
        name: 'webhook_url',
        label: 'Or full webhook URL',
        type: 'password',
        optional: true,
        placeholder: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...',
      },
    ],
  },
  {
    kind: 'wechat',
    label: 'WeChat (个人微信)',
    emoji: '💚',
    iconKey: 'wechat',
    description:
      'Push to personal WeChat via WxPusher (https://wxpusher.zjiecode.com). Outbound-only — push services do not relay user replies. Each recipient subscribes once via QR code.',
    tokenFields: ['app_token'],
    fields: [
      {
        name: 'app_token',
        label: 'App token (AT_…)',
        type: 'password',
        required: true,
        placeholder: 'AT_xxxxxxxxxxxxx',
        hint: 'WxPusher → 应用管理 → 创建应用 → 复制 App Token.',
      },
      {
        name: 'uids',
        label: 'Recipient UIDs (one per line)',
        type: 'textarea',
        optional: true,
        placeholder: 'UID_xxxxxxxxxxxx\nUID_yyyyyyyyyyyy',
        hint: 'Each recipient scans your app\'s QR code in WeChat to obtain a UID. Either UIDs or topic IDs is required.',
      },
      {
        name: 'topic_ids',
        label: 'Topic IDs (one per line)',
        type: 'textarea',
        optional: true,
        placeholder: '123\n456',
        hint: 'WxPusher → 主题管理 → create a topic → anyone subscribed to it receives every push.',
      },
      {
        name: 'url',
        label: 'Tap-through URL (optional)',
        type: 'text',
        optional: true,
        placeholder: 'https://opendray.example/',
        hint: 'When set, tapping the WeChat notification opens this page.',
      },
    ],
  },
]

export function getKindDef(kind: string): KindDef | undefined {
  return KIND_DEFS.find((k) => k.kind === kind)
}

/**
 * Build the per-channel public webhook URL the platform's admin
 * console needs to call. Used for feishu / dingtalk / wecom (kinds
 * that opt-in via `webhookBased: true`).
 */
export function buildWebhookURL(channelID: string): string {
  const origin =
    typeof window !== 'undefined' ? window.location.origin : 'http://localhost:5173'
  return origin + '/api/v1/channels/' + channelID + '/webhook'
}
