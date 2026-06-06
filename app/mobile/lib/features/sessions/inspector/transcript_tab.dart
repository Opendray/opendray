import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Conversation surface inside the session inspector. Renders the
// session's reconstructed conversation (user prompts + assistant
// prose) parsed from the CLI's JSONL as a scrollable list. This is
// the scrollback the live terminal can't give: the agent CLIs run a
// full-screen alternate-screen TUI, which has no scrollback.
class TranscriptTab extends ConsumerStatefulWidget {
  const TranscriptTab({required this.sessionId, super.key});

  final String sessionId;

  @override
  ConsumerState<TranscriptTab> createState() => _TranscriptTabState();
}

class _TranscriptTabState extends ConsumerState<TranscriptTab>
    with AutomaticKeepAliveClientMixin {
  AsyncValue<TranscriptResponse> _state = const AsyncValue.loading();

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
      final res =
          await ref.read(sessionsApiProvider).transcript(widget.sessionId);
      if (!mounted) return;
      setState(() => _state = AsyncValue.data(res));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _copyAll(List<TranscriptTurn> turns) async {
    final text = turns
        .map((tn) => '${tn.role.toUpperCase()}: ${tn.text}')
        .join('\n\n');
    await Clipboard.setData(ClipboardData(text: text));
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(t.sessions.inspector.conversation.copied),
        duration: const Duration(seconds: 2),
        behavior: SnackBarBehavior.floating,
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    super.build(context);
    return _state.when(
      data: (res) {
        if (res.turns.isEmpty) {
          return _EmptyView(onRefresh: _load);
        }
        return Column(
          children: [
            _Toolbar(
              count: res.turns.length,
              onCopyAll: () => _copyAll(res.turns),
              onRefresh: _load,
            ),
            const Divider(height: 1),
            Expanded(
              child: RefreshIndicator(
                onRefresh: _load,
                child: ListView.separated(
                  padding: const EdgeInsets.all(12),
                  itemCount: res.turns.length,
                  separatorBuilder: (_, __) => const SizedBox(height: 8),
                  itemBuilder: (_, i) => _TurnCard(turn: res.turns[i]),
                ),
              ),
            ),
          ],
        );
      },
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorView(error: e, onRetry: _load),
    );
  }
}

class _Toolbar extends StatelessWidget {
  const _Toolbar({
    required this.count,
    required this.onCopyAll,
    required this.onRefresh,
  });

  final int count;
  final VoidCallback onCopyAll;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 4, 8),
      child: Row(
        children: [
          Expanded(
            child: Text(
              t.sessions.inspector.conversation.count(count: count.toString()),
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ),
          TextButton.icon(
            icon: const Icon(Icons.copy_all_outlined, size: 18),
            label: Text(t.sessions.inspector.conversation.copyAll),
            onPressed: onCopyAll,
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

class _TurnCard extends StatelessWidget {
  const _TurnCard({required this.turn});

  final TranscriptTurn turn;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final isUser = turn.role == 'user';
    final roleColor =
        isUser ? theme.colorScheme.primary : theme.colorScheme.secondary;
    return Container(
      width: double.infinity,
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: theme.colorScheme.surfaceContainerHighest.withValues(alpha: 0.3),
        borderRadius: BorderRadius.circular(8),
        border: Border.all(color: theme.dividerColor),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            turn.role.toUpperCase(),
            style: theme.textTheme.labelSmall?.copyWith(
              color: roleColor,
              fontWeight: FontWeight.w700,
              letterSpacing: 0.5,
            ),
          ),
          const SizedBox(height: 4),
          SelectableText(
            turn.text,
            style: theme.textTheme.bodyMedium,
          ),
        ],
      ),
    );
  }
}

class _EmptyView extends StatelessWidget {
  const _EmptyView({required this.onRefresh});
  final Future<void> Function() onRefresh;

  @override
  Widget build(BuildContext context) {
    return RefreshIndicator(
      onRefresh: onRefresh,
      child: ListView(
        children: [
          const SizedBox(height: 96),
          Icon(
            Icons.forum_outlined,
            size: 48,
            color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
          ),
          const SizedBox(height: 12),
          Text(
            t.sessions.inspector.conversation.empty,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
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
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              t.sessions.inspector.conversation.loadFailed,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 8),
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
