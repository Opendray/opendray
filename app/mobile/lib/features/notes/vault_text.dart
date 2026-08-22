// The vault's pure text layer: what a path means, what a document
// contains, and how its wiki-links resolve. No Flutter, no I/O — so it
// is testable on its own and can be shared by the vault browser, the
// editor and the preview without dragging a webview into any of them.
//
// Ported from the web so the two clients agree on the same documents:
//
//   docKindOf        ← docKind            app/shared/src/lib/notes.ts
//   sanitizeNotePath ← sanitizeNotePath   app/shared/src/lib/notes.ts
//   extractOutline   ← extractOutline     app/shared/src/lib/outline.ts
//   slugify          ← slugify            app/shared/src/lib/outline.ts
//   extractTags      ← extractTags        NoteEditor.tsx
//   resolveWikiLink  ← resolveLink        NoteEditor.tsx
//   detectWikiLinkContext ← same          app/shared/src/lib/caret.ts
//
// Two places deliberately diverge from the web; both are called out at
// the function that does it.

/// What kind of document a vault path holds. Mirrors notes.KindOf in
/// internal/notes/doc.go and docKind() in app/shared/src/lib/notes.ts.
enum DocKind { markdown, html, unknown }

DocKind docKindOf(String path) {
  final base = path.split('/').last;
  final dot = base.lastIndexOf('.');
  // A leading dot is a dotfile, not an extension.
  if (dot <= 0) return DocKind.unknown;
  switch (base.substring(dot + 1).toLowerCase()) {
    case 'md':
    case 'markdown':
      return DocKind.markdown;
    case 'html':
    case 'htm':
      return DocKind.html;
    default:
      return DocKind.unknown;
  }
}

/// Scheme used for `[[wiki-link]]` anchors in the rendered preview.
///
/// The preview runs with JavaScript off — a vault document can arrive by
/// `git pull`, so it is not the operator's own writing by default. That
/// rules out a click handler, so links are ordinary anchors on a scheme
/// the webview will never resolve, and the navigation itself is what
/// gets intercepted.
const kWikiLinkScheme = 'opendray-wiki';

// ---------------------------------------------------------------- paths

final _leadingSlashes = RegExp('^/+');
final _leadingDots = RegExp(r'^\.+');
final _whitespaceRun = RegExp(r'\s+');
// Everything a filename may keep. Unlike the web's ASCII-only class this
// admits any Unicode letter or number — see sanitizeNotePath.
final _unsafePathChars = RegExp(r'[^\p{Letter}\p{Number}_.\- ]', unicode: true);
final _docExtension = RegExp(r'\.(md|markdown|html?)$', caseSensitive: false);

/// Clean a user-typed note path, one segment at a time.
///
/// Slashes are meaningful — they are how a document gets filed under
/// `features/` — so they survive, while empty and dot-only segments are
/// dropped and `..` can never reach the request.
///
/// Two things this fixes relative to what the phone did before:
///
///   * `.md` was appended unconditionally, so `guide.html` became
///     `guide.html.md` and an HTML document could not be created at all.
///   * Nothing was cleaned beyond a leading slash.
///
/// DIVERGES FROM WEB, deliberately: the web replaces every character
/// outside `[A-Za-z0-9_.- ]` with a dash, which turns `笔记.md` into
/// `--.md`. The phone accepts CJK filenames today and mirroring the web
/// would be a regression, so the class is Unicode-aware here.
String sanitizeNotePath(String input, {String defaultExt = '.md'}) {
  final segments = input
      .trim()
      .split('/')
      .map((s) => s
          .trim()
          .replaceAll(_unsafePathChars, '-')
          .replaceAll(_whitespaceRun, '-')
          .replaceFirst(_leadingDots, ''))
      .where((s) => s.isNotEmpty)
      .toList();
  if (segments.isEmpty) return 'untitled$defaultExt';
  final last = segments.last;
  // Only append an extension when the name doesn't already carry a
  // recognised one.
  if (docKindOf(last) == DocKind.unknown) {
    segments[segments.length - 1] = '$last$defaultExt';
  }
  return segments.join('/');
}

// -------------------------------------------------------------- outline

/// One heading in a document's outline.
class OutlineHeading {
  const OutlineHeading({
    required this.level,
    required this.text,
    required this.slug,
    required this.lineIndex,
    required this.charOffset,
  });

  /// 1..6.
  final int level;
  final String text;

  /// Fragment-safe id, deduped within the document. Matches the id the
  /// markdown renderer emits, so it doubles as an anchor target.
  final String slug;

  /// 0-based line the heading sits on.
  final int lineIndex;

  /// Index of the heading line's first character in the body. The phone
  /// navigates by moving the caret, so this — not [lineIndex] — is what
  /// actually drives the jump.
  final int charOffset;
}

final _headingRe = RegExp(r'^(#{1,6})\s+(.+?)\s*#*\s*$');
final _fenceRe = RegExp('^(```|~~~)');
final _nonSlugChars = RegExp(r'[^\p{Letter}\p{Number}]+', unicode: true);
final _edgeDashes = RegExp(r'^-+|-+$');

/// Lowercase, collapse non-alphanumeric runs to `-`, trim the edges.
///
/// `\p{Letter}` rather than `\w`: an ASCII-only rule collapses a Chinese
/// heading to nothing, so every one of them would share the slug
/// "section" and the outline could not tell them apart.
String slugify(String text) {
  final out = text
      .toLowerCase()
      .replaceAll(_nonSlugChars, '-')
      .replaceAll(_edgeDashes, '');
  return out.isEmpty ? 'section' : out;
}

/// Walk the body for ATX headings, skipping fenced code regions.
List<OutlineHeading> extractOutline(String body) {
  final out = <OutlineHeading>[];
  final seen = <String, int>{};
  var inFence = false;
  var offset = 0;
  final lines = body.split('\n');
  for (var i = 0; i < lines.length; i++) {
    final line = lines[i];
    final lineStart = offset;
    offset += line.length + 1; // +1 for the '\n' split consumed
    final trimmed = line.trim();
    if (_fenceRe.hasMatch(trimmed)) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    final m = _headingRe.firstMatch(line);
    if (m == null) continue;
    final text = m.group(2)!.trim();
    if (text.isEmpty) continue;
    final base = slugify(text);
    final n = seen[base] ?? 0;
    seen[base] = n + 1;
    out.add(OutlineHeading(
      level: m.group(1)!.length,
      text: text,
      slug: n > 0 ? '$base-$n' : base,
      lineIndex: i,
      charOffset: lineStart,
    ));
  }
  return out;
}

// ----------------------------------------------------------------- tags

final _fencedBlock = RegExp(r'```[\s\S]*?```');
final _inlineCode = RegExp('`[^`]*`');
final _tagRe = RegExp('(?:^|[^A-Za-z0-9_-])#([A-Za-z][A-Za-z0-9_/-]{0,40})');
final _frontmatterInlineTags = RegExp(r'tags:\s*\[(.*?)\]');
final _frontmatterBlockTags = RegExp(r'tags:\s*\n((?:\s*-\s*.+\n?)+)');
final _quotes = RegExp(r"""^['"]|['"]$""");

/// Pull `#tag` mentions and frontmatter `tags:` entries out of a body.
/// Mirrors the backend scanner in internal/notes/links.go so the chips
/// shown on a document match what the tag index will file it under.
List<String> extractTags(String body) {
  final tags = <String>{};

  if (body.startsWith('---')) {
    final end = body.substring(3).indexOf('---');
    if (end >= 0) {
      final header = body.substring(3, 3 + end);
      final inline = _frontmatterInlineTags.firstMatch(header);
      if (inline != null) {
        for (final raw in inline.group(1)!.split(',')) {
          final cleaned = raw.trim().replaceAll(_quotes, '');
          if (cleaned.isNotEmpty) tags.add(cleaned);
        }
      }
      final block = _frontmatterBlockTags.firstMatch(header);
      if (block != null) {
        for (final line in block.group(1)!.split('\n')) {
          final cleaned = line
              .replaceFirst(RegExp(r'^\s*-\s*'), '')
              .trim()
              .replaceAll(_quotes, '');
          if (cleaned.isNotEmpty) tags.add(cleaned);
        }
      }
    }
  }

  // Code is prose about code, not tags in it.
  final stripped =
      body.replaceAll(_fencedBlock, ' ').replaceAll(_inlineCode, ' ');
  for (final m in _tagRe.allMatches(stripped)) {
    tags.add(m.group(1)!.replaceFirst(RegExp(r'/$'), ''));
  }
  return tags.toList();
}

// ------------------------------------------------------------ wiki links

/// Where the caret sits inside an unclosed `[[`.
class WikiLinkContext {
  const WikiLinkContext({required this.query, required this.openIdx});

  /// Text typed since the opening brackets.
  final String query;

  /// Index of the first `[` of the pair.
  final int openIdx;
}

/// Inspect the text before [caretPos] and return the active `[[…` query,
/// or null when the caret is not inside an open wiki-link. A `]` or a
/// newline in between counts as a close.
WikiLinkContext? detectWikiLinkContext(String text, int caretPos) {
  // Hard cap the walk — a runaway buffer shouldn't burn cycles scanning
  // hundreds of KB backwards on every keystroke.
  const searchLimit = 256;
  final start = caretPos - searchLimit < 0 ? 0 : caretPos - searchLimit;
  for (var i = caretPos - 1; i >= start; i--) {
    final ch = text[i];
    if (ch == '\n' || ch == ']') return null;
    if (ch == '[' && i > 0 && text[i - 1] == '[') {
      return WikiLinkContext(
        query: text.substring(i + 1, caretPos),
        openIdx: i - 1,
      );
    }
  }
  return null;
}

/// Turn a wiki-link target into the vault path it points at.
///
/// A target with a slash is a full vault path. A bare name matches any
/// document's basename, case-insensitively. Anything unmatched resolves
/// next to the current note — the link is a document that doesn't exist
/// yet, which is a normal state in a wiki, not an error.
String resolveWikiLink(
  String target, {
  required Iterable<String> allPaths,
  String currentDir = '',
}) {
  final cleaned = target
      .trim()
      .replaceFirst(_leadingSlashes, '')
      .replaceFirst(_docExtension, '');
  if (cleaned.contains('/')) return '$cleaned.md';
  final needle = cleaned.toLowerCase();
  for (final path in allPaths) {
    final base =
        path.split('/').last.replaceFirst(_docExtension, '').toLowerCase();
    if (base == needle) return path;
  }
  return currentDir.isEmpty ? '$cleaned.md' : '$currentDir/$cleaned.md';
}

final _wikiLinkRe = RegExp(r'\[\[([^\]|\n]+)(?:\|([^\]\n]+))?\]\]');

/// Rewrite `[[Target]]` / `[[Target|Alias]]` into anchors on
/// [kWikiLinkScheme], leaving everything else untouched.
///
/// Runs on the markdown SOURCE, before conversion, because the converter
/// passes inline HTML through. Code — fenced and inline — is skipped: a
/// document explaining the syntax has to be able to show it, and the
/// converter escapes HTML inside code, so a rewritten link there would
/// render as a visible `<a href=…>` tag.
String linkifyWikiLinks(String markdown) {
  final lines = markdown.split('\n');
  var inFence = false;
  for (var i = 0; i < lines.length; i++) {
    if (_fenceRe.hasMatch(lines[i].trim())) {
      inFence = !inFence;
      continue;
    }
    if (inFence) continue;
    lines[i] = _linkifyOutsideInlineCode(lines[i]);
  }
  return lines.join('\n');
}

String _linkifyOutsideInlineCode(String line) {
  final buf = StringBuffer();
  var cursor = 0;
  for (final code in _inlineCode.allMatches(line)) {
    buf
      ..write(_linkifySpan(line.substring(cursor, code.start)))
      ..write(code.group(0)); // verbatim
    cursor = code.end;
  }
  buf.write(_linkifySpan(line.substring(cursor)));
  return buf.toString();
}

String _linkifySpan(String span) {
  return span.replaceAllMapped(_wikiLinkRe, (m) {
    final target = m.group(1)!.trim();
    final label = (m.group(2) ?? m.group(1)!).trim();
    final href =
        '$kWikiLinkScheme://open?target=${Uri.encodeQueryComponent(target)}';
    return '<a href="$href">${escapeHtml(label)}</a>';
  });
}

/// Escape text for insertion into generated HTML. The label comes from a
/// document that may have been pulled from a git remote, so it is not
/// trusted markup.
String escapeHtml(String s) => s
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;');

// ----------------------------------------------------------- daily note

/// Vault path of [day]'s daily note.
String dailyNotePath(DateTime day) =>
    'daily/${_pad4(day.year)}-${_pad2(day.month)}-${_pad2(day.day)}.md';

/// Starting body for a daily note. Deliberately NOT translated: this is
/// document content, and a note's headings must not depend on which
/// language the app happened to be in when it was created. Kept
/// character-for-character in step with handleNewDaily in
/// app/web/src/pages/Notes.tsx so one date yields one document, whether
/// it was started on the phone or in the browser.
String dailyNoteBody(DateTime day, {required String longDate}) {
  final date = '${_pad4(day.year)}-${_pad2(day.month)}-${_pad2(day.day)}';
  return '---\ndate: $date\ntype: daily\n---\n\n# $longDate\n\n'
      "## What I'm doing\n\n## What I learned\n\n## TODO\n\n";
}

String _pad2(int n) => n.toString().padLeft(2, '0');
String _pad4(int n) => n.toString().padLeft(4, '0');

/// Read the target back out of a [kWikiLinkScheme] URL, or null if the
/// URL isn't one of ours.
String? wikiLinkTarget(Uri uri) {
  if (uri.scheme != kWikiLinkScheme) return null;
  final target = uri.queryParameters['target'];
  if (target == null || target.isEmpty) return null;
  return target;
}
