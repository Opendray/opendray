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
    description: 'Listen address, operator account, token TTL.',
    restartRequired: true,
    fields: const [
      _Field(
        label: 'Listen address',
        path: 'listen',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: ':8770',
        helper: 'host:port the gateway binds to. Restart required.',
      ),
      _Field(
        label: 'Admin user',
        path: 'admin.user',
        kind: _FieldKind.text,
        monospace: true,
        helper:
            'Effective when no keyfile or env var is set. Otherwise see Settings → Account.',
      ),
      _Field(
        label: 'Admin password',
        path: 'admin.password',
        kind: _FieldKind.password,
        helper:
            'Send blank to preserve. For ongoing rotations use Settings → Account (keyfile-backed, no restart).',
      ),
      _Field(
        label: 'Token TTL (web)',
        path: 'admin.token_ttl',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '24h',
        helper: 'Go duration string, e.g. 24h, 30m.',
      ),
    ],
  ),
  _Section(
    id: 'logging',
    title: t.settings.serverSettings.sections.logging,
    description: 'Verbosity, format, and on-disk log path.',
    restartRequired: true,
    fields: const [
      _Field(
        label: 'Level',
        path: 'log.level',
        kind: _FieldKind.select,
        options: ['debug', 'info', 'warn', 'error'],
      ),
      _Field(
        label: 'Format',
        path: 'log.format',
        kind: _FieldKind.select,
        options: ['text', 'json'],
      ),
      _Field(
        label: 'File path',
        path: 'log.file',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '~/.opendray/logs/opendray.log',
        helper: 'Empty = stdout only.',
      ),
    ],
  ),
  _Section(
    id: 'sessions',
    title: t.settings.serverSettings.sections.sessions,
    description: 'Idle detection thresholds.',
    restartRequired: true,
    fields: const [
      _Field(
        label: 'Idle threshold',
        path: 'session.idle_threshold',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '5m',
        helper:
            'Quiet period before a session is flagged idle. Go duration.',
      ),
      _Field(
        label: 'Idle check interval',
        path: 'session.idle_interval',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: '15s',
        helper: 'How often the idle reaper runs.',
      ),
    ],
  ),
  _Section(
    id: 'vault',
    title: t.settings.serverSettings.sections.vault,
    description: 'Notes, skills, and git-versioned root.',
    restartRequired: true,
    fields: const [
      _Field(
        label: 'Root',
        path: 'vault.root',
        kind: _FieldKind.text,
        monospace: true,
        helper: 'Parent of notes / skills / git_root sub-paths.',
      ),
      _Field(
        label: 'Notes path',
        path: 'vault.notes',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Skills path',
        path: 'vault.skills',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Git root',
        path: 'vault.git_root',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Personal prefix',
        path: 'vault.personal_prefix',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Projects prefix',
        path: 'vault.projects_prefix',
        kind: _FieldKind.text,
        monospace: true,
      ),
    ],
  ),
  _Section(
    id: 'mcp',
    title: t.settings.serverSettings.sections.mcpRegistry,
    description: 'Vault paths for MCP servers + secrets file.',
    restartRequired: true,
    fields: const [
      _Field(
        label: 'Registry root',
        path: 'mcp.root',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Secrets file',
        path: 'mcp.secrets_file',
        kind: _FieldKind.text,
        monospace: true,
        helper: 'AES-256-GCM encrypted secrets vault.',
      ),
    ],
  ),
  _Section(
    id: 'memory',
    title: t.settings.serverSettings.sections.memory,
    description: 'Cross-CLI persistent memory subsystem.',
    restartRequired: true,
    fields: const [
      _Field(
        label: 'Backend',
        path: 'memory.backend',
        kind: _FieldKind.select,
        options: ['auto', 'bm25', 'http', 'local'],
        helper: 'auto picks the best available; local needs ONNX.',
      ),
      _Field(
        label: 'Store',
        path: 'memory.store',
        kind: _FieldKind.select,
        options: ['pgvector', 'chromem'],
      ),
      _Field(
        label: 'Default top-k',
        path: 'memory.default_top_k',
        kind: _FieldKind.numberInt,
      ),
      _Field(
        label: 'Similarity threshold',
        path: 'memory.similarity_threshold',
        kind: _FieldKind.numberDouble,
        helper: '0.0–1.0; results under this are filtered out.',
      ),
      _Field(
        label: 'Default scope',
        path: 'memory.scope.default',
        kind: _FieldKind.select,
        options: ['project', 'session', 'global'],
      ),
      _Field(
        label: 'chromem path',
        path: 'memory.chromem_path',
        kind: _FieldKind.text,
        monospace: true,
        helper: 'When store=chromem.',
      ),
      _Field(
        label: 'HTTP base URL',
        path: 'memory.http.base_url',
        kind: _FieldKind.text,
        monospace: true,
        placeholder: 'http://localhost:11434/v1',
      ),
      _Field(
        label: 'HTTP model',
        path: 'memory.http.model',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'HTTP api key',
        path: 'memory.http.api_key',
        kind: _FieldKind.password,
        helper: 'Blank to preserve current.',
      ),
      _Field(
        label: 'HTTP dimensions',
        path: 'memory.http.dimensions',
        kind: _FieldKind.numberInt,
      ),
      _Field(
        label: 'Local model name',
        path: 'memory.local.model',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Local library path',
        path: 'memory.local.library_path',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Local model path',
        path: 'memory.local.model_path',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Local tokenizer path',
        path: 'memory.local.tokenizer_path',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Local max seq len',
        path: 'memory.local.max_seq_len',
        kind: _FieldKind.numberInt,
      ),
    ],
  ),
  _Section(
    id: 'backup',
    title: t.settings.serverSettings.sections.backup,
    description:
        'Encrypted DB backups + admin data exports. Passphrase lives in the keyfile (Settings → Backups).',
    restartRequired: true,
    fields: const [
      _Field(
        label: 'Enabled',
        path: 'backup.enabled',
        kind: _FieldKind.switchToggle,
        helper:
            'Even with this on, the backup subsystem stays off until OPENDRAY_BACKUP_KEY or the keyfile is configured.',
      ),
      _Field(
        label: 'Local dir',
        path: 'backup.local_dir',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'Export dir',
        path: 'backup.export_dir',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'pg_dump path',
        path: 'backup.pg_dump_path',
        kind: _FieldKind.text,
        monospace: true,
        helper: 'Empty = resolve from PATH at startup.',
      ),
      _Field(
        label: 'pg_restore path',
        path: 'backup.pg_restore_path',
        kind: _FieldKind.text,
        monospace: true,
      ),
    ],
  ),
  _Section(
    id: 'claude',
    title: t.settings.serverSettings.sections.storageClaude,
    description: 'Where Claude transcripts live on disk.',
    restartRequired: false,
    fields: const [
      _Field(
        label: 'Accounts dir',
        path: 'providers.claude.accounts_dir',
        kind: _FieldKind.text,
        monospace: true,
        helper:
            'Parent of per-account .claude/ subdirs. Empty = ~/.claude-accounts.',
      ),
    ],
  ),
  _Section(
    id: 'codex',
    title: t.settings.serverSettings.sections.storageCodex,
    description: 'Codex sessions root.',
    restartRequired: false,
    fields: const [
      _Field(
        label: 'Sessions root',
        path: 'providers.codex.sessions_root',
        kind: _FieldKind.text,
        monospace: true,
        helper: 'Empty = ~/.codex/sessions.',
      ),
    ],
  ),
  _Section(
    id: 'gemini',
    title: t.settings.serverSettings.sections.storageGemini,
    description: 'Per-project tmp + projects.json paths.',
    restartRequired: false,
    fields: const [
      _Field(
        label: 'tmp root',
        path: 'providers.gemini.tmp_root',
        kind: _FieldKind.text,
        monospace: true,
      ),
      _Field(
        label: 'projects.json',
        path: 'providers.gemini.projects_file',
        kind: _FieldKind.text,
        monospace: true,
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
        content: const Text(
          'The gateway will exec itself. The mobile app may briefly lose '
          'connection; tokens issued before the restart stay valid.',
        ),
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
                'Loaded from: ${data.configPath}',
                style: Theme.of(context).textTheme.bodySmall?.copyWith(
                      fontFamily: 'monospace',
                      fontSize: 11,
                    ),
              ),
            ),
          const Padding(
            padding: EdgeInsets.fromLTRB(16, 8, 16, 12),
            child: Text(
              'Most sections need a gateway restart to take effect. The '
              'restart button is in the top-right.',
              style: TextStyle(fontSize: 12),
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
                _error = '"${f.label}" must be an integer';
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
                _error = '"${f.label}" must be a number';
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
              ? 'Saved. Restart the gateway to apply.'
              : 'Saved.'),
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
                        'Changes to this section need a gateway restart.',
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
        return Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Text(f.label,
                style: const TextStyle(
                    fontSize: 12, fontWeight: FontWeight.w600)),
            const SizedBox(height: 4),
            DropdownButtonFormField<String>(
              initialValue: value,
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
              'Failed to load server settings',
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
