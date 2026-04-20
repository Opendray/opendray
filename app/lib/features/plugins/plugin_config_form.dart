import 'package:flutter/material.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/theme/app_theme.dart';

/// Renders a form from a [PluginConfigField] schema + initial values.
///
/// Used in two places:
///   - Hub install flow, after the consent dialog, for plugins whose
///     manifest declares a configSchema.
///   - Plugin detail page "Configure" action, prefilled with the
///     current GET /config values.
///
/// The caller owns persistence — onSave receives the draft map and
/// performs the PUT + any post-save navigation. This widget stays
/// stateless about the server and only validates required fields.
class PluginConfigForm extends StatefulWidget {
  /// Manifest-declared fields. Rendered in declaration order and
  /// grouped by [PluginConfigField.group] — empty groups fall under a
  /// single "General" heading.
  final List<PluginConfigField> schema;

  /// Initial values keyed by field. For secret fields the caller
  /// should pass [PluginConfig.kSecretSet] when a value is already
  /// stored (GET /config semantics); the widget renders a "stored —
  /// leave blank to keep" helper.
  final Map<String, String> initialValues;

  /// Called when the user submits the form. Receives the drafts as a
  /// flat string map ready for [PluginConfig.toPutBody].
  final Future<void> Function(Map<String, String> drafts) onSave;

  /// Text of the submit button. Defaults to "Save".
  final String submitLabel;

  /// Optional cancel handler — when null the cancel button is hidden
  /// (used in Hub install flow where the outer dialog owns cancel).
  final VoidCallback? onCancel;

  const PluginConfigForm({
    required this.schema,
    required this.initialValues,
    required this.onSave,
    this.submitLabel = 'Save',
    this.onCancel,
    super.key,
  });

  @override
  State<PluginConfigForm> createState() => _PluginConfigFormState();
}

class _PluginConfigFormState extends State<PluginConfigForm> {
  final _formKey = GlobalKey<FormState>();
  final Map<String, TextEditingController> _ctrls = {};
  final Map<String, bool> _bools = {};
  final Map<String, String> _selects = {};
  final Set<String> _secretCleared = {};
  bool _busy = false;

  @override
  void initState() {
    super.initState();
    for (final f in widget.schema) {
      final initial = widget.initialValues[f.key] ?? _fromDefault(f);
      switch (f.type) {
        case 'bool':
        case 'boolean':
          _bools[f.key] = initial == 'true';
          break;
        case 'select':
          _selects[f.key] = initial.isEmpty && f.options.isNotEmpty
              ? f.options.first
              : initial;
          break;
        case 'secret':
          // Secret fields start blank. We track whether the server
          // reported a stored value via the sentinel — the helper
          // text tells the user "leave blank to keep existing".
          _ctrls[f.key] = TextEditingController();
          break;
        default:
          _ctrls[f.key] = TextEditingController(text: initial);
      }
    }
  }

  @override
  void dispose() {
    for (final c in _ctrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  String _fromDefault(PluginConfigField f) {
    final d = f.defaultValue;
    if (d == null) return '';
    return d.toString();
  }

  /// Secret fields render differently when the server reported a
  /// stored value (sentinel) vs an empty one — we keep the flag so
  /// the helper text is accurate.
  bool _hasStoredSecret(PluginConfigField f) =>
      f.isSecret &&
      (widget.initialValues[f.key] ?? '') == PluginConfig.kSecretSet &&
      !_secretCleared.contains(f.key);

  Map<String, String> _collectDrafts() {
    final drafts = <String, String>{};
    for (final f in widget.schema) {
      switch (f.type) {
        case 'bool':
        case 'boolean':
          drafts[f.key] = (_bools[f.key] ?? false) ? 'true' : 'false';
          break;
        case 'select':
          drafts[f.key] = _selects[f.key] ?? '';
          break;
        default:
          drafts[f.key] = _ctrls[f.key]?.text ?? '';
      }
    }
    return drafts;
  }

  Future<void> _submit() async {
    if (!(_formKey.currentState?.validate() ?? true)) return;
    setState(() => _busy = true);
    try {
      await widget.onSave(_collectDrafts());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final byGroup = <String, List<PluginConfigField>>{};
    for (final f in widget.schema) {
      byGroup.putIfAbsent(f.group.isEmpty ? 'General' : f.group, () => []).add(f);
    }
    return Form(
      key: _formKey,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        mainAxisSize: MainAxisSize.min,
        children: [
          for (final entry in byGroup.entries) ...[
            if (byGroup.length > 1 || entry.key != 'General')
              Padding(
                padding: const EdgeInsets.only(bottom: 6, top: 6),
                child: Text(
                  entry.key[0].toUpperCase() + entry.key.substring(1),
                  style: const TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                      color: AppColors.textMuted),
                ),
              ),
            for (final f in entry.value) _fieldWidget(f),
          ],
          const SizedBox(height: 8),
          Row(
            mainAxisAlignment: MainAxisAlignment.end,
            children: [
              if (widget.onCancel != null)
                TextButton(
                  onPressed: _busy ? null : widget.onCancel,
                  child: const Text('Cancel'),
                ),
              const SizedBox(width: 6),
              FilledButton(
                onPressed: _busy ? null : _submit,
                style:
                    FilledButton.styleFrom(backgroundColor: AppColors.accent),
                child: _busy
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(
                            strokeWidth: 2, color: Colors.white),
                      )
                    : Text(widget.submitLabel),
              ),
            ],
          ),
        ],
      ),
    );
  }

  Widget _fieldWidget(PluginConfigField f) {
    // Manifest-owned text uses context.pickL10n; the widget rebuilds
    // on locale switch because pickL10n subscribes via watch<L10n>.
    final label = context.pickL10n(f.label, f.labelZh);
    final description = context.pickL10n(f.description, f.descriptionZh);
    final placeholder = context.pickL10n(f.placeholder, f.placeholderZh);
    final requiredLabel = label + (f.required ? ' *' : '');
    String requiredError() => '$label is required';
    switch (f.type) {
      case 'bool':
      case 'boolean':
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: Row(
            children: [
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(label,
                        style: const TextStyle(fontSize: 13)),
                    if (description.isNotEmpty)
                      Text(description,
                          style: const TextStyle(
                              fontSize: 11, color: AppColors.textMuted)),
                  ],
                ),
              ),
              Switch(
                value: _bools[f.key] ?? false,
                activeTrackColor: AppColors.accent,
                onChanged: _busy
                    ? null
                    : (v) => setState(() => _bools[f.key] = v),
              ),
            ],
          ),
        );
      case 'select':
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: DropdownButtonFormField<String>(
            initialValue: _selects[f.key],
            decoration: InputDecoration(
              labelText: requiredLabel,
              helperText: description.isEmpty ? null : description,
              isDense: true,
            ),
            items: [
              for (final opt in f.options)
                DropdownMenuItem(value: opt, child: Text(opt)),
            ],
            validator: (v) {
              if (f.required && (v == null || v.isEmpty)) {
                return requiredError();
              }
              return null;
            },
            onChanged: _busy
                ? null
                : (v) => setState(() => _selects[f.key] = v ?? ''),
          ),
        );
      case 'secret':
        final stored = _hasStoredSecret(f);
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: TextFormField(
            controller: _ctrls[f.key],
            obscureText: true,
            enabled: !_busy,
            decoration: InputDecoration(
              labelText: requiredLabel,
              hintText:
                  stored ? '(stored — leave blank to keep)' : placeholder,
              helperText: description.isEmpty ? null : description,
              isDense: true,
            ),
            onChanged: (_) {
              if (stored) setState(() => _secretCleared.add(f.key));
            },
            validator: (v) {
              // Required + no value + no stored value → error.
              if (f.required && (v == null || v.isEmpty) && !_hasStoredSecret(f)) {
                return requiredError();
              }
              return null;
            },
          ),
        );
      case 'number':
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: TextFormField(
            controller: _ctrls[f.key],
            keyboardType: TextInputType.number,
            enabled: !_busy,
            decoration: InputDecoration(
              labelText: requiredLabel,
              hintText: placeholder,
              helperText: description.isEmpty ? null : description,
              isDense: true,
            ),
            validator: (v) {
              if (f.required && (v == null || v.isEmpty)) {
                return requiredError();
              }
              if (v != null && v.isNotEmpty && num.tryParse(v) == null) {
                return '$label must be numeric';
              }
              return null;
            },
          ),
        );
      default:
        // string / text
        return Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: TextFormField(
            controller: _ctrls[f.key],
            enabled: !_busy,
            decoration: InputDecoration(
              labelText: requiredLabel,
              hintText: placeholder,
              helperText: description.isEmpty ? null : description,
              isDense: true,
            ),
            validator: (v) {
              if (f.required && (v == null || v.isEmpty)) {
                return requiredError();
              }
              return null;
            },
          ),
        );
    }
  }
}
