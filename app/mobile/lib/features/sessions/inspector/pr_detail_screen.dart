import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';
import 'package:url_launcher/url_launcher.dart';

// PRDetailScreen is the full-screen, GitHub-style PR view opened when a
// row is tapped (any state). Four tabs — Conversation / Commits /
// Checks / Files changed — all read-only; the merge action (open PRs
// only) sits in a footer below the tabs.
//
// Returns `true` via Navigator.pop when the PR was merged so the caller
// can refresh its list.
class PRDetailScreen extends ConsumerStatefulWidget {
  const PRDetailScreen({required this.cwd, required this.pr, super.key});

  final String cwd;
  final GitPullRequest pr;

  @override
  ConsumerState<PRDetailScreen> createState() => _PRDetailScreenState();
}

class _PRDetailScreenState extends ConsumerState<PRDetailScreen> {
  // Seeded from the list row so the header paints instantly; replaced
  // by the detail fetch (which carries the body) once it resolves.
  late GitPullRequest _pr;
  bool _detailLoading = true;
  Object? _detailError;

  String _method = 'squash';
  bool _deleteBranch = true;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _pr = widget.pr;
    unawaited(_loadDetail());
  }

  Future<void> _loadDetail() async {
    try {
      final full = await ref
          .read(gitApiProvider)
          .getPullRequest(dir: widget.cwd, number: widget.pr.number);
      if (!mounted) return;
      setState(() {
        _pr = full;
        _detailLoading = false;
        _detailError = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() {
        _detailError = e;
        _detailLoading = false;
      });
    }
  }

  Future<void> _openInBrowser() async {
    final uri = Uri.tryParse(_pr.url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // Best-effort convenience link; never crash the screen.
    }
  }

  Future<void> _merge() async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('Merge PR #${_pr.number}?'),
        content: Text(
          '$_method${_deleteBranch ? " · delete branch" : ""}\n\n${_pr.title}',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Merge'),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    final errorColor = Theme.of(context).colorScheme.error;
    try {
      await ref
          .read(gitApiProvider)
          .mergePullRequest(
            dir: widget.cwd,
            number: _pr.number,
            method: _method,
            deleteBranch: _deleteBranch,
          );
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('Merged PR #${_pr.number}')),
      );
      navigator.pop(true); // tell the caller to refresh its list
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Merge failed: ${e.message}'),
          backgroundColor: errorColor,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final detailError = _detailError == null ? null : _errMsg(_detailError);
    return DefaultTabController(
      length: 4,
      child: Scaffold(
        appBar: AppBar(
          title: Text('PR #${_pr.number}'),
          actions: [
            if (_pr.url.isNotEmpty)
              IconButton(
                tooltip: 'Open on host',
                icon: const Icon(Icons.open_in_new),
                onPressed: _openInBrowser,
              ),
          ],
        ),
        body: Column(
          children: [
            _header(theme),
            const TabBar(
              isScrollable: true,
              tabAlignment: TabAlignment.start,
              tabs: [
                Tab(text: 'Conversation'),
                Tab(text: 'Commits'),
                Tab(text: 'Checks'),
                Tab(text: 'Files'),
              ],
            ),
            Expanded(
              child: TabBarView(
                children: [
                  _ConversationTab(
                    cwd: widget.cwd,
                    number: _pr.number,
                    author: _pr.author,
                    body: _pr.body,
                    detailLoading: _detailLoading,
                    detailError: detailError,
                  ),
                  _CommitsTab(cwd: widget.cwd, number: _pr.number),
                  _ChecksTab(cwd: widget.cwd, number: _pr.number),
                  _FilesTab(cwd: widget.cwd, number: _pr.number),
                ],
              ),
            ),
          ],
        ),
        bottomNavigationBar: _pr.state == 'open' ? _mergeBar(theme) : null,
      ),
    );
  }

  Widget _header(ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(_pr.title, style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              _StateBadge(pr: _pr),
              Text(
                '${_pr.head} → ${_pr.base}',
                style: theme.textTheme.bodySmall?.copyWith(
                  fontFamily: 'monospace',
                  color: theme.colorScheme.outline,
                ),
              ),
              Text(
                '#${_pr.number} · ${_pr.author}',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _mergeBar(ThemeData theme) {
    return SafeArea(
      child: Container(
        padding: const EdgeInsets.fromLTRB(12, 6, 12, 6),
        decoration: BoxDecoration(
          border: Border(top: BorderSide(color: theme.dividerColor)),
        ),
        child: Row(
          children: [
            DropdownButton<String>(
              value: _method,
              isDense: true,
              items: const [
                DropdownMenuItem(value: 'squash', child: Text('squash')),
                DropdownMenuItem(value: 'merge', child: Text('merge')),
                DropdownMenuItem(value: 'rebase', child: Text('rebase')),
              ],
              onChanged: _busy
                  ? null
                  : (v) {
                      if (v != null) setState(() => _method = v);
                    },
            ),
            const SizedBox(width: 8),
            Checkbox(
              value: _deleteBranch,
              onChanged: _busy
                  ? null
                  : (v) => setState(() => _deleteBranch = v ?? false),
              visualDensity: VisualDensity.compact,
            ),
            Text('Delete branch', style: theme.textTheme.bodySmall),
            const Spacer(),
            FilledButton.icon(
              onPressed: _busy ? null : _merge,
              icon: _busy
                  ? const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.merge, size: 16),
              label: const Text('Merge'),
            ),
          ],
        ),
      ),
    );
  }
}

// ── Conversation tab ──────────────────────────────────────────────

class _ConversationTab extends ConsumerStatefulWidget {
  const _ConversationTab({
    required this.cwd,
    required this.number,
    required this.author,
    required this.body,
    required this.detailLoading,
    required this.detailError,
  });

  final String cwd;
  final int number;
  final String author;
  final String body; // PR description (from the parent detail fetch)
  final bool detailLoading;
  final String? detailError;

  @override
  ConsumerState<_ConversationTab> createState() => _ConversationTabState();
}

class _ConversationTabState extends ConsumerState<_ConversationTab> {
  List<GitPRComment>? _comments;
  Object? _error;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    try {
      final c = await ref
          .read(gitApiProvider)
          .prComments(dir: widget.cwd, number: widget.number);
      if (!mounted) return;
      setState(() {
        _comments = c;
        _error = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final comments = _comments ?? const <GitPRComment>[];
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        // Description
        _CommentCard(
          author: widget.author,
          when: '',
          child: _descriptionBody(theme),
        ),
        const SizedBox(height: 8),
        // Comments
        if (_error != null)
          Text(
            'Comments unavailable: ${_errMsg(_error)}',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
          )
        else if (_comments == null)
          const Padding(
            padding: EdgeInsets.symmetric(vertical: 8),
            child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
          )
        else if (comments.isEmpty)
          Text(
            'No comments yet.',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
          )
        else
          for (final c in comments) ...[
            _CommentCard(
              author: c.author,
              when: _relTime(c.createdAt),
              state: c.state.isEmpty ? null : c.state,
              child: SelectableText(c.body, style: theme.textTheme.bodyMedium),
            ),
            const SizedBox(height: 8),
          ],
      ],
    );
  }

  Widget _descriptionBody(ThemeData theme) {
    if (widget.detailLoading && widget.body.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 4),
        child: SizedBox(
          height: 16,
          child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
        ),
      );
    }
    if (widget.detailError != null && widget.body.isEmpty) {
      return Text(
        "Couldn't load details: ${widget.detailError}",
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.error,
        ),
      );
    }
    if (widget.body.trim().isEmpty) {
      return Text(
        'No description provided.',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
          fontStyle: FontStyle.italic,
        ),
      );
    }
    return SelectableText(
      widget.body.trim(),
      style: theme.textTheme.bodyMedium,
    );
  }
}

class _CommentCard extends StatelessWidget {
  const _CommentCard({
    required this.author,
    required this.when,
    required this.child,
    this.state,
  });

  final String author;
  final String when;
  final String? state;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      decoration: BoxDecoration(
        border: Border.all(color: theme.dividerColor),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest,
              borderRadius: const BorderRadius.vertical(top: Radius.circular(8)),
            ),
            child: Row(
              children: [
                Text(
                  author,
                  style: theme.textTheme.bodySmall?.copyWith(
                    fontFamily: 'monospace',
                  ),
                ),
                if (state != null) ...[
                  const SizedBox(width: 6),
                  _ReviewStateBadge(state: state!),
                ],
                const Spacer(),
                if (when.isNotEmpty)
                  Text(
                    when,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                      fontSize: 10,
                    ),
                  ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.all(10),
            child: SizedBox(width: double.infinity, child: child),
          ),
        ],
      ),
    );
  }
}

class _ReviewStateBadge extends StatelessWidget {
  const _ReviewStateBadge({required this.state});
  final String state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = switch (state) {
      'approved' => Colors.green,
      'changes_requested' => theme.colorScheme.error,
      _ => theme.colorScheme.outline,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        border: Border.all(color: color),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        state.replaceAll('_', ' '),
        style: theme.textTheme.labelSmall?.copyWith(
          color: color,
          fontSize: 9,
        ),
      ),
    );
  }
}

// ── Commits tab ───────────────────────────────────────────────────

class _CommitsTab extends ConsumerStatefulWidget {
  const _CommitsTab({required this.cwd, required this.number});
  final String cwd;
  final int number;

  @override
  ConsumerState<_CommitsTab> createState() => _CommitsTabState();
}

class _CommitsTabState extends ConsumerState<_CommitsTab> {
  List<GitPRCommit>? _commits;
  Object? _error;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    try {
      final c = await ref
          .read(gitApiProvider)
          .prCommits(dir: widget.cwd, number: widget.number);
      if (!mounted) return;
      setState(() {
        _commits = c;
        _error = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (_error != null) {
      return _TabMessage(text: 'Commits unavailable: ${_errMsg(_error)}');
    }
    if (_commits == null) {
      return const Center(child: CircularProgressIndicator(strokeWidth: 2));
    }
    if (_commits!.isEmpty) {
      return const _TabMessage(text: 'No commits.');
    }
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: _commits!.length,
      separatorBuilder: (_, __) => Divider(height: 14, color: theme.dividerColor),
      itemBuilder: (context, i) {
        final c = _commits![i];
        return Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Icon(Icons.commit, size: 16, color: theme.colorScheme.outline),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    c.subject,
                    style: theme.textTheme.bodyMedium,
                    overflow: TextOverflow.ellipsis,
                  ),
                  Text(
                    '${c.author} · ${_relTime(c.date)}',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                      fontFamily: 'monospace',
                      fontSize: 10,
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(width: 8),
            Text(
              c.shortSha,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
                fontFamily: 'monospace',
                fontSize: 10,
              ),
            ),
          ],
        );
      },
    );
  }
}

// ── Checks tab ────────────────────────────────────────────────────

class _ChecksTab extends ConsumerStatefulWidget {
  const _ChecksTab({required this.cwd, required this.number});
  final String cwd;
  final int number;

  @override
  ConsumerState<_ChecksTab> createState() => _ChecksTabState();
}

class _ChecksTabState extends ConsumerState<_ChecksTab> {
  List<GitCheckRun>? _checks;
  Object? _error;
  Timer? _timer;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
    _timer = Timer.periodic(
      const Duration(seconds: 30),
      (_) => unawaited(_load()),
    );
  }

  @override
  void dispose() {
    _timer?.cancel();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final c = await ref
          .read(gitApiProvider)
          .prChecks(dir: widget.cwd, number: widget.number);
      if (!mounted) return;
      setState(() {
        _checks = c;
        _error = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _error = e);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (_error != null) {
      return _TabMessage(text: 'Checks unavailable: ${_errMsg(_error)}');
    }
    if (_checks == null) {
      return const Center(child: CircularProgressIndicator(strokeWidth: 2));
    }
    if (_checks!.isEmpty) {
      return const _TabMessage(text: 'No checks configured for this PR.');
    }
    return ListView(
      padding: const EdgeInsets.all(16),
      children: [
        _ChecksSummary(checks: _checks!),
        const SizedBox(height: 8),
        for (final c in _checks!)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 3),
            child: Row(
              children: [
                Icon(_checkIcon(c), size: 14, color: _checkColor(theme, c)),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    c.name,
                    style: theme.textTheme.bodySmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
          ),
      ],
    );
  }
}

// ── Files changed tab ─────────────────────────────────────────────

class _FilesTab extends ConsumerStatefulWidget {
  const _FilesTab({required this.cwd, required this.number});
  final String cwd;
  final int number;

  @override
  ConsumerState<_FilesTab> createState() => _FilesTabState();
}

class _FilesTabState extends ConsumerState<_FilesTab> {
  List<GitPRFile>? _files;
  Object? _error;
  Set<String> _expanded = {};

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    try {
      final f = await ref
          .read(gitApiProvider)
          .prFiles(dir: widget.cwd, number: widget.number);
      if (!mounted) return;
      setState(() {
        _files = f;
        _error = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _error = e);
    }
  }

  void _toggle(String name) {
    final next = Set<String>.from(_expanded);
    if (next.contains(name)) {
      next.remove(name);
    } else {
      next.add(name);
    }
    setState(() => _expanded = next);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (_error != null) {
      return _TabMessage(text: 'Files unavailable: ${_errMsg(_error)}');
    }
    if (_files == null) {
      return const Center(child: CircularProgressIndicator(strokeWidth: 2));
    }
    if (_files!.isEmpty) {
      return const _TabMessage(text: 'No changed files.');
    }
    return ListView.separated(
      padding: const EdgeInsets.all(16),
      itemCount: _files!.length,
      separatorBuilder: (_, __) => const SizedBox(height: 6),
      itemBuilder: (context, i) {
        final f = _files![i];
        final isOpen = _expanded.contains(f.filename);
        return Container(
          decoration: BoxDecoration(
            border: Border.all(color: theme.dividerColor),
            borderRadius: BorderRadius.circular(6),
          ),
          clipBehavior: Clip.antiAlias,
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              InkWell(
                onTap: () => _toggle(f.filename),
                child: Padding(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 8,
                    vertical: 8,
                  ),
                  child: Row(
                    children: [
                      _StatusLetter(status: f.status),
                      const SizedBox(width: 8),
                      Expanded(
                        child: Text(
                          f.filename,
                          style: theme.textTheme.bodySmall?.copyWith(
                            fontFamily: 'monospace',
                          ),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 8),
                      Text(
                        '+${f.additions}',
                        style: const TextStyle(
                          color: Colors.green,
                          fontSize: 11,
                        ),
                      ),
                      const SizedBox(width: 4),
                      Text(
                        '-${f.deletions}',
                        style: TextStyle(
                          color: theme.colorScheme.error,
                          fontSize: 11,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              if (isOpen)
                f.patch.isEmpty
                    ? Padding(
                        padding: const EdgeInsets.symmetric(
                          horizontal: 8,
                          vertical: 8,
                        ),
                        child: Text(
                          'No inline diff available.',
                          style: theme.textTheme.bodySmall?.copyWith(
                            color: theme.colorScheme.outline,
                          ),
                        ),
                      )
                    : _DiffView(patch: f.patch),
            ],
          ),
        );
      },
    );
  }
}

class _DiffView extends StatelessWidget {
  const _DiffView({required this.patch});
  final String patch;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final lines = patch.split('\n');
    return Container(
      width: double.infinity,
      decoration: BoxDecoration(
        border: Border(top: BorderSide(color: theme.dividerColor)),
      ),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            for (final line in lines) _diffLine(theme, line),
          ],
        ),
      ),
    );
  }

  Widget _diffLine(ThemeData theme, String line) {
    final isAdd = line.startsWith('+') && !line.startsWith('+++');
    final isDel = line.startsWith('-') && !line.startsWith('---');
    final isHunk = line.startsWith('@@');
    Color? bg;
    var fg = theme.colorScheme.onSurface;
    if (isAdd) {
      bg = Colors.green.withValues(alpha: 0.12);
      fg = Colors.green.shade400;
    } else if (isDel) {
      bg = theme.colorScheme.error.withValues(alpha: 0.12);
      fg = theme.colorScheme.error;
    } else if (isHunk) {
      fg = theme.colorScheme.outline;
    }
    return Container(
      color: bg,
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 1),
      child: Text(
        line.isEmpty ? ' ' : line,
        style: TextStyle(fontFamily: 'monospace', fontSize: 11, color: fg),
      ),
    );
  }
}

class _StatusLetter extends StatelessWidget {
  const _StatusLetter({required this.status});
  final String status;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (String ch, Color color) = switch (status) {
      'added' => ('A', Colors.green),
      'removed' => ('D', theme.colorScheme.error),
      'renamed' => ('R', Colors.purple),
      _ => ('M', theme.colorScheme.outline),
    };
    return SizedBox(
      width: 14,
      child: Text(
        ch,
        textAlign: TextAlign.center,
        style: TextStyle(
          fontFamily: 'monospace',
          fontSize: 11,
          color: color,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

// ── shared bits ───────────────────────────────────────────────────

class _TabMessage extends StatelessWidget {
  const _TabMessage({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Text(
        text,
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
        ),
      ),
    );
  }
}

String _errMsg(Object? e) {
  if (e is ApiException) return e.message;
  return '$e';
}

String _relTime(String iso) {
  final t = DateTime.tryParse(iso);
  if (t == null) return iso;
  final d = DateTime.now().difference(t);
  if (d.inDays > 365) return '${(d.inDays / 365).floor()}y ago';
  if (d.inDays > 30) return '${(d.inDays / 30).floor()}mo ago';
  if (d.inDays > 0) return '${d.inDays}d ago';
  if (d.inHours > 0) return '${d.inHours}h ago';
  if (d.inMinutes > 0) return '${d.inMinutes}m ago';
  return 'just now';
}

IconData _checkIcon(GitCheckRun c) {
  if (c.passing) return Icons.check_circle_outlined;
  if (c.failing) return Icons.cancel_outlined;
  return Icons.circle_outlined;
}

Color _checkColor(ThemeData theme, GitCheckRun c) {
  if (c.passing) return Colors.green;
  if (c.failing) return theme.colorScheme.error;
  return theme.colorScheme.outline;
}

// _StateBadge is the textual status pill in the detail header. Draft
// wins over the open/closed/merged state since a draft is always open.
class _StateBadge extends StatelessWidget {
  const _StateBadge({required this.pr});

  final GitPullRequest pr;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (String label, Color color) = pr.draft
        ? ('DRAFT', theme.colorScheme.outline)
        : switch (pr.state) {
            'merged' => ('MERGED', Colors.purple),
            'closed' => ('CLOSED', theme.colorScheme.error),
            _ => ('OPEN', Colors.green),
          };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        border: Border.all(color: color),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Text(
        label,
        style: theme.textTheme.labelSmall?.copyWith(
          color: color,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

// _ChecksSummary aggregates the check runs into a single headline row.
class _ChecksSummary extends StatelessWidget {
  const _ChecksSummary({required this.checks});

  final List<GitCheckRun> checks;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    var pending = 0;
    var passed = 0;
    var failed = 0;
    for (final c in checks) {
      if (c.passing) {
        passed++;
      } else if (c.failing) {
        failed++;
      } else {
        pending++;
      }
    }
    IconData icon;
    Color color;
    String label;
    if (pending > 0) {
      icon = Icons.circle_outlined;
      color = theme.colorScheme.outline;
      label =
          '$pending pending · $passed passed${failed > 0 ? " · $failed failed" : ""}';
    } else if (failed > 0) {
      icon = Icons.cancel_outlined;
      color = theme.colorScheme.error;
      label = '$failed failed · $passed passed';
    } else {
      icon = Icons.check_circle_outlined;
      color = Colors.green;
      label = 'All $passed passed';
    }
    return Row(
      children: [
        Icon(icon, size: 16, color: color),
        const SizedBox(width: 6),
        Expanded(
          child: Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(color: color),
          ),
        ),
      ],
    );
  }
}
