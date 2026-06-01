import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';
import 'package:url_launcher/url_launcher.dart';

// PRDetailScreen is the full-screen surface opened when a PR row is
// tapped (any state). It mirrors the web detail drawer: a status
// badge, the description (the list endpoint omits body, so we re-fetch
// the single PR here), CI checks for every state, and — only while the
// PR is open — merge controls.
//
// Returns `true` via Navigator.pop when the PR was merged so the
// caller can refresh its list.
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

  List<GitCheckRun>? _checks;
  Object? _checksError;

  String _method = 'squash';
  bool _deleteBranch = true;
  bool _busy = false;

  Timer? _checksTimer;

  @override
  void initState() {
    super.initState();
    _pr = widget.pr;
    unawaited(_loadDetail());
    unawaited(_loadChecks());
    _checksTimer = Timer.periodic(
      const Duration(seconds: 30),
      (_) => unawaited(_loadChecks()),
    );
  }

  @override
  void dispose() {
    _checksTimer?.cancel();
    super.dispose();
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

  Future<void> _loadChecks() async {
    try {
      final c = await ref
          .read(gitApiProvider)
          .prChecks(dir: widget.cwd, number: widget.pr.number);
      if (!mounted) return;
      setState(() {
        _checks = c;
        _checksError = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _checksError = e);
    }
  }

  Future<void> _openInBrowser() async {
    final uri = Uri.tryParse(_pr.url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // Best-effort: a missing browser / malformed URL shouldn't crash
      // the screen. The host link is a convenience, not load-bearing.
    }
  }

  String _errorMessage(Object? e) {
    if (e is ApiException) return e.message;
    return '$e';
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
    return Scaffold(
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
      body: ListView(
        padding: const EdgeInsets.all(16),
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
            ],
          ),
          const SizedBox(height: 4),
          Text(
            '#${_pr.number} · ${_pr.author}',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
          const Divider(height: 32),
          Text('Description', style: theme.textTheme.labelLarge),
          const SizedBox(height: 8),
          _buildDescription(theme),
          const Divider(height: 32),
          Text('Checks', style: theme.textTheme.labelLarge),
          const SizedBox(height: 8),
          _buildChecks(theme),
          if (_pr.state == 'open') ...[
            const Divider(height: 32),
            Text('Merge', style: theme.textTheme.labelLarge),
            const SizedBox(height: 8),
            _buildMergeControls(theme),
          ],
        ],
      ),
    );
  }

  Widget _buildDescription(ThemeData theme) {
    if (_detailLoading && _pr.body.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 8),
        child: SizedBox(
          height: 18,
          child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
        ),
      );
    }
    if (_detailError != null && _pr.body.isEmpty) {
      return Text(
        "Couldn't load details: ${_errorMessage(_detailError)}",
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.error,
        ),
      );
    }
    if (_pr.body.trim().isEmpty) {
      return Text(
        'No description provided.',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
          fontStyle: FontStyle.italic,
        ),
      );
    }
    // Plain-text rendering preserving newlines. Mobile carries no
    // markdown renderer dependency (the web drawer renders markdown);
    // this keeps the description readable without pulling in a package.
    return SelectableText(_pr.body.trim(), style: theme.textTheme.bodyMedium);
  }

  Widget _buildChecks(ThemeData theme) {
    if (_checksError != null) {
      return Text(
        'Checks unavailable: ${_errorMessage(_checksError)}',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
        ),
      );
    }
    if (_checks == null) {
      return const SizedBox(
        height: 18,
        child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
      );
    }
    if (_checks!.isEmpty) {
      return Text(
        'No checks configured for this PR.',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
        ),
      );
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _ChecksSummary(checks: _checks!),
        const SizedBox(height: 8),
        for (final c in _checks!)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 2),
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

  Widget _buildMergeControls(ThemeData theme) {
    return Row(
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
        Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Checkbox(
              value: _deleteBranch,
              onChanged: _busy
                  ? null
                  : (v) => setState(() => _deleteBranch = v ?? false),
              visualDensity: VisualDensity.compact,
            ),
            Text('Delete branch', style: theme.textTheme.bodySmall),
          ],
        ),
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
    );
  }
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

// _ChecksSummary aggregates the check runs into a single headline row
// (moved here from git_pr_section.dart when the inline expand panel
// was replaced by this screen).
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
