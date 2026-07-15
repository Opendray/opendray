import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';
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
                  _SeatTile(
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

// Persona field + quick-fill role preset chips. The localized preset label is
// the persona text sent to the model (models handle any language).
class _PersonaField extends StatefulWidget {
  const _PersonaField({required this.persona, required this.onPersona});
  final String persona;
  final ValueChanged<String> onPersona;

  @override
  State<_PersonaField> createState() => _PersonaFieldState();
}

class _PersonaFieldState extends State<_PersonaField> {
  late final TextEditingController _ctrl =
      TextEditingController(text: widget.persona);

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  void _apply(String value) {
    _ctrl.text = value;
    _ctrl.selection = TextSelection.collapsed(offset: value.length);
    widget.onPersona(value);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final presets = <String>[
      t.web.roundTable.dialog.personaPresets.security,
      t.web.roundTable.dialog.personaPresets.performance,
      t.web.roundTable.dialog.personaPresets.ux,
      t.web.roundTable.dialog.personaPresets.skeptic,
      t.web.roundTable.dialog.personaPresets.pragmatist,
    ];
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        TextField(
          controller: _ctrl,
          minLines: 1,
          maxLines: 3,
          decoration: InputDecoration(
            hintText: t.web.roundTable.dialog.personaPlaceholder,
            isDense: true,
            border: const OutlineInputBorder(),
          ),
          onChanged: widget.onPersona,
        ),
        const SizedBox(height: 4),
        Wrap(
          spacing: 6,
          runSpacing: 4,
          crossAxisAlignment: WrapCrossAlignment.center,
          children: [
            Text(t.web.roundTable.dialog.personaPresets.label,
                style: theme.textTheme.labelSmall),
            for (final p in presets)
              ActionChip(
                visualDensity: VisualDensity.compact,
                label: Text(p, style: theme.textTheme.labelSmall),
                onPressed: () => _apply(p),
              ),
          ],
        ),
      ],
    );
  }
}

class _SeatTile extends StatelessWidget {
  const _SeatTile({
    required this.provider,
    required this.on,
    required this.model,
    required this.account,
    required this.persona,
    required this.modelOptions,
    required this.accounts,
    required this.onToggle,
    required this.onModel,
    required this.onAccount,
    required this.onPersona,
  });

  final String provider;
  final bool on;
  final String model;
  final String account;
  final String persona;
  final List<SeatModelOption> modelOptions;
  final List<({String id, String label, bool usable})> accounts;
  final VoidCallback onToggle;
  final ValueChanged<String> onModel;
  final ValueChanged<String> onAccount;
  final ValueChanged<String> onPersona;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final showAccount = on && seatSupportsAccount.contains(provider) &&
        accounts.isNotEmpty;
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                BrandAvatar(providerId: provider, size: 28),
                const SizedBox(width: 10),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(provider,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            fontWeight: FontWeight.w600,
                          )),
                      Text(seatVendor[provider] ?? provider,
                          style: theme.textTheme.bodySmall),
                    ],
                  ),
                ),
                Switch(value: on, onChanged: (_) => onToggle()),
              ],
            ),
            if (on) ...[
              const SizedBox(height: 8),
              // Model dropdown ("" = CLI default).
              DropdownButtonFormField<String>(
                initialValue: modelOptions.any((o) => o.value == model)
                    ? model
                    : (modelOptions.isNotEmpty ? modelOptions.first.value : ''),
                isExpanded: true,
                decoration: InputDecoration(
                  labelText: t.web.roundTable.dialog.modelPlaceholder,
                  isDense: true,
                  border: const OutlineInputBorder(),
                ),
                items: [
                  if (modelOptions.isEmpty)
                    DropdownMenuItem(
                      value: '',
                      child: Text(t.web.roundTable.dialog.modelLoading),
                    ),
                  for (final o in modelOptions)
                    DropdownMenuItem(value: o.value, child: Text(o.label)),
                ],
                onChanged: (v) => onModel(v ?? ''),
              ),
              if (showAccount) ...[
                const SizedBox(height: 8),
                DropdownButtonFormField<String>(
                  initialValue:
                      accounts.any((a) => a.id == account) ? account : '',
                  isExpanded: true,
                  decoration: InputDecoration(
                    labelText: t.web.roundTable.dialog.accountPlaceholder,
                    isDense: true,
                    border: const OutlineInputBorder(),
                  ),
                  items: [
                    DropdownMenuItem(
                      value: '',
                      child: Text(t.web.roundTable.dialog.accountDefault),
                    ),
                    for (final a in accounts)
                      DropdownMenuItem(
                        value: a.id,
                        enabled: a.usable,
                        child: Text(
                          a.usable
                              ? a.label
                              : '${a.label} (${t.web.roundTable.dialog.accountNoToken})',
                        ),
                      ),
                  ],
                  onChanged: (v) => onAccount(v ?? ''),
                ),
              ],
              const SizedBox(height: 8),
              _PersonaField(persona: persona, onPersona: onPersona),
            ],
          ],
        ),
      ),
    );
  }
}
