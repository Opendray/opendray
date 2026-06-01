import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';
import 'package:opendray/features/sessions/inspector/issue_detail_screen.dart';

// GitIssueSection sits directly below GitPRSection in the Git tab. It
// mirrors the PR surface but read-only: list issues for the repo, tap a
// row to open the detail screen (description + labels + comments). There
// is no create / close action — issues are view-only here.
//
// Polls /git/issues every 60s while mounted. GitHub/Gitea expose issues
// and PRs through one endpoint; the backend filters PRs out, so this
// list is issues only.
class GitIssueSection extends ConsumerStatefulWidget {
  const GitIssueSection({required this.cwd, super.key});
  final String cwd;

  @override
  ConsumerState<GitIssueSection> createState() => _GitIssueSectionState();
}

class _GitIssueSectionState extends ConsumerState<GitIssueSection> {
  GitIssueList? _list;
  bool _loading = true;
  Object? _error;
  Timer? _refreshTimer;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
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
      final list = await ref.read(gitApiProvider).listIssues(dir: widget.cwd);
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

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 0, 12, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Icon(
                Icons.adjust_outlined,
                size: 16,
                color: theme.colorScheme.outline,
              ),
              const SizedBox(width: 6),
              Text(
                'Issues',
                style: theme.textTheme.labelMedium?.copyWith(
                  color: theme.colorScheme.outline,
                ),
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
          else if ((_list?.issues ?? []).isEmpty)
            Text(
              'No open issues.',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            )
          else
            for (final issue in _list!.issues)
              _IssueRow(issue: issue, onTap: () => unawaited(_openDetail(issue))),
        ],
      ),
    );
  }

  Future<void> _openDetail(GitIssue issue) async {
    await Navigator.of(context).push<void>(
      MaterialPageRoute(
        builder: (_) => IssueDetailScreen(cwd: widget.cwd, issue: issue),
      ),
    );
  }
}

class _IssueRow extends StatelessWidget {
  const _IssueRow({required this.issue, required this.onTap});

  final GitIssue issue;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final closed = issue.state == 'closed';
    final stateColor = closed ? Colors.purple : Colors.green;
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 4),
        child: Row(
          children: [
            Icon(
              closed ? Icons.check_circle_outline : Icons.error_outline,
              size: 14,
              color: stateColor,
            ),
            const SizedBox(width: 8),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    issue.title,
                    style: theme.textTheme.bodyMedium,
                    overflow: TextOverflow.ellipsis,
                  ),
                  Text(
                    '#${issue.number} · ${issue.author}',
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                      fontFamily: 'monospace',
                      fontSize: 10,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                  if (issue.labels.isNotEmpty) ...[
                    const SizedBox(height: 4),
                    IssueLabelChips(labels: issue.labels),
                  ],
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

// IssueLabelChips renders an issue's labels as small pills. GitHub/Gitea
// supply a hex colour (no leading '#'); GitLab leaves it empty, in which
// case the chip falls back to the theme outline. Shared between the row
// and the detail header.
class IssueLabelChips extends StatelessWidget {
  const IssueLabelChips({required this.labels, super.key});
  final List<GitLabel> labels;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Wrap(
      spacing: 4,
      runSpacing: 4,
      children: [
        for (final l in labels)
          () {
            final color = _parseHex(l.color) ?? theme.colorScheme.outline;
            return Container(
              padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
              decoration: BoxDecoration(
                border: Border.all(color: color),
                borderRadius: BorderRadius.circular(8),
                color: color.withValues(alpha: 0.1),
              ),
              child: Text(
                l.name,
                style: theme.textTheme.labelSmall?.copyWith(
                  color: color,
                  fontSize: 9,
                ),
              ),
            );
          }(),
      ],
    );
  }

  // _parseHex turns a 6-digit hex string (no leading '#') into a Color.
  // Returns null for empty / malformed input so the caller falls back.
  static Color? _parseHex(String hex) {
    if (hex.length != 6) return null;
    final v = int.tryParse(hex, radix: 16);
    if (v == null) return null;
    return Color(0xFF000000 | v);
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
