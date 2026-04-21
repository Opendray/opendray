import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/workbench/status_bar_strip.dart';
import 'package:opendray/features/workbench/workbench_models.dart';

/// Minimal mutable fake. Pushes a new item list on `items=` and fires
/// `notifyListeners` so `ListenableBuilder` inside [StatusBarStrip] rebuilds.
class _FakeSource extends ChangeNotifier implements StatusBarSource {
  _FakeSource(this._items);

  List<WorkbenchStatusBarItem> _items;

  @override
  List<WorkbenchStatusBarItem> get statusBarItems => _items;

  set items(List<WorkbenchStatusBarItem> v) {
    _items = v;
    notifyListeners();
  }

  final List<Map<String, dynamic>> invokeCalls = [];

  @override
  Future<void> invoke(
    String pluginName,
    String commandId, {
    Map<String, dynamic>? args,
  }) async {
    invokeCalls.add({
      'pluginName': pluginName,
      'commandId': commandId,
      'args': args,
    });
  }
}

Widget _wrap(Widget child) => MaterialApp(
      home: Scaffold(body: Align(alignment: Alignment.bottomCenter, child: child)),
    );

WorkbenchStatusBarItem _item({
  String pluginName = 'time-ninja',
  String id = 'status',
  String text = 'item',
  String tooltip = '',
  String command = 'time.start',
  String alignment = 'right',
  int priority = 0,
}) =>
    WorkbenchStatusBarItem(
      pluginName: pluginName,
      id: id,
      text: text,
      tooltip: tooltip,
      command: command,
      alignment: alignment,
      priority: priority,
    );

void main() {
  group('StatusBarStrip', () {
    testWidgets('renders text and tooltip for each item', (tester) async {
      final source = _FakeSource([
        _item(id: 'a', text: '🍅 25:00', tooltip: 'Pomodoro timer', command: 'time.start'),
        _item(id: 'b', text: 'Build OK', tooltip: 'Last build succeeded', command: 'build.status'),
      ]);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      expect(find.text('🍅 25:00'), findsOneWidget);
      expect(find.text('Build OK'), findsOneWidget);

      // Tooltips are materialised as Tooltip widgets with the expected messages.
      final tooltips = tester
          .widgetList<Tooltip>(find.byType(Tooltip))
          .map((t) => t.message)
          .toSet();
      expect(tooltips, contains('Pomodoro timer'));
      expect(tooltips, contains('Last build succeeded'));
    });

    testWidgets('left and right alignment groups separate correctly', (tester) async {
      final source = _FakeSource([
        _item(id: 'l1', text: 'L1', alignment: 'left', priority: 10),
        _item(id: 'l2', text: 'L2', alignment: 'left', priority: 5),
        _item(id: 'r1', text: 'R1', alignment: 'right', priority: 10),
        _item(id: 'r2', text: 'R2', alignment: 'right', priority: 5),
      ]);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      // Spacer sits between the left and right groups.
      expect(find.byType(Spacer), findsOneWidget);

      // Left items should appear before the spacer in the row, right after.
      final rowFinder = find.descendant(
        of: find.byType(StatusBarStrip),
        matching: find.byType(Row),
      );
      expect(rowFinder, findsWidgets);

      // Sanity: all four labels rendered.
      expect(find.text('L1'), findsOneWidget);
      expect(find.text('L2'), findsOneWidget);
      expect(find.text('R1'), findsOneWidget);
      expect(find.text('R2'), findsOneWidget);
    });

    testWidgets('priority sort is descending within group', (tester) async {
      final source = _FakeSource([
        _item(id: 'low', text: 'P1', priority: 1),
        _item(id: 'high', text: 'P100', priority: 100),
        _item(id: 'mid', text: 'P50', priority: 50),
      ]);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      // Rendered order — higher priority first (P100 → P50 → P1).
      final dxHigh = tester.getCenter(find.text('P100')).dx;
      final dxMid = tester.getCenter(find.text('P50')).dx;
      final dxLow = tester.getCenter(find.text('P1')).dx;
      expect(dxHigh, lessThan(dxMid));
      expect(dxMid, lessThan(dxLow));
    });

    testWidgets('empty source renders nothing (zero height)', (tester) async {
      final source = _FakeSource(const []);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      // The widget itself takes no visual space when empty.
      final size = tester.getSize(find.byType(StatusBarStrip));
      expect(size.height, 0);
    });

    testWidgets('tapping an item calls source.invoke with plugin + command', (tester) async {
      final source = _FakeSource([
        _item(
          pluginName: 'time-ninja',
          id: 'tomato',
          text: '🍅 25:00',
          command: 'time.start',
        ),
      ]);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      await tester.tap(find.text('🍅 25:00'));
      await tester.pump();

      expect(source.invokeCalls, hasLength(1));
      expect(source.invokeCalls.first['pluginName'], 'time-ninja');
      expect(source.invokeCalls.first['commandId'], 'time.start');
    });

    testWidgets('listens to source changes and rebuilds', (tester) async {
      final source = _FakeSource([
        _item(id: 'a', text: 'before'),
      ]);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      expect(find.text('before'), findsOneWidget);
      expect(find.text('after'), findsNothing);

      source.items = [_item(id: 'a', text: 'after')];
      await tester.pump();

      expect(find.text('before'), findsNothing);
      expect(find.text('after'), findsOneWidget);
    });

    testWidgets('rendered text preserves full UTF-8 including emoji', (tester) async {
      final source = _FakeSource([
        _item(id: 'pomo', text: '🍅 25:00'),
      ]);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      expect(find.text('🍅 25:00'), findsOneWidget);
    });

    testWidgets('non-declared alignment values default to right', (tester) async {
      final source = _FakeSource([
        _item(id: 'weirdo', text: 'MysteryChip', alignment: 'somewhere_else'),
        _item(id: 'l1', text: 'LeftChip', alignment: 'left'),
      ]);

      await tester.pumpWidget(_wrap(StatusBarStrip(source: source)));

      // With a Spacer between the groups, left chip should be to the left of
      // the mystery chip (which was coerced into the right group).
      final dxLeft = tester.getCenter(find.text('LeftChip')).dx;
      final dxMystery = tester.getCenter(find.text('MysteryChip')).dx;
      expect(dxLeft, lessThan(dxMystery));
    });
  });
}
