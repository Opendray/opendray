import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Live role/framing editor — reassign each member's persona and the shared
// framing directive on an active round table as the topic evolves. Mobile
// parity with app/web RolesDialog; changes take effect on the next reply.
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
  late final Map<String, TextEditingController> _personas = {
    for (final s in widget.rt.seats)
      s.provider: TextEditingController(text: s.persona),
  };
  bool _saving = false;

  @override
  void dispose() {
    _framing.dispose();
    for (final c in _personas.values) {
      c.dispose();
    }
    super.dispose();
  }

  List<String> get _presets => [
        t.web.roundTable.dialog.personaPresets.security,
        t.web.roundTable.dialog.personaPresets.performance,
        t.web.roundTable.dialog.personaPresets.ux,
        t.web.roundTable.dialog.personaPresets.skeptic,
        t.web.roundTable.dialog.personaPresets.pragmatist,
      ];

  Future<void> _save() async {
    setState(() => _saving = true);
    try {
      final seats = widget.rt.seats
          .map((s) => Seat(
                provider: s.provider,
                model: s.model,
                accountId: s.accountId,
                persona: _personas[s.provider]?.text.trim() ?? '',
              ))
          .toList();
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
                for (final s in widget.rt.seats) ...[
                  Text(s.provider,
                      style: theme.textTheme.labelLarge
                          ?.copyWith(fontWeight: FontWeight.w600)),
                  const SizedBox(height: 6),
                  TextField(
                    controller: _personas[s.provider],
                    minLines: 1,
                    maxLines: 3,
                    decoration: InputDecoration(
                      hintText: t.web.roundTable.dialog.personaPlaceholder,
                      isDense: true,
                      border: const OutlineInputBorder(),
                    ),
                  ),
                  const SizedBox(height: 4),
                  Wrap(
                    spacing: 6,
                    runSpacing: 4,
                    children: [
                      for (final p in _presets)
                        ActionChip(
                          visualDensity: VisualDensity.compact,
                          label: Text(p, style: theme.textTheme.labelSmall),
                          onPressed: () {
                            _personas[s.provider]?.text = p;
                          },
                        ),
                    ],
                  ),
                  const SizedBox(height: 12),
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
                  onPressed: _saving ? null : _save,
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
