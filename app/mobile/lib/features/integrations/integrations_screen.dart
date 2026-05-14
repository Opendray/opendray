import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:intl/intl.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/integrations_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/integrations/integration_detail_screen.dart';
import 'package:opendray/features/integrations/integration_forms.dart';

// Top-level Integrations list. Splits operator-registered rows from
// system rows (e.g. opendray-memory MCP) so the latter never visually
// dominates a vault that mostly cares about its own integrations.
//
// Each row shows: a colored health dot, name + route_prefix, scopes
// summary, last health-ping time. Tap drills into per-integration
// recent calls.
class IntegrationsScreen extends ConsumerStatefulWidget {
  const IntegrationsScreen({super.key});

  @override
  ConsumerState<IntegrationsScreen> createState() =>
      _IntegrationsScreenState();
}

class _IntegrationsScreenState extends ConsumerState<IntegrationsScreen> {
  AsyncValue<List<Integration>> _state = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(integrationsApiProvider).list();
      if (!mounted) return;
      list.sort((a, b) {
        // Operator rows first, then system rows. Within each group:
        // most-recently-pinged first so anything actively chatting
        // floats up.
        if (a.isSystem != b.isSystem) return a.isSystem ? 1 : -1;
        final aSeen =
            a.healthLastSeen ?? DateTime.fromMillisecondsSinceEpoch(0);
        final bSeen =
            b.healthLastSeen ?? DateTime.fromMillisecondsSinceEpoch(0);
        return bSeen.compareTo(aSeen);
      });
      setState(() => _state = AsyncValue.data(list));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _onRegister() async {
    final form = await RegisterIntegrationScreen.push(context);
    if (form == null || !mounted) return;
    final messenger = ScaffoldMessenger.of(context);
    try {
      final result = await ref.read(integrationsApiProvider).register(
            name: form.name,
            baseUrl: form.baseUrl,
            routePrefix: form.routePrefix,
            scopes: form.scopes,
            version: form.version,
          );
      if (!mounted) return;
      await RevealApiKeyDialog.show(
        context: context,
        apiKey: result.apiKey,
        title: t.integrations.apiKeyForName(name: result.integration.name),
        subtitle: t.integrations.apiKeySubtitleRegister(
          routePrefix: result.integration.routePrefix,
        ),
      );
      if (!mounted) return;
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(t.integrations.registerFailedApi(error: e.message)),
        ),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(
            t.integrations.registerFailedGeneric(error: e.toString()),
          ),
        ),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.integrations.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: t.sessions.inspector.shared.refresh,
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildList,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
      floatingActionButton: FloatingActionButton.extended(
        heroTag: 'integrations_fab',
        onPressed: _onRegister,
        icon: const Icon(Icons.add),
        label: Text(t.integrations.register),
      ),
    );
  }

  Widget _buildList(List<Integration> all) {
    if (all.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            t.integrations.emptyState,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    final operator = all.where((i) => !i.isSystem).toList();
    final system = all.where((i) => i.isSystem).toList();

    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        children: [
          if (operator.isNotEmpty) ...[
            _SectionHeader(label: t.integrations.sectionRegistered),
            for (final i in operator) _IntegrationTile(integration: i),
          ],
          if (system.isNotEmpty) ...[
            _SectionHeader(label: t.integrations.sectionSystem),
            for (final i in system) _IntegrationTile(integration: i),
          ],
          const SizedBox(height: 16),
        ],
      ),
    );
  }
}

class _SectionHeader extends StatelessWidget {
  const _SectionHeader({required this.label});
  final String label;

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.fromLTRB(20, 16, 16, 6),
      child: Text(
        label.toUpperCase(),
        style: Theme.of(context).textTheme.labelSmall?.copyWith(
              letterSpacing: 0.8,
              color: Theme.of(context)
                  .colorScheme
                  .onSurface
                  .withValues(alpha: 0.6),
            ),
      ),
    );
  }
}

class _IntegrationTile extends StatelessWidget {
  const _IntegrationTile({required this.integration});
  final Integration integration;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    final scope = integration.scopes.isEmpty
        ? 'no scopes'
        : integration.scopes.length == 1
            ? integration.scopes.first
            : '${integration.scopes.length} scopes';
    return ListTile(
      onTap: () => Navigator.of(context).push(
        MaterialPageRoute<void>(
          builder: (_) =>
              IntegrationDetailScreen(integrationId: integration.id),
        ),
      ),
      leading: _HealthDot(status: integration.healthStatus),
      title: Row(
        children: [
          Flexible(
            child: Text(
              integration.name,
              overflow: TextOverflow.ellipsis,
              style: TextStyle(
                color: integration.enabled
                    ? null
                    : Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.5),
              ),
            ),
          ),
          if (!integration.enabled) ...[
            const SizedBox(width: 6),
            _MiniBadge(
              label: 'disabled',
              color: Theme.of(context).colorScheme.error,
            ),
          ],
        ],
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Wrap(
          spacing: 6,
          runSpacing: 2,
          children: [
            Text(
              '/${integration.routePrefix}',
              style: const TextStyle(fontFamily: 'monospace'),
            ),
            Text('· $scope'),
            Text(
              '· ${_lastSeenLabel(integration.healthLastSeen)}',
            ),
          ],
        ),
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }

  static String _lastSeenLabel(DateTime? ts) {
    if (ts == null) return 'never pinged';
    final diff = DateTime.now().toUtc().difference(ts.toUtc());
    if (diff.inSeconds < 60) return 'just now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m ago';
    if (diff.inHours < 24) return '${diff.inHours}h ago';
    if (diff.inDays < 7) return '${diff.inDays}d ago';
    return DateFormat.yMMMd().format(ts.toLocal());
  }
}

class _HealthDot extends StatelessWidget {
  const _HealthDot({required this.status});
  final String status;

  @override
  Widget build(BuildContext context) {
    final color = switch (status) {
      'healthy' => Colors.greenAccent,
      'degraded' => Colors.amberAccent,
      'unhealthy' => Colors.redAccent,
      _ => Colors.grey,
    };
    return Container(
      width: 12,
      height: 12,
      margin: const EdgeInsets.only(top: 4),
      decoration: BoxDecoration(
        color: color,
        shape: BoxShape.circle,
        border: Border.all(
          color: Theme.of(context).colorScheme.outline.withValues(alpha: 0.4),
        ),
      ),
    );
  }
}

class _MiniBadge extends StatelessWidget {
  const _MiniBadge({required this.label, required this.color});
  final String label;
  final Color color;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 1),
      decoration: BoxDecoration(
        color: color.withValues(alpha: 0.15),
        borderRadius: BorderRadius.circular(4),
        border: Border.all(color: color.withValues(alpha: 0.6), width: 0.5),
      ),
      child: Text(
        label,
        style: TextStyle(color: color, fontSize: 10),
      ),
    );
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
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(
              t.integrations.listLoadFailed,
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
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
