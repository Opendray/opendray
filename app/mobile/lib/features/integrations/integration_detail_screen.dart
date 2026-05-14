import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/integrations_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/integrations/integration_forms.dart';

// Per-integration detail: header card with the registration metadata
// + cursor-paginated recent calls. Filter chips above the call list
// narrow by direction (in/out) and HTTP status family (2xx/4xx/5xx)
// — these are the dimensions the operator actually triages on:
// "show me what just failed" or "show me what this integration is
// pulling from us". Long-press a row to copy its request_id (the
// one anchor a backend operator needs to grep the gateway logs).
class IntegrationDetailScreen extends ConsumerStatefulWidget {
  const IntegrationDetailScreen({required this.integrationId, super.key});
  final String integrationId;

  @override
  ConsumerState<IntegrationDetailScreen> createState() =>
      _IntegrationDetailScreenState();
}

class _IntegrationDetailScreenState
    extends ConsumerState<IntegrationDetailScreen> {
  Integration? _integration;
  String? _detailError;

  final List<CallEntry> _calls = [];
  final _scroll = ScrollController();
  String? _cursor;
  bool _hasMore = true;
  bool _loadingCalls = false;
  bool _loadingMore = false;
  String? _callsError;

  _DirectionFilter _direction = _DirectionFilter.all;
  _StatusFilter _status = _StatusFilter.all;

  @override
  void initState() {
    super.initState();
    _scroll.addListener(_maybeLoadMore);
    _loadAll();
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _loadAll() async {
    await Future.wait([_loadDetail(), _reloadCalls()]);
  }

  Future<void> _loadDetail() async {
    try {
      final i =
          await ref.read(integrationsApiProvider).get(widget.integrationId);
      if (!mounted) return;
      setState(() {
        _integration = i;
        _detailError = null;
      });
    } on ApiException catch (e) {
      if (mounted) setState(() => _detailError = e.message);
    } on Object catch (e) {
      if (mounted) setState(() => _detailError = e.toString());
    }
  }

  Future<void> _reloadCalls() async {
    setState(() {
      _calls.clear();
      _cursor = null;
      _hasMore = true;
      _loadingCalls = true;
      _callsError = null;
    });
    await _fetchCallsPage(reset: true);
    if (mounted) setState(() => _loadingCalls = false);
  }

  void _maybeLoadMore() {
    if (!_hasMore || _loadingMore || _loadingCalls) return;
    if (_scroll.position.pixels >
        _scroll.position.maxScrollExtent - 200) {
      _loadMoreCalls();
    }
  }

  Future<void> _loadMoreCalls() async {
    if (!_hasMore) return;
    setState(() => _loadingMore = true);
    await _fetchCallsPage(reset: false);
    if (mounted) setState(() => _loadingMore = false);
  }

  Future<void> _fetchCallsPage({required bool reset}) async {
    try {
      final page = await ref.read(integrationsApiProvider).calls(
            integrationId: widget.integrationId,
            direction: _direction.value,
            statusClass: _status.value,
            cursor: reset ? null : _cursor,
            limit: 100,
          );
      if (!mounted) return;
      setState(() {
        if (reset) _calls.clear();
        _calls.addAll(page.entries);
        _cursor = page.nextCursor;
        _hasMore = page.nextCursor != null;
      });
    } on ApiException catch (e) {
      if (mounted) setState(() => _callsError = e.message);
    } on Object catch (e) {
      if (mounted) setState(() => _callsError = e.toString());
    }
  }

  Future<void> _copyRequestId(CallEntry e) async {
    final id = e.requestId;
    if (id == null || id.isEmpty) return;
    await Clipboard.setData(ClipboardData(text: id));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(t.integrations.copiedRequestId(id: id)),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  Future<void> _onEdit() async {
    final i = _integration;
    if (i == null) return;
    final patch = await EditIntegrationScreen.push(context, i);
    if (patch == null || patch.isEmpty || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(integrationsApiProvider).update(
            i.id,
            baseUrl: patch.baseUrl,
            scopes: patch.scopes,
            version: patch.version,
            enabled: patch.enabled,
          );
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.integrations.updateOk),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _loadDetail();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.integrations.updateFailedApi(error: e.message)),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.integrations.updateFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  Future<void> _onDelete() async {
    final i = _integration;
    if (i == null) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.integrations.deleteTitle),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(i.name, style: Theme.of(ctx).textTheme.titleSmall),
            const SizedBox(height: 4),
            Text(
              '/${i.routePrefix}',
              style: Theme.of(ctx)
                  .textTheme
                  .bodySmall
                  ?.copyWith(fontFamily: 'monospace'),
            ),
            const SizedBox(height: 12),
            Text(
              t.integrations.deleteBody,
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
    final messenger = ScaffoldMessenger.of(context);
    final navigator = Navigator.of(context);
    try {
      await ref.read(integrationsApiProvider).delete(i.id);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.integrations.deletedSnack(name: i.name)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      navigator.pop();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.integrations.deleteFailedApi(error: e.message)),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.integrations.deleteFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  Future<void> _onRotateKey() async {
    final i = _integration;
    if (i == null) return;
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.integrations.rotateConfirmTitle),
        content: Text(
          t.integrations.rotateBody(name: i.name),
          style: Theme.of(ctx).textTheme.bodyMedium,
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.integrations.rotate),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      final result = await ref.read(integrationsApiProvider).rotateKey(i.id);
      if (!mounted) return;
      await RevealApiKeyDialog.show(
        context: context,
        apiKey: result.apiKey,
        title: t.integrations.newApiKeyTitle(name: i.name),
        subtitle: t.integrations.newApiKeySubtitle,
      );
      if (!mounted) return;
      await _loadDetail();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.integrations.rotateFailedApi(error: e.message)),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.integrations.rotateFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    final i = _integration;
    final mutable = i != null && !i.isSystem;
    return Scaffold(
      appBar: AppBar(
        title: Text(i?.name ?? t.integrations.appBarFallback),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: _loadingCalls ? null : _loadAll,
          ),
          PopupMenuButton<_DetailAction>(
            enabled: mutable,
            tooltip: mutable
                ? t.integrations.tooltipMore
                : t.integrations.tooltipReadOnly,
            onSelected: (a) {
              switch (a) {
                case _DetailAction.edit:
                  _onEdit();
                case _DetailAction.rotateKey:
                  _onRotateKey();
                case _DetailAction.delete:
                  _onDelete();
              }
            },
            itemBuilder: (_) => [
              PopupMenuItem(
                value: _DetailAction.edit,
                child: ListTile(
                  leading: const Icon(Icons.edit_outlined),
                  title: Text(t.integrations.edit),
                ),
              ),
              PopupMenuItem(
                value: _DetailAction.rotateKey,
                child: ListTile(
                  leading: const Icon(Icons.vpn_key_outlined),
                  title: Text(t.integrations.rotateKey),
                ),
              ),
              PopupMenuItem(
                value: _DetailAction.delete,
                child: ListTile(
                  leading: const Icon(Icons.delete_outline,
                      color: Colors.redAccent),
                  title: Text(
                    t.common.delete,
                    style: const TextStyle(color: Colors.redAccent),
                  ),
                ),
              ),
            ],
          ),
        ],
      ),
      body: RefreshIndicator(
        onRefresh: _loadAll,
        child: CustomScrollView(
          controller: _scroll,
          slivers: [
            SliverToBoxAdapter(child: _header()),
            SliverPersistentHeader(
              pinned: true,
              delegate: _FilterStripDelegate(
                child: _FilterStrip(
                  direction: _direction,
                  status: _status,
                  onDirection: (d) {
                    setState(() => _direction = d);
                    _reloadCalls();
                  },
                  onStatus: (s) {
                    setState(() => _status = s);
                    _reloadCalls();
                  },
                ),
              ),
            ),
            if (_loadingCalls && _calls.isEmpty)
              const SliverFillRemaining(
                hasScrollBody: false,
                child: Center(child: CircularProgressIndicator()),
              )
            else if (_callsError != null && _calls.isEmpty)
              SliverFillRemaining(
                hasScrollBody: false,
                child: _InlineError(
                  message: _callsError!,
                  onRetry: _reloadCalls,
                ),
              )
            else if (_calls.isEmpty)
              SliverFillRemaining(
                hasScrollBody: false,
                child: Center(
                  child: Padding(
                    padding: const EdgeInsets.all(24),
                    child: Text(
                      t.integrations.noMatchingCalls,
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  ),
                ),
              )
            else
              SliverList.separated(
                itemCount: _calls.length + (_hasMore ? 1 : 0),
                separatorBuilder: (_, __) => Divider(
                  height: 1,
                  color: Theme.of(context).dividerColor,
                ),
                itemBuilder: (_, i) {
                  if (i >= _calls.length) {
                    return const Padding(
                      padding: EdgeInsets.all(20),
                      child: Center(
                        child: SizedBox(
                          width: 18,
                          height: 18,
                          child: CircularProgressIndicator(strokeWidth: 2),
                        ),
                      ),
                    );
                  }
                  return _CallTile(
                    entry: _calls[i],
                    onLongPress: () => _copyRequestId(_calls[i]),
                  );
                },
              ),
          ],
        ),
      ),
    );
  }

  Widget _header() {
    if (_detailError != null && _integration == null) {
      return Padding(
        padding: const EdgeInsets.all(16),
        child: Text(
          t.integrations.detailLoadFailed(error: _detailError ?? ''),
          style: TextStyle(color: Theme.of(context).colorScheme.error),
        ),
      );
    }
    final i = _integration;
    if (i == null) {
      return const Padding(
        padding: EdgeInsets.all(24),
        child: Center(child: CircularProgressIndicator()),
      );
    }
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 4),
      child: Card(
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                children: [
                  _StatusBadge(status: i.healthStatus),
                  const SizedBox(width: 8),
                  if (!i.enabled)
                    Padding(
                      padding: const EdgeInsets.only(left: 4),
                      child: _Tag(
                        label: 'disabled',
                        color: Theme.of(context).colorScheme.error,
                      ),
                    ),
                  if (i.isSystem)
                    Padding(
                      padding: const EdgeInsets.only(left: 4),
                      child: _Tag(
                        label: 'system',
                        color: Theme.of(context).colorScheme.tertiary,
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 12),
              _KV(label: t.integrations.kvRoutePrefix, value: '/${i.routePrefix}', mono: true),
              _KV(label: t.integrations.kvBaseUrl, value: i.baseUrl, mono: true),
              _KV(
                label: t.integrations.kvScopes,
                value: i.scopes.isEmpty ? '(none)' : i.scopes.join(', '),
              ),
              if ((i.version ?? '').isNotEmpty)
                _KV(label: t.integrations.kvVersion, value: i.version!),
              _KV(
                label: t.integrations.kvLastHealthPing,
                value: i.healthLastSeen == null
                    ? 'never'
                    : DateFormat.yMMMd()
                        .add_Hms()
                        .format(i.healthLastSeen!.toLocal()),
              ),
              _KV(
                label: t.integrations.kvCreated,
                value: DateFormat.yMMMd()
                    .add_Hms()
                    .format(i.createdAt.toLocal()),
              ),
              if (i.rotatedAt != null)
                _KV(
                  label: t.integrations.kvKeyRotated,
                  value: DateFormat.yMMMd()
                      .add_Hms()
                      .format(i.rotatedAt!.toLocal()),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

enum _DetailAction { edit, rotateKey, delete }

enum _DirectionFilter {
  all('All', null),
  inbound('Inbound', 'inbound'),
  outbound('Outbound', 'outbound');

  const _DirectionFilter(this.label, this.value);
  final String label;
  final String? value;
}

enum _StatusFilter {
  all('All', null),
  ok('2xx', 2),
  client('4xx', 4),
  server('5xx', 5);

  const _StatusFilter(this.label, this.value);
  final String label;
  final int? value;
}

class _FilterStrip extends StatelessWidget {
  const _FilterStrip({
    required this.direction,
    required this.status,
    required this.onDirection,
    required this.onStatus,
  });

  final _DirectionFilter direction;
  final _StatusFilter status;
  final ValueChanged<_DirectionFilter> onDirection;
  final ValueChanged<_StatusFilter> onStatus;

  String _directionLabel(_DirectionFilter d) {
    switch (d) {
      case _DirectionFilter.all:
        return t.integrations.directionAll;
      case _DirectionFilter.inbound:
        return t.integrations.directionInbound;
      case _DirectionFilter.outbound:
        return t.integrations.directionOutbound;
    }
  }

  @override
  Widget build(BuildContext context) {
    return Container(
      color: Theme.of(context).colorScheme.surface,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      child: SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          children: [
            for (final d in _DirectionFilter.values) ...[
              ChoiceChip(
                label: Text(_directionLabel(d)),
                selected: d == direction,
                onSelected: (_) => onDirection(d),
                visualDensity: VisualDensity.compact,
              ),
              const SizedBox(width: 6),
            ],
            const SizedBox(width: 6),
            Container(width: 1, height: 24, color: Theme.of(context).dividerColor),
            const SizedBox(width: 6),
            for (final s in _StatusFilter.values) ...[
              ChoiceChip(
                label: Text(s.label),
                selected: s == status,
                onSelected: (_) => onStatus(s),
                visualDensity: VisualDensity.compact,
              ),
              const SizedBox(width: 6),
            ],
          ],
        ),
      ),
    );
  }
}

class _FilterStripDelegate extends SliverPersistentHeaderDelegate {
  _FilterStripDelegate({required this.child});
  final Widget child;

  @override
  double get minExtent => 52;
  @override
  double get maxExtent => 52;

  @override
  Widget build(BuildContext context, double shrinkOffset, bool overlapsContent) {
    return Material(
      elevation: overlapsContent ? 1 : 0,
      child: child,
    );
  }

  @override
  bool shouldRebuild(_FilterStripDelegate oldDelegate) =>
      oldDelegate.child != child;
}

class _CallTile extends StatelessWidget {
  const _CallTile({required this.entry, required this.onLongPress});
  final CallEntry entry;
  final VoidCallback onLongPress;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return InkWell(
      onLongPress: onLongPress,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 10, 14, 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  entry.direction == 'inbound'
                      ? Icons.south_west
                      : Icons.north_east,
                  size: 14,
                  color: muted?.color,
                ),
                const SizedBox(width: 6),
                _StatusCodeBadge(code: entry.statusCode),
                const SizedBox(width: 8),
                Text(
                  entry.method,
                  style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                  ),
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    entry.path,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Text('${entry.durationMs}ms', style: muted),
              ],
            ),
            const SizedBox(height: 2),
            Text(
              DateFormat.yMMMd().add_Hms().format(entry.timestamp.toLocal()),
              style: muted,
            ),
          ],
        ),
      ),
    );
  }
}

class _StatusBadge extends StatelessWidget {
  const _StatusBadge({required this.status});
  final String status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      'healthy' => Colors.green,
      'degraded' => Colors.amber,
      'unhealthy' => Colors.red,
      _ => Colors.grey,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: color.withValues(alpha: 0.6)),
      ),
      child: Text(
        status,
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _StatusCodeBadge extends StatelessWidget {
  const _StatusCodeBadge({required this.code});
  final int code;

  @override
  Widget build(BuildContext context) {
    final color = switch (code ~/ 100) {
      2 => Colors.greenAccent,
      3 => Colors.lightBlueAccent,
      4 => Colors.amberAccent,
      5 => Colors.redAccent,
      _ => Colors.grey,
    };
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.18),
        borderRadius: BorderRadius.circular(4),
      ),
      child: Text(
        '$code',
        style: TextStyle(
          color: color,
          fontSize: 11,
          fontFamily: 'monospace',
          fontWeight: FontWeight.w600,
        ),
      ),
    );
  }
}

class _Tag extends StatelessWidget {
  const _Tag({required this.label, required this.color});
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

class _KV extends StatelessWidget {
  const _KV({required this.label, required this.value, this.mono = false});
  final String label;
  final String value;
  final bool mono;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.only(bottom: 6),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(label, style: Theme.of(context).textTheme.bodySmall),
          SelectableText(
            value,
            style: TextStyle(
              fontSize: 13,
              fontFamily: mono ? 'monospace' : null,
            ),
          ),
        ],
      ),
    );
  }
}

class _InlineError extends StatelessWidget {
  const _InlineError({required this.message, required this.onRetry});
  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Text(
              t.integrations.callsLoadFailed,
              style: Theme.of(context).textTheme.titleSmall,
            ),
            const SizedBox(height: 6),
            Text(
              message,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
