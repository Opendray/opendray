import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/githosts_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/githosts/githost_form_screen.dart';

// Git hosts list screen — registered host credentials the gateway
// uses to talk to GitHub / GitLab / Bitbucket / self-hosted forges.
// Sessions consume these via /git/prs and /git/remote endpoints.
//
// Tokens are write-only over the API; the list response only carries
// a server-redacted preview ("ghp_…abcd"). Editing means re-typing
// the token if you want to rotate it; leaving the token field blank
// on edit keeps the existing one.
class GitHostsScreen extends ConsumerStatefulWidget {
  const GitHostsScreen({super.key});

  @override
  ConsumerState<GitHostsScreen> createState() => _GitHostsScreenState();
}

class _GitHostsScreenState extends ConsumerState<GitHostsScreen> {
  AsyncValue<List<GitHost>> _state = const AsyncValue.loading();
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(gitHostsApiProvider).list();
      if (!mounted) return;
      list.sort((a, b) {
        if (a.enabled != b.enabled) return a.enabled ? -1 : 1;
        return a.name.toLowerCase().compareTo(b.name.toLowerCase());
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
            t.githosts.errorWithMessage(prefix: failPrefix, error: e.message),
          ),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.githosts.errorWithMessage(prefix: failPrefix, error: e.toString()),
          ),
        ),
      );
    } finally {
      if (mounted) setState(() => _busy.remove(key));
    }
  }

  Future<void> _onCreate() async {
    final res = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (_) => const GitHostFormScreen(),
        fullscreenDialog: true,
      ),
    );
    if ((res ?? false) && mounted) await _load();
  }

  Future<void> _onEdit(GitHost h) async {
    final res = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (_) => GitHostFormScreen(existing: h),
      ),
    );
    if ((res ?? false) && mounted) await _load();
  }

  Future<void> _onDelete(GitHost h) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.githosts.deleteTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(h.name, style: Theme.of(ctx).textTheme.titleSmall),
            const SizedBox(height: 4),
            Text(
              h.host,
              style: Theme.of(ctx)
                  .textTheme
                  .bodySmall
                  ?.copyWith(fontFamily: 'monospace'),
            ),
            const SizedBox(height: 12),
            Text(
              t.githosts.deleteBody(host: h.host),
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
      key: h.id,
      okMsg: t.githosts.deletedSnack(name: h.name),
      failPrefix: t.githosts.errorPrefix.delete,
      op: () => ref.read(gitHostsApiProvider).delete(h.id),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.githosts.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.common.refresh,
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildList,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
      floatingActionButton: FloatingActionButton.extended(
        heroTag: 'githosts_fab',
        onPressed: _onCreate,
        icon: const Icon(Icons.add),
        label: Text(t.githosts.addHost),
      ),
    );
  }

  Widget _buildList(List<GitHost> list) {
    if (list.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            t.githosts.emptyList,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        itemCount: list.length,
        separatorBuilder: (_, __) => Divider(
          height: 1,
          color: Theme.of(context).dividerColor,
        ),
        itemBuilder: (_, i) {
          final h = list[i];
          return _HostTile(
            host: h,
            busy: _busy.contains(h.id),
            onTap: () => _onEdit(h),
            onToggle: (next) => _runOp(
              key: h.id,
              okMsg: next
                  ? t.githosts.enabledSnack(name: h.name)
                  : t.githosts.disabledSnack(name: h.name),
              failPrefix: t.githosts.errorPrefix.toggle,
              op: () => ref
                  .read(gitHostsApiProvider)
                  .update(h.id, enabled: next)
                  .then((_) {}),
            ),
            onDelete: () => _onDelete(h),
          );
        },
      ),
    );
  }
}

class _HostTile extends StatelessWidget {
  const _HostTile({
    required this.host,
    required this.busy,
    required this.onTap,
    required this.onToggle,
    required this.onDelete,
  });
  final GitHost host;
  final bool busy;
  final VoidCallback onTap;
  final ValueChanged<bool> onToggle;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return ListTile(
      onTap: busy ? null : onTap,
      leading: Container(
        width: 36,
        height: 36,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: Theme.of(context)
              .colorScheme
              .primary
              .withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          host.kind.isNotEmpty ? host.kind[0].toUpperCase() : '?',
          style: TextStyle(
            color: Theme.of(context).colorScheme.primary,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      title: Row(
        children: [
          Flexible(
            child: Text(
              host.name,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(width: 6),
          _Badge(
            label: host.kind,
            color: Theme.of(context).colorScheme.tertiary,
          ),
        ],
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Wrap(
          spacing: 6,
          runSpacing: 2,
          children: [
            Text(
              host.host,
              style: const TextStyle(fontFamily: 'monospace'),
            ),
            Text('· token ${host.tokenMask.isEmpty ? '—' : host.tokenMask}'),
            Text(
              '· updated ${DateFormat.yMMMd().format(host.updatedAt.toLocal())}',
            ),
          ],
        ),
      ),
      trailing: busy
          ? const SizedBox(
              width: 18,
              height: 18,
              child: CircularProgressIndicator(strokeWidth: 2),
            )
          : Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Switch(
                  value: host.enabled,
                  onChanged: onToggle,
                ),
                IconButton(
                  icon: Icon(
                    Icons.delete_outline,
                    color: Theme.of(context).colorScheme.error,
                  ),
                  tooltip: t.common.delete,
                  onPressed: onDelete,
                ),
              ],
            ),
    );
  }
}

class _Badge extends StatelessWidget {
  const _Badge({required this.label, required this.color});
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

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final String error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              t.githosts.failedToLoad,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
