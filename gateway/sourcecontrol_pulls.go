package gateway

// Pull-request HTTP handlers for the source-control plugin. Unlike
// the legacy git-forge panel, these routes take ?repo=owner/name as
// a query parameter so one forge instance can answer for any number
// of repositories rather than locking one in at config time — which
// was the core "one repo at a time" complaint driving the merge.

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opendray/opendray/gateway/forge"
)

// scPullsResolve bundles the "load forge config + validate repo +
// read PR number" preamble every PR handler repeats. Writes the
// error response directly and returns ok=false so callers can early-
// return without touching `err`.
//
// Side effect: every successful resolve bumps lastUsedAt on the
// matching saved-repo entry (no-op when the repo isn't saved). Fired
// in a detached goroutine so the PR request isn't penalised on kv
// write latency; the handler doesn't depend on the write succeeding.
func (s *Server) scPullsResolve(w http.ResponseWriter, r *http.Request, needNumber bool) (cfg forge.Config, number int, ok bool) {
	pluginName := chi.URLParam(r, "plugin")
	forgeID := chi.URLParam(r, "id")
	repo := strings.TrimSpace(r.URL.Query().Get("repo"))
	if repo == "" {
		respondError(w, http.StatusBadRequest, "repo query parameter is required (owner/name)")
		return forge.Config{}, 0, false
	}
	c, err := s.buildForgeConfig(r.Context(), pluginName, forgeID, repo)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return forge.Config{}, 0, false
	}
	if needNumber {
		raw := chi.URLParam(r, "number")
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			respondError(w, http.StatusBadRequest,
				fmt.Sprintf("pull number must be a positive integer (got %q)", raw))
			return forge.Config{}, 0, false
		}
		// fire-and-forget; use a fresh background ctx so the bump
		// survives the request's cancellation after we write the
		// response. 3-second budget is plenty for a tiny kv update.
		go func(pn, fid, rp string) {
			bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			s.bumpSavedRepoLastUsed(bgCtx, pn, fid, rp)
		}(pluginName, forgeID, repo)
		return c, n, true
	}
	go func(pn, fid, rp string) {
		bgCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		s.bumpSavedRepoLastUsed(bgCtx, pn, fid, rp)
	}(pluginName, forgeID, repo)
	return c, 0, true
}

func (s *Server) scPullsList(w http.ResponseWriter, r *http.Request) {
	cfg, _, ok := s.scPullsResolve(w, r, false)
	if !ok {
		return
	}
	q := r.URL.Query()
	state := forge.State(q.Get("state"))
	limit := 0
	if v := q.Get("limit"); v != "" {
		_, _ = fmt.Sscanf(v, "%d", &limit)
	}
	prs, err := forge.List(r.Context(), cfg, state, limit)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, prs)
}

func (s *Server) scPullDetail(w http.ResponseWriter, r *http.Request) {
	cfg, n, ok := s.scPullsResolve(w, r, true)
	if !ok {
		return
	}
	pr, err := forge.Detail(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, pr)
}

func (s *Server) scPullDiff(w http.ResponseWriter, r *http.Request) {
	cfg, n, ok := s.scPullsResolve(w, r, true)
	if !ok {
		return
	}
	files, err := forge.Diff(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, files)
}

func (s *Server) scPullComments(w http.ResponseWriter, r *http.Request) {
	cfg, n, ok := s.scPullsResolve(w, r, true)
	if !ok {
		return
	}
	cs, err := forge.Comments(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, cs)
}

func (s *Server) scPullReviews(w http.ResponseWriter, r *http.Request) {
	cfg, n, ok := s.scPullsResolve(w, r, true)
	if !ok {
		return
	}
	rs, err := forge.Reviews(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, rs)
}

func (s *Server) scPullReviewComments(w http.ResponseWriter, r *http.Request) {
	cfg, n, ok := s.scPullsResolve(w, r, true)
	if !ok {
		return
	}
	rcs, err := forge.ReviewComments(r.Context(), cfg, n)
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, rcs)
}

func (s *Server) scPullChecks(w http.ResponseWriter, r *http.Request) {
	cfg, n, ok := s.scPullsResolve(w, r, true)
	if !ok {
		return
	}
	// Empty headSHA lets forge.Checks resolve it via Detail() — saves
	// callers an explicit round-trip on the hot PR-detail page flow.
	crs, err := forge.Checks(r.Context(), cfg, n, "")
	if err != nil {
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, crs)
}
