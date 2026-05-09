import 'package:flutter/material.dart';

import 'package:opendray/features/sessions/inspector/_inspector_placeholder.dart';

class NotesTab extends StatelessWidget {
  const NotesTab({required this.sessionId, required this.cwd, super.key});

  final String sessionId;
  final String cwd;

  @override
  Widget build(BuildContext context) {
    return InspectorPlaceholder(
      icon: Icons.description_outlined,
      title: 'Notes',
      bullets: [
        'project-mapped notes for $cwd from the operator vault',
        'tap a note → @reference into the live terminal',
        'edit / append from the inspector — same vault as web admin',
      ],
      apis: const [
        'GET /api/v1/notes/project-mapping?path=…',
        'GET /api/v1/notes/list',
        'GET /api/v1/notes/read?path=…',
      ],
    );
  }
}
