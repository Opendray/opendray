import 'package:flutter/material.dart';
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:opendray/features/project/project_screen.dart';
import 'package:path/path.dart' as p;

// CortexTab — the session inspector's entry into this project's Cortex
// workspace (overview / goal / plan / tech / activity / journal / inbox /
// memory hygiene). Mirrors the web inspector's "Cortex" tab.
//
// Kept deliberately separate from the Vault tab: the Vault is the markdown
// notes utility (personal scratchpad + vault-mapped project docs), the Cortex
// is the AI-maintained memory. The session detail screen already exposes a
// flag-icon shortcut to the same workspace; this tab gives the inspector its
// own discoverable entry so the operator doesn't have to back out.
class CortexTab extends StatelessWidget {
  const CortexTab({required this.cwd, super.key});

  final String cwd;

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final base = p.basename(cwd);
    return ListView(
      padding: const EdgeInsets.all(24),
      children: [
        const SizedBox(height: 16),
        Icon(
          Icons.psychology_outlined,
          size: 48,
          color: theme.colorScheme.primary,
        ),
        const SizedBox(height: 16),
        Text(
          t.sessions.inspector.cortex.title,
          style: theme.textTheme.titleMedium,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 4),
        Text(
          base.isEmpty ? cwd : base,
          style: theme.textTheme.bodySmall?.copyWith(
            fontFamily: 'monospace',
            color: theme.colorScheme.onSurfaceVariant,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 12),
        Text(
          t.sessions.inspector.cortex.blurb,
          style: theme.textTheme.bodySmall,
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 24),
        ElevatedButton.icon(
          icon: const Icon(Icons.open_in_new),
          label: Text(t.sessions.inspector.cortex.open),
          onPressed: () {
            Navigator.of(context).push(
              MaterialPageRoute<void>(
                builder: (_) => ProjectScreen(initialCwd: cwd),
              ),
            );
          },
        ),
      ],
    );
  }
}
