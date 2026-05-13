import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/skills_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/skills/skill_editor_screen.dart';

// Skills list screen. Two visual classes:
// - builtin (embedded in the gateway binary) — read-only, but
//   "Customize" clones the body into a vault entry.
// - vault (operator-edited) — fully editable. If a vault row
//   shadows a builtin id, the delete action is labeled
//   "Reset to built-in" instead of "Delete".
class SkillsScreen extends ConsumerStatefulWidget {
  const SkillsScreen({super.key});

  @override
  ConsumerState<SkillsScreen> createState() => _SkillsScreenState();
}

class _SkillsScreenState extends ConsumerState<SkillsScreen> {
  AsyncValue<List<SkillSummary>> _state = const AsyncValue.loading();

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() => _state = const AsyncValue.loading());
    try {
      final list = await ref.read(skillsApiProvider).list();
      if (!mounted) return;
      list.sort((a, b) {
        // Vault overrides first (most-edited surface), then plain
        // builtins, then plain vault rows. Within each group: name.
        int rank(SkillSummary s) {
          if (s.overridesBuiltin) return 0;
          if (s.isBuiltin) return 1;
          return 2;
        }

        final ra = rank(a);
        final rb = rank(b);
        if (ra != rb) return ra - rb;
        return a.name.toLowerCase().compareTo(b.name.toLowerCase());
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

  Future<void> _openEditor({SkillSummary? existing}) async {
    final res = await Navigator.of(context).push<bool>(
      MaterialPageRoute<bool>(
        builder: (_) => SkillEditorScreen(existing: existing),
        fullscreenDialog: existing == null,
      ),
    );
    if (!mounted) return;
    if (res ?? false) await _load();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(t.skills.title),
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
        heroTag: 'skills_fab',
        // ignore: unnecessary_lambdas — _openEditor returns Future<void>
        onPressed: () => _openEditor(),
        icon: const Icon(Icons.add),
        label: Text(t.skills.newSkill),
      ),
    );
  }

  Widget _buildList(List<SkillSummary> list) {
    if (list.isEmpty) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Text(
            t.skills.emptyList,
            textAlign: TextAlign.center,
            style: Theme.of(context).textTheme.bodyMedium,
          ),
        ),
      );
    }
    return RefreshIndicator(
      onRefresh: _load,
      child: ListView.separated(
        itemCount: list.length,
        separatorBuilder: (_, __) => Divider(
          height: 1,
          color: Theme.of(context).dividerColor,
        ),
        itemBuilder: (_, i) => _SkillTile(
          skill: list[i],
          onTap: () => _openEditor(existing: list[i]),
        ),
      ),
    );
  }
}

class _SkillTile extends StatelessWidget {
  const _SkillTile({required this.skill, required this.onTap});
  final SkillSummary skill;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context).textTheme.bodySmall;
    return ListTile(
      onTap: onTap,
      leading: Icon(
        skill.isBuiltin
            ? Icons.lock_outline
            : Icons.edit_note_outlined,
        color: skill.isBuiltin
            ? Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6)
            : Theme.of(context).colorScheme.primary,
      ),
      title: Row(
        children: [
          Flexible(
            child: Text(
              skill.name.isEmpty ? skill.id : skill.name,
              overflow: TextOverflow.ellipsis,
              style: const TextStyle(fontWeight: FontWeight.w600),
            ),
          ),
          const SizedBox(width: 6),
          if (skill.isBuiltin)
            _Badge(
              label: 'built-in',
              color: Theme.of(context).colorScheme.onSurface.withValues(alpha: 0.6),
            )
          else if (skill.overridesBuiltin)
            _Badge(
              label: 'overrides',
              color: Theme.of(context).colorScheme.tertiary,
            )
          else
            _Badge(
              label: 'vault',
              color: Theme.of(context).colorScheme.primary,
            ),
        ],
      ),
      subtitle: DefaultTextStyle.merge(
        style: muted ?? const TextStyle(),
        child: Wrap(
          spacing: 6,
          runSpacing: 2,
          children: [
            Text(
              skill.id,
              style: const TextStyle(fontFamily: 'monospace'),
            ),
            if (skill.description.isNotEmpty)
              Text('· ${skill.description}'),
          ],
        ),
      ),
      trailing: const Icon(Icons.chevron_right),
    );
  }
}

class _Badge extends StatelessWidget {
  const _Badge({required this.label, required this.color});
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
              t.skills.failedToLoad,
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
