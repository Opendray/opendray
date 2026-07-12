// "What's new" bottom sheet — mobile mirror of the web sidebar
// UpdatesDrawer (app/web/src/components/UpdatesDrawer.tsx). Fetches the
// latest GitHub release (changelog fallback) via releases_api.dart and
// keeps read state in SharedPreferences. Opened from More → Resources →
// Updates.

import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';
import 'package:opendray/core/api/releases_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:url_launcher/url_launcher.dart';

const _releasesFallbackUrl = 'https://github.com/Opendray/opendray/releases';

Future<void> showUpdatesSheet(BuildContext context) {
  return showModalBottomSheet<void>(
    context: context,
    isScrollControlled: true,
    showDragHandle: true,
    builder: (_) => const _UpdatesSheet(),
  );
}

class _UpdatesSheet extends ConsumerWidget {
  const _UpdatesSheet();

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(latestReleaseProvider);
    final maxHeight = MediaQuery.of(context).size.height * 0.7;
    return SafeArea(
      child: ConstrainedBox(
        constraints: BoxConstraints(maxHeight: maxHeight),
        child: async.when(
          loading: () => const _SheetPadding(child: _Loading()),
          error: (e, _) => _SheetPadding(
            child: _LoadError(
              message: e.toString(),
              onRetry: () => ref.invalidate(latestReleaseProvider),
            ),
          ),
          data: (rel) => _Content(rel: rel),
        ),
      ),
    );
  }
}

class _SheetPadding extends StatelessWidget {
  const _SheetPadding({required this.child});
  final Widget child;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 8, 20, 28),
      child: child,
    );
  }
}

class _Loading extends StatelessWidget {
  const _Loading();

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisAlignment: MainAxisAlignment.center,
      children: [
        const SizedBox(
          width: 18,
          height: 18,
          child: CircularProgressIndicator(strokeWidth: 2),
        ),
        const SizedBox(width: 12),
        Text(t.nav.updates.loading,
            style: Theme.of(context).textTheme.bodyMedium),
      ],
    );
  }
}

class _LoadError extends StatelessWidget {
  const _LoadError({required this.message, required this.onRetry});
  final String message;
  final VoidCallback onRetry;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          t.nav.updates.loadFailed(error: message),
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 12),
        Align(
          alignment: Alignment.centerLeft,
          child: OutlinedButton(
            onPressed: onRetry,
            child: Text(t.common.retry),
          ),
        ),
      ],
    );
  }
}

class _Content extends ConsumerWidget {
  const _Content({required this.rel});
  final ReleaseInfo rel;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final theme = Theme.of(context);
    final date = _formatPublished(rel.publishedAt);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // Header
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 0, 20, 8),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Container(
                width: 30,
                height: 30,
                decoration: BoxDecoration(
                  color: theme.colorScheme.primary.withValues(alpha: 0.15),
                  borderRadius: BorderRadius.circular(8),
                ),
                alignment: Alignment.center,
                child: Icon(Icons.auto_awesome,
                    size: 16, color: theme.colorScheme.primary),
              ),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      t.nav.updates.whatsNew(version: rel.tag),
                      style: theme.textTheme.titleSmall,
                    ),
                    if (date != null)
                      Padding(
                        padding: const EdgeInsets.only(top: 2),
                        child: Text(date,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurface
                                  .withValues(alpha: 0.6),
                            )),
                      ),
                  ],
                ),
              ),
            ],
          ),
        ),
        const Divider(height: 1),
        // Highlights
        Flexible(
          child: SingleChildScrollView(
            padding: const EdgeInsets.fromLTRB(20, 12, 20, 12),
            child: rel.highlights.isEmpty
                ? Text(
                    t.nav.updates.noHighlights,
                    style: theme.textTheme.bodyMedium?.copyWith(
                      color:
                          theme.colorScheme.onSurface.withValues(alpha: 0.7),
                    ),
                  )
                : Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      for (final h in rel.highlights)
                        Padding(
                          padding: const EdgeInsets.only(bottom: 10),
                          child: Row(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Padding(
                                padding: const EdgeInsets.only(top: 6),
                                child: Container(
                                  width: 6,
                                  height: 6,
                                  decoration: BoxDecoration(
                                    color: theme.colorScheme.primary,
                                    shape: BoxShape.circle,
                                  ),
                                ),
                              ),
                              const SizedBox(width: 10),
                              Expanded(
                                child: Text(h,
                                    style: theme.textTheme.bodyMedium),
                              ),
                            ],
                          ),
                        ),
                      if (rel.source == ReleaseSource.changelog)
                        Padding(
                          padding: const EdgeInsets.only(top: 2),
                          child: Text(
                            t.nav.updates.sourceChangelog,
                            style: theme.textTheme.bodySmall?.copyWith(
                              color: theme.colorScheme.onSurface
                                  .withValues(alpha: 0.5),
                            ),
                          ),
                        ),
                    ],
                  ),
          ),
        ),
        const Divider(height: 1),
        // Actions
        Padding(
          padding: const EdgeInsets.fromLTRB(20, 12, 20, 20),
          child: Column(
            children: [
              SizedBox(
                width: double.infinity,
                child: OutlinedButton.icon(
                  icon: const Icon(Icons.open_in_new, size: 16),
                  label: Text(t.nav.updates.openFull),
                  onPressed: () => _openUrl(
                    rel.htmlUrl.isNotEmpty ? rel.htmlUrl : _releasesFallbackUrl,
                  ),
                ),
              ),
              const SizedBox(height: 8),
              SizedBox(
                width: double.infinity,
                child: FilledButton(
                  onPressed: () {
                    ref
                        .read(lastReadReleaseProvider.notifier)
                        .markRead(rel.version);
                    Navigator.of(context).pop();
                  },
                  child: Text(t.nav.updates.markRead),
                ),
              ),
            ],
          ),
        ),
      ],
    );
  }

  static String? _formatPublished(String? iso) {
    if (iso == null || iso.isEmpty) return null;
    try {
      return DateFormat('MMM d, yyyy').format(DateTime.parse(iso).toLocal());
    } on Object {
      return iso.length >= 10 ? iso.substring(0, 10) : iso;
    }
  }

  static Future<void> _openUrl(String url) async {
    final uri = Uri.tryParse(url);
    if (uri == null) return;
    try {
      await launchUrl(uri, mode: LaunchMode.externalApplication);
    } on Object {
      // Best-effort convenience link.
    }
  }
}
