import 'dart:async';
import 'dart:io';

import 'package:file_picker/file_picker.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/backups_api.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// User-level data export & import — mirrors the web Export page.
// Two halves on one scrollable surface: Export (create + history)
// and Import (upload + history). Bundles expire 7 days after
// creation; download URL is a single-use bearer.
//
// Mobile UX trade-off: the web "click to download" maps to "copy
// URL to clipboard" here (single-use token paste-into-browser).
// Avoids adding url_launcher just to download an admin blob.
class DataExportScreen extends ConsumerStatefulWidget {
  const DataExportScreen({super.key});

  @override
  ConsumerState<DataExportScreen> createState() => _DataExportScreenState();
}

class _DataExportScreenState extends ConsumerState<DataExportScreen> {
  Timer? _refreshTimer;
  List<ExportRecord>? _exports;
  List<ImportRecord>? _imports;
  Object? _exportsError;
  Object? _importsError;

  @override
  void initState() {
    super.initState();
    unawaited(_refresh());
    // Poll while the screen is mounted — pending → running → ready
    // transitions are server-driven and the user can't push them.
    _refreshTimer = Timer.periodic(
      const Duration(seconds: 5),
      (_) => unawaited(_refresh(silent: true)),
    );
  }

  @override
  void dispose() {
    _refreshTimer?.cancel();
    super.dispose();
  }

  Future<void> _refresh({bool silent = false}) async {
    final api = ref.read(backupsApiProvider);
    final results = await Future.wait<Object>([
      api.listExports().catchError((Object e) {
        if (!silent && mounted) setState(() => _exportsError = e);
        return <ExportRecord>[];
      }),
      api.listImports().catchError((Object e) {
        if (!silent && mounted) setState(() => _importsError = e);
        return <ImportRecord>[];
      }),
    ]);
    if (!mounted) return;
    setState(() {
      _exports = results[0] as List<ExportRecord>;
      _imports = results[1] as List<ImportRecord>;
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.dataExport.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.common.refresh,
            onPressed: () => unawaited(_refresh()),
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: ListView(
          physics: const AlwaysScrollableScrollPhysics(),
          padding: const EdgeInsets.fromLTRB(12, 12, 12, 24),
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4),
              child: Text(
                t.dataExport.subtitle,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ),
            const SizedBox(height: 16),
            _SectionHeader(label: t.dataExport.sections.export),
            const SizedBox(height: 8),
            _ExportForm(onCreated: () => unawaited(_refresh())),
            const SizedBox(height: 16),
            _ExportHistory(
              rows: _exports,
              error: _exportsError,
              onChanged: () => unawaited(_refresh()),
            ),
            const SizedBox(height: 24),
            _SectionHeader(label: t.dataExport.sections.import),
            const SizedBox(height: 8),
            _ImportForm(onDone: () => unawaited(_refresh())),
            const SizedBox(height: 16),
            _ImportHistory(rows: _imports, error: _importsError),
          ],
        ),
      ),
    );
  }
}

// ── shared bits ─────────────────────────────────────────────────

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(4, 4, 4, 6),
      child: Container(
        padding: const EdgeInsets.only(bottom: 6),
        decoration: BoxDecoration(
          border: Border(
            bottom: BorderSide(color: Theme.of(context).dividerColor),
          ),
        ),
        child: Text(
          label.toUpperCase(),
          style: Theme.of(context).textTheme.labelMedium?.copyWith(
            letterSpacing: 1.2,
            color: Theme.of(context).colorScheme.outline,
          ),
        ),
      ),
    );
  }
}

// ── export form ─────────────────────────────────────────────────

class _ExportForm extends ConsumerStatefulWidget {
  const _ExportForm({required this.onCreated});
  final VoidCallback onCreated;

  @override
  ConsumerState<_ExportForm> createState() => _ExportFormState();
}

class _ExportFormState extends ConsumerState<_ExportForm> {
  bool _memories = true;
  bool _customTasks = true;
  IntegrationExportMode _integrations = IntegrationExportMode.metadata;
  final _confirmCtrl = TextEditingController();
  bool _busy = false;

  @override
  void dispose() {
    _confirmCtrl.dispose();
    super.dispose();
  }

  bool get _wantsPlaintext => _integrations == IntegrationExportMode.plaintext;
  bool get _confirmReady =>
      !_wantsPlaintext ||
      _confirmCtrl.text.trim() == t.dataExport.form.confirmSentinel;

  Future<void> _submit() async {
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final exp = await ref
          .read(backupsApiProvider)
          .createExport(
            memories: _memories,
            integrations: _integrations,
            customTasks: _customTasks,
          );
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.dataExport.form.readyDescription(bytes: exp.bytes.toString()),
          ),
        ),
      );
      widget.onCreated();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.dataExport.form.failedToast(error: e.message)),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.dataExport.form.failedToast(error: e.toString())),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              t.dataExport.form.scope.toUpperCase(),
              style: theme.textTheme.labelSmall?.copyWith(
                letterSpacing: 1.2,
                color: theme.colorScheme.outline,
              ),
            ),
            const SizedBox(height: 8),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              value: _memories,
              onChanged: _busy ? null : (v) => setState(() => _memories = v),
              title: Text(t.dataExport.form.memories),
              subtitle: Text(t.dataExport.form.memoriesHint),
            ),
            const SizedBox(height: 4),
            Text(
              t.dataExport.form.integrations,
              style: theme.textTheme.labelMedium,
            ),
            const SizedBox(height: 4),
            _RadioRow(
              checked: _integrations == IntegrationExportMode.none,
              onTap: _busy
                  ? null
                  : () => setState(
                      () => _integrations = IntegrationExportMode.none,
                    ),
              label: t.dataExport.form.integrationOptions.none,
              hint: t.dataExport.form.integrationOptions.noneHint,
            ),
            _RadioRow(
              checked: _integrations == IntegrationExportMode.metadata,
              onTap: _busy
                  ? null
                  : () => setState(
                      () => _integrations = IntegrationExportMode.metadata,
                    ),
              label: t.dataExport.form.integrationOptions.metadata,
              hint: t.dataExport.form.integrationOptions.metadataHint,
            ),
            _RadioRow(
              checked: _integrations == IntegrationExportMode.plaintext,
              onTap: _busy
                  ? null
                  : () => setState(
                      () => _integrations = IntegrationExportMode.plaintext,
                    ),
              label: t.dataExport.form.integrationOptions.plaintext,
              hint: t.dataExport.form.integrationOptions.plaintextHint,
              danger: true,
            ),
            if (_wantsPlaintext) ...[
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: theme.colorScheme.errorContainer.withValues(
                    alpha: 0.4,
                  ),
                  border: Border.all(color: theme.colorScheme.error),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(
                      children: [
                        Icon(
                          Icons.shield_outlined,
                          size: 16,
                          color: theme.colorScheme.error,
                        ),
                        const SizedBox(width: 6),
                        Expanded(
                          child: Text(
                            t.dataExport.form.confirmWarning,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.error,
                            ),
                          ),
                        ),
                      ],
                    ),
                    const SizedBox(height: 8),
                    TextField(
                      controller: _confirmCtrl,
                      enabled: !_busy,
                      decoration: InputDecoration(
                        isDense: true,
                        border: const OutlineInputBorder(),
                        hintText: t.dataExport.form.confirmPlaceholder,
                      ),
                      onChanged: (_) => setState(() {}),
                    ),
                  ],
                ),
              ),
            ],
            const SizedBox(height: 12),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              value: _customTasks,
              onChanged: _busy ? null : (v) => setState(() => _customTasks = v),
              title: Text(t.dataExport.form.customTasks),
              subtitle: Text(t.dataExport.form.customTasksHint),
            ),
            const Divider(height: 24),
            Row(
              children: [
                Expanded(
                  child: Text(
                    t.dataExport.form.footnote,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                    ),
                  ),
                ),
                FilledButton(
                  onPressed: _busy || !_confirmReady ? null : _submit,
                  child: Text(
                    _busy
                        ? t.dataExport.form.building
                        : t.dataExport.form.create,
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

class _RadioRow extends StatelessWidget {
  const _RadioRow({
    required this.checked,
    required this.onTap,
    required this.label,
    required this.hint,
    this.danger = false,
  });
  final bool checked;
  final VoidCallback? onTap;
  final String label;
  final String hint;
  final bool danger;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final accent = danger ? theme.colorScheme.error : theme.colorScheme.primary;
    return InkWell(
      onTap: onTap,
      borderRadius: BorderRadius.circular(6),
      child: Container(
        margin: const EdgeInsets.only(bottom: 4),
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 6),
        decoration: BoxDecoration(
          border: Border.all(
            color: checked ? accent : theme.dividerColor,
            width: checked ? 1.5 : 0.5,
          ),
          borderRadius: BorderRadius.circular(6),
          color: checked ? accent.withValues(alpha: 0.06) : Colors.transparent,
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Container(
              width: 16,
              height: 16,
              margin: const EdgeInsets.only(top: 2, right: 8),
              decoration: BoxDecoration(
                shape: BoxShape.circle,
                border: Border.all(
                  color: checked ? accent : theme.colorScheme.outline,
                ),
              ),
              child: Center(
                child: checked
                    ? Container(
                        width: 8,
                        height: 8,
                        decoration: BoxDecoration(
                          shape: BoxShape.circle,
                          color: accent,
                        ),
                      )
                    : const SizedBox(),
              ),
            ),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(label, style: theme.textTheme.bodyMedium),
                  Text(
                    hint,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

// ── export history ───────────────────────────────────────────────

class _ExportHistory extends ConsumerStatefulWidget {
  const _ExportHistory({
    required this.rows,
    required this.error,
    required this.onChanged,
  });
  final List<ExportRecord>? rows;
  final Object? error;
  final VoidCallback onChanged;

  @override
  ConsumerState<_ExportHistory> createState() => _ExportHistoryState();
}

class _ExportHistoryState extends ConsumerState<_ExportHistory> {
  // Caches per-id download tokens. The list endpoint redacts the
  // token (sensitive); /exports/{id} returns it once. We refetch on
  // demand and stash so subsequent taps don't burn the rate limit.
  final Map<String, String> _tokenCache = {};

  Future<void> _onDownload(ExportRecord exp) async {
    try {
      var token = _tokenCache[exp.id];
      if (token == null) {
        final detail = await ref.read(backupsApiProvider).getExport(exp.id);
        token = detail.downloadToken;
      }
      if (token == null || token.isEmpty) {
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(t.dataExport.history.noTokenToast)),
        );
        return;
      }
      _tokenCache[exp.id] = token;
      final url = ref.read(backupsApiProvider).exportDownloadUrl(exp.id, token);
      final auth = ref.read(authControllerProvider);
      final base = switch (auth) {
        AuthLoggedIn(serverUrl: final s) => s,
        AuthLoggedOut(serverUrl: final s) => s,
        _ => '',
      };
      final full = base.isEmpty
          ? url
          : '${base.replaceAll(RegExp(r'/+$'), '')}$url';
      await Clipboard.setData(ClipboardData(text: full));
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(t.dataExport.history.downloadCopiedToast)),
      );
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.dataExport.history.downloadFailedToast(error: e.message),
          ),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.dataExport.history.downloadFailedToast(error: e.toString()),
          ),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  Future<void> _onDelete(ExportRecord exp) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.dataExport.history.deleteConfirmTitle),
        content: Text(t.dataExport.history.deleteConfirmBody(id: exp.id)),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.common.delete),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(backupsApiProvider).deleteExport(exp.id);
      _tokenCache.remove(exp.id);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text(t.dataExport.history.deletedToast)),
      );
      widget.onChanged();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.dataExport.history.deleteFailedToast(error: e.message),
          ),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.dataExport.history.deleteFailedToast(error: e.toString()),
          ),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final rows = widget.rows;
    final err = widget.error;
    if (rows == null && err == null) {
      return Text(
        t.dataExport.history.loading,
        style: theme.textTheme.bodySmall,
      );
    }
    if (err != null && rows == null) {
      return Text(
        t.dataExport.history.listFailedToast(
          error: err is ApiException ? err.message : err.toString(),
        ),
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.error,
        ),
      );
    }
    if (rows == null || rows.isEmpty) {
      return Text(t.dataExport.history.empty, style: theme.textTheme.bodySmall);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(t.dataExport.history.title, style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        ...rows.map(
          (exp) => _ExportRow(
            exp: exp,
            onDownload: () => unawaited(_onDownload(exp)),
            onDelete: () => unawaited(_onDelete(exp)),
          ),
        ),
      ],
    );
  }
}

class _ExportRow extends StatelessWidget {
  const _ExportRow({
    required this.exp,
    required this.onDownload,
    required this.onDelete,
  });
  final ExportRecord exp;
  final VoidCallback onDownload;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final ready = exp.status == 'ready';
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 8, 8, 8),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: SelectableText(
                    exp.id,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 11,
                    ),
                  ),
                ),
                _StatusBadge(status: exp.status),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              _scopeSummary(exp.scope),
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
            const SizedBox(height: 2),
            Row(
              children: [
                Text(
                  exp.bytes > 0 ? _formatBytes(exp.bytes) : '—',
                  style: theme.textTheme.bodySmall,
                ),
                const SizedBox(width: 12),
                Text(
                  '${t.dataExport.history.columns.expires}: '
                  '${_formatRelative(exp.expiresAt)}',
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.outline,
                  ),
                ),
                const Spacer(),
                if (ready)
                  TextButton.icon(
                    onPressed: onDownload,
                    icon: const Icon(Icons.download, size: 16),
                    label: Text(t.dataExport.history.download),
                  ),
                IconButton(
                  tooltip: t.dataExport.history.delete,
                  icon: const Icon(Icons.delete_outline, size: 18),
                  onPressed: onDelete,
                ),
              ],
            ),
            if ((exp.error ?? '').isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: SelectableText(
                  exp.error!,
                  style: TextStyle(
                    color: theme.colorScheme.error,
                    fontSize: 11,
                    fontFamily: 'monospace',
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

String _scopeSummary(ExportScope s) {
  final parts = <String>[];
  if (s.memories) parts.add(t.dataExport.history.scopeMemories);
  if (s.integrations != IntegrationExportMode.none) {
    parts.add(
      t.dataExport.history.scopeIntegrations(mode: s.integrations.wire),
    );
  }
  if (s.customTasks) parts.add(t.dataExport.history.scopeCustomTasks);
  if (parts.isEmpty) return t.dataExport.history.scopeEmpty;
  return parts.join(' · ');
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});
  final String status;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final (color, label) = switch (status) {
      'pending' => (Colors.amber, t.dataExport.status.pending),
      'running' => (Colors.amber, t.dataExport.status.running),
      'ready' => (Colors.green, t.dataExport.status.ready),
      'succeeded' => (Colors.green, t.dataExport.status.succeeded),
      'failed' => (theme.colorScheme.error, t.dataExport.status.failed),
      'expired' => (theme.colorScheme.outline, t.dataExport.status.expired),
      _ => (theme.colorScheme.outline, status),
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withValues(alpha: 0.4), width: 0.5),
      ),
      child: Text(
        label,
        style: TextStyle(
          fontSize: 10,
          color: color,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

// ── import form ─────────────────────────────────────────────────

class _ImportForm extends ConsumerStatefulWidget {
  const _ImportForm({required this.onDone});
  final VoidCallback onDone;

  @override
  ConsumerState<_ImportForm> createState() => _ImportFormState();
}

class _ImportFormState extends ConsumerState<_ImportForm> {
  PlatformFile? _picked;
  bool _memories = true;
  bool _integrations = true;
  bool _customTasks = true;
  bool _busy = false;
  ImportRecord? _last;

  Future<void> _pickBundle() async {
    final result = await FilePicker.platform.pickFiles(
      type: FileType.any,
      withData: false,
    );
    if (result == null || result.files.isEmpty) return;
    final f = result.files.single;
    if (f.path == null || f.path!.isEmpty) return;
    setState(() => _picked = f);
  }

  Future<void> _submit() async {
    final picked = _picked;
    if (picked == null || picked.path == null) {
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(t.dataExport.import.pickFileToast)),
      );
      return;
    }
    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    try {
      final imp = await ref
          .read(backupsApiProvider)
          .createImport(
            bundle: File(picked.path!),
            memories: _memories,
            integrations: _integrations,
            customTasks: _customTasks,
          );
      if (!mounted) return;
      setState(() {
        _last = imp;
        _picked = null;
      });
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            imp.status == 'succeeded'
                ? t.dataExport.import.doneToast
                : t.dataExport.import.finishedWithErrors,
          ),
        ),
      );
      widget.onDone();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.dataExport.import.failedToast(error: e.message)),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.dataExport.import.failedToast(error: e.toString())),
          backgroundColor: Theme.of(context).colorScheme.error,
        ),
      );
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Card(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(t.dataExport.import.intro, style: theme.textTheme.bodySmall),
            const SizedBox(height: 12),
            Text(
              t.dataExport.import.bundleLabel,
              style: theme.textTheme.labelMedium,
            ),
            const SizedBox(height: 4),
            Row(
              children: [
                OutlinedButton.icon(
                  onPressed: _busy ? null : _pickBundle,
                  icon: const Icon(Icons.attach_file, size: 16),
                  label: Text(t.dataExport.import.pickFile),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    _picked == null
                        ? t.dataExport.import.noFile
                        : t.dataExport.import.fileSelected(
                            name: _picked!.name,
                            size: _formatBytes(_picked!.size),
                          ),
                    style: theme.textTheme.bodySmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              ],
            ),
            const SizedBox(height: 12),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              dense: true,
              value: _memories,
              onChanged: _busy ? null : (v) => setState(() => _memories = v),
              title: Text(t.dataExport.import.memoriesLabel),
            ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              dense: true,
              value: _integrations,
              onChanged: _busy
                  ? null
                  : (v) => setState(() => _integrations = v),
              title: Text(t.dataExport.import.integrationsLabel),
            ),
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              dense: true,
              value: _customTasks,
              onChanged: _busy ? null : (v) => setState(() => _customTasks = v),
              title: Text(t.dataExport.import.customTasksLabel),
            ),
            const SizedBox(height: 8),
            Align(
              alignment: Alignment.centerRight,
              child: FilledButton.icon(
                onPressed:
                    _busy ||
                        _picked == null ||
                        (!_memories && !_integrations && !_customTasks)
                    ? null
                    : _submit,
                icon: const Icon(Icons.file_upload_outlined, size: 16),
                label: Text(
                  _busy
                      ? t.dataExport.import.importing
                      : t.dataExport.import.importBundle,
                ),
              ),
            ),
            if (_last != null) ...[
              const SizedBox(height: 12),
              _ImportSummaryCard(imp: _last!),
            ],
          ],
        ),
      ),
    );
  }
}

class _ImportSummaryCard extends StatelessWidget {
  const _ImportSummaryCard({required this.imp});
  final ImportRecord imp;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        border: Border.all(color: theme.dividerColor),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: SelectableText(
                  imp.id,
                  style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
                ),
              ),
              _StatusBadge(status: imp.status),
            ],
          ),
          const SizedBox(height: 6),
          _CountsRow(
            label: t.dataExport.import.summaryCard.memories,
            c: imp.memories,
          ),
          _CountsRow(
            label: t.dataExport.import.summaryCard.integrations,
            c: imp.integrations,
          ),
          _CountsRow(
            label: t.dataExport.import.summaryCard.customTasks,
            c: imp.customTasks,
          ),
          if ((imp.error ?? '').isNotEmpty)
            Padding(
              padding: const EdgeInsets.only(top: 4),
              child: SelectableText(
                imp.error!,
                style: TextStyle(
                  color: theme.colorScheme.error,
                  fontSize: 11,
                  fontFamily: 'monospace',
                ),
              ),
            ),
        ],
      ),
    );
  }
}

class _CountsRow extends StatelessWidget {
  const _CountsRow({required this.label, required this.c});
  final String label;
  final EntityCounts c;

  @override
  Widget build(BuildContext context) {
    if (c.created + c.skipped + c.failed == 0) return const SizedBox.shrink();
    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 2),
      child: Wrap(
        crossAxisAlignment: WrapCrossAlignment.center,
        spacing: 10,
        children: [
          SizedBox(
            width: 96,
            child: Text(label, style: theme.textTheme.bodySmall),
          ),
          Text(
            '${c.created} ${t.dataExport.import.summaryCard.created}',
            style: theme.textTheme.bodySmall,
          ),
          Text(
            '${c.skipped} ${t.dataExport.import.summaryCard.skipped}',
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.outline,
            ),
          ),
          if (c.failed > 0)
            Text(
              '${c.failed} ${t.dataExport.import.summaryCard.failed}',
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.error,
              ),
            ),
        ],
      ),
    );
  }
}

// ── import history ───────────────────────────────────────────────

class _ImportHistory extends StatelessWidget {
  const _ImportHistory({required this.rows, required this.error});
  final List<ImportRecord>? rows;
  final Object? error;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (rows == null && error == null) {
      return Text(
        t.dataExport.imports.loading,
        style: theme.textTheme.bodySmall,
      );
    }
    if (error != null && rows == null) {
      return Text(
        t.dataExport.imports.listFailedToast(
          error: error is ApiException
              ? (error! as ApiException).message
              : error.toString(),
        ),
        style: theme.textTheme.bodySmall?.copyWith(
          color: theme.colorScheme.error,
        ),
      );
    }
    if (rows == null || rows!.isEmpty) {
      return Text(t.dataExport.imports.empty, style: theme.textTheme.bodySmall);
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(t.dataExport.imports.title, style: theme.textTheme.titleSmall),
        const SizedBox(height: 8),
        ...rows!.map((imp) => _ImportRow(imp: imp)),
      ],
    );
  }
}

class _ImportRow extends StatelessWidget {
  const _ImportRow({required this.imp});
  final ImportRecord imp;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final source = imp.sourceFilename ?? t.dataExport.imports.sourceUnknown;
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      child: Padding(
        padding: const EdgeInsets.all(12),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Expanded(
                  child: SelectableText(
                    imp.id,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 11,
                    ),
                  ),
                ),
                _StatusBadge(status: imp.status),
              ],
            ),
            const SizedBox(height: 4),
            Text(
              imp.sourceBytes > 0
                  ? '$source · ${_formatBytes(imp.sourceBytes)}'
                  : source,
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
            const SizedBox(height: 2),
            Text(_importSummary(imp), style: theme.textTheme.bodySmall),
            Text(
              _formatRelative(imp.startedAt),
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
            if ((imp.error ?? '').isNotEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 4),
                child: SelectableText(
                  imp.error!,
                  style: TextStyle(
                    color: theme.colorScheme.error,
                    fontSize: 11,
                    fontFamily: 'monospace',
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

String _importSummary(ImportRecord imp) {
  final parts = <String>[];
  final m = imp.memories;
  final i = imp.integrations;
  final c = imp.customTasks;
  if (m.created > 0 || m.skipped > 0) {
    parts.add('memories: ${m.created}/${m.created + m.skipped}');
  }
  if (i.created > 0 || i.skipped > 0) {
    parts.add('integrations: ${i.created}/${i.created + i.skipped}');
  }
  if (c.created > 0 || c.skipped > 0) {
    parts.add('custom_tasks: ${c.created}/${c.created + c.skipped}');
  }
  return parts.isEmpty ? t.dataExport.imports.noneCounts : parts.join(' · ');
}

// ── helpers ──────────────────────────────────────────────────────

String _formatBytes(int n) {
  if (n <= 0) return '—';
  if (n < 1024) return '$n B';
  if (n < 1024 * 1024) return '${(n / 1024).toStringAsFixed(1)} KiB';
  if (n < 1024 * 1024 * 1024) {
    return '${(n / (1024 * 1024)).toStringAsFixed(1)} MiB';
  }
  return '${(n / (1024 * 1024 * 1024)).toStringAsFixed(2)} GiB';
}

// Relative-time labels match web's behaviour: negative diff (future)
// → "in Nx", positive diff (past) → "Nx ago". Honours the locale of
// the running app via slang interpolation; same time-bucket
// thresholds as web for consistency.
String _formatRelative(DateTime ts) {
  final diffMs = DateTime.now().toUtc().difference(ts.toUtc()).inMilliseconds;
  if (diffMs < 0) {
    final inSec = (-diffMs / 1000).round();
    if (inSec < 60) return t.dataExport.relative.inSeconds(n: inSec.toString());
    if (inSec < 3600) {
      return t.dataExport.relative.inMinutes(
        n: (inSec / 60).round().toString(),
      );
    }
    if (inSec < 86400) {
      return t.dataExport.relative.inHours(
        n: (inSec / 3600).round().toString(),
      );
    }
    return t.dataExport.relative.inDays(n: (inSec / 86400).round().toString());
  }
  final sec = (diffMs / 1000).round();
  if (sec < 60) return t.dataExport.relative.secondsAgo(n: sec.toString());
  if (sec < 3600) {
    return t.dataExport.relative.minutesAgo(n: (sec / 60).round().toString());
  }
  return t.dataExport.relative.hoursAgo(n: (sec / 3600).round().toString());
}
