import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/channels_api.dart';

// Notification destinations (Slack / Feishu / DingTalk / WeCom /
// bridge). Read-only list. Per-row actions: test-send, toggle
// enabled, toggle muted, view raw config. Create/edit/delete are
// scoped out — kind-specific config schemas (workspace IDs, app
// secrets, group tokens) would need a different form per kind, and
// none of them are operator-tweakable on mobile in practice.
class ChannelsScreen extends ConsumerStatefulWidget {
  const ChannelsScreen({super.key});

  @override
  ConsumerState<ChannelsScreen> createState() => _ChannelsScreenState();
}

class _ChannelsScreenState extends ConsumerState<ChannelsScreen> {
  AsyncValue<List<ChannelView>> _state = const AsyncValue.loading();
  // Track per-row in-flight ops so the UI can disable just that row's
  // action sheet entries while a PATCH is in flight.
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(channelsApiProvider).list();
      if (!mounted) return;
      list.sort((a, b) {
        // Running first so the channels actively delivering notifications
        // float up. Then by kind (groups Slack/Feishu/etc together).
        if (a.running != b.running) return a.running ? -1 : 1;
        return a.kind.compareTo(b.kind);
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

  Future<void> _runAction({
    required String id,
    required String okMessage,
    required String failPrefix,
    required Future<void> Function() op,
  }) async {
    setState(() => _busy.add(id));
    final messenger = ScaffoldMessenger.of(context);
    try {
      await op();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(okMessage),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('$failPrefix: ${e.message}')),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('$failPrefix: $e')),
      );
    } finally {
      if (mounted) setState(() => _busy.remove(id));
    }
  }

  Future<void> _onTap(ChannelView ch) async {
    final isBusy = _busy.contains(ch.id);
    final action = await showModalBottomSheet<_RowAction>(
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
                  Text(
                    ch.kind,
                    style: Theme.of(sheetCtx).textTheme.titleSmall,
                  ),
                  Text(
                    ch.id,
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
              enabled: !isBusy,
              leading: const Icon(Icons.send_outlined),
              title: const Text('Send test message'),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.test),
            ),
            ListTile(
              enabled: !isBusy,
              leading: Icon(
                ch.enabled ? Icons.pause_circle_outline : Icons.play_circle_outline,
              ),
              title: Text(ch.enabled ? 'Disable' : 'Enable'),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.toggleEnabled),
            ),
            ListTile(
              enabled: !isBusy,
              leading: Icon(
                ch.muted ? Icons.notifications_active_outlined : Icons.notifications_off_outlined,
              ),
              title: Text(ch.muted ? 'Unmute' : 'Mute'),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.toggleMuted),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.code),
              title: const Text('View raw config'),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.viewConfig),
            ),
            ListTile(
              leading: const Icon(Icons.copy_outlined),
              title: const Text('Copy channel id'),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.copyId),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _RowAction.test:
        await _runAction(
          id: ch.id,
          okMessage: 'Test message dispatched.',
          failPrefix: 'Test failed',
          op: () => ref.read(channelsApiProvider).test(ch.id),
        );
      case _RowAction.toggleEnabled:
        final next = !ch.enabled;
        await _runAction(
          id: ch.id,
          okMessage: next ? 'Channel enabled.' : 'Channel disabled.',
          failPrefix: 'Toggle failed',
          op: () => ref
              .read(channelsApiProvider)
              .setEnabled(ch.id, enabled: next)
              .then((_) {}),
        );
      case _RowAction.toggleMuted:
        final next = !ch.muted;
        await _runAction(
          id: ch.id,
          okMessage: next ? 'Channel muted.' : 'Channel unmuted.',
          failPrefix: 'Mute toggle failed',
          op: () => ref
              .read(channelsApiProvider)
              .setMuted(ch.id, muted: next)
              .then((_) {}),
        );
      case _RowAction.viewConfig:
        await _showConfig(ch);
      case _RowAction.copyId:
        await Clipboard.setData(ClipboardData(text: ch.id));
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text('Copied ${ch.id}'),
            duration: const Duration(seconds: 2),
            behavior: SnackBarBehavior.floating,
          ),
        );
    }
  }

  Future<void> _showConfig(ChannelView ch) async {
    const enc = JsonEncoder.withIndent('  ');
    final pretty = enc.convert(ch.config);
    await showDialog<void>(
      context: context,
      builder: (dialogCtx) => AlertDialog(
        title: Text('${ch.kind} config'),
        content: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.of(dialogCtx).size.height * 0.6,
            maxWidth: 480,
          ),
          child: SingleChildScrollView(
            child: SelectableText(
              pretty,
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 11,
                height: 1.4,
              ),
            ),
          ),
        ),
        actions: [
          TextButton(
            onPressed: () async {
              await Clipboard.setData(ClipboardData(text: pretty));
              if (!dialogCtx.mounted) return;
              Navigator.of(dialogCtx).pop();
            },
            child: const Text('Copy'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogCtx).pop(),
            child: const Text('Close'),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Channels'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildList,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
    );
  }

  Widget _buildList(List<ChannelView> list) {
    if (list.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'No channels configured yet.\n\n'
            'Add one from the web admin: Channels → New.',
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
          final ch = list[i];
          return _ChannelTile(
            channel: ch,
            busy: _busy.contains(ch.id),
            onTap: () => _onTap(ch),
          );
        },
      ),
    );
  }
}

enum _RowAction { test, toggleEnabled, toggleMuted, viewConfig, copyId }

class _ChannelTile extends StatelessWidget {
  const _ChannelTile({
    required this.channel,
    required this.busy,
    required this.onTap,
  });

  final ChannelView channel;
  final bool busy;
  final VoidCallback onTap;

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
          channel.kind.isNotEmpty ? channel.kind[0].toUpperCase() : '?',
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
              channel.kind,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(width: 6),
          _StatusBadges(channel: channel),
        ],
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Wrap(
          spacing: 6,
          runSpacing: 2,
          children: [
            Text(
              channel.id,
              style: const TextStyle(fontFamily: 'monospace'),
            ),
            if (channel.capabilities.isNotEmpty)
              Text('· caps: ${channel.capabilities.join(", ")}'),
          ],
        ),
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

class _StatusBadges extends StatelessWidget {
  const _StatusBadges({required this.channel});
  final ChannelView channel;

  @override
  Widget build(BuildContext context) {
    final badges = <Widget>[];
    if (channel.running) {
      badges.add(const _Badge(label: 'running', color: Colors.greenAccent));
    } else if (channel.enabled) {
      badges.add(const _Badge(label: 'starting…', color: Colors.amberAccent));
    } else {
      badges.add(_Badge(
        label: 'disabled',
        color: Theme.of(context).colorScheme.error,
      ));
    }
    if (channel.muted) {
      badges.add(const _Badge(label: 'muted', color: Colors.amberAccent));
    }
    return Wrap(spacing: 4, runSpacing: 2, children: badges);
  }
}

class _Badge extends StatelessWidget {
  const _Badge({required this.label, required this.color});
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
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
              'Failed to load channels',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('Retry')),
          ],
        ),
      ),
    );
  }
}
