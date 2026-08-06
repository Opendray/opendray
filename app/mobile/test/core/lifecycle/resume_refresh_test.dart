import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/lifecycle/resume_refresh.dart';

// A resume only refetches after a real suspension. Incidental
// interruptions (notification shade, app switcher, a permission sheet)
// leave the sockets alive, so refreshing on those is pure noise.
void main() {
  group('shouldRefreshOnResume', () {
    final now = DateTime(2026, 8, 7, 12);

    test('no refresh when the app was never backgrounded', () {
      expect(shouldRefreshOnResume(pausedAt: null, resumedAt: now), isFalse);
    });

    test('no refresh for a brief interruption', () {
      expect(
        shouldRefreshOnResume(
          pausedAt: now.subtract(const Duration(seconds: 2)),
          resumedAt: now,
        ),
        isFalse,
      );
    });

    test('refreshes once the suspension reaches the threshold', () {
      expect(
        shouldRefreshOnResume(
          pausedAt: now.subtract(kResumeRefreshThreshold),
          resumedAt: now,
        ),
        isTrue,
      );
    });

    test('refreshes after a long sleep — the case this exists for', () {
      expect(
        shouldRefreshOnResume(
          pausedAt: now.subtract(const Duration(hours: 3)),
          resumedAt: now,
        ),
        isTrue,
      );
    });
  });
}
