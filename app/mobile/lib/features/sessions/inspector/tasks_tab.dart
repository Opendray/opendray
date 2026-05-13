import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/fs_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;
import 'package:yaml/yaml.dart';

// Tasks surface inside the session inspector. Walks the cwd's
// fs/list looking for the four common task definition files
// (package.json scripts / Makefile / Taskfile.yml / justfile),
// pulls their contents via /fs/read, and parses each into a flat
// list of runnable tasks. Tapping a task offers two actions:
//
//   • Run command — pastes the command + newline, the CLI runs
//     it immediately
//   • Insert command — pastes without newline so the user can edit
//     before sending
//
// Parsing is intentionally lightweight on purpose. A full Taskfile
// schema implementation is out of scope; we only resolve the
// flat `tasks: <name>: cmds: …` shape that most projects use.
class TasksTab extends ConsumerStatefulWidget {
  const TasksTab({required this.sessionId, required this.cwd, super.key});

  final String sessionId;
  final String cwd;

  @override
  ConsumerState<TasksTab> createState() => _TasksTabState();
}

class _TasksTabState extends ConsumerState<TasksTab>
    with AutomaticKeepAliveClientMixin {
  AsyncValue<List<_TaskGroup>> _state = const AsyncValue.loading();

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final fs = ref.read(fsApiProvider);
      final list = await fs.list(path: widget.cwd);
      final groups = <_TaskGroup>[];
      for (final entry in list.entries) {
        if (entry.isDir) continue;
        final name = entry.name;
        try {
          if (name == 'package.json') {
            final bytes = await fs.read(entry.path);
            groups.add(_parsePackageJson(utf8.decode(bytes, allowMalformed: true)));
          } else if (name == 'Makefile' || name == 'GNUmakefile') {
            final bytes = await fs.read(entry.path);
            groups.add(_parseMakefile(utf8.decode(bytes, allowMalformed: true)));
          } else if (name == 'Taskfile.yml' || name == 'Taskfile.yaml') {
            final bytes = await fs.read(entry.path);
            groups.add(_parseTaskfile(utf8.decode(bytes, allowMalformed: true)));
          } else if (name == 'justfile' || name == 'Justfile') {
            final bytes = await fs.read(entry.path);
            groups.add(_parseJustfile(utf8.decode(bytes, allowMalformed: true)));
          }
        } on Object {
          // One file failing to parse shouldn't blow up the whole tab.
          // Add an empty group so the user knows we tried.
          groups.add(_TaskGroup(
            kind: _TaskKind.fromFilename(name),
            filename: name,
            tasks: const [],
            parseError: 'parse failed',
          ));
        }
      }
      // Filter out empty groups whose source file wasn't even found.
      final nonEmpty = groups.where((g) =>
          g.tasks.isNotEmpty || g.parseError != null).toList();
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(nonEmpty));
    } on ApiException catch (e) {
      if (mounted) setState(() => _state = AsyncValue.error(e, StackTrace.current));
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _onTaskTap(_Task task) async {
    final cmd = task.runCommand;
    final action = await showModalBottomSheet<_TaskAction>(
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
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(
                    task.name,
                    style: Theme.of(sheetCtx).textTheme.titleSmall,
                  ),
                  const SizedBox(height: 4),
                  SelectableText(
                    cmd,
                    style: const TextStyle(
                      fontFamily: 'monospace',
                      fontSize: 12,
                    ),
                  ),
                  if (task.detail != null) ...[
                    const SizedBox(height: 6),
                    Text(
                      task.detail!,
                      style: Theme.of(sheetCtx).textTheme.bodySmall,
                    ),
                  ],
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.play_arrow),
              title: Text(t.sessions.inspector.tasks.runCommand),
              subtitle: const Text(
                'Pastes the command and presses return — runs immediately',
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_TaskAction.run),
            ),
            ListTile(
              leading: const Icon(Icons.content_paste_go),
              title: Text(t.sessions.inspector.tasks.insertCommand),
              subtitle: Text(t.sessions.inspector.tasks.insertCommandSubtitle),
              onTap: () => Navigator.of(sheetCtx).pop(_TaskAction.insert),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    final payload = action == _TaskAction.run ? '$cmd\r' : cmd;
    try {
      await ref.read(sessionsApiProvider).input(widget.sessionId, payload);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            action == _TaskAction.run
                ? 'Running: $cmd'
                : 'Inserted: $cmd',
          ),
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

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return Column(
      children: [
        _Header(cwd: widget.cwd, onRefresh: _load),
        const Divider(height: 1),
        Expanded(
          child: _state.when(
            data: (groups) {
              if (groups.isEmpty) {
                return _EmptyView(cwd: widget.cwd);
              }
              return RefreshIndicator(
                onRefresh: _load,
                child: ListView(
                  children: [
                    for (final g in groups) _GroupCard(
                      group: g,
                      onTap: _onTaskTap,
                    ),
                  ],
                ),
              );
            },
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => _ErrorView(error: e, onRetry: _load),
          ),
        ),
      ],
    );
  }
}

enum _TaskAction { run, insert }

class _Header extends StatelessWidget {
  const _Header({required this.cwd, required this.onRefresh});
  final String cwd;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 4, 8),
      child: Row(
        children: [
          Expanded(
            child: Text(
              p.basename(cwd),
              style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
              overflow: TextOverflow.ellipsis,
            ),
          ),
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: onRefresh,
          ),
        ],
      ),
    );
  }
}

class _GroupCard extends StatelessWidget {
  const _GroupCard({required this.group, required this.onTap});

  final _TaskGroup group;
  final ValueChanged<_Task> onTap;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 12, 12, 0),
      child: Container(
        decoration: BoxDecoration(
          color: Theme.of(context).colorScheme.surface,
          border: Border.all(color: Theme.of(context).dividerColor),
          borderRadius: BorderRadius.circular(10),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.fromLTRB(14, 10, 14, 6),
              child: Row(
                children: [
                  Icon(
                    group.kind.icon,
                    size: 16,
                    color: Theme.of(context).colorScheme.primary,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      group.filename,
                      style: const TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 12,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                  ),
                  Text(
                    '${group.tasks.length} task${group.tasks.length == 1 ? '' : 's'}',
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                ],
              ),
            ),
            if (group.parseError != null)
              Padding(
                padding: const EdgeInsets.fromLTRB(14, 0, 14, 10),
                child: Text(
                  group.parseError!,
                  style: Theme.of(context).textTheme.bodySmall?.copyWith(
                        color: Theme.of(context).colorScheme.error,
                      ),
                ),
              )
            else if (group.tasks.isEmpty)
              Padding(
                padding: const EdgeInsets.fromLTRB(14, 0, 14, 10),
                child: Text(
                  '(no tasks defined)',
                  style: Theme.of(context).textTheme.bodySmall,
                ),
              )
            else
              for (var i = 0; i < group.tasks.length; i++)
                Column(
                  children: [
                    Divider(height: 1, color: Theme.of(context).dividerColor),
                    ListTile(
                      onTap: () => onTap(group.tasks[i]),
                      title: Text(
                        group.tasks[i].name,
                        style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 13,
                          fontWeight: FontWeight.w500,
                        ),
                      ),
                      subtitle: Text(
                        group.tasks[i].runCommand,
                        style: const TextStyle(
                          fontFamily: 'monospace',
                          fontSize: 11,
                        ),
                        maxLines: 2,
                        overflow: TextOverflow.ellipsis,
                      ),
                      trailing: const Icon(Icons.play_arrow),
                    ),
                  ],
                ),
          ],
        ),
      ),
    );
  }
}

class _EmptyView extends StatelessWidget {
  const _EmptyView({required this.cwd});
  final String cwd;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.5);
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.task_alt, size: 56, color: muted),
            const SizedBox(height: 12),
            Text(
              'No task files in this folder',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              'Looking for package.json / Makefile / Taskfile.yml / justfile',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
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
              Icons.error_outline,
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

// ─── data model ──────────────────────────────────────────────────────

enum _TaskKind {
  packageJson,
  makefile,
  taskfile,
  justfile;

  IconData get icon => switch (this) {
        _TaskKind.packageJson => Icons.javascript,
        _TaskKind.makefile => Icons.terminal,
        _TaskKind.taskfile => Icons.checklist,
        _TaskKind.justfile => Icons.bolt,
      };

  static _TaskKind fromFilename(String name) => switch (name) {
        'package.json' => _TaskKind.packageJson,
        'Makefile' || 'GNUmakefile' => _TaskKind.makefile,
        'Taskfile.yml' || 'Taskfile.yaml' => _TaskKind.taskfile,
        'justfile' || 'Justfile' => _TaskKind.justfile,
        _ => _TaskKind.makefile,
      };
}

class _Task {
  _Task({required this.name, required this.runCommand, this.detail});
  // The task's identifier as it appears in the source file.
  final String name;
  // The command we'll paste into the terminal to run it.
  final String runCommand;
  // Optional human-readable detail (the underlying script body, the
  // first line of a Makefile recipe, etc.).
  final String? detail;
}

class _TaskGroup {
  _TaskGroup({
    required this.kind,
    required this.filename,
    required this.tasks,
    this.parseError,
  });

  final _TaskKind kind;
  final String filename;
  final List<_Task> tasks;
  final String? parseError;
}

// ─── parsers ─────────────────────────────────────────────────────────

_TaskGroup _parsePackageJson(String src) {
  final tasks = <_Task>[];
  try {
    final root = jsonDecode(src);
    if (root is Map && root['scripts'] is Map) {
      final scripts = (root['scripts'] as Map).cast<String, dynamic>();
      for (final entry in scripts.entries) {
        final body = entry.value?.toString() ?? '';
        tasks.add(_Task(
          name: entry.key,
          runCommand: 'npm run ${entry.key}',
          detail: body,
        ));
      }
    }
  } on Object {
    return _TaskGroup(
      kind: _TaskKind.packageJson,
      filename: 'package.json',
      tasks: const [],
      parseError: 'malformed package.json',
    );
  }
  return _TaskGroup(
    kind: _TaskKind.packageJson,
    filename: 'package.json',
    tasks: tasks,
  );
}

// Makefile parser — finds top-level recipe lines like `target:` or
// `target: deps`. Skips .PHONY / variable assignments / empty lines /
// comment-only lines.
_TaskGroup _parseMakefile(String src) {
  final tasks = <_Task>[];
  final re = RegExp(r'^([a-zA-Z_][a-zA-Z0-9_./-]*)\s*:(?!=)');
  for (final line in src.split('\n')) {
    if (line.startsWith('\t')) continue; // recipe body, not a target
    if (line.startsWith('#')) continue;
    final m = re.firstMatch(line);
    if (m == null) continue;
    final name = m.group(1)!;
    if (name.startsWith('.')) continue; // .PHONY etc.
    if (tasks.any((t) => t.name == name)) continue;
    tasks.add(_Task(name: name, runCommand: 'make $name'));
  }
  return _TaskGroup(
    kind: _TaskKind.makefile,
    filename: 'Makefile',
    tasks: tasks,
  );
}

_TaskGroup _parseTaskfile(String src) {
  final tasks = <_Task>[];
  try {
    final root = loadYaml(src);
    if (root is YamlMap && root['tasks'] is YamlMap) {
      final taskMap = root['tasks'] as YamlMap;
      for (final entry in taskMap.entries) {
        final name = entry.key.toString();
        String? desc;
        final body = entry.value;
        if (body is YamlMap) {
          desc = body['desc']?.toString() ?? body['summary']?.toString();
        }
        tasks.add(_Task(
          name: name,
          runCommand: 'task $name',
          detail: desc,
        ));
      }
    }
  } on Object {
    return _TaskGroup(
      kind: _TaskKind.taskfile,
      filename: 'Taskfile.yml',
      tasks: const [],
      parseError: 'malformed Taskfile YAML',
    );
  }
  return _TaskGroup(
    kind: _TaskKind.taskfile,
    filename: 'Taskfile.yml',
    tasks: tasks,
  );
}

_TaskGroup _parseJustfile(String src) {
  final tasks = <_Task>[];
  // Recipe heads look like `name [args...]:` at column 0. Skip
  // continuation lines (indented), `set ...` directives, `alias`
  // declarations, comments.
  final re = RegExp(r'^([a-zA-Z_][a-zA-Z0-9_-]*)\s*(?:[a-zA-Z_].*)?:(?!=)');
  for (final line in src.split('\n')) {
    if (line.isEmpty) continue;
    if (line.startsWith(' ') || line.startsWith('\t')) continue;
    if (line.startsWith('#')) continue;
    if (line.startsWith('set ')) continue;
    if (line.startsWith('alias ')) continue;
    final m = re.firstMatch(line);
    if (m == null) continue;
    final name = m.group(1)!;
    if (tasks.any((t) => t.name == name)) continue;
    tasks.add(_Task(name: name, runCommand: 'just $name'));
  }
  return _TaskGroup(
    kind: _TaskKind.justfile,
    filename: 'justfile',
    tasks: tasks,
  );
}
