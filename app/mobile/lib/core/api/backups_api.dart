import 'dart:io';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/backups — full feature parity with the web admin:
// schedules + targets CRUD, restore from bundle, live inventory,
// Plan-C exports + imports. The file-upload paths (restore +
// imports) accept a phone-side File via file_picker; the web client
// uses <input type=file>, which dart:html doesn't expose on mobile.

class BackupRow {
  BackupRow({
    required this.id,
    required this.targetId,
    required this.status,
    required this.triggeredBy,
    required this.kind,
    required this.startedAt,
    required this.bytes,
    required this.encrypted,
    this.scheduleId,
    this.finishedAt,
    this.targetPath,
    this.error,
    this.verifiedAt,
    this.verifyError,
  });

  factory BackupRow.fromJson(Map<String, dynamic> json) => BackupRow(
    id: json['id'] as String? ?? '',
    scheduleId: json['schedule_id'] as String?,
    targetId: json['target_id'] as String? ?? '',
    status: json['status'] as String? ?? '',
    triggeredBy: json['triggered_by'] as String? ?? '',
    kind: json['kind'] as String? ?? 'db_only',
    startedAt:
        DateTime.tryParse(json['started_at'] as String? ?? '')?.toUtc() ??
        DateTime.now().toUtc(),
    finishedAt: DateTime.tryParse(
      json['finished_at'] as String? ?? '',
    )?.toUtc(),
    bytes: (json['bytes'] as num?)?.toInt() ?? 0,
    encrypted: json['encrypted'] as bool? ?? false,
    targetPath: json['target_path'] as String?,
    error: json['error'] as String?,
    verifiedAt: DateTime.tryParse(
      json['verified_at'] as String? ?? '',
    )?.toUtc(),
    verifyError: json['verify_error'] as String?,
  );

  final String id;
  final String? scheduleId;
  final String targetId;
  // pending | running | succeeded | failed | deleted
  final String status;
  // scheduler | manual | api | pre_migrate | pre_restore
  final String triggeredBy;
  // db_only | full_instance
  final String kind;
  final DateTime startedAt;
  final DateTime? finishedAt;
  final int bytes;
  final bool encrypted;
  final String? targetPath;
  final String? error;
  // Post-backup verification: verifiedAt set + verifyError null = the
  // blob decrypted and pg_restore --list passed; verifyError set = it
  // failed; both null = not verified yet.
  final DateTime? verifiedAt;
  final String? verifyError;
}

class BackupSchedule {
  BackupSchedule({
    required this.id,
    required this.targetId,
    required this.intervalSec,
    required this.retention,
    required this.enabled,
    required this.nextRunAt,
    required this.createdAt,
    this.lastRunAt,
  });

  factory BackupSchedule.fromJson(Map<String, dynamic> json) => BackupSchedule(
    id: json['id'] as String? ?? '',
    targetId: json['target_id'] as String? ?? '',
    intervalSec: (json['interval_sec'] as num?)?.toInt() ?? 0,
    retention: (json['retention'] as num?)?.toInt() ?? 0,
    enabled: json['enabled'] as bool? ?? false,
    lastRunAt: DateTime.tryParse(json['last_run_at'] as String? ?? '')?.toUtc(),
    nextRunAt:
        DateTime.tryParse(json['next_run_at'] as String? ?? '')?.toUtc() ??
        DateTime.now().toUtc(),
    createdAt:
        DateTime.tryParse(json['created_at'] as String? ?? '')?.toUtc() ??
        DateTime.fromMillisecondsSinceEpoch(0),
  );

  final String id;
  final String targetId;
  final int intervalSec;
  // Number of backups to retain — older runs auto-pruned.
  final int retention;
  final bool enabled;
  final DateTime? lastRunAt;
  final DateTime nextRunAt;
  final DateTime createdAt;
}

// Full feature state from /api/v1/backup-status. Since PR #49 the
// endpoint always returns 200 — the boolean fields below tell the
// client where on the off/on spectrum the server is, so the UI
// can show a Setup wizard, a Restart prompt, or the live Backup
// dashboard without ever distinguishing 404 from other errors.
class BackupStatusReport {
  BackupStatusReport({
    required this.enabled,
    required this.configured,
    required this.configuredVia,
    required this.canDisableViaUi,
    required this.requiresRestart,
    required this.keyFilePath,
    required this.ok,
    required this.keyFingerprint,
    required this.pgDumpVersion,
    required this.pgRestoreVersion,
    this.pgDumpError,
  });

  factory BackupStatusReport.fromJson(Map<String, dynamic> json) =>
      BackupStatusReport(
        enabled: json['enabled'] as bool? ?? false,
        configured: json['configured'] as bool? ?? false,
        configuredVia: json['configured_via'] as String? ?? '',
        canDisableViaUi: json['can_disable_via_ui'] as bool? ?? false,
        requiresRestart: json['requires_restart'] as bool? ?? false,
        keyFilePath: json['key_file_path'] as String? ?? '',
        ok: json['ok'] as bool? ?? false,
        keyFingerprint: json['key_fingerprint'] as String? ?? '',
        pgDumpVersion: json['pg_dump_version'] as String? ?? '',
        pgRestoreVersion: json['pg_restore_version'] as String? ?? '',
        pgDumpError: json['pg_dump_error'] as String?,
      );

  // True when backup is actively running in the gateway process.
  final bool enabled;
  // True when a passphrase is available from any source (env or
  // file) — orthogonal to `enabled` during the post-setup pre-
  // restart window.
  final bool configured;
  // "env" | "file" | "" — empty means no passphrase configured yet.
  final String configuredVia;
  // False when configuredVia == "env" (UI can't unset env vars
  // out from under the running process).
  final bool canDisableViaUi;
  // True when configured but !enabled — i.e. setup just wrote a
  // key file and the operator needs to restart opendray.
  final bool requiresRestart;
  // Canonical default location for the key file. Always populated
  // so the UI can show "your key will be written to <path>" even
  // before the first setup call.
  final String keyFilePath;

  // The next four fields are populated only when enabled=true.
  final bool ok;
  final String keyFingerprint;
  final String pgDumpVersion;
  final String pgRestoreVersion;
  final String? pgDumpError;
}

// Result of /api/v1/backup-setup. When `passphrase` is non-null it
// was server-generated (mode=generate) and MUST be saved by the
// operator before continuing — there's no recovery path if they
// lose it.
class BackupSetupResult {
  BackupSetupResult({
    required this.keyFilePath,
    required this.requiresRestart,
    this.passphrase,
  });

  factory BackupSetupResult.fromJson(Map<String, dynamic> json) =>
      BackupSetupResult(
        keyFilePath: json['key_file_path'] as String? ?? '',
        requiresRestart: json['requires_restart'] as bool? ?? false,
        passphrase: json['passphrase'] as String?,
      );

  final String keyFilePath;
  final bool requiresRestart;
  // Only set when mode=generate. Null on paste mode (operator
  // already knows it).
  final String? passphrase;
}

class BackupTarget {
  BackupTarget({
    required this.id,
    required this.kind,
    required this.config,
    required this.enabled,
    required this.createdAt,
    required this.updatedAt,
  });

  factory BackupTarget.fromJson(Map<String, dynamic> json) {
    final cfg = json['config'];
    return BackupTarget(
      id: json['id'] as String? ?? '',
      kind: json['kind'] as String? ?? '',
      config: cfg is Map ? Map<String, dynamic>.from(cfg) : <String, dynamic>{},
      enabled: json['enabled'] as bool? ?? false,
      createdAt:
          DateTime.tryParse(json['created_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
      updatedAt:
          DateTime.tryParse(json['updated_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
    );
  }

  final String id;
  // local | smb | s3 | webdav | sftp | rclone
  final String kind;
  // Sensitive fields (passwords, keys) are returned redacted by the
  // server; this map is fine to display in a "view raw config" modal.
  final Map<String, dynamic> config;
  final bool enabled;
  final DateTime createdAt;
  final DateTime updatedAt;
}

// ── restore ──────────────────────────────────────────────────────

class RestoreManifest {
  RestoreManifest({
    required this.version,
    required this.backupId,
    required this.createdAt,
    required this.encryptionAlgo,
    required this.encryptionFingerprint,
    this.opendrayVersion,
    this.pgVersion,
  });

  factory RestoreManifest.fromJson(Map<String, dynamic> json) {
    final enc = json['encryption'];
    final encMap = enc is Map
        ? Map<String, dynamic>.from(enc)
        : <String, dynamic>{};
    return RestoreManifest(
      version: json['version'] as String? ?? '',
      backupId: json['backup_id'] as String? ?? '',
      createdAt:
          DateTime.tryParse(json['created_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
      opendrayVersion: json['opendray_version'] as String?,
      pgVersion: json['pg_version'] as String?,
      encryptionAlgo: encMap['algo'] as String? ?? '',
      encryptionFingerprint: encMap['fingerprint'] as String? ?? '',
    );
  }

  final String version;
  final String backupId;
  final DateTime createdAt;
  final String? opendrayVersion;
  final String? pgVersion;
  final String encryptionAlgo;
  final String encryptionFingerprint;
}

// Result body returned by POST /backups/restore. On success the
// fingerprint matched, pg_restore ran cleanly, and `bytesRead`
// reflects how much of the bundle was processed. On partial
// failure (mid-restore pg_restore exit) the server returns 500
// with a nested `result` carrying this same shape; the API
// client surfaces that as ApiException and the caller is on its
// own to inspect the body.
class RestoreResult {
  RestoreResult({
    required this.manifest,
    required this.bytesRead,
    required this.targetDsnUsed,
    required this.fingerprintOk,
    required this.pgRestoreOutput,
    required this.startedAt,
    required this.finishedAt,
  });

  factory RestoreResult.fromJson(Map<String, dynamic> json) {
    final manifest = json['manifest'];
    final manifestMap = manifest is Map
        ? Map<String, dynamic>.from(manifest)
        : <String, dynamic>{};
    return RestoreResult(
      manifest: RestoreManifest.fromJson(manifestMap),
      bytesRead: (json['bytes_read'] as num?)?.toInt() ?? 0,
      targetDsnUsed: json['target_dsn_used'] as String? ?? '',
      fingerprintOk: json['fingerprint_ok'] as bool? ?? false,
      pgRestoreOutput: json['pg_restore_output'] as String? ?? '',
      startedAt:
          DateTime.tryParse(json['started_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
      finishedAt:
          DateTime.tryParse(json['finished_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
    );
  }

  final RestoreManifest manifest;
  final int bytesRead;
  // Empty string when target_dsn was omitted (restored into
  // opendray's own DB).
  final String targetDsnUsed;
  final bool fingerprintOk;
  final String pgRestoreOutput;
  final DateTime startedAt;
  final DateTime finishedAt;
}

// ── inventory ────────────────────────────────────────────────────

class InventoryTable {
  InventoryTable({required this.name, required this.count});

  factory InventoryTable.fromJson(Map<String, dynamic> json) => InventoryTable(
    name: json['name'] as String? ?? '',
    count: (json['count'] as num?)?.toInt() ?? 0,
  );

  final String name;
  final int count;
}

class InventoryGroup {
  InventoryGroup({
    required this.id,
    required this.label,
    required this.description,
    required this.tables,
  });

  factory InventoryGroup.fromJson(Map<String, dynamic> json) {
    final raw = json['tables'];
    final tables = raw is List
        ? raw
              .whereType<Map<String, dynamic>>()
              .map(InventoryTable.fromJson)
              .toList()
        : <InventoryTable>[];
    return InventoryGroup(
      id: json['id'] as String? ?? '',
      label: json['label'] as String? ?? '',
      description: json['description'] as String? ?? '',
      tables: tables,
    );
  }

  final String id;
  final String label;
  final String description;
  final List<InventoryTable> tables;
}

// ── exports (Plan C) ─────────────────────────────────────────────

// Mode flag for the `integrations` scope on an export. Wire values
// match the Go side and the web TS.
enum IntegrationExportMode {
  none('none'),
  metadata('metadata'),
  plaintext('plaintext');

  const IntegrationExportMode(this.wire);
  final String wire;

  static IntegrationExportMode fromWire(String? wire) {
    return switch (wire) {
      'plaintext' => IntegrationExportMode.plaintext,
      'metadata' => IntegrationExportMode.metadata,
      _ => IntegrationExportMode.none,
    };
  }
}

class ExportScope {
  ExportScope({
    required this.memories,
    required this.integrations,
    required this.customTasks,
  });

  factory ExportScope.fromJson(Map<String, dynamic> json) => ExportScope(
    memories: json['memories'] as bool? ?? false,
    integrations: IntegrationExportMode.fromWire(
      json['integrations'] as String?,
    ),
    customTasks: json['custom_tasks'] as bool? ?? false,
  );

  final bool memories;
  final IntegrationExportMode integrations;
  final bool customTasks;
}

class ExportRecord {
  ExportRecord({
    required this.id,
    required this.status,
    required this.requestedBy,
    required this.scope,
    required this.startedAt,
    required this.expiresAt,
    required this.bytes,
    this.finishedAt,
    this.sha256,
    this.downloadToken,
    this.error,
  });

  factory ExportRecord.fromJson(Map<String, dynamic> json) {
    final scope = json['scope'];
    final scopeMap = scope is Map
        ? Map<String, dynamic>.from(scope)
        : <String, dynamic>{};
    return ExportRecord(
      id: json['id'] as String? ?? '',
      status: json['status'] as String? ?? '',
      requestedBy: json['requested_by'] as String? ?? '',
      scope: ExportScope.fromJson(scopeMap),
      startedAt:
          DateTime.tryParse(json['started_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
      finishedAt: DateTime.tryParse(
        json['finished_at'] as String? ?? '',
      )?.toUtc(),
      expiresAt:
          DateTime.tryParse(json['expires_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
      bytes: (json['bytes'] as num?)?.toInt() ?? 0,
      sha256: json['sha256'] as String?,
      downloadToken: json['download_token'] as String?,
      error: json['error'] as String?,
    );
  }

  final String id;
  // pending | running | ready | failed | expired
  final String status;
  final String requestedBy;
  final ExportScope scope;
  final DateTime startedAt;
  final DateTime? finishedAt;
  final DateTime expiresAt;
  final int bytes;
  final String? sha256;
  // Single-use bearer token for the download URL — only present
  // once the export is in status==ready.
  final String? downloadToken;
  final String? error;
}

// ── imports (C reverse) ──────────────────────────────────────────

class EntityCounts {
  EntityCounts({
    required this.created,
    required this.skipped,
    required this.failed,
  });

  factory EntityCounts.fromJson(Map<String, dynamic>? json) {
    final m = json ?? <String, dynamic>{};
    return EntityCounts(
      created: (m['created'] as num?)?.toInt() ?? 0,
      skipped: (m['skipped'] as num?)?.toInt() ?? 0,
      failed: (m['failed'] as num?)?.toInt() ?? 0,
    );
  }

  final int created;
  final int skipped;
  final int failed;
}

class ImportRecord {
  ImportRecord({
    required this.id,
    required this.status,
    required this.requestedBy,
    required this.startedAt,
    required this.sourceBytes,
    required this.memories,
    required this.integrations,
    required this.customTasks,
    this.finishedAt,
    this.sourceFilename,
    this.error,
  });

  factory ImportRecord.fromJson(Map<String, dynamic> json) {
    final counts = json['counts'];
    final countsMap = counts is Map
        ? Map<String, dynamic>.from(counts)
        : <String, dynamic>{};
    return ImportRecord(
      id: json['id'] as String? ?? '',
      status: json['status'] as String? ?? '',
      requestedBy: json['requested_by'] as String? ?? '',
      startedAt:
          DateTime.tryParse(json['started_at'] as String? ?? '')?.toUtc() ??
          DateTime.fromMillisecondsSinceEpoch(0),
      finishedAt: DateTime.tryParse(
        json['finished_at'] as String? ?? '',
      )?.toUtc(),
      sourceFilename: json['source_filename'] as String?,
      sourceBytes: (json['source_bytes'] as num?)?.toInt() ?? 0,
      memories: EntityCounts.fromJson(
        countsMap['memories'] is Map
            ? Map<String, dynamic>.from(countsMap['memories'] as Map)
            : null,
      ),
      integrations: EntityCounts.fromJson(
        countsMap['integrations'] is Map
            ? Map<String, dynamic>.from(countsMap['integrations'] as Map)
            : null,
      ),
      customTasks: EntityCounts.fromJson(
        countsMap['custom_tasks'] is Map
            ? Map<String, dynamic>.from(countsMap['custom_tasks'] as Map)
            : null,
      ),
      error: json['error'] as String?,
    );
  }

  final String id;
  // pending | running | succeeded | failed
  final String status;
  final String requestedBy;
  final DateTime startedAt;
  final DateTime? finishedAt;
  final String? sourceFilename;
  final int sourceBytes;
  final EntityCounts memories;
  final EntityCounts integrations;
  final EntityCounts customTasks;
  final String? error;
}

class BackupsApi {
  BackupsApi(this._dio);
  final Dio _dio;

  // GET /backup-status — always-200 since PR #49. The response
  // carries explicit booleans (enabled, configured, requires_restart)
  // so the UI can render the right screen without inferring state
  // from HTTP error codes.
  Future<BackupStatusReport> status() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/backup-status');
      return BackupStatusReport.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /backup-setup. mode is either 'generate' (server picks
  // random key, returns it once) or 'paste' (caller supplies it).
  // Returns the key file path and requires_restart=true; the
  // generated passphrase is in result.passphrase iff mode=generate.
  Future<BackupSetupResult> setup({
    required String mode,
    String? passphrase,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/backup-setup',
        data: {'mode': mode, if (passphrase != null) 'passphrase': passphrase},
      );
      return BackupSetupResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /backup-setup/disable. Removes the key file. Refused 409
  // when bootSource is env (UI can't unset env vars).
  Future<void> disableSetup() async {
    try {
      await _dio.post<void>('/api/v1/backup-setup/disable');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<BackupRow>> list({int limit = 50, String? status}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/backups',
        queryParameters: {
          'limit': limit,
          if (status != null && status.isNotEmpty) 'status': status,
        },
      );
      final raw = res.data?['backups'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(BackupRow.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /backups → 202 with the freshly-inserted row (status='pending').
  // The actual dump runs async on the server; client should poll list
  // to watch the row transition to running → succeeded/failed.
  Future<BackupRow> runNow({
    String targetId = 'local',
    String kind = 'db_only',
    bool includeConfig = false,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/backups',
        data: {
          'target_id': targetId,
          'kind': kind,
          'include_config': includeConfig,
        },
      );
      return BackupRow.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /backup-recovery-kit → the backup passphrase wrapped under a
  // recovery passphrase the operator stores out-of-band. Returns the
  // raw kit JSON (caller copies/saves it).
  Future<Map<String, dynamic>> recoveryKit(String recoveryPassphrase) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/backup-recovery-kit',
        data: {'recovery_passphrase': recoveryPassphrase},
      );
      return res.data ?? <String, dynamic>{};
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // DELETE /backups/{id} — server marks the row deleted and removes
  // the underlying blob from its target. Audit row is retained.
  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/backups/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // /backup-schedules — recurring backup specs. Server picks `local`
  // as the default target if you POST without specifying one.
  Future<List<BackupSchedule>> listSchedules() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/backup-schedules',
      );
      final raw = res.data?['schedules'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(BackupSchedule.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<BackupSchedule> createSchedule({
    required String targetId,
    required int intervalSec,
    required int retention,
    required bool enabled,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/backup-schedules',
        data: {
          'target_id': targetId,
          'interval_sec': intervalSec,
          'retention': retention,
          'enabled': enabled,
        },
      );
      return BackupSchedule.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /backup-schedules/{id}. Each field optional — pass null to
  // leave the existing value untouched.
  Future<BackupSchedule> updateSchedule(
    String id, {
    int? intervalSec,
    int? retention,
    bool? enabled,
  }) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/backup-schedules/$id',
        data: {
          if (intervalSec != null) 'interval_sec': intervalSec,
          if (retention != null) 'retention': retention,
          if (enabled != null) 'enabled': enabled,
        },
      );
      return BackupSchedule.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> deleteSchedule(String id) async {
    try {
      await _dio.delete<void>('/api/v1/backup-schedules/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // /backup-targets — destinations a backup can be written to. Mobile
  // exposes list / test / toggle / delete; per-kind create/edit
  // (S3 / SMB / SFTP / WebDAV / rclone — each with 5+ fields and
  // long secrets) stays web-only.
  Future<List<BackupTarget>> listTargets() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/backup-targets',
      );
      final raw = res.data?['targets'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(BackupTarget.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /backup-targets — create a new destination. id is
  // optional (server auto-derives from kind when omitted, e.g.
  // "smb-1"). kind is required. config is the per-kind field
  // bag (see TARGET_KIND_FIELDS for what each kind accepts).
  Future<BackupTarget> createTarget({
    required String kind,
    required Map<String, dynamic> config,
    String? id,
    bool enabled = true,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/backup-targets',
        data: {
          if (id != null && id.isNotEmpty) 'id': id,
          'kind': kind,
          'config': config,
          'enabled': enabled,
        },
      );
      return BackupTarget.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // PATCH /backup-targets/{id} — partial update. Either field is
  // optional; pass null to leave it untouched. Used for both
  // editing config and the dedicated toggle-enabled flow (see
  // setTargetEnabled below).
  Future<BackupTarget> updateTarget(
    String id, {
    Map<String, dynamic>? config,
    bool? enabled,
  }) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/backup-targets/$id',
        data: {
          if (config != null) 'config': config,
          if (enabled != null) 'enabled': enabled,
        },
      );
      return BackupTarget.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<BackupTarget> setTargetEnabled(
    String id, {
    required bool enabled,
  }) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/backup-targets/$id',
        data: {'enabled': enabled},
      );
      return BackupTarget.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /backup-targets/{id}/test — server runs a connectivity
  // check (e.g. dial S3, list bucket) and returns 204 on success
  // or 502 with a payload on failure.
  Future<void> testTarget(String id) async {
    try {
      await _dio.post<void>('/api/v1/backup-targets/$id/test');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> deleteTarget(String id) async {
    try {
      await _dio.delete<void>('/api/v1/backup-targets/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── restore ──────────────────────────────────────────────────

  // POST /backups/restore — multipart bundle upload. When
  // [targetDsn] is empty the server restores into opendray's own
  // database and requires [confirm]=="I understand"; otherwise it
  // dials [targetDsn] and runs pg_restore against that DSN.
  //
  // Cap is 256 MiB (server-side ParseMultipartForm); the mobile
  // client honors that by streaming via MultipartFile.fromFile.
  // [apply] defaults to false (a dry run that validates the bundle and
  // returns a plan, changing nothing). Pass apply=true to commit. This
  // mirrors the server, whose restore now defaults to dry-run.
  Future<RestoreResult> restore({
    required File bundle,
    String? targetDsn,
    bool clean = false,
    bool apply = false,
    bool force = false,
    String? confirm,
    String? note,
  }) async {
    try {
      final form = FormData.fromMap({
        'bundle': await MultipartFile.fromFile(
          bundle.path,
          filename: bundle.uri.pathSegments.isNotEmpty
              ? bundle.uri.pathSegments.last
              : 'bundle.tar.gz.enc',
        ),
        if (targetDsn != null && targetDsn.isNotEmpty) 'target_dsn': targetDsn,
        'clean': clean ? 'true' : 'false',
        'apply': apply ? 'true' : 'false',
        if (force) 'force': 'true',
        if (confirm != null && confirm.isNotEmpty) 'confirm': confirm,
        if (note != null && note.isNotEmpty) 'note': note,
      });
      // Restore can run for minutes on a large DB; the default
      // dio receive timeout (the gateway's) is fine, but we
      // explicitly disable any per-request timeout here so a
      // long pg_restore doesn't get killed mid-stream.
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/backups/restore',
        data: form,
        options: Options(
          sendTimeout: Duration.zero,
          receiveTimeout: Duration.zero,
        ),
      );
      return RestoreResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── inventory ────────────────────────────────────────────────

  // GET /backup-inventory — live row counts grouped by feature
  // area (memories, integrations, custom tasks, ...). Used by
  // the main screen's "what's actually backed up" card.
  Future<List<InventoryGroup>> inventory() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/backup-inventory',
      );
      final raw = res.data?['groups'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(InventoryGroup.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── exports (Plan C) ─────────────────────────────────────────

  Future<List<ExportRecord>> listExports() async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/exports');
      final raw = res.data?['exports'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ExportRecord.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<ExportRecord> getExport(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/exports/$id');
      return ExportRecord.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /exports — kick off a new export. The server returns
  // immediately with status=pending; the worker runs the actual
  // bundling asynchronously. Client should poll list() (or
  // getExport(id)) until status transitions to ready/failed.
  Future<ExportRecord> createExport({
    required bool memories,
    required IntegrationExportMode integrations,
    required bool customTasks,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/exports',
        data: {
          'memories': memories,
          'integrations': integrations.wire,
          'custom_tasks': customTasks,
        },
      );
      return ExportRecord.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> deleteExport(String id) async {
    try {
      await _dio.delete<void>('/api/v1/exports/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Absolute URL for the export download endpoint. The token
  // is a single-use bearer in the `download_token` field of a
  // ready ExportRecord; the gateway invalidates it after one
  // successful GET. Returns a relative path that
  // url_launcher / dio download can resolve against baseUrl.
  String exportDownloadUrl(String id, String token) {
    final encId = Uri.encodeComponent(id);
    final encTok = Uri.encodeComponent(token);
    return '/api/v1/exports/$encId/download?token=$encTok';
  }

  // ── imports (C reverse) ──────────────────────────────────────

  Future<List<ImportRecord>> listImports({int limit = 20}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/imports',
        queryParameters: {'limit': limit},
      );
      final raw = res.data?['imports'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(ImportRecord.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<ImportRecord> getImport(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/imports/$id');
      return ImportRecord.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /imports — multipart bundle upload. Bundle must be a
  // .opd JSON package previously produced by /exports.
  Future<ImportRecord> createImport({
    required File bundle,
    required bool memories,
    required bool integrations,
    required bool customTasks,
  }) async {
    try {
      final form = FormData.fromMap({
        'bundle': await MultipartFile.fromFile(
          bundle.path,
          filename: bundle.uri.pathSegments.isNotEmpty
              ? bundle.uri.pathSegments.last
              : 'bundle.opd',
        ),
        'memories': memories ? 'true' : 'false',
        'integrations': integrations ? 'true' : 'false',
        'custom_tasks': customTasks ? 'true' : 'false',
      });
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/imports',
        data: form,
        options: Options(
          sendTimeout: Duration.zero,
          receiveTimeout: Duration.zero,
        ),
      );
      return ImportRecord.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final backupsApiProvider = Provider<BackupsApi>((ref) {
  return BackupsApi(ref.watch(dioProvider));
});
