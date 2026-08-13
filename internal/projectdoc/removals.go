package projectdoc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/knowledge"
)

// ─── deletion as signal ────────────────────────────────────────────
//
// An operator deleting a line from a global knowledge page is the
// highest-confidence negative signal the curation loop can receive,
// and it used to be discarded: the drafter regenerates pages from
// feedstock, so a deleted line whose evidence still existed upstream
// was re-derived on the next sweep — a tug-of-war the operator could
// not win, because nothing recorded that they were pulling.
//
// Every operator save of a global KB page now diffs against the
// previous body and records what disappeared. The escalation ladder:
//
//	count 1  soft — the line becomes negative context in the
//	         drafter's prompt ("the operator removed these").
//	count 2  hard — a second deletion of the SAME normalized line can
//	         only mean the system reintroduced it and the operator
//	         removed it again. The line is banned: deterministically
//	         scrubbed from every future draft of that page.
//
// A reworded line normalizes differently and starts a fresh row, so
// escalation structurally cannot fire on an ordinary rewrite. Line
// identity is knowledge.NormalizeLine — the same function the drafter
// uses to enforce bans, so the two ends cannot drift.

// Removal is one recorded line deletion on one page.
type Removal struct {
	ID           string    `json:"id"`
	Cwd          string    `json:"cwd"`
	Kind         string    `json:"kind"`
	LineText     string    `json:"line_text"`
	RemovalCount int       `json:"removal_count"`
	Status       string    `json:"status"` // active | banned | dismissed
	LastRemoved  time.Time `json:"last_removed_at"`
}

func lineHash(norm string) string {
	sum := sha256.Sum256([]byte(norm))
	return hex.EncodeToString(sum[:8])
}

// minedRemovals returns the lines present in prev but absent from next,
// keyed by normalized identity. Set difference rather than positional
// diff: for "which content disappeared", position is noise — a line that
// merely MOVED must not register as deleted. Blank lines and the
// drafter's own machinery (kb-sig markers, doc_read frames) are ignored.
func minedRemovals(prev, next string) map[string]string {
	nextSet := map[string]struct{}{}
	for _, ln := range strings.Split(next, "\n") {
		if n := knowledge.NormalizeLine(ln); n != "" {
			nextSet[n] = struct{}{}
		}
	}
	removed := map[string]string{}
	for _, ln := range strings.Split(prev, "\n") {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "<!--") {
			continue // sig markers / doc frames are not operator content
		}
		n := knowledge.NormalizeLine(ln)
		if n == "" {
			continue
		}
		if _, kept := nextSet[n]; !kept {
			removed[n] = trimmed
		}
	}
	return removed
}

// recordRemovals mines prev→next and upserts one row per deleted line.
// Escalation happens in SQL so two concurrent saves cannot double-count:
// a re-removal of an 'active' line bumps the count and flips to 'banned'
// at 2; a re-removal of a 'dismissed' line starts over at count 1 (the
// operator told us to forget it — a later deletion is a fresh signal,
// not strike two); a re-removal of a 'banned' line just refreshes the
// timestamp. Best-effort by design: a failure here must never fail the
// operator's save, so the caller logs and moves on.
func (s *Service) recordRemovals(ctx context.Context, cwd string, kind Kind, prev, next string) (int, error) {
	removed := minedRemovals(prev, next)
	for norm, raw := range removed {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO doc_line_removals
				(id, cwd, kind, line_hash, line_text, removal_count, status)
			VALUES ($1, $2, $3, $4, $5, 1, 'active')
			ON CONFLICT (cwd, kind, line_hash) DO UPDATE SET
				removal_count = CASE
					WHEN doc_line_removals.status = 'dismissed' THEN 1
					WHEN doc_line_removals.status = 'banned'    THEN doc_line_removals.removal_count
					ELSE doc_line_removals.removal_count + 1
				END,
				status = CASE
					WHEN doc_line_removals.status = 'dismissed' THEN 'active'
					WHEN doc_line_removals.status = 'banned'    THEN 'banned'
					WHEN doc_line_removals.removal_count + 1 >= 2 THEN 'banned'
					ELSE 'active'
				END,
				line_text       = EXCLUDED.line_text,
				last_removed_at = NOW()`,
			newID("dlr_"), cwd, string(kind), lineHash(norm), raw); err != nil {
			return 0, fmt.Errorf("projectdoc: record removal: %w", err)
		}
	}
	return len(removed), nil
}

// ListRemovals returns a page's recorded removals, newest first —
// banned and active rows only (dismissed rows are forgotten history).
// Powers the page-settings UI.
func (s *Service) ListRemovals(ctx context.Context, cwd, kind string) ([]Removal, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, cwd, kind, line_text, removal_count, status, last_removed_at
		  FROM doc_line_removals
		 WHERE cwd = $1 AND kind = $2 AND status <> 'dismissed'
		 ORDER BY last_removed_at DESC
		 LIMIT 100`, cwd, kind)
	if err != nil {
		return nil, fmt.Errorf("projectdoc: list removals: %w", err)
	}
	defer rows.Close()
	out := []Removal{}
	for rows.Next() {
		var r Removal
		if err := rows.Scan(&r.ID, &r.Cwd, &r.Kind, &r.LineText,
			&r.RemovalCount, &r.Status, &r.LastRemoved); err != nil {
			return nil, fmt.Errorf("projectdoc: scan removal: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// DismissRemoval unbans / forgets one recorded line. A later deletion of
// the same line starts a fresh count-1 row rather than resuming the old
// escalation — dismissal means "you were wrong to track this".
func (s *Service) DismissRemoval(ctx context.Context, id string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE doc_line_removals SET status = 'dismissed' WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("projectdoc: dismiss removal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RemovalSignals returns what the drafter consumes: the raw text of
// recent removals (soft prompt context) and the normalized form of
// banned lines (hard scrub list). One query, split two ways.
func (s *Service) RemovalSignals(ctx context.Context, cwd, kind string) (removed []string, bannedNorm []string, err error) {
	rows, err := s.ListRemovals(ctx, cwd, kind)
	if err != nil {
		return nil, nil, err
	}
	for i, r := range rows {
		if i < 20 { // prompt budget — most recent K
			removed = append(removed, r.LineText)
		}
		if r.Status == "banned" {
			bannedNorm = append(bannedNorm, knowledge.NormalizeLine(r.LineText))
		}
	}
	return removed, bannedNorm, nil
}
