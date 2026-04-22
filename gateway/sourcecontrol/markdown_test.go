package sourcecontrol

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsMarkdownPath(t *testing.T) {
	yes := []string{"README.md", "docs/guide.markdown", "notes.MD", "a.mkd"}
	no := []string{"main.go", "README", "script.sh", "image.png"}
	for _, p := range yes {
		if !IsMarkdownPath(p) {
			t.Errorf("IsMarkdownPath(%q) should be true", p)
		}
	}
	for _, p := range no {
		if IsMarkdownPath(p) {
			t.Errorf("IsMarkdownPath(%q) should be false", p)
		}
	}
}

func TestRenderMarkdown_BasicConversion(t *testing.T) {
	html, err := RenderMarkdown([]byte("# Title\n\nParagraph **bold**."))
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	if !strings.Contains(html, "<h1") || !strings.Contains(html, "<strong>bold</strong>") {
		t.Errorf("expected heading + strong in %q", html)
	}
}

func TestRenderMarkdown_StripsScript(t *testing.T) {
	// Raw HTML <script> is disabled at goldmark level (no Unsafe
	// option), but defense-in-depth: bluemonday strips it too.
	html, err := RenderMarkdown([]byte("Hi <script>alert(1)</script>"))
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	if strings.Contains(strings.ToLower(html), "<script") {
		t.Errorf("script tag must be sanitised; got %q", html)
	}
}

func TestRenderMarkdown_StripsJavaScriptURL(t *testing.T) {
	html, err := RenderMarkdown([]byte("[click](javascript:alert(1))"))
	if err != nil {
		t.Fatalf("RenderMarkdown: %v", err)
	}
	// bluemonday either drops the href or strips the scheme.
	lower := strings.ToLower(html)
	if strings.Contains(lower, "javascript:") {
		t.Errorf("javascript: URL must be blocked; got %q", html)
	}
}

func TestRenderMarkdown_EmptyInputReturnsEmpty(t *testing.T) {
	for _, s := range []string{"", "   ", "\n\n"} {
		html, err := RenderMarkdown([]byte(s))
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		if html != "" {
			t.Errorf("whitespace-only input should yield empty, got %q", html)
		}
	}
}

func TestAttachMarkdownPreviews_PopulatesPreviewForMarkdown(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git binary not available")
	}
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hello\n\nBody."), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	files := []FileDiff{
		{Path: "README.md", Status: "modified"},
		{Path: "main.go", Status: "modified"},
	}
	got := attachMarkdownPreviews(context.Background(), dir, files, 0)
	if !strings.Contains(got[0].PreviewHTML, "<h1") {
		t.Errorf("README.md preview missing heading; got %q", got[0].PreviewHTML)
	}
	if got[1].PreviewHTML != "" {
		t.Errorf(".go file should NOT have preview; got %q", got[1].PreviewHTML)
	}
}

func TestAttachMarkdownPreviews_SkipsDeleted(t *testing.T) {
	dir := t.TempDir()
	// No file on disk; deleted status should skip the read attempt.
	files := []FileDiff{{Path: "README.md", Status: "deleted"}}
	got := attachMarkdownPreviews(context.Background(), dir, files, 0)
	if got[0].PreviewHTML != "" {
		t.Errorf("deleted file shouldn't produce preview; got %q", got[0].PreviewHTML)
	}
}

func TestAttachMarkdownPreviews_RespectsSizeCap(t *testing.T) {
	dir := t.TempDir()
	big := strings.Repeat("# line\n", 200_000) // ~1.4 MB of markdown
	if err := os.WriteFile(filepath.Join(dir, "big.md"), []byte(big), 0o644); err != nil {
		t.Fatal(err)
	}
	files := []FileDiff{{Path: "big.md", Status: "modified"}}
	got := attachMarkdownPreviews(context.Background(), dir, files, 1024) // 1 KiB cap
	if got[0].PreviewHTML != "" {
		t.Errorf("over-cap file should be skipped; got preview of len %d", len(got[0].PreviewHTML))
	}
}
