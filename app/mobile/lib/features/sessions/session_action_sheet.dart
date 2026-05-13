import 'package:flutter/material.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';

import 'package:opendray/core/api/api_exception.dart';
import 'package:opendray/core/api/models.dart';
import 'package:opendray/core/api/sessions_api.dart';
import 'package:opendray/core/i18n/strings.g.dart';

// Result returned to the caller so it knows what happened (and
// can choose to refresh the list, navigate away, etc.).
enum SessionActionResult { stopped, started, deleted }

// State-aware action sheet for a single session card.
//   running / idle / pending → Stop, Delete
//   stopped / ended           → Restart, Delete
// Delete always asks a second confirmation step.
class SessionActionSheet extends ConsumerStatefulWidget {
  const SessionActionSheet({required this.session, super.key});

  final SessionSummary session;

  static Future<SessionActionResult?> show(
    BuildContext context, {
    required SessionSummary session,
  }) {
    return showModalBottomSheet<SessionActionResult>(
      context: context,
      useSafeArea: true,
      backgroundColor: Theme.of(context).colorScheme.surface,
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (_) => SessionActionSheet(session: session),
    );
  }

  @override
  ConsumerState<SessionActionSheet> createState() =>
      _SessionActionSheetState();
}

class _SessionActionSheetState extends ConsumerState<SessionActionSheet> {
  bool _busy = false;
  bool _confirmDelete = false;
  String? _busyVerb;
  String? _error;

  Future<void> _run(
    String verb,
    Future<void> Function(SessionsApi) op,
    SessionActionResult result,
  ) async {
    setState(() {
      _busy = true;
      _busyVerb = verb;
      _error = null;
    });
    try {
      await op(ref.read(sessionsApiProvider));
      if (!mounted) return;
      Navigator.of(context).pop(result);
    } on ApiException catch (e) {
      setState(() => _error = e.message);
    } on Object catch (e) {
      final errStr = e.toString();
      setState(() => _error = switch (verb) {
            'stop' => t.sessions.action.errors.stop(error: errStr),
            'start' => t.sessions.action.errors.start(error: errStr),
            'delete' => t.sessions.action.errors.delete(error: errStr),
            _ => errStr,
          });
    } finally {
      if (mounted) setState(() => _busy = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final s = widget.session;
    return SafeArea(
      top: false,
      child: Padding(
        padding: const EdgeInsets.fromLTRB(20, 16, 20, 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            _SheetHandle(),
            const SizedBox(height: 12),
            Text(
              s.displayName,
              style: Theme.of(context).textTheme.titleMedium,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            const SizedBox(height: 4),
            Text(
              '${s.providerId}  ·  ${s.state.wire}',
              style: Theme.of(context).textTheme.bodySmall,
            ),
            Text(
              s.cwd,
              style: Theme.of(context).textTheme.bodySmall,
              maxLines: 1,
              overflow: TextOverflow.ellipsis,
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              _InlineError(message: _error!),
            ],
            const SizedBox(height: 16),
            if (_confirmDelete)
              _DeleteConfirm(
                busy: _busy,
                onCancel: () => setState(() => _confirmDelete = false),
                onConfirm: () => _run(
                  'delete',
                  (api) => api.delete(s.id),
                  SessionActionResult.deleted,
                ),
              )
            else
              _Actions(
                session: s,
                busy: _busy,
                busyVerb: _busyVerb,
                onStop: () => _run(
                  'stop',
                  (api) async {
                    await api.stop(s.id);
                  },
                  SessionActionResult.stopped,
                ),
                onStart: () => _run(
                  'start',
                  (api) async {
                    await api.start(s.id);
                  },
                  SessionActionResult.started,
                ),
                onDelete: () => setState(() => _confirmDelete = true),
              ),
          ],
        ),
      ),
    );
  }
}

class _Actions extends StatelessWidget {
  const _Actions({
    required this.session,
    required this.busy,
    required this.busyVerb,
    required this.onStop,
    required this.onStart,
    required this.onDelete,
  });

  final SessionSummary session;
  final bool busy;
  final String? busyVerb;
  final VoidCallback onStop;
  final VoidCallback onStart;
  final VoidCallback onDelete;

  @override
  Widget build(BuildContext context) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      children: [
        if (session.isLive)
          _ActionTile(
            icon: Icons.stop_circle_outlined,
            label: busyVerb == 'stop'
                ? t.sessions.action.stopping
                : t.sessions.action.stop,
            description: t.sessions.action.stopDescription,
            onTap: busy ? null : onStop,
          ),
        if (session.isFinished)
          _ActionTile(
            icon: Icons.play_circle_outline,
            label: busyVerb == 'start'
                ? t.sessions.action.restarting
                : t.sessions.action.restart,
            description: t.sessions.action.restartDescription,
            onTap: busy ? null : onStart,
          ),
        _ActionTile(
          icon: Icons.delete_outline,
          label: t.sessions.action.delete,
          description: t.sessions.action.deleteDescription,
          destructive: true,
          onTap: busy ? null : onDelete,
        ),
      ],
    );
  }
}

class _ActionTile extends StatelessWidget {
  const _ActionTile({
    required this.icon,
    required this.label,
    required this.description,
    required this.onTap,
    this.destructive = false,
  });

  final IconData icon;
  final String label;
  final String description;
  final VoidCallback? onTap;
  final bool destructive;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    final fg = destructive ? scheme.error : scheme.onSurface;
    return Padding(
      padding: const EdgeInsets.only(bottom: 8),
      child: InkWell(
        onTap: onTap,
        borderRadius: BorderRadius.circular(10),
        child: Container(
          padding: const EdgeInsets.all(14),
          decoration: BoxDecoration(
            color: destructive
                ? scheme.error.withValues(alpha: 0.08)
                : null,
            border: Border.all(
              color: destructive
                  ? scheme.error.withValues(alpha: 0.3)
                  : scheme.outline,
            ),
            borderRadius: BorderRadius.circular(10),
          ),
          child: Row(
            children: [
              Icon(icon, color: fg),
              const SizedBox(width: 12),
              Expanded(
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.start,
                  children: [
                    Text(
                      label,
                      style: TextStyle(
                        color: fg,
                        fontWeight: FontWeight.w600,
                      ),
                    ),
                    const SizedBox(height: 2),
                    Text(
                      description,
                      style: Theme.of(context).textTheme.bodySmall,
                    ),
                  ],
                ),
              ),
            ],
          ),
        ),
      ),
    );
  }
}

class _DeleteConfirm extends StatelessWidget {
  const _DeleteConfirm({
    required this.busy,
    required this.onCancel,
    required this.onConfirm,
  });

  final bool busy;
  final VoidCallback onCancel;
  final VoidCallback onConfirm;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Text(
          t.sessions.action.deleteConfirm,
          style: Theme.of(context).textTheme.bodyMedium,
        ),
        const SizedBox(height: 16),
        Row(
          children: [
            Expanded(
              child: OutlinedButton(
                onPressed: busy ? null : onCancel,
                child: Text(t.common.cancel),
              ),
            ),
            const SizedBox(width: 12),
            Expanded(
              child: FilledButton(
                style: FilledButton.styleFrom(
                  backgroundColor: scheme.error,
                  foregroundColor: scheme.onError,
                ),
                onPressed: busy ? null : onConfirm,
                child: busy
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : Text(t.common.delete),
              ),
            ),
          ],
        ),
      ],
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
