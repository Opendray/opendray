import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/antigravity_accounts_api.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Self-contained widget that owns the Antigravity accounts list (fetch +
// per-row actions). Lives inside the Antigravity provider's config page
// because accounts are scoped to that provider. Mirrors the web
// AntigravityAccountsPanel — and the mobile ClaudeAccountsSection — but
// is import-only: agy accounts are per-account HOME dirs surfaced by
// import-local, so there is no token-paste, rename, or identity-drift
// flow. Per-row actions are just enable/disable + remove.
class AntigravityAccountsSection extends ConsumerStatefulWidget {
  const AntigravityAccountsSection({super.key});

  @override
  ConsumerState<AntigravityAccountsSection> createState() =>
      _AntigravityAccountsSectionState();
}

class _AntigravityAccountsSectionState
    extends ConsumerState<AntigravityAccountsSection> {
  AsyncValue<List<AntigravityAccountSummary>> _state =
      const AsyncValue.loading();
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final list = await ref.read(antigravityAccountsApiProvider).list();
      if (!mounted) return;
      list.sort((a, b) {
        if (a.isUsable != b.isUsable) return a.isUsable ? -1 : 1;
        return a.displayName
            .toLowerCase()
            .compareTo(b.displayName.toLowerCase());
      });
      setState(() => _state = AsyncValue.data(list));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _runOp({
    required String key,
    required String okMsg,
    required String failPrefix,
    required Future<void> Function() op,
  }) async {
    setState(() => _busy.add(key));
    final messenger = ScaffoldMessenger.of(context);
    try {
      await op();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(okMsg),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.providers.errorWithMessage(prefix: failPrefix, error: e.message),
          ),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.providers
                .errorWithMessage(prefix: failPrefix, error: e.toString()),
          ),
        ),
      );
    } finally {
      if (mounted) setState(() => _busy.remove(key));
    }
  }

  Future<void> _openActions(AntigravityAccountSummary a) async {
    final action = await showModalBottomSheet<_AccountAction>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(a.displayName,
                      style: Theme.of(sheetCtx).textTheme.titleSmall),
                  Text(
                    a.name,
                    style: Theme.of(sheetCtx)
                        .textTheme
                        .bodySmall
                        ?.copyWith(fontFamily: 'monospace'),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: Icon(
                a.enabled
                    ? Icons.pause_circle_outline
                    : Icons.play_circle_outline,
              ),
              title: Text(a.enabled
                  ? t.providers.accounts.disable
                  : t.providers.accounts.enable),
              onTap: () =>
                  Navigator.of(sheetCtx).pop(_AccountAction.toggleEnabled),
            ),
            const Divider(height: 1),
            ListTile(
              leading: Icon(
                Icons.delete_outline,
                color: Theme.of(sheetCtx).colorScheme.error,
              ),
              title: Text(
                t.providers.accounts.deleteLabel,
                style: TextStyle(color: Theme.of(sheetCtx).colorScheme.error),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_AccountAction.delete),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _AccountAction.toggleEnabled:
        final next = !a.enabled;
        await _runOp(
          key: 'a:${a.id}',
          okMsg: next
              ? t.providers.accounts.enabledSnack(name: a.displayName)
              : t.providers.accounts.disabledSnack(name: a.displayName),
          failPrefix: t.providers.errorPrefix.toggle,
          op: () => ref
              .read(antigravityAccountsApiProvider)
              .setEnabled(a.id, enabled: next),
        );
      case _AccountAction.delete:
        final ok = await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: Text(t.providers.antigravityAccounts.deleteTitle),
            content: Column(
              mainAxisSize: MainAxisSize.min,
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  '${a.displayName} (${a.name})',
                  style: Theme.of(ctx).textTheme.bodyMedium,
                ),
                const SizedBox(height: 8),
                Text(
                  t.providers.antigravityAccounts.deleteBody,
                  style: Theme.of(ctx).textTheme.bodySmall,
                ),
              ],
            ),
            actions: [
              TextButton(
                onPressed: () => Navigator.of(ctx).pop(false),
                child: Text(t.common.cancel),
              ),
              FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: Theme.of(ctx).colorScheme.error,
                ),
                onPressed: () => Navigator.of(ctx).pop(true),
                child: Text(t.common.delete),
              ),
            ],
          ),
        );
        if (ok != true || !mounted) return;
        await _runOp(
          key: 'a:${a.id}',
          okMsg: t.providers.accounts.deletedSnack(name: a.name),
          failPrefix: t.providers.errorPrefix.delete,
          op: () => ref.read(antigravityAccountsApiProvider).delete(a.id),
        );
    }
  }

  Future<void> _importLocal() async {
    setState(() => _busy.add('import'));
    final messenger = ScaffoldMessenger.of(context);
    try {
      final n = await ref.read(antigravityAccountsApiProvider).importLocal();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            n == 0
                ? t.providers.antigravityAccounts.importSyncedSnack
                : (n == 1
                    ? t.providers.accounts.importedSnackOne(n: n.toString())
                    : t.providers.accounts.importedSnackOther(n: n.toString())),
          ),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.providers.accounts.importFailedApi(error: e.message),
          ),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.providers.accounts.importFailedGeneric(error: e.toString()),
          ),
        ),
      );
    } finally {
      if (mounted) setState(() => _busy.remove('import'));
    }
  }

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final importing = _busy.contains('import');
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(4, 14, 0, 6),
          child: Row(
            children: [
              Expanded(
                child: Text(
                  'ACCOUNTS',
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                        letterSpacing: 0.8,
                        color: Theme.of(context)
                            .colorScheme
                            .onSurface
                            .withValues(alpha: 0.6),
                      ),
                ),
              ),
              TextButton.icon(
                icon: importing
                    ? const SizedBox(
                        width: 14,
                        height: 14,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Icon(Icons.cloud_sync_outlined, size: 16),
                label: Text(importing
                    ? t.providers.accounts.importing
                    : t.providers.accounts.importLocal),
                onPressed: importing ? null : _importLocal,
              ),
            ],
          ),
        ),
        Container(
          padding: const EdgeInsets.all(10),
          margin: const EdgeInsets.only(bottom: 6),
          decoration: BoxDecoration(
            color: Theme.of(context)
                .colorScheme
                .tertiary
                .withValues(alpha: 0.08),
            borderRadius: BorderRadius.circular(6),
          ),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                t.providers.antigravityAccounts.addHint,
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: Theme.of(context).colorScheme.tertiary,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                t.providers.antigravityAccounts.addBody,
                style: TextStyle(
                  fontSize: 11,
                  height: 1.4,
                  color: Theme.of(context)
                      .colorScheme
                      .onSurface
                      .withValues(alpha: 0.75),
                ),
              ),
            ],
          ),
        ),
        _state.when(
          data: _buildList,
          loading: () => const Padding(
            padding: EdgeInsets.symmetric(vertical: 16),
            child: Center(
              child: SizedBox(
                width: 18,
                height: 18,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          ),
          error: (e, _) => Padding(
            padding: const EdgeInsets.symmetric(vertical: 8),
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  t.providers.accounts.loadFailed(error: e.toString()),
                  style: TextStyle(
                    color: Theme.of(context).colorScheme.error,
                    fontSize: 12,
                  ),
                ),
                const SizedBox(height: 6),
                OutlinedButton(
                  onPressed: _load,
                  child: Text(t.common.retry),
                ),
              ],
            ),
          ),
        ),
        Padding(
          padding: const EdgeInsets.only(top: 6, left: 4),
          child: Text(
            t.providers.antigravityAccounts.intro,
            style: muted,
          ),
        ),
      ],
    );
  }

  Widget _buildList(List<AntigravityAccountSummary> list) {
    if (list.isEmpty) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 4),
        child: Text(
          t.providers.antigravityAccounts.empty,
          style: Theme.of(context).textTheme.bodySmall,
        ),
      );
    }
    return Column(
      children: [
        for (final a in list)
          _AccountTile(
            account: a,
            busy: _busy.contains('a:${a.id}'),
            onTap: () => _openActions(a),
          ),
      ],
    );
  }
}

enum _AccountAction { toggleEnabled, delete }

class _AccountTile extends StatelessWidget {
  const _AccountTile({
    required this.account,
    required this.busy,
    required this.onTap,
  });

  final AntigravityAccountSummary account;
  final bool busy;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final usable = account.isUsable;
    final muted = theme.colorScheme.onSurface.withValues(alpha: 0.6);
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        ListTile(
          onTap: busy ? null : onTap,
          contentPadding: EdgeInsets.zero,
          leading: Icon(
            usable ? Icons.vpn_key : Icons.vpn_key_outlined,
            color: usable
                ? theme.colorScheme.primary
                : theme.colorScheme.onSurface.withValues(alpha: 0.4),
          ),
          title: Row(
            children: [
              Flexible(
                child: Text(
                  account.displayName,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(
                    color: account.enabled
                        ? null
                        : theme.colorScheme.onSurface.withValues(alpha: 0.55),
                  ),
                ),
              ),
              const SizedBox(width: 6),
              if (!account.enabled)
                _MiniBadge(label: 'disabled', color: theme.colorScheme.error),
              if (!account.tokenFilled) ...[
                const SizedBox(width: 4),
                _MiniBadge(
                  label: t.providers.antigravityAccounts.noTokenYet,
                  color: theme.colorScheme.error,
                ),
              ],
            ],
          ),
          subtitle: Text(
            account.id,
            style:
                theme.textTheme.bodySmall?.copyWith(fontFamily: 'monospace'),
          ),
          trailing: busy
              ? const SizedBox(
                  width: 18,
                  height: 18,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.more_vert),
        ),
        // Metadata chips live below the ListTile (not in its subtitle
        // slot, which clips to a fixed height) so the HOME path + chips
        // never get cut off on a phone row.
        if (_hasMeta)
          Padding(
            padding: const EdgeInsets.only(left: 40, bottom: 8),
            child: _metaChips(context, muted),
          ),
      ],
    );
  }

  bool get _hasMeta =>
      account.configDir.isNotEmpty ||
      account.activeSessions != null ||
      account.lastUsedAt != null;

  // Capacity-at-a-glance metadata, wrapped so it never overflows a phone
  // row. Mirrors the inline chips in web's AntigravityAccountsPanel.
  Widget _metaChips(BuildContext context, Color muted) {
    final theme = Theme.of(context);
    final used = _relTime(account.lastUsedAt);
    return Wrap(
      spacing: 6,
      runSpacing: 4,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: [
        if (account.activeSessions != null)
          _MiniBadge(
            label: t.providers.accounts
                .activeSessions(count: account.activeSessions!.toString()),
            color: muted,
          ),
        if (used != null)
          Text(
            t.providers.accounts.usedAgo(when: used),
            style: theme.textTheme.bodySmall?.copyWith(color: muted),
          ),
        if (account.configDir.isNotEmpty)
          Text(
            t.providers.antigravityAccounts.homeDir(dir: account.configDir),
            style: theme.textTheme.bodySmall
                ?.copyWith(color: muted, fontFamily: 'monospace'),
          ),
      ],
    );
  }

  // Coarse relative time for last_used_at; null when absent/unparseable.
  static String? _relTime(String? iso) {
    if (iso == null) return null;
    final t = DateTime.tryParse(iso);
    if (t == null) return null;
    final d = DateTime.now().difference(t);
    if (d.inDays > 365) return '${(d.inDays / 365).floor()}y ago';
    if (d.inDays > 30) return '${(d.inDays / 30).floor()}mo ago';
    if (d.inDays > 0) return '${d.inDays}d ago';
    if (d.inHours > 0) return '${d.inHours}h ago';
    if (d.inMinutes > 0) return '${d.inMinutes}m ago';
    return 'just now';
  }
}

class _MiniBadge extends StatelessWidget {
  const _MiniBadge({required this.label, required this.color});
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withValues(alpha: 0.6), width: 0.5),
      ),
      child: Text(
        label,
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
  }
}
