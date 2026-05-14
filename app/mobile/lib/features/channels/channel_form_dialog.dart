import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/channels/channel_kinds.dart';

// ChannelFormScreen renders a kind-driven form for create or edit on
// a full Scaffold so multi-field configs (slack, feishu, wechat) get
// the whole screen above the keyboard rather than the cramped middle
// strip a dialog leaves them with on a phone.
//
// Field types map to TextField (text, password) or multi-line
// TextField (textarea). Required fields are validated locally before
// submit; optional fields with empty values are dropped from the
// returned config so server defaults apply.
//
// On create the screen returns a Map<String, dynamic> for the new
// channel's config. On edit the same shape is returned, suitable to
// PATCH directly (the server replaces the config wholesale).
class ChannelFormScreen extends StatefulWidget {
  const ChannelFormScreen({
    required this.kind,
    this.initial,
    super.key,
  });

  final ChannelKind kind;
  // Pre-fill values, e.g. existing channel config when editing.
  final Map<String, dynamic>? initial;

  static Future<Map<String, dynamic>?> push({
    required BuildContext context,
    required ChannelKind kind,
    Map<String, dynamic>? initial,
  }) {
    return Navigator.of(context).push<Map<String, dynamic>>(
      MaterialPageRoute<Map<String, dynamic>>(
        builder: (_) => ChannelFormScreen(kind: kind, initial: initial),
        fullscreenDialog: true,
      ),
    );
  }

  @override
  State<ChannelFormScreen> createState() => _ChannelFormScreenState();
}

class _ChannelFormScreenState extends State<ChannelFormScreen> {
  final _ctrls = <String, TextEditingController>{};
  String? _error;

  @override
  void initState() {
    super.initState();
    final init = widget.initial ?? const {};
    for (final f in widget.kind.fields) {
      final v = init[f.name];
      _ctrls[f.name] = TextEditingController(text: v is String ? v : '');
    }
  }

  @override
  void dispose() {
    for (final c in _ctrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  void _submit() {
    final cfg = <String, dynamic>{};
    for (final f in widget.kind.fields) {
      final raw = _ctrls[f.name]?.text.trim() ?? '';
      if (raw.isEmpty) {
        if (f.required) {
          setState(() => _error = '${f.label} is required.');
          return;
        }
        if (f.optional) continue; // server default
      }
      cfg[f.name] = raw;
    }
    Navigator.of(context).pop(cfg);
  }

  @override
  Widget build(BuildContext context) {
    final isEdit = widget.initial != null;
    return Scaffold(
      appBar: AppBar(
        title: Text(isEdit
            ? 'Edit ${widget.kind.label}'
            : 'New ${widget.kind.label} channel'),
        actions: [
          TextButton(
            onPressed: _submit,
            child: Text(isEdit ? 'Save' : 'Create'),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Padding(
            padding: const EdgeInsets.only(bottom: 12),
            child: Text(
              widget.kind.description,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
          for (final f in widget.kind.fields)
            _ChannelFieldEditor(
              field: f,
              controller: _ctrls[f.name]!,
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
          if (widget.kind.webhookBased && !isEdit) ...[
            const SizedBox(height: 16),
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: Theme.of(context)
                    .colorScheme
                    .tertiary
                    .withValues(alpha: 0.12),
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                children: [
                  Icon(
                    Icons.info_outline,
                    size: 18,
                    color: Theme.of(context).colorScheme.tertiary,
                  ),
                  const SizedBox(width: 10),
                  const Expanded(
                    child: Text(
                      'After creation, the channel card will surface a '
                      'webhook URL — paste it into the platform admin '
                      'console to finish setup.',
                      style: TextStyle(fontSize: 12),
                    ),
                  ),
                ],
              ),
            ),
          ],
        ],
      ),
    );
  }
}

class _ChannelFieldEditor extends StatefulWidget {
  const _ChannelFieldEditor({
    required this.field,
    required this.controller,
  });

  final ChannelField field;
  final TextEditingController controller;

  @override
  State<_ChannelFieldEditor> createState() => _ChannelFieldEditorState();
}

class _ChannelFieldEditorState extends State<_ChannelFieldEditor> {
  bool _hidden = true;

  @override
  Widget build(BuildContext context) {
    final f = widget.field;
    final isPassword = f.type == ChannelFieldType.password;
    final isMulti = f.type == ChannelFieldType.textarea;
    return Padding(
      padding: const EdgeInsets.only(bottom: 12),
      child: TextField(
        controller: widget.controller,
        autocorrect: false,
        obscureText: isPassword && _hidden,
        maxLines: isMulti ? 4 : 1,
        keyboardType: isMulti ? TextInputType.multiline : null,
        decoration: InputDecoration(
          labelText: f.label,
          hintText: f.placeholder,
          helperText: f.hint,
          helperMaxLines: 3,
          border: const OutlineInputBorder(),
          suffixIcon: isPassword
              ? IconButton(
                  icon: Icon(
                    _hidden
                        ? Icons.visibility_outlined
                        : Icons.visibility_off_outlined,
                    size: 18,
                  ),
                  onPressed: () => setState(() => _hidden = !_hidden),
                )
              : null,
        ),
      ),
    );
  }
}

// ChannelKindPickerSheet asks the operator which kind they want to
// create. Bridge is intentionally absent — its create flow needs a
// token generator and capability multiselect that don't translate
// cleanly to a sheet, so it stays web-only.
class ChannelKindPickerSheet extends StatelessWidget {
  const ChannelKindPickerSheet({super.key});

  static Future<ChannelKind?> show(BuildContext context) {
    return showModalBottomSheet<ChannelKind>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => const ChannelKindPickerSheet(),
    );
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 14, 16, 4),
            child: Text(
              'Choose channel kind',
              style: Theme.of(context).textTheme.titleSmall,
            ),
          ),
          const Divider(height: 1),
          Flexible(
            child: Builder(builder: (context) {
              final kinds = channelKindsList();
              return ListView.separated(
                shrinkWrap: true,
                itemCount: kinds.length,
                separatorBuilder: (_, __) => Divider(
                  height: 1,
                  color: Theme.of(context).dividerColor,
                ),
                itemBuilder: (_, i) {
                  final k = kinds[i];
                return ListTile(
                  leading: Container(
                    width: 36,
                    height: 36,
                    alignment: Alignment.center,
                    decoration: BoxDecoration(
                      color: Theme.of(context)
                          .colorScheme
                          .primary
                          .withValues(alpha: 0.12),
                      borderRadius: BorderRadius.circular(8),
                    ),
                    child: Text(
                      k.kind[0].toUpperCase(),
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.primary,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  title: Text(k.label),
                  subtitle: Text(
                    k.description,
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  onTap: () => Navigator.of(context).pop(k),
                );
              },
              );
            }),
          ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }
}

// PostCreateWebhookDialog shows the webhook URL the operator must
// paste into the platform admin (feishu / dingtalk / wecom). Only
// shown for kinds with webhookBased=true.
class PostCreateWebhookDialog extends StatelessWidget {
  const PostCreateWebhookDialog({
    required this.serverUrl,
    required this.channelId,
    required this.kind,
    super.key,
  });

  final String serverUrl;
  final String channelId;
  final ChannelKind kind;

  static Future<void> show({
    required BuildContext context,
    required String serverUrl,
    required String channelId,
    required ChannelKind kind,
  }) {
    return showDialog<void>(
      context: context,
      builder: (_) => PostCreateWebhookDialog(
        serverUrl: serverUrl,
        channelId: channelId,
        kind: kind,
      ),
    );
  }

  String get _webhookUrl =>
      '${serverUrl.replaceAll(RegExp(r'/+$'), '')}/api/v1/channels/$channelId/webhook';

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(t.channels.webhookDialog.title(kind: kind.label)),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          if (kind.afterCreateHint != null)
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: Text(
                kind.afterCreateHint!,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
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
              _webhookUrl,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            ),
          ),
        ],
      ),
      actions: [
        TextButton.icon(
          icon: const Icon(Icons.copy_outlined, size: 18),
          label: Text(t.common.copy),
          onPressed: () async {
            await Clipboard.setData(ClipboardData(text: _webhookUrl));
            if (!context.mounted) return;
            ScaffoldMessenger.of(context).showSnackBar(
              SnackBar(
                content: Text(t.channels.webhookDialog.copiedSnack),
                duration: const Duration(seconds: 2),
                behavior: SnackBarBehavior.floating,
              ),
            );
          },
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(t.common.done),
        ),
      ],
    );
  }
}
