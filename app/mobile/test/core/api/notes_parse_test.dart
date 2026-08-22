import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/core/api/notes_api.dart';

// Parsing for the two vault endpoints the phone previously had no
// client for at all. Shapes come from internal/notes/links.go.
void main() {
  group('Backlink.fromJson', () {
    test('reads path, title and the context lines', () {
      final b = Backlink.fromJson(const {
        'path': 'projects/opendray/spec.md',
        'title': 'Spec',
        'modified': '2026-08-10T09:00:00Z',
        'lines': ['see [[canvas]] for the shape', 'and [[canvas|it]] again'],
      });
      expect(b.path, 'projects/opendray/spec.md');
      expect(b.title, 'Spec');
      expect(b.lines, hasLength(2));
    });

    test('survives a response with no lines at all', () {
      // `lines` is omitempty-adjacent on the Go side; a missing key must
      // not take the whole pane down.
      final b = Backlink.fromJson(const {'path': 'a.md'});
      expect(b.lines, isEmpty);
      expect(b.title, '');
    });

    test('drops non-string entries rather than throwing', () {
      final b = Backlink.fromJson(const {
        'path': 'a.md',
        'lines': ['ok', 42, null],
      });
      expect(b.lines, ['ok']);
    });
  });

  group('TagCount.fromJson', () {
    test('reads the tag, its count and the notes carrying it', () {
      final t = TagCount.fromJson(const {
        'tag': 'infra',
        'count': 2,
        'notes': ['a.md', 'b.md'],
      });
      expect(t.tag, 'infra');
      expect(t.count, 2);
      expect(t.notes, ['a.md', 'b.md']);
    });

    test('defaults notes to empty — the field is omitempty', () {
      final t = TagCount.fromJson(const {'tag': 'solo', 'count': 1});
      expect(t.notes, isEmpty);
    });

    test('tolerates a count arriving as a double', () {
      final t = TagCount.fromJson(const {'tag': 'x', 'count': 3.0});
      expect(t.count, 3);
    });
  });
}
