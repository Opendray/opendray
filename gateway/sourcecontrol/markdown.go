package sourcecontrol

// Markdown preview rendering for the Source Control panel.
//
// When MultiDiff encounters a *.md / *.markdown file AND
// cfg.MarkdownPreview is true, it reads the post-change file content
// (working tree for unstaged, HEAD for staged, baseline-vs-HEAD for
// baseline mode) and renders it to sanitised HTML. The panel shows
// both the diff and the rendered preview side-by-side so users can
// eyeball documentation changes without leaving the panel — borrowed
// directly from yepanywhere's UX.

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

// Markdown engine is cheap to build but reused anyway — goldmark does
// not expose an "allocate once" parser, and the extensions list is
// static, so sharing the instance saves a few KB per render.
var mdEngine = goldmark.New(
	goldmark.WithExtensions(
		extension.GFM, // tables, strikethrough, autolinks, task lists
	),
	goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		// Intentionally no html.WithUnsafe() — inline <script> / raw
		// HTML in user markdown would defeat the point of the
		// bluemonday pass below.
	),
)

// mdPolicy is the HTML sanitisation whitelist. UGCPolicy is the right
// default for "untrusted user content rendered alongside trusted UI":
// keeps links, images, tables, code blocks, headings, lists; strips
// <script>, event handlers, javascript: URLs, form elements.
var mdPolicy = bluemonday.UGCPolicy()

// IsMarkdownPath reports whether a file path looks like a Markdown
// document we should render a preview for.
func IsMarkdownPath(p string) bool {
	ext := strings.ToLower(filepath.Ext(p))
	return ext == ".md" || ext == ".markdown" || ext == ".mkd"
}

// RenderMarkdown converts source bytes to sanitised HTML. Returns an
// empty string for empty input so callers can drop the Preview field
// entirely when there's nothing to show.
func RenderMarkdown(src []byte) (string, error) {
	if len(bytes.TrimSpace(src)) == 0 {
		return "", nil
	}
	var buf bytes.Buffer
	if err := mdEngine.Convert(src, &buf); err != nil {
		return "", fmt.Errorf("markdown: render: %w", err)
	}
	clean := mdPolicy.Sanitize(buf.String())
	return clean, nil
}

// attachMarkdownPreviews augments every .md / .markdown FileDiff with
// a rendered preview of the *post-change* content. The source is read
// from the working tree — for unstaged diffs that's the on-disk file;
// for staged / baseline modes the working tree and the index usually
// agree, and when they don't the preview still reflects something
// useful (the user's current view of the file). File sizes are capped
// so a 50MB README can't stall the handler.
//
// Errors are soft: a file that can't be read is left without a
// preview rather than failing the entire diff.
func attachMarkdownPreviews(ctx context.Context, repoPath string, files []FileDiff, maxBytes int64) []FileDiff {
	if maxBytes <= 0 {
		maxBytes = 1 << 20 // 1 MiB default cap
	}
	for i := range files {
		if ctx.Err() != nil {
			return files
		}
		f := &files[i]
		if !IsMarkdownPath(f.Path) || f.Status == "deleted" {
			continue
		}
		abs := filepath.Join(repoPath, f.Path)
		info, err := os.Stat(abs)
		if err != nil || info.IsDir() {
			continue
		}
		if info.Size() > maxBytes {
			continue // silent skip — oversize files aren't worth rendering
		}
		src, err := os.ReadFile(abs)
		if err != nil {
			continue
		}
		html, err := RenderMarkdown(src)
		if err != nil || html == "" {
			continue
		}
		f.PreviewHTML = html
	}
	return files
}
