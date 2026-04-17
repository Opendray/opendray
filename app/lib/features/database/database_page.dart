import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

class DatabasePage extends StatefulWidget {
  const DatabasePage({super.key});
  @override
  State<DatabasePage> createState() => _DatabasePageState();
}

class _DatabasePageState extends State<DatabasePage> {
  List<ProviderInfo> _dbPlugins = [];
  String? _activePlugin;
  List<Map<String, dynamic>> _databases = [];
  String? _activeDatabase;
  List<Map<String, dynamic>> _schemas = [];
  List<Map<String, dynamic>> _tables = [];
  String _activeSchema = 'public';
  String? _selectedTable;
  List<Map<String, dynamic>> _columns = [];
  Map<String, dynamic>? _queryResult;
  bool _loading = false;
  String? _error;

  final _sqlController = TextEditingController();
  bool _runningQuery = false;
  bool _sqlExpanded = false;
  _ResultView _resultView = _ResultView.cards;
  int? _expandedRowIndex;
  StreamSubscription<void>? _providersSub;

  static const _narrowBreakpoint = 700.0;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _loadPlugins();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugins());
  }

  @override
  void dispose() {
    _sqlController.dispose();
    _providersSub?.cancel();
    super.dispose();
  }

  Future<void> _loadPlugins() async {
    try {
      final providers = await _api.listProviders();
      final dbs = providers.where((p) =>
        p.provider.type == 'panel' && p.provider.category == 'database' && p.enabled).toList();
      if (!mounted) return;
      final stillActive = _activePlugin != null &&
          dbs.any((p) => p.provider.name == _activePlugin);
      setState(() {
        _dbPlugins = dbs;
        if (!stillActive) {
          _activePlugin = null;
          _databases = [];
          _activeDatabase = null;
          _schemas = [];
          _tables = [];
          _columns = [];
          _selectedTable = null;
          _queryResult = null;
        }
      });
      if (dbs.isNotEmpty && _activePlugin == null) {
        _activePlugin = dbs.first.provider.name;
        _loadDatabases();
      }
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    }
  }

  Future<void> _loadDatabases() async {
    if (_activePlugin == null) return;
    setState(() { _loading = true; _error = null; });
    try {
      final dbs = await _api.dbDatabases(_activePlugin!);
      setState(() {
        _databases = dbs;
        if (dbs.isNotEmpty && _activeDatabase == null) {
          _activeDatabase = dbs.first['name'] as String;
        }
      });
      await _loadSchemas();
    } catch (e) {
      setState(() { _error = _extractError(e); _loading = false; });
    }
  }

  Future<void> _loadSchemas() async {
    if (_activePlugin == null) return;
    setState(() { _loading = true; _error = null; });
    try {
      final schemas = await _api.dbSchemas(_activePlugin!, db: _activeDatabase);
      setState(() {
        _schemas = schemas;
        if (schemas.isNotEmpty && !schemas.any((s) => s['name'] == _activeSchema)) {
          _activeSchema = schemas.first['name'] as String;
        }
      });
      await _loadTables();
    } catch (e) {
      setState(() { _error = _extractError(e); _loading = false; });
    }
  }

  Future<void> _loadTables() async {
    if (_activePlugin == null) return;
    setState(() { _loading = true; _error = null; });
    try {
      final tables = await _api.dbTables(_activePlugin!, schema: _activeSchema, db: _activeDatabase);
      setState(() { _tables = tables; _loading = false; });
    } catch (e) {
      setState(() { _error = _extractError(e); _loading = false; });
    }
  }

  Future<void> _previewTable(Map<String, dynamic> table) async {
    if (_activePlugin == null) return;
    final schema = table['schema'] as String;
    final name = table['name'] as String;
    setState(() {
      _loading = true;
      _error = null;
      _selectedTable = name;
      _queryResult = null;
      _expandedRowIndex = null;
    });
    Navigator.maybeOf(context)?.maybePop();
    try {
      final columns = await _api.dbColumns(_activePlugin!, schema, name, db: _activeDatabase);
      final result = await _api.dbPreview(_activePlugin!, schema, name, limit: 100, db: _activeDatabase);
      setState(() { _columns = columns; _queryResult = result; _loading = false; });
    } catch (e) {
      setState(() { _error = _extractError(e); _loading = false; });
    }
  }

  Future<void> _runQuery() async {
    if (_activePlugin == null) return;
    final sql = _sqlController.text.trim();
    if (sql.isEmpty) return;
    setState(() {
      _runningQuery = true;
      _error = null;
      _selectedTable = null;
      _columns = [];
      _expandedRowIndex = null;
    });
    try {
      final result = await _api.dbQuery(_activePlugin!, sql, db: _activeDatabase);
      setState(() { _queryResult = result; _runningQuery = false; });
    } catch (e) {
      setState(() { _error = _extractError(e); _runningQuery = false; });
    }
  }

  String _extractError(Object e) {
    final s = e.toString();
    final idx = s.indexOf('"error":"');
    if (idx >= 0) {
      final end = s.indexOf('"', idx + 9);
      if (end > idx) return s.substring(idx + 9, end);
    }
    return s;
  }

  @override
  Widget build(BuildContext context) {
    if (_dbPlugins.isEmpty) return _buildNoPlugins();
    return LayoutBuilder(builder: (ctx, constraints) {
      final narrow = constraints.maxWidth < _narrowBreakpoint;
      return narrow ? _buildNarrowLayout() : _buildWideLayout();
    });
  }

  // ── Layouts ───────────────────────────────────────────────

  Widget _buildNarrowLayout() {
    return Scaffold(
      backgroundColor: Colors.transparent,
      drawer: Drawer(
        backgroundColor: AppColors.surface,
        width: 280,
        child: SafeArea(
          child: Column(
            children: [
              _buildDatabaseBar(),
              const Divider(height: 1, color: AppColors.border),
              _buildSchemaBar(),
              const Divider(height: 1, color: AppColors.border),
              Expanded(child: _buildTableList()),
            ],
          ),
        ),
      ),
      body: Column(
        children: [
          if (_dbPlugins.length > 1) _buildPluginSelector(),
          _buildMobileToolbar(),
          if (_error != null) _buildError(),
          _buildSqlInput(compact: true),
          Expanded(child: _buildResultArea()),
        ],
      ),
    );
  }

  Widget _buildWideLayout() {
    return Column(
      children: [
        if (_dbPlugins.length > 1) _buildPluginSelector(),
        _buildDatabaseBar(),
        _buildSchemaBar(),
        if (_error != null) _buildError(),
        Expanded(
          child: Row(
            children: [
              SizedBox(width: 240, child: _buildTableList()),
              const VerticalDivider(width: 1, color: AppColors.border),
              Expanded(child: Column(
                children: [
                  _buildSqlInput(compact: false),
                  Expanded(child: _buildResultArea()),
                ],
              )),
            ],
          ),
        ),
      ],
    );
  }

  // ── Widgets ───────────────────────────────────────────────

  Widget _buildNoPlugins() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.storage_outlined, size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(context.tr('No PostgreSQL browser configured'),
              style: const TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 8),
          Text(
            _error ??
                context.tr('Enable PostgreSQL Browser in Settings → Plugins and configure the connection.'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
            textAlign: TextAlign.center,
          ),
        ]),
      ),
    );
  }

  Widget _buildPluginSelector() {
    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
      child: ListView(
        scrollDirection: Axis.horizontal,
        children: _dbPlugins.map((p) => Padding(
          padding: const EdgeInsets.only(right: 8),
          child: ChoiceChip(
            label: Text('${p.provider.icon} ${p.provider.displayName}'),
            selected: _activePlugin == p.provider.name,
            onSelected: (_) {
              setState(() {
                _activePlugin = p.provider.name;
                _activeDatabase = null;
                _databases = [];
                _selectedTable = null;
                _queryResult = null;
                _schemas = [];
                _tables = [];
              });
              _loadDatabases();
            },
            selectedColor: AppColors.accentSoft,
            backgroundColor: AppColors.surfaceAlt,
            labelStyle: TextStyle(fontSize: 12, color: _activePlugin == p.provider.name ? AppColors.accent : AppColors.textMuted),
            side: BorderSide.none,
          ),
        )).toList(),
      ),
    );
  }

  Widget _buildMobileToolbar() {
    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 8),
      decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Row(
        children: [
          Builder(builder: (ctx) => IconButton(
            icon: const Icon(Icons.menu, size: 20),
            tooltip: 'Tables',
            onPressed: () => Scaffold.of(ctx).openDrawer(),
            visualDensity: VisualDensity.compact,
          )),
          Expanded(
            child: _selectedTable != null
              ? Text(
                  '${_activeDatabase ?? ''}.$_activeSchema.$_selectedTable',
                  style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w500, fontFamily: 'JetBrains Mono'),
                  overflow: TextOverflow.ellipsis,
                )
              : Text('${_activeDatabase ?? '—'} · $_activeSchema',
                  style: const TextStyle(fontSize: 11, color: AppColors.textMuted, fontFamily: 'JetBrains Mono'),
                  overflow: TextOverflow.ellipsis),
          ),
          if (_queryResult != null)
            Text(
              '${_queryResult!['rowCount']}r · ${_queryResult!['duration']}',
              style: const TextStyle(fontSize: 10, color: AppColors.textMuted),
            ),
          IconButton(
            icon: Icon(_resultView == _ResultView.cards ? Icons.view_agenda : Icons.table_rows, size: 18),
            tooltip: _resultView == _ResultView.cards ? 'Switch to table view' : 'Switch to card view',
            onPressed: () => setState(() =>
              _resultView = _resultView == _ResultView.cards ? _ResultView.table : _ResultView.cards),
            visualDensity: VisualDensity.compact,
          ),
        ],
      ),
    );
  }

  Widget _buildDatabaseBar() {
    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Row(
        children: [
          const Icon(Icons.storage, size: 14, color: AppColors.accent),
          const SizedBox(width: 6),
          const Text('DB:', style: TextStyle(fontSize: 12, color: AppColors.textMuted)),
          const SizedBox(width: 8),
          Expanded(
            child: _databases.isEmpty
              ? const Text('—', style: TextStyle(fontSize: 12, color: AppColors.textMuted))
              : DropdownButton<String>(
                  value: _activeDatabase,
                  isDense: true,
                  isExpanded: true,
                  underline: const SizedBox.shrink(),
                  style: const TextStyle(fontSize: 12, color: AppColors.text, fontFamily: 'JetBrains Mono'),
                  items: _databases.map((d) {
                    final name = d['name'] as String;
                    final size = d['size'] as String? ?? '';
                    return DropdownMenuItem(
                      value: name,
                      child: Row(children: [
                        Expanded(child: Text(name, overflow: TextOverflow.ellipsis)),
                        if (size.isNotEmpty)
                          Text(size, style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
                      ]),
                    );
                  }).toList(),
                  onChanged: (v) {
                    if (v == null || v == _activeDatabase) return;
                    setState(() {
                      _activeDatabase = v;
                      _selectedTable = null;
                      _queryResult = null;
                      _columns = [];
                      _schemas = [];
                      _tables = [];
                      _activeSchema = 'public';
                    });
                    _loadSchemas();
                  },
                ),
          ),
          Text('${_databases.length}', style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
          IconButton(
            icon: const Icon(Icons.refresh, size: 16),
            onPressed: _loadDatabases,
            tooltip: 'Reload databases',
            visualDensity: VisualDensity.compact,
          ),
        ],
      ),
    );
  }

  Widget _buildSchemaBar() {
    return Container(
      height: 44,
      padding: const EdgeInsets.symmetric(horizontal: 12),
      decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Row(
        children: [
          const Icon(Icons.schema, size: 14, color: AppColors.textMuted),
          const SizedBox(width: 6),
          const Text('Schema:', style: TextStyle(fontSize: 12, color: AppColors.textMuted)),
          const SizedBox(width: 8),
          Expanded(
            child: _schemas.isEmpty
              ? const Text('—', style: TextStyle(fontSize: 12, color: AppColors.textMuted))
              : DropdownButton<String>(
                  value: _activeSchema,
                  isDense: true,
                  isExpanded: true,
                  underline: const SizedBox.shrink(),
                  style: const TextStyle(fontSize: 12, color: AppColors.text),
                  items: _schemas.map((s) => DropdownMenuItem(
                    value: s['name'] as String,
                    child: Text(s['name'] as String, overflow: TextOverflow.ellipsis),
                  )).toList(),
                  onChanged: (v) {
                    if (v == null) return;
                    setState(() { _activeSchema = v; _selectedTable = null; _queryResult = null; });
                    _loadTables();
                  },
                ),
          ),
          IconButton(
            icon: const Icon(Icons.refresh, size: 16),
            onPressed: _loadTables,
            tooltip: 'Reload tables',
            visualDensity: VisualDensity.compact,
          ),
        ],
      ),
    );
  }

  Widget _buildError() {
    return Container(
      margin: const EdgeInsets.all(8),
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(color: AppColors.errorSoft, borderRadius: BorderRadius.circular(6)),
      child: Row(
        children: [
          const Icon(Icons.error_outline, size: 14, color: AppColors.error),
          const SizedBox(width: 8),
          Expanded(child: Text(_error!, style: const TextStyle(fontSize: 11, color: AppColors.error, fontFamily: 'monospace'))),
          IconButton(
            icon: const Icon(Icons.close, size: 14, color: AppColors.error),
            onPressed: () => setState(() => _error = null),
            visualDensity: VisualDensity.compact,
          ),
        ],
      ),
    );
  }

  Widget _buildTableList() {
    if (_loading && _tables.isEmpty) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_tables.isEmpty) {
      return const Center(child: Text('No tables', style: TextStyle(color: AppColors.textMuted, fontSize: 12)));
    }
    return ListView.separated(
      padding: const EdgeInsets.symmetric(vertical: 4),
      itemCount: _tables.length,
      separatorBuilder: (_, _) => const Divider(height: 1, color: AppColors.border),
      itemBuilder: (_, i) {
        final t = _tables[i];
        final selected = _selectedTable == t['name'];
        return ListTile(
          dense: true,
          selected: selected,
          selectedTileColor: AppColors.accentSoft,
          leading: Icon(_tableIcon(t['kind'] as String? ?? 'table'), size: 14, color: selected ? AppColors.accent : AppColors.textMuted),
          title: Text(t['name'] as String, style: TextStyle(fontSize: 12, color: selected ? AppColors.accent : AppColors.text), overflow: TextOverflow.ellipsis),
          subtitle: Text('${t['rows'] ?? 0} rows · ${t['size'] ?? ''}', style: const TextStyle(fontSize: 10, color: AppColors.textMuted)),
          onTap: () => _previewTable(t),
        );
      },
    );
  }

  IconData _tableIcon(String kind) => switch (kind) {
    'view' => Icons.visibility,
    'matview' => Icons.cached,
    _ => Icons.table_chart,
  };

  Widget _buildSqlInput({required bool compact}) {
    final expanded = !compact || _sqlExpanded;
    return Container(
      padding: const EdgeInsets.all(8),
      decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Expanded(
                child: TextField(
                  controller: _sqlController,
                  maxLines: expanded ? 4 : 1,
                  minLines: expanded ? 2 : 1,
                  style: const TextStyle(fontSize: 12, fontFamily: 'JetBrains Mono'),
                  decoration: InputDecoration(
                    hintText: expanded
                      ? 'SELECT * FROM table LIMIT 10   — only SELECT / SHOW / EXPLAIN allowed'
                      : 'SELECT ...',
                    hintStyle: const TextStyle(fontSize: 11, color: AppColors.textMuted),
                    contentPadding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
                    isDense: true,
                    border: OutlineInputBorder(borderRadius: BorderRadius.circular(6), borderSide: const BorderSide(color: AppColors.border)),
                  ),
                  onTap: () { if (compact && !_sqlExpanded) setState(() => _sqlExpanded = true); },
                ),
              ),
              if (compact) ...[
                const SizedBox(width: 4),
                IconButton(
                  icon: Icon(_sqlExpanded ? Icons.unfold_less : Icons.unfold_more, size: 18),
                  onPressed: () => setState(() => _sqlExpanded = !_sqlExpanded),
                  visualDensity: VisualDensity.compact,
                  tooltip: _sqlExpanded ? 'Collapse' : 'Expand',
                ),
              ],
            ],
          ),
          if (expanded) ...[
            const SizedBox(height: 6),
            Row(
              children: [
                FilledButton.icon(
                  onPressed: _runningQuery ? null : _runQuery,
                  icon: _runningQuery
                    ? const SizedBox(width: 12, height: 12, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                    : const Icon(Icons.play_arrow, size: 14),
                  label: const Text('Run', style: TextStyle(fontSize: 12)),
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.accent,
                    padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 6),
                    minimumSize: const Size(0, 32),
                  ),
                ),
                const SizedBox(width: 8),
                TextButton(
                  onPressed: () => _sqlController.clear(),
                  style: TextButton.styleFrom(minimumSize: const Size(0, 32), padding: const EdgeInsets.symmetric(horizontal: 10)),
                  child: const Text('Clear', style: TextStyle(fontSize: 12)),
                ),
                const Spacer(),
                if (!compact && _queryResult != null)
                  Text(
                    '${_queryResult!['rowCount']} rows · ${_queryResult!['duration']}${_queryResult!['truncated'] == true ? ' · truncated' : ''}',
                    style: const TextStyle(fontSize: 11, color: AppColors.textMuted),
                  ),
              ],
            ),
          ] else ...[
            // Compact: inline Run button
            const SizedBox(height: 6),
            SizedBox(
              width: double.infinity,
              child: FilledButton.icon(
                onPressed: _runningQuery ? null : _runQuery,
                icon: _runningQuery
                  ? const SizedBox(width: 12, height: 12, child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white))
                  : const Icon(Icons.play_arrow, size: 14),
                label: const Text('Run', style: TextStyle(fontSize: 12)),
                style: FilledButton.styleFrom(
                  backgroundColor: AppColors.accent,
                  padding: const EdgeInsets.symmetric(vertical: 6),
                  minimumSize: const Size(0, 32),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  Widget _buildResultArea() {
    if (_loading || _runningQuery) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_queryResult == null) {
      if (_selectedTable != null && _columns.isNotEmpty) {
        return _buildColumnsView();
      }
      return const Center(child: Text('Select a table or run a query', style: TextStyle(color: AppColors.textMuted, fontSize: 12)));
    }
    return _resultView == _ResultView.cards ? _buildRowsCards() : _buildRowsTable();
  }

  Widget _buildColumnsView() {
    return Column(
      children: [
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 6),
          decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
          child: Row(children: [
            const Icon(Icons.table_chart, size: 13, color: AppColors.accent),
            const SizedBox(width: 6),
            Expanded(child: Text('$_activeSchema.$_selectedTable',
              style: const TextStyle(fontSize: 12, fontWeight: FontWeight.w500, fontFamily: 'JetBrains Mono'),
              overflow: TextOverflow.ellipsis)),
            Text('${_columns.length} cols', style: const TextStyle(fontSize: 11, color: AppColors.textMuted)),
          ]),
        ),
        Expanded(
          child: ListView.separated(
            padding: const EdgeInsets.all(8),
            itemCount: _columns.length,
            separatorBuilder: (_, _) => const Divider(height: 1, color: AppColors.border),
            itemBuilder: (_, i) {
              final c = _columns[i];
              return ListTile(
                dense: true,
                leading: c['isPk'] == true
                  ? const Icon(Icons.key, size: 14, color: AppColors.warning)
                  : const Icon(Icons.circle, size: 8, color: AppColors.textMuted),
                title: Text(c['name'] as String, style: const TextStyle(fontSize: 12, fontFamily: 'JetBrains Mono')),
                subtitle: Text(
                  '${c['type']}${c['nullable'] == true ? ' NULL' : ' NOT NULL'}${c['default'] != null && c['default'] != '' ? ' DEFAULT ${c['default']}' : ''}',
                  style: const TextStyle(fontSize: 10, color: AppColors.textMuted, fontFamily: 'JetBrains Mono'),
                ),
              );
            },
          ),
        ),
      ],
    );
  }

  // Row list: one card per row, columns stacked as key/value pairs.
  // Best for narrow screens where horizontal scrolling is awkward.
  Widget _buildRowsCards() {
    final cols = (_queryResult!['columns'] as List?)?.cast<String>() ?? [];
    final types = (_queryResult!['types'] as List?)?.cast<String>() ?? [];
    final rows = (_queryResult!['rows'] as List?) ?? [];

    if (rows.isEmpty) {
      return const Center(child: Text('No rows', style: TextStyle(color: AppColors.textMuted, fontSize: 12)));
    }

    return ListView.separated(
      padding: const EdgeInsets.all(8),
      itemCount: rows.length,
      separatorBuilder: (_, _) => const SizedBox(height: 6),
      itemBuilder: (_, rowIdx) {
        final cells = rows[rowIdx] as List;
        final expanded = _expandedRowIndex == rowIdx;
        final preview = cells.isNotEmpty ? _formatValue(cells.first) : '';
        return Card(
          margin: EdgeInsets.zero,
          child: InkWell(
            onTap: () => setState(() => _expandedRowIndex = expanded ? null : rowIdx),
            child: Padding(
              padding: const EdgeInsets.all(10),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Row(
                    children: [
                      Text('#${rowIdx + 1}', style: const TextStyle(fontSize: 10, color: AppColors.textMuted, fontFamily: 'JetBrains Mono')),
                      const SizedBox(width: 8),
                      if (cols.isNotEmpty)
                        Expanded(
                          child: Text(
                            '${cols.first}: $preview',
                            style: const TextStyle(fontSize: 12, fontFamily: 'JetBrains Mono'),
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                      Icon(expanded ? Icons.expand_less : Icons.expand_more, size: 16, color: AppColors.textMuted),
                    ],
                  ),
                  if (expanded) ...[
                    const SizedBox(height: 8),
                    const Divider(height: 1, color: AppColors.border),
                    const SizedBox(height: 8),
                    for (int i = 0; i < cols.length; i++)
                      _buildKvRow(cols[i], i < types.length ? types[i] : '', i < cells.length ? cells[i] : null),
                  ],
                ],
              ),
            ),
          ),
        );
      },
    );
  }

  Widget _buildKvRow(String col, String type, dynamic value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 3),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Row(
            children: [
              Expanded(
                child: Text(col,
                  style: const TextStyle(fontSize: 11, fontWeight: FontWeight.w600, color: AppColors.accent, fontFamily: 'JetBrains Mono'),
                  overflow: TextOverflow.ellipsis),
              ),
              if (type.isNotEmpty)
                Text(type, style: const TextStyle(fontSize: 9, color: AppColors.textMuted, fontFamily: 'JetBrains Mono')),
              const SizedBox(width: 4),
              GestureDetector(
                onTap: () => _copyCell(value),
                child: const Icon(Icons.copy, size: 12, color: AppColors.textMuted),
              ),
            ],
          ),
          const SizedBox(height: 2),
          SelectableText(
            value == null ? 'NULL' : _formatValue(value),
            style: TextStyle(
              fontSize: 12,
              fontFamily: 'JetBrains Mono',
              color: value == null ? AppColors.textMuted : AppColors.text,
              fontStyle: value == null ? FontStyle.italic : FontStyle.normal,
            ),
          ),
        ],
      ),
    );
  }

  String _formatValue(dynamic v) {
    if (v == null) return 'NULL';
    final s = v.toString();
    return s.length > 500 ? '${s.substring(0, 500)}…' : s;
  }

  // Traditional wide DataTable — kept for desktop and users who toggle to it.
  Widget _buildRowsTable() {
    final cols = (_queryResult!['columns'] as List?)?.cast<String>() ?? [];
    final types = (_queryResult!['types'] as List?)?.cast<String>() ?? [];
    final rows = (_queryResult!['rows'] as List?) ?? [];

    if (rows.isEmpty) {
      return const Center(child: Text('No rows', style: TextStyle(color: AppColors.textMuted, fontSize: 12)));
    }

    return Scrollbar(
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: SingleChildScrollView(
          child: DataTable(
            headingRowHeight: 32,
            dataRowMinHeight: 28,
            dataRowMaxHeight: 36,
            columnSpacing: 20,
            horizontalMargin: 12,
            headingTextStyle: const TextStyle(fontSize: 11, fontWeight: FontWeight.w600, fontFamily: 'JetBrains Mono'),
            dataTextStyle: const TextStyle(fontSize: 11, fontFamily: 'JetBrains Mono'),
            columns: [
              for (int i = 0; i < cols.length; i++)
                DataColumn(label: Tooltip(
                  message: i < types.length ? types[i] : '',
                  child: Text(cols[i]),
                )),
            ],
            rows: rows.map((r) {
              final cells = (r as List);
              return DataRow(cells: [
                for (int i = 0; i < cells.length; i++)
                  DataCell(_buildCell(cells[i]), onTap: () => _copyCell(cells[i])),
              ]);
            }).toList(),
          ),
        ),
      ),
    );
  }

  Widget _buildCell(dynamic value) {
    if (value == null) {
      return const Text('NULL', style: TextStyle(color: AppColors.textMuted, fontStyle: FontStyle.italic));
    }
    final s = value.toString();
    final display = s.length > 80 ? '${s.substring(0, 80)}…' : s;
    return ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 320),
      child: Text(display, overflow: TextOverflow.ellipsis),
    );
  }

  void _copyCell(dynamic value) {
    if (value == null) return;
    Clipboard.setData(ClipboardData(text: value.toString()));
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(content: Text('Copied'), duration: Duration(seconds: 1), backgroundColor: AppColors.success),
    );
  }
}

enum _ResultView { cards, table }
