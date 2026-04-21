import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:pointer_interceptor/pointer_interceptor.dart';

import 'terminal_iframe_gate.dart';
import 'theme/app_theme.dart';
import 'theme/responsive.dart';

/// Wraps [child] so pointer events land on Flutter instead of any HtmlElementView
/// (iframe) rendered below on Flutter Web. A no-op on other platforms.
///
/// Needed because the xterm.js iframe sits in its own DOM layer and
/// absorbs clicks/taps before they reach Flutter's canvas, which means
/// buttons on modals (Close, Select, etc.) never fire until the user
/// hits browser-back.
Widget interceptOnWeb(Widget child) {
  if (!kIsWeb) return child;
  return PointerInterceptor(child: child);
}

/// Drop-in replacement for [showModalBottomSheet] that keeps modal
/// taps working on Flutter Web above the terminal iframe.
///
/// On web, while the sheet is open, every `<iframe>` in the document has
/// its `pointer-events` disabled so the barrier + content receive all
/// input. The original state is restored when the sheet pops.
///
/// ### Desktop web routing
/// When the viewport is wide enough to look like a desktop browser
/// ([Responsive.isDesktopWeb]), the sheet is automatically re-rendered
/// as a centered [Dialog] card (width-capped by [Responsive.dialogMaxWidth],
/// height-capped at 80% of viewport). Mobile devices and narrow web
/// windows keep the native bottom-sheet animation.
Future<T?> showAppModalBottomSheet<T>({
  required BuildContext context,
  required WidgetBuilder builder,
  Color? backgroundColor,
  ShapeBorder? shape,
  bool isScrollControlled = false,
  bool useRootNavigator = false,
  bool isDismissible = true,
  bool enableDrag = true,
}) async {
  if (Responsive.isDesktopWeb(context)) {
    return _showAsDesktopDialog<T>(
      context: context,
      barrierDismissible: isDismissible,
      useRootNavigator: useRootNavigator,
      backgroundColor: backgroundColor,
      builder: builder,
    );
  }

  pushModalIframeMute();
  try {
    return await showModalBottomSheet<T>(
      context: context,
      backgroundColor: backgroundColor,
      shape: shape,
      isScrollControlled: isScrollControlled,
      useRootNavigator: useRootNavigator,
      isDismissible: isDismissible,
      enableDrag: enableDrag,
      builder: (ctx) => interceptOnWeb(Builder(builder: builder)),
    );
  } finally {
    popModalIframeMute();
  }
}

/// Drop-in replacement for [showDialog] with the same pointer-interception
/// and iframe muting. On desktop web the dialog is centered and its
/// max-width is soft-clamped so content-rich custom dialogs don't turn
/// into viewport-wide stripes.
Future<T?> showAppDialog<T>({
  required BuildContext context,
  required WidgetBuilder builder,
  bool barrierDismissible = true,
  bool useRootNavigator = true,
}) async {
  pushModalIframeMute();
  try {
    return await showDialog<T>(
      context: context,
      barrierDismissible: barrierDismissible,
      useRootNavigator: useRootNavigator,
      builder: (ctx) => interceptOnWeb(Builder(builder: builder)),
    );
  } finally {
    popModalIframeMute();
  }
}

/// Renders a bottom-sheet builder as a centered, width-capped card on
/// desktop web. The builder's output is typically a Column / SizedBox /
/// scrollable — we don't wrap it in Material ourselves because many
/// callers already include their own padding and surface color; the
/// outer [Dialog] is enough to give it a web-dialog shell.
Future<T?> _showAsDesktopDialog<T>({
  required BuildContext context,
  required bool barrierDismissible,
  required bool useRootNavigator,
  required Color? backgroundColor,
  required WidgetBuilder builder,
}) async {
  pushModalIframeMute();
  try {
    return await showDialog<T>(
      context: context,
      barrierDismissible: barrierDismissible,
      useRootNavigator: useRootNavigator,
      builder: (ctx) => interceptOnWeb(
        Builder(builder: (bctx) {
          final maxWidth = Responsive.dialogMaxWidth(bctx);
          final maxHeight = MediaQuery.of(bctx).size.height * 0.8;
          return Dialog(
            backgroundColor: backgroundColor ?? AppColors.surface,
            insetPadding: const EdgeInsets.symmetric(
              horizontal: 24,
              vertical: 24,
            ),
            shape: RoundedRectangleBorder(
              borderRadius: BorderRadius.circular(14),
            ),
            clipBehavior: Clip.antiAlias,
            child: ConstrainedBox(
              constraints: BoxConstraints(
                maxWidth: maxWidth,
                maxHeight: maxHeight,
              ),
              child: builder(bctx),
            ),
          );
        }),
      ),
    );
  } finally {
    popModalIframeMute();
  }
}
