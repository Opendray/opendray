import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_highlight/flutter_highlight.dart';
import 'package:flutter_highlight/themes/monokai-sublime.dart';
import 'package:provider/provider.dart';
import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/session_launcher.dart';
import '../../shared/theme/app_theme.dart';

class FilesPage extends StatefulWidget {
  const FilesPage({super.key});
  @override
  State<FilesPage> createState() => _FilesPageState();
}

class _FilesPageState extends State<FilesPage> {
  List<ProviderInfo> _filePlugins = [];
  String? _activePlugin;
  List<Map<String, dynamic>> _entries = [];
  String _currentPath = '';
  List<String> _pathStack = [];
  Map<String, dynamic>? _currentFile;
  String _searchQuery = '';
  List<Map<String, dynamic>> _searchResults = [];
  bool _loading = false;
  String? _error;
  StreamSubscription<void>? _providersSub;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _loadPlugins();
    _providersSub = ProvidersBus.instance.changes.listen((_) => _loadPlugins());
  }

  @override
  void dispose() {
    _providersSub?.cancel();
    super.dispose();
  }

  Future<void> _loadPlugins() async {
    try {
      final providers = await _api.listProviders();
      final files = providers.where((p) =>
        p.provider.type == 'panel' && p.provider.category == 'files' && p.enabled).toList();
      if (!mounted) return;
      final stillActive = _activePlugin != null &&
          files.any((p) => p.provider.name == _activePlugin);
      setState(() {
        _filePlugins = files;
        if (!stillActive) {
          // The active plugin was just disabled — clear file-tree state so
          // its breadcrumb / "Session" button / list of entries disappears.
          _activePlugin = null;
          _entries = [];
          _searchResults = [];
          _currentPath = '';
          _pathStack = [];
          _currentFile = null;
        }
      });
      if (files.isNotEmpty && _activePlugin == null) {
        _activePlugin = files.first.provider.name;
        _loadTree();
      }
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    }
  }

  Future<void> _loadTree([String? path]) async {
    if (_activePlugin == null) return;
    setState(() { _loading = true; _error = null; _currentFile = null; _searchResults = []; });
    try {
      final entries = await _api.filesTree(_activePlugin!, path: path ?? _currentPath);
      setState(() { _entries = entries; _loading = false; if (path != null) _currentPath = path; });
    } catch (e) {
      setState(() { _error = e.toString(); _loading = false; });
    }
  }

  Future<void> _openFile(String path) async {
    if (_activePlugin == null) return;
    setState(() { _loading = true; _error = null; });
    try {
      final file = await _api.filesFile(_activePlugin!, path);
      setState(() { _currentFile = file; _loading = false; _searchResults = []; });
    } catch (e) {
      setState(() { _error = e.toString(); _loading = false; });
    }
  }

  Future<void> _search(String query) async {
    if (_activePlugin == null || query.isEmpty) return;
    setState(() { _loading = true; });
    try {
      final results = await _api.filesSearch(_activePlugin!, query, basePath: _currentPath);
      setState(() { _searchResults = results; _loading = false; });
    } catch (e) {
      setState(() { _loading = false; });
    }
  }

  void _navigateToDir(Map<String, dynamic> entry) {
    final path = entry['path'] as String;
    _pathStack.add(_currentPath);
    _loadTree(path);
  }

  Future<void> _showFolderActions(Map<String, dynamic> entry) async {
    final path = entry['path'] as String? ?? '';
    final name = entry['name'] as String? ?? '';
    if (path.isEmpty) return;
    await showModalBottomSheet<void>(
      context: context,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (ctx) => SafeArea(child: Column(mainAxisSize: MainAxisSize.min, children: [
        Padding(
          padding: const EdgeInsets.fromLTRB(16, 14, 16, 8),
          child: Row(children: [
            const Icon(Icons.folder, size: 16, color: AppColors.warning),
            const SizedBox(width: 8),
            Expanded(child: Text(name,
                style: const TextStyle(fontWeight: FontWeight.w600, fontSize: 14),
                maxLines: 1, overflow: TextOverflow.ellipsis)),
          ]),
        ),
        const Divider(height: 1),
        ListTile(
          leading: const Icon(Icons.terminal, color: AppColors.accent),
          title: const Text('Create session here', style: TextStyle(fontSize: 14)),
          subtitle: Text(path,
              style: const TextStyle(fontSize: 10, color: AppColors.textMuted, fontFamily: 'monospace'),
              maxLines: 1, overflow: TextOverflow.ellipsis),
          onTap: () {
            Navigator.pop(ctx);
            launchNewSession(context, initialCwd: path);
          },
        ),
        ListTile(
          leading: const Icon(Icons.folder_open, color: AppColors.textMuted),
          title: const Text('Open', style: TextStyle(fontSize: 14)),
          onTap: () {
            Navigator.pop(ctx);
            _navigateToDir(entry);
          },
        ),
        ListTile(
          leading: const Icon(Icons.copy, color: AppColors.textMuted),
          title: const Text('Copy path', style: TextStyle(fontSize: 14)),
          onTap: () {
            Navigator.pop(ctx);
            Clipboard.setData(ClipboardData(text: path));
            ScaffoldMessenger.of(context).showSnackBar(const SnackBar(
              content: Text('Path copied'), duration: Duration(seconds: 1)));
          },
        ),
      ])),
    );
  }

  void _navigateUp() {
    if (_pathStack.isNotEmpty) {
      final prev = _pathStack.removeLast();
      _loadTree(prev);
    }
  }

  String _shortenPath(String path) {
    final parts = path.split('/');
    return parts.length > 4 ? '.../${parts.sublist(parts.length - 3).join('/')}' : path;
  }

  @override
  Widget build(BuildContext context) {
    if (_filePlugins.isEmpty) return _buildNoPlugins();
    if (_currentFile != null) return _buildCodeView();
    return _buildBrowser();
  }

  Widget _buildNoPlugins() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          const Icon(Icons.folder_off, size: 48, color: AppColors.textMuted),
          const SizedBox(height: 16),
          Text(context.tr('No file browser configured'),
              style: const TextStyle(fontWeight: FontWeight.w500)),
          const SizedBox(height: 8),
          Text(_error ??
              context.tr('Enable File Browser in Settings → Plugins and configure the allowed directories.'),
            style: const TextStyle(color: AppColors.textMuted, fontSize: 12), textAlign: TextAlign.center),
        ]),
      ),
    );
  }

  Widget _buildBrowser() {
    return Column(
      children: [
        // Search
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
          child: TextField(
            decoration: InputDecoration(
              hintText: 'Search files...',
              prefixIcon: const Icon(Icons.search, size: 18),
              suffixIcon: _searchQuery.isNotEmpty
                  ? IconButton(icon: const Icon(Icons.clear, size: 18), onPressed: () {
                      setState(() { _searchQuery = ''; _searchResults = []; });
                    })
                  : null,
              contentPadding: const EdgeInsets.symmetric(vertical: 8),
              isDense: true,
            ),
            style: const TextStyle(fontSize: 13),
            onChanged: (v) { _searchQuery = v; if (v.length >= 2) _search(v); },
          ),
        ),

        // Current path breadcrumb + "Create session here" action
        if (_currentPath.isNotEmpty && _searchResults.isEmpty)
          Container(
            height: 40,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: Row(children: [
              Expanded(
                child: SingleChildScrollView(
                  scrollDirection: Axis.horizontal,
                  child: Text(_shortenPath(_currentPath),
                      style: const TextStyle(
                          color: AppColors.textMuted,
                          fontSize: 12,
                          fontFamily: 'monospace')),
                ),
              ),
              TextButton.icon(
                onPressed: () => launchNewSession(context, initialCwd: _currentPath),
                icon: const Icon(Icons.terminal, size: 14),
                label: const Text('Session', style: TextStyle(fontSize: 11)),
                style: TextButton.styleFrom(
                  foregroundColor: AppColors.accent,
                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                  minimumSize: Size.zero,
                  tapTargetSize: MaterialTapTargetSize.shrinkWrap,
                ),
              ),
            ]),
          ),

        // Error
        if (_error != null)
          Container(
            margin: const EdgeInsets.all(12),
            padding: const EdgeInsets.all(10),
            decoration: BoxDecoration(color: AppColors.errorSoft, borderRadius: BorderRadius.circular(8)),
            child: Text(_error!, style: const TextStyle(color: AppColors.error, fontSize: 12)),
          ),

        // Content
        Expanded(
          child: _loading
              ? const Center(child: CircularProgressIndicator(color: AppColors.accent))
              : _searchResults.isNotEmpty
                  ? _buildSearchResults()
                  : _buildEntryList(),
        ),
      ],
    );
  }

  Widget _buildEntryList() {
    if (_entries.isEmpty) {
      return const Center(child: Text('Empty directory', style: TextStyle(color: AppColors.textMuted)));
    }
    return RefreshIndicator(
      onRefresh: () => _loadTree(),
      child: ListView.separated(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        itemCount: (_pathStack.isNotEmpty ? 1 : 0) + _entries.length,
        separatorBuilder: (_, _) => const Divider(height: 1, color: AppColors.border),
        itemBuilder: (_, i) {
          if (_pathStack.isNotEmpty && i == 0) {
            return ListTile(
              dense: true,
              leading: const Icon(Icons.arrow_upward, size: 18, color: AppColors.textMuted),
              title: const Text('..', style: TextStyle(fontSize: 13, color: AppColors.textMuted)),
              onTap: _navigateUp,
            );
          }
          final idx = _pathStack.isNotEmpty ? i - 1 : i;
          final entry = _entries[idx];
          final isDir = entry['type'] == 'dir';
          final isGit = entry['isGit'] == true;

          return ListTile(
            dense: true,
            leading: Icon(
              isDir ? (isGit ? Icons.source : Icons.folder) : _fileIcon(entry['ext'] ?? ''),
              size: 18,
              color: isDir ? (isGit ? AppColors.accent : AppColors.warning) : AppColors.textMuted,
            ),
            title: Row(children: [
              Expanded(child: Text(entry['name'] ?? '', style: const TextStyle(fontSize: 13))),
              if (isGit) Container(
                padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
                decoration: BoxDecoration(color: AppColors.accentSoft, borderRadius: BorderRadius.circular(3)),
                child: const Text('git', style: TextStyle(color: AppColors.accent, fontSize: 9)),
              ),
            ]),
            // On folders: quick-launch button → New Session with this path.
            // On files: show size, unchanged.
            trailing: isDir
                ? IconButton(
                    icon: const Icon(Icons.terminal, size: 16, color: AppColors.accent),
                    onPressed: () => launchNewSession(context, initialCwd: entry['path']),
                    tooltip: 'Create session here',
                    padding: EdgeInsets.zero,
                    constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
                  )
                : Text(_formatSize(entry['size'] ?? 0),
                    style: const TextStyle(color: AppColors.textMuted, fontSize: 10)),
            onTap: () {
              if (isDir) {
                _navigateToDir(entry);
              } else {
                _openFile(entry['path']);
              }
            },
            onLongPress: !isDir
                ? null
                : () => _showFolderActions(entry),
          );
        },
      ),
    );
  }

  Widget _buildSearchResults() {
    return ListView.separated(
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
      itemCount: _searchResults.length,
      separatorBuilder: (_, _) => const Divider(height: 1, color: AppColors.border),
      itemBuilder: (_, i) {
        final entry = _searchResults[i];
        return ListTile(
          dense: true,
          leading: Icon(_fileIcon(entry['ext'] ?? ''), size: 18, color: AppColors.textMuted),
          title: Text(entry['name'] ?? '', style: const TextStyle(fontSize: 13)),
          subtitle: Text(_shortenPath(entry['path'] ?? ''), style: const TextStyle(color: AppColors.textMuted, fontSize: 11, fontFamily: 'monospace')),
          onTap: () => _openFile(entry['path']),
        );
      },
    );
  }

  Widget _buildCodeView() {
    final name = _currentFile!['name'] ?? '';
    final content = _currentFile!['content'] ?? '';
    final language = _currentFile!['language'] ?? 'text';
    final size = _currentFile!['size'] ?? 0;
    final binary = _currentFile!['binary'] == true;

    return Column(
      children: [
        // File header
        Container(
          padding: const EdgeInsets.symmetric(horizontal: 14, vertical: 10),
          decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
          child: Row(
            children: [
              IconButton(
                icon: const Icon(Icons.arrow_back, size: 18),
                onPressed: () => setState(() => _currentFile = null),
                tooltip: 'Back',
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              ),
              const SizedBox(width: 4),
              Icon(_fileIcon(_currentFile!['ext'] ?? ''), size: 16, color: AppColors.accent),
              const SizedBox(width: 8),
              Expanded(child: Text(name, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500))),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                decoration: BoxDecoration(color: AppColors.surfaceAlt, borderRadius: BorderRadius.circular(3)),
                child: Text(language, style: const TextStyle(color: AppColors.textMuted, fontSize: 10)),
              ),
              const SizedBox(width: 8),
              Text(_formatSize(size), style: const TextStyle(color: AppColors.textMuted, fontSize: 11)),
              const SizedBox(width: 8),
              IconButton(
                icon: const Icon(Icons.copy, size: 16, color: AppColors.textMuted),
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: content));
                  ScaffoldMessenger.of(context).showSnackBar(
                    const SnackBar(content: Text('Copied'), duration: Duration(seconds: 1), backgroundColor: AppColors.success),
                  );
                },
                tooltip: 'Copy',
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 32, minHeight: 32),
              ),
            ],
          ),
        ),
        // Code content
        Expanded(
          child: binary
              ? Center(child: Text(content, style: const TextStyle(color: AppColors.textMuted)))
              : SingleChildScrollView(
                  child: HighlightView(
                    content,
                    language: language,
                    theme: monokaiSublimeTheme,
                    padding: const EdgeInsets.all(14),
                    textStyle: const TextStyle(fontSize: 13, fontFamily: 'JetBrains Mono', height: 1.5),
                  ),
                ),
        ),
      ],
    );
  }

  IconData _fileIcon(String ext) {
    return switch (ext) {
      'go' || 'dart' || 'py' || 'js' || 'ts' || 'rs' || 'java' || 'kt' || 'swift' || 'c' || 'cpp' => Icons.code,
      'md' || 'txt' || 'rst' => Icons.description,
      'json' || 'yaml' || 'yml' || 'toml' || 'xml' => Icons.data_object,
      'html' || 'css' || 'scss' || 'vue' || 'svelte' => Icons.web,
      'sh' || 'bash' || 'zsh' => Icons.terminal,
      'sql' => Icons.storage,
      'png' || 'jpg' || 'gif' || 'svg' || 'webp' => Icons.image,
      'pdf' => Icons.picture_as_pdf,
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
