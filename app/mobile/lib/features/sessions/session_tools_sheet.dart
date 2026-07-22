import 'package:flutter/material.dart';

import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/sessions/account_switch_sheet.dart';
import 'package:opendray/features/sessions/session_action_sheet.dart';
import 'package:opendray/features/sessions/session_tool_sheets.dart';

// The full "More" tool sheet, lifted from the dock's More button. It
// is the single home for every project + session tool that isn't hot
// enough to earn a dock slot: all seven Inspector panels (Files, Git,
// Tasks, History, Vault, Cortex, Database) plus the session actions
// (account switch, lifecycle). Searchable, grouped and thumb-sized —
// so nothing lives in an unlabelled overflow menu anymore.
class SessionToolsSheet {
  const SessionToolsSheet._();

  static Future<void> show(BuildContext context, SessionSummary session) {
    return showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: Colors.transparent,
      builder: (_) => _SessionToolsSheetBody(
        session: session,
        hostContext: context,
      ),
    );
  }
}

class _SessionToolsSheetBody extends StatefulWidget {
  const _SessionToolsSheetBody({
    required this.session,
    required this.hostContext,
  });

  final SessionSummary session;

  /// The screen-level context that outlives this sheet — used to open a
  /// follow-up sheet after this one is dismissed.
  final BuildContext hostContext;

  @override
  State<_SessionToolsSheetBody> createState() => _SessionToolsSheetBodyState();
}

class _SessionToolsSheetBodyState extends State<_SessionToolsSheetBody> {
  String _query = '';

  bool get _canAccount {
    final s = widget.session;
    return (s.providerId == 'claude' || s.providerId == 'antigravity') &&
        s.isLive;
  }

  List<SessionTool> get _visibleTools {
    final q = _query.trim().toLowerCase();
    const all = SessionTool.values;
    if (q.isEmpty) return all;
    return all
        .where((tool) => sessionToolLabel(tool).toLowerCase().contains(q))
        .toList();
  }

  void _openTool(SessionTool tool) {
    Navigator.of(context).pop();
    openSessionToolSheet(widget.hostContext, widget.session, tool);
  }

  Future<void> _account() async {
    Navigator.of(context).pop();
    await AccountSwitchSheet.show(widget.hostContext, session: widget.session);
  }

  Future<void> _actions() async {
    Navigator.of(context).pop();
    await SessionActionSheet.show(widget.hostContext, session: widget.session);
  }

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final tools = _visibleTools;
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
              padding: const EdgeInsets.fromLTRB(16, 2, 16, 12),
              child: TextField(
                autofocus: false,
                onChanged: (v) => setState(() => _query = v),
                decoration: InputDecoration(
                  prefixIcon: const Icon(Icons.search, size: 20),
                  hintText: t.sessions.tools.searchHint,
                  isDense: true,
                ),
              ),
            ),
            Expanded(
              child: ListView(
                padding: const EdgeInsets.fromLTRB(16, 0, 16, 24),
                children: [
                  _SectionLabel(
                    t.sessions.tools.inspectSection,
                    trailing: widget.session.displayName,
                  ),
                  const SizedBox(height: 10),
                  GridView.count(
                    crossAxisCount: 2,
                    shrinkWrap: true,
                    physics: const NeverScrollableScrollPhysics(),
                    mainAxisSpacing: 10,
                    crossAxisSpacing: 10,
                    childAspectRatio: 2.6,
                    children: [
                      for (final tool in tools)
                        _ToolTile(
                          icon: sessionToolIcon(tool),
                          label: sessionToolLabel(tool),
                          onTap: () => _openTool(tool),
                        ),
                    ],
                  ),
                  const SizedBox(height: 22),
                  _SectionLabel(
                    t.sessions.tools.sessionSection,
                  ),
                  const SizedBox(height: 10),
                  Row(
                    children: [
                      if (_canAccount) ...[
                        Expanded(
                          child: _ActionButton(
                            icon: Icons.manage_accounts_outlined,
                            label: t.sessions.detail.accountSwitcher.tooltip,
                            onTap: _account,
                          ),
                        ),
                        const SizedBox(width: 10),
                      ],
                      Expanded(
                        child: _ActionButton(
                          icon: Icons.tune,
                          label: t.sessions.detail.actions,
                          onTap: _actions,
                        ),
                      ),
                    ],
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _SectionLabel extends StatelessWidget {
  const _SectionLabel(this.text, {this.trailing});
  final String text;
  final String? trailing;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Row(
      children: [
        Expanded(
          child: Text(
            text,
            style: theme.textTheme.labelSmall?.copyWith(
              letterSpacing: 0.9,
              fontWeight: FontWeight.w600,
              color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
            ),
          ),
        ),
        if (trailing != null)
          Text(
            trailing!,
            style: theme.textTheme.bodySmall?.copyWith(
              fontFamily: 'monospace',
              color: theme.colorScheme.onSurface.withValues(alpha: 0.5),
            ),
          ),
      ],
    );
  }
}

class _ToolTile extends StatelessWidget {
  const _ToolTile({
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
    return Material(
      color: scheme.surfaceContainerHighest.withValues(alpha: 0.35),
      shape: RoundedRectangleBorder(
        side: BorderSide(color: Theme.of(context).dividerColor),
        borderRadius: BorderRadius.circular(12),
      ),
      clipBehavior: Clip.antiAlias,
      child: InkWell(
        onTap: onTap,
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12),
          child: Row(
            children: [
              Icon(icon, size: 21, color: scheme.primary),
              const SizedBox(width: 11),
              Expanded(
                child: Text(
                  label,
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: Theme.of(context)
                      .textTheme
                      .bodyMedium
                      ?.copyWith(fontWeight: FontWeight.w600),
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _ActionButton extends StatelessWidget {
  const _ActionButton({
    required this.icon,
    required this.label,
    required this.onTap,
  });
  final IconData icon;
  final String label;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return OutlinedButton.icon(
      onPressed: onTap,
      icon: Icon(icon, size: 18),
      label: Text(label, overflow: TextOverflow.ellipsis),
      style: OutlinedButton.styleFrom(
        padding: const EdgeInsets.symmetric(vertical: 12),
        side: BorderSide(color: Theme.of(context).dividerColor),
      ),
    );
  }
}
