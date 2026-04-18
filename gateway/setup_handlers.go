package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/opendray/opendray/kernel/config"
	"github.com/opendray/opendray/kernel/setup"
)

// setupHandlers owns the /api/setup/* endpoints. Wired by the gateway
// when the manager is non-nil (i.e. setup mode is active at boot).
type setupHandlers struct {
	mgr *setup.Manager
}

func newSetupHandlers(mgr *setup.Manager) *setupHandlers {
	return &setupHandlers{mgr: mgr}
}

// tokenGate requires X-Setup-Token or ?token= matching mgr.BootstrapToken.
// Public-by-necessity — the whole setup flow happens before any JWT or
// DB is available.
func (h *setupHandlers) tokenGate(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tok := r.Header.Get("X-Setup-Token")
		if tok == "" {
			tok = r.URL.Query().Get("token")
		}
		if !h.mgr.ValidateToken(tok) {
			respondError(w, http.StatusUnauthorized, "invalid setup token")
			return
		}
		next(w, r)
	}
}

// status — GET /api/setup/status
func (h *setupHandlers) status(w http.ResponseWriter, r *http.Request) {
	respondJSON(w, http.StatusOK, h.mgr.Status())
}

// envProbe — GET /api/setup/env
// Detects presence of Node, npm, claude, codex, gemini + reports OS info
// so the wizard's CLI step can render accurate enable/disable state.
func (h *setupHandlers) envProbe(w http.ResponseWriter, r *http.Request) {
	type tool struct {
		Installed bool   `json:"installed"`
		Version   string `json:"version,omitempty"`
		Path      string `json:"path,omitempty"`
	}
	probe := func(cmd string, versionArgs ...string) tool {
		path, err := exec.LookPath(cmd)
		if err != nil {
			return tool{Installed: false}
		}
		args := versionArgs
		if len(args) == 0 {
			args = []string{"--version"}
		}
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		out, _ := exec.CommandContext(ctx, cmd, args...).Output()
		v := strings.TrimSpace(string(out))
		// Keep the first line only — claude prints a banner.
		if nl := strings.IndexByte(v, '\n'); nl > 0 {
			v = v[:nl]
		}
		return tool{Installed: true, Version: v, Path: path}
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"os":     runtime.GOOS,
		"arch":   runtime.GOARCH,
		"node":   probe("node"),
		"npm":    probe("npm"),
		"clis": map[string]tool{
			"claude": probe("claude"),
			"codex":  probe("codex"),
			"gemini": probe("gemini"),
		},
	})
}

// dbTest — POST /api/setup/db/test
// Validates that the supplied external DB params accept a connection.
// Doesn't touch the draft — caller decides whether to call /db/commit
// after a green test.
func (h *setupHandlers) dbTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
		Name     string `json:"name"`
		SSLMode  string `json:"sslmode"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Port == 0 {
		req.Port = 5432
	}
	if req.SSLMode == "" {
		req.SSLMode = "disable"
	}
	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		req.Host, req.Port, req.User, req.Password, req.Name, req.SSLMode)

	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		respondError(w, http.StatusBadRequest, "connection failed: "+err.Error())
		return
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		respondError(w, http.StatusBadRequest, "ping failed: "+err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// dbCommit — POST /api/setup/db/commit
// Stores the chosen DB config in the draft. No migrations here — those
// happen at finalize time when the full boot stack stands up.
func (h *setupHandlers) dbCommit(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode     string              `json:"mode"`
		External *externalDBRequest  `json:"external,omitempty"`
		Embedded *embeddedDBRequest  `json:"embedded,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	switch req.Mode {
	case "embedded":
		h.mgr.UpdateDraft(func(c *config.Config) {
			c.DB.Mode = "embedded"
			if req.Embedded != nil {
				if req.Embedded.DataDir != "" {
					c.DB.Embedded.DataDir = req.Embedded.DataDir
				}
				if req.Embedded.CacheDir != "" {
					c.DB.Embedded.CacheDir = req.Embedded.CacheDir
				}
				if req.Embedded.Port != 0 {
					c.DB.Embedded.Port = req.Embedded.Port
				}
			}
		})
		h.mgr.MarkDBTested("embedded")
	case "external":
		if req.External == nil {
			respondError(w, http.StatusBadRequest, "external block required when mode=external")
			return
		}
		h.mgr.UpdateDraft(func(c *config.Config) {
			c.DB.Mode = "external"
			c.DB.External = config.ExternalDB{
				Host:     req.External.Host,
				Port:     req.External.Port,
				User:     req.External.User,
				Password: req.External.Password,
				Name:     req.External.Name,
				SSLMode:  req.External.SSLMode,
			}
			if c.DB.External.Port == 0 {
				c.DB.External.Port = 5432
			}
			if c.DB.External.SSLMode == "" {
				c.DB.External.SSLMode = "disable"
			}
		})
		h.mgr.MarkDBTested("external")
	default:
		respondError(w, http.StatusBadRequest, "mode must be embedded or external")
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type externalDBRequest struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
	SSLMode  string `json:"sslmode"`
}

type embeddedDBRequest struct {
	DataDir  string `json:"dataDir"`
	CacheDir string `json:"cacheDir"`
	Port     int    `json:"port"`
}

// adminSet — POST /api/setup/admin
// Stages the admin username/password in the draft. The password is
// persisted to the DB's admin_auth row at finalize time (after the DB
// is actually up).
func (h *setupHandlers) adminSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	if req.Username == "" {
		req.Username = "admin"
	}
	if len(req.Password) < 8 {
		respondError(w, http.StatusBadRequest, "password must be at least 8 characters")
		return
	}
	h.mgr.UpdateDraft(func(c *config.Config) {
		c.Auth.AdminBootstrapUsername = req.Username
		c.Auth.AdminBootstrapPassword = req.Password
	})
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// jwtSet — POST /api/setup/jwt
// Either {mode:"auto"} (server generates) or {mode:"custom", value:"…"}.
func (h *setupHandlers) jwtSet(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Mode  string `json:"mode"`
		Value string `json:"value"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request")
		return
	}
	var secret string
	switch req.Mode {
	case "auto", "":
		s, err := config.GenerateJWTSecret()
		if err != nil {
			respondError(w, http.StatusInternalServerError, "generate: "+err.Error())
			return
		}
		secret = s
	case "custom":
		if len(req.Value) < 32 {
			respondError(w, http.StatusBadRequest, "custom JWT secret must be at least 32 characters")
			return
		}
		secret = req.Value
	default:
		respondError(w, http.StatusBadRequest, "mode must be auto or custom")
		return
	}
	h.mgr.UpdateDraft(func(c *config.Config) {
		c.Auth.JWTSecret = secret
	})
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// finalize — POST /api/setup/finalize
// Writes the draft to config.toml and triggers the gateway's in-place
// router rebuild. The Flutter wizard then navigates the user to /login.
func (h *setupHandlers) finalize(w http.ResponseWriter, r *http.Request) {
	// Sanity — every required field must be staged.
	draft := h.mgr.Draft()
	if draft.Auth.JWTSecret == "" {
		respondError(w, http.StatusBadRequest, "JWT secret missing — POST /api/setup/jwt first")
		return
	}
	if draft.Auth.AdminBootstrapPassword == "" {
		respondError(w, http.StatusBadRequest, "admin password missing — POST /api/setup/admin first")
		return
	}
	tested, _ := h.mgr.DBTested()
	if !tested {
		respondError(w, http.StatusBadRequest, "database not committed — POST /api/setup/db/commit first")
		return
	}

	// Always stamp the completion time so operators can tell "this config
	// came from the wizard" vs. "someone hand-wrote it".
	h.mgr.UpdateDraft(func(c *config.Config) {
		c.SetupCompletedAt = time.Now().UTC().Format(time.RFC3339)
		c.SchemaVersion = config.SchemaVersion
	})

	if err := h.mgr.Finalize(); err != nil {
		respondError(w, http.StatusInternalServerError, "finalize: "+err.Error())
		return
	}
	// setup.Manager.Finalize() invoked onFinish — main.go is already
	// tearing down the setup-mode server and reloading config. The client
	// should retry for 2-5s until /api/auth/status answers on the normal
	// stack.
	respondJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// setupStatusString is shown from the 503 body when the setup-mode
// middleware blocks a non-setup /api/* call, so the user knows where
// to go.
func setupStatusString() string {
	return "setup required — open /setup (bootstrap token in stderr)"
}
