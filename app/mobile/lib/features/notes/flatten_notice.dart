// A vault that still files every project inside a folder called
// `projects/` can be converted, and its owner has no way to learn that
// unless something says so where they are already looking.
//
// It is an offer, not a nag: dismissing it is remembered, and it never
// appears for a vault that is already flat or has nothing to move.
//
// Nothing moves without a preview. The sheet runs the migration as a
// dry run first and shows the real list — including what it refuses to
// touch and why — because "convert my documents" is not something to
// agree to from a one-line description.
//
// Mirrors app/web/src/components/notes/FlattenNotice.tsx.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/notes_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:shared_preferences/shared_preferences.dart';

const _dismissKey = 'vault.flatten.dismissed';

class FlattenNotice extends ConsumerStatefulWidget {
  const FlattenNotice({required this.onConverted, super.key});

  /// Called after a successful conversion so the caller can reload the
  /// listing — every path it is holding has just changed.
  final Future<void> Function() onConverted;

  @override
  ConsumerState<FlattenNotice> createState() => _FlattenNoticeState();
}

class _FlattenNoticeState extends ConsumerState<FlattenNotice> {
  bool _dismissed = false;
  bool _checked = false;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    _loadDismissed();
  }

  Future<void> _loadDismissed() async {
    var dismissed = false;
    try {
      final prefs = await SharedPreferences.getInstance();
      dismissed = prefs.getBool(_dismissKey) ?? false;
    } on Object {
      // Unavailable prefs cost the memory of the choice, not the choice.
    }
    if (!mounted) return;
    setState(() {
      _dismissed = dismissed;
      _checked = true;
    });
  }

  Future<void> _dismiss() async {
    setState(() => _dismissed = true);
    try {
      final prefs = await SharedPreferences.getInstance();
      await prefs.setBool(_dismissKey, true);
    } on Object {
      // Same as above.
    }
  }

  Future<void> _preview() async {
    setState(() => _busy = true);
    try {
      final plan = await ref.read(notesApiProvider).flatten();
      if (!mounted) return;
      await _showPlan(plan);
    } on ApiException catch (e) {
      if (mounted) _toast(e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _showPlan(FlattenResult plan) async {
    final confirmed = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => _PlanSheet(plan: plan),
    );
    if (confirmed != true || !mounted) return;
    await _apply();
  }

  Future<void> _apply() async {
    setState(() => _busy = true);
    try {
      final res = await ref.read(notesApiProvider).flatten(apply: true);
      if (!mounted) return;
      _toast(t.notesPage.flatten.done(count: res.moves.length));
      await _dismiss();
      await widget.onConverted();
    } on ApiException catch (e) {
      if (mounted) _toast(e.message);
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  void _toast(String message) {
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(content: Text(message), behavior: SnackBarBehavior.floating),
    );
  }

  @override
  Widget build(BuildContext context) {
    if (!_checked || _dismissed) return const SizedBox.shrink();
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
      color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: .5),
      child: Row(
        children: [
          Icon(
            Icons.account_tree_outlined,
            size: 16,
            color: theme.colorScheme.onSurfaceVariant,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              t.notesPage.flatten.notice,
              style: TextStyle(
                fontSize: 12,
                color: theme.colorScheme.onSurfaceVariant,
              ),
            ),
          ),
          if (_busy)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 12),
              child: SizedBox(
                width: 14,
                height: 14,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          else ...[
            TextButton(
              onPressed: _preview,
              child: Text(
                t.notesPage.flatten.preview,
                style: const TextStyle(fontSize: 12),
              ),
            ),
            TextButton(
              onPressed: _dismiss,
              child: Text(
                t.notesPage.flatten.dismiss,
                style: const TextStyle(fontSize: 12),
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _PlanSheet extends StatelessWidget {
  const _PlanSheet({required this.plan});

  final FlattenResult plan;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 14, 16, 10),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(t.notesPage.flatten.title, style: theme.textTheme.titleSmall),
            const SizedBox(height: 6),
            Text(
              t.notesPage.flatten.description,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            Flexible(
              child: SingleChildScrollView(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    if (plan.moves.isEmpty)
                      Text(
                        t.notesPage.flatten.nothingToMove,
                        style: theme.textTheme.bodySmall,
                      ),
                    for (final m in plan.moves)
                      Padding(
                        padding: const EdgeInsets.symmetric(vertical: 1),
                        child: Text(
                          '${m.from}  →  ${m.to}',
                          style: const TextStyle(
                            fontFamily: 'monospace',
                            fontSize: 11,
                          ),
                        ),
                      ),
                    if (plan.skips.isNotEmpty) ...[
                      const SizedBox(height: 10),
                      Text(
                        t.notesPage.flatten.skipped(count: plan.skips.length),
                        style: theme.textTheme.bodySmall,
                      ),
                      for (final s in plan.skips)
                        Padding(
                          padding: const EdgeInsets.symmetric(vertical: 1),
                          child: Text(
                            '${s.path} — ${s.reason}',
                            style: TextStyle(
                              fontFamily: 'monospace',
                              fontSize: 11,
                              color: theme.colorScheme.onSurfaceVariant,
                            ),
                          ),
                        ),
                    ],
                  ],
                ),
              ),
            ),
            const SizedBox(height: 12),
            Text(
              t.notesPage.flatten.restartHint,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 8),
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                TextButton(
                  onPressed: () => Navigator.of(context).pop(false),
                  child: Text(t.common.cancel),
                ),
                const SizedBox(width: 8),
                FilledButton(
                  onPressed: plan.moves.isEmpty
                      ? null
                      : () => Navigator.of(context).pop(true),
                  child: Text(
                    t.notesPage.flatten.convert(count: plan.moves.length),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
