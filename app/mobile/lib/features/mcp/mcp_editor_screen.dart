// MCP server JSON editor — full-screen create/edit form for the
// `server` body of POST /mcps and PUT /mcps/{id}. Mirrors the web
// McpEditor's pretty/parse pattern: the textarea holds everything
// EXCEPT the id (id is a separate field, doubles as the URL
// segment server-side).
//
// Validation is server-side: a malformed body returns 400 with the
// parse error, surfaced inline. We don't try to schema-check
// client-side — the server's schema evolves and duplicating it
// here would just rot.

import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/mcp_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

class McpEditorScreen extends ConsumerStatefulWidget {
  const McpEditorScreen({super.key, this.existing});

  /// When non-null, edit mode — the JSON textarea is pre-filled
  /// from the server, the id field is locked.
  final McpServer? existing;

  @override
  ConsumerState<McpEditorScreen> createState() => _McpEditorScreenState();
}

class _McpEditorScreenState extends ConsumerState<McpEditorScreen> {
  final _idCtrl = TextEditingController();
  final _bodyCtrl = TextEditingController();
  bool _submitting = false;
  String? _error;

  bool get _isEdit => widget.existing != null;

  @override
  void initState() {
    super.initState();
    final ex = widget.existing;
    if (ex != null) {
      _idCtrl.text = ex.id;
      _bodyCtrl.text = _prettyServer(ex);
    } else {
      _bodyCtrl.text = _scaffoldJson();
    }
  }

  @override
  void dispose() {
    _idCtrl.dispose();
    _bodyCtrl.dispose();
    super.dispose();
  }

  // Build the JSON body sans `id` (the id is the URL segment + the
  // editor's top field, not part of the body textarea).
  static String _prettyServer(McpServer s) {
    final map = <String, dynamic>{
      'name': s.name,
      if (s.description != null) 'description': s.description,
      'transport': s.transport,
      if (s.command != null) 'command': s.command,
      if (s.args != null) 'args': s.args,
      if (s.env != null) 'env': s.env,
      if (s.url != null) 'url': s.url,
      if (s.headers != null) 'headers': s.headers,
      'enabled': s.enabled,
    };
    return const JsonEncoder.withIndent('  ').convert(map);
  }

  // Starter template for a new stdio MCP server — the operator
  // edits this into shape rather than facing a blank box.
  static String _scaffoldJson() {
    final example = {
      'name': 'my-mcp-server',
      'description': t.mcp.editor.descriptionPlaceholder,
      'transport': 'stdio',
      'command': '/path/to/binary',
      'args': ['--arg1'],
      'env': {'KEY': r'$secret:KEY'},
      'enabled': true,
    };
    return const JsonEncoder.withIndent('  ').convert(example);
  }

  Future<void> _submit() async {
    final id = _idCtrl.text.trim();
    if (id.isEmpty) {
      setState(() => _error = t.mcp.editor.idRequired);
      return;
    }
    Map<String, dynamic> parsed;
    try {
      final decoded = jsonDecode(_bodyCtrl.text);
      if (decoded is! Map<String, dynamic>) {
        setState(() => _error = t.mcp.editor.validateJsonObject);
        return;
      }
      parsed = decoded;
    } on FormatException catch (e) {
      setState(() => _error = t.mcp.editor.validateJsonInvalid(error: e.message));
      return;
    }
    // Inject id into the server body; the server requires
    // body.id == URL id, web's parseMcp does the same.
    parsed['id'] = id;
    parsed['name'] = parsed['name'] ?? id;
    parsed['enabled'] = parsed['enabled'] ?? true;

    setState(() {
      _submitting = true;
      _error = null;
    });
    try {
      final api = ref.read(mcpApiProvider);
      if (_isEdit) {
        await api.replace(id: id, server: parsed);
      } else {
        await api.create(id: id, server: parsed);
      }
      if (!mounted) return;
      Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _error = e.message;
          _submitting = false;
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _submitting = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(_isEdit ? t.mcp.editor.appBarEdit : t.mcp.editor.appBarNew),
      ),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          children: [
            Text(
              t.mcp.editor.idLabel,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 4),
            TextField(
              controller: _idCtrl,
              enabled: !_isEdit && !_submitting,
              autocorrect: false,
              textCapitalization: TextCapitalization.none,
              decoration: InputDecoration(
                hintText: t.mcp.editor.nameHint,
                helperText: _isEdit ? t.mcp.editor.idLockedHint : null,
              ),
              style: const TextStyle(fontFamily: 'monospace'),
            ),
            const SizedBox(height: 14),
            Text(
              t.mcp.editor.jsonLabel,
              style: theme.textTheme.bodySmall,
            ),
            const SizedBox(height: 4),
            TextField(
              controller: _bodyCtrl,
              enabled: !_submitting,
              autocorrect: false,
              maxLines: 18,
              minLines: 10,
              keyboardType: TextInputType.multiline,
              decoration: InputDecoration(
                hintText: t.mcp.editor.jsonHint,
              ),
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 12,
                height: 1.4,
              ),
              inputFormatters: const [],
            ),
            const SizedBox(height: 6),
            Text(
              t.mcp.editor.jsonSchemaHelp,
              style: theme.textTheme.bodySmall?.copyWith(fontSize: 11),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: theme.colorScheme.error.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(
                    color: theme.colorScheme.error.withValues(alpha: 0.4),
                  ),
                ),
                child: SelectableText(
                  _error!,
                  style: TextStyle(
                    color: theme.colorScheme.error,
                    fontFamily: 'monospace',
                    fontSize: 12,
                  ),
                ),
              ),
            ],
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _submitting
                        ? null
                        : () => Navigator.of(context).pop(false),
                    child: Text(t.common.cancel),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton.icon(
                    onPressed: _submitting ? null : _submit,
                    icon: _submitting
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.check, size: 18),
                    label: Text(_submitting
                        ? t.mcp.editor.saving
                        : (_isEdit
                            ? t.mcp.editor.save
                            : t.mcp.editor.create)),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
