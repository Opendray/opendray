// URL extraction for the Flutter mobile session terminal.
//
// Mirrors the algorithm in
// app/web/src/components/sessions/url-extractor.ts — anchor on
// `https?://`, walk char-by-char, allow ONE intermediate `\n` as a
// CLI soft-wrap when the current line is long enough that wrapping
// was plausible.
//
// Why a state machine and not a plain regex: AI CLIs (claude-code,
// codex, gemini) hard-wrap long OAuth URLs at the terminal column
// width by emitting literal `\n` characters every ~55 chars. A
// `\bhttps?://[^\s]+` regex stops at the first `\n` and captures
// only the first wrapped segment. The user ends up tapping a
// truncated URL and OAuth fails. See the web extractor's docstring
// for the long version.

import 'dart:convert';

/// Strip ANSI CSI escape sequences (colour resets, cursor moves)
/// so the URL extractor isn't confused by them.
final _ansiCsiRegex = RegExp(r'\x1B\[[0-9;?]*[a-zA-Z]');
String stripAnsi(String text) => text.replaceAll(_ansiCsiRegex, '');

/// URL syntax characters per RFC 3986 — the body of an http(s) URL
/// is any of these. Used by the state machine to decide whether a
/// `\n` mid-URL is a CLI soft-wrap (continue) or a real terminator
/// (stop).
bool _isUrlBodyChar(String ch) {
  final c = ch.codeUnitAt(0);
  // A-Z, a-z, 0-9
  if (c >= 0x30 && c <= 0x39) return true;
  if (c >= 0x41 && c <= 0x5A) return true;
  if (c >= 0x61 && c <= 0x7A) return true;
  // ! $ & ' ( ) * + , - . / : ; = ? @ _ ~ % # [ ]
  const others = {
    0x21, 0x24, 0x26, 0x27, 0x28, 0x29, 0x2A, 0x2B, 0x2C, 0x2D,
    0x2E, 0x2F, 0x3A, 0x3B, 0x3D, 0x3F, 0x40, 0x5B, 0x5D, 0x5F,
    0x7E, 0x25, 0x23,
  };
  return others.contains(c);
}

// Same rationale + value as the web extractor: minimum length of the
// "current line" for a single `\n` to be treated as a soft-wrap.
const _softWrapMinLineLen = 40;

final _urlStartRegex = RegExp('https?://');

/// Extract http(s) URLs from text, including those the CLI has
/// hard-wrapped onto multiple lines. Returns URLs in the order
/// they were encountered, deduped.
List<String> extractUrls(String text) {
  final results = <String>{};
  for (final match in _urlStartRegex.allMatches(text)) {
    final start = match.start;
    var i = start + match.group(0)!.length;

    while (i < text.length) {
      final ch = text[i];
      final code = ch.codeUnitAt(0);

      // Hard terminators — never appear inside URLs.
      if (ch == ' ' ||
          ch == '\t' ||
          ch == '<' ||
          ch == '>' ||
          ch == '"' ||
          ch == "'") {
        break;
      }

      // Newlines — allow ONE as CLI soft-wrap when conditions match.
      if (ch == '\n' || ch == '\r') {
        var j = i + 1;
        var nlCount = ch == '\n' ? 1 : 0;
        while (j < text.length &&
            (text[j] == '\n' || text[j] == '\r')) {
          if (text[j] == '\n') nlCount++;
          j++;
        }
        if (nlCount >= 2) break; // paragraph break terminates
        if (j >= text.length) break; // trailing newline
        if (!_isUrlBodyChar(text[j])) break; // followed by non-URL

        // Heuristic: only treat as soft-wrap when the current line
        // is plausibly wrap-width (≥ 40 chars). Short prose lines
        // like "see\nhttps://..." stay separate.
        final prevNlIdx = text.lastIndexOf('\n', i - 1);
        final lineStart = prevNlIdx == -1 ? 0 : prevNlIdx + 1;
        final lineLen = i - lineStart;
        if (lineLen < _softWrapMinLineLen) break;

        i = j;
        continue;
      }

      // Other ASCII control chars terminate.
      if (code < 0x20) break;

      i++;
    }

    final raw = text
        .substring(start, i)
        .replaceAll(RegExp(r'[\r\n]+'), '');
    final cleaned = _trimTrailingPunctuation(raw);
    if (cleaned.isNotEmpty) results.add(cleaned);
  }
  return results.toList();
}

String _trimTrailingPunctuation(String url) {
  var end = url.length;
  while (end > 0) {
    final ch = url[end - 1];
    final isStripChar = ch == '.' ||
        ch == ',' ||
        ch == ';' ||
        ch == ':' ||
        ch == '!' ||
        ch == '?' ||
        ch == "'";
    final isUnpairedClose = (ch == ')' &&
            !url.substring(0, end - 1).contains('(')) ||
        (ch == ']' &&
            !url.substring(0, end - 1).contains('['));
    if (isStripChar || isUnpairedClose) {
      end -= 1;
    } else {
      break;
    }
  }
  return url.substring(0, end);
}

/// Decode UTF-8 bytes leniently (matches the terminal's `_decode`
/// behaviour — never throws on invalid sequences mid-stream).
String decodeForUrlScan(List<int> bytes) {
  return const Utf8Decoder(allowMalformed: true).convert(bytes);
}
