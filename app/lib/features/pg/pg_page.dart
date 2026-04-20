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
/// Two layouts, chosen responsively:
///
///   * **Phone** (<700 px): a tabbed UX — *Browse* (default) lists
///     tables under chip-based DB/schema pickers; tapping a table
///     pushes a full-screen rows page that renders each row as a
///     key→value card (far more readable than a sideways-scrolling
///     grid on a 400 px screen). *SQL* tab is the opt-in editor for
///     power users.
///   * **Wide** (≥700 px): three-pane layout — sidebar with tables,
///     centre SQL editor, bottom DataTable results. Cmd/Ctrl+Enter
///     runs the editor query.
///
/// Shared contract across both layouts:
///   * DB override threads through every API call as a per-request
///     `database` field — no plugin re-configuration to switch DBs.
///   * Destructive statements (DROP / TRUNCATE / DELETE without
///     WHERE) pop a confirmation before going to the server.
///   * Read-only mode refuses write verbs client-side with a clear
///     message before hitting the network.
class PGPage extends StatefulWidget {
  const PGPage({super.key});

  @override
  State<PGPage> createState() => _PGPageState();
}

class _PGPageState extends State<PGPage>
    with SingleTickerProviderStateMixin {
  ProviderInfo? _plugin;
  StreamSubscription<void>? _providersSub;

  final _sqlCtrl = TextEditingController(text: 'SELECT 1;');
  final _sqlFocus = FocusNode();
  final _tableFilterCtrl = TextEditingController();
  late final TabController _tabCtrl;

  List<String> _databases = [];
  String? _activeDatabase;

  List<String> _schemas = [];
  String? _activeSchema;
  List<Map<String, dynamic>> _tables = [];
  String _tableFilter = '';

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

  List<Map<String, dynamic>> get _filteredTables {
    if (_tableFilter.isEmpty) return _tables;
    final q = _tableFilter.toLowerCase();
    return _tables.where((t) {
      final n = (t['name'] as String? ?? '').toLowerCase();
      return n.contains(q);
    }).toList();
  }

  @override
  void initState() {
    super.initState();
    _tabCtrl = TabController(length: 2, vsync: this);
    _tableFilterCtrl.addListener(() {
      if (_tableFilter != _tableFilterCtrl.text) {
        setState(() => _tableFilter = _tableFilterCtrl.text);
      }
    });
    _loadPlugin();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugin());
  }

  @override
  void dispose() {
    _sqlCtrl.dispose();
    _sqlFocus.dispose();
    _tableFilterCtrl.dispose();
    _tabCtrl.dispose();
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
    return LayoutBuilder(builder: (ctx, constraints) {
      final isPhone = constraints.maxWidth < 700;
      if (isPhone) return _buildPhone();
      return _buildDesktop();
    });
  }

  Widget _buildDesktop() {
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
            Expanded(child: _desktopBody()),
          ]),
        ),
      ),
    );
  }

  // ─── Phone layout ────────────────────────────────────────

  Widget _buildPhone() {
    return Column(children: [
      _phoneHeader(),
      if (_error != null) _errorBar(_error!),
      Container(
        color: AppColors.surface,
        child: TabBar(
          controller: _tabCtrl,
          labelColor: AppColors.accent,
          unselectedLabelColor: AppColors.textMuted,
          indicatorColor: AppColors.accent,
          labelStyle:
              const TextStyle(fontSize: 13, fontWeight: FontWeight.w600),
          tabs: [
            Tab(height: 36, text: context.tr('Browse')),
            Tab(height: 36, text: context.tr('SQL')),
          ],
        ),
      ),
      const Divider(height: 1, color: AppColors.border),
      Expanded(
        child: TabBarView(
          controller: _tabCtrl,
          children: [_phoneBrowseTab(), _phoneSqlTab()],
        ),
      ),
    ]);
  }

  /// Phone header: compact chip row with DB + schema + read-only
  /// badge. Tap a chip to open a bottom-sheet picker — much easier
  /// to hit with a thumb than an inline dropdown button.
  Widget _phoneHeader() {
    return Container(
      padding: const EdgeInsets.fromLTRB(10, 8, 10, 8),
      decoration: const BoxDecoration(
        color: AppColors.surface,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        Expanded(
          child: Wrap(
            spacing: 6,
            runSpacing: 6,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              _chipButton(
                icon: Icons.storage,
                label: _activeDatabase ?? context.tr('Database'),
                onTap: _databases.isEmpty ? null : _showDatabasePicker,
              ),
              _chipButton(
                icon: Icons.folder_outlined,
                label: _activeSchema ?? context.tr('Schema'),
                onTap: _schemas.isEmpty ? null : _showSchemaPicker,
              ),
              if (_readOnly)
                Container(
                  padding: const EdgeInsets.symmetric(
                      horizontal: 8, vertical: 4),
                  decoration: BoxDecoration(
                    color: AppColors.textMuted.withValues(alpha: 0.14),
                    borderRadius: BorderRadius.circular(12),
                  ),
                  child: const Text('read-only',
                      style: TextStyle(
                          color: AppColors.textMuted,
                          fontSize: 10,
                          fontWeight: FontWeight.w600)),
                ),
            ],
          ),
        ),
      ]),
    );
  }

  Widget _chipButton({
    required IconData icon,
    required String label,
    VoidCallback? onTap,
  }) {
    final enabled = onTap != null;
    return Material(
      color: enabled
          ? AppColors.accent.withValues(alpha: 0.12)
          : AppColors.textMuted.withValues(alpha: 0.08),
      borderRadius: BorderRadius.circular(16),
      child: InkWell(
        borderRadius: BorderRadius.circular(16),
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          child: Row(mainAxisSize: MainAxisSize.min, children: [
            Icon(icon,
                size: 14,
                color: enabled ? AppColors.accent : AppColors.textMuted),
            const SizedBox(width: 6),
            ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 140),
              child: Text(label,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      color: enabled ? AppColors.accent : AppColors.textMuted)),
            ),
            const SizedBox(width: 4),
            Icon(Icons.arrow_drop_down,
                size: 16,
                color: enabled ? AppColors.accent : AppColors.textMuted),
          ]),
        ),
      ),
    );
  }

  Widget _phoneBrowseTab() {
    return Column(children: [
      Padding(
        padding: const EdgeInsets.fromLTRB(10, 8, 10, 4),
        child: TextField(
          controller: _tableFilterCtrl,
          decoration: InputDecoration(
            hintText: context.tr('Search tables'),
            prefixIcon: const Icon(Icons.search, size: 18),
            isDense: true,
            filled: true,
            fillColor: AppColors.surfaceAlt,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            border: OutlineInputBorder(
              borderRadius: BorderRadius.circular(8),
              borderSide: BorderSide.none,
            ),
            suffixIcon: _tableFilter.isEmpty
                ? null
                : IconButton(
                    icon: const Icon(Icons.clear, size: 16),
                    onPressed: () => _tableFilterCtrl.clear(),
                  ),
          ),
        ),
      ),
      Expanded(
        child: _filteredTables.isEmpty
            ? Center(
                child: Text(
                  _tables.isEmpty
                      ? context.tr('No tables in this schema')
                      : context.tr('No matches'),
                  style: const TextStyle(
                      color: AppColors.textMuted, fontSize: 13),
                ),
              )
            : ListView.separated(
                itemCount: _filteredTables.length,
                separatorBuilder: (_, _) =>
                    const Divider(height: 1, color: AppColors.border),
                itemBuilder: (_, i) => _phoneTableRow(_filteredTables[i]),
              ),
      ),
    ]);
  }

  Widget _phoneTableRow(Map<String, dynamic> t) {
    final name = t['name'] as String? ?? '';
    final kind = t['kind'] as String? ?? 'table';
    return ListTile(
      dense: true,
      leading: Icon(
        kind == 'view' || kind == 'matview'
            ? Icons.visibility_outlined
            : Icons.table_chart_outlined,
        size: 20,
        color: AppColors.textMuted,
      ),
      title: Text(name,
          style:
              const TextStyle(fontSize: 14, fontWeight: FontWeight.w500)),
      subtitle: kind == 'table'
          ? null
          : Text(kind,
              style:
                  const TextStyle(fontSize: 11, color: AppColors.textMuted)),
      trailing: const Icon(Icons.chevron_right,
          size: 18, color: AppColors.textMuted),
      onTap: () => _pushPhoneRowsPage(t),
    );
  }

  /// Full-screen rows page. Dedicated route so the rows view gets
  /// the entire screen — no toolbar, no tab bar eating pixels on a
  /// 400 px phone.
  void _pushPhoneRowsPage(Map<String, dynamic> t) {
    final pname = _pluginName;
    if (pname == null) return;
    Navigator.of(context).push(MaterialPageRoute(
      builder: (_) => PGTableRowsPage(
        pluginName: pname,
        database: _activeDatabase,
        schema: t['schema'] as String? ?? _activeSchema ?? 'public',
        table: t['name'] as String? ?? '',
        kind: t['kind'] as String? ?? 'table',
        limit: _previewLimit(),
      ),
    ));
  }

  Widget _phoneSqlTab() {
    return Column(children: [
      Padding(
        padding: const EdgeInsets.fromLTRB(10, 8, 10, 4),
        child: Row(children: [
          const Icon(Icons.terminal, size: 16, color: AppColors.textMuted),
          const SizedBox(width: 6),
          Text(context.tr('SQL editor'),
              style: const TextStyle(
                  fontSize: 12,
                  color: AppColors.textMuted,
                  fontWeight: FontWeight.w600)),
          const Spacer(),
          TextButton.icon(
            onPressed: () {
              setState(() {
                _queryResult = null;
                _executeResult = null;
                _error = null;
              });
            },
            style: TextButton.styleFrom(
              padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              minimumSize: const Size(0, 28),
            ),
            icon: const Icon(Icons.clear_all, size: 14),
            label: Text(context.tr('Clear'),
                style: const TextStyle(fontSize: 12)),
          ),
          const SizedBox(width: 4),
          FilledButton.icon(
            onPressed: _running ? null : _run,
            style: FilledButton.styleFrom(
              backgroundColor: AppColors.accent,
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
              minimumSize: const Size(0, 32),
            ),
            icon: _running
                ? const SizedBox(
                    width: 12,
                    height: 12,
                    child: CircularProgressIndicator(
                        strokeWidth: 2, color: Colors.white))
                : const Icon(Icons.play_arrow, size: 14),
            label: Text(context.tr('Run'),
                style: const TextStyle(fontSize: 12)),
          ),
        ]),
      ),
      Padding(
        padding: const EdgeInsets.fromLTRB(10, 0, 10, 8),
        child: SizedBox(
          height: 140,
          child: TextField(
            controller: _sqlCtrl,
            focusNode: _sqlFocus,
            maxLines: null,
            expands: true,
            keyboardType: TextInputType.multiline,
            textAlignVertical: TextAlignVertical.top,
            style: const TextStyle(
                fontFamily: 'monospace', fontSize: 13, height: 1.4),
            decoration: InputDecoration(
              hintText: 'SELECT * FROM users LIMIT 100;',
              filled: true,
              fillColor: AppColors.surfaceAlt,
              contentPadding: const EdgeInsets.all(10),
              border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: const BorderSide(color: AppColors.border),
              ),
            ),
          ),
        ),
      ),
      const Divider(height: 1, color: AppColors.border),
      Expanded(child: _resultsPane(phone: true)),
    ]);
  }

  Future<void> _showDatabasePicker() async {
    final picked = await _pickFromList(
      title: context.tr('Select database'),
      items: _databases,
      selected: _activeDatabase,
      leading: Icons.storage,
    );
    if (picked != null && picked != _activeDatabase) {
      setState(() {
        _activeDatabase = picked;
        _activeSchema = null;
        _schemas = [];
        _tables = [];
      });
      await _loadSchemas();
    }
  }

  Future<void> _showSchemaPicker() async {
    final picked = await _pickFromList(
      title: context.tr('Select schema'),
      items: _schemas,
      selected: _activeSchema,
      leading: Icons.folder_outlined,
    );
    if (picked != null && picked != _activeSchema) {
      setState(() => _activeSchema = picked);
      await _loadTables();
    }
  }

  Future<String?> _pickFromList({
    required String title,
    required List<String> items,
    required String? selected,
    required IconData leading,
  }) async {
    if (items.isEmpty) return null;
    return showModalBottomSheet<String>(
      context: context,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(12)),
      ),
      builder: (ctx) {
        return SafeArea(
          child: Column(mainAxisSize: MainAxisSize.min, children: [
            Container(
              width: 36,
              height: 4,
              margin: const EdgeInsets.only(top: 8, bottom: 8),
              decoration: BoxDecoration(
                color: AppColors.border,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 4, 16, 8),
              child: Align(
                alignment: Alignment.centerLeft,
                child: Text(title,
                    style: const TextStyle(
                        fontSize: 14, fontWeight: FontWeight.w600)),
              ),
            ),
            const Divider(height: 1, color: AppColors.border),
            Flexible(
              child: ListView.builder(
                shrinkWrap: true,
                itemCount: items.length,
                itemBuilder: (_, i) {
                  final v = items[i];
                  final isSelected = v == selected;
                  return ListTile(
                    dense: true,
                    leading: Icon(leading,
                        size: 18,
                        color: isSelected
                            ? AppColors.accent
                            : AppColors.textMuted),
                    title: Text(v,
                        style: TextStyle(
                            fontSize: 14,
                            fontWeight: isSelected
                                ? FontWeight.w600
                                : FontWeight.w400,
                            color: isSelected
                                ? AppColors.accent
                                : AppColors.text)),
                    trailing: isSelected
                        ? const Icon(Icons.check,
                            size: 18, color: AppColors.accent)
                        : null,
                    onTap: () => Navigator.of(ctx).pop(v),
                  );
                },
              ),
            ),
          ]),
        );
      },
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

  Widget _desktopBody() {
    final editorArea = Column(children: [
      Expanded(flex: 2, child: _editor()),
      const Divider(height: 1, color: AppColors.border),
      Expanded(flex: 3, child: _resultsPane()),
    ]);
    return Row(children: [
      SizedBox(width: 240, child: _tablesSidebar()),
      const VerticalDivider(width: 1, color: AppColors.border),
      Expanded(child: editorArea),
    ]);
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

  Widget _resultsPane({bool phone = false}) {
    if (_queryResult != null) {
      return _queryResultView(_queryResult!, phone: phone);
    }
    if (_executeResult != null) return _executeResultView(_executeResult!);
    return Center(
      child: Text(context.tr('Run a query to see results here.'),
          style: const TextStyle(color: AppColors.textMuted, fontSize: 12)),
    );
  }

  Widget _queryResultView(Map<String, dynamic> res, {bool phone = false}) {
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
            : (phone
                ? pgRowCardList(cols: cols, rows: rows)
                : _queryResultTable(cols, rows)),
      ),
    ]);
  }

  Widget _queryResultTable(List cols, List rows) {
    return SingleChildScrollView(
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
                      color:
                          v == null ? AppColors.textMuted : AppColors.text,
                      fontStyle:
                          v == null ? FontStyle.italic : FontStyle.normal,
                    ),
                  )),
              ]),
          ],
        ),
      ),
    );
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

/// Renders query rows as a scrollable list of cards (one card per
/// row, key→value pairs inside). Way more legible on a phone than a
/// sideways-scrolling grid: every column name is a visible label
/// next to its value instead of hiding above a scrolled-away header.
Widget pgRowCardList({required List cols, required List rows}) {
  final colNames = [
    for (final c in cols) (c as Map)['name'] as String? ?? '',
  ];
  final colTypes = [
    for (final c in cols) (c as Map)['type'] as String? ?? '',
  ];

  return ListView.separated(
    padding: const EdgeInsets.fromLTRB(10, 10, 10, 16),
    itemCount: rows.length,
    separatorBuilder: (_, _) => const SizedBox(height: 8),
    itemBuilder: (_, i) {
      final row = (rows[i] as List);
      return Container(
        decoration: BoxDecoration(
          color: AppColors.surface,
          border: Border.all(color: AppColors.border),
          borderRadius: BorderRadius.circular(8),
        ),
        padding: const EdgeInsets.all(10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(children: [
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                decoration: BoxDecoration(
                  color: AppColors.accent.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text('#${i + 1}',
                    style: const TextStyle(
                        color: AppColors.accent,
                        fontSize: 10,
                        fontWeight: FontWeight.w600)),
              ),
            ]),
            const SizedBox(height: 6),
            for (int j = 0; j < colNames.length && j < row.length; j++)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 3),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Row(children: [
                      Text(colNames[j],
                          style: const TextStyle(
                              fontFamily: 'monospace',
                              fontSize: 11,
                              fontWeight: FontWeight.w600,
                              color: AppColors.text)),
                      const SizedBox(width: 6),
                      if (colTypes[j].isNotEmpty)
                        Text(colTypes[j],
                            style: const TextStyle(
                                fontFamily: 'monospace',
                                fontSize: 10,
                                color: AppColors.textMuted)),
                    ]),
                    const SizedBox(height: 2),
                    SelectableText(
                      row[j] == null ? 'NULL' : row[j].toString(),
                      style: TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 12,
                        color: row[j] == null
                            ? AppColors.textMuted
                            : AppColors.text,
                        fontStyle: row[j] == null
                            ? FontStyle.italic
                            : FontStyle.normal,
                      ),
                    ),
                  ],
                ),
              ),
          ],
        ),
      );
    },
  );
}

/// Full-screen rows page pushed from the phone Browse tab. Gets the
/// entire screen to display data — no lateral toolbars eating pixels.
/// Auto-loads on init, supports pull-to-refresh, offers a "copy SELECT
/// to SQL tab" action in the app bar menu so a user can jump to
/// custom querying without retyping the SELECT.
class PGTableRowsPage extends StatefulWidget {
  final String pluginName;
  final String? database;
  final String schema;
  final String table;
  final String kind;
  final int limit;

  const PGTableRowsPage({
    super.key,
    required this.pluginName,
    required this.database,
    required this.schema,
    required this.table,
    required this.kind,
    required this.limit,
  });

  @override
  State<PGTableRowsPage> createState() => _PGTableRowsPageState();
}

class _PGTableRowsPageState extends State<PGTableRowsPage> {
  Map<String, dynamic>? _result;
  bool _loading = false;
  String? _error;

  String get _selectSql =>
      'SELECT * FROM "${widget.schema}"."${widget.table}" LIMIT ${widget.limit};';

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    final api = context.read<ApiClient>();
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final res = await ApiClient.describeErrors(() => api.pgQuery(
          widget.pluginName, _selectSql,
          database: widget.database ?? ''));
      if (!mounted) return;
      setState(() => _result = res);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final cols = (_result?['columns'] as List?) ?? [];
    final rows = (_result?['rows'] as List?) ?? [];
    final rowCount = (_result?['rowCount'] as num?)?.toInt() ?? 0;
    final durationMs = (_result?['durationMs'] as num?)?.toInt() ?? 0;
    final truncated = _result?['truncated'] == true;

    return Scaffold(
      appBar: AppBar(
        backgroundColor: AppColors.surface,
        foregroundColor: AppColors.text,
        elevation: 0,
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(widget.table,
                style: const TextStyle(
                    fontSize: 15,
                    fontFamily: 'monospace',
                    fontWeight: FontWeight.w600)),
            Text(
              '${widget.database ?? ''} · ${widget.schema}'
              '${widget.kind == 'table' ? '' : ' · ${widget.kind}'}',
              style: const TextStyle(
                  fontSize: 11, color: AppColors.textMuted),
            ),
          ],
        ),
        actions: [
          IconButton(
            tooltip: context.tr('Refresh'),
            onPressed: _loading ? null : _load,
            icon: _loading
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2))
                : const Icon(Icons.refresh, size: 20),
          ),
          IconButton(
            tooltip: context.tr('Copy SELECT'),
            onPressed: () async {
              final messenger = ScaffoldMessenger.of(context);
              final msg = context.tr('SELECT copied to clipboard');
              await Clipboard.setData(ClipboardData(text: _selectSql));
              if (!mounted) return;
              messenger.showSnackBar(SnackBar(
                content: Text(msg),
                behavior: SnackBarBehavior.floating,
                duration: const Duration(seconds: 2),
              ));
            },
            icon: const Icon(Icons.copy, size: 18),
          ),
        ],
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(26),
          child: Container(
            width: double.infinity,
            padding:
                const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            color: AppColors.surfaceAlt,
            child: Row(children: [
              Text('$rowCount ${context.tr('rows')}',
                  style: const TextStyle(fontSize: 11)),
              const SizedBox(width: 12),
              Text('${durationMs}ms',
                  style: const TextStyle(
                      fontSize: 11, color: AppColors.textMuted)),
              if (truncated) ...[
                const SizedBox(width: 12),
                Text('⚠️ ${context.tr('truncated')}',
                    style: const TextStyle(
                        fontSize: 11, color: AppColors.warning)),
              ],
            ]),
          ),
        ),
      ),
      body: _error != null
          ? _errorBody(_error!)
          : (_loading && _result == null
              ? const Center(child: CircularProgressIndicator())
              : (rows.isEmpty
                  ? Center(
                      child: Text(context.tr('No rows'),
                          style: const TextStyle(
                              color: AppColors.textMuted, fontSize: 13)),
                    )
                  : RefreshIndicator(
                      onRefresh: _load,
                      child: pgRowCardList(cols: cols, rows: rows),
                    ))),
    );
  }

  Widget _errorBody(String msg) {
    return Padding(
      padding: const EdgeInsets.all(16),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Container(
            padding: const EdgeInsets.all(12),
            decoration: BoxDecoration(
              color: AppColors.errorSoft,
              borderRadius: BorderRadius.circular(8),
              border: Border.all(
                  color: AppColors.error.withValues(alpha: 0.3)),
            ),
            child: Row(crossAxisAlignment: CrossAxisAlignment.start, children: [
              const Icon(Icons.error_outline,
                  color: AppColors.error, size: 18),
              const SizedBox(width: 8),
              Expanded(
                  child: SelectableText(msg,
                      style: const TextStyle(
                          color: AppColors.error, fontSize: 12))),
            ]),
          ),
          const SizedBox(height: 12),
          OutlinedButton.icon(
            onPressed: _load,
            icon: const Icon(Icons.refresh, size: 16),
            label: Text(context.tr('Retry')),
          ),
        ],
      ),
    );
  }
}
