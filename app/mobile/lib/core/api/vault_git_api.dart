import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps the read + sync half of /api/v1/vault/git — the Vault's own git
// repo, which is what carries documents to GitHub.
//
// Deliberately partial. The repository-shaping endpoints the web client
// also exposes (init, remote configuration, abort, reset-to-remote) are
// NOT here: reset-to-remote discards local work, and a phone is the
// worst place to reach for it by accident. Do that from the web UI.

class VaultStatusFile {
  const VaultStatusFile({required this.xy, required this.path});

  factory VaultStatusFile.fromJson(Map<String, dynamic> json) =>
      VaultStatusFile(
        // Two-character porcelain code, e.g. " M", "??", "A ".
        xy: json['xy'] as String? ?? '',
        path: json['path'] as String? ?? '',
      );

  final String xy;
  final String path;
}

class VaultGitState {
  const VaultGitState({
    required this.rebaseInProgress,
    required this.mergeInProgress,
    required this.cherryPickInProgress,
    required this.conflictedFiles,
  });

  factory VaultGitState.fromJson(Map<String, dynamic> json) => VaultGitState(
        rebaseInProgress: json['rebase_in_progress'] as bool? ?? false,
        mergeInProgress: json['merge_in_progress'] as bool? ?? false,
        cherryPickInProgress: json['cherry_pick_in_progress'] as bool? ?? false,
        conflictedFiles: (json['conflicted_files'] as List?)
                ?.whereType<String>()
                .toList() ??
            const [],
      );

  final bool rebaseInProgress;
  final bool mergeInProgress;
  final bool cherryPickInProgress;
  final List<String> conflictedFiles;

  /// True while the repo is mid-operation. Committing or pushing on top
  /// of that is how you make a mess, so the UI blocks the actions and
  /// points at the web client instead.
  bool get isMidOperation =>
      rebaseInProgress || mergeInProgress || cherryPickInProgress;
}

class VaultStatus {
  const VaultStatus({
    required this.isRepo,
    required this.branch,
    required this.upstream,
    required this.ahead,
    required this.behind,
    required this.files,
    required this.root,
    required this.state,
  });

  factory VaultStatus.fromJson(Map<String, dynamic> json) => VaultStatus(
        isRepo: json['is_repo'] as bool? ?? false,
        branch: json['branch'] as String? ?? '',
        upstream: json['upstream'] as String? ?? '',
        ahead: (json['ahead'] as num?)?.toInt() ?? 0,
        behind: (json['behind'] as num?)?.toInt() ?? 0,
        files: (json['files'] as List?)
                ?.whereType<Map<String, dynamic>>()
                .map(VaultStatusFile.fromJson)
                .toList() ??
            const [],
        root: json['root'] as String? ?? '',
        state: json['state'] is Map<String, dynamic>
            ? VaultGitState.fromJson(json['state'] as Map<String, dynamic>)
            : null,
      );

  final bool isRepo;
  final String branch;
  final String upstream;
  final int ahead;
  final int behind;
  final List<VaultStatusFile> files;
  final String root;
  final VaultGitState? state;

  bool get hasRemote => upstream.isNotEmpty;
  bool get isClean => files.isEmpty;
}

// Mirrors vaultgit.SyncConfig. Intervals arrive as Go duration strings
// ("10m0s", "1h0m0s") and must be sent back in a form
// time.ParseDuration accepts — the server rejects anything else with a
// 400 rather than silently keeping the old value.
class VaultSyncConfig {
  const VaultSyncConfig({
    required this.enabled,
    required this.commitInterval,
    required this.pushEnabled,
    required this.pullEnabled,
    required this.pullInterval,
    required this.commitMessage,
    required this.lastCommitAt,
    required this.lastCommitHash,
    required this.lastPushAt,
    required this.lastPullAt,
    required this.lastError,
    required this.lastErrorAt,
  });

  factory VaultSyncConfig.fromJson(Map<String, dynamic> json) {
    DateTime? at(String key) {
      final raw = json[key] as String?;
      if (raw == null || raw.isEmpty) return null;
      return DateTime.tryParse(raw)?.toLocal();
    }

    return VaultSyncConfig(
      enabled: json['enabled'] as bool? ?? false,
      commitInterval: json['commit_interval'] as String? ?? '',
      pushEnabled: json['push_enabled'] as bool? ?? false,
      pullEnabled: json['pull_enabled'] as bool? ?? false,
      pullInterval: json['pull_interval'] as String? ?? '',
      commitMessage: json['commit_message'] as String? ?? '',
      lastCommitAt: at('last_commit_at'),
      lastCommitHash: json['last_commit_hash'] as String? ?? '',
      lastPushAt: at('last_push_at'),
      lastPullAt: at('last_pull_at'),
      lastError: json['last_error'] as String? ?? '',
      lastErrorAt: at('last_error_at'),
    );
  }

  final bool enabled;
  final String commitInterval;
  final bool pushEnabled;
  final bool pullEnabled;
  final String pullInterval;
  final String commitMessage;
  final DateTime? lastCommitAt;
  final String lastCommitHash;
  final DateTime? lastPushAt;
  final DateTime? lastPullAt;
  final String lastError;
  final DateTime? lastErrorAt;
}

class VaultGitApi {
  VaultGitApi(this._dio);
  final Dio _dio;

  Future<VaultStatus> status() async {
    try {
      final res =
          await _dio.get<Map<String, dynamic>>('/api/v1/vault/git/status');
      return VaultStatus.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Stages everything and commits, returning the new short hash — the
  /// only thing that actually proves a commit happened. The server
  /// answers 422 with git's stderr when there was nothing to commit, so
  /// callers should surface the message rather than claim success.
  Future<String> commit({String message = ''}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/vault/git/commit',
        data: {'message': message},
      );
      return res.data?['hash'] as String? ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<String> push() async {
    try {
      final res =
          await _dio.post<Map<String, dynamic>>('/api/v1/vault/git/push');
      return res.data?['output'] as String? ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<String> pull() async {
    try {
      final res =
          await _dio.post<Map<String, dynamic>>('/api/v1/vault/git/pull');
      return res.data?['output'] as String? ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<VaultSyncConfig> syncConfig() async {
    try {
      final res = await _dio
          .get<Map<String, dynamic>>('/api/v1/vault/git/sync/config');
      return VaultSyncConfig.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Pointer-style patch: only the fields passed are changed. Intervals
  /// must be Go duration strings; the server answers 400 on anything it
  /// can't parse, and that message is worth surfacing verbatim.
  Future<VaultSyncConfig> setSyncConfig({
    bool? enabled,
    String? commitInterval,
    bool? pushEnabled,
    bool? pullEnabled,
    String? pullInterval,
    String? commitMessage,
  }) async {
    try {
      final body = <String, dynamic>{
        if (enabled != null) 'enabled': enabled,
        if (commitInterval != null) 'commit_interval': commitInterval,
        if (pushEnabled != null) 'push_enabled': pushEnabled,
        if (pullEnabled != null) 'pull_enabled': pullEnabled,
        if (pullInterval != null) 'pull_interval': pullInterval,
        if (commitMessage != null) 'commit_message': commitMessage,
      };
      final res = await _dio.put<Map<String, dynamic>>(
        '/api/v1/vault/git/sync/config',
        data: body,
      );
      return VaultSyncConfig.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Wakes the sync loop so it doesn't wait for its next tick.
  Future<void> syncRunNow() async {
    try {
      await _dio.post<Map<String, dynamic>>('/api/v1/vault/git/sync/run');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final vaultGitApiProvider = Provider<VaultGitApi>((ref) {
  return VaultGitApi(ref.watch(dioProvider));
});

final vaultStatusProvider = FutureProvider.autoDispose<VaultStatus>((ref) {
  return ref.watch(vaultGitApiProvider).status();
});

final vaultSyncConfigProvider =
    FutureProvider.autoDispose<VaultSyncConfig>((ref) {
  return ref.watch(vaultGitApiProvider).syncConfig();
});
