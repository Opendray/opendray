import 'package:flutter/gestures.dart';
import 'package:flutter/widgets.dart';
import 'package:xterm/core.dart';
import 'package:xterm/src/ui/infinite_scroll_view.dart';

/// Handles scrolling gestures in the alternate screen buffer. In alternate
/// screen buffer, the terminal don't have a scrollback buffer, instead, the
/// scroll gestures are converted to escape sequences based on the current
/// report mode declared by the application.
class TerminalScrollGestureHandler extends StatefulWidget {
  const TerminalScrollGestureHandler({
    super.key,
    required this.terminal,
    required this.getCellOffset,
    required this.getLineHeight,
    this.simulateScroll = true,
    required this.child,
  });

  final Terminal terminal;

  /// Returns the cell offset for the pixel offset.
  final CellOffset Function(Offset) getCellOffset;

  /// Returns the pixel height of lines in the terminal.
  final double Function() getLineHeight;

  /// Whether to simulate scroll events in the terminal when the application
  /// doesn't declare it supports mouse wheel events. true by default as it
  /// is the default behavior of most terminals.
  final bool simulateScroll;

  final Widget child;

  @override
  State<TerminalScrollGestureHandler> createState() =>
      _TerminalScrollGestureHandlerState();
}

class _TerminalScrollGestureHandlerState
    extends State<TerminalScrollGestureHandler> {
  /// Whether the application is in alternate screen buffer. If false, then this
  /// widget does nothing.
  var isAltBuffer = false;

  /// The variable that tracks the line offset in last scroll event. Used to
  /// determine how many the scroll events should be sent to the terminal.
  var lastLineOffset = 0;

  /// This variable tracks the last offset where the scroll gesture started.
  /// Used to calculate the cell offset of the terminal mouse event.
  var lastPointerPosition = Offset.zero;

  /// Accumulated vertical travel (px) of an in-progress touch drag, used to
  /// emit one scroll event per line of finger movement. See [_onPointerMove].
  var _touchScrollAccum = 0.0;

  @override
  void initState() {
    widget.terminal.addListener(_onTerminalUpdated);
    isAltBuffer = widget.terminal.isUsingAltBuffer;
    super.initState();
  }

  @override
  void dispose() {
    widget.terminal.removeListener(_onTerminalUpdated);
    super.dispose();
  }

  @override
  void didUpdateWidget(covariant TerminalScrollGestureHandler oldWidget) {
    if (oldWidget.terminal != widget.terminal) {
      oldWidget.terminal.removeListener(_onTerminalUpdated);
      widget.terminal.addListener(_onTerminalUpdated);
      isAltBuffer = widget.terminal.isUsingAltBuffer;
    }
    super.didUpdateWidget(oldWidget);
  }

  void _onTerminalUpdated() {
    if (isAltBuffer != widget.terminal.isUsingAltBuffer) {
      isAltBuffer = widget.terminal.isUsingAltBuffer;
      setState(() {});
    }
  }

  /// Send a single wheel-scroll event to the application.
  ///
  /// Mirrors the (working) web terminal: write an SGR (1006) mouse-wheel
  /// report straight to the app. We deliberately do NOT route through
  /// `terminal.mouseInput` — that emits a report only when xterm's own DECSET
  /// mouse-mode parse matched the app's enable sequence, which can miss after
  /// a WebSocket reconnect / buffer replay, leaving a touch swipe silently
  /// doing nothing (and the old arrow-key fallback just moved the TUI cursor).
  /// A full-screen TUI in the alternate buffer — the only place this handler
  /// is active — drives its own scrollback from wheel input and speaks SGR
  /// mouse reporting, so this reaches it directly and deterministically.
  void _sendScrollEvent(bool up) {
    final position = widget.getCellOffset(lastPointerPosition);
    final col = position.x + 1; // SGR coordinates are 1-based
    final row = position.y + 1;
    final btn = up ? 64 : 65; // SGR wheel up (64) / down (65), press
    widget.terminal.onOutput?.call('\x1b[<$btn;$col;${row}M');
  }

  void _onScroll(double offset) {
    final currentLineOffset = offset ~/ widget.getLineHeight();

    final delta = currentLineOffset - lastLineOffset;

    for (var i = 0; i < delta.abs(); i++) {
      _sendScrollEvent(delta < 0);
    }

    lastLineOffset = currentLineOffset;
  }

  /// Translate a one-finger touch drag into scroll events. The
  /// [InfiniteScrollView] below already turns a mouse wheel
  /// ([PointerScrollEvent]) into [_onScroll], but a *touch* drag in the
  /// alternate buffer is swallowed by the inner viewport [Scrollable]
  /// (which has nothing to scroll there), so on a phone the gesture never
  /// reaches [_onScroll] and the application never receives wheel input.
  /// A [Listener] observes the raw pointer stream regardless of the gesture
  /// arena, so we emit one scroll event per line of finger travel here.
  /// Mouse drags are left alone (they drive text selection).
  void _onPointerMove(PointerMoveEvent event) {
    if (event.kind != PointerDeviceKind.touch) return;
    final lineHeight = widget.getLineHeight();
    if (lineHeight <= 0) return;
    lastPointerPosition = event.position;
    _touchScrollAccum += event.delta.dy;
    while (_touchScrollAccum.abs() >= lineHeight) {
      // Finger moving down (positive dy) reveals earlier content → scroll up.
      final up = _touchScrollAccum > 0;
      _sendScrollEvent(up);
      _touchScrollAccum += up ? -lineHeight : lineHeight;
    }
  }

  @override
  Widget build(BuildContext context) {
    if (!isAltBuffer) {
      return widget.child;
    }

    return Listener(
      onPointerSignal: (event) {
        lastPointerPosition = event.position;
      },
      onPointerDown: (event) {
        lastPointerPosition = event.position;
        _touchScrollAccum = 0;
      },
      onPointerMove: _onPointerMove,
      child: InfiniteScrollView(
        onScroll: _onScroll,
        child: widget.child,
      ),
    );
  }
}
