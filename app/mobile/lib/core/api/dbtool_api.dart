import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/dio_provider.dart';

// Wraps /api/v1/dbtool/* — the Database tool. Mobile ships a simplified
// surface: connection management + test, schema/table browsing, and
// read-only query execution. Row-level editing and write SQL are
// web-only; this client deliberately omits the insert/update/delete
// endpoints.
class DbtoolApi {
  DbtoolApi(this._dio);
  final Dio _dio;

  Future<List<DbConnection>> listConnections(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/dbtool/connections',
        queryParameters: {'cwd': cwd},
      );
      final raw = res.data?['connections'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(DbConnection.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<DbConnection> createConnection(DbConnectionInput input) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/dbtool/connections',
        data: input.toJson(),
      );
      return DbConnection.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<DbConnection> updateConnection(
    String id,
    Map<String, dynamic> patch,
  ) async {
    try {
      final res = await _dio.patch<Map<String, dynamic>>(
        '/api/v1/dbtool/connections/$id',
        data: patch,
      );
      return DbConnection.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> deleteConnection(String id) async {
    try {
      await _dio.delete<void>('/api/v1/dbtool/connections/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<DbPingResult> testParams(DbConnectionInput input) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/dbtool/connections/test',
        data: input.toJson(),
      );
      return DbPingResult.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<String>> listSchemas(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/dbtool/connections/$id/schemas',
      );
      final raw = res.data?['schemas'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map((m) => m['name']?.toString() ?? '')
          .where((s) => s.isNotEmpty)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<List<DbTable>> listTables(String id, String schema) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/dbtool/connections/$id/schemas/'
        '${Uri.encodeComponent(schema)}/tables',
      );
      final raw = res.data?['tables'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(DbTable.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<DbResultSet> tableData(
    String id, {
    required String schema,
    required String table,
    int limit = 100,
    int offset = 0,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/dbtool/connections/$id/table-data',
        data: {
          'schema': schema,
          'table': table,
          'limit': limit,
          'offset': offset,
        },
      );
      return DbResultSet.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<DbResultSet> query(String id, String sql, {int? maxRows}) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/dbtool/connections/$id/query',
        data: {'sql': sql, if (maxRows != null) 'max_rows': maxRows},
      );
      return DbResultSet.fromJson(res.data ?? {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

class DbConnection {
  DbConnection({
    required this.id,
    required this.cwd,
    required this.name,
    required this.driver,
    required this.host,
    required this.port,
    required this.dbName,
    required this.username,
    required this.sslMode,
    required this.readOnly,
    required this.hasPassword,
  });

  factory DbConnection.fromJson(Map<String, dynamic> j) => DbConnection(
    id: j['id']?.toString() ?? '',
    cwd: j['cwd']?.toString() ?? '',
    name: j['name']?.toString() ?? '',
    driver: j['driver']?.toString() ?? 'postgres',
    host: j['host']?.toString() ?? '',
    port: (j['port'] as num?)?.toInt() ?? 5432,
    dbName: j['db_name']?.toString() ?? '',
    username: j['username']?.toString() ?? '',
    sslMode: j['ssl_mode']?.toString() ?? 'prefer',
    readOnly: j['read_only'] == true,
    hasPassword: j['has_password'] == true,
  );

  final String id;
  final String cwd;
  final String name;
  final String driver;
  final String host;
  final int port;
  final String dbName;
  final String username;
  final String sslMode;
  final bool readOnly;
  final bool hasPassword;
}

class DbConnectionInput {
  DbConnectionInput({
    required this.cwd,
    required this.name,
    required this.host,
    required this.port,
    required this.dbName,
    required this.username,
    required this.password,
    this.sslMode = 'prefer',
    this.readOnly = false,
  });

  final String cwd;
  final String name;
  final String host;
  final int port;
  final String dbName;
  final String username;
  final String password;
  final String sslMode;
  final bool readOnly;

  Map<String, dynamic> toJson() => {
    'cwd': cwd,
    'name': name,
    'driver': 'postgres',
    'host': host,
    'port': port,
    'db_name': dbName,
    'username': username,
    'password': password,
    'ssl_mode': sslMode,
    'read_only': readOnly,
  };
}

class DbPingResult {
  DbPingResult({
    required this.ok,
    required this.serverVersion,
    required this.isSuperuser,
    required this.latencyMs,
    required this.error,
  });

  factory DbPingResult.fromJson(Map<String, dynamic> j) => DbPingResult(
    ok: j['ok'] == true,
    serverVersion: j['server_version']?.toString() ?? '',
    isSuperuser: j['is_superuser'] == true,
    latencyMs: (j['latency_ms'] as num?)?.toInt() ?? 0,
    error: j['error']?.toString() ?? '',
  );

  final bool ok;
  final String serverVersion;
  final bool isSuperuser;
  final int latencyMs;
  final String error;
}

class DbTable {
  DbTable({required this.name, required this.kind, required this.rowEstimate});

  factory DbTable.fromJson(Map<String, dynamic> j) => DbTable(
    name: j['name']?.toString() ?? '',
    kind: j['kind']?.toString() ?? 'table',
    rowEstimate: (j['row_estimate'] as num?)?.toInt() ?? 0,
  );

  final String name;
  final String kind;
  final int rowEstimate;
}

class DbColumnMeta {
  DbColumnMeta({required this.name, required this.type});

  factory DbColumnMeta.fromJson(Map<String, dynamic> j) => DbColumnMeta(
    name: j['name']?.toString() ?? '',
    type: j['type']?.toString() ?? '',
  );

  final String name;
  final String type;
}

class DbResultSet {
  DbResultSet({
    required this.columns,
    required this.rows,
    required this.rowsAffected,
    required this.command,
    required this.truncated,
    required this.durationMs,
  });

  factory DbResultSet.fromJson(Map<String, dynamic> j) {
    final cols = <DbColumnMeta>[];
    final rawCols = j['columns'];
    if (rawCols is List) {
      for (final c in rawCols.whereType<Map<String, dynamic>>()) {
        cols.add(DbColumnMeta.fromJson(c));
      }
    }
    final rows = <List<dynamic>>[];
    final rawRows = j['rows'];
    if (rawRows is List) {
      for (final r in rawRows) {
        if (r is List) rows.add(r);
      }
    }
    return DbResultSet(
      columns: cols,
      rows: rows,
      rowsAffected: (j['rows_affected'] as num?)?.toInt() ?? 0,
      command: j['command']?.toString() ?? '',
      truncated: j['truncated'] == true,
      durationMs: (j['duration_ms'] as num?)?.toInt() ?? 0,
    );
  }

  final List<DbColumnMeta> columns;
  final List<List<dynamic>> rows;
  final int rowsAffected;
  final String command;
  final bool truncated;
  final int durationMs;
}

final dbtoolApiProvider = Provider<DbtoolApi>((ref) {
  return DbtoolApi(ref.watch(dioProvider));
});
