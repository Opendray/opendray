import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/version_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/cortex/cortex_hub_screen.dart';
import 'package:opendray/features/integrations/integrations_screen.dart';
import 'package:opendray/features/more/more_screen.dart';
import 'package:opendray/features/roundtable/round_table_list_screen.dart';
import 'package:opendray/features/sessions/sessions_screen.dart';

// Home scaffold: 5 bottom-nav tabs — Sessions / Cortex / Round Table /
// Integrations / More. Uses IndexedStack so each tab keeps its own
// state across switches; important for in-progress forms and scroll
// position.
//
// Cortex is the single front door to the experience flywheel: the
// former Memory / Notes / Knowledge silo tabs are now rung detail
// screens pushed from the CortexHubScreen, mirroring the web's unified
// /cortex page ("one module, three rungs, one loop — no more silo
// tabs"). Round Table (cross-vendor AI group chat) takes a top-level
// slot for quick access; Activity (per-call integration audit) moves
// down into More. Integrations (who's calling me) keeps its slot.
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
      RoundTableListScreen(),
      IntegrationsScreen(),
      MoreScreen(),
    ];
    final tabs = <_TabSpec>[
      _TabSpec(icon: Icons.terminal_outlined, label: t.nav.sessions),
      _TabSpec(icon: Icons.psychology_outlined, label: t.nav.cortex),
      _TabSpec(icon: Icons.groups_outlined, label: t.nav.roundTable),
      _TabSpec(icon: Icons.api_outlined, label: t.nav.integrations),
      _TabSpec(icon: Icons.more_horiz, label: t.nav.more),
    ];
    // Badge the More tab (which leads to Settings) when the gateway has an
    // update waiting — the mobile mirror of the web's Settings-icon badge.
    final updateAvailable =
        ref.watch(versionInfoProvider).asData?.value.updateAvailable ?? false;

    return Scaffold(
      body: IndexedStack(index: _index, children: pages),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed,
        currentIndex: _index,
        onTap: (i) => setState(() => _index = i),
        items: [
          for (var i = 0; i < tabs.length; i++)
            BottomNavigationBarItem(
              icon: (i == tabs.length - 1 && updateAvailable)
                  ? Badge(smallSize: 8, child: Icon(tabs[i].icon))
                  : Icon(tabs[i].icon),
              label: tabs[i].label,
            ),
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
