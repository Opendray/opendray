package memconflict

import (
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/memory"
)

func TestParseConflicts_Empty(t *testing.T) {
	got, err := ParseConflicts("")
	if err != nil {
		t.Fatalf("empty should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty findings, got %d", len(got))
	}
}

func TestParseConflicts_CleanJSON(t *testing.T) {
	raw := `{"conflicts":[{"layer_a":"fact","ref_a":"mem_1","layer_b":"plan","ref_b":"plan-doc","evidence":"X says A, Y says B","severity":"high"}]}`
	got, err := ParseConflicts(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 1 || got[0].LayerA != LayerFact || got[0].RefB != "plan-doc" || got[0].Severity != SeverityHigh {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestParseConflicts_FencedAndPreamble(t *testing.T) {
	raw := "Here are the conflicts:\n```json\n{\"conflicts\":[]}\n```\nthanks"
	got, err := ParseConflicts(raw)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestNormaliseOrder_SwappedFindingsCollapse(t *testing.T) {
	la, ra, lb, rb := normaliseOrder(LayerPlan, "plan-doc", LayerFact, "mem_1")
	la2, ra2, lb2, rb2 := normaliseOrder(LayerFact, "mem_1", LayerPlan, "plan-doc")
	if la != la2 || ra != ra2 || lb != lb2 || rb != rb2 {
		t.Errorf("swap not normalised: (%s,%s,%s,%s) vs (%s,%s,%s,%s)",
			la, ra, lb, rb, la2, ra2, lb2, rb2)
	}
}

func TestPickTopByHits(t *testing.T) {
	now := time.Now()
	mems := []memory.Memory{
		{ID: "a", HitCount: 1, CreatedAt: now},
		{ID: "b", HitCount: 10, CreatedAt: now},
		{ID: "c", HitCount: 5, CreatedAt: now},
		{ID: "d", HitCount: 10, CreatedAt: now.Add(time.Hour)}, // tie-break newer
	}
	top := pickTopByHits(mems, 3)
	if len(top) != 3 {
		t.Fatalf("expected 3, got %d", len(top))
	}
	if top[0].ID != "d" || top[1].ID != "b" || top[2].ID != "c" {
		t.Errorf("wrong order: %v %v %v", top[0].ID, top[1].ID, top[2].ID)
	}
}

func TestOneLine(t *testing.T) {
	got := oneLine("hello\nworld   ", 100)
	if got != "hello world" {
		t.Errorf("got %q", got)
	}
	got = oneLine("abcdefghij", 5)
	if got != "abcde…" {
		t.Errorf("got %q", got)
	}
}

func TestBundleEmpty(t *testing.T) {
	b := detectionBundle{}
	if !b.empty() {
		t.Error("expected empty")
	}
	b.plan = "x"
	if b.empty() {
		t.Error("expected non-empty when plan set")
	}
}

func TestBundleRender(t *testing.T) {
	b := detectionBundle{
		cwd:  "/p",
		plan: "build app",
		facts: []memory.Memory{
			{ID: "mem_1", Text: "x", HitCount: 3},
		},
	}
	got := b.render()
	for _, want := range []string{"# Project context for /p", "## Project plan", "build app", "mem_1", "hits=3"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestNewID(t *testing.T) {
	a, b := newID(), newID()
	if a == b {
		t.Error("expected different ids")
	}
	if !strings.HasPrefix(a, "mc_") {
		t.Errorf("expected mc_ prefix, got %s", a)
	}
}

func TestValidLayer(t *testing.T) {
	for _, l := range []Layer{LayerFact, LayerPlan, LayerGoal, LayerJournal} {
		if !validLayer(l) {
			t.Errorf("%s should be valid", l)
		}
	}
	if validLayer(Layer("nonsense")) {
		t.Error("nonsense should not be valid")
	}
}
