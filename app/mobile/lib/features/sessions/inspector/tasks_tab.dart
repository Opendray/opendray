import 'dart:async';
import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/custom_tasks_api.dart';
import 'package:opendray/core/api/fs_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:path/path.dart' as p;
import 'package:yaml/yaml.dart';

// Tasks surface inside the session inspector. Mirrors the web Task
// Runner: walks the cwd's fs/list and surfaces runnable tasks from
// every source the web supports —
//
//   • package.json scripts        (npm/pnpm/yarn/bun, runner auto-detected)
//   • Makefile / GNUmakefile targets
//   • Taskfile.yml / Taskfile.yaml tasks
//   • justfile / Justfile recipes
//   • Cargo.toml                  (canonical cargo commands)
//   • go.mod                      (canonical go commands)
//   • pyproject.toml              (canonical pytest/ruff/mypy + poetry scripts)
//   • shell scripts (.sh/.bash/.zsh) in cwd root and scripts/ bin/ tools/
//   • custom tasks                (/api/v1/custom-tasks, global + cwd-scoped)
//
// Manifest contents are pulled via /fs/read and parsed on-device.
// Tapping a task spawns a new shell session nested under the current
// one (cwd inherited), runs the command there, and navigates to it —
// exactly like the web Task Runner. We deliberately never paste into
// the current session: it's usually a cloud agent (claude/codex/…),
// not a shell, so the command would just land in that CLI's prompt
// instead of executing.
//
// Parsing is intentionally lightweight: we only resolve the flat
// `tasks: <name>: cmds: …` Taskfile shape, presence-detect Cargo/Go,
// and skim pyproject's `[*.scripts]` sections — the same surface the
// web panel offers.
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
  final TextEditingController _filterCtrl = TextEditingController();
  String _filter = '';

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _filterCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final fs = ref.read(fsApiProvider);
      final cwd = widget.cwd;
      final list = await fs.list(path: cwd);

      // Split the listing into files and dirs, keyed by name so the
      // rest of discovery is a series of cheap lookups.
      final files = <String, String>{}; // name -> absolute path
      final dirs = <String, String>{}; // name -> absolute path
      for (final e in list.entries) {
        if (e.isDir) {
          dirs[e.name] = e.path;
        } else {
          files[e.name] = e.path;
        }
      }
      final fileNames = files.keys.toSet();
      final runner = _pickRunner(fileNames);

      final groups = <_TaskGroup>[];

      String? pathOf(List<String> names) {
        for (final n in names) {
          final found = files[n];
          if (found != null) return found;
        }
        return null;
      }

      // Read a present manifest and parse it, isolating per-file
      // failures so one bad manifest doesn't blank the whole tab.
      Future<void> addManifest(
        List<String> names,
        _TaskSource source,
        String label,
        List<_Task> Function(String src) parse,
      ) async {
        final path = pathOf(names);
        if (path == null) return;
        try {
          final src = utf8.decode(await fs.read(path), allowMalformed: true);
          final tasks = parse(src);
          if (tasks.isNotEmpty) {
            groups.add(_TaskGroup(source: source, label: label, tasks: tasks));
          }
        } on Object {
          groups.add(_TaskGroup(
            source: source,
            label: label,
            tasks: const [],
            parseError: 'parse failed',
          ));
        }
      }

      // Custom tasks first — mirrors the web ordering (global +
      // cwd-scoped). Optional: a failure here must not blank the tab.
      try {
        final custom =
            await ref.read(customTasksApiProvider).listForCwd(cwd);
        if (custom.isNotEmpty) {
          groups.add(_TaskGroup(
            source: _TaskSource.custom,
            label: 'Custom',
            tasks: [
              for (final t in custom)
                _Task(
                  name: t.name,
                  runCommand: t.command,
                  detail: t.description.isEmpty ? null : t.description,
                ),
            ],
          ));
        }
      } on Object {
        // Custom tasks are best-effort; ignore and keep manifest tasks.
      }

      await addManifest(
        ['package.json'],
        _TaskSource.packageJson,
        'package.json · $runner',
        (src) => _parsePackageJson(src, runner),
      );
      await addManifest(
        ['Makefile', 'GNUmakefile'],
        _TaskSource.makefile,
        'Makefile',
        _parseMakefile,
      );
      await addManifest(
        ['Taskfile.yml', 'Taskfile.yaml'],
        _TaskSource.taskfile,
        'Taskfile.yml',
        _parseTaskfile,
      );
      await addManifest(
        ['justfile', 'Justfile'],
        _TaskSource.justfile,
        'justfile',
        _parseJustfile,
      );

      // Cargo / Go: no script discovery — just surface the canonical
      // set so the user can run them with one tap, like most IDEs.
      if (fileNames.contains('Cargo.toml')) {
        groups.add(_TaskGroup(
          source: _TaskSource.cargo,
          label: 'Cargo.toml',
          tasks: _cargoTasks(),
        ));
      }
      if (fileNames.contains('go.mod')) {
        groups.add(_TaskGroup(
          source: _TaskSource.goMod,
          label: 'go.mod',
          tasks: _goTasks(),
        ));
      }

      // pyproject: canonical pytest/ruff/mypy always, plus any poetry /
      // PEP-621 scripts we can skim. Canonical set shows even if the
      // file can't be read.
      if (fileNames.contains('pyproject.toml')) {
        String? src;
        try {
          src = utf8.decode(
            await fs.read(files['pyproject.toml']!),
            allowMalformed: true,
          );
        } on Object {
          src = null;
        }
        groups.add(_TaskGroup(
          source: _TaskSource.pyproject,
          label: 'pyproject.toml',
          tasks: _pyprojectTasks(src),
        ));
      }

      // Shell scripts at the cwd root.
      final rootScripts = fileNames.where(_isShellScript).toList()..sort();
      if (rootScripts.isNotEmpty) {
        groups.add(_TaskGroup(
          source: _TaskSource.scripts,
          label: 'Scripts · ./',
          tasks: [
            for (final n in rootScripts)
              _Task(name: n, runCommand: './$n'),
          ],
        ));
      }

      // Shell scripts under common script dirs (scripts/, bin/, tools/)
      // — one group per dir, capped so a huge folder can't drown the
      // panel. A missing/unreadable dir is simply skipped.
      for (final dir in _scriptDirs) {
        final dirPath = dirs[dir];
        if (dirPath == null) continue;
        try {
          final sub = await fs.list(path: dirPath);
          final scripts = sub.entries
              .where((e) => !e.isDir && _isShellScript(e.name))
              .map((e) => e.name)
              .toList()
            ..sort();
          final capped = scripts.take(_maxScriptsPerDir).toList();
          if (capped.isNotEmpty) {
            groups.add(_TaskGroup(
              source: _TaskSource.scripts,
              label: 'Scripts · $dir/',
              tasks: [
                for (final n in capped)
                  _Task(name: n, runCommand: './$dir/$n'),
              ],
            ));
          }
        } on Object {
          // Best-effort: skip dirs we can't list.
        }
      }

      // Drop groups with neither tasks nor a parse error to report.
      final nonEmpty = groups
          .where((g) => g.tasks.isNotEmpty || g.parseError != null)
          .toList();
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(nonEmpty));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  List<_TaskGroup> _applyFilter(List<_TaskGroup> groups) {
    final q = _filter.trim().toLowerCase();
    if (q.isEmpty) return groups;
    final out = <_TaskGroup>[];
    for (final g in groups) {
      final matches = g.tasks
          .where((t) =>
              t.name.toLowerCase().contains(q) ||
              t.runCommand.toLowerCase().contains(q) ||
              (t.detail ?? '').toLowerCase().contains(q))
          .toList();
      if (matches.isNotEmpty) {
        out.add(_TaskGroup(
          source: g.source,
          label: g.label,
          tasks: matches,
          parseError: g.parseError,
        ));
      }
    }
    return out;
  }

  Future<void> _onTaskTap(_Task task) async {
    final confirmed = await showModalBottomSheet<bool>(
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
                    task.runCommand,
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
              subtitle: Text(t.sessions.inspector.tasks.runCommandSubtitle),
              onTap: () => Navigator.of(sheetCtx).pop(true),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (confirmed != true || !mounted) return;
    await _runInNewShell(task);
  }

  // Run a task the way the web Task Runner does: spawn a fresh shell
  // session nested under the current one (cwd inherited), fire the
  // command into its PTY, then navigate to it. Running in a dedicated
  // shell means the command executes in zsh/bash rather than being
  // typed into whatever CLI (claude/codex/…) owns the current
  // session's prompt.
  Future<void> _runInNewShell(_Task task) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      final api = ref.read(sessionsApiProvider);
      final created = await api.create(
        CreateSessionRequest(
          providerId: 'shell',
          cwd: widget.cwd,
          name: 'task: ${task.name}',
          parentSessionId: widget.sessionId,
        ),
      );
      // The manager has the PTY up by the time create() returns; the
      // trailing newline commits the line.
      await api.input(created.id, '${task.runCommand}\n');
      ref.invalidate(sessionsListProvider);
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text('Running in new shell: ${task.runCommand}'),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      unawaited(context.push('/session/${created.id}'));
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
              final total =
                  groups.fold<int>(0, (n, g) => n + g.tasks.length);
              final visible = _applyFilter(groups);
              return Column(
                children: [
                  if (total > 0) _buildFilterField(context),
                  Expanded(
                    child: RefreshIndicator(
                      onRefresh: _load,
                      child: visible.isEmpty
                          ? ListView(
                              children: [
                                Padding(
                                  padding: const EdgeInsets.all(24),
                                  child: Center(
                                    child: Text(
                                      t.sessions.inspector.tasks
                                          .noMatch(query: _filter),
                                      style: Theme.of(context)
                                          .textTheme
                                          .bodySmall,
                                    ),
                                  ),
                                ),
                              ],
                            )
                          : ListView(
                              children: [
                                for (final g in visible)
                                  _GroupCard(group: g, onTap: _onTaskTap),
                              ],
                            ),
                    ),
                  ),
                ],
              );
            },
            loading: () => const Center(child: CircularProgressIndicator()),
            error: (e, _) => _ErrorView(error: e, onRetry: _load),
          ),
        ),
      ],
    );
  }

  Widget _buildFilterField(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 4),
      child: TextField(
        controller: _filterCtrl,
        onChanged: (v) => setState(() => _filter = v),
        decoration: InputDecoration(
          isDense: true,
          prefixIcon: const Icon(Icons.search, size: 18),
          hintText: t.sessions.inspector.tasks.filterHint,
          border: OutlineInputBorder(
            borderRadius: BorderRadius.circular(8),
          ),
          suffixIcon: _filter.isEmpty
              ? null
              : IconButton(
                  icon: const Icon(Icons.clear, size: 18),
                  tooltip: t.common.clear,
                  onPressed: () {
                    _filterCtrl.clear();
                    setState(() => _filter = '');
                  },
                ),
        ),
      ),
    );
  }
}

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
                    group.source.icon,
                    size: 16,
                    color: Theme.of(context).colorScheme.primary,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      group.label,
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
              t.sessions.inspector.tasks.emptyTitle,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              t.sessions.inspector.tasks.emptyHint,
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

// The origin of a task group, used to pick the leading icon. Mirrors
// the web panel's per-source icons.
enum _TaskSource {
  packageJson,
  makefile,
  taskfile,
  justfile,
  cargo,
  goMod,
  pyproject,
  scripts,
  custom;

  IconData get icon => switch (this) {
        _TaskSource.packageJson => Icons.javascript,
        _TaskSource.makefile => Icons.terminal,
        _TaskSource.taskfile => Icons.checklist,
        _TaskSource.justfile => Icons.bolt,
        _TaskSource.cargo => Icons.inventory_2_outlined,
        _TaskSource.goMod => Icons.inventory_2_outlined,
        _TaskSource.pyproject => Icons.coffee_outlined,
        _TaskSource.scripts => Icons.code,
        _TaskSource.custom => Icons.person_add_alt_1,
      };
}

class _Task {
  const _Task({required this.name, required this.runCommand, this.detail});
  // The task's identifier as it appears in the source file.
  final String name;
  // The command we'll paste into the terminal to run it.
  final String runCommand;
  // Optional human-readable detail (the underlying script body, a
  // Taskfile `desc:`, a custom task's description, etc.).
  final String? detail;
}

class _TaskGroup {
  _TaskGroup({
    required this.source,
    required this.label,
    required this.tasks,
    this.parseError,
  });

  final _TaskSource source;
  // Display label for the group header, e.g. "package.json · pnpm",
  // "Scripts · scripts/", "Cargo.toml", "Custom".
  final String label;
  final List<_Task> tasks;
  final String? parseError;
}

// ─── discovery helpers ───────────────────────────────────────────────

// Picks the JS package manager from the lockfile present in cwd.
// Falls back to npm when none is found. Mirrors the web panel.
String _pickRunner(Set<String> files) {
  if (files.contains('pnpm-lock.yaml')) return 'pnpm';
  if (files.contains('bun.lockb') || files.contains('bun.lock')) return 'bun';
  if (files.contains('yarn.lock')) return 'yarn';
  return 'npm';
}

// Shell-script extensions surfaced as runnable tasks. .py / .rb etc.
// are intentionally left out — they need a runtime prefix we'd have
// to guess; users can add those as custom tasks.
const List<String> _shellExts = ['.sh', '.bash', '.zsh'];

bool _isShellScript(String name) {
  final lower = name.toLowerCase();
  return _shellExts.any(lower.endsWith);
}

// Common locations to scan for shell scripts beyond the cwd root.
// Capped per directory so a scripts/ with hundreds of files doesn't
// drown the panel.
const List<String> _scriptDirs = ['scripts', 'bin', 'tools'];
const int _maxScriptsPerDir = 25;

// ─── parsers ─────────────────────────────────────────────────────────

List<_Task> _parsePackageJson(String src, String runner) {
  final tasks = <_Task>[];
  final root = jsonDecode(src);
  if (root is Map && root['scripts'] is Map) {
    final scripts = (root['scripts'] as Map).cast<String, dynamic>();
    for (final entry in scripts.entries) {
      tasks.add(_Task(
        name: entry.key,
        runCommand: '$runner run ${entry.key}',
        detail: entry.value?.toString(),
      ));
    }
  }
  return tasks;
}

// Makefile parser — finds top-level recipe lines like `target:` or
// `target: deps`. Skips .PHONY / variable assignments / recipe bodies
// / comment-only lines.
List<_Task> _parseMakefile(String src) {
  final tasks = <_Task>[];
  final seen = <String>{};
  final re = RegExp(r'^([a-zA-Z_][a-zA-Z0-9_./-]*)\s*:(?!=)');
  for (final line in src.split('\n')) {
    if (line.startsWith('\t')) continue; // recipe body, not a target
    if (line.startsWith('#')) continue;
    final m = re.firstMatch(line);
    if (m == null) continue;
    final name = m.group(1)!;
    if (name.startsWith('.')) continue; // .PHONY etc.
    if (!seen.add(name)) continue;
    tasks.add(_Task(name: name, runCommand: 'make $name'));
  }
  return tasks;
}

List<_Task> _parseTaskfile(String src) {
  final tasks = <_Task>[];
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
      tasks.add(_Task(name: name, runCommand: 'task $name', detail: desc));
    }
  }
  return tasks;
}

List<_Task> _parseJustfile(String src) {
  final tasks = <_Task>[];
  final seen = <String>{};
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
    if (!seen.add(name)) continue;
    tasks.add(_Task(name: name, runCommand: 'just $name'));
  }
  return tasks;
}

// Cargo / Go: surface the canonical command set one-tap, same as the
// web panel. No manifest parsing needed.
List<_Task> _cargoTasks() => const [
      _Task(name: 'check', runCommand: 'cargo check'),
      _Task(name: 'build', runCommand: 'cargo build'),
      _Task(name: 'test', runCommand: 'cargo test'),
      _Task(name: 'run', runCommand: 'cargo run'),
      _Task(name: 'fmt', runCommand: 'cargo fmt'),
      _Task(name: 'clippy', runCommand: 'cargo clippy'),
    ];

List<_Task> _goTasks() => const [
      _Task(name: 'build', runCommand: 'go build ./...'),
      _Task(name: 'test', runCommand: 'go test ./...'),
      _Task(name: 'test:race', runCommand: 'go test -race ./...'),
      _Task(name: 'vet', runCommand: 'go vet ./...'),
      _Task(name: 'tidy', runCommand: 'go mod tidy'),
    ];

// pyproject: canonical pytest/ruff/mypy plus any poetry / PEP-621
// scripts we can skim from the `[*.scripts]` sections.
List<_Task> _pyprojectTasks(String? src) {
  final out = <_Task>[
    const _Task(name: 'pytest', runCommand: 'pytest'),
    const _Task(name: 'ruff', runCommand: 'ruff check'),
    const _Task(name: 'mypy', runCommand: 'mypy .'),
  ];
  if (src == null) return out;
  const sections = ['tool.poetry.scripts', 'project.scripts'];
  final entry = RegExp(r'^([A-Za-z0-9_-]+)\s*=');
  var inSection = false;
  for (final line in src.split('\n')) {
    final trimmed = line.trim();
    if (trimmed.startsWith('[')) {
      inSection = sections.any((s) => trimmed == '[$s]');
      continue;
    }
    if (!inSection) continue;
    final m = entry.firstMatch(trimmed);
    if (m == null) continue;
    final name = m.group(1)!;
    out.add(_Task(name: name, runCommand: 'poetry run $name'));
  }
  return out;
}
