import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/channels_api.dart';
import 'package:opendray/core/auth/auth_state.dart';
import 'package:opendray/core/i18n/strings.g.dart' as i18n;
import 'package:opendray/features/channels/channel_form_dialog.dart';
import 'package:opendray/features/channels/channel_kinds.dart';

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
        SnackBar(
          content: Text(
            i18n.t.channels.errorWithMessage(
              prefix: failPrefix,
              error: e.message,
            ),
          ),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            i18n.t.channels.errorWithMessage(
              prefix: failPrefix,
              error: e.toString(),
            ),
          ),
        ),
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
              title: Text(i18n.t.channels.sendTest),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.test),
            ),
            ListTile(
              enabled: !isBusy,
              leading: Icon(
                ch.enabled ? Icons.pause_circle_outline : Icons.play_circle_outline,
              ),
              title: Text(ch.enabled ? i18n.t.channels.popup.disable : i18n.t.channels.popup.enable),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.toggleEnabled),
            ),
            ListTile(
              enabled: !isBusy,
              leading: Icon(
                ch.muted ? Icons.notifications_active_outlined : Icons.notifications_off_outlined,
              ),
              title: Text(ch.muted ? i18n.t.channels.popup.unmute : i18n.t.channels.popup.mute),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.toggleMuted),
            ),
            const Divider(height: 1),
            ListTile(
              enabled: !isBusy && findKind(ch.kind) != null,
              leading: const Icon(Icons.edit_outlined),
              title: Text(i18n.t.channels.editConfig),
              subtitle: findKind(ch.kind) == null
                  ? Text(
                      i18n.t.channels.bridgeWebOnly,
                      style: const TextStyle(fontSize: 11),
                    )
                  : null,
              onTap: () =>
                  Navigator.of(sheetCtx).pop(_RowAction.editKindConfig),
            ),
            ListTile(
              enabled: !isBusy,
              leading: const Icon(Icons.tune_outlined),
              title: Text(i18n.t.channels.editNotifications),
              onTap: () =>
                  Navigator.of(sheetCtx).pop(_RowAction.editNotify),
            ),
            ListTile(
              leading: const Icon(Icons.code),
              title: Text(i18n.t.channels.viewRawConfig),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.viewConfig),
            ),
            ListTile(
              leading: const Icon(Icons.copy_outlined),
              title: Text(i18n.t.channels.copyChannelId),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.copyId),
            ),
            const Divider(height: 1),
            ListTile(
              enabled: !isBusy,
              leading: Icon(
                Icons.delete_outline,
                color: Theme.of(sheetCtx).colorScheme.error,
              ),
              title: Text(
                i18n.t.channels.popup.deleteLabel,
                style: TextStyle(color: Theme.of(sheetCtx).colorScheme.error),
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_RowAction.delete),
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
          okMessage: i18n.t.channels.snacks.testDispatched,
          failPrefix: i18n.t.channels.errorPrefix.test,
          op: () => ref.read(channelsApiProvider).test(ch.id),
        );
      case _RowAction.toggleEnabled:
        final next = !ch.enabled;
        await _runAction(
          id: ch.id,
          okMessage: next ? i18n.t.channels.snacks.channelEnabled : i18n.t.channels.snacks.channelDisabled,
          failPrefix: i18n.t.channels.errorPrefix.toggle,
          op: () => ref
              .read(channelsApiProvider)
              .setEnabled(ch.id, enabled: next)
              .then((_) {}),
        );
      case _RowAction.toggleMuted:
        final next = !ch.muted;
        await _runAction(
          id: ch.id,
          okMessage: next ? i18n.t.channels.snacks.channelMuted : i18n.t.channels.snacks.channelUnmuted,
          failPrefix: i18n.t.channels.errorPrefix.muteToggle,
          op: () => ref
              .read(channelsApiProvider)
              .setMuted(ch.id, muted: next)
              .then((_) {}),
        );
      case _RowAction.editKindConfig:
        await _editKindConfig(ch);
      case _RowAction.editNotify:
        await _editNotifyPrefs(ch);
      case _RowAction.viewConfig:
        await _showConfig(ch);
      case _RowAction.copyId:
        await Clipboard.setData(ClipboardData(text: ch.id));
        if (!mounted) return;
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            content: Text(i18n.t.channels.copiedSnack(id: ch.id)),
            duration: const Duration(seconds: 2),
            behavior: SnackBarBehavior.floating,
          ),
        );
      case _RowAction.delete:
        await _confirmAndDelete(ch);
    }
  }

  Future<void> _onCreate() async {
    final kind = await ChannelKindPickerSheet.show(context);
    if (kind == null || !mounted) return;
    final cfg = await ChannelFormScreen.push(context: context, kind: kind);
    if (cfg == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      final created = await ref
          .read(channelsApiProvider)
          .create(kind: kind.kind, config: cfg);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(i18n.t.channels.createdSnack(kind: kind.label)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      if (kind.webhookBased) {
        final auth = ref.read(authControllerProvider);
        if (auth is AuthLoggedIn) {
          await PostCreateWebhookDialog.show(
            context: context,
            serverUrl: auth.serverUrl,
            channelId: created.id,
            kind: kind,
          );
        }
      }
      if (!mounted) return;
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            i18n.t.channels.createFailedApi(error: e.message),
          ),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            i18n.t.channels.createFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  Future<void> _editKindConfig(ChannelView ch) async {
    final kind = findKind(ch.kind);
    if (kind == null) return;
    // Pull non-notify_* keys out so the kind-specific form only shows
    // its own fields. Notify prefs and the muted flag round-trip
    // untouched via the merge in updateConfig.
    final cfg = await ChannelFormScreen.push(
      context: context,
      kind: kind,
      initial: ch.config,
    );
    if (cfg == null || !mounted) return;
    // Build a sparse patch: any field present in the form (cfg) gets
    // upserted; any kind-defined optional field whose form value is
    // empty becomes a remove sentinel so the server-side default
    // applies cleanly.
    final patch = <String, Object?>{};
    for (final f in kind.fields) {
      if (cfg.containsKey(f.name)) {
        patch[f.name] = cfg[f.name];
      } else {
        // Field was omitted from the form result (optional + blank) —
        // explicitly drop it from the saved config.
        patch[f.name] = removeChannelConfigKey;
      }
    }
    await _runAction(
      id: ch.id,
      okMessage: i18n.t.channels.snacks.configUpdated,
      failPrefix: i18n.t.channels.errorPrefix.update,
      op: () => ref
          .read(channelsApiProvider)
          .updateConfig(ch.id, patch)
          .then((_) {}),
    );
  }

  Future<void> _editNotifyPrefs(ChannelView ch) async {
    final patch = await Navigator.of(context).push<Map<String, Object?>>(
      MaterialPageRoute<Map<String, Object?>>(
        builder: (_) => _NotifyPrefsScreen(channel: ch),
        fullscreenDialog: true,
      ),
    );
    if (patch == null || patch.isEmpty || !mounted) return;
    await _runAction(
      id: ch.id,
      okMessage: i18n.t.channels.notifications.updatedSnack,
      failPrefix: i18n.t.channels.errorPrefix.update,
      op: () => ref
          .read(channelsApiProvider)
          .updateConfig(ch.id, patch)
          .then((_) {}),
    );
  }

  Future<void> _confirmAndDelete(ChannelView ch) async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(i18n.t.channels.deleteTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(ch.kind, style: Theme.of(ctx).textTheme.titleSmall),
            const SizedBox(height: 4),
            Text(
              ch.id,
              style: Theme.of(ctx)
                  .textTheme
                  .bodySmall
                  ?.copyWith(fontFamily: 'monospace'),
            ),
            const SizedBox(height: 12),
            Text(
              i18n.t.channels.deleteBody,
              style: Theme.of(ctx).textTheme.bodySmall,
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(i18n.t.common.cancel),
          ),
          FilledButton(
            style: FilledButton.styleFrom(
              backgroundColor: Theme.of(ctx).colorScheme.error,
            ),
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(i18n.t.common.delete),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    await _runAction(
      id: ch.id,
      okMessage: i18n.t.channels.snacks.channelDeleted,
      failPrefix: i18n.t.channels.errorPrefix.delete,
      op: () => ref.read(channelsApiProvider).delete(ch.id),
    );
  }

  Future<void> _showConfig(ChannelView ch) async {
    const enc = JsonEncoder.withIndent('  ');
    final pretty = enc.convert(ch.config);
    await showDialog<void>(
      context: context,
      builder: (dialogCtx) => AlertDialog(
        title: Text(i18n.t.channels.configDialog.title(kind: ch.kind)),
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
            child: Text(i18n.t.common.copy),
          ),
          FilledButton(
            onPressed: () => Navigator.of(dialogCtx).pop(),
            child: Text(i18n.t.common.close),
          ),
        ],
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(i18n.t.channels.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: i18n.t.common.refresh,
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
        heroTag: 'channels_fab',
        onPressed: _onCreate,
        icon: const Icon(Icons.add),
        label: Text(i18n.t.channels.kNew),
      ),
    );
  }

  Widget _buildList(List<ChannelView> list) {
    if (list.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            i18n.t.channels.bridgeEmptyAdd,
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

enum _RowAction {
  test,
  toggleEnabled,
  toggleMuted,
  editKindConfig,
  editNotify,
  viewConfig,
  copyId,
  delete,
}

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
              Text(i18n.t.channels.capsLabel(list: channel.capabilities.join(', '))),
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
      badges.add(_Badge(label: i18n.t.channels.badges.running, color: Colors.greenAccent));
    } else if (channel.enabled) {
      badges.add(_Badge(label: i18n.t.channels.badges.starting, color: Colors.amberAccent));
    } else {
      badges.add(_Badge(
        label: i18n.t.channels.badges.disabled,
        color: Theme.of(context).colorScheme.error,
      ));
    }
    if (channel.muted) {
      badges.add(_Badge(label: i18n.t.channels.badges.muted, color: Colors.amberAccent));
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

// _NotifyPrefsScreen edits the kind-agnostic notification keys that
// every channel supports: which session topics to fire on, which
// repeat policy, snippet inclusion + cap. Returns a sparse patch
// map suitable for ChannelsApi.updateConfig — keys the operator
// reverted to defaults are emitted with the removeChannelConfigKey
// sentinel so the merge clears them.
class _NotifyPrefsScreen extends StatefulWidget {
  const _NotifyPrefsScreen({required this.channel});
  final ChannelView channel;

  @override
  State<_NotifyPrefsScreen> createState() => _NotifyPrefsScreenState();
}

class _NotifyPrefsScreenState extends State<_NotifyPrefsScreen> {
  static const _allTopics = ['session.started', 'session.idle', 'session.ended'];
  static List<(String, String, String)> _modes() => [
    (
      'once',
      i18n.t.channels.notifications.modes.onceLabel,
      i18n.t.channels.notifications.modes.onceDescription,
    ),
    (
      'cooldown',
      i18n.t.channels.notifications.modes.cooldownLabel,
      i18n.t.channels.notifications.modes.cooldownDescription,
    ),
    (
      'every',
      i18n.t.channels.notifications.modes.everyLabel,
      i18n.t.channels.notifications.modes.everyDescription,
    ),
  ];
  static const _cooldownPresets = [
    (60, '1m'),
    (300, '5m'),
    (900, '15m'),
    (1800, '30m'),
    (3600, '1h'),
  ];

  late Set<String> _topics;
  // 'once' is the server default; UI shows it but emits a remove
  // sentinel so the saved config doesn't pin an explicit override.
  late String _mode;
  late int _cooldownSec;
  late bool _includeSnippet;
  late int _snippetCap; // 0 = no cap

  @override
  void initState() {
    super.initState();
    final cfg = widget.channel.config;
    final topicsRaw = cfg['notify_on'];
    _topics = topicsRaw is List
        ? Set<String>.from(topicsRaw.whereType<String>())
        : <String>{..._allTopics};
    if (_topics.isEmpty) _topics = {..._allTopics};
    _mode = (cfg['notify_mode'] as String?) ?? 'once';
    _cooldownSec = (cfg['notify_cooldown_s'] as num?)?.toInt() ?? 300;
    _includeSnippet = cfg['notify_include_snippet'] as bool? ?? true;
    _snippetCap = (cfg['notify_snippet_max_chars'] as num?)?.toInt() ?? 0;
  }

  Map<String, Object?> _buildPatch() {
    final patch = <String, Object?>{};

    // Topics: persist only when partial selection. All-three or empty
    // means "any topic" (server default), so drop the key.
    if (_topics.length == _allTopics.length || _topics.isEmpty) {
      patch['notify_on'] = removeChannelConfigKey;
    } else {
      patch['notify_on'] = _allTopics.where(_topics.contains).toList();
    }

    if (_mode == 'once') {
      patch['notify_mode'] = removeChannelConfigKey;
    } else {
      patch['notify_mode'] = _mode;
    }

    if (_mode == 'cooldown') {
      patch['notify_cooldown_s'] = _cooldownSec;
    } else {
      patch['notify_cooldown_s'] = removeChannelConfigKey;
    }

    if (_includeSnippet) {
      patch['notify_include_snippet'] = removeChannelConfigKey;
    } else {
      patch['notify_include_snippet'] = false;
    }

    if (_snippetCap > 0) {
      patch['notify_snippet_max_chars'] = _snippetCap;
    } else {
      patch['notify_snippet_max_chars'] = removeChannelConfigKey;
    }

    return patch;
  }

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return Scaffold(
      appBar: AppBar(
        title: Text(i18n.t.channels.notifications.title),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(_buildPatch()),
            child: Text(i18n.t.common.save),
          ),
        ],
      ),
      body: ListView(
        padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
        children: [
          Text(i18n.t.channels.notifications.notifyOn, style: muted),
          const SizedBox(height: 6),
          Wrap(
            spacing: 6,
            runSpacing: 4,
            children: [
              for (final t in _allTopics)
                FilterChip(
                  label: Text(t,
                      style: const TextStyle(
                          fontFamily: 'monospace', fontSize: 12)),
                  selected: _topics.contains(t),
                  onSelected: (v) => setState(() {
                    if (v) {
                      _topics.add(t);
                    } else {
                      _topics.remove(t);
                    }
                  }),
                ),
            ],
          ),
          Padding(
            padding: const EdgeInsets.only(top: 6),
            child: Text(
              _topics.length == _allTopics.length
                  ? i18n.t.channels.notifications.notifyOnAll
                  : _topics.isEmpty
                      ? i18n.t.channels.notifications.notifyOnEmpty
                      : '${_topics.length} of ${_allTopics.length} selected.',
              style: muted,
            ),
          ),
          const SizedBox(height: 24),
          Text(i18n.t.channels.notifications.repeatPolicy, style: muted),
          const SizedBox(height: 6),
          RadioGroup<String>(
            groupValue: _mode,
            onChanged: (v) => setState(() => _mode = v ?? _mode),
            child: Column(
              children: [
                for (final (val, label, hint) in _modes())
                  RadioListTile<String>(
                    value: val,
                    title: Text(label),
                    subtitle: Text(hint, style: muted),
                    contentPadding: EdgeInsets.zero,
                  ),
              ],
            ),
          ),
          if (_mode == 'cooldown') ...[
            const SizedBox(height: 8),
            Text(i18n.t.channels.notifications.cooldownWindow, style: muted),
            const SizedBox(height: 6),
            Wrap(
              spacing: 6,
              runSpacing: 4,
              children: [
                for (final (sec, label) in _cooldownPresets)
                  ChoiceChip(
                    label: Text(label),
                    selected: _cooldownSec == sec,
                    onSelected: (_) => setState(() => _cooldownSec = sec),
                  ),
              ],
            ),
          ],
          const SizedBox(height: 24),
          SwitchListTile(
            contentPadding: EdgeInsets.zero,
            title: Text(i18n.t.channels.notifications.includeSnippet),
            subtitle: Text(
              i18n.t.channels.notifications.snippetHelper,
              style: muted,
            ),
            value: _includeSnippet,
            onChanged: (v) => setState(() => _includeSnippet = v),
          ),
          if (_includeSnippet) ...[
            const SizedBox(height: 8),
            Text(i18n.t.channels.notifications.snippetLengthCap, style: muted),
            const SizedBox(height: 6),
            Wrap(
              spacing: 6,
              runSpacing: 4,
              children: [
                for (final n in const [0, 200, 500, 1000, 2000])
                  ChoiceChip(
                    label: Text(n == 0 ? i18n.t.channels.notifications.snippetNoCap : i18n.t.channels.notifications.snippetChars(n: n.toString())),
                    selected: _snippetCap == n,
                    onSelected: (_) => setState(() => _snippetCap = n),
                  ),
              ],
            ),
          ],
        ],
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
              i18n.t.channels.failedToLoad,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(i18n.t.common.retry)),
          ],
        ),
      ),
    );
  }
}
