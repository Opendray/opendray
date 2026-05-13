import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/memory/memory_screen.dart';
import 'package:opendray/features/more/more_screen.dart';
import 'package:opendray/features/notes/notes_screen.dart';
import 'package:opendray/features/sessions/sessions_screen.dart';

// Home scaffold: 4 bottom-nav tabs — Sessions / Memory / Notes /
// More. Uses IndexedStack so each tab keeps its own state across
// switches; important for in-progress forms and scroll position.
//
// Activity was removed in PR #54 — it surfaced the raw audit_log
// stream which is >95% routine noise (idle polls, login self-pings,
// successful sends). The operator has no daily "look at events"
// workflow on a phone — moves are taken on Sessions / Memory / More
// instead. If a future use case appears (e.g. an Inbox of events
// that demand operator action), it earns its own tab with explicit
// action surfaces rather than a generic log dump.
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
      MemoryScreen(),
      NotesScreen(),
      MoreScreen(),
    ];
    final tabs = <_TabSpec>[
      _TabSpec(icon: Icons.terminal_outlined, label: t.nav.sessions),
      _TabSpec(icon: Icons.psychology_outlined, label: t.nav.memory),
      _TabSpec(icon: Icons.description_outlined, label: t.nav.notes),
      _TabSpec(icon: Icons.more_horiz, label: t.nav.more),
    ];

    return Scaffold(
      body: IndexedStack(index: _index, children: pages),
      bottomNavigationBar: BottomNavigationBar(
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
