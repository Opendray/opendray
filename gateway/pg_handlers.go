package gateway

// HTTP handlers for the pg-browser panel plugin. See gateway/pg/ for
// the pgxpool-backed execution engine + schema introspection.
//
//   GET  /api/pg/{plugin}/databases                        → [name, ...]
//   GET  /api/pg/{plugin}/schemas?database=                → [schema, ...]
//   GET  /api/pg/{plugin}/tables?schema=&database=         → [{schema, name, kind}, ...]
//   GET  /api/pg/{plugin}/columns?schema=&table=&database= → [{name, type, nullable, default}, ...]
//   POST /api/pg/{plugin}/query    { sql, database? }      → { columns, rows, rowCount, truncated, durationMs }
//   POST /api/pg/{plugin}/execute  { sql, database? }      → { rowsAffected, verb, durationMs }
//
// Config flows through Server.effectiveConfig, so host/port/user/
// database come from plugin_kv.__config.* and the password (secret
// field) is decrypted from plugin_secret before reaching the pool.
// Every route accepts an optional `database` override (query param on
// GETs, body field on POSTs) so the client can switch databases
// without re-configuring — Manager.pool() keys by cfg.Database so the
// override naturally produces an independent connection pool.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/gateway/pg"
	"github.com/opendray/opendray/plugin"
)

// getPGConfig resolves the pg-browser plugin's config via
// effectiveConfig (which decrypts the password secret) and layers
// the safety rail defaults on top.
func (s *Server) getPGConfig(r *http.Request, pluginName string) (pg.Config, error) {
	info := s.plugins.ListInfo()
	var pi *plugin.ProviderInfo
	for i := range info {
		if info[i].Provider.Name == pluginName {
			pi = &info[i]
			break
		}
	}
	if pi == nil {
		return pg.Config{}, fmt.Errorf("pg plugin %q not found", pluginName)
	}
	if pi.Provider.Type != plugin.ProviderTypePanel || !pi.Enabled {
		return pg.Config{}, fmt.Errorf("pg plugin %q not enabled", pluginName)
	}
	cfg := s.effectiveConfig(r.Context(), pluginName, pi.Config)

	// ReadOnly default: TRUE. The plugin's configSchema default
	// agrees, but we re-enforce here so a manually-edited DB row
	// with readOnly unset doesn't silently grant write access.
	readOnly := true
	if v, ok := cfg["readOnly"]; ok {
		switch x := v.(type) {
		case bool:
			readOnly = x
		case string:
			readOnly = x != "false" && x != "0" && x != ""
		}
	}

	return pg.Config{
		Host:             stringVal(cfg, "host", "127.0.0.1"),
		Port:             intVal(cfg, "port", 5432),
		User:             stringVal(cfg, "user", ""),
		Password:         stringVal(cfg, "password", ""),
		Database:         stringVal(cfg, "database", "postgres"),
		SSLMode:          stringVal(cfg, "sslMode", "disable"),
		ReadOnly:         readOnly,
		StatementTimeout: time.Duration(intVal(cfg, "statementTimeoutSec", 30)) * time.Second,
		MaxRows:          intVal(cfg, "maxRows", 1000),
	}, nil
}

// overrideDatabase patches cfg.Database to db when db is non-empty.
// Kept as a helper (rather than inlined four times) so the semantics
// stay in one place: empty = use cfg default, non-empty = switch DB
// (produces a fresh pool via Manager.pool's cacheKey).
func overrideDatabase(cfg pg.Config, db string) pg.Config {
	if db != "" {
		cfg.Database = db
	}
	return cfg
}

// pgDatabases handles GET /api/pg/{plugin}/databases. Lists every
// user-visible database on the server (minus templates). Used by the
// Flutter database picker on first load.
func (s *Server) pgDatabases(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getPGConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	dbs, err := s.pg.ListDatabases(r.Context(), chi.URLParam(r, "plugin"), cfg)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if dbs == nil {
		dbs = []string{}
	}
	respondJSON(w, http.StatusOK, dbs)
}

// pgQuery handles POST /api/pg/{plugin}/query. Returns columns +
// rows + metadata. Destructive verbs (DROP / TRUNCATE / DELETE
// without WHERE) are rejected on the Query path regardless of
// read-only mode — they belong on Execute with explicit intent.
func (s *Server) pgQuery(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getPGConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		SQL      string `json:"sql"`
		Database string `json:"database"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	res, err := s.pg.Query(r.Context(), chi.URLParam(r, "plugin"),
		overrideDatabase(cfg, req.Database), req.SQL)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, res)
}

// pgExecute handles POST /api/pg/{plugin}/execute. Rejects any SQL
// that isn't a recognised write verb, so accidental SELECTs on this
// path return a clean error. Destructive verbs go through the same
// path — the Flutter client is responsible for the confirmation
// dialog; the server double-checks via pg.IsDestructiveVerb only
// for the audit log (future Phase 3 when /api/plugins/{name}/audit
// grows a "skipped confirm" column).
func (s *Server) pgExecute(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getPGConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var req struct {
		SQL      string `json:"sql"`
		Database string `json:"database"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	res, err := s.pg.Execute(r.Context(), chi.URLParam(r, "plugin"),
		overrideDatabase(cfg, req.Database), req.SQL)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, res)
}

func (s *Server) pgSchemas(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getPGConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	schemas, err := s.pg.ListSchemas(r.Context(), chi.URLParam(r, "plugin"),
		overrideDatabase(cfg, r.URL.Query().Get("database")))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if schemas == nil {
		schemas = []string{}
	}
	respondJSON(w, http.StatusOK, schemas)
}

func (s *Server) pgTables(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getPGConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	q := r.URL.Query()
	tables, err := s.pg.ListTables(r.Context(), chi.URLParam(r, "plugin"),
		overrideDatabase(cfg, q.Get("database")), q.Get("schema"))
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if tables == nil {
		tables = []pg.TableRow{}
	}
	respondJSON(w, http.StatusOK, tables)
}

func (s *Server) pgColumns(w http.ResponseWriter, r *http.Request) {
	cfg, err := s.getPGConfig(r, chi.URLParam(r, "plugin"))
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	q := r.URL.Query()
	schema := q.Get("schema")
	table := q.Get("table")
	if table == "" {
		respondError(w, http.StatusBadRequest, "table query param is required")
		return
	}
	cols, err := s.pg.DescribeColumns(r.Context(), chi.URLParam(r, "plugin"),
		overrideDatabase(cfg, q.Get("database")), schema, table)
	if err != nil {
		respondError(w, http.StatusBadRequest, err.Error())
		return
	}
	if cols == nil {
		cols = []pg.ColumnRow{}
	}
	respondJSON(w, http.StatusOK, cols)
}
