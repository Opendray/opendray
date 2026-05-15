package app

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/opendray/opendray-v2/internal/channel"
	"github.com/opendray/opendray-v2/internal/session"
)

// fakeSessionOps satisfies sessionOps without spinning up a real
// PTY-backed Manager. Records every Stop/Start call so tests can
// assert what the handlers actually invoked. Configurable errors
// per-method drive the error-path assertions.
type fakeSessionOps struct {
	sessions []session.Session
	listErr  error

	stopCalls []string
	stopErr   error

	startCalls []string
	startResp  session.Session
	startErr   error
}

func (f *fakeSessionOps) List(_ context.Context) ([]session.Session, error) {
	return f.sessions, f.listErr
}
func (f *fakeSessionOps) Stop(_ context.Context, id string) error {
	f.stopCalls = append(f.stopCalls, id)
	return f.stopErr
}
func (f *fakeSessionOps) Start(_ context.Context, id string) (session.Session, error) {
	f.startCalls = append(f.startCalls, id)
	return f.startResp, f.startErr
}

// cardText extracts the markdown body from a /list card so the
// recency / sort assertions read like the old plain-text ones.
func cardText(t *testing.T, c *channel.Card) string {
	t.Helper()
	if c == nil {
		return ""
	}
	for _, el := range c.Elements {
		if md, ok := el.(channel.CardMarkdown); ok {
			return md.Content
		}
	}
	return ""
}

// cardButtonValues returns every button's callback value in row-
// major order. Lets tests assert that each session got the right
// `cmd:/end <full_id>` or `cmd:/resume <full_id>` payload, even
// when the visible label is abbreviated.
func cardButtonValues(c *channel.Card) []string {
	if c == nil {
		return nil
	}
	var out []string
	for _, el := range c.Elements {
		acts, ok := el.(channel.CardActions)
		if !ok {
			continue
		}
		for _, row := range acts.Buttons {
			for _, b := range row {
				out = append(out, b.Value)
			}
		}
	}
	return out
}

func TestListSessionsCardHandler_EmptyState(t *testing.T) {
	h := listSessionsCardHandler(&fakeSessionOps{})
	got, err := h(context.Background(), channel.CommandContext{})
	if err != nil {
		t.Fatal(err)
	}
	if cardText(t, got) != "No sessions yet." {
		t.Errorf("body = %q", cardText(t, got))
	}
	if vals := cardButtonValues(got); len(vals) != 0 {
		t.Errorf("empty state should emit no buttons, got %v", vals)
	}
}

func TestListSessionsCardHandler_LiveFirstThenTerminated(t *testing.T) {
	now := time.Now().UTC()
	end := now.Add(-1 * time.Hour)
	sessions := []session.Session{
		{ID: "ses_old_ended", ProviderID: "claude", State: session.StateEnded,
			StartedAt: now.Add(-2 * time.Hour), EndedAt: &end},
		{ID: "ses_fresh_idle", ProviderID: "gemini", State: session.StateIdle,
			StartedAt: now.Add(-5 * time.Minute)},
		{ID: "ses_newest_running", ProviderID: "claude", State: session.StateRunning,
			StartedAt: now.Add(-30 * time.Second)},
	}
	h := listSessionsCardHandler(&fakeSessionOps{sessions: sessions})
	got, err := h(context.Background(), channel.CommandContext{})
	if err != nil {
		t.Fatal(err)
	}
	body := cardText(t, got)
	// Header counts active sessions only.
	if !strings.HasPrefix(body, "2 sessions") {
		t.Errorf("header wrong:\n%s", body)
	}
	// Live first (sorted by recency), terminated last.
	wantOrder := []string{"ses_newest_running", "ses_fresh_idle", "ses_old_ended"}
	for i, want := range wantOrder {
		if idx := strings.Index(body, want); idx < 0 {
			t.Errorf("missing %q in output:\n%s", want, body)
		} else if i+1 < len(wantOrder) {
			next := wantOrder[i+1]
			nextIdx := strings.Index(body, next)
			if nextIdx < idx {
				t.Errorf("order wrong: %q should precede %q\n%s", want, next, body)
			}
		}
	}
}

func TestListSessionsCardHandler_CapsAtMax(t *testing.T) {
	now := time.Now().UTC()
	sessions := make([]session.Session, 0, listSessionsMax+5)
	for i := 0; i < listSessionsMax+5; i++ {
		sessions = append(sessions, session.Session{
			ID:         fmt.Sprintf("ses_%02d", i),
			ProviderID: "claude",
			State:      session.StateRunning,
			StartedAt:  now.Add(-time.Duration(i) * time.Minute),
		})
	}
	h := listSessionsCardHandler(&fakeSessionOps{sessions: sessions})
	got, err := h(context.Background(), channel.CommandContext{})
	if err != nil {
		t.Fatal(err)
	}
	body := cardText(t, got)
	// listSessionsMax data rows + 1 header line; not 17 rows.
	lines := strings.Count(body, "\n") + 1
	if lines != listSessionsMax+1 {
		t.Errorf("want %d lines (cap + header), got %d\n%s",
			listSessionsMax+1, lines, body)
	}
	// Every shown session contributed exactly one button.
	if got := len(cardButtonValues(got)); got != listSessionsMax {
		t.Errorf("want %d buttons (one per shown session), got %d",
			listSessionsMax, got)
	}
}

// The whole point of this PR: each session row in /list produces a
// tappable button whose callback value carries the FULL session id,
// so the operator never has to type it. Live sessions get an /end
// button; terminated ones get /resume.
func TestListSessionsCardHandler_ButtonsCarryFullIdAndCorrectVerb(t *testing.T) {
	now := time.Now().UTC()
	end := now.Add(-1 * time.Hour)
	sessions := []session.Session{
		{ID: "ses_running1", ProviderID: "claude", State: session.StateRunning,
			StartedAt: now.Add(-1 * time.Minute)},
		{ID: "ses_idle1", ProviderID: "gemini", State: session.StateIdle,
			StartedAt: now.Add(-2 * time.Minute)},
		{ID: "ses_ended1", ProviderID: "claude", State: session.StateEnded,
			StartedAt: now.Add(-1 * time.Hour), EndedAt: &end},
		{ID: "ses_stopped1", ProviderID: "claude", State: session.StateStopped,
			StartedAt: now.Add(-2 * time.Hour), EndedAt: &end},
	}
	h := listSessionsCardHandler(&fakeSessionOps{sessions: sessions})
	got, err := h(context.Background(), channel.CommandContext{})
	if err != nil {
		t.Fatal(err)
	}
	values := cardButtonValues(got)
	want := []string{
		"cmd:/end ses_running1",
		"cmd:/end ses_idle1",
		"cmd:/resume ses_ended1",
		"cmd:/resume ses_stopped1",
	}
	if len(values) != len(want) {
		t.Fatalf("got %d buttons, want %d\nvalues: %v", len(values), len(want), values)
	}
	// Order: end buttons first (live sessions, in input recency
	// order), then resume buttons (terminated sessions).
	for i, w := range want {
		if values[i] != w {
			t.Errorf("button[%d] = %q, want %q\nall: %v", i, values[i], w, values)
		}
	}
}

func TestSessionShortID(t *testing.T) {
	cases := []struct{ in, want string }{
		{"ses_jwwDK7iAGqA-", "jwwDK7…"},
		{"ses_abc", "abc"},
		{"ses_", ""},
		{"plain", "plain"}, // unprefixed: still abbreviated past 6 chars
		{"plain12345", "plain1…"},
	}
	for _, c := range cases {
		if got := sessionShortID(c.in); got != c.want {
			t.Errorf("sessionShortID(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestEndSessionHandler_Usage(t *testing.T) {
	h := endSessionHandler(&fakeSessionOps{})
	got, err := h(context.Background(), channel.CommandContext{Args: nil})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "Usage:") {
		t.Errorf("missing usage hint: %q", got)
	}
}

func TestEndSessionHandler_Success(t *testing.T) {
	mgr := &fakeSessionOps{}
	h := endSessionHandler(mgr)
	got, err := h(context.Background(),
		channel.CommandContext{Args: []string{"ses_abc"}})
	if err != nil {
		t.Fatal(err)
	}
	if got != "Session ses_abc stopped." {
		t.Errorf("reply: %q", got)
	}
	if len(mgr.stopCalls) != 1 || mgr.stopCalls[0] != "ses_abc" {
		t.Errorf("Stop not called with ses_abc: %v", mgr.stopCalls)
	}
}

func TestEndSessionHandler_NotFoundIsFriendly(t *testing.T) {
	h := endSessionHandler(&fakeSessionOps{stopErr: session.ErrNotFound})
	got, err := h(context.Background(),
		channel.CommandContext{Args: []string{"ses_ghost"}})
	if err != nil {
		t.Fatalf("ErrNotFound should not propagate as error: %v", err)
	}
	if !strings.Contains(got, "not found") {
		t.Errorf("friendly message missing: %q", got)
	}
}

func TestEndSessionHandler_AlreadyEndedIsFriendly(t *testing.T) {
	h := endSessionHandler(&fakeSessionOps{stopErr: session.ErrAlreadyEnded})
	got, err := h(context.Background(),
		channel.CommandContext{Args: []string{"ses_done"}})
	if err != nil {
		t.Fatalf("ErrAlreadyEnded should not propagate: %v", err)
	}
	if !strings.Contains(got, "already ended") {
		t.Errorf("friendly message missing: %q", got)
	}
}

func TestEndSessionHandler_UnexpectedErrorPropagates(t *testing.T) {
	h := endSessionHandler(&fakeSessionOps{stopErr: errors.New("disk full")})
	_, err := h(context.Background(),
		channel.CommandContext{Args: []string{"ses_x"}})
	if err == nil {
		t.Error("unexpected errors should propagate so the channel layer logs them")
	}
}

func TestResumeSessionHandler_Success(t *testing.T) {
	mgr := &fakeSessionOps{startResp: session.Session{
		ID: "ses_abc", State: session.StateRunning,
	}}
	h := resumeSessionHandler(mgr)
	got, err := h(context.Background(),
		channel.CommandContext{Args: []string{"ses_abc"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "ses_abc") || !strings.Contains(got, "running") {
		t.Errorf("reply missing id/state: %q", got)
	}
	if len(mgr.startCalls) != 1 || mgr.startCalls[0] != "ses_abc" {
		t.Errorf("Start not called with ses_abc: %v", mgr.startCalls)
	}
}

func TestResumeSessionHandler_AlreadyRunningIsFriendly(t *testing.T) {
	h := resumeSessionHandler(&fakeSessionOps{startErr: session.ErrAlreadyRunning})
	got, err := h(context.Background(),
		channel.CommandContext{Args: []string{"ses_live"}})
	if err != nil {
		t.Fatalf("ErrAlreadyRunning should not propagate: %v", err)
	}
	if !strings.Contains(got, "already running") {
		t.Errorf("friendly message missing: %q", got)
	}
}

func TestResumeSessionHandler_NotFoundIsFriendly(t *testing.T) {
	h := resumeSessionHandler(&fakeSessionOps{startErr: session.ErrNotFound})
	got, err := h(context.Background(),
		channel.CommandContext{Args: []string{"ses_ghost"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "not found") {
		t.Errorf("friendly message missing: %q", got)
	}
}

func TestSingleSessionArg_StripsLeadingSlash(t *testing.T) {
	cases := []struct {
		in   []string
		want string
		ok   bool
	}{
		{nil, "", false},
		{[]string{""}, "", false},
		{[]string{"ses_abc"}, "ses_abc", true},
		// Operators sometimes type "/end /ses_abc" or paste IDs
		// alongside leading slashes — defang that.
		{[]string{"/ses_abc"}, "ses_abc", true},
		{[]string{"  ses_abc  "}, "ses_abc", true},
	}
	for _, c := range cases {
		got, ok := singleSessionArg(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("input %v: got %q,%v want %q,%v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestRelativeAge_Buckets(t *testing.T) {
	base := time.Date(2026, 5, 16, 12, 0, 0, 0, time.UTC)
	cases := []struct {
		ts   time.Time
		want string
	}{
		{base.Add(-30 * time.Second), "now"},
		{base.Add(-5 * time.Minute), "5m ago"},
		{base.Add(-3 * time.Hour), "3h ago"},
		{base.Add(-2 * 24 * time.Hour), "2d ago"},
		{time.Time{}, "(unknown)"},
	}
	for _, c := range cases {
		if got := relativeAge(c.ts, base); got != c.want {
			t.Errorf("ts=%v got %q want %q", c.ts, got, c.want)
		}
	}
}
