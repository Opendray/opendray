import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/provider_accounts_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// AccountSwitchSheet rebinds a *running* session to a different account —
// the mobile mirror of the web header AccountSwitcher
// (app/web/src/components/sessions/AccountSwitcher.tsx). It serves every
// provider with switchable accounts — Claude (OAuth account), Antigravity,
// Grok and Codex (per-account home) and OpenCode (credential bundle): the
// gateway terminates the current child process and respawns it under the
// new credential, carrying the conversation across so it resumes with its
// full history. The session id / tab is preserved. A confirm dialog gates
// the switch.
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

  // Provider ids whose live sessions can switch accounts, mapped to the
  // provider's display name (used in the tooltip / sheet title).
  static const providerNames = {
    'claude': 'Claude',
    'antigravity': 'Antigravity',
    'grok': 'Grok',
    'codex': 'Codex',
    'opencode': 'OpenCode',
  };

  static bool supports(SessionSummary s) =>
      providerNames.containsKey(s.providerId) && s.isLive;

  // "Switch <Provider> account" for the session's provider.
  static String title(Translations t, String providerId) =>
      t.sessions.detail.accountSwitcher
          .tooltipFor(provider: providerNames[providerId] ?? providerId);

  // Refreshes the provider's account list after a switch (the bound
  // account's active-session count / last-used change).
  static void invalidateAccounts(WidgetRef ref, String providerId) {
    switch (providerId) {
      case 'claude':
        ref.invalidate(claudeAccountsListProvider);
      case 'antigravity':
        ref.invalidate(antigravityAccountsListProvider);
      default:
        ref.invalidate(providerAccountsListProvider(providerId));
    }
  }

  @override
  ConsumerState<AccountSwitchSheet> createState() => _AccountSwitchSheetState();
}

// A provider-agnostic view of a switchable account, projected from
// ClaudeAccountSummary, AntigravityAccountSummary or ProviderAccountSummary
// so the sheet renders one list regardless of provider.
class _AccountOption {
  const _AccountOption({
    required this.id,
    required this.title,
    required this.subtitle,
    required this.enabled,
    required this.tokenFilled,
  });

  final String id;
  final String title;
  final String subtitle;
  final bool enabled;
  final bool tokenFilled;
}

class _AccountSwitchSheetState extends ConsumerState<AccountSwitchSheet> {
  bool _busy = false;

  Translations get _t => Translations.of(context);

  String get _provider => widget.session.providerId;

  bool get _isAgy => _provider == 'antigravity';

  String get _currentId {
    final s = widget.session;
    return switch (_provider) {
          'antigravity' => s.antigravityAccountId,
          'grok' => s.grokAccountId,
          'codex' => s.codexAccountId,
          'opencode' => s.opencodeAccountId,
          _ => s.claudeAccountId,
        } ??
        '';
  }

  // Claude and Antigravity accounts are managed in the mobile Providers
  // screen; the others only on the web.
  String get _noneHint {
    final tr = _t.sessions.detail.accountSwitcher;
    return switch (_provider) {
      'claude' => tr.noneHint,
      'antigravity' => tr.noneHintAgy,
      _ => tr.noneHintWeb(
          provider: AccountSwitchSheet.providerNames[_provider] ?? _provider,
        ),
    };
  }

  Future<void> _pick(String accountId, String label) async {
    final tr = _t.sessions.detail.accountSwitcher;
    // No-op when picking the already-bound account.
    if (accountId == _currentId) {
      Navigator.of(context).pop(false);
      return;
    }
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(tr.confirmTitle),
        content: Text(_isAgy ? tr.confirmBodyAgy : tr.confirmBody),
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
      final api = ref.read(sessionsApiProvider);
      final id = widget.session.id;
      await switch (_provider) {
        'antigravity' => api.switchAntigravityAccount(id, accountId),
        'grok' => api.switchGrokAccount(id, accountId),
        'codex' => api.switchCodexAccount(id, accountId),
        'opencode' => api.switchOpenCodeAccount(id, accountId),
        _ => api.switchClaudeAccount(id, accountId),
      };
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
    // Project the right provider's accounts into a common option list so
    // the body below is provider-agnostic.
    final optionsAsync = switch (_provider) {
      'claude' => ref.watch(claudeAccountsListProvider).whenData(
            (list) => [
              for (final a in list)
                _AccountOption(
                  id: a.id,
                  title: a.displayName,
                  subtitle:
                      a.tokenFilled ? (a.oauthEmail ?? a.name) : tr.tokenEmpty,
                  enabled: a.enabled,
                  tokenFilled: a.tokenFilled,
                ),
            ],
          ),
      'antigravity' => ref.watch(antigravityAccountsListProvider).whenData(
            (list) => [
              for (final a in list)
                _AccountOption(
                  id: a.id,
                  title: a.displayName,
                  subtitle: a.tokenFilled ? a.name : tr.tokenEmpty,
                  enabled: a.enabled,
                  tokenFilled: a.tokenFilled,
                ),
            ],
          ),
      _ => ref.watch(providerAccountsListProvider(_provider)).whenData(
            (list) => [
              for (final a in list)
                _AccountOption(
                  id: a.id,
                  title: a.displayName,
                  subtitle:
                      a.tokenFilled ? (a.oauthEmail ?? a.name) : tr.tokenEmpty,
                  enabled: a.enabled,
                  tokenFilled: a.tokenFilled,
                ),
            ],
          ),
    };
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
              child: Text(
                AccountSwitchSheet.title(_t, _provider),
                style: theme.textTheme.titleMedium,
              ),
            ),
            if (_busy) const LinearProgressIndicator(minHeight: 2),
            optionsAsync.when(
              data: (options) {
                final enabled = options.where((a) => a.enabled).toList();
                return Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    _AccountRow(
                      selected: _currentId.isEmpty,
                      title: tr.defaultName,
                      subtitle: tr.defaultSubtitle,
                      enabled: !_busy,
                      onTap: () => _pick('', tr.defaultShort),
                    ),
                    if (enabled.isNotEmpty) const Divider(height: 1),
                    for (final a in enabled)
                      _AccountRow(
                        selected: _currentId == a.id,
                        title: a.title,
                        subtitle: a.subtitle,
                        enabled: !_busy && a.tokenFilled,
                        onTap: () => _pick(a.id, a.title),
                      ),
                    if (options.isEmpty)
                      Padding(
                        padding: const EdgeInsets.fromLTRB(20, 12, 20, 4),
                        child: Text(
                          _noneHint,
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
