import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../api/api_client.dart';
import 'secure_credential_store.dart';
import 'server_profile.dart';

/// Address-book of OpenDray deployments the user can point the app at.
/// Stores zero-or-more [ServerProfile]s plus an "active" pointer; the
/// rest of the app reads the active profile's URL through the legacy
/// [url] / [effectiveUrl] getters so existing call sites don't change.
///
/// Legacy single-URL key `server_url` is migrated on first load into
/// a synthetic profile aliased "Default". The pre-v3 Cloudflare Access
/// keys get swept too.
class ServerConfig extends ChangeNotifier {
  static const _keyStore = 'opendray.servers.v1';
  static const _legacyKeyUrl = 'server_url';
  static const _legacyKeyCfId = 'cf_access_client_id';
  static const _legacyKeyCfSecret = 'cf_access_client_secret';

  final SecureCredentialStore _creds;

  List<ServerProfile> _profiles = const [];
  String _activeId = '';

  ServerConfig({SecureCredentialStore? credentialStore})
      : _creds = credentialStore ?? SecureCredentialStore();

  List<ServerProfile> get profiles => List.unmodifiable(_profiles);
  String get activeId => _activeId;

  ServerProfile? get activeProfile {
    if (_activeId.isEmpty) return null;
    for (final p in _profiles) {
      if (p.id == _activeId) return p;
    }
    return null;
  }

  /// True iff an active profile with a non-empty URL is selected — the
  /// router's gate condition. An empty profile list (fresh install) or
  /// a dangling activeId (corrupt prefs) both fall through to /connect.
  bool get isConfigured {
    final p = activeProfile;
    return p != null && p.url.isNotEmpty;
  }

  bool get isWeb => kIsWeb;

  /// Active profile URL — kept as `url` / `effectiveUrl` / `wsBaseUrl`
  /// for backward compat with every call site that reads the legacy
  /// single-URL API (ApiClient, AuthService.probe, session WS builders,
  /// Settings page, etc).
  String get url => activeProfile?.url ?? '';
  String get effectiveUrl => url;
  String get wsBaseUrl => url;

  /// Access to the keychain vault so UI forms (Add/Edit dialog, login
  /// page) can read/write per-profile passwords without instantiating
  /// their own store.
  SecureCredentialStore get credentialStore => _creds;

  // ── Lifecycle ───────────────────────────────────────────────

  Future<void> load() async {
    final prefs = await SharedPreferences.getInstance();

    // One-time cleanup of the old Cloudflare Access keys.
    if (prefs.containsKey(_legacyKeyCfId)) {
      await prefs.remove(_legacyKeyCfId);
    }
    if (prefs.containsKey(_legacyKeyCfSecret)) {
      await prefs.remove(_legacyKeyCfSecret);
    }

    final raw = prefs.getString(_keyStore);
    if (raw != null && raw.isNotEmpty) {
      final blob = ServerProfileStoreBlob.fromJsonString(raw);
      _profiles = blob.profiles;
      _activeId = _resolveActive(blob.activeId, _profiles);
      notifyListeners();
      return;
    }

    // No v1 store yet — migrate legacy single-URL blob if present.
    final legacy = prefs.getString(_legacyKeyUrl);
    if (legacy != null && legacy.isNotEmpty) {
      final p = ServerProfile(
        id: ServerProfile.newId(),
        alias: 'Default',
        url: ServerProfile.normalizeUrl(legacy),
        lastUsedAt: DateTime.now(),
      );
      _profiles = [p];
      _activeId = p.id;
      await _persist(prefs);
      await prefs.remove(_legacyKeyUrl);
      notifyListeners();
      return;
    }

    // First launch on web — auto-probe window.origin so the web client
    // opens straight into the app instead of forcing a manual URL.
    if (isWeb) {
      final origin = Uri.base.origin;
      try {
        final api = ApiClient(baseUrl: origin);
        await api.health();
        final p = ServerProfile(
          id: ServerProfile.newId(),
          alias: 'This Browser',
          url: origin,
          lastUsedAt: DateTime.now(),
        );
        _profiles = [p];
        _activeId = p.id;
        await _persist(prefs);
      } catch (_) {}
    }
    notifyListeners();
  }

  // ── Profile CRUD ────────────────────────────────────────────

  /// Persists a brand-new profile. If [makeActive] (default true) the
  /// new profile becomes the active one so the app immediately targets
  /// it — the common "Add Server → Save" UX.
  Future<ServerProfile> addProfile({
    required String alias,
    required String url,
    String username = '',
    bool rememberPassword = false,
    String? password,
    bool makeActive = true,
  }) async {
    final p = ServerProfile(
      id: ServerProfile.newId(),
      alias: alias.trim(),
      url: ServerProfile.normalizeUrl(url),
      username: username.trim(),
      rememberPassword: rememberPassword,
      lastUsedAt: makeActive ? DateTime.now() : null,
    );
    _profiles = [..._profiles, p];
    if (makeActive) _activeId = p.id;
    await _persistWithPrefs();
    if (rememberPassword && password != null && password.isNotEmpty) {
      await _creds.savePassword(p.id, password);
    }
    notifyListeners();
    return p;
  }

  /// Edits an existing profile in place. Passing a new [password] also
  /// rewrites the vault entry; passing null leaves it alone. When
  /// [rememberPassword] flips false, the stored password is deleted.
  Future<ServerProfile?> updateProfile(
    String id, {
    String? alias,
    String? url,
    String? username,
    bool? rememberPassword,
    String? password,
  }) async {
    final idx = _profiles.indexWhere((p) => p.id == id);
    if (idx < 0) return null;
    final current = _profiles[idx];
    final updated = current.copyWith(
      alias: alias?.trim(),
      url: url == null ? null : ServerProfile.normalizeUrl(url),
      username: username?.trim(),
      rememberPassword: rememberPassword,
    );
    _profiles = [
      for (var i = 0; i < _profiles.length; i++)
        if (i == idx) updated else _profiles[i],
    ];
    await _persistWithPrefs();

    // Credential store follow-up. Three cases:
    //   • rememberPassword now false → delete any stored password
    //   • rememberPassword true + new password provided → overwrite
    //   • rememberPassword true + no new password → keep existing blob
    if (updated.rememberPassword == false) {
      await _creds.deletePassword(id);
    } else if (password != null && password.isNotEmpty) {
      await _creds.savePassword(id, password);
    }

    notifyListeners();
    return updated;
  }

  Future<void> deleteProfile(String id) async {
    final remaining = _profiles.where((p) => p.id != id).toList();
    if (remaining.length == _profiles.length) return;
    _profiles = remaining;
    if (_activeId == id) {
      _activeId = remaining.isEmpty ? '' : remaining.first.id;
    }
    await _persistWithPrefs();
    await _creds.deletePassword(id);
    notifyListeners();
  }

  /// Switches the active profile. Bumps [ServerProfile.lastUsedAt] so
  /// the list surfaces recently-used servers on top.
  Future<void> setActive(String id) async {
    if (id == _activeId) return;
    final idx = _profiles.indexWhere((p) => p.id == id);
    if (idx < 0) return;
    final refreshed = _profiles[idx].copyWith(lastUsedAt: DateTime.now());
    _profiles = [
      for (var i = 0; i < _profiles.length; i++)
        if (i == idx) refreshed else _profiles[i],
    ];
    _activeId = id;
    await _persistWithPrefs();
    notifyListeners();
  }

  /// Legacy entry point — setup_page still calls `setUrl(url)` after
  /// the Test-Connection pass. Preserve that path:
  ///   • empty URL  → clear active (router bounces to /connect)
  ///   • non-empty  → reuse the first profile that already has this
  ///                  URL, otherwise synthesize a fresh profile
  Future<void> setUrl(String url) async {
    final clean = ServerProfile.normalizeUrl(url);
    if (clean.isEmpty) {
      _activeId = '';
      await _persistWithPrefs();
      notifyListeners();
      return;
    }
    final existing = _profiles.indexWhere((p) => p.url == clean);
    if (existing >= 0) {
      await setActive(_profiles[existing].id);
      return;
    }
    await addProfile(alias: _deriveAlias(clean), url: clean);
  }

  // ── Helpers ─────────────────────────────────────────────────

  static String _resolveActive(String desired, List<ServerProfile> list) {
    if (list.isEmpty) return '';
    for (final p in list) {
      if (p.id == desired) return desired;
    }
    return list.first.id;
  }

  /// Derives a sensible default alias from a URL so the legacy
  /// setup_page flow (no alias field) still produces a distinct,
  /// human-readable profile name. Strips scheme + path; keeps
  /// host:port. Falls back to "Server" when parsing fails.
  static String _deriveAlias(String url) {
    try {
      final u = Uri.parse(url);
      final host = u.host;
      if (host.isEmpty) return 'Server';
      return u.hasPort ? '$host:${u.port}' : host;
    } catch (_) {
      return 'Server';
    }
  }

  Future<void> _persistWithPrefs() async {
    final prefs = await SharedPreferences.getInstance();
    await _persist(prefs);
  }

  Future<void> _persist(SharedPreferences prefs) async {
    final blob = ServerProfileStoreBlob(
      profiles: _profiles,
      activeId: _activeId,
    );
    await prefs.setString(_keyStore, blob.toJsonString());
  }
}
