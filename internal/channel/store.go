package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// channelRow is the package-private DB shape for one row of the
// channels table.
type channelRow struct {
	ID      string          `json:"id"`
	Kind    string          `json:"kind"`
	Config  json.RawMessage `json:"config"`
	Enabled bool            `json:"enabled"`
}

type store struct{ pool *pgxpool.Pool }

func newStore(pool *pgxpool.Pool) *store { return &store{pool: pool} }

func (s *store) Insert(ctx context.Context, id, kind string, config json.RawMessage, enabled bool) error {
	if len(config) == 0 {
		config = []byte("{}")
	}
	_, err := s.pool.Exec(ctx, `
        INSERT INTO channels (id, kind, config, enabled) VALUES ($1, $2, $3::jsonb, $4)`,
		id, kind, []byte(config), enabled)
	if err != nil {
		return fmt.Errorf("insert channel: %w", err)
	}
	return nil
}

func (s *store) Get(ctx context.Context, id string) (channelRow, error) {
	row := s.pool.QueryRow(ctx,
		`SELECT id, kind, config, enabled FROM channels WHERE id=$1`, id)
	var r channelRow
	if err := row.Scan(&r.ID, &r.Kind, &r.Config, &r.Enabled); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return channelRow{}, ErrNotFound
		}
		return channelRow{}, fmt.Errorf("scan channel: %w", err)
	}
	return r, nil
}

func (s *store) List(ctx context.Context) ([]channelRow, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, kind, config, enabled FROM channels ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("list channels: %w", err)
	}
	defer rows.Close()
	var out []channelRow
	for rows.Next() {
		var r channelRow
		if err := rows.Scan(&r.ID, &r.Kind, &r.Config, &r.Enabled); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *store) Update(ctx context.Context, id string, config json.RawMessage, enabled *bool) error {
	if config != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE channels SET config=$1::jsonb WHERE id=$2`,
			[]byte(config), id); err != nil {
			return fmt.Errorf("update config: %w", err)
		}
	}
	if enabled != nil {
		if _, err := s.pool.Exec(ctx,
			`UPDATE channels SET enabled=$1 WHERE id=$2`, *enabled, id); err != nil {
			return fmt.Errorf("update enabled: %w", err)
		}
	}
	return nil
}

func (s *store) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM channels WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete channel: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *store) InsertMessage(ctx context.Context, msg ChannelMessage) (int64, error) {
	metaJSON, err := json.Marshal(msg.Metadata)
	if err != nil {
		return 0, fmt.Errorf("marshal metadata: %w", err)
	}
	if msg.Metadata == nil {
		metaJSON = []byte("{}")
	}
	var id int64
	err = s.pool.QueryRow(ctx, `
        INSERT INTO channel_messages
            (channel_id, direction, conversation_id, session_id, author, text, metadata, ts)
        VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8)
        RETURNING id`,
		msg.ChannelID, string(msg.Direction), msg.ConversationID,
		nullIfEmpty(msg.SessionID), nullIfEmpty(msg.Author), msg.Text,
		metaJSON, msg.Timestamp).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert channel_message: %w", err)
	}
	return id, nil
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
