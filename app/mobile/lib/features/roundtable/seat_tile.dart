import 'package:flutter/material.dart';

import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';

// Shared seat widgets for the Round Table create + members sheets. Mirrors
// app/web/src/components/roundtable/SeatPicker.tsx — one tile per vendor with a
// toggle, and for a seated member a model dropdown, an optional account
// dropdown (claude/antigravity) and a persona field with role presets.

// PersonaField — persona text input + quick-fill role preset chips. The
// localized preset label is the persona text sent to the model (models handle
// any language). Keyed by provider so switching members rebuilds the controller.
class PersonaField extends StatefulWidget {
  const PersonaField({required this.persona, required this.onPersona, super.key});
  final String persona;
  final ValueChanged<String> onPersona;

  @override
  State<PersonaField> createState() => _PersonaFieldState();
}

class _PersonaFieldState extends State<PersonaField> {
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

class SeatTile extends StatelessWidget {
  const SeatTile({
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
    super.key,
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
    final showAccount =
        on && seatSupportsAccount.contains(provider) && accounts.isNotEmpty;
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
              // Keyed by provider so each seat keeps its own persona controller;
              // seeded once from `persona` (the parent owns the value).
              PersonaField(
                key: ValueKey(provider),
                persona: persona,
                onPersona: onPersona,
              ),
            ],
          ],
        ),
      ),
    );
  }
}
