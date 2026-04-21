import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

/// Web-only responsive design tokens.
///
/// All getters here return a "web desktop" value **only** when the app
/// is running in a browser AND the viewport is wide enough to deserve
/// a desktop-shaped UI. On mobile platforms (iOS/Android) — or on a
/// phone-width browser window — they return the original mobile value,
/// so nothing changes for phone builds.
///
/// The three surfaces we care about:
///   * typography scale (fonts look cramped at 14px on a 4K monitor)
///   * dialog / modal sheet width (mobile bottom-sheet becomes a
///     centered card on desktop)
///   * content padding (tight phone gutters look miserly on desktop)
class Responsive {
  Responsive._();

  /// Minimum viewport width at which we consider the app "desktop web".
  /// Matches the Shell's rail breakpoint so the rail + scaled theme
  /// turn on together.
  static const double desktopWebBreakpoint = 900;

  /// `true` when the viewport is wide enough to warrant a desktop
  /// layout AND we are running on web. On native mobile this is
  /// always `false` regardless of window size (iPads on macOS, etc.).
  static bool isDesktopWeb(BuildContext context) {
    if (!kIsWeb) return false;
    final width = MediaQuery.maybeOf(context)?.size.width ?? 0;
    return width >= desktopWebBreakpoint;
  }

  /// Global font size multiplier. 1.0 = mobile-native values. Applied
  /// once at the top of the tree so every [TextTheme]-derived style
  /// grows proportionally without touching the 60+ hard-coded fontSize
  /// call sites scattered through feature pages.
  /// Global text scale factor applied via [MediaQueryData.textScaler] on
  /// desktop web. This is the **only** knob for font sizing — it scales
  /// every `Text` widget uniformly, including the ~60 sites that hard-code
  /// `fontSize: 12/13/14` constants (those don't inherit from textTheme,
  /// so a theme-level copyWith can't reach them). Keep the multiplier
  /// modest so buttons, chips, and AppBar actions don't explode.
  static double fontScale(BuildContext context) {
    if (!isDesktopWeb(context)) return 1.0;
    final width = MediaQuery.of(context).size.width;
    if (width >= 1600) return 1.35;
    if (width >= 1200) return 1.30;
    return 1.25;
  }

  /// Max width for centered dialogs on desktop web. Mobile keeps the
  /// native behavior (full width for bottom sheets, Flutter default
  /// for dialogs), so this is only read when [isDesktopWeb] is true.
  static double dialogMaxWidth(BuildContext context) {
    final width = MediaQuery.of(context).size.width;
    // Content-rich sheets (pickers, settings subforms) breathe best
    // around 560-640; keep a hard cap even on ultra-wide monitors
    // so the dialog doesn't turn into a landscape stripe.
    if (width >= 1400) return 640;
    return 560;
  }

  /// Horizontal content padding for routed pages. Phones / narrow web
  /// keep 16; wide web gets 24 so content doesn't hug the rail divider.
  static double contentHGutter(BuildContext context) {
    return isDesktopWeb(context) ? 24 : 16;
  }
}
