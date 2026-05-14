// Package memconflict runs the M-PC cross-layer conflict
// detector. A daily LLM librarian asks "do any of these claims
// contradict each other?" across the project's plan + top facts +
// recent journal; matches land in memory_conflicts for operator
// review.
//
// Why a separate package: like memquery and memhealth, this code
// needs to read from both memory and projectdoc but neither
// should depend on the other directly. The package depends on
// pgxpool + worker.Registry + small read-only views into the
// other two services.
package memconflict

import (
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray-v2/internal/memory"
	"github.com/opendray/opendray-v2/internal/memory/worker"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// Layer enumerates which memory subsystem a conflict references.
// Mirrors the DB CHECK constraint in migration 0032.
type Layer string

const (
	LayerFact    Layer = "fact"
	LayerPlan    Layer = "plan"
	LayerGoal    Layer = "goal"
	LayerJournal Layer = "journal"
)

// Severity tags the urgency of a conflict — operators sort by it
// in the inbox. The detector LLM picks one of three based on how
// load-bearing the conflicting claim is.
type Severity string

const (
	SeverityLow    Severity = "low"
	SeverityMedium Severity = "medium"
	SeverityHigh   Severity = "high"
)

// Status mirrors the DB CHECK.
type Status string

const (
	StatusPending   Status = "pending"
	StatusAccepted  Status = "accepted"  // operator agrees, applied a fix
	StatusDismissed Status = "dismissed" // operator rejected the finding
	StatusExpired   Status = "expired"   // superseded by a later run
)

// Conflict is one row from memory_conflicts.
type Conflict struct {
	ID         string     `json:"id"`
	Cwd        string     `json:"cwd"`
	LayerA     Layer      `json:"layer_a"`
	RefA       string     `json:"ref_a"`
	LayerB     Layer      `json:"layer_b"`
	RefB       string     `json:"ref_b"`
	Evidence   string     `json:"evidence"`
	Severity   Severity   `json:"severity"`
	Status     Status     `json:"status"`
	DetectedAt time.Time  `json:"detected_at"`
	DecidedAt  *time.Time `json:"decided_at,omitempty"`
	DecidedBy  string     `json:"decided_by,omitempty"`
}

// Service runs detection cycles + serves the CRUD surface
// (list, decide). Stateless beyond the deps.
type Service struct {
	pool     *pgxpool.Pool
	mem      *memory.Service
	docs     *projectdoc.Service
	registry *worker.Registry
	log      *slog.Logger
}

// New wires the service. Every dep is required; nil → error so
// the composition root surfaces a boot-time misconfig.
func New(pool *pgxpool.Pool, mem *memory.Service, docs *projectdoc.Service, reg *worker.Registry, log *slog.Logger) (*Service, error) {
	if pool == nil {
		return nil, errors.New("memconflict: pool required")
	}
	if mem == nil {
		return nil, errors.New("memconflict: memory service required")
	}
	if docs == nil {
		return nil, errors.New("memconflict: projectdoc service required")
	}
	if reg == nil {
		return nil, errors.New("memconflict: worker registry required")
	}
	if log == nil {
		log = slog.Default()
	}
	return &Service{
		pool:     pool,
		mem:      mem,
		docs:     docs,
		registry: reg,
		log:      log.With("component", "memconflict"),
	}, nil
}

// DetectForCwd runs one detection cycle against a single cwd:
//
//  1. Gather inputs: plan doc, top-K facts ordered by hit_count,
//     last 14 days of journal.
//  2. Skip when the bundle is too thin to produce useful
//     conflicts (no plan + no facts).
//  3. Ask the configured conflict_detector worker to return a
//     JSON list of conflicts.
//  4. INSERT each new conflict (status=pending). Existing pending
//     conflicts for the same (layer_a, ref_a, layer_b, ref_b)
//     pair are left alone so the operator's prior judgements
//     aren't re-spammed.
//
// Returns the number of new conflicts written.
func (s *Service) DetectForCwd(ctx context.Context, cwd string) (int, error) {
	if strings.TrimSpace(cwd) == "" {
		return 0, errors.New("memconflict: cwd required")
	}
	bundle, err := s.gather(ctx, cwd)
	if err != nil {
		return 0, fmt.Errorf("gather inputs: %w", err)
	}
	if bundle.empty() {
		return 0, nil
	}

	userInput := bundle.render()
	resp, err := s.registry.Run(ctx, worker.Request{
		Task:                     worker.TaskConflictDetector,
		SystemPrompt:             ConflictDetectorSystemPrompt,
		UserInput:                userInput,
		MaxTokens:                4096,
		Timeout:                  5 * time.Minute,
		ResponseFormatJSONSchema: conflictJSONSchema,
	})
	if err != nil {
		return 0, fmt.Errorf("worker run: %w", err)
	}
	findings, err := ParseConflicts(resp.Content)
	if err != nil {
		s.log.Debug("memconflict: parse error", "err", err, "raw_len", len(resp.Content))
		return 0, nil
	}
	written := 0
	for _, f := range findings {
		if !validLayer(f.LayerA) || !validLayer(f.LayerB) {
			continue
		}
		ok, err := s.insertIfNew(ctx, cwd, f)
		if err != nil {
			s.log.Warn("memconflict: insert failed", "err", err)
			continue
		}
		if ok {
			written++
		}
	}
	if written > 0 {
		s.log.Info("memconflict: detection cycle done",
			"cwd", cwd, "new_conflicts", written, "total_findings", len(findings))
	}
	return written, nil
}

// List returns conflicts under cwd filtered by status (empty =
// all). limit ≤ 0 → 50, max 200.
func (s *Service) List(ctx context.Context, cwd, status string, limit int) ([]Conflict, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, errors.New("memconflict: cwd required")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	var (
		rows = s.pool
	)
	if status == "" {
		r, err := rows.Query(ctx, `
			SELECT id, cwd, layer_a, ref_a, layer_b, ref_b, evidence,
			       severity, status, detected_at, decided_at,
			       COALESCE(decided_by, '')
			  FROM memory_conflicts
			 WHERE cwd = $1
			 ORDER BY detected_at DESC
			 LIMIT $2`, cwd, limit)
		if err != nil {
			return nil, err
		}
		defer r.Close()
		return scanConflicts(r)
	}
	r, err := rows.Query(ctx, `
		SELECT id, cwd, layer_a, ref_a, layer_b, ref_b, evidence,
		       severity, status, detected_at, decided_at,
		       COALESCE(decided_by, '')
		  FROM memory_conflicts
		 WHERE cwd = $1 AND status = $2
		 ORDER BY detected_at DESC
		 LIMIT $3`, cwd, status, limit)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return scanConflicts(r)
}

// Decide updates one conflict's status. action must be one of
// "accepted" / "dismissed". Returns ErrNotFound when the id is
// missing or already decided.
func (s *Service) Decide(ctx context.Context, id, action, by string) error {
	if action != string(StatusAccepted) && action != string(StatusDismissed) {
		return fmt.Errorf("memconflict: invalid action %q", action)
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE memory_conflicts
		   SET status = $1,
		       decided_at = NOW(),
		       decided_by = $2
		 WHERE id = $3 AND status = 'pending'`,
		action, by, id)
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ErrNotFound — the conflict id doesn't exist or is no longer
// in pending state.
var ErrNotFound = errors.New("memconflict: not found or already decided")

// insertIfNew writes a conflict row unless an identical pending
// pair already exists. Returns (written, error). Existing
// non-pending rows for the same pair are ignored — operator
// already decided; don't resurrect.
func (s *Service) insertIfNew(ctx context.Context, cwd string, f Finding) (bool, error) {
	// Identity is the (layers + refs) tuple; order normalises to
	// (a < b) so swap-ordered findings dedup.
	la, ra, lb, rb := normaliseOrder(f.LayerA, f.RefA, f.LayerB, f.RefB)
	var existing int
	err := s.pool.QueryRow(ctx, `
		SELECT 1 FROM memory_conflicts
		 WHERE cwd = $1
		   AND layer_a = $2 AND ref_a = $3
		   AND layer_b = $4 AND ref_b = $5
		 LIMIT 1`, cwd, la, ra, lb, rb).Scan(&existing)
	if err == nil {
		return false, nil
	}
	id := newID()
	_, err = s.pool.Exec(ctx, `
		INSERT INTO memory_conflicts
			(id, cwd, layer_a, ref_a, layer_b, ref_b, evidence, severity, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending')`,
		id, cwd, la, ra, lb, rb, f.Evidence, severityOrDefault(f.Severity))
	if err != nil {
		return false, err
	}
	return true, nil
}

func validLayer(l Layer) bool {
	switch l {
	case LayerFact, LayerPlan, LayerGoal, LayerJournal:
		return true
	}
	return false
}

// normaliseOrder makes (layer_a, ref_a) < (layer_b, ref_b)
// lexically so swapped-pair findings collapse to one row.
func normaliseOrder(la Layer, ra string, lb Layer, rb string) (Layer, string, Layer, string) {
	keyA := string(la) + "|" + ra
	keyB := string(lb) + "|" + rb
	if keyA > keyB {
		return lb, rb, la, ra
	}
	return la, ra, lb, rb
}

func severityOrDefault(s Severity) Severity {
	switch s {
	case SeverityLow, SeverityMedium, SeverityHigh:
		return s
	}
	return SeverityMedium
}

func newID() string {
	var b [14]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic("memconflict: rand: " + err.Error())
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b[:])
	if len(enc) > 22 {
		enc = enc[:22]
	}
	return "mc_" + strings.ToLower(enc)
}

func scanConflicts(rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}) ([]Conflict, error) {
	var out []Conflict
	for rows.Next() {
		var c Conflict
		var (
			la, lb, status, severity string
			dec                      *time.Time
		)
		if err := rows.Scan(
			&c.ID, &c.Cwd, &la, &c.RefA, &lb, &c.RefB, &c.Evidence,
			&severity, &status, &c.DetectedAt, &dec, &c.DecidedBy,
		); err != nil {
			return nil, err
		}
		c.LayerA = Layer(la)
		c.LayerB = Layer(lb)
		c.Severity = Severity(severity)
		c.Status = Status(status)
		c.DecidedAt = dec
		out = append(out, c)
	}
	return out, rows.Err()
}

// conflictJSONSchema is the response_format=json_schema body that
// asks the model for a strict-shape reply. Same convention as the
// plan_drift detector — workers that don't natively support
// structured output append schema instructions to the prompt.
const conflictJSONSchema = `{
  "name": "memory_conflicts",
  "schema": {
    "type": "object",
    "additionalProperties": false,
    "properties": {
      "conflicts": {
        "type": "array",
        "items": {
          "type": "object",
          "additionalProperties": false,
          "properties": {
            "layer_a":  {"type": "string", "enum": ["fact","plan","goal","journal"]},
            "ref_a":    {"type": "string"},
            "layer_b":  {"type": "string", "enum": ["fact","plan","goal","journal"]},
            "ref_b":    {"type": "string"},
            "evidence": {"type": "string"},
            "severity": {"type": "string", "enum": ["low","medium","high"]}
          },
          "required": ["layer_a","ref_a","layer_b","ref_b","evidence","severity"]
        }
      }
    },
    "required": ["conflicts"]
  },
  "strict": true
}`

// ConflictDetectorSystemPrompt is the role block the worker
// receives. Exported for tests + alternate harness wiring.
const ConflictDetectorSystemPrompt = `You are an opendray memory librarian.

You will be given snapshots of one project's persistent context:
- the project PLAN document
- a list of high-hit-count FACTS the agent has stored
- the most recent JOURNAL entries

Your job: find pairs of claims that CONTRADICT each other.
Examples of contradictions worth flagging:
- plan says "we use sqlite", a fact says "DB is Postgres at host:5432"
- one fact says "user prefers pnpm", another says "user wants npm"
- journal says "we shipped feature X", plan still lists X as upcoming

Do NOT flag:
- complementary information ("X is at version Y" + "X exposes endpoint Z")
- old facts that are simply less detailed than newer ones
- facts that AGREE with the plan
- speculation — only flag genuine contradictions backed by the text

Output ONLY valid JSON of shape:
{"conflicts":[
  {"layer_a":"fact","ref_a":"<id>","layer_b":"plan","ref_b":"<id>",
   "evidence":"one short paragraph quoting both sides","severity":"low|medium|high"}
]}

Return an empty array when nothing conflicts. Do not invent ids —
use only the ids you were shown.`

// Finding is the unparsed shape returned by the detector LLM.
type Finding struct {
	LayerA   Layer    `json:"layer_a"`
	RefA     string   `json:"ref_a"`
	LayerB   Layer    `json:"layer_b"`
	RefB     string   `json:"ref_b"`
	Evidence string   `json:"evidence"`
	Severity Severity `json:"severity"`
}

// ParseConflicts pulls a []Finding out of a raw LLM response.
// Tolerates clean JSON, fenced blocks, leading prose — same
// resilience as projectdoc.ParseDriftResponse.
func ParseConflicts(raw string) ([]Finding, error) {
	body := strings.TrimSpace(raw)
	if body == "" {
		return nil, nil
	}
	if fenced := stripJSONFence(body); fenced != "" {
		body = fenced
	}
	if i := strings.IndexByte(body, '{'); i >= 0 {
		if j := strings.LastIndexByte(body, '}'); j > i {
			body = body[i : j+1]
		}
	}
	var wrapper struct {
		Conflicts []Finding `json:"conflicts"`
	}
	if err := json.Unmarshal([]byte(body), &wrapper); err != nil {
		return nil, fmt.Errorf("memconflict: parse: %w", err)
	}
	return wrapper.Conflicts, nil
}

func stripJSONFence(s string) string {
	const fence = "```"
	i := strings.Index(s, fence)
	if i < 0 {
		return ""
	}
	rest := s[i+len(fence):]
	rest = strings.TrimPrefix(rest, "json")
	rest = strings.TrimLeft(rest, " \t\r\n")
	j := strings.Index(rest, fence)
	if j < 0 {
		return strings.TrimSpace(rest)
	}
	return strings.TrimSpace(rest[:j])
}
