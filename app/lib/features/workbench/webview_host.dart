import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

import 'plugin_bridge_channel.dart';

/// Factory signature used by tests to inject a fake bridge channel.
typedef PluginBridgeChannelFactory = PluginBridgeChannel Function(
  Uri url,
  String bearerToken,
);

/// PluginWebView renders one plugin view inside an embedded WebView,
/// wired to the plugin's bridge WebSocket via an injected JS shim
/// (see [pluginPreloadShim]).
///
/// Per-plugin isolation (M2-PLAN §T16): every instance creates a fresh
/// [WebViewController]; asset requests land on
/// `/api/plugins/{name}/assets/*` with the user's bearer token as a
/// query param (Android WebView can't carry headers for subresources,
/// which is the explicit M2 design decision documented in the plan).
///
/// Platform matrix:
/// - Android / iOS — supported in T16.
/// - Web — iframe src pointing at the asset URL (webview_flutter_web).
/// - Desktop — **NOT supported in T16**; see T16b for the fallback plan.
class PluginWebView extends StatefulWidget {
  const PluginWebView({
    super.key,
    required this.pluginName,
    required this.viewId,
    required this.entryPath,
    required this.baseUrl,
    required this.bearerToken,
    this.onMessage,
    PluginBridgeChannelFactory? bridgeFactory,
    @visibleForTesting bool skipControllerForTests = false,
  })  : _bridgeFactory = bridgeFactory,
        _skipControllerForTests = skipControllerForTests;

  /// Plugin name — e.g. `"kanban"`. Drives both the asset path and the
  /// bridge WebSocket URL.
  final String pluginName;

  /// Logical view id from `contributes.views[].id`. Used only for
  /// diagnostics / debug logging at this layer.
  final String viewId;

  /// Entry path inside the plugin bundle — e.g. `"index.html"`.
  final String entryPath;

  /// Gateway base URL (scheme + host + port). Must be an `http://` or
  /// `https://` URL; the bridge WS URL is derived by swapping the scheme.
  final String baseUrl;

  /// JWT bearer token for the current user. Sent as the `?token=` query
  /// param on both the asset URL and the bridge WS URL.
  final String bearerToken;

  /// Optional callback fired for any envelope the JS shim posts that
  /// does **not** match a pending round-trip id. Hosts can use this to
  /// surface plugin-initiated messages (future: push notifications).
  final void Function(Map<String, dynamic> envelope)? onMessage;

  final PluginBridgeChannelFactory? _bridgeFactory;
  final bool _skipControllerForTests;

  @override
  State<PluginWebView> createState() => _PluginWebViewState();
}

class _PluginWebViewState extends State<PluginWebView> {
  static const _channelName = 'OpenDrayBridge';

  WebViewController? _controller;
  PluginBridgeChannel? _bridge;
  bool _shimInjected = false;
  Object? _fatalError;

  @override
  void initState() {
    super.initState();
    _setup();
  }

  void _setup() {
    final bridgeUrl = _buildBridgeUrl(widget.baseUrl, widget.pluginName);
    _bridge = widget._bridgeFactory != null
        ? widget._bridgeFactory!(bridgeUrl, widget.bearerToken)
        : PluginBridgeChannel(url: bridgeUrl, bearerToken: widget.bearerToken);

    if (widget._skipControllerForTests) {
      // Tests exercise the bridge+lifecycle wiring without needing a
      // platform WebView; leave _controller null so build() renders a
      // placeholder and no controller method is ever invoked.
      return;
    }

    final controller = _buildController();
    _controller = controller;

    final assetUrl = _buildAssetUrl(
      widget.baseUrl,
      widget.pluginName,
      widget.entryPath,
      widget.bearerToken,
    );
    try {
      controller.loadRequest(assetUrl);
    } catch (e) {
      // Rare: e.g. desktop platforms without a WebView implementation.
      setState(() => _fatalError = e);
    }
  }

  WebViewController _buildController() {
    final controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..addJavaScriptChannel(_channelName, onMessageReceived: _onJsMessage)
      ..setNavigationDelegate(
        NavigationDelegate(
          onPageFinished: (_) => _injectShim(),
        ),
      );
    return controller;
  }

  Future<void> _injectShim() async {
    final controller = _controller;
    if (controller == null || _shimInjected) return;
    _shimInjected = true;
    try {
      await controller.runJavaScript(pluginPreloadShim);
    } catch (e) {
      if (kDebugMode) {
        debugPrint('PluginWebView: shim injection failed: $e');
      }
    }
  }

  void _onJsMessage(JavaScriptMessage message) {
    final raw = message.message;
    Map<String, dynamic> env;
    try {
      final decoded = jsonDecode(raw);
      if (decoded is! Map) return;
      env = decoded.cast<String, dynamic>();
    } catch (_) {
      return;
    }
    _dispatchOutgoing(env);
  }

  void _dispatchOutgoing(Map<String, dynamic> env) {
    final bridge = _bridge;
    if (bridge == null) return;

    final id = env['id'];
    final ns = env['ns'];
    final method = env['method'];
    final rawArgs = env['args'];
    if (id is! String || ns is! String || method is! String) {
      widget.onMessage?.call(env);
      return;
    }
    final args = rawArgs is List ? List<dynamic>.from(rawArgs) : const <dynamic>[];

    // M2 T16 ships the round-trip path; events.subscribe (streaming)
    // will be finalised in T20. We still hand streaming calls to the
    // bridge so the JS shim sees the same response shape.
    final isSubscribe = ns == 'events' && method == 'subscribe';
    if (isSubscribe) {
      final stream = bridge.subscribe(ns, method, args);
      stream.listen(
        (data) => _forwardStreamChunk(id, data),
        onError: (Object err) => _forwardError(id, err),
        onDone: () => _forwardStreamEnd(id),
      );
      return;
    }

    bridge.call(ns, method, args).then(
      (result) => _forwardResult(id, result),
      onError: (Object err, StackTrace _) => _forwardError(id, err),
    );
  }

  Future<void> _forwardResult(String id, dynamic result) async {
    await _postToShim(<String, dynamic>{'v': 1, 'id': id, 'result': result});
  }

  Future<void> _forwardError(String id, Object err) async {
    final (code, message) = _describeError(err);
    await _postToShim(<String, dynamic>{
      'v': 1,
      'id': id,
      'error': <String, dynamic>{'code': code, 'message': message},
    });
  }

  Future<void> _forwardStreamChunk(String id, dynamic data) async {
    await _postToShim(<String, dynamic>{
      'v': 1,
      'id': id,
      'stream': 'chunk',
      'data': data,
    });
  }

  Future<void> _forwardStreamEnd(String id) async {
    await _postToShim(<String, dynamic>{'v': 1, 'id': id, 'stream': 'end'});
  }

  Future<void> _postToShim(Map<String, dynamic> envelope) async {
    final controller = _controller;
    if (controller == null) return;
    final payload = jsonEncode(jsonEncode(envelope));
    try {
      await controller.runJavaScript(
        'window.__opendray_onMessage && window.__opendray_onMessage($payload);',
      );
    } catch (e) {
      if (kDebugMode) {
        debugPrint('PluginWebView: postToShim failed: $e');
      }
    }
  }

  (String, String) _describeError(Object err) {
    if (err is BridgeException) return (err.code, err.message);
    return ('EINTERNAL', err.toString());
  }

  @override
  void dispose() {
    final bridge = _bridge;
    _bridge = null;
    _controller = null;
    if (bridge != null) {
      // Fire-and-forget — dispose never throws.
      unawaited(bridge.dispose());
    }
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    if (_fatalError != null) {
      return _ErrorPlaceholder(error: _fatalError!);
    }
    final controller = _controller;
    if (controller == null) {
      // Either still constructing, or running in tests where we skip
      // real WebView construction. A loading indicator covers both.
      return const _LoadingPlaceholder();
    }
    return WebViewWidget(controller: controller);
  }
}

class _LoadingPlaceholder extends StatelessWidget {
  const _LoadingPlaceholder();

  @override
  Widget build(BuildContext context) {
    return const Center(child: CircularProgressIndicator());
  }
}

class _ErrorPlaceholder extends StatelessWidget {
  const _ErrorPlaceholder({required this.error});
  final Object error;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Text(
          'Plugin view unavailable: $error',
          textAlign: TextAlign.center,
        ),
      ),
    );
  }
}

/// Builds the plugin asset URL — e.g.
/// `https://host/api/plugins/kanban/assets/index.html?token=...`.
@visibleForTesting
Uri buildPluginAssetUrl(
  String baseUrl,
  String pluginName,
  String entryPath,
  String bearerToken,
) =>
    _buildAssetUrl(baseUrl, pluginName, entryPath, bearerToken);

/// Builds the plugin bridge WebSocket URL — e.g.
/// `ws://host/api/plugins/kanban/bridge/ws`.
@visibleForTesting
Uri buildPluginBridgeUrl(String baseUrl, String pluginName) =>
    _buildBridgeUrl(baseUrl, pluginName);

Uri _buildAssetUrl(
  String baseUrl,
  String pluginName,
  String entryPath,
  String bearerToken,
) {
  final trimmed = baseUrl.endsWith('/')
      ? baseUrl.substring(0, baseUrl.length - 1)
      : baseUrl;
  final entry = entryPath.startsWith('/') ? entryPath.substring(1) : entryPath;
  final uri = Uri.parse('$trimmed/api/plugins/$pluginName/assets/$entry');
  if (bearerToken.isEmpty) return uri;
  final qp = Map<String, String>.from(uri.queryParameters);
  qp['token'] = bearerToken;
  return uri.replace(queryParameters: qp);
}

Uri _buildBridgeUrl(String baseUrl, String pluginName) {
  final uri = Uri.parse(baseUrl);
  final wsScheme = uri.scheme == 'https' ? 'wss' : 'ws';
  return uri.replace(
    scheme: wsScheme,
    path: '/api/plugins/$pluginName/bridge/ws',
    query: null,
  );
}

/// The preload shim from M2-PLAN §10 (extended in T20 for streams).
/// Injected into every plugin WebView on `onPageFinished`; exposes
/// `window.opendray` with `workbench`, `storage`, and `events`.
///
/// Events namespace notes (T20):
///  * `events.subscribe(name, cb)` posts args as a JSON object
///    `{name}` (NOT an array) — the Go side unmarshals into
///    `plugin/bridge/api_events.go` parses [name]. Same array shape for
///    unsubscribe ([subId]) and publish ([name, data]) — matches the
///    workbench/storage namespaces so the Dart bridge channel doesn't
///    need per-namespace arg marshalling.
///  * Chunk envelopes (`stream:"chunk"`) call the subscriber with
///    `cb(data)`. The stream-end envelope (`stream:"end"`) silently
///    removes the sub from the map; if the end carries an `error` we
///    surface it via `console.warn` (no error path to the plugin cb
///    since `subscribe` returns an unsubscribe fn, not a Promise).
///  * `unsubscribe()` (the returned cleanup fn) posts a normal
///    round-trip `events.unsubscribe` call and deletes the local sub
///    entry eagerly so late chunks after cleanup are dropped.
const String pluginPreloadShim = r'''
(() => {
  const calls = new Map();
  const streams = new Map();
  let nextId = 1;
  function call(ns, method, args) {
    const id = String(nextId++);
    return new Promise((resolve, reject) => {
      calls.set(id, { resolve, reject });
      window.OpenDrayBridge.postMessage(JSON.stringify({ v: 1, id, ns, method, args }));
    });
  }
  window.__opendray_onMessage = (raw) => {
    const env = JSON.parse(raw);
    if (env.stream === "chunk") { const cb = streams.get(env.id); if (cb) cb(env.data); return; }
    if (env.stream === "end")   { streams.delete(env.id); if (env.error) console.warn("opendray events stream ended with error:", env.error); return; }
    const pending = calls.get(env.id);
    if (!pending) return;
    calls.delete(env.id);
    if (env.error) pending.reject(new Error(`${env.error.code}: ${env.error.message}`));
    else pending.resolve(env.result);
  };
  const nsProxy = (ns, methods) => Object.fromEntries(
    methods.map(m => [m, (...args) => call(ns, m, args)]));
  function subscribe(name, cb) {
    const id = String(nextId++);
    streams.set(id, cb);
    window.OpenDrayBridge.postMessage(JSON.stringify({ v: 1, id, ns: "events", method: "subscribe", args: [name] }));
    return () => { streams.delete(id); call("events", "unsubscribe", [id]).catch(() => {}); };
  }
  window.opendray = {
    version: "1",
    plugin: window.__opendray_plugin_ctx || {},
    workbench: nsProxy("workbench", ["showMessage","openView","updateStatusBar","runCommand","theme","onThemeChange"]),
    storage:   nsProxy("storage",   ["get","set","delete","list"]),
    events:    { subscribe, publish: (name, data) => call("events","publish",[name, data]) },
  };
})();
''';
