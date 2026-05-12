// Wraps /api/v1/admin/logs/* — the gateway's in-process log ring
// buffer that the web admin's "Logging" section displays via WS
// live-tail. Mobile uses the same surface (no separate API),
// so this file is intentionally narrow:
//
//   * LogRecord — JSON shape mirroring applog.Record (the Go side).
//   * streamLogs(...) — opens a WebSocket against the gateway and
//     decodes records as they arrive.
//
// The tail endpoint (REST snapshot) isn't wired here because the
// WS replays the in-memory ring on connect; one channel is enough.

import 'dart:async';
import 'dart:convert';

import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

class LogRecord {
  LogRecord({
    required this.time,
    required this.level,
    required this.message,
    required this.text,
    this.attrs,
  });

  factory LogRecord.fromJson(Map<String, dynamic> json) => LogRecord(
        time: json['time'] as String? ?? '',
        // Server emits uppercase strings (DEBUG/INFO/WARN/ERROR).
        level: (json['level'] as String? ?? 'INFO').toUpperCase(),
        message: json['message'] as String? ?? '',
        text: json['text'] as String? ?? '',
        attrs: json['attrs'] is Map<String, dynamic>
            ? json['attrs'] as Map<String, dynamic>
            : null,
      );

  // ISO timestamp from the server (RFC3339Nano).
  final String time;
  // "DEBUG" | "INFO" | "WARN" | "ERROR" — we don't enum it so a
  // future level (e.g. TRACE) renders rather than crashes.
  final String level;
  // Operator-readable log message (the slog "msg" key).
  final String message;
  // Pre-rendered single-line representation the server already
  // formatted. Falling back to this means the row always has
  // *something* to display even when message / attrs are empty.
  final String text;
  // Structured attributes from the slog call (e.g. session_id,
  // err, count). Rendered as a faint suffix on each row.
  final Map<String, dynamic>? attrs;
}

// LogsStream wraps the WebSocket so callers get a typed
// Stream<LogRecord> + an explicit close. Reuses the
// web_socket_channel package the terminal view already pulls in
// — no new dependency.
class LogsStream {
  LogsStream._(this._channel);

  final WebSocketChannel _channel;
  StreamSubscription<dynamic>? _sub;
  // Why a separate broadcast controller: the WS channel exposes a
  // single-subscription stream; we want the UI to be able to
  // attach/detach without tearing the WS down (e.g. on hot
  // reload). The controller buffers nothing — drops to subscribers.
  final _ctrl = StreamController<LogRecord>.broadcast();

  // Static rather than a factory ctor: we want to construct the
  // LogsStream first and then mutate _sub after the WS subscription
  // is attached, which factory ctors don't permit cleanly.
  // ignore: prefer_constructors_over_static_methods
  static LogsStream open({required String serverUrl, required String token}) {
    final base = serverUrl.replaceAll(RegExp(r'/+$'), '');
    final wsBase = base.startsWith('https')
        ? base.replaceFirst('https', 'wss')
        : base.replaceFirst('http', 'ws');
    final url = Uri.parse(
      '$wsBase/api/v1/admin/logs/stream?token=${Uri.encodeQueryComponent(token)}',
    );
    final channel = WebSocketChannel.connect(url);
    final s = LogsStream._(channel);
    s._sub = channel.stream.listen(
      s._onMessage,
      onError: (Object err) {
        if (!s._ctrl.isClosed) s._ctrl.addError(err);
      },
      onDone: () {
        if (!s._ctrl.isClosed) s._ctrl.close();
      },
      cancelOnError: false,
    );
    return s;
  }

  Stream<LogRecord> get stream => _ctrl.stream;

  void _onMessage(Object? msg) {
    if (msg is! String) return;
    try {
      final decoded = jsonDecode(msg);
      if (decoded is! Map<String, dynamic>) return;
      _ctrl.add(LogRecord.fromJson(decoded));
    } on FormatException {
      // Server should only emit valid JSON — drop anything else.
    }
  }

  Future<void> close() async {
    await _sub?.cancel();
    try {
      await _channel.sink.close();
    } on Object {
      // ignore — already closed
    }
    if (!_ctrl.isClosed) await _ctrl.close();
  }
}

/// Convenience opener: pulls the server URL + token out of
/// AuthState and returns null when the operator isn't signed in.
LogsStream? openLogsStream(WidgetRef ref) {
  final auth = ref.read(authControllerProvider);
  if (auth is! AuthLoggedIn) return null;
  return LogsStream.open(serverUrl: auth.serverUrl, token: auth.token);
}
