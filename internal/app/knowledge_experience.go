package app

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	customtask "github.com/opendray/opendray-v2/internal/customtask"
	"github.com/opendray/opendray-v2/internal/knowledge"
	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// knowledgeEpisodeSource feeds the experience compiler with cross-project
// work episodes: session-summary journal rows joined with the session's real
// outcome (exit code / terminal state) and duration. A read-only SQL adapter
// over tables other packages own (same precedent as memconflict.SQLCwdLister)
// so knowledge keeps zero table coupling.
type knowledgeEpisodeSource struct{ pool *pgxpool.Pool }

func (e knowledgeEpisodeSource) ListEpisodes(ctx context.Context, since time.Time, limit int) ([]knowledge.Episode, error) {
	if limit <= 0 {
		limit = 400
	}
	// Read the outcome straight off session_logs (denormalized by the
	// journaler, migration 0070) — NO JOIN to the ephemeral `sessions` table,
	// which is pruned and used to silently drop ~all historical episodes.
	rows, err := e.pool.Query(ctx, `
		SELECT l.session_id, l.cwd, l.title, l.content, l.created_at,
		       COALESCE(l.embedding::text, ''), COALESCE(l.embedder, ''),
		       l.started_at, l.ended_at, l.exit_code, COALESCE(l.outcome_state, '')
		  FROM session_logs l
		 WHERE l.kind = 'session_summary'
		   AND l.session_id IS NOT NULL
		   AND l.created_at >= $1
		 ORDER BY l.created_at DESC
		 LIMIT $2`, since, limit)
	if err != nil {
		return nil, fmt.Errorf("episode source: query: %w", err)
	}
	defer rows.Close()
	out := []knowledge.Episode{}
	for rows.Next() {
		var (
			ep        knowledge.Episode
			vecText   string
			startedAt *time.Time // nullable now (un-backfilled rows)
			endedAt   *time.Time
			exitCode  *int
			state     string
		)
		if err := rows.Scan(&ep.SessionID, &ep.Cwd, &ep.Title, &ep.Content, &ep.CreatedAt,
			&vecText, &ep.Embedder, &startedAt, &endedAt, &exitCode, &state); err != nil {
			return nil, err
		}
		// Temp dirs are not projects — they leave no experience either.
		if projectdoc.IsEphemeralCwd(ep.Cwd) {
			continue
		}
		ep.Embedding = knowledge.ParseVector(vecText)
		if startedAt != nil && endedAt != nil {
			ep.Duration = endedAt.Sub(*startedAt)
		}
		// Same heuristic the journaler applies to the skill-outcome loop.
		// An unknown outcome (NULL state + NULL exit, e.g. an un-backfilled
		// legacy row) is treated as success — lenient on purpose so the
		// recovered corpus feeds the compiler rather than being dropped.
		ep.Success = state == "stopped" || exitCode == nil || *exitCode == 0
		out = append(out, ep)
	}
	return out, rows.Err()
}

// knowledgeTaskSink registers a compiled skill's run.sh as an opendray custom
// task (upsert by name) so the operator can click-run the procedure from the
// Inspector and integrations can trigger it through the gateway.
type knowledgeTaskSink struct {
	tasks      *customtask.Service
	skillsRoot string
}

func (s knowledgeTaskSink) EnsureSkillTask(ctx context.Context, slug, title, description, cwd string) error {
	name := "skill:" + slug
	command := "bash " + filepath.Join(s.skillsRoot, slug, "run.sh")
	existing, err := s.tasks.List(ctx, "", true)
	if err != nil {
		return err
	}
	for _, t := range existing {
		if t.Name != name {
			continue
		}
		_, err := s.tasks.Update(ctx, t.ID, customtask.UpdateRequest{
			Command:     &command,
			Description: &description,
			Cwd:         &cwd,
		})
		return err
	}
	_, err = s.tasks.Create(ctx, customtask.CreateRequest{
		Name:        name,
		Command:     command,
		Description: description,
		Cwd:         cwd,
	})
	return err
}
