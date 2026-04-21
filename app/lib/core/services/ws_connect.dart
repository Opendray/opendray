import 'package:web_socket_channel/web_socket_channel.dart';

/// Cross-platform WebSocket connector.
///
/// `WebSocketChannel.connect` picks the right underlying implementation
/// automatically: `dart:io` on mobile/desktop, browser `WebSocket` on
/// web. The previous `IOWebSocketChannel` hard-coded the dart:io path,
/// which throws `Unsupported operation: Platform._version` the moment
/// a WS route is touched in a browser (logs tail, tasks live stream).
///
/// Note: custom HTTP headers (e.g. `Authorization: Bearer <jwt>`) are
/// not supported by the browser WebSocket API. All callers here rely
/// on token-in-URL auth (see terminal.html, logs endpoint), so the
/// `headers` parameter is intentionally dropped.
WebSocketChannel connectWs(Uri uri) {
  return WebSocketChannel.connect(uri);
}
