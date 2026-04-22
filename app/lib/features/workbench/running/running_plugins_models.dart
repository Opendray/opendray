import 'dart:typed_data';

import 'package:flutter/widgets.dart';

/// What flavour of surface a running entry represents. Drives thumbnail
/// capture: webview plugins need the JS-bridge fallback because
/// `RepaintBoundary.toImage()` often produces blank pixels over a
/// `WKWebView` platform view on iOS.
enum RunningPluginKind { builtin, webview }

/// Thumbnail shown on a switcher card. Sealed so the switcher can
/// render each case without a null check.
sealed class PluginThumbnail {
  const PluginThumbnail();
}

/// Placeholder until the first capture fires. The switcher draws the
/// entry icon on a filled tile.
class PendingThumbnail extends PluginThumbnail {
  const PendingThumbnail();
}

/// Last-resort thumbnail: the entry's icon drawn on a tile. Picked
/// whenever the capture chain exhausts itself.
class IconThumbnail extends PluginThumbnail {
  const IconThumbnail();
}

/// Real captured preview. `pngBytes` is already compressed so swapping
/// in a new snapshot just rebinds the bytes — no decode kept alive.
class ImageThumbnail extends PluginThumbnail {
  final Uint8List pngBytes;
  final int width;
  final int height;
  const ImageThumbnail({
    required this.pngBytes,
    required this.width,
    required this.height,
  });
}

/// One row in the running-plugins set. Immutable — [RunningPluginsService]
/// rebuilds entries via [copyWith] rather than mutating in place so
/// listeners always see a fresh snapshot.
class RunningPluginEntry {
  /// Stable id — `builtin:<panel>` for the built-in `/browser/*` panels,
  /// `webview:<pluginName>` for generic v1 webview plugins. Two prefixes
  /// keep the namespaces disjoint so re-navigation to the same route
  /// finds the same entry rather than duplicating it.
  final String id;

  /// Display title. Stored as the raw English key used with
  /// `context.tr(...)` so translation happens at render time.
  final String titleKey;

  /// Icon drawn in the bottom-nav badge and as the icon-placeholder
  /// thumbnail fallback.
  final IconData icon;

  /// GoRouter location the switcher navigates to when the user taps the
  /// card. Matches the registered route exactly (e.g. `/browser/docs`,
  /// `/browser/plugin/kanban`).
  final String route;

  final RunningPluginKind kind;

  /// Builds the actual page widget that lives inside the Offstage host.
  /// Invoked exactly once per entry (the result is cached via a
  /// [KeyedSubtree] upstream), so the builder is free to return a
  /// stateful widget whose `State` carries the surviving state.
  final Widget Function(BuildContext) builder;

  /// Current thumbnail. [PendingThumbnail] until the first capture.
  final PluginThumbnail thumbnail;

  final DateTime openedAt;
  final DateTime lastActiveAt;

  const RunningPluginEntry({
    required this.id,
    required this.titleKey,
    required this.icon,
    required this.route,
    required this.kind,
    required this.builder,
    required this.openedAt,
    required this.lastActiveAt,
    this.thumbnail = const PendingThumbnail(),
  });

  RunningPluginEntry copyWith({
    PluginThumbnail? thumbnail,
    DateTime? lastActiveAt,
  }) {
    return RunningPluginEntry(
      id: id,
      titleKey: titleKey,
      icon: icon,
      route: route,
      kind: kind,
      builder: builder,
      openedAt: openedAt,
      lastActiveAt: lastActiveAt ?? this.lastActiveAt,
      thumbnail: thumbnail ?? this.thumbnail,
    );
  }
}
