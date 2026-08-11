import 'package:flutter/material.dart';
import 'package:opendray/core/api/project_docs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/cortex/proposal_diff.dart';

/// Full-screen review of one pending proposal.
///
/// The diff used to live inside the amber banner on the page itself: a
/// 240px window, on a tinted background that swallowed the red/green, with
/// approve and reject sharing a row that overflowed on a phone — so the
/// reject button was pushed off-screen entirely and the only visible
/// action was Approve.
///
/// A decision you cannot read is not a decision. This gives the diff the
/// whole screen on a normal surface, and puts both actions in a footer
/// that cannot be squeezed out.
class ProposalReviewScreen extends StatelessWidget {
  const ProposalReviewScreen({
    required this.pageTitle,
    required this.proposal,
    required this.busy,
    required this.onDecide,
    super.key,
  });

  final String pageTitle;
  final DocProposal proposal;
  final bool busy;

  /// Called with true to approve, false to reject. The caller owns the
  /// API call and closes this screen.
  final void Function({required bool approve}) onDecide;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final diff = proposal.diff;

    return Scaffold(
      appBar: AppBar(
        title: Text(pageTitle),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(28),
          child: Padding(
            padding: const EdgeInsets.only(left: 16, right: 16, bottom: 8),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(
                t.web.knowledge.kb.proposal.text,
                style: theme.textTheme.bodySmall,
              ),
            ),
          ),
        ),
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(12, 12, 12, 24),
        children: [
          if (proposal.reason.trim().isNotEmpty) ...[
            Card(
              margin: EdgeInsets.zero,
              child: Padding(
                padding: const EdgeInsets.all(12),
                child: Text(proposal.reason, style: theme.textTheme.bodySmall),
              ),
            ),
            const SizedBox(height: 12),
          ],
          // Proposals filed before the server attached diffs fall back to
          // the full body — still reviewable, just not as a diff.
          if (diff != null)
            ProposalDiffView(diff: diff)
          else
            SelectableText(
              proposal.proposedContent,
              style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
            ),
        ],
      ),
      // A footer, not a row inside the content: on a narrow screen the
      // previous layout pushed reject off the edge.
      bottomNavigationBar: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
          child: Row(
            children: [
              Expanded(
                child: OutlinedButton(
                  onPressed: busy ? null : () => onDecide(approve: false),
                  child: Text(t.web.knowledge.kb.proposal.reject),
                ),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: FilledButton(
                  onPressed: busy ? null : () => onDecide(approve: true),
                  child: Text(t.web.knowledge.kb.proposal.approve),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
