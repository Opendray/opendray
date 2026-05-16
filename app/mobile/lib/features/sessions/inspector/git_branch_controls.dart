import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';

// GitBranchControls is the per-status-pane action strip: current
// branch + dropdown to switch + "+ New" + Push. Switching refuses
// on a dirty tree (server returns 409 → friendly toast). Push uses
// set-upstream on first push (no upstream tracked yet) and stays
// disabled when nothing's ahead of an existing upstream.
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

  @override
  Widget build(BuildContext context) {
    final branches = _branches;
    if (branches == null) {
      return const SizedBox(height: 0);
    }
    final local = branches.branches.where((b) => !b.isRemote).toList();
    final current = branches.current;
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
            child: DropdownButtonFormField<String>(
              initialValue: current.isNotEmpty ? current : null,
              isDense: true,
              decoration: const InputDecoration(
                isDense: true,
                contentPadding: EdgeInsets.symmetric(
                  horizontal: 10,
                  vertical: 6,
                ),
                border: OutlineInputBorder(),
                labelText: 'Branch',
              ),
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              items: [
                for (final b in local)
                  DropdownMenuItem(value: b.name, child: Text(b.name)),
              ],
              onChanged: _busy
                  ? null
                  : (v) {
                      if (v != null && v != current) {
                        unawaited(_checkout(v));
                      }
                    },
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
