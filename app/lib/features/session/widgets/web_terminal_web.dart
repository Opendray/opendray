import 'dart:async';
import 'dart:js_interop';
import 'dart:ui_web' as ui_web;
import 'package:flutter/material.dart';
import 'package:web/web.dart' as web;

/// Callback types for terminal events from the iframe.
typedef TerminalEventCallback = void Function(String type);

/// A web-only terminal widget that uses xterm.js via iframe.
/// The Go backend serves /terminal.html with full xterm.js setup.
class WebTerminalView extends StatefulWidget {
  final String serverUrl;
  final String sessionId;
  final bool isRunning;
  final TerminalEventCallback? onEvent;

  const WebTerminalView({
    super.key,
    required this.serverUrl,
    required this.sessionId,
    required this.isRunning,
    this.onEvent,
  });

  @override
  State<WebTerminalView> createState() => _WebTerminalViewState();
}

class _WebTerminalViewState extends State<WebTerminalView> {
  late final String _viewType;
  web.HTMLIFrameElement? _iframe;
  StreamSubscription? _messageSub;

  @override
  void initState() {
    super.initState();
    _viewType = 'ntc-terminal-${widget.sessionId.hashCode}-${DateTime.now().millisecondsSinceEpoch}';
    _registerView();
    _listenMessages();
  }

  void _registerView() {
    ui_web.platformViewRegistry.registerViewFactory(_viewType, (int viewId) {
      final iframe = web.document.createElement('iframe') as web.HTMLIFrameElement;
      iframe.style.width = '100%';
      iframe.style.height = '100%';
      iframe.style.border = 'none';
      iframe.style.backgroundColor = '#0b0d11';

      if (widget.isRunning) {
        _setIframeSrc(iframe);
      }

      _iframe = iframe;
      return iframe;
    });
  }

  void _setIframeSrc(web.HTMLIFrameElement iframe) {
    final base = widget.serverUrl;
    iframe.src = '$base/terminal.html?session=${widget.sessionId}&wsBase=$base';
  }

  void _listenMessages() {
    _messageSub = web.window.onMessage.listen((event) {
      final data = (event as web.MessageEvent).data;
      if (data == null) return;

      // Convert JSAny to map
      try {
        final type = _getProperty(data, 'type');
        if (type == null) return;

        switch (type) {
          case 'ntc:idle':
            widget.onEvent?.call('idle');
          case 'ntc:exit':
            widget.onEvent?.call('exit');
          case 'ntc:connected':
            widget.onEvent?.call('connected');
          case 'ntc:disconnected':
            widget.onEvent?.call('disconnected');
          case 'ntc:ready':
            widget.onEvent?.call('ready');
        }
      } catch (_) {}
    });
  }

  String? _getProperty(dynamic data, String key) {
    try {
      // Access JS object property
      final jsObj = data;
      return (jsObj as dynamic)?[key]?.toString();
    } catch (_) {
      return null;
    }
  }

  @override
  void didUpdateWidget(WebTerminalView oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (widget.isRunning && !oldWidget.isRunning && _iframe != null) {
      _setIframeSrc(_iframe!);
    }
  }

  /// Focus the terminal inside the iframe.
  void focus() {
    _iframe?.contentWindow?.postMessage('{"type":"ntc:focus"}'.toJS, '*'.toJS);
  }

  @override
  void dispose() {
    _messageSub?.cancel();
    _iframe?.contentWindow?.postMessage('{"type":"ntc:close"}'.toJS, '*'.toJS);
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return HtmlElementView(viewType: _viewType);
  }
}
