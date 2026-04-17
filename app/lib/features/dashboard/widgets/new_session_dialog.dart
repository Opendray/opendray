import 'package:flutter/material.dart';
import 'package:provider/provider.dart';
import '../../../core/api/api_client.dart';
import '../../../core/models/provider.dart';
import '../../../shared/directory_picker.dart';
import '../../../shared/theme/app_theme.dart';

class NewSessionDialog extends StatefulWidget {
  final List<ProviderInfo> providers;
  /// When set, pre-fills the working-directory field (e.g. coming from the
  /// file browser's "Create session here" action).
  final String? initialCwd;
  const NewSessionDialog({super.key, required this.providers, this.initialCwd});
  @override
  State<NewSessionDialog> createState() => _NewSessionDialogState();
}

class _NewSessionDialogState extends State<NewSessionDialog> {
  String _sessionType = 'claude';
  final _cwdController = TextEditingController();
  final _nameController = TextEditingController();
  final _modelController = TextEditingController();
  String _model = '';

  // Claude multi-account support — loaded once for the lifetime of the
  // dialog. Only shown when the selected provider is `claude`. Empty
  // accountId === "use system keychain / env" (legacy behaviour).
  List<Map<String, dynamic>> _claudeAccounts = [];
  String _claudeAccountId = '';

  // LLM provider support — shown when the selected session type speaks
  // OpenAI natively (opencode, future additions). Model picker uses an
  // auto-detect pass (/v1/models on the upstream) with a free-text
  // fallback if the upstream is unreachable or doesn't advertise a list.
  List<Map<String, dynamic>> _llmProviders = [];
  String _llmProviderId = '';
  List<String> _probedModels = [];
  bool _probingModels = false;
  String? _probeError;

  bool _cwdValid = false;

  /// Session types we recognise as "OpenAI-compatible agent" — they
  /// need a provider + model picker instead of the Claude-specific
  /// account selector. Kept as a small whitelist for now; when more
  /// such agents land we can flip this to a manifest capability flag.
  static const _openaiNativeAgents = {'opencode'};
  bool get _isOpenAINative => _openaiNativeAgents.contains(_sessionType);

  /// A session can only run an *agent*-style provider (CLI tools, shells,
  /// local model runners). Panel providers (File Browser, Obsidian Reader,
  /// Web/Simulator Preview) are side-panel UIs, not processes you can
  /// attach a terminal to — they must not appear as session types.
  List<ProviderInfo> get _enabled => widget.providers
      .where((p) => p.enabled && p.provider.type != 'panel')
      .toList();
  ProviderInfo? get _selected =>
      _enabled.where((p) => p.provider.name == _sessionType).firstOrNull;

  @override
  void initState() {
    super.initState();
    // Default to the first enabled agent provider — if 'claude' isn't
    // installed/enabled the dialog used to silently land on it anyway and
    // produce a useless empty selection.
    final enabledAgents = widget.providers
        .where((p) => p.enabled && p.provider.type != 'panel')
        .toList();
    if (!enabledAgents.any((p) => p.provider.name == _sessionType) &&
        enabledAgents.isNotEmpty) {
      _sessionType = enabledAgents.first.provider.name;
    }
    if (widget.initialCwd != null && widget.initialCwd!.isNotEmpty) {
      _cwdController.text = widget.initialCwd!;
      _cwdValid = true;
    }
    _cwdController.addListener(() {
      final valid = _cwdController.text.trim().isNotEmpty;
      if (valid != _cwdValid) setState(() => _cwdValid = valid);
    });
    _loadClaudeAccounts();
    _loadLLMProviders();
  }

  Future<void> _loadLLMProviders() async {
    try {
      final rows = await context.read<ApiClient>().llmProviders();
      if (!mounted) return;
      setState(() {
        _llmProviders = rows.where((p) => p['enabled'] as bool? ?? true).toList();
      });
    } catch (_) {
      // Older server without /api/llm-providers — hide the picker.
    }
  }

  Future<void> _probeModels() async {
    if (_llmProviderId.isEmpty) return;
    setState(() {
      _probingModels = true;
      _probeError = null;
      _probedModels = [];
    });
    final models = await context.read<ApiClient>().llmProviderModels(_llmProviderId);
    if (!mounted) return;
    setState(() {
      _probingModels = false;
      if (models == null) {
        _probeError = 'Upstream unreachable — enter a model name manually.';
      } else {
        _probedModels = models;
      }
    });
  }

  Future<void> _loadClaudeAccounts() async {
    try {
      final accounts = await context.read<ApiClient>().claudeAccounts();
      if (!mounted) return;
      setState(() {
        _claudeAccounts = accounts
            .where((a) => (a['enabled'] as bool? ?? true) &&
                (a['tokenFilled'] as bool? ?? false))
            .toList();
      });
    } catch (_) {
      // Best-effort: if the server doesn't support claude-accounts yet
      // (older build) we just hide the selector.
    }
  }

  @override
  void dispose() {
    _cwdController.dispose();
    _nameController.dispose();
    _modelController.dispose();
    super.dispose();
  }

  void _submit() {
    final cwd = _cwdController.text.trim();
    if (cwd.isEmpty) return;
    // For OpenAI-native agents the effective model comes from the
    // free-text controller (which is kept in sync with the dropdown
    // selection when the user picks from probed results).
    final effectiveModel = _isOpenAINative ? _modelController.text.trim() : _model;
    Navigator.of(context).pop({
      'cwd': cwd,
      'sessionType': _sessionType,
      'name': _nameController.text.trim(),
      'model': effectiveModel,
      'extraArgs': <String>[],
      if (_sessionType == 'claude' && _claudeAccountId.isNotEmpty)
        'claudeAccountId': _claudeAccountId,
      if (_isOpenAINative && _llmProviderId.isNotEmpty)
        'llmProviderId': _llmProviderId,
    });
  }

  Future<void> _browseCwd() async {
    final initial = _cwdController.text.trim();
    final picked = await pickDirectory(context,
        initialPath: initial.isEmpty ? null : initial);
    if (picked == null || !mounted) return;
    setState(() {
      _cwdController.text = picked;
      _cwdValid = true;
    });
  }

  @override
  Widget build(BuildContext context) {
    final bottomInset = MediaQuery.of(context).viewInsets.bottom;
    // Cap the sheet height so content scrolls on small screens (fixes the
    // "RenderFlex overflowed by X pixels" from the old fixed Column).
    final maxHeight = MediaQuery.of(context).size.height * 0.85;

    return ConstrainedBox(
      constraints: BoxConstraints(maxHeight: maxHeight),
      child: Padding(
        padding: EdgeInsets.only(
            left: 20, right: 20, top: 20, bottom: bottomInset + 20),
        child: Column(mainAxisSize: MainAxisSize.min, children: [
          Center(
            child: Container(
                width: 36, height: 4,
                decoration: BoxDecoration(
                    color: AppColors.border,
                    borderRadius: BorderRadius.circular(2))),
          ),
          const SizedBox(height: 16),
          const Align(
            alignment: Alignment.centerLeft,
            child: Text('New Session',
                style: TextStyle(fontSize: 18, fontWeight: FontWeight.w600)),
          ),
          const SizedBox(height: 16),

          // Scrollable body — prevents overflow when keyboard is up or
          // when there are many providers / capabilities / a model dropdown.
          Flexible(
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  // Provider grid
                  const Text('Provider',
                      style: TextStyle(fontSize: 12, color: AppColors.textMuted)),
                  const SizedBox(height: 8),
                  Wrap(
                    spacing: 8, runSpacing: 8,
                    children: _enabled.map((pi) {
                      final selected = _sessionType == pi.provider.name;
                      return GestureDetector(
                        onTap: () => setState(() {
                          _sessionType = pi.provider.name;
                          _model = '';
                        }),
                        child: Container(
                          padding: const EdgeInsets.symmetric(
                              horizontal: 12, vertical: 10),
                          decoration: BoxDecoration(
                            color: selected
                                ? AppColors.accentSoft
                                : AppColors.surfaceAlt,
                            border: Border.all(
                                color: selected
                                    ? AppColors.accent
                                    : AppColors.border),
                            borderRadius: BorderRadius.circular(10),
                          ),
                          child: Row(mainAxisSize: MainAxisSize.min, children: [
                            Text(pi.provider.icon,
                                style: const TextStyle(fontSize: 18)),
                            const SizedBox(width: 6),
                            Text(pi.provider.displayName,
                                style: TextStyle(
                                  fontSize: 13,
                                  color: selected
                                      ? AppColors.accent
                                      : AppColors.text,
                                  fontWeight: selected
                                      ? FontWeight.w600
                                      : FontWeight.normal,
                                )),
                            if (!pi.installed) ...[
                              const SizedBox(width: 4),
                              Container(
                                padding: const EdgeInsets.symmetric(
                                    horizontal: 3, vertical: 1),
                                decoration: BoxDecoration(
                                    color: AppColors.warningSoft,
                                    borderRadius: BorderRadius.circular(3)),
                                child: const Text('!',
                                    style: TextStyle(
                                        color: AppColors.warning, fontSize: 9)),
                              ),
                            ],
                          ]),
                        ),
                      );
                    }).toList(),
                  ),
                  const SizedBox(height: 16),

                  // CWD + Browse button
                  Row(crossAxisAlignment: CrossAxisAlignment.end, children: [
                    Expanded(
                      child: TextField(
                        controller: _cwdController,
                        autocorrect: false,
                        enableSuggestions: false,
                        textCapitalization: TextCapitalization.none,
                        style: const TextStyle(
                            fontSize: 13, fontFamily: 'monospace'),
                        decoration: const InputDecoration(
                          labelText: 'Working Directory',
                          hintText: '/path/to/project',
                        ),
                      ),
                    ),
                    const SizedBox(width: 8),
                    SizedBox(
                      height: 48,
                      child: OutlinedButton.icon(
                        onPressed: _browseCwd,
                        icon: const Icon(Icons.folder_open, size: 16),
                        label: const Text('Browse',
                            style: TextStyle(fontSize: 12)),
                        style: OutlinedButton.styleFrom(
                          foregroundColor: AppColors.accent,
                          padding: const EdgeInsets.symmetric(horizontal: 10),
                        ),
                      ),
                    ),
                  ]),
                  const SizedBox(height: 12),

                  // Name
                  TextField(
                    controller: _nameController,
                    decoration: const InputDecoration(
                        labelText: 'Session Name', hintText: 'Optional'),
                  ),
                  const SizedBox(height: 12),

                  // Model (static — only for agents that advertise a
                  // fixed model list in their manifest, i.e. Claude /
                  // Codex / Gemini CLI. OpenAI-native agents route
                  // through the LLM-provider picker below instead.)
                  if (!_isOpenAINative &&
                      _selected != null &&
                      _selected!.provider.capabilities.models.isNotEmpty) ...[
                    DropdownButtonFormField<String>(
                      initialValue: _model.isEmpty ? null : _model,
                      decoration: const InputDecoration(labelText: 'Model'),
                      dropdownColor: AppColors.surfaceAlt,
                      items: [
                        const DropdownMenuItem(
                            value: '',
                            child: Text('Default',
                                style: TextStyle(color: AppColors.textMuted))),
                        ..._selected!.provider.capabilities.models.map((m) =>
                            DropdownMenuItem(
                                value: m.id, child: Text(m.name))),
                      ],
                      onChanged: (v) => setState(() => _model = v ?? ''),
                    ),
                    const SizedBox(height: 12),
                  ],

                  // LLM provider + model — for OpenAI-native agents
                  // (OpenCode, …). Provider comes from the address
                  // book; model list is probed from upstream with a
                  // free-text fallback when probing fails.
                  if (_isOpenAINative) ...[
                    if (_llmProviders.isEmpty)
                      Container(
                        padding: const EdgeInsets.all(10),
                        decoration: BoxDecoration(
                          color: AppColors.warningSoft,
                          borderRadius: BorderRadius.circular(8),
                          border: Border.all(color: AppColors.warning.withOpacity(0.4)),
                        ),
                        child: const Text(
                          'No LLM providers configured yet. Open Browser → LLM Providers to add one before starting this session.',
                          style: TextStyle(fontSize: 12, color: AppColors.warning),
                        ),
                      )
                    else ...[
                      DropdownButtonFormField<String>(
                        initialValue: _llmProviderId.isEmpty ? '' : _llmProviderId,
                        decoration: const InputDecoration(labelText: 'LLM Provider'),
                        dropdownColor: AppColors.surfaceAlt,
                        items: [
                          const DropdownMenuItem(
                              value: '',
                              child: Text('— pick one —',
                                  style: TextStyle(color: AppColors.textMuted))),
                          ..._llmProviders.map((p) {
                            final display = (p['displayName'] as String?)?.isNotEmpty == true
                                ? p['displayName'] as String
                                : p['name'] as String? ?? '';
                            return DropdownMenuItem(
                                value: p['id'] as String,
                                child: Text('$display  (${p['providerType']})'));
                          }),
                        ],
                        onChanged: (v) {
                          setState(() {
                            _llmProviderId = v ?? '';
                            _probedModels = [];
                            _probeError = null;
                            _modelController.clear();
                          });
                          if (_llmProviderId.isNotEmpty) _probeModels();
                        },
                      ),
                      const SizedBox(height: 12),
                      if (_llmProviderId.isNotEmpty) ...[
                        Row(children: [
                          Expanded(
                            child: TextField(
                              controller: _modelController,
                              decoration: InputDecoration(
                                labelText: 'Model',
                                hintText: _probedModels.isNotEmpty
                                    ? _probedModels.first
                                    : 'qwen3-coder:30b',
                                helperText: _probeError,
                                helperStyle: const TextStyle(color: AppColors.warning),
                              ),
                            ),
                          ),
                          const SizedBox(width: 8),
                          IconButton(
                            tooltip: 'Detect models',
                            icon: _probingModels
                                ? const SizedBox(
                                    width: 16, height: 16,
                                    child: CircularProgressIndicator(
                                        strokeWidth: 2, color: AppColors.accent))
                                : const Icon(Icons.radar, color: AppColors.accent),
                            onPressed: _probingModels ? null : _probeModels,
                          ),
                        ]),
                        if (_probedModels.isNotEmpty) ...[
                          const SizedBox(height: 8),
                          Wrap(
                            spacing: 6, runSpacing: 6,
                            children: _probedModels.take(12).map((m) {
                              final selected = _modelController.text == m;
                              return GestureDetector(
                                onTap: () => setState(() => _modelController.text = m),
                                child: Container(
                                  padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
                                  decoration: BoxDecoration(
                                    color: selected ? AppColors.accentSoft : AppColors.surfaceAlt,
                                    borderRadius: BorderRadius.circular(4),
                                    border: Border.all(
                                      color: selected ? AppColors.accent : AppColors.border,
                                    ),
                                  ),
                                  child: Text(m,
                                      style: TextStyle(
                                          fontSize: 11,
                                          fontFamily: 'monospace',
                                          color: selected ? AppColors.accent : AppColors.text)),
                                ),
                              );
                            }).toList(),
                          ),
                        ],
                        const SizedBox(height: 12),
                      ],
                    ],
                  ],

                  // Claude account — only shown for claude sessions AND
                  // only when the server reports at least one enabled/
                  // token-filled account. Otherwise we silently keep the
                  // legacy behaviour (system keychain).
                  if (_sessionType == 'claude' && _claudeAccounts.isNotEmpty) ...[
                    DropdownButtonFormField<String>(
                      initialValue: _claudeAccountId.isEmpty ? '' : _claudeAccountId,
                      decoration: const InputDecoration(labelText: 'Claude account'),
                      dropdownColor: AppColors.surfaceAlt,
                      items: [
                        const DropdownMenuItem(
                            value: '',
                            child: Text('System (keychain / env)',
                                style: TextStyle(color: AppColors.textMuted))),
                        ..._claudeAccounts.map((a) {
                          final display = (a['displayName'] as String?)?.isNotEmpty == true
                              ? a['displayName'] as String
                              : a['name'] as String? ?? '';
                          return DropdownMenuItem(
                              value: a['id'] as String,
                              child: Text('$display  (claude-${a['name']})'));
                        }),
                      ],
                      onChanged: (v) =>
                          setState(() => _claudeAccountId = v ?? ''),
                    ),
                    const SizedBox(height: 12),
                  ],

                  // Capabilities
                  if (_selected != null)
                    Wrap(spacing: 6, children: [
                      if (_selected!.provider.capabilities.supportsResume)
                        _CapBadge('Resume'),
                      if (_selected!.provider.capabilities.supportsImages)
                        _CapBadge('Images'),
                      if (_selected!.provider.capabilities.supportsMcp)
                        _CapBadge('MCP'),
                      if (_selected!.provider.capabilities.supportsStream)
                        _CapBadge('Stream'),
                    ]),
                ],
              ),
            ),
          ),

          const SizedBox(height: 16),
          SizedBox(
            width: double.infinity,
            child: FilledButton(
              onPressed: _cwdValid ? _submit : null,
              style: FilledButton.styleFrom(
                backgroundColor: AppColors.accent,
                padding: const EdgeInsets.symmetric(vertical: 14),
                shape: RoundedRectangleBorder(
                    borderRadius: BorderRadius.circular(10)),
              ),
              child: const Text('Create & Start',
                  style: TextStyle(fontSize: 14, fontWeight: FontWeight.w500)),
            ),
          ),
        ]),
      ),
    );
  }
}

class _CapBadge extends StatelessWidget {
  final String text;
  const _CapBadge(this.text);
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(
          color: AppColors.surfaceAlt, borderRadius: BorderRadius.circular(4)),
      child: Text(text,
          style: const TextStyle(color: AppColors.textMuted, fontSize: 10)),
    );
  }
}
