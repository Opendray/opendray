import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/channels/channel_kinds.dart';

// Regression guard for the mobile Telegram channel that failed to start
// with: "factory: telegram: parse config: json: cannot unmarshal string
// into Go struct field config.chat_id of type int64".
//
// The mobile form previously submitted every non-boolean field as a raw
// string, but the gateway stores Telegram's chat_id as an int64. These
// tests pin the serialization rule (mirrors the web form's
// buildConfigFromValues in app/web/src/pages/Channels.tsx).
void main() {
  group('coerceChannelConfigValue', () {
    test('numeric Telegram chat_id becomes an int', () {
      final v = coerceChannelConfigValue('chat_id', '123456789');
      expect(v, isA<int>());
      expect(v, 123456789);
    });

    test('negative group/supergroup chat_id becomes an int', () {
      final v = coerceChannelConfigValue('chat_id', '-1001234567890');
      expect(v, isA<int>());
      expect(v, -1001234567890);
    });

    test('Feishu oc_ chat_id stays a string', () {
      final v = coerceChannelConfigValue('chat_id', 'oc_abc123');
      expect(v, isA<String>());
      expect(v, 'oc_abc123');
    });

    test('Slack/Discord channel_id snowflake is never narrowed', () {
      // 18-digit ID: coercing to a number would lose precision, and the
      // gateway wants channel_id as a string anyway.
      const snowflake = '123456789012345678';
      final v = coerceChannelConfigValue('channel_id', snowflake);
      expect(v, isA<String>());
      expect(v, snowflake);
    });

    test('owner_user_ids stays a string (comma-separated allowlist)', () {
      final v = coerceChannelConfigValue('owner_user_ids', '123,456');
      expect(v, isA<String>());
      expect(v, '123,456');
    });

    test('reply_max_chars stays a string (server parses either form)', () {
      final v = coerceChannelConfigValue('reply_max_chars', '3500');
      expect(v, isA<String>());
      expect(v, '3500');
    });

    test('non-numeric chat_id is left untouched (no false coercion)', () {
      expect(coerceChannelConfigValue('chat_id', '@mychannel'), '@mychannel');
      expect(coerceChannelConfigValue('chat_id', '4 2'), '4 2');
      expect(coerceChannelConfigValue('chat_id', ''), '');
    });
  });

  // The edit form seeds TextFields from the stored config. A numeric
  // chat_id must render back to its digits, else editing blanks it and
  // the next save drops it (losing outbound notifications).
  group('channelConfigFieldText', () {
    test('numeric chat_id round-trips to its digits', () {
      expect(channelConfigFieldText(123456789), '123456789');
    });

    test('negative numeric chat_id round-trips', () {
      expect(channelConfigFieldText(-1001234567890), '-1001234567890');
    });

    test('string value passes through unchanged', () {
      expect(channelConfigFieldText('oc_abc123'), 'oc_abc123');
    });

    test('absent / non-scalar value seeds empty', () {
      expect(channelConfigFieldText(null), '');
      expect(channelConfigFieldText(<String>['a', 'b']), '');
    });

    test('serialize -> seed round-trip is stable for a numeric chat_id', () {
      final stored = coerceChannelConfigValue('chat_id', '123456789');
      expect(stored, isA<int>());
      expect(channelConfigFieldText(stored), '123456789');
    });
  });
}
