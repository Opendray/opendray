import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/workbench/view_host.dart';
import 'package:opendray/features/workbench/workbench_models.dart';
import 'package:opendray/features/workbench/workbench_service.dart';

/// Fake service subclassing the real one so `ListenableBuilder` inside
/// [ViewHost] sees real notifications. We override the getters [ViewHost]
/// touches and track close calls to validate the unmounted-plugin branch.
class _FakeService extends WorkbenchService {
  _FakeService({
    List<WorkbenchView> views = const [],
    String? currentViewID,
  })  : _views = views,
        super(api: _NoopApi(), showMessage: _noShow) {
    if (currentViewID != null) {
      openView(currentViewID);
    }
  }

  List<WorkbenchView> _views;

  @override
  List<WorkbenchView> get views => _views;

  int closeCalls = 0;

  @override
  void closeView() {
    closeCalls++;
    super.closeView();
  }

  set viewsList(List<WorkbenchView> v) {
    _views = v;
    notifyListeners();
  }
}

class _NoopApi extends ApiClient {
  _NoopApi() : super(baseUrl: 'http://fake.local');
}

void _noShow(String _, {bool isError = false}) {}

Widget _wrap(Widget child) => MaterialApp(home: Scaffold(body: child));

void main() {
  group('ViewHost', () {
    testWidgets('currentViewID null renders fallback', (tester) async {
      final service = _FakeService();

      await tester.pumpWidget(_wrap(ViewHost(
        service: service,
        baseUrl: 'http://127.0.0.1:8640',
        bearerToken: 'tok',
        fallback: const Text('FALLBACK'),
      )));

      expect(find.text('FALLBACK'), findsOneWidget);
    });

    testWidgets('webview-render view routes to the webview builder', (tester) async {
      // T16's PluginWebView constructs a real WebViewController in its
      // initState, which fails in a headless widget test (no platform
      // webview implementation). We swap it via the test seam so we
      // assert routing + wiring without touching the platform layer.
      final service = _FakeService(
        views: const [
          WorkbenchView(
            pluginName: 'kanban',
            id: 'board',
            title: 'Kanban Board',
            container: 'activityBar',
            render: 'webview',
            entry: 'index.html',
          ),
        ],
        currentViewID: 'board',
      );

      WorkbenchView? routed;
      await tester.pumpWidget(_wrap(ViewHost(
        service: service,
        baseUrl: 'http://127.0.0.1:8640',
        bearerToken: 'tok',
        fallback: const Text('FALLBACK'),
        webViewBuilder: (view) {
          routed = view;
          return const Text('WEBVIEW-STUB');
        },
      )));

      expect(routed?.id, 'board');
      expect(routed?.pluginName, 'kanban');
      expect(routed?.entry, 'index.html');
      expect(find.text('WEBVIEW-STUB'), findsOneWidget);
      expect(find.text('FALLBACK'), findsNothing);
      // Top bar surfaces the view title + a close button.
      expect(find.text('Kanban Board'), findsOneWidget);
      expect(find.byTooltip('Close view'), findsOneWidget);
    });

    testWidgets('declarative-render view shows the M5 placeholder', (tester) async {
      final service = _FakeService(
        views: const [
          WorkbenchView(
            pluginName: 'p',
            id: 'v',
            title: 'Decl',
            render: 'declarative',
            entry: '',
          ),
        ],
        currentViewID: 'v',
      );

      await tester.pumpWidget(_wrap(ViewHost(
        service: service,
        baseUrl: 'http://127.0.0.1:8640',
        bearerToken: 'tok',
        fallback: const Text('FALLBACK'),
      )));

      expect(find.textContaining('M5'), findsOneWidget);
    });

    testWidgets('id without a matching view falls back + closes', (tester) async {
      final service = _FakeService(
        views: const [],
        currentViewID: 'ghost',
      );

      await tester.pumpWidget(_wrap(ViewHost(
        service: service,
        baseUrl: 'http://127.0.0.1:8640',
        bearerToken: 'tok',
        fallback: const Text('FALLBACK'),
      )));

      // First frame: fallback rendered, post-frame schedules closeView.
      expect(find.text('FALLBACK'), findsOneWidget);

      // Pump the post-frame callback that calls service.closeView.
      await tester.pump();
      expect(service.closeCalls, greaterThanOrEqualTo(1));
      expect(service.currentViewID, isNull);
    });

    testWidgets('close button calls service.closeView', (tester) async {
      final service = _FakeService(
        views: const [
          WorkbenchView(
            pluginName: 'p',
            id: 'v',
            title: 'T',
            render: 'declarative',
            entry: '',
          ),
        ],
        currentViewID: 'v',
      );

      await tester.pumpWidget(_wrap(ViewHost(
        service: service,
        baseUrl: 'http://127.0.0.1:8640',
        bearerToken: 'tok',
        fallback: const Text('FALLBACK'),
      )));

      await tester.tap(find.byTooltip('Close view'));
      await tester.pump();

      expect(service.closeCalls, greaterThanOrEqualTo(1));
      expect(service.currentViewID, isNull);
      // Fallback now visible.
      expect(find.text('FALLBACK'), findsOneWidget);
    });

    testWidgets('unknown render kind degrades to placeholder (not crash)', (tester) async {
      final service = _FakeService(
        views: const [
          WorkbenchView(
            pluginName: 'p',
            id: 'v',
            title: 'Weird',
            render: 'quantum',
            entry: '',
          ),
        ],
        currentViewID: 'v',
      );

      await tester.pumpWidget(_wrap(ViewHost(
        service: service,
        baseUrl: 'http://127.0.0.1:8640',
        bearerToken: 'tok',
        fallback: const Text('FALLBACK'),
      )));

      expect(find.textContaining('M5'), findsOneWidget);
      expect(tester.takeException(), isNull);
    });
  });
}
