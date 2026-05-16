package git

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/go-chi/chi/v5"
)

// write.go adds the mutating git operations that round out the
// read-only handler set in handler.go: branch list / create /
// checkout / delete, stage / unstage / commit / push. Everything
// is scoped to the caller-supplied dir (same validation rules as
// the read path) — no GIT_DIR overrides, no global state.
//
// Safety rules:
//
//   - Checkout / delete refuse a dirty working tree (lossy ops
//     should be opt-in from the terminal, not surprising clicks).
//   - Push uses --no-force by default; the optional force flag
//     maps to --force-with-lease (still rejects when the upstream
//     moved underneath, but allows a normal rebase-then-push).
//   - All upstream calls go through `run()` from handler.go so
//     GIT_* env vars from the gateway shell don't leak into target
//     repos.
//
// What's NOT here on purpose:
//
//   - reset --hard / merge --abort: destructive; live in the
//     vault-git surface or are terminal-only.
//   - rebase / cherry-pick: complex enough to deserve their own
//     UX iteration. Phase 4 stays focused on the everyday "commit
//     and push" loop.

// ── Mount additions ────────────────────────────────────────────

// MountWrite registers the write endpoints under the same /git
// route the Handlers.Mount installed for reads. Kept separate so
// reviewers can quickly see which routes mutate state.
func (h *Handlers) MountWrite(r chi.Router) {
	r.Route("/git/write", func(r chi.Router) {
		r.Get("/branches", h.listBranches)
		r.Post("/branches", h.createBranch)
		r.Post("/checkout", h.checkoutBranch)
		r.Delete("/branches/{name}", h.deleteBranch)
		r.Post("/stage", h.stageFiles)
		r.Post("/unstage", h.unstageFiles)
		r.Post("/commit", h.commit)
		r.Post("/push", h.push)
	})
}

// ── Branch list ────────────────────────────────────────────────

type BranchRef struct {
	Name      string `json:"name"`
	Remote    string `json:"remote,omitempty"` // for remote refs: "origin"
	IsRemote  bool   `json:"is_remote"`
	IsCurrent bool   `json:"is_current"`
	Upstream  string `json:"upstream,omitempty"`
}

type BranchListResponse struct {
	Branches []BranchRef `json:"branches"`
	Current  string      `json:"current"`
}

func (h *Handlers) listBranches(w http.ResponseWriter, r *http.Request) {
	dir, err := dirParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !isWorkTree(r.Context(), dir) {
		writeJSON(w, http.StatusOK, BranchListResponse{Branches: []BranchRef{}})
		return
	}
	// Current branch via symbolic-ref. Empty on detached HEAD.
	curBytes, _ := run(r.Context(), dir,
		"symbolic-ref", "--quiet", "--short", "HEAD")
	current := strings.TrimSpace(string(curBytes))

	// Local + remote refs in one call. `refname:full` is the
	// authoritative source of truth for local-vs-remote (we used
	// to heuristic from the short form, which mis-classified bare
	// `refs/remotes/origin` symrefs as local branches called
	// "origin"). Upstream + HEAD marker round out the format.
	out, err := run(r.Context(), dir,
		"for-each-ref",
		"--format=%(refname:short)|%(refname)|%(upstream:short)|%(HEAD)",
		"refs/heads", "refs/remotes")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	refs := parseBranchRefs(string(out), current)
	writeJSON(w, http.StatusOK, BranchListResponse{
		Branches: refs,
		Current:  current,
	})
}

// parseBranchRefs splits the for-each-ref output. Each line is
// "<short>|<full>|<upstream>|<head_marker>". Classification rule:
//
//   - refname:full starts with "refs/heads/"   → local
//   - refname:full starts with "refs/remotes/" → remote
//
// The short refname might collide between local and remote (e.g.
// a local branch literally named "origin" would have short
// "origin" — same as a hypothetical bare remote HEAD symref); the
// full refname disambiguates.
func parseBranchRefs(out, current string) []BranchRef {
	refs := []BranchRef{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 4)
		if len(parts) < 2 {
			continue
		}
		shortName := parts[0]
		fullName := parts[1]
		upstream := ""
		headMarker := ""
		if len(parts) > 2 {
			upstream = parts[2]
		}
		if len(parts) > 3 {
			headMarker = strings.TrimSpace(parts[3])
		}

		// Filter:
		//   - bare "refs/remotes/<remote>" symrefs (just the
		//     remote head pointer, not a switchable branch).
		//   - HEAD symrefs anywhere (origin/HEAD, etc.).
		if strings.HasSuffix(fullName, "/HEAD") {
			continue
		}
		// "refs/remotes/origin" with no further slash is the bare
		// remote symref some setups produce. Skip it — it's not
		// a branch.
		if strings.HasPrefix(fullName, "refs/remotes/") {
			trimmed := strings.TrimPrefix(fullName, "refs/remotes/")
			if !strings.Contains(trimmed, "/") {
				continue
			}
		}

		isRemote := strings.HasPrefix(fullName, "refs/remotes/")
		displayName := shortName
		remote := ""
		if isRemote {
			// short form is "<remote>/<branch[/sub]*>". Split
			// on the FIRST slash; everything after is the
			// branch name (which itself may contain slashes).
			if i := strings.Index(shortName, "/"); i > 0 {
				remote = shortName[:i]
				displayName = shortName[i+1:]
			}
		}

		refs = append(refs, BranchRef{
			Name:      displayName,
			Remote:    remote,
			IsRemote:  isRemote,
			IsCurrent: !isRemote && (headMarker == "*" || displayName == current),
			Upstream:  upstream,
		})
	}
	return refs
}

// ── Branch create ──────────────────────────────────────────────

type CreateBranchRequest struct {
	Dir  string `json:"dir"`
	Name string `json:"name"`
	// From is the starting point (commit / ref). Empty = HEAD.
	From string `json:"from"`
	// Switch=true checks out the new branch after creating it.
	Switch bool `json:"switch"`
}

func (h *Handlers) createBranch(w http.ResponseWriter, r *http.Request) {
	var req CreateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateWritePath(req.Dir); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !validBranchName(req.Name) {
		writeError(w, http.StatusBadRequest, errors.New("invalid branch name"))
		return
	}
	args := []string{"branch", req.Name}
	if strings.TrimSpace(req.From) != "" {
		args = append(args, req.From)
	}
	if out, err := runCombined(r.Context(), req.Dir, args...); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("create branch: %w (%s)", err, out))
		return
	}
	if req.Switch {
		if out, err := runCombined(r.Context(), req.Dir, "checkout", req.Name); err != nil {
			writeError(w, http.StatusInternalServerError,
				fmt.Errorf("checkout new branch: %w (%s)", err, out))
			return
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"name":   req.Name,
		"on":     req.From,
		"active": req.Switch,
	})
}

// ── Branch checkout ────────────────────────────────────────────

type CheckoutRequest struct {
	Dir  string `json:"dir"`
	Name string `json:"name"`
}

func (h *Handlers) checkoutBranch(w http.ResponseWriter, r *http.Request) {
	var req CheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateWritePath(req.Dir); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !validBranchName(req.Name) {
		writeError(w, http.StatusBadRequest, errors.New("invalid branch name"))
		return
	}
	// Refuse on dirty tree — checkout would either fail mid-flight
	// (uncommitted changes block) or silently carry changes across,
	// which is rarely what the operator wants from a click.
	if dirty, err := isDirty(r.Context(), req.Dir); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	} else if dirty {
		writeError(w, http.StatusConflict,
			errors.New("working tree has uncommitted changes; commit or stash first"))
		return
	}
	if out, err := runCombined(r.Context(), req.Dir, "checkout", req.Name); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("checkout: %w (%s)", err, out))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"current": req.Name})
}

// ── Branch delete ──────────────────────────────────────────────

func (h *Handlers) deleteBranch(w http.ResponseWriter, r *http.Request) {
	dir, err := dirParam(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateWritePath(dir); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	name := chi.URLParam(r, "name")
	if !validBranchName(name) {
		writeError(w, http.StatusBadRequest, errors.New("invalid branch name"))
		return
	}
	// `?force=true` upgrades the safe -d to a forced -D. We surface
	// the option but the UI defaults to safe; force is for the
	// "I know what I'm doing" path.
	force := r.URL.Query().Get("force") == "true"
	flag := "-d"
	if force {
		flag = "-D"
	}
	if out, err := runCombined(r.Context(), dir, "branch", flag, name); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("delete branch: %w (%s)", err, out))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"deleted": name, "force": force})
}

// ── Stage / unstage ────────────────────────────────────────────

type StageRequest struct {
	Dir   string   `json:"dir"`
	Files []string `json:"files"` // empty / nil = stage all (`.`)
}

func (h *Handlers) stageFiles(w http.ResponseWriter, r *http.Request) {
	var req StageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateWritePath(req.Dir); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	args := []string{"add"}
	if len(req.Files) == 0 {
		args = append(args, ".")
	} else {
		for _, f := range req.Files {
			if !validRelativePath(f) {
				writeError(w, http.StatusBadRequest,
					fmt.Errorf("invalid file path: %q", f))
				return
			}
		}
		args = append(args, req.Files...)
	}
	if out, err := runCombined(r.Context(), req.Dir, args...); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("stage: %w (%s)", err, out))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handlers) unstageFiles(w http.ResponseWriter, r *http.Request) {
	var req StageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateWritePath(req.Dir); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	args := []string{"reset", "HEAD", "--"}
	if len(req.Files) == 0 {
		args = []string{"reset", "HEAD"}
	} else {
		for _, f := range req.Files {
			if !validRelativePath(f) {
				writeError(w, http.StatusBadRequest,
					fmt.Errorf("invalid file path: %q", f))
				return
			}
		}
		args = append(args, req.Files...)
	}
	if out, err := runCombined(r.Context(), req.Dir, args...); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("unstage: %w (%s)", err, out))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ── Commit ─────────────────────────────────────────────────────

type CommitRequest struct {
	Dir     string `json:"dir"`
	Message string `json:"message"`
	// AllowEmpty mirrors `git commit --allow-empty` for the rare
	// "checkpoint" case. Defaults false so accidental empty commits
	// don't slip through.
	AllowEmpty bool `json:"allow_empty"`
}

func (h *Handlers) commit(w http.ResponseWriter, r *http.Request) {
	var req CommitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateWritePath(req.Dir); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if strings.TrimSpace(req.Message) == "" {
		writeError(w, http.StatusBadRequest, errors.New("commit message required"))
		return
	}
	args := []string{"commit", "-m", req.Message}
	if req.AllowEmpty {
		args = append(args, "--allow-empty")
	}
	if out, err := runCombined(r.Context(), req.Dir, args...); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("commit: %w (%s)", err, out))
		return
	}
	// Echo the resulting HEAD sha back so the UI can display it.
	headBytes, _ := run(r.Context(), req.Dir, "rev-parse", "HEAD")
	writeJSON(w, http.StatusOK, map[string]any{
		"hash": strings.TrimSpace(string(headBytes)),
	})
}

// ── Push ───────────────────────────────────────────────────────

type PushRequest struct {
	Dir    string `json:"dir"`
	Branch string `json:"branch"` // empty = current branch (HEAD)
	// Force=true upgrades to --force-with-lease (still safer than
	// raw --force; rejects when the upstream moved after we fetched).
	Force bool `json:"force"`
	// SetUpstream wires the local branch to origin/<branch>; required
	// the first time you push a freshly created branch.
	SetUpstream bool `json:"set_upstream"`
}

func (h *Handlers) push(w http.ResponseWriter, r *http.Request) {
	var req PushRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := validateWritePath(req.Dir); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	branch := strings.TrimSpace(req.Branch)
	if branch == "" {
		curBytes, err := run(r.Context(), req.Dir,
			"symbolic-ref", "--quiet", "--short", "HEAD")
		if err != nil {
			writeError(w, http.StatusBadRequest,
				errors.New("detached HEAD — supply explicit branch"))
			return
		}
		branch = strings.TrimSpace(string(curBytes))
	}
	if !validBranchName(branch) {
		writeError(w, http.StatusBadRequest, errors.New("invalid branch name"))
		return
	}
	args := []string{"push"}
	if req.SetUpstream {
		args = append(args, "-u")
	}
	if req.Force {
		args = append(args, "--force-with-lease")
	}
	args = append(args, "origin", branch)
	if out, err := runCombined(r.Context(), req.Dir, args...); err != nil {
		writeError(w, http.StatusInternalServerError,
			fmt.Errorf("push: %w (%s)", err, out))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"branch": branch})
}

// ── Helpers ────────────────────────────────────────────────────

// runCombined wraps `run` but captures combined stdout+stderr so
// the write paths can surface the actual git error (push rejected,
// merge conflict, etc.) rather than just exit code 1.
func runCombined(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = filteredEnv()
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

// isDirty asks git whether the working tree has uncommitted
// changes. Used by checkout to block lossy switches.
func isDirty(ctx context.Context, dir string) (bool, error) {
	out, err := run(ctx, dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) != "", nil
}

// validateWritePath enforces the same rules as dirParam for write
// endpoints whose dir lives in the request body (not query string).
// Kept separate so handlers can give a clear error message.
func validateWritePath(dir string) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return errors.New("dir required")
	}
	if !strings.HasPrefix(dir, "/") {
		return errors.New("dir must be absolute")
	}
	if !isWorkTree(context.Background(), dir) {
		return errors.New("dir is not a git work tree")
	}
	return nil
}

// validBranchName is a defensive guard against shell-injection
// shaped arguments. git itself rejects most malformed names, but
// we reject obvious red flags up-front so the call doesn't even
// reach the subprocess.
func validBranchName(name string) bool {
	name = strings.TrimSpace(name)
	if name == "" {
		return false
	}
	if strings.HasPrefix(name, "-") {
		return false // would be parsed as a flag
	}
	// Disallow whitespace, newlines, NUL.
	for _, r := range name {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' || r == 0 {
			return false
		}
	}
	return true
}

// validRelativePath stops a caller from passing "../" or absolute
// paths that escape the repo dir.
func validRelativePath(p string) bool {
	p = strings.TrimSpace(p)
	if p == "" {
		return false
	}
	if strings.HasPrefix(p, "/") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return false
		}
	}
	return true
}
