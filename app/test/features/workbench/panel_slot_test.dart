/// Widget tests for the PanelSlot (T19 — M2 plugin platform).
///
/// These tests exercise the layout + tab-switching behaviour by injecting
/// a stand-in for the plugin WebView so we don't need to pump a real
/// platform WebView. The fake [WorkbenchService] mirrors the pattern
/// used in `activity_bar_test.dart` (subclass the real service, override
/// only the getters we exercise) so the widget's `ListenableBuilder`
/// path uses the production `notifyListeners` code path.
library;

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/panel_slot.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_service.dart';

class _FakeService extends WorkbenchService {
  _FakeService(this._panels)
      : super(api: _NoopApi(), showMessage: _noShow);

  List<WorkbenchPanel> _panels;

  @override
  List<WorkbenchPanel> get panels => _panels;

  set panelsList(List<WorkbenchPanel> v) {
    _panels = v;
    notifyListeners();
  }
}

class _NoopApi extends ApiClient {
  _NoopApi() : super(baseUrl: 'http://fake.local');
}

void _noShow(String _, {bool isError = false}) {}

WorkbenchPanel _panel({
  String pluginName = 'plug',
  required String id,
  String title = 'Panel',
  String entry = 'panel.html',
  String render = 'webview',
}) =>
    WorkbenchPanel(
      pluginName: pluginName,
      id: id,
      title: title,
      entry: entry,
      render: render,
    );

/// Wrap [child] in a MaterialApp and force a virtual viewport size so
/// `MediaQuery.of(context).size.width` returns [width]. Using the
/// MaterialApp-level `MediaQuery` override is simpler + more reliable
/// than poking `tester.view.physicalSize` + `devicePixelRatio`.
Widget _wrap(Widget child, {double width = 800, double height = 800}) {
  return MaterialApp(
    home: MediaQuery(
      data: MediaQueryData(size: Size(width, height)),
      child: Scaffold(body: child),
    ),
  );
}

/// Builder args captured by the test stub so assertions stay concise.
class _BuilderCall {
  _BuilderCall({
    required this.pluginName,
    required this.viewId,
    required this.entryPath,
    required this.baseUrl,
    required this.bearerToken,
  });
  final String pluginName;
  final String viewId;
  final String entryPath;
  final String baseUrl;
  final String bearerToken;
}

void main() {
  group('PanelSlot', () {
    testWidgets('empty panels → SizedBox.shrink', (tester) async {
      final service = _FakeService(const []);

      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'http://127.0.0.1:8640',
          bearerToken: 'tok',
        ),
      ));

      final size = tester.getSize(find.byType(PanelSlot));
      expect(size.width * size.height, 0,
          reason: 'empty panels must render zero footprint');
    });

    testWidgets('single panel on tablet width renders tab + webview',
        (tester) async {
      final service = _FakeService([_panel(id: 'p1', title: 'P1')]);
      final calls = <_BuilderCall>[];

      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'http://127.0.0.1:8640',
          bearerToken: 'tok',
          webViewBuilder: (
            _, {
            required pluginName,
            required viewId,
            required entryPath,
            required baseUrl,
            required bearerToken,
          }) {
            calls.add(_BuilderCall(
              pluginName: pluginName,
              viewId: viewId,
              entryPath: entryPath,
              baseUrl: baseUrl,
              bearerToken: bearerToken,
            ));
            return const Text('WEBVIEW-STUB');
          },
        ),
        width: 800,
      ));

      // Tab label is visible + webview builder got called once.
      expect(find.text('P1'), findsOneWidget);
      expect(find.text('WEBVIEW-STUB'), findsOneWidget);
      expect(calls.length, 1);
      expect(calls.single.pluginName, 'plug');
      expect(calls.single.viewId, 'p1');
      expect(calls.single.entryPath, 'panel.html');
      expect(calls.single.baseUrl, 'http://127.0.0.1:8640');
      expect(calls.single.bearerToken, 'tok');
    });

    testWidgets('two panels render two tabs; tap second switches',
        (tester) async {
      final service = _FakeService([
        _panel(id: 'p1', title: 'P1', entry: 'p1.html'),
        _panel(id: 'p2', title: 'P2', entry: 'p2.html'),
      ]);
      final calls = <_BuilderCall>[];

      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'http://127.0.0.1:8640',
          bearerToken: 'tok',
          webViewBuilder: (
            _, {
            required pluginName,
            required viewId,
            required entryPath,
            required baseUrl,
            required bearerToken,
          }) {
            calls.add(_BuilderCall(
              pluginName: pluginName,
              viewId: viewId,
              entryPath: entryPath,
              baseUrl: baseUrl,
              bearerToken: bearerToken,
            ));
            return Text('WV-$viewId');
          },
        ),
        width: 800,
      ));

      // Initial: first panel active.
      expect(find.text('P1'), findsOneWidget);
      expect(find.text('P2'), findsOneWidget);
      expect(find.text('WV-p1'), findsOneWidget);
      expect(find.text('WV-p2'), findsNothing);

      // Tap second tab — now second panel should render.
      await tester.tap(find.text('P2'));
      await tester.pumpAndSettle();

      expect(find.text('WV-p2'), findsOneWidget);
      expect(find.text('WV-p1'), findsNothing);
      final lastCall = calls.last;
      expect(lastCall.entryPath, 'p2.html');
      expect(lastCall.viewId, 'p2');
    });

    testWidgets('phone width is collapsed by default — chevron + first title',
        (tester) async {
      final service = _FakeService([
        _panel(id: 'p1', title: 'P1'),
        _panel(id: 'p2', title: 'P2'),
      ]);

      int builderCalls = 0;
      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'http://127.0.0.1:8640',
          bearerToken: 'tok',
          webViewBuilder: (
            _, {
            required pluginName,
            required viewId,
            required entryPath,
            required baseUrl,
            required bearerToken,
          }) {
            builderCalls++;
            return const Text('WEBVIEW-STUB');
          },
        ),
        width: 360,
      ));

      // Collapsed bar: chevron + first-panel title. No webview built yet.
      expect(find.byIcon(Icons.keyboard_arrow_up), findsOneWidget);
      expect(find.text('P1'), findsOneWidget);
      expect(find.text('WEBVIEW-STUB'), findsNothing);
      expect(builderCalls, 0);

      // Height is the collapsed height (default 32).
      final size = tester.getSize(find.byType(PanelSlot));
      expect(size.height, 32);
    });

    testWidgets('phone width: tapping drag handle expands the panel',
        (tester) async {
      final service = _FakeService([
        _panel(id: 'p1', title: 'P1'),
      ]);

      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'http://127.0.0.1:8640',
          bearerToken: 'tok',
          webViewBuilder: (
            _, {
            required pluginName,
            required viewId,
            required entryPath,
            required baseUrl,
            required bearerToken,
          }) =>
              const Text('WEBVIEW-STUB'),
        ),
        width: 360,
      ));

      // Pre: collapsed.
      expect(find.text('WEBVIEW-STUB'), findsNothing);

      await tester.tap(find.byIcon(Icons.keyboard_arrow_up));
      await tester.pumpAndSettle();

      // Post: expanded body visible with the webview.
      expect(find.text('WEBVIEW-STUB'), findsOneWidget);
      // Down chevron replaces the up one once expanded.
      expect(find.byIcon(Icons.keyboard_arrow_down), findsOneWidget);
      final size = tester.getSize(find.byType(PanelSlot));
      expect(size.height, 280);
    });

    testWidgets('service.notifyListeners adds a new panel tab', (tester) async {
      final service = _FakeService([_panel(id: 'p1', title: 'P1')]);

      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'http://127.0.0.1:8640',
          bearerToken: 'tok',
          webViewBuilder: (
            _, {
            required pluginName,
            required viewId,
            required entryPath,
            required baseUrl,
            required bearerToken,
          }) =>
              Text('WV-$viewId'),
        ),
        width: 800,
      ));

      expect(find.text('P2'), findsNothing);

      service.panelsList = [
        _panel(id: 'p1', title: 'P1'),
        _panel(id: 'p2', title: 'P2'),
      ];
      await tester.pump();

      expect(find.text('P2'), findsOneWidget);
    });

    testWidgets('unmount disposes cleanly (no pending setState)',
        (tester) async {
      final service = _FakeService([_panel(id: 'p1', title: 'P1')]);

      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'http://127.0.0.1:8640',
          bearerToken: 'tok',
          webViewBuilder: (
            _, {
            required pluginName,
            required viewId,
            required entryPath,
            required baseUrl,
            required bearerToken,
          }) =>
              const Text('WEBVIEW-STUB'),
        ),
        width: 360,
      ));

      // Expand first so the animation controller has to spin down.
      await tester.tap(find.byIcon(Icons.keyboard_arrow_up));
      await tester.pump(const Duration(milliseconds: 10));

      // Replace the tree — unmounts PanelSlot mid-animation.
      await tester.pumpWidget(const MaterialApp(
        home: Scaffold(body: Text('GONE')),
      ));
      await tester.pumpAndSettle();

      expect(tester.takeException(), isNull);
    });

    testWidgets('webViewBuilder receives pluginName/viewId/entryPath + base+token',
        (tester) async {
      final service = _FakeService([
        _panel(
          pluginName: 'kanban',
          id: 'board',
          title: 'Kanban',
          entry: 'panel/index.html',
        ),
      ]);
      _BuilderCall? lastCall;

      await tester.pumpWidget(_wrap(
        PanelSlot(
          service: service,
          baseUrl: 'https://api.example.com',
          bearerToken: 'jwt-abc',
          webViewBuilder: (
            _, {
            required pluginName,
            required viewId,
            required entryPath,
            required baseUrl,
            required bearerToken,
          }) {
            lastCall = _BuilderCall(
              pluginName: pluginName,
              viewId: viewId,
              entryPath: entryPath,
              baseUrl: baseUrl,
              bearerToken: bearerToken,
            );
            return const Text('WEBVIEW-STUB');
          },
        ),
        width: 800,
      ));

      expect(lastCall, isNotNull);
      expect(lastCall!.pluginName, 'kanban');
      expect(lastCall!.viewId, 'board');
      expect(lastCall!.entryPath, 'panel/index.html');
      expect(lastCall!.baseUrl, 'https://api.example.com');
      expect(lastCall!.bearerToken, 'jwt-abc');
    });
  });
}
