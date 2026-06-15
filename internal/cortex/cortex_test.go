package cortex

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

type fakeDocs struct {
	projects  []projectdoc.ProjectSummary
	proposals []projectdoc.Proposal
	err       error
}

func (f *fakeDocs) ListProjects(ctx context.Context, idleSuggestDays int) ([]projectdoc.ProjectSummary, error) {
	return f.projects, f.err
}

func (f *fakeDocs) ListPendingProposals(ctx context.Context, cwd string) ([]projectdoc.Proposal, error) {
	return f.proposals, f.err
}

func TestStatusAggregation(t *testing.T) {
	tests := []struct {
		name      string
		docs      *fakeDocs
		opts      []Option
		want      Status
		wantError bool
	}{
		{
			name: "empty install",
			docs: &fakeDocs{},
			want: Status{},
		},
		{
			name: "splits proposals by global cwd and counts frozen projects",
			docs: &fakeDocs{
				projects: []projectdoc.ProjectSummary{
					{Cwd: "/a", Status: projectdoc.StatusActive},
					{Cwd: "/b", Status: projectdoc.StatusPaused},
					{Cwd: "/c", Status: projectdoc.StatusArchived},
				},
				proposals: []projectdoc.Proposal{
					{Cwd: "/a", Kind: projectdoc.KindPlan},
					{Cwd: "/a", Kind: projectdoc.KindGoal},
					{Cwd: projectdoc.GlobalCwd, Kind: "kb_lessons"},
				},
			},
			opts: []Option{
				WithMemoryEnabled(true),
				WithKnowledgeEnabled(true),
				WithQuarantineCounter(func(ctx context.Context) (int, error) { return 7, nil }),
			},
			want: Status{
				Notes: NotesStatus{
					Projects:         3,
					ActiveProjects:   1,
					FrozenProjects:   2,
					PendingProposals: 2,
				},
				Memory:    MemoryStatus{Enabled: true, QuarantineCount: 7},
				Knowledge: KnowledgeStatus{Enabled: true, PendingProposals: 1},
			},
		},
		{
			name: "quarantine counter error soft-fails to zero",
			docs: &fakeDocs{},
			opts: []Option{
				WithMemoryEnabled(true),
				WithQuarantineCounter(func(ctx context.Context) (int, error) {
					return 0, errors.New("boom")
				}),
			},
			want: Status{Memory: MemoryStatus{Enabled: true}},
		},
		{
			name:      "docs error propagates",
			docs:      &fakeDocs{err: errors.New("db down")},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := NewService(tt.docs, nil, tt.opts...)
			got, err := svc.Status(context.Background())
			if tt.wantError {
				if err == nil {
					t.Fatalf("Status() error = nil, want error")
				}
				return
			}
			if err != nil {
				t.Fatalf("Status() error = %v", err)
			}
			if got != tt.want {
				t.Errorf("Status() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestStatusEndpoint(t *testing.T) {
	docs := &fakeDocs{
		projects:  []projectdoc.ProjectSummary{{Cwd: "/a", Status: projectdoc.StatusActive}},
		proposals: []projectdoc.Proposal{{Cwd: projectdoc.GlobalCwd, Kind: "kb_conventions"}},
	}
	svc := NewService(docs, nil, WithKnowledgeEnabled(true))

	r := chi.NewRouter()
	r.Get("/cortex/status", (&Handlers{svc: svc, log: slog.New(slog.DiscardHandler)}).status)

	req := httptest.NewRequest(http.MethodGet, "/cortex/status", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", rec.Code)
	}
	var got Status
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	want := Status{
		Notes:     NotesStatus{Projects: 1, ActiveProjects: 1},
		Knowledge: KnowledgeStatus{Enabled: true, PendingProposals: 1},
	}
	if got != want {
		t.Errorf("body = %+v, want %+v", got, want)
	}
}
