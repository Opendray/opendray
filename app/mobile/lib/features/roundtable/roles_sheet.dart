import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/roundtable/seat_tile.dart';

// Live members + roles editor — add or remove members and reassign each seat's
// model/account/persona plus the shared framing directive on an active round
// table as the topic evolves. Mobile parity with app/web RolesDialog; the
// backend re-reads seats each reply, so a member added here is @mentionable on
// the next turn and a removed one stops replying (its past messages stay).
class RolesSheet extends ConsumerStatefulWidget {
  const RolesSheet({required this.rt, super.key});
  final RoundTable rt;

  static Future<bool?> show(BuildContext context, RoundTable rt) {
    return showModalBottomSheet<bool>(
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
      builder: (_) => RolesSheet(rt: rt),
    );
  }

  @override
  ConsumerState<RolesSheet> createState() => _RolesSheetState();
}

class _RolesSheetState extends ConsumerState<RolesSheet> {
  late final TextEditingController _framing =
      TextEditingController(text: widget.rt.framing);
  // Editable seat state, seeded from the table's current seats.
  late final Set<String> _seats = {
    for (final s in widget.rt.seats) s.provider,
  };
  late final Map<String, String> _models = {
    for (final s in widget.rt.seats) s.provider: s.model,
  };
  late final Map<String, String> _accounts = {
    for (final s in widget.rt.seats) s.provider: s.accountId,
  };
  late final Map<String, String> _personas = {
    for (final s in widget.rt.seats) s.provider: s.persona,
  };
  bool _saving = false;

  @override
  void dispose() {
    _framing.dispose();
    super.dispose();
  }

  bool get _canSave => _seats.isNotEmpty && !_saving;

  void _toggle(String p) {
    setState(() {
      if (_seats.contains(p)) {
        _seats.remove(p);
      } else {
        _seats.add(p);
        // Pre-seed the default model (e.g. codex → gpt-5.4-mini) the first time.
        _models.putIfAbsent(p, () => seatModelDefault[p] ?? '');
      }
    });
  }

  Future<void> _save() async {
    setState(() => _saving = true);
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
      await ref.read(roundtableApiProvider).update(
            widget.rt.id,
            seats: seats,
            framing: _framing.text.trim(),
          );
      if (mounted) Navigator.of(context).pop(true);
    } on Object catch (e) {
      if (mounted) {
        setState(() => _saving = false);
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
      padding: EdgeInsets.only(bottom: MediaQuery.of(context).viewInsets.bottom),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 16, 16, 4),
            child: Row(
              children: [
                Text(t.web.roundTable.detail.rolesTitle,
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
              child: Text(t.web.roundTable.detail.rolesHint,
                  style: theme.textTheme.bodySmall),
            ),
          ),
          Flexible(
            child: ListView(
              shrinkWrap: true,
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
              children: [
                Text(t.web.roundTable.detail.rolesFraming,
                    style: theme.textTheme.labelLarge),
                const SizedBox(height: 6),
                TextField(
                  controller: _framing,
                  minLines: 2,
                  maxLines: 5,
                  decoration: InputDecoration(
                    hintText: t.web.roundTable.dialog.framingPlaceholder,
                    isDense: true,
                    border: const OutlineInputBorder(),
                  ),
                ),
                const SizedBox(height: 16),
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
                    onToggle: () => _toggle(p),
                    onModel: (v) => setState(() => _models[p] = v),
                    onAccount: (v) => setState(() => _accounts[p] = v),
                    onPersona: (v) => _personas[p] = v,
                  ),
                if (_seats.isEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 4),
                    child: Text(
                      t.web.roundTable.detail.membersMin,
                      style: theme.textTheme.bodySmall
                          ?.copyWith(color: theme.colorScheme.error),
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
                  onPressed: _canSave ? _save : null,
                  child: _saving
                      ? const SizedBox(
                          height: 18,
                          width: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : Text(t.web.roundTable.detail.rolesSave),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }
}
