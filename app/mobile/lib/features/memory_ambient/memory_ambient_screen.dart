import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/ambient_api.dart';
import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Capture & injection — the runtime knobs that decide WHEN a session
// is summarised into memory (capture rules) and WHAT context gets
// pre-loaded into a new session (injection profiles).
//
// Mobile adaptation: read-mostly. Capture rules show an enable toggle
// + a run-now action; injection profiles expose a strategy switch via
// a bottom sheet. Creating rules and editing trigger configs stay on
// the web admin (too dense for a phone form) — surfaced by the intro.
class MemoryAmbientScreen extends ConsumerStatefulWidget {
  const MemoryAmbientScreen({super.key, this.embedded = false});

  /// When embedded inside the unified Cortex settings tabs, drop the
  /// Scaffold/AppBar and render just the scrollable body.
  final bool embedded;

  @override
  ConsumerState<MemoryAmbientScreen> createState() =>
      _MemoryAmbientScreenState();
}

class _MemoryAmbientScreenState extends ConsumerState<MemoryAmbientScreen> {
  AsyncValue<(List<CaptureRule>, List<InjectionProfile>)> _state =
      const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final api = ref.read(ambientApiProvider);
      final rules = await api.listCaptureRules();
      final profiles = await api.listInjectionProfiles();
      if (!mounted) return;
      setState(() => _state = AsyncValue.data((rules, profiles)));
    } on ApiException catch (e) {
      if (mounted) {
        setState(() => _state = AsyncValue.error(e, StackTrace.current));
      }
    } on Object catch (e, st) {
      if (mounted) setState(() => _state = AsyncValue.error(e, st));
    }
  }

  void _snack(String msg) {
    if (!mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(SnackBar(content: Text(msg)));
  }

  Future<void> _toggleRule(CaptureRule rule, bool enabled) async {
    try {
      await ref
          .read(ambientApiProvider)
          .setCaptureRuleEnabled(rule.id, enabled: enabled);
      await _load();
    } on Object catch (e) {
      _snack(t.memoryAmbient.actionFailed(error: e.toString()));
    }
  }

  Future<void> _runNow(CaptureRule rule) async {
    try {
      final n = await ref.read(ambientApiProvider).runCaptureRuleNow(rule.id);
      _snack(t.memoryAmbient.ranSnack(count: n));
    } on Object catch (e) {
      _snack(t.memoryAmbient.actionFailed(error: e.toString()));
    }
  }

  Future<void> _editStrategy(InjectionProfile p) async {
    final picked = await showModalBottomSheet<String>(
      context: context,
      showDragHandle: true,
      builder: (_) => _StrategySheet(current: p.strategyKind),
    );
    if (picked == null || picked == p.strategyKind || !mounted) return;
    try {
      await ref.read(ambientApiProvider).setInjectionStrategy(p.id, picked);
      await _load();
    } on Object catch (e) {
      _snack(t.memoryAmbient.actionFailed(error: e.toString()));
    }
  }

  @override
  Widget build(BuildContext context) {
    final body = _state.when(
      data: (d) => _buildBody(d.$1, d.$2),
      loading: () => const Center(child: CircularProgressIndicator()),
      error: (e, _) => _ErrorView(error: e.toString(), onRetry: _load),
    );
    if (widget.embedded) return body;
    return Scaffold(
      appBar: AppBar(
        title: Text(t.memoryAmbient.title),
        actions: [
          IconButton(
            icon: const Icon(Icons.refresh),
            onPressed: _state is AsyncLoading ? null : _load,
          ),
        ],
      ),
      body: body,
    );
  }

  Widget _buildBody(List<CaptureRule> rules, List<InjectionProfile> profiles) {
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView(
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(16, 12, 16, 4),
            child: Text(
              t.memoryAmbient.intro,
              style: Theme.of(context).textTheme.bodySmall?.copyWith(
                    color: Theme.of(context)
                        .colorScheme
                        .onSurface
                        .withValues(alpha: 0.6),
                  ),
            ),
          ),
          _SectionHeader(label: t.memoryAmbient.captureSection),
          if (rules.isEmpty)
            _emptyRow()
          else
            for (final r in rules) _CaptureTile(
              rule: r,
              onToggle: (v) => _toggleRule(r, v),
              onRunNow: () => _runNow(r),
            ),
          _SectionHeader(label: t.memoryAmbient.injectionSection),
          if (profiles.isEmpty)
            _emptyRow()
          else
            for (final p in profiles) _InjectionTile(
              profile: p,
              onTap: () => _editStrategy(p),
            ),
          const SizedBox(height: 16),
        ],
      ),
    );
  }

  Widget _emptyRow() => Padding(
        padding: const EdgeInsets.fromLTRB(20, 4, 20, 12),
        child: Text(
          t.memoryAmbient.empty,
          style: Theme.of(context).textTheme.bodyMedium?.copyWith(
                color: Theme.of(context)
                    .colorScheme
                    .onSurface
                    .withValues(alpha: 0.5),
              ),
        ),
      );
}

String triggerLabel(String kind) => switch (kind) {
      'after_messages' => t.memoryAmbient.triggerAfterMessages,
      'on_idle' => t.memoryAmbient.triggerOnIdle,
      'k_chars' => t.memoryAmbient.triggerKChars,
      'manual' => t.memoryAmbient.triggerManual,
      _ => t.memoryAmbient.triggerUnknown,
    };

String strategyLabel(String kind) => switch (kind) {
      'none' => t.memoryAmbient.strategyNone,
      'top_k_recent' => t.memoryAmbient.strategyTopKRecent,
      'top_k_relevant' => t.memoryAmbient.strategyTopKRelevant,
      'on_keyword' => t.memoryAmbient.strategyOnKeyword,
      'manual_only' => t.memoryAmbient.strategyManualOnly,
      'hybrid' => t.memoryAmbient.strategyHybrid,
      _ => t.memoryAmbient.strategyUnknown,
    };

const _allStrategies = [
  'none',
  'top_k_recent',
  'top_k_relevant',
  'on_keyword',
  'manual_only',
  'hybrid',
];

class _CaptureTile extends StatelessWidget {
  const _CaptureTile({
    required this.rule,
    required this.onToggle,
    required this.onRunNow,
  });
  final CaptureRule rule;
  final ValueChanged<bool> onToggle;
  final VoidCallback onRunNow;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall?.copyWith(
          color:
              Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
        );
    final scope = rule.targetScope == 'global'
        ? t.memoryAmbient.scopeGlobal
        : t.memoryAmbient.scopeProject;
    return ListTile(
      title: Text(rule.name, overflow: TextOverflow.ellipsis),
      subtitle: Text('${triggerLabel(rule.triggerKind)} · $scope', style: muted),
      trailing: Row(
        mainAxisSize: MainAxisSize.min,
        children: [
          IconButton(
            icon: const Icon(Icons.play_arrow_outlined),
            tooltip: t.memoryAmbient.runNow,
            onPressed: onRunNow,
          ),
          Switch(value: rule.enabled, onChanged: onToggle),
        ],
      ),
    );
  }
}

class _InjectionTile extends StatelessWidget {
  const _InjectionTile({required this.profile, required this.onTap});
  final InjectionProfile profile;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall?.copyWith(
          color:
              Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
        );
    return ListTile(
      onTap: onTap,
      title: Text(strategyLabel(profile.strategyKind)),
      subtitle: Text(
        profile.sessionId == null || profile.sessionId!.isEmpty
            ? t.memoryAmbient.strategyLabel
            : profile.sessionId!,
        style: muted,
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }
}

class _StrategySheet extends StatelessWidget {
  const _StrategySheet({required this.current});
  final String current;

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          Padding(
            padding: const EdgeInsets.fromLTRB(20, 4, 20, 8),
            child: Align(
              alignment: Alignment.centerLeft,
              child: Text(
                t.memoryAmbient.strategyLabel,
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ),
          ),
          for (final s in _allStrategies)
            ListTile(
              title: Text(strategyLabel(s)),
              trailing: s == current
                  ? Icon(Icons.check, color: Theme.of(context).colorScheme.primary)
                  : null,
              onTap: () => Navigator.of(context).pop(s),
            ),
          const SizedBox(height: 8),
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
              color:
                  Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
            ),
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
            Icon(Icons.error_outline,
                size: 48, color: Theme.of(context).colorScheme.error),
            const SizedBox(height: 12),
            Text(t.memoryAmbient.loadFailed,
                style: Theme.of(context).textTheme.titleMedium),
            const SizedBox(height: 6),
            Text(error,
                textAlign: TextAlign.center,
                style: Theme.of(context).textTheme.bodySmall),
            const SizedBox(height: 16),
            FilledButton(onPressed: onRetry, child: Text(t.common.retry)),
          ],
        ),
      ),
    );
  }
}
