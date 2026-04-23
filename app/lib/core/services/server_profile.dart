import 'dart:convert';
import 'dart:math' as math;

/// One saved OpenDray deployment the user can connect to. Profiles are
/// an address-book entry — URL + human alias + optional username, plus
/// a "remember password" hint. The actual password (if stored) lives
/// in the platform keychain via [SecureCredentialStore], never in this
/// record.
///
/// IDs are opaque and stable across renames so secure-storage keys and
/// AuthService token-map entries survive editing the alias or URL. The
/// generator produces a URL-safe alphanumeric string long enough that
/// collision inside a single user's profile list is not a real concern.
class ServerProfile {
  final String id;
  final String alias;
  final String url;
  final String username;
  final bool rememberPassword;
  final DateTime? lastUsedAt;

  const ServerProfile({
    required this.id,
    required this.alias,
    required this.url,
    this.username = '',
    this.rememberPassword = false,
    this.lastUsedAt,
  });

  ServerProfile copyWith({
    String? alias,
    String? url,
    String? username,
    bool? rememberPassword,
    DateTime? lastUsedAt,
  }) {
    return ServerProfile(
      id: id,
      alias: alias ?? this.alias,
      url: url ?? this.url,
      username: username ?? this.username,
      rememberPassword: rememberPassword ?? this.rememberPassword,
      lastUsedAt: lastUsedAt ?? this.lastUsedAt,
    );
  }

  Map<String, dynamic> toJson() => {
        'id': id,
        'alias': alias,
        'url': url,
        'username': username,
        'rememberPassword': rememberPassword,
        if (lastUsedAt != null) 'lastUsedAt': lastUsedAt!.toIso8601String(),
      };

  factory ServerProfile.fromJson(Map<String, dynamic> j) {
    final ts = j['lastUsedAt'];
    return ServerProfile(
      id: (j['id'] as String?) ?? '',
      alias: (j['alias'] as String?) ?? '',
      url: (j['url'] as String?) ?? '',
      username: (j['username'] as String?) ?? '',
      rememberPassword: (j['rememberPassword'] as bool?) ?? false,
      lastUsedAt: ts is String ? DateTime.tryParse(ts) : null,
    );
  }

  /// Normalizes a user-typed URL the same way everywhere: trims
  /// whitespace and strips any trailing slashes. Does not add a scheme
  /// — that's the user's responsibility; the Test button on the form
  /// surfaces scheme typos as connection errors.
  static String normalizeUrl(String raw) =>
      raw.trim().replaceAll(RegExp(r'/+$'), '');

  /// Produces a stable id. Not cryptographic — just needs to be unique
  /// within this user's profile list. 12 alphanumeric chars seeded from
  /// time + Random is plenty for the 1–20 profile range we expect.
  static String newId() {
    const chars =
        'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789';
    final r = math.Random();
    final now = DateTime.now().millisecondsSinceEpoch.toRadixString(36);
    final suffix = List.generate(6, (_) => chars[r.nextInt(chars.length)]).join();
    return '${now}_$suffix';
  }
}

/// On-disk shape of the profile registry: the list plus a pointer to
/// whichever profile is currently active. Stored as JSON under a
/// single SharedPreferences key so load/save is atomic.
class ServerProfileStoreBlob {
  final List<ServerProfile> profiles;
  final String activeId;

  const ServerProfileStoreBlob({
    required this.profiles,
    required this.activeId,
  });

  String toJsonString() => jsonEncode({
        'profiles': [for (final p in profiles) p.toJson()],
        'activeId': activeId,
      });

  static ServerProfileStoreBlob fromJsonString(String raw) {
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map) {
        return const ServerProfileStoreBlob(profiles: [], activeId: '');
      }
      final rawList = decoded['profiles'];
      final profiles = <ServerProfile>[];
      if (rawList is List) {
        for (final e in rawList) {
          if (e is Map) {
            final p = ServerProfile.fromJson(Map<String, dynamic>.from(e));
            if (p.id.isNotEmpty && p.url.isNotEmpty) profiles.add(p);
          }
        }
      }
      return ServerProfileStoreBlob(
        profiles: profiles,
        activeId: (decoded['activeId'] as String?) ?? '',
      );
    } catch (_) {
      return const ServerProfileStoreBlob(profiles: [], activeId: '');
    }
  }
}
