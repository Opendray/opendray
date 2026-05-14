package memconflict

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// SQLCwdLister implements CwdLister by reading distinct cwds out
// of project_docs — the canonical "projects opendray knows about"
// list. Sessions live in their own table but spawning into a cwd
// without a project_doc row is rare enough to skip here; once an
// operator seeds a goal or plan the detector picks it up.
type SQLCwdLister struct {
	pool *pgxpool.Pool
}

func NewSQLCwdLister(pool *pgxpool.Pool) *SQLCwdLister {
	return &SQLCwdLister{pool: pool}
}

func (l *SQLCwdLister) ListProjectCwds(ctx context.Context) ([]string, error) {
	rows, err := l.pool.Query(ctx, `
		SELECT DISTINCT cwd FROM project_docs
		 WHERE cwd <> ''
		 ORDER BY cwd`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
