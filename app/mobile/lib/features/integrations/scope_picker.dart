import 'package:flutter/material.dart';

import 'package:opendray/features/integrations/scopes.dart';

// ScopePicker — the Dart mirror of the web ScopePicker. Renders every
// known scope grouped by topic with a human-readable title + one-line
// description, so the operator toggles scopes by meaning instead of
// memorising tokens like "event:subscribe:session.*".
//
// A scope already on the integration but absent from kScopeInfo (a newer
// gateway scope) is not rendered as a row, but stays in `selected` and is
// preserved on save — toggling only ever adds/removes a known id.
class ScopePicker extends StatelessWidget {
  const ScopePicker({
    required this.selected,
    required this.onChanged,
    this.intro,
    super.key,
  });

  final List<String> selected;
  final ValueChanged<List<String>> onChanged;
  final String? intro;

  void _toggle(String id) {
    onChanged(
      selected.contains(id)
          ? selected.where((x) => x != id).toList()
          : [...selected, id],
    );
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        if (intro != null && intro!.isNotEmpty) ...[
          Text(
            intro!,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 12),
        ],
        for (final g in kScopeGroups)
          ..._group(context, g),
      ],
    );
  }

  List<Widget> _group(BuildContext context, ScopeGroupMeta g) {
    final theme = Theme.of(context);
    final items = kScopeInfo.where((s) => s.group == g.id).toList();
    if (items.isEmpty) return const [];
    return [
      Padding(
        padding: const EdgeInsets.only(bottom: 2, top: 4),
        child: Text(
          g.label.toUpperCase(),
          style: theme.textTheme.labelSmall?.copyWith(
            letterSpacing: 0.6,
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ),
      Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Text(
          g.blurb,
          style: theme.textTheme.bodySmall?.copyWith(
            color: theme.colorScheme.onSurfaceVariant,
          ),
        ),
      ),
      for (final s in items) _row(context, s),
      const SizedBox(height: 12),
    ];
  }

  Widget _row(BuildContext context, ScopeInfo s) {
    final theme = Theme.of(context);
    final isOn = selected.contains(s.id);
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: InkWell(
        borderRadius: BorderRadius.circular(8),
        onTap: () => _toggle(s.id),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
          decoration: BoxDecoration(
            borderRadius: BorderRadius.circular(8),
            border: Border.all(
              color: isOn
                  ? theme.colorScheme.primary.withValues(alpha: 0.55)
                  : theme.dividerColor.withValues(alpha: 0.4),
            ),
            color: isOn
                ? theme.colorScheme.primary.withValues(alpha: 0.06)
                : null,
          ),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              SizedBox(
                width: 22,
                height: 22,
                child: Checkbox(
                  value: isOn,
                  visualDensity: VisualDensity.compact,
                  materialTapTargetSize: MaterialTapTargetSize.shrinkWrap,
                  onChanged: (_) => _toggle(s.id),
                ),
              ),
              const SizedBox(width: 8),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Wrap(
                      crossAxisAlignment: WrapCrossAlignment.center,
                      spacing: 8,
                      children: [
                        Text(
                          s.title,
                          style: theme.textTheme.bodyMedium?.copyWith(
                            fontWeight: FontWeight.w600,
                          ),
                        ),
                        Text(
                          s.id,
                          style: const TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 11,
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 2),
                    Text(
                      s.description,
                      style: theme.textTheme.bodySmall?.copyWith(
                        color: theme.colorScheme.onSurfaceVariant,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
