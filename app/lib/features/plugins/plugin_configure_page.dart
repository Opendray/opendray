import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/services/l10n.dart';
import '../../shared/theme/app_theme.dart';
import 'plugin_config_form.dart';

/// Configure-form entry point from Plugin → popup menu → Configure.
///
/// Pulls the current schema + values from GET /api/plugins/{name}/config,
/// renders a [PluginConfigForm], and on save PUTs back. The server
/// handles sidecar restart — by the time this page pops, the plugin
/// is already picking up new config on its next invoke.
class PluginConfigurePage extends StatefulWidget {
  final String pluginName;
  final String displayName;

  const PluginConfigurePage({
    required this.pluginName,
    this.displayName = '',
    super.key,
  });

  @override
  State<PluginConfigurePage> createState() => _PluginConfigurePageState();
}

class _PluginConfigurePageState extends State<PluginConfigurePage> {
  PluginConfig? _config;
  String? _error;
  bool _loading = true;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    if (!mounted) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final c = await _api.getPluginConfig(widget.pluginName);
      if (!mounted) return;
      setState(() {
        _config = c;
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

  Future<void> _save(Map<String, String> drafts) async {
    final cfg = _config;
    if (cfg == null) return;
    final body = cfg.toPutBody(drafts);
    try {
      await _api.putPluginConfig(widget.pluginName, body);
      if (!mounted) return;
      ScaffoldMessenger.maybeOf(context)?.showSnackBar(
        SnackBar(content: Text('${context.trOnce('Saved')} ${widget.pluginName}')),
      );
      Navigator.of(context).pop(true);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.maybeOf(context)?.showSnackBar(
        SnackBar(
          content: Text('${context.trOnce('Save failed')}: $e'),
          backgroundColor: AppColors.error,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final title = widget.displayName.isEmpty
        ? widget.pluginName
        : widget.displayName;
    return Scaffold(
      appBar: AppBar(title: Text('${context.tr('Configure')} $title')),
      body: _buildBody(),
    );
  }

  Widget _buildBody() {
    if (_loading) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_error != null) {
      return Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(context.tr('Failed to load config'),
                style: const TextStyle(
                    color: AppColors.error, fontWeight: FontWeight.w500)),
            const SizedBox(height: 6),
            Text(_error ?? '',
                style: const TextStyle(color: AppColors.error, fontSize: 12)),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: _load,
              style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
              child: Text(context.tr('Retry')),
            ),
          ],
        ),
      );
    }
    final cfg = _config!;
    if (cfg.schema.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            context.tr('This plugin has no configurable fields.'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 13),
            textAlign: TextAlign.center,
          ),
        ),
      );
    }
    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: PluginConfigForm(
            schema: cfg.schema,
            initialValues: cfg.values,
            submitLabel: context.tr('Save'),
            onCancel: () => Navigator.of(context).pop(false),
            onSave: _save,
          ),
        ),
      ),
    );
  }
}
