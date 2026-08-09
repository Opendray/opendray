package notes

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKindOf(t *testing.T) {
	tests := []struct {
		path string
		want Kind
	}{
		{"notes/a.md", KindMarkdown},
		{"notes/a.markdown", KindMarkdown},
		{"docs/index.html", KindHTML},
		{"docs/index.htm", KindHTML},
		// Extensions are matched case-insensitively: an export from a
		// Windows tool arriving as .HTML is the same document.
		{"docs/INDEX.HTML", KindHTML},
		{"notes/A.MD", KindMarkdown},
		// Not documents. The vault lists and writes documents only —
		// assets living alongside them are none of its business.
		{"assets/logo.png", KindUnknown},
		{"notes/a.txt", KindUnknown},
		{"noextension", KindUnknown},
		{"", KindUnknown},
		// A dotfile is an extension-less name, not an ".md" document.
		{".md", KindUnknown},
	}
	for _, tt := range tests {
		if got := KindOf(tt.path); got != tt.want {
			t.Errorf("KindOf(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

func TestTrimDocExt(t *testing.T) {
	tests := [][2]string{
		{"projects/app/README.md", "projects/app/README"},
		{"docs/guide.html", "docs/guide"},
		{"docs/guide.HTM", "docs/guide"},
		// An unknown extension is left intact — trimming it would make
		// `logo.png` collide with a document called `logo`.
		{"assets/logo.png", "assets/logo.png"},
		{"plain", "plain"},
	}
	for _, tt := range tests {
		if got := TrimDocExt(tt[0]); got != tt[1] {
			t.Errorf("TrimDocExt(%q) = %q, want %q", tt[0], got, tt[1])
		}
	}
}

// The listing is what the tree, the pickers and the mobile list all
// read, so HTML has to appear there or it does not exist as far as the
// product is concerned.
func TestList_IncludesHTMLAndSkipsNonDocuments(t *testing.T) {
	root := t.TempDir()
	for _, rel := range []string{
		"notes/guide.md", "docs/api.html", "docs/legacy.htm",
		"assets/logo.png", "notes/scratch.txt",
	} {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	v, err := New(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.List("")
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, n := range got {
		seen[n.Path] = true
	}
	for _, want := range []string{"notes/guide.md", "docs/api.html", "docs/legacy.htm"} {
		if !seen[want] {
			t.Errorf("%s missing from the listing", want)
		}
	}
	for _, unwanted := range []string{"assets/logo.png", "notes/scratch.txt"} {
		if seen[unwanted] {
			t.Errorf("%s should not be listed as a document", unwanted)
		}
	}
}

func TestWrite_AcceptsHTMLRejectsOthers(t *testing.T) {
	v, err := New(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Write("docs/api.html", "<h1>API</h1>"); err != nil {
		t.Fatalf("HTML write rejected: %v", err)
	}
	if _, err := v.Write("notes/a.md", "# a"); err != nil {
		t.Fatalf("markdown write rejected: %v", err)
	}
	if _, err := v.Write("assets/logo.png", "binary"); err == nil {
		t.Error("the vault must not accept arbitrary files through the API")
	}
}

// Wiki links are markdown's. An HTML file may be POINTED AT — that is
// free and useful — but its own <a href> links are not rewritten,
// because relative paths, anchors and assets make that a different job
// with different failure modes.
func TestBacklinks_HTMLIsALinkTargetButNotAScannedSource(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A markdown note linking to an HTML document.
	write("notes/index.md", "see [[docs/api]] for details\n")
	// An HTML document containing something that merely looks like a
	// wiki link. It must not be scanned.
	write("docs/api.html", "<p>[[notes/index]]</p>\n")

	v, err := New(root, Options{})
	if err != nil {
		t.Fatal(err)
	}

	toHTML, err := v.Backlinks(context.Background(), "docs/api.html")
	if err != nil {
		t.Fatal(err)
	}
	if len(toHTML) != 1 || toHTML[0].Path != "notes/index.md" {
		t.Fatalf("backlinks to the HTML doc = %v, want the markdown note", toHTML)
	}

	toMD, err := v.Backlinks(context.Background(), "notes/index.md")
	if err != nil {
		t.Fatal(err)
	}
	if len(toMD) != 0 {
		t.Fatalf("HTML was scanned for wiki links: %v", toMD)
	}
}

// Moving an HTML document must work — it is a document — and markdown
// notes pointing at it must follow.
func TestMove_HTMLDocumentAndItsInboundLinks(t *testing.T) {
	root := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("docs/api.html", "<h1>API</h1>")
	write("notes/index.md", "see [[docs/api]]\n")

	v, err := New(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := v.Move(context.Background(), "docs/api.html", "reference/api.html"); err != nil {
		t.Fatalf("moving an HTML document failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "reference", "api.html")); err != nil {
		t.Fatalf("document not at its new path: %v", err)
	}
	body, err := os.ReadFile(filepath.Join(root, "notes", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "[[reference/api]]"; string(body) != "see "+want+"\n" {
		t.Errorf("inbound link not rewritten: %q", string(body))
	}
}

// Creating `guide.html` from a markdown template used to produce a
// document that renders as literal `# Guide` in a browser. When the
// filename and the template disagree, the filename wins — the operator
// chose the extension; the template was probably just the default.
func TestNewFromTemplate_HTMLTargetGetsAnHTMLSkeleton(t *testing.T) {
	v, err := New(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := v.NewFromTemplate("docs/guide.html", "blank")
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.Read(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Body, "<!doctype html>") {
		t.Fatalf("HTML target did not get an HTML skeleton:\n%s", got.Body)
	}
	// Placeholders still render.
	if !strings.Contains(got.Body, "<title>Guide</title>") {
		t.Errorf("title placeholder not substituted:\n%s", got.Body)
	}
	if strings.Contains(got.Body, "{{") {
		t.Errorf("unsubstituted placeholder left behind:\n%s", got.Body)
	}
}

// A markdown target must be untouched by the above.
func TestNewFromTemplate_MarkdownTargetKeepsItsTemplate(t *testing.T) {
	v, err := New(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := v.NewFromTemplate("docs/guide.md", "blank")
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.Read(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Body, "# Guide") {
		t.Fatalf("markdown template was replaced:\n%s", got.Body)
	}
}

// A vault template that is itself HTML must be used verbatim — the
// skeleton is a fallback for a mismatch, not an override.
func TestNewFromTemplate_HTMLVaultTemplateWins(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, TemplateDir)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	body := "<!doctype html><html><body><main>{{title}}</main></body></html>\n"
	if err := os.WriteFile(filepath.Join(dir, "report.html"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	v, err := New(root, Options{})
	if err != nil {
		t.Fatal(err)
	}
	n, err := v.NewFromTemplate("docs/q3.html", "report")
	if err != nil {
		t.Fatal(err)
	}
	got, err := v.Read(n.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Body, "<main>Q3</main>") {
		t.Fatalf("vault HTML template not used:\n%s", got.Body)
	}
}
