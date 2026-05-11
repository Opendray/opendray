// Mobile mirror of app/shared/src/lib/channelKinds.ts. Drives the
// kind-picker FAB sheet, the create/edit form, and the per-row
// metadata (which field to mask as a "token preview"). Bridge is
// intentionally absent — its create flow needs a token generator
// and capability multiselect that don't translate cleanly to the
// mobile form pattern; bridge channels stay web-only on creation.

enum ChannelFieldType { text, password, textarea }

class ChannelField {
  const ChannelField({
    required this.name,
    required this.label,
    required this.type,
    this.required = false,
    this.placeholder,
    this.hint,
    this.optional = false,
  });

  final String name;
  final String label;
  final ChannelFieldType type;
  final bool required;
  final String? placeholder;
  final String? hint;
  // optional = true means: when the field is left blank on create,
  // omit it from the submitted config so server defaults apply
  // (rather than persisting an empty string).
  final bool optional;
}

class ChannelKind {
  const ChannelKind({
    required this.kind,
    required this.label,
    required this.description,
    required this.fields,
    this.tokenFields = const [],
    this.webhookBased = false,
    this.afterCreateHint,
  });

  final String kind;
  final String label;
  final String description;
  final List<ChannelField> fields;
  // Fields whose presence in `config` is used as the masked "token"
  // preview on the row. First non-empty wins.
  final List<String> tokenFields;
  // When true the kind needs the operator to paste a publicly-
  // reachable webhook URL into the platform's admin console; the
  // post-create dialog surfaces the URL with a copy button.
  final bool webhookBased;
  // Optional callout shown after a successful create.
  final String? afterCreateHint;
}

const channelKinds = <ChannelKind>[
  ChannelKind(
    kind: 'telegram',
    label: 'Telegram',
    description:
        'Bot via @BotFather. opendray long-polls getUpdates and sends '
        'via REST. Buttons + reply_to_message work natively.',
    tokenFields: ['bot_token'],
    fields: [
      ChannelField(
        name: 'bot_token',
        label: 'Bot token',
        type: ChannelFieldType.password,
        required: true,
        placeholder: '123456:ABC-DEF...',
        hint: 'From @BotFather. Stored in channel config; admin-only API.',
      ),
      ChannelField(
        name: 'chat_id',
        label: 'Default chat ID',
        type: ChannelFieldType.text,
        placeholder: '42 (optional — used when no ReplyCtx)',
        optional: true,
      ),
    ],
  ),
  ChannelKind(
    kind: 'slack',
    label: 'Slack',
    description:
        'Socket Mode — no public webhook needed. Requires a bot OAuth '
        'token (xoxb-) and an app-level token (xapp-) with '
        'connections:write.',
    tokenFields: ['bot_token'],
    fields: [
      ChannelField(
        name: 'bot_token',
        label: 'Bot token (xoxb-…)',
        type: ChannelFieldType.password,
        required: true,
        placeholder: 'xoxb-...',
        hint:
            'OAuth & Permissions → Bot User OAuth Token. Needs chat:write.',
      ),
      ChannelField(
        name: 'app_token',
        label: 'App-level token (xapp-…)',
        type: ChannelFieldType.password,
        required: true,
        placeholder: 'xapp-...',
        hint:
            'Settings → Basic Information → App-Level Tokens. Scope: '
            'connections:write.',
      ),
      ChannelField(
        name: 'channel_id',
        label: 'Default channel ID',
        type: ChannelFieldType.text,
        placeholder: 'C0123ABC456 (optional)',
        optional: true,
      ),
    ],
  ),
  ChannelKind(
    kind: 'discord',
    label: 'Discord',
    description:
        'Bot via Discord Developer Portal with MESSAGE CONTENT INTENT '
        'enabled. Connects to Gateway WS — no public URL required.',
    tokenFields: ['bot_token'],
    fields: [
      ChannelField(
        name: 'bot_token',
        label: 'Bot token',
        type: ChannelFieldType.password,
        required: true,
        placeholder: 'Bot token from Discord Developer Portal',
        hint:
            'Application → Bot → Reset Token. Invite bot with '
            'send_messages + embed_links.',
      ),
      ChannelField(
        name: 'channel_id',
        label: 'Default channel ID',
        type: ChannelFieldType.text,
        placeholder: '123456789012345678 (optional)',
        optional: true,
      ),
    ],
  ),
  ChannelKind(
    kind: 'feishu',
    label: 'Feishu (飞书)',
    description:
        'App-level credentials. Uses event subscription webhook for '
        'inbound. Public webhook URL is generated below — paste it '
        'into the Feishu dev console.',
    tokenFields: ['app_secret'],
    webhookBased: true,
    afterCreateHint:
        'Open the webhook URL from the channel card and paste it into '
        'Feishu Open Platform → Event Subscriptions → Request URL.',
    fields: [
      ChannelField(
        name: 'app_id',
        label: 'App ID',
        type: ChannelFieldType.text,
        required: true,
        placeholder: 'cli_a1b2c3d4...',
      ),
      ChannelField(
        name: 'app_secret',
        label: 'App secret',
        type: ChannelFieldType.password,
        required: true,
        placeholder: 'Application credential secret',
      ),
      ChannelField(
        name: 'verification_token',
        label: 'Verification token',
        type: ChannelFieldType.password,
        optional: true,
        hint:
            'From Event Subscriptions → Verification Token. When set, '
            'opendray rejects webhooks with a different token.',
      ),
      ChannelField(
        name: 'chat_id',
        label: 'Default chat ID (oc_…)',
        type: ChannelFieldType.text,
        placeholder: 'oc_xxxxxxxxxx (optional)',
        optional: true,
      ),
    ],
  ),
  ChannelKind(
    kind: 'dingtalk',
    label: 'DingTalk (钉钉)',
    description:
        'Custom group robot. Outbound only. Group chat → Robots → Add '
        '→ Sign mode → copy webhook + secret.',
    tokenFields: ['secret', 'webhook_url'],
    fields: [
      ChannelField(
        name: 'webhook_url',
        label: 'Webhook URL',
        type: ChannelFieldType.password,
        required: true,
        placeholder: 'https://oapi.dingtalk.com/robot/send?access_token=...',
      ),
      ChannelField(
        name: 'secret',
        label: 'Sign secret',
        type: ChannelFieldType.password,
        optional: true,
        placeholder: 'SEC...',
        hint:
            'When the robot is set to "Sign" security mode, copy the '
            'secret here. opendray adds the timestamp + sign params '
            'automatically.',
      ),
    ],
  ),
  ChannelKind(
    kind: 'wecom',
    label: 'WeCom (企业微信)',
    description:
        'Group robot webhook. Outbound only (text + markdown). Group '
        'settings → Group robots → Add → copy webhook URL.',
    tokenFields: ['webhook_key', 'webhook_url'],
    fields: [
      ChannelField(
        name: 'webhook_key',
        label: 'Webhook key',
        type: ChannelFieldType.password,
        required: true,
        placeholder: 'The "key=" query value',
        hint:
            'Or paste the whole webhook URL into the field below — '
            'either is enough.',
      ),
      ChannelField(
        name: 'webhook_url',
        label: 'Or full webhook URL',
        type: ChannelFieldType.password,
        optional: true,
        placeholder: 'https://qyapi.weixin.qq.com/...',
      ),
    ],
  ),
  ChannelKind(
    kind: 'wechat',
    label: 'WeChat (个人微信)',
    description:
        'Push to personal WeChat via WxPusher. Outbound-only — push '
        'services do not relay user replies. Each recipient '
        'subscribes once via QR code.',
    tokenFields: ['app_token'],
    fields: [
      ChannelField(
        name: 'app_token',
        label: 'App token (AT_…)',
        type: ChannelFieldType.password,
        required: true,
        placeholder: 'AT_xxxxxxxxxxxxx',
        hint: 'WxPusher → 应用管理 → 创建应用 → 复制 App Token.',
      ),
      ChannelField(
        name: 'uids',
        label: 'Recipient UIDs (one per line)',
        type: ChannelFieldType.textarea,
        optional: true,
        placeholder: 'UID_xxxxxxxxxxxx\nUID_yyyyyyyyyyyy',
        hint: 'Either UIDs or topic IDs is required.',
      ),
      ChannelField(
        name: 'topic_ids',
        label: 'Topic IDs (one per line)',
        type: ChannelFieldType.textarea,
        optional: true,
        placeholder: '123\n456',
      ),
      ChannelField(
        name: 'url',
        label: 'Tap-through URL',
        type: ChannelFieldType.text,
        optional: true,
        placeholder: 'https://opendray.example/',
        hint: 'When set, tapping the WeChat notification opens this page.',
      ),
    ],
  ),
];

ChannelKind? findKind(String kind) {
  for (final k in channelKinds) {
    if (k.kind == kind) return k;
  }
  return null;
}

// Reads the masked token preview value from a config map for a kind.
// Returns the full string — caller decides whether to mask.
String? extractTokenPreview(ChannelKind kind, Map<String, dynamic> config) {
  for (final f in kind.tokenFields) {
    final v = config[f];
    if (v is String && v.isNotEmpty) return v;
  }
  return null;
}

String maskToken(String raw) {
  if (raw.length <= 10) return '••••';
  return '${raw.substring(0, 6)}…${raw.substring(raw.length - 4)}';
}
