import 'package:flutter/material.dart';

// Shared "this surface lands later" view for the inspector tabs
// that haven't been wired yet. Lists the planned UX bullets and
// the gateway endpoints the implementation will hit, so a future
// reader (or another agent) can pick up the task without spelunking
// the docs.
class InspectorPlaceholder extends StatelessWidget {
  const InspectorPlaceholder({
    required this.icon,
    required this.title,
    required this.bullets,
    required this.apis,
    super.key,
  });

  final IconData icon;
  final String title;
  final List<String> bullets;
  final List<String> apis;

  @override
  Widget build(BuildContext context) {
    final muted = Theme.of(context)
        .colorScheme
        .onSurface
        .withValues(alpha: 0.55);
    return ListView(
      padding: const EdgeInsets.fromLTRB(20, 32, 20, 32),
      children: [
        Center(
          child: Icon(icon, size: 56, color: muted),
        ),
        const SizedBox(height: 12),
        Center(
          child: Text(
            '$title — coming soon',
            style: Theme.of(context).textTheme.titleMedium,
          ),
        ),
        const SizedBox(height: 16),
        Text(
          'Planned UX',
          style: Theme.of(context).textTheme.labelMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
        ),
        const SizedBox(height: 6),
        for (final b in bullets)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 2),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Padding(
                  padding: const EdgeInsets.only(top: 8),
                  child: Container(
                    width: 4,
                    height: 4,
                    decoration: BoxDecoration(
                      color: muted,
                      shape: BoxShape.circle,
                    ),
                  ),
                ),
                const SizedBox(width: 10),
                Expanded(
                  child: Text(
                    b,
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ),
              ],
            ),
          ),
        const SizedBox(height: 20),
        Text(
          'Backend endpoints (already in the gateway)',
          style: Theme.of(context).textTheme.labelMedium?.copyWith(
                fontWeight: FontWeight.w600,
              ),
        ),
        const SizedBox(height: 6),
        for (final a in apis)
          Padding(
            padding: const EdgeInsets.symmetric(vertical: 2),
            child: SelectableText(
              a,
              style: const TextStyle(
                fontFamily: 'monospace',
                fontSize: 12,
                height: 1.4,
              ),
            ),
          ),
      ],
    );
  }
}
