import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/gateway_url.dart';

// Both halves of a real failure: the operator typed
// "http://opendray.example.com" into a field pre-filled with
// "http://", Cloudflare answered 301 to the https form, and the app
// reported a network error instead of either following the upgrade or
// saying which scheme it wanted.

void main() {
  group('normalizeGatewayUrl', () {
    // A published gateway is behind TLS and answers plain http only
    // with a redirect, so https is the right guess for a hostname.
    test('a bare public hostname gets https', () {
      expect(
        normalizeGatewayUrl('opendray.example.com'),
        'https://opendray.example.com',
      );
    });

    // A LAN gateway has no certificate. Guessing https here would
    // break every self-hosted install, which is the common case.
    test('a bare private address gets http', () {
      expect(normalizeGatewayUrl('192.168.3.5:8770'), 'http://192.168.3.5:8770');
      expect(normalizeGatewayUrl('10.0.0.4:8770'), 'http://10.0.0.4:8770');
      expect(normalizeGatewayUrl('172.16.5.9:8770'), 'http://172.16.5.9:8770');
      expect(normalizeGatewayUrl('127.0.0.1:8770'), 'http://127.0.0.1:8770');
      expect(normalizeGatewayUrl('localhost:8770'), 'http://localhost:8770');
      expect(normalizeGatewayUrl('nas.local:8770'), 'http://nas.local:8770');
    });

    // A public IP is not a LAN address; treat it like a hostname.
    test('a bare public IP gets https', () {
      expect(normalizeGatewayUrl('203.0.113.9'), 'https://203.0.113.9');
    });

    test('an explicit scheme is always respected', () {
      expect(
        normalizeGatewayUrl('http://opendray.example.com'),
        'http://opendray.example.com',
      );
      expect(
        normalizeGatewayUrl('https://192.168.3.5:8770'),
        'https://192.168.3.5:8770',
      );
    });

    test('trailing slashes and whitespace are trimmed', () {
      expect(
        normalizeGatewayUrl('  https://opendray.example.com//  '),
        'https://opendray.example.com',
      );
    });
  });

  group('protocolUpgradeTarget', () {
    // Cloudflare's answer to plain http. Safe to act on: same host,
    // same path, strictly more secure, and the server asked for it.
    test('same-host http to https is an upgrade', () {
      expect(
        protocolUpgradeTarget(
          from: Uri.parse('http://od.example.com/api/v1/health'),
          location: 'https://od.example.com/api/v1/health',
        ),
        'https://od.example.com/api/v1/health',
      );
    });

    test('a relative Location resolves against the request', () {
      expect(
        protocolUpgradeTarget(
          from: Uri.parse('http://od.example.com/api/v1/health'),
          location: '//od.example.com/api/v1/health',
        ),
        isNull,
      );
    });

    // The one that must never be auto-followed: this is Access (or
    // anything else) sending us somewhere new, and the whole point of
    // not following redirects is to report it.
    test('a redirect to a different host is not an upgrade', () {
      expect(
        protocolUpgradeTarget(
          from: Uri.parse('https://od.example.com/api/v1/health'),
          location: 'https://team.cloudflareaccess.com/cdn-cgi/access/login/x',
        ),
        isNull,
      );
    });

    test('a redirect to a different path is not an upgrade', () {
      expect(
        protocolUpgradeTarget(
          from: Uri.parse('http://od.example.com/api/v1/health'),
          location: 'https://od.example.com/login',
        ),
        isNull,
      );
    });

    // Downgrades and lateral moves are never followed.
    test('https to http is not an upgrade', () {
      expect(
        protocolUpgradeTarget(
          from: Uri.parse('https://od.example.com/api/v1/health'),
          location: 'http://od.example.com/api/v1/health',
        ),
        isNull,
      );
    });

    test('an empty or missing Location is not an upgrade', () {
      expect(
        protocolUpgradeTarget(
          from: Uri.parse('http://od.example.com/api/v1/health'),
          location: null,
        ),
        isNull,
      );
      expect(
        protocolUpgradeTarget(
          from: Uri.parse('http://od.example.com/api/v1/health'),
          location: '   ',
        ),
        isNull,
      );
    });
  });

  group('upgradedBaseUrl', () {
    // What onboarding persists after an upgrade: the base URL, not
    // the probe path that triggered it.
    test('strips the probe path back off', () {
      expect(
        upgradedBaseUrl(
          baseUrl: 'http://od.example.com',
          upgraded: 'https://od.example.com/api/v1/health',
        ),
        'https://od.example.com',
      );
    });

    test('keeps a port', () {
      expect(
        upgradedBaseUrl(
          baseUrl: 'http://od.example.com:8443',
          upgraded: 'https://od.example.com:8443/api/v1/health',
        ),
        'https://od.example.com:8443',
      );
    });
  });
}
