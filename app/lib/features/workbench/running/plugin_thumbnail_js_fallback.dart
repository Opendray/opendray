import 'dart:convert';

import 'package:flutter/foundation.dart';

import '../webview_host.dart';
import 'running_plugins_models.dart';

/// iOS-friendly fallback for webview thumbnails. `RepaintBoundary.toImage`
/// over a `WKWebView` platform view often yields a blank frame, so we
/// ask the page itself to render a snapshot.
///
/// Mechanism: wrap the page's `documentElement` in an inline SVG
/// `<foreignObject>`, draw that SVG into a `<canvas>`, and read back a
/// PNG data URL. This is an established DOM technique (the same one
/// `html-to-image` uses under the hood) and doesn't require any
/// bundled dependency.
///
/// Known limits:
///   • Cross-origin `<img>` or CSS `url()` resources taint the canvas
///     and cause `toDataURL` to throw a `SecurityError`. Plugin assets
///     are served from OpenDray's own origin so this only fires for
///     plugins that embed external CDN resources — in that case we
///     return null and the capture chain falls through to the icon
///     placeholder.
///   • Safari / WebKit treat some foreignObject quirks differently
///     across versions. Any unhandled error is swallowed below and
///     falls through as `null`.
const String _captureJs = '''
(function() {
  try {
    var el = document.documentElement;
    var rect = el.getBoundingClientRect();
    var w = Math.max(1, Math.min(Math.floor(rect.width), 1600));
    var h = Math.max(1, Math.min(Math.floor(rect.height), 1600));
    var serializer = new XMLSerializer();
    var clone = el.cloneNode(true);
    var xhtml = serializer.serializeToString(clone);
    var svg =
      '<svg xmlns="http://www.w3.org/2000/svg" width="' + w + '" height="' + h + '">'
      + '<foreignObject width="100%" height="100%">'
      + '<div xmlns="http://www.w3.org/1999/xhtml">' + xhtml + '</div>'
      + '</foreignObject></svg>';
    var img = new Image();
    return new Promise(function(resolve) {
      img.onload = function() {
        try {
          var canvas = document.createElement('canvas');
          // Target ~0.4x of the viewport to match the Flutter side's
          // pixelRatio budget.
          canvas.width = Math.max(1, Math.floor(w * 0.4));
          canvas.height = Math.max(1, Math.floor(h * 0.4));
          var ctx = canvas.getContext('2d');
          ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
          resolve(canvas.toDataURL('image/png'));
        } catch (e) {
          resolve(null);
        }
      };
      img.onerror = function() { resolve(null); };
      img.src = 'data:image/svg+xml;charset=utf-8,' + encodeURIComponent(svg);
    });
  } catch (e) {
    return null;
  }
})()
''';

/// Evaluates [_captureJs] against the webview registered for [entry]'s
/// plugin name and decodes the returned data URL. Returns `null` on
/// any error; never throws.
Future<Uint8List?> webviewJsThumbnailFallback(
    RunningPluginEntry entry) async {
  final controller = PluginWebView.controllers[_pluginNameOf(entry.id)];
  if (controller == null) return null;
  try {
    final raw = await controller.runJavaScriptReturningResult(_captureJs);
    return _decodeDataUrl(raw);
  } catch (e) {
    if (kDebugMode) {
      debugPrint('webviewJsThumbnailFallback: $e');
    }
    return null;
  }
}

/// Extracts the `<pluginName>` part of a `webview:<pluginName>` entry id.
String _pluginNameOf(String entryId) {
  const prefix = 'webview:';
  return entryId.startsWith(prefix) ? entryId.substring(prefix.length) : entryId;
}

/// `runJavaScriptReturningResult` wraps values in platform-specific
/// quoting — iOS returns a String, Android sometimes returns a
/// JSON-encoded String. Peel back to the raw data URL and then decode.
Uint8List? _decodeDataUrl(Object? raw) {
  if (raw == null) return null;
  String str = raw.toString();
  // Some platforms wrap the result in a JSON string literal.
  if (str.length >= 2 && str.startsWith('"') && str.endsWith('"')) {
    try {
      final decoded = jsonDecode(str);
      if (decoded is String) str = decoded;
    } catch (_) {
      // Fall through — treat as raw.
    }
  }
  if (str == 'null' || str.isEmpty) return null;
  final commaIdx = str.indexOf(',');
  if (!str.startsWith('data:image/png;base64,') || commaIdx < 0) return null;
  final b64 = str.substring(commaIdx + 1);
  try {
    return base64Decode(b64);
  } catch (_) {
    return null;
  }
}
