// ServerSettingsScreen mirrors the web admin's "Server" settings
// surface — the same 11 sub-sections, the same field set per
// section, the same save → restart workflow. The structure is
// driven by a declarative `_sections` table so each section
// doesn't get its own ~150-line handcrafted form; one renderer
// handles all 60+ fields by dot-path lookup into the raw config
// Map.
//
// Why Map<String, dynamic> instead of a typed Dart model: the
// backend schema has 60+ leaf fields and evolves with every
// feature PR (memory backends, vault sub-trees, etc.). Mirroring
// it in typed Dart would force every backend tweak through a
// mobile pubspec bump. The Map keeps the contract one-directional
// — backend defines, mobile renders by dot-path.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/settings_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// ── Field spec table ────────────────────────────────────────────

enum _FieldKind { text, password, switchToggle, numberInt, numberDouble, select }

class _Field {
  const _Field({
    required this.label,
    required this.path,
    required this.kind,
    this.helper,
    this.options,
    this.monospace = false,
    this.placeholder,
  });

  final String label;
  // Dot-path into the config map (e.g. 'admin.token_ttl' or
  // 'memory.local.max_seq_len').
  final String path;
  final _FieldKind kind;
  final String? helper;
  // For select kind only.
  final List<String>? options;
  final bool monospace;
  final String? placeholder;
}

class _Section {
  const _Section({
    required this.id,
    required this.title,
    required this.description,
    required this.fields,
    required this.restartRequired,
  });

  final String id;
  final String title;
  final String description;
  final List<_Field> fields;
  // True when changes to any field in this section require a
  // gateway restart to take effect. Mirrors web's
  // RESTART_REQUIRED_SECTIONS table.
  final bool restartRequired;
}

List<_Section> _buildSections() => <_Section>[
  _Section(
    id: 'general',
    title: t.settings.serverSettings.sections.general,
    description: t.settings.serverSettings.sectionDescriptions.general,
    restartRequired: true,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.listenAddress,
        path: 'listen',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: ':8770',
        helper: t.settings.serverSettings.fields.listenHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.adminUser,
        path: 'admin.user',
        kind: _FieldKind.text,
        monospace: true,
        helper: t.settings.serverSettings.fields.adminUserHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.adminPassword,
        path: 'admin.password',
        kind: _FieldKind.password,
        helper: t.settings.serverSettings.fields.adminPasswordHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.tokenTtlWeb,
        path: 'admin.token_ttl',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '24h',
        helper: t.settings.serverSettings.fields.tokenTtlHelper,
      ),
    ],
  ),
  _Section(
    id: 'logging',
    title: t.settings.serverSettings.sections.logging,
    description: t.settings.serverSettings.sectionDescriptions.logging,
    restartRequired: true,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.level,
        path: 'log.level',
        kind: _FieldKind.select,
        options: const ['debug', 'info', 'warn', 'error'],
      ),
      _Field(
        label: t.settings.serverSettings.fields.format,
        path: 'log.format',
        kind: _FieldKind.select,
        options: const ['text', 'json'],
      ),
      _Field(
        label: t.settings.serverSettings.fields.filePath,
        path: 'log.file',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '~/.opendray/logs/opendray.log',
        helper: t.settings.serverSettings.fields.filePathHelper,
      ),
    ],
  ),
  _Section(
    id: 'sessions',
    title: t.settings.serverSettings.sections.sessions,
    description: t.settings.serverSettings.sectionDescriptions.sessions,
    restartRequired: true,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.idleThreshold,
        path: 'session.idle_threshold',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '5m',
        helper: t.settings.serverSettings.fields.idleThresholdHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.idleCheckInterval,
        path: 'session.idle_interval',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '15s',
        helper: t.settings.serverSettings.fields.idleCheckHelper,
      ),
    ],
  ),
  _Section(
    id: 'vault',
    title: t.settings.serverSettings.sections.vault,
    description: t.settings.serverSettings.sectionDescriptions.vault,
    restartRequired: true,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.root,
        path: 'vault.root',
        kind: _FieldKind.text,
        monospace: true,
        helper: t.settings.serverSettings.fields.rootHelper,
        // resolveVaultPaths falls back to ~/.opendray/vault.
        placeholder: '~/.opendray/vault',
      ),
      _Field(
        label: t.settings.serverSettings.fields.notesPath,
        path: 'vault.notes',
        kind: _FieldKind.text,
        monospace: true,
        // Defaults to <root>/notes when blank.
        placeholder: '~/.opendray/vault/notes',
      ),
      _Field(
        label: t.settings.serverSettings.fields.skillsPath,
        path: 'vault.skills',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '~/.opendray/vault/skills',
      ),
      _Field(
        label: t.settings.serverSettings.fields.gitRoot,
        path: 'vault.git_root',
        kind: _FieldKind.text,
        monospace: true,
        // Defaults to vault.root (or vault.notes when notes is
        // pinned to a custom Obsidian-style location).
        placeholder: '~/.opendray/vault',
      ),
      _Field(
        label: t.settings.serverSettings.fields.personalPrefix,
        path: 'vault.personal_prefix',
        kind: _FieldKind.text,
        monospace: true,
        // No implicit fallback; left empty by design when blank.
      ),
      _Field(
        label: t.settings.serverSettings.fields.projectsPrefix,
        path: 'vault.projects_prefix',
        kind: _FieldKind.text,
        monospace: true,
      ),
    ],
  ),
  _Section(
    id: 'mcp',
    title: t.settings.serverSettings.sections.mcpRegistry,
    description: t.settings.serverSettings.sectionDescriptions.mcpRegistry,
    restartRequired: true,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.registryRoot,
        path: 'mcp.root',
        kind: _FieldKind.text,
        monospace: true,
        // resolveMCPPaths defaults to <vault root>/mcp.
        placeholder: '~/.opendray/vault/mcp',
      ),
      _Field(
        label: t.settings.serverSettings.fields.secretsFile,
        path: 'mcp.secrets_file',
        kind: _FieldKind.text,
        monospace: true,
        helper: t.settings.serverSettings.fields.secretsHelper,
        // Intentionally OUTSIDE the vault so `git add .` doesn't
        // pick it up; see resolveMCPPaths.
        placeholder: '~/.opendray/secrets.env',
      ),
    ],
  ),
  _Section(
    id: 'memory',
    title: t.settings.serverSettings.sections.memory,
    description: t.settings.serverSettings.sectionDescriptions.memory,
    restartRequired: true,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.backend,
        path: 'memory.backend',
        kind: _FieldKind.select,
        options: const ['auto', 'bm25', 'http', 'local'],
        helper: t.settings.serverSettings.fields.backendHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.store,
        path: 'memory.store',
        kind: _FieldKind.select,
        options: const ['pgvector'],
        // Backend defaults to pgvector when empty — see
        // internal/app/app.go:resolveMemoryService. We surface
        // that fallback in the picker hint so the operator
        // doesn't have to guess what blank means.
        placeholder: 'pgvector',
      ),
      _Field(
        label: t.settings.serverSettings.fields.defaultTopK,
        path: 'memory.default_top_k',
        kind: _FieldKind.numberInt,
        // memory.Service.NewOptions defaults to 5 when ≤ 0.
        placeholder: '5',
      ),
      _Field(
        label: t.settings.serverSettings.fields.similarityThreshold,
        path: 'memory.similarity_threshold',
        kind: _FieldKind.numberDouble,
        helper: t.settings.serverSettings.fields.similarityHelper,
        // memory.Service.NewOptions defaults to 0.1 when ≤ 0.
        placeholder: '0.1',
      ),
      _Field(
        label: t.settings.serverSettings.fields.defaultScope,
        path: 'memory.scope.default',
        kind: _FieldKind.select,
        options: const ['project', 'global'],
        // memory.Service.NewOptions defaults to ScopeProject. The legacy
        // "session" scope was removed in M-U Phase 1 (session ≡ project).
        placeholder: 'project',
      ),
      _Field(
        label: t.settings.serverSettings.fields.dedupThreshold,
        path: 'memory.dedup_threshold',
        kind: _FieldKind.numberDouble,
        helper: t.settings.serverSettings.fields.dedupHelper,
        placeholder: '0',
      ),
      // Background governance (config.toml gates; provider routing for
      // gatekeeper/cleaner lives in Cortex settings → Workers).
      _Field(
        label: t.settings.serverSettings.fields.gatekeeperEnabled,
        path: 'memory.gatekeeper.enabled',
        kind: _FieldKind.switchToggle,
        helper: t.settings.serverSettings.fields.gatekeeperHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.cleanerEnabled,
        path: 'memory.cleaner.enabled',
        kind: _FieldKind.switchToggle,
        helper: t.settings.serverSettings.fields.cleanerHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.knowledgeEnabled,
        path: 'knowledge.enabled',
        kind: _FieldKind.switchToggle,
        helper: t.settings.serverSettings.fields.knowledgeHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.httpBaseUrl,
        path: 'memory.http.base_url',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: 'http://localhost:11434/v1',
      ),
      _Field(
        label: t.settings.serverSettings.fields.httpModel,
        path: 'memory.http.model',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: t.settings.serverSettings.fields.httpApiKey,
        path: 'memory.http.api_key',
        kind: _FieldKind.password,
        helper: t.settings.serverSettings.fields.preserveHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.httpDimensions,
        path: 'memory.http.dimensions',
        kind: _FieldKind.numberInt,
      ),
      _Field(
        label: t.settings.serverSettings.fields.localModelName,
        path: 'memory.local.model',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: t.settings.serverSettings.fields.localLibraryPath,
        path: 'memory.local.library_path',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: t.settings.serverSettings.fields.localModelPath,
        path: 'memory.local.model_path',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: t.settings.serverSettings.fields.localTokenizerPath,
        path: 'memory.local.tokenizer_path',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: t.settings.serverSettings.fields.localMaxSeqLen,
        path: 'memory.local.max_seq_len',
        kind: _FieldKind.numberInt,
      ),
    ],
  ),
  _Section(
    id: 'backup',
    title: t.settings.serverSettings.sections.backup,
    description: t.settings.serverSettings.sectionDescriptions.backup,
    restartRequired: true,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.backupEnabled,
        path: 'backup.enabled',
        kind: _FieldKind.switchToggle,
        helper: t.settings.serverSettings.fields.backupEnabledHelper,
      ),
      _Field(
        label: t.settings.serverSettings.fields.backupLocalDir,
        path: 'backup.local_dir',
        kind: _FieldKind.text,
        monospace: true,
        // BackupConfig comment in internal/config/config.go.
        placeholder: '~/.opendray/backups',
      ),
      _Field(
        label: t.settings.serverSettings.fields.backupExportDir,
        path: 'backup.export_dir',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '~/.opendray/exports',
      ),
      _Field(
        label: t.settings.serverSettings.fields.pgDumpPath,
        path: 'backup.pg_dump_path',
        kind: _FieldKind.text,
        monospace: true,
        helper: t.settings.serverSettings.fields.pathHelper,
        // Resolved from PATH at startup when blank.
        placeholder: 'pg_dump',
      ),
      _Field(
        label: t.settings.serverSettings.fields.pgRestorePath,
        path: 'backup.pg_restore_path',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: 'pg_restore',
      ),
    ],
  ),
  _Section(
    id: 'claude',
    title: t.settings.serverSettings.sections.storageClaude,
    description: t.settings.serverSettings.sectionDescriptions.storageClaude,
    restartRequired: false,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.accountsDir,
        path: 'providers.claude.accounts_dir',
        kind: _FieldKind.text,
        monospace: true,
        helper: t.settings.serverSettings.fields.accountsHelper,
        // Auto-discovered under ~/.claude-accounts when blank.
        placeholder: '~/.claude-accounts',
      ),
    ],
  ),
  _Section(
    id: 'codex',
    title: t.settings.serverSettings.sections.storageCodex,
    description: t.settings.serverSettings.sectionDescriptions.storageCodex,
    restartRequired: false,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.sessionsRoot,
        path: 'providers.codex.sessions_root',
        kind: _FieldKind.text,
        monospace: true,
        helper: t.settings.serverSettings.fields.sessionsRootHelper,
        // session/codex_jsonl.go falls back to ~/.codex/sessions.
        placeholder: '~/.codex/sessions',
      ),
    ],
  ),
  _Section(
    id: 'gemini',
    title: t.settings.serverSettings.sections.storageGemini,
    description: t.settings.serverSettings.sectionDescriptions.storageGemini,
    restartRequired: false,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.tmpRoot,
        path: 'providers.gemini.tmp_root',
        kind: _FieldKind.text,
        monospace: true,
        // session/gemini_jsonl.go falls back to ~/.gemini/tmp.
        placeholder: '~/.gemini/tmp',
      ),
      _Field(
        label: t.settings.serverSettings.fields.projectsJson,
        path: 'providers.gemini.projects_file',
        kind: _FieldKind.text,
        monospace: true,
        // session/gemini_jsonl.go falls back to ~/.gemini/projects.json.
        placeholder: '~/.gemini/projects.json',
      ),
    ],
  ),
  _Section(
    id: 'antigravity',
    title: t.settings.serverSettings.sections.storageAntigravity,
    description: t.settings.serverSettings.sectionDescriptions.storageAntigravity,
    restartRequired: false,
    fields: [
      _Field(
        label: t.settings.serverSettings.fields.conversationsRoot,
        path: 'providers.antigravity.conversations_root',
        kind: _FieldKind.text,
        monospace: true,
        // session/antigravity_db.go falls back to
        // ~/.gemini/antigravity-cli/conversations.
        placeholder: '~/.gemini/antigravity-cli/conversations',
      ),
    ],
  ),
];

// ── Dot-path helpers ────────────────────────────────────────────

// Read a dot-path from the config map. Returns null when any step
// along the path is missing — the renderer treats null as "blank
// input" so an absent server-side key just shows up as empty.
Object? _readPath(Map<String, dynamic> root, String path) {
  Object? cur = root;
  for (final seg in path.split('.')) {
    if (cur is! Map) return null;
    cur = cur[seg];
  }
  return cur;
}

// Write a dot-path into the config map, creating intermediate
// nested maps as needed. The submit path serializes the whole
// (mutated) config back via PUT, so we want the structure intact
// even when the operator writes into a previously-null subtree.
void _writePath(Map<String, dynamic> root, String path, Object? value) {
  final segs = path.split('.');
  var cur = root;
  for (var i = 0; i < segs.length - 1; i++) {
    final seg = segs[i];
    final next = cur[seg];
    if (next is Map<String, dynamic>) {
      cur = next;
    } else {
      final fresh = <String, dynamic>{};
      cur[seg] = fresh;
      cur = fresh;
    }
  }
  cur[segs.last] = value;
}

// ── Index screen ────────────────────────────────────────────────

class ServerSettingsScreen extends ConsumerStatefulWidget {
  const ServerSettingsScreen({super.key});

  @override
  ConsumerState<ServerSettingsScreen> createState() =>
      _ServerSettingsScreenState();
}

class _ServerSettingsScreenState
    extends ConsumerState<ServerSettingsScreen> {
  AsyncValue<({Map<String, dynamic> config, String configPath})> _state =
      const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final r = await ref.read(settingsApiProvider).get();
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(r));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _confirmRestart() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.settings.serverSettings.restartConfirmTitle),
        content: Text(t.settings.serverSettings.restartConfirmBody),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: Text(t.settings.serverSettings.restart),
          ),
        ],
      ),
    );
    if (ok != true || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(settingsApiProvider).restart();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.settings.serverSettings.restartQueuedSnack),
          duration: const Duration(seconds: 3),
          behavior: SnackBarBehavior.floating,
        ),
      );
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
            content: Text(t.settings.serverSettings
                .restartFailedApi(error: e.message))),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(SnackBar(
          content: Text(t.settings.serverSettings
              .restartFailedGeneric(error: e.toString()))));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.settings.serverSettings.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.settings.serverSettings.reloadTooltip,
            onPressed: _state is AsyncLoading ? null : _load,
          ),
          IconButton(
            icon: const Icon(Icons.restart_alt),
            tooltip: t.settings.serverSettings.restartTooltip,
            onPressed: _state is AsyncLoading ? null : _confirmRestart,
          ),
        ],
      ),
      body: _state.when(
        data: _buildIndex,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
    );
  }

  Widget _buildIndex(({Map<String, dynamic> config, String configPath}) data) {
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        children: [
          if (data.configPath.isNotEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
              child: Text(
                t.settings.serverSettings.loadedFrom(path: data.configPath),
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      fontFamily: 'monospace',
                      fontSize: 11,
                    ),
              ),
            ),
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 8, 16, 12),
            child: Text(
              t.settings.serverSettings.restartHint,
              style: const TextStyle(fontSize: 12),
            ),
          ),
          for (final s in _buildSections())
            ListTile(
              title: Text(s.title),
              subtitle: Text(
                s.description,
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
                style: Theme.of(context).textTheme.bodySmall,
              ),
              trailing: Row(
                mainAxisSize: MainAxisSize.min,
                children: [
                  if (s.restartRequired)
                    Icon(
                      Icons.power_settings_new,
                      size: 14,
                      color: Theme.of(context).colorScheme.outline,
                    ),
                  const SizedBox(width: 6),
                  const Icon(Icons.chevron_right, size: 18),
                ],
              ),
              onTap: () async {
                final saved = await Navigator.of(context).push<bool>(
                  MaterialPageRoute(
                    builder: (_) => _SectionEditorScreen(
                      section: s,
                      initial: data.config,
                    ),
                  ),
                );
                if ((saved ?? false) && mounted) {
                  await _load();
                }
              },
            ),
        ],
      ),
    );
  }
}

// ── Section editor ──────────────────────────────────────────────

class _SectionEditorScreen extends ConsumerStatefulWidget {
  const _SectionEditorScreen({
    required this.section,
    required this.initial,
  });
  final _Section section;
  final Map<String, dynamic> initial;

  @override
  ConsumerState<_SectionEditorScreen> createState() =>
      _SectionEditorScreenState();
}

class _SectionEditorScreenState
    extends ConsumerState<_SectionEditorScreen> {
  // Working copy of the whole config — we PUT the full thing on
  // save (matches web behaviour and avoids backend partial-merge
  // logic). The map is deep-copied at construction so editing a
  // nested key doesn't poison the parent screen's state.
  late Map<String, dynamic> _draft;
  final Map<String, TextEditingController> _ctrls = {};
  bool _submitting = false;
  String? _error;

  @override
  void initState() {
    super.initState();
    _draft = _deepCopy(widget.initial);
    // Pre-fill controllers from the draft for text-like fields.
    for (final f in widget.section.fields) {
      if (f.kind == _FieldKind.text ||
          f.kind == _FieldKind.password ||
          f.kind == _FieldKind.numberInt ||
          f.kind == _FieldKind.numberDouble) {
        final v = _readPath(_draft, f.path);
        _ctrls[f.path] = TextEditingController(text: _stringify(v));
      }
    }
  }

  @override
  void dispose() {
    for (final c in _ctrls.values) {
      c.dispose();
    }
    super.dispose();
  }

  static Map<String, dynamic> _deepCopy(Map<String, dynamic> src) {
    final out = <String, dynamic>{};
    src.forEach((k, v) {
      if (v is Map<String, dynamic>) {
        out[k] = _deepCopy(v);
      } else if (v is List) {
        out[k] = List<dynamic>.from(v);
      } else {
        out[k] = v;
      }
    });
    return out;
  }

  static String _stringify(Object? v) {
    if (v == null) return '';
    if (v is num) return v == 0 ? '' : v.toString();
    return v.toString();
  }

  Future<void> _save() async {
    setState(() {
      _submitting = true;
      _error = null;
    });
    // Pull text-controller values back into the draft. Numeric
    // fields parse here; bad numbers surface as inline errors
    // rather than getting silently coerced.
    for (final f in widget.section.fields) {
      switch (f.kind) {
        case _FieldKind.text:
        case _FieldKind.password:
          _writePath(_draft, f.path, _ctrls[f.path]?.text ?? '');
        case _FieldKind.numberInt:
          final raw = _ctrls[f.path]?.text.trim() ?? '';
          if (raw.isEmpty) {
            _writePath(_draft, f.path, 0);
          } else {
            final parsed = int.tryParse(raw);
            if (parsed == null) {
              setState(() {
                _error = t.settings.serverSettings.validateInteger(field: f.label);
                _submitting = false;
              });
              return;
            }
            _writePath(_draft, f.path, parsed);
          }
        case _FieldKind.numberDouble:
          final raw = _ctrls[f.path]?.text.trim() ?? '';
          if (raw.isEmpty) {
            _writePath(_draft, f.path, 0.0);
          } else {
            final parsed = double.tryParse(raw);
            if (parsed == null) {
              setState(() {
                _error = t.settings.serverSettings.validateNumber(field: f.label);
                _submitting = false;
              });
              return;
            }
            _writePath(_draft, f.path, parsed);
          }
        case _FieldKind.switchToggle:
        case _FieldKind.select:
          // Already written into _draft directly on toggle / pick.
          break;
      }
    }

    try {
      await ref.read(settingsApiProvider).put(_draft);
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(
          content: Text(widget.section.restartRequired
              ? t.settings.serverSettings.savedNeedsRestart
              : t.settings.serverSettings.savedSimple),
          duration: const Duration(seconds: 3),
          behavior: SnackBarBehavior.floating,
        ),
      );
      Navigator.of(context).pop(true);
    } on ApiException catch (e) {
      if (mounted) {
        setState(() {
          _error = e.message;
          _submitting = false;
        });
      }
    } on Object catch (e) {
      if (mounted) {
        setState(() {
          _error = e.toString();
          _submitting = false;
        });
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Scaffold(
      appBar: AppBar(title: Text(widget.section.title)),
      body: SafeArea(
        bottom: false,
        child: ListView(
          padding: const EdgeInsets.fromLTRB(16, 12, 16, 24),
          children: [
            Text(
              widget.section.description,
              style: theme.textTheme.bodySmall,
            ),
            if (widget.section.restartRequired) ...[
              const SizedBox(height: 8),
              Container(
                padding: const EdgeInsets.all(8),
                decoration: BoxDecoration(
                  color:
                      theme.colorScheme.primary.withValues(alpha: 0.08),
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(
                      color: theme.colorScheme.primary.withValues(alpha: 0.3)),
                ),
                child: Row(
                  children: [
                    Icon(Icons.power_settings_new,
                        size: 14, color: theme.colorScheme.primary),
                    const SizedBox(width: 6),
                    Expanded(
                      child: Text(
                        t.settings.serverSettings.changesNeedRestart,
                        style: TextStyle(
                            fontSize: 12, color: theme.colorScheme.primary),
                      ),
                    ),
                  ],
                ),
              ),
            ],
            const SizedBox(height: 16),
            for (final f in widget.section.fields) ...[
              _renderField(f),
              const SizedBox(height: 14),
            ],
            if (_error != null) ...[
              const SizedBox(height: 6),
              Container(
                padding: const EdgeInsets.all(10),
                decoration: BoxDecoration(
                  color: theme.colorScheme.error.withValues(alpha: 0.1),
                  borderRadius: BorderRadius.circular(6),
                  border: Border.all(
                    color: theme.colorScheme.error.withValues(alpha: 0.4),
                  ),
                ),
                child: SelectableText(
                  _error!,
                  style: TextStyle(
                    color: theme.colorScheme.error,
                    fontFamily: 'monospace',
                    fontSize: 12,
                  ),
                ),
              ),
            ],
            const SizedBox(height: 20),
            Row(
              children: [
                Expanded(
                  child: OutlinedButton(
                    onPressed: _submitting
                        ? null
                        : () => Navigator.of(context).pop(false),
                    child: Text(t.common.cancel),
                  ),
                ),
                const SizedBox(width: 12),
                Expanded(
                  child: FilledButton.icon(
                    onPressed: _submitting ? null : _save,
                    icon: _submitting
                        ? const SizedBox(
                            width: 16,
                            height: 16,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.check, size: 18),
                    label: Text(_submitting ? t.settings.changeCredentials.saving : t.common.save),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _renderField(_Field f) {
    switch (f.kind) {
      case _FieldKind.text:
      case _FieldKind.password:
      case _FieldKind.numberInt:
      case _FieldKind.numberDouble:
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(f.label,
                style: const TextStyle(
                    fontSize: 12, fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            TextField(
              controller: _ctrls[f.path],
              enabled: !_submitting,
              obscureText: f.kind == _FieldKind.password,
              autocorrect: false,
              keyboardType: switch (f.kind) {
                _FieldKind.numberInt => TextInputType.number,
                _FieldKind.numberDouble =>
                  const TextInputType.numberWithOptions(decimal: true),
                _ => TextInputType.text,
              },
              decoration: InputDecoration(
                hintText: f.placeholder,
                helperText: f.helper,
                helperMaxLines: 3,
              ),
              style: f.monospace
                  ? const TextStyle(fontFamily: 'monospace', fontSize: 13)
                  : null,
            ),
          ],
        );
      case _FieldKind.switchToggle:
        return SwitchListTile.adaptive(
          value: (_readPath(_draft, f.path) as bool?) ?? false,
          onChanged: _submitting
              ? null
              : (v) => setState(() => _writePath(_draft, f.path, v)),
          title: Text(f.label),
          subtitle: f.helper != null ? Text(f.helper!) : null,
          contentPadding: EdgeInsets.zero,
        );
      case _FieldKind.select:
        final current = _readPath(_draft, f.path)?.toString() ?? '';
        final value =
            f.options?.contains(current) ?? false ? current : null;
        // When the value is empty and we know the implicit default,
        // surface it as the dropdown's hint so operators know what
        // the backend will use. Without this the blank-row UX was
        // ambiguous — looked like "nothing configured" but actually
        // "the system falls back to <something>".
        final hintText = value == null && f.placeholder != null
            ? t.settings.serverSettings.fields
                .defaultFallback(value: f.placeholder!)
            : null;
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(f.label,
                style: const TextStyle(
                    fontSize: 12, fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            DropdownButtonFormField<String>(
              initialValue: value,
              hint: hintText != null
                  ? Text(
                      hintText,
                      style: TextStyle(
                        color: Theme.of(context).colorScheme.outline,
                        fontStyle: FontStyle.italic,
                      ),
                    )
                  : null,
              decoration: InputDecoration(
                helperText: f.helper,
                helperMaxLines: 3,
              ),
              items: [
                for (final opt in f.options ?? const <String>[])
                  DropdownMenuItem<String>(
                    value: opt,
                    child: Text(opt),
                  ),
              ],
              onChanged: _submitting
                  ? null
                  : (v) => setState(() => _writePath(_draft, f.path, v ?? '')),
            ),
          ],
        );
    }
  }
}

class _ErrorView extends StatelessWidget {
  const _ErrorView({required this.error, required this.onRetry});
  final String error;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(Icons.error_outline,
                size: 40, color: Theme.of(context).colorScheme.error),
            const SizedBox(height: 8),
            Text(
              t.settings.serverSettings.loadFailed,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 4),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 12),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
