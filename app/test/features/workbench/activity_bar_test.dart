import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/activity_bar.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_service.dart';

/// A minimal fake that subclasses the real [WorkbenchService] so the
/// widget's `ListenableBuilder` + `service.openView` call paths exercise
/// the production code. We intercept network-backed bits by never
/// invoking [WorkbenchService.refresh] or [WorkbenchService.invoke].
class _FakeService extends WorkbenchService {
  _FakeService(this._items)
      : super(
          api: _NoopApi(),
          showMessage: _noShow,
        );

  List<WorkbenchActivityBarItem> _items;

  @override
  List<WorkbenchActivityBarItem> get activityBarItems => _items;

  @override
  List<WorkbenchView> get views => const [];

  set items(List<WorkbenchActivityBarItem> v) {
    _items = v;
    notifyListeners();
  }

  final List<String> openCalls = [];

  @override
  void openView(String viewID) {
    openCalls.add(viewID);
    super.openView(viewID);
  }
}

class _NoopApi extends ApiClient {
  _NoopApi() : super(baseUrl: 'http://fake.local');
}

void _noShow(String _, {bool isError = false}) {}

WorkbenchActivityBarItem _item({
  String pluginName = 'plug',
  required String id,
  String icon = '🍅',
  String title = 'Item',
  String? viewId,
}) =>
    WorkbenchActivityBarItem(
      pluginName: pluginName,
      id: id,
      icon: icon,
      title: title,
      viewId: viewId ?? '$id-view',
    );

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('ActivityBar', () {
    testWidgets('empty list collapses to SizedBox.shrink', (tester) async {
      final service = _FakeService(const []);

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));

      // Rail collapses: no buttons, no row chrome.
      expect(find.byType(InkWell), findsNothing);
      final size = tester.getSize(find.byType(ActivityBar));
      expect(size.width * size.height, 0);
    });

    testWidgets('3 items render 3 tap targets', (tester) async {
      final service = _FakeService([
        _item(id: 'a'),
        _item(id: 'b'),
        _item(id: 'c'),
      ]);

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));

      // One tap target per item, no More button yet.
      expect(find.byTooltip('Item'), findsNWidgets(3));
      expect(find.byTooltip('More'), findsNothing);
    });

    testWidgets('5 items render 3 + More; tapping More opens sheet', (tester) async {
      final service = _FakeService([
        _item(id: 'a', title: 'A'),
        _item(id: 'b', title: 'B'),
        _item(id: 'c', title: 'C'),
        _item(id: 'd', title: 'D'),
        _item(id: 'e', title: 'E'),
      ]);

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));

      // 3 inline item buttons + 1 More button.
      expect(find.byTooltip('A'), findsOneWidget);
      expect(find.byTooltip('B'), findsOneWidget);
      expect(find.byTooltip('C'), findsOneWidget);
      expect(find.byTooltip('D'), findsNothing);
      expect(find.byTooltip('E'), findsNothing);
      expect(find.byTooltip('More'), findsOneWidget);

      await tester.tap(find.byTooltip('More'));
      await tester.pumpAndSettle();

      // Bottom sheet lists the two overflow items.
      expect(find.text('D'), findsOneWidget);
      expect(find.text('E'), findsOneWidget);
    });

    testWidgets('tapping an item calls service.openView(viewId)', (tester) async {
      final service = _FakeService([
        _item(id: 'a', title: 'Aaa', viewId: 'view-a'),
      ]);

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));

      await tester.tap(find.byTooltip('Aaa'));
      await tester.pump();

      expect(service.openCalls, ['view-a']);
      expect(service.currentViewID, 'view-a');
    });

    testWidgets('tapping an overflow sheet item calls service.openView', (tester) async {
      final service = _FakeService([
        _item(id: 'a', title: 'A'),
        _item(id: 'b', title: 'B'),
        _item(id: 'c', title: 'C'),
        _item(id: 'd', title: 'D', viewId: 'view-d'),
        _item(id: 'e', title: 'E'),
      ]);

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));
      await tester.tap(find.byTooltip('More'));
      await tester.pumpAndSettle();

      await tester.tap(find.text('D'));
      await tester.pumpAndSettle();

      expect(service.openCalls, contains('view-d'));
    });

    testWidgets('item without viewId tap is a no-op', (tester) async {
      final service = _FakeService([
        WorkbenchActivityBarItem(
          pluginName: 'p',
          id: 'a',
          icon: 'X',
          title: 'NoView',
          viewId: '',
        ),
      ]);

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));

      await tester.tap(find.byTooltip('NoView'));
      await tester.pump();

      expect(service.openCalls, isEmpty);
      expect(service.currentViewID, isNull);
    });

    testWidgets('selected item (viewId == currentViewID) highlights', (tester) async {
      final service = _FakeService([
        _item(id: 'a', title: 'A', viewId: 'view-a'),
        _item(id: 'b', title: 'B', viewId: 'view-b'),
      ]);
      service.openView('view-b');

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));

      // Find the Container inside the selected button and check it has
      // a non-transparent background. We look up by tooltip then the
      // Container descendant.
      final selectedContainer = tester.widget<Container>(
        find.descendant(
          of: find.byTooltip('B'),
          matching: find.byType(Container),
        ),
      );
      final decoration = selectedContainer.decoration as BoxDecoration;
      expect(decoration.color, isNot(Colors.transparent));

      final unselectedContainer = tester.widget<Container>(
        find.descendant(
          of: find.byTooltip('A'),
          matching: find.byType(Container),
        ),
      );
      final unselectedDecoration = unselectedContainer.decoration as BoxDecoration;
      expect(unselectedDecoration.color, Colors.transparent);
    });

    testWidgets('rebuilds when service notifies', (tester) async {
      final service = _FakeService([
        _item(id: 'a', title: 'Before'),
      ]);

      await tester.pumpWidget(_wrap(ActivityBar(service: service)));
      expect(find.byTooltip('Before'), findsOneWidget);

      service.items = [_item(id: 'a', title: 'After')];
      await tester.pump();

      expect(find.byTooltip('Before'), findsNothing);
      expect(find.byTooltip('After'), findsOneWidget);
    });

    testWidgets('horizontal axis uses phone bottom-nav layout', (tester) async {
      final service = _FakeService([_item(id: 'a')]);

      await tester.pumpWidget(_wrap(ActivityBar(service: service, axis: Axis.horizontal)));

      // Sanity: one row of item buttons at the bottom, non-zero height.
      final size = tester.getSize(find.byType(ActivityBar));
      expect(size.height, greaterThan(0));
      expect(find.byTooltip('Item'), findsOneWidget);
    });
  });
}
