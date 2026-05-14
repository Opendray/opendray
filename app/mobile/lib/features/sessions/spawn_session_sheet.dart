import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/providers_api.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/sessions/directory_picker_sheet.dart';

// Provider id that triggers the Claude-account picker. Multi-account
// support is Claude-only on the gateway today; other providers
// spawn against env-var credentials and have no account concept.
const _claudeProviderId = 'claude';

// Per-session "bypass" flags. The provider config (in Providers
// settings) can bake these into every session; the per-session
// toggle below is purely additive — when OFF we send no extra
// args, when ON we append the flag(s) the user picked.
//
// Backend already concatenates session args after config args, so
// passing the flag a second time when the provider config already
// has it is harmless: claude/gemini boolean flags are idempotent;
// codex's --ask-for-approval gets overridden by the second occurrence
// (later wins in cobra parse).
const Map<String, List<String>> _bypassFlagsByProvider = {
  'claude': ['--dangerously-skip-permissions'],
  'codex': ['--ask-for-approval', 'never', '-c', 'approval_policy="never"'],
  'gemini': ['--yolo'],
};

// Per-provider label for the bypass toggle. Different CLIs name
// the concept differently and operators recognise their tool's
// term; pretending it's all "Auto-approve" would be confusing.
String? _bypassLabelFor(String providerId) {
  return switch (providerId) {
    'claude' => t.sessions.spawnSheet.bypass.labelClaude,
    'codex' => t.sessions.spawnSheet.bypass.labelCodex,
    'gemini' => t.sessions.spawnSheet.bypass.labelGemini,
    _ => null,
  };
}

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
  // null = "default" (let the gateway use env / system credentials),
  // any other value = a specific Claude account row id.
  //
  // Multi-account mode (≥2 enabled accounts) forces a non-null
  // value — see _ClaudeAccountField for the auto-pick logic.
  String? _claudeAccountId;
  // Per-session bypass toggle. Defaults OFF — operators opt in
  // explicitly per spawn. When ON, _submit appends the right
  // flag(s) for the selected provider to the args list.
  bool _bypassEnabled = false;
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
      setState(() => _error = t.sessions.spawnSheet.errorRequired);
      return;
    }
    setState(() {
      _submitting = true;
      _error = null;
    });

    final argsRaw = _argsCtrl.text.trim();
    final userArgs = argsRaw.isEmpty
        ? <String>[]
        : argsRaw
            .split(RegExp(r'\s+'))
            .where((s) => s.isNotEmpty)
            .toList();

    // Compose final args: bypass flags first (so the operator's
    // explicit Extra args can still override them by appearing
    // later — important for codex where --ask-for-approval is a
    // last-wins string).
    final bypassFlags = _bypassEnabled
        ? (_bypassFlagsByProvider[_providerId] ?? const <String>[])
        : const <String>[];
    final composed = [...bypassFlags, ...userArgs];
    final args = composed.isEmpty ? null : composed;

    try {
      // claude_account_id is only relevant when the picked provider
      // is Claude — for everything else it's a no-op on the gateway,
      // but we still skip sending it to keep the request payload tight.
      final isClaude = _providerId == _claudeProviderId;
      final session = await ref.read(sessionsApiProvider).create(
            CreateSessionRequest(
              providerId: _providerId!,
              cwd: cwd,
              name: _nameCtrl.text.trim().isEmpty ? null : _nameCtrl.text.trim(),
              args: args,
              claudeAccountId: isClaude ? _claudeAccountId : null,
            ),
          );
      if (!mounted) return;
      Navigator.of(context).pop(session);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } on Object catch (e) {
      setState(() => _error = t.sessions.spawnSheet.errorGeneric(error: e.toString()));
    } finally {
      if (mounted) setState(() => _submitting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final asyncProviders = ref.watch(providersListProvider);
    final mq = MediaQuery.of(context);

    // Pre-pick the default provider as soon as the list loads. This
    // is what makes the Claude-account section appear immediately
    // when the operator opens the sheet — previously _providerId
    // stayed null until the user manually re-tapped the dropdown,
    // because the dropdown's "first enabled" fallback was a UI
    // display-only computation that never wrote back to state.
    asyncProviders.whenData((providers) {
      if (_providerId == null && providers.isNotEmpty) {
        final first = providers.firstWhere(
          (p) => p.enabled,
          orElse: () => providers.first,
        );
        WidgetsBinding.instance.addPostFrameCallback((_) {
          if (mounted && _providerId == null) {
            setState(() => _providerId = first.id);
          }
        });
      }
    });

    final isClaude = _providerId == _claudeProviderId;
    final bypassLabel =
        _providerId == null ? null : _bypassLabelFor(_providerId!);

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
                      t.sessions.spawnSheet.title,
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
                    : (v) => setState(() {
                          _providerId = v;
                          // Reset account when provider changes — the
                          // account picker is provider-scoped.
                          _claudeAccountId = null;
                          // Reset bypass too — each provider has its
                          // own flag, and a stale "ON" carrying the
                          // wrong flag would surprise the operator.
                          _bypassEnabled = false;
                        }),
              ),
              if (isClaude) ...[
                const SizedBox(height: 14),
                _ClaudeAccountField(
                  value: _claudeAccountId,
                  onChanged: _submitting
                      ? null
                      : (v) => setState(() => _claudeAccountId = v),
                ),
              ],
              const SizedBox(height: 14),
              TextField(
                controller: _cwdCtrl,
                enabled: !_submitting,
                autocorrect: false,
                keyboardType: TextInputType.url,
                decoration: InputDecoration(
                  labelText: t.sessions.spawnSheet.cwdLabel,
                  hintText: t.sessions.spawnSheet.cwdHint,
                  helperText: t.sessions.spawnSheet.cwdHelper,
                  suffixIcon: IconButton(
                    icon: const Icon(Icons.folder_open_outlined),
                    tooltip: t.sessions.spawnSheet.browse,
                    onPressed: _submitting ? null : _browseCwd,
                  ),
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _nameCtrl,
                enabled: !_submitting,
                decoration: InputDecoration(
                  labelText: t.sessions.spawnSheet.nameLabel,
                  hintText: t.sessions.spawnSheet.nameHint,
                ),
              ),
              const SizedBox(height: 14),
              TextField(
                controller: _argsCtrl,
                enabled: !_submitting,
                autocorrect: false,
                decoration: InputDecoration(
                  labelText: t.sessions.spawnSheet.argsLabel,
                  hintText: t.sessions.spawnSheet.argsHint,
                  helperText: t.sessions.spawnSheet.argsHelper,
                ),
              ),
              if (bypassLabel != null) ...[
                const SizedBox(height: 6),
                SwitchListTile.adaptive(
                  value: _bypassEnabled,
                  onChanged: _submitting
                      ? null
                      : (v) => setState(() => _bypassEnabled = v),
                  title: Text(bypassLabel),
                  subtitle: Text(
                    _bypassEnabled
                        ? t.sessions.spawnSheet.bypass.subtitleOn
                        : t.sessions.spawnSheet.bypass.subtitleOff,
                    style: Theme.of(context).textTheme.bodySmall,
                  ),
                  contentPadding: EdgeInsets.zero,
                  visualDensity: VisualDensity.compact,
                ),
              ],
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
                      child: Text(t.common.cancel),
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
                          : Text(t.sessions.spawnSheet.spawn),
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
            title: t.sessions.spawnSheet.noProviders.title,
            message: t.sessions.spawnSheet.noProviders.message,
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
          decoration: InputDecoration(
            labelText: t.sessions.spawnSheet.providerLabel,
          ),
          onChanged: onChanged,
          items: [
            for (final p in providers)
              DropdownMenuItem<String>(
                value: p.id,
                child: Text(
                  p.enabled
                      ? p.name
                      : '${p.name}${t.sessions.spawnSheet.disabledSuffix}',
                ),
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
        title: t.sessions.spawnSheet.providerLoadError.title,
        message: e is ApiException
            ? t.sessions.spawnSheet.providerLoadError.format(
                prefix: e.statusCode == 0
                    ? t.sessions.spawnSheet.providerLoadError.networkError
                    : t.sessions.spawnSheet.providerLoadError
                        .serverPrefix(code: e.statusCode.toString()),
                message: e.message,
              )
            : e.toString(),
        onReload: () => ref.invalidate(providersListProvider),
      ),
    );
  }
}

class _ClaudeAccountField extends ConsumerWidget {
  const _ClaudeAccountField({required this.value, required this.onChanged});

  final String? value;
  final ValueChanged<String?>? onChanged;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final async = ref.watch(claudeAccountsListProvider);
    return async.when(
      data: (accounts) {
        // No accounts configured → hide the dropdown entirely. The
        // gateway will fall back to env-var auth, which is fine for
        // single-account setups. Operators who want multi-account
        // configure them in web admin → Settings → Accounts.
        if (accounts.isEmpty) {
          return _AccountHint(
            text: t.sessions.spawnSheet.claudeAccount.noneHint,
          );
        }
        // Multi-account (2+ enabled): drop the Default option and
        // force an explicit pick. When you've gone to the trouble of
        // registering multiple Claude accounts, "default to env"
        // almost certainly isn't what you want for the next session.
        // Single-account mode keeps Default as an option for parity
        // with the pre-PR-54 behaviour.
        final enabledCount = accounts.where((a) => a.enabled).length;
        final multiAccount = enabledCount >= 2;
        final firstEnabled = multiAccount
            ? accounts.firstWhere(
                (a) => a.enabled,
                orElse: () => accounts.first,
              )
            : null;

        // In multi-account mode, the controller-managed value must
        // be one of the account ids — never null. We can't call
        // setState from a build, so we use a post-frame callback to
        // surface the auto-pick to the parent's onChanged.
        if (multiAccount && value == null && onChanged != null) {
          WidgetsBinding.instance.addPostFrameCallback((_) {
            onChanged!(firstEnabled!.id);
          });
        }

        final effectiveValue =
            multiAccount && value == null ? firstEnabled!.id : value;

        return DropdownButtonFormField<String?>(
          initialValue: effectiveValue,
          decoration: InputDecoration(
            labelText: t.sessions.spawnSheet.claudeAccount.label,
            helperText: multiAccount
                ? t.sessions.spawnSheet.claudeAccount.helperMulti
                : t.sessions.spawnSheet.claudeAccount.helperSingle,
          ),
          onChanged: onChanged,
          items: [
            if (!multiAccount)
              DropdownMenuItem<String?>(
                value: null,
                child: Text(t.sessions.spawnSheet.claudeAccount.kDefault),
              ),
            for (final a in accounts)
              DropdownMenuItem<String?>(
                value: a.id,
                child: Text(_accountLabel(a)),
              ),
          ],
        );
      },
      loading: () => const SizedBox(
        height: 56,
        child: Center(child: CircularProgressIndicator(strokeWidth: 2)),
      ),
      // Error here is not fatal — fall back to "Default" silently with
      // a small inline note so the user can still spawn the session.
      error: (e, _) => _AccountHint(
        text: t.sessions.spawnSheet.claudeAccount.errorHint(
          error: e is ApiException ? e.message : e.toString(),
        ),
      ),
    );
  }

  static String _accountLabel(ClaudeAccountSummary a) {
    final base = a.displayName;
    if (!a.enabled) {
      return '$base${t.sessions.spawnSheet.claudeAccount.disabledSuffix}';
    }
    if (!a.tokenFilled) {
      return '$base${t.sessions.spawnSheet.claudeAccount.noTokenSuffix}';
    }
    return base;
  }
}

class _AccountHint extends StatelessWidget {
  const _AccountHint({required this.text});
  final String text;

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.all(10),
      decoration: BoxDecoration(
        color: Theme.of(context).dividerColor.withValues(alpha: 0.18),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Text(
        text,
        style: Theme.of(context).textTheme.bodySmall,
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
                  label: Text(t.sessions.spawnSheet.noProviders.reload),
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
