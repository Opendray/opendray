import 'package:flutter/material.dart';

import 'package:opendray/core/i18n/strings.g.dart';

/// Names which credential a PR/issue call authenticated as.
///
/// It used to be invisible: the panel said nothing while a token was
/// working, and on failure showed the forge's error verbatim. That
/// error never mentions the credential — GitHub answers a token that
/// lacks the repo with a 403 reading "Write access to repository not
/// granted", even for a read — so the single fact that would explain it
/// was the one fact not on screen.
///
/// Deliberately quiet. When an owner-scoped credential matches exactly
/// there is nothing to explain, and a line on every panel would be
/// noise that trains people to ignore it. It speaks only when the
/// answer is non-obvious: a host-wide credential standing in for an
/// owner that has none, or a call that actually failed.
class CredentialNote extends StatelessWidget {
  const CredentialNote({
    required this.scope,
    required this.remoteOwner,
    required this.isFallback,
    required this.failed,
    super.key,
  });

  /// "host" or "host/owner" for the resolved credential; empty hides.
  final String scope;
  final String remoteOwner;
  final bool isFallback;

  /// True when the call errored — then the credential is always worth
  /// naming, fallback or not.
  final bool failed;

  @override
  Widget build(BuildContext context) {
    if (scope.isEmpty) return const SizedBox.shrink();
    if (!isFallback && !failed) return const SizedBox.shrink();

    final theme = Theme.of(context);
    return Padding(
      padding: const EdgeInsets.only(top: 6),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Icon(Icons.key_outlined, size: 14, color: theme.colorScheme.outline),
          const SizedBox(width: 6),
          Expanded(
            child: Text(
              isFallback
                  ? t.sessions.inspector.git
                        .credentialFallback(owner: remoteOwner, scope: scope)
                  : t.sessions.inspector.git.credentialUsed(scope: scope),
              style: theme.textTheme.bodySmall?.copyWith(
                color: theme.colorScheme.outline,
              ),
            ),
          ),
        ],
      ),
    );
  }
}
