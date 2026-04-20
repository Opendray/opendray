import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// PostgreSQL SQL editor + schema browser for the pg-browser@2.0.0
/// plugin.
///
/// Toolbar exposes a **database** picker (lists every non-template
/// database the configured user can connect to) and a schema picker.
/// The selected database threads through every downstream API call
/// as the `database` override — the server's overrideDatabase helper
/// swaps cfg.Database per-request, so switching DBs is a pure client
/// concern and doesn't require re-configuring the plugin.
///
/// Tapping a table **immediately** runs a preview (SELECT * LIMIT N)
/// against the active schema.database.table and renders rows — the
/// SQL editor is reserved for custom power-user queries, not the
/// required path to viewing data.
///
/// Cmd/Ctrl+Enter runs the editor query. Destructive statements
/// (DROP / TRUNCATE / DELETE without WHERE) pop a confirmation.
class PGPage extends StatefulWidget {
  const PGPage({super.key});

  @override
  State<PGPage> createState() => _PGPageState();
}

class _PGPageState extends State<PGPage> {
  ProviderInfo? _plugin;
  StreamSubscription<void>? _providersSub;

  final _sqlCtrl = TextEditingController(text: 'SELECT 1;');
  final _sqlFocus = FocusNode();

  List<String> _databases = [];
  String? _activeDatabase;

  List<String> _schemas = [];
  String? _activeSchema;
  List<Map<String, dynamic>> _tables = [];

  Map<String, dynamic>? _queryResult;
  Map<String, dynamic>? _executeResult;
  bool _running = false;
  String? _error;

  ApiClient get _api => context.read<ApiClient>();
  String? get _pluginName => _plugin?.provider.name;
  bool get _readOnly {
    final v = _plugin?.config['readOnly'];
    if (v is bool) return v;
    if (v is String) return v != 'false' && v != '0' && v.isNotEmpty;
    return true;
  }

  @override
  void initState() {
    super.initState();
    _loadPlugin();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugin());
  }

  @override
  void dispose() {
    _sqlCtrl.dispose();
    _sqlFocus.dispose();
    _providersSub?.cancel();
    super.dispose();
  }

  // ─── Plugin / schema boot ────────────────────────────────

  Future<void> _loadPlugin() async {
    try {
      final all = await _api.listProviders();
      final match = all
          .where((p) =>
              p.provider.type == 'panel' &&
              p.provider.name == 'pg-browser' &&
              p.enabled)
          .toList();
      if (!mounted) return;
      if (match.isEmpty) {
        setState(() => _plugin = null);
        return;
      }
      setState(() => _plugin = match.first);
      await _loadDatabases();
      await _loadSchemas();
    } catch (_) {}
  }

  /// Fetches the list of user-visible databases on the server. Seeds
  /// `_activeDatabase` from the plugin config's default DB if present,
  /// else from the first DB returned. Empty lists leave `_activeDatabase`
  /// null, which causes API calls to fall through to the server default.
  Future<void> _loadDatabases() async {
    final name = _pluginName;
    if (name == null) return;
    try {
      final dbs = await _api.pgDatabases(name);
      if (!mounted) return;
      final configured = _plugin?.config['database'] as String?;
      setState(() {
        _databases = dbs;
        _activeDatabase ??= (configured != null && dbs.contains(configured))
            ? configured
            : (dbs.isNotEmpty ? dbs.first : null);
      });
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  Future<void> _loadSchemas() async {
    final name = _pluginName;
    if (name == null) return;
    try {
      final schemas =
          await _api.pgSchemas(name, database: _activeDatabase ?? '');
      if (!mounted) return;
      setState(() {
        _schemas = schemas;
        if (_activeSchema == null || !schemas.contains(_activeSchema)) {
          _activeSchema = schemas.contains('public')
              ? 'public'
              : (schemas.isNotEmpty ? schemas.first : null);
        }
      });
      await _loadTables();
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  Future<void> _loadTables() async {
    final name = _pluginName;
    final schema = _activeSchema;
    if (name == null || schema == null) return;
    try {
      final tables = await _api.pgTables(name,
          schema: schema, database: _activeDatabase ?? '');
      if (!mounted) return;
      setState(() => _tables = tables);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    }
  }

  // ─── Run / Execute ───────────────────────────────────────

  /// Returns true when the SQL is a "write verb" per the same rule the
  /// server uses in gateway/pg.IsWriteVerb. Kept in sync by copying
  /// the short set — cross-boundary duplication is cheaper than a
  /// round-trip to pre-classify every query.
  bool _isWriteVerb(String sql) {
    final v = _firstVerb(sql);
    return const {
      'INSERT', 'UPDATE', 'DELETE', 'DROP', 'CREATE', 'ALTER',
      'TRUNCATE', 'GRANT', 'REVOKE', 'REINDEX', 'VACUUM', 'COMMENT',
      'CLUSTER',
    }.contains(v);
  }

  bool _isDestructive(String sql) {
    final v = _firstVerb(sql);
    if (v == 'DROP' || v == 'TRUNCATE') return true;
    if (v == 'DELETE') {
      return !sql.toUpperCase().contains('WHERE');
    }
    return false;
  }

  String _firstVerb(String sql) {
    var s = sql.trim();
    // Strip leading -- line comments, repeatedly.
    while (s.startsWith('--')) {
      final nl = s.indexOf('\n');
      if (nl < 0) return '';
      s = s.substring(nl + 1).trim();
    }
    if (s.startsWith('/*')) {
      final end = s.indexOf('*/');
      if (end < 0) return '';
      s = s.substring(end + 2).trim();
    }
    for (var i = 0; i < s.length; i++) {
      final c = s[i];
      if (c == ' ' || c == '\t' || c == '\n' || c == ';' || c == '(') {
        return s.substring(0, i).toUpperCase();
      }
    }
    return s.toUpperCase();
  }

  Future<void> _run() async {
    final name = _pluginName;
    if (name == null) return;
    final sql = _sqlCtrl.text.trim();
    if (sql.isEmpty) return;

    final write = _isWriteVerb(sql);
    if (write && _readOnly) {
      setState(() => _error =
          'Plugin is in read-only mode. Toggle readOnly in Configure to enable writes.');
      return;
    }
    if (write && _isDestructive(sql)) {
      final confirmed = await _confirmDestructive(_firstVerb(sql));
      if (!confirmed) return;
    }

    setState(() {
      _running = true;
      _error = null;
      _queryResult = null;
      _executeResult = null;
    });
    try {
      if (write) {
        final res = await ApiClient.describeErrors(() => _api.pgExecute(
            name, sql,
            database: _activeDatabase ?? ''));
        if (!mounted) return;
        setState(() => _executeResult = res);
        // If the write was a DDL (CREATE / ALTER / DROP), refresh
        // the sidebar tables list so the user's new table appears
        // without a manual refresh.
        if (_ddlVerbs.contains(_firstVerb(sql))) await _loadTables();
      } else {
        final res = await ApiClient.describeErrors(
            () => _api.pgQuery(name, sql, database: _activeDatabase ?? ''));
        if (!mounted) return;
        setState(() => _queryResult = res);
      }
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _running = false);
    }
  }

  static const _ddlVerbs = {'CREATE', 'ALTER', 'DROP', 'TRUNCATE'};

  Future<bool> _confirmDestructive(String verb) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text('⚠️ $verb detected'),
        content: Text(
          verb == 'DELETE'
              ? 'This DELETE has no WHERE clause and will remove every row in the target table. This cannot be undone.'
              : 'This statement is destructive and cannot be undone without a backup.',
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx, false),
              child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: AppColors.error),
            onPressed: () => Navigator.pop(ctx, true),
            child: const Text('Run anyway'),
          ),
        ],
      ),
    );
    return ok ?? false;
  }

  // ─── UI ──────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    if (_plugin == null) {
      return _centered(
        icon: Icons.extension_off,
        title: context.tr('pg-browser plugin not enabled'),
        subtitle: context.tr(
            'Install pg-browser from the Hub and configure host/user/password in Plugins → Configure.'),
      );
    }
    return Shortcuts(
      shortcuts: {
        LogicalKeySet(LogicalKeyboardKey.control, LogicalKeyboardKey.enter):
            const _RunIntent(),
        LogicalKeySet(LogicalKeyboardKey.meta, LogicalKeyboardKey.enter):
            const _RunIntent(),
      },
      child: Actions(
        actions: {
          _RunIntent: CallbackAction<_RunIntent>(onInvoke: (_) {
            _run();
            return null;
          }),
        },
        child: Focus(
          autofocus: true,
          child: Column(children: [
            _toolbar(),
            if (_error != null) _errorBar(_error!),
            Expanded(child: _bodyLayout()),
          ]),
        ),
      ),
    );
  }

  Widget _toolbar() {
    return Container(
      padding: const EdgeInsets.fromLTRB(10, 6, 10, 6),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        // Database selector — swaps cfg.Database per-request via the
        // gateway's overrideDatabase helper. Changing DB triggers a
        // full schema reload (and tables reload through the chain).
        if (_databases.isNotEmpty) ...[
          const Icon(Icons.storage, size: 14, color: AppColors.textMuted),
          const SizedBox(width: 4),
          DropdownButton<String>(
            value: _activeDatabase,
            items: [
              for (final d in _databases)
                DropdownMenuItem(value: d, child: Text(d)),
            ],
            onChanged: (v) {
              if (v != null && v != _activeDatabase) {
                setState(() {
                  _activeDatabase = v;
                  _activeSchema = null;
                  _schemas = [];
                  _tables = [];
                });
                _loadSchemas();
              }
            },
            underline: const SizedBox.shrink(),
            isDense: true,
          ),
          const SizedBox(width: 10),
        ],
        // Schema selector
        if (_schemas.isNotEmpty) ...[
          const Icon(Icons.folder_outlined,
              size: 14, color: AppColors.textMuted),
          const SizedBox(width: 4),
          DropdownButton<String>(
            value: _activeSchema,
            items: [
              for (final s in _schemas) DropdownMenuItem(value: s, child: Text(s)),
            ],
            onChanged: (v) {
              if (v != null) {
                setState(() => _activeSchema = v);
                _loadTables();
              }
            },
            underline: const SizedBox.shrink(),
            isDense: true,
          ),
          const SizedBox(width: 8),
        ],
        // Read-only badge
        if (_readOnly)
          Container(
            padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
            decoration: BoxDecoration(
              color: AppColors.textMuted.withValues(alpha: 0.14),
              borderRadius: BorderRadius.circular(4),
            ),
            child: const Text('read-only',
                style: TextStyle(
                    color: AppColors.textMuted,
                    fontSize: 10,
                    fontWeight: FontWeight.w600)),
          ),
        const Spacer(),
        TextButton.icon(
          onPressed: () {
            setState(() {
              _queryResult = null;
              _executeResult = null;
              _error = null;
            });
          },
          icon: const Icon(Icons.clear_all, size: 16),
          label: Text(context.tr('Clear')),
        ),
        const SizedBox(width: 6),
        FilledButton.icon(
          onPressed: _running ? null : _run,
          style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
          icon: _running
              ? const SizedBox(
                  width: 14,
                  height: 14,
                  child: CircularProgressIndicator(
                      strokeWidth: 2, color: Colors.white))
              : const Icon(Icons.play_arrow, size: 16),
          label: Text(context.tr('Run')),
        ),
      ]),
    );
  }

  Widget _bodyLayout() {
    return LayoutBuilder(builder: (ctx, constraints) {
      final isWide = constraints.maxWidth > 720;
      final editorArea = Column(children: [
        Expanded(flex: 2, child: _editor()),
        const Divider(height: 1, color: AppColors.border),
        Expanded(flex: 3, child: _resultsPane()),
      ]);
      if (!isWide) {
        return Column(children: [
          _tablesStrip(),
          const Divider(height: 1, color: AppColors.border),
          Expanded(child: editorArea),
        ]);
      }
      return Row(children: [
        SizedBox(width: 240, child: _tablesSidebar()),
        const VerticalDivider(width: 1, color: AppColors.border),
        Expanded(child: editorArea),
      ]);
    });
  }

  Widget _tablesSidebar() {
    return Container(
      color: AppColors.surface,
      child: _tables.isEmpty
          ? Center(
              child: Text(context.tr('No tables'),
                  style:
                      const TextStyle(color: AppColors.textMuted, fontSize: 12)))
          : ListView.builder(
              itemCount: _tables.length,
              itemBuilder: (_, i) => _tableTile(_tables[i]),
            ),
    );
  }

  /// Narrow-screen horizontal strip (one chip per table). Keeps the
  /// UX intelligible on phones where a 240px sidebar would swallow
  /// the editor.
  Widget _tablesStrip() {
    if (_tables.isEmpty) return const SizedBox.shrink();
    return SizedBox(
      height: 36,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        itemCount: _tables.length,
        separatorBuilder: (_, _) => const SizedBox(width: 4),
        itemBuilder: (_, i) {
          final t = _tables[i];
          return ActionChip(
            label: Text(t['name'] as String? ?? '',
                style: const TextStyle(fontSize: 11)),
            onPressed: () => _previewTable(t),
          );
        },
      ),
    );
  }

  Widget _tableTile(Map<String, dynamic> t) {
    final name = t['name'] as String? ?? '';
    final kind = t['kind'] as String? ?? 'table';
    return InkWell(
      onTap: () => _previewTable(t),
      onLongPress: () => _insertTableSelect(t),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
        child: Row(children: [
          Icon(
            kind == 'view' || kind == 'matview'
                ? Icons.visibility_outlined
                : Icons.table_chart_outlined,
            size: 16,
            color: AppColors.textMuted,
          ),
          const SizedBox(width: 8),
          Expanded(
              child: Text(name,
                  overflow: TextOverflow.ellipsis,
                  style: const TextStyle(fontSize: 12))),
          if (kind != 'table')
            Text(kind,
                style: const TextStyle(color: AppColors.textMuted, fontSize: 10)),
        ]),
      ),
    );
  }

  /// Fires a SELECT * LIMIT N against the tapped table and renders
  /// the rows in the results pane — no manual Run step. This is the
  /// primary way to view data; the SQL editor is for custom queries.
  /// The SELECT is also written into the editor so the user can
  /// tweak-and-re-run without retyping.
  Future<void> _previewTable(Map<String, dynamic> t) async {
    final pname = _pluginName;
    if (pname == null) return;
    final schema = t['schema'] as String? ?? 'public';
    final tname = t['name'] as String? ?? '';
    if (tname.isEmpty) return;

    final limit = _previewLimit();
    final sql = 'SELECT * FROM "$schema"."$tname" LIMIT $limit;';
    _sqlCtrl.text = sql;
    _sqlCtrl.selection = TextSelection.collapsed(offset: sql.length);

    setState(() {
      _running = true;
      _error = null;
      _queryResult = null;
      _executeResult = null;
    });
    try {
      final res = await ApiClient.describeErrors(
          () => _api.pgQuery(pname, sql, database: _activeDatabase ?? ''));
      if (!mounted) return;
      setState(() => _queryResult = res);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _running = false);
    }
  }

  /// Reads the configured maxRows cap (server enforces too). Falls
  /// back to 100 which matches the plugin's configSchema default.
  int _previewLimit() {
    final v = _plugin?.config['maxRows'];
    if (v is int) return v;
    if (v is num) return v.toInt();
    if (v is String) {
      final parsed = int.tryParse(v);
      if (parsed != null) return parsed;
    }
    return 100;
  }

  /// Long-press fallback: drop the SELECT into the editor without
  /// executing, for users who want to tweak the query before running.
  void _insertTableSelect(Map<String, dynamic> t) {
    final schema = t['schema'] as String? ?? 'public';
    final name = t['name'] as String? ?? '';
    if (name.isEmpty) return;
    final snippet = 'SELECT * FROM "$schema"."$name" LIMIT ${_previewLimit()};';
    _sqlCtrl.text = snippet;
    _sqlCtrl.selection = TextSelection.collapsed(offset: snippet.length);
    _sqlFocus.requestFocus();
  }

  Widget _editor() {
    return Container(
      color: AppColors.surface,
      padding: const EdgeInsets.all(8),
      child: TextField(
        controller: _sqlCtrl,
        focusNode: _sqlFocus,
        maxLines: null,
        expands: true,
        keyboardType: TextInputType.multiline,
        textAlignVertical: TextAlignVertical.top,
        style: const TextStyle(fontFamily: 'monospace', fontSize: 12, height: 1.4),
        decoration: InputDecoration(
          hintText: 'SELECT * FROM users LIMIT 100;',
          border: const OutlineInputBorder(),
          filled: true,
          fillColor: AppColors.surfaceAlt,
          contentPadding: const EdgeInsets.all(10),
        ),
      ),
    );
  }

  Widget _resultsPane() {
    if (_queryResult != null) return _queryResultView(_queryResult!);
    if (_executeResult != null) return _executeResultView(_executeResult!);
    return Center(
      child: Text(context.tr('Run a query to see results here.'),
          style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
    );
  }

  Widget _queryResultView(Map<String, dynamic> res) {
    final cols = (res['columns'] as List?) ?? [];
    final rows = (res['rows'] as List?) ?? [];
    final rowCount = (res['rowCount'] as num?)?.toInt() ?? 0;
    final durationMs = (res['durationMs'] as num?)?.toInt() ?? 0;
    final truncated = res['truncated'] == true;

    return Column(children: [
      _metaBar(children: [
        Text('$rowCount ${context.tr('rows')}',
            style: const TextStyle(fontSize: 11)),
        Text('${durationMs}ms',
            style: const TextStyle(
                fontSize: 11, color: AppColors.textMuted)),
        if (truncated)
          Text(
              '⚠️ ${context.tr('truncated')}',
              style: const TextStyle(fontSize: 11, color: AppColors.warning)),
      ]),
      Expanded(
        child: rows.isEmpty
            ? Center(
                child: Text(context.tr('No rows'),
                    style: const TextStyle(
                        color: AppColors.textMuted, fontSize: 12)),
              )
            : SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: SingleChildScrollView(
                  scrollDirection: Axis.vertical,
                  child: DataTable(
                    columnSpacing: 20,
                    headingRowHeight: 32,
                    dataRowMinHeight: 28,
                    dataRowMaxHeight: 44,
                    columns: [
                      for (final c in cols)
                        DataColumn(
                          label: Tooltip(
                            message: (c as Map)['type'] as String? ?? '',
                            child: Text(c['name'] as String? ?? '',
                                style: const TextStyle(
                                    fontFamily: 'monospace',
                                    fontWeight: FontWeight.w600,
                                    fontSize: 11)),
                          ),
                        ),
                    ],
                    rows: [
                      for (final row in rows)
                        DataRow(cells: [
                          for (final v in (row as List))
                            DataCell(SelectableText(
                              v == null ? 'NULL' : v.toString(),
                              style: TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 11,
                                color: v == null
                                    ? AppColors.textMuted
                                    : AppColors.text,
                                fontStyle: v == null
                                    ? FontStyle.italic
                                    : FontStyle.normal,
                              ),
                            )),
                        ]),
                    ],
                  ),
                ),
              ),
      ),
    ]);
  }

  Widget _executeResultView(Map<String, dynamic> res) {
    final rowsAffected = (res['rowsAffected'] as num?)?.toInt() ?? 0;
    final verb = res['verb'] as String? ?? '';
    final durationMs = (res['durationMs'] as num?)?.toInt() ?? 0;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.check_circle_outline,
              size: 42, color: AppColors.success),
          const SizedBox(height: 10),
          Text('${context.tr('Executed')}: $verb',
              style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 15)),
          const SizedBox(height: 6),
          Text('$rowsAffected ${context.tr('rows affected')} · ${durationMs}ms',
              style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
        ]),
      ),
    );
  }

  Widget _metaBar({required List<Widget> children}) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
      color: AppColors.surface,
      child: Row(children: [
        for (int i = 0; i < children.length; i++) ...[
          children[i],
          if (i < children.length - 1) const SizedBox(width: 14),
        ],
      ]),
    );
  }

  Widget _errorBar(String msg) {
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      color: AppColors.errorSoft,
      child: Row(children: [
        const Icon(Icons.error_outline, color: AppColors.error, size: 16),
        const SizedBox(width: 8),
        Expanded(
            child: SelectableText(msg,
                style:
                    const TextStyle(color: AppColors.error, fontSize: 12))),
      ]),
    );
  }

  Widget _centered({
    required IconData icon,
    required String title,
    String? subtitle,
  }) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(28),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Icon(icon, size: 44, color: AppColors.textMuted),
          const SizedBox(height: 12),
          Text(title,
              textAlign: TextAlign.center,
              style:
                  const TextStyle(fontWeight: FontWeight.w500, fontSize: 15)),
          if (subtitle != null) ...[
            const SizedBox(height: 8),
            Text(subtitle,
                textAlign: TextAlign.center,
                style:
                    const TextStyle(color: AppColors.textMuted, fontSize: 12)),
          ],
        ]),
      ),
    );
  }
}

class _RunIntent extends Intent {
  const _RunIntent();
}
