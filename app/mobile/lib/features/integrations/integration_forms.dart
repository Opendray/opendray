import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:opendray/core/api/integrations_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Form pages and small read-only dialogs shared by the Integrations
// screens. Multi-field forms (register / edit) are full-screen pages
// because AlertDialog leaves only half the screen above the keyboard
// on phones, which crushes a five-input form. Read-only or
// single-confirmation flows (RevealApiKey) stay as dialogs.

// RegisterIntegrationScreen asks for the five fields the server
// requires (name, base_url, route_prefix) plus the two optional ones
// (scopes, version). Returns the form values; the caller invokes the
// API and handles the reveal-once flow.
class RegisterIntegrationScreen extends StatefulWidget {
  const RegisterIntegrationScreen({super.key});

  static Future<RegisterIntegrationFormResult?> push(BuildContext context) {
    return Navigator.of(context).push<RegisterIntegrationFormResult>(
      MaterialPageRoute<RegisterIntegrationFormResult>(
        builder: (_) => const RegisterIntegrationScreen(),
        fullscreenDialog: true,
      ),
    );
  }

  @override
  State<RegisterIntegrationScreen> createState() =>
      _RegisterIntegrationScreenState();
}

class _RegisterIntegrationScreenState
    extends State<RegisterIntegrationScreen> {
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
      setState(
        () => _error = t.integrations.form.validateRequired,
      );
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
    return Scaffold(
      appBar: AppBar(
        title: Text(t.integrations.registerDialogTitle),
        actions: [
          TextButton(
            onPressed: _submit,
            child: Text(t.integrations.register),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          _Field(
            controller: _name,
            label: t.integrations.form.fieldName,
            hint: t.integrations.form.fieldNameHint,
            autofocus: true,
          ),
          _Field(
            controller: _baseUrl,
            label: t.integrations.form.fieldBaseUrl,
            hint: 'https://api.example.com',
            keyboardType: TextInputType.url,
          ),
          _Field(
            controller: _prefix,
            label: t.integrations.form.fieldRoutePrefix,
            hint: 'mybot',
            helper: t.integrations.form.routePrefixHelper,
          ),
          _Field(
            controller: _scopes,
            label: t.integrations.form.fieldScopes,
            hint: 'session:read, session:events',
            helper: t.integrations.form.scopesHelper,
          ),
          _Field(
            controller: _version,
            label: t.integrations.form.fieldVersion,
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

// EditIntegrationScreen patches base_url / scopes / version / enabled.
// Pre-fills from the existing record and only reports the fields the
// operator actually changed (the API tolerates omitted fields).
class EditIntegrationScreen extends StatefulWidget {
  const EditIntegrationScreen({required this.current, super.key});
  final Integration current;

  static Future<EditIntegrationFormResult?> push(
    BuildContext context,
    Integration current,
  ) {
    return Navigator.of(context).push<EditIntegrationFormResult>(
      MaterialPageRoute<EditIntegrationFormResult>(
        builder: (_) => EditIntegrationScreen(current: current),
        fullscreenDialog: true,
      ),
    );
  }

  @override
  State<EditIntegrationScreen> createState() => _EditIntegrationScreenState();
}

class _EditIntegrationScreenState extends State<EditIntegrationScreen> {
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
      setState(() => _error = t.integrations.form.validateBaseUrl);
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
    return Scaffold(
      appBar: AppBar(
        title: Text(t.integrations.editTitle(name: widget.current.name)),
        actions: [
          TextButton(
            onPressed: _submit,
            child: Text(t.common.save),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          _Field(
            controller: _baseUrl,
            label: t.integrations.form.fieldBaseUrl,
            keyboardType: TextInputType.url,
          ),
          _Field(
            controller: _scopes,
            label: t.integrations.form.editFieldScopes,
            helper: t.integrations.form.editScopesHelper,
          ),
          _Field(
            controller: _version,
            label: t.integrations.form.editFieldVersion,
          ),
          const SizedBox(height: 8),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Text(t.integrations.enabledLabel),
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
// only chance to capture the plaintext. Stays a dialog because it's
// short-form, read-only, and the user is making a single decision.
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
                  t.integrations.form.apiKeyWarn,
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
          label: Text(_copied
              ? t.integrations.form.copyCopied
              : t.integrations.form.copyCopy),
          onPressed: () async {
            await Clipboard.setData(ClipboardData(text: widget.apiKey));
            if (!mounted) return;
            setState(() => _copied = true);
          },
        ),
        FilledButton(
          onPressed: _copied ? () => Navigator.of(context).pop() : null,
          child: Text(t.integrations.iSavedIt),
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
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: controller,
        autofocus: autofocus,
        autocorrect: false,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          helperText: helper,
          helperMaxLines: 2,
          border: const OutlineInputBorder(),
        ),
      ),
    );
  }
}
