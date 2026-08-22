import 'package:flutter_test/flutter_test.dart';
import 'package:opendray/features/notes/vault_text.dart';

// The vault's text layer is ported from app/shared/src/lib/{notes,outline,
// caret}.ts and NoteEditor.tsx. These tests pin the behaviours the phone
// shares with the web so the two can't drift apart silently — the pattern
// that produced the mobile-parity gaps in the first place.
void main() {
  group('slugify', () {
    test('lowercases and joins on non-alphanumerics', () {
      expect(slugify('Getting Started'), 'getting-started');
      expect(slugify('  Trim  Me  '), 'trim-me');
      expect(slugify('a/b:c'), 'a-b-c');
    });

    test(r'keeps CJK — \p{Letter} is not \w', () {
      // A heading written in Chinese must still get a usable slug. An
      // ASCII-only rule collapses the whole thing to "section", which
      // makes every Chinese heading in a document share one id.
      expect(slugify('安装步骤'), '安装步骤');
      expect(slugify('第 1 章 · 概览'), '第-1-章-概览');
    });

    test('falls back to "section" when nothing survives', () {
      expect(slugify('---'), 'section');
      expect(slugify(''), 'section');
    });
  });

  group('extractOutline', () {
    test('picks ATX headings with level, text and line index', () {
      const body = '# Title\n\nintro\n\n## Section A\n\n### Deep\n';
      final out = extractOutline(body);
      expect(out.map((h) => h.text), ['Title', 'Section A', 'Deep']);
      expect(out.map((h) => h.level), [1, 2, 3]);
      expect(out.map((h) => h.lineIndex), [0, 4, 6]);
    });

    test('skips headings inside fenced code', () {
      const body = '# Real\n\n```md\n# Not a heading\n```\n\n## Also real\n';
      final out = extractOutline(body);
      expect(out.map((h) => h.text), ['Real', 'Also real']);
    });

    test('dedupes repeated slugs with a numeric suffix', () {
      const body = '## Notes\n\n## Notes\n\n## Notes\n';
      expect(
        extractOutline(body).map((h) => h.slug),
        ['notes', 'notes-1', 'notes-2'],
      );
    });

    test('strips closing hashes and ignores empty headings', () {
      const body = '## Closed ##\n\n###\n';
      final out = extractOutline(body);
      expect(out.length, 1);
      expect(out.single.text, 'Closed');
    });

    test('charOffset points at the start of the heading line', () {
      // The phone jumps by moving the caret, so the offset — not the
      // line index — is what actually drives navigation.
      const body = '# One\n\nbody\n\n## Two\n';
      final out = extractOutline(body);
      expect(body.substring(out[0].charOffset).startsWith('# One'), isTrue);
      expect(body.substring(out[1].charOffset).startsWith('## Two'), isTrue);
    });
  });

  group('extractTags', () {
    test('reads inline frontmatter arrays', () {
      const body = '---\ntags: [alpha, "beta", \'gamma\']\n---\n\n# Doc\n';
      expect(extractTags(body), containsAll(['alpha', 'beta', 'gamma']));
    });

    test('reads block frontmatter lists', () {
      const body = '---\ntags:\n  - one\n  - two\n---\n\nbody\n';
      expect(extractTags(body), containsAll(['one', 'two']));
    });

    test('reads #tags from the body', () {
      const body = 'see #alpha and #beta/nested here';
      expect(extractTags(body), containsAll(['alpha', 'beta/nested']));
    });

    test('ignores tags inside code', () {
      const body = 'text\n\n```\n#notatag\n```\n\nand `#alsonot` here';
      expect(extractTags(body), isEmpty);
    });

    test('does not treat a markdown heading as a tag', () {
      // `# Heading` is a hash followed by a space — the tag pattern
      // requires a letter straight after the hash.
      expect(extractTags('# Heading\n'), isEmpty);
    });
  });

  group('sanitizeNotePath', () {
    test('keeps slashes so folders can be created', () {
      expect(sanitizeNotePath('features/canvas'), 'features/canvas.md');
    });

    test('leaves a recognised extension alone', () {
      // The bug this replaces: the phone appended .md unconditionally,
      // so an HTML document could not be created at all.
      expect(sanitizeNotePath('guide.html'), 'guide.html');
      expect(sanitizeNotePath('guide.htm'), 'guide.htm');
      expect(sanitizeNotePath('notes.md'), 'notes.md');
      expect(sanitizeNotePath('README.markdown'), 'README.markdown');
    });

    test('appends the default extension to an unknown one', () {
      expect(sanitizeNotePath('report.2026'), 'report.2026.md');
      expect(sanitizeNotePath('plain'), 'plain.md');
    });

    test('drops traversal and empty segments', () {
      expect(sanitizeNotePath('../../etc/passwd'), 'etc/passwd.md');
      expect(sanitizeNotePath('//a///b//'), 'a/b.md');
      expect(sanitizeNotePath('.'), 'untitled.md');
      expect(sanitizeNotePath('   '), 'untitled.md');
    });

    test('collapses whitespace into dashes', () {
      expect(sanitizeNotePath('my great note'), 'my-great-note.md');
    });

    test('keeps CJK filenames intact', () {
      // The web sanitiser replaces every non-ASCII character with a
      // dash, which turns 笔记.md into --.md. Mirroring that here would
      // regress the phone, which accepts Chinese filenames today.
      expect(sanitizeNotePath('笔记/会议纪要'), '笔记/会议纪要.md');
    });
  });

  group('detectWikiLinkContext', () {
    test('returns the query when the caret sits inside an open [[', () {
      const text = 'see [[can';
      final ctx = detectWikiLinkContext(text, text.length);
      expect(ctx, isNotNull);
      expect(ctx!.query, 'can');
      expect(ctx.openIdx, 4);
    });

    test('is null once the link is closed', () {
      const text = 'see [[canvas]] ';
      expect(detectWikiLinkContext(text, text.length), isNull);
    });

    test('is null across a newline', () {
      const text = 'see [[\nstill';
      expect(detectWikiLinkContext(text, text.length), isNull);
    });

    test('matches an empty query right after [[', () {
      const text = 'x [[';
      final ctx = detectWikiLinkContext(text, text.length);
      expect(ctx?.query, '');
      expect(ctx?.openIdx, 2);
    });

    test('a single bracket is not a wiki link', () {
      const text = 'array[idx';
      expect(detectWikiLinkContext(text, text.length), isNull);
    });
  });

  group('resolveWikiLink', () {
    const all = [
      'projects/opendray/spec.md',
      'daily/2026-08-10.md',
      'personal/Canvas.md',
    ];

    test('matches a bare basename case-insensitively', () {
      expect(
        resolveWikiLink('canvas', allPaths: all),
        'personal/Canvas.md',
      );
    });

    test('treats a target containing a slash as a full path', () {
      expect(
        resolveWikiLink('projects/opendray/spec', allPaths: all),
        'projects/opendray/spec.md',
      );
    });

    test('falls back to a sibling of the current note when unknown', () {
      expect(
        resolveWikiLink('brand new', allPaths: all, currentDir: 'daily'),
        'daily/brand new.md',
      );
      expect(resolveWikiLink('orphan', allPaths: all), 'orphan.md');
    });

    test('tolerates an explicit .md and leading slashes', () {
      expect(resolveWikiLink('/canvas.md', allPaths: all), 'personal/Canvas.md');
    });
  });

  group('linkifyWikiLinks', () {
    test('rewrites [[Target]] into an anchor on the wiki scheme', () {
      final out = linkifyWikiLinks('see [[Canvas]] here');
      expect(out, contains('href="$kWikiLinkScheme://open?target=Canvas"'));
      expect(out, contains('>Canvas</a>'));
    });

    test('uses the alias as the visible label', () {
      final out = linkifyWikiLinks('[[projects/spec|the spec]]');
      expect(out, contains('>the spec</a>'));
      expect(out, contains('target=projects%2Fspec'));
    });

    test('escapes the label so a document cannot inject markup', () {
      final out = linkifyWikiLinks('[[x|<script>alert(1)</script>]]');
      expect(out, isNot(contains('<script>')));
      expect(out, contains('&lt;script&gt;'));
    });

    test('leaves wiki syntax inside fenced code alone', () {
      // A document explaining the syntax must still show it. Rewriting
      // it here would render the raw <a …> tag as visible text, since
      // the markdown converter escapes HTML inside code.
      const body = 'real [[One]]\n\n```\nliteral [[Two]]\n```\n';
      final out = linkifyWikiLinks(body);
      expect(out, contains('>One</a>'));
      expect(out, contains('literal [[Two]]'));
    });

    test('leaves wiki syntax inside inline code alone', () {
      final out = linkifyWikiLinks('type `[[Name]]` to link');
      expect(out, contains('`[[Name]]`'));
    });

    test('leaves plain text untouched', () {
      expect(linkifyWikiLinks('nothing to see'), 'nothing to see');
    });
  });

  group('daily note', () {
    test('files the note under daily/ with a zero-padded date', () {
      expect(dailyNotePath(DateTime(2026, 8, 9)), 'daily/2026-08-09.md');
      expect(dailyNotePath(DateTime(2026, 12, 31)), 'daily/2026-12-31.md');
    });

    test('body carries the frontmatter the web template does', () {
      final body = dailyNoteBody(
        DateTime(2026, 8, 10),
        longDate: 'Monday, August 10, 2026',
      );
      expect(body, startsWith('---\ndate: 2026-08-10\ntype: daily\n---\n'));
      expect(body, contains('# Monday, August 10, 2026'));
      expect(body, contains("## What I'm doing"));
      expect(body, contains('## What I learned'));
      expect(body, contains('## TODO'));
    });

    test('frontmatter marks it as a daily note for the tag index', () {
      final body = dailyNoteBody(DateTime(2026, 1, 2), longDate: 'x');
      expect(extractOutline(body).first.text, 'x');
    });
  });
}
