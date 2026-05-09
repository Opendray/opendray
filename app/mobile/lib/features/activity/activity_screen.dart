import 'package:flutter/material.dart';

import 'package:opendray/features/_shared/placeholder_screen.dart';

class ActivityScreen extends StatelessWidget {
  const ActivityScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Activity',
      icon: Icons.timeline_outlined,
      body: 'Audit feed lands in F6 — actor / subject / action filters '
          'with time-range navigation.',
    );
  }
}
