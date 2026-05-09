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
}
