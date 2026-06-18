import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/integrations_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:shared_preferences/shared_preferences.dart';

// ProxyConsoleView — the mobile mirror of the web ProxyConsole. Lets the
// operator fire a request at a registered integration through
// /api/v1/proxy/{prefix}/* and inspect the raw upstream response (status,
// headers, body, latency). Reuses the shared web.integrations.proxy.*
// strings so both surfaces read identically. Per-integration request
// history is kept in shared_preferences and is tap-to-replay.
class ProxyConsoleView extends ConsumerStatefulWidget {
  const ProxyConsoleView({required this.integrations, super.key});

  final List<Integration> integrations;

  @override
  ConsumerState<ProxyConsoleView> createState() => _ProxyConsoleViewState();
}

const _methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE'];

class _HistoryEntry {
  _HistoryEntry({
    required this.ts,
    required this.method,
    required this.path,
    required this.status,
    required this.durationMs,
  });

  factory _HistoryEntry.fromJson(Map<String, dynamic> j) => _HistoryEntry(
        ts: (j['ts'] as num?)?.toInt() ?? 0,
        method: j['method'] as String? ?? 'GET',
        path: j['path'] as String? ?? '/',
        status: (j['status'] as num?)?.toInt() ?? 0,
        durationMs: (j['durationMs'] as num?)?.toInt() ?? 0,
      );

  Map<String, dynamic> toJson() => {
        'ts': ts,
        'method': method,
        'path': path,
        'status': status,
        'durationMs': durationMs,
      };

  final int ts;
  final String method;
  final String path;
  final int status;
  final int durationMs;
}

class _ProxyConsoleViewState extends ConsumerState<ProxyConsoleView> {
  String? _selectedId;
  String _method = 'GET';
  final _path = TextEditingController(text: '/health');
  final _headers = TextEditingController();
  final _body = TextEditingController();

  ProxyResponse? _response;
  bool _sending = false;
  String? _error;
  List<_HistoryEntry> _history = const [];

  static String _historyKey(String id) => 'opendray.proxy-history.$id';

  @override
  void initState() {
    super.initState();
    if (widget.integrations.isNotEmpty) {
      _selectedId = widget.integrations.first.id;
      _loadHistory();
    }
  }

  @override
  void dispose() {
    _path.dispose();
    _headers.dispose();
    _body.dispose();
    super.dispose();
  }

  Integration? get _selected {
    for (final i in widget.integrations) {
      if (i.id == _selectedId) return i;
    }
    return null;
  }

  Future<void> _loadHistory() async {
    final id = _selectedId;
    if (id == null) {
      setState(() => _history = const []);
      return;
    }
    final prefs = await SharedPreferences.getInstance();
    final raw = prefs.getString(_historyKey(id));
    var list = const <_HistoryEntry>[];
    if (raw != null) {
      try {
        final decoded = jsonDecode(raw);
        if (decoded is List) {
          list = decoded
              .whereType<Map<String, dynamic>>()
              .map(_HistoryEntry.fromJson)
              .toList();
        }
      } on Object {
        // Corrupt history — start clean.
      }
    }
    if (mounted) setState(() => _history = list);
  }

  Future<void> _saveHistory(List<_HistoryEntry> list) async {
    final id = _selectedId;
    if (id == null) return;
    final prefs = await SharedPreferences.getInstance();
    final trimmed = list.take(30).toList();
    await prefs.setString(
      _historyKey(id),
      jsonEncode(trimmed.map((e) => e.toJson()).toList()),
    );
  }

  void _onSelect(String? id) {
    setState(() {
      _selectedId = id;
      _response = null;
      _error = null;
    });
    _loadHistory();
  }

  Map<String, String> _parseHeaders() {
    final out = <String, String>{};
    for (final line in _headers.text.split('\n')) {
      final idx = line.indexOf(':');
      if (idx <= 0) continue;
      final name = line.substring(0, idx).trim();
      final value = line.substring(idx + 1).trim();
      if (name.isNotEmpty && value.isNotEmpty) out[name] = value;
    }
    return out;
  }

  Future<void> _send() async {
    final target = _selected;
    if (target == null) return;
    setState(() {
      _sending = true;
      _error = null;
      _response = null;
    });
    try {
      final res = await ref.read(integrationsApiProvider).proxy(
            routePrefix: target.routePrefix,
            method: _method,
            path: _path.text,
            extraHeaders: _parseHeaders(),
            body: _body.text,
          );
      if (!mounted) return;
      setState(() => _response = res);
      final entry = _HistoryEntry(
        ts: DateTime.now().millisecondsSinceEpoch,
        method: _method,
        path: _path.text,
        status: res.status,
        durationMs: res.durationMs,
      );
      final next = [
        entry,
        ..._history
            .where((h) => !(h.method == entry.method && h.path == entry.path)),
      ];
      setState(() => _history = next);
      await _saveHistory(next);
    } on ApiException catch (e) {
      if (mounted) setState(() => _error = e.message);
    } on Object catch (e) {
      if (mounted) {
        setState(() => _error = '${t.web.integrations.proxy.requestFailed}: $e');
      }
    } finally {
      if (mounted) setState(() => _sending = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final proxy = t.web.integrations.proxy;
    if (widget.integrations.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(Icons.cable_outlined,
                  size: 40,
                  color: Theme.of(context)
                      .colorScheme
                      .onSurfaceVariant
                      .withValues(alpha: 0.5)),
              const SizedBox(height: 12),
              Text(proxy.emptyTitle,
                  style: Theme.of(context).textTheme.titleSmall),
              const SizedBox(height: 6),
              Text(
                // slang parses the literal "{prefix}" in the URL path as a
                // param (web uses {{ }} so it's literal there); pass it back.
                proxy.emptyDescription(prefix: '{prefix}'),
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          ),
        ),
      );
    }

    final selected = _selected;
    return ListView(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
      children: [
        // Target integration.
        DropdownButtonFormField<String>(
          initialValue: _selectedId,
          isExpanded: true,
          decoration: InputDecoration(
            labelText: proxy.targetLabel,
            border: const OutlineInputBorder(),
          ),
          items: [
            for (final i in widget.integrations)
              DropdownMenuItem(
                value: i.id,
                child: Text('${i.name} · /${i.routePrefix}',
                    overflow: TextOverflow.ellipsis),
              ),
          ],
          onChanged: _onSelect,
        ),
        if (selected != null && selected.baseUrl.isNotEmpty)
          Padding(
            padding: const EdgeInsets.only(top: 6),
            child: Text(
              '${proxy.baseLabel} ${selected.baseUrl}',
              style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
            ),
          ),
        const SizedBox(height: 16),

        // Method + path.
        Row(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SizedBox(
              width: 110,
              child: DropdownButtonFormField<String>(
                initialValue: _method,
                decoration: const InputDecoration(
                  border: OutlineInputBorder(),
                  isDense: true,
                ),
                items: [
                  for (final m in _methods)
                    DropdownMenuItem(
                      value: m,
                      child: Text(m,
                          style: const TextStyle(
                              fontFamily: 'monospace', fontSize: 13)),
                    ),
                ],
                onChanged: (v) => setState(() => _method = v ?? 'GET'),
              ),
            ),
            const SizedBox(width: 8),
            Expanded(
              child: TextField(
                controller: _path,
                autocorrect: false,
                style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
                decoration: InputDecoration(
                  isDense: true,
                  border: const OutlineInputBorder(),
                  prefixText: '/proxy/${selected?.routePrefix ?? ''}',
                  hintText: '/health',
                ),
              ),
            ),
          ],
        ),
        const SizedBox(height: 12),

        // Extra headers.
        TextField(
          controller: _headers,
          maxLines: 3,
          autocorrect: false,
          style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
          decoration: InputDecoration(
            labelText: proxy.extraHeadersLabel,
            hintText: 'X-Foo: bar',
            border: const OutlineInputBorder(),
            alignLabelWithHint: true,
          ),
        ),
        const SizedBox(height: 12),

        // Body (only for methods that carry one).
        TextField(
          controller: _body,
          maxLines: 4,
          enabled: _method != 'GET' && _method != 'DELETE',
          autocorrect: false,
          style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
          decoration: InputDecoration(
            labelText: proxy.bodyLabel,
            hintText: '{"hello":"world"}',
            border: const OutlineInputBorder(),
            alignLabelWithHint: true,
          ),
        ),
        const SizedBox(height: 12),

        FilledButton.icon(
          onPressed: (selected == null || _sending) ? null : _send,
          icon: _sending
              ? const SizedBox(
                  width: 16,
                  height: 16,
                  child: CircularProgressIndicator(strokeWidth: 2),
                )
              : const Icon(Icons.send, size: 16),
          label: Text(_sending ? proxy.sending : proxy.send),
        ),
        const SizedBox(height: 8),
        Text(
          proxy.stubText,
          style: Theme.of(context).textTheme.bodySmall?.copyWith(
                color: Theme.of(context).colorScheme.onSurfaceVariant,
              ),
        ),

        if (_error != null) ...[
          const SizedBox(height: 12),
          Container(
            width: double.infinity,
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.errorContainer,
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(
              _error!,
              style: TextStyle(
                color: Theme.of(context).colorScheme.onErrorContainer,
                fontSize: 12,
              ),
            ),
          ),
        ],

        if (_response != null) ...[
          const SizedBox(height: 16),
          _ResponseCard(response: _response!),
        ],

        // History.
        const SizedBox(height: 20),
        Row(
          children: [
            const Icon(Icons.history, size: 16),
            const SizedBox(width: 6),
            Text(proxy.history,
                style: Theme.of(context).textTheme.labelLarge),
          ],
        ),
        const SizedBox(height: 6),
        if (_history.isEmpty)
          Text(
            proxy.historyEmpty,
            style: Theme.of(context).textTheme.bodySmall?.copyWith(
                  fontStyle: FontStyle.italic,
                  color: Theme.of(context).colorScheme.onSurfaceVariant,
                ),
          )
        else
          for (final h in _history) _historyTile(h),
      ],
    );
  }

  Widget _historyTile(_HistoryEntry h) {
    final color = h.status >= 500
        ? Colors.redAccent
        : h.status >= 400
            ? Colors.orangeAccent
            : Colors.greenAccent;
    return InkWell(
      onTap: () => setState(() {
        _method = h.method;
        _path.text = h.path;
      }),
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Row(
          children: [
            SizedBox(
              width: 52,
              child: Text(
                h.method,
                style: TextStyle(
                    fontFamily: 'monospace', fontSize: 12, color: color),
              ),
            ),
            Expanded(
              child: Text(
                h.path,
                overflow: TextOverflow.ellipsis,
                style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
              ),
            ),
            Text(
              '${h.status}',
              style: TextStyle(
                  fontFamily: 'monospace', fontSize: 11, color: color),
            ),
          ],
        ),
      ),
    );
  }
}

class _ResponseCard extends StatelessWidget {
  const _ResponseCard({required this.response});
  final ProxyResponse response;

  @override
  Widget build(BuildContext context) {
    final proxy = t.web.integrations.proxy;
    final scheme = Theme.of(context).colorScheme;
    final r = response;
    final statusColor = r.status >= 500
        ? scheme.error
        : r.status >= 400
            ? Colors.orange
            : Colors.green;
    final headersText =
        r.headers.map((e) => '${e.key}: ${e.value}').join('\n');
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        border: Border.all(color: Theme.of(context).dividerColor),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Wrap(
            spacing: 8,
            crossAxisAlignment: WrapCrossAlignment.center,
            children: [
              Container(
                padding:
                    const EdgeInsets.symmetric(horizontal: 8, vertical: 2),
                decoration: BoxDecoration(
                  color: statusColor.withValues(alpha: 0.15),
                  borderRadius: BorderRadius.circular(4),
                ),
                child: Text(
                  '${r.status}',
                  style: TextStyle(
                    color: statusColor,
                    fontWeight: FontWeight.w700,
                    fontSize: 12,
                  ),
                ),
              ),
              Text('${r.durationMs} ms',
                  style: const TextStyle(
                      fontFamily: 'monospace', fontSize: 11)),
              if (r.contentType != null)
                Text(
                  r.contentType!,
                  style: TextStyle(
                    fontFamily: 'monospace',
                    fontSize: 11,
                    color: scheme.onSurfaceVariant,
                  ),
                ),
            ],
          ),
          if (headersText.isNotEmpty) ...[
            const SizedBox(height: 12),
            _label(context, proxy.headers),
            const SizedBox(height: 4),
            _codeBlock(context, headersText),
          ],
          const SizedBox(height: 12),
          _label(context, proxy.body),
          const SizedBox(height: 4),
          _codeBlock(
            context,
            r.body.isEmpty ? proxy.emptyBody : r.body,
            copyable: r.body.isNotEmpty,
          ),
        ],
      ),
    );
  }

  Widget _label(BuildContext context, String text) => Text(
        text.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              letterSpacing: 0.6,
              color: Theme.of(context).colorScheme.onSurfaceVariant,
            ),
      );

  Widget _codeBlock(BuildContext context, String text,
      {bool copyable = false}) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: scheme.surfaceContainerHighest.withValues(alpha: 0.5),
        borderRadius: BorderRadius.circular(6),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Expanded(
            child: SelectableText(
              text,
              style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
            ),
          ),
          if (copyable)
            IconButton(
              visualDensity: VisualDensity.compact,
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(),
              icon: const Icon(Icons.copy_outlined, size: 16),
              onPressed: () =>
                  Clipboard.setData(ClipboardData(text: text)),
            ),
        ],
      ),
    );
  }
}
