import 'package:flutter/material.dart';
import 'package:opendray/core/api/project_docs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

/// Renders the server-computed line-level diff of a proposal.
///
/// Reviewing used to mean reading the whole proposed document and spotting
/// by eye what moved — unworkable once a knowledge page runs to hundreds of
/// lines. This shows only changed lines plus a little context, and marks
/// each collapsed region so the reviewer can tell "nothing else changed"
/// from "the rest is hidden".
///
/// Unified (single column) rather than side-by-side: two columns of a long
/// document are unreadable on a phone. Mirrors the web ProposalDiff.
class ProposalDiffView extends StatelessWidget {
  const ProposalDiffView({required this.diff, super.key});

  final DocDiff diff;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (diff.unchanged || diff.hunks.isEmpty) {
      return Padding(
        padding: const EdgeInsets.all(12),
        child: Text(t.web.diff.noChanges, style: theme.textTheme.bodySmall),
      );
    }

    final rows = <Widget>[];
    for (final hunk in diff.hunks) {
      if (hunk.skippedBefore > 0) {
        rows.add(_CollapsedMarker(count: hunk.skippedBefore));
      }
      var lineNo = hunk.startLine;
      for (final line in hunk.lines) {
        // Removed lines don't exist in the new document, so they carry no
        // new-document line number.
        final n = line.kind == 'remove' ? null : lineNo++;
        rows.add(_DiffRow(line: line, lineNo: n));
      }
    }

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.only(bottom: 6),
          child: Row(
            children: [
              Text(
                t.web.diff.added(count: diff.added),
                style: theme.textTheme.bodySmall?.copyWith(
                  color: Colors.green.shade400,
                ),
              ),
              const SizedBox(width: 12),
              Text(
                t.web.diff.removed(count: diff.removed),
                style: theme.textTheme.bodySmall?.copyWith(
                  color: Colors.red.shade300,
                ),
              ),
            ],
          ),
        ),
        DecoratedBox(
          decoration: BoxDecoration(
            border: Border.all(color: theme.dividerColor),
            borderRadius: BorderRadius.circular(8),
          ),
          child: ClipRRect(
            borderRadius: BorderRadius.circular(8),
            child: Column(children: rows),
          ),
        ),
      ],
    );
  }
}

class _CollapsedMarker extends StatelessWidget {
  const _CollapsedMarker({required this.count});

  final int count;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      width: double.infinity,
      color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4),
      padding: const EdgeInsets.symmetric(vertical: 4, horizontal: 8),
      child: Text(
        t.web.diff.collapsed(count: count),
        textAlign: TextAlign.center,
        style: theme.textTheme.labelSmall?.copyWith(
          fontStyle: FontStyle.italic,
          color: theme.hintColor,
        ),
      ),
    );
  }
}

class _DiffRow extends StatelessWidget {
  const _DiffRow({required this.line, required this.lineNo});

  final DiffLine line;
  final int? lineNo;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (bg, fg, sign) = switch (line.kind) {
      'add' => (
        Colors.green.withValues(alpha: 0.12),
        Colors.green.shade300,
        '+',
      ),
      'remove' => (
        Colors.red.withValues(alpha: 0.12),
        Colors.red.shade300,
        '-',
      ),
      _ => (Colors.transparent, theme.textTheme.bodySmall?.color, ' '),
    };

    return Container(
      color: bg,
      padding: const EdgeInsets.symmetric(vertical: 1, horizontal: 4),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 28,
            child: Text(
              lineNo?.toString() ?? '',
              textAlign: TextAlign.right,
              style: theme.textTheme.labelSmall?.copyWith(
                fontFamily: 'monospace',
                color: theme.hintColor,
              ),
            ),
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              '$sign ${line.text}',
              style: theme.textTheme.bodySmall?.copyWith(
                fontFamily: 'monospace',
                fontSize: 11,
                color: fg,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
