import 'package:flutter/material.dart';

import 'package:opendray/features/sessions/inspector/_inspector_placeholder.dart';

class TasksTab extends StatelessWidget {
  const TasksTab({required this.sessionId, required this.cwd, super.key});

  final String sessionId;
  final String cwd;

  @override
  Widget build(BuildContext context) {
    return InspectorPlaceholder(
      icon: Icons.play_circle_outline,
      title: 'Tasks',
      bullets: [
        'parse package.json scripts / Makefile / Taskfile / justfile in $cwd',
        'tap a task → push the run command into the live terminal',
        'no execution off the running PTY — keeps the session model clean',
      ],
      apis: const [
        'GET /api/v1/fs/read?path=…/package.json',
        'GET /api/v1/fs/read?path=…/Makefile',
        'GET /api/v1/fs/read?path=…/Taskfile.yml',
        'GET /api/v1/fs/read?path=…/justfile',
      ],
    );
  }
}
