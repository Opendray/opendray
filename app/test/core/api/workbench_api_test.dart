import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/workbench_models.dart';

/// A Dio interceptor that short-circuits every request with a preset
/// response, so these tests never hit the network.
class _StubInterceptor extends Interceptor {
  _StubInterceptor(this._handler);
  final Response<dynamic> Function(RequestOptions) _handler;

  @override
  void onRequest(RequestOptions options, RequestInterceptorHandler h) {
    h.resolve(_handler(options));
  }
}

// ApiClient keeps its Dio private, so these tests exercise the wire
// logic through a minimal re-implementation on a stubbed Dio. When
// ApiClient grows a public testable entrypoint in M2, switch to call
// it directly — the error-mapping rules below are the canonical
// contract until then.
Future<FlatContributions> _callGetContributions(Dio dio) async {
  final res = await dio.get('/api/workbench/contributions');
  final data = res.data;
  if (data is Map) {
    return FlatContributions.fromJson(Map<String, dynamic>.from(data));
  }
  return FlatContributions.empty;
}

Future<InvokeResult> _callInvokePluginCommand(
  Dio dio,
  String pluginName,
  String commandId, {
  Map<String, dynamic>? args,
}) async {
  try {
    final res = await dio.post(
      '/api/plugins/$pluginName/commands/$commandId/invoke',
      data: {'args': args ?? const <String, dynamic>{}},
    );
    final data = res.data;
    if (data is Map) {
      return InvokeResult.fromJson(Map<String, dynamic>.from(data));
    }
    return const InvokeResult(kind: '');
  } on DioException catch (e) {
    final status = e.response?.statusCode ?? 0;
    final body = e.response?.data;
    final code = body is Map ? body['code']?.toString() ?? '' : '';
    final msg = body is Map ? body['msg']?.toString() ?? '' : '';
    if (status == 403 && code == 'EPERM') {
      throw PluginPermissionDeniedException(pluginName, commandId,
          msg.isEmpty ? 'permission denied' : msg);
    }
    if (status == 404 && code == 'ENOTFOUND') {
      throw PluginCommandUnavailableException(pluginName, commandId,
          msg.isEmpty ? 'command not found' : msg);
    }
    if (status == 501 && code == 'ENOTIMPL') {
      throw PluginCommandUnavailableException(pluginName, commandId,
          msg.isEmpty ? 'run kind deferred' : msg,
          deferred: true);
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
  group('getContributions wire mapping', () {
    test('maps JSON payload into FlatContributions', () async {
      final dio = _stubDio((opts) => _jsonRes(opts, 200, {
            'commands': [
              {'id': 'time.start', 'pluginName': 'time-ninja', 'title': 'Start'}
            ],
            'statusBar': [],
            'keybindings': [],
            'menus': <String, List<Map<String, dynamic>>>{},
          }));
      final c = await _callGetContributions(dio);
      expect(c.commands.single.id, 'time.start');
      expect(c.statusBar, isEmpty);
    });

    test('non-map body maps to empty', () async {
      final dio = _stubDio((opts) => _jsonRes(opts, 200, 'oops'));
      final c = await _callGetContributions(dio);
      expect(c.commands, isEmpty);
    });
  });

  group('invokePluginCommand wire mapping', () {
    test('200 notify → InvokeResult', () async {
      final dio = _stubDio((opts) => _jsonRes(opts, 200, {
            'kind': 'notify',
            'message': 'Pomodoro started',
          }));
      final r = await _callInvokePluginCommand(dio, 'time-ninja', 'time.start');
      expect(r.kind, 'notify');
      expect(r.message, 'Pomodoro started');
    });

    test('403 EPERM → PluginPermissionDeniedException', () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 403,
            data: {'code': 'EPERM', 'msg': 'exec not granted'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      expect(
        () => _callInvokePluginCommand(dio, 'bad', 'x'),
        throwsA(isA<PluginPermissionDeniedException>()
            .having((e) => e.reason, 'reason', contains('exec'))),
      );
    });

    test('404 ENOTFOUND → PluginCommandUnavailableException (not deferred)',
        () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 404,
            data: {'code': 'ENOTFOUND', 'msg': 'no such command'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      try {
        await _callInvokePluginCommand(dio, 'p', 'c');
        fail('expected PluginCommandUnavailableException');
      } on PluginCommandUnavailableException catch (e) {
        expect(e.deferred, isFalse);
        expect(e.reason, contains('no such'));
      }
    });

    test('501 ENOTIMPL → PluginCommandUnavailableException (deferred)',
        () async {
      final dio = _stubDio((opts) {
        throw DioException(
          requestOptions: opts,
          response: Response(
            requestOptions: opts,
            statusCode: 501,
            data: {'code': 'ENOTIMPL', 'msg': 'requires M2'},
          ),
          type: DioExceptionType.badResponse,
        );
      });
      try {
        await _callInvokePluginCommand(dio, 'p', 'c');
        fail('expected PluginCommandUnavailableException');
      } on PluginCommandUnavailableException catch (e) {
        expect(e.deferred, isTrue);
      }
    });

    test('500 falls through to ApiException', () async {
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
        () => _callInvokePluginCommand(dio, 'p', 'c'),
        throwsA(isA<ApiException>()),
      );
    });

    test('args map posted under "args" key', () async {
      Map<String, dynamic>? captured;
      final dio = _stubDio((opts) {
        captured = opts.data as Map<String, dynamic>?;
        return _jsonRes(opts, 200, {'kind': 'notify'});
      });
      await _callInvokePluginCommand(dio, 'p', 'c', args: {'x': 1});
      expect(captured?['args'], {'x': 1});
    });
  });
}
