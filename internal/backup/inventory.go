package backup

import (
	"context"
	"fmt"
	"sort"
)

// InventoryGroup is one logical bucket of tables shown in the
// "What's in a backup?" panel.
type InventoryGroup struct {
	ID          string           `json:"id"`
	Label       string           `json:"label"`
	Description string           `json:"description"`
	Tables      []InventoryTable `json:"tables"`
}

// InventoryTable is one row in the panel: table name + live count.
// Count is the current row count; the bundle includes whatever's
// there at backup time, which may differ slightly.
type InventoryTable struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

// inventoryGroups defines the groupings shown in the UI. We don't
// auto-discover from information_schema because (a) we want stable
// human-friendly labels, and (b) some tables (memory_index_state,
// schema_migrations) are bookkeeping and grouping them together
// gives operators a cleaner picture than a flat alphabetical list.
//
// Order within groups roughly follows "what an operator cares
// about most when restoring".
var inventoryGroups = []struct {
	id          string
	label       string
	description string
	tables      []string
}{
	{
		id:          "core",
		label:       "Core runtime",
		description: "Sessions, channels, audit. The day-to-day operational state.",
		tables:      []string{"sessions", "channels", "channel_messages", "audit_log", "providers"},
	},
	{
		id:          "integrations",
		label:       "Integrations",
		description: "Third-party API gateways + their call history. API keys are bcrypt — restoring preserves the hash, but rotation is recommended on a fresh install.",
		tables:      []string{"integrations", "integration_call_log"},
	},
	{
		id:          "memory",
		label:       "Memory",
		description: "Cross-CLI persistent memory, retrieval index state, and hit counters.",
		tables:      []string{"memories", "memory_index_state"},
	},
	{
		id:          "config",
		label:       "Configuration & operator content",
		description: "Custom tasks, Claude account configs, git host credentials, vault sync state.",
		tables:      []string{"custom_tasks", "claude_accounts", "git_hosts", "vault_sync"},
	},
	{
		id:          "backup-self",
		label:       "Backup subsystem (self-reference)",
		description: "Rows tracking prior backups, schedules, targets, exports, imports. A restored backup will list itself here.",
		tables:      []string{"backups", "backup_schedules", "backup_targets", "exports", "imports"},
	},
	{
		id:          "schema",
		label:       "Schema bookkeeping",
		description: "Migration version table. Restored alongside the schema itself.",
		tables:      []string{"schema_migrations"},
	},
}

// Inventory enumerates every table the next backup will capture
// (i.e. everything pg_dump sees), grouped for UI display, with a
// live row count alongside each.
//
// This is read-only and cheap — a parallel COUNT(*) per table.
// Tables that don't exist (e.g. older / branch deployments) are
// silently dropped from the response.
func (s *Service) Inventory(ctx context.Context) ([]InventoryGroup, error) {
	existing, err := s.listExistingTables(ctx)
	if err != nil {
		return nil, fmt.Errorf("inventory: list tables: %w", err)
	}
	out := make([]InventoryGroup, 0, len(inventoryGroups))
	for _, g := range inventoryGroups {
		grp := InventoryGroup{
			ID:          g.id,
			Label:       g.label,
			Description: g.description,
			Tables:      []InventoryTable{},
		}
		for _, name := range g.tables {
			if !existing[name] {
				continue
			}
			n, err := s.tableCount(ctx, name)
			if err != nil {
				s.log.Warn("inventory: count failed", "table", name, "err", err)
				continue
			}
			grp.Tables = append(grp.Tables, InventoryTable{Name: name, Count: n})
		}
		if len(grp.Tables) > 0 {
			out = append(out, grp)
		}
	}
	return out, nil
}

func (s *Service) listExistingTables(ctx context.Context) (map[string]bool, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT table_name FROM information_schema.tables
		 WHERE table_schema = 'public' AND table_type = 'BASE TABLE'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var n string
		if err := rows.Scan(&n); err != nil {
			return nil, err
		}
		out[n] = true
	}
	return out, rows.Err()
}

func (s *Service) tableCount(ctx context.Context, table string) (int64, error) {
	// Table name comes from the inventory whitelist + verified
	// against information_schema, so direct interpolation is safe.
	// pgx parameters can't substitute identifiers anyway.
	if !isValidIdent(table) {
		return 0, fmt.Errorf("invalid table name %q", table)
	}
	var n int64
	err := s.pool.QueryRow(ctx, fmt.Sprintf("SELECT COUNT(*) FROM %s", table)).Scan(&n)
	return n, err
}

// isValidIdent guards the table-name interpolation in tableCount
// against any future code path that bypasses the whitelist.
func isValidIdent(s string) bool {
	if s == "" || len(s) > 64 {
		return false
	}
	for _, r := range s {
		allowed := r == '_' ||
			(r >= 'a' && r <= 'z') ||
			(r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9')
		if !allowed {
			return false
		}
	}
	return true
}

// _ keeps sort imported in case future iterations sort by row count.
var _ = sort.Strings
