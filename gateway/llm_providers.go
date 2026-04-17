package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/linivek/ntc/kernel/store"
)

// llmProviderView is the public shape. We never leak the API key
// value itself — just the env var name the host reads at spawn time.
type llmProviderView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	DisplayName  string `json:"displayName"`
	ProviderType string `json:"providerType"`
	BaseURL      string `json:"baseUrl"`
	APIKeyEnv    string `json:"apiKeyEnv"`
	APIKeySet    bool   `json:"apiKeySet"` // whether the named env var is currently populated on the host
	Description  string `json:"description"`
	Enabled      bool   `json:"enabled"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

func llmProviderViewOf(p store.LLMProvider) llmProviderView {
	return llmProviderView{
		ID:           p.ID,
		Name:         p.Name,
		DisplayName:  p.DisplayName,
		ProviderType: p.ProviderType,
		BaseURL:      p.BaseURL,
		APIKeyEnv:    p.APIKeyEnv,
		APIKeySet:    p.APIKeyEnv != "" && os.Getenv(p.APIKeyEnv) != "",
		Description:  p.Description,
		Enabled:      p.Enabled,
		CreatedAt:    p.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    p.UpdatedAt.Format(time.RFC3339),
	}
}

var llmProviderNameRE = regexp.MustCompile(`^[a-z0-9_-]{1,48}$`)

func (s *Server) listLLMProviders(w http.ResponseWriter, r *http.Request) {
	rows, err := s.hub.DB().ListLLMProviders(r.Context())
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	out := make([]llmProviderView, 0, len(rows))
	for _, p := range rows {
		out = append(out, llmProviderViewOf(p))
	}
	respondJSON(w, http.StatusOK, map[string]any{"providers": out})
}

func (s *Server) getLLMProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.hub.DB().GetLLMProvider(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, llmProviderViewOf(p))
}

func (s *Server) createLLMProvider(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name         string `json:"name"`
		DisplayName  string `json:"displayName"`
		ProviderType string `json:"providerType"`
		BaseURL      string `json:"baseUrl"`
		APIKeyEnv    string `json:"apiKeyEnv"`
		Description  string `json:"description"`
		Enabled      *bool  `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	body.Name = strings.TrimSpace(body.Name)
	if !llmProviderNameRE.MatchString(body.Name) {
		respondError(w, http.StatusBadRequest, "name must be 1-48 chars of [a-z0-9_-]")
		return
	}
	if strings.TrimSpace(body.BaseURL) == "" {
		respondError(w, http.StatusBadRequest, "baseUrl is required")
		return
	}
	enabled := true
	if body.Enabled != nil {
		enabled = *body.Enabled
	}
	created, err := s.hub.DB().CreateLLMProvider(r.Context(), store.LLMProvider{
		Name:         body.Name,
		DisplayName:  body.DisplayName,
		ProviderType: strings.TrimSpace(body.ProviderType),
		BaseURL:      strings.TrimSpace(body.BaseURL),
		APIKeyEnv:    strings.TrimSpace(body.APIKeyEnv),
		Description:  body.Description,
		Enabled:      enabled,
	})
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusCreated, llmProviderViewOf(created))
}

func (s *Server) updateLLMProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	current, err := s.hub.DB().GetLLMProvider(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}
	var body struct {
		DisplayName  *string `json:"displayName"`
		ProviderType *string `json:"providerType"`
		BaseURL      *string `json:"baseUrl"`
		APIKeyEnv    *string `json:"apiKeyEnv"`
		Description  *string `json:"description"`
		Enabled      *bool   `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if body.DisplayName != nil {
		current.DisplayName = *body.DisplayName
	}
	if body.ProviderType != nil {
		current.ProviderType = strings.TrimSpace(*body.ProviderType)
	}
	if body.BaseURL != nil {
		current.BaseURL = strings.TrimSpace(*body.BaseURL)
	}
	if body.APIKeyEnv != nil {
		current.APIKeyEnv = strings.TrimSpace(*body.APIKeyEnv)
	}
	if body.Description != nil {
		current.Description = *body.Description
	}
	if body.Enabled != nil {
		current.Enabled = *body.Enabled
	}
	updated, err := s.hub.DB().UpdateLLMProvider(r.Context(), id, current)
	if err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, llmProviderViewOf(updated))
}

func (s *Server) toggleLLMProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respondError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if err := s.hub.DB().SetLLMProviderEnabled(r.Context(), id, body.Enabled); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{"enabled": body.Enabled})
}

func (s *Server) deleteLLMProvider(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.hub.DB().DeleteLLMProvider(r.Context(), id); err != nil {
		respondError(w, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// probeLLMProviderModels hits the upstream's /v1/models endpoint so the
// UI can offer a dropdown. Any error (timeout, 4xx, missing key,
// non-OpenAI-shaped response) is surfaced with 502 so the UI can fall
// back to a free-text model field.
func (s *Server) probeLLMProviderModels(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	p, err := s.hub.DB().GetLLMProvider(r.Context(), id)
	if err != nil {
		respondError(w, http.StatusNotFound, err.Error())
		return
	}

	models, err := probeModels(r.Context(), p)
	if err != nil {
		// Bad gateway with a human-readable reason — the Flutter
		// side special-cases this and flips to manual entry.
		respondError(w, http.StatusBadGateway, err.Error())
		return
	}
	respondJSON(w, http.StatusOK, map[string]any{
		"provider": p.Name,
		"models":   models,
	})
}

// probeModels does the actual HTTP call. Tolerant of a few common
// upstream quirks:
//   - strips one trailing /v1 from baseUrl if the user copied the
//     whole /v1/chat/completions path convention and then we double it
//   - accepts either {"data":[{"id":"..."}]} (OpenAI) or
//     {"models":[{"name":"..."}]} (Ollama native /api/tags shape), but
//     the latter only matches when baseUrl points at Ollama's root (we
//     don't synthesise /api/tags here — users set a real OpenAI base
//     URL like http://host:11434/v1 and we hit /v1/models).
func probeModels(ctx context.Context, p store.LLMProvider) ([]string, error) {
	base := strings.TrimRight(p.BaseURL, "/")
	url := base + "/models"

	reqCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if p.APIKeyEnv != "" {
		if key := os.Getenv(p.APIKeyEnv); key != "" {
			req.Header.Set("Authorization", "Bearer "+key)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("upstream unreachable: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return nil, fmt.Errorf("upstream %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}

	var out struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode models: %w", err)
	}

	ids := make([]string, 0, len(out.Data))
	for _, m := range out.Data {
		if strings.TrimSpace(m.ID) != "" {
			ids = append(ids, m.ID)
		}
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("upstream returned no models (wrong base URL?)")
	}
	return ids, nil
}
