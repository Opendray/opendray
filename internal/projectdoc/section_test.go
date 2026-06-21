package projectdoc

import (
	"reflect"
	"testing"
)

func TestSliceSection(t *testing.T) {
	doc := "preamble line\n" +
		"## Alpha\n" +
		"alpha body\n" +
		"### Alpha One\n" +
		"alpha one body\n" +
		"## Beta\n" +
		"beta body\n"

	tests := []struct {
		name    string
		heading string
		want    string
		wantOK  bool
	}{
		{
			name:    "top section stops at next same-level heading and includes deeper subsections",
			heading: "Alpha",
			want:    "## Alpha\nalpha body\n### Alpha One\nalpha one body\n",
			wantOK:  true,
		},
		{
			name:    "subsection stops at next same-or-higher heading",
			heading: "Alpha One",
			want:    "### Alpha One\nalpha one body\n",
			wantOK:  true,
		},
		{
			name:    "last section runs to end of document",
			heading: "Beta",
			want:    "## Beta\nbeta body\n",
			wantOK:  true,
		},
		{
			name:    "case-insensitive + whitespace-collapsed match",
			heading: "  alpha   ONE ",
			want:    "### Alpha One\nalpha one body\n",
			wantOK:  true,
		},
		{
			name:    "substring fallback matches a numbered heading",
			heading: "Alpha", // exact; covered above — keep explicit substring case below
			want:    "## Alpha\nalpha body\n### Alpha One\nalpha one body\n",
			wantOK:  true,
		},
		{
			name:    "no match returns false",
			heading: "Gamma",
			want:    "",
			wantOK:  false,
		},
		{
			name:    "empty heading returns false",
			heading: "   ",
			want:    "",
			wantOK:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := SliceSection(doc, tt.heading)
			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (got=%q)", ok, tt.wantOK, got)
			}
			if got != tt.want {
				t.Errorf("section =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}

func TestSliceSection_SubstringFallbackOnNumberedHeading(t *testing.T) {
	doc := "## 6. The Unified Spawn Profile\nbody six\n## 7. Driving A Session\nbody seven\n"
	// Requested text is a substring of the numbered heading.
	got, ok := SliceSection(doc, "unified spawn profile")
	if !ok {
		t.Fatalf("expected substring match")
	}
	if got != "## 6. The Unified Spawn Profile\nbody six\n" {
		t.Errorf("got %q", got)
	}
}

func TestSliceSection_IgnoresHeadingsInsideCodeFence(t *testing.T) {
	doc := "## Real\n" +
		"text\n" +
		"```bash\n" +
		"# not a heading (shell comment)\n" +
		"## also not a heading\n" +
		"```\n" +
		"more text\n" +
		"## Next\n" +
		"next body\n"
	got, ok := SliceSection(doc, "Real")
	if !ok {
		t.Fatal("expected match for Real")
	}
	want := "## Real\ntext\n```bash\n# not a heading (shell comment)\n## also not a heading\n```\nmore text\n"
	if got != want {
		t.Errorf("fenced content leaked or truncated:\n got=%q\nwant=%q", got, want)
	}
	// A heading-looking line inside the fence must NOT be selectable.
	if _, ok := SliceSection(doc, "also not a heading"); ok {
		t.Error("matched a heading-looking line inside a code fence")
	}
}

func TestSliceSection_FenceTypesDoNotCrossClose(t *testing.T) {
	// A ~~~ line inside a ``` block must NOT close it (CommonMark).
	doc := "## Real\n" +
		"```\n" +
		"~~~\n" +
		"## still in fence\n" +
		"```\n" +
		"## After\n" +
		"after body\n"
	got, ok := SliceSection(doc, "Real")
	if !ok {
		t.Fatal("Real should match")
	}
	want := "## Real\n```\n~~~\n## still in fence\n```\n"
	if got != want {
		t.Errorf("got %q\nwant %q", got, want)
	}
	if _, ok := SliceSection(doc, "still in fence"); ok {
		t.Error("a heading inside the fence was selectable")
	}
}

func TestListSectionHeadings(t *testing.T) {
	doc := "preamble\n" +
		"# Title\n" +
		"## One\n" +
		"```\n## fenced not counted\n```\n" +
		"### Two\n" +
		"body\n"
	got := ListSectionHeadings(doc)
	want := []string{"Title", "One", "Two"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("headings = %v, want %v", got, want)
	}
}
