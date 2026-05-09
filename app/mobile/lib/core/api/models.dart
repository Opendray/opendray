// Wire-format models — manual JSON parsers (not freezed) for F1 to
// keep the bootstrap simple. F2 will migrate to freezed +
// json_serializable when the model count grows beyond ~5.
//
// Field names mirror the Go struct tags exactly; the gateway is
// the source of truth.

class HealthResponse {
  HealthResponse({
    required this.status,
    required this.version,
    required this.commit,
    required this.uptimeSeconds,
    required this.dbOk,
  });

  factory HealthResponse.fromJson(Map<String, dynamic> json) => HealthResponse(
        status: json['status'] as String? ?? '',
        version: json['version'] as String? ?? '',
        commit: json['commit'] as String? ?? '',
        uptimeSeconds: (json['uptime_s'] as num?)?.toInt() ?? 0,
        dbOk: json['db_ok'] as bool? ?? false,
      );

  final String status;
  final String version;
  final String commit;
  final int uptimeSeconds;
  final bool dbOk;
}

class MobileLoginResponse {
  MobileLoginResponse({
    required this.token,
    required this.username,
    required this.issuedAt,
    required this.expiresAt,
  });

  factory MobileLoginResponse.fromJson(Map<String, dynamic> json) =>
      MobileLoginResponse(
        token: json['token'] as String,
        username: json['username'] as String,
        issuedAt:
            DateTime.parse(json['issued_at'] as String).toUtc(),
        expiresAt:
            DateTime.parse(json['expires_at'] as String).toUtc(),
      );

  final String token;
  final String username;
  final DateTime issuedAt;
  final DateTime expiresAt;
}

enum SessionState {
  pending,
  running,
  idle,
  stopped,
  ended,
  unknown;

  static SessionState parse(String? raw) => switch (raw) {
        'pending' => SessionState.pending,
        'running' => SessionState.running,
        'idle' => SessionState.idle,
        'stopped' => SessionState.stopped,
        'ended' => SessionState.ended,
        _ => SessionState.unknown,
      };

  String get wire => switch (this) {
        SessionState.pending => 'pending',
        SessionState.running => 'running',
        SessionState.idle => 'idle',
        SessionState.stopped => 'stopped',
        SessionState.ended => 'ended',
        SessionState.unknown => 'unknown',
      };
}

class SessionSummary {
  SessionSummary({
    required this.id,
    required this.providerId,
    required this.cwd,
    required this.state,
    required this.startedAt,
    this.name,
    this.endedAt,
  });

  factory SessionSummary.fromJson(Map<String, dynamic> json) => SessionSummary(
        id: json['id'] as String,
        name: json['name'] as String?,
        providerId: json['provider_id'] as String? ?? '',
        cwd: json['cwd'] as String? ?? '',
        state: SessionState.parse(json['state'] as String?),
        startedAt: DateTime.tryParse(json['started_at'] as String? ?? '') ??
            DateTime.now().toUtc(),
        endedAt: (json['ended_at'] is String)
            ? DateTime.tryParse(json['ended_at'] as String)
            : null,
      );

  final String id;
  final String? name;
  final String providerId;
  final String cwd;
  final SessionState state;
  final DateTime startedAt;
  final DateTime? endedAt;

  String get displayName => name?.isNotEmpty ?? false ? name! : id;

  bool get isLive =>
      state == SessionState.running ||
      state == SessionState.idle ||
      state == SessionState.pending;

  bool get isFinished =>
      state == SessionState.stopped || state == SessionState.ended;
}

// Provider summary as rendered by the spawn-session form. The
// gateway returns the full Provider shape; we project to the
// 4 fields the picker actually shows.
class ProviderSummary {
  ProviderSummary({
    required this.id,
    required this.name,
    required this.manifestHash,
    required this.enabled,
  });

  factory ProviderSummary.fromGatewayJson(Map<String, dynamic> json) {
    // Server returns {manifest: Manifest, manifest_hash, config, enabled}.
    // Manifest fields use camelCase (`displayName`, not `name`); we
    // fall back through displayName → displayName_zh → id so the
    // picker label is never blank.
    final manifest = json['manifest'] as Map<String, dynamic>? ?? {};
    final id = manifest['id'] as String? ?? '';
    final display = manifest['displayName'] as String? ?? '';
    final displayZh = manifest['displayName_zh'] as String? ?? '';
    final name = display.isNotEmpty
        ? display
        : (displayZh.isNotEmpty ? displayZh : id);
    return ProviderSummary(
      id: id,
      name: name,
      manifestHash: json['manifest_hash'] as String? ?? '',
      enabled: json['enabled'] as bool? ?? false,
    );
  }

  final String id;
  final String name;
  final String manifestHash;
  final bool enabled;
}

// Multi-account picker option for the Claude provider. Mirrors the
// fields the spawn-session form actually renders; the full
// account record lives in /api/v1/claude-accounts.
class ClaudeAccountSummary {
  ClaudeAccountSummary({
    required this.id,
    required this.name,
    required this.displayName,
    required this.enabled,
    required this.tokenFilled,
  });

  factory ClaudeAccountSummary.fromJson(Map<String, dynamic> json) {
    final id = json['id'] as String? ?? '';
    final name = json['name'] as String? ?? '';
    final display = json['display_name'] as String? ?? '';
    return ClaudeAccountSummary(
      id: id,
      name: name,
      // Fall back through display_name → name → id so the picker
      // never shows a blank row.
      displayName:
          display.isNotEmpty ? display : (name.isNotEmpty ? name : id),
      enabled: json['enabled'] as bool? ?? false,
      tokenFilled: json['token_filled'] as bool? ?? false,
    );
  }

  final String id;
  final String name;
  final String displayName;
  final bool enabled;
  final bool tokenFilled;

  bool get isUsable => enabled && tokenFilled;
}

class CreateSessionRequest {
  const CreateSessionRequest({
    required this.providerId,
    required this.cwd,
    this.name,
    this.args,
    this.claudeAccountId,
  });

  final String providerId;
  final String cwd;
  final String? name;
  final List<String>? args;
  final String? claudeAccountId;

  Map<String, dynamic> toJson() => <String, dynamic>{
        'provider_id': providerId,
        'cwd': cwd,
        if (name != null && name!.isNotEmpty) 'name': name,
        if (args != null && args!.isNotEmpty) 'args': args,
        if (claudeAccountId != null && claudeAccountId!.isNotEmpty)
          'claude_account_id': claudeAccountId,
      };
}
