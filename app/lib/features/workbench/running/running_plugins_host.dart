import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/services/l10n.dart';
import 'plugin_thumbnail_capture.dart';
import 'running_plugins_models.dart';
import 'running_plugins_service.dart';

/// Keeps every running plugin widget mounted across navigation events.
///
/// Renders inside `_Shell.body`. The earlier overlay-passthrough design
/// (transparent shell on top, plugin paints through from below) turned
/// out to be fragile under real-device hit-testing. This version uses
/// an [IndexedStack] instead:
///
///   • On a plugin route, the host paints the IndexedStack directly —
///     only the active entry is visible, its siblings stay mounted but
///     don't paint. Hit-testing routes exclusively to the active
///     child (standard IndexedStack semantics).
///   • On a non-plugin route (Sessions, Plugins launcher, Settings…)
///     the non-plugin child paints on top of the IndexedStack, and the
///     IndexedStack is wrapped in [Offstage] so it neither paints nor
///     receives pointer events — yet its children remain mounted, so
///     a later return to the plugin re-attaches to the same State.
///
/// Thumbnail capture: whenever the active entry flips away, the host
/// schedules a post-frame `RepaintBoundary.toImage` against the
/// outgoing entry's boundary. The webview fallback and icon-placeholder
/// paths live inside [PluginThumbnailCapture].
class RunningPluginsHost extends StatefulWidget {
  const RunningPluginsHost({
    required this.isPluginRoute,
    required this.targetPluginId,
    required this.nonPluginChild,
    super.key,
  });

  /// True when the current location matches `/browser/*`. Controls
  /// whether the IndexedStack is foregrounded (plugin route) or
  /// kept offstage behind [nonPluginChild] (non-plugin route).
  final bool isPluginRoute;

  /// Plugin id the current route resolves to, or null when on a
  /// non-plugin route. Used to pick the IndexedStack index without
  /// waiting for the post-frame `setActive` round-trip.
  final String? targetPluginId;

  /// Widget rendered on top when [isPluginRoute] is false — the normal
  /// GoRouter-supplied page (DashboardPage, PluginsPage, SettingsPage,
  /// …). Ignored on plugin routes.
  final Widget nonPluginChild;

  @override
  State<RunningPluginsHost> createState() => _RunningPluginsHostState();
}

class _RunningPluginsHostState extends State<RunningPluginsHost> {
  /// One boundary key per entry id. Built lazily, held across rebuilds
  /// so `capture` can find the same RenderObject every time.
  final Map<String, GlobalKey> _boundaryKeys = {};

  /// Tracks the previously-observed active id so we can fire a capture
  /// exactly when the user navigates away from an entry.
  String? _lastSeenActive;

  GlobalKey _keyFor(String id) =>
      _boundaryKeys.putIfAbsent(id, GlobalKey.new);

  @override
  Widget build(BuildContext context) {
    final service = context.watch<RunningPluginsService>();

    // Detect "active entry changed" — if so, capture a thumbnail of
    // the one we're leaving. Scheduled post-frame so the outgoing
    // boundary still has valid painted pixels when toImage runs.
    final currentActive = service.activeId;
    if (currentActive != _lastSeenActive) {
      final outgoing = _lastSeenActive;
      _lastSeenActive = currentActive;
      if (outgoing != null) {
        final entry = _entryOrNull(service, outgoing);
        if (entry != null) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            _captureThumbnail(entry);
          });
        }
      }
    }

    // Prune boundary keys for entries that have been closed.
    if (_boundaryKeys.length > service.entries.length) {
      final liveIds = service.entries.map((e) => e.id).toSet();
      _boundaryKeys.removeWhere((id, _) => !liveIds.contains(id));
    }

    final entries = service.entries;

    // Pick the IndexedStack index. On plugin routes, prefer the route-
    // derived target so the IndexedStack has the right page visible on
    // the very first frame (before the post-frame `setActive` fires).
    // Off plugin routes, lean on the service's activeId purely so the
    // Offstage'd stack isn't pointed at a stale index when the user
    // comes back. An out-of-range index (no matching entry yet) is
    // clamped to 0 — IndexedStack asserts index < children.length.
    int idx = 0;
    String? activeForTickers;
    if (widget.isPluginRoute && widget.targetPluginId != null) {
      activeForTickers = widget.targetPluginId;
      final found =
          entries.indexWhere((e) => e.id == widget.targetPluginId);
      if (found >= 0) idx = found;
    } else if (service.activeId != null) {
      activeForTickers = service.activeId;
      final found =
          entries.indexWhere((e) => e.id == service.activeId);
      if (found >= 0) idx = found;
    }

    // Build the IndexedStack's children. Each is wrapped in a
    // RepaintBoundary (for thumbnail capture) and TickerMode (off for
    // background entries so their animation tickers don't burn CPU
    // while the user is elsewhere). A ValueKey on the surface means
    // Flutter keeps the same Element — and therefore the same State —
    // across any rebuild that reshuffles the list.
    final children = <Widget>[
      for (final entry in entries)
        RepaintBoundary(
          key: _keyFor(entry.id),
          child: TickerMode(
            enabled: entry.id == activeForTickers,
            child: _PluginSurface(
              key: ValueKey('running-${entry.id}'),
              entry: entry,
            ),
          ),
        ),
    ];

    // Plugin route before any entry is registered (first frame, before
    // the post-frame `ensureOpened` callback has run): render a blank
    // placeholder. One frame later the IndexedStack takes over.
    if (widget.isPluginRoute && children.isEmpty) {
      return const SizedBox.expand();
    }

    final indexedStack = children.isEmpty
        ? const SizedBox.expand()
        : IndexedStack(index: idx, children: children);

    if (widget.isPluginRoute) {
      return indexedStack;
    }

    // Non-plugin route: paint nonPluginChild on top; keep the
    // IndexedStack offstage so mounted plugin state survives.
    // Stack hit-testing iterates reverse — `nonPluginChild` is last,
    // tested first, and is opaque, so taps land there. The Offstage
    // returns false from hitTest either way.
    return Stack(
      fit: StackFit.expand,
      children: [
        Offstage(offstage: true, child: indexedStack),
        widget.nonPluginChild,
      ],
    );
  }

  Future<void> _captureThumbnail(RunningPluginEntry entry) async {
    if (!mounted) return;
    final key = _boundaryKeys[entry.id];
    if (key == null) return;
    final thumb = await PluginThumbnailCapture.capture(
      boundaryKey: key,
      entry: entry,
    );
    if (!mounted) return;
    // Use read — a service notify would re-trigger our build, and we
    // don't want to re-fire a capture on the same transition.
    context.read<RunningPluginsService>().updateThumbnail(entry.id, thumb);
  }

  RunningPluginEntry? _entryOrNull(RunningPluginsService service, String id) {
    for (final e in service.entries) {
      if (e.id == id) return e;
    }
    return null;
  }
}

/// Full-page Scaffold wrapping a single running plugin's widget. The
/// AppBar title is resolved through [L10n] so language changes
/// propagate without remounting the plugin.
class _PluginSurface extends StatelessWidget {
  const _PluginSurface({required this.entry, super.key});
  final RunningPluginEntry entry;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(context.tr(entry.titleKey))),
      body: Builder(builder: entry.builder),
    );
  }
}
