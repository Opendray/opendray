package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/opendray/opendray/plugin"
)

// ─── Types ───────────────────────────────────────────────────────────────────

// installOpts holds all validated inputs for the install command.
type installOpts struct {
	SrcArg    string // raw source argument (path or URL)
	AssumeYes bool   // --yes / -y: skip the consent prompt
	Help      bool   // --help / -h: print usage and exit
}

// httpClient is the minimum HTTP surface that runInstallWith needs.
// Allows a stubbed *http.Client in tests against httptest.NewServer.
type httpClient interface {
	Do(*http.Request) (*http.Response, error)
}

// cliConfig is the server URL + auth token source for the install command.
// Both fields have sane defaults when empty.
type cliConfig struct {
	ServerURL string // default http://127.0.0.1:8640
	Token     string // default empty (some servers allow unauthenticated local)
}

// ─── installResponse mirrors the gateway/plugins_install.go shape ──────────

// installResponseDTO is the JSON shape returned by POST /api/plugins/install
// (202 Accepted). Only the fields we act on are included; unknown fields are
// silently dropped (standard json.Decoder behaviour).
type installResponseDTO struct {
	Token   string               `json:"token"`
	Name    string               `json:"name"`
	Version string               `json:"version"`
	Perms   plugin.PermissionsV1 `json:"perms"`
}

// confirmResponseDTO is the JSON shape returned by POST /api/plugins/install/confirm.
type confirmResponseDTO struct {
	Installed bool   `json:"installed"`
	Name      string `json:"name"`
}

// apiErrorDTO is the structured error body {"code":"...","msg":"..."} returned
// by the gateway on non-2xx responses.
type apiErrorDTO struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// ─── parseInstallArgs ────────────────────────────────────────────────────────

// parseInstallArgs parses the argument slice for `opendray plugin install`.
// Flags (--yes / -y, --help / -h) may appear before or after the positional
// path-or-url argument. Any other flag-shaped token is an error.
//
// Returns:
//   - installOpts with Help=true when --help / -h is present (src may be empty).
//   - error containing "path-or-url required" when no positional arg is found.
func parseInstallArgs(args []string) (installOpts, error) {
	var opts installOpts
	for _, arg := range args {
		switch arg {
		case "--help", "-h":
			opts.Help = true
		case "--yes", "-y":
			opts.AssumeYes = true
		default:
			// Non-flag tokens are treated as the positional source argument.
			// Only the first one is recorded; extras are silently ignored (M1
			// simplification — a stricter parser can be added later).
			if opts.SrcArg == "" {
				opts.SrcArg = arg
			}
		}
	}

	// Help short-circuits before the required-arg check so that
	// `opendray plugin install --help` always exits 0.
	if opts.Help {
		return opts, nil
	}

	if opts.SrcArg == "" {
		return installOpts{}, fmt.Errorf("path-or-url required")
	}
	return opts, nil
}

// ─── loadCLIConfig ───────────────────────────────────────────────────────────

// loadCLIConfig reads OPENDRAY_SERVER_URL / OPENDRAY_TOKEN from the environment,
// then falls back to ~/.opendray/cli.toml if present (minimal TOML parsing —
// only `server_url` and `token` keys are recognised). In M1, env vars always
// win over the config file.
//
// Default server URL: http://127.0.0.1:8640
// Default token: empty (unauthenticated local install is permitted by some setups)
func loadCLIConfig() cliConfig {
	home, _ := os.UserHomeDir()
	return loadCLIConfigFrom(home)
}

// loadCLIConfigFrom is the injectable form of loadCLIConfig used in tests.
// homeDir overrides the home directory used to locate ~/.opendray/cli.toml.
// An empty homeDir skips the config-file fallback.
func loadCLIConfigFrom(homeDir string) cliConfig {
	cfg := cliConfig{
		ServerURL: "http://127.0.0.1:8640",
	}

	// Env vars are highest priority.
	if v := os.Getenv("OPENDRAY_SERVER_URL"); v != "" {
		cfg.ServerURL = v
	}
	if v := os.Getenv("OPENDRAY_TOKEN"); v != "" {
		cfg.Token = v
	}

	// If both env vars were set, we're done — env wins.
	if os.Getenv("OPENDRAY_SERVER_URL") != "" && os.Getenv("OPENDRAY_TOKEN") != "" {
		return cfg
	}

	// Fall back to <homeDir>/.opendray/cli.toml for whichever fields the env left unset.
	if homeDir == "" {
		return cfg
	}
	tomlPath := filepath.Join(homeDir, ".opendray", "cli.toml")
	data, err := os.ReadFile(tomlPath)
	if err != nil {
		return cfg // file absent or unreadable — use env/defaults
	}

	// Minimal TOML parsing: scan line by line for `key = "value"` patterns.
	// Full TOML parsing would require an external library; in M1 we only
	// need two keys, so a simple scanner is preferable to a new dependency.
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		// Strip surrounding quotes (both single and double).
		if len(val) >= 2 && (val[0] == '"' || val[0] == '\'') && val[0] == val[len(val)-1] {
			val = val[1 : len(val)-1]
		}

		switch key {
		case "server_url":
			// Env wins — only apply file value if env was not set.
			if os.Getenv("OPENDRAY_SERVER_URL") == "" {
				cfg.ServerURL = val
			}
		case "token":
			if os.Getenv("OPENDRAY_TOKEN") == "" {
				cfg.Token = val
			}
		}
	}

	return cfg
}

// ─── printInstallHelp ────────────────────────────────────────────────────────

// printInstallHelp writes the install subcommand usage to w.
func printInstallHelp(w io.Writer) {
	fmt.Fprint(w, `Usage: opendray plugin install <path-or-url> [--yes]

Install a plugin from a local path, https:// URL, or marketplace:// identifier.
The server prompts for consent; use --yes to skip interactively.

Arguments:
  <path-or-url>   Local path, https:// URL, or marketplace:// identifier.
                  Absolute paths are automatically prefixed with "local:".

Flags:
  --yes, -y    Skip the consent prompt
  --help, -h   Show this help

Environment:
  OPENDRAY_SERVER_URL   Server base URL (default http://127.0.0.1:8640)
  OPENDRAY_TOKEN        Bearer auth token

Exit codes:
  0  success
  1  user error (bad args, declined consent, missing token)
  2  runtime error (server unreachable, I/O failure)
`)
}

// ─── rewriteSrc ──────────────────────────────────────────────────────────────

// rewriteSrc transforms the raw source argument into the canonical form
// expected by POST /api/plugins/install:
//
//   - Absolute paths (starting with "/") → "local:<abs>"
//   - "https://", "marketplace://", "local:" prefixed → pass through
//   - Relative paths → pass through (server will reject if not local:)
func rewriteSrc(raw string) string {
	if filepath.IsAbs(raw) {
		return "local:" + raw
	}
	return raw
}

// ─── printPerms ──────────────────────────────────────────────────────────────

// printPerms renders the plugin's PermissionsV1 to stdout in a human-readable
// bullet list. Empty permissions produce a single "(no special permissions
// requested)" line.
func printPerms(w io.Writer, perms plugin.PermissionsV1) {
	var lines []string

	if len(perms.Exec) > 0 && string(perms.Exec) != "null" && string(perms.Exec) != "false" {
		lines = append(lines, fmt.Sprintf("  * Run commands: %s", string(perms.Exec)))
	}
	if len(perms.Fs) > 0 && string(perms.Fs) != "null" && string(perms.Fs) != "false" {
		lines = append(lines, fmt.Sprintf("  * File access: %s", string(perms.Fs)))
	}
	if len(perms.HTTP) > 0 && string(perms.HTTP) != "null" && string(perms.HTTP) != "false" {
		lines = append(lines, fmt.Sprintf("  * HTTP calls: %s", string(perms.HTTP)))
	}
	if perms.Storage {
		lines = append(lines, "  * Per-plugin storage")
	}
	if perms.Secret {
		lines = append(lines, "  * Secret storage")
	}
	if perms.Session != "" {
		lines = append(lines, fmt.Sprintf("  * Session access: %s", perms.Session))
	}
	if perms.Clipboard != "" {
		lines = append(lines, fmt.Sprintf("  * Clipboard: %s", perms.Clipboard))
	}
	if perms.Telegram {
		lines = append(lines, "  * Telegram integration")
	}
	if perms.Git != "" {
		lines = append(lines, fmt.Sprintf("  * Git access: %s", perms.Git))
	}
	if perms.LLM {
		lines = append(lines, "  * LLM access")
	}
	if len(perms.Events) > 0 {
		lines = append(lines, fmt.Sprintf("  * Events: %s", strings.Join(perms.Events, ", ")))
	}

	if len(lines) == 0 {
		fmt.Fprintln(w, "  (no special permissions requested)")
		return
	}
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
}

// ─── doPost ──────────────────────────────────────────────────────────────────

// doPost is a helper that builds and executes a POST request with a JSON body.
// It sets Content-Type: application/json and, if token is non-empty, the
// Authorization: Bearer <token> header.
func doPost(client httpClient, url, token string, body any) (*http.Response, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request body: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	return client.Do(req)
}

// ─── runInstall / runInstallWith ─────────────────────────────────────────────

// runInstall is the public entry point — it loads config from the environment
// and delegates to runInstallWith with the real http.DefaultClient and stdin.
func runInstall(args []string, stdout, stderr io.Writer) int {
	cfg := loadCLIConfig()
	return runInstallWith(args, stdout, stderr, http.DefaultClient, cfg, os.Stdin)
}

// runInstallWith is the injectable form used by tests.
//
// Argument semantics:
//   - args: everything after "install" (flags + path-or-url)
//   - stdout, stderr: output writers
//   - client: HTTP client (use http.DefaultClient in production)
//   - cfg: server URL + token (loaded from env/file by the caller)
//   - consentReader: io.Reader used to read the y/N prompt line
//
// Exit codes:
//
//	0  success
//	1  user error (bad args, declined consent, 4xx server error)
//	2  runtime error (network failure, 5xx server error)
func runInstallWith(args []string, stdout, stderr io.Writer, client httpClient, cfg cliConfig, consentReader io.Reader) int {
	// ── 1. Parse arguments ────────────────────────────────────────────────
	opts, err := parseInstallArgs(args)
	if err != nil {
		fmt.Fprintf(stderr, "error: %v\n\n", err)
		printInstallHelp(stderr)
		return 1
	}
	if opts.Help {
		printInstallHelp(stdout)
		return 0
	}

	// ── 2. Resolve server URL ─────────────────────────────────────────────
	serverURL := cfg.ServerURL
	if serverURL == "" {
		serverURL = "http://127.0.0.1:8640"
	}

	// ── 3. Rewrite source argument ────────────────────────────────────────
	src := rewriteSrc(opts.SrcArg)

	// ── 4. POST /api/plugins/install ──────────────────────────────────────
	installURL := serverURL + "/api/plugins/install"
	resp, err := doPost(client, installURL, cfg.Token, map[string]string{"src": src})
	if err != nil {
		fmt.Fprintf(stderr, "server unreachable: %v\n", err)
		return 2
	}
	defer resp.Body.Close()

	// Handle non-202 responses.
	if resp.StatusCode != http.StatusAccepted {
		return handleInstallError(resp, stderr)
	}

	// Decode the install response.
	var installResp installResponseDTO
	if err := json.NewDecoder(resp.Body).Decode(&installResp); err != nil {
		fmt.Fprintf(stderr, "error decoding install response: %v\n", err)
		return 2
	}

	// ── 5. Display plugin info + permissions ──────────────────────────────
	fmt.Fprintf(stdout, "\nPlugin: %s v%s\n", installResp.Name, installResp.Version)
	fmt.Fprintln(stdout, "Permissions requested:")
	printPerms(stdout, installResp.Perms)

	// ── 6. Consent prompt ─────────────────────────────────────────────────
	if !opts.AssumeYes {
		fmt.Fprint(stdout, "\nInstall this plugin? [y/N]: ")
		reader := bufio.NewReader(consentReader)
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			fmt.Fprintf(stderr, "error reading consent: %v\n", err)
			return 2
		}
		line = strings.TrimSpace(line)
		switch strings.ToLower(line) {
		case "y", "yes":
			// Confirmed — continue.
		default:
			fmt.Fprintln(stderr, "aborted.")
			return 1
		}
	}

	// ── 7. POST /api/plugins/install/confirm ──────────────────────────────
	confirmURL := serverURL + "/api/plugins/install/confirm"
	confirmResp, err := doPost(client, confirmURL, cfg.Token, map[string]string{"token": installResp.Token})
	if err != nil {
		fmt.Fprintf(stderr, "server unreachable: %v\n", err)
		return 2
	}
	defer confirmResp.Body.Close()

	if confirmResp.StatusCode != http.StatusOK {
		return handleInstallError(confirmResp, stderr)
	}

	// ── 8. Success ────────────────────────────────────────────────────────
	var cr confirmResponseDTO
	_ = json.NewDecoder(confirmResp.Body).Decode(&cr)
	name := cr.Name
	if name == "" {
		name = installResp.Name
	}
	fmt.Fprintf(stdout, "installed. (%s)\n", name)
	return 0
}

// ─── handleInstallError ───────────────────────────────────────────────────────

// handleInstallError reads a non-success HTTP response, parses the structured
// error body if possible, and writes a human-readable message to stderr.
// Returns 1 for 4xx errors and 2 for 5xx (and other non-2xx) errors.
func handleInstallError(resp *http.Response, stderr io.Writer) int {
	var apiErr apiErrorDTO
	_ = json.NewDecoder(resp.Body).Decode(&apiErr)

	code := apiErr.Code
	msg := apiErr.Msg
	if code == "" {
		code = http.StatusText(resp.StatusCode)
	}

	// Special-case 401: point the user at the env var.
	if resp.StatusCode == http.StatusUnauthorized {
		fmt.Fprintf(stderr,
			"install failed (HTTP %d): %s — %s\n"+
				"hint: set OPENDRAY_TOKEN to your server auth token\n",
			resp.StatusCode, code, msg)
		return 1
	}

	fmt.Fprintf(stderr, "install failed (HTTP %d): %s — %s\n", resp.StatusCode, code, msg)

	if resp.StatusCode >= 500 {
		return 2
	}
	return 1
}
