import 'dart:async';
import 'dart:convert';

import 'package:dio/dio.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/dio_provider.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

// Canvas — opendray's visual channel between an agent and the operator. The
// agent renders a self-contained HTML page with the `canvas_render` MCP tool;
// the operator sees it, pins/region-marks it, and those marks are seeded back
// into the session as a prompt. Scoped by cwd (project), exactly like the other
// Inspector tools. Mirrors app/shared/src/lib/canvas.ts.

/// What a canvas IS. Not only screen mocks — an agent also draws flowcharts,
/// mind maps and relationship diagrams.
enum CanvasKind { ui, flow, mindmap, graph, doc }

CanvasKind canvasKindFrom(String? raw) => switch (raw) {
      'flow' => CanvasKind.flow,
      'mindmap' => CanvasKind.mindmap,
      'graph' => CanvasKind.graph,
      'doc' => CanvasKind.doc,
      _ => CanvasKind.ui,
    };

String canvasKindWire(CanvasKind kind) => kind.name;

/// One canvas in a project, without its (potentially large) html body.
class CanvasSummary {
  const CanvasSummary({
    required this.id,
    required this.cwd,
    required this.slug,
    required this.title,
    required this.kind,
    required this.version,
  });

  factory CanvasSummary.fromJson(Map<String, dynamic> json) => CanvasSummary(
        id: json['id'] as String? ?? '',
        cwd: json['cwd'] as String? ?? '',
        slug: json['slug'] as String? ?? '',
        title: json['title'] as String? ?? '',
        kind: canvasKindFrom(json['kind'] as String?),
        version: (json['version'] as num?)?.toInt() ?? 1,
      );

  final String id;
  final String cwd;
  final String slug;
  final String title;
  final CanvasKind kind;
  final int version;

  /// What to show in a list — the agent's title, else the slug.
  String get label => title.isNotEmpty ? title : slug;
}

/// A canvas WITH its html body, for rendering.
class CanvasArtifact extends CanvasSummary {
  const CanvasArtifact({
    required super.id,
    required super.cwd,
    required super.slug,
    required super.title,
    required super.kind,
    required super.version,
    required this.html,
  });

  factory CanvasArtifact.fromJson(Map<String, dynamic> json) => CanvasArtifact(
        id: json['id'] as String? ?? '',
        cwd: json['cwd'] as String? ?? '',
        slug: json['slug'] as String? ?? '',
        title: json['title'] as String? ?? '',
        kind: canvasKindFrom(json['kind'] as String?),
        version: (json['version'] as num?)?.toInt() ?? 1,
        html: json['html'] as String? ?? '',
      );

  final String html;
}

/// One operator mark on a canvas. Coordinates are percentages of the preview
/// (0–100) so they survive a resize, exactly like the web panel.
class CanvasAnnotation {
  const CanvasAnnotation({
    required this.n,
    required this.kind,
    required this.note,
    this.selector = '',
    this.html = '',
    this.elements = const [],
    this.x = 0,
    this.y = 0,
    this.w = 0,
    this.h = 0,
  });

  final int n;

  /// 'pin' (a point on an element) or 'region' (a dragged rectangle).
  final String kind;
  final String note;
  final String selector;
  final String html;
  final List<String> elements;
  final double x;
  final double y;
  final double w;
  final double h;

  Map<String, dynamic> toJson() => {
        'n': n,
        'kind': kind,
        'note': note,
        if (selector.isNotEmpty) 'selector': selector,
        if (html.isNotEmpty) 'html': html,
        if (elements.isNotEmpty) 'elements': elements,
        'x': x,
        'y': y,
        if (kind == 'region') 'w': w,
        if (kind == 'region') 'h': h,
      };
}

/// The project's focused canvas — what the operator means by "this canvas" in
/// plain session conversation.
class CanvasFocus {
  const CanvasFocus({required this.slug, this.title = '', this.kind});

  factory CanvasFocus.fromJson(Map<String, dynamic> json) => CanvasFocus(
        slug: json['slug'] as String? ?? '',
        title: json['title'] as String? ?? '',
        kind: json['kind'] == null
            ? null
            : canvasKindFrom(json['kind'] as String?),
      );

  final String slug;
  final String title;
  final CanvasKind? kind;
}

/// REST client for the gateway's /api/v1/canvas surface.
class CanvasApi {
  CanvasApi(this._dio);

  final Dio _dio;

  /// All canvases in a project, newest first (no html bodies).
  Future<List<CanvasSummary>> list(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/canvas',
        queryParameters: {'cwd': cwd},
      );
      final raw = res.data?['artifacts'];
      if (raw is! List) return const [];
      return raw
          .whereType<Map<String, dynamic>>()
          .map(CanvasSummary.fromJson)
          .toList();
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// One canvas with its html, for rendering.
  Future<CanvasArtifact> get(String id) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>('/api/v1/canvas/$id');
      return CanvasArtifact.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Record which canvas the operator is working on. [notify] (an explicit
  /// switch, not a mount) also seeds a one-line focus note into the session so
  /// the agent follows along in ordinary conversation.
  Future<CanvasFocus> setFocus({
    required String cwd,
    required String slug,
    String? sessionId,
    bool notify = false,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/canvas/focus',
        data: {
          'cwd': cwd,
          'slug': slug,
          if (sessionId != null) 'session_id': sessionId,
          'notify': notify,
        },
      );
      return CanvasFocus.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// The project's focused canvas (empty slug when none).
  Future<CanvasFocus> getFocus(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/canvas/focus',
        queryParameters: {'cwd': cwd},
      );
      return CanvasFocus.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Seed a "design/draw this" request into the session. With [slug] the agent
  /// updates that canvas in place; without it, [kind] says what to draw new.
  Future<void> requestDesign({
    required String sessionId,
    required String cwd,
    required String prompt,
    String? slug,
    String? title,
    CanvasKind? kind,
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/canvas/request',
        data: {
          'session_id': sessionId,
          'cwd': cwd,
          'prompt': prompt,
          if (slug != null) 'slug': slug,
          if (title != null) 'title': title,
          if (kind != null) 'kind': canvasKindWire(kind),
        },
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Seed the operator's on-canvas annotations into the session as a prompt.
  /// The gateway formats the prompt — mobile only posts the marks.
  Future<void> submitFeedback({
    required String id,
    required String sessionId,
    String? message,
    List<CanvasAnnotation> annotations = const [],
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/canvas/$id/feedback',
        data: {
          'session_id': sessionId,
          if (message != null && message.isNotEmpty) 'message': message,
          'annotations': annotations.map((a) => a.toJson()).toList(),
        },
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// The project's canvas design system (empty when unset).
  Future<CanvasDesignSystem> getDesign(String cwd) async {
    try {
      final res = await _dio.get<Map<String, dynamic>>(
        '/api/v1/canvas/design',
        queryParameters: {'cwd': cwd},
      );
      return CanvasDesignSystem.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Replaces the project's canvas design system.
  Future<CanvasDesignSystem> setDesign({
    required String cwd,
    required Map<String, String> tokens,
    required Map<String, String> tokensDark,
    required String notes,
  }) async {
    try {
      final res = await _dio.post<Map<String, dynamic>>(
        '/api/v1/canvas/design',
        data: {
          'cwd': cwd,
          'tokens': tokens,
          'tokens_dark': tokensDark,
          'notes': notes,
          // The panel's colour picker can only emit hex, so ask the gateway to
          // write a picked colour back in whatever notation that token already
          // used — otherwise editing one swatch of an oklch theme converts that
          // token to hex and the palette ends up written two ways.
          'preserve_notation': true,
        },
      );
      return CanvasDesignSystem.fromJson(res.data ?? const {});
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  /// Hand a one-click design-system job to the agent: 'extract' (read the
  /// project's real theme and record it) or 'showcase' (draw it as a canvas).
  /// The prompt lives in the gateway so every surface sends the same wording.
  Future<void> runDesignTask({
    required String sessionId,
    required String cwd,
    required String task,
  }) async {
    try {
      await _dio.post<void>(
        '/api/v1/canvas/design/task',
        data: {'session_id': sessionId, 'cwd': cwd, 'task': task},
      );
    } on Object catch (e) {
      throw toApiException(e);
    }
  }

  Future<void> delete(String id) async {
    try {
      await _dio.delete<void>('/api/v1/canvas/$id');
    } on Object catch (e) {
      throw toApiException(e);
    }
  }
}

/// A project's canvas design system: the tokens every canvas must use plus the
/// style rules tokens can't express. The gateway puts it in every canvas prompt
/// AND injects the tokens into each canvas as CSS variables — that pairing is
/// what stops successive renders from drifting apart.
class CanvasDesignSystem {
  const CanvasDesignSystem({
    this.tokens = const {},
    this.tokensDark = const {},
    this.notes = '',
    this.tokensResolved = const {},
    this.tokensDarkResolved = const {},
    this.warnings = const [],
  });

  factory CanvasDesignSystem.fromJson(Map<String, dynamic> json) =>
      CanvasDesignSystem(
        tokens: _tokenMap(json['tokens']),
        tokensDark: _tokenMap(json['tokens_dark']),
        notes: json['notes'] as String? ?? '',
        tokensResolved: _tokenMap(json['tokens_resolved']),
        tokensDarkResolved: _tokenMap(json['tokens_dark_resolved']),
        warnings: (json['warnings'] as List<dynamic>? ?? const [])
            .whereType<Map<String, dynamic>>()
            .map(CanvasDesignWarning.fromJson)
            .toList(growable: false),
      );

  final Map<String, String> tokens;

  /// Dark-mode values for the colour tokens; missing keys inherit the light one.
  final Map<String, String> tokensDark;
  final String notes;

  /// Each token that IS a colour, resolved by the gateway to #rrggbb. Projects
  /// write oklch/hsl/rgb and Dart has no CSS colour parser, so this is the only
  /// way the swatches can show anything but hex. Tokens that aren't colours
  /// (radius, font, the shadow triple) are absent — that's how we know not to
  /// draw a swatch beside them.
  final Map<String, String> tokensResolved;
  final Map<String, String> tokensDarkResolved;

  /// Advice about the saved system — never a reason the save failed.
  final List<CanvasDesignWarning> warnings;

  bool get isEmpty =>
      tokens.isEmpty && tokensDark.isEmpty && notes.trim().isEmpty;
}

/// A system whose every colour resolves to a grey — usually tokens mapped by
/// variable name off a shadcn theme, where `--primary` is an ink, not a brand
/// colour.
const String kCanvasWarningAchromaticPalette = 'achromatic_palette';

class CanvasDesignWarning {
  const CanvasDesignWarning({required this.code, required this.message});

  factory CanvasDesignWarning.fromJson(Map<String, dynamic> json) =>
      CanvasDesignWarning(
        code: json['code'] as String? ?? '',
        // English fallback, shown when a code has no translation yet.
        message: json['message'] as String? ?? '',
      );

  final String code;
  final String message;
}

Map<String, String> _tokenMap(Object? raw) {
  final out = <String, String>{};
  if (raw is Map) {
    raw.forEach((k, v) {
      if (v is String && v.trim().isNotEmpty) out['$k'] = v;
    });
  }
  return out;
}

/// The tokens the Canvas documents, in presentation order. Each becomes a CSS
/// variable (primary → var(--od-primary)).
/// The colour tokens that differ between themes; the rest are theme-independent.
const canvasThemedTokens = <String>[
  'primary',
  'secondary',
  'background',
  'surface',
  'text',
  'muted',
  'border',
  'shadow',
];

/// Starting palettes. Choosing colours from memory is the part people can't do,
/// so a new project starts from a complete, contrast-checked light + dark pair
/// and tweaks from there. Mirrors CANVAS_PALETTES in app/shared/src/lib/canvas.ts.
class CanvasPalette {
  const CanvasPalette({
    required this.id,
    required this.accent,
    required this.tokens,
    required this.tokensDark,
  });

  final String id;
  final int accent;
  final Map<String, String> tokens;
  final Map<String, String> tokensDark;
}

const _sans =
    'Inter, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", sans-serif';

Map<String, String> _light(String accent, _Neutral n, {String radius = '8px'}) => {
      'primary': accent,
      'secondary': n.border,
      'background': n.bg,
      'surface': n.surface,
      'text': n.text,
      'muted': n.muted,
      'border': n.border,
      'font': _sans,
      'headingFont': _sans,
      'baseSize': '14px',
      'radius': radius,
      'spacing': '4px',
      'shadow': '0 1px 2px rgba(0,0,0,0.06)',
    };

Map<String, String> _dark(String accent, _Neutral n) => {
      'primary': accent,
      'secondary': n.dBorder,
      'background': n.dBg,
      'surface': n.dSurface,
      'text': n.dText,
      'muted': n.dMuted,
      'border': n.dBorder,
      'shadow': '0 1px 2px rgba(0,0,0,0.4)',
    };

class _Neutral {
  const _Neutral(this.bg, this.surface, this.text, this.muted, this.border,
      this.dBg, this.dSurface, this.dText, this.dMuted, this.dBorder);
  final String bg;
  final String surface;
  final String text;
  final String muted;
  final String border;
  final String dBg;
  final String dSurface;
  final String dText;
  final String dMuted;
  final String dBorder;
}

const _slate = _Neutral('#f8fafc', '#ffffff', '#0f172a', '#64748b', '#e2e8f0',
    '#0b1120', '#151d2e', '#e2e8f0', '#94a3b8', '#26324a');
const _zinc = _Neutral('#fafafa', '#ffffff', '#18181b', '#71717a', '#e4e4e7',
    '#0c0c0e', '#18181b', '#e4e4e7', '#a1a1aa', '#2a2a2e');
const _stone = _Neutral('#faf9f7', '#ffffff', '#1c1917', '#78716c', '#e7e5e4',
    '#0e0d0c', '#1c1917', '#e7e5e4', '#a8a29e', '#2d2926');

final canvasPalettes = <CanvasPalette>[
  CanvasPalette(id: 'indigo', accent: 0xFF4F46E5, tokens: _light('#4f46e5', _slate), tokensDark: _dark('#818cf8', _slate)),
  CanvasPalette(id: 'sky', accent: 0xFF0284C7, tokens: _light('#0284c7', _slate), tokensDark: _dark('#38bdf8', _slate)),
  CanvasPalette(id: 'emerald', accent: 0xFF059669, tokens: _light('#059669', _slate), tokensDark: _dark('#34d399', _slate)),
  CanvasPalette(id: 'amber', accent: 0xFFD97706, tokens: _light('#d97706', _stone), tokensDark: _dark('#fbbf24', _stone)),
  CanvasPalette(id: 'rose', accent: 0xFFE11D48, tokens: _light('#e11d48', _stone), tokensDark: _dark('#fb7185', _stone)),
  CanvasPalette(id: 'violet', accent: 0xFF7C3AED, tokens: _light('#7c3aed', _zinc), tokensDark: _dark('#a78bfa', _zinc)),
  CanvasPalette(id: 'graphite', accent: 0xFFF97316, tokens: _light('#f97316', _zinc, radius: '6px'), tokensDark: _dark('#fb923c', _zinc)),
];

const canvasDesignTokens = <String>[
  'primary',
  'secondary',
  'background',
  'surface',
  'text',
  'muted',
  'border',
  'font',
  'headingFont',
  'baseSize',
  'radius',
  'spacing',
  'shadow',
];

/// A canvas event off the integration eventbus.
class CanvasEvent {
  const CanvasEvent({required this.topic, required this.cwd, this.slug = ''});

  final String topic;
  final String cwd;
  final String slug;

  bool get isUpdated => topic == 'canvas.updated';
  bool get isFocus => topic == 'canvas.focus_changed';
}

/// Live canvas events (renders + focus changes) over the integration eventbus,
/// so the panel refreshes the moment an agent re-renders. Reconnects with
/// backoff like the terminal socket; mobile can't set WS headers, so the token
/// rides in the query string.
class CanvasEvents {
  CanvasEvents({required this.serverUrl, required this.token});

  final String serverUrl;
  final String token;

  final _controller = StreamController<CanvasEvent>.broadcast();
  WebSocketChannel? _channel;
  StreamSubscription<dynamic>? _sub;
  Timer? _retry;
  int _attempts = 0;
  bool _closed = false;

  Stream<CanvasEvent> get stream => _controller.stream;

  void start() {
    if (_closed) return;
    _connect();
  }

  void _connect() {
    if (_closed) return;
    final base = serverUrl.startsWith('https')
        ? serverUrl.replaceFirst('https', 'wss')
        : serverUrl.replaceFirst('http', 'ws');
    final url = '$base/api/v1/integrations/_events'
        '?topics=canvas.updated,canvas.focus_changed'
        '&token=${Uri.encodeQueryComponent(token)}';
    try {
      final channel = WebSocketChannel.connect(Uri.parse(url));
      _channel = channel;
      _sub = channel.stream.listen(
        _onMessage,
        onError: (_) => _scheduleReconnect(),
        onDone: _scheduleReconnect,
        cancelOnError: false,
      );
      _attempts = 0;
    } on Object catch (_) {
      _scheduleReconnect();
    }
  }

  void _onMessage(dynamic raw) {
    final text = switch (raw) {
      final String s => s,
      final List<int> b => utf8.decode(b, allowMalformed: true),
      _ => '',
    };
    if (text.isEmpty) return;
    try {
      final frame = jsonDecode(text);
      if (frame is! Map<String, dynamic>) return;
      final topic = frame['topic'] as String? ?? '';
      final data = frame['data'];
      if (data is! Map<String, dynamic>) return;
      _controller.add(
        CanvasEvent(
          topic: topic,
          cwd: data['cwd'] as String? ?? '',
          slug: data['slug'] as String? ?? '',
        ),
      );
    } on Object catch (_) {
      // Ignore non-JSON frames (pings, malformed) — the socket stays up.
    }
  }

  void _scheduleReconnect() {
    if (_closed || _retry != null) return;
    _attempts += 1;
    if (_attempts > 5) return;
    final delay = Duration(milliseconds: 500 * (1 << (_attempts - 1)));
    _retry = Timer(delay, () {
      _retry = null;
      _connect();
    });
  }

  Future<void> close() async {
    _closed = true;
    _retry?.cancel();
    await _sub?.cancel();
    await _channel?.sink.close();
    await _controller.close();
  }
}

final canvasApiProvider =
    Provider<CanvasApi>((ref) => CanvasApi(ref.watch(dioProvider)));

/// All canvases for a project.
final canvasListProvider =
    FutureProvider.autoDispose.family<List<CanvasSummary>, String>(
  (ref, cwd) => ref.watch(canvasApiProvider).list(cwd),
);

/// Live canvas events for the signed-in server, or null when logged out.
final canvasEventsProvider = Provider.autoDispose<CanvasEvents?>((ref) {
  final auth = ref.watch(authControllerProvider);
  if (auth is! AuthLoggedIn) return null;
  final events = CanvasEvents(serverUrl: auth.serverUrl, token: auth.token)
    ..start();
  ref.onDispose(events.close);
  return events;
});
