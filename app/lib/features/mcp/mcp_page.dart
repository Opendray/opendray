import 'dart:async';

import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// MCP Servers panel.
///
/// Lists MCP server definitions stored on the OpenDray server. Each entry can be
/// toggled, edited, or deleted. Enabled entries are rendered into a
/// per-session temp config at spawn and handed to the selected agents via
/// their native flags (e.g. claude `--mcp-config`, codex `CODEX_HOME`).
class MCPPage extends StatefulWidget {
  const MCPPage({super.key});
  @override
  State<MCPPage> createState() => _MCPPageState();
}

class _MCPPageState extends State<MCPPage> {
  List<Map<String, dynamic>> _servers = [];
  List<String> _agents = [];
  bool _loading = true;
  String? _error;
  StreamSubscription<void>? _providersSub;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _refresh();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _refresh());
  }

  @override
  void dispose() {
    _providersSub?.cancel();
    super.dispose();
  }

  Future<void> _refresh() async {
    try {
      final results = await Future.wait([_api.mcpServers(), _api.mcpAgents()]);
      if (!mounted) return;
      setState(() {
        _servers = results[0] as List<Map<String, dynamic>>;
        _agents  = results[1] as List<String>;
        _error   = null;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error   = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _openEditor([Map<String, dynamic>? server]) async {
    final changed = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (_) => _ServerEditorDialog(
        api: _api,
        initial: server,
        agents: _agents,
      ),
    );
    if (changed == true) _refresh();
  }

  Future<void> _toggle(Map<String, dynamic> s, bool enabled) async {
    try {
      await _api.mcpToggleServer(s['id'] as String, enabled);
      setState(() => s['enabled'] = enabled);
      ProvidersBus.instance.notify();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
      }
    }
  }

  Future<void> _delete(Map<String, dynamic> s) async {
    final ok = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (dialogCtx) => AlertDialog(
        title: Text(dialogCtx.tr('Delete MCP server?')),
        content: Text(
          dialogCtx.tr('This removes "@name" from OpenDray. Sessions already running keep their injected config.')
              .replaceAll('@name', s['name'] as String? ?? ''),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(dialogCtx, rootNavigator: true).pop(false),
            child: Text(dialogCtx.tr('Cancel')),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(dialogCtx, rootNavigator: true).pop(true),
            child: Text(dialogCtx.tr('Delete')),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.mcpDeleteServer(s['id'] as String);
      _refresh();
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_error != null) {
      return _emptyState(
        icon: Icons.error_outline,
        title: context.tr('Failed to load MCP servers'),
        body: _error!,
      );
    }
    return Scaffold(
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _openEditor(),
        backgroundColor: AppColors.accent,
        icon: const Icon(Icons.add),
        label: Text(context.tr('Add MCP server')),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: _servers.isEmpty
            ? ListView(children: [
                SizedBox(height: 60),
                _emptyState(
                  icon: Icons.electrical_services,
                  title: context.tr('No MCP servers yet'),
                  body: context.tr(
                    'Add a server entry — it will be injected into Claude / Codex sessions as a temporary config file at spawn time. Your user home configs stay untouched.',
                  ),
                ),
              ])
            : ListView.separated(
                padding: const EdgeInsets.fromLTRB(12, 12, 12, 96),
                itemCount: _servers.length,
                separatorBuilder: (_, _) => const SizedBox(height: 10),
                itemBuilder: (_, i) => _serverCard(_servers[i]),
              ),
      ),
    );
  }

  Widget _emptyState({required IconData icon, required String title, required String body}) {
    return Padding(
      padding: const EdgeInsets.all(28),
      child: Column(
        mainAxisAlignment: MainAxisAlignment.center,
        children: [
          Icon(icon, size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(title, style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 15)),
          const SizedBox(height: 8),
          Text(body, style: const TextStyle(color: AppColors.textMuted, fontSize: 12), textAlign: TextAlign.center),
        ],
      ),
    );
  }

  Widget _serverCard(Map<String, dynamic> s) {
    final enabled = s['enabled'] as bool? ?? true;
    final transport = s['transport'] as String? ?? 'stdio';
    final applies = (s['appliesTo'] as List?)?.cast<String>() ?? ['*'];
    final summary = _summarize(s);

    return Card(
      color: AppColors.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(color: enabled ? AppColors.accent.withValues(alpha: 0.25) : AppColors.border),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 12, 8, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Row(children: [
                        Flexible(
                          child: Text(
                            s['name'] as String? ?? '',
                            style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const SizedBox(width: 8),
                        _badge(transport),
                      ]),
                      if ((s['description'] as String?)?.isNotEmpty == true) ...[
                        const SizedBox(height: 4),
                        Text(
                          s['description'] as String,
                          style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
                        ),
                      ],
                    ],
                  ),
                ),
                Switch(
                  value: enabled,
                  onChanged: (v) => _toggle(s, v),
                  activeThumbColor: AppColors.accent,
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              summary,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11, color: AppColors.textMuted),
              maxLines: 2,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 10),
            Wrap(
              spacing: 6,
              runSpacing: 6,
              children: applies.map((a) {
                return Container(
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                  decoration: BoxDecoration(
                    color: AppColors.accent.withValues(alpha: 0.12),
                    borderRadius: BorderRadius.circular(10),
                  ),
                  child: Text(
                    a == '*' ? context.tr('all agents') : a,
                    style: const TextStyle(fontSize: 11, color: AppColors.accent, fontWeight: FontWeight.w500),
                  ),
                );
              }).toList(),
            ),
            const SizedBox(height: 4),
            Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              TextButton.icon(
                onPressed: () => _openEditor(s),
                icon: const Icon(Icons.edit, size: 16),
                label: Text(context.tr('Edit')),
              ),
              TextButton.icon(
                onPressed: () => _delete(s),
                icon: const Icon(Icons.delete_outline, size: 16, color: Colors.red),
                label: Text(context.tr('Delete'), style: const TextStyle(color: Colors.red)),
              ),
            ]),
          ],
        ),
      ),
    );
  }

  String _summarize(Map<String, dynamic> s) {
    final transport = s['transport'] as String? ?? 'stdio';
    if (transport == 'stdio') {
      final cmd = s['command'] as String? ?? '';
      final args = (s['args'] as List?)?.cast<String>() ?? const <String>[];
      return '\$ $cmd ${args.join(' ')}'.trim();
    }
    return s['url'] as String? ?? '';
  }

  Widget _badge(String transport) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: AppColors.border,
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        transport.toUpperCase(),
        style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: AppColors.textMuted),
      ),
    );
  }
}

// ═════════════════════════════════════════════════════════════════
// Editor dialog
// ═════════════════════════════════════════════════════════════════

class _ServerEditorDialog extends StatefulWidget {
  final ApiClient api;
  final Map<String, dynamic>? initial;
  final List<String> agents;
  const _ServerEditorDialog({required this.api, required this.initial, required this.agents});

  @override
  State<_ServerEditorDialog> createState() => _ServerEditorDialogState();
}

class _ServerEditorDialogState extends State<_ServerEditorDialog> {
  late final TextEditingController _name;
  late final TextEditingController _desc;
  late final TextEditingController _command;
  late final TextEditingController _argsLine;
  late final TextEditingController _url;
  late String _transport;
  late bool _enabled;
  late Set<String> _applies; // "*" OR subset of agents
  List<MapEntry<TextEditingController, TextEditingController>> _envRows = [];

  @override
  void initState() {
    super.initState();
    final s = widget.initial ?? const <String, dynamic>{};
    _name    = TextEditingController(text: s['name']        as String? ?? '');
    _desc    = TextEditingController(text: s['description'] as String? ?? '');
    _command = TextEditingController(text: s['command']     as String? ?? '');
    _argsLine = TextEditingController(
      text: ((s['args'] as List?)?.cast<String>() ?? const <String>[]).join(' '),
    );
    _url       = TextEditingController(text: s['url'] as String? ?? '');
    _transport = (s['transport'] as String?) ?? 'stdio';
    _enabled   = s['enabled'] as bool? ?? true;
    final applies = (s['appliesTo'] as List?)?.cast<String>() ?? ['*'];
    _applies = Set<String>.from(applies);

    final env = (s['env'] as Map?)?.cast<String, dynamic>() ?? {};
    _envRows = env.entries.map((e) {
      return MapEntry(
        TextEditingController(text: e.key),
        TextEditingController(text: e.value.toString()),
      );
    }).toList();
  }

  @override
  void dispose() {
    _name.dispose();
    _desc.dispose();
    _command.dispose();
    _argsLine.dispose();
    _url.dispose();
    for (final e in _envRows) {
      e.key.dispose();
      e.value.dispose();
    }
    super.dispose();
  }

  List<String> _tokenize(String s) =>
      s.split(RegExp(r'\s+')).where((t) => t.isNotEmpty).toList();

  Map<String, String> _collectEnv() {
    final out = <String, String>{};
    for (final row in _envRows) {
      final k = row.key.text.trim();
      if (k.isNotEmpty) out[k] = row.value.text;
    }
    return out;
  }

  Future<void> _save() async {
    if (_name.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(context.tr('Name is required'))),
      );
      return;
    }
    final applies = _applies.contains('*') ? ['*'] : _applies.toList()..sort();
    final body = <String, dynamic>{
      'name':        _name.text.trim(),
      'description': _desc.text.trim(),
      'transport':   _transport,
      'command':     _command.text.trim(),
      'args':        _tokenize(_argsLine.text),
      'env':         _collectEnv(),
      'url':         _url.text.trim(),
      'headers':     <String, String>{},
      'appliesTo':   applies,
      'enabled':     _enabled,
    };
    try {
      if (widget.initial == null) {
        await ApiClient.describeErrors(() => widget.api.mcpCreateServer(body));
      } else {
        await ApiClient.describeErrors(
          () => widget.api.mcpUpdateServer(widget.initial!['id'] as String, body),
        );
      }
      ProvidersBus.instance.notify();
      if (!mounted) return;
      Navigator.of(context, rootNavigator: true).pop(true);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final isStdio = _transport == 'stdio';
    return AlertDialog(
      title: Text(widget.initial == null
          ? context.tr('New MCP server')
          : context.tr('Edit MCP server')),
      contentPadding: const EdgeInsets.fromLTRB(20, 12, 20, 0),
      content: SizedBox(
        width: 520,
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              _field(context.tr('Name'), _name, hint: 'filesystem'),
              _field(context.tr('Description'), _desc),
              const SizedBox(height: 6),
              Text(context.tr('Transport'), style: _labelStyle),
              const SizedBox(height: 4),
              SegmentedButton<String>(
                segments: const [
                  ButtonSegment(value: 'stdio', label: Text('stdio')),
                  ButtonSegment(value: 'sse',   label: Text('sse')),
                  ButtonSegment(value: 'http',  label: Text('http')),
                ],
                selected: {_transport},
                onSelectionChanged: (v) => setState(() => _transport = v.first),
              ),
              const SizedBox(height: 12),
              if (isStdio) ...[
                _field(context.tr('Command'), _command, hint: 'npx'),
                _field(
                  context.tr('Args (space-separated)'),
                  _argsLine,
                  hint: '-y @modelcontextprotocol/server-filesystem /tmp',
                ),
                const SizedBox(height: 6),
                Row(children: [
                  Text(context.tr('Environment'), style: _labelStyle),
                  const Spacer(),
                  IconButton(
                    iconSize: 20,
                    onPressed: () => setState(() => _envRows.add(MapEntry(
                      TextEditingController(), TextEditingController(),
                    ))),
                    icon: const Icon(Icons.add_circle_outline),
                  ),
                ]),
                ..._envRows.asMap().entries.map((entry) {
                  final i = entry.key;
                  final row = entry.value;
                  return Padding(
                    padding: const EdgeInsets.only(bottom: 6),
                    child: Row(children: [
                      Expanded(child: TextField(
                        controller: row.key,
                        decoration: InputDecoration(hintText: context.tr('KEY')),
                      )),
                      const SizedBox(width: 8),
                      Expanded(child: TextField(
                        controller: row.value,
                        decoration: InputDecoration(hintText: context.tr('value')),
                      )),
                      IconButton(
                        iconSize: 20,
                        onPressed: () => setState(() {
                          final removed = _envRows.removeAt(i);
                          removed.key.dispose();
                          removed.value.dispose();
                        }),
                        icon: const Icon(Icons.remove_circle_outline),
                      ),
                    ]),
                  );
                }),
              ] else ...[
                _field(context.tr('URL'), _url, hint: 'https://example.com/sse'),
              ],
              const SizedBox(height: 12),
              Text(context.tr('Applies to'), style: _labelStyle),
              const SizedBox(height: 4),
              Wrap(spacing: 6, runSpacing: 6, children: [
                FilterChip(
                  label: Text(context.tr('all agents')),
                  selected: _applies.contains('*'),
                  onSelected: (v) => setState(() {
                    if (v) {
                      _applies = {'*'};
                    } else {
                      _applies.remove('*');
                    }
                  }),
                ),
                for (final a in widget.agents)
                  FilterChip(
                    label: Text(a),
                    selected: _applies.contains(a),
                    onSelected: _applies.contains('*')
                        ? null
                        : (v) => setState(() {
                              if (v) {
                                _applies.add(a);
                              } else {
                                _applies.remove(a);
                              }
                            }),
                  ),
              ]),
              const SizedBox(height: 12),
              SwitchListTile(
                value: _enabled,
                onChanged: (v) => setState(() => _enabled = v),
                title: Text(context.tr('Enabled')),
                contentPadding: EdgeInsets.zero,
                activeThumbColor: AppColors.accent,
              ),
            ],
          ),
        ),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context, rootNavigator: true).pop(false),
          child: Text(context.tr('Cancel')),
        ),
        FilledButton(
          onPressed: _save,
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
          child: Text(context.tr('Save')),
        ),
      ],
    );
  }

  Widget _field(String label, TextEditingController c, {String? hint}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: TextField(
        controller: c,
        decoration: InputDecoration(labelText: label, hintText: hint),
      ),
    );
  }

  static const _labelStyle = TextStyle(
    fontSize: 12, fontWeight: FontWeight.w500, color: AppColors.textMuted,
  );
}
