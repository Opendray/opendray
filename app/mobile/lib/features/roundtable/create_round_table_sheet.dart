import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/roundtable/seat_tile.dart';
import 'package:opendray/features/sessions/directory_picker_sheet.dart';

// Create a Round Table — mobile parity with
// app/web/src/components/roundtable/CreateRoundTableDialog.tsx. Pick members
// (seat toggles), each with an optional model, account (claude/antigravity)
// and persona; optionally bind a project. No topic — the chat names itself
// from the first message.
class CreateRoundTableSheet extends ConsumerStatefulWidget {
  const CreateRoundTableSheet({super.key});

  static Future<RoundTable?> show(BuildContext context) {
    return showModalBottomSheet<RoundTable>(
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
      builder: (_) => const CreateRoundTableSheet(),
    );
  }

  @override
  ConsumerState<CreateRoundTableSheet> createState() =>
      _CreateRoundTableSheetState();
}

class _CreateRoundTableSheetState extends ConsumerState<CreateRoundTableSheet> {
  // Seat the three vendors that work out of the box; grok/opencode are toggles
  // (off initially — they need their own host login/config).
  final Set<String> _seats = {'claude', 'codex', 'antigravity'};
  final Map<String, String> _models = {...seatModelDefault};
  final Map<String, String> _accounts = {};
  final Map<String, String> _personas = {};
  String _cwd = '';
  final _framing = TextEditingController();
  bool _submitting = false;

  @override
  void dispose() {
    _framing.dispose();
    super.dispose();
  }

  bool get _canSubmit => _seats.isNotEmpty && !_submitting;

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
      final seats = _seats.map((p) {
        return Seat(
          provider: p,
          model: (_models[p] ?? '').trim(),
          accountId:
              seatSupportsAccount.contains(p) ? (_accounts[p] ?? '').trim() : '',
          persona: (_personas[p] ?? '').trim(),
        );
      }).toList();
      final rt = await ref.read(roundtableApiProvider).create(
            cwd: _cwd.trim().isEmpty ? null : _cwd.trim(),
            framing: _framing.text.trim(),
            seats: seats,
          );
      if (mounted) Navigator.of(context).pop(rt);
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
    final modelsAsync = ref.watch(seatModelsProvider);
    final claudeAccounts = ref.watch(claudeAccountsListProvider).asData?.value
            .where((a) => a.enabled)
            .toList() ??
        const [];
    final agyAccounts = ref.watch(antigravityAccountsListProvider).asData?.value
            .where((a) => a.enabled)
            .toList() ??
        const [];

    List<({String id, String label, bool usable})> accountsFor(String p) {
      if (p == 'claude') {
        return claudeAccounts
            .map((a) => (id: a.id, label: a.displayName, usable: a.isUsable))
            .toList();
      }
      if (p == 'antigravity') {
        return agyAccounts
            .map((a) => (id: a.id, label: a.displayName, usable: a.isUsable))
            .toList();
      }
      return const [];
    }

    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Row(
              children: [
                Text(t.web.roundTable.dialog.title,
                    style: theme.textTheme.titleMedium),
                const Spacer(),
                IconButton(
                  onPressed: () => Navigator.of(context).pop(),
                  icon: const Icon(Icons.close),
                ),
              ],
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 16),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(
                t.web.roundTable.dialog.description,
                style: theme.textTheme.bodySmall,
              ),
            ),
          ),
          Flexible(
            child: ListView(
              shrinkWrap: true,
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
              children: [
                Text(t.web.roundTable.dialog.seats,
                    style: theme.textTheme.labelLarge),
                const SizedBox(height: 8),
                for (final p in seatProviders)
                  SeatTile(
                    provider: p,
                    on: _seats.contains(p),
                    model: _models[p] ?? '',
                    account: _accounts[p] ?? '',
                    persona: _personas[p] ?? '',
                    modelOptions: modelsAsync.asData?.value[p] ?? const [],
                    accounts: accountsFor(p),
                    onToggle: () => setState(() {
                      if (_seats.contains(p)) {
                        _seats.remove(p);
                      } else {
                        _seats.add(p);
                      }
                    }),
                    onModel: (v) => setState(() => _models[p] = v),
                    onAccount: (v) => setState(() => _accounts[p] = v),
                    onPersona: (v) => _personas[p] = v,
                  ),
                const SizedBox(height: 12),
                Text(t.web.roundTable.dialog.project,
                    style: theme.textTheme.labelLarge),
                const SizedBox(height: 6),
                OutlinedButton.icon(
                  onPressed: _pickProject,
                  icon: const Icon(Icons.folder_outlined, size: 18),
                  label: Text(
                    _cwd.isEmpty ? t.web.roundTable.dialog.browse : _cwd,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Padding(
                  padding: const EdgeInsets.only(top: 4),
                  child: Text(t.web.roundTable.dialog.cwdHint,
                      style: theme.textTheme.bodySmall),
                ),
                const SizedBox(height: 12),
                Text(t.web.roundTable.dialog.framing,
                    style: theme.textTheme.labelLarge),
                const SizedBox(height: 6),
                TextField(
                  controller: _framing,
                  minLines: 2,
                  maxLines: 4,
                  decoration: InputDecoration(
                    hintText: t.web.roundTable.dialog.framingPlaceholder,
                    isDense: true,
                    border: const OutlineInputBorder(),
                  ),
                ),
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
                      : Text(t.web.roundTable.dialog.start),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
