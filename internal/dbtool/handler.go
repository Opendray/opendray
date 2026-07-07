package dbtool

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray-v2/internal/integration"
)

// Dbtool scopes for integration keys. Browsing (connections list, schema
// introspection, table data, read statements) needs ScopeDBRead; row CRUD
// and write statements need ScopeDBWrite. Admins pass either. Connection
// management (create/update/delete/test) stays admin-only regardless of
// scope — an integration must never be able to point opendray at a new
// host (the SSRF-shaped surface); it can only use operator-approved
// connections, further fenced by each connection's read_only flag.
const (
	ScopeDBRead  = "db:read"
	ScopeDBWrite = "db:write"
)

// Handlers exposes the dbtool subsystem over HTTP under /dbtool/*.
// Mount under the dual-auth route group (admin OR integration key).
type Handlers struct {
	svc *Service
	log *slog.Logger
}

// NewHandlers builds the HTTP wrapper around svc.
func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "dbtool.http")}
}

// requireScope allows admins, and integration keys holding `scope`.
func (h *Handlers) requireScope(scope string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, ok := integration.CurrentPrincipal(r.Context())
			if !ok {
				writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
				return
			}
			if p.Kind == integration.KindAdmin || integration.HasScope(p.Scopes, scope) {
				next.ServeHTTP(w, r)
				return
			}
			writeError(w, http.StatusForbidden, fmt.Errorf("requires admin or the %q scope", scope))
		})
	}
}

// requireAdmin allows only admin principals — connection management.
func (h *Handlers) requireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p, ok := integration.CurrentPrincipal(r.Context()); !ok || p.Kind != integration.KindAdmin {
			writeError(w, http.StatusForbidden, errors.New("requires admin"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// requireConnCwd enforces per-project isolation for non-admin principals:
// the target connection must belong to the cwd the caller is bound to.
// The auto-attached opendray-dbtool MCP always sends its spawn cwd as the
// `cwd` query parameter, so an agent session driving project A can only
// reach connections registered under project A — it cannot enumerate or
// touch another project's database by guessing an id. Admin principals
// (the web UI) bypass so the operator can browse across projects.
//
// This is the honest-path boundary; it is NOT a cryptographic one — an
// agent that extracts the injected system key from its rendered MCP
// config could still call the REST API directly with a forged cwd. That
// residual is identical to the memory MCP's shared-key model; a
// per-project key would close it (tracked as a follow-up).
//
// Returns false (and writes the response) when access is denied.
func (h *Handlers) requireConnCwd(w http.ResponseWriter, r *http.Request) bool {
	p, _ := integration.CurrentPrincipal(r.Context())
	if p.Kind == integration.KindAdmin {
		return true
	}
	cwd := r.URL.Query().Get("cwd")
	if cwd == "" {
		writeError(w, http.StatusForbidden,
			errors.New("dbtool: cwd query parameter is required for non-admin callers"))
		return false
	}
	conn, err := h.svc.GetConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return false
	}
	if conn.Cwd != cwd {
		// Do not reveal that the id exists under another project.
		writeError(w, http.StatusNotFound, ErrNotFound)
		return false
	}
	return true
}

// Mount registers all /dbtool/* routes on r. r should already carry the
// dual-auth middleware (admin OR integration).
func (h *Handlers) Mount(r chi.Router) {
	r.Route("/dbtool", func(r chi.Router) {
		// Connection management — admin only.
		r.With(h.requireAdmin).Post("/connections", h.createConnection)
		r.With(h.requireAdmin).Post("/connections/test", h.testParams)
		r.With(h.requireAdmin).Patch("/connections/{id}", h.updateConnection)
		r.With(h.requireAdmin).Delete("/connections/{id}", h.deleteConnection)
		r.With(h.requireAdmin).Post("/connections/{id}/test", h.testConnection)

		// Browsing + read statements — db:read.
		r.With(h.requireScope(ScopeDBRead)).Get("/connections", h.listConnections)
		r.With(h.requireScope(ScopeDBRead)).Get("/connections/{id}/schemas", h.schemas)
		r.With(h.requireScope(ScopeDBRead)).Get("/connections/{id}/schemas/{schema}/tables", h.tables)
		r.With(h.requireScope(ScopeDBRead)).Get("/connections/{id}/schemas/{schema}/tables/{table}/meta", h.tableMeta)
		r.With(h.requireScope(ScopeDBRead)).Post("/connections/{id}/table-data", h.tableData)
		// query is read-gated at the route; write statements additionally
		// require write authorization, resolved per-principal inside.
		r.With(h.requireScope(ScopeDBRead)).Post("/connections/{id}/query", h.query)

		// Row CRUD — db:write.
		r.With(h.requireScope(ScopeDBWrite)).Post("/connections/{id}/rows/insert", h.insertRow)
		r.With(h.requireScope(ScopeDBWrite)).Post("/connections/{id}/rows/update", h.updateRow)
		r.With(h.requireScope(ScopeDBWrite)).Post("/connections/{id}/rows/delete", h.deleteRows)
	})
}

func (h *Handlers) createConnection(w http.ResponseWriter, r *http.Request) {
	var p CreateParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	c, err := h.svc.CreateConnection(r.Context(), p)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

// updatePayload is the PATCH body: pointer fields distinguish "absent"
// from "set to zero value". Password is write-only: absent (or empty)
// keeps the stored secret.
type updatePayload struct {
	Name     *string         `json:"name"`
	Host     *string         `json:"host"`
	Port     *int            `json:"port"`
	DBName   *string         `json:"db_name"`
	Username *string         `json:"username"`
	Password *string         `json:"password"`
	SSLMode  *string         `json:"ssl_mode"`
	ReadOnly *bool           `json:"read_only"`
	Options  *map[string]any `json:"options"`
}

func (h *Handlers) updateConnection(w http.ResponseWriter, r *http.Request) {
	var p updatePayload
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	patch := UpdatePatch{
		Name: p.Name, Host: p.Host, Port: p.Port, DBName: p.DBName,
		Username: p.Username, SSLMode: p.SSLMode, ReadOnly: p.ReadOnly,
		Options: p.Options,
	}
	// Empty-string password in a PATCH means "unchanged" (the edit form
	// leaves the field blank); there is no "clear password" operation.
	if p.Password != nil && *p.Password != "" {
		patch.Password = p.Password
	}
	c, err := h.svc.UpdateConnection(r.Context(), chi.URLParam(r, "id"), patch)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (h *Handlers) deleteConnection(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteConnection(r.Context(), chi.URLParam(r, "id")); err != nil {
		writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) testParams(w http.ResponseWriter, r *http.Request) {
	var p CreateParams
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	writeJSON(w, http.StatusOK, h.svc.TestParams(r.Context(), p))
}

func (h *Handlers) testConnection(w http.ResponseWriter, r *http.Request) {
	res, err := h.svc.TestConnection(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *Handlers) listConnections(w http.ResponseWriter, r *http.Request) {
	cwd := r.URL.Query().Get("cwd")
	// Non-admin callers may only enumerate their own project's
	// connections — never the whole cross-project registry. Admin (web
	// UI) may omit cwd to list everything.
	if p, _ := integration.CurrentPrincipal(r.Context()); p.Kind != integration.KindAdmin && cwd == "" {
		writeError(w, http.StatusForbidden,
			errors.New("dbtool: cwd query parameter is required for non-admin callers"))
		return
	}
	conns, err := h.svc.ListConnections(r.Context(), cwd)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if conns == nil {
		conns = []Connection{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"connections": conns})
}

func (h *Handlers) schemas(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	schemas, err := h.svc.Schemas(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if schemas == nil {
		schemas = []Schema{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"schemas": schemas})
}

func (h *Handlers) tables(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	tables, err := h.svc.Tables(r.Context(), chi.URLParam(r, "id"), pathParam(r, "schema"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	if tables == nil {
		tables = []Table{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"tables": tables})
}

func (h *Handlers) tableMeta(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	meta, err := h.svc.TableMeta(r.Context(), chi.URLParam(r, "id"),
		pathParam(r, "schema"), pathParam(r, "table"))
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (h *Handlers) tableData(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	var req TableDataReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	rs, err := h.svc.TableData(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rs)
}

func (h *Handlers) query(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	var req QueryReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	p, ok := integration.CurrentPrincipal(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	allowWrite := p.Kind == integration.KindAdmin || integration.HasScope(p.Scopes, ScopeDBWrite)
	rs, err := h.svc.Query(r.Context(), chi.URLParam(r, "id"), req, allowWrite)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rs)
}

func (h *Handlers) insertRow(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	var req RowInsertReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	rs, err := h.svc.InsertRow(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, rs)
}

func (h *Handlers) updateRow(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	var req RowUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	n, err := h.svc.UpdateRow(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows_affected": n})
}

func (h *Handlers) deleteRows(w http.ResponseWriter, r *http.Request) {
	if !h.requireConnCwd(w, r) {
		return
	}
	var req RowDeleteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid JSON: %w", err))
		return
	}
	n, err := h.svc.DeleteRows(r.Context(), chi.URLParam(r, "id"), req)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"rows_affected": n})
}

// pathParam URL-decodes a chi path parameter (schema/table names may
// contain characters the frontend percent-encodes).
func pathParam(r *http.Request, name string) string {
	v := chi.URLParam(r, name)
	if dec, err := url.PathUnescape(v); err == nil {
		return dec
	}
	return v
}

// writeServiceError maps package sentinel errors onto HTTP statuses.
// Anything else — including target-database errors, which the console
// must surface verbatim — is a 400 rather than a 500: the gateway did
// its job, the statement or payload didn't.
func writeServiceError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, ErrDuplicateName):
		writeError(w, http.StatusConflict, err)
	case errors.Is(err, ErrReadOnlyConnection), errors.Is(err, ErrWriteScope):
		writeError(w, http.StatusForbidden, err)
	default:
		writeError(w, http.StatusBadRequest, err)
	}
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}

func writeError(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
}
