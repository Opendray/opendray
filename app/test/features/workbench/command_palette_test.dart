import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/command_palette.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_service.dart';

class _FakeApi extends ApiClient {
  _FakeApi() : super(baseUrl: 'http://fake.local');

  FlatContributions contribsReturn = FlatContributions.empty;
  InvokeResult? invokeReturn;
  Exception? invokeThrow;
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
    final err = invokeThrow;
    if (err != null) throw err;
    return invokeReturn ?? const InvokeResult(kind: 'notify', message: 'ok');
  }
}

Future<WorkbenchService> _serviceWith(List<WorkbenchCommand> cmds) async {
  final api = _FakeApi()..contribsReturn = FlatContributions(commands: cmds);
  final service = WorkbenchService(
    api: api,
    showMessage: (_, {bool isError = false}) {},
  );
  await service.refresh();
  return service;
}

Widget _host(WorkbenchService service) {
  return MaterialApp(
    home: Builder(
      builder: (ctx) => Scaffold(
        body: Center(
          child: ElevatedButton(
            onPressed: () => CommandPalette.show(ctx, service),
            child: const Text('open'),
          ),
        ),
      ),
    ),
  );
}

void main() {
  testWidgets('renders one tile per command', (tester) async {
    final service = await _serviceWith(const [
      WorkbenchCommand(
          pluginName: 'p1', id: 'c1', title: 'First Command'),
      WorkbenchCommand(
          pluginName: 'p2', id: 'c2', title: 'Second Command'),
      WorkbenchCommand(
          pluginName: 'p3', id: 'c3', title: 'Third Command'),
    ]);

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();

    expect(find.text('First Command'), findsOneWidget);
    expect(find.text('Second Command'), findsOneWidget);
    expect(find.text('Third Command'), findsOneWidget);
  });

  testWidgets('typing filters to matching commands', (tester) async {
    final service = await _serviceWith(const [
      WorkbenchCommand(
          pluginName: 'time-ninja', id: 'time.start', title: 'Start Pomodoro'),
      WorkbenchCommand(
          pluginName: 'docs', id: 'docs.open', title: 'Open Docs'),
    ]);

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'pom');
    await tester.pumpAndSettle();

    expect(find.text('Start Pomodoro'), findsOneWidget);
    expect(find.text('Open Docs'), findsNothing);
  });

  testWidgets('filter matches plugin name too', (tester) async {
    final service = await _serviceWith(const [
      WorkbenchCommand(
          pluginName: 'time-ninja', id: 'time.start', title: 'Start'),
      WorkbenchCommand(
          pluginName: 'git-panel', id: 'git.pull', title: 'Pull'),
    ]);

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'ninja');
    await tester.pumpAndSettle();

    expect(find.text('Start'), findsOneWidget);
    expect(find.text('Pull'), findsNothing);
  });

  testWidgets('tapping a tile invokes the command and pops the dialog',
      (tester) async {
    final api = _FakeApi()
      ..contribsReturn = const FlatContributions(commands: [
        WorkbenchCommand(
            pluginName: 'time-ninja', id: 'time.start', title: 'Start'),
      ]);
    final service = WorkbenchService(
      api: api,
      showMessage: (_, {bool isError = false}) {},
    );
    await service.refresh();

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();

    await tester.tap(find.text('Start'));
    await tester.pumpAndSettle();

    expect(api.invokeCalls.length, 1);
    expect(api.invokeCalls.first[0], 'time-ninja');
    expect(api.invokeCalls.first[1], 'time.start');
    // Dialog dismissed.
    expect(find.byType(TextField), findsNothing);
  });

  testWidgets('empty service shows "No plugins installed" hint',
      (tester) async {
    final service = await _serviceWith(const []);

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();

    expect(find.textContaining('No plugins installed'), findsOneWidget);
  });

  testWidgets('non-matching query shows "No commands match"',
      (tester) async {
    final service = await _serviceWith(const [
      WorkbenchCommand(pluginName: 'p', id: 'c', title: 'Hello'),
    ]);

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();

    await tester.enterText(find.byType(TextField), 'zzzzz');
    await tester.pumpAndSettle();

    expect(find.textContaining('No commands match'), findsOneWidget);
  });

  testWidgets('Esc closes the palette', (tester) async {
    final service = await _serviceWith(const [
      WorkbenchCommand(pluginName: 'p', id: 'c', title: 'X'),
    ]);

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();
    expect(find.byType(TextField), findsOneWidget);

    await tester.sendKeyEvent(LogicalKeyboardKey.escape);
    await tester.pumpAndSettle();

    expect(find.byType(TextField), findsNothing);
  });

  testWidgets('Up/Down arrow shifts selection; Enter invokes',
      (tester) async {
    final api = _FakeApi()
      ..contribsReturn = const FlatContributions(commands: [
        WorkbenchCommand(pluginName: 'p', id: 'c1', title: 'Alpha'),
        WorkbenchCommand(pluginName: 'p', id: 'c2', title: 'Bravo'),
        WorkbenchCommand(pluginName: 'p', id: 'c3', title: 'Charlie'),
      ]);
    final service = WorkbenchService(
      api: api,
      showMessage: (_, {bool isError = false}) {},
    );
    await service.refresh();

    await tester.pumpWidget(_host(service));
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();

    // Default selection is index 0. Down → 1, Down → 2, Up → 1.
    await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
    await tester.sendKeyEvent(LogicalKeyboardKey.arrowDown);
    await tester.sendKeyEvent(LogicalKeyboardKey.arrowUp);
    await tester.pumpAndSettle();

    await tester.sendKeyEvent(LogicalKeyboardKey.enter);
    await tester.pumpAndSettle();

    expect(api.invokeCalls.length, 1);
    expect(api.invokeCalls.first[1], 'c2');
  });
}
