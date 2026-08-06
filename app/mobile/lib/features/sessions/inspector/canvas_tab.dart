import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/canvas_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/sessions/inspector/canvas_common.dart';
import 'package:opendray/features/sessions/inspector/canvas_design_sheet.dart';
import 'package:opendray/features/sessions/inspector/canvas_viewer_screen.dart';

// CanvasTab — the session inspector's Canvas: the visual channel between the
// agent and the operator. The agent renders a self-contained HTML page with
// `canvas_render`; here the operator picks WHICH canvas is focused (so plain
// talk in the terminal resolves to it) and asks for new ones.
//
// The tab stays a chooser: a chip rail of the project's canvases, a thumbnail,
// and a request bar. Reviewing and marking happen in CanvasViewerScreen, which
// takes the whole screen — a phone has no room to do both at once.

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

  final _requestCtl = TextEditingController();
  CanvasKind _newKind = CanvasKind.ui;
  bool _newCanvas = false;
  bool _requesting = false;

  InAppWebViewController? _thumb;
  StreamSubscription<CanvasEvent>? _events;

  /// The canvas the agent actually works on. Browsing the rail does NOT change
  /// it — picking a canvas only previews it (free), while making it the
  /// workspace is a deliberate act that costs one seeded note. Otherwise
  /// flicking through canvases to decide would burn a turn per tap.
  String _workspace = '';
  bool _committing = false;

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    unawaited(_load());
    unawaited(_loadWorkspace());
    _events = ref.read(canvasEventsProvider)?.stream.listen(_onEvent);
  }

  @override
  void dispose() {
    unawaited(_events?.cancel());
    _requestCtl.dispose();
    super.dispose();
  }

  void _onEvent(CanvasEvent ev) {
    if (ev.cwd != widget.cwd || !mounted) return;
    if (ev.isUpdated) {
      unawaited(_load(keepSelection: true));
    } else if (ev.isFocus) {
      // Another surface (the web panel) set the workspace — just reflect it.
      // Nothing is seeded from here; that side already did it.
      setState(() => _workspace = ev.slug);
    }
  }

  Future<void> _load({bool keepSelection = false}) async {
    if (!keepSelection) setState(() => _list = const AsyncValue.loading());
    try {
      final items = await ref.read(canvasApiProvider).list(widget.cwd);
      if (!mounted) return;
      setState(() => _list = AsyncValue.data(items));
      final current = _selectedId;
      if (items.isEmpty) {
        setState(() {
          _selectedId = null;
          _artifact = null;
        });
      } else if (!items.any((a) => a.id == current)) {
        await _select(items.first);
      } else if (keepSelection && current != null) {
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

  /// Preview a canvas. Free: no request, no tokens — the operator is browsing.
  Future<void> _select(CanvasSummary item) async {
    setState(() {
      _selectedId = item.id;
      _newCanvas = false;
    });
    await _loadArtifact(item.id);
  }

  Future<void> _loadWorkspace() async {
    try {
      final f = await ref.read(canvasApiProvider).getFocus(widget.cwd);
      if (!mounted) return;
      setState(() => _workspace = f.slug);
    } on Object catch (_) {
      // Unknown workspace just means no badge.
    }
  }

  /// The deliberate "work on THIS one" action: records the workspace and seeds
  /// the single note that makes plain terminal conversation resolve here.
  Future<void> _commitWorkspace(CanvasSummary item) async {
    setState(() => _committing = true);
    try {
      await ref.read(canvasApiProvider).setFocus(
            cwd: widget.cwd,
            slug: item.slug,
            sessionId: widget.sessionId,
            notify: true,
          );
      if (!mounted) return;
      setState(() => _workspace = item.slug);
      _snack(t.sessions.inspector.canvas.workspaceSet);
    } on Object catch (e) {
      _snack(t.sessions.inspector.shared.insertFailedGeneric(error: '$e'));
    } finally {
      if (mounted) setState(() => _committing = false);
    }
  }

  Future<void> _loadArtifact(String id) async {
    try {
      final a = await ref.read(canvasApiProvider).get(id);
      if (!mounted) return;
      setState(() => _artifact = a);
      await _thumb?.loadData(
        data: canvasDocument(a.html, canvasViewportWidth(CanvasViewport.phone)),
        mimeType: 'text/html',
      );
    } on Object catch (_) {
      // Keep the previous preview rather than blanking the panel.
    }
  }

  Future<void> _openViewer() async {
    final a = _artifact;
    if (a == null) return;
    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => CanvasViewerScreen(sessionId: widget.sessionId, artifact: a),
      ),
    );
  }

  Future<void> _sendRequest() async {
    final prompt = _requestCtl.text.trim();
    if (prompt.isEmpty) return;
    final target = _newCanvas
        ? null
        : _list.valueOrNull?.where((a) => a.id == _selectedId).firstOrNull;
    setState(() => _requesting = true);
    try {
      await ref.read(canvasApiProvider).requestDesign(
            sessionId: widget.sessionId,
            cwd: widget.cwd,
            prompt: prompt,
            slug: target?.slug,
            title: target?.title,
            kind: _newCanvas ? _newKind : null,
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
    final a = _artifact;
    if (a == null) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.sessions.inspector.canvas.deleteTitle),
        content: Text(
          t.sessions.inspector.canvas.deleteBody(
            title: a.title.isEmpty ? a.slug : a.title,
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
      await ref.read(canvasApiProvider).delete(a.id);
      if (a.slug == _workspace) setState(() => _workspace = '');
      await _load();
    } on Object catch (e) {
      _snack(t.sessions.inspector.shared.insertFailedGeneric(error: '$e'));
    }
  }

  void _snack(String text) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(text)));
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    final theme = Theme.of(context);
    return _list.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorBody(error: '$e', onRetry: () => unawaited(_load())),
      data: (items) => Column(
        children: [
          if (items.isNotEmpty) _rail(items),
          if (_newCanvas) _kindPicker(theme),
          if (!_newCanvas && _artifact != null) _focusLine(theme),
          Divider(height: 1, color: theme.dividerColor),
          Expanded(
            child: items.isEmpty || _newCanvas
                ? _EmptyBody(newCanvas: _newCanvas)
                : _thumbnail(theme),
          ),
          _requestBar(theme),
        ],
      ),
    );
  }

  Widget _rail(List<CanvasSummary> items) {
    return SizedBox(
      height: 46,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        itemCount: items.length + 1,
        separatorBuilder: (_, __) => const SizedBox(width: 6),
        itemBuilder: (_, i) {
          if (i == items.length) {
            return ChoiceChip(
              selected: _newCanvas,
              avatar: const Icon(Icons.add, size: 16),
              label: Text(t.sessions.inspector.canvas.newCanvas),
              onSelected: (_) => setState(() => _newCanvas = true),
            );
          }
          final a = items[i];
          final isWorkspace = a.slug == _workspace;
          return ChoiceChip(
            selected: !_newCanvas && a.id == _selectedId,
            avatar: Icon(canvasKindIcon(a.kind), size: 16),
            label: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Flexible(child: Text(a.label, overflow: TextOverflow.ellipsis)),
                if (isWorkspace) ...[
                  const SizedBox(width: 5),
                  Container(
                    width: 6,
                    height: 6,
                    decoration: BoxDecoration(
                      color: Theme.of(context).colorScheme.primary,
                      shape: BoxShape.circle,
                    ),
                  ),
                ],
              ],
            ),
            onSelected: (_) => unawaited(_select(a)),
          );
        },
      ),
    );
  }

  Widget _kindPicker(ThemeData theme) {
    return SizedBox(
      height: 44,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
        itemCount: CanvasKind.values.length,
        separatorBuilder: (_, __) => const SizedBox(width: 6),
        itemBuilder: (_, i) {
          final k = CanvasKind.values[i];
          return ChoiceChip(
            selected: _newKind == k,
            avatar: Icon(canvasKindIcon(k), size: 16),
            label: Text(canvasKindLabel(k)),
            onSelected: (_) => setState(() => _newKind = k),
          );
        },
      ),
    );
  }

  /// Says whether the previewed canvas is merely being looked at or is the
  /// agent's workspace — and offers the one deliberate action that changes it.
  Widget _focusLine(ThemeData theme) {
    final a = _artifact!;
    final isWorkspace = a.slug == _workspace;
    return Padding(
      padding: const EdgeInsets.fromLTRB(14, 0, 4, 4),
      child: Row(
        children: [
          Expanded(
            child: Text(
              isWorkspace
                  ? t.sessions.inspector.canvas.focusLine(
                      title: a.title.isEmpty ? a.slug : a.title,
                      kind: canvasKindLabel(a.kind),
                    )
                  : t.sessions.inspector.canvas.previewOnlyHint,
              style: theme.textTheme.bodySmall?.copyWith(
                color: isWorkspace
                    ? theme.colorScheme.primary
                    : theme.colorScheme.onSurfaceVariant,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          if (!isWorkspace)
            TextButton(
              onPressed: _committing
                  ? null
                  : () {
                      final item = _list.valueOrNull
                          ?.where((c) => c.id == _selectedId)
                          .firstOrNull;
                      if (item != null) unawaited(_commitWorkspace(item));
                    },
              child: Text(t.sessions.inspector.canvas.setWorkspace),
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

  /// A live but non-interactive thumbnail — tapping anywhere opens the
  /// full-screen viewer, where there is room to zoom and mark.
  Widget _thumbnail(ThemeData theme) {
    return Stack(
      children: [
        Positioned.fill(
          child: InAppWebView(
            initialSettings: InAppWebViewSettings(
              transparentBackground: true,
              supportZoom: false,
              useWideViewPort: true,
              loadWithOverviewMode: true,
            ),
            onWebViewCreated: (ctl) {
              _thumb = ctl;
              final a = _artifact;
              if (a != null) {
                unawaited(
                  ctl.loadData(
                    data: canvasDocument(
                      a.html,
                      canvasViewportWidth(CanvasViewport.phone),
                    ),
                    mimeType: 'text/html',
                  ),
                );
              }
            },
          ),
        ),
        // Swallow gestures so a stray swipe can't scroll the thumbnail — the
        // whole surface is one big "open" affordance instead.
        Positioned.fill(
          child: GestureDetector(
            behavior: HitTestBehavior.opaque,
            onTap: () => unawaited(_openViewer()),
          ),
        ),
        Positioned(
          right: 12,
          bottom: 12,
          child: FloatingActionButton.extended(
            heroTag: null,
            onPressed: () => unawaited(_openViewer()),
            icon: const Icon(Icons.open_in_full, size: 18),
            label: Text(t.sessions.inspector.canvas.openFull),
          ),
        ),
      ],
    );
  }

  Widget _requestBar(ThemeData theme) {
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(12, 6, 12, 8),
        child: Row(
          children: [
            IconButton(
              icon: const Icon(Icons.palette_outlined, size: 20),
              tooltip: t.sessions.inspector.canvas.designTitle,
              onPressed: () => unawaited(
                showModalBottomSheet<void>(
                  context: context,
                  isScrollControlled: true,
                  builder: (_) => CanvasDesignSheet(cwd: widget.cwd),
                ),
              ),
            ),
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
        ),
      ),
    );
  }
}

class _EmptyBody extends StatelessWidget {
  const _EmptyBody({required this.newCanvas});

  final bool newCanvas;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Center(
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
              newCanvas
                  ? t.sessions.inspector.canvas.newCanvasTitle
                  : t.sessions.inspector.canvas.emptyTitle,
              style: theme.textTheme.titleMedium,
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 8),
            Text(
              newCanvas
                  ? t.sessions.inspector.canvas.newCanvasBlurb
                  : t.sessions.inspector.canvas.emptyBlurb,
              style: theme.textTheme.bodySmall,
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
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
