import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_markdown/flutter_markdown.dart';
import 'package:provider/provider.dart';
import '../../core/api/api_client.dart';
import '../../core/models/provider.dart';
import '../../core/services/l10n.dart';
import '../../shared/providers_bus.dart';
import '../../shared/theme/app_theme.dart';

class DocsPage extends StatefulWidget {
  const DocsPage({super.key});
  @override
  State<DocsPage> createState() => _DocsPageState();
}

class _DocsPageState extends State<DocsPage> {
  List<ProviderInfo> _docPlugins = [];
  String? _activePlugin;
  List<Map<String, dynamic>> _entries = [];
  List<String> _pathStack = []; // breadcrumb
  Map<String, dynamic>? _currentFile;
  String _searchQuery = '';
  List<Map<String, dynamic>> _searchResults = [];
  bool _loading = false;
  // ignore: unused_field
  bool _searching = false;
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
      final docs = providers.where((p) =>
        p.provider.type == 'panel' && p.provider.category == 'docs' && p.enabled).toList();
      if (!mounted) return;
      final stillActive = _activePlugin != null &&
          docs.any((p) => p.provider.name == _activePlugin);
      setState(() {
        _docPlugins = docs;
        if (!stillActive) {
          _activePlugin = null;
          _entries = [];
          _searchResults = [];
          _pathStack = [];
          _currentFile = null;
        }
      });
      if (docs.isNotEmpty && _activePlugin == null) {
        _activePlugin = docs.first.provider.name;
        _loadTree();
      }
    } catch (e) {
      if (mounted) setState(() => _error = e.toString());
    }
  }

  Future<void> _loadTree() async {
    if (_activePlugin == null) return;
    setState(() { _loading = true; _error = null; _currentFile = null; });
    try {
      final path = _pathStack.join('/');
      final entries = await _api.docsTree(_activePlugin!, path: path);
      setState(() { _entries = entries; _loading = false; });
    } catch (e) {
      setState(() { _error = e.toString(); _loading = false; });
    }
  }

  Future<void> _openFile(String path) async {
    if (_activePlugin == null) return;
    setState(() { _loading = true; _error = null; });
    try {
      final file = await _api.docsFile(_activePlugin!, path);
      setState(() { _currentFile = file; _loading = false; _searchResults = []; });
    } catch (e) {
      setState(() { _error = e.toString(); _loading = false; });
    }
  }

  Future<void> _search(String query) async {
    if (_activePlugin == null || query.isEmpty) return;
    setState(() { _searching = true; _searchResults = []; });
    try {
      final results = await _api.docsSearch(_activePlugin!, query);
      setState(() { _searchResults = results; _searching = false; });
    } catch (e) {
      setState(() { _searching = false; });
    }
  }

  void _navigateToDir(String name) {
    _pathStack.add(name);
    _loadTree();
  }

  void _navigateUp() {
    if (_pathStack.isNotEmpty) {
      _pathStack.removeLast();
      _loadTree();
    }
  }

  void _navigateToBreadcrumb(int index) {
    _pathStack.removeRange(index + 1, _pathStack.length);
    _loadTree();
  }

  @override
  Widget build(BuildContext context) {
    if (_docPlugins.isEmpty) return _buildNoPlugins();
    if (_currentFile != null) return _buildFileView();
    return _buildBrowser();
  }

  Widget _buildNoPlugins() {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            const Icon(Icons.description_outlined, size: 48, color: AppColors.textMuted),
            const SizedBox(height: 16),
            Text(context.tr('No docs browser configured'),
                style: const TextStyle(fontWeight: FontWeight.w500)),
            const SizedBox(height: 8),
            Text(
              _error ??
                  context.tr('Enable a docs-type plugin (e.g. Obsidian Reader) in Settings → Plugins and configure the connection.'),
              style: const TextStyle(color: AppColors.textMuted, fontSize: 12),
              textAlign: TextAlign.center,
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildBrowser() {
    return Column(
      children: [
        // Plugin selector (if multiple)
        if (_docPlugins.length > 1)
          Container(
            height: 44,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            decoration: const BoxDecoration(border: Border(bottom: BorderSide(color: AppColors.border))),
            child: ListView(
              scrollDirection: Axis.horizontal,
              children: _docPlugins.map((p) => Padding(
                padding: const EdgeInsets.only(right: 8),
                child: ChoiceChip(
                  label: Text('${p.provider.icon} '
                      '${context.pickL10n(p.provider.displayName, p.provider.displayNameZh)}'),
                  selected: _activePlugin == p.provider.name,
                  onSelected: (_) {
                    setState(() { _activePlugin = p.provider.name; _pathStack.clear(); });
                    _loadTree();
                  },
                  selectedColor: AppColors.accentSoft,
                  backgroundColor: AppColors.surfaceAlt,
                  labelStyle: TextStyle(fontSize: 12, color: _activePlugin == p.provider.name ? AppColors.accent : AppColors.textMuted),
                  side: BorderSide.none,
                ),
              )).toList(),
            ),
          ),

        // Search bar
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
          child: TextField(
            decoration: InputDecoration(
              hintText: 'Search docs...',
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
            onChanged: (v) {
              _searchQuery = v;
              if (v.length >= 2) _search(v);
            },
          ),
        ),

        // Breadcrumb
        if (_pathStack.isNotEmpty && _searchResults.isEmpty)
          Container(
            height: 36,
            padding: const EdgeInsets.symmetric(horizontal: 12),
            alignment: Alignment.centerLeft,
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
                  GestureDetector(
                    onTap: () { _pathStack.clear(); _loadTree(); },
                    child: const Text('/', style: TextStyle(color: AppColors.accent, fontSize: 12)),
                  ),
                  for (int i = 0; i < _pathStack.length; i++) ...[
                    const Text(' / ', style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
                    GestureDetector(
                      onTap: () => _navigateToBreadcrumb(i),
                      child: Text(_pathStack[i], style: TextStyle(
                        color: i == _pathStack.length - 1 ? AppColors.text : AppColors.accent,
                        fontSize: 12,
                      )),
                    ),
                  ],
                ],
              ),
            ),
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
      onRefresh: _loadTree,
      child: ListView.separated(
        padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
        itemCount: (_pathStack.isNotEmpty ? 1 : 0) + _entries.length,
        separatorBuilder: (_, _) => const Divider(height: 1, color: AppColors.border),
        itemBuilder: (_, i) {
          // ".." entry
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
          return ListTile(
            dense: true,
            leading: Icon(
              isDir ? Icons.folder : Icons.description,
              size: 18,
              color: isDir ? AppColors.warning : AppColors.textMuted,
            ),
            title: Text(entry['name'] ?? '', style: const TextStyle(fontSize: 13)),
            trailing: !isDir ? Text(_formatSize(entry['size'] ?? 0), style: const TextStyle(color: AppColors.textMuted, fontSize: 10)) : null,
            onTap: () {
              if (isDir) {
                _navigateToDir(entry['name']);
              } else {
                final path = _pathStack.isNotEmpty ? '${_pathStack.join('/')}/${entry['name']}' : entry['name'];
                _openFile(path);
              }
            },
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
          leading: const Icon(Icons.description, size: 18, color: AppColors.textMuted),
          title: Text(entry['name'] ?? '', style: const TextStyle(fontSize: 13)),
          subtitle: Text(entry['path'] ?? '', style: const TextStyle(color: AppColors.textMuted, fontSize: 11)),
          onTap: () => _openFile(entry['path']),
        );
      },
    );
  }

  Widget _buildFileView() {
    final name = _currentFile!['name'] ?? '';
    final content = _currentFile!['content'] ?? '';

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
              const Icon(Icons.description, size: 16, color: AppColors.accent),
              const SizedBox(width: 8),
              Expanded(child: Text(name, style: const TextStyle(fontSize: 13, fontWeight: FontWeight.w500))),
              Text(_formatSize(_currentFile!['size'] ?? 0), style: const TextStyle(color: AppColors.textMuted, fontSize: 11)),
            ],
          ),
        ),
        // Markdown content
        Expanded(
          child: Markdown(
            data: content,
            selectable: true,
            padding: const EdgeInsets.all(16),
            styleSheet: MarkdownStyleSheet(
              h1: const TextStyle(color: AppColors.text, fontSize: 22, fontWeight: FontWeight.w600),
              h2: const TextStyle(color: AppColors.text, fontSize: 18, fontWeight: FontWeight.w600),
              h3: const TextStyle(color: AppColors.text, fontSize: 15, fontWeight: FontWeight.w600),
              p: const TextStyle(color: AppColors.text, fontSize: 13, height: 1.6),
              code: TextStyle(color: AppColors.accent, fontSize: 12, fontFamily: 'JetBrains Mono', backgroundColor: AppColors.surfaceAlt),
              codeblockDecoration: BoxDecoration(color: AppColors.surfaceAlt, borderRadius: BorderRadius.circular(8)),
              codeblockPadding: const EdgeInsets.all(12),
              blockquoteDecoration: BoxDecoration(
                border: Border(left: BorderSide(color: AppColors.accent, width: 3)),
              ),
              blockquotePadding: const EdgeInsets.only(left: 12),
              listBullet: const TextStyle(color: AppColors.textMuted, fontSize: 13),
              tableHead: const TextStyle(color: AppColors.text, fontSize: 12, fontWeight: FontWeight.w600),
              tableBody: const TextStyle(color: AppColors.text, fontSize: 12),
              tableBorder: TableBorder.all(color: AppColors.border, width: 0.5),
              tableCellsPadding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
              horizontalRuleDecoration: BoxDecoration(border: Border(top: BorderSide(color: AppColors.border))),
              a: const TextStyle(color: AppColors.accent, decoration: TextDecoration.underline),
              strong: const TextStyle(color: AppColors.text, fontWeight: FontWeight.w600),
              em: const TextStyle(color: AppColors.text, fontStyle: FontStyle.italic),
            ),
          ),
        ),
      ],
    );
  }

  String _formatSize(dynamic size) {
    final bytes = size is int ? size : 0;
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }
}
