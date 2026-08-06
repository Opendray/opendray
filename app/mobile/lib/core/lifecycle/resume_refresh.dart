import 'package:flutter/widgets.dart';
import 'package:flutter_riverpod/flutter_riverpod.dart';
import 'package:opendray/core/api/sessions_api.dart';

// Anything shorter than this is an incidental interruption — a
// notification shade, a permission sheet, the app switcher — and the
// sockets are still good. Refreshing on those would just add noise.
const kResumeRefreshThreshold = Duration(seconds: 5);

/// Whether a suspension that began at [pausedAt] and ended at [resumedAt]
/// was long enough to be worth refetching for.
bool shouldRefreshOnResume({
  required DateTime? pausedAt,
  required DateTime resumedAt,
}) {
  if (pausedAt == null) return false;
  return resumedAt.difference(pausedAt) >= kResumeRefreshThreshold;
}

/// Refetches session data when the app comes back from a real
/// suspension.
///
/// While suspended the OS quietly tears down our TCP connections, so the
/// first request afterwards tends to go out on a dead socket and stall
/// until it times out. The retry interceptor in `dio_provider.dart`
/// recovers that transparently; this widget makes the recovery start
/// immediately on resume rather than whenever the operator next taps
/// something, so the list they are looking at is already correct.
///
/// Deliberately narrow: it invalidates the session providers only, not
/// `dioProvider`. Rebuilding the client would tear down every other API
/// provider with it, including long-running backup transfers.
class ResumeRefresh extends ConsumerStatefulWidget {
  const ResumeRefresh({required this.child, super.key});

  final Widget child;

  @override
  ConsumerState<ResumeRefresh> createState() => _ResumeRefreshState();
}

class _ResumeRefreshState extends ConsumerState<ResumeRefresh>
    with WidgetsBindingObserver {
  DateTime? _pausedAt;

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    super.dispose();
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    switch (state) {
      case AppLifecycleState.paused:
      case AppLifecycleState.detached:
      case AppLifecycleState.hidden:
        _pausedAt ??= DateTime.now();
      case AppLifecycleState.resumed:
        final pausedAt = _pausedAt;
        _pausedAt = null;
        if (shouldRefreshOnResume(
          pausedAt: pausedAt,
          resumedAt: DateTime.now(),
        )) {
          ref
            ..invalidate(sessionsListProvider)
            ..invalidate(sessionByIdProvider);
        }
      case AppLifecycleState.inactive:
        break;
    }
  }

  @override
  Widget build(BuildContext context) => widget.child;
}
