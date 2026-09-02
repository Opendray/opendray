import 'dart:async';

import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import 'package:opendray/core/auth/cf_access.dart';
import 'package:opendray/core/storage/secure_store.dart';

// Holds the Cloudflare Access cookie for the whole app.
//
// Deliberately separate from AuthController: Access and opendray are
// two independent gates and either can expire without the other. A
// combined state would force a full opendray re-login every time the
// Access session rolled over, which on a 24h Access policy is daily.

/// How long the SSO sheet stays closed after the operator dismisses it.
const kAccessDismissSuppression = Duration(seconds: 30);

sealed class CfAccessState {
  const CfAccessState();
}

/// Reading secure storage.
class CfAccessLoading extends CfAccessState {
  const CfAccessLoading();
}

/// No cookie and nothing has challenged us — the normal state on the
/// LAN, where the gateway is reached directly and Access is not in
/// the path at all.
class CfAccessIdle extends CfAccessState {
  const CfAccessIdle();
}

/// We hold a cookie we believe is live.
class CfAccessReady extends CfAccessState {
  const CfAccessReady(this.session);
  final CfAccessSession session;
}

/// Access challenged us, or the cookie we had expired. The UI must
/// run the SSO WebView before anything else will work.
class CfAccessNeedsLogin extends CfAccessState {
  const CfAccessNeedsLogin({this.reason});

  /// Null when the cookie simply aged out; set when a live request
  /// was bounced, so the operator sees why the sheet appeared.
  final String? reason;
}

class CfAccessController extends StateNotifier<CfAccessState> {
  CfAccessController(this._storage) : super(const CfAccessLoading()) {
    _bootstrap();
  }

  final FlutterSecureStorage _storage;

  // Set when the operator cancels the SSO sheet. Without it, the very
  // next background poll challenges again and the sheet reappears
  // instantly -- the operator would have no way to dismiss it and
  // read the error underneath.
  DateTime? _suppressedUntil;

  // Learned from a challenge redirect or from the cookie's iss claim.
  // Persisted because sign-out needs it after the cookie is gone.
  String? _teamDomain;

  Future<void> _bootstrap() async {
    try {
      _teamDomain = await _storage.read(key: SecureKeys.cfAccessTeamDomain);
      final cookie = await _storage.read(key: SecureKeys.cfAccessCookie);
      if (cookie == null || cookie.isEmpty) {
        state = const CfAccessIdle();
        return;
      }
      final session = CfAccessSession(
        cookie: cookie,
        expiresAt: cfAuthorizationExpiry(cookie),
      );
      // An expired cookie is dropped rather than kept and retried:
      // sending it would just earn a challenge on the first request.
      if (session.isExpired) {
        await _storage.delete(key: SecureKeys.cfAccessCookie);
        state = const CfAccessNeedsLogin();
        return;
      }
      state = CfAccessReady(session);
    } on Object catch (_) {
      // Same failure mode AuthController guards against — a Keystore
      // that can no longer decrypt its own store. Falling back to
      // "no cookie" always recovers, because the SSO flow can just
      // be run again.
      state = const CfAccessIdle();
    }
  }

  /// The cookie to attach to outgoing requests, or null when we have
  /// none worth sending.
  String? get cookie => switch (state) {
        CfAccessReady(session: final s) when !s.isExpired => s.cookie,
        _ => null,
      };

  /// Stores a cookie lifted out of the SSO WebView.
  Future<void> save(String cookie) async {
    if (cookie.isEmpty) return;
    _suppressedUntil = null;
    await _rememberTeamDomain(teamDomainFromJwt(cookie));
    await _storage.write(key: SecureKeys.cfAccessCookie, value: cookie);
    state = CfAccessReady(
      CfAccessSession(cookie: cookie, expiresAt: cfAuthorizationExpiry(cookie)),
    );
  }

  /// Called from the HTTP layer when Access bounced a request.
  ///
  /// Idempotent on purpose: a screen that fires six parallel requests
  /// gets six challenges, and each one must not re-trigger the login
  /// sheet on top of itself.
  void challenged({String? reason, String? teamDomain}) {
    if (teamDomain != null) unawaited(_rememberTeamDomain(teamDomain));
    if (state is CfAccessNeedsLogin) return;
    final until = _suppressedUntil;
    if (until != null && DateTime.now().isBefore(until)) return;
    state = CfAccessNeedsLogin(reason: reason);
  }

  /// The operator closed the SSO sheet without finishing. Requests
  /// keep failing (and say why), but we stop re-opening the sheet for
  /// [kAccessDismissSuppression].
  void dismiss() {
    _suppressedUntil = DateTime.now().add(kAccessDismissSuppression);
    state = const CfAccessIdle();
  }

  /// Forgets the cookie, in both places it lives.
  ///
  /// Deleting only our secure-storage copy would not do what the
  /// operator asked: the same cookie is also in the platform cookie
  /// store (Android CookieManager / WKHTTPCookieStore), where the
  /// SSO WebView put it. Leaving that behind means the next sign-in
  /// silently succeeds off the surviving cookie with no identity
  /// check at all, and switching gateways strands a live cookie for
  /// the old host that no app state knows about.
  ///
  /// Reaching for the WebView's cookie store from a controller is
  /// the price of the cookie having two homes; the alternative is
  /// every call site remembering to clear the second one.
  ///
  /// [baseUrl] is the gateway the cookie belongs to. It is optional
  /// only because a caller may no longer know it, in which case the
  /// platform copy is left alone rather than guessed at.
  Future<void> clear({String? baseUrl}) async {
    // Order matters. Deleting cookies first would leave the logout
    // requests unauthenticated, and Cloudflare answers those with an
    // error page instead of ending anything.
    //
    // The team domain goes first because it is the half that decides
    // whether the next sign-in has to involve the identity provider
    // at all. Clearing only the app domain's CF_Authorization looked
    // like it worked and wasn't: Access still recognised the device
    // on the team domain and re-issued a cookie without asking for
    // anything, so "clear" appeared to do nothing.
    final team = _teamDomain;
    if (team != null && team.isNotEmpty) {
      await _visitLogout(accessLogoutUrl(team));
    }
    if (baseUrl != null && baseUrl.isNotEmpty) {
      await _visitLogout(accessLogoutUrl(baseUrl));
      try {
        // Sweep whatever the logout did not take. CF_AppSession lives
        // here too, alongside CF_Authorization, and deleting one
        // named cookie left the other behind.
        await CookieManager.instance().deleteCookies(url: WebUri(baseUrl));
      } on Object catch (_) {
        // Best effort: a cookie store we cannot reach is not a reason
        // to keep our own copy.
      }
    }
    await _storage.delete(key: SecureKeys.cfAccessCookie);
    state = const CfAccessIdle();
  }

  /// Loads a Cloudflare logout URL in an offscreen WebView.
  ///
  /// It has to be a WebView, not an HTTP call: the session being
  /// ended is the WebView cookie jar's, and only a request made from
  /// there carries the cookies and receives the clearing Set-Cookie
  /// headers back.
  Future<void> _visitLogout(String url) async {
    final done = Completer<void>();
    HeadlessInAppWebView? web;
    try {
      web = HeadlessInAppWebView(
        initialUrlRequest: URLRequest(url: WebUri(url)),
        onLoadStop: (_, __) {
          if (!done.isCompleted) done.complete();
        },
        onReceivedError: (_, __, ___) {
          if (!done.isCompleted) done.complete();
        },
      );
      await web.run();
      // Bounded: sign-out must not hang the settings screen because
      // the gateway is unreachable. A logout we could not deliver
      // still gets the local cookies wiped by the caller.
      await done.future.timeout(const Duration(seconds: 10));
    } on Object catch (_) {
      // Best effort by design.
    } finally {
      await web?.dispose();
    }
  }

  Future<void> _rememberTeamDomain(String? domain) async {
    if (domain == null || domain.isEmpty || domain == _teamDomain) return;
    _teamDomain = domain;
    await _storage.write(
      key: SecureKeys.cfAccessTeamDomain,
      value: domain,
    );
  }
}

final cfAccessControllerProvider =
    StateNotifierProvider<CfAccessController, CfAccessState>((ref) {
  return CfAccessController(ref.watch(secureStorageProvider));
});
