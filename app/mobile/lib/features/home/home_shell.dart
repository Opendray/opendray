import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/activity/activity_screen.dart';
import 'package:opendray/features/cortex/cortex_hub_screen.dart';
import 'package:opendray/features/integrations/integrations_screen.dart';
import 'package:opendray/features/more/more_screen.dart';
import 'package:opendray/features/sessions/sessions_screen.dart';

// Home scaffold: 5 bottom-nav tabs — Sessions / Cortex / Activity /
// Integrations / More. Uses IndexedStack so each tab keeps its own
// state across switches; important for in-progress forms and scroll
// position.
//
// Cortex is the single front door to the experience flywheel: the
// former Memory / Notes / Knowledge silo tabs are now rung detail
// screens pushed from the CortexHubScreen, mirroring the web's unified
// /cortex page ("one module, three rungs, one loop — no more silo
// tabs"). The two freed slots go to the gateway's highest-signal
// operator views: Activity (per-call integration audit) and
// Integrations (who's calling me) — both promoted out of More.
class HomeShell extends ConsumerStatefulWidget {
  const HomeShell({super.key});

  @override
  ConsumerState<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends ConsumerState<HomeShell> {
  int _index = 0;

  @override
  Widget build(BuildContext context) {
    const pages = [
      SessionsScreen(),
      CortexHubScreen(),
      ActivityScreen(),
      IntegrationsScreen(),
      MoreScreen(),
    ];
    final tabs = <_TabSpec>[
      _TabSpec(icon: Icons.terminal_outlined, label: t.nav.sessions),
      _TabSpec(icon: Icons.psychology_outlined, label: t.nav.cortex),
      _TabSpec(icon: Icons.timeline_outlined, label: t.nav.activity),
      _TabSpec(icon: Icons.api_outlined, label: t.nav.integrations),
      _TabSpec(icon: Icons.more_horiz, label: t.nav.more),
    ];

    return Scaffold(
      body: IndexedStack(index: _index, children: pages),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: _index,
        onTap: (i) => setState(() => _index = i),
        items: [
          for (final tab in tabs)
            BottomNavigationBarItem(icon: Icon(tab.icon), label: tab.label),
        ],
      ),
    );
  }
}

class _TabSpec {
  const _TabSpec({required this.icon, required this.label});
  final IconData icon;
  final String label;
}
