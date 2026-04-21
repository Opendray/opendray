import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../shared/theme/app_theme.dart';
import '../workbench/workbench_models.dart';
import '../workbench/workbench_service.dart';

/// The "Open" target for host-form plugins that contribute commands
/// (pg-browser being the canonical example).
///
/// Lists every command the plugin declared in its manifest, each with
/// a Run button. Invoking a command calls the standard plugin command
/// pipeline (`POST /api/plugins/{name}/commands/{id}/invoke`) and
/// renders the returned output — typically a JSON blob from a host
/// sidecar — in a pretty-printed, selectable, copyable view.
///
/// Error paths (permission denied, unavailable, timeout) render
/// inline so the user can retry without leaving the page.
class PluginRunPage extends StatefulWidget {
  final String pluginName;
  final String displayName;

  const PluginRunPage({
    required this.pluginName,
    this.displayName = '',
    super.key,
  });

  @override
  State<PluginRunPage> createState() => _PluginRunPageState();
}

class _PluginRunPageState extends State<PluginRunPage> {
  String? _selectedCommandId;
  bool _busy = false;
  InvokeResult? _result;
  String? _error;

  ApiClient get _api => context.read<ApiClient>();

  @override
  Widget build(BuildContext context) {
    final service = context.watch<WorkbenchService>();
    final myCommands = service.commands
        .where((c) => c.pluginName == widget.pluginName)
        .toList();
    final title =
        widget.displayName.isEmpty ? widget.pluginName : widget.displayName;

    return Scaffold(
      appBar: AppBar(title: Text(title)),
      body: myCommands.isEmpty
          ? _emptyState()
          : ListView(
              padding: const EdgeInsets.all(16),
              children: [
                _header(myCommands.length),
                const SizedBox(height: 12),
                for (final cmd in myCommands) _commandCard(cmd),
                if (_result != null || _error != null) ...[
                  const SizedBox(height: 16),
                  _resultCard(),
                ],
              ],
            ),
    );
  }

  Widget _header(int count) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(14),
        child: Row(
          children: [
            const Icon(Icons.terminal, color: AppColors.accent, size: 20),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(widget.pluginName,
                      style: const TextStyle(
                          fontWeight: FontWeight.w600, fontSize: 14)),
                  Text(
                    '$count command${count == 1 ? '' : 's'} contributed',
                    style: const TextStyle(
                        color: AppColors.textMuted, fontSize: 12),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _emptyState() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.inbox_outlined, size: 40, color: AppColors.textMuted),
            const SizedBox(height: 10),
            const Text('No commands',
                style: TextStyle(
                    fontWeight: FontWeight.w500,
                    fontSize: 14,
                    color: AppColors.text)),
            const SizedBox(height: 4),
            Text(
              '${widget.pluginName} does not contribute any commands to run.',
              textAlign: TextAlign.center,
              style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
            ),
          ],
        ),
      ),
    );
  }

  Widget _commandCard(WorkbenchCommand cmd) {
    final running = _busy && _selectedCommandId == cmd.id;
    return Card(
      margin: const EdgeInsets.only(bottom: 8),
      child: ListTile(
        leading: _iconBadge(cmd.icon),
        title: Text(cmd.title,
            style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500)),
        subtitle: Text(cmd.id,
            style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 11,
                color: AppColors.textMuted)),
        trailing: FilledButton.icon(
          onPressed: _busy ? null : () => _run(cmd),
          icon: running
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.white),
                )
              : const Icon(Icons.play_arrow, size: 16),
          label: Text(running ? 'Running…' : 'Run'),
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
        ),
      ),
    );
  }

  Widget _iconBadge(String icon) {
    final isEmoji = icon.isNotEmpty && icon.length <= 4 && !icon.contains('/');
    return Container(
      width: 32,
      height: 32,
      decoration: BoxDecoration(
        color: AppColors.accent.withValues(alpha: 0.12),
        borderRadius: BorderRadius.circular(7),
      ),
      alignment: Alignment.center,
      child: isEmoji
          ? Text(icon, style: const TextStyle(fontSize: 16))
          : const Icon(Icons.play_arrow, color: AppColors.accent, size: 18),
    );
  }

  Future<void> _run(WorkbenchCommand cmd) async {
    setState(() {
      _busy = true;
      _selectedCommandId = cmd.id;
      _result = null;
      _error = null;
    });
    try {
      final res = await _api.invokePluginCommand(widget.pluginName, cmd.id);
      if (!mounted) return;
      setState(() => _result = res);
    } on PluginPermissionDeniedException catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Permission denied: $e');
    } on PluginCommandUnavailableException catch (e) {
      if (!mounted) return;
      setState(() => _error = 'Command unavailable: $e');
    } catch (e) {
      if (!mounted) return;
      setState(() => _error = e.toString());
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  Widget _resultCard() {
    if (_error != null) {
      return Card(
        color: AppColors.errorSoft,
        child: Padding(
          padding: const EdgeInsets.all(12),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              const Icon(Icons.error_outline,
                  color: AppColors.error, size: 18),
              const SizedBox(width: 8),
              Expanded(
                child: Text(
                  _error!,
                  style: const TextStyle(
                      color: AppColors.error, fontSize: 12, height: 1.4),
                ),
              ),
            ],
          ),
        ),
      );
    }
    final r = _result!;
    final pretty = _prettifyOutput(r.output);
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                const Icon(Icons.check_circle,
                    color: AppColors.accent, size: 18),
                const SizedBox(width: 8),
                Text('Result (${r.kind})',
                    style: const TextStyle(
                        fontWeight: FontWeight.w600, fontSize: 13)),
                const Spacer(),
                IconButton(
                  icon: const Icon(Icons.copy, size: 16),
                  tooltip: 'Copy output',
                  onPressed: () async {
                    await Clipboard.setData(ClipboardData(text: pretty));
                    if (!mounted) return;
                    ScaffoldMessenger.maybeOf(context)?.showSnackBar(
                      const SnackBar(content: Text('Output copied')),
                    );
                  },
                ),
              ],
            ),
            const SizedBox(height: 8),
            Container(
              width: double.infinity,
              padding: const EdgeInsets.all(10),
              decoration: BoxDecoration(
                color: AppColors.surfaceAlt,
                borderRadius: BorderRadius.circular(6),
              ),
              child: SelectableText(
                pretty.isEmpty ? '(empty)' : pretty,
                style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 11,
                    height: 1.4),
              ),
            ),
          ],
        ),
      ),
    );
  }

  /// Host-form commands return their JSON payload as a string in
  /// `InvokeResult.output`. Pretty-print it when it parses; fall back
  /// to the raw string otherwise so errors aren't hidden.
  String _prettifyOutput(String raw) {
    if (raw.isEmpty) return '';
    try {
      final decoded = jsonDecode(raw);
      const encoder = JsonEncoder.withIndent('  ');
      return encoder.convert(decoded);
    } catch (_) {
      return raw;
    }
  }
}
