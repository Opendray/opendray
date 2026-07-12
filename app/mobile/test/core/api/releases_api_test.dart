import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/releases_api.dart';

// Guards the "what's new" parsing that mirrors app/shared/src/lib/
// releases.ts. These pure functions must stay in lockstep with the web
// version so mobile shows the same highlights / unread badge behaviour.
void main() {
  group('normalizeReleaseVersion', () {
    test('strips leading v and build metadata', () {
      expect(normalizeReleaseVersion('v2.11.2'), '2.11.2');
      expect(normalizeReleaseVersion('2.11.2+47'), '2.11.2');
      expect(normalizeReleaseVersion('  V2.12.0  '), '2.12.0');
      expect(normalizeReleaseVersion(null), '');
      expect(normalizeReleaseVersion(''), '');
    });
  });

  group('formatReleaseTag', () {
    test('re-adds the v prefix, empty stays empty', () {
      expect(formatReleaseTag('2.11.2'), 'v2.11.2');
      expect(formatReleaseTag('v2.11.2'), 'v2.11.2');
      expect(formatReleaseTag(''), '');
    });
  });

  group('isReleaseUnread', () {
    test('unread when never marked and a latest exists', () {
      expect(isReleaseUnread('2.12.0', null), isTrue);
    });
    test('read when marked version matches latest', () {
      expect(isReleaseUnread('2.12.0', 'v2.12.0'), isFalse);
    });
    test('unread when marked version is older', () {
      expect(isReleaseUnread('2.12.0', '2.11.2'), isTrue);
    });
    test('no badge when there is no latest version', () {
      expect(isReleaseUnread('', '2.11.2'), isFalse);
      expect(isReleaseUnread(null, null), isFalse);
    });
  });

  group('extractHighlights', () {
    test('prefers the bold title in CHANGELOG-style bullets', () {
      const md = '''
### Added

- **Sidebar Updates drawer.** A new drawer pulling release highlights.
- **Grok logo fix.** Sessions list now shows the mark.
''';
      expect(
        extractHighlights(md),
        ['Sidebar Updates drawer', 'Grok logo fix'],
      );
    });

    test('falls back to the plain bullet text and strips markdown', () {
      const md = '- Fixed a `race` in the [worker](http://x) loop.';
      expect(extractHighlights(md), ['Fixed a race in the worker loop']);
    });

    test('ignores indented continuation lines and honours the limit', () {
      // Titles must be >= 4 chars — the parser drops shorter fragments,
      // matching releases.ts.
      const md = '''
- First item here
  continuation line under first
- Second item here
- Third item here
''';
      expect(
        extractHighlights(md, limit: 2),
        ['First item here', 'Second item here'],
      );
    });

    test('drops the Announce on X block and empty input', () {
      const md = '''
- Real highlight here
## Announce on X
- do not surface this tweet bullet
''';
      expect(extractHighlights(md), ['Real highlight here']);
      expect(extractHighlights('   '), isEmpty);
    });
  });

  group('extractChangelogSection', () {
    const changelog = '''
# Changelog

## [Unreleased]

## [v2.12.0] — 2026-07-09

### Added

- **Feature A.** Something new.

## [v2.11.2] — 2026-07-01

### Added

- **Old feature.** Should not appear.
''';

    test('returns only the requested version section', () {
      final section = extractChangelogSection(changelog, '2.12.0');
      expect(section.contains('Feature A'), isTrue);
      expect(section.contains('Old feature'), isFalse);
      expect(extractHighlights(section), ['Feature A']);
    });

    test('empty when the version is missing', () {
      expect(extractChangelogSection(changelog, '9.9.9'), '');
      expect(extractChangelogSection('', '2.12.0'), '');
    });
  });
}
