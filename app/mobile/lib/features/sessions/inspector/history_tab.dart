import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// History surface inside the session inspector. Shows the user's
// past prompts in this project (cwd) across every session that
// ran here, pulled from Claude's JSONL transcripts on the gateway
// host. Tap a prompt to push it into the live terminal as a re-
// run; type in the search field to filter long histories.
class HistoryTab extends ConsumerStatefulWidget {
  const HistoryTab({required this.sessionId, super.key});

  final String sessionId;

  @override
  ConsumerState<HistoryTab> createState() => _HistoryTabState();
}

class _HistoryTabState extends ConsumerState<HistoryTab>
    with AutomaticKeepAliveClientMixin {
  AsyncValue<HistoryResponse> _state = const AsyncValue.loading();
  final _searchCtrl = TextEditingController();
  String _query = '';

  @override
  bool get wantKeepAlive => true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  @override
  void dispose() {
    _searchCtrl.dispose();
    super.dispose();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final res = await ref.read(sessionsApiProvider).history(widget.sessionId);
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

  Future<void> _onPromptTap(ProjectInput entry) async {
    final action = await showModalBottomSheet<_HistoryAction>(
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
                    DateFormat.yMMMd().add_Hm().format(
                          entry.timestamp.toLocal(),
                        ),
                    style: Theme.of(sheetCtx).textTheme.bodySmall,
                  ),
                  const SizedBox(height: 4),
                  ConstrainedBox(
                    constraints: const BoxConstraints(maxHeight: 180),
                    child: SingleChildScrollView(
                      child: SelectableText(
                        entry.text,
                        style: Theme.of(sheetCtx).textTheme.bodyMedium,
                      ),
                    ),
                  ),
                ],
              ),
            ),
            const Divider(height: 1),
            ListTile(
              leading: const Icon(Icons.send_outlined),
              title: Text(t.sessions.inspector.history.insertIntoTerminal),
              subtitle: const Text(
                'Pastes the prompt as keystrokes; press Return to send',
              ),
              onTap: () => Navigator.of(sheetCtx).pop(_HistoryAction.insert),
            ),
            const SizedBox(height: 4),
          ],
        ),
      ),
    );
    if (action != _HistoryAction.insert || !mounted) return;
    await _pushInput(entry.text);
  }

  Future<void> _pushInput(String text) async {
    final messenger = ScaffoldMessenger.of(context);
    try {
      await ref.read(sessionsApiProvider).input(widget.sessionId, text);
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            'Pasted prompt (${text.length} chars). Press Return in the '
            'terminal to send.',
          ),
          duration: const Duration(seconds: 3),
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
        _SearchBar(
          controller: _searchCtrl,
          onChanged: (v) => setState(() => _query = v.trim().toLowerCase()),
          onRefresh: _load,
        ),
        const Divider(height: 1),
        Expanded(
          child: _state.when(
            data: (res) {
              if (res.unsupportedProvider) {
                return _UnsupportedView();
              }
              if (res.entries.isEmpty) {
                return _EmptyView(query: _query);
              }
              final filtered = _query.isEmpty
                  ? res.entries
                  : res.entries
                      .where((e) => e.text.toLowerCase().contains(_query))
                      .toList();
              if (filtered.isEmpty) {
                return _EmptyView(query: _query);
              }
              return RefreshIndicator(
                onRefresh: _load,
                child: ListView.separated(
                  itemCount: filtered.length,
                  separatorBuilder: (_, __) => Divider(
                    height: 1,
                    color: Theme.of(context).dividerColor,
                  ),
                  itemBuilder: (_, i) => _PromptTile(
                    entry: filtered[i],
                    onTap: () => _onPromptTap(filtered[i]),
                  ),
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

enum _HistoryAction { insert }

class _SearchBar extends StatelessWidget {
  const _SearchBar({
    required this.controller,
    required this.onChanged,
    required this.onRefresh,
  });

  final TextEditingController controller;
  final ValueChanged<String> onChanged;
  final VoidCallback onRefresh;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(12, 8, 4, 8),
      child: Row(
        children: [
          Expanded(
            child: TextField(
              controller: controller,
              autocorrect: false,
              textInputAction: TextInputAction.search,
              onChanged: onChanged,
              decoration: InputDecoration(
                hintText: t.sessions.inspector.history.searchHint,
                prefixIcon: const Icon(Icons.search, size: 20),
                suffixIcon: controller.text.isEmpty
                    ? null
                    : IconButton(
                        icon: const Icon(Icons.clear, size: 18),
                        onPressed: () {
                          controller.clear();
                          onChanged('');
                        },
                      ),
                isDense: true,
                contentPadding: const EdgeInsets.symmetric(vertical: 10),
              ),
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

class _PromptTile extends StatelessWidget {
  const _PromptTile({required this.entry, required this.onTap});

  final ProjectInput entry;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final preview = entry.text.replaceAll(RegExp(r'\s+'), ' ').trim();
    return ListTile(
      onTap: onTap,
      title: Text(
        preview.isEmpty ? '(empty prompt)' : preview,
        maxLines: 2,
        overflow: TextOverflow.ellipsis,
        style: Theme.of(context).textTheme.bodyMedium,
      ),
      subtitle: Text(
        _relative(entry.timestamp),
        style: Theme.of(context).textTheme.bodySmall,
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }

  static String _relative(DateTime ts) {
    final diff = DateTime.now().toUtc().difference(ts.toUtc());
    if (diff.inSeconds < 60) return '${diff.inSeconds}s ago';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return DateFormat.yMMMd().format(ts.toLocal());
  }
}

class _UnsupportedView extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.history_toggle_off,
              size: 56,
              color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
            ),
            const SizedBox(height: 12),
            Text(
              'History is Claude-only',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              'This session uses a different provider (codex / gemini / '
              'shell). The transcript-driven prompt history only exists '
              'for Claude Code today.',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
          ],
        ),
      ),
    );
  }
}

class _EmptyView extends StatelessWidget {
  const _EmptyView({required this.query});
  final String query;

  @override
  Widget build(BuildContext context) {
    final isFiltered = query.isNotEmpty;
    return Center(
      child: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.search_off,
              size: 48,
              color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
            ),
            const SizedBox(height: 12),
            Text(
              isFiltered
                  ? 'No prompts match "$query"'
                  : 'No prompts in this project yet',
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
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
              'Failed to load history',
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
