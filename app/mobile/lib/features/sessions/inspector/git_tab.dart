import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/sessions/inspector/git_branch_controls.dart';
import 'package:opendray/features/sessions/inspector/git_commit_form.dart';
import 'package:opendray/features/sessions/inspector/git_issue_section.dart';
import 'package:opendray/features/sessions/inspector/git_pr_section.dart';

// Git surface inside the session inspector. Two views toggle via a
// segmented control:
//   • Status — branch + ahead/behind + a list of changed files. Tap
//     a file to push its path / @reference into the live terminal,
//     or to view its diff in a scrollable dialog.
//   • Log    — recent commits. Tap a commit to view the full patch
//     or push the hash into the terminal as a quick reference.
class GitTab extends ConsumerStatefulWidget {
  const GitTab({required this.sessionId, required this.cwd, super.key});

  final String sessionId;
  final String cwd;

  @override
  ConsumerState<GitTab> createState() => _GitTabState();
}

class _GitTabState extends ConsumerState<GitTab>
    with AutomaticKeepAliveClientMixin {
  _Pane _pane = _Pane.status;
  AsyncValue<GitStatusResponse> _status = const AsyncValue.loading();
  AsyncValue<GitLogResponse> _log = const AsyncValue.loading();

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _loadStatus();
    _loadLog();
  }

  Future<void> _loadStatus() async {
    setState(() => _status = const AsyncValue.loading());
    try {
      final res = await ref.read(gitApiProvider).status(widget.cwd);
      if (!mounted) return;
      setState(() => _status = AsyncValue.data(res));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _status = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _status = AsyncValue.error(e, st));
    }
  }

  Future<void> _loadLog() async {
    setState(() => _log = const AsyncValue.loading());
    try {
      final res = await ref.read(gitApiProvider).log(widget.cwd);
      if (!mounted) return;
      setState(() => _log = AsyncValue.data(res));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _log = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _log = AsyncValue.error(e, st));
    }
  }

  Future<void> _pushInput(String text) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(sessionsApiProvider).input(widget.sessionId, text);
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.sessions.inspector.shared.inserted(text: text)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.shared.insertFailedApi(
              status: e.statusCode.toString(),
              message: e.message,
            ),
          ),
        ),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.shared.insertFailedGeneric(
              error: e.toString(),
            ),
          ),
        ),
      );
    }
  }

  Future<void> _onFileTap(GitStatusFile file) async {
    final action = await showModalBottomSheet<_FileAction>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      _StatusChip(xy: file.xy),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          file.path,
                          style: Theme.of(sheetCtx).textTheme.bodyMedium,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.alternate_email),
              title: Text(t.sessions.inspector.git.insertAtRef),
              onTap: () => Navigator.of(sheetCtx).pop(_FileAction.insertAt),
            ),
            ListTile(
              leading: const Icon(Icons.content_paste_go),
              title: Text(t.sessions.inspector.git.insertPath),
              onTap: () => Navigator.of(sheetCtx).pop(_FileAction.insertPath),
            ),
            if (!file.isUntracked)
              ListTile(
                leading: const Icon(Icons.compare_arrows),
                title: Text(t.sessions.inspector.git.showDiff),
                subtitle: Text(
                  file.isStaged ? 'staged changes' : 'unstaged changes',
                ),
                onTap: () => Navigator.of(sheetCtx).pop(_FileAction.diff),
              ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _FileAction.insertAt:
        await _pushInput('@${file.path}');
      case _FileAction.insertPath:
        await _pushInput(file.path);
      case _FileAction.diff:
        await _showDiff(file);
    }
  }

  Future<void> _showDiff(GitStatusFile file) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      final body = await ref
          .read(gitApiProvider)
          .diff(
            path: widget.cwd,
            file: file.path,
            scope: file.isStaged ? GitDiffScope.staged : GitDiffScope.unstaged,
          );
      if (!mounted) return;
      await _showTextDialog(title: file.path, body: body);
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.git.diffFailedApi(
              status: e.statusCode.toString(),
              message: e.message,
            ),
          ),
        ),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.git.diffFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  Future<void> _onCommitTap(GitCommit commit) async {
    final action = await showModalBottomSheet<_CommitAction>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    commit.shortHash,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 13,
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const SizedBox(height: 4),
                  Text(
                    commit.subject,
                    style: Theme.of(sheetCtx).textTheme.bodyMedium,
                  ),
                  const SizedBox(height: 4),
                  Text(
                    '${commit.author}  ·  ${commit.when}',
                    style: Theme.of(sheetCtx).textTheme.bodySmall,
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.tag_outlined),
              title: Text(t.sessions.inspector.git.insertHash),
              subtitle: Text(commit.shortHash),
              onTap: () => Navigator.of(sheetCtx).pop(_CommitAction.insertHash),
            ),
            ListTile(
              leading: const Icon(Icons.description_outlined),
              title: Text(t.sessions.inspector.git.showFullPatch),
              onTap: () => Navigator.of(sheetCtx).pop(_CommitAction.show),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _CommitAction.insertHash:
        await _pushInput(commit.shortHash);
      case _CommitAction.show:
        await _showCommit(commit);
    }
  }

  Future<void> _showCommit(GitCommit commit) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      final body = await ref
          .read(gitApiProvider)
          .show(path: widget.cwd, hash: commit.hash);
      if (!mounted) return;
      await _showTextDialog(title: commit.shortHash, body: body);
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.git.showFailedApi(
              status: e.statusCode.toString(),
              message: e.message,
            ),
          ),
        ),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.git.showFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  Future<void> _showTextDialog({
    required String title,
    required String body,
  }) async {
    await showDialog<void>(
      context: context,
      builder: (dialogCtx) => Dialog(
        insetPadding: const EdgeInsets.all(16),
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(dialogCtx).size.height * 0.85,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 12, 8, 8),
                child: Row(
                  children: [
                    Expanded(
                      child: Text(
                        title,
                        style: Theme.of(dialogCtx).textTheme.titleSmall,
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.close),
                      onPressed: () => Navigator.of(dialogCtx).pop(),
                    ),
                  ],
                ),
              ),
              const Divider(height: 1),
              Flexible(
                child: SingleChildScrollView(
                  scrollDirection: Axis.vertical,
                  padding: const EdgeInsets.all(12),
                  child: SingleChildScrollView(
                    scrollDirection: Axis.horizontal,
                    child: SelectableText(
                      body.isEmpty ? '(empty)' : body,
                      style: const TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 11,
                      ),
                    ),
                  ),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return Column(
      children: [
        _PaneSwitch(
          pane: _pane,
          onChanged: (p) => setState(() => _pane = p),
          onRefresh: _pane == _Pane.status ? _loadStatus : _loadLog,
        ),
        const Divider(height: 1),
        Expanded(child: _pane == _Pane.status ? _statusBody() : _logBody()),
      ],
    );
  }

  Widget _statusBody() {
    return _status.when(
      data: (res) {
        if (!res.isRepo) {
          return _NotARepoView(cwd: widget.cwd);
        }
        return RefreshIndicator(
          onRefresh: _loadStatus,
          child: ListView(
            children: [
              _BranchHeader(status: res),
              // Phase 4: branch picker + create + push. Refreshes
              // status / log when any branch op succeeds so the
              // header stays in sync with the actual repo state.
              GitBranchControls(
                cwd: widget.cwd,
                ahead: res.ahead,
                upstream: res.upstream,
                onChanged: () {
                  _loadStatus();
                  _loadLog();
                },
              ),
              const Divider(height: 1),
              if (res.files.isEmpty)
                Padding(
                  padding: const EdgeInsets.all(32),
                  child: Center(
                    child: Text(
                      'Working tree is clean',
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  ),
                )
              else ...[
                for (final f in res.files)
                  Column(
                    children: [
                      ListTile(
                        leading: _StatusChip(xy: f.xy),
                        title: Text(
                          f.path,
                          style: const TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 13,
                          ),
                          overflow: TextOverflow.ellipsis,
                        ),
                        onTap: () => _onFileTap(f),
                      ),
                      Divider(height: 1, color: Theme.of(context).dividerColor),
                    ],
                  ),
                // Phase 4: commit form under the file list when
                // there's anything to commit. Refreshes status +
                // log on success so the new commit shows up.
                GitCommitForm(
                  cwd: widget.cwd,
                  files: res.files,
                  onChanged: () {
                    _loadStatus();
                    _loadLog();
                  },
                ),
                const Divider(height: 1),
              ],
              // Phase 1+2: PR command-center. Always renders so
              // operators can create a PR even on a clean tree.
              GitPRSection(cwd: widget.cwd),
              // Issues — read-only sibling of the PR section.
              GitIssueSection(cwd: widget.cwd),
            ],
          ),
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorView(error: e, onRetry: _loadStatus),
    );
  }

  Widget _logBody() {
    return _log.when(
      data: (res) {
        if (!res.isRepo) {
          return _NotARepoView(cwd: widget.cwd);
        }
        if (res.commits.isEmpty) {
          return Center(
            child: Padding(
              padding: const EdgeInsets.all(32),
              child: Text(
                'No commits yet',
                style: Theme.of(context).textTheme.bodyMedium,
              ),
            ),
          );
        }
        return RefreshIndicator(
          onRefresh: _loadLog,
          child: ListView.separated(
            itemCount: res.commits.length,
            separatorBuilder: (_, __) =>
                Divider(height: 1, color: Theme.of(context).dividerColor),
            itemBuilder: (_, i) {
              final c = res.commits[i];
              return ListTile(
                onTap: () => _onCommitTap(c),
                title: Text(
                  c.subject,
                  style: Theme.of(context).textTheme.bodyMedium,
                  overflow: TextOverflow.ellipsis,
                ),
                subtitle: Text(
                  '${c.shortHash}  ·  ${c.author}  ·  ${c.when}',
                  style: Theme.of(context).textTheme.bodySmall,
                  overflow: TextOverflow.ellipsis,
                ),
                trailing: const Icon(Icons.chevron_right),
              );
            },
          ),
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorView(error: e, onRetry: _loadLog),
    );
  }
}

enum _Pane { status, log }

enum _FileAction { insertAt, insertPath, diff }

enum _CommitAction { insertHash, show }

class _PaneSwitch extends StatelessWidget {
  const _PaneSwitch({
    required this.pane,
    required this.onChanged,
    required this.onRefresh,
  });

  final _Pane pane;
  final ValueChanged<_Pane> onChanged;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 4, 8),
      child: Row(
        children: [
          Expanded(
            child: SegmentedButton<_Pane>(
              segments: [
                ButtonSegment<_Pane>(
                  value: _Pane.status,
                  icon: const Icon(Icons.list_alt, size: 18),
                  label: Text(t.sessions.inspector.git.tabStatus),
                ),
                ButtonSegment<_Pane>(
                  value: _Pane.log,
                  icon: const Icon(Icons.history, size: 18),
                  label: Text(t.sessions.inspector.git.tabLog),
                ),
              ],
              selected: {pane},
              onSelectionChanged: (s) => onChanged(s.first),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: onRefresh,
          ),
        ],
      ),
    );
  }
}

class _BranchHeader extends StatelessWidget {
  const _BranchHeader({required this.status});
  final GitStatusResponse status;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final muted = Theme.of(context).textTheme.bodySmall;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Icon(
                Icons.account_tree_outlined,
                size: 16,
                color: scheme.primary,
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  status.branch.isEmpty ? '(detached HEAD)' : status.branch,
                  style: Theme.of(
                    context,
                  ).textTheme.titleSmall?.copyWith(fontWeight: FontWeight.w600),
                ),
              ),
              if (status.ahead > 0)
                Padding(
                  padding: const EdgeInsets.only(left: 6),
                  child: Text('↑${status.ahead}', style: muted),
                ),
              if (status.behind > 0)
                Padding(
                  padding: const EdgeInsets.only(left: 6),
                  child: Text('↓${status.behind}', style: muted),
                ),
            ],
          ),
          if (status.upstream.isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 2),
              child: Text('tracking ${status.upstream}', style: muted),
            ),
        ],
      ),
    );
  }
}

class _StatusChip extends StatelessWidget {
  const _StatusChip({required this.xy});
  final String xy;

  @override
  Widget build(BuildContext context) {
    final (label, color) = _label(xy);
    return Container(
      width: 32,
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        border: Border.all(color: color.withValues(alpha: 0.5)),
        borderRadius: BorderRadius.circular(4),
      ),
      alignment: Alignment.center,
      child: Text(
        label,
        style: TextStyle(
          fontFamily: 'monospace',
          fontSize: 11,
          fontWeight: FontWeight.w600,
          color: color,
        ),
      ),
    );
  }

  static (String, Color) _label(String xy) {
    if (xy == '??') return ('??', Colors.amber);
    if (xy.startsWith('A')) return ('A', Colors.green);
    if (xy.startsWith('M') || xy.endsWith('M')) return ('M', Colors.blueAccent);
    if (xy.startsWith('D') || xy.endsWith('D')) return ('D', Colors.redAccent);
    if (xy.startsWith('R')) return ('R', Colors.purpleAccent);
    return (xy.trim().isEmpty ? '·' : xy.trim(), Colors.grey);
  }
}

class _NotARepoView extends StatelessWidget {
  const _NotARepoView({required this.cwd});
  final String cwd;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(
      context,
    ).colorScheme.onSurface.withValues(alpha: 0.5);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.folder_off_outlined, size: 56, color: muted),
            const SizedBox(height: 12),
            Text(
              'Not a git repository',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              cwd,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final Object error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              error.toString(),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
