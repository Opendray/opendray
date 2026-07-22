import 'package:flutter/material.dart';

import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/database/database_tab.dart';
import 'package:opendray/features/sessions/inspector/cortex_tab.dart';
import 'package:opendray/features/sessions/inspector/files_tab.dart';
import 'package:opendray/features/sessions/inspector/git_tab.dart';
import 'package:opendray/features/sessions/inspector/history_tab.dart';
import 'package:opendray/features/sessions/inspector/notes_tab.dart';
import 'package:opendray/features/sessions/inspector/tasks_tab.dart';

// Shared plumbing for the session "tool" surfaces used by both the
// bottom dock (SessionToolDock) and the full "More" sheet
// (SessionToolsSheet).
//
// Each cwd-scoped Inspector panel opens as a tall modal bottom sheet
// that floats OVER the live terminal instead of replacing it — the
// terminal stays the hero of the screen and the operator never loses
// their place. The panel widgets themselves (FilesTab, GitTab, …) are
// the exact same ones the full-screen SessionInspectorScreen hosts in
// its TabBarView, so behaviour and state handling stay identical.

/// The cwd-scoped project tools that can be opened over a session.
enum SessionTool { files, git, tasks, history, vault, cortex, database }

IconData sessionToolIcon(SessionTool tool) => switch (tool) {
      SessionTool.files => Icons.folder_outlined,
      SessionTool.git => Icons.account_tree_outlined,
      SessionTool.tasks => Icons.play_circle_outline,
      SessionTool.history => Icons.history,
      SessionTool.vault => Icons.menu_book_outlined,
      SessionTool.cortex => Icons.psychology_outlined,
      SessionTool.database => Icons.storage_outlined,
    };

String sessionToolLabel(SessionTool tool) => switch (tool) {
      SessionTool.files => t.sessions.inspector.shell.tabs.files,
      SessionTool.git => t.sessions.inspector.shell.tabs.git,
      SessionTool.tasks => t.sessions.inspector.shell.tabs.tasks,
      SessionTool.history => t.sessions.inspector.shell.tabs.history,
      SessionTool.vault => t.sessions.inspector.shell.tabs.vault,
      SessionTool.cortex => t.sessions.inspector.shell.tabs.cortex,
      SessionTool.database => t.sessions.inspector.shell.tabs.database,
    };

/// Open one project tool as a bottom sheet over the terminal.
Future<void> openSessionToolSheet(
  BuildContext context,
  SessionSummary session,
  SessionTool tool,
) {
  final child = switch (tool) {
    SessionTool.files =>
      FilesTab(sessionId: session.id, initialPath: session.cwd),
    SessionTool.git => GitTab(sessionId: session.id, cwd: session.cwd),
    SessionTool.tasks => TasksTab(sessionId: session.id, cwd: session.cwd),
    SessionTool.history => HistoryTab(sessionId: session.id),
    SessionTool.vault => NotesTab(sessionId: session.id, cwd: session.cwd),
    SessionTool.cortex => CortexTab(cwd: session.cwd),
    SessionTool.database => DatabaseTab(cwd: session.cwd),
  };
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    backgroundColor: Colors.transparent,
    builder: (_) => ToolSheetScaffold(
      title: sessionToolLabel(tool),
      child: child,
    ),
  );
}

/// Rounded, near-full-height sheet chrome: grabber + title bar + body.
/// Reused by every tool sheet so they all share one look.
class ToolSheetScaffold extends StatelessWidget {
  const ToolSheetScaffold({required this.title, required this.child, super.key});

  final String title;
  final Widget child;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return FractionallySizedBox(
      heightFactor: 0.92,
      child: Container(
        decoration: BoxDecoration(
          color: scheme.surface,
          borderRadius: const BorderRadius.vertical(top: Radius.circular(22)),
          border: Border.all(color: Theme.of(context).dividerColor),
        ),
        clipBehavior: Clip.antiAlias,
        child: Column(
          children: [
            const SheetGrabber(),
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 2, 6, 8),
              child: Row(
                children: [
                  Expanded(
                    child: Text(
                      title,
                      style: Theme.of(context).textTheme.titleMedium,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.close),
                    tooltip: MaterialLocalizations.of(context).closeButtonLabel,
                    onPressed: () => Navigator.of(context).pop(),
                  ),
                ],
              ),
            ),
            Divider(height: 1, color: Theme.of(context).dividerColor),
            Expanded(child: child),
          ],
        ),
      ),
    );
  }
}

/// The little drag handle at the top of every sheet.
class SheetGrabber extends StatelessWidget {
  const SheetGrabber({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      width: 38,
      height: 5,
      margin: const EdgeInsets.only(top: 10, bottom: 8),
      decoration: BoxDecoration(
        color: Theme.of(context).dividerColor,
        borderRadius: BorderRadius.circular(3),
      ),
    );
  }
}
