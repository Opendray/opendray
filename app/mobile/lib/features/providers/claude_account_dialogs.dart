import 'package:flutter/material.dart';

import 'package:opendray/core/api/models.dart';

// Claude account UI dialogs/pages. Mobile only exposes the
// "reserve a row" affordance — populating the OAuth credentials
// requires shell access on the gateway host (the canonical
// `claude login` flow with CLAUDE_CONFIG_DIR pointed at the
// per-account dir). Pasting the access token into a phone keyboard
// produces an account that dies in ~1 hour and never refreshes —
// that path is intentionally not surfaced here.

class _CreateAccountResult {
  _CreateAccountResult({required this.name, required this.displayName});
  final String name;
  final String displayName;
}

class CreateClaudeAccountScreen extends StatefulWidget {
  const CreateClaudeAccountScreen({super.key});

  static Future<({String name, String displayName})?> push(
    BuildContext context,
  ) async {
    final res = await Navigator.of(context).push<_CreateAccountResult>(
      MaterialPageRoute<_CreateAccountResult>(
        builder: (_) => const CreateClaudeAccountScreen(),
        fullscreenDialog: true,
      ),
    );
    if (res == null) return null;
    return (name: res.name, displayName: res.displayName);
  }

  @override
  State<CreateClaudeAccountScreen> createState() =>
      _CreateClaudeAccountScreenState();
}

class _CreateClaudeAccountScreenState
    extends State<CreateClaudeAccountScreen> {
  final _name = TextEditingController();
  final _display = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    _display.dispose();
    super.dispose();
  }

  void _submit() {
    final name = _name.text.trim();
    if (name.isEmpty) {
      setState(() => _error = 'Name is required.');
      return;
    }
    Navigator.of(context).pop(_CreateAccountResult(
      name: name,
      displayName: _display.text.trim(),
    ));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Add Claude account'),
        actions: [
          TextButton(onPressed: _submit, child: const Text('Add')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Container(
            padding: const EdgeInsets.all(12),
            margin: const EdgeInsets.only(bottom: 16),
            decoration: BoxDecoration(
              color: Theme.of(context)
                  .colorScheme
                  .tertiary
                  .withValues(alpha: 0.10),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Row(
                  children: [
                    Icon(
                      Icons.info_outline,
                      size: 16,
                      color: Theme.of(context).colorScheme.tertiary,
                    ),
                    const SizedBox(width: 6),
                    Text(
                      'This only reserves the row.',
                      style: TextStyle(
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                        color: Theme.of(context).colorScheme.tertiary,
                      ),
                    ),
                  ],
                ),
                const SizedBox(height: 6),
                Text(
                  'The OAuth login itself runs on the gateway host. '
                  'After saving, SSH into the gateway and run:',
                  style: TextStyle(
                    fontSize: 11,
                    height: 1.4,
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.75),
                  ),
                ),
                const SizedBox(height: 6),
                Container(
                  padding: const EdgeInsets.all(8),
                  decoration: BoxDecoration(
                    color: Theme.of(context)
                        .colorScheme
                        .surface
                        .withValues(alpha: 0.5),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: const SelectableText(
                    'mkdir -p ~/.claude-accounts/<name>\n'
                    'CLAUDE_CONFIG_DIR=~/.claude-accounts/<name> claude login',
                    style: TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 10.5,
                      height: 1.4,
                    ),
                  ),
                ),
                const SizedBox(height: 6),
                Text(
                  'opendray watches the directory and links the row '
                  'automatically. See the Providers → Claude accounts '
                  'section in the web tutorial for the full guide.',
                  style: TextStyle(
                    fontSize: 11,
                    height: 1.4,
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.75),
                  ),
                ),
              ],
            ),
          ),
          TextField(
            controller: _name,
            autofocus: true,
            autocorrect: false,
            decoration: const InputDecoration(
              labelText: 'Name (slug)',
              hintText: 'work, personal, …',
              helperText: 'Lowercase id used in spawn picker.',
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),
          TextField(
            controller: _display,
            decoration: const InputDecoration(
              labelText: 'Display name (optional)',
              hintText: 'Work account',
              border: OutlineInputBorder(),
            ),
          ),
          if (_error != null) ...[
            const SizedBox(height: 8),
            Text(
              _error!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.error,
                fontSize: 12,
              ),
            ),
          ],
        ],
      ),
    );
  }
}

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
