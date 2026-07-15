import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';
import 'package:opendray/features/roundtable/run_step_options_sheet.dart';
import 'package:opendray/features/sessions/directory_picker_sheet.dart';

// Role-based execution plan — draft an ordered, member-assigned plan from the
// discussion, tweak it, then run each step (spawns a real session in the shared
// project cwd). Mobile parity with app/web PlanDialog. Operator-driven.
class PlanSheet extends ConsumerStatefulWidget {
  const PlanSheet({required this.rt, super.key});
  final RoundTable rt;

  static Future<void> show(BuildContext context, RoundTable rt) {
    return showModalBottomSheet<void>(
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
      builder: (_) => PlanSheet(rt: rt),
    );
  }

  @override
  ConsumerState<PlanSheet> createState() => _PlanSheetState();
}

class _PlanSheetState extends ConsumerState<PlanSheet> {
  late final List<PlanStep> _steps = List.of(widget.rt.plan);
  late final List<TextEditingController> _taskCtrls = [
    for (final s in _steps) TextEditingController(text: s.task),
  ];
  bool _busy = false;
  // Editable copy of the bound project — a table can be created without one,
  // and a plan can only run once it's bound (steps share the working tree).
  late String _cwd = widget.rt.cwd;

  List<String> get _providers =>
      widget.rt.seats.map((s) => s.provider).toList();
  bool get _hasCwd => _cwd.isNotEmpty;

  @override
  void dispose() {
    for (final c in _taskCtrls) {
      c.dispose();
    }
    super.dispose();
  }

  void _addStep() {
    setState(() {
      _steps.add(PlanStep(
        assignee: _providers.isNotEmpty ? _providers.first : '',
        task: '',
      ));
      _taskCtrls.add(TextEditingController());
    });
  }

  void _removeStep(int i) {
    setState(() {
      _steps.removeAt(i);
      _taskCtrls.removeAt(i).dispose();
    });
  }

  List<PlanStep> _collect() => [
        for (var i = 0; i < _steps.length; i++)
          _steps[i].copyWith(task: _taskCtrls[i].text.trim()),
      ];

  Future<void> _draft() async {
    setState(() => _busy = true);
    try {
      await ref.read(roundtableApiProvider).draftPlan(widget.rt.id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(t.web.roundTable.plan.drafting)),
        );
        Navigator.of(context).pop();
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() => _busy = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  Future<void> _save() async {
    setState(() => _busy = true);
    try {
      await ref.read(roundtableApiProvider).setPlan(widget.rt.id, _collect());
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(t.web.roundTable.plan.saved)),
        );
        Navigator.of(context).pop();
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() => _busy = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  // Bind the shared project working dir after the fact so the plan's steps
  // become runnable. Picking a dir PATCHes the table, then flips _hasCwd true.
  Future<void> _bindProject() async {
    final picked = await DirectoryPickerSheet.show(
      context,
      initialPath: _cwd.isNotEmpty ? _cwd : null,
    );
    if (picked == null || picked.trim().isEmpty || !mounted) return;
    setState(() => _busy = true);
    try {
      await ref.read(roundtableApiProvider).update(widget.rt.id, cwd: picked.trim());
      if (mounted) {
        setState(() {
          _cwd = picked.trim();
          _busy = false;
        });
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(t.web.roundTable.plan.projectBound)),
        );
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() => _busy = false);
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  Future<void> _run(int i) async {
    // Pre-launch options: account (multi-account) + bypass/YOLO.
    final opts = await RunStepOptionsSheet.show(context, _collect()[i], i);
    if (opts == null || !mounted) return;
    setState(() => _busy = true);
    try {
      // Persist edits before running so the seed matches what's on screen.
      await ref.read(roundtableApiProvider).setPlan(widget.rt.id, _collect());
      final sid = await ref.read(roundtableApiProvider).runPlanStep(
            widget.rt.id,
            i,
            accountId: opts.accountId,
            args: opts.args,
          );
      if (mounted && sid.isNotEmpty) {
        Navigator.of(context).pop();
        context.go('/session/$sid');
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() => _busy = false);
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
                Text(t.web.roundTable.plan.title,
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
              child: Text(t.web.roundTable.plan.hint,
                  style: theme.textTheme.bodySmall),
            ),
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: Row(
              children: [
                OutlinedButton.icon(
                  onPressed: _busy ? null : _draft,
                  icon: const Icon(Icons.auto_awesome_outlined, size: 16),
                  label: Text(t.web.roundTable.plan.draft),
                ),
                const SizedBox(width: 8),
                TextButton.icon(
                  onPressed: _busy ? null : _addStep,
                  icon: const Icon(Icons.add, size: 16),
                  label: Text(t.web.roundTable.plan.addStep),
                ),
              ],
            ),
          ),
          if (!_hasCwd)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 0),
              child: Row(
                children: [
                  Expanded(
                    child: Text(t.web.roundTable.plan.needProject,
                        style: theme.textTheme.bodySmall
                            ?.copyWith(color: theme.colorScheme.error)),
                  ),
                  const SizedBox(width: 8),
                  FilledButton.icon(
                    onPressed: _busy ? null : _bindProject,
                    icon: const Icon(Icons.folder_open, size: 16),
                    label: Text(t.web.roundTable.plan.bindProject),
                  ),
                ],
              ),
            ),
          Flexible(
            child: _steps.isEmpty
                ? Center(
                    child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Text(t.web.roundTable.plan.empty,
                          style: theme.textTheme.bodySmall),
                    ),
                  )
                : ListView.builder(
                    shrinkWrap: true,
                    padding: const EdgeInsets.fromLTRB(16, 8, 16, 8),
                    itemCount: _steps.length,
                    itemBuilder: (_, i) => _stepCard(i),
                  ),
          ),
          SafeArea(
            top: false,
            child: Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
              child: SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: _busy ? null : _save,
                  child: Text(t.web.roundTable.plan.save),
                ),
              ),
            ),
          ),
        ],
      ),
    );
  }

  Widget _stepCard(int i) {
    final theme = Theme.of(context);
    final step = _steps[i];
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: Padding(
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Text('${i + 1}', style: theme.textTheme.labelLarge),
                const SizedBox(width: 8),
                Expanded(
                  child: DropdownButtonFormField<String>(
                    initialValue: _providers.contains(step.assignee)
                        ? step.assignee
                        : (_providers.isNotEmpty ? _providers.first : null),
                    isExpanded: true,
                    decoration: const InputDecoration(
                        isDense: true, border: OutlineInputBorder()),
                    items: [
                      for (final p in _providers)
                        DropdownMenuItem(
                          value: p,
                          child: Row(
                            mainAxisSize: MainAxisSize.min,
                            children: [
                              BrandAvatar(providerId: p, size: 16),
                              const SizedBox(width: 6),
                              Text(p),
                            ],
                          ),
                        ),
                    ],
                    onChanged: (v) => setState(
                        () => _steps[i] = step.copyWith(assignee: v ?? '')),
                  ),
                ),
                IconButton(
                  visualDensity: VisualDensity.compact,
                  onPressed: () => _removeStep(i),
                  icon: const Icon(Icons.delete_outline, size: 18),
                ),
              ],
            ),
            const SizedBox(height: 6),
            TextField(
              controller: _taskCtrls[i],
              minLines: 1,
              maxLines: 3,
              decoration: InputDecoration(
                hintText: t.web.roundTable.plan.taskPlaceholder,
                isDense: true,
                border: const OutlineInputBorder(),
              ),
            ),
            const SizedBox(height: 6),
            Row(
              children: [
                _statusPill(step.status),
                const Spacer(),
                if (step.sessionId.isNotEmpty)
                  TextButton(
                    onPressed: () {
                      Navigator.of(context).pop();
                      context.go('/session/${step.sessionId}');
                    },
                    child: Text(t.web.roundTable.plan.openSession),
                  ),
                FilledButton.icon(
                  onPressed: (_busy || !_hasCwd) ? null : () => _run(i),
                  icon: const Icon(Icons.play_arrow, size: 16),
                  label: Text(step.status == 'done'
                      ? t.web.roundTable.plan.rerun
                      : t.web.roundTable.plan.run),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _statusPill(String status) {
    final theme = Theme.of(context);
    final color = status == 'done'
        ? Colors.green
        : status == 'running'
            ? Colors.blue
            : theme.colorScheme.outline;
    final label = status == 'done'
        ? t.web.roundTable.plan.done
        : status == 'running'
            ? t.web.roundTable.plan.running
            : t.web.roundTable.plan.pending;
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        border: Border.all(color: color.withValues(alpha: 0.5)),
        borderRadius: BorderRadius.circular(999),
      ),
      child: Text(label,
          style: theme.textTheme.labelSmall?.copyWith(color: color)),
    );
  }
}
