import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/integrations_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Activity — the gateway's per-call audit log: every API request made
// by a registered integration (inbound: app → opendray; outbound:
// opendray → integration). Web renders this as a filter-bar + table;
// on mobile we adapt to a reverse-chronological card list with filters
// behind a bottom sheet and a tap-through detail sheet, so the gateway's
// core observability surface fits the phone.
//
// Reuses IntegrationsApi.calls()/list() — no new client needed. The
// latest 100 calls are shown (server default); pagination is deferred
// to match the web view's current behaviour.
class ActivityScreen extends ConsumerStatefulWidget {
  const ActivityScreen({super.key});

  @override
  ConsumerState<ActivityScreen> createState() => _ActivityScreenState();
}

class _ActivityScreenState extends ConsumerState<ActivityScreen> {
  AsyncValue<List<CallEntry>> _state = const AsyncValue.loading();
  Map<String, String> _names = const {};

  // Filters. Empty string / null = unset.
  String _integrationId = '';
  String _direction = '';
  int? _statusClass;

  int get _activeFilters =>
      (_integrationId.isEmpty ? 0 : 1) +
      (_direction.isEmpty ? 0 : 1) +
      (_statusClass == null ? 0 : 1);

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final api = ref.read(integrationsApiProvider);
      // Names power the filter picker + per-row label resolution.
      final integrations = await api.list();
      final page = await api.calls(
        integrationId: _integrationId,
        direction: _direction,
        statusClass: _statusClass,
      );
      if (!mounted) return;
      setState(() {
        _names = {for (final i in integrations) i.id: i.name};
        _state = AsyncValue.data(page.entries);
      });
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _openFilter() async {
    final integrations = _names.entries.toList()
      ..sort((a, b) => a.value.toLowerCase().compareTo(b.value.toLowerCase()));
    final result = await showModalBottomSheet<_FilterResult>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (_) => _FilterSheet(
        integrations: integrations,
        integrationId: _integrationId,
        direction: _direction,
        statusClass: _statusClass,
      ),
    );
    if (result == null || !mounted) return;
    setState(() {
      _integrationId = result.integrationId;
      _direction = result.direction;
      _statusClass = result.statusClass;
    });
    await _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.activity.title),
        actions: [
          IconButton(
            icon: Badge(
              isLabelVisible: _activeFilters > 0,
              label: Text('$_activeFilters'),
              child: const Icon(Icons.filter_list),
            ),
            tooltip: t.activity.filter.title,
            onPressed: _openFilter,
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
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

  Widget _buildList(List<CallEntry> calls) {
    if (calls.isEmpty) {
      return RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          children: [
            const SizedBox(height: 120),
            Center(
              child: Padding(
                padding: const EdgeInsets.all(24),
                child: Text(
                  t.activity.empty,
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
              ),
            ),
          ],
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.builder(
        itemCount: calls.length + 1,
        itemBuilder: (context, i) {
          if (i == calls.length) {
            return Padding(
              padding: const EdgeInsets.fromLTRB(16, 8, 16, 24),
              child: Text(
                t.activity.callsCount(count: calls.length),
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      color: Theme.of(context)
                          .colorScheme
                          .onSurface
                          .withValues(alpha: 0.5),
                    ),
              ),
            );
          }
          final call = calls[i];
          return _CallCard(
            call: call,
            integrationName: _names[call.integrationId] ?? call.integrationId,
            onTap: () => _openDetail(call),
          );
        },
      ),
    );
  }

  void _openDetail(CallEntry call) {
    showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      showDragHandle: true,
      builder: (_) => _DetailSheet(
        call: call,
        integrationName: _names[call.integrationId] ?? call.integrationId,
      ),
    );
  }
}

// ── shared helpers ────────────────────────────────────────────────

Color _statusColor(BuildContext context, int code) {
  switch (code ~/ 100) {
    case 2:
      return Colors.greenAccent;
    case 3:
      return Colors.lightBlueAccent;
    case 4:
      return Colors.amberAccent;
    case 5:
      return Colors.redAccent;
    default:
      return Theme.of(context).colorScheme.outline;
  }
}

String _relativeTime(DateTime ts) {
  final diff = DateTime.now().toUtc().difference(ts.toUtc());
  if (diff.inSeconds < 60) return 'just now';
  if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
  if (diff.inHours < 24) return '${diff.inHours}h ago';
  if (diff.inDays < 7) return '${diff.inDays}d ago';
  return DateFormat.MMMd().add_Hm().format(ts.toLocal());
}

String _directionLabel(String direction) =>
    direction == 'inbound' ? t.activity.directionInbound : t.activity.directionOutbound;

IconData _directionIcon(String direction) =>
    direction == 'inbound' ? Icons.south_west : Icons.north_east;

// ── call card ─────────────────────────────────────────────────────

class _CallCard extends StatelessWidget {
  const _CallCard({
    required this.call,
    required this.integrationName,
    required this.onTap,
  });

  final CallEntry call;
  final String integrationName;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall?.copyWith(
          color:
              Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
        );
    final statusColor = _statusColor(context, call.statusCode);
    return ListTile(
      onTap: onTap,
      leading: _MethodChip(method: call.method),
      title: Text(
        call.path,
        maxLines: 1,
        overflow: TextOverflow.ellipsis,
        style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Padding(
          padding: const EdgeInsets.only(top: 2),
          child: Row(
            children: [
              Icon(_directionIcon(call.direction), size: 12, color: muted?.color),
              const SizedBox(width: 3),
              Flexible(
                child: Text(integrationName, overflow: TextOverflow.ellipsis),
              ),
              const SizedBox(width: 6),
              Text('· ${call.durationMs}ms'),
              const SizedBox(width: 6),
              Text('· ${_relativeTime(call.timestamp)}'),
            ],
          ),
        ),
      ),
      trailing: _StatusPill(code: call.statusCode, color: statusColor),
    );
  }
}

class _MethodChip extends StatelessWidget {
  const _MethodChip({required this.method});
  final String method;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      width: 48,
      child: Text(
        method.toUpperCase(),
        textAlign: TextAlign.center,
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              fontFamily: 'monospace',
              fontWeight: FontWeight.w700,
              color:
                  Theme.of(context).colorScheme.primary.withValues(alpha: 0.9),
            ),
      ),
    );
  }
}

class _StatusPill extends StatelessWidget {
  const _StatusPill({required this.code, required this.color});
  final int code;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(6),
        border: Border.all(color: color.withValues(alpha: 0.6), width: 0.5),
      ),
      child: Text(
        '$code',
        style: TextStyle(
          color: color,
          fontFamily: 'monospace',
          fontWeight: FontWeight.w700,
          fontSize: 12,
        ),
      ),
    );
  }
}

// ── filter sheet ──────────────────────────────────────────────────

class _FilterResult {
  const _FilterResult({
    required this.integrationId,
    required this.direction,
    required this.statusClass,
  });
  final String integrationId;
  final String direction;
  final int? statusClass;
}

class _FilterSheet extends StatefulWidget {
  const _FilterSheet({
    required this.integrations,
    required this.integrationId,
    required this.direction,
    required this.statusClass,
  });

  final List<MapEntry<String, String>> integrations;
  final String integrationId;
  final String direction;
  final int? statusClass;

  @override
  State<_FilterSheet> createState() => _FilterSheetState();
}

class _FilterSheetState extends State<_FilterSheet> {
  late String _integrationId = widget.integrationId;
  late String _direction = widget.direction;
  late int? _statusClass = widget.statusClass;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 4, 20, 20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(
              t.activity.filter.title,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 16),
            _label(context, t.activity.filter.direction),
            Wrap(
              spacing: 8,
              children: [
                _choice(t.activity.filter.directionAll, _direction.isEmpty,
                    () => setState(() => _direction = '')),
                _choice(t.activity.directionInbound, _direction == 'inbound',
                    () => setState(() => _direction = 'inbound')),
                _choice(t.activity.directionOutbound, _direction == 'outbound',
                    () => setState(() => _direction = 'outbound')),
              ],
            ),
            const SizedBox(height: 16),
            _label(context, t.activity.filter.status),
            Wrap(
              spacing: 8,
              children: [
                _choice(t.activity.filter.statusAll, _statusClass == null,
                    () => setState(() => _statusClass = null)),
                for (final c in const [2, 3, 4, 5])
                  _choice('${c}xx', _statusClass == c,
                      () => setState(() => _statusClass = c)),
              ],
            ),
            const SizedBox(height: 16),
            _label(context, t.activity.filter.integration),
            DropdownButtonFormField<String>(
              initialValue: _integrationId.isEmpty ? '' : _integrationId,
              decoration: const InputDecoration(
                border: OutlineInputBorder(),
                isDense: true,
              ),
              items: [
                DropdownMenuItem(
                  value: '',
                  child: Text(t.activity.filter.integrationAll),
                ),
                for (final e in widget.integrations)
                  DropdownMenuItem(value: e.key, child: Text(e.value)),
              ],
              onChanged: (v) => setState(() => _integrationId = v ?? ''),
            ),
            const SizedBox(height: 20),
            Row(
              children: [
                TextButton(
                  onPressed: () => setState(() {
                    _integrationId = '';
                    _direction = '';
                    _statusClass = null;
                  }),
                  child: Text(t.activity.filter.clear),
                ),
                const Spacer(),
                FilledButton(
                  onPressed: () => Navigator.of(context).pop(
                    _FilterResult(
                      integrationId: _integrationId,
                      direction: _direction,
                      statusClass: _statusClass,
                    ),
                  ),
                  child: Text(t.activity.filter.apply),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _label(BuildContext context, String text) => Padding(
        padding: const EdgeInsets.only(bottom: 8),
        child: Text(
          text.toUpperCase(),
          style: Theme.of(context).textTheme.labelSmall?.copyWith(
                letterSpacing: 0.8,
                color: Theme.of(context)
                    .colorScheme
                    .onSurface
                    .withValues(alpha: 0.6),
              ),
        ),
      );

  Widget _choice(String label, bool selected, VoidCallback onTap) =>
      ChoiceChip(
        label: Text(label),
        selected: selected,
        onSelected: (_) => onTap(),
      );
}

// ── detail sheet ──────────────────────────────────────────────────

class _DetailSheet extends StatelessWidget {
  const _DetailSheet({required this.call, required this.integrationName});
  final CallEntry call;
  final String integrationName;

  @override
  Widget build(BuildContext context) {
    final statusColor = _statusColor(context, call.statusCode);
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 4, 20, 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                _MethodChip(method: call.method),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    call.path,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 13,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                _StatusPill(code: call.statusCode, color: statusColor),
              ],
            ),
            const SizedBox(height: 16),
            _row(context, t.activity.detail.integration, integrationName),
            _row(context, t.activity.detail.direction,
                _directionLabel(call.direction)),
            _row(context, t.activity.detail.duration, '${call.durationMs} ms'),
            if (call.bytesWritten != null)
              _row(context, t.activity.detail.bytes, '${call.bytesWritten}'),
            if ((call.requestId ?? '').isNotEmpty)
              _row(context, t.activity.detail.requestId, call.requestId!,
                  mono: true),
            if ((call.resourceKind ?? '').isNotEmpty)
              _row(
                context,
                t.activity.detail.resource,
                '${call.resourceKind}${(call.resourceId ?? '').isEmpty ? '' : ' · ${call.resourceId}'}',
              ),
            _row(
              context,
              t.activity.detail.timestamp,
              DateFormat.yMMMd().add_Hms().format(call.timestamp.toLocal()),
            ),
          ],
        ),
      ),
    );
  }

  Widget _row(BuildContext context, String label, String value,
      {bool mono = false}) {
    final muted = Theme.of(context).textTheme.bodySmall?.copyWith(
          color:
              Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
        );
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(width: 96, child: Text(label, style: muted)),
          Expanded(
            child: Text(
              value,
              style: mono ? const TextStyle(fontFamily: 'monospace') : null,
            ),
          ),
        ],
      ),
    );
  }
}

// ── error view ────────────────────────────────────────────────────

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
            Icon(Icons.error_outline,
                size: 48, color: Theme.of(context).colorScheme.error),
            const SizedBox(height: 12),
            Text(t.activity.loadFailed,
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 6),
            Text(error,
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodySmall),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
