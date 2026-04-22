import 'dart:typed_data';

import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/workbench/running/running_plugins_models.dart';
import 'package:opendray/features/workbench/running/running_plugins_service.dart';

RunningPluginEntry _seed(String id, {DateTime? openedAt}) {
  final now = openedAt ?? DateTime.now();
  return RunningPluginEntry(
    id: id,
    titleKey: id,
    icon: const IconData(0xe000),
    route: '/browser/$id',
    kind: RunningPluginKind.builtin,
    builder: (_) => const SizedBox(),
    openedAt: now,
    lastActiveAt: now,
  );
}

void main() {
  group('RunningPluginsService', () {
    test('ensureOpened adds a new entry and notifies once', () {
      final svc = RunningPluginsService();
      var notifies = 0;
      svc.addListener(() => notifies++);

      svc.ensureOpened(_seed('a'));
      expect(svc.entries.length, 1);
      expect(svc.entries.first.id, 'a');
      expect(notifies, 1);
    });

    test('ensureOpened is idempotent on same id', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      var notifies = 0;
      svc.addListener(() => notifies++);

      svc.ensureOpened(_seed('a'));
      expect(svc.entries.length, 1);
      expect(notifies, 0, reason: 'duplicate open should not notify');
    });

    test('setActive updates activeId and bumps lastActiveAt', () {
      final svc = RunningPluginsService();
      final t0 = DateTime(2024);
      svc.ensureOpened(_seed('a', openedAt: t0));
      expect(svc.activeId, isNull);

      svc.setActive('a');
      expect(svc.activeId, 'a');
      expect(svc.entries.first.lastActiveAt.isAfter(t0), isTrue);
    });

    test('setActive for unknown id is a no-op', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      var notifies = 0;
      svc.addListener(() => notifies++);

      svc.setActive('does-not-exist');
      expect(svc.activeId, isNull);
      expect(notifies, 0);
    });

    test('setActive same id twice only notifies once', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      svc.setActive('a');
      var notifies = 0;
      svc.addListener(() => notifies++);

      svc.setActive('a');
      expect(notifies, 0);
    });

    test('clearActive sets activeId null and remembers previous', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      svc.setActive('a');

      svc.clearActive();
      expect(svc.activeId, isNull);
      expect(svc.previousActiveId, 'a');
    });

    test('close removes entry and clears active if matching', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      svc.ensureOpened(_seed('b'));
      svc.setActive('a');

      svc.close('a');
      expect(svc.entries.length, 1);
      expect(svc.entries.first.id, 'b');
      expect(svc.activeId, isNull);
    });

    test('close of non-active entry leaves active untouched', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      svc.ensureOpened(_seed('b'));
      svc.setActive('a');

      svc.close('b');
      expect(svc.activeId, 'a');
      expect(svc.entries.length, 1);
    });

    test('updateThumbnail replaces the entry with new thumbnail', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      final bytes = Uint8List.fromList([1, 2, 3, 4]);

      svc.updateThumbnail(
          'a', ImageThumbnail(pngBytes: bytes, width: 10, height: 10));
      final thumb = svc.entries.first.thumbnail;
      expect(thumb, isA<ImageThumbnail>());
      expect((thumb as ImageThumbnail).pngBytes, bytes);
    });

    test('updateThumbnail on missing id is silently dropped', () {
      final svc = RunningPluginsService();
      expect(() => svc.updateThumbnail('nope', const IconThumbnail()),
          returnsNormally);
    });

    test('entries list is unmodifiable', () {
      final svc = RunningPluginsService();
      svc.ensureOpened(_seed('a'));
      expect(() => svc.entries.add(_seed('b')), throwsUnsupportedError);
    });
  });
}
