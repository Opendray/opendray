import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/dbtool_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/database/query_screen.dart';

// SchemaBrowserScreen drills into one connection: schema → table list →
// read-only paged rows. A SQL console entry lives in the app bar. All
// read-only — row editing is web-only.
class SchemaBrowserScreen extends ConsumerStatefulWidget {
  const SchemaBrowserScreen({required this.connection, super.key});

  final DbConnection connection;

  @override
  ConsumerState<SchemaBrowserScreen> createState() =>
      _SchemaBrowserScreenState();
}

class _SchemaBrowserScreenState extends ConsumerState<SchemaBrowserScreen> {
  late Future<List<String>> _schemas;

  @override
  void initState() {
    super.initState();
    _schemas = ref.read(dbtoolApiProvider).listSchemas(widget.connection.id);
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.connection.name),
        actions: [
          IconButton(
            icon: const Icon(Icons.terminal),
            tooltip: t.web.database.panel.console,
            onPressed: () {
              Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => QueryScreen(connection: widget.connection),
                ),
              );
            },
          ),
        ],
      ),
      body: FutureBuilder<List<String>>(
        future: _schemas,
        builder: (context, snap) {
          if (snap.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError) {
            final err = snap.error;
            return Center(
              child: Text(err is ApiException ? err.message : '$err'),
            );
          }
          final schemas = snap.data ?? const [];
          if (schemas.isEmpty) {
            return Center(child: Text(t.web.database.tree.noSchemas));
          }
          return ListView(
            children: [
              for (final s in schemas)
                _SchemaTile(connection: widget.connection, schema: s),
            ],
          );
        },
      ),
    );
  }
}

class _SchemaTile extends ConsumerWidget {
  const _SchemaTile({required this.connection, required this.schema});

  final DbConnection connection;
  final String schema;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return ExpansionTile(
      leading: const Icon(Icons.folder_outlined),
      title: Text(schema),
      initiallyExpanded: schema == 'public',
      children: [
        FutureBuilder<List<DbTable>>(
          future: ref
              .read(dbtoolApiProvider)
              .listTables(connection.id, schema),
          builder: (context, snap) {
            if (snap.connectionState == ConnectionState.waiting) {
              return const Padding(
                padding: EdgeInsets.all(12),
                child: Center(child: CircularProgressIndicator()),
              );
            }
            final tables = snap.data ?? const [];
            return Column(
              children: [
                for (final tb in tables)
                  ListTile(
                    dense: true,
                    contentPadding: const EdgeInsets.only(left: 32, right: 16),
                    leading: Icon(
                      tb.kind == 'view'
                          ? Icons.visibility_outlined
                          : Icons.table_chart_outlined,
                      size: 18,
                    ),
                    title: Text(tb.name),
                    onTap: () {
                      Navigator.of(context).push(
                        MaterialPageRoute<void>(
                          builder: (_) => _TableDataScreen(
                            connection: connection,
                            schema: schema,
                            table: tb.name,
                          ),
                        ),
                      );
                    },
                  ),
              ],
            );
          },
        ),
      ],
    );
  }
}

// _TableDataScreen shows one table's rows, read-only, paginated. Rows
// scroll horizontally inside a DataTable so wide tables stay usable.
class _TableDataScreen extends ConsumerStatefulWidget {
  const _TableDataScreen({
    required this.connection,
    required this.schema,
    required this.table,
  });

  final DbConnection connection;
  final String schema;
  final String table;

  @override
  ConsumerState<_TableDataScreen> createState() => _TableDataScreenState();
}

class _TableDataScreenState extends ConsumerState<_TableDataScreen> {
  static const _pageSize = 50;
  int _page = 0;
  DbTableMeta? _meta;
  late Future<DbResultSet> _future;

  @override
  void initState() {
    super.initState();
    _loadMeta();
    _load();
  }

  Future<void> _loadMeta() async {
    try {
      final m = await ref
          .read(dbtoolApiProvider)
          .tableMeta(widget.connection.id, widget.schema, widget.table);
      if (mounted) setState(() => _meta = m);
    } on Object {
      // Metadata is best-effort; without it, editing stays disabled.
    }
  }

  void _load() {
    _future = ref.read(dbtoolApiProvider).tableData(
          widget.connection.id,
          schema: widget.schema,
          table: widget.table,
          limit: _pageSize,
          offset: _page * _pageSize,
        );
  }

  void _reload() => setState(_load);

  void _go(int delta) {
    setState(() {
      _page = (_page + delta).clamp(0, 1 << 30);
      _load();
    });
  }

  // Editing needs a writable connection AND a primary key to address rows.
  bool get _editable =>
      !widget.connection.readOnly && (_meta?.primaryKey.isNotEmpty ?? false);

  Map<String, dynamic> _rowMap(DbResultSet rs, List<dynamic> row) {
    final m = <String, dynamic>{};
    for (var i = 0; i < rs.columns.length && i < row.length; i++) {
      m[rs.columns[i].name] = row[i];
    }
    return m;
  }

  Future<void> _openEditor({Map<String, dynamic>? row}) async {
    final meta = _meta;
    if (meta == null) return;
    final saved = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (_) =>
          _RowEditorSheet(connection: widget.connection, meta: meta, row: row),
    );
    if (saved ?? false) _reload();
  }

  Future<void> _deleteRow(Map<String, dynamic> rowMap) async {
    final meta = _meta;
    if (meta == null) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(t.web.database.grid.confirmDelete),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: Text(t.web.database.grid.delete),
          ),
        ],
      ),
    );
    if (ok != true) return;
    final pk = <String, dynamic>{};
    for (final k in meta.primaryKey) {
      pk[k] = rowMap[k];
    }
    try {
      await ref.read(dbtoolApiProvider).deleteRows(
            widget.connection.id,
            schema: widget.schema,
            table: widget.table,
            pks: [pk],
          );
      if (mounted) {
        _reload();
        _snack(t.web.database.grid.deleted);
      }
    } on ApiException catch (e) {
      _snack(e.message);
    }
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text('${widget.schema}.${widget.table}'),
        actions: [
          if (_editable)
            IconButton(
              icon: const Icon(Icons.add),
              tooltip: t.web.database.grid.insert,
              onPressed: _openEditor,
            ),
        ],
      ),
      body: FutureBuilder<DbResultSet>(
        future: _future,
        builder: (context, snap) {
          if (snap.connectionState == ConnectionState.waiting) {
            return const Center(child: CircularProgressIndicator());
          }
          if (snap.hasError) {
            final err = snap.error;
            return Center(
              child: Text(err is ApiException ? err.message : '$err'),
            );
          }
          final rs = snap.data;
          if (rs == null || rs.columns.isEmpty) {
            return Center(child: Text(t.web.database.grid.loading));
          }
          return Column(
            children: [
              if (widget.connection.readOnly)
                _hint(t.web.database.grid.readOnlyHint)
              else if (_meta != null && _meta!.primaryKey.isEmpty)
                _hint(t.web.database.grid.noPkHint),
              Expanded(child: _grid(rs)),
              _pager(rs),
            ],
          );
        },
      ),
    );
  }

  Widget _hint(String text) => Container(
        width: double.infinity,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        color: Theme.of(context).colorScheme.surfaceContainerHighest,
        child: Text(text, style: Theme.of(context).textTheme.bodySmall),
      );

  Widget _grid(DbResultSet rs) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: SingleChildScrollView(
        child: DataTable(
          columns: [
            if (_editable) const DataColumn(label: Text('')),
            for (final c in rs.columns) DataColumn(label: Text(c.name)),
          ],
          rows: [
            for (final row in rs.rows)
              DataRow(
                cells: [
                  if (_editable)
                    DataCell(Row(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        IconButton(
                          icon: const Icon(Icons.edit_outlined, size: 18),
                          onPressed: () => _openEditor(row: _rowMap(rs, row)),
                        ),
                        IconButton(
                          icon: const Icon(Icons.delete_outline, size: 18),
                          onPressed: () => _deleteRow(_rowMap(rs, row)),
                        ),
                      ],
                    )),
                  for (final cell in row) DataCell(Text(_cell(cell))),
                ],
              ),
          ],
        ),
      ),
    );
  }

  Widget _pager(DbResultSet rs) {
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.spaceBetween,
        children: [
          Text(
            t.web.database.grid.pageInfo(
              from: _page * _pageSize + 1,
              to: _page * _pageSize + rs.rows.length,
            ),
          ),
          Row(
            children: [
              IconButton(
                icon: const Icon(Icons.chevron_left),
                onPressed: _page == 0 ? null : () => _go(-1),
              ),
              Text('${_page + 1}'),
              IconButton(
                icon: const Icon(Icons.chevron_right),
                onPressed: rs.truncated ? () => _go(1) : null,
              ),
            ],
          ),
        ],
      ),
    );
  }

  String _cell(dynamic v) {
    if (v == null) return 'NULL';
    if (v is Map || v is List) return v.toString();
    return '$v';
  }
}

// _RowEditorSheet is the insert/edit form for one row. Every column gets
// a text field plus a NULL toggle (nullable columns). On edit, the
// primary key addresses the row; on insert, non-empty / explicit-NULL
// fields are sent. Mirrors the web row dialog.
class _RowEditorSheet extends ConsumerStatefulWidget {
  const _RowEditorSheet({
    required this.connection,
    required this.meta,
    this.row,
  });

  final DbConnection connection;
  final DbTableMeta meta;
  final Map<String, dynamic>? row; // null = insert

  @override
  ConsumerState<_RowEditorSheet> createState() => _RowEditorSheetState();
}

class _RowEditorSheetState extends ConsumerState<_RowEditorSheet> {
  final _controllers = <String, TextEditingController>{};
  final _isNull = <String, bool>{};
  bool _busy = false;

  bool get _editing => widget.row != null;

  @override
  void initState() {
    super.initState();
    for (final c in widget.meta.columns) {
      final v = widget.row?[c.name];
      _controllers[c.name] =
          TextEditingController(text: v == null ? '' : _cellToInput(v));
      _isNull[c.name] = _editing && v == null;
    }
  }

  @override
  void dispose() {
    for (final c in _controllers.values) {
      c.dispose();
    }
    super.dispose();
  }

  Map<String, dynamic> _buildValues() {
    final out = <String, dynamic>{};
    for (final c in widget.meta.columns) {
      final isNull = _isNull[c.name] ?? false;
      final text = _controllers[c.name]!.text;
      final newVal = isNull ? null : _coerce(text);
      if (_editing) {
        final orig = widget.row![c.name];
        if (orig != newVal) out[c.name] = newVal;
      } else if (isNull) {
        out[c.name] = null;
      } else if (text.isNotEmpty) {
        out[c.name] = newVal;
      }
    }
    return out;
  }

  Future<void> _save() async {
    setState(() => _busy = true);
    try {
      final api = ref.read(dbtoolApiProvider);
      if (_editing) {
        final pk = <String, dynamic>{};
        for (final k in widget.meta.primaryKey) {
          pk[k] = widget.row![k];
        }
        await api.updateRow(
          widget.connection.id,
          schema: widget.meta.schema,
          table: widget.meta.table,
          pk: pk,
          values: _buildValues(),
        );
      } else {
        await api.insertRow(
          widget.connection.id,
          schema: widget.meta.schema,
          table: widget.meta.table,
          values: _buildValues(),
        );
      }
      if (mounted) Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.message)));
        setState(() => _busy = false);
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final pkSet = widget.meta.primaryKey.toSet();
    return Padding(
      padding: EdgeInsets.only(
        left: 16,
        right: 16,
        top: 16,
        bottom: MediaQuery.of(context).viewInsets.bottom + 16,
      ),
      child: SingleChildScrollView(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              _editing
                  ? t.web.database.row.editTitle
                  : t.web.database.row.insertTitle,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            Text(
              '${widget.meta.schema}.${widget.meta.table}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            for (final col in widget.meta.columns)
              _field(col, isPk: pkSet.contains(col.name)),
            const SizedBox(height: 12),
            FilledButton(
              onPressed: _busy ? null : _save,
              child: _busy
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(t.web.database.row.save),
            ),
          ],
        ),
      ),
    );
  }

  Widget _field(DbColumn col, {required bool isPk}) {
    final isNull = _isNull[col.name] ?? false;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          TextField(
            controller: _controllers[col.name],
            enabled: !isNull,
            style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
            decoration: InputDecoration(
              labelText: isPk ? '${col.name}  (PK)' : col.name,
              hintText: col.dataType,
              isDense: true,
              border: const OutlineInputBorder(),
            ),
          ),
          if (col.nullable)
            Row(
              mainAxisAlignment: MainAxisAlignment.end,
              children: [
                Text(
                  t.web.database.row.setNull,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                Checkbox(
                  value: isNull,
                  onChanged: (v) =>
                      setState(() => _isNull[col.name] = v ?? false),
                ),
              ],
            ),
        ],
      ),
    );
  }
}

// _cellToInput renders a stored value for the text field.
String _cellToInput(dynamic v) {
  if (v is Map || v is List) return v.toString();
  return '$v';
}

// _coerce turns the raw input text into a JSON value so numbers/bools
// reach Postgres as the right type (mirrors the web row editor).
dynamic _coerce(String text) {
  final trimmed = text.trim();
  if (trimmed.isEmpty) return '';
  if (trimmed == 'true') return true;
  if (trimmed == 'false') return false;
  final asInt = int.tryParse(trimmed);
  if (asInt != null) return asInt;
  final asDouble = double.tryParse(trimmed);
  if (asDouble != null) return asDouble;
  return text;
}
