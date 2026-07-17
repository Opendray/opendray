import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/cortex_api.dart';
import 'package:opendray/core/api/memory_api.dart';
import 'package:opendray/core/api/memory_summarizers_api.dart';
import 'package:opendray/core/api/memory_workers_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// CurationChat — the conversational maintenance channel (Cortex Phase 4),
// mobile parity with app/web/src/components/cortex/CurationChat.tsx. Bound
// to a doc section or knowledge page; the operator asks the AI to update /
// re-draft it. The model/provider picker mirrors web: global curation
// worker, a cloud-agent CLI (claude/codex/antigravity) + model, or a local
// summarizer/HTTP provider + a probed model.

// Cloud-agent providers selectable for a discussion (mirrors the backend's
// validConvProvider set + the worker's buildCommand switch). '' = the global
// curation worker.
const _curationProviders = <({String id, String label})>[
  (id: 'claude', label: 'Claude'),
  (id: 'codex', label: 'Codex'),
  (id: 'antigravity', label: 'Antigravity'),
  (id: 'grok', label: 'Grok'),
  (id: 'opencode', label: 'OpenCode'),
];

class CurationChatScreen extends ConsumerStatefulWidget {
  const CurationChatScreen({
    required this.targetKind, // doc_section | kb_page | blueprint
    required this.targetCwd,
    required this.targetSlug,
    this.onRevision,
    super.key,
  });

  final String targetKind;
  final String targetCwd;
  final String targetSlug;
  // Called after the AI applied/proposed a revision, so the host can
  // refetch the underlying doc.
  final VoidCallback? onRevision;

  @override
  ConsumerState<CurationChatScreen> createState() => _CurationChatScreenState();
}

class _CurationChatScreenState extends ConsumerState<CurationChatScreen> {
  CortexConversation? _conv;
  List<ConversationMessage> _messages = const [];
  final _draftCtrl = TextEditingController();
  final _scrollCtrl = ScrollController();

  bool _loading = true;
  bool _sending = false;
  bool _escalating = false;
  Timer? _poll;
  String _seenRevisionId = '';

  // Picker: '' | 'agent:<id>' | 'local:<id>', plus the chosen model.
  String _selection = '';
  String _model = '';
  // Claude is multi-account — which account a claude turn runs against
  // ('' = default). Only used when the agent is claude.
  String _claudeAccountId = '';
  List<ClaudeAccountSummary> _claudeAccounts = const [];
  List<SummarizerProvider> _localProviders = const [];
  List<ModelOption> _agentModels = const [];
  List<String> _localModels = const [];

  bool get _awaitingReply =>
      _messages.isNotEmpty && _messages.last.role == 'operator';

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _poll?.cancel();
    _draftCtrl.dispose();
    _scrollCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    try {
      final cortex = ref.read(cortexApiProvider);
      final summarizers = await ref.read(memorySummarizersApiProvider).list();
      final accounts = await ref.read(claudeAccountsApiProvider).list();
      final convs = await cortex.listConversations(
        cwd: widget.targetCwd,
        slug: widget.targetSlug,
      );
      final active = convs.where((c) => c.status != 'closed').toList();
      CortexConversation? conv;
      var msgs = const <ConversationMessage>[];
      if (active.isNotEmpty) {
        final (c, m) = await cortex.getConversation(active.first.id);
        conv = c;
        msgs = m;
      }
      if (!mounted) return;
      setState(() {
        _localProviders = summarizers.where((p) => p.enabled).toList();
        _claudeAccounts = accounts.where((a) => a.enabled).toList();
        _conv = conv;
        _messages = msgs;
        _loading = false;
        if (conv != null) {
          if (conv.summarizerId.isNotEmpty) {
            _selection = 'local:${conv.summarizerId}';
          } else if (conv.providerId.isNotEmpty) {
            _selection = 'agent:${conv.providerId}';
          } else {
            _selection = '';
          }
          _model = conv.model;
          _claudeAccountId = conv.claudeAccountId;
        }
      });
      await _refreshModelCatalog();
      _maybePoll();
      _scrollToBottom();
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _loading = false);
      _snack('${t.web.cortex.chat.sendFailed}: $e');
    }
  }

  String get _agentProvider =>
      _selection.startsWith('agent:') ? _selection.substring(6) : '';

  SummarizerProvider? get _localSelected => _selection.startsWith('local:')
      ? _localProviders
            .where((p) => p.id == _selection.substring(6))
            .cast<SummarizerProvider?>()
            .firstWhere((p) => true, orElse: () => null)
      : null;

  // Loads the second-dropdown model catalog for the current selection.
  Future<void> _refreshModelCatalog() async {
    if (_agentProvider.isNotEmpty) {
      try {
        final models = await ref
            .read(memoryWorkersApiProvider)
            .listAgentModels(_agentProvider);
        if (mounted) setState(() => _agentModels = models);
      } on Object {
        if (mounted) setState(() => _agentModels = const []);
      }
    } else {
      setState(() => _agentModels = const []);
    }
    final local = _localSelected;
    if (local != null && local.baseUrl.isNotEmpty) {
      try {
        final probe = await ref
            .read(memoryApiProvider)
            .probe(baseUrl: local.baseUrl);
        if (mounted) setState(() => _localModels = probe.models);
      } on Object {
        if (mounted) setState(() => _localModels = const []);
      }
    } else {
      setState(() => _localModels = const []);
    }
  }

  // Maps the picker state to the backend override shape. The claude
  // account is only carried for the claude agent (backend clears it
  // otherwise).
  ({String providerId, String model, String claudeAccountId, String summarizerId})
      _override() {
    if (_selection.startsWith('agent:')) {
      final providerId = _selection.substring(6);
      return (
        providerId: providerId,
        model: _model,
        claudeAccountId: providerId == 'claude' ? _claudeAccountId : '',
        summarizerId: '',
      );
    }
    if (_selection.startsWith('local:')) {
      return (
        providerId: '',
        model: '',
        claudeAccountId: '',
        summarizerId: _selection.substring(6),
      );
    }
    return (providerId: '', model: '', claudeAccountId: '', summarizerId: '');
  }

  Future<void> _persistOverride() async {
    final conv = _conv;
    if (conv == null) return; // applied at create time otherwise
    final o = _override();
    try {
      final updated = await ref
          .read(cortexApiProvider)
          .setConversationProvider(
            conv.id,
            providerId: o.providerId,
            model: o.model,
            claudeAccountId: o.claudeAccountId,
            summarizerId: o.summarizerId,
          );
      if (mounted) setState(() => _conv = updated);
    } on Object catch (e) {
      _snack('${t.web.cortex.chat.modelChangeFailed}: $e');
    }
  }

  Future<void> _onSelectionChanged(String sel) async {
    setState(() {
      _selection = sel;
      _model = '';
    });
    await _refreshModelCatalog();
    await _persistOverride();
  }

  Future<void> _onModelChanged(String m) async {
    setState(() => _model = m);
    await _persistOverride();
  }

  Future<void> _onAccountChanged(String acct) async {
    setState(() => _claudeAccountId = acct);
    await _persistOverride();
  }

  Future<void> _send() async {
    final text = _draftCtrl.text.trim();
    if (text.isEmpty || _sending || _awaitingReply) return;
    setState(() => _sending = true);
    try {
      final cortex = ref.read(cortexApiProvider);
      var conv = _conv;
      if (conv == null) {
        final o = _override();
        conv = await cortex.createConversation(
          targetKind: widget.targetKind,
          targetCwd: widget.targetCwd,
          targetSlug: widget.targetSlug,
          providerId: o.providerId,
          model: o.model,
          claudeAccountId: o.claudeAccountId,
          summarizerId: o.summarizerId,
        );
      }
      await cortex.sendMessage(conv.id, text);
      _draftCtrl.clear();
      // Reload detail so the operator message shows immediately, then poll.
      final (c, m) = await cortex.getConversation(conv.id);
      if (!mounted) return;
      setState(() {
        _conv = c;
        _messages = m;
        _sending = false;
      });
      _maybePoll();
      _scrollToBottom();
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _sending = false);
      _snack('${t.web.cortex.chat.sendFailed}: $e');
    }
  }

  // Polls for the AI reply while the last message is still the operator's.
  void _maybePoll() {
    _poll?.cancel();
    if (!_awaitingReply || _conv == null) return;
    _poll = Timer.periodic(const Duration(milliseconds: 2500), (_) async {
      final conv = _conv;
      if (conv == null) return;
      try {
        final (c, m) = await ref.read(cortexApiProvider).getConversation(
          conv.id,
        );
        if (!mounted) return;
        setState(() {
          _conv = c;
          _messages = m;
        });
        _notifyRevision();
        if (!_awaitingReply) {
          _poll?.cancel();
          _scrollToBottom();
        }
      } on Object {
        // transient; keep polling
      }
    });
  }

  void _notifyRevision() {
    for (var i = _messages.length - 1; i >= 0; i--) {
      final m = _messages[i];
      if (m.revisionAction.isNotEmpty) {
        if (m.id != _seenRevisionId) {
          _seenRevisionId = m.id;
          widget.onRevision?.call();
        }
        return;
      }
    }
  }

  Future<void> _escalate() async {
    final conv = _conv;
    if (conv == null || _escalating || conv.status == 'escalated') return;
    setState(() => _escalating = true);
    try {
      final updated = await ref.read(cortexApiProvider).escalate(conv.id);
      if (!mounted) return;
      setState(() {
        _conv = updated;
        _escalating = false;
      });
      _snack(t.web.cortex.chat.escalatedToast);
      // Refresh the session list and jump straight into the freshly
      // spawned session — mirrors web's ?open= deep-link. Without this
      // the new session only surfaces on the next manual list refresh.
      ref.invalidate(sessionsListProvider);
      final sid = updated.escalatedSessionId;
      if (sid.isNotEmpty) {
        unawaited(context.push('/session/$sid'));
      }
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _escalating = false);
      _snack('${t.web.cortex.chat.escalateFailed}: $e');
    }
  }

  Future<void> _close() async {
    final conv = _conv;
    if (conv != null) {
      try {
        await ref.read(cortexApiProvider).closeConversation(conv.id);
      } on Object {
        // best-effort
      }
    }
    if (mounted) Navigator.of(context).pop();
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scrollCtrl.hasClients) {
        _scrollCtrl.animateTo(
          _scrollCtrl.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  @override
  Widget build(BuildContext context) {
    final conv = _conv;
    return Scaffold(
      appBar: AppBar(
        title: Text(t.web.cortex.chat.title),
        actions: [
          if (conv != null)
            TextButton.icon(
              onPressed: conv.status == 'escalated' || _escalating
                  ? null
                  : _escalate,
              icon: _escalating
                  ? const SizedBox(
                      width: 14,
                      height: 14,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.north_east, size: 16),
              label: Text(
                conv.status == 'escalated'
                    ? t.web.cortex.chat.escalated
                    : t.web.cortex.chat.escalate,
              ),
            ),
          IconButton(
            tooltip: t.web.cortex.chat.closeHint,
            icon: const Icon(Icons.close),
            onPressed: _close,
          ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : Column(
              children: [
                _modelPicker(context),
                Expanded(child: _messageList(context)),
                _inputBar(context),
              ],
            ),
    );
  }

  Widget _modelPicker(BuildContext context) {
    final theme = Theme.of(context);
    final items = <DropdownMenuItem<String>>[
      DropdownMenuItem(value: '', child: Text(t.web.cortex.chat.modelGlobalDefault)),
      for (final p in _curationProviders)
        DropdownMenuItem(
          value: 'agent:${p.id}',
          child: Text('${t.web.cortex.chat.modelGroupCloud} · ${p.label}'),
        ),
      for (final p in _localProviders)
        DropdownMenuItem(
          value: 'local:${p.id}',
          child: Text('${t.web.cortex.chat.modelGroupLocal} · ${p.label}'),
        ),
    ];
    return Container(
      padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
      decoration: BoxDecoration(
        border: Border(bottom: BorderSide(color: theme.dividerColor)),
      ),
      child: Row(
        children: [
          Text(
            t.web.cortex.chat.modelLabel,
            style: theme.textTheme.bodySmall,
          ),
          const SizedBox(width: 8),
          Expanded(
            child: DropdownButton<String>(
              value: _selection,
              isExpanded: true,
              isDense: true,
              underline: const SizedBox.shrink(),
              items: items,
              onChanged: (v) => _onSelectionChanged(v ?? ''),
            ),
          ),
          if (_agentProvider.isNotEmpty || _selection.startsWith('local:')) ...[
            const SizedBox(width: 8),
            Expanded(child: _modelDropdown(context)),
          ],
          if (_agentProvider == 'claude') ...[
            const SizedBox(width: 8),
            Expanded(child: _accountDropdown(context)),
          ],
        ],
      ),
    );
  }

  // Claude account picker — Claude is multi-account; pin which account
  // this discussion's claude turns run against ('' = default).
  Widget _accountDropdown(BuildContext context) {
    final items = <DropdownMenuItem<String>>[
      DropdownMenuItem(
        value: '',
        child: Text(t.web.cortex.chat.accountDefault),
      ),
      for (final a in _claudeAccounts)
        DropdownMenuItem(
          value: a.id,
          child: Text(
            a.displayName.isNotEmpty
                ? a.displayName
                : (a.name.isNotEmpty ? a.name : a.id),
          ),
        ),
    ];
    final value = items.any((it) => it.value == _claudeAccountId)
        ? _claudeAccountId
        : '';
    return DropdownButton<String>(
      value: value,
      isExpanded: true,
      isDense: true,
      underline: const SizedBox.shrink(),
      items: items,
      onChanged: (v) => _onAccountChanged(v ?? ''),
    );
  }

  Widget _modelDropdown(BuildContext context) {
    final isAgent = _agentProvider.isNotEmpty;
    final local = _localSelected;
    final defaultLabel = isAgent
        ? t.web.cortex.chat.modelCliDefault
        : (local != null && local.model.isNotEmpty)
        ? '${t.web.cortex.chat.modelProviderDefault} · ${local.model}'
        : t.web.cortex.chat.modelCliDefault;
    final items = <DropdownMenuItem<String>>[
      DropdownMenuItem(value: '', child: Text(defaultLabel)),
      if (isAgent)
        for (final m in _agentModels)
          DropdownMenuItem(value: m.id, child: Text(m.label))
      else
        for (final m in _localModels)
          DropdownMenuItem(value: m, child: Text(m)),
    ];
    // Guard: if the pinned model isn't in the catalog, fall back to default.
    final value = items.any((it) => it.value == _model) ? _model : '';
    return DropdownButton<String>(
      value: value,
      isExpanded: true,
      isDense: true,
      underline: const SizedBox.shrink(),
      items: items,
      onChanged: (v) => _onModelChanged(v ?? ''),
    );
  }

  Widget _messageList(BuildContext context) {
    if (_messages.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Icon(
                Icons.forum_outlined,
                size: 28,
                color: Theme.of(context).colorScheme.outline,
              ),
              const SizedBox(height: 8),
              Text(
                t.web.cortex.chat.emptyHint,
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodySmall,
              ),
            ],
          ),
        ),
      );
    }
    return ListView.builder(
      controller: _scrollCtrl,
      padding: const EdgeInsets.all(12),
      itemCount: _messages.length + (_awaitingReply ? 1 : 0),
      itemBuilder: (context, i) {
        if (i >= _messages.length) return _thinkingBubble(context);
        return _bubble(context, _messages[i]);
      },
    );
  }

  Widget _bubble(BuildContext context, ConversationMessage m) {
    final theme = Theme.of(context);
    final isOperator = m.role == 'operator';
    final isSystem = m.role == 'system';
    final align = isOperator ? Alignment.centerRight : Alignment.centerLeft;
    final color = isOperator
        ? theme.colorScheme.primary.withValues(alpha: 0.12)
        : isSystem
        ? theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.4)
        : theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.7);
    return Align(
      alignment: align,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.85,
        ),
        decoration: BoxDecoration(
          color: color,
          borderRadius: BorderRadius.circular(8),
          border: isSystem
              ? Border.all(
                  color: theme.dividerColor,
                  style: BorderStyle.solid,
                )
              : null,
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            SelectableText(
              m.content,
              style: theme.textTheme.bodyMedium?.copyWith(
                color: isSystem ? theme.colorScheme.outline : null,
              ),
            ),
            if (m.revisionAction == 'applied') _revisionBadge(context, true),
            if (m.revisionAction == 'proposed') _revisionBadge(context, false),
          ],
        ),
      ),
    );
  }

  Widget _revisionBadge(BuildContext context, bool applied) {
    final theme = Theme.of(context);
    final c = applied ? Colors.green : theme.colorScheme.tertiary;
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          Icon(applied ? Icons.check_circle : Icons.inbox, size: 13, color: c),
          const SizedBox(width: 4),
          Text(
            applied
                ? t.web.cortex.chat.revisionApplied
                : t.web.cortex.chat.revisionProposed,
            style: theme.textTheme.labelSmall?.copyWith(color: c),
          ),
        ],
      ),
    );
  }

  Widget _thinkingBubble(BuildContext context) {
    final theme = Theme.of(context);
    return Align(
      alignment: Alignment.centerLeft,
      child: Container(
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.symmetric(horizontal: 10, vertical: 8),
        decoration: BoxDecoration(
          color: theme.colorScheme.surfaceContainerHighest.withValues(
            alpha: 0.7,
          ),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            const SizedBox(
              width: 14,
              height: 14,
              child: CircularProgressIndicator(strokeWidth: 2),
            ),
            const SizedBox(width: 8),
            Text(t.web.cortex.chat.thinking, style: theme.textTheme.bodySmall),
          ],
        ),
      ),
    );
  }

  Widget _inputBar(BuildContext context) {
    final theme = Theme.of(context);
    return SafeArea(
      top: false,
      child: Container(
        padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
        decoration: BoxDecoration(
          border: Border(top: BorderSide(color: theme.dividerColor)),
        ),
        child: Row(
          crossAxisAlignment: CrossAxisAlignment.end,
          children: [
            Expanded(
              child: TextField(
                controller: _draftCtrl,
                minLines: 1,
                maxLines: 4,
                textInputAction: TextInputAction.newline,
                decoration: InputDecoration(
                  isDense: true,
                  border: const OutlineInputBorder(),
                  hintText: t.web.cortex.chat.placeholder,
                ),
                onChanged: (_) => setState(() {}),
              ),
            ),
            const SizedBox(width: 8),
            FilledButton(
              onPressed:
                  _draftCtrl.text.trim().isEmpty || _sending || _awaitingReply
                  ? null
                  : _send,
              child: _sending
                  ? const SizedBox(
                      width: 16,
                      height: 16,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Icon(Icons.send, size: 18),
            ),
          ],
        ),
      ),
    );
  }
}
