import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// PluginBridgeChannel owns a single WebSocket to a plugin's bridge
/// endpoint. Handles envelope encoding + correlation-id tracking per
/// M2-PLAN §11 ("Bridge WS protocol — concrete semantics").
///
/// Lifecycle:
/// ```
///   final ch = PluginBridgeChannel(url: u, bearerToken: t);
///   await ch.ready;
///   final result = await ch.call('workbench', 'showMessage', ['hi']);
///   await ch.dispose();
/// ```
///
/// Streams: [subscribe] returns a Stream that closes when the server
/// sends `{stream:"end"}` (or on error envelope tied to the sub id).
class PluginBridgeChannel {
  /// Production constructor — opens an [IOWebSocketChannel] to [url]
  /// with a bearer token carried as a `?token=` query param. Flutter's
  /// web target can't set `Authorization` on a WS handshake, so the
  /// token ride-along matches the rest of the app (see `ws_client.dart`).
  PluginBridgeChannel({
    required this.url,
    required this.bearerToken,
  })  : _ownsChannel = true,
        _channel = null {
    _connect();
  }

  /// Test-only constructor. Accepts a pre-built [WebSocketChannel] so
  /// widget/unit tests can pipe scripted frames through a fake without
  /// a real socket. The channel is considered ready as soon as it is
  /// passed in.
  PluginBridgeChannel.withChannel(
    WebSocketChannel channel, {
    this.bearerToken = '',
  })  : url = Uri.parse('ws://test.invalid/'),
        _ownsChannel = false,
        _channel = channel {
    _bind(channel);
    _readyCompleter.complete();
  }

  /// Test-only constructor that accepts a raw stream/sink pair.
  /// Lets tests avoid mocking [WebSocketChannel] (which extends
  /// StreamChannelMixin and demands a wide surface).
  @visibleForTesting
  PluginBridgeChannel.withStreams({
    required Stream<dynamic> incoming,
    required StreamSink<dynamic> outgoing,
  })  : url = Uri.parse('ws://test.invalid/'),
        bearerToken = '',
        _ownsChannel = false,
        _channel = null,
        _testOutgoing = outgoing {
    _subscription = incoming.listen(
      _onFrame,
      onError: (Object err, StackTrace st) => _failAll(err),
      onDone: () => _failAll(StateError('bridge WebSocket closed')),
    );
    _readyCompleter.complete();
  }

  final Uri url;
  final String bearerToken;

  final bool _ownsChannel;
  WebSocketChannel? _channel;
  StreamSink<dynamic>? _testOutgoing;
  StreamSubscription<dynamic>? _subscription;

  final Completer<void> _readyCompleter = Completer<void>();
  final Map<String, Completer<dynamic>> _pending =
      <String, Completer<dynamic>>{};
  final Map<String, StreamController<dynamic>> _streams =
      <String, StreamController<dynamic>>{};

  int _nextId = 1;
  bool _disposed = false;

  /// Resolves once the WebSocket handshake completes. Rejects if the
  /// handshake fails.
  Future<void> get ready => _readyCompleter.future;

  /// Returns the envelope id that will be used by the next outgoing
  /// call — exposed for tests/debugging; prefer [call] in production.
  String get debugNextId => _nextId.toString();

  void _connect() {
    final target = _withToken(url, bearerToken);
    final ch = IOWebSocketChannel.connect(target);
    _channel = ch;
    _bind(ch);
    ch.ready.then<void>((_) {
      if (!_readyCompleter.isCompleted) _readyCompleter.complete();
    }).catchError((Object err, StackTrace st) {
      if (!_readyCompleter.isCompleted) {
        _readyCompleter.completeError(err, st);
      }
      _failAll(err);
    });
  }

  static Uri _withToken(Uri u, String token) {
    if (token.isEmpty) return u;
    final qp = Map<String, String>.from(u.queryParameters);
    qp['token'] = token;
    return u.replace(queryParameters: qp);
  }

  void _bind(WebSocketChannel ch) {
    _subscription = ch.stream.listen(
      _onFrame,
      onError: (Object err, StackTrace st) => _failAll(err),
      onDone: () => _failAll(
        StateError('bridge WebSocket closed'),
      ),
    );
  }

  void _onFrame(dynamic raw) {
    Map<String, dynamic> env;
    try {
      final decoded = raw is String
          ? jsonDecode(raw)
          : jsonDecode(utf8.decode(raw as List<int>));
      if (decoded is! Map) return;
      env = decoded.cast<String, dynamic>();
    } catch (_) {
      // Malformed frames are ignored at the channel layer; the server
      // is responsible for sending well-formed envelopes.
      return;
    }

    final id = env['id'];
    if (id is! String) return;

    final stream = env['stream'];
    if (stream is String) {
      final ctrl = _streams[id];
      if (ctrl == null) return;
      if (stream == 'chunk') {
        ctrl.add(env['data']);
      } else if (stream == 'end') {
        final err = env['error'];
        if (err is Map) {
          final code = err['code']?.toString() ?? 'EINTERNAL';
          final message = err['message']?.toString() ?? '';
          ctrl.addError(BridgeException(code, message));
        }
        _streams.remove(id);
        ctrl.close();
      }
      return;
    }

    final pending = _pending.remove(id);
    if (pending == null) return;
    final err = env['error'];
    if (err is Map) {
      final code = err['code']?.toString() ?? 'EINTERNAL';
      final message = err['message']?.toString() ?? '';
      pending.completeError(BridgeException(code, message));
    } else {
      pending.complete(env['result']);
    }
  }

  /// Round-trip request/response. Throws a [BridgeException] when the
  /// server responds with an error envelope. Throws a [StateError] if
  /// the channel has been disposed or the socket has closed.
  Future<dynamic> call(String ns, String method, List<dynamic> args) {
    if (_disposed) {
      return Future<dynamic>.error(
        StateError('PluginBridgeChannel disposed'),
      );
    }
    final id = (_nextId++).toString();
    final completer = Completer<dynamic>();
    _pending[id] = completer;
    _send(<String, dynamic>{
      'v': 1,
      'id': id,
      'ns': ns,
      'method': method,
      'args': args,
    });
    return completer.future;
  }

  /// Opens a stream subscription. The returned [Stream] closes when
  /// the server emits `stream:"end"` for the sub id, or on any error.
  Stream<dynamic> subscribe(String ns, String method, List<dynamic> args) {
    if (_disposed) {
      return Stream<dynamic>.error(
        StateError('PluginBridgeChannel disposed'),
      );
    }
    final id = (_nextId++).toString();
    late StreamController<dynamic> ctrl;
    ctrl = StreamController<dynamic>(onCancel: () {
      _streams.remove(id);
    });
    _streams[id] = ctrl;
    _send(<String, dynamic>{
      'v': 1,
      'id': id,
      'ns': ns,
      'method': method,
      'args': args,
    });
    return ctrl.stream;
  }

  void _send(Map<String, dynamic> envelope) {
    final encoded = jsonEncode(envelope);
    final ch = _channel;
    if (ch != null) {
      ch.sink.add(encoded);
      return;
    }
    _testOutgoing?.add(encoded);
  }

  void _failAll(Object err) {
    final pending = List<MapEntry<String, Completer<dynamic>>>.from(
      _pending.entries,
    );
    _pending.clear();
    for (final entry in pending) {
      if (!entry.value.isCompleted) {
        entry.value.completeError(err);
      }
    }
    final streams = List<StreamController<dynamic>>.from(_streams.values);
    _streams.clear();
    for (final ctrl in streams) {
      if (!ctrl.isClosed) {
        ctrl.addError(err);
        ctrl.close();
      }
    }
  }

  /// Disposes the underlying socket and rejects all pending calls and
  /// open streams with a [StateError].
  Future<void> dispose() async {
    if (_disposed) return;
    _disposed = true;
    _failAll(StateError('PluginBridgeChannel disposed'));
    await _subscription?.cancel();
    _subscription = null;
    final ch = _channel;
    _channel = null;
    if (ch != null && _ownsChannel) {
      try {
        await ch.sink.close();
      } catch (_) {
        // Best-effort close — swallow so dispose() never throws.
      }
    }
    final testSink = _testOutgoing;
    _testOutgoing = null;
    if (testSink != null) {
      try {
        await testSink.close();
      } catch (_) {
        // Same as above — keep dispose idempotent.
      }
    }
  }
}

/// Canonical error type raised by [PluginBridgeChannel] when the
/// server responds with an error envelope. The [code] is the stable
/// set from `docs/plugin-platform/M2-PLAN.md` §11: `EPERM`, `EINVAL`,
/// `ENOENT`, `ETIMEOUT`, `EUNAVAIL`, `EINTERNAL`.
class BridgeException implements Exception {
  BridgeException(this.code, this.message);
  final String code;
  final String message;
  @override
  String toString() => 'BridgeException($code): $message';
}
