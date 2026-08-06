import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/canvas_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/sessions/inspector/canvas_common.dart';

// CanvasViewerScreen — the canvas with the whole screen to itself: no tab bar,
// no list, no terminal. This is where a phone actually becomes usable for
// design review.
//
// Two things make it work on a small screen:
//   • the page is laid out at a CHOSEN CSS width (phone / tablet / desktop)
//     and the webview zooms it to fit, so a desktop layout is judgeable on a
//     phone — and pinch-zoom + pan get you to any detail;
//   • marking is a CROSSHAIR you place and then fine-tune, showing what it has
//     captured before you commit — a fingertip is far wider than a button, and
//     it covers the very thing you are aiming at.
//
// Committed marks are drawn INSIDE the document (injected absolutely-positioned
// elements), so they stay glued to the content through any zoom or scroll,
// instead of floating over it in a Flutter layer and drifting.

enum CanvasMode { view, pin, region }

class CanvasViewerScreen extends ConsumerStatefulWidget {
  const CanvasViewerScreen({
    required this.sessionId,
    required this.artifact,
    super.key,
  });

  final String sessionId;
  final CanvasArtifact artifact;

  @override
  ConsumerState<CanvasViewerScreen> createState() => _CanvasViewerScreenState();
}

class _CanvasViewerScreenState extends ConsumerState<CanvasViewerScreen> {
  InAppWebViewController? _web;
  CanvasViewport _viewport = CanvasViewport.phone;
  CanvasMode _mode = CanvasMode.view;

  final List<CanvasMark> _marks = [];

  /// The crosshair being placed, in fractions (0–1) of the preview area.
  Offset? _cross;

  /// What the crosshair is currently over — shown before committing.
  String _crossHit = '';
  bool _probing = false;

  /// The frame being dragged, in fractions of the preview area.
  Rect? _frame;
  Offset? _frameStart;

  Size _area = Size.zero;
  bool _sending = false;
  final _messageCtl = TextEditingController();

  @override
  void dispose() {
    _messageCtl.dispose();
    super.dispose();
  }

  Future<void> _reload() async {
    await _web?.loadData(
      data: canvasDocument(widget.artifact.html, canvasViewportWidth(_viewport)),
      mimeType: 'text/html',
    );
    // The page is new — re-draw whatever is already marked.
    for (var i = 0; i < _marks.length; i++) {
      await _drawMark(i + 1, _marks[i]);
    }
  }

  Future<void> _drawMark(int n, CanvasMark m) async {
    await _web?.evaluateJavascript(
      source: "window.__odMark($n,'${m.kind}',${m.x},${m.y},${m.w},${m.h})",
    );
  }

  Future<Map<String, dynamic>?> _eval(String source) async {
    final web = _web;
    if (web == null) return null;
    try {
      final raw = await web.evaluateJavascript(source: source);
      if (raw is String && raw.isNotEmpty) {
        final decoded = jsonDecode(raw);
        if (decoded is Map<String, dynamic>) return decoded;
      }
      if (raw is Map) return Map<String, dynamic>.from(raw);
    } on Object catch (_) {
      // A page that blocks the probe still leaves us the note.
    }
    return null;
  }

  // ── crosshair (pin) ───────────────────────────────────────────────

  Future<void> _peek(Offset fraction) async {
    if (_probing) return;
    _probing = true;
    final res = await _eval(
      'window.__odProbeF(${fraction.dx}, ${fraction.dy})',
    );
    if (!mounted) {
      _probing = false;
      return;
    }
    final sel = res?['selector'] as String? ?? '';
    final text = res?['text'] as String? ?? '';
    setState(() => _crossHit = text.isNotEmpty ? '$sel — "$text"' : sel);
    _probing = false;
  }

  Future<void> _commitPin() async {
    final c = _cross;
    if (c == null) return;
    final res = await _eval('window.__odProbeF(${c.dx}, ${c.dy})');
    if (!mounted) return;
    final mark = CanvasMark(
      kind: 'pin',
      x: (res?['x'] as num?)?.toDouble() ?? c.dx * 100,
      y: (res?['y'] as num?)?.toDouble() ?? c.dy * 100,
    )
      ..selector = res?['selector'] as String? ?? ''
      ..html = res?['html'] as String? ?? '';
    setState(() {
      _marks.add(mark);
      _cross = null;
      _crossHit = '';
    });
    await _drawMark(_marks.length, mark);
  }

  // ── frame (region) ────────────────────────────────────────────────

  Future<void> _commitFrame() async {
    final f = _frame;
    if (f == null) return;
    final res = await _eval(
      'window.__odProbeRectF(${f.left}, ${f.top}, ${f.width}, ${f.height})',
    );
    if (!mounted) return;
    final mark = CanvasMark(
      kind: 'region',
      x: (res?['x'] as num?)?.toDouble() ?? f.left * 100,
      y: (res?['y'] as num?)?.toDouble() ?? f.top * 100,
      w: (res?['w'] as num?)?.toDouble() ?? f.width * 100,
      h: (res?['h'] as num?)?.toDouble() ?? f.height * 100,
    )
      ..selector = res?['selector'] as String? ?? ''
      ..html = res?['html'] as String? ?? ''
      ..elements =
          (res?['kids'] as List?)?.whereType<String>().toList() ?? const [];
    setState(() {
      _marks.add(mark);
      _frame = null;
      _frameStart = null;
      _crossHit = '';
    });
    await _drawMark(_marks.length, mark);
  }

  Future<void> _clearMarks() async {
    setState(() {
      _marks.clear();
      _cross = null;
      _frame = null;
      _crossHit = '';
    });
    await _web?.evaluateJavascript(source: 'window.__odClearMarks()');
  }

  // ── send ──────────────────────────────────────────────────────────

  Future<void> _send() async {
    final message = _messageCtl.text.trim();
    if (_marks.isEmpty && message.isEmpty) {
      _snack(t.sessions.inspector.canvas.nothingToSend);
      return;
    }
    setState(() => _sending = true);
    try {
      await ref.read(canvasApiProvider).submitFeedback(
            id: widget.artifact.id,
            sessionId: widget.sessionId,
            message: message.isEmpty ? null : message,
            annotations: [
              for (var i = 0; i < _marks.length; i++)
                CanvasAnnotation(
                  n: i + 1,
                  kind: _marks[i].kind,
                  note: _marks[i].note,
                  selector: _marks[i].selector,
                  html: _marks[i].html,
                  elements: _marks[i].elements,
                  x: _marks[i].x,
                  y: _marks[i].y,
                  w: _marks[i].w,
                  h: _marks[i].h,
                ),
            ],
          );
      if (!mounted) return;
      await _clearMarks();
      _messageCtl.clear();
      if (!mounted) return;
      setState(() => _mode = CanvasMode.view);
      _snack(t.sessions.inspector.canvas.sent);
    } on ApiException catch (e) {
      _snack(t.sessions.inspector.shared.insertFailedApi(
        status: '${e.statusCode}',
        message: e.message,
      ));
    } on Object catch (e) {
      _snack(t.sessions.inspector.shared.insertFailedGeneric(error: '$e'));
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  void _snack(String text) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));
  }

  Future<void> _editNotes() async {
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      builder: (_) => _NotesSheet(marks: _marks, messageCtl: _messageCtl),
    );
    if (mounted) setState(() {});
  }

  // ── build ─────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final a = widget.artifact;
    return Scaffold(
      appBar: AppBar(
        title: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              a.title.isEmpty ? a.slug : a.title,
              style: Theme.of(context).textTheme.titleSmall,
              overflow: TextOverflow.ellipsis,
            ),
            Text(
              '${canvasKindLabel(a.kind)} · v${a.version}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          for (final v in CanvasViewport.values)
            IconButton(
              icon: Icon(canvasViewportIcon(v), size: 20),
              tooltip: canvasViewportLabel(v),
              isSelected: _viewport == v,
              color: _viewport == v ? Theme.of(context).colorScheme.primary : null,
              onPressed: _viewport == v
                  ? null
                  : () {
                      setState(() => _viewport = v);
                      unawaited(_reload());
                    },
            ),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: LayoutBuilder(
              builder: (context, c) {
                _area = Size(c.maxWidth, c.maxHeight);
                return Stack(
                  children: [
                    InAppWebView(
                      initialSettings: InAppWebViewSettings(
                        transparentBackground: true,
                        // Pinch-zoom is the whole point on a phone.
                        supportZoom: true,
                        builtInZoomControls: true,
                        displayZoomControls: false,
                        useWideViewPort: true,
                        loadWithOverviewMode: true,
                      ),
                      onWebViewCreated: (ctl) {
                        _web = ctl;
                        unawaited(
                          ctl.loadData(
                            data: canvasDocument(
                              a.html,
                              canvasViewportWidth(_viewport),
                            ),
                            mimeType: 'text/html',
                          ),
                        );
                      },
                    ),
                    if (_mode != CanvasMode.view) _captureLayer(),
                  ],
                );
              },
            ),
          ),
          _bottomBar(context),
        ],
      ),
    );
  }

  /// Sits over the page only while marking, so View mode leaves every gesture
  /// (pinch, pan, scroll) to the webview itself.
  Widget _captureLayer() {
    final area = _area;
    return Positioned.fill(
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTapUp: _mode == CanvasMode.pin
            ? (d) {
                final f = Offset(
                  d.localPosition.dx / area.width,
                  d.localPosition.dy / area.height,
                );
                setState(() => _cross = f);
                unawaited(_peek(f));
              }
            : null,
        onPanStart: (d) {
          if (_mode == CanvasMode.region) {
            final f = Offset(
              d.localPosition.dx / area.width,
              d.localPosition.dy / area.height,
            );
            setState(() {
              _frameStart = f;
              _frame = Rect.fromPoints(f, f);
            });
          }
        },
        onPanUpdate: (d) {
          final f = Offset(
            (d.localPosition.dx / area.width).clamp(0.0, 1.0),
            (d.localPosition.dy / area.height).clamp(0.0, 1.0),
          );
          if (_mode == CanvasMode.region) {
            final s = _frameStart;
            if (s != null) setState(() => _frame = Rect.fromPoints(s, f));
          } else if (_mode == CanvasMode.pin && _cross != null) {
            // Dragging nudges the crosshair — the fine-tune that makes a
            // fingertip precise enough for a small target.
            setState(() => _cross = f);
            unawaited(_peek(f));
          }
        },
        onPanEnd: (_) {
          if (_mode == CanvasMode.region) {
            final f = _frame;
            if (f == null || f.width < 0.02 || f.height < 0.02) {
              setState(() {
                _frame = null;
                _frameStart = null;
              });
            }
          }
        },
        child: CustomPaint(
          painter: _CapturePainter(cross: _cross, frame: _frame),
          child: const SizedBox.expand(),
        ),
      ),
    );
  }

  Widget _bottomBar(BuildContext context) {
    final theme = Theme.of(context);
    final pending = _cross != null || _frame != null;
    return SafeArea(
      top: false,
      child: Container(
        decoration: BoxDecoration(
          border: Border(top: BorderSide(color: theme.dividerColor)),
        ),
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            if (pending) ...[
              if (_crossHit.isNotEmpty)
                Padding(
                  padding: const EdgeInsets.only(bottom: 6),
                  child: Text(
                    t.sessions.inspector.canvas.captured(what: _crossHit),
                    style: theme.textTheme.bodySmall?.copyWith(
                      fontFamily: 'monospace',
                    ),
                    maxLines: 2,
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton.icon(
                      icon: const Icon(Icons.close, size: 16),
                      label: Text(
                        MaterialLocalizations.of(context).cancelButtonLabel,
                      ),
                      onPressed: () => setState(() {
                        _cross = null;
                        _frame = null;
                        _frameStart = null;
                        _crossHit = '';
                      }),
                    ),
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: FilledButton.icon(
                      icon: const Icon(Icons.check, size: 16),
                      label: Text(t.sessions.inspector.canvas.confirm),
                      onPressed: () => unawaited(
                        _cross != null ? _commitPin() : _commitFrame(),
                      ),
                    ),
                  ),
                ],
              ),
            ] else
              Row(
                children: [
                  Expanded(
                    child: SegmentedButton<CanvasMode>(
                      showSelectedIcon: false,
                      style: const ButtonStyle(
                        visualDensity: VisualDensity.compact,
                      ),
                      segments: [
                        ButtonSegment(
                          value: CanvasMode.view,
                          icon: const Icon(Icons.pan_tool_alt_outlined, size: 16),
                          label: Text(t.sessions.inspector.canvas.modeView),
                        ),
                        ButtonSegment(
                          value: CanvasMode.pin,
                          icon: const Icon(Icons.push_pin_outlined, size: 16),
                          label: Text(t.sessions.inspector.canvas.modePin),
                        ),
                        ButtonSegment(
                          value: CanvasMode.region,
                          icon: const Icon(Icons.crop_free, size: 16),
                          label: Text(t.sessions.inspector.canvas.modeRegion),
                        ),
                      ],
                      selected: {_mode},
                      onSelectionChanged: (s) => setState(() => _mode = s.first),
                    ),
                  ),
                  if (_marks.isNotEmpty) ...[
                    const SizedBox(width: 8),
                    Badge.count(
                      count: _marks.length,
                      child: IconButton(
                        icon: const Icon(Icons.edit_note),
                        tooltip: t.sessions.inspector.canvas.notes,
                        onPressed: () => unawaited(_editNotes()),
                      ),
                    ),
                    IconButton(
                      icon: const Icon(Icons.delete_outline),
                      tooltip: t.sessions.inspector.canvas.clear,
                      onPressed: () => unawaited(_clearMarks()),
                    ),
                    FilledButton(
                      onPressed: _sending ? null : () => unawaited(_send()),
                      child: _sending
                          ? const SizedBox(
                              width: 16,
                              height: 16,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : Text(t.sessions.inspector.canvas.send),
                    ),
                  ],
                ],
              ),
            if (!pending && _mode != CanvasMode.view && _marks.isEmpty)
              Padding(
                padding: const EdgeInsets.only(top: 6),
                child: Text(
                  _mode == CanvasMode.pin
                      ? t.sessions.inspector.canvas.hintPin
                      : t.sessions.inspector.canvas.hintRegion,
                  style: theme.textTheme.bodySmall?.copyWith(
                    color: theme.colorScheme.onSurfaceVariant,
                  ),
                ),
              ),
          ],
        ),
      ),
    );
  }
}

/// Draws the transient crosshair / frame. Committed marks live in the page
/// itself, so nothing here has to survive a zoom.
class _CapturePainter extends CustomPainter {
  const _CapturePainter({this.cross, this.frame});

  final Offset? cross;
  final Rect? frame;

  @override
  void paint(Canvas canvas, Size size) {
    const rose = Color(0xFFF43F5E);
    final f = frame;
    if (f != null) {
      final r = Rect.fromLTRB(
        f.left * size.width,
        f.top * size.height,
        f.right * size.width,
        f.bottom * size.height,
      );
      canvas
        ..drawRect(r, Paint()..color = const Color(0x1FF43F5E))
        ..drawRect(
          r,
          Paint()
            ..color = rose
            ..style = PaintingStyle.stroke
            ..strokeWidth = 2,
        );
    }
    final c = cross;
    if (c != null) {
      final p = Offset(c.dx * size.width, c.dy * size.height);
      final line = Paint()
        ..color = rose
        ..strokeWidth = 1.5;
      // Full-width guides: the target stays visible beside the fingertip.
      canvas
        ..drawLine(Offset(0, p.dy), Offset(size.width, p.dy), line)
        ..drawLine(Offset(p.dx, 0), Offset(p.dx, size.height), line)
        ..drawCircle(
          p,
          14,
          Paint()
            ..color = rose
            ..style = PaintingStyle.stroke
            ..strokeWidth = 2,
        )
        ..drawCircle(p, 3, Paint()..color = rose);
    }
  }

  @override
  bool shouldRepaint(_CapturePainter old) =>
      old.cross != cross || old.frame != frame;
}

/// Per-mark notes + the overall message, as a sheet so the preview keeps the
/// screen while marking.
class _NotesSheet extends StatefulWidget {
  const _NotesSheet({required this.marks, required this.messageCtl});

  final List<CanvasMark> marks;
  final TextEditingController messageCtl;

  @override
  State<_NotesSheet> createState() => _NotesSheetState();
}

class _NotesSheetState extends State<_NotesSheet> {
  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Padding(
      padding: EdgeInsets.only(
        bottom: MediaQuery.of(context).viewInsets.bottom,
      ),
      child: SafeArea(
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 12),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                t.sessions.inspector.canvas.notes,
                style: theme.textTheme.titleMedium,
              ),
              const SizedBox(height: 8),
              Flexible(
                child: ListView.builder(
                  shrinkWrap: true,
                  itemCount: widget.marks.length,
                  itemBuilder: (_, i) {
                    final m = widget.marks[i];
                    return Padding(
                      padding: const EdgeInsets.only(bottom: 10),
                      child: Row(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Container(
                            width: 22,
                            height: 22,
                            margin: const EdgeInsets.only(top: 10),
                            alignment: Alignment.center,
                            decoration: const BoxDecoration(
                              color: Color(0xFFF43F5E),
                              shape: BoxShape.circle,
                            ),
                            child: Text(
                              '${i + 1}',
                              style: const TextStyle(
                                color: Colors.white,
                                fontSize: 11,
                                fontWeight: FontWeight.bold,
                              ),
                            ),
                          ),
                          const SizedBox(width: 10),
                          Expanded(
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                if (m.selector.isNotEmpty)
                                  Text(
                                    m.selector,
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                    style: theme.textTheme.bodySmall?.copyWith(
                                      fontFamily: 'monospace',
                                      color: theme.colorScheme.onSurfaceVariant,
                                    ),
                                  ),
                                TextFormField(
                                  initialValue: m.note,
                                  decoration: InputDecoration(
                                    isDense: true,
                                    hintText: t.sessions.inspector.canvas
                                        .notePlaceholder,
                                  ),
                                  onChanged: (v) => m.note = v,
                                ),
                              ],
                            ),
                          ),
                        ],
                      ),
                    );
                  },
                ),
              ),
              const SizedBox(height: 8),
              TextField(
                controller: widget.messageCtl,
                minLines: 1,
                maxLines: 3,
                decoration: InputDecoration(
                  isDense: true,
                  hintText: t.sessions.inspector.canvas.messagePlaceholder,
                ),
              ),
              const SizedBox(height: 10),
              Align(
                alignment: Alignment.centerRight,
                child: FilledButton(
                  onPressed: () => Navigator.of(context).pop(),
                  child: Text(t.sessions.inspector.canvas.done),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
