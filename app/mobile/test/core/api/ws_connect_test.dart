import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/ws_connect.dart';

// The regression this guards: the bearer token used to travel in the
// WebSocket query string, which put it in the access log of every
// proxy on the path. Once the gateway is published through Cloudflare
// that log belongs to someone else.

void main() {
  group('gatewayWsUri', () {
    test('https becomes wss', () {
      expect(
        gatewayWsUri(
          serverUrl: 'https://od.example.com',
          path: '/api/v1/sessions/abc/stream',
        ).toString(),
        'wss://od.example.com/api/v1/sessions/abc/stream',
      );
    });

    test('http becomes ws for a LAN gateway', () {
      expect(
        gatewayWsUri(
          serverUrl: 'http://192.168.3.5:8770',
          path: '/api/v1/admin/logs/stream',
        ).toString(),
        'ws://192.168.3.5:8770/api/v1/admin/logs/stream',
      );
    });

    test('trailing slashes do not produce a doubled path', () {
      expect(
        gatewayWsUri(
          serverUrl: 'https://od.example.com//',
          path: '/api/v1/health',
        ).toString(),
        'wss://od.example.com/api/v1/health',
      );
    });

    test('carries no query string at all', () {
      final uri = gatewayWsUri(
        serverUrl: 'https://od.example.com',
        path: '/api/v1/sessions/abc/stream',
      );
      expect(uri.hasQuery, isFalse);
      expect(uri.toString(), isNot(contains('token')));
    });
  });

  group('gatewayWsHeaders', () {
    test('the bearer travels in a header', () {
      expect(
        gatewayWsHeaders(token: 'secret'),
        containsPair('Authorization', 'Bearer secret'),
      );
    });

    test('the Access cookie rides along when we hold one', () {
      expect(
        gatewayWsHeaders(token: 'secret', cfCookie: 'jwt'),
        containsPair('Cookie', 'CF_Authorization=jwt'),
      );
    });

    test('no Cookie header on a LAN gateway', () {
      expect(gatewayWsHeaders(token: 'secret').containsKey('Cookie'), isFalse);
    });

    // The gateway's CSWSH guard treats a missing Origin as a
    // non-browser client and allows it. Sending one would make us
    // look like a page on some other site.
    test('no Origin header', () {
      expect(gatewayWsHeaders(token: 't').containsKey('Origin'), isFalse);
    });
  });

  group('reconnectBackoff', () {
    // The budget has to actually be spendable. It was not before:
    // the attempt counter was reset every time a socket object was
    // created, which is before the handshake is known to have been
    // accepted, so a gateway rejecting every connection produced an
    // unbounded retry loop rather than five tries and a message.
    test('backs off exponentially from 500ms', () {
      expect(
        List.generate(5, (i) => reconnectBackoff(i + 1).inMilliseconds),
        [500, 1000, 2000, 4000, 8000],
      );
    });

    test('the whole budget spans a bounded, short window', () {
      final total = List.generate(
        kMaxReconnectAttempts,
        (i) => reconnectBackoff(i + 1),
      ).fold(Duration.zero, (a, b) => a + b);
      expect(total, lessThan(const Duration(seconds: 20)));
    });

    test('a zero or negative attempt is treated as the first', () {
      expect(reconnectBackoff(0), reconnectBackoff(1));
      expect(reconnectBackoff(-3), reconnectBackoff(1));
    });
  });
}
