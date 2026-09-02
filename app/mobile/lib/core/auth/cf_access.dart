import 'dart:convert';

import 'package:dio/dio.dart';

// Cloudflare Access support.
//
// The gateway is published through a Cloudflare Tunnel and gated by
// Access, so every request from outside the LAN has to carry proof
// that the operator passed the Access identity check. Browsers get
// that for free: Access sets a `CF_Authorization` cookie on the app
// hostname after SSO and the browser replays it. A native app has no
// such flow, so we run the same SSO in a WebView, lift the resulting
// cookie out of the platform cookie store, and attach it by hand to
// every HTTP and WebSocket request.
//
// This is layered *under* opendray's own bearer auth, not instead of
// it: Access decides whether the request reaches the tunnel at all,
// opendray still decides whether the caller is signed in.

/// Name of the cookie Cloudflare Access sets after a successful login.
const kCfAccessCookieName = 'CF_Authorization';

/// Where the login flow lands once Access is satisfied. A tiny JSON
/// endpoint rather than `/`, so the WebView does not have to boot the
/// whole admin SPA just to prove the cookie works.
const kAccessProbePath = '/api/v1/health';

/// Every Access team domain ends in this. Landing here means we were
/// challenged rather than served.
const _accessHostSuffix = '.cloudflareaccess.com';

/// Treat a cookie as dead slightly early. A cookie that expires while
/// a request is in flight surfaces as a confusing HTML parse failure;
/// re-authenticating a minute early costs nothing.
const kAccessExpiryGrace = Duration(minutes: 2);

/// A live Cloudflare Access cookie.
class CfAccessSession {
  const CfAccessSession({required this.cookie, this.expiresAt});

  /// Raw `CF_Authorization` value.
  final String cookie;

  /// `exp` from the Access JWT, when it could be read. Null means the
  /// cookie is still usable — we just cannot pre-empt its expiry and
  /// have to rely on challenge detection instead.
  final DateTime? expiresAt;

  bool get isExpired {
    final exp = expiresAt;
    if (exp == null) return false;
    return DateTime.now().toUtc().isAfter(exp.subtract(kAccessExpiryGrace));
  }
}

/// Whether a response is Cloudflare Access blocking us rather than the
/// gateway answering.
///
/// Four signals, because which one shows up depends on whether Dio
/// followed the redirect and on how Access chose to answer a request
/// that asked for JSON.
bool isAccessChallengeResponse({
  required int statusCode,
  required Uri realUri,
  String? location,
  String? cfMitigated,
  String? contentType,
}) {
  // Dio followed the 302 and we are now sitting on the Access login
  // page itself.
  if (realUri.host.toLowerCase().endsWith(_accessHostSuffix)) return true;

  // A redirect we deliberately did not follow: the clients set
  // followRedirects: false precisely so this stays reachable.
  if (statusCode >= 300 && statusCode < 400) {
    final loc = (location ?? '').trim();
    if (loc.toLowerCase().contains(_accessHostSuffix)) return true;
    // Any redirect that leaves the gateway's own host was injected by
    // something sitting in front of it. opendray answers an API
    // request with data or an error, never a 3xx: the one redirect
    // the gateway serves is GET / -> /admin/, which this app never
    // requests and which is same-host anyway. Catching the general
    // case matters because Access configured with a single identity
    // provider can bounce straight past its own team domain to the
    // IdP, whose hostname matches none of the other signals.
    final target = Uri.tryParse(loc);
    if (target != null &&
        target.host.isNotEmpty &&
        target.host.toLowerCase() != realUri.host.toLowerCase()) {
      return true;
    }
  }

  // Cloudflare says it handled the request at the edge.
  if ((cfMitigated ?? '').isNotEmpty) return true;

  final isHtml = (contentType ?? '').toLowerCase().contains('text/html');

  // HTML served from Cloudflare's own path namespace: the Access
  // sign-in form, rendered on the app hostname rather than on the
  // team domain.
  if (isHtml && realUri.path.startsWith('/cdn-cgi/')) return true;

  // An auth-shaped rejection that did not come from us. opendray
  // answers 401 and 403 as JSON, always, so an HTML one is the edge
  // talking -- including a hard Access deny, which renders a page
  // rather than redirecting anywhere.
  //
  // That "always" is a real invariant of the Go side, not an
  // assumption: writeUnauth in internal/auth/auth.go and every
  // writeError helper set application/json, and internal/web only
  // serves HTML on 200. Nothing enforces it mechanically, though,
  // so a future handler answering 401/403 with an HTML body would
  // be silently swallowed here and sent to a sign-in that cannot
  // help.
  if (isHtml && (statusCode == 401 || statusCode == 403)) return true;

  // Deliberately NOT "any HTML anywhere". The gateway does serve
  // HTML on real endpoints (downloading an .html file through
  // /api/v1/fs), and treating one of those as a challenge would
  // discard the response and push the operator into a sign-in that
  // fixes nothing, because nothing was wrong with their session.
  return false;
}

/// Response-shaped wrapper over [isAccessChallengeResponse].
bool isAccessChallenge(Response<dynamic> res) {
  return isAccessChallengeResponse(
    statusCode: res.statusCode ?? 0,
    realUri: res.realUri,
    location: res.headers.value('location'),
    cfMitigated: res.headers.value('cf-mitigated'),
    contentType: res.headers.value(Headers.contentTypeHeader),
  );
}

/// Reads `exp` out of an Access JWT. Returns null for anything we
/// cannot parse — a cookie we do not understand is still worth
/// sending, we just cannot predict when it dies.
DateTime? cfAuthorizationExpiry(String jwt) {
  final parts = jwt.split('.');
  if (parts.length != 3) return null;
  try {
    final raw = utf8.decode(base64Url.decode(base64Url.normalize(parts[1])));
    final payload = jsonDecode(raw);
    if (payload is! Map) return null;
    final exp = payload['exp'];
    if (exp is! int) return null;
    return DateTime.fromMillisecondsSinceEpoch(exp * 1000, isUtc: true);
  } on Object catch (_) {
    return null;
  }
}

/// URL to load in the sign-in WebView.
///
/// Deliberately a protected resource, NOT Cloudflare's login endpoint.
/// Access owns that URL and it cannot be built by hand: it lives on
/// the team domain rather than the app's, its path carries the app
/// hostname, and it is signed with `kid` and `meta` parameters. A
/// hand-assembled /cdn-cgi/access/login on the app hostname simply
/// fails to load (ERR_HTTP_RESPONSE_CODE_FAILURE), which is what a
/// first attempt at this did.
///
/// Loading the protected resource instead lets Access issue its own
/// redirect, correctly signed, run the identity provider, and land
/// back here with the cookie set. Landing back on a small JSON
/// endpoint also means the WebView shows something harmless for the
/// instant before we read the cookie and close it.
String accessSignInUrl(String baseUrl) =>
    '${_trimSlashes(baseUrl)}$kAccessProbePath';

/// The `Cookie` header value for [cookie], or null when there is no
/// session — sending `Cookie: CF_Authorization=` would be worse than
/// sending nothing, since Access reads it as a malformed token.
String? cfCookieHeader(String? cookie) {
  if (cookie == null || cookie.isEmpty) return null;
  return '$kCfAccessCookieName=$cookie';
}

String _trimSlashes(String url) => url.replaceAll(RegExp(r'/+$'), '');
