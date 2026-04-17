import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../core/api/api_client.dart';
import '../core/models/provider.dart';
import 'theme/app_theme.dart';

/// Shows a directory picker backed by an installed file-browser plugin.
/// Lets the user navigate the allowed filesystem, create new folders, and
/// select a directory to use as a working directory.
///
/// Returns the absolute path of the chosen directory, or null on cancel.
Future<String?> pickDirectory(BuildContext context, {String? initialPath}) async {
  return showModalBottomSheet<String>(
    context: context,
    isScrollControlled: true,
    backgroundColor: AppColors.surface,
    shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
    builder: (_) => FractionallySizedBox(
      heightFactor: 0.82,
      child: _DirectoryPickerSheet(initialPath: initialPath),
    ),
  );
}

class _DirectoryPickerSheet extends StatefulWidget {
  final String? initialPath;
  const _DirectoryPickerSheet({this.initialPath});
  @override
  State<_DirectoryPickerSheet> createState() => _DirectoryPickerSheetState();
}

class _DirectoryPickerSheetState extends State<_DirectoryPickerSheet> {
  List<ProviderInfo> _plugins = [];
  String? _plugin;
  String _currentPath = '';
  final List<String> _pathStack = [];
  List<Map<String, dynamic>> _entries = [];
  bool _loading = true;
  String? _error;

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _init();
  }

  Future<void> _init() async {
    try {
      final all = await _api.listProviders();
      final files = all.where((p) =>
        p.provider.type == 'panel' &&
        p.provider.category == 'files' && p.enabled).toList();
      if (!mounted) return;
      setState(() {
        _plugins = files;
        if (files.isNotEmpty) _plugin = files.first.provider.name;
      });
      if (_plugin != null) _loadTree(widget.initialPath ?? '');
    } catch (e) {
      if (mounted) setState(() { _error = e.toString(); _loading = false; });
    }
  }

  Future<void> _loadTree(String path) async {
    if (_plugin == null) return;
    setState(() { _loading = true; _error = null; });
    try {
      final list = await _api.filesTree(_plugin!, path: path);
      if (!mounted) return;
      setState(() {
        _entries = list;
        _currentPath = path;
        _loading = false;
      });
    } catch (e) {
      if (mounted) setState(() { _error = e.toString(); _loading = false; });
    }
  }

  void _enter(String path) {
    _pathStack.add(_currentPath);
    _loadTree(path);
  }

  /// Computes the parent directory from the current absolute path. Returns
  /// null when we're already at root ("/", "") or there's no parent.
  String? _parentOf(String path) {
    if (path.isEmpty || path == '/') return null;
    final idx = path.lastIndexOf('/');
    if (idx < 0) return null;
    if (idx == 0) return '/';
    return path.substring(0, idx);
  }

  /// Goes up the filesystem, independent of the in-session path stack.
  /// Server-side [securePath] enforces allowedRoots — any attempt to go
  /// above the sandbox surfaces as an error banner instead of silently
  /// staying put.
  void _up() {
    final parent = _parentOf(_currentPath);
    if (parent == null) return;
    _pathStack.clear();
    _loadTree(parent);
  }

  Future<void> _createFolder() async {
    if (_plugin == null) return;
    final name = await _promptName();
    if (name == null || name.trim().isEmpty) return;
    try {
      final abs = await _api.filesMkdir(_plugin!, _currentPath, name.trim());
      if (!mounted) return;
      await _loadTree(_currentPath);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text('Created $abs'),
        duration: const Duration(seconds: 2),
      ));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text('Create failed: $e'),
        backgroundColor: AppColors.error,
      ));
    }
  }

  Future<String?> _promptName() async {
    final ctrl = TextEditingController();
    return showDialog<String>(
      context: context,
      builder: (ctx) => AlertDialog(
        backgroundColor: AppColors.surface,
        title: const Text('New Folder', style: TextStyle(fontSize: 15)),
        content: TextField(
          controller: ctrl,
          autofocus: true,
          autocorrect: false,
          enableSuggestions: false,
          textCapitalization: TextCapitalization.none,
          decoration: const InputDecoration(
            hintText: 'my-project',
            labelText: 'Folder name',
          ),
          style: const TextStyle(fontSize: 13, fontFamily: 'monospace'),
          onSubmitted: (v) => Navigator.pop(ctx, v),
        ),
        actions: [
          TextButton(
              onPressed: () => Navigator.pop(ctx), child: const Text('Cancel')),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, ctrl.text),
            style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
            child: const Text('Create'),
          ),
        ],
      ),
    );
  }

  String _shorten(String path) {
    if (path.isEmpty) return '(root)';
    final parts = path.split('/');
    if (parts.length <= 4) return path;
    return '.../${parts.sublist(parts.length - 3).join('/')}';
  }

  @override
  Widget build(BuildContext context) {
    return Column(mainAxisSize: MainAxisSize.min, children: [
      // Grab handle
      Padding(
        padding: const EdgeInsets.only(top: 8, bottom: 4),
        child: Container(
          width: 36, height: 4,
          decoration: BoxDecoration(
              color: AppColors.border, borderRadius: BorderRadius.circular(2)),
        ),
      ),
      // Header
      Padding(
        padding: const EdgeInsets.fromLTRB(16, 4, 8, 8),
        child: Row(children: [
          const Expanded(
            child: Text('Select Directory',
                style: TextStyle(fontSize: 15, fontWeight: FontWeight.w600)),
          ),
          IconButton(
            icon: const Icon(Icons.close, color: AppColors.textMuted),
            onPressed: () => Navigator.pop(context),
            tooltip: 'Cancel',
          ),
        ]),
      ),
      const Divider(height: 1, color: AppColors.border),

      // Plugin selector (only if multiple)
      if (_plugins.length > 1)
        Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
          child: DropdownButtonFormField<String>(
            initialValue: _plugin,
            decoration: const InputDecoration(
              labelText: 'Browse from',
              isDense: true,
            ),
            dropdownColor: AppColors.surfaceAlt,
            items: _plugins.map((p) => DropdownMenuItem(
              value: p.provider.name,
              child: Text(p.provider.displayName,
                  style: const TextStyle(fontSize: 13)),
            )).toList(),
            onChanged: (v) {
              if (v == null) return;
              _pathStack.clear();
              setState(() { _plugin = v; });
              _loadTree('');
            },
          ),
        ),

      // Path breadcrumb + actions
      Container(
        padding: const EdgeInsets.fromLTRB(8, 4, 8, 4),
        decoration: const BoxDecoration(
            border: Border(bottom: BorderSide(color: AppColors.border))),
        child: Row(children: [
          IconButton(
            icon: const Icon(Icons.arrow_upward, size: 18),
            onPressed: _parentOf(_currentPath) == null ? null : _up,
            tooltip: 'Up',
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
          ),
          Expanded(
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Text(
                _shorten(_currentPath),
                style: const TextStyle(
                    fontSize: 11, fontFamily: 'monospace', color: AppColors.textMuted),
              ),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.create_new_folder_outlined,
                size: 18, color: AppColors.accent),
            onPressed: _plugin == null ? null : _createFolder,
            tooltip: 'New folder',
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
          ),
        ]),
      ),

      // Entries
      Expanded(child: _buildEntries()),

      // Footer
      Container(
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 12),
        decoration: const BoxDecoration(
            border: Border(top: BorderSide(color: AppColors.border))),
        child: SafeArea(
          top: false,
          child: Row(children: [
            Expanded(child: OutlinedButton(
              onPressed: () => Navigator.pop(context),
              child: const Text('Cancel'),
            )),
            const SizedBox(width: 12),
            Expanded(flex: 2, child: FilledButton.icon(
              onPressed: _currentPath.isEmpty
                  ? null
                  : () => Navigator.pop(context, _currentPath),
              style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
              icon: const Icon(Icons.check, size: 16),
              label: const Text('Use this folder'),
            )),
          ]),
        ),
      ),
    ]);
  }

  Widget _buildEntries() {
    if (_plugins.isEmpty) {
      return const Center(child: Padding(
        padding: EdgeInsets.all(20),
        child: Text(
          'No file browser plugin enabled.\nEnable & configure one in Settings → Plugins.',
          style: TextStyle(color: AppColors.textMuted, fontSize: 12),
          textAlign: TextAlign.center,
        ),
      ));
    }
    if (_error != null) {
      return Center(child: Padding(
        padding: const EdgeInsets.all(20),
        child: Text(_error!,
            style: const TextStyle(color: AppColors.error, fontSize: 12),
            textAlign: TextAlign.center),
      ));
    }
    if (_loading) {
      return const Center(child: CircularProgressIndicator(color: AppColors.accent));
    }
    final dirs = _entries.where((e) => e['type'] == 'dir').toList();
    if (dirs.isEmpty) {
      return const Center(
        child: Text('No sub-folders here',
            style: TextStyle(color: AppColors.textMuted, fontSize: 12)),
      );
    }
    return ListView.separated(
      padding: const EdgeInsets.symmetric(vertical: 4),
      itemCount: dirs.length,
      separatorBuilder: (_, _) => const Divider(height: 1, color: AppColors.border),
      itemBuilder: (_, i) {
        final e = dirs[i];
        final name = e['name'] as String? ?? '';
        final path = e['path'] as String? ?? '';
        final isGit = e['isGit'] == true;
        return ListTile(
          dense: true,
          leading: Icon(isGit ? Icons.source : Icons.folder,
              size: 18,
              color: isGit ? AppColors.accent : AppColors.warning),
          title: Row(children: [
            Expanded(child: Text(name, style: const TextStyle(fontSize: 13))),
            if (isGit) Container(
              padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 1),
              decoration: BoxDecoration(
                  color: AppColors.accentSoft,
                  borderRadius: BorderRadius.circular(3)),
              child: const Text('git',
                  style: TextStyle(color: AppColors.accent, fontSize: 9)),
            ),
          ]),
          trailing: IconButton(
            icon: const Icon(Icons.check_circle_outline,
                size: 18, color: AppColors.accent),
            onPressed: () => Navigator.pop(context, path),
            tooltip: 'Use this folder',
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
          ),
          onTap: () => _enter(path),
        );
      },
    );
  }
}
