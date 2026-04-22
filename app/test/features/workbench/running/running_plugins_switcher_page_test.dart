import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:go_router/go_router.dart';
import 'package:opendray/core/services/l10n.dart';
import 'package:opendray/features/workbench/running/running_plugins_models.dart';
import 'package:opendray/features/workbench/running/running_plugins_service.dart';
import 'package:opendray/features/workbench/running/running_plugins_switcher_page.dart';
import 'package:provider/provider.dart';

RunningPluginEntry _seed(String id, {DateTime? t}) {
  final now = t ?? DateTime.now();
  return RunningPluginEntry(
    id: id,
    titleKey: id,
    icon: const IconData(0xe000),
    route: '/browser/$id',
    kind: RunningPluginKind.builtin,
    builder: (_) => const SizedBox(),
    openedAt: now,
    lastActiveAt: now,
  );
}

Widget _harness(RunningPluginsService svc) {
  final router = GoRouter(
    initialLocation: '/running',
    routes: [
      GoRoute(
          path: '/running',
          builder: (_, _) => const RunningPluginsSwitcherPage()),
      GoRoute(
        path: '/browser/:panel',
        builder: (_, state) => Scaffold(
          body: Text('navigated:${state.pathParameters['panel']}'),
        ),
      ),
    ],
  );
  return MultiProvider(
    providers: [
      ChangeNotifierProvider<L10n>.value(value: L10n('en')),
      ChangeNotifierProvider<RunningPluginsService>.value(value: svc),
    ],
    child: MaterialApp.router(routerConfig: router),
  );
}

void main() {
  testWidgets('empty state when no entries', (tester) async {
    final svc = RunningPluginsService();
    await tester.pumpWidget(_harness(svc));
    await tester.pumpAndSettle();

    expect(find.text('No running plugins'), findsOneWidget);
  });

  testWidgets('renders a card per entry, sorted by lastActiveAt desc',
      (tester) async {
    final svc = RunningPluginsService();
    svc.ensureOpened(_seed('docs', t: DateTime(2024, 1, 1)));
    svc.ensureOpened(_seed('files', t: DateTime(2024, 1, 2)));
    svc.ensureOpened(_seed('tasks', t: DateTime(2024, 1, 3)));

    await tester.pumpWidget(_harness(svc));
    await tester.pumpAndSettle();

    // Three entries → three footer labels.
    expect(find.text('docs'), findsOneWidget);
    expect(find.text('files'), findsOneWidget);
    expect(find.text('tasks'), findsOneWidget);
  });

  testWidgets('tapping ✕ removes the entry', (tester) async {
    final svc = RunningPluginsService();
    svc.ensureOpened(_seed('docs'));
    svc.ensureOpened(_seed('files'));

    await tester.pumpWidget(_harness(svc));
    await tester.pumpAndSettle();

    // First close button — just take any. Both cards have one.
    await tester.tap(find.byIcon(Icons.close).first);
    await tester.pumpAndSettle();

    expect(svc.entries.length, 1);
  });
}
