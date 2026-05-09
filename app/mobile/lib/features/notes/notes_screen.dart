import 'package:flutter/material.dart';

import 'package:opendray/features/_shared/placeholder_screen.dart';

class NotesScreen extends StatelessWidget {
  const NotesScreen({super.key});

  @override
  Widget build(BuildContext context) {
    return const PlaceholderScreen(
      title: 'Notes',
      icon: Icons.description_outlined,
      body: 'Notes — markdown view, edit, git push/pull — lands in F5.',
    );
  }
}
