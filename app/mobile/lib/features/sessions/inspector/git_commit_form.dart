import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/git_api.dart';

// GitCommitForm renders the stage/commit panel that sits beneath
// the worktree file list. It's a separate widget (vs. an inline
// expansion of GitTab) so the keyboard / focus state stays
// localised when the parent re-renders from the status poll.
//
// Rules:
//   - Counts staged vs unstaged via the porcelain xy code: first
//     char != ' '/'?' means staged.
//   - "Stage all" appears when nothing is staged AND there are
//     modified files.
//   - Commit button is enabled only when message AND staged>0.
class GitCommitForm extends ConsumerStatefulWidget {
  const GitCommitForm({
    required this.cwd,
    required this.files,
    required this.onChanged,
    super.key,
  });

  final String cwd;
  final List<GitStatusFile> files;
  final VoidCallback onChanged;

  @override
  ConsumerState<GitCommitForm> createState() => _GitCommitFormState();
}

class _GitCommitFormState extends ConsumerState<GitCommitForm> {
  final _messageCtrl = TextEditingController();
  bool _busy = false;

  @override
  void dispose() {
    _messageCtrl.dispose();
    super.dispose();
  }

  bool _isStaged(GitStatusFile f) =>
      f.xy.isNotEmpty && f.xy[0] != ' ' && f.xy[0] != '?';

  int get _stagedCount => widget.files.where(_isStaged).length;

  Future<void> _stageAll() async {
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    final errorColor = Theme.of(context).colorScheme.error;
    try {
      await ref
          .read(gitApiProvider)
          .stageFiles(dir: widget.cwd, files: const []);
      if (!mounted) return;
      widget.onChanged();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Stage failed: ${e.message}'),
          backgroundColor: errorColor,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Future<void> _commit() async {
    final msg = _messageCtrl.text.trim();
    if (msg.isEmpty) return;
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    final errorColor = Theme.of(context).colorScheme.error;
    try {
      final hash = await ref
          .read(gitApiProvider)
          .commit(dir: widget.cwd, message: msg);
      if (!mounted) return;
      final short = hash.length >= 12 ? hash.substring(0, 12) : hash;
      messenger.showSnackBar(SnackBar(content: Text('Committed $short')));
      _messageCtrl.clear();
      widget.onChanged();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Commit failed: ${e.message}'),
          backgroundColor: errorColor,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final staged = _stagedCount;
    final hasMessage = _messageCtrl.text.trim().isNotEmpty;
    final canCommit = hasMessage && staged > 0 && !_busy;
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 4, 12, 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(
                  staged == 0
                      ? 'Nothing staged'
                      : '$staged staged · ${widget.files.length - staged} unstaged',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.outline,
                  ),
                ),
              ),
              if (staged == 0 && widget.files.isNotEmpty)
                TextButton(
                  onPressed: _busy ? null : _stageAll,
                  child: const Text('Stage all'),
                ),
            ],
          ),
          const SizedBox(height: 4),
          TextField(
            controller: _messageCtrl,
            enabled: !_busy,
            decoration: InputDecoration(
              isDense: true,
              border: const OutlineInputBorder(),
              hintText: staged == 0 ? 'Stage files first' : 'Commit message',
            ),
            style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            minLines: 2,
            maxLines: 4,
            onChanged: (_) => setState(() {}),
          ),
          const SizedBox(height: 6),
          Align(
            alignment: Alignment.centerRight,
            child: FilledButton.icon(
              onPressed: canCommit ? _commit : null,
              icon: _busy
                  ? const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.check, size: 18),
              label: const Text('Commit'),
            ),
          ),
        ],
      ),
    );
  }
}
