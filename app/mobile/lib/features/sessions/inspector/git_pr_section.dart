import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';
import 'package:opendray/features/sessions/inspector/pr_detail_screen.dart';

// GitPRSection sits at the bottom of the status pane and exposes
// the PR command-center surface on mobile. Mirrors the web's
// PullRequestsSection: list open PRs, tap to expand for CI checks
// + merge button, top-bar button to create a new PR.
//
// Polls /git/prs every 60s while expanded; PR check polling
// happens on demand when a PR row expands.
class GitPRSection extends ConsumerStatefulWidget {
  const GitPRSection({required this.cwd, super.key});
  final String cwd;

  @override
  ConsumerState<GitPRSection> createState() => _GitPRSectionState();
}

class _GitPRSectionState extends ConsumerState<GitPRSection> {
  GitPullRequestList? _list;
  bool _loading = true;
  Object? _error;
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
    // Refresh every 60s while the widget is mounted. PR state
    // (open / merged / closed) changes infrequently enough that
    // a tighter cadence would just burn rate limit.
    _refreshTimer = Timer.periodic(
      const Duration(seconds: 60),
      (_) => unawaited(_load()),
    );
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  String _errorMessage() {
    final e = _error;
    if (e is ApiException) return e.message;
    return '$e';
  }

  Future<void> _load() async {
    try {
      final list = await ref
          .read(gitApiProvider)
          .listPullRequests(dir: widget.cwd);
      if (!mounted) return;
      setState(() {
        _list = list;
        _loading = false;
        _error = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e;
        _loading = false;
      });
    }
  }

  Future<void> _onCreate() async {
    final created = await showModalBottomSheet<GitPullRequest>(
      context: context,
      isScrollControlled: true,
      builder: (_) => Padding(
        padding: EdgeInsets.only(
          bottom: MediaQuery.of(context).viewInsets.bottom,
        ),
        child: _CreatePRSheet(cwd: widget.cwd),
      ),
    );
    if (created != null && mounted) {
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('PR #${created.number} opened')));
      await _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Icon(
                Icons.merge_outlined,
                size: 16,
                color: theme.colorScheme.outline,
              ),
              const SizedBox(width: 6),
              Text(
                'Pull requests',
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
              const Spacer(),
              if (_list != null && !_list!.needsToken)
                TextButton.icon(
                  onPressed: _onCreate,
                  icon: const Icon(Icons.add, size: 16),
                  label: const Text('Create'),
                ),
            ],
          ),
          if (_loading)
            const Padding(
              padding: EdgeInsets.symmetric(vertical: 6),
              child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
            )
          else if (_error != null)
            Text(
              'Error: ${_errorMessage()}',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.error,
              ),
            )
          else if (_list?.needsToken ?? false)
            _NeedTokenHint(host: _list!.host)
          else if ((_list?.errorMessage ?? '').isNotEmpty)
            Text(
              _list!.errorMessage,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.error,
              ),
            )
          else if ((_list?.prs ?? []).isEmpty)
            Text(
              'No open PRs.',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            )
          else
            for (final pr in _list!.prs)
              _PRRow(pr: pr, onTap: () => unawaited(_openDetail(pr))),
        ],
      ),
    );
  }

  Future<void> _openDetail(GitPullRequest pr) async {
    final merged = await Navigator.of(context).push<bool>(
      MaterialPageRoute(
        builder: (_) => PRDetailScreen(cwd: widget.cwd, pr: pr),
      ),
    );
    // The detail screen pops `true` after a successful merge; refresh
    // the list so the merged PR drops out of the open view.
    if ((merged ?? false) && mounted) {
      await _load();
    }
  }
}

class _PRRow extends StatelessWidget {
  const _PRRow({required this.pr, required this.onTap});

  final GitPullRequest pr;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final stateColor = pr.draft
        ? theme.colorScheme.outline
        : switch (pr.state) {
            'merged' => Colors.purple,
            'closed' => theme.colorScheme.error,
            _ => Colors.green,
          };
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 4),
        child: Row(
          children: [
            Icon(
              pr.draft ? Icons.merge_outlined : Icons.merge,
              size: 14,
              color: stateColor,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    pr.title,
                    style: theme.textTheme.bodyMedium,
                    overflow: TextOverflow.ellipsis,
                  ),
                  Text(
                    '#${pr.number} · ${pr.author} · ${pr.head} → ${pr.base}',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                      fontFamily: 'monospace',
                      fontSize: 10,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ],
              ),
            ),
            Icon(
              Icons.chevron_right,
              size: 16,
              color: theme.colorScheme.outline,
            ),
          ],
        ),
      ),
    );
  }
}

class _CreatePRSheet extends ConsumerStatefulWidget {
  const _CreatePRSheet({required this.cwd});
  final String cwd;
  @override
  ConsumerState<_CreatePRSheet> createState() => _CreatePRSheetState();
}

class _CreatePRSheetState extends ConsumerState<_CreatePRSheet> {
  final _title = TextEditingController();
  final _head = TextEditingController();
  final _body = TextEditingController();
  bool _draft = false;
  bool _busy = false;

  @override
  void dispose() {
    _title.dispose();
    _head.dispose();
    _body.dispose();
    super.dispose();
  }

  Future<void> _submit() async {
    final title = _title.text.trim();
    final head = _head.text.trim();
    if (title.isEmpty || head.isEmpty) return;
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    final errorColor = Theme.of(context).colorScheme.error;
    try {
      final pr = await ref
          .read(gitApiProvider)
          .createPullRequest(
            dir: widget.cwd,
            title: title,
            head: head,
            body: _body.text.trim().isEmpty ? null : _body.text.trim(),
            draft: _draft,
          );
      if (!mounted) return;
      Navigator.of(context).pop(pr);
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Create PR failed: ${e.message}'),
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
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Row(
              children: [
                Text('Create pull request', style: theme.textTheme.titleMedium),
                const Spacer(),
                IconButton(
                  icon: const Icon(Icons.close),
                  onPressed: _busy ? null : () => Navigator.of(context).pop(),
                ),
              ],
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _title,
              enabled: !_busy,
              decoration: const InputDecoration(
                labelText: 'Title',
                border: OutlineInputBorder(),
                isDense: true,
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _head,
              enabled: !_busy,
              decoration: const InputDecoration(
                labelText: 'Source branch (head)',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              style: const TextStyle(fontFamily: 'monospace'),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _body,
              enabled: !_busy,
              decoration: const InputDecoration(
                labelText: 'Description (optional)',
                border: OutlineInputBorder(),
                isDense: true,
              ),
              minLines: 3,
              maxLines: 6,
            ),
            const SizedBox(height: 6),
            Row(
              children: [
                Checkbox(
                  value: _draft,
                  onChanged: _busy
                      ? null
                      : (v) => setState(() => _draft = v ?? false),
                  visualDensity: VisualDensity.compact,
                ),
                Text('Draft', style: theme.textTheme.bodySmall),
                const Spacer(),
                FilledButton(
                  onPressed: _busy ? null : _submit,
                  child: _busy
                      ? const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Text('Create PR'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}

class _NeedTokenHint extends StatelessWidget {
  const _NeedTokenHint({required this.host});
  final String host;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: theme.dividerColor, style: BorderStyle.solid),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.key_outlined, size: 16, color: theme.colorScheme.outline),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              'No token configured for $host. Add one in Plugins → Git hosts.',
              style: theme.textTheme.bodySmall,
            ),
          ),
        ],
      ),
    );
  }
}
