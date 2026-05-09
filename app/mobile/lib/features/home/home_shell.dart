import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/features/activity/activity_screen.dart';
import 'package:opendray/features/memory/memory_screen.dart';
import 'package:opendray/features/more/more_screen.dart';
import 'package:opendray/features/notes/notes_screen.dart';
import 'package:opendray/features/sessions/sessions_screen.dart';

// Home scaffold: 5 bottom-nav tabs matching the web admin's
// information architecture (Sessions / Memory / Notes / Activity /
// More). Uses IndexedStack so each tab keeps its own state across
// switches — important for in-progress forms and scroll position.
class HomeShell extends ConsumerStatefulWidget {
  const HomeShell({super.key});

  @override
  ConsumerState<HomeShell> createState() => _HomeShellState();
}

class _HomeShellState extends ConsumerState<HomeShell> {
  int _index = 0;

  static const _tabs = <_TabSpec>[
    _TabSpec(icon: Icons.terminal_outlined, label: 'Sessions'),
    _TabSpec(icon: Icons.psychology_outlined, label: 'Memory'),
    _TabSpec(icon: Icons.description_outlined, label: 'Notes'),
    _TabSpec(icon: Icons.timeline_outlined, label: 'Activity'),
    _TabSpec(icon: Icons.more_horiz, label: 'More'),
  ];

  @override
  Widget build(BuildContext context) {
    const pages = [
      SessionsScreen(),
      MemoryScreen(),
      NotesScreen(),
      ActivityScreen(),
      MoreScreen(),
    ];

    return Scaffold(
      body: IndexedStack(index: _index, children: pages),
      bottomNavigationBar: BottomNavigationBar(
        currentIndex: _index,
        onTap: (i) => setState(() => _index = i),
        items: [
          for (final t in _tabs)
            BottomNavigationBarItem(icon: Icon(t.icon), label: t.label),
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
