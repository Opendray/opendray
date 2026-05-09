import 'package:flutter/material.dart';

import 'package:opendray/features/sessions/inspector/_inspector_placeholder.dart';

class GitTab extends StatelessWidget {
  const GitTab({required this.sessionId, required this.cwd, super.key});

  final String sessionId;
  final String cwd;

  @override
  Widget build(BuildContext context) {
    return InspectorPlaceholder(
      icon: Icons.account_tree_outlined,
      title: 'Git',
      bullets: [
        'status — modified / staged / untracked at $cwd',
        'log — recent commits, tap to view diff',
        'diff — push relevant hash to terminal as quick-action',
      ],
      apis: const [
        'GET /api/v1/git/status?path=…',
        'GET /api/v1/git/log?path=…',
        'GET /api/v1/git/diff?path=…',
      ],
    );
  }
}
