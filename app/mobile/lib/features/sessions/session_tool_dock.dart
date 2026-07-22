import 'package:flutter/material.dart';

import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/project/project_screen.dart';
import 'package:opendray/features/sessions/session_tool_sheets.dart';
import 'package:opendray/features/sessions/session_tools_sheet.dart';

// Persistent tool dock shown directly under the live terminal on the
// session detail screen. It promotes the four highest-traffic project
// tools to a single tap — Files / Git / Tasks / Project memory — and
// tucks the rest behind "More" (SessionToolsSheet). This replaces the
// old flow where every one of those destinations was buried in the
// AppBar's "⋮" overflow menu, two taps and a text list away.
//
// Files / Git / Tasks open as bottom sheets that float over the
// terminal (see openSessionToolSheet); Project memory pushes its own
// full workspace; More lifts the complete grid of tools.
class SessionToolDock extends StatelessWidget {
  const SessionToolDock({required this.session, super.key});

  final SessionSummary session;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Material(
      color: scheme.surface,
      child: SafeArea(
        top: false,
        child: Container(
          decoration: BoxDecoration(
            border: Border(
              top: BorderSide(color: Theme.of(context).dividerColor),
            ),
          ),
          padding: const EdgeInsets.symmetric(vertical: 6, horizontal: 4),
          child: Row(
            children: [
              _DockItem(
                icon: sessionToolIcon(SessionTool.files),
                label: sessionToolLabel(SessionTool.files),
                onTap: () =>
                    openSessionToolSheet(context, session, SessionTool.files),
              ),
              _DockItem(
                icon: sessionToolIcon(SessionTool.git),
                label: sessionToolLabel(SessionTool.git),
                onTap: () =>
                    openSessionToolSheet(context, session, SessionTool.git),
              ),
              _DockItem(
                icon: sessionToolIcon(SessionTool.tasks),
                label: sessionToolLabel(SessionTool.tasks),
                onTap: () =>
                    openSessionToolSheet(context, session, SessionTool.tasks),
              ),
              _DockItem(
                icon: Icons.flag_outlined,
                label: t.sessions.detail.projectMemory,
                onTap: () => Navigator.of(context).push(
                  MaterialPageRoute<void>(
                    builder: (_) => ProjectScreen(initialCwd: session.cwd),
                  ),
                ),
              ),
              _DockItem(
                icon: Icons.grid_view_outlined,
                label: t.sessions.dock.more,
                onTap: () => SessionToolsSheet.show(context, session),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _DockItem extends StatelessWidget {
  const _DockItem({
    required this.icon,
    required this.label,
    required this.onTap,
  });

  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Expanded(
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 6),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(icon, size: 22, color: scheme.onSurface),
              const SizedBox(height: 4),
              Text(
                label,
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.labelSmall?.copyWith(
                      color: scheme.onSurface.withValues(alpha: 0.7),
                      fontWeight: FontWeight.w500,
                    ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}
