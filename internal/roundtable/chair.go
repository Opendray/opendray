package roundtable

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/opendray/opendray-v2/internal/eventbus"
	"github.com/opendray/opendray-v2/internal/memory/worker"
)

// ContextSource supplies relevant prior context (memories / journal / docs)
// for the propose prompt, keyed off the table's cwd + topic. Backed by
// memquery in the app; nil disables enrichment. Mirrors cortex.ContextSource.
type ContextSource interface {
	RelevantContext(ctx context.Context, cwd, query string, topK int) (string, error)
}

// tuning — Phase 1 fixed budget. A round is exactly two model beats
// (propose + critique); synthesize is pure code, zero model calls. Cost
// upper bound = seats × 2 × maxTokens.
const (
	beatMaxTokens = 4096
	beatTimeout   = 5 * time.Minute
	runTimeout    = 15 * time.Minute
)

// Service is the deterministic chair. It owns the schedule; the seats only
// ever produce structured proposals + critiques. The chair never calls a
// model — synthesis is mechanical aggregation (see synthesize).
type Service struct {
	store    *Store
	registry *worker.Registry
	bus      *eventbus.Hub
	context  ContextSource // optional
	log      *slog.Logger
}

// NewService wires the chair. store + registry required.
func NewService(store *Store, reg *worker.Registry, bus *eventbus.Hub, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		store:    store,
		registry: reg,
		bus:      bus,
		log:      log.With("component", "roundtable.chair"),
	}
}

// WithContextSource enables propose-prompt enrichment with relevant memories.
func (s *Service) WithContextSource(c ContextSource) *Service {
	s.context = c
	return s
}

// Start kicks off the discussion in the background and returns immediately.
// The table moves draft → running; the async run lands the Verdict
// (status awaiting_verdict) or an error (status failed). Progress is
// announced on the eventbus topic "roundtable.updated".
func (s *Service) Start(ctx context.Context, id string) error {
	rt, err := s.store.Get(ctx, id)
	if err != nil {
		return err
	}
	if rt.Status != StatusDraft {
		return fmt.Errorf("roundtable: table %s is %s, can only start a draft", id, rt.Status)
	}
	if err := s.store.SetStatus(ctx, id, StatusRunning, ""); err != nil {
		return err
	}
	s.announce(id)
	// Detached: agent-mode beats take minutes and must not hold the HTTP
	// request. Pass the id (re-read fresh inside run).
	go s.run(rt.ID)
	return nil
}

// proposal is the strict shape each seat returns in the PROPOSE beat.
type proposal struct {
	Summary    string   `json:"summary"`
	Plan       string   `json:"plan"`
	Tasks      []string `json:"tasks"`
	Tradeoffs  []string `json:"tradeoffs"`
	Confidence float64  `json:"confidence"`
}

const proposeSchema = `{
  "name": "round_table_proposal",
  "schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "summary":    {"type": "string"},
      "plan":       {"type": "string"},
      "tasks":      {"type": "array", "items": {"type": "string"}},
      "tradeoffs":  {"type": "array", "items": {"type": "string"}},
      "confidence": {"type": "number"}
    },
    "required": ["summary", "plan", "tasks", "tradeoffs", "confidence"]
  },
  "strict": true
}`

// critique is one seat's assessment of another seat's proposal.
type critique struct {
	TargetProvider string `json:"target_provider"`
	Severity       string `json:"severity"` // blocker | concern | nit
	Point          string `json:"point"`
}

type critiqueSet struct {
	Critiques []critique `json:"critiques"`
}

const critiqueSchema = `{
  "name": "round_table_critique",
  "schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "critiques": {
        "type": "array",
        "items": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "target_provider": {"type": "string"},
            "severity": {"type": "string", "enum": ["blocker", "concern", "nit"]},
            "point": {"type": "string"}
          },
          "required": ["target_provider", "severity", "point"]
        }
      }
    },
    "required": ["critiques"]
  },
  "strict": true
}`

// run executes the fixed three-beat schedule. Every failure path records a
// system turn and, if fatal, moves the table to failed — the operator is
// never left staring at a silently dead table.
func (s *Service) run(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), runTimeout)
	defer cancel()

	rt, err := s.store.Get(ctx, id)
	if err != nil {
		s.log.Error("roundtable run: load failed", "id", id, "err", err)
		return
	}

	fatal := func(stage string, err error) {
		s.log.Warn("roundtable run failed", "id", id, "stage", stage, "err", err)
		_, _ = s.store.AppendTurn(ctx, Turn{
			RoundTableID: id, Beat: BeatSynthesize, Role: RoleSystem,
			Content: fmt.Sprintf("Round table failed (%s): %v", stage, err),
		})
		_ = s.store.SetStatus(ctx, id, StatusFailed, fmt.Sprintf("%s: %v", stage, err))
		s.announce(id)
	}

	// Shared context enrichment, computed once for all seats.
	enrich := s.relevantContext(ctx, rt)

	// ── Beat 1: PROPOSE (parallel, seats blind to each other) ──
	proposals := s.propose(ctx, rt, enrich)
	if len(proposals) == 0 {
		fatal("propose", fmt.Errorf("no seat produced a usable proposal"))
		return
	}

	// ── Beat 2: CRITIQUE (parallel, each seat sees the others) ──
	critiques := s.critique(ctx, rt, proposals)

	// ── Beat 3: SYNTHESIZE (deterministic, no model call) ──
	verdict := synthesize(proposals, seatOrder(rt.Seats), critiques)
	vJSON, _ := json.Marshal(verdict)
	_, _ = s.store.AppendTurn(ctx, Turn{
		RoundTableID: id, Beat: BeatSynthesize, Role: RoleChair,
		Content:    fmt.Sprintf("Chair synthesis: recommended %s's approach.", verdict.RecommendedBy),
		Structured: json.RawMessage(vJSON),
	})
	if err := s.store.SetVerdict(ctx, id, verdict); err != nil {
		fatal("verdict", err)
		return
	}
	s.announce(id)
}

// propose runs the PROPOSE beat: every seat independently, in parallel.
// A seat that errors is skipped (graceful degradation — the table survives
// a down provider) with a system turn recording why.
func (s *Service) propose(ctx context.Context, rt RoundTable, enrich string) map[string]proposal {
	type result struct {
		provider string
		prop     proposal
		ok       bool
	}
	results := make([]result, len(rt.Seats))
	var wg sync.WaitGroup
	for i, seat := range rt.Seats {
		wg.Add(1)
		go func(i int, seat Seat) {
			defer wg.Done()
			system := proposeSystemPrompt()
			user := proposeUserPrompt(rt, enrich)
			resp, err := s.runSeat(ctx, seat, system, user, proposeSchema)
			if err != nil {
				s.seatFailed(ctx, rt.ID, BeatPropose, seat, err)
				return
			}
			var p proposal
			if err := parseJSON(resp, &p); err != nil {
				s.seatFailed(ctx, rt.ID, BeatPropose, seat, fmt.Errorf("parse: %w", err))
				return
			}
			raw, _ := json.Marshal(p)
			_, _ = s.store.AppendTurn(ctx, Turn{
				RoundTableID: rt.ID, Beat: BeatPropose, Role: RoleSeat,
				SeatProvider: seat.Provider, SeatModel: seat.Model,
				Content: p.Summary, Structured: json.RawMessage(raw),
			})
			results[i] = result{provider: seat.Provider, prop: p, ok: true}
		}(i, seat)
	}
	wg.Wait()

	out := make(map[string]proposal, len(results))
	for _, r := range results {
		if r.ok {
			out[r.provider] = r.prop
		}
	}
	return out
}

// critique runs the CRITIQUE beat: each seat that produced a proposal sees
// every OTHER seat's proposal and critiques them, in parallel.
func (s *Service) critique(ctx context.Context, rt RoundTable, proposals map[string]proposal) []critique {
	var (
		mu  sync.Mutex
		all []critique
		wg  sync.WaitGroup
	)
	for _, seat := range rt.Seats {
		own, ok := proposals[seat.Provider]
		if !ok {
			continue // seat had no proposal → it doesn't critique
		}
		wg.Add(1)
		go func(seat Seat, own proposal) {
			defer wg.Done()
			system := critiqueSystemPrompt()
			user := critiqueUserPrompt(rt, seat.Provider, own, proposals)
			resp, err := s.runSeat(ctx, seat, system, user, critiqueSchema)
			if err != nil {
				s.seatFailed(ctx, rt.ID, BeatCritique, seat, err)
				return
			}
			var cs critiqueSet
			if err := parseJSON(resp, &cs); err != nil {
				s.seatFailed(ctx, rt.ID, BeatCritique, seat, fmt.Errorf("parse: %w", err))
				return
			}
			// Drop self-targeted and unknown-target critiques defensively.
			valid := cs.Critiques[:0]
			for _, c := range cs.Critiques {
				c.TargetProvider = strings.TrimSpace(c.TargetProvider)
				if c.TargetProvider == seat.Provider {
					continue
				}
				if _, exists := proposals[c.TargetProvider]; !exists {
					continue
				}
				valid = append(valid, c)
			}
			raw, _ := json.Marshal(critiqueSet{Critiques: valid})
			_, _ = s.store.AppendTurn(ctx, Turn{
				RoundTableID: rt.ID, Beat: BeatCritique, Role: RoleSeat,
				SeatProvider: seat.Provider, SeatModel: seat.Model,
				Content:    fmt.Sprintf("%d critique(s)", len(valid)),
				Structured: json.RawMessage(raw),
			})
			mu.Lock()
			all = append(all, valid...)
			mu.Unlock()
		}(seat, own)
	}
	wg.Wait()
	return all
}

// runSeat dispatches one headless agent call for a seat via the worker
// registry's per-call override path. TaskCuration is used purely as the
// metrics label (round table needs no memory_workers row of its own — see
// ROLLBACK.md).
func (s *Service) runSeat(ctx context.Context, seat Seat, system, user, schema string) (string, error) {
	resp, err := s.registry.RunWith(ctx, worker.Config{
		Task:       worker.TaskCuration,
		Kind:       worker.WorkerAgent,
		ProviderID: seat.Provider,
		Model:      seat.Model,
		AccountID:  seat.AccountID,
		Enabled:    true,
	}, worker.Request{
		Task:                     worker.TaskCuration,
		SystemPrompt:             system,
		UserInput:                user,
		MaxTokens:                beatMaxTokens,
		Timeout:                  beatTimeout,
		ResponseFormatJSONSchema: schema,
	})
	if err != nil {
		return "", err
	}
	return resp.Content, nil
}

func (s *Service) seatFailed(ctx context.Context, id, beat string, seat Seat, err error) {
	s.log.Warn("roundtable seat failed", "id", id, "beat", beat, "provider", seat.Provider, "err", err)
	_, _ = s.store.AppendTurn(ctx, Turn{
		RoundTableID: id, Beat: beat, Role: RoleSystem,
		SeatProvider: seat.Provider,
		Content:      fmt.Sprintf("seat %s failed at %s: %v", seat.Provider, beat, err),
	})
}

func (s *Service) relevantContext(ctx context.Context, rt RoundTable) string {
	if s.context == nil || strings.TrimSpace(rt.Cwd) == "" {
		return ""
	}
	extra, err := s.context.RelevantContext(ctx, rt.Cwd, rt.Topic, 8)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(extra)
}

func (s *Service) announce(id string) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(eventbus.Event{
		Topic: "roundtable.updated",
		Data:  map[string]any{"round_table_id": id},
	})
}

// ─── deterministic synthesis (pure, unit-tested) ───────────────

// seatOrder returns provider ids in seat order — the stable tie-break key
// for ranking so the Verdict is fully reproducible.
func seatOrder(seats []Seat) []string {
	out := make([]string, len(seats))
	for i, s := range seats {
		out[i] = s.Provider
	}
	return out
}

// synthesize mechanically assembles the Verdict from proposals + critiques.
// The chair makes NO judgement call and runs NO model: it ranks by
// (fewest blockers, fewest concerns, highest self-confidence, seat order),
// picks the top proposal as recommended, lists the rest as alternatives,
// unions tradeoffs, and surfaces every blocker/concern critique as an open
// question for the human to resolve. Fully deterministic → reproducible.
func synthesize(proposals map[string]proposal, order []string, critiques []critique) Verdict {
	blockers := map[string]int{}
	concerns := map[string]int{}
	for _, c := range critiques {
		switch c.Severity {
		case "blocker":
			blockers[c.TargetProvider]++
		case "concern":
			concerns[c.TargetProvider]++
		}
	}

	orderIdx := map[string]int{}
	for i, p := range order {
		orderIdx[p] = i
	}

	scores := make([]SeatScore, 0, len(proposals))
	for provider, p := range proposals {
		scores = append(scores, SeatScore{
			Provider:   provider,
			Blockers:   blockers[provider],
			Concerns:   concerns[provider],
			Confidence: p.Confidence,
		})
	}
	sort.SliceStable(scores, func(i, j int) bool {
		a, b := scores[i], scores[j]
		if a.Blockers != b.Blockers {
			return a.Blockers < b.Blockers
		}
		if a.Concerns != b.Concerns {
			return a.Concerns < b.Concerns
		}
		if a.Confidence != b.Confidence {
			return a.Confidence > b.Confidence
		}
		return orderIdx[a.Provider] < orderIdx[b.Provider]
	})

	v := Verdict{Ranking: scores}
	if len(scores) == 0 {
		return v
	}

	top := scores[0].Provider
	winner := proposals[top]
	v.Recommended = winner.Plan
	v.RecommendedBy = top
	v.TaskBreakdown = winner.Tasks

	for _, sc := range scores[1:] {
		if summary := proposals[sc.Provider].Summary; summary != "" {
			v.Alternatives = append(v.Alternatives, fmt.Sprintf("%s: %s", sc.Provider, summary))
		}
	}

	// Tradeoffs = union of every proposal's declared tradeoffs (dedup).
	seen := map[string]bool{}
	for _, p := range order {
		for _, t := range proposals[p].Tradeoffs {
			t = strings.TrimSpace(t)
			if t != "" && !seen[t] {
				seen[t] = true
				v.Tradeoffs = append(v.Tradeoffs, t)
			}
		}
	}

	// Open questions = every blocker/concern critique, so nothing the panel
	// flagged gets silently dropped before the human decides.
	for _, c := range critiques {
		if c.Severity == "blocker" || c.Severity == "concern" {
			v.OpenQuestions = append(v.OpenQuestions,
				fmt.Sprintf("[%s → %s] %s", c.Severity, c.TargetProvider, strings.TrimSpace(c.Point)))
		}
	}
	return v
}

// parseJSON tolerates fenced / preambled JSON around the object.
func parseJSON(raw string, out any) error {
	body := strings.TrimSpace(raw)
	if i := strings.IndexByte(body, '{'); i >= 0 {
		if j := strings.LastIndexByte(body, '}'); j > i {
			body = body[i : j+1]
		}
	}
	if err := json.Unmarshal([]byte(body), out); err != nil {
		return fmt.Errorf("roundtable: unparseable seat reply: %w", err)
	}
	return nil
}
