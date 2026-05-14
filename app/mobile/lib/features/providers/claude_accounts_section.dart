import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/providers/claude_account_dialogs.dart'
    show RenameClaudeAccountDialog;

// Self-contained widget that owns the Claude accounts list (fetch +
// per-row actions). Lives inside the Claude provider's config page
// because accounts are scoped to the Claude provider — the other
// providers (codex / gemini / shell) have no equivalent concept, so
// surfacing a peer "Claude accounts" section under all providers
// reads as a flat-list bug.
class ClaudeAccountsSection extends ConsumerStatefulWidget {
  const ClaudeAccountsSection({super.key});

  @override
  ConsumerState<ClaudeAccountsSection> createState() =>
      _ClaudeAccountsSectionState();
}

class _ClaudeAccountsSectionState
    extends ConsumerState<ClaudeAccountsSection> {
  AsyncValue<List<ClaudeAccountSummary>> _state =
      const AsyncValue.loading();
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    try {
      final list = await ref.read(claudeAccountsApiProvider).list();
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

  Future<void> _openActions(ClaudeAccountSummary a) async {
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
                a.enabled ? Icons.pause_circle_outline : Icons.play_circle_outline,
              ),
              title: Text(a.enabled
                  ? t.providers.accounts.disable
                  : t.providers.accounts.enable),
              onTap: () =>
                  Navigator.of(sheetCtx).pop(_AccountAction.toggleEnabled),
            ),
            ListTile(
              leading: const Icon(Icons.drive_file_rename_outline),
              title: Text(t.providers.accounts.rename),
              onTap: () => Navigator.of(sheetCtx).pop(_AccountAction.rename),
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
              .read(claudeAccountsApiProvider)
              .setEnabled(a.id, enabled: next),
        );
      case _AccountAction.rename:
        final next = await RenameClaudeAccountDialog.show(context, a);
        if (next == null || !mounted) return;
        await _runOp(
          key: 'a:${a.id}',
          okMsg: t.providers.accounts.renamedSnack(name: next),
          failPrefix: t.providers.errorPrefix.rename,
          op: () => ref
              .read(claudeAccountsApiProvider)
              .update(a.id, displayName: next),
        );
      case _AccountAction.delete:
        final ok = await showDialog<bool>(
          context: context,
          builder: (ctx) => AlertDialog(
            title: Text(t.providers.accounts.deleteTitle),
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
                  t.providers.accounts.deleteBody,
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
          op: () => ref.read(claudeAccountsApiProvider).delete(a.id),
        );
    }
  }

  Future<void> _importLocal() async {
    setState(() => _busy.add('import'));
    final messenger = ScaffoldMessenger.of(context);
    try {
      final n = await ref.read(claudeAccountsApiProvider).importLocal();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            n == 0
                ? t.providers.accounts.importSyncedSnack
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
                t.providers.accounts.addHint,
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: Theme.of(context).colorScheme.tertiary,
                ),
              ),
              const SizedBox(height: 4),
              Text(
                t.providers.accounts.addBody,
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
            t.providers.accounts.intro,
            style: muted,
          ),
        ),
      ],
    );
  }

  Widget _buildList(List<ClaudeAccountSummary> list) {
    if (list.isEmpty) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 4),
        child: Text(
          'No Claude accounts yet. Run the shell command above on the '
          'gateway host; the row will appear here on the next refresh.',
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

enum _AccountAction { toggleEnabled, rename, delete }

class _AccountTile extends StatelessWidget {
  const _AccountTile({
    required this.account,
    required this.busy,
    required this.onTap,
  });

  final ClaudeAccountSummary account;
  final bool busy;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final usable = account.isUsable;
    return ListTile(
      onTap: busy ? null : onTap,
      contentPadding: EdgeInsets.zero,
      leading: Icon(
        usable ? Icons.account_circle : Icons.account_circle_outlined,
        color: usable
            ? Theme.of(context).colorScheme.primary
            : Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
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
                    : Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.55),
              ),
            ),
          ),
          const SizedBox(width: 6),
          if (!account.enabled)
            _MiniBadge(
              label: 'disabled',
              color: Theme.of(context).colorScheme.error,
            ),
          if (!account.tokenFilled) ...[
            const SizedBox(width: 4),
            _MiniBadge(
              label: 'no token',
              color: Theme.of(context).colorScheme.error,
            ),
          ],
        ],
      ),
      subtitle: Text(
        account.id,
        style: Theme.of(context)
            .textTheme
            .bodySmall
            ?.copyWith(fontFamily: 'monospace'),
      ),
      trailing: busy
          ? const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : const Icon(Icons.more_vert),
    );
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
