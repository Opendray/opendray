import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:opendray/core/api/integrations_api.dart';

// Dialogs/forms shared by IntegrationsScreen (register via FAB) and
// IntegrationDetailScreen (edit via overflow menu, reveal-once after
// rotate-key). Kept in one place because the field validation rules
// for base_url / route_prefix / scopes are identical across register
// and edit, and the reveal dialog is the same UI in both flows.

// _RegisterDialog asks for the five fields the server requires (name,
// base_url, route_prefix) plus the two optional ones (scopes, version).
// Returns the form values; the caller invokes the API and handles the
// reveal-once flow.
class RegisterIntegrationDialog extends StatefulWidget {
  const RegisterIntegrationDialog({super.key});

  static Future<RegisterIntegrationFormResult?> show(
    BuildContext context,
  ) {
    return showDialog<RegisterIntegrationFormResult>(
      context: context,
      builder: (_) => const RegisterIntegrationDialog(),
    );
  }

  @override
  State<RegisterIntegrationDialog> createState() =>
      _RegisterIntegrationDialogState();
}

class _RegisterIntegrationDialogState
    extends State<RegisterIntegrationDialog> {
  final _name = TextEditingController();
  final _baseUrl = TextEditingController();
  final _prefix = TextEditingController();
  final _scopes = TextEditingController();
  final _version = TextEditingController();
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    _baseUrl.dispose();
    _prefix.dispose();
    _scopes.dispose();
    _version.dispose();
    super.dispose();
  }

  void _submit() {
    final name = _name.text.trim();
    final baseUrl = _baseUrl.text.trim();
    final prefix = _prefix.text.trim();
    if (name.isEmpty || baseUrl.isEmpty || prefix.isEmpty) {
      setState(() =>
          _error = 'Name, base URL, and route prefix are required.');
      return;
    }
    final scopes = _scopes.text
        .split(',')
        .map((s) => s.trim())
        .where((s) => s.isNotEmpty)
        .toList();
    Navigator.of(context).pop(
      RegisterIntegrationFormResult(
        name: name,
        baseUrl: baseUrl,
        routePrefix: prefix.replaceAll(RegExp(r'^/+|/+$'), ''),
        scopes: scopes,
        version: _version.text.trim(),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: const Text('Register integration'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _Field(
              controller: _name,
              label: 'Name',
              hint: 'My Bot',
              autofocus: true,
            ),
            _Field(
              controller: _baseUrl,
              label: 'Base URL',
              hint: 'https://api.example.com',
              keyboardType: TextInputType.url,
            ),
            _Field(
              controller: _prefix,
              label: 'Route prefix',
              hint: 'mybot',
              helper: 'Reachable as /api/v1/<prefix>/...',
            ),
            _Field(
              controller: _scopes,
              label: 'Scopes (optional)',
              hint: 'session:read, session:events',
              helper: 'Comma-separated. Empty = server defaults.',
            ),
            _Field(
              controller: _version,
              label: 'Version (optional)',
              hint: '1.0.0',
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
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(onPressed: _submit, child: const Text('Register')),
      ],
    );
  }
}

class RegisterIntegrationFormResult {
  RegisterIntegrationFormResult({
    required this.name,
    required this.baseUrl,
    required this.routePrefix,
    required this.scopes,
    required this.version,
  });
  final String name;
  final String baseUrl;
  final String routePrefix;
  final List<String> scopes;
  final String version;
}

// EditIntegrationDialog patches base_url / scopes / version / enabled.
// Pre-fills from the existing record and only PATCHes the fields the
// operator actually changed (the API tolerates omitted fields).
class EditIntegrationDialog extends StatefulWidget {
  const EditIntegrationDialog({required this.current, super.key});
  final Integration current;

  static Future<EditIntegrationFormResult?> show(
    BuildContext context,
    Integration current,
  ) {
    return showDialog<EditIntegrationFormResult>(
      context: context,
      builder: (_) => EditIntegrationDialog(current: current),
    );
  }

  @override
  State<EditIntegrationDialog> createState() => _EditIntegrationDialogState();
}

class _EditIntegrationDialogState extends State<EditIntegrationDialog> {
  late final TextEditingController _baseUrl;
  late final TextEditingController _scopes;
  late final TextEditingController _version;
  late bool _enabled;
  String? _error;

  @override
  void initState() {
    super.initState();
    _baseUrl = TextEditingController(text: widget.current.baseUrl);
    _scopes = TextEditingController(text: widget.current.scopes.join(', '));
    _version = TextEditingController(text: widget.current.version ?? '');
    _enabled = widget.current.enabled;
  }

  @override
  void dispose() {
    _baseUrl.dispose();
    _scopes.dispose();
    _version.dispose();
    super.dispose();
  }

  void _submit() {
    final baseUrl = _baseUrl.text.trim();
    if (baseUrl.isEmpty) {
      setState(() => _error = 'Base URL is required.');
      return;
    }
    final scopes = _scopes.text
        .split(',')
        .map((s) => s.trim())
        .where((s) => s.isNotEmpty)
        .toList();
    final version = _version.text.trim();
    final initialScopes = widget.current.scopes;
    final scopesChanged = scopes.length != initialScopes.length ||
        !scopes.asMap().entries.every((e) => e.value == initialScopes[e.key]);
    Navigator.of(context).pop(
      EditIntegrationFormResult(
        baseUrl: baseUrl != widget.current.baseUrl ? baseUrl : null,
        scopes: scopesChanged ? scopes : null,
        version: version != (widget.current.version ?? '') ? version : null,
        enabled: _enabled != widget.current.enabled ? _enabled : null,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text('Edit ${widget.current.name}'),
      content: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _Field(
              controller: _baseUrl,
              label: 'Base URL',
              keyboardType: TextInputType.url,
            ),
            _Field(
              controller: _scopes,
              label: 'Scopes',
              helper: 'Comma-separated.',
            ),
            _Field(
              controller: _version,
              label: 'Version',
            ),
            const SizedBox(height: 8),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: const Text('Enabled'),
              value: _enabled,
              onChanged: (v) => setState(() => _enabled = v),
            ),
            if (_error != null)
              Text(
                _error!,
                style: TextStyle(
                  color: Theme.of(context).colorScheme.error,
                  fontSize: 12,
                ),
              ),
          ],
        ),
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

class EditIntegrationFormResult {
  EditIntegrationFormResult({
    this.baseUrl,
    this.scopes,
    this.version,
    this.enabled,
  });
  final String? baseUrl;
  final List<String>? scopes;
  final String? version;
  final bool? enabled;

  bool get isEmpty =>
      baseUrl == null && scopes == null && version == null && enabled == null;
}

// RevealApiKeyDialog displays a freshly-minted API key once. The
// "I've saved it" button is grey until copy is tapped — this is the
// only chance to capture the plaintext. Used by both the register
// and rotate-key flows.
class RevealApiKeyDialog extends StatefulWidget {
  const RevealApiKeyDialog({
    required this.apiKey,
    required this.title,
    required this.subtitle,
    super.key,
  });

  final String apiKey;
  final String title;
  final String subtitle;

  static Future<void> show({
    required BuildContext context,
    required String apiKey,
    required String title,
    required String subtitle,
  }) {
    return showDialog<void>(
      context: context,
      barrierDismissible: false,
      builder: (_) => RevealApiKeyDialog(
        apiKey: apiKey,
        title: title,
        subtitle: subtitle,
      ),
    );
  }

  @override
  State<RevealApiKeyDialog> createState() => _RevealApiKeyDialogState();
}

class _RevealApiKeyDialogState extends State<RevealApiKeyDialog> {
  bool _copied = false;

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(widget.title),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            widget.subtitle,
            style: Theme.of(context).textTheme.bodySmall,
          ),
          const SizedBox(height: 12),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: Theme.of(context)
                  .colorScheme
                  .surfaceContainerHighest
                  .withValues(alpha: 0.6),
              borderRadius: BorderRadius.circular(6),
            ),
            child: SelectableText(
              widget.apiKey,
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 12,
              ),
            ),
          ),
          const SizedBox(height: 8),
          Row(
            children: [
              Icon(
                Icons.warning_amber_rounded,
                size: 16,
                color: Theme.of(context).colorScheme.tertiary,
              ),
              const SizedBox(width: 6),
              Expanded(
                child: Text(
                  "You won't see this key again.",
                  style: TextStyle(
                    fontSize: 11,
                    color: Theme.of(context).colorScheme.tertiary,
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
      actions: [
        TextButton.icon(
          icon: Icon(
            _copied ? Icons.check : Icons.copy_outlined,
            size: 18,
          ),
          label: Text(_copied ? 'Copied' : 'Copy'),
          onPressed: () async {
            await Clipboard.setData(ClipboardData(text: widget.apiKey));
            if (!mounted) return;
            setState(() => _copied = true);
          },
        ),
        FilledButton(
          onPressed: _copied ? () => Navigator.of(context).pop() : null,
          child: const Text("I've saved it"),
        ),
      ],
    );
  }
}

class _Field extends StatelessWidget {
  const _Field({
    required this.controller,
    required this.label,
    this.hint,
    this.helper,
    this.keyboardType,
    this.autofocus = false,
  });

  final TextEditingController controller;
  final String label;
  final String? hint;
  final String? helper;
  final TextInputType? keyboardType;
  final bool autofocus;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: TextField(
        controller: controller,
        autofocus: autofocus,
        autocorrect: false,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          helperText: helper,
          isDense: true,
        ),
      ),
    );
  }
}
