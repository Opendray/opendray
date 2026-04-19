import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_service.dart';

class _FakeApi extends ApiClient {
  _FakeApi() : super(baseUrl: 'http://fake.local');

  FlatContributions contribsReturn = FlatContributions.empty;
  Exception? contribsThrow;
  InvokeResult? invokeReturn;
  Exception? invokeThrow;

  int contribsCalls = 0;
  final List<List<Object?>> invokeCalls = [];

  @override
  Future<FlatContributions> getContributions() async {
    contribsCalls++;
    final err = contribsThrow;
    if (err != null) throw err;
    return contribsReturn;
  }

  @override
  Future<InvokeResult> invokePluginCommand(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    invokeCalls.add([pluginName, commandId, args]);
    final err = invokeThrow;
    if (err != null) throw err;
    return invokeReturn ?? const InvokeResult(kind: 'notify', message: '');
  }
}

void main() {
  group('WorkbenchService.refresh', () {
    test('populates contributions from ApiClient', () async {
      final api = _FakeApi()
        ..contribsReturn = const FlatContributions(
          commands: [
            WorkbenchCommand(
              pluginName: 'time-ninja',
              id: 'time.start',
              title: 'Start',
            ),
          ],
          statusBar: [
            WorkbenchStatusBarItem(
              pluginName: 'time-ninja',
              id: 'bar',
              text: '🍅',
            ),
          ],
          keybindings: [
            WorkbenchKeybinding(
              pluginName: 'time-ninja',
              command: 'time.start',
              key: 'ctrl+alt+p',
            ),
          ],
          menus: {
            'appBar/right': [
              WorkbenchMenuEntry(
                pluginName: 'time-ninja',
                command: 'time.start',
              ),
            ],
          },
        );
      final service =
          WorkbenchService(api: api, showMessage: (_, {bool isError = false}) {});

      expect(service.commands, isEmpty);
      await service.refresh();
      expect(service.commands.length, 1);
      expect(service.statusBarItems.length, 1);
      expect(service.keybindings.length, 1);
      expect(service.menus['appBar/right']!.length, 1);
      expect(service.contributions.commands.single.id, 'time.start');
      expect(api.contribsCalls, 1);
      expect(service.isLoading, isFalse);
      expect(service.lastError, isNull);
    });

    test('notifies listeners on change', () async {
      final api = _FakeApi()
        ..contribsReturn = const FlatContributions(commands: [
          WorkbenchCommand(pluginName: 'p', id: 'c', title: 'T'),
        ]);
      final service =
          WorkbenchService(api: api, showMessage: (_, {bool isError = false}) {});

      var notifyCount = 0;
      service.addListener(() => notifyCount++);
      await service.refresh();
      expect(notifyCount, greaterThanOrEqualTo(1));
    });

    test('captures error on failure without crashing', () async {
      final api = _FakeApi()
        ..contribsThrow =
            ApiException(500, 'boom', '/api/workbench/contributions');
      String? lastMessage;
      bool? lastIsError;
      final service = WorkbenchService(
        api: api,
        showMessage: (m, {bool isError = false}) {
          lastMessage = m;
          lastIsError = isError;
        },
      );
      await service.refresh();
      expect(service.lastError, isNotNull);
      expect(service.commands, isEmpty);
      expect(lastMessage, contains('boom'));
      expect(lastIsError, isTrue);
    });
  });

  group('WorkbenchService.invoke', () {
    test('notify kind calls showMessage with the message', () async {
      final api = _FakeApi()
        ..invokeReturn = const InvokeResult(
          kind: 'notify',
          message: 'Pomodoro started',
        );
      String? lastMessage;
      bool? lastIsError;
      final service = WorkbenchService(
        api: api,
        showMessage: (m, {bool isError = false}) {
          lastMessage = m;
          lastIsError = isError;
        },
      );
      final r = await service.invoke('time-ninja', 'time.start');
      expect(r?.kind, 'notify');
      expect(lastMessage, 'Pomodoro started');
      expect(lastIsError, isFalse);
      expect(api.invokeCalls.single[0], 'time-ninja');
      expect(api.invokeCalls.single[1], 'time.start');
    });

    test('openUrl returns the Result to caller (no message)', () async {
      final api = _FakeApi()
        ..invokeReturn =
            const InvokeResult(kind: 'openUrl', url: 'https://example.com');
      String? lastMessage;
      final service = WorkbenchService(
        api: api,
        showMessage: (m, {bool isError = false}) => lastMessage = m,
      );
      final r = await service.invoke('p', 'c');
      expect(r?.kind, 'openUrl');
      expect(r?.url, 'https://example.com');
      expect(lastMessage, isNull);
    });

    test('exec returns the Result to caller', () async {
      final api = _FakeApi()
        ..invokeReturn =
            const InvokeResult(kind: 'exec', output: 'hi\n', exit: 0);
      final service =
          WorkbenchService(api: api, showMessage: (_, {bool isError = false}) {});
      final r = await service.invoke('p', 'c');
      expect(r?.kind, 'exec');
      expect(r?.output, 'hi\n');
    });

    test('runTask returns the Result to caller', () async {
      final api = _FakeApi()
        ..invokeReturn =
            const InvokeResult(kind: 'runTask', taskId: 'abc');
      final service =
          WorkbenchService(api: api, showMessage: (_, {bool isError = false}) {});
      final r = await service.invoke('p', 'c');
      expect(r?.kind, 'runTask');
      expect(r?.taskId, 'abc');
    });

    test('permission denied → error SnackBar, returns null', () async {
      final api = _FakeApi()
        ..invokeThrow =
            PluginPermissionDeniedException('p', 'c', 'exec not granted');
      String? lastMessage;
      bool? lastIsError;
      final service = WorkbenchService(
        api: api,
        showMessage: (m, {bool isError = false}) {
          lastMessage = m;
          lastIsError = isError;
        },
      );
      final r = await service.invoke('p', 'c');
      expect(r, isNull);
      expect(lastIsError, isTrue);
      expect(lastMessage, contains('exec not granted'));
    });

    test('deferred (501) → hint mentions M2', () async {
      final api = _FakeApi()
        ..invokeThrow = PluginCommandUnavailableException(
          'p',
          'c',
          'host run kind',
          deferred: true,
        );
      String? lastMessage;
      bool? lastIsError;
      final service = WorkbenchService(
        api: api,
        showMessage: (m, {bool isError = false}) {
          lastMessage = m;
          lastIsError = isError;
        },
      );
      final r = await service.invoke('p', 'c');
      expect(r, isNull);
      expect(lastIsError, isTrue);
      expect(lastMessage!.toLowerCase(), contains('m2'));
    });

    test('not-found (404) → unavailable message', () async {
      final api = _FakeApi()
        ..invokeThrow =
            PluginCommandUnavailableException('p', 'c', 'no such command');
      String? lastMessage;
      final service = WorkbenchService(
        api: api,
        showMessage: (m, {bool isError = false}) => lastMessage = m,
      );
      final r = await service.invoke('p', 'c');
      expect(r, isNull);
      expect(lastMessage, contains('no such command'));
    });

    test('generic ApiException → error SnackBar with body', () async {
      final api = _FakeApi()
        ..invokeThrow = ApiException(500, 'internal error', '/foo');
      String? lastMessage;
      bool? lastIsError;
      final service = WorkbenchService(
        api: api,
        showMessage: (m, {bool isError = false}) {
          lastMessage = m;
          lastIsError = isError;
        },
      );
      final r = await service.invoke('p', 'c');
      expect(r, isNull);
      expect(lastMessage, contains('internal error'));
      expect(lastIsError, isTrue);
    });

    test('forwards args to ApiClient', () async {
      final api = _FakeApi();
      final service =
          WorkbenchService(api: api, showMessage: (_, {bool isError = false}) {});
      await service.invoke('p', 'c', args: {'x': 1});
      expect(api.invokeCalls.single[2], {'x': 1});
    });
  });
}
