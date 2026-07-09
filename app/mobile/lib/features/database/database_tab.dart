import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/dbtool_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/database/schema_browser.dart';

// DatabaseTab is the mobile Database surface (simplified vs web): manage
// per-project connections, then drill into a connection to browse its
// schema, edit rows, and run SQL (reads + writes on writable connections).
class DatabaseTab extends ConsumerStatefulWidget {
  const DatabaseTab({required this.cwd, super.key});

  final String cwd;

  @override
  ConsumerState<DatabaseTab> createState() => _DatabaseTabState();
}

class _DatabaseTabState extends ConsumerState<DatabaseTab> {
  late Future<List<DbConnection>> _future;

  @override
  void initState() {
    super.initState();
    _reload();
  }

  @override
  void didUpdateWidget(DatabaseTab old) {
    super.didUpdateWidget(old);
    if (old.cwd != widget.cwd) _reload();
  }

  void _reload() {
    _future = ref.read(dbtoolApiProvider).listConnections(widget.cwd);
  }

  Future<void> _refresh() async {
    setState(_reload);
    await _future;
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      floatingActionButton: FloatingActionButton.small(
        onPressed: _openAddSheet,
        child: const Icon(Icons.add),
      ),
      body: RefreshIndicator(
        onRefresh: _refresh,
        child: FutureBuilder<List<DbConnection>>(
          future: _future,
          builder: (context, snap) {
            if (snap.connectionState == ConnectionState.waiting) {
              return const Center(child: CircularProgressIndicator());
            }
            if (snap.hasError) {
              final err = snap.error;
              return _ErrorView(
                message: err is ApiException ? err.message : '$err',
                onRetry: _refresh,
              );
            }
            final conns = snap.data ?? const [];
            if (conns.isEmpty) return _emptyState();
            return ListView.separated(
              padding: const EdgeInsets.all(12),
              itemCount: conns.length,
              separatorBuilder: (_, __) => const SizedBox(height: 8),
              itemBuilder: (context, i) => _connectionCard(conns[i]),
            );
          },
        ),
      ),
    );
  }

  Widget _emptyState() {
    return ListView(
      children: [
        const SizedBox(height: 80),
        const Icon(Icons.storage_outlined, size: 48, color: Colors.grey),
        const SizedBox(height: 12),
        Center(
          child: Text(
            t.web.database.panel.emptyTitle,
            style: Theme.of(context).textTheme.titleMedium,
          ),
        ),
        const SizedBox(height: 8),
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 32),
          child: Text(
            t.web.database.panel.emptyBody,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
        const SizedBox(height: 16),
        Center(
          child: FilledButton.icon(
            onPressed: _openAddSheet,
            icon: const Icon(Icons.add),
            label: Text(t.web.database.panel.addConnection),
          ),
        ),
      ],
    );
  }

  Widget _connectionCard(DbConnection c) {
    return Card(
      margin: EdgeInsets.zero,
      child: ListTile(
        leading: Icon(
          c.readOnly ? Icons.lock_outline : Icons.storage_outlined,
        ),
        title: Text(c.name),
        subtitle: Text(
          c.driver == 'sqlite'
              ? '${c.driver} · ${c.dbName}'
              : '${c.driver} · ${c.host}:${c.port}/${c.dbName}',
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        trailing: PopupMenuButton<String>(
          onSelected: (v) {
            if (v == 'edit') _openEditSheet(c);
            if (v == 'delete') _confirmDelete(c);
          },
          itemBuilder: (context) => [
            PopupMenuItem(
              value: 'edit',
              child: Text(t.web.database.panel.edit),
            ),
            PopupMenuItem(
              value: 'delete',
              child: Text(t.web.database.panel.delete),
            ),
          ],
        ),
        onTap: () {
          Navigator.of(context).push(
            MaterialPageRoute<void>(
              builder: (_) => SchemaBrowserScreen(connection: c),
            ),
          );
        },
      ),
    );
  }

  Future<void> _confirmDelete(DbConnection c) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (context) => AlertDialog(
        title: Text(t.web.database.panel.confirmDelete),
        content: Text(c.name),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: Text(t.web.database.panel.delete),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ref.read(dbtoolApiProvider).deleteConnection(c.id);
      if (mounted) await _refresh();
    } on ApiException catch (e) {
      _snack(e.message);
    }
  }

  Future<void> _openAddSheet() async {
    final saved = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (context) => _ConnectionForm(cwd: widget.cwd),
    );
    if ((saved ?? false) && mounted) await _refresh();
  }

  Future<void> _openEditSheet(DbConnection c) async {
    final saved = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      builder: (context) => _ConnectionForm(cwd: widget.cwd, existing: c),
    );
    if ((saved ?? false) && mounted) await _refresh();
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }
}

// _ConnectionForm is the add/edit-connection bottom sheet: the minimal
// fields plus a Test button. When `existing` is set it edits that
// connection (driver locked, password kept if left blank), mirroring the
// web ConnectionDialog; otherwise it creates a new one.
class _ConnectionForm extends ConsumerStatefulWidget {
  const _ConnectionForm({required this.cwd, this.existing});

  final String cwd;
  final DbConnection? existing;

  @override
  ConsumerState<_ConnectionForm> createState() => _ConnectionFormState();
}

class _ConnectionFormState extends ConsumerState<_ConnectionForm> {
  final _name = TextEditingController();
  final _host = TextEditingController();
  final _port = TextEditingController(text: '5432');
  final _db = TextEditingController();
  final _user = TextEditingController();
  final _pass = TextEditingController();
  bool _readOnly = false;
  bool _busy = false;
  DbPingResult? _ping;
  String _driver = 'postgres';

  static const _drivers = ['postgres', 'mysql', 'mariadb', 'sqlite'];
  static const _defaultPorts = <String, int>{
    'postgres': 5432,
    'mysql': 3306,
    'mariadb': 3306,
    'sqlite': 0,
  };

  bool get _isSqlite => _driver == 'sqlite';
  bool get _editing => widget.existing != null;

  @override
  void initState() {
    super.initState();
    final e = widget.existing;
    if (e != null) {
      _name.text = e.name;
      _driver = e.driver;
      _host.text = e.host;
      _port.text = e.driver == 'sqlite' ? '' : '${e.port}';
      _db.text = e.dbName;
      _user.text = e.username;
      _readOnly = e.readOnly;
      // password left blank — kept unless the user types a new one
    }
  }

  // Switching engine resets the port to that engine's default (SQLite is a
  // file path, no port) and clears the last ping.
  void _onDriver(String? v) {
    if (v == null) return;
    setState(() {
      _driver = v;
      _port.text = v == 'sqlite' ? '' : '${_defaultPorts[v] ?? 5432}';
      _ping = null;
    });
  }

  @override
  void dispose() {
    for (final c in [_name, _host, _port, _db, _user, _pass]) {
      c.dispose();
    }
    super.dispose();
  }

  DbConnectionInput _input() => DbConnectionInput(
    cwd: widget.cwd,
    name: _name.text.trim(),
    driver: _driver,
    host: _isSqlite ? '' : _host.text.trim(),
    port: _isSqlite
        ? 0
        : (int.tryParse(_port.text.trim()) ?? (_defaultPorts[_driver] ?? 5432)),
    dbName: _db.text.trim(),
    username: _isSqlite ? '' : _user.text.trim(),
    password: _isSqlite ? '' : _pass.text,
    readOnly: _readOnly,
  );

  bool get _valid =>
      _name.text.trim().isNotEmpty &&
      _db.text.trim().isNotEmpty &&
      (_isSqlite ||
          (_host.text.trim().isNotEmpty && _user.text.trim().isNotEmpty));

  Future<void> _test() async {
    setState(() => _busy = true);
    try {
      final res = await ref.read(dbtoolApiProvider).testParams(_input());
      setState(() => _ping = res);
    } on ApiException catch (e) {
      setState(() => _ping = DbPingResult(
        ok: false,
        serverVersion: '',
        isSuperuser: false,
        latencyMs: 0,
        error: e.message,
      ));
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  // Edit patch mirrors the web dialog: SQLite carries only name / path /
  // read_only; server engines carry host/port/user/ssl/read_only, and the
  // password only when the user typed a new one (blank keeps the stored
  // secret). ssl_mode is preserved from the existing connection (the mobile
  // form has no ssl field).
  Map<String, dynamic> _patch() {
    if (_isSqlite) {
      return {
        'name': _name.text.trim(),
        'db_name': _db.text.trim(),
        'read_only': _readOnly,
      };
    }
    final p = <String, dynamic>{
      'name': _name.text.trim(),
      'host': _host.text.trim(),
      'port':
          int.tryParse(_port.text.trim()) ?? (_defaultPorts[_driver] ?? 5432),
      'db_name': _db.text.trim(),
      'username': _user.text.trim(),
      'ssl_mode': widget.existing!.sslMode,
      'read_only': _readOnly,
    };
    if (_pass.text.isNotEmpty) p['password'] = _pass.text;
    return p;
  }

  Future<void> _save() async {
    if (!_valid) return;
    setState(() => _busy = true);
    try {
      final api = ref.read(dbtoolApiProvider);
      if (widget.existing != null) {
        await api.updateConnection(widget.existing!.id, _patch());
      } else {
        await api.createConnection(_input());
      }
      if (mounted) Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.message)));
      }
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final d = t.web.database.dialog;
    final superWarn = (_ping?.isSuperuser ?? false) ||
        _user.text.trim() == 'linivek';
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
            Text(_editing ? d.editTitle : d.createTitle,
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 12),
            _field(_name, d.name, onChanged: () => setState(() {})),
            Padding(
              padding: const EdgeInsets.only(bottom: 8),
              child: DropdownButtonFormField<String>(
                initialValue: _driver,
                decoration: InputDecoration(
                  labelText: d.driver,
                  isDense: true,
                  border: const OutlineInputBorder(),
                ),
                items: _drivers.map((v) {
                  final label = switch (v) {
                    'mysql' => d.drivers.mysql,
                    'mariadb' => d.drivers.mariadb,
                    'sqlite' => d.drivers.sqlite,
                    _ => d.drivers.postgres,
                  };
                  return DropdownMenuItem(value: v, child: Text(label));
                }).toList(),
                onChanged: _editing ? null : _onDriver,
              ),
            ),
            if (!_isSqlite) ...[
              _field(_host, d.host),
              _field(_port, d.port, keyboard: TextInputType.number),
            ],
            _field(
              _db,
              _isSqlite ? d.filePath : d.database,
              onChanged: () => setState(() {}),
            ),
            if (_isSqlite)
              Padding(
                padding: const EdgeInsets.only(bottom: 8),
                child: Text(
                  d.filePathHint,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              ),
            if (!_isSqlite) ...[
              _field(_user, d.username, onChanged: () => setState(() {})),
              _field(
                _pass,
                d.password,
                obscure: true,
                hint: _editing && widget.existing!.hasPassword
                    ? d.passwordKept
                    : null,
              ),
            ],
            SwitchListTile(
              contentPadding: EdgeInsets.zero,
              title: Text(d.readOnly),
              value: _readOnly,
              onChanged: (v) => setState(() => _readOnly = v),
            ),
            if (superWarn)
              _banner(d.superuserWarning, Colors.amber),
            if (_ping != null)
              _banner(
                _ping!.ok
                    ? d.testOk(
                        version: _ping!.serverVersion,
                        ms: _ping!.latencyMs,
                      )
                    : _ping!.error,
                _ping!.ok ? Colors.green : Colors.red,
              ),
            const SizedBox(height: 12),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _busy ? null : _test,
                    child: Text(d.test),
                  ),
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: FilledButton(
                    onPressed: _busy || !_valid ? null : _save,
                    child: Text(d.save),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _field(
    TextEditingController c,
    String label, {
    bool obscure = false,
    TextInputType? keyboard,
    VoidCallback? onChanged,
    String? hint,
  }) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: TextField(
        controller: c,
        obscureText: obscure,
        keyboardType: keyboard,
        onChanged: onChanged == null ? null : (_) => onChanged(),
        decoration: InputDecoration(
          labelText: label,
          hintText: hint,
          isDense: true,
          border: const OutlineInputBorder(),
        ),
      ),
    );
  }

  Widget _banner(String text, MaterialColor color) {
    return Container(
      margin: const EdgeInsets.only(top: 8),
      padding: const EdgeInsets.all(8),
      decoration: BoxDecoration(
        color: color.shade100,
        borderRadius: BorderRadius.circular(6),
      ),
      child: Text(
        text,
        style: TextStyle(fontSize: 12, color: color.shade800),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.message, required this.onRetry});

  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return ListView(
      children: [
        const SizedBox(height: 80),
        Center(child: Text(message, textAlign: TextAlign.center)),
        const SizedBox(height: 12),
        Center(
          child: OutlinedButton(
            onPressed: onRetry,
            child: Text(t.common.retry),
          ),
        ),
      ],
    );
  }
}
