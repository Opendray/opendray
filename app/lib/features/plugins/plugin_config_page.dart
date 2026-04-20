import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Form editor for a single plugin's [ConfigField] schema.
///
/// Renders one control per field:
///   - `string`   → TextFormField
///   - `secret`   → obscured TextFormField
///   - `boolean`  → Switch
///   - `select`   → Dropdown from `options`
///   - `number`   → numeric-keyboard TextFormField
///   - `args`     → multiline TextFormField (space-separated, roundtripped
///                  as a `List<String>`)
///
/// Respects `dependsOn` / `dependsVal` — fields dependent on a parent
/// value render disabled until the parent matches.
///
/// Save hits `PATCH /api/providers/{name}/config` via
/// [ApiClient.updateProviderConfig]. On success, fires ProvidersBus so
/// other surfaces (the Hub page, the Browser card launcher) refetch.
class PluginConfigPage extends StatefulWidget {
  const PluginConfigPage({required this.info, super.key});

  final ProviderInfo info;

  @override
  State<PluginConfigPage> createState() => _PluginConfigPageState();
}

class _PluginConfigPageState extends State<PluginConfigPage> {
  late Map<String, dynamic> _values;
  final _formKey = GlobalKey<FormState>();
  bool _saving = false;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    // Clone so cancel discards without mutating the ProviderInfo held
    // upstream (the Hub list row would otherwise show stale-but-edited
    // values after the user backs out).
    _values = Map<String, dynamic>.from(widget.info.config);
    // Seed defaults for unset fields — makes the form show placeholders
    // as values, consistent with how the plugin sees them at runtime.
    for (final f in widget.info.provider.configSchema) {
      if (!_values.containsKey(f.key) && f.defaultValue != null) {
        _values[f.key] = f.defaultValue;
      }
    }
  }

  Future<void> _save() async {
    if (!(_formKey.currentState?.validate() ?? false)) return;
    _formKey.currentState!.save();
    setState(() => _saving = true);
    try {
      await _api.updateProviderConfig(widget.info.provider.name, _values);
      ProvidersBus.instance.notify();
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Saved')),
      );
      Navigator.of(context).pop();
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text('Failed: $e'),
        backgroundColor: AppColors.error,
      ));
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  /// True when `field.dependsOn` is empty OR the current value of that
  /// parent key matches `field.dependsVal`. Fields whose dependency
  /// isn't satisfied render disabled but visible.
  bool _isEnabled(ConfigField f) {
    if (f.dependsOn == null || f.dependsOn!.isEmpty) return true;
    final parent = _values[f.dependsOn];
    final want = f.dependsVal;
    if (want == null) return parent != null && parent != false && parent != '';
    return parent?.toString() == want;
  }

  @override
  Widget build(BuildContext context) {
    final schema = widget.info.provider.configSchema;
    return Scaffold(
      appBar: AppBar(
        title: Text('Configure ${widget.info.provider.displayName}'),
        actions: [
          TextButton(
            onPressed: _saving ? null : _save,
            child: Text(_saving ? 'Saving…' : 'Save'),
          ),
        ],
      ),
      body: schema.isEmpty ? _emptyState() : _form(schema),
    );
  }

  Widget _emptyState() {
    return const Center(
      child: Padding(
        padding: EdgeInsets.all(32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.tune, size: 40, color: AppColors.textMuted),
            SizedBox(height: 12),
            Text('This plugin has no user-configurable settings.',
                style: TextStyle(color: AppColors.textMuted, fontSize: 13),
                textAlign: TextAlign.center),
          ],
        ),
      ),
    );
  }

  Widget _form(List<ConfigField> schema) {
    final groups = <String, List<ConfigField>>{};
    for (final f in schema) {
      groups.putIfAbsent(f.group ?? '', () => []).add(f);
    }
    return Form(
      key: _formKey,
      child: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          for (final entry in groups.entries) ...[
            if (entry.key.isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 8, bottom: 6),
                child: Text(
                  entry.key,
                  style: const TextStyle(
                      fontWeight: FontWeight.w600,
                      fontSize: 12,
                      letterSpacing: 0.5,
                      color: AppColors.textMuted),
                ),
              ),
            Card(
              child: Column(
                children: [
                  for (int i = 0; i < entry.value.length; i++) ...[
                    _fieldTile(entry.value[i]),
                    if (i < entry.value.length - 1)
                      const Divider(height: 1, indent: 16, endIndent: 16),
                  ],
                ],
              ),
            ),
            const SizedBox(height: 16),
          ],
        ],
      ),
    );
  }

  Widget _fieldTile(ConfigField f) {
    final enabled = _isEnabled(f);
    final current = _values[f.key] ?? f.defaultValue;
    switch (f.type) {
      case 'boolean':
        return SwitchListTile(
          title: Text(f.label),
          subtitle:
              f.description == null ? null : Text(f.description!),
          value: current == true,
          onChanged: !enabled ? null : (v) => setState(() => _values[f.key] = v),
        );
      case 'select':
        return ListTile(
          title: Text(f.label),
          subtitle: f.description == null ? null : Text(f.description!),
          trailing: DropdownButton<String>(
            value: current?.toString(),
            hint: Text(f.placeholder ?? '—'),
            onChanged: !enabled
                ? null
                : (v) => setState(() => _values[f.key] = v ?? ''),
            items: [
              for (final opt in f.options ?? const [])
                DropdownMenuItem(
                    value: opt.toString(), child: Text(opt.toString())),
            ],
          ),
        );
      case 'args':
        final initial = switch (current) {
          List l => l.map((e) => e.toString()).join(' '),
          String s => s,
          _ => '',
        };
        return Padding(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 10),
          child: TextFormField(
            initialValue: initial,
            enabled: enabled,
            maxLines: 2,
            decoration: InputDecoration(
              labelText: f.label,
              hintText: f.placeholder,
              helperText: f.description,
              border: const OutlineInputBorder(),
            ),
            onSaved: (v) => _values[f.key] = (v ?? '')
                .trim()
                .split(RegExp(r'\s+'))
                .where((s) => s.isNotEmpty)
                .toList(),
          ),
        );
      case 'number':
        return _textTile(
          f,
          enabled,
          current?.toString() ?? '',
          keyboardType: TextInputType.number,
          save: (v) {
            if (v == null || v.isEmpty) {
              _values.remove(f.key);
            } else {
              _values[f.key] = int.tryParse(v) ?? double.tryParse(v) ?? v;
            }
          },
        );
      case 'secret':
        return _textTile(
          f,
          enabled,
          current?.toString() ?? '',
          obscureText: true,
          save: (v) => _values[f.key] = v ?? '',
        );
      case 'string':
      default:
        return _textTile(
          f,
          enabled,
          current?.toString() ?? '',
          save: (v) => _values[f.key] = v ?? '',
        );
    }
  }

  Widget _textTile(
    ConfigField f,
    bool enabled,
    String initial, {
    TextInputType? keyboardType,
    bool obscureText = false,
    required void Function(String?) save,
  }) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 10, 16, 10),
      child: TextFormField(
        initialValue: initial,
        enabled: enabled,
        obscureText: obscureText,
        keyboardType: keyboardType,
        decoration: InputDecoration(
          labelText: f.label,
          hintText: f.placeholder,
          helperText: f.description,
          border: const OutlineInputBorder(),
          suffixIcon: f.envVar != null
              ? Tooltip(
                  message: 'Overrides \$${f.envVar}',
                  child: const Icon(Icons.info_outline, size: 16),
                )
              : null,
        ),
        validator: f.required && enabled
            ? (v) => (v == null || v.isEmpty) ? '${f.label} is required' : null
            : null,
        onSaved: save,
      ),
    );
  }
}
