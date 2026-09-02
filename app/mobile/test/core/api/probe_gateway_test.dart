import 'dart:typed_data';

import 'package:dio/dio.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/auth_api.dart';

// Reproduces the reported onboarding failure end to end.
//
// Entering the published gateway URL showed "Server replied 0:
// Network error". Nothing was wrong with the network: Cloudflare
// Access answered with an HTML login page, and the probe asked Dio
// for Map<String, dynamic>. That cast runs inside the await, so it
// threw _TypeError with a null message before a single challenge
// check could execute, and the null message rendered as a network
// fault.

class _ScriptedAdapter implements HttpClientAdapter {
  _ScriptedAdapter(this.steps);

  /// One entry per expected request, in order.
  final List<ResponseBody Function()> steps;
  final List<Uri> seen = [];

  @override
  Future<ResponseBody> fetch(
    RequestOptions options,
    Stream<Uint8List>? requestStream,
    Future<void>? cancelFuture,
  ) async {
    seen.add(options.uri);
    return steps[seen.length - 1]();
  }

  @override
  void close({bool force = false}) {}
}

ResponseBody _html(int status, {String? location}) => ResponseBody.fromString(
      '<!DOCTYPE html><title>Sign in</title>',
      status,
      headers: {
        Headers.contentTypeHeader: ['text/html; charset=utf-8'],
        if (location != null) 'location': [location],
      },
    );

ResponseBody _healthJson() => ResponseBody.fromString(
      '{"status":"ok","version":"2.14.0"}',
      200,
      headers: {
        Headers.contentTypeHeader: [Headers.jsonContentType],
      },
    );

Dio Function(String, {String? cfCookie}) _factory(_ScriptedAdapter adapter) {
  return (String baseUrl, {String? cfCookie}) => buildOnboardingDio(
        baseUrl,
        cfCookie: cfCookie,
      )..httpClientAdapter = adapter;
}

void main() {
  test('an Access challenge is reported as one, not as a type error', () async {
    final adapter = _ScriptedAdapter([
      () => _html(
            302,
            location:
                'https://team.cloudflareaccess.com/cdn-cgi/access/login/od.example.com',
          ),
    ]);

    await expectLater(
      probeGateway(
        'https://od.example.com',
        dioFactory: _factory(adapter),
      ),
      throwsA(isA<AccessChallengeException>()),
    );
  });

  // The scheme half of the same report: the field used to be
  // pre-filled with "http://", so a published hostname typed after it
  // produced cleartext, and the TLS front end answered 301.
  test('http is upgraded to https and the https URL is what we keep',
      () async {
    final adapter = _ScriptedAdapter([
      () => _html(301, location: 'https://od.example.com/api/v1/health'),
      _healthJson,
    ]);

    final probe = await probeGateway(
      'http://od.example.com',
      dioFactory: _factory(adapter),
    );

    expect(probe.baseUrl, 'https://od.example.com');
    expect(probe.health.version, '2.14.0');
    expect(adapter.seen.first.scheme, 'http');
    expect(adapter.seen.last.scheme, 'https');
  });

  test('a plain LAN gateway still answers on the first try', () async {
    final adapter = _ScriptedAdapter([_healthJson]);

    final probe = await probeGateway(
      'http://192.168.3.5:8770',
      dioFactory: _factory(adapter),
    );

    expect(probe.baseUrl, 'http://192.168.3.5:8770');
    expect(adapter.seen, hasLength(1));
  });

  // A 2xx that is not JSON means the URL points at something that is
  // not opendray. Saying so beats letting a cast failure downstream
  // turn it back into "Network error".
  test('a non-JSON 200 names what actually came back', () async {
    final adapter = _ScriptedAdapter([
      () => ResponseBody.fromString(
            '<html>some other service</html>',
            200,
            headers: {
              Headers.contentTypeHeader: ['text/plain'],
            },
          ),
    ]);

    await expectLater(
      probeGateway('https://od.example.com', dioFactory: _factory(adapter)),
      throwsA(
        isA<ApiException>().having(
          (e) => e.message,
          'message',
          contains('expected JSON'),
        ),
      ),
    );
  });

  // The Secure-cookie trap: an http:// entry that gets upgraded must
  // report the https URL, because the sign-in WebView runs against it
  // and CF_Authorization is unreadable through an http:// URL.
  test('a challenge after an upgrade reports the https base URL', () async {
    final adapter = _ScriptedAdapter([
      () => _html(301, location: 'https://od.example.com/api/v1/health'),
      () => _html(
            302,
            location:
                'https://team.cloudflareaccess.com/cdn-cgi/access/login/od.example.com',
          ),
    ]);

    await expectLater(
      probeGateway('http://od.example.com', dioFactory: _factory(adapter)),
      throwsA(
        isA<AccessChallengeException>()
            .having((e) => e.baseUrl, 'baseUrl', 'https://od.example.com'),
      ),
    );
  });
}
