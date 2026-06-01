import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';
import 'package:opendray/features/sessions/inspector/git_issue_section.dart';
import 'package:url_launcher/url_launcher.dart';

// IssueDetailScreen is the full-screen issue view opened when a row is
// tapped. Read-only: a single scrolling pane with the description
// followed by the comment thread. Mirrors PRDetailScreen minus the
// tabs and merge footer (issues have no commits / checks / files /
// merge).
class IssueDetailScreen extends ConsumerStatefulWidget {
  const IssueDetailScreen({required this.cwd, required this.issue, super.key});

  final String cwd;
  final GitIssue issue;

  @override
  ConsumerState<IssueDetailScreen> createState() => _IssueDetailScreenState();
}

class _IssueDetailScreenState extends ConsumerState<IssueDetailScreen> {
  // Seeded from the list row so the header paints instantly; replaced by
  // the detail fetch (which carries the body) once it resolves.
  late GitIssue _issue;
  bool _detailLoading = true;
  Object? _detailError;

  List<GitPRComment>? _comments;
  Object? _commentsError;

  @override
  void initState() {
    super.initState();
    _issue = widget.issue;
    unawaited(_loadDetail());
    unawaited(_loadComments());
  }

  Future<void> _loadDetail() async {
    try {
      final full = await ref
          .read(gitApiProvider)
          .getIssue(dir: widget.cwd, number: widget.issue.number);
      if (!mounted) return;
      setState(() {
        _issue = full;
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

  Future<void> _loadComments() async {
    try {
      final c = await ref
          .read(gitApiProvider)
          .issueComments(dir: widget.cwd, number: widget.issue.number);
      if (!mounted) return;
      setState(() {
        _comments = c;
        _commentsError = null;
      });
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _commentsError = e);
    }
  }

  Future<void> _openInBrowser() async {
    final uri = Uri.tryParse(_issue.url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // Best-effort convenience link; never crash the screen.
    }
  }

  String _errMsg(Object? e) {
    if (e is ApiException) return e.message;
    return '$e';
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final comments = _comments ?? const <GitPRComment>[];
    return Scaffold(
      appBar: AppBar(
        title: Text('Issue #${_issue.number}'),
        actions: [
          if (_issue.url.isNotEmpty)
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
          // Header: title + state + labels
          Text(_issue.title, style: theme.textTheme.titleMedium),
          const SizedBox(height: 8),
          Wrap(
            spacing: 8,
            runSpacing: 4,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              _StateBadge(state: _issue.state),
              Text(
                '#${_issue.number} · ${_issue.author}',
                style: theme.textTheme.bodySmall?.copyWith(
                  color: theme.colorScheme.outline,
                ),
              ),
            ],
          ),
          if (_issue.labels.isNotEmpty) ...[
            const SizedBox(height: 8),
            IssueLabelChips(labels: _issue.labels),
          ],
          const SizedBox(height: 16),
          // Description
          _CommentCard(
            author: _issue.author,
            when: '',
            child: _descriptionBody(theme),
          ),
          const SizedBox(height: 8),
          // Comments
          if (_commentsError != null)
            Text(
              'Comments unavailable: ${_errMsg(_commentsError)}',
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
                child: SelectableText(c.body, style: theme.textTheme.bodyMedium),
              ),
              const SizedBox(height: 8),
            ],
        ],
      ),
    );
  }

  Widget _descriptionBody(ThemeData theme) {
    if (_detailLoading && _issue.body.isEmpty) {
      return const Padding(
        padding: EdgeInsets.symmetric(vertical: 4),
        child: SizedBox(
          height: 16,
          child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
        ),
      );
    }
    if (_detailError != null && _issue.body.isEmpty) {
      return Text(
        "Couldn't load details: ${_errMsg(_detailError)}",
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.error,
        ),
      );
    }
    if (_issue.body.trim().isEmpty) {
      return Text(
        'No description provided.',
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.outline,
          fontStyle: FontStyle.italic,
        ),
      );
    }
    return SelectableText(_issue.body.trim(), style: theme.textTheme.bodyMedium);
  }
}

// _CommentCard is the bordered card used for both the description and
// each comment. Mirrors the PR detail screen's card.
class _CommentCard extends StatelessWidget {
  const _CommentCard({
    required this.author,
    required this.when,
    required this.child,
  });

  final String author;
  final String when;
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

// _StateBadge is the textual status pill in the detail header.
class _StateBadge extends StatelessWidget {
  const _StateBadge({required this.state});

  final String state;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (String label, Color color) = state == 'closed'
        ? ('CLOSED', Colors.purple)
        : ('OPEN', Colors.green);
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
