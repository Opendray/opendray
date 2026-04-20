import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:provider/provider.dart';

import '../core/api/api_client.dart';
import '../core/models/provider.dart';
import '../core/services/cwd_prefs.dart';
import '../core/services/l10n.dart';
import 'app_modals.dart';
import 'theme/app_theme.dart';

/// Shows a directory picker backed by an installed file-browser plugin.
/// Lets the user navigate the allowed filesystem, create new folders, and
/// select a directory to use as a working directory.
///
/// Returns the absolute path of the chosen directory, or null on cancel.
Future<String?> pickDirectory(BuildContext context, {String? initialPath}) async {
  return showAppModalBottomSheet<String>(
    context: context,
    isScrollControlled: true,
    backgroundColor: AppColors.surface,
    shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
    builder: (_) => FractionallySizedBox(
      heightFactor: 0.9,
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

  List<String> _recent = [];
  List<String> _favorites = [];
  bool _manualEntryOpen = false;
  final TextEditingController _manualCtrl = TextEditingController();

  ApiClient get _api => context.read<ApiClient>();

  @override
  void initState() {
    super.initState();
    _init();
  }

  @override
  void dispose() {
    _manualCtrl.dispose();
    super.dispose();
  }

  Future<void> _init() async {
    try {
      final results = await Future.wait([
        _api.listProviders(),
        CwdPrefs.getRecent(),
        CwdPrefs.getFavorites(),
      ]);
      final all = results[0] as List<ProviderInfo>;
      final recent = results[1] as List<String>;
      final favorites = results[2] as List<String>;
      final files = all.where((p) =>
        p.provider.type == 'panel' &&
        p.provider.category == 'files' && p.enabled).toList();
      if (!mounted) return;
      setState(() {
        _plugins = files;
        _recent = recent;
        _favorites = favorites;
        if (files.isNotEmpty) _plugin = files.first.provider.name;
      });
      if (_plugin != null) _loadTree(widget.initialPath ?? '');
    } catch (e) {
      if (mounted) setState(() { _error = e.toString(); _loading = false; });
    }
  }

  /// Confirms and returns [path] to the caller, recording it as recent.
  Future<void> _confirm(String path) async {
    if (path.isEmpty) return;
    await CwdPrefs.addRecent(path);
    if (!mounted) return;
    Navigator.pop(context, path);
  }

  Future<void> _toggleFavoriteCurrent() async {
    if (_currentPath.isEmpty) return;
    final added = await CwdPrefs.toggleFavorite(_currentPath);
    final favorites = await CwdPrefs.getFavorites();
    if (!mounted) return;
    setState(() { _favorites = favorites; });
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(
      content: Text(added ? 'Starred $_currentPath' : 'Unstarred $_currentPath'),
      duration: const Duration(seconds: 2),
    ));
  }

  Future<void> _loadTree(String path) async {
    if (_plugin == null) return;
    setState(() { _loading = true; _error = null; });
    try {
      // describeErrors unwraps DioException so we see the server's actual
      // message ("path X is outside allowed roots") instead of Dio's
      // multi-paragraph 400 lecture.
      final list = await ApiClient.describeErrors(
          () => _api.filesTree(_plugin!, path: path));
      if (!mounted) return;
      setState(() {
        _entries = list;
        _currentPath = path;
        _loading = false;
      });
    } catch (e) {
      // Leave _currentPath pointing at the previous good folder so the
      // breadcrumb / Up / Star / New-folder actions keep working — the
      // user just gets an inline banner explaining the failed request.
      if (mounted) setState(() { _error = _friendlyError(e); _loading = false; });
    }
  }

  /// Jumps back to the plugin's default root. The server resolves an empty
  /// path to [BrowserConfig.DefaultPath] or the first allowed root, so this
  /// is the safest "get me out of here" escape hatch.
  Future<void> _goHome() async {
    _pathStack.clear();
    await _loadTree('');
  }

  /// Reduces API exceptions (and anything else) to a single readable line
  /// for the inline error banner.
  String _friendlyError(Object e) {
    if (e is ApiException) {
      // Strip the package prefix the backend likes ("files: …") so the
      // banner reads cleanly.
      final m = e.message.replaceFirst(RegExp(r'^(files:|git:|docs:)\s*'), '');
      return m.isNotEmpty ? m : 'Request failed (${e.statusCode})';
    }
    final s = e.toString();
    // Fallback: just show the first line to avoid multi-paragraph dumps.
    final nl = s.indexOf('\n');
    return nl > 0 ? s.substring(0, nl) : s;
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

  /// Creates a new folder under the current path and immediately returns
  /// its absolute path to the caller. This collapses the old 5-tap flow
  /// (create → reload → locate → confirm) into a single "create & use" action.
  Future<void> _createFolder() async {
    if (_plugin == null) return;
    final name = await _promptName();
    if (name == null || name.trim().isEmpty) return;
    try {
      final abs = await ApiClient.describeErrors(
          () => _api.filesMkdir(_plugin!, _currentPath, name.trim()));
      if (!mounted) return;
      await CwdPrefs.addRecent(abs);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text('Created $abs'),
        duration: const Duration(seconds: 2),
      ));
      Navigator.pop(context, abs);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(SnackBar(
        content: Text('Create failed: ${_friendlyError(e)}'),
        backgroundColor: AppColors.error,
      ));
    }
  }

  /// Prompts for a new folder name via a keyboard-aware bottom sheet.
  ///
  /// The previous implementation used an AlertDialog, which on Android was
  /// re-centered on every keyboard animation tick. That combined with the
  /// parent bottom sheet caused the dialog to visibly shrink after 1–2
  /// keystrokes and drop focus. A bottom sheet anchored to
  /// MediaQuery.viewInsets.bottom is stable across the IME show animation
  /// and gives the TextField plenty of room.
  Future<String?> _promptName() async {
    final ctrl = TextEditingController();
    return showAppModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      backgroundColor: AppColors.surface,
      shape: const RoundedRectangleBorder(
          borderRadius: BorderRadius.vertical(top: Radius.circular(16))),
      builder: (ctx) => _NewFolderSheet(
        parentPath: _currentPath,
        controller: ctrl,
      ),
    );
  }

  /// Tries to jump directly to [path] (from manual input or a pasted path).
  /// Server enforces allowedRoots; any rejection surfaces as an error banner.
  Future<void> _goToManualPath() async {
    final raw = _manualCtrl.text.trim();
    if (raw.isEmpty) return;
    // Tolerate typical paste errors: trailing slash, whitespace, file://
    var p = raw.replaceFirst(RegExp(r'^file://'), '');
    if (p.length > 1 && p.endsWith('/')) p = p.substring(0, p.length - 1);
    FocusScope.of(context).unfocus();
    setState(() { _manualEntryOpen = false; });
    _pathStack.clear();
    await _loadTree(p);
  }

  /// Walks the current path into segments that render as tappable breadcrumbs.
  /// Each segment carries the full path up to and including that segment so
  /// one tap jumps us directly there, no multi-step _up().
  List<({String label, String path})> _breadcrumbs(String path) {
    if (path.isEmpty) return [(label: '(root)', path: '')];
    final out = <({String label, String path})>[];
    final parts = path.split('/');
    var acc = '';
    for (final part in parts) {
      if (part.isEmpty) {
        out.add((label: '/', path: '/'));
        continue;
      }
      acc = acc.isEmpty || acc == '/' ? '$acc$part' : '$acc/$part';
      if (!acc.startsWith('/')) acc = '/$acc';
      out.add((label: part, path: acc));
    }
    return out;
  }

  @override
  Widget build(BuildContext context) {
    // Anchor everything to viewInsets.bottom so the manual-path input stays
    // above the keyboard when it opens.
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;
    return Padding(
      padding: EdgeInsets.only(bottom: bottomInset),
      child: Column(mainAxisSize: MainAxisSize.min, children: [
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
            icon: Icon(
              _manualEntryOpen ? Icons.close : Icons.edit_outlined,
              color: AppColors.textMuted,
              size: 20,
            ),
            onPressed: () {
              setState(() {
                _manualEntryOpen = !_manualEntryOpen;
                if (_manualEntryOpen && _manualCtrl.text.isEmpty) {
                  _manualCtrl.text = _currentPath;
                }
              });
            },
            tooltip: _manualEntryOpen ? 'Hide manual entry' : 'Type a path',
          ),
          IconButton(
            icon: const Icon(Icons.close, color: AppColors.textMuted),
            onPressed: () => Navigator.pop(context),
            tooltip: 'Cancel',
          ),
        ]),
      ),
      const Divider(height: 1, color: AppColors.border),

      // Manual path entry — collapsed by default to keep the normal
      // browse-and-tap flow clean, but one tap away for power users.
      if (_manualEntryOpen) _buildManualEntry(),

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
              child: Text(
                context.pickL10n(p.provider.displayName, p.provider.displayNameZh),
                style: const TextStyle(fontSize: 13),
              ),
            )).toList(),
            onChanged: (v) {
              if (v == null) return;
              _pathStack.clear();
              setState(() { _plugin = v; });
              _loadTree('');
            },
          ),
        ),

      // Quick-access chips (Recent / Starred). Hidden when both empty.
      _buildQuickAccess(),

      // Tappable breadcrumb — each segment jumps straight to that ancestor.
      Container(
        padding: const EdgeInsets.fromLTRB(4, 6, 4, 6),
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
          IconButton(
            icon: const Icon(Icons.home_outlined, size: 18),
            onPressed: _plugin == null ? null : _goHome,
            tooltip: 'Default root',
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
          ),
          Expanded(child: _buildBreadcrumb()),
          IconButton(
            icon: Icon(
              _favorites.contains(_currentPath)
                  ? Icons.star
                  : Icons.star_border,
              size: 18,
              color: _favorites.contains(_currentPath)
                  ? AppColors.accent
                  : AppColors.textMuted,
            ),
            onPressed: _currentPath.isEmpty ? null : _toggleFavoriteCurrent,
            tooltip: _favorites.contains(_currentPath) ? 'Unstar' : 'Star',
            padding: EdgeInsets.zero,
            constraints: const BoxConstraints(minWidth: 36, minHeight: 36),
          ),
          IconButton(
            icon: const Icon(Icons.create_new_folder_outlined,
                size: 18, color: AppColors.accent),
            onPressed: _plugin == null ? null : _createFolder,
            tooltip: 'New folder & use',
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
                  : () => _confirm(_currentPath),
              style: FilledButton.styleFrom(backgroundColor: AppColors.accent),
              icon: const Icon(Icons.check, size: 16),
              label: const Text('Use this folder'),
            )),
          ]),
        ),
      ),
    ]),
    );
  }

  Widget _buildManualEntry() {
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 10, 12, 10),
      decoration: const BoxDecoration(
        color: AppColors.surfaceAlt,
        border: Border(bottom: BorderSide(color: AppColors.border)),
      ),
      child: Row(children: [
        Expanded(
          child: TextField(
            controller: _manualCtrl,
            autofocus: true,
            autocorrect: false,
            enableSuggestions: false,
            textCapitalization: TextCapitalization.none,
            style: const TextStyle(fontSize: 13, fontFamily: 'monospace'),
            decoration: const InputDecoration(
              isDense: true,
              hintText: '/absolute/path',
              prefixIcon: Icon(Icons.terminal, size: 16),
              contentPadding: EdgeInsets.symmetric(horizontal: 8, vertical: 10),
            ),
            onSubmitted: (_) => _goToManualPath(),
          ),
        ),
        const SizedBox(width: 8),
        IconButton(
          icon: const Icon(Icons.content_paste, size: 18),
          tooltip: 'Paste',
          onPressed: () async {
            final data = await Clipboard.getData('text/plain');
            final t = data?.text;
            if (t == null || t.isEmpty) return;
            _manualCtrl.text = t.trim();
          },
        ),
        FilledButton.icon(
          onPressed: _goToManualPath,
          style: FilledButton.styleFrom(
              backgroundColor: AppColors.accent,
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10)),
          icon: const Icon(Icons.arrow_forward, size: 16),
          label: const Text('Go'),
        ),
      ]),
    );
  }

  Widget _buildBreadcrumb() {
    final crumbs = _breadcrumbs(_currentPath);
    return SingleChildScrollView(
      scrollDirection: Axis.horizontal,
      reverse: true, // keep the rightmost (current) segment visible by default
      child: Row(
        children: [
          for (int i = 0; i < crumbs.length; i++) ...[
            if (i > 0)
              const Padding(
                padding: EdgeInsets.symmetric(horizontal: 2),
                child: Icon(Icons.chevron_right,
                    size: 14, color: AppColors.textMuted),
              ),
            InkWell(
              borderRadius: BorderRadius.circular(4),
              onTap: crumbs[i].path == _currentPath
                  ? null
                  : () {
                      _pathStack.clear();
                      _loadTree(crumbs[i].path);
                    },
              child: Padding(
                padding: const EdgeInsets.symmetric(
                    horizontal: 6, vertical: 4),
                child: Text(
                  crumbs[i].label,
                  style: TextStyle(
                    fontSize: 12,
                    fontFamily: 'monospace',
                    color: crumbs[i].path == _currentPath
                        ? AppColors.accent
                        : AppColors.textMuted,
                    fontWeight: crumbs[i].path == _currentPath
                        ? FontWeight.w600
                        : FontWeight.normal,
                  ),
                ),
              ),
            ),
          ],
        ],
      ),
    );
  }

  /// Two optional horizontal chip rows above the browser. Recent chips
  /// confirm-return on tap (single-tap reuse); Starred chips navigate into
  /// the folder so users can browse or create inside a curated root.
  Widget _buildQuickAccess() {
    if (_plugins.isEmpty) return const SizedBox.shrink();
    if (_recent.isEmpty && _favorites.isEmpty) return const SizedBox.shrink();
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 6, 12, 6),
      decoration: const BoxDecoration(
          border: Border(bottom: BorderSide(color: AppColors.border))),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        mainAxisSize: MainAxisSize.min,
        children: [
          if (_favorites.isNotEmpty)
            _quickRow(
              icon: Icons.star,
              iconColor: AppColors.accent,
              label: 'Starred',
              paths: _favorites,
              onTap: _loadTree,
            ),
          if (_favorites.isNotEmpty && _recent.isNotEmpty)
            const SizedBox(height: 4),
          if (_recent.isNotEmpty)
            _quickRow(
              icon: Icons.history,
              iconColor: AppColors.textMuted,
              label: 'Recent',
              paths: _recent,
              onTap: _confirm,
            ),
        ],
      ),
    );
  }

  Widget _quickRow({
    required IconData icon,
    required Color iconColor,
    required String label,
    required List<String> paths,
    required void Function(String) onTap,
  }) {
    return Row(
      crossAxisAlignment: CrossAxisAlignment.center,
      children: [
        Icon(icon, size: 13, color: iconColor),
        const SizedBox(width: 4),
        SizedBox(
          width: 52,
          child: Text(label,
              style: const TextStyle(
                  fontSize: 10, color: AppColors.textMuted)),
        ),
        Expanded(
          child: SingleChildScrollView(
            scrollDirection: Axis.horizontal,
            child: Row(
              children: paths
                  .map((p) => Padding(
                        padding: const EdgeInsets.only(right: 6),
                        child: _pathChip(p, onTap: () => onTap(p)),
                      ))
                  .toList(),
            ),
          ),
        ),
      ],
    );
  }

  Widget _pathChip(String path, {required VoidCallback onTap}) {
    final label = _chipLabel(path);
    return InkWell(
      borderRadius: BorderRadius.circular(12),
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: AppColors.surfaceAlt,
          border: Border.all(color: AppColors.border),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Text(label,
            style: const TextStyle(
                fontSize: 11, fontFamily: 'monospace', color: AppColors.text)),
      ),
    );
  }

  /// Compact label for chips: last two path segments, prefixed with "…/"
  /// when the path is deeper than that.
  String _chipLabel(String path) {
    if (path.isEmpty || path == '/') return '/';
    final parts = path.split('/').where((s) => s.isNotEmpty).toList();
    if (parts.length <= 2) return '/${parts.join('/')}';
    return '…/${parts.sublist(parts.length - 2).join('/')}';
  }

  /// Readable error card with explicit escape hatches so the user is never
  /// stranded on a failed-load screen.
  Widget _errorState(String msg) {
    return Center(
      child: SingleChildScrollView(
        padding: const EdgeInsets.all(20),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Container(
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: AppColors.errorSoft,
                borderRadius: BorderRadius.circular(10),
                border: Border.all(color: AppColors.error.withValues(alpha: 0.4)),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  const Icon(Icons.error_outline,
                      size: 18, color: AppColors.error),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Text(
                      msg,
                      style: const TextStyle(
                          color: AppColors.error, fontSize: 12, height: 1.4),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            Wrap(
              spacing: 10,
              runSpacing: 10,
              alignment: WrapAlignment.center,
              children: [
                if (_currentPath.isNotEmpty)
                  OutlinedButton.icon(
                    onPressed: () {
                      // Re-load the current (last good) folder — clears the
                      // error banner without moving.
                      setState(() => _error = null);
                      _loadTree(_currentPath);
                    },
                    icon: const Icon(Icons.refresh, size: 16),
                    label: Text('Back to ${_shortLabel(_currentPath)}'),
                  ),
                FilledButton.icon(
                  onPressed: _plugin == null ? null : _goHome,
                  style: FilledButton.styleFrom(
                    backgroundColor: AppColors.accent,
                  ),
                  icon: const Icon(Icons.home_outlined, size: 16),
                  label: const Text('Go home'),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  /// One-line label for the "back to X" button — last two segments so it
  /// fits in a button without ellipsis.
  String _shortLabel(String path) {
    if (path.isEmpty || path == '/') return 'root';
    final parts = path.split('/').where((s) => s.isNotEmpty).toList();
    if (parts.isEmpty) return 'root';
    if (parts.length == 1) return parts.first;
    return '…/${parts.last}';
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
      return _errorState(_error!);
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
        // Bigger, non-dense row — easier tap target on phones. Full-width
        // tap enters the folder; explicit check icon selects it.
        return ListTile(
          leading: Icon(isGit ? Icons.source : Icons.folder,
              size: 22,
              color: isGit ? AppColors.accent : AppColors.warning),
          title: Row(children: [
            Expanded(
              child: Text(name,
                  style: const TextStyle(
                      fontSize: 14, fontWeight: FontWeight.w500)),
            ),
            if (isGit)
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 5, vertical: 1),
                decoration: BoxDecoration(
                    color: AppColors.accentSoft,
                    borderRadius: BorderRadius.circular(3)),
                child: const Text('git',
                    style: TextStyle(color: AppColors.accent, fontSize: 9)),
              ),
          ]),
          trailing: IconButton(
            icon: const Icon(Icons.check_circle_outline,
                size: 22, color: AppColors.accent),
            onPressed: () => _confirm(path),
            tooltip: 'Use this folder',
          ),
          onTap: () => _enter(path),
        );
      },
    );
  }
}

/// Keyboard-aware "New Folder" prompt. Anchored to
/// MediaQuery.viewInsets.bottom so the input stays above the IME.
///
/// Shown as a `showAppModalBottomSheet` child — when popped with a
/// non-empty string, the caller treats that as the chosen folder name.
class _NewFolderSheet extends StatefulWidget {
  final String parentPath;
  final TextEditingController controller;
  const _NewFolderSheet({
    required this.parentPath,
    required this.controller,
  });

  @override
  State<_NewFolderSheet> createState() => _NewFolderSheetState();
}

class _NewFolderSheetState extends State<_NewFolderSheet> {
  final FocusNode _focus = FocusNode();

  @override
  void initState() {
    super.initState();
    // Give the sheet one frame to settle before raising the keyboard —
    // opening the IME too early collides with the sheet's entrance
    // animation on some Android skins.
    WidgetsBinding.instance.addPostFrameCallback((_) => _focus.requestFocus());
  }

  @override
  void dispose() {
    _focus.dispose();
    super.dispose();
  }

  void _submit() {
    final v = widget.controller.text.trim();
    if (v.isEmpty) return;
    Navigator.pop(context, v);
  }

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;
    return Padding(
      padding: EdgeInsets.only(bottom: bottomInset),
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 14),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Grab handle
              Center(
                child: Container(
                  width: 36,
                  height: 4,
                  margin: const EdgeInsets.only(bottom: 10),
                  decoration: BoxDecoration(
                    color: AppColors.border,
                    borderRadius: BorderRadius.circular(2),
                  ),
                ),
              ),
              Row(children: [
                const Icon(Icons.create_new_folder_outlined,
                    size: 18, color: AppColors.accent),
                const SizedBox(width: 8),
                const Expanded(
                  child: Text('New Folder',
                      style: TextStyle(
                          fontSize: 15, fontWeight: FontWeight.w600)),
                ),
                IconButton(
                  icon: const Icon(Icons.close, size: 18),
                  onPressed: () => Navigator.pop(context),
                  padding: EdgeInsets.zero,
                  constraints:
                      const BoxConstraints(minWidth: 32, minHeight: 32),
                ),
              ]),
              const SizedBox(height: 4),
              Text(
                'Creates under ${widget.parentPath.isEmpty ? "(root)" : widget.parentPath}',
                style: const TextStyle(
                    fontSize: 11,
                    fontFamily: 'monospace',
                    color: AppColors.textMuted),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: widget.controller,
                focusNode: _focus,
                autocorrect: false,
                enableSuggestions: false,
                textCapitalization: TextCapitalization.none,
                textInputAction: TextInputAction.done,
                onSubmitted: (_) => _submit(),
                // Block path separators and characters that would confuse the
                // backend; the server also validates, but pre-filtering keeps
                // the UX honest.
                inputFormatters: [
                  FilteringTextInputFormatter.deny(RegExp(r'[/\\\n\r\t]')),
                ],
                style: const TextStyle(fontSize: 15, fontFamily: 'monospace'),
                decoration: const InputDecoration(
                  hintText: 'my-project',
                  labelText: 'Folder name',
                  filled: true,
                  fillColor: AppColors.surfaceAlt,
                  contentPadding:
                      EdgeInsets.symmetric(horizontal: 12, vertical: 14),
                  border: OutlineInputBorder(
                    borderRadius: BorderRadius.all(Radius.circular(10)),
                    borderSide: BorderSide.none,
                  ),
                ),
              ),
              const SizedBox(height: 14),
              Row(children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: () => Navigator.pop(context),
                    style: OutlinedButton.styleFrom(
                      padding: const EdgeInsets.symmetric(vertical: 14),
                    ),
                    child: const Text('Cancel'),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  flex: 2,
                  child: ValueListenableBuilder<TextEditingValue>(
                    valueListenable: widget.controller,
                    builder: (context, value, _) {
                      final enabled = value.text.trim().isNotEmpty;
                      return FilledButton.icon(
                        onPressed: enabled ? _submit : null,
                        style: FilledButton.styleFrom(
                          backgroundColor: AppColors.accent,
                          padding: const EdgeInsets.symmetric(vertical: 14),
                        ),
                        icon: const Icon(Icons.check, size: 16),
                        label: const Text('Create & use'),
                      );
                    },
                  ),
                ),
              ]),
            ],
          ),
        ),
      ),
    );
  }
}
