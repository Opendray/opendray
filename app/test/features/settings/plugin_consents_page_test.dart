import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/api_client.dart';
import 'package:opendray/features/settings/plugin_consents_page.dart';

/// Fake ApiClient records invocations and returns scripted
/// PluginConsents snapshots. Scripted error injections flow through
/// [getThrow] / [revokeCapThrow] / [revokeAllThrow] so each test case
/// can drive a specific branch without MockWebServer plumbing.
class _FakeApi extends ApiClient {
  _FakeApi() : super(baseUrl: 'http://fake.local');

  PluginConsents getReturn = const PluginConsents(
    pluginName: 'kanban',
    perms: {},
  );
  Exception? getThrow;
  Exception? revokeCapThrow;
  Exception? revokeAllThrow;

  int getCalls = 0;
  final List<List<String>> revokeCapCalls = [];
  final List<String> revokeAllCalls = [];

  @override
  Future<PluginConsents> getPluginConsents(String pluginName) async {
    getCalls++;
    final err = getThrow;
    if (err != null) throw err;
    return getReturn;
  }

  @override
  Future<void> revokePluginCapability(String pluginName, String cap) async {
    revokeCapCalls.add([pluginName, cap]);
    final err = revokeCapThrow;
    if (err != null) throw err;
  }

  @override
  Future<void> revokeAllPluginConsents(String pluginName) async {
    revokeAllCalls.add(pluginName);
    final err = revokeAllThrow;
    if (err != null) throw err;
  }
}

PluginConsents _consents(Map<String, dynamic> perms) =>
    PluginConsents(pluginName: 'kanban', perms: perms);

Widget _host(_FakeApi api,
    {String pluginName = 'kanban',
    void Function(String, {bool isError})? onMessage}) {
  return MaterialApp(
    home: PluginConsentsPage(
      pluginName: pluginName,
      api: api,
      onMessage: onMessage,
    ),
  );
}

void main() {
  group('PluginConsentsPage — loading / empty / error', () {
    testWidgets('shows CircularProgressIndicator while loading',
        (tester) async {
      final api = _FakeApi();
      // Block getPluginConsents by scripting a never-completing future
      // via overriding the return only after the first pump.
      await tester.pumpWidget(_host(api));
      expect(find.byType(CircularProgressIndicator), findsOneWidget);
      // Let the microtask that resolved the scripted future drain, so
      // the next pump advances past loading and doesn't leave a dangling
      // timer for the test runner to complain about.
      await tester.pumpAndSettle();
    });

    testWidgets('renders empty state when ENOCONSENT', (tester) async {
      final api = _FakeApi()
        ..getThrow = PluginConsentNotFoundException('kanban');
      await tester.pumpWidget(_host(api));
      await tester.pumpAndSettle();

      expect(find.text('No consent on record'), findsOneWidget);
      expect(find.byType(Switch), findsNothing);
    });

    testWidgets('renders retry banner on other errors', (tester) async {
      final api = _FakeApi()
        ..getThrow = ApiException(500, 'boom', '/api/plugins/kanban/consents');
      await tester.pumpWidget(_host(api));
      await tester.pumpAndSettle();

      expect(find.textContaining('boom'), findsOneWidget);
      expect(find.text('Retry'), findsOneWidget);

      // Flip to a successful response and tap Retry — the page should
      // refetch and swap into the loaded state.
      api.getThrow = null;
      api.getReturn = _consents({'storage': true});
      await tester.tap(find.text('Retry'));
      await tester.pumpAndSettle();
      expect(find.byType(Switch), findsNWidgets(11));
    });
  });

  group('PluginConsentsPage — loaded state', () {
    testWidgets('renders 11 capability switches with correct state',
        (tester) async {
      final api = _FakeApi()
        ..getReturn = _consents({
          'storage': true,
          'secret': false,
          'session': 'read',
          'exec': ['git *'],
          'events': ['session.*'],
        });
      await tester.pumpWidget(_host(api));
      await tester.pumpAndSettle();

      // One switch per capability (11 total).
      expect(find.byType(Switch), findsNWidgets(11));

      // Granted caps → switch value true; ungranted → false.
      final switches = tester.widgetList<Switch>(find.byType(Switch)).toList();
      final onCount = switches.where((s) => s.value == true).length;
      // storage + session + exec + events = 4 granted
      expect(onCount, 4);
    });

    testWidgets('ungranted caps → switch is disabled', (tester) async {
      final api = _FakeApi()
        ..getReturn = _consents({'storage': true}); // only storage granted
      await tester.pumpWidget(_host(api));
      await tester.pumpAndSettle();

      final switches = tester.widgetList<Switch>(find.byType(Switch)).toList();
      final disabled = switches.where((s) => s.onChanged == null).length;
      // 10 ungranted caps → 10 disabled switches
      expect(disabled, 10);
    });

    testWidgets('toggling a granted cap off calls revokePluginCapability',
        (tester) async {
      final messages = <String>[];
      final api = _FakeApi()
        ..getReturn = _consents({'storage': true});
      await tester.pumpWidget(_host(api,
          onMessage: (msg, {bool isError = false}) => messages.add(msg)));
      await tester.pumpAndSettle();

      // Find the enabled switch — only storage is granted, so only
      // one switch has onChanged != null.
      final switchFinder = find.byWidgetPredicate((w) {
        if (w is! Switch) return false;
        return w.onChanged != null;
      });
      expect(switchFinder, findsOneWidget);

      // After the tap we script a follow-up getPluginConsents response
      // that reflects the revocation, so the refetch lands on a
      // consistent state.
      api.getReturn = _consents(const {});

      await tester.tap(switchFinder);
      await tester.pumpAndSettle();

      expect(api.revokeCapCalls, [
        ['kanban', 'storage']
      ]);
      expect(messages.any((m) => m.toLowerCase().contains('storage')), isTrue);
    });

    testWidgets('Revoke all opens confirm dialog then calls API',
        (tester) async {
      final messages = <String>[];
      final api = _FakeApi()
        ..getReturn = _consents({'storage': true, 'secret': true});
      await tester.pumpWidget(_host(api,
          onMessage: (msg, {bool isError = false}) => messages.add(msg)));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Revoke all'));
      await tester.pumpAndSettle();

      // Dialog is up.
      expect(find.byType(AlertDialog), findsOneWidget);

      // After confirming we script a 404 on refetch to prove the page
      // lands in the empty state rather than crashing.
      api.getThrow = PluginConsentNotFoundException('kanban');

      await tester.tap(find.widgetWithText(FilledButton, 'Revoke all'));
      await tester.pumpAndSettle();

      expect(api.revokeAllCalls, ['kanban']);
      expect(find.text('No consent on record'), findsOneWidget);
    });

    testWidgets('Revoke all cancel closes dialog without calling API',
        (tester) async {
      final api = _FakeApi()
        ..getReturn = _consents({'storage': true});
      await tester.pumpWidget(_host(api));
      await tester.pumpAndSettle();

      await tester.tap(find.text('Revoke all'));
      await tester.pumpAndSettle();

      await tester.tap(find.widgetWithText(TextButton, 'Cancel'));
      await tester.pumpAndSettle();

      expect(find.byType(AlertDialog), findsNothing);
      expect(api.revokeAllCalls, isEmpty);
    });
  });

  group('PluginConsentsPage — error surfacing on revoke', () {
    testWidgets('revoke failure calls onMessage with isError',
        (tester) async {
      final errors = <String>[];
      final api = _FakeApi()
        ..getReturn = _consents({'storage': true})
        ..revokeCapThrow = ApiException(500, 'server bad', '/x');
      await tester.pumpWidget(_host(api, onMessage: (msg, {bool isError = false}) {
        if (isError) errors.add(msg);
      }));
      await tester.pumpAndSettle();

      final switchFinder = find.byWidgetPredicate((w) {
        if (w is! Switch) return false;
        return w.onChanged != null;
      });
      await tester.tap(switchFinder);
      await tester.pumpAndSettle();

      expect(errors, isNotEmpty);
      expect(errors.first.toLowerCase(), contains('server bad'));
    });
  });
}
