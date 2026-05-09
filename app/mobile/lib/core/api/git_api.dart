import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/git/* — read-only git operations against a working
// tree on the gateway host (status / log / diff / show). The
// inspector's Git tab uses this to surface project state without
// shelling into the running PTY.
class GitStatusFile {
  GitStatusFile({required this.xy, required this.path, this.oldPath});

  factory GitStatusFile.fromJson(Map<String, dynamic> json) => GitStatusFile(
        xy: json['xy'] as String? ?? '',
        path: json['path'] as String? ?? '',
        oldPath: json['old_path'] as String?,
      );

  // Two-letter porcelain code: index status + worktree status.
  // e.g. "M ", " M", "A ", "??", "MM", "R "
  final String xy;
  final String path;
  final String? oldPath;

  bool get isUntracked => xy == '??';
  bool get isStaged => xy.isNotEmpty && xy[0] != ' ' && xy[0] != '?';
  bool get isUnstaged => xy.length == 2 && xy[1] != ' ' && xy[1] != '?';
}

class GitStatusResponse {
  GitStatusResponse({
    required this.isRepo,
    required this.branch,
    required this.ahead,
    required this.behind,
    required this.upstream,
    required this.files,
  });

  factory GitStatusResponse.fromJson(Map<String, dynamic> json) {
    final raw = json['files'];
    final files = raw is List
        ? raw
            .whereType<Map<String, dynamic>>()
            .map(GitStatusFile.fromJson)
            .toList()
        : <GitStatusFile>[];
    return GitStatusResponse(
      isRepo: json['is_repo'] as bool? ?? false,
      branch: json['branch'] as String? ?? '',
      ahead: (json['ahead'] as num?)?.toInt() ?? 0,
      behind: (json['behind'] as num?)?.toInt() ?? 0,
      upstream: json['upstream'] as String? ?? '',
      files: files,
    );
  }

  final bool isRepo;
  final String branch;
  final int ahead;
  final int behind;
  final String upstream;
  final List<GitStatusFile> files;
}

class GitCommit {
  GitCommit({
    required this.hash,
    required this.shortHash,
    required this.author,
    required this.when,
    required this.subject,
  });

  factory GitCommit.fromJson(Map<String, dynamic> json) => GitCommit(
        hash: json['hash'] as String? ?? '',
        shortHash: json['short_hash'] as String? ?? '',
        author: json['author'] as String? ?? '',
        when: json['when'] as String? ?? '',
        subject: json['subject'] as String? ?? '',
      );

  final String hash;
  final String shortHash;
  final String author;
  final String when;
  final String subject;
}

class GitLogResponse {
  GitLogResponse({required this.isRepo, required this.commits});

  factory GitLogResponse.fromJson(Map<String, dynamic> json) {
    final raw = json['commits'];
    final commits = raw is List
        ? raw
            .whereType<Map<String, dynamic>>()
            .map(GitCommit.fromJson)
            .toList()
        : <GitCommit>[];
    return GitLogResponse(
      isRepo: json['is_repo'] as bool? ?? false,
      commits: commits,
    );
  }

  final bool isRepo;
  final List<GitCommit> commits;
}

enum GitDiffScope { unstaged, staged, all }

class GitApi {
  GitApi(this._dio);
  final Dio _dio;

  Future<GitStatusResponse> status(String path) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/git/status',
        queryParameters: {'path': path},
      );
      return GitStatusResponse.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<GitLogResponse> log(String path, {int limit = 50}) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/git/log',
        queryParameters: {'path': path, 'limit': limit},
      );
      return GitLogResponse.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // Returns the diff as plain text. `file` is repo-relative; empty
  // means "the whole working tree". `scope` selects whether we look
  // at unstaged (default), staged, or all (HEAD vs worktree).
  Future<String> diff({
    required String path,
    String file = '',
    GitDiffScope scope = GitDiffScope.unstaged,
  }) async {
    try {
      final res = await _dio.get<String>(
        '/api/v1/git/diff',
        queryParameters: {
          'path': path,
          if (file.isNotEmpty) 'file': file,
          'scope': scope.name,
        },
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

  Future<String> show({required String path, required String hash}) async {
    try {
      final res = await _dio.get<String>(
        '/api/v1/git/show',
        queryParameters: {'path': path, 'hash': hash},
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
}

final gitApiProvider = Provider<GitApi>((ref) => GitApi(ref.watch(dioProvider)));
