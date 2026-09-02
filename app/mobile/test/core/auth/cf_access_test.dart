import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/auth/cf_access.dart';

String _jwt(Map<String, dynamic> payload) {
  String seg(Map<String, dynamic> m) =>
      base64Url.encode(utf8.encode(jsonEncode(m))).replaceAll('=', '');
  return '${seg({'alg': 'RS256'})}.${seg(payload)}.sig';
}

void main() {
  group('isAccessChallengeResponse', () {
    // The common case: Dio follows Access's 302 and lands on the team
    // login page, which answers 200 + HTML. Without this the caller
    // sees "200 OK" and then fails to parse a login page as JSON.
    test('final URI on the Access team domain is a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 200,
          realUri: Uri.parse(
            'https://acme.cloudflareaccess.com/cdn-cgi/access/login/od.example.com',
          ),
          contentType: 'text/html; charset=utf-8',
        ),
        isTrue,
      );
    });

    // The live failure this was found through: Access with a single
    // identity provider bounces past its own team domain straight to
    // the IdP. With followRedirects on, Dio chased that chain and
    // came back holding Google's HTML login page from a host that
    // matched no signal at all, which then failed to parse as JSON
    // and surfaced as "Server replied 0: Network error".
    test('a redirect off the gateway host is a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 302,
          realUri: Uri.parse('https://od.example.com/api/v1/health'),
          location: 'https://accounts.google.com/o/oauth2/v2/auth?foo=bar',
        ),
        isTrue,
      );
    });

    // A same-host redirect is the gateway's own (GET / -> /admin/),
    // or a proxy normalising a path. Not a challenge, and treating it
    // as one would pop a sign-in that fixes nothing.
    test('a same-host redirect is not a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 302,
          realUri: Uri.parse('https://od.example.com/'),
          location: 'https://od.example.com/admin/',
        ),
        isFalse,
      );
    });

    test('a relative same-host redirect is not a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 302,
          realUri: Uri.parse('https://od.example.com/'),
          location: '/admin/',
        ),
        isFalse,
      );
    });

    test('redirect to the Access domain is a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 302,
          realUri: Uri.parse('https://od.example.com/api/v1/health'),
          location:
              'https://acme.cloudflareaccess.com/cdn-cgi/access/login/od.example.com',
        ),
        isTrue,
      );
    });

    test('cf-mitigated header is a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 403,
          realUri: Uri.parse('https://od.example.com/api/v1/health'),
          cfMitigated: 'challenge',
        ),
        isTrue,
      );
    });

    // Access can render its sign-in form on the app hostname instead
    // of bouncing to the team domain, in which case the only tell is
    // the Cloudflare-owned path.
    test('HTML under /cdn-cgi/ is a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 200,
          realUri: Uri.parse(
            'https://od.example.com/cdn-cgi/access/login/od.example.com',
          ),
          contentType: 'text/html',
        ),
        isTrue,
      );
    });

    // A hard-deny Access policy renders a page rather than
    // redirecting, so none of the redirect signals fire. opendray
    // answers 401/403 as JSON, so an HTML one is not ours.
    test('an HTML 403 is a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 403,
          realUri: Uri.parse('https://od.example.com/api/v1/sessions'),
          contentType: 'text/html',
        ),
        isTrue,
      );
    });

    // The false positive that would break a real feature: the
    // gateway serves the file you asked it for, and it happens to be
    // HTML. Discarding that and demanding a sign-in would be wrong
    // twice over, since the Access session was never the problem.
    test('a downloaded HTML file is not a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 200,
          realUri: Uri.parse(
            'https://od.example.com/api/v1/fs/download?path=notes.html',
          ),
          contentType: 'text/html; charset=utf-8',
        ),
        isFalse,
      );
    });

    test('a normal JSON answer is not a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 200,
          realUri: Uri.parse('https://od.example.com/api/v1/health'),
          contentType: 'application/json',
        ),
        isFalse,
      );
    });

    // A 401 from opendray itself must stay a 401 — routing it to the
    // Access login flow would hide an expired bearer token behind a
    // WebView that immediately succeeds and changes nothing.
    test('opendray 401 JSON is not a challenge', () {
      expect(
        isAccessChallengeResponse(
          statusCode: 401,
          realUri: Uri.parse('https://od.example.com/api/v1/sessions'),
          contentType: 'application/json',
        ),
        isFalse,
      );
    });
  });

  group('cfAuthorizationExpiry', () {
    test('reads exp out of the Access JWT', () {
      final exp = DateTime.utc(2030, 1, 2, 3, 4, 5);
      final got = cfAuthorizationExpiry(
        _jwt({'exp': exp.millisecondsSinceEpoch ~/ 1000}),
      );
      expect(got, exp);
    });

    test('returns null for junk rather than throwing', () {
      expect(cfAuthorizationExpiry('not-a-jwt'), isNull);
      expect(cfAuthorizationExpiry('a.b.c'), isNull);
      expect(cfAuthorizationExpiry(''), isNull);
      expect(cfAuthorizationExpiry(_jwt({'no': 'exp'})), isNull);
    });
  });

  group('CfAccessSession', () {
    test('an unexpired cookie is usable', () {
      final s = CfAccessSession(
        cookie: 'abc',
        expiresAt: DateTime.now().toUtc().add(const Duration(hours: 4)),
      );
      expect(s.isExpired, isFalse);
    });

    test('expiry grace treats a nearly-dead cookie as expired', () {
      final s = CfAccessSession(
        cookie: 'abc',
        expiresAt: DateTime.now().toUtc().add(const Duration(seconds: 30)),
      );
      expect(s.isExpired, isTrue);
    });

    // Access cookies without a readable exp still work; we just cannot
    // pre-empt their expiry, so we rely on the challenge detection.
    test('a cookie with no expiry is not treated as expired', () {
      expect(const CfAccessSession(cookie: 'abc').isExpired, isFalse);
    });
  });

  group('accessSignInUrl', () {
    // Must be a protected resource, never a hand-built
    // /cdn-cgi/access/login. Access's real login URL lives on the
    // team domain, carries the app hostname in its path, and is
    // signed with kid/meta parameters; assembling one by hand
    // produced ERR_HTTP_RESPONSE_CODE_FAILURE on a real device.
    test('loads a protected resource so Access issues its own redirect', () {
      final u = Uri.parse(accessSignInUrl('https://od.example.com'));
      expect(u.host, 'od.example.com');
      expect(u.path, kAccessProbePath);
      expect(u.path, isNot(contains('cdn-cgi')));
      expect(u.hasQuery, isFalse);
    });

    test('tolerates a trailing slash on the base URL', () {
      expect(
        accessSignInUrl('https://od.example.com/'),
        accessSignInUrl('https://od.example.com'),
      );
    });
  });

  group('cfCookieHeader', () {
    test('renders the cookie pair', () {
      expect(cfCookieHeader('tok'), 'CF_Authorization=tok');
    });

    test('is null when there is no session, so no empty header is sent', () {
      expect(cfCookieHeader(null), isNull);
      expect(cfCookieHeader(''), isNull);
    });
  });
}
