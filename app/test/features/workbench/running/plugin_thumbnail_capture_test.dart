import 'dart:typed_data';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/workbench/running/plugin_thumbnail_capture.dart';
import 'package:opendray/features/workbench/running/running_plugins_models.dart';

RunningPluginEntry _webviewEntry() {
  return RunningPluginEntry(
    id: 'webview:kanban',
    titleKey: 'Kanban',
    icon: const IconData(0xe000),
    route: '/browser/plugin/kanban',
    kind: RunningPluginKind.webview,
    builder: (_) => const SizedBox(),
    openedAt: DateTime.now(),
    lastActiveAt: DateTime.now(),
  );
}

void main() {
  group('PluginThumbnailCapture.isBlank', () {
    test('treats short byte sequences as blank', () {
      expect(PluginThumbnailCapture.isBlank(Uint8List(500)), isTrue);
    });

    test('treats longer byte sequences as non-blank', () {
      expect(PluginThumbnailCapture.isBlank(Uint8List(4096)), isFalse);
    });
  });

  group('PluginThumbnailCapture.capture fallback chain', () {
    setUp(() => PluginThumbnailCapture.webviewJsFallback = null);

    test('returns IconThumbnail when boundary key has no render object',
        () async {
      final key = GlobalKey();
      final thumb = await PluginThumbnailCapture.capture(
        boundaryKey: key,
        entry: _webviewEntry(),
      );
      expect(thumb, isA<IconThumbnail>());
    });

    test(
        'falls through to webview JS fallback when boundary capture misses',
        () async {
      PluginThumbnailCapture.webviewJsFallback = (_) async =>
          Uint8List.fromList(List.filled(4096, 7));

      final key = GlobalKey();
      final thumb = await PluginThumbnailCapture.capture(
        boundaryKey: key,
        entry: _webviewEntry(),
      );
      expect(thumb, isA<ImageThumbnail>());
    });

    test('treats a short JS-fallback payload as blank and returns icon',
        () async {
      PluginThumbnailCapture.webviewJsFallback = (_) async => Uint8List(100);

      final key = GlobalKey();
      final thumb = await PluginThumbnailCapture.capture(
        boundaryKey: key,
        entry: _webviewEntry(),
      );
      expect(thumb, isA<IconThumbnail>());
    });

    test('treats a null JS-fallback response as miss', () async {
      PluginThumbnailCapture.webviewJsFallback = (_) async => null;

      final key = GlobalKey();
      final thumb = await PluginThumbnailCapture.capture(
        boundaryKey: key,
        entry: _webviewEntry(),
      );
      expect(thumb, isA<IconThumbnail>());
    });

    test('swallows exceptions from the JS fallback', () async {
      PluginThumbnailCapture.webviewJsFallback =
          (_) async => throw StateError('boom');

      final key = GlobalKey();
      final thumb = await PluginThumbnailCapture.capture(
        boundaryKey: key,
        entry: _webviewEntry(),
      );
      expect(thumb, isA<IconThumbnail>());
    });
  });
}
