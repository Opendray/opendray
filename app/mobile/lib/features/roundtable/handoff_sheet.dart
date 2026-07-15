import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/providers_api.dart';
import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';
import 'package:opendray/features/sessions/directory_picker_sheet.dart';

// Per-provider "bypass" flags — mirror app/web SpawnDialog + the mobile
// spawn sheet. Starting the executor with these skips approval prompts.
const _bypassFlags = <String, List<String>>{
  'claude': ['--dangerously-skip-permissions'],
  'codex': ['--dangerously-bypass-approvals-and-sandbox'],
  'antigravity': ['--dangerously-skip-permissions'],
  'opencode': ['--dangerously-skip-permissions'],
  'grok': ['--always-approve'],
};

// Hand a Round Table discussion off to a real agent session — mobile parity
// with app/web/src/components/roundtable/HandoffDialog.tsx. Pick any executor
// provider (+ claude account), a project, optional bypass; reuse the prior
// session when one is still alive. Returns the spawned session id.
class HandoffSheet extends ConsumerStatefulWidget {
  const HandoffSheet({required this.rt, super.key});
  final RoundTable rt;

  static Future<String?> show(BuildContext context, RoundTable rt) {
    return showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      constraints: BoxConstraints(
        maxHeight: MediaQuery.of(context).size.height * 0.92,
      ),
      builder: (_) => HandoffSheet(rt: rt),
    );
  }

  @override
  ConsumerState<HandoffSheet> createState() => _HandoffSheetState();
}

class _HandoffSheetState extends ConsumerState<HandoffSheet> {
  String _provider = '';
  String _accountId = '';
  late String _cwd = widget.rt.cwd;
  bool _bypass = false;
  late bool _continueExisting = widget.rt.resultingSessionId.isNotEmpty;
  bool _submitting = false;

  bool get _canSubmit =>
      _provider.isNotEmpty && _cwd.trim().isNotEmpty && !_submitting;

  bool get _isNewSession =>
      widget.rt.resultingSessionId.isEmpty || !_continueExisting;

  Future<void> _pickProject() async {
    final picked = await DirectoryPickerSheet.show(
      context,
      initialPath: _cwd.isNotEmpty ? _cwd : null,
    );
    if (picked != null) setState(() => _cwd = picked);
  }

  Future<void> _submit() async {
    setState(() => _submitting = true);
    try {
      final args = (_bypass && _isNewSession)
          ? (_bypassFlags[_provider] ?? const <String>[])
          : const <String>[];
      final sessionId = await ref.read(roundtableApiProvider).handoff(
            widget.rt.id,
            provider: _provider,
            cwd: _cwd.trim(),
            accountId: _provider == 'claude' ? _accountId : '',
            forceNew: !_continueExisting,
            args: args,
          );
      if (mounted) Navigator.of(context).pop(sessionId);
    } on Object catch (e) {
      if (mounted) {
        setState(() => _submitting = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final providers = ref.watch(providersListProvider).asData?.value
            .where((p) => p.enabled)
            .toList() ??
        const [];
    final claudeAccounts = ref.watch(claudeAccountsListProvider).asData?.value
            .where((a) => a.enabled)
            .toList() ??
        const [];
    final hasPrior = widget.rt.resultingSessionId.isNotEmpty;

    return Padding(
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Row(
              children: [
                Text(t.web.roundTable.handoff.title,
                    style: theme.textTheme.titleMedium),
                const Spacer(),
                IconButton(
                  onPressed: () => Navigator.of(context).pop(),
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
          ),
          Flexible(
            child: ListView(
              shrinkWrap: true,
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
              children: [
                Text(t.web.roundTable.handoff.description,
                    style: theme.textTheme.bodySmall),
                const SizedBox(height: 12),
                Text(t.web.roundTable.handoff.executor,
                    style: theme.textTheme.labelLarge),
                const SizedBox(height: 6),
                Wrap(
                  spacing: 8,
                  runSpacing: 8,
                  children: [
                    for (final p in providers)
                      ChoiceChip(
                        selected: _provider == p.id,
                        avatar: BrandAvatar(providerId: p.id, size: 18),
                        label: Text(p.name),
                        onSelected: (_) => setState(() {
                          _provider = p.id;
                          _accountId = '';
                          _bypass = false;
                        }),
                      ),
                  ],
                ),
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Text(t.web.roundTable.handoff.executorHint,
                      style: theme.textTheme.bodySmall),
                ),
                // Claude account picker.
                if (_provider == 'claude' && claudeAccounts.isNotEmpty) ...[
                  const SizedBox(height: 12),
                  Text(t.web.roundTable.handoff.claudeAccount,
                      style: theme.textTheme.labelLarge),
                  const SizedBox(height: 6),
                  DropdownButtonFormField<String>(
                    initialValue: _accountId,
                    isExpanded: true,
                    decoration: const InputDecoration(
                      isDense: true,
                      border: OutlineInputBorder(),
                    ),
                    items: [
                      DropdownMenuItem(
                        value: '',
                        child: Text(t.web.roundTable.handoff.accountDefault),
                      ),
                      for (final a in claudeAccounts)
                        DropdownMenuItem(
                          value: a.id,
                          enabled: a.isUsable,
                          child: Text(a.displayName),
                        ),
                    ],
                    onChanged: (v) => setState(() => _accountId = v ?? ''),
                  ),
                ],
                const SizedBox(height: 12),
                Text(t.web.roundTable.handoff.project,
                    style: theme.textTheme.labelLarge),
                const SizedBox(height: 6),
                OutlinedButton.icon(
                  onPressed: _pickProject,
                  icon: const Icon(Icons.folder_outlined, size: 18),
                  label: Text(
                    _cwd.isEmpty
                        ? t.web.roundTable.handoff.projectPlaceholder
                        : _cwd,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Text(t.web.roundTable.handoff.projectHint,
                      style: theme.textTheme.bodySmall),
                ),
                // Continue-in-existing toggle (only when a prior session exists).
                if (hasPrior) ...[
                  const SizedBox(height: 8),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: _continueExisting,
                    onChanged: (v) => setState(() => _continueExisting = v),
                    title: Text(t.web.roundTable.handoff.continueLabel),
                    subtitle: Text(t.web.roundTable.handoff.continueHint),
                  ),
                ],
                // Bypass toggle — only for a fresh session.
                if (_provider.isNotEmpty && _isNewSession) ...[
                  const SizedBox(height: 4),
                  SwitchListTile(
                    contentPadding: EdgeInsets.zero,
                    value: _bypass,
                    onChanged: (v) => setState(() => _bypass = v),
                    title: Text(t.web.roundTable.handoff.bypassLabel),
                    subtitle: Text(t.web.roundTable.handoff.bypassHint),
                  ),
                ],
              ],
            ),
          ),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
              child: SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: _canSubmit ? _submit : null,
                  child: _submitting
                      ? const SizedBox(
                          height: 18,
                          width: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Text(_continueExisting && hasPrior
                          ? t.web.roundTable.handoff.runContinue
                          : t.web.roundTable.handoff.run),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
