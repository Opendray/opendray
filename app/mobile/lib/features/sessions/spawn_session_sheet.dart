import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/providers_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/features/sessions/directory_picker_sheet.dart';

// Spawn-session bottom sheet. Loads providers live from
// /api/v1/providers when opened so the picker reflects whatever
// the operator has enabled.
//
// Returns the freshly-created SessionSummary via Navigator.pop
// so the caller can either refresh the list or jump straight
// into the new session's detail.
class SpawnSessionSheet extends ConsumerStatefulWidget {
  const SpawnSessionSheet({super.key});

  static Future<SessionSummary?> show(BuildContext context) {
    return showModalBottomSheet<SessionSummary>(
      context: context,
      isScrollControlled: true,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => const SpawnSessionSheet(),
    );
  }

  @override
  ConsumerState<SpawnSessionSheet> createState() => _SpawnSessionSheetState();
}

class _SpawnSessionSheetState extends ConsumerState<SpawnSessionSheet> {
  final _cwdCtrl = TextEditingController();
  final _nameCtrl = TextEditingController();
  final _argsCtrl = TextEditingController();
  String? _providerId;
  bool _submitting = false;
  String? _error;

  @override
  void dispose() {
    _cwdCtrl.dispose();
    _nameCtrl.dispose();
    _argsCtrl.dispose();
    super.dispose();
  }

  Future<void> _browseCwd() async {
    final picked = await DirectoryPickerSheet.show(
      context,
      initialPath: _cwdCtrl.text.trim().isEmpty ? null : _cwdCtrl.text.trim(),
    );
    if (picked != null && picked.isNotEmpty) {
      setState(() => _cwdCtrl.text = picked);
    }
  }

  Future<void> _submit() async {
    final cwd = _cwdCtrl.text.trim();
    if (_providerId == null || _providerId!.isEmpty || cwd.isEmpty) {
      setState(() => _error = 'Provider and working directory are required');
      return;
    }
    setState(() {
      _submitting = true;
      _error = null;
    });

    final argsRaw = _argsCtrl.text.trim();
    final args = argsRaw.isEmpty
        ? null
        : argsRaw
            .split(RegExp(r'\s+'))
            .where((s) => s.isNotEmpty)
            .toList();

    try {
      final session = await ref.read(sessionsApiProvider).create(
            CreateSessionRequest(
              providerId: _providerId!,
              cwd: cwd,
              name: _nameCtrl.text.trim().isEmpty ? null : _nameCtrl.text.trim(),
              args: args,
            ),
          );
      if (!mounted) return;
      Navigator.of(context).pop(session);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } on Object catch (e) {
      setState(() => _error = 'Failed to spawn session: $e');
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final asyncProviders = ref.watch(providersListProvider);
    final mq = MediaQuery.of(context);

    return Padding(
      padding: EdgeInsets.only(bottom: mq.viewInsets.bottom),
      child: SafeArea(
        top: false,
        child: SingleChildScrollView(
          padding: const EdgeInsets.fromLTRB(20, 16, 20, 24),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.stretch,
            mainAxisSize: MainAxisSize.min,
            children: [
              _SheetHandle(),
              const SizedBox(height: 8),
              Row(
                children: [
                  Expanded(
                    child: Text(
                      'New session',
                      style: Theme.of(context).textTheme.titleLarge,
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.close),
                    onPressed: _submitting
                        ? null
                        : () => Navigator.of(context).pop(),
                  ),
                ],
              ),
              const SizedBox(height: 16),
              _ProviderField(
                async: asyncProviders,
                value: _providerId,
                onChanged: _submitting
                    ? null
                    : (v) => setState(() => _providerId = v),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _cwdCtrl,
                enabled: !_submitting,
                autocorrect: false,
                keyboardType: TextInputType.url,
                decoration: InputDecoration(
                  labelText: 'Working directory',
                  hintText: '/Users/you/projects/foo',
                  helperText: 'Absolute path on the gateway host.',
                  suffixIcon: IconButton(
                    icon: const Icon(Icons.folder_open_outlined),
                    tooltip: 'Browse',
                    onPressed: _submitting ? null : _browseCwd,
                  ),
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _nameCtrl,
                enabled: !_submitting,
                decoration: const InputDecoration(
                  labelText: 'Name (optional)',
                  hintText: 'e.g. backend-refactor',
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _argsCtrl,
                enabled: !_submitting,
                autocorrect: false,
                decoration: const InputDecoration(
                  labelText: 'Extra args (optional)',
                  hintText: '--continue --verbose',
                  helperText:
                      "Whitespace-separated; blank uses the provider's defaults.",
                ),
              ),
              if (_error != null) ...[
                const SizedBox(height: 14),
                _InlineError(message: _error!),
              ],
              const SizedBox(height: 22),
              Row(
                children: [
                  Expanded(
                    child: OutlinedButton(
                      onPressed: _submitting
                          ? null
                          : () => Navigator.of(context).pop(),
                      child: const Text('Cancel'),
                    ),
                  ),
                  const SizedBox(width: 12),
                  Expanded(
                    child: FilledButton(
                      onPressed: _submitting ? null : _submit,
                      child: _submitting
                          ? const SizedBox(
                              height: 18,
                              width: 18,
                              child: CircularProgressIndicator(strokeWidth: 2),
                            )
                          : const Text('Spawn'),
                    ),
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

class _ProviderField extends ConsumerWidget {
  const _ProviderField({
    required this.async,
    required this.value,
    required this.onChanged,
  });

  final AsyncValue<List<ProviderSummary>> async;
  final String? value;
  final ValueChanged<String?>? onChanged;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    return async.when(
      data: (providers) {
        if (providers.isEmpty) {
          return _ProviderProblem(
            icon: Icons.inventory_2_outlined,
            title: 'No providers configured',
            message: 'The gateway has no CLI providers enabled. Configure '
                'one under Providers (web admin) or [providers] in '
                'config.toml, then tap Reload.',
            onReload: () => ref.invalidate(providersListProvider),
          );
        }
        // Default to the first enabled provider when nothing picked yet.
        final effectiveValue = value ??
            providers
                .firstWhere(
                  (p) => p.enabled,
                  orElse: () => providers.first,
                )
                .id;
        return DropdownButtonFormField<String>(
          initialValue: effectiveValue,
          decoration: const InputDecoration(labelText: 'Provider'),
          onChanged: onChanged,
          items: [
            for (final p in providers)
              DropdownMenuItem<String>(
                value: p.id,
                child: Text(p.enabled ? p.name : '${p.name} (disabled)'),
              ),
          ],
        );
      },
      loading: () => const SizedBox(
        height: 56,
        child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
      ),
      error: (e, _) => _ProviderProblem(
        icon: Icons.cloud_off_outlined,
        title: 'Could not load providers',
        message: e is ApiException
            ? '${e.statusCode == 0 ? "Network error" : "Server ${e.statusCode}"}: ${e.message}'
            : e.toString(),
        onReload: () => ref.invalidate(providersListProvider),
      ),
    );
  }
}

class _ProviderProblem extends StatelessWidget {
  const _ProviderProblem({
    required this.icon,
    required this.title,
    required this.message,
    required this.onReload,
  });

  final IconData icon;
  final String title;
  final String message;
  final VoidCallback onReload;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(14),
      decoration: BoxDecoration(
        color: scheme.error.withValues(alpha: 0.08),
        border: Border.all(color: scheme.error.withValues(alpha: 0.3)),
        borderRadius: BorderRadius.circular(10),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(icon, color: scheme.error),
          const SizedBox(width: 12),
          Expanded(
            child: Column(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Text(
                  title,
                  style: TextStyle(color: scheme.error, fontWeight: FontWeight.w600),
                ),
                const SizedBox(height: 4),
                Text(
                  message,
                  style: Theme.of(context).textTheme.bodySmall,
                ),
                const SizedBox(height: 8),
                OutlinedButton.icon(
                  onPressed: onReload,
                  icon: const Icon(Icons.refresh, size: 16),
                  label: const Text('Reload'),
                  style: OutlinedButton.styleFrom(
                    visualDensity: VisualDensity.compact,
                    foregroundColor: scheme.error,
                    side: BorderSide(color: scheme.error.withValues(alpha: 0.4)),
                  ),
                ),
              ],
            ),
          ),
        ],
      ),
    );
  }
}

class _SheetHandle extends StatelessWidget {
  @override
  Widget build(BuildContext context) {
    return Center(
      child: Container(
        width: 36,
        height: 4,
        decoration: BoxDecoration(
          color: Theme.of(context).dividerColor,
          borderRadius: BorderRadius.circular(2),
        ),
      ),
    );
  }
}

class _InlineError extends StatelessWidget {
  const _InlineError({required this.message});
  final String message;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Container(
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        color: scheme.error.withValues(alpha: 0.1),
        border: Border.all(color: scheme.error.withValues(alpha: 0.3)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(message, style: TextStyle(color: scheme.error)),
    );
  }
}
