import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:opendray/core/api/integrations_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/integrations/default_agent_fields.dart';
import 'package:opendray/features/integrations/scope_picker.dart';

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
  // Default grant mirrors the web register dialog.
  List<String> _scopes = const ['session:read', 'event:subscribe:session.*'];
  final _version = TextEditingController();
  final _defaultModel = TextEditingController();
  String _defaultProviderId = '';
  String _defaultClaudeAccountId = '';
  String? _error;

  @override
  void dispose() {
    _name.dispose();
    _baseUrl.dispose();
    _prefix.dispose();
    _version.dispose();
    _defaultModel.dispose();
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
    Navigator.of(context).pop(
      RegisterIntegrationFormResult(
        name: name,
        baseUrl: baseUrl,
        routePrefix: prefix.replaceAll(RegExp(r'^/+|/+$'), ''),
        scopes: _scopes,
        version: _version.text.trim(),
        defaultProviderId: _defaultProviderId,
        defaultModel: _defaultModel.text.trim(),
        defaultClaudeAccountId: _defaultClaudeAccountId,
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
            controller: _version,
            label: t.integrations.form.fieldVersion,
            hint: '1.0.0',
          ),
          const SizedBox(height: 4),
          ScopePicker(
            selected: _scopes,
            intro: t.web.integrations.register_dialog.scopesIntro,
            onChanged: (next) => setState(() => _scopes = next),
          ),
          const SizedBox(height: 12),
          DefaultAgentFields(
            providerId: _defaultProviderId,
            claudeAccountId: _defaultClaudeAccountId,
            modelController: _defaultModel,
            onProviderChanged: (v) => setState(() {
              _defaultProviderId = v;
              // The account default is only meaningful for claude.
              if (v != 'claude') _defaultClaudeAccountId = '';
            }),
            onAccountChanged: (v) =>
                setState(() => _defaultClaudeAccountId = v),
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
    this.defaultProviderId = '',
    this.defaultModel = '',
    this.defaultClaudeAccountId = '',
  });
  final String name;
  final String baseUrl;
  final String routePrefix;
  final List<String> scopes;
  final String version;
  final String defaultProviderId;
  final String defaultModel;
  final String defaultClaudeAccountId;
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
  late List<String> _scopes;
  late final TextEditingController _version;
  late final TextEditingController _defaultModel;
  late final TextEditingController _systemPrompt;
  late final TextEditingController _mcpServers;
  late bool _enabled;
  late String _defaultProviderId;
  late String _defaultClaudeAccountId;
  late bool _bypassPermissions;
  String? _error;

  // The pretty-printed JSON the mcp_servers textarea started with, so
  // _submit can tell whether the operator actually edited it.
  late final String _mcpServersInitial;

  @override
  void initState() {
    super.initState();
    _baseUrl = TextEditingController(text: widget.current.baseUrl);
    _scopes = List.of(widget.current.scopes);
    _version = TextEditingController(text: widget.current.version ?? '');
    _defaultModel =
        TextEditingController(text: widget.current.defaultModel);
    _systemPrompt =
        TextEditingController(text: widget.current.systemPrompt);
    _mcpServersInitial = const JsonEncoder.withIndent('  ')
        .convert(widget.current.mcpServers);
    _mcpServers = TextEditingController(text: _mcpServersInitial);
    _enabled = widget.current.enabled;
    _defaultProviderId = widget.current.defaultProviderId;
    _defaultClaudeAccountId = widget.current.defaultClaudeAccountId;
    _bypassPermissions = widget.current.bypassPermissions;
  }

  @override
  void dispose() {
    _baseUrl.dispose();
    _version.dispose();
    _defaultModel.dispose();
    _systemPrompt.dispose();
    _mcpServers.dispose();
    super.dispose();
  }

  void _submit() {
    final baseUrl = _baseUrl.text.trim();
    if (baseUrl.isEmpty) {
      setState(() => _error = t.integrations.form.validateBaseUrl);
      return;
    }
    final scopes = _scopes;
    final version = _version.text.trim();
    final initialScopes = widget.current.scopes;
    final scopesChanged = scopes.length != initialScopes.length ||
        !scopes.asMap().entries.every((e) => e.value == initialScopes[e.key]);
    final model = _defaultModel.text.trim();
    final systemPrompt = _systemPrompt.text;
    // Parse the mcp_servers JSON textarea. Must be a JSON array; a
    // malformed body or non-list aborts the submit with an inline error.
    final mcpText = _mcpServers.text;
    final mcpChanged = mcpText != _mcpServersInitial;
    List<dynamic>? mcpServers;
    if (mcpChanged) {
      final trimmed = mcpText.trim();
      try {
        final decoded = trimmed.isEmpty ? <dynamic>[] : jsonDecode(trimmed);
        if (decoded is! List) {
          setState(
              () => _error = t.web.integrations.edit_dialog.mcpServersInvalid);
          return;
        }
        mcpServers = decoded;
      } on FormatException {
        setState(
            () => _error = t.web.integrations.edit_dialog.mcpServersInvalid);
        return;
      }
    }
    Navigator.of(context).pop(
      EditIntegrationFormResult(
        baseUrl: baseUrl != widget.current.baseUrl ? baseUrl : null,
        scopes: scopesChanged ? scopes : null,
        version: version != (widget.current.version ?? '') ? version : null,
        enabled: _enabled != widget.current.enabled ? _enabled : null,
        defaultProviderId:
            _defaultProviderId != widget.current.defaultProviderId
                ? _defaultProviderId
                : null,
        defaultModel:
            model != widget.current.defaultModel ? model : null,
        defaultClaudeAccountId:
            _defaultClaudeAccountId != widget.current.defaultClaudeAccountId
                ? _defaultClaudeAccountId
                : null,
        systemPrompt:
            systemPrompt != widget.current.systemPrompt ? systemPrompt : null,
        mcpServers: mcpChanged ? mcpServers : null,
        bypassPermissions:
            _bypassPermissions != widget.current.bypassPermissions
                ? _bypassPermissions
                : null,
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
            controller: _version,
            label: t.integrations.form.editFieldVersion,
          ),
          const SizedBox(height: 8),
          ScopePicker(
            selected: _scopes,
            intro: t.web.integrations.edit_dialog.scopesIntro,
            onChanged: (next) => setState(() => _scopes = next),
          ),
          const SizedBox(height: 12),
          DefaultAgentFields(
            providerId: _defaultProviderId,
            claudeAccountId: _defaultClaudeAccountId,
            modelController: _defaultModel,
            onProviderChanged: (v) => setState(() {
              _defaultProviderId = v;
              if (v != 'claude') _defaultClaudeAccountId = '';
            }),
            onAccountChanged: (v) =>
                setState(() => _defaultClaudeAccountId = v),
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Text(t.integrations.enabledLabel),
            value: _enabled,
            onChanged: (v) => setState(() => _enabled = v),
          ),
          const SizedBox(height: 8),
          _Field(
            controller: _systemPrompt,
            label: t.web.integrations.edit_dialog.systemPromptLabel,
            helper: t.web.integrations.edit_dialog.systemPromptHint,
            maxLines: 6,
          ),
          _Field(
            controller: _mcpServers,
            label: t.web.integrations.edit_dialog.mcpServersLabel,
            helper: t.web.integrations.edit_dialog.mcpServersHint,
            maxLines: 8,
            monospace: true,
          ),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Text(t.web.integrations.edit_dialog.bypassPermissionsLabel),
            subtitle: Text(t.web.integrations.edit_dialog.bypassPermissionsHint),
            value: _bypassPermissions,
            onChanged: (v) => setState(() => _bypassPermissions = v),
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
    this.defaultProviderId,
    this.defaultModel,
    this.defaultClaudeAccountId,
    this.systemPrompt,
    this.mcpServers,
    this.bypassPermissions,
  });
  final String? baseUrl;
  final List<String>? scopes;
  final String? version;
  final bool? enabled;
  final String? defaultProviderId;
  final String? defaultModel;
  final String? defaultClaudeAccountId;
  final String? systemPrompt;
  final List<dynamic>? mcpServers;
  final bool? bypassPermissions;

  bool get isEmpty =>
      baseUrl == null &&
      scopes == null &&
      version == null &&
      enabled == null &&
      defaultProviderId == null &&
      defaultModel == null &&
      defaultClaudeAccountId == null &&
      systemPrompt == null &&
      mcpServers == null &&
      bypassPermissions == null;
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
    this.maxLines = 1,
    this.monospace = false,
  });

  final TextEditingController controller;
  final String label;
  final String? hint;
  final String? helper;
  final TextInputType? keyboardType;
  final bool autofocus;
  final int maxLines;
  final bool monospace;

  @override
  Widget build(BuildContext context) {
    final multiline = maxLines > 1;
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: controller,
        autofocus: autofocus,
        autocorrect: false,
        maxLines: maxLines,
        keyboardType:
            multiline ? TextInputType.multiline : keyboardType,
        style: monospace
            ? const TextStyle(fontFamily: 'monospace', fontSize: 12)
            : null,
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
