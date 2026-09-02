import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';

import 'package:opendray/core/auth/cf_access.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Runs the Cloudflare Access identity check in an embedded WebView
// and hands back the `CF_Authorization` cookie it produces.
//
// Why a WebView and not the system browser: the cookie is the whole
// point, and a cookie set in Safari/Chrome is not readable from the
// app. flutter_inappwebview's CookieManager reads the platform cookie
// store (Android CookieManager / WKHTTPCookieStore), which does
// return HttpOnly cookies -- the ordinary JS `document.cookie` route
// would not, because Access sets CF_Authorization HttpOnly.
//
// The address bar below the title is not decoration. The operator is
// about to type identity-provider credentials into a WebView we
// control, so they get to see exactly which host is asking.
//
// Reports the cookie through [onResult] (null when the operator backed
// out). When [onResult] is omitted it pops the route with the same
// value instead, so it works both as a pushed screen (onboarding,
// settings) and as an inline overlay (AccessGate), where there is no
// route of its own to pop.
class CfAccessLoginScreen extends StatefulWidget {
  const CfAccessLoginScreen({required this.baseUrl, this.onResult, super.key});

  /// Gateway base URL, e.g. `https://opendray.example.com`.
  final String baseUrl;

  /// Called once with the cookie, or null if the operator cancelled.
  final void Function(String? cookie)? onResult;

  static Future<String?> show(BuildContext context, String baseUrl) {
    return Navigator.of(context).push<String>(
      MaterialPageRoute<String>(
        fullscreenDialog: true,
        builder: (_) => CfAccessLoginScreen(baseUrl: baseUrl),
      ),
    );
  }

  @override
  State<CfAccessLoginScreen> createState() => _CfAccessLoginScreenState();
}

class _CfAccessLoginScreenState extends State<CfAccessLoginScreen> {
  String _currentHost = '';
  String _currentPath = '';
  bool _loading = true;
  bool _finishing = false;
  String? _error;

  late final WebUri _start = WebUri(accessSignInUrl(widget.baseUrl));
  late final String _gatewayHost = Uri.parse(widget.baseUrl).host;

  // Access can bounce through the team domain and back more than
  // once (IdP redirect, then the app's own redirect). We only look
  // for the cookie once we are back on the gateway host, and we stop
  // after the first success so a late onLoadStop cannot pop twice.
  Future<void> _tryHarvestCookie() async {
    if (_finishing || !mounted) return;
    if (_currentHost != _gatewayHost) return;
    try {
      var value = await _readCookie();
      if (value.isEmpty) {
        // Cloudflare sets the cookie on the redirect that lands us
        // here, and on some platforms the cookie store settles a
        // frame after the page reports done.
        await Future<void>.delayed(const Duration(milliseconds: 400));
        if (!mounted) return;
        value = await _readCookie();
      }
      if (value.isEmpty) {
        // Still inside Cloudflare's own pages: this is the sign-in
        // form rendered on the app hostname, not a failure.
        if (_currentPath.startsWith('/cdn-cgi/access/')) return;
        // Back on a gateway path with no cookie. The likely cause is
        // that this hostname is not behind Access at all, which
        // otherwise strands the operator on a bare 404.
        setState(() => _error = t.access.notEnabled);
        return;
      }
      _finishing = true;
      if (!mounted) return;
      _finish(value);
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _error = t.access.failed(error: '$e'));
    }
  }

  Future<String> _readCookie() async {
    final cookie = await CookieManager.instance().getCookie(
      url: WebUri(widget.baseUrl),
      name: kCfAccessCookieName,
    );
    return cookie?.value?.toString() ?? '';
  }

  void _finish(String? cookie) {
    final cb = widget.onResult;
    if (cb != null) {
      cb(cookie);
      return;
    }
    Navigator.of(context).pop(cookie);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(
        title: Text(t.access.signInTitle),
        // Explicit rather than the implied back button: as an inline
        // overlay there is no route behind us to pop to, so cancel
        // has to be a callback either way.
        leading: IconButton(
          icon: const Icon(Icons.close),
          tooltip: t.access.cancel,
          onPressed: () => _finish(null),
        ),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(24),
          child: Padding(
            padding: const EdgeInsets.fromLTRB(16, 0, 16, 6),
            child: Row(
              children: [
                Icon(
                  Icons.lock_outline,
                  size: 13,
                  color: theme.colorScheme.outline,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    _currentHost.isEmpty ? widget.baseUrl : _currentHost,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: theme.textTheme.bodySmall?.copyWith(
                      color: theme.colorScheme.outline,
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
      body: Column(
        children: [
          if (_loading) const LinearProgressIndicator(minHeight: 2),
          if (_error != null)
            Container(
              width: double.infinity,
              color: theme.colorScheme.error.withValues(alpha: 0.1),
              padding: const EdgeInsets.all(12),
              child: Text(
                _error!,
                style: TextStyle(color: theme.colorScheme.error),
              ),
            ),
          Expanded(
            // The WebView paints its own background, but only once it
            // has something to paint. Until then, and on Chrome's own
            // error pages, an unpainted WebView used to let the app's
            // dark Scaffold show through behind dark text -- the page
            // was there and completely unreadable. A light ground
            // under it matches what Cloudflare's sign-in and the
            // error pages actually draw.
            child: ColoredBox(
              color: Colors.white,
              child: InAppWebView(
                initialUrlRequest: URLRequest(url: _start),
                initialSettings: InAppWebViewSettings(
                  // Access sign-in is a plain web flow; nothing here
                  // needs to talk back to Dart.
                  javaScriptEnabled: true,
                  // No navigation interception. This does NOT affect
                  // the user-agent (we deliberately do not spoof one);
                  // it only turns off the shouldOverrideUrlLoading
                  // callback. There is nothing useful to do in it: an
                  // Access sign-in hops through whichever identity
                  // provider the operator configured, so the set of
                  // legitimate hosts cannot be enumerated ahead of
                  // time, and a guessed allowlist would break real
                  // sign-ins while stopping nothing.
                  useShouldOverrideUrlLoading: false,
                ),
                onLoadStart: (_, url) {
                  if (!mounted) return;
                  setState(() {
                    _loading = true;
                    _error = null;
                    _currentHost = url?.host ?? _currentHost;
                    _currentPath = url?.path ?? _currentPath;
                  });
                },
                onLoadStop: (_, url) async {
                  if (!mounted) return;
                  setState(() {
                    _loading = false;
                    _currentHost = url?.host ?? _currentHost;
                    _currentPath = url?.path ?? _currentPath;
                  });
                  await _tryHarvestCookie();
                },
                onReceivedError: (_, request, error) {
                  // Subresource failures are noise; only report the
                  // main document failing to load, which is what the
                  // operator can actually act on.
                  if (!(request.isForMainFrame ?? false)) return;
                  if (!mounted) return;
                  setState(() {
                    _loading = false;
                    _error = t.access.loadFailed(error: error.description);
                  });
                },
              ),
            ),
          ),
        ],
      ),
    );
  }
}
