import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'package:opendray/core/services/secure_credential_store.dart';
import 'package:opendray/core/services/server_config.dart';
import 'package:opendray/core/services/server_profile.dart';

/// In-memory stand-in for [SecureCredentialStore] so tests don't reach
/// the platform keychain (which isn't wired up in the flutter_test
/// harness). Claims to be "supported" so the tests can exercise the
/// remember-password branch just like on a real device.
class _FakeCredentialStore extends SecureCredentialStore {
  final Map<String, String> _data = {};

  @override
  bool get isSupported => true;

  @override
  Future<void> savePassword(String profileId, String password) async {
    _data[profileId] = password;
  }

  @override
  Future<String?> readPassword(String profileId) async => _data[profileId];

  @override
  Future<void> deletePassword(String profileId) async {
    _data.remove(profileId);
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  group('ServerConfig', () {
    late _FakeCredentialStore creds;

    setUp(() async {
      creds = _FakeCredentialStore();
    });

    test('load() migrates legacy server_url into a Default profile', () async {
      SharedPreferences.setMockInitialValues({
        'server_url': 'https://legacy.example.com/',
      });
      final cfg = ServerConfig(credentialStore: creds);
      await cfg.load();

      expect(cfg.profiles, hasLength(1));
      final p = cfg.profiles.single;
      expect(p.alias, 'Default');
      expect(p.url, 'https://legacy.example.com'); // trailing slash stripped
      expect(cfg.activeId, p.id);
      expect(cfg.isConfigured, true);
      expect(cfg.effectiveUrl, 'https://legacy.example.com');
    });

    test('load() sweeps legacy Cloudflare Access keys', () async {
      SharedPreferences.setMockInitialValues({
        'cf_access_client_id': 'id',
        'cf_access_client_secret': 'secret',
      });
      final cfg = ServerConfig(credentialStore: creds);
      await cfg.load();
      final prefs = await SharedPreferences.getInstance();
      expect(prefs.containsKey('cf_access_client_id'), false);
      expect(prefs.containsKey('cf_access_client_secret'), false);
    });

    test('addProfile stores normalized URL and activates it', () async {
      SharedPreferences.setMockInitialValues({});
      final cfg = ServerConfig(credentialStore: creds);
      await cfg.load();

      final p = await cfg.addProfile(
        alias: 'Home',
        url: '  http://10.0.0.1:8640///  ',
        username: 'admin',
        rememberPassword: true,
        password: 'hunter2',
      );
      expect(p.url, 'http://10.0.0.1:8640');
      expect(cfg.activeId, p.id);
      expect(cfg.profiles, hasLength(1));
      expect(await creds.readPassword(p.id), 'hunter2');
    });

    test('updateProfile toggling remember off wipes the stored password',
        () async {
      SharedPreferences.setMockInitialValues({});
      final cfg = ServerConfig(credentialStore: creds);
      await cfg.load();
      final p = await cfg.addProfile(
        alias: 'Home',
        url: 'http://10.0.0.1:8640',
        username: 'admin',
        rememberPassword: true,
        password: 'hunter2',
      );
      expect(await creds.readPassword(p.id), 'hunter2');

      await cfg.updateProfile(p.id, rememberPassword: false);
      expect(await creds.readPassword(p.id), isNull);
      expect(cfg.profiles.single.rememberPassword, false);
    });

    test('deleteProfile promotes another profile and scrubs credentials',
        () async {
      SharedPreferences.setMockInitialValues({});
      final cfg = ServerConfig(credentialStore: creds);
      await cfg.load();
      final a = await cfg.addProfile(
        alias: 'A',
        url: 'http://a.local',
        rememberPassword: true,
        password: 'pa',
      );
      final b = await cfg.addProfile(
        alias: 'B',
        url: 'http://b.local',
        rememberPassword: true,
        password: 'pb',
      );
      expect(cfg.activeId, b.id);

      await cfg.deleteProfile(b.id);
      expect(cfg.profiles, hasLength(1));
      expect(cfg.activeId, a.id);
      expect(await creds.readPassword(b.id), isNull);
      expect(await creds.readPassword(a.id), 'pa');
    });

    test('setUrl with an existing URL re-activates that profile', () async {
      SharedPreferences.setMockInitialValues({});
      final cfg = ServerConfig(credentialStore: creds);
      await cfg.load();
      final a = await cfg.addProfile(alias: 'A', url: 'http://a.local');
      await cfg.addProfile(alias: 'B', url: 'http://b.local');
      expect(cfg.profiles, hasLength(2));

      await cfg.setUrl('http://a.local/');
      expect(cfg.activeId, a.id);
      expect(cfg.profiles, hasLength(2)); // no duplicate created
    });

    test('setUrl with empty string clears active (triggers /connect)',
        () async {
      SharedPreferences.setMockInitialValues({});
      final cfg = ServerConfig(credentialStore: creds);
      await cfg.load();
      await cfg.addProfile(alias: 'A', url: 'http://a.local');
      expect(cfg.isConfigured, true);

      await cfg.setUrl('');
      expect(cfg.isConfigured, false);
      expect(cfg.effectiveUrl, '');
    });

    test('v1 store round-trips through SharedPreferences', () async {
      SharedPreferences.setMockInitialValues({});
      final first = ServerConfig(credentialStore: creds);
      await first.load();
      final p = await first.addProfile(
        alias: 'Home',
        url: 'http://10.0.0.1:8640',
        username: 'admin',
      );

      final second = ServerConfig(credentialStore: creds);
      await second.load();
      expect(second.profiles.map((e) => e.id), [p.id]);
      expect(second.activeId, p.id);
      expect(second.activeProfile?.alias, 'Home');
      expect(second.activeProfile?.username, 'admin');
    });
  });
}
