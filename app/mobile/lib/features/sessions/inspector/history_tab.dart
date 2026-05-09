import 'package:flutter/material.dart';

import 'package:opendray/features/sessions/inspector/_inspector_placeholder.dart';

class HistoryTab extends StatelessWidget {
  const HistoryTab({required this.sessionId, super.key});

  final String sessionId;

  @override
  Widget build(BuildContext context) {
    return const InspectorPlaceholder(
      icon: Icons.history,
      title: 'History',
      bullets: [
        'past prompts from this project, pulled from Claude / Codex / Gemini transcripts',
        'tap a prompt → push it into the live terminal as a re-run',
        'search bar to filter long histories',
      ],
      apis: [
        'GET /api/v1/sessions/:id/history?limit=200',
      ],
    );
  }
}
