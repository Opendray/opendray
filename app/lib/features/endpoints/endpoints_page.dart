import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// LLM Providers — address book of OpenAI-compatible model endpoints.
///
/// Each row becomes a spawn-time route for OpenCode (and any future
/// OpenAI-native agent plugin) sessions. The page lets the user add /
/// edit / toggle / delete providers, and offers a "Detect models"
/// action that hits the upstream's /v1/models so the New Session
/// dialog can show a dropdown.
///
/// API keys themselves are never kept in the DB — users configure an
/// env var name on the OpenDray host (e.g. GROQ_API_KEY). The gateway
/// reports `apiKeySet` so this page can warn when the var is missing.
class EndpointsPage extends StatefulWidget {
  const EndpointsPage({super.key});
  @override
  State<EndpointsPage> createState() => _EndpointsPageState();
}

class _EndpointsPageState extends State<EndpointsPage> {
  List<Map<String, dynamic>> _providers = [];
  bool _loading = true;
  String? _error;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _refresh();
  }

  Future<void> _refresh() async {
    try {
      final rows = await _api.llmProviders();
      if (!mounted) return;
      setState(() {
        _providers = rows;
        _error = null;
        _loading = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _error = e.toString();
        _loading = false;
      });
    }
  }

  Future<void> _openEditor([Map<String, dynamic>? row]) async {
    final changed = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (_) => _ProviderEditorDialog(api: _api, initial: row),
    );
    if (changed == true) {
      _refresh();
      ProvidersBus.instance.notify();
    }
  }

  Future<void> _detectModels(Map<String, dynamic> row) async {
    final messenger = ScaffoldMessenger.of(context);
    messenger.showSnackBar(SnackBar(
      content: Text(context.trOnce('Probing @name …').replaceAll('@name', row['name'] as String)),
      duration: const Duration(seconds: 2),
    ));
    final models = await _api.llmProviderModels(row['id'] as String);
    if (!mounted) return;
    if (models == null) {
      messenger.showSnackBar(SnackBar(
        content: Text(context.trOnce('Upstream unreachable — you can still enter a model name by hand when creating a session.')),
        backgroundColor: AppColors.warning,
      ));
      return;
    }
    messenger.showSnackBar(SnackBar(
      content: Text('${models.length} ${context.trOnce('models available')}: ${models.take(5).join(', ')}${models.length > 5 ? ' …' : ''}'),
      backgroundColor: AppColors.success,
      duration: const Duration(seconds: 5),
    ));
  }

  Future<void> _toggle(Map<String, dynamic> row, bool enabled) async {
    try {
      await _api.llmProviderToggle(row['id'] as String, enabled);
      setState(() => row['enabled'] = enabled);
      ProvidersBus.instance.notify();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  Future<void> _delete(Map<String, dynamic> row) async {
    final ok = await showDialog<bool>(
      context: context,
      useRootNavigator: true,
      builder: (ctx) => AlertDialog(
        title: Text(ctx.tr('Delete LLM provider?')),
        content: Text(
          ctx.tr('This removes the provider from OpenDray. Sessions currently bound to it will be unbound.'),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx, rootNavigator: true).pop(false),
            child: Text(ctx.tr('Cancel')),
          ),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: Colors.red),
            onPressed: () => Navigator.of(ctx, rootNavigator: true).pop(true),
            child: Text(ctx.tr('Delete')),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await _api.llmProviderDelete(row['id'] as String);
      _refresh();
      ProvidersBus.instance.notify();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
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
        title: context.tr('Failed to load LLM providers'),
        body: _error!,
      );
    }
    return Scaffold(
      floatingActionButton: FloatingActionButton.extended(
        onPressed: () => _openEditor(),
        backgroundColor: AppColors.accent,
        icon: const Icon(Icons.add),
        label: Text(context.tr('Add provider')),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 96),
          children: [
            _intro(),
            const SizedBox(height: 12),
            if (_providers.isEmpty)
              _emptyState(
                icon: Icons.satellite_alt_outlined,
                title: context.tr('No LLM providers yet'),
                body: context.tr('Add one endpoint per model host: a Mac running Ollama, LM Studio, or a free cloud provider like Groq / Gemini. Sessions route through these at spawn.'),
              )
            else
              ..._providers.map((p) => Padding(
                    padding: const EdgeInsets.only(bottom: 10),
                    child: _card(p),
                  )),
          ],
        ),
      ),
    );
  }

  Widget _intro() {
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: AppColors.accentSoft,
        borderRadius: BorderRadius.circular(10),
        border: Border.all(color: AppColors.accent.withValues(alpha: 0.35)),
      ),
      child: Row(children: [
        const Icon(Icons.info_outline, size: 18, color: AppColors.accent),
        const SizedBox(width: 8),
        Expanded(
          child: Text(
            context.tr('Each provider is an OpenAI-compatible HTTPS endpoint. Models are picked per session from the upstream /v1/models listing, with free-text fallback.'),
            style: const TextStyle(fontSize: 11, color: AppColors.text),
          ),
        ),
      ]),
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

  Widget _card(Map<String, dynamic> p) {
    final enabled = p['enabled'] as bool? ?? true;
    final apiKeyEnv = p['apiKeyEnv'] as String? ?? '';
    final apiKeySet = p['apiKeySet'] as bool? ?? false;
    final display = (p['displayName'] as String?)?.isNotEmpty == true
        ? p['displayName'] as String
        : p['name'] as String? ?? '';
    return Card(
      color: AppColors.surface,
      elevation: 0,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(12),
        side: BorderSide(
          color: enabled ? AppColors.accent.withValues(alpha: 0.25) : AppColors.border,
        ),
      ),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 12, 8, 12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Flexible(
                        child: Text(
                          display,
                          style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      const SizedBox(width: 8),
                      _chip(p['providerType'] as String? ?? 'openai-compat'),
                    ]),
                    const SizedBox(height: 2),
                    Text(p['baseUrl'] as String? ?? '',
                        style: const TextStyle(
                            fontFamily: 'monospace', fontSize: 11, color: AppColors.textMuted)),
                    if ((p['description'] as String?)?.isNotEmpty == true) ...[
                      const SizedBox(height: 4),
                      Text(p['description'] as String,
                          style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
                    ],
                    if (apiKeyEnv.isNotEmpty) ...[
                      const SizedBox(height: 4),
                      Row(children: [
                        Icon(
                          apiKeySet ? Icons.vpn_key : Icons.vpn_key_off,
                          size: 12,
                          color: apiKeySet ? AppColors.success : AppColors.warning,
                        ),
                        const SizedBox(width: 4),
                        Text('\$$apiKeyEnv',
                            style: TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 11,
                                color: apiKeySet ? AppColors.textMuted : AppColors.warning)),
                        if (!apiKeySet) ...[
                          const SizedBox(width: 4),
                          Text('(${context.tr("not set on host")})',
                              style: const TextStyle(fontSize: 11, color: AppColors.warning)),
                        ],
                      ]),
                    ],
                  ],
                ),
              ),
              Switch(
                value: enabled,
                onChanged: (v) => _toggle(p, v),
                activeThumbColor: AppColors.accent,
              ),
            ]),
            const SizedBox(height: 4),
            Row(mainAxisAlignment: MainAxisAlignment.end, children: [
              TextButton.icon(
                onPressed: () => _detectModels(p),
                icon: const Icon(Icons.radar, size: 16),
                label: Text(context.tr('Detect models')),
              ),
              TextButton.icon(
                onPressed: () => _openEditor(p),
                icon: const Icon(Icons.edit, size: 16),
                label: Text(context.tr('Edit')),
              ),
              TextButton.icon(
                onPressed: () => _delete(p),
                icon: const Icon(Icons.delete_outline, size: 16, color: Colors.red),
                label: Text(context.tr('Delete'), style: const TextStyle(color: Colors.red)),
              ),
            ]),
          ],
        ),
      ),
    );
  }

  Widget _chip(String providerType) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(color: AppColors.accentSoft, borderRadius: BorderRadius.circular(4)),
      child: Text(providerType,
          style: const TextStyle(fontSize: 10, fontWeight: FontWeight.w600, color: AppColors.accent)),
    );
  }
}

// ═════════════════════════════════════════════════════════════════
// Editor dialog
// ═════════════════════════════════════════════════════════════════

class _ProviderEditorDialog extends StatefulWidget {
  final ApiClient api;
  final Map<String, dynamic>? initial;
  const _ProviderEditorDialog({required this.api, required this.initial});
  @override
  State<_ProviderEditorDialog> createState() => _ProviderEditorDialogState();
}

class _ProviderEditorDialogState extends State<_ProviderEditorDialog> {
  static const _typeOptions = <String>[
    'ollama', 'lmstudio', 'openai-compat', 'groq', 'gemini', 'custom',
  ];

  late final TextEditingController _name;
  late final TextEditingController _displayName;
  late final TextEditingController _baseUrl;
  late final TextEditingController _apiKeyEnv;
  late final TextEditingController _description;
  late bool _enabled;
  late String _providerType;

  @override
  void initState() {
    super.initState();
    final s = widget.initial ?? const <String, dynamic>{};
    _name        = TextEditingController(text: s['name']        as String? ?? '');
    _displayName = TextEditingController(text: s['displayName'] as String? ?? '');
    _baseUrl     = TextEditingController(text: s['baseUrl']     as String? ?? '');
    _apiKeyEnv   = TextEditingController(text: s['apiKeyEnv']   as String? ?? '');
    _description = TextEditingController(text: s['description'] as String? ?? '');
    _enabled     = s['enabled'] as bool? ?? true;
    final t = s['providerType'] as String?;
    _providerType = (t != null && _typeOptions.contains(t)) ? t : 'ollama';
  }

  @override
  void dispose() {
    _name.dispose();
    _displayName.dispose();
    _baseUrl.dispose();
    _apiKeyEnv.dispose();
    _description.dispose();
    super.dispose();
  }

  String _defaultBaseUrl() {
    switch (_providerType) {
      case 'ollama':   return 'http://<host>:11434/v1';
      case 'lmstudio': return 'http://<host>:1234/v1';
      case 'groq':     return 'https://api.groq.com/openai/v1';
      case 'gemini':   return 'https://generativelanguage.googleapis.com/v1beta/openai';
      default:         return 'https://...';
    }
  }

  Future<void> _save() async {
    if (widget.initial == null) {
      final name = _name.text.trim();
      if (!RegExp(r'^[a-z0-9_-]{1,48}$').hasMatch(name)) {
        ScaffoldMessenger.of(context).showSnackBar(SnackBar(
            content: Text(context.tr('Name must be 1-48 chars of [a-z0-9_-]'))));
        return;
      }
    }
    if (_baseUrl.text.trim().isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
          content: Text(context.tr('Base URL is required'))));
      return;
    }
    try {
      if (widget.initial == null) {
        await ApiClient.describeErrors(() => widget.api.llmProviderCreate(
              name: _name.text.trim(),
              displayName: _displayName.text.trim(),
              providerType: _providerType,
              baseUrl: _baseUrl.text.trim(),
              apiKeyEnv: _apiKeyEnv.text.trim(),
              description: _description.text.trim(),
              enabled: _enabled,
            ));
      } else {
        await ApiClient.describeErrors(() => widget.api.llmProviderUpdate(
              widget.initial!['id'] as String,
              {
                'displayName': _displayName.text.trim(),
                'providerType': _providerType,
                'baseUrl': _baseUrl.text.trim(),
                'apiKeyEnv': _apiKeyEnv.text.trim(),
                'description': _description.text.trim(),
                'enabled': _enabled,
              },
            ));
      }
      if (!mounted) return;
      Navigator.of(context, rootNavigator: true).pop(true);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text('$e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final isNew = widget.initial == null;
    return AlertDialog(
      title: Text(isNew ? context.tr('New LLM provider') : context.tr('Edit LLM provider')),
      contentPadding: const EdgeInsets.fromLTRB(20, 12, 20, 0),
      content: SizedBox(
        width: 520,
        child: SingleChildScrollView(
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              if (isNew) _field(context.tr('Name'), _name,
                  hint: 'mac-ollama',
                  help: context.tr('Lowercase alphanumeric, dash or underscore.')),
              _field(context.tr('Display Name'), _displayName, hint: 'Mac Ollama'),
              Padding(
                padding: const EdgeInsets.only(bottom: 10),
                child: DropdownButtonFormField<String>(
                  initialValue: _providerType,
                  items: _typeOptions
                      .map((t) => DropdownMenuItem(value: t, child: Text(t)))
                      .toList(),
                  decoration: InputDecoration(labelText: context.tr('Provider Type')),
                  onChanged: (v) => setState(() {
                    if (v != null) _providerType = v;
                  }),
                ),
              ),
              _field(context.tr('Base URL'), _baseUrl,
                  hint: _defaultBaseUrl(),
                  help: context.tr('OpenAI-compatible endpoint. Mac Ollama: http://<mac-ip>:11434/v1. LM Studio: http://<mac-ip>:1234/v1. Groq: https://api.groq.com/openai/v1.')),
              _field(context.tr('API Key env var'), _apiKeyEnv,
                  hint: 'GROQ_API_KEY',
                  help: context.tr('Optional. Name of an env var on the OpenDray server whose value the gateway forwards as Bearer token. Leave empty for local Ollama / LM Studio.')),
              _field(context.tr('Description'), _description),
              const SizedBox(height: 8),
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

  Widget _field(String label, TextEditingController c,
      {String? hint, String? help, bool obscure = false}) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 10),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
            controller: c,
            obscureText: obscure,
            decoration: InputDecoration(labelText: label, hintText: hint),
          ),
          if (help != null) ...[
            const SizedBox(height: 4),
            Text(help, style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
          ],
        ],
      ),
    );
  }
}
