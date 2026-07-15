import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';

// Pre-launch options for running one plan step — account (for claude/
// antigravity multi-account) + a bypass/YOLO toggle. Mirrors the web
// RunStepDialog and the handoff/spawn sheets. Returns the chosen options, or
// null if cancelled.
class RunStepOptions {
  const RunStepOptions({required this.accountId, required this.args});
  final String accountId;
  final List<String> args;
}

const _bypassFlags = <String, List<String>>{
  'claude': ['--dangerously-skip-permissions'],
  'codex': ['--dangerously-bypass-approvals-and-sandbox'],
  'antigravity': ['--dangerously-skip-permissions'],
  'opencode': ['--dangerously-skip-permissions'],
  'grok': ['--always-approve'],
};

class RunStepOptionsSheet extends ConsumerStatefulWidget {
  const RunStepOptionsSheet({required this.step, required this.index, super.key});
  final PlanStep step;
  final int index;

  static Future<RunStepOptions?> show(
      BuildContext context, PlanStep step, int index) {
    return showModalBottomSheet<RunStepOptions>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => RunStepOptionsSheet(step: step, index: index),
    );
  }

  @override
  ConsumerState<RunStepOptionsSheet> createState() =>
      _RunStepOptionsSheetState();
}

class _RunStepOptionsSheetState extends ConsumerState<RunStepOptionsSheet> {
  String _accountId = '';
  bool _bypass = false;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final p = widget.step.assignee;
    final isClaude = p == 'claude';
    final isAgy = p == 'antigravity';
    final accounts = <({String id, String label, bool usable})>[];
    if (isClaude) {
      final list = ref.watch(claudeAccountsListProvider).asData?.value;
      if (list != null) {
        for (final a in list) {
          if (a.enabled) {
            accounts.add((id: a.id, label: a.displayName, usable: a.isUsable));
          }
        }
      }
    } else if (isAgy) {
      final list = ref.watch(antigravityAccountsListProvider).asData?.value;
      if (list != null) {
        for (final a in list) {
          if (a.enabled) {
            accounts.add((id: a.id, label: a.displayName, usable: a.isUsable));
          }
        }
      }
    }
    final showAccount = (isClaude || isAgy) && accounts.isNotEmpty;

    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                BrandAvatar(providerId: p, size: 22),
                const SizedBox(width: 8),
                Text('${t.web.roundTable.plan.runTitle} · ${widget.index + 1}',
                    style: theme.textTheme.titleMedium),
              ],
            ),
            const SizedBox(height: 4),
            Text(widget.step.task,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.bodySmall),
            const SizedBox(height: 16),
            if (showAccount) ...[
              Text(t.web.roundTable.plan.account,
                  style: theme.textTheme.labelLarge),
              const SizedBox(height: 6),
              DropdownButtonFormField<String>(
                initialValue: _accountId,
                isExpanded: true,
                decoration: const InputDecoration(
                    isDense: true, border: OutlineInputBorder()),
                items: [
                  DropdownMenuItem(
                      value: '',
                      child: Text(t.web.roundTable.plan.accountDefault)),
                  for (final a in accounts)
                    DropdownMenuItem(
                        value: a.id,
                        enabled: a.usable,
                        child: Text(a.label)),
                ],
                onChanged: (v) => setState(() => _accountId = v ?? ''),
              ),
              const SizedBox(height: 12),
            ],
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              value: _bypass,
              onChanged: (v) => setState(() => _bypass = v),
              title: Text(t.web.roundTable.plan.bypass),
              subtitle: Text(t.web.roundTable.plan.bypassHint),
            ),
            const SizedBox(height: 8),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: () => Navigator.of(context).pop(RunStepOptions(
                  accountId: _accountId,
                  args: _bypass ? (_bypassFlags[p] ?? const []) : const [],
                )),
                icon: const Icon(Icons.play_arrow, size: 18),
                label: Text(t.web.roundTable.plan.runStep),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
