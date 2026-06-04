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
