import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/providers_api.dart';
import 'package:opendray/features/providers/provider_config_screen.dart';

// Providers — view CLI providers (Claude / Codex / Gemini / …) and,
// for Claude specifically, the multi-account list. Each row exposes
// only the enable/disable switch. Token entry, OAuth imports, manifest
// edits stay on the web admin where pasting long secrets is sane.
class ProvidersScreen extends ConsumerStatefulWidget {
  const ProvidersScreen({super.key});

  @override
  ConsumerState<ProvidersScreen> createState() => _ProvidersScreenState();
}

class _ProvidersScreenState extends ConsumerState<ProvidersScreen> {
  AsyncValue<_ProvidersData> _state = const AsyncValue.loading();
  // Track in-flight toggles so we can show a spinner only on the
  // affected row instead of grey-locking the whole screen.
  final Set<String> _busy = {};

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      // Both lists fetched in parallel — they're independent.
      final providersFuture = ref.read(providersApiProvider).list();
      final accountsFuture =
          ref.read(claudeAccountsApiProvider).list();
      final results = await Future.wait([providersFuture, accountsFuture]);
      if (!mounted) return;
      final providers = (results[0] as List<ProviderSummary>)
        ..sort((a, b) {
          if (a.enabled != b.enabled) return a.enabled ? -1 : 1;
          return a.name.toLowerCase().compareTo(b.name.toLowerCase());
        });
      final accounts = (results[1] as List<ClaudeAccountSummary>)
        ..sort((a, b) {
          if (a.isUsable != b.isUsable) return a.isUsable ? -1 : 1;
          return a.displayName
              .toLowerCase()
              .compareTo(b.displayName.toLowerCase());
        });
      setState(
        () => _state = AsyncValue.data(
          _ProvidersData(providers: providers, accounts: accounts),
        ),
      );
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  Future<void> _runToggle({
    required String key,
    required String okMsg,
    required String failPrefix,
    required Future<void> Function() op,
  }) async {
    setState(() => _busy.add(key));
    final messenger = ScaffoldMessenger.of(context);
    try {
      await op();
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(
          content: Text(okMsg),
          duration: const Duration(seconds: 2),
          behavior: SnackBarBehavior.floating,
        ),
      );
      await _load();
    } on ApiException catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('$failPrefix: ${e.message}')),
      );
    } on Object catch (e) {
      if (!mounted) return;
      messenger.showSnackBar(
        SnackBar(content: Text('$failPrefix: $e')),
      );
    } finally {
      if (mounted) setState(() => _busy.remove(key));
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: const Text('Providers'),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            tooltip: 'Refresh',
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: _state.when(
        data: _buildBody,
        loading: () => const Center(child: CircularProgressIndicator()),
        error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
      ),
    );
  }

  Widget _buildBody(_ProvidersData data) {
    if (data.providers.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            'No providers loaded.\n\n'
            'The gateway resolves providers from its plugin directory '
            '— check the operator guide if this looks wrong.',
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        children: [
          const _SectionHeader(label: 'CLI providers'),
          for (final p in data.providers)
            _ProviderTile(
              provider: p,
              busy: _busy.contains('p:${p.id}'),
              onToggle: (next) => _runToggle(
                key: 'p:${p.id}',
                okMsg: next ? '${p.name} enabled.' : '${p.name} disabled.',
                failPrefix: 'Toggle failed',
                op: () => ref
                    .read(providersApiProvider)
                    .setEnabled(p.id, enabled: next),
              ),
              onTap: () => Navigator.of(context).push(
                MaterialPageRoute<void>(
                  builder: (_) => ProviderConfigScreen(providerId: p.id),
                ),
              ),
            ),
          const SizedBox(height: 8),
          const _SectionHeader(label: 'Claude accounts'),
          if (data.accounts.isEmpty)
            Padding(
              padding: const EdgeInsets.fromLTRB(20, 4, 16, 12),
              child: Text(
                'No Claude accounts imported. Import via the web admin:\n'
                'Providers → Claude → Add account.',
                style: Theme.of(context).textTheme.bodySmall,
              ),
            )
          else
            for (final a in data.accounts)
              _AccountTile(
                account: a,
                busy: _busy.contains('a:${a.id}'),
                onToggle: (next) => _runToggle(
                  key: 'a:${a.id}',
                  okMsg: next
                      ? '${a.displayName} enabled.'
                      : '${a.displayName} disabled.',
                  failPrefix: 'Toggle failed',
                  op: () => ref
                      .read(claudeAccountsApiProvider)
                      .setEnabled(a.id, enabled: next),
                ),
              ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }
}

class _ProvidersData {
  _ProvidersData({required this.providers, required this.accounts});
  final List<ProviderSummary> providers;
  final List<ClaudeAccountSummary> accounts;
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

class _ProviderTile extends StatelessWidget {
  const _ProviderTile({
    required this.provider,
    required this.busy,
    required this.onToggle,
    required this.onTap,
  });
  final ProviderSummary provider;
  final bool busy;
  final ValueChanged<bool> onToggle;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return ListTile(
      onTap: onTap,
      leading: Container(
        width: 36,
        height: 36,
        alignment: Alignment.center,
        decoration: BoxDecoration(
          color: Theme.of(context)
              .colorScheme
              .primary
              .withValues(alpha: 0.12),
          borderRadius: BorderRadius.circular(8),
        ),
        child: Text(
          provider.name.isNotEmpty
              ? provider.name[0].toUpperCase()
              : provider.id[0].toUpperCase(),
          style: TextStyle(
            color: Theme.of(context).colorScheme.primary,
            fontWeight: FontWeight.w600,
          ),
        ),
      ),
      title: Text(
        provider.name,
        style: const TextStyle(fontWeight: FontWeight.w600),
      ),
      subtitle: Text(
        provider.id,
        style: Theme.of(context)
            .textTheme
            .bodySmall
            ?.copyWith(fontFamily: 'monospace'),
      ),
      trailing: busy
          ? const SizedBox(
              width: 32,
              height: 32,
              child: Padding(
                padding: EdgeInsets.all(8),
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          : Switch(
              value: provider.enabled,
              onChanged: onToggle,
            ),
    );
  }
}

class _AccountTile extends StatelessWidget {
  const _AccountTile({
    required this.account,
    required this.busy,
    required this.onToggle,
  });
  final ClaudeAccountSummary account;
  final bool busy;
  final ValueChanged<bool> onToggle;

  @override
  Widget build(BuildContext context) {
    final usable = account.isUsable;
    return ListTile(
      leading: Icon(
        usable ? Icons.account_circle : Icons.account_circle_outlined,
        color: usable
            ? Theme.of(context).colorScheme.primary
            : Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.4),
      ),
      title: Text(account.displayName),
      subtitle: Wrap(
        spacing: 6,
        children: [
          Text(
            account.id,
            style: Theme.of(context)
                .textTheme
                .bodySmall
                ?.copyWith(fontFamily: 'monospace'),
          ),
          if (!account.tokenFilled)
            _MiniBadge(
              label: 'no token',
              color: Theme.of(context).colorScheme.error,
            ),
        ],
      ),
      trailing: busy
          ? const SizedBox(
              width: 32,
              height: 32,
              child: Padding(
                padding: EdgeInsets.all(8),
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          : Switch(
              value: account.enabled,
              onChanged: onToggle,
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
              'Failed to load providers',
              style: Theme.of(context).textTheme.titleMedium,
            ),
            const SizedBox(height: 6),
            Text(
              error,
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodySmall,
            ),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: const Text('Retry')),
          ],
        ),
      ),
    );
  }
}
