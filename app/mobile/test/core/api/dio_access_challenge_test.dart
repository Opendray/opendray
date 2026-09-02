import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/dio_provider.dart';

// Locks in how the HTTP client behaves when Cloudflare Access, rather
// than the gateway, answers a request. The failure this prevents: the
// operator sees "FormatException: Unexpected character" because an
// Access login page was parsed as JSON, with no hint that the fix is
// to sign in again at the edge.

/// Answers every request with [body] under [status] / [headers],
/// optionally reporting a different final URI the way a followed
/// redirect would.
class _CannedAdapter implements HttpClientAdapter {
  _CannedAdapter({
    required this.status,
    required this.body,
    required this.headers,
    this.redirectTo,
  });

  final int status;
  final String body;
  final Map<String, List<String>> headers;
  final Uri? redirectTo;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    final res = ResponseBody.fromString(body, status, headers: headers);
    if (redirectTo != null) {
      res.redirects = [
        RedirectRecord(302, 'GET', redirectTo!),
      ];
    }
    return res;
  }

  @override
  void close({bool force = false}) {}
}

const _accessLoginPage = '<!DOCTYPE html><title>Sign in</title>';

void main() {
  test('an Access login page becomes AccessChallengeException', () async {
    var challengedHost = '';
    final dio = buildDio(
      baseUrl: 'https://od.example.com',
      token: 't',
      onAccessChallenge: (h) => challengedHost = h,
    )..httpClientAdapter = _CannedAdapter(
        status: 200,
        body: _accessLoginPage,
        headers: {
          Headers.contentTypeHeader: ['text/html; charset=utf-8'],
        },
        redirectTo: Uri.parse(
          'https://acme.cloudflareaccess.com/cdn-cgi/access/login/od.example.com',
        ),
      );

    await expectLater(
      dio.get<dynamic>('/api/v1/sessions'),
      throwsA(
        isA<DioException>().having(
          (e) => e.error,
          'error',
          isA<AccessChallengeException>()
              .having((e) => e.host, 'host', 'acme.cloudflareaccess.com'),
        ),
      ),
    );
    expect(challengedHost, 'acme.cloudflareaccess.com');
  });

  test('the cookie rides on every request when we hold one', () async {
    late RequestOptions seen;
    final dio = buildDio(
      baseUrl: 'https://od.example.com',
      token: 't',
      cfCookie: 'jwt-value',
    )..httpClientAdapter = _RecordingAdapter((o) => seen = o);

    await dio.get<dynamic>('/api/v1/health');
    expect(seen.headers['Cookie'], 'CF_Authorization=jwt-value');
  });

  test('no cookie header at all on a LAN deployment', () async {
    late RequestOptions seen;
    final dio = buildDio(baseUrl: 'http://192.168.3.5:8770', token: 't')
      ..httpClientAdapter = _RecordingAdapter((o) => seen = o);

    await dio.get<dynamic>('/api/v1/health');
    expect(seen.headers.containsKey('Cookie'), isFalse);
  });

  // An expired bearer must keep reaching onUnauthorized. Routing it
  // into the Access flow instead would pop a WebView that signs in
  // successfully and leaves the app just as logged out.
  test('a gateway 401 is still a plain 401', () async {
    var unauthorized = false;
    var challenged = false;
    final dio = buildDio(
      baseUrl: 'https://od.example.com',
      token: 'stale',
      cfCookie: 'jwt-value',
      onUnauthorized: () => unauthorized = true,
      onAccessChallenge: (_) => challenged = true,
    )..httpClientAdapter = _CannedAdapter(
        status: 401,
        body: '{"error":"unauthorized"}',
        headers: {
          Headers.contentTypeHeader: [Headers.jsonContentType],
        },
      );

    await expectLater(
      dio.get<dynamic>('/api/v1/sessions'),
      throwsA(
        isA<DioException>().having(
          (e) => e.error,
          'error',
          isA<ApiException>().having((e) => e.statusCode, 'status', 401),
        ),
      ),
    );
    expect(unauthorized, isTrue);
    expect(challenged, isFalse);
  });
}

class _RecordingAdapter implements HttpClientAdapter {
  _RecordingAdapter(this.onFetch);
  final void Function(RequestOptions) onFetch;

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    onFetch(options);
    return ResponseBody.fromString(
      '{"status":"ok"}',
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );
  }

  @override
  void close({bool force = false}) {}
}
