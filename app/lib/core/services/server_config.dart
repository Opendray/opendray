import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../api/api_client.dart';

class ServerConfig extends ChangeNotifier {
  static const _keyUrl = 'server_url';

  // Legacy prefs keys kept only for one-time cleanup on load(). Remove
  // once enough versions have shipped that nobody's device still has
  // these stored. Safe to delete today if you don't care about leaving
  // orphan entries on a few devices.
  static const _legacyKeyCfId = 'cf_access_client_id';
  static const _legacyKeyCfSecret = 'cf_access_client_secret';

  String _url = '';
  bool _configured = false;

  String get url => _url;
  bool get isWeb => kIsWeb;
  bool get isConfigured => _configured;

  String get effectiveUrl => _url;
  String get wsBaseUrl => _url;

  Future<void> load() async {
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString(_keyUrl) ?? '';

    // One-time cleanup of the old Cloudflare Access keys. Cheap to keep.
    if (prefs.containsKey(_legacyKeyCfId)) {
      await prefs.remove(_legacyKeyCfId);
    }
    if (prefs.containsKey(_legacyKeyCfSecret)) {
      await prefs.remove(_legacyKeyCfSecret);
    }

    if (saved.isNotEmpty) {
      _url = saved;
      _configured = true;
      notifyListeners();
      return;
    }

    if (isWeb) {
      final origin = Uri.base.origin;
      try {
        final api = ApiClient(baseUrl: origin);
        await api.health();
        _url = origin;
        _configured = true;
        await prefs.setString(_keyUrl, _url);
      } catch (_) {
        _configured = false;
      }
    }
    notifyListeners();
  }

  Future<void> setUrl(String url) async {
    final clean = url.trim().replaceAll(RegExp(r'/+$'), '');
    _url = clean;
    _configured = _url.isNotEmpty;
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyUrl, _url);
    notifyListeners();
  }
}
