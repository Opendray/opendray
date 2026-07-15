import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:go_router/go_router.dart';

import 'package:opendray/core/api/roundtable_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/core/providers/provider_visual.dart';
import 'package:opendray/core/widgets/brand_avatar.dart';
import 'package:opendray/features/roundtable/handoff_sheet.dart';
import 'package:opendray/features/roundtable/plan_sheet.dart';
import 'package:opendray/features/roundtable/roles_sheet.dart';

// Round Table chat — mobile parity with
// app/web/src/components/roundtable/RoundTableDetail.tsx. A live group chat:
// operator @mentions members who reply in the thread; optional Summarize;
// Execute hands the discussion to a real session. Polls every 3s while active.
class RoundTableDetailScreen extends ConsumerStatefulWidget {
  const RoundTableDetailScreen({required this.id, super.key});
  final String id;

  @override
  ConsumerState<RoundTableDetailScreen> createState() =>
      _RoundTableDetailScreenState();
}

class _RoundTableDetailScreenState
    extends ConsumerState<RoundTableDetailScreen> {
  final _draft = TextEditingController();
  final _scroll = ScrollController();

  RoundTable? _rt;
  List<RtMessage> _messages = const [];
  bool _loading = true;
  bool _sending = false;
  Object? _error;
  Timer? _poll;

  @override
  void initState() {
    super.initState();
    _load();
    _poll = Timer.periodic(const Duration(seconds: 3), (_) {
      if (_rt?.isClosed ?? false) return;
      _load(silent: true);
    });
  }

  @override
  void dispose() {
    _poll?.cancel();
    _draft.dispose();
    _scroll.dispose();
    super.dispose();
  }

  Future<void> _load({bool silent = false}) async {
    try {
      final (rt, msgs) = await ref.read(roundtableApiProvider).get(widget.id);
      if (!mounted) return;
      final grew = msgs.length != _messages.length;
      setState(() {
        _rt = rt;
        _messages = msgs;
        _loading = false;
        _error = null;
      });
      if (grew) _scrollToBottom();
    } on Object catch (e) {
      if (!mounted) return;
      if (!silent) setState(() => _error = e);
      setState(() => _loading = false);
    }
  }

  void _scrollToBottom() {
    WidgetsBinding.instance.addPostFrameCallback((_) {
      if (_scroll.hasClients) {
        _scroll.animateTo(
          _scroll.position.maxScrollExtent,
          duration: const Duration(milliseconds: 200),
          curve: Curves.easeOut,
        );
      }
    });
  }

  void _insertMention(String token) {
    final cur = _draft.text;
    _draft.text = cur.isEmpty ? '@$token ' : '${cur.trimRight()} @$token ';
    _draft.selection =
        TextSelection.collapsed(offset: _draft.text.length);
  }

  Future<void> _send() async {
    final content = _draft.text.trim();
    if (content.isEmpty || _sending) return;
    setState(() => _sending = true);
    try {
      await ref.read(roundtableApiProvider).postMessage(widget.id, content);
      if (!mounted) return;
      setState(() {
        _draft.clear();
        _sending = false;
      });
      await _load(silent: true);
    } on Object catch (e) {
      if (!mounted) return;
      setState(() => _sending = false);
      ScaffoldMessenger.of(context)
          .showSnackBar(SnackBar(content: Text(e.toString())));
    }
  }

  Future<void> _continue() async {
    try {
      await ref.read(roundtableApiProvider).continueDiscussion(widget.id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(t.web.roundTable.detail.continued)),
        );
      }
      await _load(silent: true);
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  // Auto-discussion settled on a system note (round-cap pause or a member
  // failure)? The burst has stopped — offer Continue. The backend resumes the
  // pending speakers when known, else re-engages everyone.
  bool get _paused {
    if (_messages.isEmpty) return false;
    return _messages.last.isSystem;
  }

  Future<void> _summarize() async {
    try {
      await ref.read(roundtableApiProvider).summarize(widget.id);
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(t.web.roundTable.detail.summarizing)),
        );
      }
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  Future<void> _handoff() async {
    final rt = _rt;
    if (rt == null) return;
    final sessionId = await HandoffSheet.show(context, rt);
    if (sessionId == null || !mounted) return;
    context.go('/session/$sessionId');
  }

  // Close stops new messages but keeps the thread (status active→closed),
  // mirroring the web Close action. Reloading hides the composer and the
  // active-only app-bar actions, so the state change is visible.
  Future<void> _close() async {
    try {
      await ref.read(roundtableApiProvider).close(widget.id);
      if (mounted) await _load(silent: true);
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  // Reopen a closed chat — flips it back to active so the composer and the
  // active-only actions return.
  Future<void> _reopen() async {
    try {
      await ref.read(roundtableApiProvider).reopen(widget.id);
      if (mounted) await _load(silent: true);
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  // Closing keeps the thread but stops the discussion — confirm first so a
  // stray tap in the overflow menu can't end a meeting by accident.
  Future<void> _confirmClose() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: Text(t.web.roundTable.detail.close),
        content: Text(t.web.roundTable.detail.closeConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(t.web.roundTable.detail.close),
          ),
        ],
      ),
    );
    if (ok ?? false) await _close();
  }

  // Overflow-menu dispatch. Every top-bar action lives here so the app bar
  // stays a single, unambiguous ⋮ button.
  void _onMenu(String value) {
    switch (value) {
      case 'roles':
        _editRoles();
      case 'plan':
        _openPlan();
      case 'summarize':
        _summarize();
      case 'handoff':
        _handoff();
      case 'close':
        _confirmClose();
      case 'reopen':
        _reopen();
      case 'delete':
        _delete();
    }
  }

  List<PopupMenuEntry<String>> _menuItems(bool closed) {
    final scheme = Theme.of(context).colorScheme;
    return [
      if (!closed) ...[
        _menuItem('roles', Icons.manage_accounts_outlined,
            t.web.roundTable.detail.rolesTitle),
        _menuItem('plan', Icons.checklist_outlined, t.web.roundTable.plan.title),
        _menuItem('summarize', Icons.auto_awesome_outlined,
            t.web.roundTable.detail.summarize),
        _menuItem('handoff', Icons.rocket_launch_outlined,
            t.web.roundTable.handoff.button),
        _menuItem('close', Icons.lock_outline, t.web.roundTable.detail.close,
            color: scheme.error),
      ],
      if (closed)
        _menuItem('reopen', Icons.lock_open_outlined,
            t.web.roundTable.detail.reopen),
      _menuItem('delete', Icons.delete_outline, t.web.roundTable.detail.delete,
          color: scheme.error),
    ];
  }

  PopupMenuItem<String> _menuItem(String value, IconData icon, String label,
      {Color? color}) {
    return PopupMenuItem<String>(
      value: value,
      child: Row(
        children: [
          Icon(icon, size: 20, color: color),
          const SizedBox(width: 12),
          Text(label, style: color == null ? null : TextStyle(color: color)),
        ],
      ),
    );
  }

  Future<void> _editRoles() async {
    final rt = _rt;
    if (rt == null) return;
    final saved = await RolesSheet.show(context, rt);
    if ((saved ?? false) && mounted) await _load(silent: true);
  }

  Future<void> _openPlan() async {
    final rt = _rt;
    if (rt == null) return;
    await PlanSheet.show(context, rt);
    if (mounted) await _load(silent: true);
  }

  Future<void> _delete() async {
    final ok = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        content: Text(t.web.roundTable.detail.deleteConfirm),
        actions: [
          TextButton(
            onPressed: () => Navigator.pop(ctx, false),
            child: Text(t.common.cancel),
          ),
          FilledButton(
            onPressed: () => Navigator.pop(ctx, true),
            child: Text(t.web.roundTable.detail.delete),
          ),
        ],
      ),
    );
    if (ok != true) return;
    try {
      await ref.read(roundtableApiProvider).delete(widget.id);
      if (mounted) Navigator.of(context).pop();
    } on Object catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context)
            .showSnackBar(SnackBar(content: Text(e.toString())));
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    final rt = _rt;
    final closed = rt?.isClosed ?? false;
    return Scaffold(
      appBar: AppBar(
        title: Text(
          rt == null || rt.topic.isEmpty
              ? t.web.roundTable.untitled
              : rt.topic,
          maxLines: 1,
          overflow: TextOverflow.ellipsis,
        ),
        // Phone screens are narrow and a row of unlabeled icons is easy to
        // mis-tap — collapse every action into ONE labelled overflow menu so
        // each item shows its name, and the destructive ones (close / delete)
        // read in the error colour + ask before acting.
        actions: [
          if (rt != null)
            PopupMenuButton<String>(
              tooltip: MaterialLocalizations.of(context).showMenuTooltip,
              icon: const Icon(Icons.more_vert),
              onSelected: _onMenu,
              itemBuilder: (context) => _menuItems(closed),
            ),
        ],
      ),
      body: _loading
          ? const Center(child: CircularProgressIndicator())
          : _error != null
              ? Center(child: Text(t.web.roundTable.detail.loadFailed))
              : Column(
                  children: [
                    Expanded(child: _thread()),
                    if (!closed && _paused)
                      Padding(
                        padding: const EdgeInsets.fromLTRB(12, 4, 12, 0),
                        child: SizedBox(
                          width: double.infinity,
                          child: OutlinedButton.icon(
                            onPressed: _continue,
                            icon: const Icon(Icons.play_arrow, size: 18),
                            label:
                                Text(t.web.roundTable.detail.continueDiscussion),
                          ),
                        ),
                      ),
                    if (!closed) _composer(rt),
                  ],
                ),
    );
  }

  Widget _thread() {
    if (_messages.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(t.web.roundTable.detail.emptyThread),
              const SizedBox(height: 6),
              Text(
                t.web.roundTable.detail.emptyHint,
                style: Theme.of(context).textTheme.bodySmall,
                textAlign: TextAlign.center,
              ),
            ],
          ),
        ),
      );
    }
    return ListView.builder(
      controller: _scroll,
      padding: const EdgeInsets.all(12),
      itemCount: _messages.length,
      itemBuilder: (_, i) => _Bubble(message: _messages[i]),
    );
  }

  Widget _composer(RoundTable? rt) {
    final theme = Theme.of(context);
    final seats = rt?.seats ?? const <Seat>[];
    return Material(
      elevation: 8,
      child: SafeArea(
        top: false,
        child: Padding(
          padding: const EdgeInsets.fromLTRB(12, 8, 12, 8),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Mention chips.
              Wrap(
                spacing: 6,
                runSpacing: 4,
                crossAxisAlignment: WrapCrossAlignment.center,
                children: [
                  Text(t.web.roundTable.detail.mentionHint,
                      style: theme.textTheme.bodySmall),
                  for (final s in seats)
                    ActionChip(
                      visualDensity: VisualDensity.compact,
                      label: Text('@${s.provider}'),
                      onPressed: () => _insertMention(s.provider),
                    ),
                  if (seats.length > 1)
                    ActionChip(
                      visualDensity: VisualDensity.compact,
                      label: const Text('@all'),
                      onPressed: () => _insertMention('all'),
                    ),
                ],
              ),
              const SizedBox(height: 6),
              Row(
                crossAxisAlignment: CrossAxisAlignment.end,
                children: [
                  Expanded(
                    child: TextField(
                      controller: _draft,
                      minLines: 1,
                      maxLines: 4,
                      textInputAction: TextInputAction.newline,
                      decoration: InputDecoration(
                        hintText: t.web.roundTable.detail.composerPlaceholder,
                        border: const OutlineInputBorder(),
                        isDense: true,
                      ),
                    ),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    onPressed: _sending ? null : _send,
                    child: _sending
                        ? const SizedBox(
                            height: 18,
                            width: 18,
                            child: CircularProgressIndicator(strokeWidth: 2),
                          )
                        : const Icon(Icons.send, size: 18),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Bubble extends StatelessWidget {
  const _Bubble({required this.message});
  final RtMessage message;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final m = message;

    if (m.isSystem) {
      return Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Center(
          child: Container(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
            decoration: BoxDecoration(
              color: theme.colorScheme.errorContainer.withValues(alpha: 0.3),
              borderRadius: BorderRadius.circular(8),
            ),
            child: Text(m.content,
                textAlign: TextAlign.center,
                style: theme.textTheme.bodySmall),
          ),
        ),
      );
    }

    final isOp = m.isOperator;
    // Per-agent tint so it's clear at a glance who's speaking — each seat's
    // bubble carries its provider's brand colour (same source as the avatar),
    // mirroring the web's SEAT_BUBBLE palette. Operator = primary, summary =
    // tertiary.
    final seatVisual = (!isOp && !m.isSummary && m.seatProvider.isNotEmpty)
        ? providerVisualFor(m.seatProvider)
        : null;
    final Color bg;
    final Color borderColor;
    if (isOp) {
      bg = theme.colorScheme.primary.withValues(alpha: 0.12);
      borderColor = theme.colorScheme.primary.withValues(alpha: 0.35);
    } else if (m.isSummary) {
      bg = theme.colorScheme.tertiary.withValues(alpha: 0.12);
      borderColor = theme.colorScheme.tertiary.withValues(alpha: 0.4);
    } else if (seatVisual != null) {
      bg = seatVisual.brandColor.withValues(alpha: 0.13);
      borderColor = seatVisual.brandColor.withValues(alpha: 0.55);
    } else {
      bg = theme.colorScheme.surfaceContainerHighest;
      borderColor = theme.colorScheme.outlineVariant.withValues(alpha: 0.5);
    }
    return Align(
      alignment: isOp ? Alignment.centerRight : Alignment.centerLeft,
      child: Container(
        constraints: BoxConstraints(
          maxWidth: MediaQuery.of(context).size.width * 0.82,
        ),
        margin: const EdgeInsets.symmetric(vertical: 4),
        padding: const EdgeInsets.all(10),
        decoration: BoxDecoration(
          color: bg,
          border: Border.all(color: borderColor),
          borderRadius: BorderRadius.circular(12),
        ),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                if (!isOp && m.seatProvider.isNotEmpty) ...[
                  BrandAvatar(providerId: m.seatProvider, size: 16),
                  const SizedBox(width: 6),
                ],
                Text(
                  isOp ? t.web.roundTable.you : m.seatProvider,
                  style: theme.textTheme.labelSmall?.copyWith(
                    fontWeight: FontWeight.w600,
                  ),
                ),
                if (m.isSummary) ...[
                  const SizedBox(width: 6),
                  Container(
                    padding:
                        const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                    decoration: BoxDecoration(
                      color: theme.colorScheme.tertiary.withValues(alpha: 0.2),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Text(t.web.roundTable.summary,
                        style: theme.textTheme.labelSmall),
                  ),
                ],
              ],
            ),
            const SizedBox(height: 4),
            SelectableText(m.content, style: theme.textTheme.bodyMedium),
          ],
        ),
      ),
    );
  }
}
