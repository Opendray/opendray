import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:go_router/go_router.dart';
import 'package:provider/provider.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

import '../../core/services/ws_connect.dart';

import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../shared/default_path_banner.dart';
import '../../shared/directory_picker.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

const _recentPathsKey = 'tasks.recentPaths';
const _maxRecent = 8;
const _maxOutputBytes = 1024 * 1024; // cap at 1 MB to keep memory bounded
final _ansiRe = RegExp(r'\x1B\[[0-9;?]*[A-Za-z]');

/// Tasks panel — mobile-first, single-column.
///
/// Three states:
///   1. No plugin enabled → setup CTA pointing at Providers page.
///   2. Plugin enabled, no path yet → big "pick folder" CTA.
///   3. Path chosen → grouped task list. Running a task opens the output
///      as a draggable bottom sheet, so the task list stays visible.
class TasksPage extends StatefulWidget {
  const TasksPage({super.key});
  @override
  State<TasksPage> createState() => _TasksPageState();
}

class _TasksPageState extends State<TasksPage> {
  // Plugin state
  ProviderInfo? _plugin;
  StreamSubscription<void>? _providersSub;

  // Path state
  String _path = '';
  String? _defaultPath;
  List<String> _recentPaths = [];

  // Task list
  List<Map<String, dynamic>> _tasks = [];
  bool _loading = false;
  String? _error;

  // Run state (at most one at a time in this panel)
  _RunSession? _run;

  ApiClient get _api => context.read<ApiClient>();
  String? get _pluginName => _plugin?.provider.name;

  // ─── Lifecycle ────────────────────────────────────────────

  @override
  void initState() {
    super.initState();
    _loadRecents();
    _loadPlugin();
    _providersSub =
        ProvidersBus.instance.changes.listen((_) => _loadPlugin());
  }

  @override
  void dispose() {
    _providersSub?.cancel();
    _run?.removeListener(_onRunTick);
    _run?.dispose();
    super.dispose();
  }

  void _onRunTick() {
    if (mounted) setState(() {});
  }

  // ─── Plugin / path ────────────────────────────────────────

  Future<void> _loadPlugin() async {
    try {
      final all = await _api.listProviders();
      final match = all
          .where((p) =>
              p.provider.type == 'panel' &&
              p.provider.name == 'task-runner' &&
              p.enabled)
          .toList();
      if (!mounted) return;
      if (match.isEmpty) {
        setState(() {
          _plugin = null;
          _tasks = [];
          _path = '';
          _defaultPath = null;
          _error = null;
        });
        return;
      }
      final next = match.first;
      final dp = (next.config['defaultPath'] as String? ?? '').trim();
      final pluginChanged = _plugin?.provider.name != next.provider.name;
      setState(() {
        _plugin = next;
        _defaultPath = dp.isEmpty ? null : dp;
        if (pluginChanged) {
          _path = dp;
          _tasks = [];
          _error = null;
        }
      });
      if (_path.isNotEmpty) _loadTasks();
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    }
  }

  Future<void> _loadRecents() async {
    final prefs = await SharedPreferences.getInstance();
    if (!mounted) return;
    setState(() =>
        _recentPaths = prefs.getStringList(_recentPathsKey) ?? const []);
  }

  Future<void> _rememberPath(String p) async {
    if (p.isEmpty) return;
    final next = [p, ..._recentPaths.where((x) => x != p)];
    if (next.length > _maxRecent) next.removeRange(_maxRecent, next.length);
    final prefs = await SharedPreferences.getInstance();
    await prefs.setStringList(_recentPathsKey, next);
    if (!mounted) return;
    setState(() => _recentPaths = next);
  }

  Future<void> _clearRecents() async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.remove(_recentPathsKey);
    if (!mounted) return;
    setState(() => _recentPaths = []);
  }

  void _setPath(String p) {
    if (p.trim().isEmpty) return;
    setState(() {
      _path = p.trim();
      _error = null;
    });
    _rememberPath(_path);
    _loadTasks();
  }

  Future<void> _browse() async {
    final picked = await pickDirectory(context,
        initialPath: _path.isNotEmpty ? _path : _defaultPath);
    if (picked != null) _setPath(picked);
  }

  Future<void> _typePath() async {
    final ctrl = TextEditingController(text: _path);
    final entered = await showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: AppColors.surface,
        title: const Text('Project path', style: TextStyle(fontSize: 15)),
        content: TextField(
          controller: ctrl,
          autofocus: true,
          autocorrect: false,
          enableSuggestions: false,
          decoration: const InputDecoration(
            hintText: '/Users/you/Projects/foo',
            labelText: 'Absolute path',
          ),
          style: const TextStyle(fontSize: 13, fontFamily: 'monospace'),
          onSubmitted: (v) => Navigator.pop(ctx, v),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx),
              child: const Text('Cancel')),
          FilledButton(
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            onPressed: () => Navigator.pop(ctx, ctrl.text),
            child: const Text('Use'),
          ),
        ],
      ),
    );
    if (entered != null) _setPath(entered);
  }

  // ─── Tasks API ────────────────────────────────────────────

  Future<void> _loadTasks() async {
    final plugin = _pluginName;
    if (plugin == null) return;
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final out = await ApiClient.describeErrors(
          () => _api.tasksList(plugin, path: _path));
      if (!mounted) return;
      setState(() {
        _tasks = out;
        _loading = false;
      });
    } on ApiException catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _tasks = [];
        _error = e.message;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _tasks = [];
        _error = e.toString();
      });
    }
  }

  Future<void> _runTask(Map<String, dynamic> task) async {
    final plugin = _pluginName;
    if (plugin == null) return;
    _run?.removeListener(_onRunTick);
    _run?.dispose();
    try {
      final meta = await ApiClient.describeErrors(() =>
          _api.tasksRun(plugin, task['id'] as String, path: _path));
      if (!mounted) return;
      final session = _RunSession(
        api: _api,
        plugin: plugin,
        runId: meta['id'] as String,
        meta: meta,
      );
      session.addListener(_onRunTick);
      setState(() => _run = session);
      session.start();
      _showRunSheet();
    } on ApiException catch (e) {
      _showSnack('Start failed: ${e.message}', error: true);
    } catch (e) {
      _showSnack('Start failed: $e', error: true);
    }
  }

  void _showSnack(String msg, {bool error = false}) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(msg),
      backgroundColor: error ? AppColors.error : null,
      duration: const Duration(seconds: 3),
    ));
  }

  // ─── Run output sheet ─────────────────────────────────────

  Future<void> _showRunSheet() async {
    final run = _run;
    if (run == null) return;
    await showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      backgroundColor: const Color(0xFF0E1116),
      barrierColor: Colors.black54,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => _RunSheet(session: run),
    );
  }

  // ─── UI ───────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final body = switch (true) {
      _ when _plugin == null => _setupState(),
      _ when _path.isEmpty => _pickPathState(),
      _ => _taskListState(),
    };
    final pluginInfo = _plugin;
    final content = pluginInfo == null
        ? body
        : Column(children: [
            // First-run helper: banner shows where task discovery is
            // pointed when the user hasn't configured allowedRoots yet.
            DefaultPathBanner(
              pluginName: pluginInfo.provider.name,
              displayName: pluginInfo.provider.displayName,
            ),
            Expanded(child: body),
          ]);
    return Stack(children: [
      Positioned.fill(child: content),
      if (_run != null)
        Positioned(
          left: 12,
          right: 12,
          bottom: 12,
          child: _runBadge(_run!),
        ),
    ]);
  }

  // State 1 — plugin not enabled
  Widget _setupState() {
    return _centeredCard(
      icon: Icons.extension_off,
      title: 'Task Runner is not enabled',
      subtitle:
          'Enable the Task Runner plugin and set Allowed Directories in Settings.',
      primary: FilledButton.icon(
        onPressed: () => context.go('/settings'),
        style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
        icon: const Icon(Icons.settings, size: 16),
        label: const Text('Open Settings'),
      ),
    );
  }

  // State 2 — need a path
  Widget _pickPathState() {
    return LayoutBuilder(builder: (context, constraints) {
      return SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: ConstrainedBox(
          constraints: BoxConstraints(minHeight: constraints.maxHeight - 40),
          child: Column(
            mainAxisAlignment: MainAxisAlignment.center,
            children: [
              const Icon(Icons.play_circle_outline,
                  size: 48, color: AppColors.accent),
              const SizedBox(height: 12),
              const Text('Pick a project',
                  style: TextStyle(
                      fontSize: 16, fontWeight: FontWeight.w600)),
              const SizedBox(height: 6),
              const Text(
                'Task Runner discovers Makefile targets, package.json scripts, and *.sh files in the chosen directory.',
                textAlign: TextAlign.center,
                style:
                    TextStyle(color: AppColors.textMuted, fontSize: 12),
              ),
              const SizedBox(height: 24),
              SizedBox(
                width: 240,
                child: FilledButton.icon(
                  onPressed: _browse,
                  icon: const Icon(Icons.folder_open, size: 18),
                  label: const Text('Browse folders'),
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.accent,
                    padding: const EdgeInsets.symmetric(vertical: 12),
                  ),
                ),
              ),
              const SizedBox(height: 10),
              SizedBox(
                width: 240,
                child: OutlinedButton.icon(
                  onPressed: _typePath,
                  icon: const Icon(Icons.edit, size: 18),
                  label: const Text('Type path'),
                ),
              ),
              if (_defaultPath != null) ...[
                const SizedBox(height: 16),
                TextButton.icon(
                  onPressed: () => _setPath(_defaultPath!),
                  icon: const Icon(Icons.home, size: 14),
                  label: Text('Use default · ${_shorten(_defaultPath!)}',
                      style: const TextStyle(fontSize: 12)),
                ),
              ],
              if (_recentPaths.isNotEmpty) ...[
                const SizedBox(height: 20),
                const Align(
                  alignment: Alignment.centerLeft,
                  child: Text('RECENT',
                      style: TextStyle(
                          fontSize: 11,
                          color: AppColors.textMuted,
                          letterSpacing: 0.8,
                          fontWeight: FontWeight.w600)),
                ),
                const SizedBox(height: 6),
                for (final p in _recentPaths.take(5))
                  _recentTile(p),
              ],
            ],
          ),
        ),
      );
    });
  }

  // State 3 — task list
  Widget _taskListState() {
    return Column(
      children: [
        _header(),
        if (_error != null) _errorBanner(),
        Expanded(child: _taskListView()),
      ],
    );
  }

  // ─── Widgets ──────────────────────────────────────────────

  Widget _header() {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 6, 8),
      decoration: const BoxDecoration(
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        Expanded(
          child: InkWell(
            onTap: _browse,
            borderRadius: BorderRadius.circular(8),
            child: Container(
              padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
              decoration: BoxDecoration(
                color: AppColors.surfaceAlt,
                borderRadius: BorderRadius.circular(8),
                border: Border.all(color: AppColors.border),
              ),
              child: Row(children: [
                const Icon(Icons.folder_outlined,
                    size: 14, color: AppColors.textMuted),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(_shorten(_path),
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                          fontFamily: 'monospace', fontSize: 12)),
                ),
                const Icon(Icons.arrow_drop_down,
                    size: 16, color: AppColors.textMuted),
              ]),
            ),
          ),
        ),
        _menuBtn(),
        IconButton(
          tooltip: 'Reload',
          icon: const Icon(Icons.refresh, size: 18),
          onPressed: _loading ? null : _loadTasks,
          padding: EdgeInsets.zero,
          constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
        ),
      ]),
    );
  }

  Widget _menuBtn() {
    return PopupMenuButton<String>(
      tooltip: 'Path options',
      icon: const Icon(Icons.more_vert, size: 18),
      padding: EdgeInsets.zero,
      color: AppColors.surfaceAlt,
      onSelected: (v) async {
        switch (v) {
          case 'browse':
            await _browse();
            break;
          case 'type':
            await _typePath();
            break;
          case 'default':
            if (_defaultPath != null) _setPath(_defaultPath!);
            break;
          case 'clear_history':
            await _clearRecents();
            break;
          default:
            if (v.startsWith('p:')) _setPath(v.substring(2));
        }
      },
      itemBuilder: (_) => [
        const PopupMenuItem<String>(
          value: 'browse',
          child: Row(children: [
            Icon(Icons.folder_open, size: 16, color: AppColors.accent),
            SizedBox(width: 8),
            Text('Browse folders', style: TextStyle(fontSize: 13)),
          ]),
        ),
        const PopupMenuItem<String>(
          value: 'type',
          child: Row(children: [
            Icon(Icons.edit, size: 16, color: AppColors.textMuted),
            SizedBox(width: 8),
            Text('Type path', style: TextStyle(fontSize: 13)),
          ]),
        ),
        if (_defaultPath != null)
          PopupMenuItem<String>(
            value: 'default',
            child: Row(children: [
              const Icon(Icons.home, size: 16, color: AppColors.textMuted),
              const SizedBox(width: 8),
              Expanded(
                child: Text('Default · ${_shorten(_defaultPath!)}',
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(fontSize: 13)),
              ),
            ]),
          ),
        if (_recentPaths.isNotEmpty) ...[
          const PopupMenuDivider(),
          const PopupMenuItem<String>(
            enabled: false,
            height: 24,
            child: Text('RECENT',
                style: TextStyle(
                    fontSize: 10,
                    color: AppColors.textMuted,
                    letterSpacing: 0.8)),
          ),
          for (final p in _recentPaths.take(6))
            PopupMenuItem<String>(
              value: 'p:$p',
              height: 34,
              child: Row(children: [
                const Icon(Icons.history,
                    size: 14, color: AppColors.textMuted),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(_shorten(p),
                      overflow: TextOverflow.ellipsis,
                      style: const TextStyle(
                          fontSize: 12, fontFamily: 'monospace')),
                ),
              ]),
            ),
          const PopupMenuDivider(),
          const PopupMenuItem<String>(
            value: 'clear_history',
            height: 32,
            child: Text('Clear recent',
                style:
                    TextStyle(fontSize: 12, color: AppColors.textMuted)),
          ),
        ],
      ],
    );
  }

  Widget _errorBanner() {
    final msg = _error ?? '';
    final hint = _errorHint(msg);
    return Container(
      width: double.infinity,
      color: AppColors.error.withValues(alpha: 0.12),
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Icon(Icons.error_outline,
              size: 16, color: AppColors.error),
          const SizedBox(width: 8),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(msg,
                    style: const TextStyle(
                        color: AppColors.error, fontSize: 12)),
                if (hint != null) ...[
                  const SizedBox(height: 4),
                  GestureDetector(
                    onTap: hint.onTap,
                    child: Text(hint.text,
                        style: TextStyle(
                            color: AppColors.accent,
                            fontSize: 11,
                            decoration: hint.onTap != null
                                ? TextDecoration.underline
                                : null)),
                  ),
                ],
              ],
            ),
          ),
          InkWell(
            onTap: () => setState(() => _error = null),
            child: const Padding(
              padding: EdgeInsets.all(2),
              child: Icon(Icons.close,
                  size: 14, color: AppColors.textMuted),
            ),
          ),
        ],
      ),
    );
  }

  _ErrorHint? _errorHint(String msg) {
    final m = msg.toLowerCase();
    if (m.contains('outside allowed roots')) {
      return _ErrorHint(
        'Tap to pick a folder inside Allowed Directories.',
        _browse,
      );
    }
    if (m.contains('not configured') ||
        m.contains('allowedroots') ||
        m.contains('set allowedroots')) {
      return _ErrorHint(
        'Open Settings → Providers → Task Runner to set Allowed Directories.',
        () => context.go('/settings'),
      );
    }
    if (m.contains('not enabled') || m.contains('not found')) {
      return _ErrorHint(
        'Enable the Task Runner plugin in Settings → Providers.',
        () => context.go('/settings'),
      );
    }
    return null;
  }

  Widget _taskListView() {
    if (_loading) {
      return const Center(
          child: CircularProgressIndicator(color: AppColors.accent));
    }
    if (_tasks.isEmpty) {
      return RefreshIndicator(
        onRefresh: _loadTasks,
        child: ListView(
          padding: const EdgeInsets.all(20),
          children: [
            const SizedBox(height: 40),
            const Icon(Icons.search_off,
                size: 40, color: AppColors.textMuted),
            const SizedBox(height: 12),
            const Text('No tasks in this folder',
                textAlign: TextAlign.center,
                style: TextStyle(
                    fontWeight: FontWeight.w500, fontSize: 14)),
            const SizedBox(height: 4),
            const Text(
              'No Makefile, package.json, or *.sh scripts were found. Pick a different project folder.',
              textAlign: TextAlign.center,
              style: TextStyle(color: AppColors.textMuted, fontSize: 12),
            ),
            const SizedBox(height: 20),
            Center(
              child: OutlinedButton.icon(
                onPressed: _browse,
                icon: const Icon(Icons.folder_open, size: 16),
                label: const Text('Pick another folder'),
              ),
            ),
          ],
        ),
      );
    }
    final groups = <String, List<Map<String, dynamic>>>{};
    for (final t in _tasks) {
      final s = t['source'] as String? ?? 'other';
      groups.putIfAbsent(s, () => []).add(t);
    }
    final keys = groups.keys.toList()..sort();
    return RefreshIndicator(
      onRefresh: _loadTasks,
      child: ListView(
        padding: const EdgeInsets.only(bottom: 120),
        children: [
          for (final k in keys) ...[
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 16, 16, 6),
              child: Row(children: [
                Icon(_iconFor(k), size: 14, color: AppColors.textMuted),
                const SizedBox(width: 6),
                Text(k.toUpperCase(),
                    style: const TextStyle(
                        fontSize: 11,
                        color: AppColors.textMuted,
                        fontWeight: FontWeight.w600,
                        letterSpacing: 0.8)),
                const SizedBox(width: 6),
                Text('· ${groups[k]!.length}',
                    style: const TextStyle(
                        fontSize: 11, color: AppColors.textMuted)),
              ]),
            ),
            for (final t in groups[k]!) _taskTile(t),
          ],
        ],
      ),
    );
  }

  Widget _taskTile(Map<String, dynamic> t) {
    final running = _run != null &&
        _run!.meta['taskId'] == t['id'] &&
        _run!.meta['status'] == 'running';
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: running ? _showRunSheet : () => _runTask(t),
        child: Padding(
          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
          child: Row(children: [
            Container(
              width: 32,
              height: 32,
              decoration: BoxDecoration(
                color: running
                    ? AppColors.accent.withValues(alpha: 0.15)
                    : AppColors.surfaceAlt,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Icon(
                running ? Icons.sync : Icons.play_arrow_rounded,
                size: 18,
                color: running ? AppColors.accent : AppColors.text,
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(t['name'] as String? ?? '',
                      style: const TextStyle(
                          fontSize: 14, fontWeight: FontWeight.w500),
                      overflow: TextOverflow.ellipsis),
                  const SizedBox(height: 2),
                  Text(t['display'] as String? ?? '',
                      style: const TextStyle(
                          fontSize: 11,
                          color: AppColors.textMuted,
                          fontFamily: 'monospace'),
                      overflow: TextOverflow.ellipsis),
                ],
              ),
            ),
            if (running)
              const Padding(
                padding: EdgeInsets.only(left: 8),
                child: Text('running',
                    style: TextStyle(
                        fontSize: 10, color: AppColors.accent)),
              ),
          ]),
        ),
      ),
    );
  }

  Widget _runBadge(_RunSession run) {
    final status = run.meta['status'] as String? ?? 'running';
    final name = run.meta['taskName'] as String? ?? '';
    final exitCode = run.meta['exitCode'];
    final dotColor = switch (status) {
      'running' => AppColors.accent,
      'exited' => (exitCode ?? -1) == 0 ? Colors.green : AppColors.error,
      'killed' => AppColors.textMuted,
      _ => AppColors.error,
    };
    return Material(
      color: Colors.transparent,
      child: InkWell(
        onTap: _showRunSheet,
        borderRadius: BorderRadius.circular(10),
        child: Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: BoxDecoration(
            color: AppColors.surfaceAlt,
            borderRadius: BorderRadius.circular(10),
            border: Border.all(color: AppColors.border),
            boxShadow: const [
              BoxShadow(
                  color: Colors.black26,
                  blurRadius: 12,
                  offset: Offset(0, 4)),
            ],
          ),
          child: Row(children: [
            Container(
              width: 8,
              height: 8,
              decoration:
                  BoxDecoration(color: dotColor, shape: BoxShape.circle),
            ),
            const SizedBox(width: 10),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                mainAxisSize: MainAxisSize.min,
                children: [
                  Text(name,
                      style: const TextStyle(
                          fontSize: 13, fontWeight: FontWeight.w500),
                      overflow: TextOverflow.ellipsis),
                  Text(
                    status == 'running'
                        ? 'Tap to view output'
                        : 'Exit ${exitCode ?? "—"} · tap to view',
                    style: const TextStyle(
                        fontSize: 11, color: AppColors.textMuted),
                  ),
                ],
              ),
            ),
            if (status == 'running')
              IconButton(
                tooltip: 'Stop',
                onPressed: run.stop,
                icon: const Icon(Icons.stop,
                    size: 18, color: AppColors.error),
                padding: EdgeInsets.zero,
                constraints:
                    const BoxConstraints(minWidth: 32, minHeight: 32),
              )
            else
              IconButton(
                tooltip: 'Dismiss',
                onPressed: () {
                  run.removeListener(_onRunTick);
                  run.dispose();
                  setState(() => _run = null);
                },
                icon: const Icon(Icons.close,
                    size: 16, color: AppColors.textMuted),
                padding: EdgeInsets.zero,
                constraints:
                    const BoxConstraints(minWidth: 32, minHeight: 32),
              ),
          ]),
        ),
      ),
    );
  }

  Widget _recentTile(String p) {
    return Card(
      margin: const EdgeInsets.symmetric(vertical: 4),
      color: AppColors.surfaceAlt,
      elevation: 0,
      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
      child: ListTile(
        dense: true,
        onTap: () => _setPath(p),
        leading:
            const Icon(Icons.history, size: 16, color: AppColors.textMuted),
        title: Text(_shorten(p),
            style: const TextStyle(fontFamily: 'monospace', fontSize: 12)),
        trailing: const Icon(Icons.chevron_right,
            size: 16, color: AppColors.textMuted),
      ),
    );
  }

  Widget _centeredCard({
    required IconData icon,
    required String title,
    required String subtitle,
    Widget? primary,
  }) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 44, color: AppColors.textMuted),
            const SizedBox(height: 12),
            Text(title,
                textAlign: TextAlign.center,
                style: const TextStyle(
                    fontWeight: FontWeight.w600, fontSize: 15)),
            const SizedBox(height: 6),
            Text(subtitle,
                textAlign: TextAlign.center,
                style: const TextStyle(
                    color: AppColors.textMuted, fontSize: 12)),
            if (primary != null) ...[
              const SizedBox(height: 20),
              primary,
            ],
          ],
        ),
      ),
    );
  }

  // ─── Helpers ──────────────────────────────────────────────

  static String _shorten(String p) {
    if (p.isEmpty) return 'Pick a project';
    if (p.length <= 44) return p;
    final parts = p.split('/');
    if (parts.length <= 4) return p;
    return '…/${parts.sublist(parts.length - 3).join('/')}';
  }

  static IconData _iconFor(String source) => switch (source) {
        'makefile' => Icons.build,
        'package.json' => Icons.inventory_2_outlined,
        'shell' => Icons.terminal,
        _ => Icons.play_arrow,
      };
}

class _ErrorHint {
  final String text;
  final VoidCallback? onTap;
  _ErrorHint(this.text, this.onTap);
}

// ─── Run session + output sheet ─────────────────────────────

class _RunSession extends ChangeNotifier {
  final ApiClient api;
  final String plugin;
  final String runId;

  Map<String, dynamic> meta;
  final List<int> _raw = [];
  String output = '';
  String? exitMsg;

  WebSocketChannel? _channel;
  StreamSubscription? _sub;
  Timer? _flushTimer;
  bool _dirty = false;

  _RunSession({
    required this.api,
    required this.plugin,
    required this.runId,
    required this.meta,
  });

  void start() {
    final uri = api.tasksRunWsUri(plugin, runId);
    final ch = connectWs(uri);
    _channel = ch;
    _sub = ch.stream.listen(_onData, onDone: _refresh, onError: (e) {
      exitMsg = 'Stream error: $e';
      notifyListeners();
    });
    _flushTimer = Timer.periodic(const Duration(milliseconds: 80), (_) {
      if (_dirty) {
        _flush();
        notifyListeners();
      }
    });
  }

  void _onData(dynamic data) {
    if (data is List<int>) {
      _appendBytes(Uint8List.fromList(data));
    } else if (data is String) {
      try {
        final msg = jsonDecode(data) as Map<String, dynamic>;
        if (msg['type'] == 'exit') {
          meta = {
            ...meta,
            'status': msg['status'],
            'exitCode': msg['exitCode'],
          };
          exitMsg = _formatExit(msg);
          _flush();
          notifyListeners();
        }
      } catch (_) {
        _appendBytes(Uint8List.fromList(utf8.encode(data)));
      }
    }
  }

  void _appendBytes(Uint8List chunk) {
    _raw.addAll(chunk);
    if (_raw.length > _maxOutputBytes) {
      _raw.removeRange(0, _raw.length - _maxOutputBytes);
    }
    _dirty = true;
  }

  void _flush() {
    output = utf8.decode(_raw, allowMalformed: true).replaceAll(_ansiRe, '');
    _dirty = false;
  }

  Future<void> _refresh() async {
    try {
      final fresh = await api.tasksRunGet(plugin, runId);
      meta = fresh;
      exitMsg ??= _formatExit(fresh);
      notifyListeners();
    } catch (_) {}
  }

  Future<void> stop() async {
    try {
      await api.tasksRunStop(plugin, runId);
    } catch (_) {}
  }

  static String _formatExit(Map<String, dynamic> msg) {
    final status = msg['status'];
    final code = msg['exitCode'];
    final err = msg['error'];
    if (err is String && err.isNotEmpty) return '$status: $err';
    return code == null ? '$status' : '$status · exit $code';
  }

  @override
  void dispose() {
    _flushTimer?.cancel();
    _sub?.cancel();
    _channel?.sink.close();
    _channel = null;
    super.dispose();
  }
}

class _RunSheet extends StatelessWidget {
  final _RunSession session;
  const _RunSheet({required this.session});

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      initialChildSize: 0.7,
      minChildSize: 0.3,
      maxChildSize: 0.95,
      expand: false,
      builder: (ctx, scrollCtrl) {
        return _RunSheetBody(
          session: session,
          scrollController: scrollCtrl,
        );
      },
    );
  }
}

class _RunSheetBody extends StatefulWidget {
  final _RunSession session;
  final ScrollController scrollController;
  const _RunSheetBody({
    required this.session,
    required this.scrollController,
  });
  @override
  State<_RunSheetBody> createState() => _RunSheetBodyState();
}

class _RunSheetBodyState extends State<_RunSheetBody> {
  @override
  void initState() {
    super.initState();
    widget.session.addListener(_onTick);
  }

  @override
  void dispose() {
    widget.session.removeListener(_onTick);
    super.dispose();
  }

  void _onTick() {
    if (mounted) setState(() {});
  }

  void _copy() {
    Clipboard.setData(ClipboardData(text: widget.session.output));
    ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
      content: Text('Output copied'),
      duration: Duration(seconds: 1),
    ));
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.session;
    final status = s.meta['status'] as String? ?? 'running';
    final exitCode = s.meta['exitCode'];
    final dotColor = switch (status) {
      'running' => AppColors.accent,
      'exited' => (exitCode ?? -1) == 0 ? Colors.green : AppColors.error,
      'killed' => AppColors.textMuted,
      _ => AppColors.error,
    };
    return Column(children: [
      // Drag handle
      Padding(
        padding: const EdgeInsets.only(top: 8),
        child: Container(
          width: 36,
          height: 4,
          decoration: BoxDecoration(
            color: AppColors.border,
            borderRadius: BorderRadius.circular(2),
          ),
        ),
      ),
      // Header
      Padding(
        padding: const EdgeInsets.fromLTRB(16, 12, 8, 8),
        child: Row(children: [
          Container(
            width: 10,
            height: 10,
            decoration:
                BoxDecoration(color: dotColor, shape: BoxShape.circle),
          ),
          const SizedBox(width: 10),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(s.meta['taskName'] as String? ?? '',
                    style: const TextStyle(
                        fontSize: 14,
                        fontWeight: FontWeight.w600,
                        color: Colors.white)),
                Text(s.meta['display'] as String? ?? '',
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 11,
                      color: Color(0xFF8B96A3),
                    ),
                    overflow: TextOverflow.ellipsis),
              ],
            ),
          ),
          IconButton(
            tooltip: 'Copy',
            onPressed: s.output.isEmpty ? null : _copy,
            icon: const Icon(Icons.copy,
                size: 16, color: Color(0xFFD7DEE6)),
          ),
          if (status == 'running')
            TextButton.icon(
              onPressed: s.stop,
              icon: const Icon(Icons.stop,
                  size: 16, color: AppColors.error),
              label: const Text('Stop',
                  style:
                      TextStyle(color: AppColors.error, fontSize: 12)),
            ),
          IconButton(
            tooltip: 'Close',
            onPressed: () => Navigator.pop(context),
            icon: const Icon(Icons.keyboard_arrow_down,
                color: Color(0xFFD7DEE6)),
          ),
        ]),
      ),
      const Divider(height: 1, color: Color(0xFF1D232B)),
      // Output
      Expanded(
        child: Container(
          color: const Color(0xFF0E1116),
          padding: const EdgeInsets.all(12),
          child: SingleChildScrollView(
            controller: widget.scrollController,
            reverse: true,
            child: SelectableText(
              s.output.isEmpty ? '(no output yet)' : s.output,
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 12,
                color: Color(0xFFD7DEE6),
                height: 1.35,
              ),
            ),
          ),
        ),
      ),
      if (s.exitMsg != null)
        Container(
          width: double.infinity,
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          color: status == 'exited' && (exitCode ?? -1) == 0
              ? Colors.green.withValues(alpha: 0.18)
              : AppColors.error.withValues(alpha: 0.18),
          child: Text(s.exitMsg!,
              style: const TextStyle(
                  fontSize: 12,
                  fontFamily: 'monospace',
                  color: Colors.white)),
        ),
    ]);
  }
}
