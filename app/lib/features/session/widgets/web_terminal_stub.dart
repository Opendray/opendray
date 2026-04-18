import 'package:flutter/material.dart';

/// Stub for non-web platforms. The real implementation is in web_terminal_web.dart.
/// This file must not import any web-only packages.
typedef TerminalEventCallback = void Function(String type);

class WebTerminalView extends StatelessWidget {
  final String serverUrl;
  final String sessionId;
  final bool isRunning;
  final String? authToken;
  final TerminalEventCallback? onEvent;

  const WebTerminalView({
    super.key,
    required this.serverUrl,
    required this.sessionId,
    required this.isRunning,
    this.authToken,
    this.onEvent,
  });

  @override
  Widget build(BuildContext context) {
    // This should never render on mobile — session_page uses kIsWeb to gate.
    return const Center(
      child: Text('WebTerminalView is web-only', style: TextStyle(color: Colors.red)),
    );
  }
}
