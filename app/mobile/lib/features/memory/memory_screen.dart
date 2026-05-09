import 'package:flutter/material.dart';

import 'package:opendray/features/_shared/placeholder_screen.dart';

class MemoryScreen extends StatelessWidget {
  const MemoryScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Memory',
      icon: Icons.psychology_outlined,
      body: 'Memory CRUD lands in F4 — list, search, scope filter, '
          'and ambient rules.',
    );
  }
}
