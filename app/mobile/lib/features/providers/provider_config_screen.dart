import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/providers_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/providers/claude_accounts_section.dart';

// Provider config editor — schema-driven so we don't carry per-
// provider hardcoded forms. The manifest's configSchema describes
// each user-tweakable setting (Type / Default / Options / etc.); we
// render an input widget per type and PATCH the merged map back.
//
// Fields are grouped by their `group` annotation (matches web admin).
// Conditional fields (DependsOn / DependsVal) hide unless their
// dependency evaluates true so the form stays calm when the user
// isn't yet using a feature.
class ProviderConfigScreen extends ConsumerStatefulWidget {
  const ProviderConfigScreen({required this.providerId, super.key});
  final String providerId;

  @override
  ConsumerState<ProviderConfigScreen> createState() =>
      _ProviderConfigScreenState();
}

class _ProviderConfigScreenState
    extends ConsumerState<ProviderConfigScreen> {
  AsyncValue<ProviderDetail> _state = const AsyncValue.loading();
  // Working copy of the config — diverges from detail.config until save.
  Map<String, dynamic> _values = {};
  bool _saving = false;
  bool _dirty = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final p = await ref.read(providersApiProvider).get(widget.providerId);
      if (!mounted) return;
      // Seed the working values from the persisted config — fall
      // back to schema defaults for any missing keys so the user
      // sees the live runtime values, not blanks.
      final initial = <String, dynamic>{};
      for (final f in p.configSchema) {
        if (p.config.containsKey(f.key)) {
          initial[f.key] = p.config[f.key];
        } else if (f.defaultValue != null) {
          initial[f.key] = f.defaultValue;
        }
      }
      setState(() {
        _state = AsyncValue.data(p);
        _values = initial;
        _dirty = false;
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _save() async {
    if (_saving) return;
    setState(() => _saving = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      // Send only fields actually present in _values — avoids
      // clobbering keys we don't know about (e.g. ones added by a
      // newer manifest the mobile build hasn't shipped yet).
      await ref
          .read(providersApiProvider)
          .updateConfig(widget.providerId, _values);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.providers.configSaved),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(t.providers.saveFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.providers.saveFailedGeneric(error: e.toString())),
        ),
      );
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  void _setValue(String key, dynamic value) {
    setState(() {
      if (value == null) {
        _values.remove(key);
      } else {
        _values[key] = value;
      }
      _dirty = true;
    });
  }

  bool _shouldShow(ConfigField f) {
    if (f.dependsOn == null || f.dependsOn!.isEmpty) return true;
    final actual = _values[f.dependsOn!];
    final expected = f.dependsVal;
    if (expected == null) {
      // dependsOn truthy = treat any non-empty/non-false as "shown".
      if (actual == null) return false;
      if (actual is bool) return actual;
      if (actual is String) return actual.isNotEmpty;
      return true;
    }
    return actual == expected;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(_state.maybeWhen(
          data: (d) => d.displayName,
          orElse: () => 'Provider config',
        )),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.providers.reload,
            onPressed: _saving ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildEditor,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
      floatingActionButton: _dirty
          ? FloatingActionButton.extended(
              heroTag: 'provider_config_fab',
              onPressed: _saving ? null : _save,
              icon: _saving
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.save_outlined),
              label: Text(_saving ? 'Saving…' : 'Save'),
            )
          : null,
    );
  }

  Widget _buildEditor(ProviderDetail p) {
    final isClaude = p.id == 'claude';
    final groups = <String, List<ConfigField>>{};
    for (final f in p.configSchema) {
      if (!_shouldShow(f)) continue;
      final g = f.group ?? '';
      groups.putIfAbsent(g, () => []).add(f);
    }
    final hasFields = groups.isNotEmpty;
    if (!hasFields && !isClaude) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            '${p.displayName} declares no user-configurable settings.',
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return ListView(
      padding: const EdgeInsets.fromLTRB(0, 8, 0, 80),
      children: [
        if (p.description.isNotEmpty)
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 4, 16, 12),
            child: Text(
              p.description,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        if (hasFields) ...[
          for (final entry in groups.entries) ...[
            if (entry.key.isNotEmpty) _SectionHeader(label: entry.key),
            for (final f in entry.value)
              Padding(
                padding: const EdgeInsets.fromLTRB(16, 6, 16, 6),
                child: _FieldEditor(
                  field: f,
                  value: _values[f.key],
                  onChanged: (v) => _setValue(f.key, v),
                ),
              ),
          ],
        ],
        if (isClaude)
          const Padding(
            padding: EdgeInsets.fromLTRB(16, 8, 16, 0),
            child: ClaudeAccountsSection(),
          ),
      ],
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 14, 16, 4),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              letterSpacing: 0.8,
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.6),
            ),
      ),
    );
  }
}

class _FieldEditor extends StatelessWidget {
  const _FieldEditor({
    required this.field,
    required this.value,
    required this.onChanged,
  });

  final ConfigField field;
  final dynamic value;
  final ValueChanged<dynamic> onChanged;

  @override
  Widget build(BuildContext context) {
    switch (field.type) {
      case 'boolean':
        return _BoolField(field: field, value: value, onChanged: onChanged);
      case 'select':
        return _SelectField(field: field, value: value, onChanged: onChanged);
      case 'number':
        return _TextField(
          field: field,
          value: value,
          onChanged: onChanged,
          keyboardType: const TextInputType.numberWithOptions(
            decimal: true,
            signed: true,
          ),
          isNumeric: true,
        );
      case 'secret':
        return _TextField(
          field: field,
          value: value,
          onChanged: onChanged,
          obscureText: true,
        );
      case 'args':
        return _TextField(
          field: field,
          value: value,
          onChanged: onChanged,
          helperOverride: 'Whitespace-separated CLI args.',
        );
      // string and unknown types fall through to text.
      case 'string':
      default:
        return _TextField(field: field, value: value, onChanged: onChanged);
    }
  }
}

class _BoolField extends StatelessWidget {
  const _BoolField({
    required this.field,
    required this.value,
    required this.onChanged,
  });
  final ConfigField field;
  final dynamic value;
  final ValueChanged<dynamic> onChanged;

  @override
  Widget build(BuildContext context) {
    final v = value is bool ? value as bool : (field.defaultValue == true);
    return SwitchListTile(
      contentPadding: EdgeInsets.zero,
      title: Text(field.label),
      subtitle: (field.description ?? '').isEmpty
          ? null
          : Text(
              field.description!,
              style: Theme.of(context).textTheme.bodySmall,
            ),
      value: v,
      onChanged: onChanged,
    );
  }
}

class _SelectField extends StatelessWidget {
  const _SelectField({
    required this.field,
    required this.value,
    required this.onChanged,
  });
  final ConfigField field;
  final dynamic value;
  final ValueChanged<dynamic> onChanged;

  @override
  Widget build(BuildContext context) {
    final options = field.options ?? const <String>[];
    String? selected;
    if (value is String) {
      selected = value as String;
    } else if (field.defaultValue is String) {
      selected = field.defaultValue! as String;
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(field.label, style: Theme.of(context).textTheme.bodyMedium),
        if ((field.description ?? '').isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 2),
            child: Text(
              field.description!,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
        const SizedBox(height: 4),
        DropdownButtonFormField<String>(
          initialValue: options.contains(selected) ? selected : null,
          isDense: true,
          items: [
            for (final o in options)
              DropdownMenuItem(value: o, child: Text(o)),
          ],
          onChanged: onChanged,
          decoration: InputDecoration(
            border: const OutlineInputBorder(),
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            hintText: field.placeholder,
          ),
        ),
      ],
    );
  }
}

class _TextField extends StatefulWidget {
  const _TextField({
    required this.field,
    required this.value,
    required this.onChanged,
    this.keyboardType,
    this.obscureText = false,
    this.isNumeric = false,
    this.helperOverride,
  });

  final ConfigField field;
  final dynamic value;
  final ValueChanged<dynamic> onChanged;
  final TextInputType? keyboardType;
  final bool obscureText;
  final bool isNumeric;
  final String? helperOverride;

  @override
  State<_TextField> createState() => _TextFieldState();
}

class _TextFieldState extends State<_TextField> {
  late TextEditingController _ctrl;

  @override
  void initState() {
    super.initState();
    _ctrl = TextEditingController(text: _initialText());
  }

  String _initialText() {
    final v = widget.value;
    if (v == null) {
      final d = widget.field.defaultValue;
      return d == null ? '' : '$d';
    }
    return '$v';
  }

  @override
  void didUpdateWidget(_TextField oldWidget) {
    super.didUpdateWidget(oldWidget);
    // When the parent reloads the form (e.g. after Save → reload),
    // sync the controller to the new external value.
    final external = _initialText();
    if (external != _ctrl.text) {
      _ctrl.text = external;
    }
  }

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  void _onChanged(String text) {
    if (widget.isNumeric) {
      if (text.isEmpty) {
        widget.onChanged(null);
        return;
      }
      final n = num.tryParse(text);
      if (n != null) widget.onChanged(n);
      return;
    }
    widget.onChanged(text);
  }

  @override
  Widget build(BuildContext context) {
    return TextField(
      controller: _ctrl,
      keyboardType: widget.keyboardType,
      obscureText: widget.obscureText,
      autocorrect: false,
      decoration: InputDecoration(
        labelText: widget.field.label,
        hintText: widget.field.placeholder,
        helperText: widget.helperOverride ?? widget.field.description,
        helperMaxLines: 3,
        isDense: true,
        suffixIcon: widget.field.envVar == null
            ? null
            : Tooltip(
                message: 'Falls back to env: ${widget.field.envVar}',
                child: const Icon(Icons.public_outlined, size: 16),
              ),
      ),
      onChanged: _onChanged,
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final String error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              'Failed to load provider',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
