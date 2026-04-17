import 'package:flutter/material.dart';
import '../../../core/models/session.dart';
import '../../../shared/theme/app_theme.dart';

const _icons = <String, String>{
  'claude': '🟣', 'gemini': '✨', 'codex': '🤖',
  'lmstudio': '🧠', 'ollama': '🦙', 'terminal': '⬛',
};

class SessionCard extends StatelessWidget {
  final Session session;
  final VoidCallback onTap;
  final VoidCallback onStart;
  final VoidCallback onStop;
  final VoidCallback onDelete;

  const SessionCard({
    super.key,
    required this.session,
    required this.onTap,
    required this.onStart,
    required this.onStop,
    required this.onDelete,
  });

  String get _icon => _icons[session.sessionType] ?? '?';

  Color get _statusColor => switch (session.status) {
    'running' => AppColors.success,
    'error' => AppColors.error,
    _ => AppColors.textMuted,
  };

  String _shortenPath(String path) {
    final parts = path.split('/');
    return parts.length > 3 ? '.../${parts.sublist(parts.length - 2).join('/')}' : path;
  }

  String _timeAgo(DateTime dt) {
    final diff = DateTime.now().difference(dt);
    if (diff.inMinutes < 1) return 'now';
    if (diff.inMinutes < 60) return '${diff.inMinutes}m';
    if (diff.inHours < 24) return '${diff.inHours}h';
    return '${diff.inDays}d';
  }

  @override
  Widget build(BuildContext context) {
    return Card(
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(12),
        child: Padding(
          padding: const EdgeInsets.all(14),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              // Row 1: icon + name + status
              Row(
                children: [
                  Text(_icon, style: const TextStyle(fontSize: 22)),
                  const SizedBox(width: 10),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          session.name.isNotEmpty ? session.name : _shortenPath(session.cwd),
                          style: const TextStyle(fontWeight: FontWeight.w500, fontSize: 14),
                          overflow: TextOverflow.ellipsis,
                        ),
                        const SizedBox(height: 2),
                        Text(session.cwd, style: const TextStyle(color: AppColors.textMuted, fontSize: 11), overflow: TextOverflow.ellipsis),
                      ],
                    ),
                  ),
                  Container(
                    width: 9, height: 9,
                    decoration: BoxDecoration(
                      color: _statusColor,
                      shape: BoxShape.circle,
                      boxShadow: session.isRunning ? [BoxShadow(color: _statusColor.withValues(alpha: 0.5), blurRadius: 6)] : null,
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 10),
              // Row 2: meta
              Row(
                children: [
                  _Badge(session.sessionType),
                  if (session.model.isNotEmpty) ...[const SizedBox(width: 6), _Badge(session.model)],
                  const Spacer(),
                  Text(_timeAgo(session.lastActiveAt), style: const TextStyle(color: AppColors.textMuted, fontSize: 11)),
                  if (session.totalCostUsd > 0) ...[
                    const SizedBox(width: 8),
                    Text('\$${session.totalCostUsd.toStringAsFixed(2)}', style: const TextStyle(color: AppColors.textMuted, fontSize: 11)),
                  ],
                ],
              ),
              const SizedBox(height: 10),
              // Row 3: actions
              Row(
                children: [
                  if (!session.isRunning)
                    Expanded(child: _ActionButton(label: 'Start', color: AppColors.success, bgColor: AppColors.successSoft, onTap: onStart)),
                  if (session.isRunning)
                    Expanded(child: _ActionButton(label: 'Stop', color: AppColors.warning, bgColor: AppColors.warningSoft, onTap: onStop)),
                  if (!session.isRunning) ...[
                    const SizedBox(width: 8),
                    _ActionButton(label: 'Delete', color: AppColors.error, bgColor: AppColors.errorSoft, onTap: onDelete),
                  ],
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _Badge extends StatelessWidget {
  final String text;
  const _Badge(this.text);
  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
      decoration: BoxDecoration(color: AppColors.surfaceAlt, borderRadius: BorderRadius.circular(4)),
      child: Text(text, style: const TextStyle(color: AppColors.textMuted, fontSize: 10)),
    );
  }
}

class _ActionButton extends StatelessWidget {
  final String label;
  final Color color;
  final Color bgColor;
  final VoidCallback onTap;
  const _ActionButton({required this.label, required this.color, required this.bgColor, required this.onTap});
  @override
  Widget build(BuildContext context) {
    return Material(
      color: bgColor,
      borderRadius: BorderRadius.circular(8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(8),
        child: Padding(
          padding: const EdgeInsets.symmetric(vertical: 8, horizontal: 14),
          child: Center(child: Text(label, style: TextStyle(color: color, fontSize: 12, fontWeight: FontWeight.w500))),
        ),
      ),
    );
  }
}
