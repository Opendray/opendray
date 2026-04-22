import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../core/services/ws_connect.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/default_path_banner.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

/// Panel page for the log-viewer plugin.
///
/// Two-phase UI:
///   1. Browse — show allowed roots → directory tree → pick a file
///   2. Tail — once a file is selected, connect a WebSocket to stream it;
///      supports grep (regex, server-side), auto-scroll, pause, clear.
class LogsPage extends StatefulWidget {
  const LogsPage({super.key});
  @override
  State<LogsPage> createState() => _LogsPageState();
}

class _LogsPageState extends State<LogsPage> {
  List<ProviderInfo> _plugins = [];
  String? _activePlugin;

  // Browse state
  String _currentPath = '';
  List<String> _pathStack = [];
  List<Map<String, dynamic>> _entries = [];
  bool _loadingList = false;
  String? _listError;

  // Tail state
  String? _tailPath;
  final List<_LogLine> _lines = [];
  final ScrollController _scroll = ScrollController();
  WebSocketChannel? _ws;
  StreamSubscription? _wsSub;
  bool _paused = false;
  bool _autoScroll = true;
  bool _connecting = false;
  String? _tailError;
  static const int _maxLines = 5000;

  // Filter
  final TextEditingController _grepCtrl = TextEditingController();
  String _grep = '';
  Timer? _grepDebounce;

  StreamSubscription<void>? _providersSub;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _loadPlugins();
    _providersSub =
        ProvidersBus.instance.changes.listen((_) => _loadPlugins());
  }

  @override
  void dispose() {
    _providersSub?.cancel();
    _disconnectTail();
    _grepDebounce?.cancel();
    _grepCtrl.dispose();
    _scroll.dispose();
    super.dispose();
  }

  // ── Plugins ────────────────────────────────────────────────

  Future<void> _loadPlugins() async {
    try {
      final all = await _api.listProviders();
      final found = all
          .where((p) =>
              p.provider.type == 'panel' &&
              p.provider.category == 'logs' &&
              p.enabled)
          .toList();
      if (!mounted) return;
      final stillActive = _activePlugin != null &&
          found.any((p) => p.provider.name == _activePlugin);
      setState(() {
        _plugins = found;
        if (!stillActive) {
          _disconnectTail();
          _activePlugin = null;
          _currentPath = '';
          _pathStack = [];
          _entries = [];
          _tailPath = null;
          _lines.clear();
        }
      });
      if (found.isNotEmpty && _activePlugin == null) {
        _activePlugin = found.first.provider.name;
        _loadList('');
      }
    } catch (e) {
      if (mounted) setState(() => _listError = e.toString());
    }
  }

  // ── Browse ─────────────────────────────────────────────────

  Future<void> _loadList(String path) async {
    if (_activePlugin == null) return;
    setState(() {
      _loadingList = true;
      _listError = null;
    });
    try {
      final out = await _api.logsList(_activePlugin!, path: path);
      if (!mounted) return;
      setState(() {
        _entries = out;
        _currentPath = path;
        _loadingList = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _listError = e.toString();
        _loadingList = false;
      });
    }
  }

  void _enterDir(Map<String, dynamic> e) {
    final p = e['path'] as String? ?? '';
    if (p.isEmpty) return;
    _pathStack.add(_currentPath);
    _loadList(p);
  }

  void _up() {
    if (_pathStack.isEmpty) return;
    final prev = _pathStack.removeLast();
    _loadList(prev);
  }

  // ── Tail ───────────────────────────────────────────────────

  void _connectTail(String path) {
    _disconnectTail();
    setState(() {
      _tailPath = path;
      _lines.clear();
      _paused = false;
      _tailError = null;
      _connecting = true;
    });
    final uri = _api.logsTailWsUri(_activePlugin!, path, grep: _grep);
    final ch = connectWs(uri);
    _ws = ch;
    _wsSub = ch.stream.listen(
      (data) {
        if (_paused) return;
        final line = data is String ? data : data.toString();
        _appendLine(line);
      },
      onError: (e) {
        if (!mounted) return;
        setState(() {
          _tailError = e.toString();
          _connecting = false;
        });
      },
      onDone: () {
        if (!mounted) return;
        setState(() => _connecting = false);
      },
    );
    // First text frame arrives almost immediately — mark as connected once we
    // see any data OR after a short delay that proves the socket accepted us.
    Future.delayed(const Duration(milliseconds: 400), () {
      if (mounted && _connecting) setState(() => _connecting = false);
    });
  }

  void _disconnectTail() {
    _wsSub?.cancel();
    _wsSub = null;
    _ws?.sink.close();
    _ws = null;
  }

  void _appendLine(String raw) {
    final l = _LogLine(text: raw, level: _detectLevel(raw));
    setState(() {
      _lines.add(l);
      if (_lines.length > _maxLines) {
        _lines.removeRange(0, _lines.length - _maxLines);
      }
    });
    if (_autoScroll) {
      WidgetsBinding.instance.addPostFrameCallback((_) {
        if (_scroll.hasClients) {
          _scroll.jumpTo(_scroll.position.maxScrollExtent);
        }
      });
    }
  }

  _Level _detectLevel(String line) {
    final l = line.toUpperCase();
    if (l.contains('FATAL') || l.contains('PANIC')) return _Level.fatal;
    if (l.contains('ERROR') || l.contains(' ERR ')) return _Level.error;
    if (l.contains('WARN'))  return _Level.warn;
    if (l.contains('INFO'))  return _Level.info;
    if (l.contains('DEBUG')) return _Level.debug;
    if (l.contains('TRACE')) return _Level.trace;
    return _Level.plain;
  }

  void _onGrepChanged(String v) {
    _grepDebounce?.cancel();
    _grepDebounce = Timer(const Duration(milliseconds: 450), () {
      if (_grep == v) return;
      _grep = v;
      // Re-open the tail stream with the new filter — server-side grep is
      // more efficient than filtering thousands of lines in Dart.
      if (_tailPath != null) _connectTail(_tailPath!);
    });
  }

  // ── Build ──────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    if (_plugins.isEmpty) return _buildNoPlugin();
    if (_tailPath != null) return _buildTailView();
    return _buildBrowseView();
  }

  Widget _buildNoPlugin() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.article_outlined,
              size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(context.tr('No log viewer configured'),
              style: const TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 8),
          Text(
            _listError ??
                context.tr('Enable Log Viewer in Settings → Plugins and set allowed directories.'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
            textAlign: TextAlign.center,
          ),
        ]),
      ),
    );
  }

  Widget _buildBrowseView() {
    final active = _activePlugin;
    final activeInfo = active == null
        ? null
        : _plugins.where((p) => p.provider.name == active).firstOrNull;
    return Column(children: [
      // First-run helper: manifest-default allowedRoots ⇒ show banner
      // so the user sees *where* we're tailing from and can narrow it.
      if (activeInfo != null)
        DefaultPathBanner(
          pluginName: activeInfo.provider.name,
          displayName: activeInfo.provider.displayName,
        ),
      // Breadcrumb + up
      if (_currentPath.isNotEmpty)
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          height: 40,
          child: Row(children: [
            IconButton(
              icon: const Icon(Icons.arrow_back, size: 18),
              padding: EdgeInsets.zero,
              constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              onPressed: _pathStack.isEmpty ? null : _up,
              tooltip: context.tr('Back'),
            ),
            const SizedBox(width: 4),
            Expanded(
              child: SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Text(_currentPath,
                    style: const TextStyle(
                        color: AppColors.textMuted,
                        fontSize: 11,
                        fontFamily: 'monospace')),
              ),
            ),
          ]),
        ),
      if (_listError != null)
        Container(
          margin: const EdgeInsets.all(12),
          padding: const EdgeInsets.all(10),
          decoration: BoxDecoration(
              color: AppColors.errorSoft,
              borderRadius: BorderRadius.circular(8)),
          child: Text(_listError!,
              style: const TextStyle(color: AppColors.error, fontSize: 12)),
        ),
      Expanded(
        child: _loadingList
            ? const Center(
                child: CircularProgressIndicator(color: AppColors.accent))
            : _entries.isEmpty
                ? Center(
                    child: Text(
                      _currentPath.isEmpty
                          ? context.tr('No allowed roots configured')
                          : context.tr('No log files here'),
                      style: const TextStyle(
                          color: AppColors.textMuted, fontSize: 12),
                    ),
                  )
                : RefreshIndicator(
                    onRefresh: () => _loadList(_currentPath),
                    child: ListView.separated(
                      padding: const EdgeInsets.symmetric(vertical: 4),
                      itemCount: _entries.length,
                      separatorBuilder: (_, _) =>
                          const Divider(height: 1, color: AppColors.border),
                      itemBuilder: (_, i) {
                        final e = _entries[i];
                        final isDir = e['type'] == 'dir';
                        final name = e['name'] as String? ?? '';
                        return ListTile(
                          dense: true,
                          leading: Icon(
                            isDir ? Icons.folder : _fileIcon(e['ext'] ?? ''),
                            size: 18,
                            color: isDir
                                ? AppColors.warning
                                : AppColors.accent,
                          ),
                          title: Text(name,
                              style: const TextStyle(fontSize: 13)),
                          subtitle: !isDir
                              ? Text(_formatSize(e['size'] ?? 0),
                                  style: const TextStyle(
                                      color: AppColors.textMuted, fontSize: 10))
                              : null,
                          trailing: !isDir
                              ? const Icon(Icons.play_arrow,
                                  size: 16, color: AppColors.accent)
                              : const Icon(Icons.chevron_right,
                                  size: 16, color: AppColors.textMuted),
                          onTap: () {
                            if (isDir) {
                              _enterDir(e);
                            } else {
                              _connectTail(e['path'] as String);
                            }
                          },
                        );
                      },
                    ),
                  ),
      ),
    ]);
  }

  Widget _buildTailView() {
    final path = _tailPath!;
    return Column(children: [
      // Toolbar
      Container(
        padding: const EdgeInsets.fromLTRB(6, 6, 6, 6),
        decoration: const BoxDecoration(
            border: Border(bottom: BorderSide(color: AppColors.border))),
        child: Row(children: [
          IconButton(
            icon: const Icon(Icons.arrow_back, size: 18),
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
            onPressed: () {
              _disconnectTail();
              setState(() {
                _tailPath = null;
                _lines.clear();
              });
            },
            tooltip: context.tr('Back'),
          ),
          Expanded(
            child: Tooltip(
              message: path,
              child: Text(
                path.split('/').last,
                style: const TextStyle(
                    fontSize: 12,
                    fontWeight: FontWeight.w600,
                    fontFamily: 'monospace'),
                maxLines: 1,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ),
          _toolBtn(
              _paused ? Icons.play_arrow : Icons.pause,
              context.tr(_paused ? 'Resume stream' : 'Pause'),
              _paused ? AppColors.accent : AppColors.textMuted,
              () => setState(() => _paused = !_paused)),
          _toolBtn(
              _autoScroll
                  ? Icons.vertical_align_bottom
                  : Icons.vertical_align_bottom_outlined,
              context.tr(_autoScroll ? 'Auto-scroll on' : 'Auto-scroll off'),
              _autoScroll ? AppColors.accent : AppColors.textMuted,
              () => setState(() => _autoScroll = !_autoScroll)),
          _toolBtn(Icons.clear_all, context.tr('Clear'),
              AppColors.textMuted, () => setState(_lines.clear)),
          _toolBtn(Icons.copy, context.tr('Copy all'),
              AppColors.textMuted, _copyAll),
        ]),
      ),
      // Grep bar
      Container(
        padding: const EdgeInsets.fromLTRB(8, 4, 8, 6),
        decoration: const BoxDecoration(
            border: Border(bottom: BorderSide(color: AppColors.border))),
        child: TextField(
          controller: _grepCtrl,
          style: const TextStyle(fontSize: 12, fontFamily: 'monospace'),
          autocorrect: false,
          enableSuggestions: false,
          textCapitalization: TextCapitalization.none,
          decoration: InputDecoration(
            isDense: true,
            contentPadding:
                const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
            prefixIcon:
                const Icon(Icons.filter_alt, size: 16, color: AppColors.textMuted),
            hintText: context.tr('Filter lines (regex) — e.g. ERROR|WARN'),
            hintStyle: const TextStyle(fontSize: 12, color: AppColors.textMuted),
            filled: true,
            fillColor: AppColors.surfaceAlt,
            border: OutlineInputBorder(
                borderRadius: BorderRadius.circular(8),
                borderSide: BorderSide.none),
            suffixIcon: _grepCtrl.text.isEmpty
                ? null
                : IconButton(
                    icon: const Icon(Icons.clear, size: 14),
                    onPressed: () {
                      _grepCtrl.clear();
                      _onGrepChanged('');
                    },
                  ),
          ),
          onChanged: _onGrepChanged,
        ),
      ),
      if (_connecting)
        const LinearProgressIndicator(
            color: AppColors.accent, minHeight: 2),
      if (_tailError != null)
        Container(
          width: double.infinity,
          padding: const EdgeInsets.all(8),
          color: AppColors.errorSoft,
          child: Text(_tailError!,
              style: const TextStyle(color: AppColors.error, fontSize: 11)),
        ),
      // Lines
      Expanded(
        child: Container(
          color: const Color(0xFF0B0D11),
          child: _lines.isEmpty && !_connecting
              ? Center(
                  child: Text(context.tr('Waiting for log lines…'),
                      style: const TextStyle(
                          color: AppColors.textMuted, fontSize: 12)),
                )
              : ListView.builder(
                  controller: _scroll,
                  padding: const EdgeInsets.symmetric(vertical: 4),
                  itemCount: _lines.length,
                  itemBuilder: (_, i) => _LogLineTile(line: _lines[i]),
                ),
        ),
      ),
      // Status footer
      Container(
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 4),
        decoration: const BoxDecoration(
            border: Border(top: BorderSide(color: AppColors.border))),
        child: Row(children: [
          Icon(
            _ws != null ? Icons.circle : Icons.circle_outlined,
            size: 8,
            color: _ws != null ? AppColors.success : AppColors.textMuted,
          ),
          const SizedBox(width: 6),
          Text('${_lines.length} ${context.tr('lines')}',
              style: const TextStyle(color: AppColors.textMuted, fontSize: 10)),
          const Spacer(),
          if (_paused)
            Text(context.tr('paused'),
                style: const TextStyle(color: AppColors.warning, fontSize: 10)),
        ]),
      ),
    ]);
  }

  Widget _toolBtn(IconData icon, String tooltip, Color color,
          VoidCallback onPressed) =>
      IconButton(
        icon: Icon(icon, size: 18, color: color),
        padding: EdgeInsets.zero,
        constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
        tooltip: tooltip,
        onPressed: onPressed,
      );

  Future<void> _copyAll() async {
    final all = _lines.map((l) => l.text).join('\n');
    await Clipboard.setData(ClipboardData(text: all));
    if (mounted) {
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text(context.tr('Copied')),
        duration: const Duration(seconds: 1),
      ));
    }
  }

  IconData _fileIcon(String ext) {
    return switch (ext.toLowerCase()) {
      'log' || 'out' || 'err' => Icons.article,
      'txt' => Icons.description,
      'json' => Icons.data_object,
      _ => Icons.insert_drive_file,
    };
  }

  String _formatSize(dynamic size) {
    final bytes = size is int ? size : (size is double ? size.toInt() : 0);
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }
}

enum _Level { fatal, error, warn, info, debug, trace, plain }

class _LogLine {
  final String text;
  final _Level level;
  _LogLine({required this.text, required this.level});
}

class _LogLineTile extends StatelessWidget {
  final _LogLine line;
  const _LogLineTile({required this.line});

  @override
  Widget build(BuildContext context) {
    Color c;
    switch (line.level) {
      case _Level.fatal: c = const Color(0xFFFF5370); break;
      case _Level.error: c = const Color(0xFFF7768E); break;
      case _Level.warn:  c = const Color(0xFFE0AF68); break;
      case _Level.info:  c = const Color(0xFF7AA2F7); break;
      case _Level.debug: c = const Color(0xFF9ECE6A); break;
      case _Level.trace: c = const Color(0xFF7DCFFF); break;
      case _Level.plain: c = const Color(0xFFC0CAF5); break;
    }
    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 1),
      child: SelectableText(
        line.text,
        style: TextStyle(
          fontFamily: 'monospace',
          fontSize: 11,
          height: 1.35,
          color: c,
        ),
      ),
    );
  }
}
