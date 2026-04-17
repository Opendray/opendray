import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'ws_connect.dart';

typedef BinaryCallback = void Function(Uint8List data);
typedef ControlCallback = void Function(Map<String, dynamic> msg);
typedef VoidCallback = void Function();

class WsClient {
  final String baseUrl;
  final Map<String, String> extraHeaders;
  WebSocketChannel? _channel;
  StreamSubscription? _subscription;
  Timer? _reconnectTimer;
  Timer? _pingTimer;
  int _reconnectDelay = 1;
  String _sessionId = '';
  bool _disposed = false;

  bool isConnected = false;
  bool isReplaying = false;
  int reconnectAttempt = 0;

  BinaryCallback? onBinaryMessage;
  ControlCallback? onControlMessage;
  VoidCallback? onConnected;
  VoidCallback? onDisconnected;
  VoidCallback? onReplayStart;
  VoidCallback? onReplayEnd;
  void Function(int attempt, int delaySec)? onReconnecting;

  WsClient({required this.baseUrl, this.extraHeaders = const {}});

  void connect(String sessionId) {
    _sessionId = sessionId;
    reconnectAttempt = 0;
    _doConnect();
  }

  void _doConnect() {
    close();
    if (_disposed) return;

    final scheme = baseUrl.startsWith('https') ? 'wss' : 'ws';
    final host = baseUrl.replaceAll(RegExp(r'^https?://'), '');
    final uri = Uri.parse('$scheme://$host/api/sessions/$_sessionId/ws');

    _channel = connectWs(uri,
        headers: extraHeaders.isNotEmpty ? extraHeaders : null);

    _channel!.ready.then((_) {
      isConnected = true;
      reconnectAttempt = 0;
      _reconnectDelay = 1;
      onConnected?.call();

      _pingTimer = Timer.periodic(const Duration(seconds: 30), (_) {
        _channel?.sink.add(Uint8List(0));
      });
    }).catchError((_) {
      _scheduleReconnect();
    });

    _subscription = _channel!.stream.listen(
      (data) {
        if (data is Uint8List) {
          onBinaryMessage?.call(data);
        } else if (data is List<int>) {
          onBinaryMessage?.call(Uint8List.fromList(data));
        } else if (data is String) {
          try {
            final msg = jsonDecode(data) as Map<String, dynamic>;
            final type = msg['type'] as String?;

            if (type == 'replay_start') {
              isReplaying = true;
              onReplayStart?.call();
            } else if (type == 'replay_end') {
              isReplaying = false;
              onReplayEnd?.call();
            }

            onControlMessage?.call(msg);
          } catch (_) {}
        }
      },
      onDone: () {
        isConnected = false;
        onDisconnected?.call();
        _scheduleReconnect();
      },
      onError: (_) {
        isConnected = false;
        onDisconnected?.call();
        _scheduleReconnect();
      },
    );
  }

  void sendText(String text) {
    _channel?.sink.add(text);
  }

  void sendBinary(Uint8List data) {
    _channel?.sink.add(data);
  }

  void sendResize(int cols, int rows) {
    sendText(jsonEncode({'type': 'resize', 'cols': cols, 'rows': rows}));
  }

  void _scheduleReconnect() {
    if (_disposed || _reconnectTimer != null) return;
    reconnectAttempt++;
    final delay = _reconnectDelay;
    onReconnecting?.call(reconnectAttempt, delay);
    _reconnectTimer = Timer(Duration(seconds: delay), () {
      _reconnectTimer = null;
      _reconnectDelay = (_reconnectDelay * 2).clamp(1, 30);
      _doConnect();
    });
  }

  void close() {
    _pingTimer?.cancel();
    _pingTimer = null;
    _reconnectTimer?.cancel();
    _reconnectTimer = null;
    _subscription?.cancel();
    _subscription = null;
    _channel?.sink.close();
    _channel = null;
    isConnected = false;
    isReplaying = false;
  }

  void dispose() {
    _disposed = true;
    close();
  }
}
