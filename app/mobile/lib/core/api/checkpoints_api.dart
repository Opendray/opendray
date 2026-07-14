import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Context checkpoint — one captured snapshot of a session's working dir
// (uncommitted git diff + untracked files + input history). Mirrors the Go
// checkpoint.Checkpoint JSON. Manual fromJson to match the rest of models.
class Checkpoint {
  Checkpoint({
    required this.id,
    required this.sessionId,
    required this.createdAt,
    required this.trigger,
    required this.cwd,
    required this.isGit,
    required this.gitDirty,
    required this.diffBytes,
    required this.untrackedFiles,
    required this.untrackedBytes,
    required this.inputBytes,
    required this.truncated,
    this.gitHead,
    this.note,
  });

  factory Checkpoint.fromJson(Map<String, dynamic> json) => Checkpoint(
        id: json['id'] as String? ?? '',
        sessionId: json['session_id'] as String? ?? '',
        createdAt: DateTime.tryParse(json['created_at'] as String? ?? '') ??
            DateTime.now().toUtc(),
        trigger: json['trigger'] as String? ?? 'manual',
        cwd: json['cwd'] as String? ?? '',
        isGit: json['is_git'] as bool? ?? false,
        gitHead: (json['git_head'] as String?)?.isNotEmpty ?? false
            ? json['git_head'] as String
            : null,
        gitDirty: json['git_dirty'] as bool? ?? false,
        diffBytes: (json['diff_bytes'] as num?)?.toInt() ?? 0,
        untrackedFiles: (json['untracked_files'] as num?)?.toInt() ?? 0,
        untrackedBytes: (json['untracked_bytes'] as num?)?.toInt() ?? 0,
        inputBytes: (json['input_bytes'] as num?)?.toInt() ?? 0,
        truncated: json['truncated'] as bool? ?? false,
        note: (json['note'] as String?)?.isNotEmpty ?? false
            ? json['note'] as String
            : null,
      );

  final String id;
  final String sessionId;
  final DateTime createdAt;
  final String trigger; // 'interrupted' | 'manual'
  final String cwd;
  final bool isGit;
  final String? gitHead;
  final bool gitDirty;
  final int diffBytes;
  final int untrackedFiles;
  final int untrackedBytes;
  final int inputBytes;
  final bool truncated;
  final String? note;
}

// RestoreResult reports what a restore actually did (mirrors Go).
class RestoreResult {
  RestoreResult({
    required this.checkpointId,
    required this.diffApplied,
    required this.untrackedRestored,
    required this.untrackedSkipped,
  });

  factory RestoreResult.fromJson(Map<String, dynamic> json) => RestoreResult(
        checkpointId: json['checkpoint_id'] as String? ?? '',
        diffApplied: json['diff_applied'] as bool? ?? false,
        untrackedRestored: (json['untracked_restored'] as num?)?.toInt() ?? 0,
        untrackedSkipped: (json['untracked_skipped'] as List<dynamic>?)
                ?.whereType<String>()
                .toList() ??
            const [],
      );

  final String checkpointId;
  final bool diffApplied;
  final int untrackedRestored;
  final List<String> untrackedSkipped;
}

// CheckpointsApi maps /api/v1/sessions/{id}/checkpoints and
// /api/v1/checkpoints/{id}. Capture snapshots the working tree; restore
// re-applies it under the gateway's strict guards (a 409 guard failure
// surfaces as an ApiException whose message carries the reason).
class CheckpointsApi {
  CheckpointsApi(this._dio);
  final Dio _dio;

  Future<List<Checkpoint>> list(String sessionId) async {
    try {
      final res = await _dio
          .get<List<dynamic>>('/api/v1/sessions/$sessionId/checkpoints');
      return (res.data ?? [])
          .whereType<Map<String, dynamic>>()
          .map(Checkpoint.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<Checkpoint> capture(String sessionId, {String? note}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/sessions/$sessionId/checkpoints',
        data: {if (note != null && note.isNotEmpty) 'note': note},
      );
      return Checkpoint.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<String> diff(String checkpointId) async {
    try {
      final res = await _dio.get<String>(
        '/api/v1/checkpoints/$checkpointId/diff',
        options: Options(
          responseType: ResponseType.plain,
          headers: {'Accept': 'text/plain'},
        ),
      );
      return res.data ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<RestoreResult> restore(String checkpointId) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/checkpoints/$checkpointId/restore',
      );
      return RestoreResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String checkpointId) async {
    try {
      await _dio.delete<void>('/api/v1/checkpoints/$checkpointId');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final checkpointsApiProvider = Provider<CheckpointsApi>((ref) {
  return CheckpointsApi(ref.watch(dioProvider));
});
