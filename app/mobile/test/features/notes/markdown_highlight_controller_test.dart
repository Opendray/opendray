import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

import 'package:opendray/features/notes/markdown_highlight_controller.dart';

// The highlighter must never change what the field CONTAINS — it only
// changes how it looks. Every case below therefore asserts the
// reassembled span text is byte-identical to the source, because a
// scanner that drops or duplicates a character would corrupt the note
// the moment it is saved. The colouring assertions come second.

/// Builds the span for [source] and returns it plus its flattened text.
Future<(TextSpan, String)> build(WidgetTester tester, String source) async {
  late TextSpan span;
  final controller = MarkdownHighlightController(text: source);
  await tester.pumpWidget(
    MaterialApp(
      home: Builder(
        builder: (context) {
          span = controller.buildTextSpan(
            context: context,
            withComposing: false,
            style: const TextStyle(fontSize: 13),
          );
          return const SizedBox();
        },
      ),
    ),
  );
  return (span, span.toPlainText());
}

/// Collects the leaf spans containing `needle`, so a test can ask
/// what style a specific run was given.
List<TextSpan> runsFor(TextSpan root, String needle) {
  final out = <TextSpan>[];
  void walk(InlineSpan s) {
    if (s is TextSpan) {
      if (s.text != null && s.text!.contains(needle)) out.add(s);
      for (final c in s.children ?? const <InlineSpan>[]) {
        walk(c);
      }
    }
  }

  walk(root);
  return out;
}

void main() {
  testWidgets('text survives highlighting byte for byte', (tester) async {
    const source = '''
---
type: feature
---

# Host power

Some **bold** and *italic* and `code` and [[wiki-link]].
See [docs](https://example.com) and #tag.

- item one
- item two with `inline`

> a quote

```dart
final x = **not bold**;
```

Trailing text with 中文 and emoji 🚀.
''';
    final (_, plain) = await build(tester, source);
    expect(plain, source);
  });

  testWidgets('empty and whitespace documents round-trip', (tester) async {
    for (final source in ['', '\n', '   ', '\n\n\n', 'a']) {
      final (_, plain) = await build(tester, source);
      expect(plain, source, reason: 'source was ${source.codeUnits}');
    }
  });

  testWidgets('headings, quotes and fences claim their whole line',
      (tester) async {
    final (span, _) = await build(tester, '# Title\n> quoted\nplain');
    expect(runsFor(span, '# Title').first.style?.fontWeight, FontWeight.w700);
    expect(runsFor(span, '> quoted').first.style?.fontStyle, FontStyle.italic);
    // A plain line gets no colour of its own.
    expect(runsFor(span, 'plain').first.style?.color, isNull);
  });

  testWidgets('markup inside a fenced block is not styled as markup',
      (tester) async {
    final (span, plain) = await build(
      tester,
      '```\n# not a heading\n**not bold**\n```\nafter',
    );
    expect(plain, '```\n# not a heading\n**not bold**\n```\nafter');
    // The fenced line is one run coloured as code, not split into
    // heading/bold pieces.
    final fenced = runsFor(span, '# not a heading');
    expect(fenced, hasLength(1));
    expect(fenced.first.style?.fontWeight, isNot(FontWeight.w700));
  });

  testWidgets('overlapping inline patterns do not double-style',
      (tester) async {
    // A bold run inside a code span belongs to the code span; the
    // scanner must take the leftmost match and skip what it covers.
    final (span, plain) = await build(tester, 'a `**b**` c');
    expect(plain, 'a `**b**` c');
    final code = runsFor(span, '`**b**`');
    expect(code, hasLength(1), reason: 'the code span should be one run');
    expect(code.first.style?.fontWeight, isNot(FontWeight.w700));
  });

  testWidgets('wiki-links are coloured', (tester) async {
    final (span, _) = await build(tester, 'see [[canvas]] please');
    final link = runsFor(span, '[[canvas]]');
    expect(link, hasLength(1));
    expect(link.first.style?.color, isNotNull);
  });

  testWidgets('a long document does not lose content', (tester) async {
    final source = List.generate(
      400,
      (i) => '## Section $i\n\nBody **$i** with `code` and [[ref-$i]].\n',
    ).join('\n');
    final (_, plain) = await build(tester, source);
    expect(plain, source);
  });
}
