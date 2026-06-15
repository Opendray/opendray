// This file is a tight physics + canvas port: per-axis vx/vy updates read
// far clearer as separate statements than as cascades, and the `*.0`
// literals are required to keep math.min/max in the double domain.
// ignore_for_file: cascade_invocations, prefer_int_literals
import 'dart:math' as math;

import 'package:flutter/material.dart';
import 'package:flutter/scheduler.dart';

import 'package:opendray/core/api/knowledge_api.dart';

// Obsidian-style force-directed knowledge graph — a faithful Flutter port
// of the web GraphCanvas (app/web/src/pages/Knowledge.tsx): node repulsion
// + edge springs + center gravity, cooled by an alpha that decays to rest
// and re-heats on interaction. O(n²) repulsion is fine for ~500 nodes
// because the sim stops ticking once cooled.
//
// Mobile adaptation of the interaction model: one-finger drag pans, pinch
// zooms (focal-stable), tap selects a node. (Web also supports dragging an
// individual node; on a phone that conflicts with panning, so it's left
// out — pan/zoom/tap is the idiomatic set.)

// Same palette as the web KIND_COLORS (-400 hex values).
const Map<String, Color> _kindColors = {
  'entity': Color(0xFF60A5FA),
  'project': Color(0xFFA78BFA),
  'fact': Color(0xFFD4D4D8),
  'playbook': Color(0xFFFBBF24),
  'skill': Color(0xFF34D399),
};

const _sim = (
  repulsion: 1600.0,
  spring: 0.05,
  springLength: 70.0,
  gravity: 0.03,
  damping: 0.82,
  alphaDecay: 0.985,
  alphaMin: 0.02,
);

String _colorKey(KnowledgeNode n) =>
    n.kind == 'entity' && n.entityType == 'project' ? 'project' : n.kind;

String _displayTitle(KnowledgeNode n) {
  if (n.kind == 'entity' && n.entityType == 'project' && n.scopeKey.isNotEmpty) {
    final parts = n.scopeKey.split('/');
    return parts.isNotEmpty && parts.last.isNotEmpty ? parts.last : n.title;
  }
  return n.title;
}

class _SimNode {
  _SimNode({
    required this.id,
    required this.x,
    required this.y,
    required this.r,
    required this.color,
    required this.label,
    required this.degree,
  });
  final String id;
  double x;
  double y;
  double vx = 0;
  double vy = 0;
  final double r;
  final Color color;
  final String label;
  final int degree;
}

class ForceGraphView extends StatefulWidget {
  const ForceGraphView({
    required this.nodes,
    required this.edges,
    required this.selectedId,
    required this.onSelect,
    super.key,
  });

  final List<KnowledgeNode> nodes;
  final List<KnowledgeEdge> edges;
  final String? selectedId;
  final void Function(String? id) onSelect;

  @override
  State<ForceGraphView> createState() => _ForceGraphViewState();
}

// Repaint signal for the painter — a ChangeNotifier we can tick from the
// frame loop (notifyListeners is protected on the base class).
class _Repaint extends ChangeNotifier {
  void tick() => notifyListeners();
}

class _ForceGraphViewState extends State<ForceGraphView>
    with SingleTickerProviderStateMixin {
  late final Ticker _ticker;
  final _repaint = _Repaint();

  List<_SimNode> _nodes = [];
  List<(int, int)> _edges = [];
  double _alpha = 1;
  // view transform: screen = center + offset + world * k
  Offset _offset = Offset.zero;
  double _k = 1;
  bool _needsRedraw = true;

  // gesture scratch
  late double _startK;
  late Offset _startWorldFocal;

  @override
  void initState() {
    super.initState();
    _build();
    _ticker = createTicker(_tick)..start();
  }

  @override
  void didUpdateWidget(ForceGraphView old) {
    super.didUpdateWidget(old);
    if (!identical(old.nodes, widget.nodes) ||
        !identical(old.edges, widget.edges)) {
      _build();
    }
    if (old.selectedId != widget.selectedId) {
      _alpha = math.max(_alpha, _sim.alphaMin);
      _needsRedraw = true;
    }
  }

  void _build() {
    final prev = {for (final n in _nodes) n.id: n};
    final degree = <String, int>{};
    for (final e in widget.edges) {
      degree[e.srcId] = (degree[e.srcId] ?? 0) + 1;
      degree[e.dstId] = (degree[e.dstId] ?? 0) + 1;
    }
    _nodes = [
      for (var i = 0; i < widget.nodes.length; i++)
        () {
          final n = widget.nodes[i];
          final d = degree[n.id] ?? 0;
          final old = prev[n.id];
          // Phyllotaxis spiral for fresh nodes — deterministic, even spread.
          final angle = i * 2.39996;
          final radius = 22 * math.sqrt(i + 1);
          return _SimNode(
            id: n.id,
            x: old?.x ?? math.cos(angle) * radius,
            y: old?.y ?? math.sin(angle) * radius,
            r: 4 + math.min(11.0, math.sqrt(d) * 2),
            color: _kindColors[_colorKey(n)] ?? _kindColors['fact']!,
            label: _displayTitle(n),
            degree: d,
          );
        }(),
    ];
    final byId = {for (var i = 0; i < _nodes.length; i++) _nodes[i].id: i};
    _edges = [
      for (final e in widget.edges)
        if (byId[e.srcId] != null && byId[e.dstId] != null)
          (byId[e.srcId]!, byId[e.dstId]!),
    ];
    _alpha = 1;
    _needsRedraw = true;
  }

  void _tick(Duration _) {
    final ns = _nodes;
    if (_alpha > _sim.alphaMin && ns.isNotEmpty) {
      _alpha *= _sim.alphaDecay;
      final a = _alpha;
      // pairwise repulsion
      for (var i = 0; i < ns.length; i++) {
        final ni = ns[i];
        for (var j = i + 1; j < ns.length; j++) {
          final nj = ns[j];
          var dx = ni.x - nj.x;
          var dy = (nj.y == ni.y && dx == 0) ? 0.1 : ni.y - nj.y;
          final d2 = math.max(dx * dx + dy * dy, 64.0);
          final f = (_sim.repulsion * a) / d2;
          final d = math.sqrt(d2);
          dx /= d;
          dy /= d;
          ni.vx += dx * f;
          ni.vy += dy * f;
          nj.vx -= dx * f;
          nj.vy -= dy * f;
        }
      }
      // edge springs
      for (final e in _edges) {
        final na = ns[e.$1];
        final nb = ns[e.$2];
        final dx = nb.x - na.x;
        final dy = nb.y - na.y;
        final d = math.max(math.sqrt(dx * dx + dy * dy), 1.0);
        final f = _sim.spring * a * (d - _sim.springLength);
        final fx = (dx / d) * f;
        final fy = (dy / d) * f;
        na.vx += fx;
        na.vy += fy;
        nb.vx -= fx;
        nb.vy -= fy;
      }
      // center gravity + integration
      for (final n in ns) {
        n.vx -= n.x * _sim.gravity * a;
        n.vy -= n.y * _sim.gravity * a;
        n.vx *= _sim.damping;
        n.vy *= _sim.damping;
        n.x += n.vx;
        n.y += n.vy;
      }
      _repaint.tick();
    } else if (_needsRedraw) {
      _needsRedraw = false;
      _repaint.tick();
    }
  }

  Offset _toWorld(Offset screen, Size size) {
    final center = Offset(size.width / 2, size.height / 2);
    return (screen - center - _offset) / _k;
  }

  int _hitTest(Offset screen, Size size) {
    final p = _toWorld(screen, size);
    for (var i = _nodes.length - 1; i >= 0; i--) {
      final n = _nodes[i];
      final dx = n.x - p.dx;
      final dy = n.y - p.dy;
      final hr = n.r + 6 / _k; // a touch more slop than web for fingers
      if (dx * dx + dy * dy <= hr * hr) return i;
    }
    return -1;
  }

  @override
  void dispose() {
    _ticker.dispose();
    _repaint.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final fg = Theme.of(context)
        .colorScheme
        .onSurface
        .withValues(alpha: 0.85);
    return LayoutBuilder(
      builder: (context, constraints) {
        final size = Size(constraints.maxWidth, constraints.maxHeight);
        return GestureDetector(
          onScaleStart: (d) {
            _startK = _k;
            final center = Offset(size.width / 2, size.height / 2);
            _startWorldFocal =
                (d.localFocalPoint - center - _offset) / _k;
          },
          onScaleUpdate: (d) {
            final center = Offset(size.width / 2, size.height / 2);
            final newK = (_startK * d.scale).clamp(0.15, 5.0);
            // Keep the world point under the focal stationary while the
            // focal itself moves (this folds pan + zoom into one handler).
            setState(() {
              _k = newK;
              _offset = d.localFocalPoint - center - _startWorldFocal * newK;
              _alpha = math.max(_alpha, _sim.alphaMin);
              _needsRedraw = true;
            });
          },
          onTapUp: (d) {
            final i = _hitTest(d.localPosition, size);
            widget.onSelect(i >= 0 ? _nodes[i].id : null);
          },
          child: CustomPaint(
            size: size,
            painter: _GraphPainter(
              repaint: _repaint,
              state: this,
              fg: fg,
            ),
          ),
        );
      },
    );
  }
}

class _GraphPainter extends CustomPainter {
  _GraphPainter({
    required Listenable repaint,
    required this.state,
    required this.fg,
  }) : super(repaint: repaint);

  final _ForceGraphViewState state;
  final Color fg;

  @override
  void paint(Canvas canvas, Size size) {
    final nodes = state._nodes;
    final edges = state._edges;
    final k = state._k;
    canvas.save();
    canvas.translate(
        size.width / 2 + state._offset.dx, size.height / 2 + state._offset.dy);
    canvas.scale(k);

    final selIdx = state.widget.selectedId == null
        ? -1
        : nodes.indexWhere((n) => n.id == state.widget.selectedId);

    // edges
    final edgePaint = Paint()
      ..color = fg.withValues(alpha: 0.16)
      ..strokeWidth = 1 / k
      ..style = PaintingStyle.stroke;
    final path = Path();
    for (final e in edges) {
      final a = nodes[e.$1];
      final b = nodes[e.$2];
      path.moveTo(a.x, a.y);
      path.lineTo(b.x, b.y);
    }
    canvas.drawPath(path, edgePaint);

    // nodes
    final strokePaint = Paint()
      ..color = fg
      ..strokeWidth = 2 / k
      ..style = PaintingStyle.stroke;
    for (var i = 0; i < nodes.length; i++) {
      final n = nodes[i];
      canvas.drawCircle(Offset(n.x, n.y), n.r, Paint()..color = n.color);
      if (i == selIdx) {
        canvas.drawCircle(Offset(n.x, n.y), n.r, strokePaint);
      }
    }

    // labels — appear as you zoom in; hubs + the active node always show
    final fontPx = math.max(10 / k, 4.0);
    for (var i = 0; i < nodes.length; i++) {
      final n = nodes[i];
      final show = i == selIdx || k >= 1.2 || n.degree >= 5;
      if (!show) continue;
      final tp = TextPainter(
        text: TextSpan(
          text: n.label.length > 42 ? n.label.substring(0, 42) : n.label,
          style: TextStyle(
            color: fg.withValues(alpha: i == selIdx ? 1 : 0.75),
            fontSize: fontPx,
          ),
        ),
        textDirection: TextDirection.ltr,
      )..layout();
      tp.paint(canvas, Offset(n.x - tp.width / 2, n.y + n.r + 3 / k));
    }
    canvas.restore();
  }

  @override
  bool shouldRepaint(_GraphPainter old) => false; // repaint via Listenable
}
