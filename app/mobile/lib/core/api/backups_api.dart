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

class BackupsApi {
  BackupsApi(this._dio);
  final Dio _dio;

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
}

final backupsApiProvider = Provider<BackupsApi>((ref) {
  return BackupsApi(ref.watch(dioProvider));
});
