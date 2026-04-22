import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../../../core/services/l10n.dart';
import 'plugin_thumbnail_capture.dart';
import 'running_plugins_models.dart';
import 'running_plugins_service.dart';

/// Keeps every running plugin widget mounted across navigation events.
///
/// Rendered inside `_Shell.body`, above the GoRouter-supplied page.
/// The host stacks:
///   • base layer: one [_PluginSurface] per running entry — each is a
///     full-page Scaffold (AppBar + plugin widget) wrapped in a
///     [RepaintBoundary] (for thumbnail capture) and then [Offstage]
///     so only the active entry paints and reacts to pointers.
///     Inactive entries stay alive; their `State` objects and
///     subscriptions are preserved.
///   • top layer: [routerChild] — the normal page GoRouter picked for
///     the current location. Plugin routes render as
///     `SizedBox.expand()` (see `RunningPluginRevealShell`), letting
///     the active plugin layer below paint through. Non-plugin routes
///     (Sessions dashboard, switcher, Settings) render opaquely and
///     cover the plugin layer.
///
/// Also responsible for thumbnail capture: whenever `activeId` flips
/// away from an entry, the host schedules a post-frame capture of
/// that entry's boundary and writes the result back into the service.
class RunningPluginsHost extends StatefulWidget {
  final Widget routerChild;
  const RunningPluginsHost({required this.routerChild, super.key});

  @override
  State<RunningPluginsHost> createState() => _RunningPluginsHostState();
}

class _RunningPluginsHostState extends State<RunningPluginsHost> {
  /// One boundary key per entry id. Built lazily and held across
  /// rebuilds so `capture` can find the same RenderObject every time.
  final Map<String, GlobalKey> _boundaryKeys = {};

  /// Tracks the previously-observed active id so we can fire a
  /// capture exactly when the user navigates away from an entry.
  String? _lastSeenActive;

  GlobalKey _keyFor(String id) =>
      _boundaryKeys.putIfAbsent(id, GlobalKey.new);

  @override
  Widget build(BuildContext context) {
    final service = context.watch<RunningPluginsService>();

    // Detect "active entry changed" — if so, capture a thumbnail of
    // the one we're leaving. Scheduled as a post-frame callback so
    // the outgoing boundary still has valid painted pixels when
    // toImage runs. (Offstage flips _after_ this frame; combined with
    // the RepaintBoundary sitting outside Offstage, the last visible
    // frame is what gets captured.)
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

    // Prune boundary keys for entries that have been closed — avoids
    // an unbounded map when users open and close many plugins.
    if (_boundaryKeys.length > service.entries.length) {
      final liveIds = service.entries.map((e) => e.id).toSet();
      _boundaryKeys.removeWhere((id, _) => !liveIds.contains(id));
    }

    return Stack(
      fit: StackFit.expand,
      children: [
        for (final entry in service.entries)
          // RepaintBoundary sits OUTSIDE Offstage so the boundary's
          // render object still exists (and retains its last painted
          // layer) after the Offstage flip, which is the precise
          // moment the thumbnail is captured.
          RepaintBoundary(
            key: _keyFor(entry.id),
            child: Offstage(
              offstage: service.activeId != entry.id,
              child: TickerMode(
                enabled: service.activeId == entry.id,
                child: KeyedSubtree(
                  key: ValueKey('running-${entry.id}'),
                  child: _PluginSurface(entry: entry),
                ),
              ),
            ),
          ),
        widget.routerChild,
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
  final RunningPluginEntry entry;
  const _PluginSurface({required this.entry});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(context.tr(entry.titleKey))),
      body: Builder(builder: entry.builder),
    );
  }
}
