import 'package:flutter/material.dart';

import 'package:opendray/core/api/models.dart';

// Claude account UI dialogs. Account creation is gateway-host only
// (run `claude login` with CLAUDE_CONFIG_DIR), so the only dialog
// kept here is the inline rename for an existing row.

class RenameClaudeAccountDialog extends StatefulWidget {
  const RenameClaudeAccountDialog({required this.account, super.key});
  final ClaudeAccountSummary account;

  static Future<String?> show(
    BuildContext context,
    ClaudeAccountSummary account,
  ) {
    return showDialog<String>(
      context: context,
      builder: (_) => RenameClaudeAccountDialog(account: account),
    );
  }

  @override
  State<RenameClaudeAccountDialog> createState() =>
      _RenameClaudeAccountDialogState();
}

class _RenameClaudeAccountDialogState
    extends State<RenameClaudeAccountDialog> {
  late final TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: widget.account.displayName);
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('Rename ${widget.account.name}'),
      content: TextField(
        controller: _ctrl,
        autofocus: true,
        autocorrect: false,
        textInputAction: TextInputAction.done,
        onSubmitted: (v) => Navigator.of(context).pop(v.trim()),
        decoration: const InputDecoration(
          labelText: 'Display name',
          hintText: 'Work account',
          isDense: true,
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(_ctrl.text.trim()),
          child: const Text('Save'),
        ),
      ],
    );
  }
}
