import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';

// GitBranchControls is the per-status-pane action strip: current
// branch chip → opens a bottom-sheet picker that shows every local
// branch with its upstream tracking + delete affordance. The
// dropdown shape we used initially hid all this info and gave
// nowhere to delete from; on a phone screen a tappable chip plus a
// full-height sheet is the only ergonomic option. "+ New" and
// "Push" stay on the strip so the common ops are one tap away.
class GitBranchControls extends ConsumerStatefulWidget {
  const GitBranchControls({
    required this.cwd,
    required this.ahead,
    required this.upstream,
    required this.onChanged,
    super.key,
  });

  final String cwd;
  // ahead count vs upstream (from GitStatusResponse). 0 with an
  // upstream set disables the push button (nothing to ship).
  final int ahead;
  // Empty when the current branch has no upstream tracked — push
  // will use --set-upstream in that case.
  final String upstream;
  // Fired on any successful branch-changing op so the parent can
  // refresh status + log (current branch / file list changes).
  final VoidCallback onChanged;

  @override
  ConsumerState<GitBranchControls> createState() => _GitBranchControlsState();
}

class _GitBranchControlsState extends ConsumerState<GitBranchControls> {
  GitBranchList? _branches;
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
  }

  Future<void> _load() async {
    try {
      final list = await ref.read(gitApiProvider).listBranches(widget.cwd);
      if (!mounted) return;
      setState(() => _branches = list);
    } on ApiException catch (_) {
      // Not a repo / no token / network — silently keep null and
      // let the parent's "not a repo" UI dominate.
      if (mounted) setState(() => _branches = null);
    }
  }

  Future<void> _checkout(String name) async {
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref
          .read(gitApiProvider)
          .checkoutBranch(dir: widget.cwd, name: name);
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text('Switched to $name')));
      widget.onChanged();
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Checkout failed: ${e.message}'),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  // _delete runs `git branch -d` first; if the branch is unmerged
  // git refuses with a 409 + "not fully merged" body. We surface
  // that as a second confirmation that upgrades to -D (force).
  // Keeping the two-step flow means a slip can't blow away work.
  Future<void> _delete(GitBranchRef ref_) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (_) => AlertDialog(
        title: Text('Delete ${ref_.name}?'),
        content: Text(
          ref_.upstream.isEmpty
              ? 'This branch has no upstream — make sure it has nothing you need.'
              : 'Local branch only. Remote ${ref_.upstream} is untouched.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(context).colorScheme.error,
            ),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (!(confirmed ?? false) || !mounted) return;
    await _doDelete(ref_, force: false);
  }

  Future<void> _doDelete(GitBranchRef ref_, {required bool force}) async {
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref
          .read(gitApiProvider)
          .deleteBranch(dir: widget.cwd, name: ref_.name, force: force);
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text('Deleted ${ref_.name}')));
      widget.onChanged();
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      // git refuses unmerged branches with status 409 + a message
      // mentioning "not fully merged". Treat that as a force-prompt
      // signal so the operator gets a clear second confirm rather
      // than an opaque toast.
      final lower = e.message.toLowerCase();
      final canForce =
          !force &&
          (e.statusCode == 409 ||
              lower.contains('not fully merged') ||
              lower.contains('not yet merged'));
      if (canForce) {
        final go = await showDialog<bool>(
          context: context,
          builder: (_) => AlertDialog(
            title: Text('Force delete ${ref_.name}?'),
            content: const Text(
              'Branch is not fully merged. Forcing deletion will lose '
              'any commits unique to this branch.',
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(context).pop(false),
                child: const Text('Cancel'),
              ),
              FilledButton(
                onPressed: () => Navigator.of(context).pop(true),
                style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(context).colorScheme.error,
                ),
                child: const Text('Force delete'),
              ),
            ],
          ),
        );
        if ((go ?? false) && mounted) {
          await _doDelete(ref_, force: true);
        }
      } else {
        messenger.showSnackBar(
          SnackBar(
            content: Text('Delete failed: ${e.message}'),
            backgroundColor: Theme.of(context).colorScheme.error,
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _create() async {
    final name = await _promptForBranchName();
    if (name == null || name.isEmpty || !mounted) return;
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    final errorColor = Theme.of(context).colorScheme.error;
    try {
      await ref
          .read(gitApiProvider)
          .createBranch(dir: widget.cwd, name: name, switchTo: true);
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text('Created $name')));
      widget.onChanged();
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Create branch failed: ${e.message}'),
          backgroundColor: errorColor,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<String?> _promptForBranchName() async {
    // Use an internal StatefulWidget so the TextEditingController
    // is owned by the dialog's State and disposed at the right
    // time (after the TextField is detached). Disposing it from
    // the calling Future right after showDialog returns triggers
    // a "_dependents.isEmpty" assertion in framework.dart because
    // the TextField is still being torn down when we dispose its
    // listener.
    return showDialog<String>(
      context: context,
      builder: (_) => const _BranchNameDialog(),
    );
  }

  Future<void> _push() async {
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final branch = await ref
          .read(gitApiProvider)
          .push(dir: widget.cwd, setUpstream: widget.upstream.isEmpty);
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(content: Text('Pushed $branch')));
      widget.onChanged();
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Push failed: ${e.message}'),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _openPicker() async {
    final list = _branches;
    if (list == null || _busy) return;
    final picked = await showModalBottomSheet<_PickerResult>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (sheetCtx) => _BranchPickerSheet(
        branches: list.branches,
        current: list.current,
      ),
    );
    if (picked == null || !mounted) return;
    if (picked.action == _PickerAction.checkout && picked.ref.name != list.current) {
      await _checkout(picked.ref.name);
    } else if (picked.action == _PickerAction.delete) {
      await _delete(picked.ref);
    }
  }

  @override
  Widget build(BuildContext context) {
    final branches = _branches;
    if (branches == null) {
      return const SizedBox(height: 0);
    }
    final current = branches.current.isNotEmpty ? branches.current : '(detached)';
    final localCount = branches.branches.where((b) => !b.isRemote).length;
    // Push is disabled when an upstream exists AND we're not ahead.
    // First push (no upstream) is always allowed; the operator
    // typically wants to publish the branch to start a PR.
    final pushDisabled = widget.upstream.isNotEmpty && widget.ahead == 0;
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
      child: Row(
        children: [
          Expanded(
            child: InkWell(
              onTap: _busy ? null : _openPicker,
              borderRadius: BorderRadius.circular(6),
              child: Container(
                padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                decoration: BoxDecoration(
                  border: Border.all(color: theme.dividerColor),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Row(
                  children: [
                    Icon(
                      Icons.account_tree_outlined,
                      size: 16,
                      color: theme.colorScheme.primary,
                    ),
                    const SizedBox(width: 8),
                    Expanded(
                      child: Text(
                        current,
                        style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 13,
                          fontWeight: FontWeight.w600,
                        ),
                        overflow: TextOverflow.ellipsis,
                      ),
                    ),
                    Text(
                      '$localCount local',
                      style: TextStyle(
                        fontSize: 11,
                        color: theme.textTheme.bodySmall?.color,
                      ),
                    ),
                    const SizedBox(width: 4),
                    Icon(
                      Icons.unfold_more,
                      size: 16,
                      color: theme.iconTheme.color?.withValues(alpha: 0.6),
                    ),
                  ],
                ),
              ),
            ),
          ),
          const SizedBox(width: 6),
          IconButton(
            tooltip: 'New branch',
            onPressed: _busy ? null : _create,
            icon: const Icon(Icons.add_circle_outline, size: 22),
          ),
          IconButton(
            tooltip: widget.upstream.isEmpty
                ? 'Push (set upstream)'
                : 'Push (${widget.ahead} ahead)',
            onPressed: pushDisabled || _busy ? null : _push,
            icon: _busy
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : Icon(
                    Icons.upload_outlined,
                    size: 22,
                    color: pushDisabled
                        ? theme.disabledColor
                        : theme.colorScheme.primary,
                  ),
          ),
        ],
      ),
    );
  }
}

// _BranchPickerSheet lists every local branch in a scrollable
// sheet so each row has space for the upstream subtitle + delete
// affordance — none of that fits in a DropdownButtonFormField on
// a phone screen.
enum _PickerAction { checkout, delete }

class _PickerResult {
  _PickerResult(this.action, this.ref);
  final _PickerAction action;
  final GitBranchRef ref;
}

class _BranchPickerSheet extends StatelessWidget {
  const _BranchPickerSheet({required this.branches, required this.current});

  final List<GitBranchRef> branches;
  final String current;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final local = branches.where((b) => !b.isRemote).toList()
      ..sort((a, b) {
        if (a.isCurrent != b.isCurrent) return a.isCurrent ? -1 : 1;
        return a.name.compareTo(b.name);
      });
    final remote = branches.where((b) => b.isRemote).toList()
      ..sort((a, b) => a.name.compareTo(b.name));
    return SafeArea(
      child: ConstrainedBox(
        constraints: BoxConstraints(
          maxHeight: MediaQuery.of(context).size.height * 0.75,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
              child: Row(
                children: [
                  Text(
                    'Branches',
                    style: theme.textTheme.titleMedium?.copyWith(
                      fontWeight: FontWeight.w600,
                    ),
                  ),
                  const Spacer(),
                  Text(
                    '${local.length} local · ${remote.length} remote',
                    style: theme.textTheme.bodySmall,
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            Flexible(
              child: ListView(
                shrinkWrap: true,
                padding: EdgeInsets.zero,
                children: [
                  for (final b in local)
                    _BranchRow(
                      ref_: b,
                      isCurrent: b.name == current,
                      onTap: () => Navigator.of(
                        context,
                      ).pop(_PickerResult(_PickerAction.checkout, b)),
                      onDelete: b.name == current
                          ? null
                          : () => Navigator.of(
                              context,
                            ).pop(_PickerResult(_PickerAction.delete, b)),
                    ),
                  if (remote.isNotEmpty) ...[
                    Padding(
                      padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
                      child: Text(
                        'Remote',
                        style: theme.textTheme.labelSmall?.copyWith(
                          color: theme.textTheme.bodySmall?.color,
                          letterSpacing: 1.2,
                        ),
                      ),
                    ),
                    for (final b in remote)
                      _BranchRow(
                        ref_: b,
                        isCurrent: false,
                        // Tapping a remote ref checks out the local
                        // branch of the same short name — same as
                        // `git checkout <name>` resolving to the
                        // remote-tracking branch on first switch.
                        onTap: () => Navigator.of(context).pop(
                          _PickerResult(
                            _PickerAction.checkout,
                            GitBranchRef(
                              name: b.name,
                              remote: '',
                              isRemote: false,
                              isCurrent: false,
                              upstream: '${b.remote}/${b.name}',
                            ),
                          ),
                        ),
                        onDelete: null,
                      ),
                  ],
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _BranchRow extends StatelessWidget {
  const _BranchRow({
    required this.ref_,
    required this.isCurrent,
    required this.onTap,
    required this.onDelete,
  });

  final GitBranchRef ref_;
  final bool isCurrent;
  final VoidCallback onTap;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final subtitle = ref_.isRemote
        ? '${ref_.remote}/${ref_.name}'
        : (ref_.upstream.isEmpty ? 'no upstream' : '→ ${ref_.upstream}');
    return ListTile(
      onTap: onTap,
      dense: true,
      leading: Icon(
        isCurrent
            ? Icons.check_circle
            : (ref_.isRemote
                  ? Icons.cloud_outlined
                  : Icons.account_tree_outlined),
        size: 18,
        color: isCurrent
            ? theme.colorScheme.primary
            : theme.iconTheme.color?.withValues(alpha: 0.6),
      ),
      title: Text(
        ref_.name,
        style: TextStyle(
          fontFamily: 'monospace',
          fontSize: 13,
          fontWeight: isCurrent ? FontWeight.w700 : FontWeight.w500,
        ),
      ),
      subtitle: Text(
        subtitle,
        style: TextStyle(
          fontFamily: 'monospace',
          fontSize: 11,
          color: theme.textTheme.bodySmall?.color,
        ),
      ),
      trailing: onDelete == null
          ? null
          : IconButton(
              tooltip: 'Delete branch',
              icon: const Icon(Icons.delete_outline, size: 20),
              color: theme.colorScheme.error,
              onPressed: onDelete,
            ),
    );
  }
}

// _BranchNameDialog owns its TextEditingController via its own
// State so the dispose order is correct — the framework unmounts
// the TextField first, then State.dispose() runs and tears down
// the controller. Putting that controller in the caller (the
// async function) instead would dispose it while the TextField
// is still listening, tripping the _dependents.isEmpty assertion.
class _BranchNameDialog extends StatefulWidget {
  const _BranchNameDialog();

  @override
  State<_BranchNameDialog> createState() => _BranchNameDialogState();
}

class _BranchNameDialogState extends State<_BranchNameDialog> {
  final _controller = TextEditingController();

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit() {
    final name = _controller.text.trim();
    if (name.isEmpty) {
      Navigator.of(context).pop(null);
      return;
    }
    Navigator.of(context).pop(name);
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('New branch'),
      content: TextField(
        controller: _controller,
        autofocus: true,
        decoration: const InputDecoration(
          hintText: 'feat/something',
          border: OutlineInputBorder(),
        ),
        onSubmitted: (_) => _submit(),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(null),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Create')),
      ],
    );
  }
}
