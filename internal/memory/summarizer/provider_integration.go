package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// IntegrationConfig points the summarizer at an opendray-registered
// integration that implements the Summarizer Protocol described
// below. This unlocks "any local service" as a summarizer backend
// — the operator can run an in-LAN Python/Go/Node service that
// wraps ollama, LM Studio, an internal LLM gateway, or even a
// rule-based extractor without any LLM, and opendray treats it
// the same as a first-party provider.
//
// SUMMARIZER PROTOCOL (the integration must implement):
//
//	POST <base_url>/summarize
//	  body:    {"messages":[{"role","text","timestamp"}, ...],
//	            "scope": "session|project|global",
//	            "scope_key": "<cwd or session id or 'operator'>"}
//	  reply:   {"facts":[{"text","category","confidence"}, ...],
//	            "input_tokens": N,    (optional)
//	            "output_tokens": M}   (optional)
//
// Auth: the operator's IntegrationConfig.OutboundToken (when
// non-empty) is sent as Authorization: Bearer <token>. The
// integration verifies it; opendray doesn't try to enforce.
//
// Integration-level scopes ('memory:summarize') are documentation —
// the UI uses them to filter the dropdown, but opendray doesn't
// reject calls if the scope is missing. Operators wiring a
// summarizer service themselves know what they're doing.
type IntegrationConfig struct {
	IntegrationID string // "int_..."
	BaseURL       string // resolved at registry-build time from the integration row
	OutboundToken string // optional plaintext; sent as Bearer
	Name          string // operator-friendly display
}

const (
	integrationCallTimeout = 60 * time.Second
)

// IntegrationProvider speaks the opendray Summarizer Protocol over
// HTTP. The integration's exact backend (ollama, vLLM, custom
// rules, anything) is the operator's choice — the protocol is the
// contract.
type IntegrationProvider struct {
	cfg     IntegrationConfig
	client  *http.Client
	baseURL string
}

func NewIntegrationProvider(cfg IntegrationConfig) (*IntegrationProvider, error) {
	if cfg.IntegrationID == "" {
		return nil, errors.New("integration provider: IntegrationID required")
	}
	if cfg.BaseURL == "" {
		return nil, errors.New("integration provider: BaseURL required (resolved from integration row)")
	}
	base := strings.TrimRight(cfg.BaseURL, "/")
	if cfg.Name == "" {
		cfg.Name = "integration:" + cfg.IntegrationID
	}
	return &IntegrationProvider{
		cfg:     cfg,
		client:  &http.Client{Timeout: integrationCallTimeout + 5*time.Second},
		baseURL: base,
	}, nil
}

func (p *IntegrationProvider) Name() string { return p.cfg.Name }
func (p *IntegrationProvider) Kind() string { return "integration" }

// Available pings <base_url>/summarize with HEAD or
// /summarize/health (whichever the integration implements). To
// avoid mandating a particular health endpoint, we POST a tiny
// no-message body and accept "200 with empty facts" as healthy.
//
// Implementations that don't gracefully handle empty input can
// still pass by responding with any non-error status; we only
// fail the probe on transport errors / 4xx-5xx.
func (p *IntegrationProvider) Available(ctx context.Context) error {
	body := []byte(`{"messages":[],"scope":"global","scope_key":"healthcheck"}`)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/summarize", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.cfg.OutboundToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.OutboundToken)
	}
	res, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	defer res.Body.Close()
	switch res.StatusCode {
	case http.StatusOK, http.StatusAccepted, http.StatusNoContent:
		return nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return ErrAuthFailed
	default:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1024))
		return fmt.Errorf("%w: HTTP %d: %s", ErrUnreachable, res.StatusCode, body)
	}
}

// Summarize forwards the messages to the integration's /summarize
// endpoint and unmarshals the reply. The integration is free to
// run any backend — opendray doesn't care about its internals,
// only the protocol response shape.
func (p *IntegrationProvider) Summarize(ctx context.Context, msgs []Message) (SummarizeResult, error) {
	if len(msgs) == 0 {
		return SummarizeResult{}, ErrEmptyConversation
	}

	body := map[string]any{
		"messages": msgs,
		"scope":    "session", // engine fills meaningful values via context; integrations can ignore
		"scope_key": "transcript",
	}
	rawBody, err := json.Marshal(body)
	if err != nil {
		return SummarizeResult{}, fmt.Errorf("%w: marshal: %v", ErrInvalidResponse, err)
	}

	cctx, cancel := context.WithTimeout(ctx, integrationCallTimeout)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(cctx, http.MethodPost, p.baseURL+"/summarize", bytes.NewReader(rawBody))
	if err != nil {
		return SummarizeResult{}, fmt.Errorf("%w: build request: %v", ErrUnreachable, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if p.cfg.OutboundToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.OutboundToken)
	}

	res, err := p.client.Do(req)
	latency := time.Since(start)
	if err != nil {
		return SummarizeResult{Latency: latency}, fmt.Errorf("%w: %v", ErrUnreachable, err)
	}
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 64*1024))
	res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return SummarizeResult{Latency: latency, RawResponse: TruncateRaw(string(raw))},
			fmt.Errorf("%w: HTTP %d", ErrAuthFailed, res.StatusCode)
	default:
		return SummarizeResult{Latency: latency, RawResponse: TruncateRaw(string(raw))},
			fmt.Errorf("%w: HTTP %d: %s", ErrUnreachable, res.StatusCode, raw)
	}

	facts, in, out, parseErr := parseIntegrationResponse(raw)
	result := SummarizeResult{
		Facts:        facts,
		InputTokens:  in,
		OutputTokens: out,
		EstimatedUSD: 0, // integration backends aren't priced by opendray; operator owns cost accounting
		Latency:      latency,
		RawResponse:  TruncateRaw(string(raw)),
	}
	if parseErr != nil {
		return result, parseErr
	}
	return result, nil
}

// parseIntegrationResponse expects the simple Summarizer Protocol
// shape: {"facts":[...], "input_tokens": N, "output_tokens": M}.
// Token counts are optional — when missing we report 0 (no cost
// telemetry from the integration backend, which is fine since the
// operator owns billing for whatever the integration runs).
func parseIntegrationResponse(raw []byte) ([]Fact, int, int, error) {
	var envelope struct {
		Facts        []map[string]any `json:"facts"`
		InputTokens  int              `json:"input_tokens"`
		OutputTokens int              `json:"output_tokens"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, 0, 0, fmt.Errorf("%w: %v", ErrInvalidResponse, err)
	}
	rawFacts := make([]any, 0, len(envelope.Facts))
	for _, f := range envelope.Facts {
		rawFacts = append(rawFacts, f)
	}
	facts, _ := decodeFactsArray(rawFacts)
	return facts, envelope.InputTokens, envelope.OutputTokens, nil
}

// IntegrationLookup is the contract opendray's app wiring satisfies
// to bridge the summarizer registry into internal/integration.
// Defined here so the summarizer package doesn't import
// internal/integration directly (avoids cycle + clarifies surface).
type IntegrationLookup interface {
	// LookupBaseURL returns the integration's base_url + whether
	// it is enabled. Returns (_, false, nil) when the integration
	// row doesn't exist; (_, _, error) for transport problems.
	LookupBaseURL(ctx context.Context, integrationID string) (baseURL string, enabled bool, err error)
}
