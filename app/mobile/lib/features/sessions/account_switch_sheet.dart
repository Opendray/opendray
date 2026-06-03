import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// AccountSwitchSheet rebinds a *running* Claude session to a different
// OAuth account — the mobile mirror of the web header AccountSwitcher
// (app/web/src/components/sessions/AccountSwitcher.tsx). The gateway
// terminates the current child process and respawns it under the new
// credential, so the in-CLI conversation context is lost (the session
// id / tab is preserved). A confirm dialog gates the switch.
//
// Returns true via the modal result when a switch succeeded, so the
// caller can refresh the session + accounts views.
class AccountSwitchSheet extends ConsumerStatefulWidget {
  const AccountSwitchSheet({required this.session, super.key});

  final SessionSummary session;

  static Future<bool> show(
    BuildContext context, {
    required SessionSummary session,
  }) async {
    final res = await showModalBottomSheet<bool>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (_) => AccountSwitchSheet(session: session),
    );
    return res ?? false;
  }

  @override
  ConsumerState<AccountSwitchSheet> createState() => _AccountSwitchSheetState();
}

class _AccountSwitchSheetState extends ConsumerState<AccountSwitchSheet> {
  bool _busy = false;

  Translations get _t => Translations.of(context);

  Future<void> _pick(String accountId, String label) async {
    final tr = _t.sessions.detail.accountSwitcher;
    // No-op when picking the already-bound account.
    if (accountId == (widget.session.claudeAccountId ?? '')) {
      Navigator.of(context).pop(false);
      return;
    }
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(tr.confirmTitle),
        content: Text(tr.confirmBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(tr.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(tr.confirmAction),
          ),
        ],
      ),
    );
    if (confirmed != true || !mounted) return;

    setState(() => _busy = true);
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    try {
      await ref
          .read(sessionsApiProvider)
          .switchClaudeAccount(widget.session.id, accountId);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(tr.switchedSnack(account: label)),
          behavior: SnackBarBehavior.floating,
          duration: const Duration(seconds: 2),
        ),
      );
      navigator.pop(true);
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _busy = false);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            tr.switchFailed(
              error: e is ApiException ? e.message : e.toString(),
            ),
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final tr = _t.sessions.detail.accountSwitcher;
    final accountsAsync = ref.watch(claudeAccountsListProvider);
    final theme = Theme.of(context);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.only(bottom: 12),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 0, 20, 8),
              child: Text(tr.sheetTitle, style: theme.textTheme.titleMedium),
            ),
            if (_busy) const LinearProgressIndicator(minHeight: 2),
            accountsAsync.when(
              data: (accounts) {
                final enabled = accounts.where((a) => a.enabled).toList();
                return Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    _AccountRow(
                      selected: widget.session.claudeAccountId == null,
                      title: tr.defaultName,
                      subtitle: tr.defaultSubtitle,
                      enabled: !_busy,
                      onTap: () => _pick('', tr.defaultShort),
                    ),
                    if (enabled.isNotEmpty) const Divider(height: 1),
                    for (final a in enabled)
                      _AccountRow(
                        selected: widget.session.claudeAccountId == a.id,
                        title: a.displayName,
                        subtitle: a.tokenFilled
                            ? (a.oauthEmail ?? a.name)
                            : tr.tokenEmpty,
                        enabled: !_busy && a.tokenFilled,
                        onTap: () => _pick(a.id, a.displayName),
                      ),
                    if (accounts.isEmpty)
                      Padding(
                        padding: const EdgeInsets.fromLTRB(20, 12, 20, 4),
                        child: Text(
                          tr.noneHint,
                          style: theme.textTheme.bodySmall,
                        ),
                      ),
                  ],
                );
              },
              loading: () => const Padding(
                padding: EdgeInsets.all(24),
                child: Center(child: CircularProgressIndicator()),
              ),
              error: (e, _) => Padding(
                padding: const EdgeInsets.fromLTRB(20, 12, 20, 4),
                child: Text(
                  tr.switchFailed(
                    error: e is ApiException ? e.message : e.toString(),
                  ),
                  style: theme.textTheme.bodySmall,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _AccountRow extends StatelessWidget {
  const _AccountRow({
    required this.selected,
    required this.title,
    required this.subtitle,
    required this.enabled,
    required this.onTap,
  });

  final bool selected;
  final String title;
  final String subtitle;
  final bool enabled;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return ListTile(
      enabled: enabled,
      leading: Icon(
        selected ? Icons.check_circle : Icons.circle_outlined,
        color: selected ? scheme.primary : scheme.outline,
      ),
      title: Text(title),
      subtitle: Text(subtitle, maxLines: 1, overflow: TextOverflow.ellipsis),
      onTap: enabled ? onTap : null,
    );
  }
}
