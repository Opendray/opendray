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
  late Future<DbResultSet> _future;

  @override
  void initState() {
    super.initState();
    _load();
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

  void _go(int delta) {
    setState(() {
      _page = (_page + delta).clamp(0, 1 << 30);
      _load();
    });
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text('${widget.schema}.${widget.table}')),
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
              Expanded(child: _grid(rs)),
              _pager(rs),
            ],
          );
        },
      ),
    );
  }

  Widget _grid(DbResultSet rs) {
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      child: SingleChildScrollView(
        child: DataTable(
          columns: [
            for (final c in rs.columns) DataColumn(label: Text(c.name)),
          ],
          rows: [
            for (final row in rs.rows)
              DataRow(
                cells: [
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
