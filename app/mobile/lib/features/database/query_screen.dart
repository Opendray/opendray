import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/dbtool_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// QueryScreen is the mobile SQL console: a multiline field plus a
// read-only results grid. Write statements are rejected server-side
// unless the connection is writable and the key carries db:write — the
// mobile client only surfaces the outcome.
class QueryScreen extends ConsumerStatefulWidget {
  const QueryScreen({required this.connection, super.key});

  final DbConnection connection;

  @override
  ConsumerState<QueryScreen> createState() => _QueryScreenState();
}

class _QueryScreenState extends ConsumerState<QueryScreen> {
  final _sql = TextEditingController();
  bool _busy = false;
  DbResultSet? _result;
  String? _error;

  @override
  void dispose() {
    _sql.dispose();
    super.dispose();
  }

  Future<void> _run() async {
    final sql = _sql.text.trim();
    if (sql.isEmpty) return;
    setState(() {
      _busy = true;
      _error = null;
    });
    try {
      final rs = await ref.read(dbtoolApiProvider).query(
            widget.connection.id,
            sql,
          );
      setState(() {
        _result = rs;
        _error = null;
      });
    } on ApiException catch (e) {
      setState(() {
        _error = e.message;
        _result = null;
      });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final d = t.web.database.console;
    return Scaffold(
      appBar: AppBar(
        title: Text('${t.web.database.panel.console} · ${widget.connection.name}'),
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: TextField(
              controller: _sql,
              maxLines: 5,
              minLines: 3,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
              decoration: InputDecoration(
                hintText: d.placeholder,
                border: const OutlineInputBorder(),
                isDense: true,
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(
              children: [
                FilledButton.icon(
                  onPressed: _busy ? null : _run,
                  icon: _busy
                      ? const SizedBox(
                          width: 14,
                          height: 14,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        )
                      : const Icon(Icons.play_arrow, size: 18),
                  label: Text(d.run),
                ),
                const Spacer(),
                if (_result != null)
                  Text(
                    d.stats(
                      command: _result!.command,
                      rows: _result!.rows.isNotEmpty
                          ? _result!.rows.length
                          : _result!.rowsAffected,
                      ms: _result!.durationMs,
                    ),
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
              ],
            ),
          ),
          const Divider(),
          Expanded(child: _results()),
        ],
      ),
    );
  }

  Widget _results() {
    if (_error != null) {
      return SingleChildScrollView(
        padding: const EdgeInsets.all(12),
        child: Text(
          _error!,
          style: const TextStyle(
            fontFamily: 'monospace',
            fontSize: 12,
            color: Colors.red,
          ),
        ),
      );
    }
    final rs = _result;
    if (rs == null) {
      return Center(child: Text(t.web.database.console.empty));
    }
    if (rs.columns.isEmpty) {
      return Center(
        child: Text(
          t.web.database.results.noColumns(
            command: rs.command,
            rows: rs.rowsAffected,
          ),
        ),
      );
    }
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
                  for (final cell in row)
                    DataCell(Text(cell == null ? 'NULL' : '$cell')),
                ],
              ),
          ],
        ),
      ),
    );
  }
}
