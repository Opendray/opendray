import 'package:flutter/material.dart';

/// A [TextEditingController] that colours markdown as you type.
///
/// The web file viewer has always highlighted what it shows, so raw
/// markdown in the Vault — the one place people read and write it — was
/// the last flat grey surface. The web fix layers a highlighted backdrop
/// under a transparent textarea, because the DOM offers nothing better.
/// Flutter does: [buildTextSpan] is the documented hook for exactly
/// this, and because the framework lays out the very spans it renders,
/// the caret cannot drift from the glyphs. No overlay, no metric
/// matching, no scroll syncing.
///
/// Deliberately hand-rolled rather than pulling in a highlighting
/// package: a vault holds markdown and nothing else, the grammar worth
/// colouring is a dozen patterns, and a general-purpose engine would be
/// a dependency plus a theme to reconcile for no extra fidelity.
class MarkdownHighlightController extends TextEditingController {
  MarkdownHighlightController({super.text});

  // buildTextSpan runs on every rebuild of the field, not only on edits
  // — focus changes, keyboard insets and theme rebuilds all land here.
  // Re-scanning a long note each time is wasted work, so the last
  // result is kept and reused whenever neither the text nor the
  // resolved style has changed.
  String? _cachedText;
  TextStyle? _cachedStyle;
  TextSpan? _cachedSpan;

  @override
  TextSpan buildTextSpan({
    required BuildContext context,
    required bool withComposing,
    TextStyle? style,
  }) {
    final base = style ?? const TextStyle();
    if (_cachedSpan != null && _cachedText == text && _cachedStyle == base) {
      return _cachedSpan!;
    }

    final palette = _Palette.of(Theme.of(context).colorScheme);
    final spans = <TextSpan>[];

    // Line-oriented constructs (headings, quotes, fences, rules) claim
    // the whole line; anything else is split by the inline pass.
    var inFence = false;
    final lines = text.split('\n');
    for (var i = 0; i < lines.length; i++) {
      final line = lines[i];
      final trimmed = line.trimLeft();

      if (trimmed.startsWith('```') || trimmed.startsWith('~~~')) {
        inFence = !inFence;
        spans.add(TextSpan(text: line, style: base.copyWith(color: palette.code)));
      } else if (inFence) {
        spans.add(TextSpan(text: line, style: base.copyWith(color: palette.code)));
      } else if (_heading.hasMatch(line)) {
        spans.add(TextSpan(
          text: line,
          style: base.copyWith(
            color: palette.heading,
            fontWeight: FontWeight.w700,
          ),
        ));
      } else if (trimmed.startsWith('>')) {
        spans.add(TextSpan(
          text: line,
          style: base.copyWith(
            color: palette.quote,
            fontStyle: FontStyle.italic,
          ),
        ));
      } else if (_rule.hasMatch(trimmed) ||
          (i == 0 && _frontmatterFence.hasMatch(line))) {
        spans.add(TextSpan(text: line, style: base.copyWith(color: palette.punct)));
      } else {
        _appendInline(spans, line, base, palette);
      }

      if (i != lines.length - 1) spans.add(const TextSpan(text: '\n'));
    }

    final built = TextSpan(style: base, children: spans);
    _cachedText = text;
    _cachedStyle = base;
    _cachedSpan = built;
    return built;
  }

  /// Splits one line into styled runs.
  ///
  /// Each rule is scanned across the line ONCE via allMatches, and the
  /// results are then merged left to right, dropping any match that
  /// overlaps one already taken. The earlier, per-position approach
  /// re-ran every regex against a fresh substring at each cursor step,
  /// which is quadratic on a long line — fine for a paragraph, not for
  /// a doc someone actually keeps in the vault.
  void _appendInline(
    List<TextSpan> out,
    String line,
    TextStyle base,
    _Palette palette,
  ) {
    if (line.isEmpty) {
      out.add(const TextSpan(text: ''));
      return;
    }

    var cursor = 0;
    // A leading list marker colours independently of the item's text.
    final bullet = _listMarker.matchAsPrefix(line);
    if (bullet != null) {
      out.add(TextSpan(
        text: line.substring(0, bullet.end),
        style: base.copyWith(color: palette.punct),
      ));
      cursor = bullet.end;
    }

    final hits = <_InlineHit>[];
    for (final rule in _inlineRules) {
      for (final m in rule.pattern.allMatches(line, cursor)) {
        hits.add(_InlineHit(m.start, m.end, rule));
      }
    }
    // Leftmost wins; ties go to the rule declared first (code before
    // emphasis, so markup inside a code span is left alone).
    hits.sort((a, b) => a.start != b.start
        ? a.start.compareTo(b.start)
        : _inlineRules.indexOf(a.rule).compareTo(_inlineRules.indexOf(b.rule)));

    for (final hit in hits) {
      if (hit.start < cursor) continue; // overlaps something already taken
      if (hit.start > cursor) {
        out.add(TextSpan(text: line.substring(cursor, hit.start)));
      }
      out.add(TextSpan(
        text: line.substring(hit.start, hit.end),
        style: hit.rule.style(base, palette),
      ));
      cursor = hit.end;
    }
    if (cursor < line.length) {
      out.add(TextSpan(text: line.substring(cursor)));
    }
  }
}

class _InlineHit {
  _InlineHit(this.start, this.end, this.rule);
  final int start;
  final int end;
  final _InlineRule rule;
}

class _InlineRule {
  const _InlineRule(this.pattern, this.style);
  final RegExp pattern;
  final TextStyle Function(TextStyle base, _Palette p) style;
}

final _heading = RegExp(r'^#{1,6}\s');
final _rule = RegExp(r'^(-{3,}|\*{3,}|_{3,})$');
final _frontmatterFence = RegExp(r'^---\s*$');
final _listMarker = RegExp(r'^\s*([-*+]|\d+[.)])\s+');

// Order matters only for ties at the same offset; the scan always
// prefers the leftmost match.
final _inlineRules = <_InlineRule>[
  // `code` — first, so markup inside a code span is left alone.
  _InlineRule(RegExp(r'`[^`\n]+`'), (b, p) => b.copyWith(color: p.code)),
  // [[wiki-link]] — opendray's own, and the reason a stock markdown
  // grammar wouldn't have been enough on its own.
  _InlineRule(RegExp(r'\[\[[^\]\n]+\]\]'), (b, p) => b.copyWith(color: p.link)),
  // [text](target)
  _InlineRule(
    RegExp(r'\[[^\]\n]*\]\([^)\n]*\)'),
    (b, p) => b.copyWith(color: p.link),
  ),
  // **bold** / __bold__
  _InlineRule(
    RegExp(r'(\*\*|__)(?=\S)(.+?)(?<=\S)\1'),
    (b, p) => b.copyWith(color: p.strong, fontWeight: FontWeight.w700),
  ),
  // *italic* / _italic_
  _InlineRule(
    RegExp(r'(\*|_)(?=\S)([^*_\n]+?)(?<=\S)\1'),
    (b, p) => b.copyWith(color: p.em, fontStyle: FontStyle.italic),
  ),
  // #tag
  _InlineRule(
    RegExp(r'(?<![\w/])#[A-Za-z][\w/-]*'),
    (b, p) => b.copyWith(color: p.tag),
  ),
];

/// Colours derived from the active scheme so the editor tracks light
/// and dark without a second palette to maintain.
class _Palette {
  const _Palette({
    required this.heading,
    required this.strong,
    required this.em,
    required this.code,
    required this.link,
    required this.quote,
    required this.tag,
    required this.punct,
  });

  factory _Palette.of(ColorScheme s) => _Palette(
        heading: s.primary,
        strong: s.onSurface,
        em: s.onSurface.withValues(alpha: 0.85),
        code: s.tertiary,
        link: s.secondary,
        quote: s.onSurfaceVariant,
        tag: s.tertiary,
        punct: s.outline,
      );

  final Color heading;
  final Color strong;
  final Color em;
  final Color code;
  final Color link;
  final Color quote;
  final Color tag;
  final Color punct;
}
