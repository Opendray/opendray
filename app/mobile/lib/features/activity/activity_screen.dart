import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/audit_api.dart';
import 'package:opendray/core/api/models.dart';

// Global Activity tab — infinite-scroll over /api/v1/audit/log.
// Filter chips at the top narrow by subject_kind (the cheapest
// dimension to slice on); action prefix and time-range filters are
// punted to a future "advanced filter" sheet since most operators
// reach the entry they care about by scrolling.
//
// Each row renders timestamp + action badge + actor → subject; tap
// expands the metadata blob inline as monospace JSON. Long-press
// copies the metadata JSON to the clipboard.
class ActivityScreen extends ConsumerStatefulWidget {
  const ActivityScreen({super.key});

  @override
  ConsumerState<ActivityScreen> createState() => _ActivityScreenState();
}

class _ActivityScreenState extends ConsumerState<ActivityScreen> {
  final List<AuditEntry> _entries = [];
  final _scroll = ScrollController();
  final Set<int> _expanded = {};
  String? _cursor;
  bool _hasMore = true;
  bool _loading = false;
  bool _loadingMore = false;
  String? _error;
  _Filter _filter = _Filter.all;

  @override
  void initState() {
    super.initState();
    _scroll.addListener(_maybeLoadMore);
    _reload();
  }

  @override
  void dispose() {
    _scroll.dispose();
    super.dispose();
  }

  void _maybeLoadMore() {
    if (!_hasMore || _loadingMore || _loading) return;
    if (_scroll.position.pixels >
        _scroll.position.maxScrollExtent - 200) {
      _loadMore();
    }
  }

  Future<void> _reload() async {
    setState(() {
      _entries.clear();
      _expanded.clear();
      _cursor = null;
      _hasMore = true;
      _loading = true;
      _error = null;
    });
    await _fetchPage(reset: true);
    if (mounted) setState(() => _loading = false);
  }

  Future<void> _loadMore() async {
    if (!_hasMore) return;
    setState(() => _loadingMore = true);
    await _fetchPage(reset: false);
    if (mounted) setState(() => _loadingMore = false);
  }

  Future<void> _fetchPage({required bool reset}) async {
    try {
      final page = await ref.read(auditApiProvider).log(
            subjectKind: _filter.subjectKind,
            cursor: reset ? null : _cursor,
            limit: 100,
          );
      if (!mounted) return;
      setState(() {
        if (reset) _entries.clear();
        _entries.addAll(page.entries);
        _cursor = page.nextCursor;
        _hasMore = page.nextCursor != null;
      });
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } on Object catch (e) {
      if (mounted) setState(() => _error = e.toString());
    }
  }

  void _onFilter(_Filter f) {
    if (f == _filter) return;
    setState(() => _filter = f);
    _reload();
  }

  void _toggleExpanded(int id) {
    setState(() {
      if (_expanded.contains(id)) {
        _expanded.remove(id);
      } else {
        _expanded.add(id);
      }
    });
  }

  Future<void> _copyMetadata(AuditEntry e) async {
    if (e.metadata == null) return;
    await Clipboard.setData(
      ClipboardData(text: const JsonEncoder.withIndent('  ').convert(e.metadata)),
    );
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      const SnackBar(
        content: Text('Metadata copied'),
        duration: Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Activity'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _loading ? null : _reload,
          ),
        ],
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(48),
          child: _FilterStrip(value: _filter, onChanged: _onFilter),
        ),
      ),
      body: _body(),
    );
  }

  Widget _body() {
    if (_loading && _entries.isEmpty) {
      return const Center(child: CircularProgressIndicator());
    }
    if (_error != null && _entries.isEmpty) {
      return _ErrorView(error: _error!, onRetry: _reload);
    }
    if (_entries.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            _filter == _Filter.all
                ? 'No activity yet'
                : 'No ${_filter.label.toLowerCase()} activity yet',
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _reload,
      child: ListView.separated(
        controller: _scroll,
        itemCount: _entries.length + (_hasMore ? 1 : 0),
        separatorBuilder: (_, __) => Divider(
          height: 1,
          color: Theme.of(context).dividerColor,
        ),
        itemBuilder: (_, i) {
          if (i >= _entries.length) {
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
          final e = _entries[i];
          return _EntryTile(
            entry: e,
            expanded: _expanded.contains(e.id),
            onTap: () => _toggleExpanded(e.id),
            onCopyMetadata: () => _copyMetadata(e),
          );
        },
      ),
    );
  }
}

enum _Filter {
  all('All', null),
  session('Session', 'session'),
  integration('Integration', 'integration'),
  channel('Channel', 'channel'),
  admin('Admin', 'admin');

  const _Filter(this.label, this.subjectKind);
  final String label;
  // null = no subject_kind filter (show every kind).
  final String? subjectKind;
}

class _FilterStrip extends StatelessWidget {
  const _FilterStrip({required this.value, required this.onChanged});
  final _Filter value;
  final ValueChanged<_Filter> onChanged;

  @override
  Widget build(BuildContext context) {
    return SizedBox(
      height: 48,
      child: ListView.separated(
        scrollDirection: Axis.horizontal,
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
        itemCount: _Filter.values.length,
        separatorBuilder: (_, __) => const SizedBox(width: 6),
        itemBuilder: (_, i) {
          final f = _Filter.values[i];
          return ChoiceChip(
            label: Text(f.label),
            selected: f == value,
            onSelected: (_) => onChanged(f),
          );
        },
      ),
    );
  }
}

class _EntryTile extends StatelessWidget {
  const _EntryTile({
    required this.entry,
    required this.expanded,
    required this.onTap,
    required this.onCopyMetadata,
  });

  final AuditEntry entry;
  final bool expanded;
  final VoidCallback onTap;
  final VoidCallback onCopyMetadata;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final hasMeta = entry.metadata != null && entry.metadata!.isNotEmpty;
    return InkWell(
      onTap: hasMeta ? onTap : null,
      onLongPress: hasMeta ? onCopyMetadata : null,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(14, 10, 14, 10),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                _ActionBadge(action: entry.action),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    entry.action,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                      fontWeight: FontWeight.w600,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                Text(_relTime(entry.timestamp), style: muted),
              ],
            ),
            const SizedBox(height: 4),
            DefaultTextStyle.merge(
              style: muted ?? const TextStyle(),
              child: Wrap(
                spacing: 6,
                runSpacing: 2,
                children: [
                  Text('${entry.actorKind}'
                      '${entry.actorId != null && entry.actorId!.isNotEmpty ? ' · ${entry.actorId}' : ''}'),
                  if (entry.subjectKind != null && entry.subjectKind!.isNotEmpty)
                    Text(
                      '→ ${entry.subjectKind}'
                      '${entry.subjectId != null && entry.subjectId!.isNotEmpty ? ' · ${entry.subjectId}' : ''}',
                    ),
                  Text(
                    DateFormat.yMMMd().add_Hms().format(
                          entry.timestamp.toLocal(),
                        ),
                  ),
                ],
              ),
            ),
            if (expanded && hasMeta) ...[
              const SizedBox(height: 8),
              Container(
                width: double.infinity,
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: Theme.of(context)
                      .dividerColor
                      .withValues(alpha: 0.18),
                  borderRadius: BorderRadius.circular(6),
                ),
                child: SelectableText(
                  const JsonEncoder.withIndent('  ').convert(entry.metadata),
                  style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 11,
                    height: 1.4,
                  ),
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  static String _relTime(DateTime ts) {
    final diff = DateTime.now().toUtc().difference(ts.toUtc());
    if (diff.inSeconds < 60) return '${diff.inSeconds}s';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m';
    if (diff.inHours < 24) return '${diff.inHours}h';
    if (diff.inDays < 7) return '${diff.inDays}d';
    return DateFormat.yMMMd().format(ts.toLocal());
  }
}

class _ActionBadge extends StatelessWidget {
  const _ActionBadge({required this.action});
  final String action;

  @override
  Widget build(BuildContext context) {
    final color = _colorFor(action);
    return Container(
      width: 4,
      height: 28,
      decoration: BoxDecoration(
        color: color,
        borderRadius: BorderRadius.circular(2),
      ),
    );
  }

  Color _colorFor(String action) {
    if (action.startsWith('session.')) return Colors.greenAccent;
    if (action.startsWith('integration.')) return Colors.blueAccent;
    if (action.startsWith('channel.')) return Colors.amberAccent;
    if (action.startsWith('admin.')) return Colors.purpleAccent;
    if (action.contains('.fail') || action.contains('.error')) {
      return Colors.redAccent;
    }
    return Colors.grey;
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
              'Failed to load activity',
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
