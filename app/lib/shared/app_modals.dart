import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:pointer_interceptor/pointer_interceptor.dart';

import 'terminal_iframe_gate.dart';

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
/// and iframe muting.
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
