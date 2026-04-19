/// Tests for [MenuSlot] (T22). The slot renders `PopupMenuButton` with
/// one item per contributed `WorkbenchMenuEntry` for the slot id;
/// collapses to `SizedBox.shrink` when no entries are contributed.
library;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/workbench/menu_slot.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_sources.dart';

/// Minimal mutable fake, same shape as the `_FakeSource` used by the
/// status-bar strip tests. Lets widget tests drive the slot without
/// constructing a full `WorkbenchService`.
class _FakeMenuSource extends ChangeNotifier implements MenuSource {
  _FakeMenuSource(this._entries);

  Map<String, List<WorkbenchMenuEntry>> _entries;

  @override
  List<WorkbenchMenuEntry> entriesFor(String id) =>
      _entries[id] ?? const [];

  set entries(Map<String, List<WorkbenchMenuEntry>> v) {
    _entries = v;
    notifyListeners();
  }

  final List<List<Object?>> invokeCalls = [];

  @override
  Future<void> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    invokeCalls.add([pluginName, commandId, args]);
  }
}

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(
        appBar: AppBar(
          actions: [child],
        ),
      ),
    );

WorkbenchMenuEntry _entry({
  String pluginName = 'time-ninja',
  String command = 'time.start',
  String group = '',
}) =>
    WorkbenchMenuEntry(
      pluginName: pluginName,
      command: command,
      group: group,
    );

void main() {
  group('MenuSlot', () {
    testWidgets('empty source renders SizedBox.shrink', (tester) async {
      final source = _FakeMenuSource({});
      await tester.pumpWidget(_wrap(
        MenuSlot(id: 'appBar/right', source: source),
      ));

      expect(find.byType(PopupMenuButton<WorkbenchMenuEntry>), findsNothing);
      expect(find.byType(SizedBox), findsWidgets);
    });

    testWidgets('non-matching slot id yields empty', (tester) async {
      final source = _FakeMenuSource({
        'other': [_entry()],
      });
      await tester.pumpWidget(_wrap(
        MenuSlot(id: 'missing', source: source),
      ));
      expect(find.byType(PopupMenuButton<WorkbenchMenuEntry>), findsNothing);
    });

    testWidgets('non-empty source renders PopupMenuButton with icon',
        (tester) async {
      final source = _FakeMenuSource({
        'appBar/right': [_entry()],
      });
      await tester.pumpWidget(_wrap(
        MenuSlot(
          id: 'appBar/right',
          source: source,
          icon: const Icon(Icons.extension),
        ),
      ));

      expect(find.byType(PopupMenuButton<WorkbenchMenuEntry>), findsOneWidget);
      expect(find.byIcon(Icons.extension), findsOneWidget);
    });

    testWidgets('tapping the button opens the popup with one item per entry',
        (tester) async {
      final source = _FakeMenuSource({
        'appBar/right': [
          _entry(command: 'time.start'),
          _entry(command: 'time.stop'),
        ],
      });
      await tester.pumpWidget(_wrap(
        MenuSlot(id: 'appBar/right', source: source),
      ));

      await tester.tap(find.byType(PopupMenuButton<WorkbenchMenuEntry>));
      await tester.pumpAndSettle();

      expect(find.text('time.start'), findsOneWidget);
      expect(find.text('time.stop'), findsOneWidget);
    });

    testWidgets('group sort: non-empty groups appear before empty-group',
        (tester) async {
      final source = _FakeMenuSource({
        'appBar/right': [
          _entry(command: 'later', group: ''),
          _entry(command: 'earlier', group: 'g1'),
        ],
      });
      await tester.pumpWidget(_wrap(
        MenuSlot(id: 'appBar/right', source: source),
      ));

      await tester.tap(find.byType(PopupMenuButton<WorkbenchMenuEntry>));
      await tester.pumpAndSettle();

      // Both items rendered; verify "earlier" (group g1) is above "later"
      // (empty group sorts last).
      final dyEarlier = tester.getCenter(find.text('earlier')).dy;
      final dyLater = tester.getCenter(find.text('later')).dy;
      expect(dyEarlier, lessThan(dyLater));
    });

    testWidgets('selecting an entry calls source.invoke with plugin + command',
        (tester) async {
      final source = _FakeMenuSource({
        'appBar/right': [
          _entry(pluginName: 'time-ninja', command: 'time.start'),
        ],
      });
      await tester.pumpWidget(_wrap(
        MenuSlot(id: 'appBar/right', source: source),
      ));

      await tester.tap(find.byType(PopupMenuButton<WorkbenchMenuEntry>));
      await tester.pumpAndSettle();

      await tester.tap(find.text('time.start'));
      await tester.pumpAndSettle();

      expect(source.invokeCalls, hasLength(1));
      expect(source.invokeCalls.single[0], 'time-ninja');
      expect(source.invokeCalls.single[1], 'time.start');
    });

    testWidgets('listens to source changes; re-renders when new entries arrive',
        (tester) async {
      final source = _FakeMenuSource({});
      await tester.pumpWidget(_wrap(
        MenuSlot(id: 'appBar/right', source: source),
      ));

      // Starts empty, no button.
      expect(find.byType(PopupMenuButton<WorkbenchMenuEntry>), findsNothing);

      source.entries = {
        'appBar/right': [_entry()],
      };
      await tester.pump();

      expect(find.byType(PopupMenuButton<WorkbenchMenuEntry>), findsOneWidget);
    });
  });
}
