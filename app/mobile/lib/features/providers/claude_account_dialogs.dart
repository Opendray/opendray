import 'package:flutter/material.dart';

import 'package:opendray/core/api/models.dart';

// Claude account form pages and dialogs. Multi-field forms (Add,
// Set-token) are full-screen pages because the OAuth blob alone wants
// 4-5 visible lines and a dialog can't give that above the keyboard
// on a phone. Rename is a single-field flow → stays a dialog.

class _CreateAccountResult {
  _CreateAccountResult({
    required this.name,
    required this.displayName,
    required this.token,
  });
  final String name;
  final String displayName;
  final String token;
}

class CreateClaudeAccountScreen extends StatefulWidget {
  const CreateClaudeAccountScreen({super.key});

  static Future<({String name, String displayName, String token})?> push(
    BuildContext context,
  ) async {
    final res = await Navigator.of(context).push<_CreateAccountResult>(
      MaterialPageRoute<_CreateAccountResult>(
        builder: (_) => const CreateClaudeAccountScreen(),
        fullscreenDialog: true,
      ),
    );
    if (res == null) return null;
    return (
      name: res.name,
      displayName: res.displayName,
      token: res.token,
    );
  }

  @override
  State<CreateClaudeAccountScreen> createState() =>
      _CreateClaudeAccountScreenState();
}

class _CreateClaudeAccountScreenState
    extends State<CreateClaudeAccountScreen> {
  final _name = TextEditingController();
  final _display = TextEditingController();
  final _token = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    _display.dispose();
    _token.dispose();
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
      token: _token.text.trim(),
    ));
  }

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
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
                      'Where does the OAuth token come from?',
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
                  'On any machine where you ran "claude login", the JSON '
                  'is at ~/.claude/.credentials.json. If you used a '
                  'per-account directory, look under '
                  '~/.claude-accounts/<name>/.claude/.credentials.json. '
                  'Copy the whole object and paste below. Leave blank '
                  'to add the row first and set the token later.',
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
          const SizedBox(height: 12),
          TextField(
            controller: _token,
            maxLines: 5,
            autocorrect: false,
            // OAuth blobs are multi-line JSON; obscureText would
            // require maxLines==1 (Flutter asserts otherwise) and
            // the operator just pasted it from elsewhere anyway, so
            // showing it lets them sanity-check it landed cleanly.
            decoration: const InputDecoration(
              labelText: 'OAuth token (optional)',
              hintText:
                  '{"access_token":"…","refresh_token":"…"} or bare token',
              helperText: 'Leave blank to add the row first and set the '
                  'token later.',
              helperMaxLines: 2,
              border: OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 8),
          Text(
            'Tip: Claude exports the OAuth blob as a single JSON; '
            'paste the whole object.',
            style: muted,
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

class SetClaudeTokenScreen extends StatefulWidget {
  const SetClaudeTokenScreen({required this.account, super.key});
  final ClaudeAccountSummary account;

  static Future<String?> push(
    BuildContext context,
    ClaudeAccountSummary account,
  ) {
    return Navigator.of(context).push<String>(
      MaterialPageRoute<String>(
        builder: (_) => SetClaudeTokenScreen(account: account),
        fullscreenDialog: true,
      ),
    );
  }

  @override
  State<SetClaudeTokenScreen> createState() => _SetClaudeTokenScreenState();
}

class _SetClaudeTokenScreenState extends State<SetClaudeTokenScreen> {
  final _ctrl = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  void _submit() {
    final v = _ctrl.text.trim();
    if (v.isEmpty) {
      setState(() => _error = 'Token is required.');
      return;
    }
    Navigator.of(context).pop(v);
  }

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          widget.account.tokenFilled
              ? 'Replace token'
              : 'Set token',
        ),
        actions: [
          TextButton(onPressed: _submit, child: const Text('Save')),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Text(
            widget.account.name,
            style: Theme.of(context).textTheme.titleSmall,
          ),
          const SizedBox(height: 4),
          Text(
            widget.account.tokenFilled
                ? 'Replaces the existing token. The previous one is wiped.'
                : 'Pastes a new OAuth blob into the empty account row.',
            style: muted,
          ),
          const SizedBox(height: 16),
          TextField(
            controller: _ctrl,
            autofocus: true,
            autocorrect: false,
            maxLines: 8,
            // Visible by design — OAuth blobs are multi-line JSON
            // and obscureText would require maxLines==1.
            decoration: const InputDecoration(
              labelText: 'OAuth blob or access_token',
              hintText: '{"access_token":"…","refresh_token":"…"}',
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
