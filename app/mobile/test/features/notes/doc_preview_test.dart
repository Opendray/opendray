import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/notes/doc_preview.dart';
import 'package:opendray/features/notes/vault_text.dart' show kWikiLinkScheme;

void main() {
  group('docKindOf', () {
    test('classifies by extension, case-insensitively', () {
      expect(docKindOf('a/b/notes.md'), DocKind.markdown);
      expect(docKindOf('notes.MARKDOWN'), DocKind.markdown);
      expect(docKindOf('guide.html'), DocKind.html);
      expect(docKindOf('guide.HTM'), DocKind.html);
      expect(docKindOf('archive.tar.gz'), DocKind.unknown);
      expect(docKindOf('README'), DocKind.unknown);
    });

    test('treats a bare dotfile as not a document', () {
      // `.md` is a hidden file, not a markdown doc with an empty name.
      expect(docKindOf('.md'), DocKind.unknown);
      expect(docKindOf('dir/.html'), DocKind.unknown);
    });
  });

  group('retargetLinks', () {
    test('leaves in-document fragment links alone', () {
      // The bug this test exists for: a table-of-contents entry used to
      // open a blank tab instead of scrolling.
      const html = '<a href="#install">Install</a>';
      expect(retargetLinks(html), html);
    });

    test('opens links that leave the document in a new tab', () {
      expect(
        retargetLinks('<a href="https://example.com/">x</a>'),
        '<a href="https://example.com/" target="_blank" '
        'rel="noopener noreferrer">x</a>',
      );
    });

    test('retargets a relative link — it also leaves the document', () {
      expect(
        retargetLinks("<a href='other.html#top'>x</a>"),
        "<a href='other.html#top' target=\"_blank\" "
        'rel="noopener noreferrer">x</a>',
      );
    });

    test('respects an explicit target the author already chose', () {
      const html = '<a href="https://example.com/" target="_self">x</a>';
      expect(retargetLinks(html), html);
    });

    test('leaves an anchor with no href alone', () {
      const html = '<a name="section">x</a>';
      expect(retargetLinks(html), html);
    });

    test('does not match tags that merely start with "a"', () {
      const html = '<abbr title="x">y</abbr><article>z</article>';
      expect(retargetLinks(html), html);
    });

    test('does not strand attributes after a self-closing slash', () {
      final out = retargetLinks('<a href="https://example.com/"/>');
      expect(out, isNot(contains('/ target')));
      expect(out, contains('target="_blank"'));
    });
  });

  group('renderMarkdownDocument', () {
    test('renders markdown and keeps footnote links in-document', () {
      final out = renderMarkdownDocument('Text[^1]\n\n[^1]: A note.\n');
      // gitHubWeb renders footnotes as fragment links; a blanket
      // <base target="_blank"> would have sent them to a blank page.
      expect(out, contains('href="#fn-1"'));
      expect(out, isNot(contains('<base target="_blank">')));
    });

    test('retargets external links inside rendered markdown', () {
      final out = renderMarkdownDocument('[x](https://example.com/)');
      expect(out, contains('target="_blank"'));
    });

    test('leaves wiki links as literal text by default', () {
      // Where the caller has nowhere to send the tap, a dead link is
      // worse than plain text.
      final out = renderMarkdownDocument('See [[projects/foo]].');
      expect(out, contains('[[projects/foo]]'));
    });

    test('renders wiki links as anchors when asked', () {
      final out = renderMarkdownDocument(
        'See [[projects/foo]].',
        wikiLinks: true,
      );
      expect(out, contains('href="$kWikiLinkScheme://open?'));
      expect(out, isNot(contains('[[projects/foo]]')));
    });

    test('does not retarget a wiki-link anchor to a new window', () {
      // target="_blank" on an intercepted scheme asks the webview for a
      // popup it is configured to refuse.
      final out = renderMarkdownDocument('[[foo]]', wikiLinks: true);
      expect(out, isNot(contains('target="_blank"')));
    });
  });
}
