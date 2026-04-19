/// Tests for [WorkbenchStatusBarSource] and [WorkbenchMenuSource] — the
/// thin adapters that bridge `WorkbenchService` to the decoupled
/// source interfaces consumed by `StatusBarStrip` (T20) and `MenuSlot` (T22).
///
/// Strategy: build a real `WorkbenchService` using a `_FakeApi` that
/// mirrors the pattern from `workbench_service_test.dart`. The adapters
/// delegate via `addListener` + forward calls, so we exercise them
/// against the real service rather than mocking.
library;

import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_service.dart';
import 'package:opendray/features/workbench/workbench_sources.dart';

class _FakeApi extends ApiClient {
  _FakeApi() : super(baseUrl: 'http://fake.local');

  FlatContributions contribsReturn = FlatContributions.empty;
  InvokeResult? invokeReturn;

  final List<List<Object?>> invokeCalls = [];

  @override
  Future<FlatContributions> getContributions() async => contribsReturn;

  @override
  Future<InvokeResult> invokePluginCommand(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    invokeCalls.add([pluginName, commandId, args]);
    return invokeReturn ?? const InvokeResult(kind: 'notify', message: '');
  }
}

WorkbenchService _makeService({
  FlatContributions contribs = FlatContributions.empty,
  _FakeApi? api,
}) {
  final a = api ?? _FakeApi();
  a.contribsReturn = contribs;
  return WorkbenchService(api: a, showMessage: (_, {bool isError = false}) {});
}

void main() {
  group('WorkbenchStatusBarSource', () {
    test('forwards statusBarItems from service', () async {
      final service = _makeService(
        contribs: const FlatContributions(
          statusBar: [
            WorkbenchStatusBarItem(
              pluginName: 'p',
              id: 's',
              text: 'hello',
            ),
          ],
        ),
      );
      await service.refresh();

      final source = WorkbenchStatusBarSource(service);
      addTearDown(source.dispose);

      expect(source.statusBarItems, hasLength(1));
      expect(source.statusBarItems.first.text, 'hello');
    });

    test('notifies on service change', () async {
      final api = _FakeApi()
        ..contribsReturn = const FlatContributions(
          statusBar: [
            WorkbenchStatusBarItem(pluginName: 'p', id: 's', text: 'one'),
          ],
        );
      final service = _makeService(api: api);

      final source = WorkbenchStatusBarSource(service);
      addTearDown(source.dispose);

      var fired = 0;
      source.addListener(() => fired++);

      await service.refresh();
      // refresh fires at least once (loading set + finally block).
      expect(fired, greaterThanOrEqualTo(1));

      // Second refresh with different items fires again.
      api.contribsReturn = const FlatContributions(
        statusBar: [
          WorkbenchStatusBarItem(pluginName: 'p', id: 's', text: 'two'),
        ],
      );
      final before = fired;
      await service.refresh();
      expect(fired, greaterThan(before));
      expect(source.statusBarItems.single.text, 'two');
    });

    test('invoke delegates to service.invoke with args', () async {
      final api = _FakeApi();
      final service = _makeService(api: api);
      final source = WorkbenchStatusBarSource(service);
      addTearDown(source.dispose);

      await source.invoke('plug', 'cmd', args: {'x': 1});

      expect(api.invokeCalls, hasLength(1));
      expect(api.invokeCalls.single[0], 'plug');
      expect(api.invokeCalls.single[1], 'cmd');
      expect(api.invokeCalls.single[2], {'x': 1});
    });

    test('dispose removes listener from service', () async {
      final api = _FakeApi()
        ..contribsReturn = const FlatContributions(
          statusBar: [
            WorkbenchStatusBarItem(pluginName: 'p', id: 's', text: 'one'),
          ],
        );
      final service = _makeService(api: api);
      final source = WorkbenchStatusBarSource(service);

      var fired = 0;
      source.addListener(() => fired++);

      await service.refresh();
      expect(fired, greaterThanOrEqualTo(1));

      source.dispose();

      // After dispose the forwarding listener is unsubscribed. Another
      // service refresh should not move the adapter's fire count.
      final before = fired;
      api.contribsReturn = const FlatContributions(
        statusBar: [
          WorkbenchStatusBarItem(pluginName: 'p', id: 's', text: 'two'),
        ],
      );
      await service.refresh();
      expect(fired, before);
    });
  });

  group('WorkbenchMenuSource', () {
    test('entriesFor forwards menus from service', () async {
      final service = _makeService(
        contribs: const FlatContributions(
          menus: {
            'appBar/right': [
              WorkbenchMenuEntry(pluginName: 'p', command: 'c.run'),
            ],
          },
        ),
      );
      await service.refresh();

      final source = WorkbenchMenuSource(service);
      addTearDown(source.dispose);

      expect(source.entriesFor('appBar/right'), hasLength(1));
      expect(source.entriesFor('appBar/right').first.command, 'c.run');
    });

    test('entriesFor(unknown) returns empty', () async {
      final service = _makeService(
        contribs: const FlatContributions(
          menus: {
            'appBar/right': [
              WorkbenchMenuEntry(pluginName: 'p', command: 'c.run'),
            ],
          },
        ),
      );
      await service.refresh();

      final source = WorkbenchMenuSource(service);
      addTearDown(source.dispose);

      expect(source.entriesFor('missing/slot'), isEmpty);
    });

    test('notifies on service change', () async {
      final api = _FakeApi()
        ..contribsReturn = const FlatContributions(
          menus: {
            'slot': [WorkbenchMenuEntry(pluginName: 'p', command: 'c1')],
          },
        );
      final service = _makeService(api: api);

      final source = WorkbenchMenuSource(service);
      addTearDown(source.dispose);

      var fired = 0;
      source.addListener(() => fired++);

      await service.refresh();
      expect(fired, greaterThanOrEqualTo(1));

      api.contribsReturn = const FlatContributions(
        menus: {
          'slot': [WorkbenchMenuEntry(pluginName: 'p', command: 'c2')],
        },
      );
      final before = fired;
      await service.refresh();
      expect(fired, greaterThan(before));
      expect(source.entriesFor('slot').single.command, 'c2');
    });

    test('invoke delegates to service.invoke with args', () async {
      final api = _FakeApi();
      final service = _makeService(api: api);
      final source = WorkbenchMenuSource(service);
      addTearDown(source.dispose);

      await source.invoke('plug', 'cmd', args: {'y': 2});

      expect(api.invokeCalls, hasLength(1));
      expect(api.invokeCalls.single[0], 'plug');
      expect(api.invokeCalls.single[1], 'cmd');
      expect(api.invokeCalls.single[2], {'y': 2});
    });

    test('dispose removes listener from service', () async {
      final api = _FakeApi()
        ..contribsReturn = const FlatContributions(
          menus: {
            'slot': [WorkbenchMenuEntry(pluginName: 'p', command: 'c1')],
          },
        );
      final service = _makeService(api: api);
      final source = WorkbenchMenuSource(service);

      var fired = 0;
      source.addListener(() => fired++);

      await service.refresh();
      expect(fired, greaterThanOrEqualTo(1));

      source.dispose();

      final before = fired;
      api.contribsReturn = const FlatContributions(
        menus: {
          'slot': [WorkbenchMenuEntry(pluginName: 'p', command: 'c2')],
        },
      );
      await service.refresh();
      expect(fired, before);
    });
  });
}
