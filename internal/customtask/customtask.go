// Package customtask serves user-defined tasks that the Inspector's
// Tasks tab merges with auto-discovered manifests. Tasks are either
// scoped to an absolute cwd or global (cwd=""), and click-to-run on
// the client just sends the command into the session's PTY — same
// transport as the Makefile / package.json paths.
package customtask

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("custom task not found")

type Task struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Command     string    `json:"command"`
	Description string    `json:"description,omitempty"`
	// Cwd="" means global (visible from any session). Otherwise must
	// be an absolute path; tasks list filters to entries where cwd
	// equals the requesting session's cwd.
	Cwd       string    `json:"cwd"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Cwd         string `json:"cwd"`
}

type UpdateRequest struct {
	Name        *string `json:"name,omitempty"`
	Command     *string `json:"command,omitempty"`
	Description *string `json:"description,omitempty"`
	Cwd         *string `json:"cwd,omitempty"`
}

type Service struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

func NewService(pool *pgxpool.Pool, log *slog.Logger) *Service {
	if log == nil {
		log = slog.Default()
	}
	return &Service{pool: pool, log: log.With("component", "customtask")}
}

const taskSelect = `
    SELECT id, name, command, description, cwd, created_at, updated_at
    FROM custom_tasks`

// List returns global tasks (cwd='') plus tasks scoped to the given
// cwd. Pass cwd="" to get only globals, or omit the cwd param to get
// every row (admin/management view from the Plugins page).
func (s *Service) List(ctx context.Context, cwd string, all bool) ([]Task, error) {
	var (
		rows pgx.Rows
		err  error
	)
	switch {
	case all:
		rows, err = s.pool.Query(ctx, taskSelect+` ORDER BY cwd, name`)
	case cwd == "":
		rows, err = s.pool.Query(ctx, taskSelect+` WHERE cwd='' ORDER BY name`)
	default:
		rows, err = s.pool.Query(ctx,
			taskSelect+` WHERE cwd='' OR cwd=$1 ORDER BY cwd, name`, cwd)
	}
	if err != nil {
		return nil, fmt.Errorf("list custom tasks: %w", err)
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		t, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (Task, error) {
	name := strings.TrimSpace(req.Name)
	cmd := strings.TrimSpace(req.Command)
	if name == "" {
		return Task{}, errors.New("name is required")
	}
	if cmd == "" {
		return Task{}, errors.New("command is required")
	}
	cwd := strings.TrimSpace(req.Cwd)
	if cwd != "" && !filepath.IsAbs(cwd) {
		return Task{}, errors.New("cwd must be absolute or empty (global)")
	}
	row := s.pool.QueryRow(ctx, `
        INSERT INTO custom_tasks (name, command, description, cwd)
        VALUES ($1, $2, $3, $4)
        RETURNING id, name, command, description, cwd, created_at, updated_at`,
		name, cmd, strings.TrimSpace(req.Description), cwd)
	t, err := scan(row)
	if err != nil {
		return Task{}, fmt.Errorf("insert custom task: %w", err)
	}
	return t, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (Task, error) {
	current, err := s.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	if req.Name != nil {
		current.Name = strings.TrimSpace(*req.Name)
		if current.Name == "" {
			return Task{}, errors.New("name is required")
		}
	}
	if req.Command != nil {
		current.Command = strings.TrimSpace(*req.Command)
		if current.Command == "" {
			return Task{}, errors.New("command is required")
		}
	}
	if req.Description != nil {
		current.Description = strings.TrimSpace(*req.Description)
	}
	if req.Cwd != nil {
		c := strings.TrimSpace(*req.Cwd)
		if c != "" && !filepath.IsAbs(c) {
			return Task{}, errors.New("cwd must be absolute or empty (global)")
		}
		current.Cwd = c
	}
	row := s.pool.QueryRow(ctx, `
        UPDATE custom_tasks
        SET name=$1, command=$2, description=$3, cwd=$4, updated_at=NOW()
        WHERE id=$5
        RETURNING id, name, command, description, cwd, created_at, updated_at`,
		current.Name, current.Command, current.Description, current.Cwd, id)
	t, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return t, err
}

func (s *Service) Get(ctx context.Context, id string) (Task, error) {
	row := s.pool.QueryRow(ctx, taskSelect+` WHERE id=$1`, id)
	t, err := scan(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Task{}, ErrNotFound
	}
	return t, err
}

func (s *Service) Delete(ctx context.Context, id string) error {
	res, err := s.pool.Exec(ctx, `DELETE FROM custom_tasks WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("delete custom task: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type rowScanner interface{ Scan(dest ...any) error }

func scan(row rowScanner) (Task, error) {
	var t Task
	if err := row.Scan(&t.ID, &t.Name, &t.Command, &t.Description, &t.Cwd,
		&t.CreatedAt, &t.UpdatedAt); err != nil {
		return Task{}, err
	}
	return t, nil
}

// ── HTTP handlers ───────────────────────────────────────────────

type Handlers struct {
	svc *Service
	log *slog.Logger
}

func NewHandlers(svc *Service, log *slog.Logger) *Handlers {
	if log == nil {
		log = slog.Default()
	}
	return &Handlers{svc: svc, log: log.With("component", "customtask.http")}
}

func (h *Handlers) Mount(r chi.Router) {
	r.Route("/custom-tasks", func(r chi.Router) {
		r.Get("/", h.list)
		r.Post("/", h.create)
		r.Route("/{id}", func(r chi.Router) {
			r.Put("/", h.update)
			r.Delete("/", h.del)
		})
	})
}

func (h *Handlers) list(w http.ResponseWriter, r *http.Request) {
	cwd := strings.TrimSpace(r.URL.Query().Get("cwd"))
	all := r.URL.Query().Get("all") == "1"
	tasks, err := h.svc.List(r.Context(), cwd, all)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (h *Handlers) create(w http.ResponseWriter, r *http.Request) {
	var req CreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	t, err := h.svc.Create(r.Context(), req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (h *Handlers) update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req UpdateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	t, err := h.svc.Update(r.Context(), id, req)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (h *Handlers) del(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.svc.Delete(r.Context(), id); err != nil {
		if errors.Is(err, ErrNotFound) {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
