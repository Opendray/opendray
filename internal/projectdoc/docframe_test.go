package projectdoc

import (
	"strings"
	"testing"
)

// doc_read used to frame a document with a markdown H1 of its own slug.
// Agents are told to read a page and write the full body back, so that
// frame got written INTO the page — and the next read framed the result
// again. Two live pages already carry it: kb_conventions starts with
// "# kb_conventions", and kb_infrastructure has that plus its real title
// underneath.
func TestStripDocReadWrapper(t *testing.T) {
	tests := []struct {
		name string
		slug string
		in   string
		want string
	}{
		{
			name: "slug-titled H1 with metadata line",
			slug: "kb_projects",
			in:   "# kb_projects\n\n_last updated by operator_\n\nReal body\nsecond line\n",
			want: "Real body\nsecond line\n",
		},
		{
			name: "slug H1 alone — the LLM drops the metadata line but keeps the title",
			slug: "kb_conventions",
			in:   "# kb_conventions\n\n> quoted intro\n",
			want: "> quoted intro\n",
		},
		{
			name: "the comment form doc_read now emits",
			slug: "kb_projects",
			in:   "<!-- doc_read: kb_projects · last updated by operator -->\n\nBody\n",
			want: "Body\n",
		},
		{
			name: "project doc's 'Project <kind>' framing",
			slug: "goal",
			in:   "# Project goal\n\n_last updated by agent_\n\nShip it.\n",
			want: "Ship it.\n",
		},
		// The page's OWN title must survive — this is the case that makes a
		// blunt "strip the first H1" rule unsafe.
		{
			name: "real title is not the slug — untouched",
			slug: "kb_infrastructure",
			in:   "# Home Lab Infrastructure Reference\n\nHosts...\n",
			want: "# Home Lab Infrastructure Reference\n\nHosts...\n",
		},
		{
			name: "wrapper stripped but the real title beneath it survives",
			slug: "kb_infrastructure",
			in:   "# kb_infrastructure\n\n# Home Lab Infrastructure Reference\n\nHosts...\n",
			want: "# Home Lab Infrastructure Reference\n\nHosts...\n",
		},
		{
			name: "section headings are never wrappers",
			slug: "kb_lessons",
			in:   "## opendray-v2 — PR merge\n\n- thing\n",
			want: "## opendray-v2 — PR merge\n\n- thing\n",
		},
		{
			name: "a matching line later in the body is not a wrapper",
			slug: "kb_projects",
			in:   "Intro\n\n# kb_projects\n\nmentioned mid-document\n",
			want: "Intro\n\n# kb_projects\n\nmentioned mid-document\n",
		},
		{
			name: "no wrapper at all",
			slug: "kb_projects",
			in:   "Reception page for the injection-layer.\n",
			want: "Reception page for the injection-layer.\n",
		},
		{
			name: "repeated wrappers from several round-trips all come off",
			slug: "kb_projects",
			in:   "# kb_projects\n\n# kb_projects\n\n_last updated by agent_\n\nBody\n",
			want: "Body\n",
		},
		{
			name: "empty input",
			slug: "kb_projects",
			in:   "",
			want: "",
		},
		{
			name: "wrapper only, nothing beneath",
			slug: "kb_projects",
			in:   "# kb_projects\n\n_last updated by operator_\n",
			want: "",
		},
	}

	for _, tt := range tests {
		if got := StripDocFrame(tt.slug, tt.in); got != tt.want {
			t.Errorf("%s:\n StripDocFrame(%q, %q)\n  = %q\n want %q",
				tt.name, tt.slug, tt.in, got, tt.want)
		}
	}
}

// Whatever doc_read emits must survive a round-trip: strip(read(body)) has
// to give the body back, or the two sides have drifted again.
func TestDocReadWrapperRoundTrip(t *testing.T) {
	for _, body := range []string{
		"# Home Lab Infrastructure Reference\n\nHosts and rules.\n",
		"Reception page.\n\n## Mathlandia\n- DB: x\n",
		"## opendray — reusable modules\n\n- thing\n",
	} {
		framed := FrameDocForRead("kb_infrastructure", "operator", body)
		if got := StripDocFrame("kb_infrastructure", framed); got != body {
			t.Errorf("round-trip lost content:\n framed = %q\n got    = %q\n want   = %q",
				framed, got, body)
		}
	}
}

// The frame must not be a markdown heading — that is what made it look
// like page content worth keeping.
func TestFrameDocForRead_IsNotAHeading(t *testing.T) {
	out := FrameDocForRead("kb_projects", "operator", "Body\n")
	first := strings.SplitN(out, "\n", 2)[0]
	if strings.HasPrefix(first, "#") {
		t.Errorf("frame line is a markdown heading, which is how it ended up in page bodies: %q", first)
	}
	for _, want := range []string{"kb_projects", "operator"} {
		if !strings.Contains(out, want) {
			t.Errorf("frame lost %q: %q", want, first)
		}
	}
}

// --- draft signature carry-forward ---------------------------------------
//
// The clients hide the drafter's kb-sig marker while editing, so an operator
// save arrives without one. Losing it disabled the sweep's dirty check and
// the page was redrafted every cycle — which is what made hand edits look
// like they kept getting reverted.

func TestCarryDraftSig_RestoresAStrippedMarker(t *testing.T) {
	prev := "# Infra\n- old line\n\n<!-- kb-sig:abc123 -->\n"
	next := "# Infra\n- edited by hand\n"
	got := CarryDraftSig(prev, next)
	if !strings.Contains(got, "<!-- kb-sig:abc123 -->") {
		t.Errorf("signature not carried across:\n%q", got)
	}
	if !strings.Contains(got, "- edited by hand") {
		t.Errorf("carry lost the operator's edit:\n%q", got)
	}
	if strings.Contains(got, "- old line") {
		t.Errorf("carry resurrected the old body:\n%q", got)
	}
}

func TestCarryDraftSig_LeavesABodyThatHasItsOwn(t *testing.T) {
	prev := "old\n\n<!-- kb-sig:aaa -->\n"
	next := "new\n\n<!-- kb-sig:bbb -->\n"
	if got := CarryDraftSig(prev, next); got != next {
		t.Errorf("rewrote a body that already carried a signature:\n%q", got)
	}
	if strings.Contains(CarryDraftSig(prev, next), "aaa") {
		t.Error("the incoming signature must win")
	}
}

func TestCarryDraftSig_NoopWithoutAPredecessorSig(t *testing.T) {
	for _, prev := range []string{"", "plain body\n", "truncated <!-- kb-sig:abc"} {
		next := "new body\n"
		if got := CarryDraftSig(prev, next); got != next {
			t.Errorf("prev %q: got %q, want unchanged", prev, got)
		}
	}
}

// A page that round-trips through the editor repeatedly must not accumulate
// markers, and must keep exactly the newest one.
func TestCarryDraftSig_DoesNotAccumulate(t *testing.T) {
	body := "v1\n\n<!-- kb-sig:one -->\n"
	for i, edit := range []string{"v2\n", "v3\n", "v4\n"} {
		body = CarryDraftSig(body, edit)
		if n := strings.Count(body, draftSigMarker); n != 1 {
			t.Fatalf("edit %d: %d markers, want 1:\n%q", i, n, body)
		}
	}
	if !strings.Contains(body, "<!-- kb-sig:one -->") || !strings.Contains(body, "v4") {
		t.Errorf("final body wrong:\n%q", body)
	}
}
