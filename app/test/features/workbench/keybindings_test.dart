import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/keybindings.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_service.dart';

/// Minimal ApiClient test double. Subclasses the real ApiClient and
/// overrides the two plugin-platform methods — the rest of the Dio
/// machinery is never touched in these tests.
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

void main() {
  group('parseKeyCombo', () {
    test('parses modifier + letter', () {
      final k = parseKeyCombo('ctrl+alt+p')!;
      expect(k.keys, contains(LogicalKeyboardKey.control));
      expect(k.keys, contains(LogicalKeyboardKey.alt));
      expect(k.keys, contains(LogicalKeyboardKey.keyP));
    });

    test('is case insensitive', () {
      final a = parseKeyCombo('CTRL+ALT+P')!;
      final b = parseKeyCombo('ctrl+alt+p')!;
      expect(a.keys, equals(b.keys));
    });

    test('tolerates surrounding whitespace', () {
      final k = parseKeyCombo('  ctrl + alt + p  ')!;
      expect(k.keys, contains(LogicalKeyboardKey.keyP));
    });

    test('parses shift+enter', () {
      final k = parseKeyCombo('shift+enter')!;
      expect(k.keys, contains(LogicalKeyboardKey.shift));
      expect(k.keys, contains(LogicalKeyboardKey.enter));
    });

    test('parses a lone function key', () {
      final k = parseKeyCombo('f5')!;
      expect(k.keys, equals({LogicalKeyboardKey.f5}));
    });

    test('parses F1–F12', () {
      expect(parseKeyCombo('f1')!.keys, contains(LogicalKeyboardKey.f1));
      expect(parseKeyCombo('f12')!.keys, contains(LogicalKeyboardKey.f12));
    });

    test('cmd aliases to meta', () {
      final k = parseKeyCombo('cmd+p')!;
      expect(k.keys, contains(LogicalKeyboardKey.meta));
      expect(k.keys, contains(LogicalKeyboardKey.keyP));
    });

    test('meta keyword works', () {
      final k = parseKeyCombo('meta+p')!;
      expect(k.keys, contains(LogicalKeyboardKey.meta));
    });

    test('digit key', () {
      final k = parseKeyCombo('ctrl+1')!;
      expect(k.keys, contains(LogicalKeyboardKey.digit1));
    });

    test('special keys: escape, space, tab, backspace, delete', () {
      expect(parseKeyCombo('escape')!.keys,
          contains(LogicalKeyboardKey.escape));
      expect(parseKeyCombo('space')!.keys,
          contains(LogicalKeyboardKey.space));
      expect(parseKeyCombo('tab')!.keys, contains(LogicalKeyboardKey.tab));
      expect(parseKeyCombo('backspace')!.keys,
          contains(LogicalKeyboardKey.backspace));
      expect(parseKeyCombo('delete')!.keys,
          contains(LogicalKeyboardKey.delete));
    });

    test('arrow keys', () {
      expect(parseKeyCombo('up')!.keys,
          contains(LogicalKeyboardKey.arrowUp));
      expect(parseKeyCombo('down')!.keys,
          contains(LogicalKeyboardKey.arrowDown));
      expect(parseKeyCombo('left')!.keys,
          contains(LogicalKeyboardKey.arrowLeft));
      expect(parseKeyCombo('right')!.keys,
          contains(LogicalKeyboardKey.arrowRight));
    });

    test('empty string → null', () {
      expect(parseKeyCombo(''), isNull);
    });

    test('whitespace only → null', () {
      expect(parseKeyCombo('   '), isNull);
    });

    test('garbage → null', () {
      expect(parseKeyCombo('not a real combo'), isNull);
    });

    test('unknown key name → null', () {
      expect(parseKeyCombo('ctrl+flibbertigibbet'), isNull);
    });

    test('modifier-only (no real key) → null', () {
      expect(parseKeyCombo('ctrl'), isNull);
      expect(parseKeyCombo('ctrl+alt'), isNull);
    });
  });

  group('WorkbenchKeybindings widget', () {
    late _FakeApi api;
    late WorkbenchService service;

    setUp(() {
      api = _FakeApi();
      service = WorkbenchService(
        api: api,
        showMessage: (_, {bool isError = false}) {},
      );
    });

    testWidgets(
      'pressing a bound key invokes the plugin command',
      (tester) async {
        api.contribsReturn = const FlatContributions(
          keybindings: [
            WorkbenchKeybinding(
              pluginName: 'time-ninja',
              command: 'time.start',
              key: 'ctrl+alt+p',
            ),
          ],
        );
        await service.refresh();

        await tester.pumpWidget(MaterialApp(
          home: WorkbenchKeybindings(
            service: service,
            child: const Focus(autofocus: true, child: SizedBox.expand()),
          ),
        ));
        await tester.pumpAndSettle();

        await tester.sendKeyDownEvent(LogicalKeyboardKey.controlLeft);
        await tester.sendKeyDownEvent(LogicalKeyboardKey.altLeft);
        await tester.sendKeyEvent(LogicalKeyboardKey.keyP);
        await tester.sendKeyUpEvent(LogicalKeyboardKey.altLeft);
        await tester.sendKeyUpEvent(LogicalKeyboardKey.controlLeft);
        await tester.pump();

        expect(api.invokeCalls.length, 1);
        expect(api.invokeCalls.first[0], 'time-ninja');
        expect(api.invokeCalls.first[1], 'time.start');
      },
    );

    testWidgets('map rebuilds when service notifies', (tester) async {
      // Start with empty contributions.
      await tester.pumpWidget(MaterialApp(
        home: WorkbenchKeybindings(
          service: service,
          child: const Focus(autofocus: true, child: SizedBox.expand()),
        ),
      ));
      await tester.pumpAndSettle();

      await tester.sendKeyDownEvent(LogicalKeyboardKey.controlLeft);
      await tester.sendKeyDownEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyEvent(LogicalKeyboardKey.keyP);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.controlLeft);
      await tester.pump();
      expect(api.invokeCalls, isEmpty, reason: 'no bindings yet');

      // Now push a binding + notify.
      api.contribsReturn = const FlatContributions(
        keybindings: [
          WorkbenchKeybinding(
            pluginName: 'time-ninja',
            command: 'time.start',
            key: 'ctrl+alt+p',
          ),
        ],
      );
      await service.refresh();
      await tester.pump();

      await tester.sendKeyDownEvent(LogicalKeyboardKey.controlLeft);
      await tester.sendKeyDownEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyEvent(LogicalKeyboardKey.keyP);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.controlLeft);
      await tester.pump();

      expect(api.invokeCalls.length, 1);
    });

    testWidgets('mac override used on macOS', (tester) async {
      debugDefaultTargetPlatformOverride = TargetPlatform.macOS;

      api.contribsReturn = const FlatContributions(
        keybindings: [
          WorkbenchKeybinding(
            pluginName: 'time-ninja',
            command: 'time.start',
            key: 'ctrl+alt+p',
            mac: 'meta+alt+p',
          ),
        ],
      );
      await service.refresh();

      await tester.pumpWidget(MaterialApp(
        home: WorkbenchKeybindings(
          service: service,
          child: const Focus(autofocus: true, child: SizedBox.expand()),
        ),
      ));
      await tester.pumpAndSettle();

      // Non-mac combo must not fire on macOS.
      await tester.sendKeyDownEvent(LogicalKeyboardKey.controlLeft);
      await tester.sendKeyDownEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyEvent(LogicalKeyboardKey.keyP);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.controlLeft);
      await tester.pump();
      expect(api.invokeCalls, isEmpty);

      // Mac combo fires.
      await tester.sendKeyDownEvent(LogicalKeyboardKey.metaLeft);
      await tester.sendKeyDownEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyEvent(LogicalKeyboardKey.keyP);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.altLeft);
      await tester.sendKeyUpEvent(LogicalKeyboardKey.metaLeft);
      await tester.pump();
      expect(api.invokeCalls.length, 1);

      // Clear the override before the test body returns so the
      // foundation debug-vars invariant check passes.
      debugDefaultTargetPlatformOverride = null;
    });
  });
}
