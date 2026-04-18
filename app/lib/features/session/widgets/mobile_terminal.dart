import 'dart:convert';
import 'package:flutter/material.dart';
import 'package:webview_flutter/webview_flutter.dart';

typedef TerminalEventCallback = void Function(String type);

/// Mobile terminal using xterm.js via WebView.
/// Same /terminal.html as web version — native-quality rendering via VS Code's terminal engine.
class MobileTerminalView extends StatefulWidget {
  final String serverUrl;
  final String sessionId;
  final bool isRunning;
  final TerminalEventCallback? onEvent;

  const MobileTerminalView({
    super.key,
    required this.serverUrl,
    required this.sessionId,
    required this.isRunning,
    this.onEvent,
  });

  @override
  State<MobileTerminalView> createState() => MobileTerminalViewState();
}

class MobileTerminalViewState extends State<MobileTerminalView> {
  late final WebViewController _controller;

  @override
  void initState() {
    super.initState();
    _controller = WebViewController()
      ..setJavaScriptMode(JavaScriptMode.unrestricted)
      ..setBackgroundColor(const Color(0xFF0B0D11))
      ..addJavaScriptChannel('OpenDray', onMessageReceived: _onMessage)
      ..setNavigationDelegate(NavigationDelegate(
        onPageFinished: (_) => _injectMessageBridge(),
      ));

    if (widget.isRunning) _loadTerminal();
  }

  /// Send raw key data (escape sequences, text) to the terminal via WebView JS.
  /// Fire-and-forget: does NOT await the JS result. The previous implementation
  /// used `runJavaScriptReturningResult()` which blocks on every keystroke
  /// waiting for the cross-process bridge round-trip — this was the #1 cause
  /// of input lag compared to RCC's native terminal.
  Future<bool> sendKey(String data) async {
    // Encode as JSON string to safely escape all special characters in one
    // step — replaces the previous 16× replaceAll chain and handles every
    // Unicode edge case correctly.
    final jsonStr = jsonEncode(data);
    try {
      _controller.runJavaScript(
        'window.openDraySendKey && window.openDraySendKey($jsonStr);',
      );
      return true;
    } catch (_) {
      return false;
    }
  }

  /// Force-reload the terminal page. Useful after the iOS image picker or
  /// similar OS-level modal has left the WebView's WebSocket in a stale
  /// state — reloading re-establishes the socket and replays buffered output.
  void reload() {
    if (widget.isRunning) _loadTerminal();
  }

  @override
  void didUpdateWidget(MobileTerminalView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.isRunning && !oldWidget.isRunning) _loadTerminal();
  }

  void _loadTerminal() {
    final base = widget.serverUrl;
    final url = '$base/terminal.html?session=${widget.sessionId}&wsBase=$base';
    _controller.loadRequest(Uri.parse(url));
  }

  /// Bridge window.postMessage → WebViewController JavaScript channel (OpenDray)
  Future<void> _injectMessageBridge() async {
    await _controller.runJavaScript('''
      window.addEventListener('message', function(ev) {
        try {
          if (ev.data && ev.data.type && ev.data.type.startsWith('opendray:')) {
            OpenDray.postMessage(JSON.stringify(ev.data));
          }
        } catch (e) {}
      });
    ''');
  }

  void _onMessage(JavaScriptMessage msg) {
    try {
      final data = jsonDecode(msg.message) as Map<String, dynamic>;
      final type = data['type'] as String?;
      if (type == null) return;
      switch (type) {
        case 'opendray:idle':
          widget.onEvent?.call('idle');
        case 'opendray:exit':
          widget.onEvent?.call('exit');
        case 'opendray:connected':
          widget.onEvent?.call('connected');
        case 'opendray:disconnected':
          widget.onEvent?.call('disconnected');
        case 'opendray:ready':
          widget.onEvent?.call('ready');
      }
    } catch (_) {}
  }

  @override
  Widget build(BuildContext context) {
    return WebViewWidget(controller: _controller);
  }
}
