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
        ? raw.whereType<Map<String, dynamic>>().map(GitCommit.fromJson).toList()
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

// ── PR write ops + checks ─────────────────────────────────────────

class GitPullRequest {
  GitPullRequest({
    required this.number,
    required this.title,
    required this.state,
    required this.author,
    required this.head,
    required this.base,
    required this.url,
    required this.draft,
    required this.updatedAt,
  });

  factory GitPullRequest.fromJson(Map<String, dynamic> json) => GitPullRequest(
    number: (json['number'] as num?)?.toInt() ?? 0,
    title: json['title'] as String? ?? '',
    state: json['state'] as String? ?? '',
    author: json['author'] as String? ?? '',
    head: json['head'] as String? ?? '',
    base: json['base'] as String? ?? '',
    url: json['url'] as String? ?? '',
    draft: json['draft'] as bool? ?? false,
    updatedAt: json['updated_at'] as String? ?? '',
  );

  final int number;
  final String title;
  // open | closed | merged
  final String state;
  final String author;
  final String head;
  final String base;
  final String url;
  final bool draft;
  final String updatedAt;
}

// Container for the /git/prs response: PRs plus repo metadata so
// callers can render a "configure your token" hint when no host
// row matches the detected remote.
class GitPullRequestList {
  GitPullRequestList({
    required this.prs,
    required this.needsToken,
    required this.host,
    required this.errorMessage,
  });

  factory GitPullRequestList.fromJson(Map<String, dynamic> json) {
    final raw = json['prs'];
    final prs = raw is List
        ? raw
              .whereType<Map<String, dynamic>>()
              .map(GitPullRequest.fromJson)
              .toList()
        : <GitPullRequest>[];
    final remote = json['remote'];
    final host = remote is Map<String, dynamic>
        ? (remote['host'] as String? ?? '')
        : '';
    return GitPullRequestList(
      prs: prs,
      needsToken: json['need_token'] as bool? ?? false,
      host: host,
      errorMessage: json['error'] as String? ?? '',
    );
  }

  final List<GitPullRequest> prs;
  // True when the remote is detected but no git_hosts row has a
  // token for it. UI shows "Configure git host" deep link.
  final bool needsToken;
  final String host;
  // Non-empty when upstream returned a transport error (e.g.
  // rate limit). Distinct from needsToken which is a config gap.
  final String errorMessage;
}

// One ref returned by GET /git/write/branches. Mirrors the web
// shape: local + remote refs share the type, with is_remote +
// remote name disambiguating.
class GitBranchRef {
  GitBranchRef({
    required this.name,
    required this.remote,
    required this.isRemote,
    required this.isCurrent,
    required this.upstream,
  });

  factory GitBranchRef.fromJson(Map<String, dynamic> json) => GitBranchRef(
    name: json['name'] as String? ?? '',
    remote: json['remote'] as String? ?? '',
    isRemote: json['is_remote'] as bool? ?? false,
    isCurrent: json['is_current'] as bool? ?? false,
    upstream: json['upstream'] as String? ?? '',
  );

  final String name;
  final String remote; // populated only when isRemote
  final bool isRemote;
  final bool isCurrent;
  final String upstream; // e.g. "origin/main" for local branches
}

class GitBranchList {
  GitBranchList({required this.branches, required this.current});

  factory GitBranchList.fromJson(Map<String, dynamic> json) {
    final raw = json['branches'];
    final branches = raw is List
        ? raw
              .whereType<Map<String, dynamic>>()
              .map(GitBranchRef.fromJson)
              .toList()
        : <GitBranchRef>[];
    return GitBranchList(
      branches: branches,
      current: json['current'] as String? ?? '',
    );
  }

  final List<GitBranchRef> branches;
  final String current;
}

class GitCheckRun {
  GitCheckRun({
    required this.name,
    required this.status,
    required this.conclusion,
    required this.url,
    required this.updatedAt,
  });

  factory GitCheckRun.fromJson(Map<String, dynamic> json) => GitCheckRun(
    name: json['name'] as String? ?? '',
    status: json['status'] as String? ?? '',
    conclusion: json['conclusion'] as String? ?? '',
    url: json['url'] as String? ?? '',
    updatedAt: json['updated_at'] as String? ?? '',
  );

  // queued | in_progress | completed
  final String status;
  // success | failure | neutral | cancelled | skipped | timed_out |
  // action_required (filled when status == completed)
  final String conclusion;
  final String name;
  final String url;
  final String updatedAt;

  bool get passing =>
      status == 'completed' &&
      (conclusion == 'success' ||
          conclusion == 'neutral' ||
          conclusion == 'skipped');
  bool get failing =>
      status == 'completed' &&
      (conclusion == 'failure' ||
          conclusion == 'cancelled' ||
          conclusion == 'timed_out' ||
          conclusion == 'action_required');
}

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

  // POST /git/prs — create a PR. `base` is optional; the server
  // resolves the repo's default branch when omitted.
  Future<GitPullRequest> createPullRequest({
    required String dir,
    required String title,
    required String head,
    String? base,
    String? body,
    bool draft = false,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/git/prs',
        data: {
          'dir': dir,
          'title': title,
          'head': head,
          if (base != null && base.isNotEmpty) 'base': base,
          if (body != null && body.isNotEmpty) 'body': body,
          'draft': draft,
        },
      );
      return GitPullRequest.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /git/prs/{n}/merge — merges with the requested method.
  // `squash` is the default (matches the GitHub naming we use
  // everywhere; Gitea/GitLab adapters map it natively). When
  // `deleteBranch` is true the head branch is deleted after a
  // successful merge.
  Future<GitPullRequest> mergePullRequest({
    required String dir,
    required int number,
    String method = 'squash',
    String? commitTitle,
    String? commitMessage,
    bool deleteBranch = true,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/git/prs/$number/merge',
        data: {
          'dir': dir,
          'number': number,
          'method': method,
          if (commitTitle != null && commitTitle.isNotEmpty)
            'commit_title': commitTitle,
          if (commitMessage != null && commitMessage.isNotEmpty)
            'commit_message': commitMessage,
          'delete_branch': deleteBranch,
        },
      );
      return GitPullRequest.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /git/prs/{n}/checks — CI checks on the PR's head commit.
  // Multi-platform after Phase 2 (GitHub native, Gitea + GitLab
  // commit-status normalised). Other hosts still return empty.
  Future<List<GitCheckRun>> prChecks({
    required String dir,
    required int number,
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/git/prs/$number/checks',
        queryParameters: {'path': dir},
      );
      final raw = res.data?['checks'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(GitCheckRun.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // GET /git/prs?path=<dir>&state=<open|closed|all>. Mirrors the
  // web /lib/githost listGitPRs surface. Wraps the response in a
  // single GitPullRequestList because the server returns metadata
  // (remote info + need_token signal) alongside the PR array.
  Future<GitPullRequestList> listPullRequests({
    required String dir,
    String state = 'open',
  }) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/git/prs',
        queryParameters: {'path': dir, 'state': state},
      );
      return GitPullRequestList.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // ── Phase 4: branch / stage / commit / push ──────────────────

  Future<GitBranchList> listBranches(String dir) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/git/write/branches',
        queryParameters: {'path': dir},
      );
      return GitBranchList.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /git/write/branches — create + optional switch.
  Future<void> createBranch({
    required String dir,
    required String name,
    String? from,
    bool switchTo = true,
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/git/write/branches',
        data: {
          'dir': dir,
          'name': name,
          if (from != null && from.isNotEmpty) 'from': from,
          'switch': switchTo,
        },
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /git/write/checkout — refuses on dirty tree (server
  // returns 409, surfaces as ApiException with that status).
  Future<void> checkoutBranch({
    required String dir,
    required String name,
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/git/write/checkout',
        data: {'dir': dir, 'name': name},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // DELETE /git/write/branches?path=&name=&force=. Name lives in
  // the query (not the path) because feat/foo-style branches break
  // chi's single-segment {name} matcher. force=true upgrades the
  // safe -d to -D (forces deletion of unmerged branches).
  Future<void> deleteBranch({
    required String dir,
    required String name,
    bool force = false,
  }) async {
    try {
      await _dio.delete<void>(
        '/api/v1/git/write/branches',
        queryParameters: {
          'path': dir,
          'name': name,
          if (force) 'force': 'true',
        },
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /git/write/stage. Empty files = stage all (`.`).
  Future<void> stageFiles({
    required String dir,
    List<String> files = const [],
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/git/write/stage',
        data: {'dir': dir, 'files': files},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> unstageFiles({
    required String dir,
    List<String> files = const [],
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/git/write/unstage',
        data: {'dir': dir, 'files': files},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /git/write/commit. Returns the new HEAD sha.
  Future<String> commit({
    required String dir,
    required String message,
    bool allowEmpty = false,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/git/write/commit',
        data: {'dir': dir, 'message': message, 'allow_empty': allowEmpty},
      );
      return (res.data?['hash'] as String?) ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  // POST /git/write/push. branch empty = current HEAD. force
  // maps to --force-with-lease server-side. setUpstream wires
  // origin/<branch> the first time you push a branch.
  Future<String> push({
    required String dir,
    String? branch,
    bool force = false,
    bool setUpstream = false,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/git/write/push',
        data: {
          'dir': dir,
          if (branch != null && branch.isNotEmpty) 'branch': branch,
          'force': force,
          'set_upstream': setUpstream,
        },
      );
      return (res.data?['branch'] as String?) ?? '';
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

final gitApiProvider = Provider<GitApi>(
  (ref) => GitApi(ref.watch(dioProvider)),
);
