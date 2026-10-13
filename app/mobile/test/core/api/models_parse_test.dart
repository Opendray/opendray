import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/providers_api.dart';

// Guards the JSON parsing for the account-switch + CLI-update-check
// surfaces (PR: mobile account switch + provider update awareness).
void main() {
  group('SessionSummary.claudeAccountId', () {
    test('reads a non-empty claude_account_id', () {
      final s = SessionSummary.fromJson({
        'id': 'ses_1',
        'provider_id': 'claude',
        'state': 'running',
        'started_at': '2026-01-01T00:00:00Z',
        'claude_account_id': 'acc_work',
      });
      expect(s.claudeAccountId, 'acc_work');
    });

    test('treats absent / empty as null (system default binding)', () {
      final absent = SessionSummary.fromJson({
        'id': 'ses_2',
        'provider_id': 'claude',
        'state': 'running',
        'started_at': '2026-01-01T00:00:00Z',
      });
      final empty = SessionSummary.fromJson({
        'id': 'ses_3',
        'provider_id': 'claude',
        'state': 'running',
        'started_at': '2026-01-01T00:00:00Z',
        'claude_account_id': '',
      });
      expect(absent.claudeAccountId, isNull);
      expect(empty.claudeAccountId, isNull);
    });
  });

  group('SessionSummary grok/codex/opencode account ids', () {
    test('reads each provider binding, empty means default', () {
      final s = SessionSummary.fromJson({
        'id': 'ses_4',
        'provider_id': 'codex',
        'state': 'running',
        'started_at': '2026-01-01T00:00:00Z',
        'grok_account_id': 'g1',
        'codex_account_id': 'c1',
        'opencode_account_id': '',
      });
      expect(s.grokAccountId, 'g1');
      expect(s.codexAccountId, 'c1');
      expect(s.opencodeAccountId, isNull);
    });
  });

  group('ProviderAccountSummary.fromJson', () {
    test('reads the shared account row shape', () {
      final a = ProviderAccountSummary.fromJson({
        'id': 'acc_1',
        'name': 'work',
        'display_name': 'Work',
        'enabled': true,
        'token_filled': true,
        'oauth_email': 'dev@example.com',
      });
      expect(a.id, 'acc_1');
      expect(a.displayName, 'Work');
      expect(a.enabled, isTrue);
      expect(a.tokenFilled, isTrue);
      expect(a.oauthEmail, 'dev@example.com');
    });

    test('falls back display_name → name → id; empty email is null', () {
      final named = ProviderAccountSummary.fromJson({
        'id': 'acc_2',
        'name': 'personal',
        'oauth_email': '',
      });
      final bare = ProviderAccountSummary.fromJson({'id': 'acc_3'});
      expect(named.displayName, 'personal');
      expect(named.oauthEmail, isNull);
      expect(named.enabled, isFalse);
      expect(named.tokenFilled, isFalse);
      expect(bare.displayName, 'acc_3');
    });
  });

  group('ProviderRuntime.fromJson', () {
    test('parses a probed update-available state', () {
      final rt = ProviderRuntime.fromJson({
        'installed': true,
        'installedVersion': '1.2.0',
        'latestVersion': '1.3.0',
        'updateAvailable': true,
        'activeSessions': 2,
      });
      expect(rt.installed, isTrue);
      expect(rt.installedVersion, '1.2.0');
      expect(rt.latestVersion, '1.3.0');
      expect(rt.updateAvailable, isTrue);
      expect(rt.activeSessions, 2);
    });

    test('defaults are safe when fields are missing', () {
      final rt = ProviderRuntime.fromJson({});
      expect(rt.installed, isFalse);
      expect(rt.updateAvailable, isFalse);
      expect(rt.installedVersion, isNull);
      expect(rt.latestVersion, isNull);
      expect(rt.activeSessions, 0);
    });
  });

  group('ProviderUpdateResult.fromJson', () {
    test('parses a successful change', () {
      final r = ProviderUpdateResult.fromJson({
        'changed': true,
        'available': true,
        'beforeVersion': '1.2.0',
        'afterVersion': '1.3.0',
        'output': 'updated 1 package',
      });
      expect(r.changed, isTrue);
      expect(r.available, isTrue);
      expect(r.afterVersion, '1.3.0');
    });

    test('parses an unavailable result with a reason', () {
      final r = ProviderUpdateResult.fromJson({
        'changed': false,
        'available': false,
        'reason': 'npm prefix not writable',
      });
      expect(r.available, isFalse);
      expect(r.reason, 'npm prefix not writable');
    });
  });
}
