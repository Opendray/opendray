// LogViewerScreen — live tail of /admin/logs/stream, the same
// in-process ring buffer the web admin's Settings → Logging
// section reads from. Without this, the mobile app's Logging
// settings section has nothing to point at — you can tune the
// level but never read the output.
//
// Behaviour mirrors the web LogViewer:
//
//   * WS connects on mount, replays the ring buffer once, then
//     pushes every new record.
//   * Auto-scroll to bottom unless the operator pauses or has
//     scrolled away.
//   * Filter input does a substring match against the rendered
//     `text` field — same heuristic as the web client.
//   * Hard cap of 2 000 records kept in memory (mobile is more
//     memory-constrained than the browser's 5 000); older records
//     drop from the head when full. Server's ring stays the source
//     of truth.
//
// No download button — Flutter's file sharing surface is
// platform-specific and overkill for the rare "I want a .log file
// on my phone" workflow. Operators who need that pull it from the
// web admin's Download Logs button.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/logs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

const _maxBuffered = 2000;

class LogViewerScreen extends ConsumerStatefulWidget {
  const LogViewerScreen({super.key});

  @override
  ConsumerState<LogViewerScreen> createState() => _LogViewerScreenState();
}

class _LogViewerScreenState extends ConsumerState<LogViewerScreen> {
  LogsStream? _stream;
  StreamSubscription<LogRecord>? _sub;
  final _scroll = ScrollController();
  final _filterCtrl = TextEditingController();

  final List<LogRecord> _records = [];
  bool _paused = false;
  bool _connected = false;
  String? _connectError;
  String _filter = '';
  String _levelFilter = 'ALL';

  @override
  void initState() {
    super.initState();
    _connect();
  }

  @override
  void dispose() {
    _sub?.cancel();
    _stream?.close();
    _scroll.dispose();
    _filterCtrl.dispose();
    super.dispose();
  }

  void _connect() {
    final stream = openLogsStream(ref);
    if (stream == null) {
      setState(() {
        _connectError = 'Not signed in.';
        _connected = false;
      });
      return;
    }
    _stream = stream;
    _sub = stream.stream.listen(
      _onRecord,
      onError: (Object err) {
        if (!mounted) return;
        setState(() {
          _connected = false;
          _connectError = err.toString();
        });
      },
      onDone: () {
        if (!mounted) return;
        setState(() => _connected = false);
      },
    );
    setState(() {
      _connected = true;
      _connectError = null;
    });
  }

  void _onRecord(LogRecord rec) {
    if (!mounted) return;
    setState(() {
      _records.add(rec);
      if (_records.length > _maxBuffered) {
        _records.removeRange(0, _records.length - _maxBuffered);
      }
    });
    if (!_paused) _scheduleScroll();
  }

  // Defer scroll-to-bottom to the next frame so the ListView has
  // already laid out the new row. Without this, the scroll target
  // is the pre-append max extent and we land one row short.
  void _scheduleScroll() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (!mounted || !_scroll.hasClients) return;
      final pos = _scroll.position;
      final nearBottom =
          pos.maxScrollExtent - pos.pixels < 200;
      if (nearBottom) {
        _scroll.jumpTo(pos.maxScrollExtent);
      }
    });
  }

  Future<void> _reconnect() async {
    await _sub?.cancel();
    await _stream?.close();
    if (!mounted) return;
    setState(_records.clear);
    _connect();
  }

  void _copyAll() {
    if (_records.isEmpty) return;
    final body = _records.map((r) {
      if (r.text.isNotEmpty) return r.text;
      return '${r.time} ${r.level} ${r.message}';
    }).join('\n');
    Clipboard.setData(ClipboardData(text: body));
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(t.settings.logViewer.copiedSnack),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final filtered = _filteredRecords();
    return Scaffold(
      appBar: AppBar(
        title: Text(t.settings.logViewer.title),
        actions: [
          IconButton(
            tooltip: _paused ? 'Resume auto-scroll' : 'Pause auto-scroll',
            icon: Icon(_paused
                ? Icons.play_circle_outline
                : Icons.pause_circle_outline),
            onPressed: () => setState(() => _paused = !_paused),
          ),
          IconButton(
            tooltip: t.settings.logViewer.reconnect,
            icon: const Icon(Icons.refresh),
            onPressed: _reconnect,
          ),
          IconButton(
            tooltip: t.settings.logViewer.copyBuffer,
            icon: const Icon(Icons.copy_outlined),
            onPressed: _records.isEmpty ? null : _copyAll,
          ),
          IconButton(
            tooltip: t.settings.logViewer.clearLocal,
            icon: const Icon(Icons.delete_outline),
            onPressed: _records.isEmpty
                ? null
                : () => setState(_records.clear),
          ),
        ],
      ),
      body: Column(
        children: [
          _Header(
            connected: _connected,
            error: _connectError,
            paused: _paused,
            recordCount: _records.length,
            filteredCount: filtered.length,
          ),
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 6, 12, 8),
            child: Row(
              children: [
                Expanded(
                  child: TextField(
                    controller: _filterCtrl,
                    onChanged: (v) => setState(() => _filter = v),
                    decoration: InputDecoration(
                      hintText: t.settings.logViewer.filterHint,
                      prefixIcon: const Icon(Icons.search, size: 18),
                      isDense: true,
                    ),
                  ),
                ),
                const SizedBox(width: 8),
                DropdownButton<String>(
                  value: _levelFilter,
                  onChanged: (v) {
                    if (v != null) setState(() => _levelFilter = v);
                  },
                  items: [
                    DropdownMenuItem(
                        value: 'ALL',
                        child: Text(t.settings.logViewer.levels.all)),
                    DropdownMenuItem(
                        value: 'DEBUG',
                        child: Text(t.settings.logViewer.levels.debug)),
                    DropdownMenuItem(
                        value: 'INFO',
                        child: Text(t.settings.logViewer.levels.info)),
                    DropdownMenuItem(
                        value: 'WARN',
                        child: Text(t.settings.logViewer.levels.warn)),
                    DropdownMenuItem(
                        value: 'ERROR',
                        child: Text(t.settings.logViewer.levels.error)),
                  ],
                ),
              ],
            ),
          ),
          const Divider(height: 1),
          Expanded(
            child: filtered.isEmpty
                ? Center(
                    child: Padding(
                      padding: const EdgeInsets.all(24),
                      child: Text(
                        _records.isEmpty
                            ? 'Waiting for log records…'
                            : 'No records match the current filter.',
                        textAlign: TextAlign.center,
                        style: Theme.of(context).textTheme.bodyMedium,
                      ),
                    ),
                  )
                : ListView.builder(
                    controller: _scroll,
                    itemCount: filtered.length,
                    itemBuilder: (_, i) => _LogRow(record: filtered[i]),
                  ),
          ),
        ],
      ),
    );
  }

  List<LogRecord> _filteredRecords() {
    if (_filter.isEmpty && _levelFilter == 'ALL') return _records;
    final q = _filter.toLowerCase();
    return _records.where((r) {
      if (_levelFilter != 'ALL' && r.level != _levelFilter) return false;
      if (q.isEmpty) return true;
      final hay = r.text.isNotEmpty
          ? r.text.toLowerCase()
          : '${r.message} ${r.attrs ?? ''}'.toLowerCase();
      return hay.contains(q);
    }).toList();
  }
}

class _Header extends StatelessWidget {
  const _Header({
    required this.connected,
    required this.error,
    required this.paused,
    required this.recordCount,
    required this.filteredCount,
  });

  final bool connected;
  final String? error;
  final bool paused;
  final int recordCount;
  final int filteredCount;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final color = error != null
        ? theme.colorScheme.error
        : connected
            ? Colors.green
            : theme.colorScheme.outline;
    final label = error != null
        ? 'Disconnected — $error'
        : connected
            ? (paused ? 'Live (paused)' : 'Live')
            : 'Disconnected';
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
      color: color.withValues(alpha: 0.08),
      child: Row(
        children: [
          Container(
            width: 8,
            height: 8,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
            ),
          ),
          const SizedBox(width: 8),
          Expanded(
            child: Text(
              label,
              style: TextStyle(
                color: color,
                fontSize: 12,
                fontWeight: FontWeight.w600,
              ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          Text(
            '$filteredCount / $recordCount',
            style: theme.textTheme.bodySmall,
          ),
        ],
      ),
    );
  }
}

class _LogRow extends StatelessWidget {
  const _LogRow({required this.record});
  final LogRecord record;

  Color _levelColor(BuildContext ctx) {
    final scheme = Theme.of(ctx).colorScheme;
    return switch (record.level) {
      'ERROR' => scheme.error,
      'WARN' => Colors.amber.shade700,
      'INFO' => scheme.primary,
      'DEBUG' => scheme.outline,
      _ => scheme.outline,
    };
  }

  String _shortTime() {
    // Render only HH:MM:SS.mmm to save horizontal space — the
    // full ISO timestamp is rarely useful when you're tailing.
    if (record.time.length < 19) return record.time;
    // RFC3339: 2026-05-12T13:45:01.234567+09:00 → take HH:MM:SS.mmm
    final t = record.time;
    final idxT = t.indexOf('T');
    if (idxT < 0) return t;
    final rest = t.substring(idxT + 1);
    // Strip TZ offset (after + / Z); cap fractional seconds to 3.
    final endIdx = rest.indexOf(RegExp('[+Z-]'));
    var slice = endIdx >= 0 ? rest.substring(0, endIdx) : rest;
    final dot = slice.indexOf('.');
    if (dot > 0 && slice.length > dot + 4) {
      slice = slice.substring(0, dot + 4);
    }
    return slice;
  }

  String _attrSuffix() {
    final a = record.attrs;
    if (a == null || a.isEmpty) return '';
    final parts = a.entries.map((e) => '${e.key}=${e.value}').toList();
    return parts.join(' ');
  }

  @override
  Widget build(BuildContext context) {
    final levelColor = _levelColor(context);
    final theme = Theme.of(context);
    final msg = record.message.isNotEmpty
        ? record.message
        : record.text;
    final suffix = _attrSuffix();
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        border: Border(
          bottom: BorderSide(
            color: theme.dividerColor.withValues(alpha: 0.5),
          ),
        ),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 48,
            child: Text(
              record.level,
              style: TextStyle(
                color: levelColor,
                fontFamily: 'monospace',
                fontSize: 10,
                fontWeight: FontWeight.w700,
              ),
            ),
          ),
          const SizedBox(width: 6),
          SizedBox(
            width: 78,
            child: Text(
              _shortTime(),
              style: theme.textTheme.bodySmall?.copyWith(
                fontFamily: 'monospace',
                fontSize: 11,
              ),
            ),
          ),
          const SizedBox(width: 6),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                SelectableText(
                  msg,
                  style: const TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 12,
                    height: 1.3,
                  ),
                ),
                if (suffix.isNotEmpty)
                  Padding(
                    padding: const EdgeInsets.only(top: 2),
                    child: Text(
                      suffix,
                      style: theme.textTheme.bodySmall?.copyWith(
                        fontFamily: 'monospace',
                        fontSize: 10,
                      ),
                    ),
                  ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}
