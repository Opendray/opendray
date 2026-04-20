import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../core/api/api_client.dart';
import '../../../core/models/provider.dart';
import '../../../core/services/l10n.dart';
import '../../../shared/providers_bus.dart';
import '../../../shared/theme/app_theme.dart';
import '../../claude_accounts/claude_accounts_page.dart';

class PluginsSection extends StatefulWidget {
  const PluginsSection({super.key});
  @override
  State<PluginsSection> createState() => _PluginsSectionState();
}

class _PluginsSectionState extends State<PluginsSection> {
  List<ProviderInfo> _plugins = [];
  bool _loading = true;
  String? _expandedName;
  final Map<String, Map<String, dynamic>> _editConfigs = {};
  String? _error;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final plugins = await _api.listProviders();
      if (mounted) setState(() { _plugins = plugins; _loading = false; _error = null; });
    } catch (e) {
      if (mounted) setState(() { _loading = false; _error = e.toString(); });
    }
  }

  void _toggleExpand(String name) {
    setState(() {
      if (_expandedName == name) {
        _expandedName = null;
        return;
      }
      _expandedName = name;
      final pi = _plugins.firstWhere((p) => p.provider.name == name);
      _editConfigs[name] = Map<String, dynamic>.from(pi.config);
    });
  }

  Future<void> _saveConfig(String name) async {
    await _api.updateProviderConfig(name, _editConfigs[name] ?? {});
    setState(() => _expandedName = null);
    _load();
    ProvidersBus.instance.notify();
  }

  // Optimistic toggle: flip the local entry in place so the card stays put.
  // Without this, re-fetching the list after every switch tap caused rows to
  // visually jump — users lost track of which plugin they had just touched.
  // On failure we roll back and surface a SnackBar.
  Future<void> _toggle(String name, bool enabled) async {
    final idx = _plugins.indexWhere((p) => p.provider.name == name);
    if (idx < 0) return;
    final original = _plugins[idx];
    setState(() {
      _plugins = List.of(_plugins)
        ..[idx] = ProviderInfo(
          provider: original.provider,
          config: original.config,
          installed: original.installed,
          enabled: enabled,
        );
    });
    ProvidersBus.instance.notify();
    try {
      await _api.toggleProvider(name, enabled);
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _plugins = List.of(_plugins)..[idx] = original;
      });
      ProvidersBus.instance.notify();
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('${context.tr('Failed to toggle plugin')}: $e')),
      );
    }
  }

  Future<void> _confirmDelete(String name, String displayName) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(context.tr('Delete plugin?')),
        content: Text(
            '${context.tr('Remove')} "$displayName"? ${context.tr('This cannot be undone.')}'),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(context.tr('Cancel')),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            style: FilledButton.styleFrom(backgroundColor: AppColors.error),
            child: Text(context.tr('Delete')),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    try {
      await _api.deleteProvider(name);
      if (!mounted) return;
      setState(() => _expandedName = null);
      _load();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text('${context.tr('Failed to delete')}: $e')),
      );
    }
  }

  // Raw group labels — run through context.tr() at render time so the
  // Chinese catalog can swap them out.
  static const _groupLabels = {
    'connection': 'Connection',
    'auth':       'Authentication',
    'runtime':    'Runtime',
    'advanced':   'Advanced',
  };
  static const _groupOrder = ['connection', 'auth', 'runtime', 'advanced'];

  Map<String, List<ConfigField>> _groupedSchema(List<ConfigField> fields) {
    final groups = <String, List<ConfigField>>{};
    for (final f in fields) {
      final g = f.group ?? 'runtime';
      groups.putIfAbsent(g, () => []).add(f);
    }
    return groups;
  }

  bool _shouldShow(ConfigField field, Map<String, dynamic> config) {
    if (field.dependsOn == null) return true;
    return config[field.dependsOn] == field.dependsVal;
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(4, 4, 4, 8),
          child: Row(
            children: [
              const Icon(Icons.extension, color: AppColors.accent, size: 20),
              const SizedBox(width: 10),
              Text(context.tr('Plugins'), style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
              const Spacer(),
              if (!_loading)
                Text('${_plugins.length} ${context.tr('registered')}',
                    style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
              const SizedBox(width: 6),
              IconButton(
                icon: const Icon(Icons.refresh, size: 18),
                onPressed: _load,
                tooltip: context.tr('Reload'),
                visualDensity: VisualDensity.compact,
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              ),
            ],
          ),
        ),
        if (_loading)
          const Padding(
            padding: EdgeInsets.all(24),
            child: Center(child: CircularProgressIndicator(color: AppColors.accent)),
          )
        else if (_error != null)
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(color: AppColors.errorSoft, borderRadius: BorderRadius.circular(8)),
            child: Column(crossAxisAlignment: CrossAxisAlignment.start, children: [
              Text(context.tr('Failed to load plugins'),
                  style: const TextStyle(fontWeight: FontWeight.w500, color: AppColors.error)),
              const SizedBox(height: 4),
              Text(_error!, style: const TextStyle(color: AppColors.error, fontSize: 12)),
              const SizedBox(height: 8),
              FilledButton(onPressed: _load,
                  style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
                  child: Text(context.tr('Retry'))),
            ]),
          )
        else if (_plugins.isEmpty)
          Padding(
            padding: const EdgeInsets.all(16),
            child: Text(context.tr('No plugins registered'),
                style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
          )
        else
          for (int i = 0; i < _plugins.length; i++) ...[
            _buildPluginCard(_plugins[i]),
            if (i < _plugins.length - 1) const SizedBox(height: 8),
          ],
      ],
    );
  }

  Widget _buildPluginCard(ProviderInfo pi) {
    final isExpanded = _expandedName == pi.provider.name;
    final config = _editConfigs[pi.provider.name] ?? pi.config;

    return Card(
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          // ── Header row — split into three independent hit targets ────
          // Left: expand/collapse. Right: toggle (isolated so taps on the
          // pill never propagate to expand). Far right: chevron.
          Row(
            children: [
              Expanded(
                child: InkWell(
                  onTap: () => _toggleExpand(pi.provider.name),
                  child: Padding(
                    padding: const EdgeInsets.fromLTRB(14, 14, 8, 14),
                    child: Row(
                      children: [
                        Text(pi.provider.icon,
                            style: const TextStyle(fontSize: 24)),
                        const SizedBox(width: 12),
                        Expanded(
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Row(children: [
                                Flexible(
                                  child: Text(
                                    context.tr(pi.provider.displayName),
                                    style: const TextStyle(
                                        fontWeight: FontWeight.w500,
                                        fontSize: 14),
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                                const SizedBox(width: 6),
                                _TypeBadge(pi.provider.type),
                                const SizedBox(width: 4),
                                Text('v${pi.provider.version}',
                                    style: const TextStyle(
                                        color: AppColors.textMuted,
                                        fontSize: 10)),
                              ]),
                              const SizedBox(height: 2),
                              Text(context.tr(pi.provider.description),
                                  style: const TextStyle(
                                      color: AppColors.textMuted,
                                      fontSize: 11),
                                  maxLines: 2,
                                  overflow: TextOverflow.ellipsis),
                              const SizedBox(height: 4),
                              Wrap(spacing: 4, children: [
                                if (pi.provider.capabilities.supportsResume)
                                  _CapBadge(context.tr('Resume')),
                                if (pi.provider.capabilities.supportsImages)
                                  _CapBadge(context.tr('Images')),
                                if (pi.provider.capabilities.supportsMcp)
                                  _CapBadge(context.tr('MCP')),
                                if (pi.provider.capabilities.dynamicModels)
                                  _CapBadge(context.tr('Dynamic Models'),
                                      color: AppColors.warning),
                              ]),
                            ],
                          ),
                        ),
                      ],
                    ),
                  ),
                ),
              ),
              // Isolated toggle hit area — opaque so taps never fall through
              // to the expand InkWell underneath.
              GestureDetector(
                behavior: HitTestBehavior.opaque,
                onTap: () => _toggle(pi.provider.name, !pi.enabled),
                child: Padding(
                  padding: const EdgeInsets.fromLTRB(4, 10, 10, 10),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      _StatusBadge(
                          installed: pi.installed, enabled: pi.enabled),
                      const SizedBox(height: 6),
                      Container(
                        width: 40,
                        height: 22,
                        decoration: BoxDecoration(
                          color: pi.enabled
                              ? AppColors.accent
                              : AppColors.surfaceAlt,
                          borderRadius: BorderRadius.circular(11),
                        ),
                        child: AnimatedAlign(
                          alignment: pi.enabled
                              ? Alignment.centerRight
                              : Alignment.centerLeft,
                          duration: const Duration(milliseconds: 200),
                          child: Container(
                            width: 18,
                            height: 18,
                            margin: const EdgeInsets.all(2),
                            decoration: const BoxDecoration(
                                color: Colors.white, shape: BoxShape.circle),
                          ),
                        ),
                      ),
                    ],
                  ),
                ),
              ),
              InkWell(
                onTap: () => _toggleExpand(pi.provider.name),
                customBorder: const CircleBorder(),
                child: Padding(
                  padding: const EdgeInsets.all(8),
                  child: AnimatedRotation(
                    turns: isExpanded ? 0.5 : 0,
                    duration: const Duration(milliseconds: 200),
                    child: const Icon(Icons.expand_more,
                        color: AppColors.textMuted, size: 20),
                  ),
                ),
              ),
              const SizedBox(width: 4),
            ],
          ),

          // ── Animated expand body ────────────────────────────────────
          AnimatedSize(
            duration: const Duration(milliseconds: 200),
            curve: Curves.easeInOut,
            alignment: Alignment.topCenter,
            child: isExpanded
                ? _buildExpandedBody(pi, config)
                : const SizedBox(width: double.infinity),
          ),
        ],
      ),
    );
  }

  Widget _buildExpandedBody(ProviderInfo pi, Map<String, dynamic> config) {
    final grouped = _groupedSchema(pi.provider.configSchema);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Divider(height: 1),
        Padding(
          padding: const EdgeInsets.fromLTRB(14, 4, 14, 14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Claude multi-account shortcut — accounts are a kernel
              // resource, not per-plugin config; surfaced here because users
              // look inside the Claude card when wondering "how do I switch
              // accounts?".
              //
              // We deliberately DON'T surface a similar "launch login"
              // shortcut for Codex / Gemini: each CLI already prompts for
              // OAuth on first run when it has no cached token, so an
              // explicit launcher in Settings just adds noise. Users get
              // auth automatically when they start a real session.
              if (pi.provider.name == 'claude') ...[
                const SizedBox(height: 10),
                _buildClaudeAccountsShortcut(),
              ],
              for (final groupKey in _groupOrder)
                if (grouped.containsKey(groupKey))
                  _buildGroup(
                    context.tr(_groupLabels[groupKey] ?? groupKey),
                    grouped[groupKey]!,
                    pi.provider.name,
                    config,
                  ),
              if (pi.provider.capabilities.models.isNotEmpty) ...[
                _buildGroupHeader(context.tr('MODELS')),
                Wrap(
                  spacing: 6,
                  runSpacing: 6,
                  children: pi.provider.capabilities.models
                      .map((m) => Tooltip(
                            message: m.description ?? '',
                            child: Chip(
                              label: Text(m.name,
                                  style: const TextStyle(fontSize: 11)),
                              backgroundColor: AppColors.surfaceAlt,
                              side: const BorderSide(color: AppColors.border),
                              padding:
                                  const EdgeInsets.symmetric(horizontal: 2),
                            ),
                          ))
                      .toList(),
                ),
              ],
            ],
          ),
        ),
        // Pinned action bar — contrasting bg + top divider so it reads as
        // a primary action zone at the bottom of the card, even on long
        // schemas where the user has scrolled past many fields.
        Container(
          decoration: const BoxDecoration(
            color: AppColors.surfaceAlt,
            border: Border(top: BorderSide(color: AppColors.border)),
          ),
          padding: const EdgeInsets.fromLTRB(14, 10, 14, 10),
          child: Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              TextButton(
                onPressed: () => setState(() => _expandedName = null),
                child: Text(context.tr('Cancel'),
                    style: const TextStyle(fontSize: 13)),
              ),
              const SizedBox(width: 8),
              FilledButton.icon(
                onPressed: () => _saveConfig(pi.provider.name),
                icon: const Icon(Icons.save, size: 16),
                label: Text(context.tr('Save'),
                    style: const TextStyle(fontSize: 13)),
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.accent,
                  padding:
                      const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
                ),
              ),
            ],
          ),
        ),
        // Danger zone — Delete is isolated behind a confirm dialog and
        // visually separated from Save/Cancel to prevent misclicks.
        Container(
          decoration: const BoxDecoration(
            color: AppColors.errorSoft,
            border: Border(top: BorderSide(color: AppColors.border)),
          ),
          padding: const EdgeInsets.fromLTRB(14, 10, 14, 12),
          child: Row(children: [
            const Icon(Icons.warning_amber_outlined,
                size: 16, color: AppColors.error),
            const SizedBox(width: 8),
            Expanded(
              child: Text(
                context.tr('Danger zone'),
                style: const TextStyle(
                    fontSize: 11,
                    fontWeight: FontWeight.w600,
                    color: AppColors.error,
                    letterSpacing: 0.6),
              ),
            ),
            OutlinedButton.icon(
              onPressed: () => _confirmDelete(
                  pi.provider.name, context.tr(pi.provider.displayName)),
              icon: const Icon(Icons.delete_outline, size: 14),
              label: Text(context.tr('Delete'),
                  style: const TextStyle(fontSize: 12)),
              style: OutlinedButton.styleFrom(
                foregroundColor: AppColors.error,
                side: const BorderSide(color: AppColors.error),
                padding:
                    const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
                visualDensity: VisualDensity.compact,
              ),
            ),
          ]),
        ),
      ],
    );
  }

  Widget _buildGroupHeader(String label) {
    return Padding(
      padding: const EdgeInsets.only(top: 12, bottom: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label.toUpperCase(),
            style: const TextStyle(
              fontSize: 12,
              fontWeight: FontWeight.w700,
              color: AppColors.textMuted,
              letterSpacing: 1.2,
            ),
          ),
          const SizedBox(height: 4),
          const Divider(height: 1),
        ],
      ),
    );
  }

  Widget _buildGroup(String label, List<ConfigField> fields, String pluginName,
      Map<String, dynamic> config) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        _buildGroupHeader(label),
        const SizedBox(height: 6),
        for (final field in fields)
          if (_shouldShow(field, config))
            Padding(
              padding: const EdgeInsets.only(bottom: 10),
              child: _buildField(field, pluginName, config),
            ),
      ],
    );
  }

  /// M5 A3.2 — render Claude account management inline inside the
  /// Claude plugin card, replacing the shortcut-to-separate-page.
  /// Users see the full account list + add/delete/import right next
  /// to the plugin's configSchema fields instead of jumping to a
  /// separate route.
  Widget _buildClaudeAccountsShortcut() {
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(children: [
            const Icon(Icons.people_outline, size: 16, color: AppColors.accent),
            const SizedBox(width: 6),
            Text(
              context.tr('ACCOUNTS'),
              style: const TextStyle(
                fontSize: 11,
                fontWeight: FontWeight.w600,
                color: AppColors.textMuted,
                letterSpacing: 0.5,
              ),
            ),
          ]),
          const SizedBox(height: 8),
          // The inline ClaudeAccountsPage does its own data fetch +
          // refresh; no state duplication here.
          const ClaudeAccountsPage(inline: true),
        ],
      ),
    );
  }

  Widget _buildField(ConfigField field, String pluginName, Map<String, dynamic> config) {
    final value = config[field.key] ?? field.defaultValue;
    // Helpers so the three body styles below don't all duplicate the tr() calls.
    final label       = context.tr(field.label);
    final description = field.description == null ? null : context.tr(field.description!);
    final hint        = field.placeholder == null ? null : context.tr(field.placeholder!);

    switch (field.type) {
      case 'boolean':
        return Row(
          mainAxisAlignment: MainAxisAlignment.spaceBetween,
          children: [
            Expanded(child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(label, style: const TextStyle(fontSize: 12)),
                if (description != null)
                  Text(description, style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
              ],
            )),
            Switch(
              value: value == true,
              onChanged: (v) => setState(() => _editConfigs[pluginName]![field.key] = v),
              activeTrackColor: AppColors.accent,
            ),
          ],
        );
      case 'select':
        final optionStrings = field.options?.map((o) => o.toString()).toList() ?? [];
        final currentVal = value?.toString() ?? '';
        final validVal = optionStrings.contains(currentVal) ? currentVal : null;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Text(label, style: const TextStyle(fontSize: 12)),
              if (field.envVar != null) ...[
                const SizedBox(width: 6),
                Text(field.envVar!, style: const TextStyle(fontSize: 9, color: AppColors.accent, fontFamily: 'monospace')),
              ],
            ]),
            if (description != null)
              Text(description, style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
            const SizedBox(height: 4),
            DropdownButtonFormField<String>(
              initialValue: validVal,
              dropdownColor: AppColors.surfaceAlt,
              decoration: const InputDecoration(),
              items: optionStrings.map((o) => DropdownMenuItem(
                value: o,
                child: Text(o.isEmpty ? context.tr('(default)') : o,
                    style: TextStyle(fontSize: 13,
                        color: o.isEmpty ? AppColors.textMuted : AppColors.text)),
              )).toList(),
              onChanged: (v) => setState(() => _editConfigs[pluginName]![field.key] = v),
              isExpanded: true,
            ),
          ],
        );
      case 'secret':
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Text(label, style: const TextStyle(fontSize: 12)),
              if (field.envVar != null) ...[
                const SizedBox(width: 6),
                Text(field.envVar!, style: const TextStyle(fontSize: 9, color: AppColors.accent, fontFamily: 'monospace')),
              ],
            ]),
            if (description != null)
              Text(description, style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
            const SizedBox(height: 4),
            TextFormField(
              initialValue: value?.toString() ?? '',
              obscureText: true,
              decoration: InputDecoration(hintText: hint ?? ''),
              onChanged: (v) => _editConfigs[pluginName]![field.key] = v,
              style: const TextStyle(fontSize: 13),
            ),
          ],
        );
      case 'number':
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label, style: const TextStyle(fontSize: 12)),
            if (description != null)
              Text(description, style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
            const SizedBox(height: 4),
            TextFormField(
              initialValue: value?.toString() ?? '',
              keyboardType: TextInputType.number,
              decoration: InputDecoration(hintText: hint ?? ''),
              onChanged: (v) => _editConfigs[pluginName]![field.key] = int.tryParse(v) ?? v,
              style: const TextStyle(fontSize: 13),
            ),
          ],
        );
      default:
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(label, style: const TextStyle(fontSize: 12)),
            if (description != null)
              Text(description, style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
            const SizedBox(height: 4),
            TextFormField(
              initialValue: value?.toString() ?? '',
              decoration: InputDecoration(hintText: hint ?? ''),
              onChanged: (v) => _editConfigs[pluginName]![field.key] = v,
              style: const TextStyle(fontSize: 13),
            ),
          ],
        );
    }
  }
}

class _TypeBadge extends StatelessWidget {
  final String type;
  const _TypeBadge(this.type);
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
      decoration: BoxDecoration(color: AppColors.surfaceAlt, borderRadius: BorderRadius.circular(3)),
      child: Text(type, style: const TextStyle(color: AppColors.textMuted, fontSize: 9)),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  final bool installed;
  final bool enabled;
  const _StatusBadge({required this.installed, required this.enabled});
  @override
  Widget build(BuildContext context) {
    if (!installed) return _badge(context.tr('Not found'), AppColors.warning, AppColors.warningSoft);
    if (!enabled)   return _badge(context.tr('Disabled'),  AppColors.textMuted, AppColors.surfaceAlt);
    return _badge(context.tr('Active'), AppColors.success, AppColors.successSoft);
  }
  Widget _badge(String text, Color color, Color bg) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
    decoration: BoxDecoration(color: bg, borderRadius: BorderRadius.circular(4)),
    child: Text(text, style: TextStyle(color: color, fontSize: 9, fontWeight: FontWeight.w500)),
  );
}

class _CapBadge extends StatelessWidget {
  final String text;
  final Color color;
  const _CapBadge(this.text, {this.color = AppColors.accent});
  @override
  Widget build(BuildContext context) => Container(
    padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
    decoration: BoxDecoration(color: color.withValues(alpha: 0.15), borderRadius: BorderRadius.circular(3)),
    child: Text(text, style: TextStyle(color: color, fontSize: 9)),
  );
}
