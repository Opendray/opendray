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

// One row from the gateway's audit log. Mirrors
// internal/audit/service.go Entry. Metadata is kept as a raw map
// because it varies per action — formatting/highlighting happens
// at the UI layer.
class AuditEntry {
  AuditEntry({
    required this.id,
    required this.timestamp,
    required this.actorKind,
    required this.action,
    this.actorId,
    this.subjectKind,
    this.subjectId,
    this.metadata,
  });

  factory AuditEntry.fromJson(Map<String, dynamic> json) => AuditEntry(
        id: (json['id'] as num?)?.toInt() ?? 0,
        timestamp:
            DateTime.tryParse(json['ts'] as String? ?? '')?.toUtc() ??
                DateTime.now().toUtc(),
        actorKind: json['actor_kind'] as String? ?? '',
        actorId: json['actor_id'] as String?,
        action: json['action'] as String? ?? '',
        subjectKind: json['subject_kind'] as String?,
        subjectId: json['subject_id'] as String?,
        metadata: json['metadata'] is Map
            ? Map<String, dynamic>.from(json['metadata'] as Map)
            : null,
      );

  final int id;
  final DateTime timestamp;
  final String actorKind;
  final String? actorId;
  final String action;
  final String? subjectKind;
  final String? subjectId;
  final Map<String, dynamic>? metadata;
}

class AuditPage {
  AuditPage({required this.entries, required this.nextCursor});

  factory AuditPage.fromJson(Map<String, dynamic> json) {
    final raw = json['entries'];
    final entries = raw is List
        ? raw
            .whereType<Map<String, dynamic>>()
            .map(AuditEntry.fromJson)
            .toList()
        : <AuditEntry>[];
    return AuditPage(
      entries: entries,
      nextCursor: json['next_cursor'] as String?,
    );
  }

  final List<AuditEntry> entries;
  // null = no more pages; opaque string the server expects back as
  // ?cursor= for the next page.
  final String? nextCursor;
}

// Memory scope band — mirrors internal/memory/store.go.
enum MemoryScope {
  session,
  project,
  global,
  unknown;

  String get wire => switch (this) {
        MemoryScope.session => 'session',
        MemoryScope.project => 'project',
        MemoryScope.global => 'global',
        MemoryScope.unknown => '',
      };

  static MemoryScope parse(String? raw) => switch (raw) {
        'session' => MemoryScope.session,
        'project' => MemoryScope.project,
        'global' => MemoryScope.global,
        _ => MemoryScope.unknown,
      };

  String get label => switch (this) {
        MemoryScope.session => 'Session',
        MemoryScope.project => 'Project',
        MemoryScope.global => 'Global',
        MemoryScope.unknown => 'Unknown',
      };
}

class Memory {
  Memory({
    required this.id,
    required this.scope,
    required this.scopeKey,
    required this.text,
    required this.embedder,
    required this.createdAt,
    required this.updatedAt,
    this.metadata,
    this.hitCount = 0,
    this.lastHitAt,
    this.sourceKind,
    this.sourceRef,
    this.summarizerSession,
    this.confidence,
  });

  factory Memory.fromJson(Map<String, dynamic> json) => Memory(
        id: json['id'] as String? ?? '',
        scope: MemoryScope.parse(json['scope'] as String?),
        scopeKey: json['scope_key'] as String? ?? '',
        text: json['text'] as String? ?? '',
        embedder: json['embedder'] as String? ?? '',
        createdAt:
            DateTime.tryParse(json['created_at'] as String? ?? '')?.toUtc() ??
                DateTime.now().toUtc(),
        updatedAt:
            DateTime.tryParse(json['updated_at'] as String? ?? '')?.toUtc() ??
                DateTime.now().toUtc(),
        metadata: json['metadata'] is Map
            ? Map<String, dynamic>.from(json['metadata'] as Map)
            : null,
        hitCount: (json['hit_count'] as num?)?.toInt() ?? 0,
        lastHitAt: (json['last_hit_at'] is String)
            ? DateTime.tryParse(json['last_hit_at'] as String)
            : null,
        sourceKind: json['source_kind'] as String?,
        sourceRef: json['source_ref'] as String?,
        summarizerSession: json['summarizer_session'] as String?,
        confidence: (json['confidence'] as num?)?.toDouble(),
      );

  final String id;
  final MemoryScope scope;
  final String scopeKey;
  final String text;
  final String embedder;
  final Map<String, dynamic>? metadata;
  final DateTime createdAt;
  final DateTime updatedAt;
  final int hitCount;
  final DateTime? lastHitAt;
  final String? sourceKind;
  final String? sourceRef;
  final String? summarizerSession;
  final double? confidence;
}

// SearchHit pairs a Memory with its cosine similarity score so the
// UI can render "0.83" badges on results.
class MemoryHit {
  MemoryHit({required this.memory, required this.similarity});

  factory MemoryHit.fromJson(Map<String, dynamic> json) => MemoryHit(
        memory: Memory.fromJson(
          (json['memory'] as Map<String, dynamic>?) ?? {},
        ),
        similarity: (json['similarity'] as num?)?.toDouble() ?? 0,
      );

  final Memory memory;
  final double similarity;
}

class MemoryStatus {
  MemoryStatus({
    required this.embedder,
    required this.dimensions,
    required this.enabled,
    required this.autoDetected,
  });

  factory MemoryStatus.fromJson(Map<String, dynamic> json) => MemoryStatus(
        embedder: json['embedder'] as String? ?? '',
        dimensions: (json['dimensions'] as num?)?.toInt() ?? 0,
        enabled: json['enabled'] as bool? ?? false,
        autoDetected: json['auto_detected'] as bool? ?? false,
      );

  final String embedder;
  final int dimensions;
  final bool enabled;
  final bool autoDetected;
}

// One past prompt entry pulled from the running session's
// project-scoped transcript history (currently Claude-only on the
// gateway side — see internal/session/claude_jsonl.go).
class ProjectInput {
  ProjectInput({
    required this.text,
    required this.timestamp,
    required this.sessionId,
  });

  factory ProjectInput.fromJson(Map<String, dynamic> json) => ProjectInput(
        text: json['text'] as String? ?? '',
        timestamp:
            DateTime.tryParse(json['ts'] as String? ?? '')?.toUtc() ??
                DateTime.now().toUtc(),
        sessionId: json['session_id'] as String? ?? '',
      );

  // The user's typed prompt (what they actually sent to the model).
  final String text;
  // When the prompt was sent (UTC).
  final DateTime timestamp;
  // Claude's own session id (the JSONL filename), distinct from
  // opendray's session row id.
  final String sessionId;
}

class HistoryResponse {
  HistoryResponse({required this.entries, required this.unsupportedProvider});

  factory HistoryResponse.fromJson(Map<String, dynamic> json) {
    final raw = json['entries'];
    final entries = raw is List
        ? raw
            .whereType<Map<String, dynamic>>()
            .map(ProjectInput.fromJson)
            .toList()
        : <ProjectInput>[];
    return HistoryResponse(
      entries: entries,
      unsupportedProvider: json['unsupported_provider'] as bool? ?? false,
    );
  }

  final List<ProjectInput> entries;
  // True when the session's provider isn't Claude — UI renders a
  // dedicated empty state instead of the "no entries yet" copy.
  final bool unsupportedProvider;
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
    this.subscriptionType,
    this.rateLimitTier,
    this.lastUsedAt,
    this.activeSessions,
    this.oauthEmail,
    this.previousEmail,
    this.identityDrift = false,
  });

  factory ClaudeAccountSummary.fromJson(Map<String, dynamic> json) {
    final id = json['id'] as String? ?? '';
    final name = json['name'] as String? ?? '';
    final display = json['display_name'] as String? ?? '';
    String? str(String key) {
      final v = json[key];
      return v is String && v.isNotEmpty ? v : null;
    }

    return ClaudeAccountSummary(
      id: id,
      name: name,
      // Fall back through display_name → name → id so the picker
      // never shows a blank row.
      displayName:
          display.isNotEmpty ? display : (name.isNotEmpty ? name : id),
      enabled: json['enabled'] as bool? ?? false,
      tokenFilled: json['token_filled'] as bool? ?? false,
      // Decorated metadata — optional; older gateways omit these.
      subscriptionType: str('subscription_type'),
      rateLimitTier: str('rate_limit_tier'),
      lastUsedAt: str('last_used_at'),
      activeSessions: (json['active_sessions'] as num?)?.toInt(),
      oauthEmail: str('oauth_email'),
      previousEmail: str('previous_email'),
      identityDrift: json['identity_drift'] as bool? ?? false,
    );
  }

  final String id;
  final String name;
  final String displayName;
  final bool enabled;
  final bool tokenFilled;
  // Decorated fields from the gateway's account JSON (see web
  // app/shared/src/lib/types.ts ClaudeAccount). All optional.
  final String? subscriptionType;
  final String? rateLimitTier;
  final String? lastUsedAt; // ISO timestamp
  final int? activeSessions;
  final String? oauthEmail; // current Anthropic account email
  final String? previousEmail; // prior email when drift detected
  final bool identityDrift; // oauth_email differs from baseline

  bool get isUsable => enabled && tokenFilled;
}

class CreateSessionRequest {
  const CreateSessionRequest({
    required this.providerId,
    required this.cwd,
    this.name,
    this.args,
    this.claudeAccountId,
    this.parentSessionId,
  });

  final String providerId;
  final String cwd;
  final String? name;
  final List<String>? args;
  final String? claudeAccountId;
  // Links a session spawned on behalf of another (e.g. a Task Runner
  // shell session) so the gateway can group it under its originator.
  final String? parentSessionId;

  Map<String, dynamic> toJson() => <String, dynamic>{
        'provider_id': providerId,
        'cwd': cwd,
        if (name != null && name!.isNotEmpty) 'name': name,
        if (args != null && args!.isNotEmpty) 'args': args,
        if (claudeAccountId != null && claudeAccountId!.isNotEmpty)
          'claude_account_id': claudeAccountId,
        if (parentSessionId != null && parentSessionId!.isNotEmpty)
          'parent_session_id': parentSessionId,
      };
}
