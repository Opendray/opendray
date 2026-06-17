import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/claude_accounts_api.dart';
import 'package:opendray/core/api/providers_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// DefaultAgentFields renders the provider / model / claude-account
// selectors for an integration's spawn defaults, shared by the register
// and edit forms. Provider + account are dropdowns; model is a free-text
// field (its controller is owned by the parent form, matching the
// existing form pattern). Empty = no default.
//
// The account selector applies only when the default provider is
// "claude"; it stays visible but disabled otherwise, with a hint.
class DefaultAgentFields extends ConsumerWidget {
  const DefaultAgentFields({
    required this.providerId,
    required this.claudeAccountId,
    required this.modelController,
    required this.onProviderChanged,
    required this.onAccountChanged,
    super.key,
  });

  // Sentinel for "no default" — a DropdownMenuItem can't use a bare
  // empty string as a distinct value cleanly, so we map '' ⇄ _none.
  static const String _none = '__none__';

  final String providerId;
  final String claudeAccountId;
  final TextEditingController modelController;
  final ValueChanged<String> onProviderChanged;
  final ValueChanged<String> onAccountChanged;

  @override
  Widget build(BuildContext context, WidgetRef ref) {
    final providers = ref.watch(providersListProvider);
    final accounts = ref.watch(claudeAccountsListProvider);
    final isClaude = providerId == 'claude';
    final theme = Theme.of(context);

    return Container(
      margin: const EdgeInsets.only(bottom: 12),
      padding: const EdgeInsets.all(12),
      decoration: BoxDecoration(
        border: Border.all(color: theme.dividerColor.withValues(alpha: 0.4)),
        borderRadius: BorderRadius.circular(8),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            t.integrations.defaultAgent.title,
            style: theme.textTheme.titleSmall,
          ),
          const SizedBox(height: 2),
          Text(
            t.integrations.defaultAgent.description,
            style: theme.textTheme.bodySmall?.copyWith(
              color: theme.colorScheme.onSurfaceVariant,
            ),
          ),
          const SizedBox(height: 12),

          // Provider dropdown.
          DropdownButtonFormField<String>(
            initialValue: providerId.isEmpty ? _none : providerId,
            isExpanded: true,
            decoration: InputDecoration(
              labelText: t.integrations.defaultAgent.providerLabel,
              border: const OutlineInputBorder(),
            ),
            items: [
              DropdownMenuItem(
                value: _none,
                child: Text(t.integrations.defaultAgent.providerNone),
              ),
              ...providers.maybeWhen(
                data: (list) => list.map(
                  (p) => DropdownMenuItem(
                    value: p.id,
                    child: Text('${p.name} (${p.id})'),
                  ),
                ),
                orElse: () => const <DropdownMenuItem<String>>[],
              ),
            ],
            onChanged: (v) => onProviderChanged(v == _none ? '' : (v ?? '')),
          ),
          const SizedBox(height: 12),

          // Model free-text field (controller owned by parent).
          TextField(
            controller: modelController,
            autocorrect: false,
            decoration: InputDecoration(
              labelText: t.integrations.defaultAgent.modelLabel,
              hintText: t.integrations.defaultAgent.modelHint,
              border: const OutlineInputBorder(),
            ),
          ),
          const SizedBox(height: 12),

          // Claude account dropdown — only meaningful for the claude provider.
          DropdownButtonFormField<String>(
            initialValue: claudeAccountId.isEmpty ? _none : claudeAccountId,
            isExpanded: true,
            decoration: InputDecoration(
              labelText: t.integrations.defaultAgent.accountLabel,
              helperText: t.integrations.defaultAgent.accountHint,
              helperMaxLines: 2,
              border: const OutlineInputBorder(),
            ),
            items: [
              DropdownMenuItem(
                value: _none,
                child: Text(t.integrations.defaultAgent.accountNone),
              ),
              ...accounts.maybeWhen(
                data: (list) => list.map(
                  (a) => DropdownMenuItem(
                    value: a.id,
                    child: Text(
                      a.tokenFilled
                          ? a.displayName
                          : '${a.displayName} '
                              '${t.integrations.defaultAgent.accountTokenMissing}',
                    ),
                  ),
                ),
                orElse: () => const <DropdownMenuItem<String>>[],
              ),
            ],
            onChanged: isClaude
                ? (v) => onAccountChanged(v == _none ? '' : (v ?? ''))
                : null,
          ),
        ],
      ),
    );
  }
}
