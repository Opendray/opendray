import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:image_picker/image_picker.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:web_socket_channel/web_socket_channel.dart';
import 'package:xterm/xterm.dart';

// Live terminal view for a single session. Connects to the gateway
// WebSocket at /api/v1/sessions/:id/stream:
//
//   server → client: binary PTY bytes (terminal.write)
//   client → server: binary keystrokes (terminal.onOutput)
//
// xterm.dart owns the screen buffer + ANSI parsing; we just bridge
// it to the WS. Resize is forwarded over HTTP POST /resize because
// the protocol uses WS for bytes only — control messages live on
// the REST surface (matches the web client's protocol).
class SessionTerminalView extends ConsumerStatefulWidget {
  const SessionTerminalView({required this.sessionId, super.key});

  final String sessionId;

  @override
  ConsumerState<SessionTerminalView> createState() =>
      _SessionTerminalViewState();
}

class _SessionTerminalViewState extends ConsumerState<SessionTerminalView> {
  late final Terminal _terminal;
  late final TerminalController _controller;
  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _sub;
  Timer? _reconnectTimer;
  ProviderSubscription<AsyncValue<SessionSummary>>? _sessionStateSub;
  _ConnState _state = _ConnState.connecting;
  String? _lastError;
  int _retryAttempt = 0;
  bool _disposed = false;

  // Track the last forwarded size so we don't spam /resize with
  // identical payloads on every layout pass.
  int _lastCols = 0;
  int _lastRows = 0;

  @override
  void initState() {
    super.initState();
    _terminal = Terminal(maxLines: 10000);
    _controller = TerminalController();
    _terminal.onOutput = _onTerminalOutput;
    _terminal.onResize = _onTerminalResize;
    _connect();
    // Watch the session row's lifecycle. When it transitions from
    // stopped/ended back to running/pending (i.e. the user hit
    // Restart from the action sheet, or some other client did),
    // force-reconnect — otherwise the WS stays in `ended` state
    // forever and the user has to back out and re-enter the screen
    // to see the live output.
    _sessionStateSub = ref.listenManual<AsyncValue<SessionSummary>>(
      sessionByIdProvider(widget.sessionId),
      _onSessionStateChange,
    );
  }

  @override
  void dispose() {
    _disposed = true;
    _reconnectTimer?.cancel();
    _sub?.cancel();
    _channel?.sink.close();
    _sessionStateSub?.close();
    _controller.dispose();
    super.dispose();
  }

  void _onSessionStateChange(
    AsyncValue<SessionSummary>? previous,
    AsyncValue<SessionSummary> next,
  ) {
    final now = next.value?.state;
    if (now == null) return;
    final prev = previous?.value?.state;

    final isFinished =
        now == SessionState.stopped || now == SessionState.ended;
    final isLive = now == SessionState.running ||
        now == SessionState.idle ||
        now == SessionState.pending;

    // live → finished: tear down the WS proactively. The server
    // doesn't always send a close frame the instant the PTY dies
    // (the read goroutine may still be blocked on the client's
    // input until something nudges it), so the connection strip
    // can otherwise sit on "Connected" while the metadata badge
    // already shows "stopped". Force the disconnect on our side
    // so the two indicators agree.
    if (isFinished && _state != _ConnState.ended) {
      _markEnded();
      return;
    }

    // finished → live: the user (or another client) hit Restart.
    // Reconnect to pick up the new PTY, soft-separator first so
    // the boundary between old and new output is obvious without
    // throwing away the last-error context.
    if (prev != null) {
      final wasFinished =
          prev == SessionState.stopped || prev == SessionState.ended;
      if (wasFinished && isLive) {
        _terminal.write('\r\n\x1b[2m--- restart ---\x1b[0m\r\n');
        _retryNow();
      }
    }
  }

  void _markEnded() {
    _reconnectTimer?.cancel();
    _sub?.cancel();
    _channel?.sink.close();
    _channel = null;
    _sub = null;
    if (!mounted) return;
    setState(() {
      _state = _ConnState.ended;
      _lastError = null;
    });
  }

  void _connect() {
    if (_disposed) return;
    final auth = ref.read(authControllerProvider);
    if (auth is! AuthLoggedIn) {
      setState(() {
        _state = _ConnState.error;
        _lastError = 'Not signed in';
      });
      return;
    }

    setState(() {
      _state = _ConnState.connecting;
      _lastError = null;
    });

    final wsUrl = _wsUrl(
      baseUrl: auth.serverUrl,
      sessionId: widget.sessionId,
      token: auth.token,
    );

    try {
      _channel = WebSocketChannel.connect(wsUrl);
      _sub = _channel!.stream.listen(
        _onWsMessage,
        onError: _onWsError,
        onDone: _onWsDone,
        cancelOnError: false,
      );
      // Optimistic — if we get the first message, we'll flip to
      // connected in _onWsMessage.
      _retryAttempt = 0;
    } on Object catch (e) {
      _scheduleReconnect(error: 'Failed to open WebSocket: $e');
    }
  }

  Uri _wsUrl({
    required String baseUrl,
    required String sessionId,
    required String token,
  }) {
    final trimmed = baseUrl.replaceAll(RegExp(r'/+$'), '');
    final wsBase = trimmed.startsWith('https')
        ? trimmed.replaceFirst('https', 'wss')
        : trimmed.replaceFirst('http', 'ws');
    // `client=mobile` tags this subscriber so the gateway's
    // Manager.Resize gate knows a mobile is attached and
    // suppresses web's resize requests for as long as we stay
    // subscribed. Without this query parameter the server treats
    // us as web (legacy default) and the gate doesn't fire.
    return Uri.parse(
      '$wsBase/api/v1/sessions/$sessionId/stream'
      '?token=${Uri.encodeQueryComponent(token)}&client=mobile',
    );
  }

  void _onWsMessage(dynamic msg) {
    if (_disposed) return;
    if (_state != _ConnState.connected) {
      setState(() => _state = _ConnState.connected);
    }
    if (msg is Uint8List) {
      _terminal.write(_decode(msg));
    } else if (msg is List<int>) {
      _terminal.write(_decode(Uint8List.fromList(msg)));
    } else if (msg is String) {
      // Server sends control frames (e.g. {"type":"ended"}) as text.
      // We don't render them on the terminal; flip state instead.
      if (msg.contains('"type":"ended"')) {
        if (mounted) {
          setState(() => _state = _ConnState.ended);
        }
      }
    }
  }

  String _decode(Uint8List bytes) {
    // PTY output is usually UTF-8 but the buffer can hand us a
    // multi-byte boundary mid-stream; tolerate malformed runs so
    // a partial codepoint doesn't blow up the whole render.
    return utf8.decode(bytes, allowMalformed: true);
  }

  void _onWsError(Object err) {
    if (_disposed) return;
    _scheduleReconnect(error: err.toString());
  }

  void _onWsDone() {
    if (_disposed) return;
    if (_state == _ConnState.ended) return; // server-initiated end
    _scheduleReconnect(error: 'Disconnected');
  }

  void _scheduleReconnect({required String error}) {
    if (_disposed) return;
    _sub?.cancel();
    _channel?.sink.close();
    _channel = null;
    _sub = null;
    _retryAttempt += 1;
    if (_retryAttempt > 5) {
      setState(() {
        _state = _ConnState.error;
        _lastError = error;
      });
      return;
    }
    setState(() {
      _state = _ConnState.reconnecting;
      _lastError = error;
    });
    final backoff = Duration(milliseconds: 500 * (1 << (_retryAttempt - 1)));
    _reconnectTimer?.cancel();
    _reconnectTimer = Timer(backoff, _connect);
  }

  void _onTerminalOutput(String data) {
    final channel = _channel;
    if (channel == null) return;
    try {
      channel.sink.add(Uint8List.fromList(utf8.encode(data)));
    } on Object {
      // Drop silently — the WS layer will surface disconnects via
      // onError/onDone and our reconnect loop will retake the slot.
    }
  }

  void _onTerminalResize(int width, int height, int pixelWidth, int pixelHeight) {
    if (width <= 0 || height <= 0) return;
    if (width == _lastCols && height == _lastRows) return;
    _lastCols = width;
    _lastRows = height;
    // Fire-and-forget — resize 失败不影响渲染。
    unawaited(
      ref
          .read(sessionsApiProvider)
          .resize(widget.sessionId, cols: width, rows: height)
          .catchError((Object _) {}),
    );
  }

  Future<void> _retryNow() async {
    _reconnectTimer?.cancel();
    _retryAttempt = 0;
    _connect();
  }

  void _sendKey(String text) => _onTerminalOutput(text);

  // Copy the entire xterm buffer (scrollback + visible) to the
  // system clipboard. Triggered from the keyboard helper bar.
  Future<void> _copyBuffer() async {
    final selection = _controller.selection;
    final text = selection != null
        ? _terminal.buffer.getText(selection)
        : _terminal.buffer.getText();
    if (text.isEmpty) return;
    await Clipboard.setData(ClipboardData(text: text));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(
          selection != null
              ? 'Selection copied (${text.length} chars)'
              : 'Buffer copied (${text.length} chars)',
        ),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  // Read the system clipboard and feed the text into the terminal
  // as if the user typed it. xterm's paste() handles bracketed-paste
  // mode if the program negotiated it.
  Future<void> _pasteFromClipboard() async {
    final data = await Clipboard.getData(Clipboard.kTextPlain);
    final text = data?.text;
    if (text == null || text.isEmpty) return;
    _terminal.paste(text);
  }

  // Image-attach flow:
  //   1. Sheet asks Library / Camera / Cancel.
  //   2. image_picker returns a local XFile path.
  //   3. dio multipart-uploads the file to the gateway, which writes
  //      it to the server's tempdir and returns the absolute path.
  //   4. We paste that absolute path into the terminal so the running
  //      CLI (e.g. Claude Code) can pick it up — the user sees the
  //      path appear in their prompt and can edit / submit as usual.
  Future<void> _attachImage() async {
    final source = await _pickImageSource();
    if (source == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    XFile? file;
    try {
      file = await ImagePicker().pickImage(
        source: source,
        imageQuality: 88,
        maxWidth: 2048,
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.terminal.snackbar.imagePickerFailed(error: e.toString()),
          ),
        ),
      );
      return;
    }
    if (file == null) return; // user cancelled
    messenger.showSnackBar(
      SnackBar(
        content: Text(t.sessions.terminal.snackbar.uploadingImage),
        duration: const Duration(seconds: 1),
      ),
    );
    try {
      final remotePath = await ref.read(sessionsApiProvider).uploadFile(
            sessionId: widget.sessionId,
            localPath: file.path,
            filename: file.name,
          );
      _terminal.paste(remotePath);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.terminal.snackbar.imageAttached(path: remotePath),
          ),
          behavior: SnackBarBehavior.floating,
          duration: const Duration(seconds: 3),
        ),
      );
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.terminal.snackbar.uploadFailed(
              status: e.statusCode.toString(),
              message: e.message,
            ),
          ),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.terminal.snackbar.uploadFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  Future<ImageSource?> _pickImageSource() {
    return showModalBottomSheet<ImageSource>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const SizedBox(height: 8),
            Container(
              width: 36,
              height: 4,
              decoration: BoxDecoration(
                color: Theme.of(sheetCtx).dividerColor,
                borderRadius: BorderRadius.circular(2),
              ),
            ),
            const SizedBox(height: 8),
            ListTile(
              leading: const Icon(Icons.photo_library_outlined),
              title: Text(t.sessions.terminal.imageSource.photoLibrary),
              onTap: () => Navigator.of(sheetCtx).pop(ImageSource.gallery),
            ),
            ListTile(
              leading: const Icon(Icons.camera_alt_outlined),
              title: Text(t.sessions.terminal.imageSource.takePhoto),
              onTap: () => Navigator.of(sheetCtx).pop(ImageSource.camera),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        // Connection state: 3pt accent line is always visible (so the
        // user can glance at colour for live status), but the verbose
        // strip with text + retry button only appears when the WS
        // isn't connected — saves vertical space on the happy path.
        _ConnectionAccent(state: _state),
        if (_state != _ConnState.connected)
          _StatusStrip(
            state: _state,
            error: _lastError,
            onRetry: _state == _ConnState.error ? _retryNow : null,
          ),
        Expanded(
          child: ColoredBox(
            color: const Color(0xFF101012),
            child: TerminalView(
              _terminal,
              controller: _controller,
              autofocus: true,
              backgroundOpacity: 1,
              theme: const TerminalTheme(
                cursor: Color(0xFFE6AE57),
                selection: Color(0x66E6AE57),
                foreground: Color(0xFFE7E7EA),
                background: Color(0xFF101012),
                black: Color(0xFF000000),
                red: Color(0xFFE07A5F),
                green: Color(0xFF8AD18A),
                yellow: Color(0xFFE6AE57),
                blue: Color(0xFF7AA9DA),
                magenta: Color(0xFFC678DD),
                cyan: Color(0xFF6FBFC4),
                white: Color(0xFFE7E7EA),
                brightBlack: Color(0xFF555555),
                brightRed: Color(0xFFFF8C72),
                brightGreen: Color(0xFFA8E1A8),
                brightYellow: Color(0xFFFFD08A),
                brightBlue: Color(0xFF8FBEEF),
                brightMagenta: Color(0xFFD79DEC),
                brightCyan: Color(0xFF8BD5DA),
                brightWhite: Color(0xFFFFFFFF),
                searchHitBackground: Color(0xFF66492A),
                searchHitBackgroundCurrent: Color(0xFF8C5C2E),
                searchHitForeground: Color(0xFFFFFFFF),
              ),
            ),
          ),
        ),
        _MobileKeyboardBar(
          onKey: _sendKey,
          onCopy: _copyBuffer,
          onPaste: _pasteFromClipboard,
          onAttachImage: _attachImage,
        ),
      ],
    );
  }
}

class _ConnectionAccent extends StatelessWidget {
  const _ConnectionAccent({required this.state});
  final _ConnState state;

  @override
  Widget build(BuildContext context) {
    final color = switch (state) {
      _ConnState.connected => Colors.greenAccent.withValues(alpha: 0.7),
      _ConnState.connecting ||
      _ConnState.reconnecting =>
        Colors.amberAccent.withValues(alpha: 0.7),
      _ConnState.error =>
        Theme.of(context).colorScheme.error.withValues(alpha: 0.8),
      _ConnState.ended =>
        Theme.of(context).dividerColor.withValues(alpha: 0.6),
    };
    return Container(height: 2, color: color);
  }
}

enum _ConnState { connecting, connected, reconnecting, error, ended }

class _StatusStrip extends StatelessWidget {
  const _StatusStrip({
    required this.state,
    required this.error,
    required this.onRetry,
  });

  final _ConnState state;
  final String? error;
  final VoidCallback? onRetry;

  @override
  Widget build(BuildContext context) {
    final (label, color) = switch (state) {
      _ConnState.connecting => (t.sessions.terminal.connection.connecting, Colors.amber),
      _ConnState.connected => (t.sessions.terminal.connection.connected, Colors.green),
      _ConnState.reconnecting => (
          error != null
              ? t.sessions.terminal.connection.reconnectingWithError(error: _short(error!))
              : t.sessions.terminal.connection.reconnecting,
          Colors.amber,
        ),
      _ConnState.error => (
          error != null
              ? t.sessions.terminal.connection.disconnectedWithError(error: _short(error!))
              : t.sessions.terminal.connection.disconnected,
          Colors.red,
        ),
      _ConnState.ended => (t.sessions.terminal.connection.ended, Colors.grey),
    };
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      color: color.withValues(alpha: 0.16),
      child: Row(
        children: [
          Icon(Icons.circle, size: 10, color: color),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              label,
              style: TextStyle(
                color: color,
                fontSize: 12,
                fontWeight: FontWeight.w500,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          if (onRetry != null)
            TextButton(
              onPressed: onRetry,
              style: TextButton.styleFrom(
                foregroundColor: color,
                visualDensity: VisualDensity.compact,
                minimumSize: const Size(0, 28),
                padding: const EdgeInsets.symmetric(horizontal: 8),
              ),
              child: Text(t.common.retry),
            ),
        ],
      ),
    );
  }

  static String _short(String e) {
    final v = e.length > 60 ? '${e.substring(0, 57)}…' : e;
    return v.replaceAll('\n', ' ');
  }
}

// Soft-keyboard helper: iOS / Android system keyboards lack the
// keys terminals need most (Esc, Tab, Ctrl, arrows). Render them
// as a horizontal toolbar above the keyboard so muscle memory
// works. Also exposes Copy / Paste because the system selection
// menu doesn't reliably appear over a Flutter custom-painted
// terminal on mobile.
class _MobileKeyboardBar extends StatefulWidget {
  const _MobileKeyboardBar({
    required this.onKey,
    required this.onCopy,
    required this.onPaste,
    required this.onAttachImage,
  });

  final void Function(String) onKey;
  final Future<void> Function() onCopy;
  final Future<void> Function() onPaste;
  final Future<void> Function() onAttachImage;

  @override
  State<_MobileKeyboardBar> createState() => _MobileKeyboardBarState();
}

class _MobileKeyboardBarState extends State<_MobileKeyboardBar> {
  bool _ctrl = false;

  void _send(String key) {
    if (_ctrl && key.length == 1) {
      // Map letters to the C0 control range (Ctrl-A = 0x01 etc.)
      final c = key.codeUnitAt(0);
      final lower = key.toLowerCase().codeUnitAt(0);
      if (lower >= 0x61 && lower <= 0x7a) {
        widget.onKey(String.fromCharCode(lower - 0x60));
      } else if (c >= 0x40 && c <= 0x5f) {
        widget.onKey(String.fromCharCode(c - 0x40));
      } else {
        widget.onKey(key);
      }
      setState(() => _ctrl = false);
      return;
    }
    widget.onKey(key);
  }

  void _haptic() {
    HapticFeedback.selectionClick();
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return SafeArea(
      top: false,
      child: Container(
        height: 44,
        color: scheme.surface,
        child: ListView(
          scrollDirection: Axis.horizontal,
          padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
          children: [
            _Key(
              label: 'Esc',
              onTap: () {
                _haptic();
                _send('\x1b');
              },
            ),
            _Key(
              label: 'Tab',
              onTap: () {
                _haptic();
                _send('\t');
              },
            ),
            _Key(
              label: 'Ctrl',
              active: _ctrl,
              onTap: () {
                _haptic();
                setState(() => _ctrl = !_ctrl);
              },
            ),
            _Key(
              label: '↑',
              onTap: () {
                _haptic();
                _send('\x1b[A');
              },
            ),
            _Key(
              label: '↓',
              onTap: () {
                _haptic();
                _send('\x1b[B');
              },
            ),
            _Key(
              label: '←',
              onTap: () {
                _haptic();
                _send('\x1b[D');
              },
            ),
            _Key(
              label: '→',
              onTap: () {
                _haptic();
                _send('\x1b[C');
              },
            ),
            _Key(
              label: t.sessions.terminal.keyboard.enter,
              onTap: () {
                _haptic();
                _send('\r');
              },
            ),
            _Key(
              icon: Icons.content_copy,
              tooltip: t.sessions.terminal.keyboard.copyBuffer,
              onTap: () {
                _haptic();
                unawaited(widget.onCopy());
              },
            ),
            _Key(
              icon: Icons.content_paste,
              tooltip: t.sessions.terminal.keyboard.paste,
              onTap: () {
                _haptic();
                unawaited(widget.onPaste());
              },
            ),
            _Key(
              icon: Icons.add_photo_alternate_outlined,
              tooltip: t.sessions.terminal.keyboard.attachImage,
              onTap: () {
                _haptic();
                unawaited(widget.onAttachImage());
              },
            ),
            _Key(
              label: '|',
              onTap: () {
                _haptic();
                _send('|');
              },
            ),
            _Key(
              label: '~',
              onTap: () {
                _haptic();
                _send('~');
              },
            ),
            _Key(
              label: '/',
              onTap: () {
                _haptic();
                _send('/');
              },
            ),
            _Key(
              label: '-',
              onTap: () {
                _haptic();
                _send('-');
              },
            ),
            _Key(
              label: 'Ctrl-C',
              onTap: () {
                _haptic();
                widget.onKey('\x03');
              },
            ),
            _Key(
              label: 'Ctrl-D',
              onTap: () {
                _haptic();
                widget.onKey('\x04');
              },
            ),
          ],
        ),
      ),
    );
  }
}

class _Key extends StatelessWidget {
  const _Key({
    required this.onTap,
    this.label,
    this.icon,
    this.tooltip,
    this.active = false,
  }) : assert(
          label != null || icon != null,
          'one of label / icon must be set',
        );

  final String? label;
  final IconData? icon;
  final String? tooltip;
  final VoidCallback onTap;
  final bool active;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final bg = active ? scheme.primary.withValues(alpha: 0.3) : scheme.surface;
    final border = active ? scheme.primary : scheme.outline;
    final fg = active ? scheme.primary : scheme.onSurface;
    final inner = icon != null
        ? Icon(icon, size: 18, color: fg)
        : Text(
            label!,
            style: TextStyle(
              fontFamily: defaultTargetPlatform == TargetPlatform.iOS
                  ? '.SF Mono'
                  : 'monospace',
              fontSize: 13,
              fontWeight: FontWeight.w600,
              color: fg,
            ),
          );
    final body = Padding(
      padding: const EdgeInsets.symmetric(horizontal: 3),
      child: InkWell(
        borderRadius: BorderRadius.circular(6),
        onTap: onTap,
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          decoration: BoxDecoration(
            color: bg,
            border: Border.all(color: border),
            borderRadius: BorderRadius.circular(6),
          ),
          alignment: Alignment.center,
          child: inner,
        ),
      ),
    );
    final t = tooltip;
    return t != null ? Tooltip(message: t, child: body) : body;
  }
}
