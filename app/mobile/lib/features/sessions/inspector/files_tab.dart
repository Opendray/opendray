import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/fs_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;

// Files surface inside the session inspector. Browses the gateway-
// host filesystem rooted at session.cwd by default, with the option
// to ascend (the user might want to grep a sibling project, or
// peek at /etc on a self-hosted box). Tapping a file opens a
// bottom-sheet of "push to terminal" actions plus a quick-view
// for small text files.
class FilesTab extends ConsumerStatefulWidget {
  const FilesTab({
    required this.sessionId,
    required this.initialPath,
    super.key,
  });

  final String sessionId;
  final String initialPath;

  @override
  ConsumerState<FilesTab> createState() => _FilesTabState();
}

class _FilesTabState extends ConsumerState<FilesTab>
    with AutomaticKeepAliveClientMixin {
  AsyncValue<FsListResponse> _state = const AsyncValue.loading();
  String? _currentPath;
  String? _parentPath;

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _load(widget.initialPath);
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

  Future<void> _onFileTap(FsEntry entry) async {
    final action = await showModalBottomSheet<_FileAction>(
      context: context,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (sheetCtx) => SafeArea(
        top: false,
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 8),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    entry.name,
                    style: Theme.of(sheetCtx).textTheme.titleSmall,
                    overflow: TextOverflow.ellipsis,
                  ),
                  const SizedBox(height: 2),
                  Text(
                    entry.path,
                    style: Theme.of(sheetCtx).textTheme.bodySmall,
                    overflow: TextOverflow.ellipsis,
                    maxLines: 2,
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.alternate_email),
              title: Text(t.sessions.inspector.files.insertAtRef),
              subtitle: const Text(
                'Pastes "@<path>" into the running prompt',
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_FileAction.insertAt),
            ),
            ListTile(
              leading: const Icon(Icons.content_paste_go),
              title: Text(t.sessions.inspector.files.insertPath),
              subtitle: Text(t.sessions.inspector.files.insertPathSubtitle),
              onTap: () => Navigator.of(sheetCtx).pop(_FileAction.insertPath),
            ),
            ListTile(
              leading: const Icon(Icons.text_snippet_outlined),
              title: Text(t.sessions.inspector.files.readContent),
              subtitle: Text(t.sessions.inspector.files.readContentSubtitle),
              onTap: () => Navigator.of(sheetCtx).pop(_FileAction.read),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    switch (action) {
      case _FileAction.insertAt:
        await _pushInput('@${entry.path}');
      case _FileAction.insertPath:
        await _pushInput(entry.path);
      case _FileAction.read:
        await _showFile(entry);
    }
  }

  Future<void> _pushInput(String text) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(sessionsApiProvider).input(widget.sessionId, text);
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.sessions.inspector.shared.inserted(text: text)),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.shared.insertFailedApi(
              status: e.statusCode.toString(),
              message: e.message,
            ),
          ),
        ),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.shared.insertFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  Future<void> _showFile(FsEntry entry) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      final bytes = await ref.read(fsApiProvider).read(entry.path);
      final text = utf8.decode(bytes, allowMalformed: true);
      if (!mounted) return;
      await showDialog<void>(
        context: context,
        builder: (dialogCtx) => Dialog(
          insetPadding: const EdgeInsets.all(16),
          child: ConstrainedBox(
            constraints: BoxConstraints(
              maxHeight: MediaQuery.of(dialogCtx).size.height * 0.8,
            ),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                Padding(
                  padding: const EdgeInsets.fromLTRB(16, 12, 8, 8),
                  child: Row(
                    children: [
                      Expanded(
                        child: Text(
                          entry.name,
                          style: Theme.of(dialogCtx).textTheme.titleSmall,
                          overflow: TextOverflow.ellipsis,
                        ),
                      ),
                      IconButton(
                        icon: const Icon(Icons.close),
                        onPressed: () => Navigator.of(dialogCtx).pop(),
                      ),
                    ],
                  ),
                ),
                const Divider(height: 1),
                Flexible(
                  child: SingleChildScrollView(
                    padding: const EdgeInsets.all(12),
                    child: SelectableText(
                      text,
                      style: const TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 12,
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      );
    } on ApiException catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.files.readFailedApi(
              status: e.statusCode.toString(),
              message: e.message,
            ),
          ),
        ),
      );
    } on Object catch (e) {
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.sessions.inspector.files.readFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return Column(
      children: [
        _PathBar(
          path: _currentPath,
          canGoUp: _parentPath != null,
          onUp: _parentPath != null ? () => _load(_parentPath!) : null,
          onRefresh:
              _currentPath != null ? () => _load(_currentPath!) : null,
          onGoToCwd: () => _load(widget.initialPath),
        ),
        const Divider(height: 1),
        Expanded(
          child: _state.when(
            data: (res) {
              if (res.entries.isEmpty) {
                return Center(
                  child: Padding(
                    padding: const EdgeInsets.all(32),
                    child: Text(
                      'Empty folder',
                      style: Theme.of(context).textTheme.bodyMedium,
                    ),
                  ),
                );
              }
              return ListView.separated(
                itemCount: res.entries.length,
                separatorBuilder: (_, __) => Divider(
                  height: 1,
                  color: Theme.of(context).dividerColor,
                ),
                itemBuilder: (_, i) {
                  final e = res.entries[i];
                  return ListTile(
                    leading: Icon(
                      e.isDir
                          ? Icons.folder_outlined
                          : Icons.insert_drive_file_outlined,
                      color: e.isDir
                          ? Theme.of(context).colorScheme.primary
                          : Theme.of(context)
                              .colorScheme
                              .onSurface
                              .withValues(alpha: 0.7),
                    ),
                    title: Text(e.name, overflow: TextOverflow.ellipsis),
                    trailing: e.isDir
                        ? const Icon(Icons.chevron_right)
                        : null,
                    onTap: e.isDir
                        ? () => _load(e.path)
                        : () => _onFileTap(e),
                  );
                },
              );
            },
            loading: () =>
                const Center(child: CircularProgressIndicator()),
            error: (e, _) => _ErrorView(
              error: e,
              onRetry: _currentPath != null
                  ? () => _load(_currentPath!)
                  : () => _load(widget.initialPath),
            ),
          ),
        ),
      ],
    );
  }
}

enum _FileAction { insertAt, insertPath, read }

class _PathBar extends StatelessWidget {
  const _PathBar({
    required this.path,
    required this.canGoUp,
    required this.onUp,
    required this.onRefresh,
    required this.onGoToCwd,
  });

  final String? path;
  final bool canGoUp;
  final VoidCallback? onUp;
  final VoidCallback? onRefresh;
  final VoidCallback onGoToCwd;

  @override
  Widget build(BuildContext context) {
    final shown = path != null ? p.basename(path!) : '…';
    return Padding(
      padding: const EdgeInsets.fromLTRB(4, 4, 4, 4),
      child: Row(
        children: [
          IconButton(
            icon: const Icon(Icons.arrow_upward),
            tooltip: t.sessions.inspector.files.parent,
            onPressed: canGoUp ? onUp : null,
            visualDensity: VisualDensity.compact,
          ),
          IconButton(
            icon: const Icon(Icons.home_outlined),
            tooltip: t.sessions.inspector.files.backToCwd,
            onPressed: onGoToCwd,
            visualDensity: VisualDensity.compact,
          ),
          Expanded(
            child: Tooltip(
              message: path ?? '',
              child: Text(
                shown.isEmpty ? '/' : shown,
                style: Theme.of(context).textTheme.bodyMedium,
                overflow: TextOverflow.ellipsis,
              ),
            ),
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: onRefresh,
            visualDensity: VisualDensity.compact,
          ),
        ],
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
