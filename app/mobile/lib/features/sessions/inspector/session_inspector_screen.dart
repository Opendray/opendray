import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/sessions/inspector/files_tab.dart';
import 'package:opendray/features/sessions/inspector/git_tab.dart';
import 'package:opendray/features/sessions/inspector/history_tab.dart';
import 'package:opendray/features/sessions/inspector/notes_tab.dart';
import 'package:opendray/features/sessions/inspector/tasks_tab.dart';
import 'package:path/path.dart' as p;

// Per-session inspector screen. Mirrors the cwd-scoped panels the
// web admin shows beside a session (Files / Git / Tasks / History
// / Notes), but as a full-screen route with a TabBar — phones
// don't have the lateral space for a side-panel layout. AppBar
// shows the cwd's last path segment so the user always knows
// what they're operating on.
class SessionInspectorScreen extends ConsumerWidget {
  const SessionInspectorScreen({required this.sessionId, super.key});

  final String sessionId;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(sessionByIdProvider(sessionId));
    return async.when(
      data: (session) => _Body(session: session),
      loading: () => const _LoadingScaffold(),
      error: (e, _) => _ErrorScaffold(error: e),
    );
  }
}

class _Body extends StatelessWidget {
  const _Body({required this.session});
  final SessionSummary session;

  @override
  Widget build(BuildContext context) {
    final lastSegment = p.basename(session.cwd);
    return DefaultTabController(
      length: 5,
      child: Scaffold(
        appBar: AppBar(
          title: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                t.sessions.inspector.shell.title,
                style: Theme.of(context).textTheme.titleSmall,
              ),
              Text(
                lastSegment.isEmpty ? session.cwd : lastSegment,
                style: Theme.of(context).textTheme.bodySmall,
                overflow: TextOverflow.ellipsis,
              ),
            ],
          ),
          bottom: TabBar(
            isScrollable: true,
            tabAlignment: TabAlignment.start,
            tabs: [
              Tab(
                icon: const Icon(Icons.folder_outlined),
                text: t.sessions.inspector.shell.tabs.files,
              ),
              Tab(
                icon: const Icon(Icons.account_tree_outlined),
                text: t.sessions.inspector.shell.tabs.git,
              ),
              Tab(
                icon: const Icon(Icons.play_circle_outline),
                text: t.sessions.inspector.shell.tabs.tasks,
              ),
              Tab(
                icon: const Icon(Icons.history),
                text: t.sessions.inspector.shell.tabs.history,
              ),
              Tab(
                icon: const Icon(Icons.description_outlined),
                text: t.sessions.inspector.shell.tabs.notes,
              ),
            ],
          ),
        ),
        body: TabBarView(
          children: [
            FilesTab(sessionId: session.id, initialPath: session.cwd),
            GitTab(sessionId: session.id, cwd: session.cwd),
            TasksTab(sessionId: session.id, cwd: session.cwd),
            HistoryTab(sessionId: session.id),
            NotesTab(sessionId: session.id, cwd: session.cwd),
          ],
        ),
      ),
    );
  }
}

class _LoadingScaffold extends StatelessWidget {
  const _LoadingScaffold();

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(t.sessions.inspector.shell.title)),
      body: const Center(child: CircularProgressIndicator()),
    );
  }
}

class _ErrorScaffold extends StatelessWidget {
  const _ErrorScaffold({required this.error});
  final Object error;

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(t.sessions.inspector.shell.title)),
      body: Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            t.sessions.inspector.shell.loadError(error: error.toString()),
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodySmall,
          ),
        ),
      ),
    );
  }
}
