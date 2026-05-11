import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/backups — list + run-now. Mobile is intentionally a
// thin observability surface: schedule editing and download/restore
// stay on the web admin where uploading multi-GB blobs from a phone
// is neither practical nor safe.

class BackupRow {
  BackupRow({
    required this.id,
    required this.targetId,
    required this.status,
    required this.triggeredBy,
    required this.startedAt,
    required this.bytes,
    required this.encrypted,
    this.scheduleId,
    this.finishedAt,
    this.targetPath,
    this.error,
  });

  factory BackupRow.fromJson(Map<String, dynamic> json) => BackupRow(
        id: json['id'] as String? ?? '',
        scheduleId: json['schedule_id'] as String?,
        targetId: json['target_id'] as String? ?? '',
        status: json['status'] as String? ?? '',
        triggeredBy: json['triggered_by'] as String? ?? '',
        startedAt:
            DateTime.tryParse(json['started_at'] as String? ?? '')?.toUtc() ??
                DateTime.now().toUtc(),
        finishedAt:
            DateTime.tryParse(json['finished_at'] as String? ?? '')?.toUtc(),
        bytes: (json['bytes'] as num?)?.toInt() ?? 0,
        encrypted: json['encrypted'] as bool? ?? false,
        targetPath: json['target_path'] as String?,
        error: json['error'] as String?,
      );

  final String id;
  final String? scheduleId;
  final String targetId;
  // pending | running | succeeded | failed | deleted
  final String status;
  // scheduler | manual | api
  final String triggeredBy;
  final DateTime startedAt;
  final DateTime? finishedAt;
  final int bytes;
  final bool encrypted;
  final String? targetPath;
  final String? error;
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

  factory BackupSchedule.fromJson(Map<String, dynamic> json) =>
      BackupSchedule(
        id: json['id'] as String? ?? '',
        targetId: json['target_id'] as String? ?? '',
        intervalSec: (json['interval_sec'] as num?)?.toInt() ?? 0,
        retention: (json['retention'] as num?)?.toInt() ?? 0,
        enabled: json['enabled'] as bool? ?? false,
        lastRunAt:
            DateTime.tryParse(json['last_run_at'] as String? ?? '')?.toUtc(),
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

// Feature health snapshot from /api/v1/backup-status. Used to show
// a "you're good to go" / "you're broken" banner at the top of the
// Backups screen so operators don't push Run now and watch it fail
// silently when pg_dump isn't on the server's PATH.
class BackupStatusReport {
  BackupStatusReport({
    required this.ok,
    required this.keyFingerprint,
    required this.pgDumpVersion,
    required this.pgRestoreVersion,
    this.pgDumpError,
  });

  factory BackupStatusReport.fromJson(Map<String, dynamic> json) =>
      BackupStatusReport(
        ok: json['ok'] as bool? ?? false,
        keyFingerprint: json['key_fingerprint'] as String? ?? '',
        pgDumpVersion: json['pg_dump_version'] as String? ?? '',
        pgRestoreVersion: json['pg_restore_version'] as String? ?? '',
        pgDumpError: json['pg_dump_error'] as String?,
      );

  // True when pg_dump resolved cleanly. Anything else and Run now
  // can't actually produce a backup.
  final bool ok;
  // Short identifier for the OPENDRAY_BACKUP_KEY currently configured.
  // Empty string when the feature is disabled (no key set server-side).
  final String keyFingerprint;
  final String pgDumpVersion;
  final String pgRestoreVersion;
  final String? pgDumpError;
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
      config:
          cfg is Map ? Map<String, dynamic>.from(cfg) : <String, dynamic>{},
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

class BackupsApi {
  BackupsApi(this._dio);
  final Dio _dio;

  // GET /backup-status — feature health snapshot. Cheap (no I/O
  // beyond exec pg_dump --version once), safe to call on every page
  // load.
  //
  // Returns null when the server responds 404 — that's the
  // documented signal that backup is compile-disabled (no
  // OPENDRAY_BACKUP_KEY / Backup.Enabled=false in opendray.toml).
  // See internal/app/app.go for the comment. The UI uses null to
  // show a setup-required screen instead of an error.
  Future<BackupStatusReport?> status() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/backup-status');
      return BackupStatusReport.fromJson(res.data ?? {});
    } on Object catch (e) {
      final apiErr = toApiException(e);
      if (apiErr.statusCode == 404) return null;
      throw apiErr;
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
    bool includeConfig = false,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/backups',
        data: {'target_id': targetId, 'include_config': includeConfig},
      );
      return BackupRow.fromJson(res.data ?? {});
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
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/backup-schedules');
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
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/backup-targets');
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
}

final backupsApiProvider = Provider<BackupsApi>((ref) {
  return BackupsApi(ref.watch(dioProvider));
});
