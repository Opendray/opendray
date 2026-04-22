import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/services/l10n.dart';
import 'package:opendray/features/workbench/running/running_plugins_host.dart';
import 'package:opendray/features/workbench/running/running_plugins_models.dart';
import 'package:opendray/features/workbench/running/running_plugins_service.dart';
import 'package:provider/provider.dart';

/// Builds a seed whose builder renders a counter widget — the counter
/// increments on each build, so we can tell when a plugin surface is
/// (a) rebuilt or (b) mounted fresh.
RunningPluginEntry _counterSeed(String id, {required ValueNotifier<int> taps}) {
  final now = DateTime.now();
  return RunningPluginEntry(
    id: id,
    titleKey: id,
    icon: Icons.bug_report,
    route: '/browser/$id',
    kind: RunningPluginKind.builtin,
    builder: (_) => _CounterPage(id: id, onTap: () => taps.value++),
    openedAt: now,
    lastActiveAt: now,
  );
}

class _CounterPage extends StatefulWidget {
  const _CounterPage({required this.id, required this.onTap});
  final String id;
  final VoidCallback onTap;

  @override
  State<_CounterPage> createState() => _CounterPageState();
}

/// State persists across rebuilds; used to verify that navigating away
/// and back does NOT remount the widget.
class _CounterPageState extends State<_CounterPage> {
  int _mountCount = 0;

  @override
  void initState() {
    super.initState();
    _mountCount = 1;
  }

  @override
  Widget build(BuildContext context) {
    return Center(
      child: ElevatedButton(
        onPressed: widget.onTap,
        child: Text('${widget.id}#$_mountCount'),
      ),
    );
  }
}

Widget _harness({
  required RunningPluginsService svc,
  required bool isPluginRoute,
  required String? targetPluginId,
  Widget? nonPluginChild,
}) {
  return MultiProvider(
    providers: [
      ChangeNotifierProvider<L10n>.value(value: L10n('en')),
      ChangeNotifierProvider<RunningPluginsService>.value(value: svc),
    ],
    child: MaterialApp(
      home: Scaffold(
        body: RunningPluginsHost(
          isPluginRoute: isPluginRoute,
          targetPluginId: targetPluginId,
          nonPluginChild:
              nonPluginChild ?? const Center(child: Text('NON-PLUGIN')),
        ),
      ),
    ),
  );
}

void main() {
  group('RunningPluginsHost — IndexedStack architecture', () {
    testWidgets('plugin route: taps reach the active plugin widget',
        (tester) async {
      final taps = ValueNotifier<int>(0);
      final svc = RunningPluginsService();
      final seed = _counterSeed('alpha', taps: taps);
      svc.ensureOpened(seed);
      svc.setActive('alpha');

      await tester.pumpWidget(_harness(
        svc: svc,
        isPluginRoute: true,
        targetPluginId: 'alpha',
      ));
      await tester.pumpAndSettle();

      expect(find.text('alpha#1'), findsOneWidget);

      await tester.tap(find.text('alpha#1'));
      await tester.pumpAndSettle();
      expect(taps.value, 1);
    });

    testWidgets('non-plugin route: non-plugin child is visible and tappable',
        (tester) async {
      var tapped = 0;
      final svc = RunningPluginsService();

      await tester.pumpWidget(_harness(
        svc: svc,
        isPluginRoute: false,
        targetPluginId: null,
        nonPluginChild: Center(
          child: ElevatedButton(
            onPressed: () => tapped++,
            child: const Text('DASHBOARD'),
          ),
        ),
      ));
      await tester.pumpAndSettle();

      expect(find.text('DASHBOARD'), findsOneWidget);
      await tester.tap(find.text('DASHBOARD'));
      await tester.pumpAndSettle();
      expect(tapped, 1);
    });

    testWidgets(
        'non-plugin route with offstage plugin: plugin not hittable, dashboard is',
        (tester) async {
      final taps = ValueNotifier<int>(0);
      var dashTaps = 0;
      final svc = RunningPluginsService();
      svc.ensureOpened(_counterSeed('alpha', taps: taps));
      svc.setActive('alpha');

      await tester.pumpWidget(_harness(
        svc: svc,
        isPluginRoute: false,
        targetPluginId: null,
        nonPluginChild: Center(
          child: ElevatedButton(
            onPressed: () => dashTaps++,
            child: const Text('HOME'),
          ),
        ),
      ));
      await tester.pumpAndSettle();

      // Plugin is offstage → not visible, not hittable.
      expect(find.text('alpha#1'), findsNothing);
      // Home button is visible and gets the tap.
      expect(find.text('HOME'), findsOneWidget);
      await tester.tap(find.text('HOME'));
      await tester.pumpAndSettle();
      expect(dashTaps, 1);
      expect(taps.value, 0);
    });

    testWidgets('switching between plugins preserves both States',
        (tester) async {
      final aTaps = ValueNotifier<int>(0);
      final bTaps = ValueNotifier<int>(0);
      final svc = RunningPluginsService();
      svc.ensureOpened(_counterSeed('alpha', taps: aTaps));
      svc.ensureOpened(_counterSeed('beta', taps: bTaps));

      // Start with alpha active.
      svc.setActive('alpha');
      await tester.pumpWidget(_harness(
        svc: svc,
        isPluginRoute: true,
        targetPluginId: 'alpha',
      ));
      await tester.pumpAndSettle();
      expect(find.text('alpha#1'), findsOneWidget);
      // Beta is in the IndexedStack but not visible.
      expect(find.text('beta#1'), findsNothing);

      // Swap to beta.
      svc.setActive('beta');
      await tester.pumpWidget(_harness(
        svc: svc,
        isPluginRoute: true,
        targetPluginId: 'beta',
      ));
      await tester.pumpAndSettle();
      expect(find.text('beta#1'), findsOneWidget);
      expect(find.text('alpha#1'), findsNothing);

      // Tap beta — reaches its handler.
      await tester.tap(find.text('beta#1'));
      await tester.pumpAndSettle();
      expect(bTaps.value, 1);
      // Alpha's tap handler was not invoked.
      expect(aTaps.value, 0);

      // Swap back to alpha. Its mount counter is still 1 — no remount.
      svc.setActive('alpha');
      await tester.pumpWidget(_harness(
        svc: svc,
        isPluginRoute: true,
        targetPluginId: 'alpha',
      ));
      await tester.pumpAndSettle();
      expect(find.text('alpha#1'), findsOneWidget,
          reason: 'alpha State was preserved — not remounted');
    });

    testWidgets('plugin route with no entries yet renders an expanded box',
        (tester) async {
      final svc = RunningPluginsService();
      await tester.pumpWidget(_harness(
        svc: svc,
        isPluginRoute: true,
        targetPluginId: 'alpha',
      ));
      await tester.pumpAndSettle();

      // No plugin surface rendered — just the placeholder SizedBox.expand.
      expect(find.byType(IndexedStack), findsNothing);
      expect(find.byType(SizedBox), findsWidgets);
    });
  });
}
