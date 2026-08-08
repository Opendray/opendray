// Rendered view for a vault document. Until now the phone could only
// ever show a note's SOURCE — there was no preview of any kind — so
// reading a document meant reading its markup. Adding HTML support
// without a renderer would have made that worse, not better: a page of
// tags is less readable than a page of markdown.
//
// Both kinds render through ONE webview. Markdown is converted to HTML
// first rather than drawn with native widgets, because two rendering
// engines in one screen means two typographies, two link behaviours and
// two sets of bugs — and the HTML one is not optional.
//
// SECURITY. A vault document can arrive by `git pull` from a remote, so
// it is not the operator's own writing by default. Two rules:
//
//   1. The document is loaded from an in-memory string (`loadData`),
//      never fetched from opendray's own origin. There is no endpoint
//      that serves it as text/html — that would run its scripts inside
//      the app's session.
//   2. JavaScript is OFF unless the operator turns it on for that one
//      document. Exported documentation is static markup and renders
//      the same either way; anything that genuinely needs to run code
//      becomes a decision someone made.
//
// Mirrors app/web/src/components/notes/HtmlPreview.tsx — keep the two
// postures in step.

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter_inappwebview/flutter_inappwebview.dart';
import 'package:markdown/markdown.dart' as md;
import 'package:opendray/core/i18n/strings.g.dart';
import 'package:shared_preferences/shared_preferences.dart';

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

const _scriptsKeyPrefix = 'vault.html.scripts:';

class DocPreview extends StatefulWidget {
  const DocPreview({
    required this.path,
    required this.body,
    super.key,
  });

  final String path;
  final String body;

  @override
  State<DocPreview> createState() => _DocPreviewState();
}

class _DocPreviewState extends State<DocPreview> {
  bool _scripts = false;
  bool _loadedPref = false;
  InAppWebViewController? _web;

  DocKind get _kind => docKindOf(widget.path);

  @override
  void initState() {
    super.initState();
    unawaited(_loadPref());
  }

  @override
  void didUpdateWidget(DocPreview old) {
    super.didUpdateWidget(old);
    if (old.path != widget.path) {
      // The opt-in belongs to the path. Carrying a stale `true` to the
      // next document would silently extend trust the operator granted
      // to a different file.
      _scripts = false;
      _loadedPref = false;
      unawaited(_loadPref());
    } else if (old.body != widget.body) {
      unawaited(_render());
    }
  }

  Future<void> _loadPref() async {
    var on = false;
    try {
      final prefs = await SharedPreferences.getInstance();
      on = prefs.getBool('$_scriptsKeyPrefix${widget.path}') ?? false;
    } on Object {
      // Unavailable prefs cost the memory of the choice, not the choice.
    }
    if (!mounted) return;
    setState(() {
      _scripts = on;
      _loadedPref = true;
    });
  }

  Future<void> _toggleScripts() async {
    final next = !_scripts;
    setState(() => _scripts = next);
    try {
      final prefs = await SharedPreferences.getInstance();
      if (next) {
        await prefs.setBool('$_scriptsKeyPrefix${widget.path}', true);
      } else {
        await prefs.remove('$_scriptsKeyPrefix${widget.path}');
      }
    } on Object {
      // Same as above.
    }
    // javaScriptEnabled is read when the webview is built, so the
    // toggle rebuilds it via the ValueKey below rather than trying to
    // flip the setting on a live instance.
  }

  Future<void> _render() async {
    final ctl = _web;
    if (ctl == null) return;
    await ctl.loadData(data: _document(), mimeType: 'text/html');
  }

  String _document() {
    if (_kind == DocKind.html) return _withBaseTarget(widget.body);
    return _markdownDocument(widget.body);
  }

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    if (!_loadedPref) {
      return const Center(child: CircularProgressIndicator());
    }
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        // The scripts control is only meaningful for HTML: converted
        // markdown is generated here and contains no scripts to run.
        if (_kind == DocKind.html)
          Padding(
            padding: const EdgeInsets.fromLTRB(12, 8, 8, 4),
            child: Row(
              children: [
                Icon(
                  _scripts ? Icons.warning_amber : Icons.verified_user_outlined,
                  size: 15,
                  color: _scripts
                      ? Colors.amber
                      : theme.colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 6),
                Expanded(
                  child: Text(
                    _scripts
                        ? t.notesPage.html.scriptsOn
                        : t.notesPage.html.scriptsOff,
                    style: TextStyle(
                      fontSize: 12,
                      color: theme.colorScheme.onSurfaceVariant,
                    ),
                  ),
                ),
                TextButton(
                  onPressed: _toggleScripts,
                  child: Text(
                    _scripts
                        ? t.notesPage.html.disableScripts
                        : t.notesPage.html.enableScripts,
                    style: const TextStyle(fontSize: 12),
                  ),
                ),
              ],
            ),
          ),
        Expanded(
          child: InAppWebView(
            // Rebuilding on the scripts flag is deliberate: the setting
            // is applied at creation, and a live webview would keep the
            // old policy.
            key: ValueKey('${widget.path}:$_scripts'),
            initialSettings: InAppWebViewSettings(
              javaScriptEnabled: _scripts,
              // No document should be able to start a navigation that
              // leaves the viewer stranded — there is no address bar.
              javaScriptCanOpenWindowsAutomatically: false,
              transparentBackground: true,
              supportZoom: true,
              useWideViewPort: true,
              loadWithOverviewMode: true,
            ),
            onWebViewCreated: (ctl) {
              _web = ctl;
              unawaited(
                ctl.loadData(data: _document(), mimeType: 'text/html'),
              );
            },
          ),
        ),
      ],
    );
  }
}

/// Makes in-document links open outside the frame rather than
/// navigating a viewer that has no way back.
String _withBaseTarget(String html) {
  const base = '<base target="_blank">';
  final head = RegExp('<head[^>]*>', caseSensitive: false);
  final m = head.firstMatch(html);
  if (m == null) return base + html;
  return html.replaceRange(m.end, m.end, base);
}

/// Wraps rendered markdown in a document that follows the phone's
/// text size and renders legibly without any external resources — the
/// webview has no network access to a stylesheet and should not want
/// one.
String _markdownDocument(String source) {
  final body = md.markdownToHtml(
    source,
    extensionSet: md.ExtensionSet.gitHubWeb,
    // The vault's [[wiki links]] are not markdown; leaving them as
    // literal text is honest. Resolving them would need the note list,
    // which this widget deliberately does not take.
  );
  return '''
<!doctype html>
<html><head><meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<base target="_blank">
<style>
  :root { color-scheme: light dark; }
  body {
    font: 15px/1.6 -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
    margin: 12px; word-wrap: break-word;
  }
  pre { overflow-x: auto; padding: 10px; border-radius: 6px;
        background: rgba(127,127,127,.14); }
  code { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-size: .9em; }
  pre code { background: none; padding: 0; }
  :not(pre) > code { background: rgba(127,127,127,.14); padding: .1em .3em; border-radius: 3px; }
  table { border-collapse: collapse; display: block; overflow-x: auto; }
  th, td { border: 1px solid rgba(127,127,127,.35); padding: 5px 9px; }
  img { max-width: 100%; height: auto; }
  blockquote { margin-left: 0; padding-left: 12px;
               border-left: 3px solid rgba(127,127,127,.4); }
  h1, h2, h3 { line-height: 1.25; }
</style>
</head><body>
$body
</body></html>''';
}

/// Exposed for the editor's "copy as HTML" affordances and for tests.
String renderMarkdownDocument(String source) => _markdownDocument(source);
