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

  Future<void> _bootstrap() async {
    try {
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
  void challenged({String? reason}) {
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
    await _storage.delete(key: SecureKeys.cfAccessCookie);
    if (baseUrl != null && baseUrl.isNotEmpty) {
      try {
        await CookieManager.instance().deleteCookie(
          url: WebUri(baseUrl),
          name: kCfAccessCookieName,
        );
      } on Object catch (_) {
        // Best effort. A cookie store we cannot reach is not a
        // reason to leave our own copy in place.
      }
    }
    state = const CfAccessIdle();
  }
}

final cfAccessControllerProvider =
    StateNotifierProvider<CfAccessController, CfAccessState>((ref) {
  return CfAccessController(ref.watch(secureStorageProvider));
});
