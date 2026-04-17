import 'package:flutter/foundation.dart';
import 'package:shared_preferences/shared_preferences.dart';
import '../api/api_client.dart';

class ServerConfig extends ChangeNotifier {
  static const _keyUrl = 'server_url';
  static const _keyCfId = 'cf_access_client_id';
  static const _keyCfSecret = 'cf_access_client_secret';

  String _url = '';
  String _cfAccessId = '';
  String _cfAccessSecret = '';
  bool _configured = false;

  String get url => _url;
  bool get isWeb => kIsWeb;
  bool get isConfigured => _configured;

  String get effectiveUrl => _url;
  String get wsBaseUrl => _url;

  String get cfAccessId => _cfAccessId;
  String get cfAccessSecret => _cfAccessSecret;
  bool get hasCfAccess => _cfAccessId.isNotEmpty && _cfAccessSecret.isNotEmpty;

  /// Extra headers injected into every HTTP request and WS handshake.
  Map<String, String> get cfAccessHeaders => hasCfAccess
      ? {
          'CF-Access-Client-Id': _cfAccessId,
          'CF-Access-Client-Secret': _cfAccessSecret,
        }
      : const {};

  Future<void> load() async {
    final prefs = await SharedPreferences.getInstance();
    final saved = prefs.getString(_keyUrl) ?? '';
    _cfAccessId = prefs.getString(_keyCfId) ?? '';
    _cfAccessSecret = prefs.getString(_keyCfSecret) ?? '';

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

  Future<void> setCfAccess(String clientId, String clientSecret) async {
    _cfAccessId = clientId.trim();
    _cfAccessSecret = clientSecret.trim();
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_keyCfId, _cfAccessId);
    await prefs.setString(_keyCfSecret, _cfAccessSecret);
    notifyListeners();
  }

  Future<void> clearCfAccess() async {
    _cfAccessId = '';
    _cfAccessSecret = '';
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_keyCfId);
    await prefs.remove(_keyCfSecret);
    notifyListeners();
  }
}
