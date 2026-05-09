import 'package:flutter/material.dart';

import 'package:opendray/core/api/models.dart';

// Form dialogs for Claude accounts. Add (name + optional displayName +
// optional token), Rename (just display_name), Set token (paste OAuth
// JSON or bare access_token).

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

class CreateClaudeAccountDialog extends StatefulWidget {
  const CreateClaudeAccountDialog({super.key});

  static Future<({String name, String displayName, String token})?> show(
    BuildContext context,
  ) async {
    final res = await showDialog<_CreateAccountResult>(
      context: context,
      builder: (_) => const CreateClaudeAccountDialog(),
    );
    if (res == null) return null;
    return (
      name: res.name,
      displayName: res.displayName,
      token: res.token,
    );
  }

  @override
  State<CreateClaudeAccountDialog> createState() =>
      _CreateClaudeAccountDialogState();
}

class _CreateClaudeAccountDialogState
    extends State<CreateClaudeAccountDialog> {
  final _name = TextEditingController();
  final _display = TextEditingController();
  final _token = TextEditingController();
  bool _hideToken = true;
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
    return AlertDialog(
      title: const Text('Add Claude account'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            TextField(
              controller: _name,
              autofocus: true,
              autocorrect: false,
              decoration: const InputDecoration(
                labelText: 'Name (slug)',
                hintText: 'work, personal, …',
                helperText: 'Lowercase id used in spawn picker.',
                isDense: true,
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _display,
              decoration: const InputDecoration(
                labelText: 'Display name (optional)',
                hintText: 'Work account',
                isDense: true,
              ),
            ),
            const SizedBox(height: 8),
            TextField(
              controller: _token,
              maxLines: 4,
              autocorrect: false,
              obscureText: _hideToken,
              decoration: InputDecoration(
                labelText: 'OAuth token (optional)',
                hintText:
                    '{"access_token":"…","refresh_token":"…"} or bare token',
                helperText: 'Leave blank to add the row first and set the '
                    'token later.',
                helperMaxLines: 2,
                isDense: true,
                suffixIcon: IconButton(
                  icon: Icon(
                    _hideToken
                        ? Icons.visibility_outlined
                        : Icons.visibility_off_outlined,
                    size: 18,
                  ),
                  onPressed: () => setState(() => _hideToken = !_hideToken),
                ),
              ),
            ),
            const SizedBox(height: 6),
            Text(
              'Tip: Claude exports the OAuth blob as a single JSON; '
              'paste the whole object.',
              style: muted,
            ),
            if (_error != null) ...[
              const SizedBox(height: 6),
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
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Add')),
      ],
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

class SetClaudeTokenDialog extends StatefulWidget {
  const SetClaudeTokenDialog({required this.account, super.key});
  final ClaudeAccountSummary account;

  static Future<String?> show(
    BuildContext context,
    ClaudeAccountSummary account,
  ) {
    return showDialog<String>(
      context: context,
      builder: (_) => SetClaudeTokenDialog(account: account),
    );
  }

  @override
  State<SetClaudeTokenDialog> createState() => _SetClaudeTokenDialogState();
}

class _SetClaudeTokenDialogState extends State<SetClaudeTokenDialog> {
  final _ctrl = TextEditingController();
  bool _hide = true;
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
    return AlertDialog(
      title: Text(
        widget.account.tokenFilled
            ? 'Replace token for ${widget.account.name}'
            : 'Set token for ${widget.account.name}',
      ),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          TextField(
            controller: _ctrl,
            autofocus: true,
            autocorrect: false,
            obscureText: _hide,
            maxLines: 5,
            decoration: InputDecoration(
              labelText: 'OAuth blob or access_token',
              hintText: '{"access_token":"…","refresh_token":"…"}',
              helperText: widget.account.tokenFilled
                  ? 'Replaces the existing token. The previous one is wiped.'
                  : null,
              helperMaxLines: 2,
              isDense: true,
              suffixIcon: IconButton(
                icon: Icon(
                  _hide
                      ? Icons.visibility_outlined
                      : Icons.visibility_off_outlined,
                  size: 18,
                ),
                onPressed: () => setState(() => _hide = !_hide),
              ),
            ),
          ),
          if (_error != null) ...[
            const SizedBox(height: 6),
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
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Save')),
      ],
    );
  }
}
