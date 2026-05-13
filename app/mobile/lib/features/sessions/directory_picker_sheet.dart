import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/fs_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Server-side directory picker. Renders a full-height bottom sheet
// (90% screen) with a path bar, breadcrumb-style "up" affordance,
// and a list of subdirectories. Tap a folder to enter it; tap the
// "Use this folder" sticky footer to return its path to the caller.
//
// Why server-side: the gateway often runs on a different host than
// the phone (LAN / Cloudflare tunnel). The phone can't stat the
// gateway's filesystem directly — every step round-trips through
// /api/v1/fs/list.
class DirectoryPickerSheet extends ConsumerStatefulWidget {
  const DirectoryPickerSheet({this.initialPath, super.key});

  final String? initialPath;

  static Future<String?> show(
    BuildContext context, {
    String? initialPath,
  }) {
    return showModalBottomSheet<String>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      constraints: BoxConstraints(
        maxHeight: MediaQuery.of(context).size.height * 0.92,
      ),
      builder: (_) => DirectoryPickerSheet(initialPath: initialPath),
    );
  }

  @override
  ConsumerState<DirectoryPickerSheet> createState() =>
      _DirectoryPickerSheetState();
}

class _DirectoryPickerSheetState extends ConsumerState<DirectoryPickerSheet> {
  AsyncValue<FsListResponse> _state = const AsyncValue.loading();
  String? _currentPath;
  String? _parentPath;

  @override
  void initState() {
    super.initState();
    _bootstrap();
  }

  Future<void> _bootstrap() async {
    final api = ref.read(fsApiProvider);
    try {
      // Use given path if provided AND non-empty, else server's home.
      var startPath = widget.initialPath?.trim();
      if (startPath == null || startPath.isEmpty) {
        startPath = await api.home();
      }
      await _load(startPath);
    } on ApiException catch (e) {
      if (mounted) setState(() => _state = AsyncValue.error(e, StackTrace.current));
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _load(String path) async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final res = await ref.read(fsApiProvider).list(path: path);
      if (!mounted) return;
      setState(() {
        _state = AsyncValue.data(res);
        _currentPath = res.path;
        _parentPath = res.parent.isEmpty ? null : res.parent;
      });
    } on ApiException catch (e) {
      if (mounted) setState(() => _state = AsyncValue.error(e, StackTrace.current));
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _newFolder() async {
    final parent = _currentPath;
    if (parent == null) return;
    final name = await showDialog<String>(
      context: context,
      builder: (_) => const _NewFolderDialog(),
    );
    if (name == null || name.trim().isEmpty) return;
    try {
      final created = await ref
          .read(fsApiProvider)
          .mkdir(parent: parent, name: name.trim());
      if (!mounted) return;
      // Reload + scroll to new item by navigating into it then back.
      await _load(parent);
      // Surface a confirm.
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(t.sessions.dirPicker.createdSnack(path: created))),
      );
    } on ApiException catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.dirPicker.mkdirFailedSnack(error: e.message),
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(0, 12, 0, 0),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            _SheetHandle(),
            const SizedBox(height: 8),
            _Header(
              path: _currentPath,
              canGoUp: _parentPath != null,
              onUp: _parentPath != null ? () => _load(_parentPath!) : null,
              onNewFolder: _currentPath != null ? _newFolder : null,
              onClose: () => Navigator.of(context).pop(),
            ),
            const Divider(height: 1),
            Expanded(
              child: _state.when(
                data: _Body.new,
                loading: () =>
                    const Center(child: CircularProgressIndicator()),
                error: (e, _) => _ErrorView(
                  error: e,
                  onRetry: _currentPath != null
                      ? () => _load(_currentPath!)
                      : _bootstrap,
                ),
              ),
            ),
            if (_state is AsyncData<FsListResponse>)
              _Footer(
                path: _currentPath,
                onUse: _currentPath == null
                    ? null
                    : () => Navigator.of(context).pop(_currentPath),
              ),
          ],
        ),
      ),
    );
  }
}

class _Body extends ConsumerWidget {
  const _Body(this.res);
  final FsListResponse res;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final dirs = res.entries.where((e) => e.isDir).toList();
    if (dirs.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(32),
          child: Text(
            t.sessions.dirPicker.empty,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return ListView.separated(
      itemCount: dirs.length,
      separatorBuilder: (_, __) =>
          Divider(height: 1, color: Theme.of(context).dividerColor),
      itemBuilder: (_, i) {
        final d = dirs[i];
        return ListTile(
          leading: const Icon(Icons.folder_outlined),
          title: Text(d.name),
          trailing: const Icon(Icons.chevron_right),
          onTap: () {
            // Find the picker state and navigate.
            context
                .findAncestorStateOfType<_DirectoryPickerSheetState>()
                ?._load(d.path);
          },
        );
      },
    );
  }
}

class _Header extends StatelessWidget {
  const _Header({
    required this.path,
    required this.canGoUp,
    required this.onUp,
    required this.onNewFolder,
    required this.onClose,
  });

  final String? path;
  final bool canGoUp;
  final VoidCallback? onUp;
  final VoidCallback? onNewFolder;
  final VoidCallback onClose;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(8, 4, 8, 8),
      child: Row(
        children: [
          IconButton(
            icon: const Icon(Icons.arrow_upward),
            tooltip: t.sessions.dirPicker.parent,
            onPressed: canGoUp ? onUp : null,
          ),
          Expanded(
            child: Padding(
              padding: const EdgeInsets.symmetric(horizontal: 4),
              child: Text(
                path ?? '…',
                style: Theme.of(context).textTheme.bodyMedium,
                overflow: TextOverflow.ellipsis,
                maxLines: 1,
              ),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.create_new_folder_outlined),
            tooltip: t.sessions.dirPicker.newFolder,
            onPressed: onNewFolder,
          ),
          IconButton(
            icon: const Icon(Icons.close),
            tooltip: t.common.close,
            onPressed: onClose,
          ),
        ],
      ),
    );
  }
}

class _Footer extends StatelessWidget {
  const _Footer({required this.path, required this.onUse});

  final String? path;
  final VoidCallback? onUse;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.fromLTRB(16, 12, 16, 16),
      decoration: BoxDecoration(
        color: Theme.of(context).colorScheme.surface,
        border: Border(
          top: BorderSide(color: Theme.of(context).dividerColor),
        ),
      ),
      child: SafeArea(
        top: false,
        child: Row(
          children: [
            Expanded(
              child: FilledButton.icon(
                onPressed: onUse,
                icon: const Icon(Icons.check),
                label: Text(
                  path != null
                      ? t.sessions.dirPicker.useThisFolder
                      : t.sessions.dirPicker.loading,
                  overflow: TextOverflow.ellipsis,
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}

class _NewFolderDialog extends StatefulWidget {
  const _NewFolderDialog();

  @override
  State<_NewFolderDialog> createState() => _NewFolderDialogState();
}

class _NewFolderDialogState extends State<_NewFolderDialog> {
  final _ctrl = TextEditingController();

  @override
  void dispose() {
    _ctrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      title: Text(t.sessions.dirPicker.dialog.title),
      content: TextField(
        controller: _ctrl,
        autofocus: true,
        autocorrect: false,
        decoration: InputDecoration(
          hintText: t.sessions.dirPicker.dialog.hint,
        ),
        onSubmitted: (v) => Navigator.of(context).pop(v),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: Text(t.common.cancel),
        ),
        FilledButton(
          onPressed: () => Navigator.of(context).pop(_ctrl.text),
          child: Text(t.sessions.dirPicker.dialog.create),
        ),
      ],
    );
  }
}

class _SheetHandle extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Container(
        width: 36,
        height: 4,
        decoration: BoxDecoration(
          color: Theme.of(context).dividerColor,
          borderRadius: BorderRadius.circular(2),
        ),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final Object error;
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
              Icons.folder_off_outlined,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              error.toString(),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
