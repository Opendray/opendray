import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/memory_summarizers_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/memory_ambient/memory_ambient_screen.dart';
import 'package:opendray/features/memory_workers/memory_workers_screen.dart';

// CortexSettingsScreen — the single home for every AI-drive memory knob,
// mirroring the web's unified /cortex/settings (MemoryWorkers.tsx): no
// more hunting between a Workers screen and a separate Capture/Injection
// screen buried in More. Web lays it out as one long page with jump-nav;
// on a phone that becomes tabs so each section stays focused.
//
//   Workers              — per-task summarizer/agent routing + model pin
//   Capture & injection  — when memory is captured + what gets pre-loaded
//   Providers            — the LLM endpoints workers route to (read-only;
//                          add/edit on web, like the web provider CRUD)
class CortexSettingsScreen extends ConsumerStatefulWidget {
  const CortexSettingsScreen({super.key});

  @override
  ConsumerState<CortexSettingsScreen> createState() =>
      _CortexSettingsScreenState();
}

class _CortexSettingsScreenState extends ConsumerState<CortexSettingsScreen>
    with SingleTickerProviderStateMixin {
  late final TabController _tabs = TabController(length: 3, vsync: this);
  // Bumped by the AppBar refresh so the embedded tab states recreate and
  // reload their data.
  int _reloadKey = 0;

  @override
  void dispose() {
    _tabs.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.cortexSettings.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.common.refresh,
            onPressed: () => setState(() => _reloadKey++),
          ),
        ],
        bottom: TabBar(
          controller: _tabs,
          isScrollable: true,
          tabAlignment: TabAlignment.start,
          tabs: [
            Tab(text: t.cortexSettings.tabWorkers),
            Tab(text: t.cortexSettings.tabCapture),
            Tab(text: t.cortexSettings.tabProviders),
          ],
        ),
      ),
      body: TabBarView(
        controller: _tabs,
        children: [
          MemoryWorkersScreen(key: ValueKey('workers-$_reloadKey'), embedded: true),
          MemoryAmbientScreen(key: ValueKey('ambient-$_reloadKey'), embedded: true),
          _ProvidersTab(key: ValueKey('providers-$_reloadKey')),
        ],
      ),
    );
  }
}

// Read-only summarizer-provider registry. Full CRUD (add ollama / lmstudio
// / API endpoints) lives on the web admin — too dense for a phone form —
// so mobile lists them and points there, mirroring the parity stance.
class _ProvidersTab extends ConsumerStatefulWidget {
  const _ProvidersTab({super.key});

  @override
  ConsumerState<_ProvidersTab> createState() => _ProvidersTabState();
}

class _ProvidersTabState extends ConsumerState<_ProvidersTab> {
  AsyncValue<List<SummarizerProvider>> _state = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(memorySummarizersApiProvider).list();
      if (mounted) setState(() => _state = AsyncValue.data(list));
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  @override
  Widget build(BuildContext context) {
    return _state.when(
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(t.cortexSettings.providersLoadFailed,
                  style: Theme.of(context).textTheme.titleMedium),
              const SizedBox(height: 6),
              Text('$e',
                  textAlign: TextAlign.center,
                  style: Theme.of(context).textTheme.bodySmall),
              const SizedBox(height: 16),
              FilledButton(onPressed: _load, child: Text(t.common.retry)),
            ],
          ),
        ),
      ),
      data: (providers) => RefreshIndicator(
        onRefresh: _load,
        child: ListView(
          padding: const EdgeInsets.all(12),
          children: [
            Text(
              t.cortexSettings.providersHint,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.6),
                  ),
            ),
            const SizedBox(height: 8),
            if (providers.isEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(vertical: 24),
                child: Text(t.cortexSettings.providersEmpty,
                    textAlign: TextAlign.center,
                    style: Theme.of(context).textTheme.bodyMedium),
              )
            else
              for (final p in providers) _ProviderTile(provider: p),
            const SizedBox(height: 8),
            Text(
              t.cortexSettings.providersManageOnWeb,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.5),
                    fontStyle: FontStyle.italic,
                  ),
            ),
          ],
        ),
      ),
    );
  }
}

class _ProviderTile extends StatelessWidget {
  const _ProviderTile({required this.provider});
  final SummarizerProvider provider;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Card(
      margin: const EdgeInsets.only(bottom: 6),
      child: ListTile(
        dense: true,
        leading: const Icon(Icons.dns_outlined, size: 20),
        title: Row(
          children: [
            Flexible(
              child: Text(provider.name, overflow: TextOverflow.ellipsis),
            ),
            if (provider.isDefault) ...[
              const SizedBox(width: 6),
              Container(
                padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
                decoration: BoxDecoration(
                  color: scheme.primary.withValues(alpha: 0.12),
                  borderRadius: BorderRadius.circular(4),
                  border: Border.all(color: scheme.primary.withValues(alpha: 0.4)),
                ),
                child: Text(t.cortexSettings.defaultBadge,
                    style: TextStyle(fontSize: 10, color: scheme.primary)),
              ),
            ],
          ],
        ),
        subtitle: Text(
          provider.model.isEmpty
              ? provider.kind
              : '${provider.kind} · ${provider.model}',
          style: const TextStyle(fontFamily: 'monospace', fontSize: 11),
        ),
      ),
    );
  }
}
