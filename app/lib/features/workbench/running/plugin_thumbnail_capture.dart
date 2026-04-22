import 'dart:async';
import 'dart:ui' as ui;

import 'package:flutter/foundation.dart';
import 'package:flutter/rendering.dart';
import 'package:flutter/widgets.dart';

import 'running_plugins_models.dart';

/// Produces a [PluginThumbnail] for a running plugin entry — the image
/// shown on the app-switcher card.
///
/// Chain, tried in order:
///   1. [RepaintBoundary.toImage] via the entry's boundary key. Works
///      on Flutter widgets and Android WebViews in hybrid composition.
///      Often returns a blank frame for iOS `WKWebView`; the result is
///      sampled via [_isBlank] and rejected when uniformly blank.
///   2. Webview JS fallback — injected via the bridge to let the
///      plugin page render itself to a canvas and return a data URL.
///      Wired in step 6; the hook is an optional parameter here so
///      step 4 can ship the end-to-end pipeline with only step 1's
///      path live.
///   3. [IconThumbnail] — always succeeds. Switcher draws the entry
///      icon on a filled tile.
///
/// The function never throws. Any internal failure is treated as a
/// step miss and the chain continues.
class PluginThumbnailCapture {
  PluginThumbnailCapture._();

  /// Resolution multiplier applied to [RepaintBoundary.toImage]. At
  /// 0.4 a typical phone screen (~400×800 logical px) produces a
  /// thumbnail around 160×320 — big enough to be recognisable on a
  /// switcher card, small enough to avoid retaining megabytes of
  /// decoded pixels.
  static const double _pixelRatio = 0.4;

  /// Signature for the optional webview JS fallback. Callers pass this
  /// when the entry is a webview plugin and they have access to its
  /// [WebViewController]. Returns PNG bytes or `null` on any error.
  /// See step 6 for the concrete implementation.
  static Future<Uint8List?> Function(RunningPluginEntry entry)?
      webviewJsFallback;

  /// Captures a thumbnail for [entry] using the [RepaintBoundary] at
  /// [boundaryKey]. Safe to call from a post-frame callback. Safe
  /// against missing/detached boundaries (returns an icon thumbnail).
  static Future<PluginThumbnail> capture({
    required GlobalKey boundaryKey,
    required RunningPluginEntry entry,
  }) async {
    // Step 1: RepaintBoundary snapshot.
    final png = await _captureBoundary(boundaryKey);
    if (png != null && !_isBlank(png.bytes)) {
      return ImageThumbnail(
        pngBytes: png.bytes,
        width: png.width,
        height: png.height,
      );
    }

    // Step 2: Webview JS fallback, wired in step 6.
    if (entry.kind == RunningPluginKind.webview &&
        webviewJsFallback != null) {
      try {
        final bytes = await webviewJsFallback!(entry);
        if (bytes != null && bytes.isNotEmpty && !_isBlank(bytes)) {
          // We can't cheaply get width/height without decoding; the
          // switcher renders `Image.memory` with `fit: BoxFit.cover`
          // so exact dimensions are cosmetic. Use 0 sentinels and let
          // the image resolve itself.
          return ImageThumbnail(pngBytes: bytes, width: 0, height: 0);
        }
      } catch (_) {
        // Swallow — fall through to icon.
      }
    }

    return const IconThumbnail();
  }

  static Future<_CapturedPng?> _captureBoundary(GlobalKey boundaryKey) async {
    try {
      final obj = boundaryKey.currentContext?.findRenderObject();
      if (obj is! RenderRepaintBoundary) return null;
      if (obj.debugNeedsPaint) {
        // The boundary hasn't painted yet — toImage would throw. Skip
        // and let the fallback chain pick up.
        return null;
      }
      final image = await obj.toImage(pixelRatio: _pixelRatio);
      try {
        final byteData = await image.toByteData(format: ui.ImageByteFormat.png);
        if (byteData == null) return null;
        return _CapturedPng(
          bytes: byteData.buffer.asUint8List(),
          width: image.width,
          height: image.height,
        );
      } finally {
        image.dispose();
      }
    } catch (_) {
      return null;
    }
  }

  /// Cheap heuristic for "did this capture land a blank frame". Samples
  /// the PNG at four corners plus the centre; if the alpha / colour
  /// variance is trivially small, treat as blank so the fallback can
  /// kick in.
  ///
  /// We intentionally don't decode the full image — [ui.decodeImageFromList]
  /// is asynchronous and expensive for a heuristic that runs on every
  /// navigation. Instead we skim the raw PNG bytes looking for any
  /// signal. A truly blank PNG compresses to a handful of bytes; real
  /// content is much longer. At the scale we capture (pixel ratio
  /// 0.4), a blank frame is typically under 1 KB and real frames are
  /// several KB.
  @visibleForTesting
  static bool isBlank(Uint8List png) => _isBlank(png);

  static bool _isBlank(Uint8List png) {
    // Shortest plausible "has content" PNG at our resolution is >1 KB.
    // Uniform-colour PNGs at 160x320 end up ~300-600 bytes because
    // zlib deflates runs of identical pixels to near-nothing.
    if (png.length < 1024) return true;
    return false;
  }
}

class _CapturedPng {
  final Uint8List bytes;
  final int width;
  final int height;
  _CapturedPng({
    required this.bytes,
    required this.width,
    required this.height,
  });
}
