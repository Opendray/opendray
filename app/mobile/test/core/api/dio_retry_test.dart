import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/dio_provider.dart';

// Guards the retry interceptor that keeps a suspended-then-resumed app
// from turning one dead socket into a full-screen "failed to load".
// The behaviour these lock in: replay reads on connection failures,
// never replay writes or real HTTP answers, and keep the caller's
// response type intact across the replay.

/// Fails the first [failures] calls with a connection error, then
/// answers 200. The failure is built from the options actually received
/// so it describes the real request, the way a live adapter's would.
class _FlakyAdapter implements HttpClientAdapter {
  _FlakyAdapter({required this.failures});

  final int failures;
  int calls = 0;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    calls++;
    if (calls <= failures) throw _connectionFailure(options);
    return ResponseBody.fromString(
      jsonEncode({'sessions': <dynamic>[]}),
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}

DioException _connectionFailure(RequestOptions options) => DioException(
      requestOptions: options,
      type: DioExceptionType.connectionError,
      error: const SocketException('Connection reset by peer'),
    );

Dio _dioWith(_FlakyAdapter adapter) {
  return buildDio(baseUrl: 'http://127.0.0.1:8770', token: 't')
    ..httpClientAdapter = adapter;
}

void main() {
  group('isTransientFailure', () {
    RequestOptions opts() => RequestOptions(path: '/x');

    test('connection-level failures are transient', () {
      for (final type in [
        DioExceptionType.connectionTimeout,
        DioExceptionType.sendTimeout,
        DioExceptionType.receiveTimeout,
        DioExceptionType.connectionError,
      ]) {
        expect(
          isTransientFailure(
            DioException(requestOptions: opts(), type: type),
          ),
          isTrue,
          reason: '$type should be retried',
        );
      }
    });

    test('a real HTTP answer is not transient', () {
      expect(
        isTransientFailure(
          DioException(
            requestOptions: opts(),
            type: DioExceptionType.badResponse,
          ),
        ),
        isFalse,
      );
    });

    test('cancellation is not transient', () {
      expect(
        isTransientFailure(
          DioException(requestOptions: opts(), type: DioExceptionType.cancel),
        ),
        isFalse,
      );
    });

    test('unknown counts only when it wraps a socket failure', () {
      expect(
        isTransientFailure(
          DioException(
            requestOptions: opts(),
            type: DioExceptionType.unknown,
            error: const SocketException('boom'),
          ),
        ),
        isTrue,
      );
      expect(
        isTransientFailure(
          DioException(
            requestOptions: opts(),
            type: DioExceptionType.unknown,
            error: const FormatException('bad json'),
          ),
        ),
        isFalse,
      );
    });
  });

  group('isReplayable', () {
    test('reads are replayable', () {
      expect(isReplayable(RequestOptions(path: '/x', method: 'GET')), isTrue);
      expect(isReplayable(RequestOptions(path: '/x', method: 'head')), isTrue);
    });

    test('writes are not — a replayed POST could double-create', () {
      for (final method in ['POST', 'PUT', 'PATCH', 'DELETE']) {
        expect(
          isReplayable(RequestOptions(path: '/x', method: method)),
          isFalse,
          reason: '$method must not be replayed',
        );
      }
    });

    test('streaming downloads opt out via a zero receive timeout', () {
      expect(
        isReplayable(
          RequestOptions(
            path: '/x',
            method: 'GET',
            receiveTimeout: Duration.zero,
          ),
        ),
        isFalse,
      );
    });
  });

  group('retry interceptor', () {
    test('recovers a dead socket and keeps the response type', () async {
      final adapter = _FlakyAdapter(failures: 1);
      final dio = _dioWith(adapter);

      // The typed call is the point: the replay must not hand back a
      // Response<dynamic> where the caller declared Map<String, dynamic>.
      final res = await dio.get<Map<String, dynamic>>('/api/v1/sessions');

      expect(adapter.calls, 2);
      expect(res.statusCode, 200);
      expect(res.data?['sessions'], isEmpty);
    });

    test('gives up after kMaxRetries and surfaces the failure', () async {
      final adapter = _FlakyAdapter(failures: 99);
      final dio = _dioWith(adapter);

      await expectLater(
        dio.get<Map<String, dynamic>>('/api/v1/sessions'),
        throwsA(isA<DioException>()),
      );
      expect(adapter.calls, kMaxRetries + 1);
    });

    test('does not replay a write', () async {
      final adapter = _FlakyAdapter(failures: 1);
      final dio = _dioWith(adapter);

      await expectLater(
        dio.post<Map<String, dynamic>>('/api/v1/sessions'),
        throwsA(isA<DioException>()),
      );
      expect(adapter.calls, 1);
    });
  });
}
