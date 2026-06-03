// Mobile mirror of app/shared/src/lib/channelKinds.ts. Drives the
// kind-picker FAB sheet, the create/edit form, and the per-row
// metadata (which field to mask as a "token preview"). Bridge is
// intentionally absent — its create flow needs a token generator
// and capability multiselect that don't translate cleanly to the
// mobile form pattern; bridge channels stay web-only on creation.

import 'package:opendray/core/i18n/strings.g.dart';

enum ChannelFieldType { text, password, textarea, boolean }

class ChannelField {
  const ChannelField({
    required this.name,
    required this.label,
    required this.type,
    this.required = false,
    this.placeholder,
    this.hint,
    this.optional = false,
    this.defaultValue,
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
  // Initial value for `boolean` fields when the channel config has no
  // stored value. (`default` is a Dart reserved word, hence the name.)
  final bool? defaultValue;
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

// Runtime builder — i18n strings resolved against the active locale
// each time it's called. Cheap (a few dozen string lookups) and the
// callers already invoke it once per rebuild, so memoizing isn't
// worth the staleness risk.
List<ChannelKind> channelKindsList() => [
      ChannelKind(
        kind: 'telegram',
        label: 'Telegram',
        description: t.channels.kinds.telegram.description,
        tokenFields: const ['bot_token'],
        fields: [
          ChannelField(
            name: 'bot_token',
            label: t.channels.kinds.telegram.botTokenLabel,
            type: ChannelFieldType.password,
            required: true,
            placeholder: '123456:ABC-DEF...',
            hint: t.channels.kinds.telegram.botTokenHint,
          ),
          ChannelField(
            name: 'chat_id',
            label: t.channels.kinds.telegram.chatIdLabel,
            type: ChannelFieldType.text,
            placeholder: t.channels.kinds.telegram.chatIdPlaceholder,
            optional: true,
          ),
          ChannelField(
            name: 'owner_user_ids',
            label: t.channels.kinds.telegram.ownerUserIdsLabel,
            type: ChannelFieldType.text,
            optional: true,
            placeholder: t.channels.kinds.telegram.ownerUserIdsPlaceholder,
            hint: t.channels.kinds.telegram.ownerUserIdsHint,
          ),
          ChannelField(
            name: 'chat_enabled',
            label: t.channels.kinds.telegram.chatEnabledLabel,
            type: ChannelFieldType.boolean,
            defaultValue: true,
            hint: t.channels.kinds.telegram.chatEnabledHint,
          ),
          ChannelField(
            name: 'chat_typing',
            label: t.channels.kinds.telegram.chatTypingLabel,
            type: ChannelFieldType.boolean,
            defaultValue: true,
            hint: t.channels.kinds.telegram.chatTypingHint,
          ),
          ChannelField(
            name: 'reply_max_chars',
            label: t.channels.kinds.telegram.replyMaxCharsLabel,
            type: ChannelFieldType.text,
            optional: true,
            placeholder: t.channels.kinds.telegram.replyMaxCharsPlaceholder,
            hint: t.channels.kinds.telegram.replyMaxCharsHint,
          ),
        ],
      ),
      ChannelKind(
        kind: 'slack',
        label: 'Slack',
        description: t.channels.kinds.slack.description,
        tokenFields: const ['bot_token'],
        fields: [
          ChannelField(
            name: 'bot_token',
            label: t.channels.kinds.slack.botTokenLabel,
            type: ChannelFieldType.password,
            required: true,
            placeholder: 'xoxb-...',
            hint: t.channels.kinds.slack.botTokenHint,
          ),
          ChannelField(
            name: 'app_token',
            label: t.channels.kinds.slack.appTokenLabel,
            type: ChannelFieldType.password,
            required: true,
            placeholder: 'xapp-...',
            hint: t.channels.kinds.slack.appTokenHint,
          ),
          ChannelField(
            name: 'channel_id',
            label: t.channels.kinds.slack.channelIdLabel,
            type: ChannelFieldType.text,
            placeholder: t.channels.kinds.slack.channelIdPlaceholder,
            optional: true,
          ),
        ],
      ),
      ChannelKind(
        kind: 'discord',
        label: 'Discord',
        description: t.channels.kinds.discord.description,
        tokenFields: const ['bot_token'],
        fields: [
          ChannelField(
            name: 'bot_token',
            label: t.channels.kinds.discord.botTokenLabel,
            type: ChannelFieldType.password,
            required: true,
            placeholder: t.channels.kinds.discord.botTokenPlaceholder,
            hint: t.channels.kinds.discord.botTokenHint,
          ),
          ChannelField(
            name: 'channel_id',
            label: t.channels.kinds.discord.channelIdLabel,
            type: ChannelFieldType.text,
            placeholder: t.channels.kinds.discord.channelIdPlaceholder,
            optional: true,
          ),
        ],
      ),
      ChannelKind(
        kind: 'feishu',
        label: 'Feishu (飞书)',
        description: t.channels.kinds.feishu.description,
        tokenFields: const ['app_secret'],
        webhookBased: true,
        afterCreateHint: t.channels.kinds.feishu.afterCreateHint,
        fields: [
          ChannelField(
            name: 'app_id',
            label: t.channels.kinds.feishu.appIdLabel,
            type: ChannelFieldType.text,
            required: true,
            placeholder: 'cli_a1b2c3d4...',
          ),
          ChannelField(
            name: 'app_secret',
            label: t.channels.kinds.feishu.appSecretLabel,
            type: ChannelFieldType.password,
            required: true,
            placeholder: t.channels.kinds.feishu.appSecretPlaceholder,
          ),
          ChannelField(
            name: 'verification_token',
            label: t.channels.kinds.feishu.verificationTokenLabel,
            type: ChannelFieldType.password,
            optional: true,
            hint: t.channels.kinds.feishu.verificationTokenHint,
          ),
          ChannelField(
            name: 'chat_id',
            label: t.channels.kinds.feishu.chatIdLabel,
            type: ChannelFieldType.text,
            placeholder: t.channels.kinds.feishu.chatIdPlaceholder,
            optional: true,
          ),
        ],
      ),
      ChannelKind(
        kind: 'dingtalk',
        label: 'DingTalk (钉钉)',
        description: t.channels.kinds.dingtalk.description,
        tokenFields: const ['secret', 'webhook_url'],
        fields: [
          ChannelField(
            name: 'webhook_url',
            label: t.channels.kinds.dingtalk.webhookUrlLabel,
            type: ChannelFieldType.password,
            required: true,
            placeholder: 'https://oapi.dingtalk.com/robot/send?access_token=...',
          ),
          ChannelField(
            name: 'secret',
            label: t.channels.kinds.dingtalk.secretLabel,
            type: ChannelFieldType.password,
            optional: true,
            placeholder: 'SEC...',
            hint: t.channels.kinds.dingtalk.secretHint,
          ),
        ],
      ),
      ChannelKind(
        kind: 'wecom',
        label: 'WeCom (企业微信)',
        description: t.channels.kinds.wecom.description,
        tokenFields: const ['webhook_key', 'webhook_url'],
        fields: [
          ChannelField(
            name: 'webhook_key',
            label: t.channels.kinds.wecom.webhookKeyLabel,
            type: ChannelFieldType.password,
            required: true,
            placeholder: t.channels.kinds.wecom.webhookKeyPlaceholder,
            hint: t.channels.kinds.wecom.webhookKeyHint,
          ),
          ChannelField(
            name: 'webhook_url',
            label: t.channels.kinds.wecom.webhookUrlLabel,
            type: ChannelFieldType.password,
            optional: true,
            placeholder: 'https://qyapi.weixin.qq.com/...',
          ),
        ],
      ),
      // Personal-WeChat (WxPusher) entry removed from the create
      // flow — see app/shared/src/lib/channelKinds.ts for the
      // rationale. The server adapter remains for back-compat.
    ];

ChannelKind? findKind(String kind) {
  for (final k in channelKindsList()) {
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

/// Serialize a channel-config field's (already-trimmed) text value to the
/// JSON type the gateway expects, mirroring the web form
/// (app/web/src/pages/Channels.tsx → buildConfigFromValues).
///
/// Only `chat_id` needs coercion among the kinds the mobile form
/// supports: Telegram persists it as an `int64`, so submitting it as a
/// string makes the server reject the whole config —
/// "json: cannot unmarshal string into Go struct field config.chat_id
/// of type int64" — which is why a Telegram channel configured from the
/// phone failed to start. The numeric guard leaves every other value a
/// string, exactly as the web form submits it:
///   - Feishu's `oc_…` chat_id (non-numeric, stays a string),
///   - Slack/Discord `channel_id` snowflakes (18-digit IDs that must NOT
///     be narrowed to a number — precision loss; note this is a
///     different field name, so it is never matched here anyway),
///   - `owner_user_ids` / `reply_max_chars` / tokens (the server reads
///     reply_max_chars from a number or a numeric string, so a string is
///     fine).
///
/// The mobile form exposes no `topic_ids` / `uids` fields, so — unlike
/// the web builder — those need no handling here.
Object coerceChannelConfigValue(String name, String raw) {
  if (name == 'chat_id' && _bareIntPattern.hasMatch(raw)) {
    return int.tryParse(raw) ?? raw;
  }
  return raw;
}

// A bare integer with an optional leading '-' (Telegram group /
// supergroup ids look like -1001234567890). Mirrors the web guard
// /^-?\d+$/ so numeric-looking-but-not-numeric values stay strings.
final RegExp _bareIntPattern = RegExp(r'^-?\d+$');

/// Render a stored config value as the text its TextField should show
/// when editing — the inverse of [coerceChannelConfigValue]. A numeric
/// chat_id (persisted as a JSON number) must round-trip back to its
/// digits; without this the edit form seeds the field empty and the next
/// save silently drops the chat_id (losing outbound notifications).
/// Absent / non-scalar values seed empty.
String channelConfigFieldText(Object? value) {
  if (value is String) return value;
  if (value is num) return value.toString();
  return '';
}
