import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/canvas_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// CanvasTab — the session inspector's Canvas: the visual channel between the
// agent and the operator. The agent renders a self-contained HTML page with
// `canvas_render`; here the operator sees it, switches which canvas is FOCUSED
// (so plain talk in the terminal resolves to it), pins/region-marks it, and
// sends those marks back into the session.
//
// Mirrors the web Inspector's Canvas tab, reshaped for a phone: the canvas list
// is a horizontal chip rail, the preview owns the screen, and annotating is a
// deliberate mode (tap to pin, drag to frame) so scrolling the design stays the
// default gesture.

/// The probe injected into every preview so a pin resolves the element under
/// the finger and a region resolves the framed block plus what it contains —
/// the agent gets real DOM, not just coordinates. Same logic as the web panel.
const _probeScript = r'''
window.__odCssPath = function(el){
  if(!(el instanceof Element)) return '';
  var path=[];
  while(el && el.nodeType===1 && path.length<5){
    var sel=el.nodeName.toLowerCase();
    if(el.id){ sel+='#'+el.id; path.unshift(sel); break; }
    var cls=(typeof el.className==='string'?el.className:'').trim();
    if(cls){ sel+='.'+cls.split(/\s+/).slice(0,2).join('.'); }
    var p=el.parentNode;
    if(p && p.children){
      var same=Array.prototype.filter.call(p.children,function(c){return c.nodeName===el.nodeName;});
      if(same.length>1){ sel+=':nth-child('+(Array.prototype.indexOf.call(p.children,el)+1)+')'; }
    }
    path.unshift(sel);
    el=el.parentNode;
  }
  return path.join(' > ');
};
window.__odProbe = function(x,y){
  var el=document.elementFromPoint(x,y);
  return JSON.stringify({selector: el?window.__odCssPath(el):'', html: el?el.outerHTML.slice(0,1500):''});
};
window.__odProbeRect = function(x,y,w,h){
  function fits(el){var r=el.getBoundingClientRect(); return r.left<=x+2 && r.top<=y+2 && r.right>=x+w-2 && r.bottom>=y+h-2;}
  var box=document.elementFromPoint(x+w/2, y+h/2);
  while(box && box.parentElement && box!==document.body && !fits(box)) box=box.parentElement;
  var kids=[];
  if(box){
    var all=box.querySelectorAll('*');
    for(var i=0;i<all.length && kids.length<10;i++){
      var c=all[i], r=c.getBoundingClientRect();
      if(r.width<12 || r.height<12) continue;
      if(r.right<x || r.left>x+w || r.bottom<y || r.top>y+h) continue;
      var cls=(typeof c.className==='string'?c.className:'').trim();
      var label=c.nodeName.toLowerCase()+(cls?'.'+cls.split(/\s+/).slice(0,2).join('.'):'');
      var txt=(c.textContent||'').replace(/\s+/g,' ').trim().slice(0,32);
      kids.push(label+(txt?' "'+txt+'"':''));
    }
  }
  return JSON.stringify({selector: box?window.__odCssPath(box):'', html: box?box.outerHTML.slice(0,1500):'', kids: kids});
};
''';

IconData canvasKindIcon(CanvasKind kind) => switch (kind) {
      CanvasKind.flow => Icons.account_tree_outlined,
      CanvasKind.mindmap => Icons.hub_outlined,
      CanvasKind.graph => Icons.share_outlined,
      CanvasKind.doc => Icons.description_outlined,
      CanvasKind.ui => Icons.web_asset_outlined,
    };

String canvasKindLabel(CanvasKind kind) => switch (kind) {
      CanvasKind.flow => t.sessions.inspector.canvas.kindFlow,
      CanvasKind.mindmap => t.sessions.inspector.canvas.kindMindmap,
      CanvasKind.graph => t.sessions.inspector.canvas.kindGraph,
      CanvasKind.doc => t.sessions.inspector.canvas.kindDoc,
      CanvasKind.ui => t.sessions.inspector.canvas.kindUi,
    };

/// A mark the operator has placed but not yet sent.
class _Mark {
  _Mark({
    required this.kind,
    required this.x,
    required this.y,
    this.w = 0,
    this.h = 0,
  });

  final String kind;
  final double x;
  final double y;
  final double w;
  final double h;

  // Resolved from the page by the injected probe right after the mark is made.
  String selector = '';
  String html = '';
  List<String> elements = const [];
  String note = '';
}

enum _Mode { view, pin, region }

class CanvasTab extends ConsumerStatefulWidget {
  const CanvasTab({required this.sessionId, required this.cwd, super.key});

  final String sessionId;
  final String cwd;

  @override
  ConsumerState<CanvasTab> createState() => _CanvasTabState();
}

class _CanvasTabState extends ConsumerState<CanvasTab>
    with AutomaticKeepAliveClientMixin {
  AsyncValue<List<CanvasSummary>> _list = const AsyncValue.loading();
  CanvasArtifact? _artifact;
  String? _selectedId;
  bool _loadingArtifact = false;

  _Mode _mode = _Mode.view;
  final List<_Mark> _marks = [];
  Rect? _drag;
  Offset? _dragStart;

  final _messageCtl = TextEditingController();
  final _requestCtl = TextEditingController();
  bool _sending = false;
  bool _requesting = false;

  InAppWebViewController? _web;
  Size _previewSize = Size.zero;
  StreamSubscription<CanvasEvent>? _events;

  /// The slug we last told the gateway we're on, so a silent assert can't
  /// double-send or race the explicit (notifying) one.
  String? _assertedSlug;

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
    final events = ref.read(canvasEventsProvider);
    _events = events?.stream.listen(_onEvent);
  }

  @override
  void dispose() {
    unawaited(_events?.cancel());
    _messageCtl.dispose();
    _requestCtl.dispose();
    super.dispose();
  }

  void _onEvent(CanvasEvent ev) {
    if (ev.cwd != widget.cwd || !mounted) return;
    if (ev.isUpdated) {
      unawaited(_load(keepSelection: true));
    } else if (ev.isFocus && ev.slug.isNotEmpty && ev.slug != _assertedSlug) {
      // Another surface (the web panel) switched canvases — follow it silently;
      // it already notified the session.
      _assertedSlug = ev.slug;
      final hit = _list.valueOrNull?.where((a) => a.slug == ev.slug).firstOrNull;
      if (hit != null) unawaited(_select(hit, notify: false));
    }
  }

  Future<void> _load({bool keepSelection = false}) async {
    if (!keepSelection) setState(() => _list = const AsyncValue.loading());
    try {
      final items = await ref.read(canvasApiProvider).list(widget.cwd);
      if (!mounted) return;
      setState(() => _list = AsyncValue.data(items));
      final current = _selectedId;
      final stillThere = items.any((a) => a.id == current);
      if (items.isEmpty) {
        setState(() {
          _selectedId = null;
          _artifact = null;
        });
      } else if (!stillThere) {
        await _select(items.first, notify: false);
      } else if (keepSelection && current != null) {
        // A re-render bumped the version — reload the html in place.
        await _loadArtifact(current);
      }
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() => _list = AsyncValue.error(e, StackTrace.current));
    } on Object catch (e, st) {
      if (!mounted) return;
      setState(() => _list = AsyncValue.error(e, st));
    }
  }

  /// Select a canvas. [notify] marks an explicit operator switch, which also
  /// seeds the "[Canvas focus]" note into the session so the agent knows which
  /// canvas plain conversation refers to.
  Future<void> _select(CanvasSummary item, {required bool notify}) async {
    setState(() {
      _selectedId = item.id;
      _marks.clear();
      _mode = _Mode.view;
    });
    await _loadArtifact(item.id);
    if (_assertedSlug == item.slug) return;
    _assertedSlug = item.slug;
    try {
      await ref.read(canvasApiProvider).setFocus(
            cwd: widget.cwd,
            slug: item.slug,
            sessionId: notify ? widget.sessionId : null,
            notify: notify,
          );
    } on Object catch (_) {
      // Focus is a convenience; a failure must not block viewing.
    }
  }

  Future<void> _loadArtifact(String id) async {
    setState(() => _loadingArtifact = true);
    try {
      final a = await ref.read(canvasApiProvider).get(id);
      if (!mounted) return;
      setState(() => _artifact = a);
      await _web?.loadData(data: _withProbe(a.html), mimeType: 'text/html');
    } on Object catch (_) {
      // Leave the previous preview up rather than blanking the screen.
    } finally {
      if (mounted) setState(() => _loadingArtifact = false);
    }
  }

  String _withProbe(String html) {
    const tag = '<script>$_probeScript</script>';
    if (html.contains('</body>')) return html.replaceFirst('</body>', '$tag</body>');
    return '$html$tag';
  }

  // ── annotating ────────────────────────────────────────────────────

  Future<void> _addPin(Offset local) async {
    final size = _previewSize;
    if (size.isEmpty) return;
    final mark = _Mark(
      kind: 'pin',
      x: local.dx / size.width * 100,
      y: local.dy / size.height * 100,
    );
    final probed = await _probe('window.__odProbe(${local.dx}, ${local.dy})');
    mark
      ..selector = probed?['selector'] as String? ?? ''
      ..html = probed?['html'] as String? ?? '';
    if (!mounted) return;
    setState(() => _marks.add(mark));
  }

  Future<void> _addRegion(Rect r) async {
    final size = _previewSize;
    if (size.isEmpty) return;
    final mark = _Mark(
      kind: 'region',
      x: r.left / size.width * 100,
      y: r.top / size.height * 100,
      w: r.width / size.width * 100,
      h: r.height / size.height * 100,
    );
    final probed = await _probe(
      'window.__odProbeRect(${r.left}, ${r.top}, ${r.width}, ${r.height})',
    );
    mark
      ..selector = probed?['selector'] as String? ?? ''
      ..html = probed?['html'] as String? ?? ''
      ..elements = (probed?['kids'] as List?)?.whereType<String>().toList() ??
          const [];
    if (!mounted) return;
    setState(() => _marks.add(mark));
  }

  Future<Map<String, dynamic>?> _probe(String source) async {
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
      // A page that blocks the probe still gives us coordinates.
    }
    return null;
  }

  // ── sending ───────────────────────────────────────────────────────

  Future<void> _send() async {
    final artifact = _artifact;
    if (artifact == null) return;
    final message = _messageCtl.text.trim();
    if (_marks.isEmpty && message.isEmpty) {
      _snack(t.sessions.inspector.canvas.nothingToSend);
      return;
    }
    setState(() => _sending = true);
    try {
      await ref.read(canvasApiProvider).submitFeedback(
            id: artifact.id,
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
      setState(() {
        _marks.clear();
        _mode = _Mode.view;
      });
      _messageCtl.clear();
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

  Future<void> _sendRequest() async {
    final prompt = _requestCtl.text.trim();
    if (prompt.isEmpty) return;
    final target = _list.valueOrNull
        ?.where((a) => a.id == _selectedId)
        .firstOrNull;
    setState(() => _requesting = true);
    try {
      await ref.read(canvasApiProvider).requestDesign(
            sessionId: widget.sessionId,
            prompt: prompt,
            slug: target?.slug,
            title: target?.title,
          );
      if (!mounted) return;
      _requestCtl.clear();
      _snack(t.sessions.inspector.canvas.requested);
    } on ApiException catch (e) {
      _snack(t.sessions.inspector.shared.insertFailedApi(
        status: '${e.statusCode}',
        message: e.message,
      ));
    } on Object catch (e) {
      _snack(t.sessions.inspector.shared.insertFailedGeneric(error: '$e'));
    } finally {
      if (mounted) setState(() => _requesting = false);
    }
  }

  Future<void> _deleteSelected() async {
    final artifact = _artifact;
    if (artifact == null) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.sessions.inspector.canvas.deleteTitle),
        content: Text(
          t.sessions.inspector.canvas.deleteBody(
            title: artifact.title.isEmpty ? artifact.slug : artifact.title,
          ),
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(MaterialLocalizations.of(ctx).cancelButtonLabel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.sessions.inspector.canvas.delete),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ref.read(canvasApiProvider).delete(artifact.id);
      _assertedSlug = null;
      await _load();
    } on Object catch (e) {
      _snack(t.sessions.inspector.shared.insertFailedGeneric(error: '$e'));
    }
  }

  void _snack(String text) {
    if (!mounted) return;
    ScaffoldMessenger.of(context)
        .showSnackBar(SnackBar(content: Text(text)));
  }

  // ── build ─────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    super.build(context);
    final theme = Theme.of(context);
    return _list.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorBody(error: '$e', onRetry: () => unawaited(_load())),
      data: (items) {
        if (items.isEmpty) return _EmptyBody(onRequest: _requestBar(theme));
        return Column(
          children: [
            _canvasRail(items, theme),
            _focusLine(theme),
            Divider(height: 1, color: theme.dividerColor),
            Expanded(child: _preview(theme)),
            if (_marks.isNotEmpty) _marksList(theme),
            _composer(theme),
          ],
        );
      },
    );
  }

  Widget _canvasRail(List<CanvasSummary> items, ThemeData theme) {
    return SizedBox(
      height: 44,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        itemCount: items.length,
        separatorBuilder: (_, __) => const SizedBox(width: 6),
        itemBuilder: (_, i) {
          final a = items[i];
          final active = a.id == _selectedId;
          return ChoiceChip(
            selected: active,
            avatar: Icon(canvasKindIcon(a.kind), size: 16),
            label: Text(a.label, overflow: TextOverflow.ellipsis),
            onSelected: (_) => unawaited(_select(a, notify: true)),
          );
        },
      ),
    );
  }

  Widget _focusLine(ThemeData theme) {
    final a = _artifact;
    if (a == null) return const SizedBox.shrink();
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 0, 4, 6),
      child: Row(
        children: [
          Expanded(
            child: Text(
              t.sessions.inspector.canvas.focusLine(
                title: a.title.isEmpty ? a.slug : a.title,
                kind: canvasKindLabel(a.kind),
              ),
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.onSurfaceVariant,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          IconButton(
            icon: const Icon(Icons.delete_outline, size: 18),
            tooltip: t.sessions.inspector.canvas.delete,
            onPressed: () => unawaited(_deleteSelected()),
          ),
        ],
      ),
    );
  }

  Widget _preview(ThemeData theme) {
    final artifact = _artifact;
    if (artifact == null) {
      return const Center(child: CircularProgressIndicator());
    }
    return Column(
      children: [
        _modeBar(theme),
        Expanded(
          child: LayoutBuilder(
            builder: (context, constraints) {
              _previewSize = Size(constraints.maxWidth, constraints.maxHeight);
              return Stack(
                children: [
                  InAppWebView(
                    // While annotating, the overlay below sits on top with
                    // opaque hit-testing, so gestures never reach the page —
                    // no need to toggle the webview's own scrolling.
                    initialSettings: InAppWebViewSettings(
                      transparentBackground: true,
                      supportZoom: false,
                    ),
                    onWebViewCreated: (c) {
                      _web = c;
                      unawaited(
                        c.loadData(
                          data: _withProbe(artifact.html),
                          mimeType: 'text/html',
                        ),
                      );
                    },
                  ),
                  if (_mode != _Mode.view) _annotationLayer(),
                  ..._markWidgets(),
                  if (_loadingArtifact)
                    const Positioned(
                      top: 8,
                      right: 8,
                      child: SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      ),
                    ),
                ],
              );
            },
          ),
        ),
      ],
    );
  }

  Widget _modeBar(ThemeData theme) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 4, 12, 8),
      child: Row(
        children: [
          Expanded(
            child: SegmentedButton<_Mode>(
              showSelectedIcon: false,
              style: const ButtonStyle(visualDensity: VisualDensity.compact),
              segments: [
                ButtonSegment(
                  value: _Mode.view,
                  icon: const Icon(Icons.visibility_outlined, size: 16),
                  label: Text(t.sessions.inspector.canvas.modeView),
                ),
                ButtonSegment(
                  value: _Mode.pin,
                  icon: const Icon(Icons.push_pin_outlined, size: 16),
                  label: Text(t.sessions.inspector.canvas.modePin),
                ),
                ButtonSegment(
                  value: _Mode.region,
                  icon: const Icon(Icons.crop_free, size: 16),
                  label: Text(t.sessions.inspector.canvas.modeRegion),
                ),
              ],
              selected: {_mode},
              onSelectionChanged: (s) => setState(() => _mode = s.first),
            ),
          ),
        ],
      ),
    );
  }

  /// Captures taps (pin) and drags (region) over the preview. Only mounted
  /// while annotating, so scrolling the design stays the default gesture.
  Widget _annotationLayer() {
    return Positioned.fill(
      child: GestureDetector(
        behavior: HitTestBehavior.opaque,
        onTapUp: _mode == _Mode.pin
            ? (d) => unawaited(_addPin(d.localPosition))
            : null,
        onPanStart: _mode == _Mode.region
            ? (d) => setState(() {
                  _dragStart = d.localPosition;
                  _drag = Rect.fromPoints(d.localPosition, d.localPosition);
                })
            : null,
        onPanUpdate: _mode == _Mode.region
            ? (d) => setState(() {
                  final start = _dragStart;
                  if (start != null) {
                    _drag = Rect.fromPoints(start, d.localPosition);
                  }
                })
            : null,
        onPanEnd: _mode == _Mode.region
            ? (_) {
                final r = _drag;
                setState(() {
                  _drag = null;
                  _dragStart = null;
                });
                if (r != null && r.width > 12 && r.height > 12) {
                  unawaited(_addRegion(r));
                }
              }
            : null,
        child: _drag == null
            ? const SizedBox.expand()
            : CustomPaint(painter: _DragPainter(_drag!), child: const SizedBox.expand()),
      ),
    );
  }

  List<Widget> _markWidgets() {
    final size = _previewSize;
    if (size.isEmpty) return const [];
    final out = <Widget>[];
    for (var i = 0; i < _marks.length; i++) {
      final m = _marks[i];
      final left = m.x / 100 * size.width;
      final top = m.y / 100 * size.height;
      if (m.kind == 'region') {
        out.add(
          Positioned(
            left: left,
            top: top,
            width: m.w / 100 * size.width,
            height: m.h / 100 * size.height,
            child: IgnorePointer(
              child: DecoratedBox(
                decoration: BoxDecoration(
                  border: Border.all(color: const Color(0xFFF43F5E), width: 2),
                  color: const Color(0x1FF43F5E),
                ),
              ),
            ),
          ),
        );
      }
      out.add(
        Positioned(
          left: left - 11,
          top: top - 11,
          child: IgnorePointer(child: _Badge(n: i + 1)),
        ),
      );
    }
    return out;
  }

  Widget _marksList(ThemeData theme) {
    return ConstrainedBox(
      constraints: const BoxConstraints(maxHeight: 168),
      child: ListView.builder(
        shrinkWrap: true,
        padding: const EdgeInsets.symmetric(horizontal: 12),
        itemCount: _marks.length,
        itemBuilder: (_, i) {
          final m = _marks[i];
          return Padding(
            padding: const EdgeInsets.only(bottom: 6),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.only(top: 10),
                  child: _Badge(n: i + 1),
                ),
                const SizedBox(width: 8),
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
                      TextField(
                        decoration: InputDecoration(
                          isDense: true,
                          hintText: t.sessions.inspector.canvas.notePlaceholder,
                        ),
                        onChanged: (v) => m.note = v,
                      ),
                    ],
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.close, size: 16),
                  onPressed: () => setState(() => _marks.removeAt(i)),
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  Widget _composer(ThemeData theme) {
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 6, 12, 8),
        child: Column(
          children: [
            if (_marks.isNotEmpty || _mode != _Mode.view)
              Row(
                children: [
                  Expanded(
                    child: TextField(
                      controller: _messageCtl,
                      minLines: 1,
                      maxLines: 3,
                      decoration: InputDecoration(
                        isDense: true,
                        hintText: t.sessions.inspector.canvas.messagePlaceholder,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
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
              )
            else
              _requestBar(theme),
          ],
        ),
      ),
    );
  }

  Widget _requestBar(ThemeData theme) {
    return Row(
      children: [
        Expanded(
          child: TextField(
            controller: _requestCtl,
            decoration: InputDecoration(
              isDense: true,
              hintText: t.sessions.inspector.canvas.requestPlaceholder,
            ),
            onSubmitted: (_) => unawaited(_sendRequest()),
          ),
        ),
        const SizedBox(width: 8),
        FilledButton.icon(
          onPressed: _requesting ? null : () => unawaited(_sendRequest()),
          icon: const Icon(Icons.auto_awesome, size: 16),
          label: Text(t.sessions.inspector.canvas.requestSend),
        ),
      ],
    );
  }
}

class _Badge extends StatelessWidget {
  const _Badge({required this.n});

  final int n;

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 22,
      height: 22,
      alignment: Alignment.center,
      decoration: const BoxDecoration(
        color: Color(0xFFF43F5E),
        shape: BoxShape.circle,
        boxShadow: [BoxShadow(color: Color(0x66000000), blurRadius: 4)],
      ),
      child: Text(
        '$n',
        style: const TextStyle(
          color: Colors.white,
          fontSize: 11,
          fontWeight: FontWeight.bold,
        ),
      ),
    );
  }
}

class _DragPainter extends CustomPainter {
  const _DragPainter(this.rect);

  final Rect rect;

  @override
  void paint(Canvas canvas, Size size) {
    final fill = Paint()..color = const Color(0x1FF43F5E);
    final stroke = Paint()
      ..color = const Color(0xFFF43F5E)
      ..style = PaintingStyle.stroke
      ..strokeWidth = 2;
    canvas
      ..drawRect(rect, fill)
      ..drawRect(rect, stroke);
  }

  @override
  bool shouldRepaint(_DragPainter old) => old.rect != rect;
}

class _EmptyBody extends StatelessWidget {
  const _EmptyBody({required this.onRequest});

  final Widget onRequest;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      children: [
        Expanded(
          child: Center(
            child: Padding(
              padding: const EdgeInsets.all(24),
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  Icon(
                    Icons.palette_outlined,
                    size: 48,
                    color: theme.colorScheme.primary,
                  ),
                  const SizedBox(height: 16),
                  Text(
                    t.sessions.inspector.canvas.emptyTitle,
                    style: theme.textTheme.titleMedium,
                    textAlign: TextAlign.center,
                  ),
                  const SizedBox(height: 8),
                  Text(
                    t.sessions.inspector.canvas.emptyBlurb,
                    style: theme.textTheme.bodySmall,
                    textAlign: TextAlign.center,
                  ),
                ],
              ),
            ),
          ),
        ),
        SafeArea(
          top: false,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(12, 0, 12, 8),
            child: onRequest,
          ),
        ),
      ],
    );
  }
}

class _ErrorBody extends StatelessWidget {
  const _ErrorBody({required this.error, required this.onRetry});

  final String error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(error, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              onPressed: onRetry,
              icon: const Icon(Icons.refresh),
              label: Text(t.sessions.inspector.shared.refresh),
            ),
          ],
        ),
      ),
    );
  }
}
