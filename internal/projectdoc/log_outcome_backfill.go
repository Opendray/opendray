package projectdoc

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// log_outcome_backfill.go recovers the session OUTCOME for historical
// session_summary rows (migration 0070) so the experience compiler's episode
// feedstock includes the whole journal — not just the ~2 rows whose ephemeral
// `sessions` row happens to survive. The outcome was always serialized into
// the summary by buildJournalBody (state in the title, started/ended/exit_code
// in the body), so we parse it back out. Forward-written rows already carry
// the columns; this is a one-shot for the legacy backlog.

// parseLogOutcome extracts the outcome a journaler-written session_summary
// encodes in its title + body:
//
//	title: "Session <id> — <provider> — <state>"
//	body:  "- started: <RFC3339>" / "- ended: <RFC3339>" / "- exit_code: <int>"
//
// NOTE: the title separator is an EM DASH (U+2014, " — "), matching
// buildJournalBody — do NOT "fix" it to an ASCII hyphen or state extraction
// silently breaks. Returns zero values for anything it can't find; callers
// default state to "ended" (a session_summary means the session terminated).
func parseLogOutcome(title, content string) (state string, exitCode *int, startedAt, endedAt *time.Time) {
	if parts := strings.Split(title, " — "); len(parts) >= 2 {
		if s := strings.TrimSpace(parts[len(parts)-1]); s == "ended" || s == "stopped" {
			state = s
		}
	}
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		switch {
		case strings.HasPrefix(line, "- started:"):
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(strings.TrimPrefix(line, "- started:"))); err == nil {
				startedAt = &t
			}
		case strings.HasPrefix(line, "- ended:"):
			if t, err := time.Parse(time.RFC3339, strings.TrimSpace(strings.TrimPrefix(line, "- ended:"))); err == nil {
				endedAt = &t
			}
		case strings.HasPrefix(line, "- exit_code:"):
			if n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "- exit_code:"))); err == nil {
				exitCode = &n
			}
		}
	}
	return state, exitCode, startedAt, endedAt
}

// BackfillLogOutcomes is a one-shot, idempotent pass that fills the outcome
// columns for every session_summary row that still has a NULL outcome_state,
// by parsing the row's own body. Returns the number of rows updated. Runs at
// boot, best-effort: every recovered row becomes experience-compiler feedstock
// it was previously invisible to.
func (s *Service) BackfillLogOutcomes(ctx context.Context) (int, error) {
	// LIMIT bounds memory + connection-hold time. Realistic journals are tiny
	// (~100s of rows); if a backlog ever exceeds this, each boot drains
	// another batch (recovered rows get a non-NULL outcome_state, so they drop
	// out of the predicate) until it converges.
	rows, err := s.pool.Query(ctx, `
		SELECT id, title, content
		  FROM session_logs
		 WHERE kind = 'session_summary' AND outcome_state IS NULL
		 LIMIT 50000`)
	if err != nil {
		return 0, fmt.Errorf("backfill outcomes: query: %w", err)
	}
	type patch struct {
		id        string
		state     string
		exitCode  *int
		startedAt *time.Time
		endedAt   *time.Time
	}
	var patches []patch
	for rows.Next() {
		var id, title, content string
		if err := rows.Scan(&id, &title, &content); err != nil {
			rows.Close()
			return 0, fmt.Errorf("backfill outcomes: scan: %w", err)
		}
		state, exitCode, startedAt, endedAt := parseLogOutcome(title, content)
		if state == "" {
			// A session_summary always means the session terminated; default
			// to the lenient terminal state so the row is marked processed
			// (won't re-scan next boot) and counts as success-eligible.
			state = "ended"
		}
		patches = append(patches, patch{id, state, exitCode, startedAt, endedAt})
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("backfill outcomes: rows: %w", err)
	}
	// Close the cursor (release the pooled conn) BEFORE the UPDATE loop — do
	// NOT convert this to `defer rows.Close()`, which would hold the cursor's
	// connection open across every UPDATE below.
	rows.Close()

	n := 0
	for _, p := range patches {
		if _, err := s.pool.Exec(ctx, `
			UPDATE session_logs
			   SET outcome_state = $1, exit_code = $2, started_at = $3, ended_at = $4
			 WHERE id = $5`, p.state, p.exitCode, p.startedAt, p.endedAt, p.id); err != nil {
			s.log.Warn("backfill outcomes: update failed", "id", p.id, "err", err)
			continue
		}
		n++
	}
	return n, nil
}
