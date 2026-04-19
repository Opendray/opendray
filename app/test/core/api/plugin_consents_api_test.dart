import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';

/// Dio interceptor that short-circuits every request with a preset
/// response, so these tests never hit the network. Same pattern as
/// workbench_api_test.dart — keep the two files in the same shape so
/// the stubbed-Dio convention stays recognisable.
class _StubInterceptor extends Interceptor {
  _StubInterceptor(this._handler);
  final Response<dynamic> Function(RequestOptions) _handler;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler h) {
    h.resolve(_handler(options));
  }
}

// ApiClient keeps its Dio private, so these tests mirror the wire logic
// in a minimal re-implementation on a stubbed Dio. When ApiClient gains
// a public testable entrypoint the re-implementation can be dropped —
// the error-mapping rules encoded below are the canonical contract
// until then.
Future<PluginConsents> _callGetPluginConsents(Dio dio, String name) async {
  try {
    final res = await dio.get('/api/plugins/$name/consents');
    final data = res.data;
    if (data is Map) {
      return PluginConsents.fromJson(
        Map<String, dynamic>.from(data),
        pluginName: name,
      );
    }
    return PluginConsents(pluginName: name, perms: const {});
  } on DioException catch (e) {
    final status = e.response?.statusCode ?? 0;
    if (status == 404) {
      throw PluginConsentNotFoundException(name);
    }
    throw apiExceptionFrom(e);
  }
}

Future<void> _callRevokePluginCapability(
  Dio dio,
  String name,
  String cap,
) async {
  try {
    await dio.delete('/api/plugins/$name/consents/$cap');
  } on DioException catch (e) {
    final status = e.response?.statusCode ?? 0;
    if (status == 404) {
      throw PluginConsentNotFoundException(name);
    }
    throw apiExceptionFrom(e);
  }
}

Future<void> _callRevokeAllPluginConsents(Dio dio, String name) async {
  try {
    await dio.delete('/api/plugins/$name/consents');
  } on DioException catch (e) {
    final status = e.response?.statusCode ?? 0;
    if (status == 404) {
      throw PluginConsentNotFoundException(name);
    }
    throw apiExceptionFrom(e);
  }
}

Dio _stubDio(Response<dynamic> Function(RequestOptions) fn) {
  final dio = Dio(BaseOptions(baseUrl: 'http://stub.local'));
  dio.interceptors.add(_StubInterceptor(fn));
  return dio;
}

Response<dynamic> _jsonRes(RequestOptions opts, int status, Object body) {
  return Response(
    requestOptions: opts,
    statusCode: status,
    data: body,
    statusMessage: 'stub',
  );
}

void main() {
  group('getPluginConsents wire mapping', () {
    test('parses perms map + timestamps', () async {
      final dio = _stubDio((opts) {
        expect(opts.method, 'GET');
        expect(opts.path, '/api/plugins/kanban/consents');
        return _jsonRes(opts, 200, {
          'perms': {
            'storage': true,
            'exec': ['git *', 'npm *'],
            'session': 'read',
          },
          'grantedAt': '2026-04-01T12:00:00Z',
          'updatedAt': '2026-04-10T08:00:00Z',
        });
      });
      final c = await _callGetPluginConsents(dio, 'kanban');
      expect(c.pluginName, 'kanban');
      expect(c.perms['storage'], true);
      expect(c.perms['session'], 'read');
      expect((c.perms['exec'] as List).length, 2);
      expect(c.grantedAt?.toUtc().toIso8601String(),
          '2026-04-01T12:00:00.000Z');
      expect(c.updatedAt?.toUtc().toIso8601String(),
          '2026-04-10T08:00:00.000Z');
    });

    test('tolerates missing timestamps (nullable)', () async {
      final dio = _stubDio(
          (opts) => _jsonRes(opts, 200, {'perms': {}, 'grantedAt': null}));
      final c = await _callGetPluginConsents(dio, 'p');
      expect(c.perms, isEmpty);
      expect(c.grantedAt, isNull);
      expect(c.updatedAt, isNull);
    });

    test('404 → PluginConsentNotFoundException', () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 404,
            data: {'code': 'ENOCONSENT', 'msg': 'no consent row for plugin p'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      expect(
        () => _callGetPluginConsents(dio, 'p'),
        throwsA(isA<PluginConsentNotFoundException>()
            .having((e) => e.pluginName, 'pluginName', 'p')),
      );
    });

    test('5xx → ApiException', () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 500,
            data: {'error': 'boom'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      expect(
        () => _callGetPluginConsents(dio, 'p'),
        throwsA(isA<ApiException>()),
      );
    });
  });

  group('revokePluginCapability wire mapping', () {
    test('sends DELETE to correct URL', () async {
      final calls = <String>[];
      final dio = _stubDio((opts) {
        calls.add('${opts.method} ${opts.path}');
        return _jsonRes(opts, 200, {'revoked': true, 'cap': 'storage'});
      });
      await _callRevokePluginCapability(dio, 'kanban', 'storage');
      expect(calls, ['DELETE /api/plugins/kanban/consents/storage']);
    });

    test('404 → PluginConsentNotFoundException', () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 404,
            data: {'code': 'ENOCONSENT'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      expect(
        () => _callRevokePluginCapability(dio, 'kanban', 'storage'),
        throwsA(isA<PluginConsentNotFoundException>()),
      );
    });

    test('400 EINVAL unknown cap → ApiException', () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 400,
            data: {'code': 'EINVAL', 'msg': 'unknown capability: nope'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      expect(
        () => _callRevokePluginCapability(dio, 'kanban', 'nope'),
        throwsA(isA<ApiException>()),
      );
    });
  });

  group('revokeAllPluginConsents wire mapping', () {
    test('sends DELETE without cap suffix', () async {
      final calls = <String>[];
      final dio = _stubDio((opts) {
        calls.add('${opts.method} ${opts.path}');
        return _jsonRes(opts, 200, {'revoked': 'all', 'name': 'kanban'});
      });
      await _callRevokeAllPluginConsents(dio, 'kanban');
      expect(calls, ['DELETE /api/plugins/kanban/consents']);
    });

    test('404 → PluginConsentNotFoundException', () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 404,
            data: {'code': 'ENOCONSENT'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      expect(
        () => _callRevokeAllPluginConsents(dio, 'kanban'),
        throwsA(isA<PluginConsentNotFoundException>()),
      );
    });
  });

  group('PluginConsents.isCapGranted rules', () {
    // Table-driven coverage for every cap shape handled by the rule
    // matrix. Keep these in lock-step with the rule table in
    // ApiClient's PluginConsents.isCapGranted docstring.
    final cases = <Map<String, dynamic>>[
      // storage / secret / telegram / llm → bool
      {'cap': 'storage', 'granted': {'storage': true}, 'expect': true},
      {'cap': 'storage', 'granted': {'storage': false}, 'expect': false},
      {'cap': 'storage', 'granted': {}, 'expect': false},
      {'cap': 'secret', 'granted': {'secret': true}, 'expect': true},
      {'cap': 'secret', 'granted': {'secret': false}, 'expect': false},
      {'cap': 'telegram', 'granted': {'telegram': true}, 'expect': true},
      {'cap': 'telegram', 'granted': {}, 'expect': false},
      {'cap': 'llm', 'granted': {'llm': true}, 'expect': true},
      {'cap': 'llm', 'granted': {'llm': false}, 'expect': false},

      // session / clipboard / git → non-empty string
      {'cap': 'session', 'granted': {'session': 'read'}, 'expect': true},
      {'cap': 'session', 'granted': {'session': ''}, 'expect': false},
      {'cap': 'session', 'granted': {}, 'expect': false},
      {'cap': 'clipboard', 'granted': {'clipboard': 'read-write'}, 'expect': true},
      {'cap': 'clipboard', 'granted': {'clipboard': ''}, 'expect': false},
      {'cap': 'git', 'granted': {'git': 'read'}, 'expect': true},
      {'cap': 'git', 'granted': {}, 'expect': false},

      // fs / exec / http → presence of any non-null value
      {'cap': 'fs', 'granted': {'fs': true}, 'expect': true},
      {'cap': 'fs', 'granted': {'fs': {'read': ['/tmp']}}, 'expect': true},
      {'cap': 'fs', 'granted': {'fs': ['/tmp']}, 'expect': true},
      {'cap': 'fs', 'granted': {'fs': null}, 'expect': false},
      {'cap': 'fs', 'granted': {}, 'expect': false},
      {'cap': 'exec', 'granted': {'exec': ['git *', 'npm *']}, 'expect': true},
      {'cap': 'exec', 'granted': {'exec': []}, 'expect': false},
      {'cap': 'exec', 'granted': {}, 'expect': false},
      {'cap': 'http', 'granted': {'http': {'hosts': ['api.example.com']}}, 'expect': true},
      {'cap': 'http', 'granted': {'http': null}, 'expect': false},
      {'cap': 'http', 'granted': {}, 'expect': false},

      // events → non-empty array
      {'cap': 'events', 'granted': {'events': ['session.*']}, 'expect': true},
      {'cap': 'events', 'granted': {'events': []}, 'expect': false},
      {'cap': 'events', 'granted': {}, 'expect': false},

      // unknown cap → false (defensive)
      {'cap': 'bogus', 'granted': {'bogus': true}, 'expect': false},
    ];

    for (final c in cases) {
      test('cap=${c['cap']} perms=${c['granted']} → ${c['expect']}', () {
        final consents = PluginConsents(
          pluginName: 't',
          perms: Map<String, dynamic>.from(c['granted'] as Map),
        );
        expect(consents.isCapGranted(c['cap'] as String),
            c['expect'] as bool);
      });
    }
  });

  group('PluginConsents.fromJson', () {
    test('accepts perms absent (defaults to empty map)', () {
      final c = PluginConsents.fromJson(const {}, pluginName: 'p');
      expect(c.perms, isEmpty);
      expect(c.pluginName, 'p');
    });

    test('parses timestamps that are plain strings', () {
      final c = PluginConsents.fromJson({
        'perms': {'storage': true},
        'grantedAt': '2026-04-19T10:00:00Z',
        'updatedAt': '2026-04-19T11:00:00Z',
      }, pluginName: 'p');
      expect(c.grantedAt, isA<DateTime>());
      expect(c.updatedAt, isA<DateTime>());
      expect(c.isCapGranted('storage'), true);
    });
  });
}
