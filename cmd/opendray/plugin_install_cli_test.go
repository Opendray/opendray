package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/opendray/opendray/plugin"
)

// ─── helpers ──────────────────────────────────────────────────────────────────

// makeInstallResponse returns a minimal installResponse JSON body.
func makeInstallResponse(token, name string, perms plugin.PermissionsV1) []byte {
	b, _ := json.Marshal(map[string]any{
		"token":   token,
		"name":    name,
		"version": "1.0.0",
		"perms":   perms,
	})
	return b
}

// makeConfirmResponse returns a minimal confirmResponse JSON body.
func makeConfirmResponse(installed bool, name string) []byte {
	b, _ := json.Marshal(map[string]any{
		"installed": installed,
		"name":      name,
	})
	return b
}

// makeErrorBody returns a JSON error body {"code":"...","msg":"..."}.
func makeErrorBody(code, msg string) []byte {
	b, _ := json.Marshal(map[string]string{"code": code, "msg": msg})
	return b
}

// runWithServer is a test helper that wires up runInstallWith with a given
// httptest server URL, a bytes.Reader for consentReader, and captures
// stdout/stderr. Returns exit code, stdout string, stderr string.
func runWithServer(t *testing.T, args []string, serverURL string, consentInput string) (int, string, string) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cfg := cliConfig{ServerURL: serverURL, Token: ""}
	reader := strings.NewReader(consentInput)
	code := runInstallWith(args, &stdout, &stderr, http.DefaultClient, cfg, reader)
	return code, stdout.String(), stderr.String()
}

// ─── TestParseInstallArgs ─────────────────────────────────────────────────────

// TestParseInstallArgs_HappyPath verifies that a bare path is parsed correctly.
func TestParseInstallArgs_HappyPath(t *testing.T) {
	opts, err := parseInstallArgs([]string{"./x"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if opts.SrcArg != "./x" {
		t.Errorf("SrcArg: want %q, got %q", "./x", opts.SrcArg)
	}
	if opts.AssumeYes {
		t.Error("expected AssumeYes=false")
	}
	if opts.Help {
		t.Error("expected Help=false")
	}
}

// TestParseInstallArgs_WithYes verifies --yes flag before and after the path.
func TestParseInstallArgs_WithYes(t *testing.T) {
	cases := [][]string{
		{"--yes", "./x"},
		{"./x", "--yes"},
		{"-y", "./x"},
		{"./x", "-y"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			opts, err := parseInstallArgs(args)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !opts.AssumeYes {
				t.Error("expected AssumeYes=true")
			}
			if opts.SrcArg != "./x" {
				t.Errorf("SrcArg: want %q, got %q", "./x", opts.SrcArg)
			}
		})
	}
}

// TestParseInstallArgs_Help verifies that --help sets Help=true and returns nil error.
func TestParseInstallArgs_Help(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			opts, err := parseInstallArgs([]string{flag})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !opts.Help {
				t.Error("expected Help=true")
			}
		})
	}
}

// TestParseInstallArgs_Missing verifies that empty args returns an error.
func TestParseInstallArgs_Missing(t *testing.T) {
	_, err := parseInstallArgs([]string{})
	if err == nil {
		t.Fatal("expected error for empty args")
	}
	if !strings.Contains(err.Error(), "path-or-url required") {
		t.Errorf("expected error to mention 'path-or-url required', got %q", err.Error())
	}
}

// ─── TestRunInstall_HappyPath ─────────────────────────────────────────────────

// TestRunInstall_HappyPath verifies the full two-leg install flow with --yes.
// The fake server asserts both endpoints are called in order.
func TestRunInstall_HappyPath(t *testing.T) {
	var installCalled, confirmCalled bool
	var installCallOrder, confirmCallOrder int
	callCount := 0

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			installCalled = true
			installCallOrder = callCount
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok123", "test-plug", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			confirmCalled = true
			confirmCallOrder = callCount
			// Assert the token from the first call is forwarded.
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			if body["token"] != "tok123" {
				t.Errorf("confirm: want token=%q, got %q", "tok123", body["token"])
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "test-plug"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	code, stdout, _ := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stdout=%q", code, stdout)
	}
	if !installCalled {
		t.Error("install endpoint not called")
	}
	if !confirmCalled {
		t.Error("confirm endpoint not called")
	}
	if installCallOrder >= confirmCallOrder {
		t.Error("install must be called before confirm")
	}
	if !strings.Contains(stdout, "installed.") {
		t.Errorf("expected stdout to contain \"installed.\", got %q", stdout)
	}
}

// ─── TestRunInstall_ConsentDeclined ──────────────────────────────────────────

// TestRunInstall_ConsentDeclined verifies that declining the prompt exits 1
// and does NOT call the confirm endpoint.
func TestRunInstall_ConsentDeclined(t *testing.T) {
	confirmCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-decline", "test-plug", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			confirmCalled = true
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "test-plug"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	code, _, stderr := runWithServer(t, []string{"./x"}, ts.URL, "n\n")

	if code != 1 {
		t.Errorf("expected exit code 1 (declined), got %d", code)
	}
	if confirmCalled {
		t.Error("confirm endpoint must NOT be called when consent is declined")
	}
	if !strings.Contains(stderr, "aborted.") {
		t.Errorf("expected stderr to contain \"aborted.\", got %q", stderr)
	}
}

// ─── TestRunInstall_ConsentAccepted ──────────────────────────────────────────

// TestRunInstall_ConsentAccepted verifies that "y\n" input accepts the prompt.
func TestRunInstall_ConsentAccepted(t *testing.T) {
	confirmCalled := false

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-accept", "test-plug", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			confirmCalled = true
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "test-plug"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	code, stdout, _ := runWithServer(t, []string{"./x"}, ts.URL, "y\n")

	if code != 0 {
		t.Errorf("expected exit code 0 (accepted), got %d; stdout=%q", code, stdout)
	}
	if !confirmCalled {
		t.Error("confirm endpoint must be called when consent is accepted")
	}
	if !strings.Contains(stdout, "installed.") {
		t.Errorf("expected stdout to contain \"installed.\", got %q", stdout)
	}
}

// ─── TestRunInstall_PermsDisplay ─────────────────────────────────────────────

// TestRunInstall_PermsDisplay verifies that perms fields are rendered on stdout.
func TestRunInstall_PermsDisplay(t *testing.T) {
	perms := plugin.PermissionsV1{
		Exec:    json.RawMessage(`["git *"]`),
		Storage: true,
	}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-perms", "test-plug", perms))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "test-plug"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	code, stdout, _ := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout, "Run commands") {
		t.Errorf("expected stdout to mention exec/\"Run commands\", got %q", stdout)
	}
	if !strings.Contains(stdout, "Per-plugin storage") {
		t.Errorf("expected stdout to mention storage/\"Per-plugin storage\", got %q", stdout)
	}
}

// ─── TestRunInstall_EmptyPerms ────────────────────────────────────────────────

// TestRunInstall_EmptyPerms verifies the "no special permissions" fallback.
func TestRunInstall_EmptyPerms(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-empty", "test-plug", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "test-plug"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	_, stdout, _ := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if !strings.Contains(stdout, "no special permissions requested") {
		t.Errorf("expected stdout to contain \"no special permissions requested\", got %q", stdout)
	}
}

// ─── TestRunInstall_InstallErrorStatus ───────────────────────────────────────

// TestRunInstall_InstallErrorStatus verifies 400 returns exit 1 with code+msg.
func TestRunInstall_InstallErrorStatus(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(makeErrorBody("EBADMANIFEST", "bad"))
	}))
	defer ts.Close()

	code, _, stderr := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if code != 1 {
		t.Errorf("expected exit code 1 for 400 response, got %d", code)
	}
	if !strings.Contains(stderr, "EBADMANIFEST") {
		t.Errorf("expected stderr to contain error code %q, got %q", "EBADMANIFEST", stderr)
	}
	if !strings.Contains(stderr, "bad") {
		t.Errorf("expected stderr to contain msg %q, got %q", "bad", stderr)
	}
}

// ─── TestRunInstall_ServerError ───────────────────────────────────────────────

// TestRunInstall_ServerError verifies 500 returns exit 2.
func TestRunInstall_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(makeErrorBody("ESTAGEFAIL", "internal"))
	}))
	defer ts.Close()

	code, _, stderr := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if code != 2 {
		t.Errorf("expected exit code 2 for 5xx response, got %d; stderr=%q", code, stderr)
	}
}

// ─── TestRunInstall_NetworkError ─────────────────────────────────────────────

// TestRunInstall_NetworkError verifies a connection refusal returns exit 2
// with "unreachable" on stderr.
func TestRunInstall_NetworkError(t *testing.T) {
	// Start a server and immediately close it so any TCP connection fails.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close() // close now — client will get connection refused

	code, _, stderr := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if code != 2 {
		t.Errorf("expected exit code 2 for network error, got %d; stderr=%q", code, stderr)
	}
	if !strings.Contains(stderr, "unreachable") {
		t.Errorf("expected stderr to contain \"unreachable\", got %q", stderr)
	}
}

// ─── TestRunInstall_AbsolutePathRewrite ──────────────────────────────────────

// TestRunInstall_AbsolutePathRewrite verifies absolute paths are rewritten to local: scheme.
func TestRunInstall_AbsolutePathRewrite(t *testing.T) {
	var gotSrc string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			gotSrc = body["src"]
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-abs", "x", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "x"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	runWithServer(t, []string{"--yes", "/tmp/foo"}, ts.URL, "")

	if gotSrc != "local:/tmp/foo" {
		t.Errorf("expected src to be rewritten to %q, got %q", "local:/tmp/foo", gotSrc)
	}
}

// ─── TestRunInstall_LocalSchemePassthrough ────────────────────────────────────

// TestRunInstall_LocalSchemePassthrough verifies local: prefix is passed through unchanged.
func TestRunInstall_LocalSchemePassthrough(t *testing.T) {
	var gotSrc string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			gotSrc = body["src"]
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-local", "x", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "x"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	runWithServer(t, []string{"--yes", "local:/foo"}, ts.URL, "")

	if gotSrc != "local:/foo" {
		t.Errorf("expected src to be unchanged %q, got %q", "local:/foo", gotSrc)
	}
}

// ─── TestRunInstall_HTTPSPassthrough ─────────────────────────────────────────

// TestRunInstall_HTTPSPassthrough verifies https:// src is passed through unchanged.
func TestRunInstall_HTTPSPassthrough(t *testing.T) {
	var gotSrc string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			gotSrc = body["src"]
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-https", "x", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "x"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	const httpsURL = "https://x.com/p.zip"
	runWithServer(t, []string{"--yes", httpsURL}, ts.URL, "")

	if gotSrc != httpsURL {
		t.Errorf("expected src to be unchanged %q, got %q", httpsURL, gotSrc)
	}
}

// ─── TestRunInstall_HelpFlag ──────────────────────────────────────────────────

// TestRunInstall_HelpFlag verifies --help prints usage and exits 0.
func TestRunInstall_HelpFlag(t *testing.T) {
	for _, flag := range []string{"--help", "-h"} {
		t.Run(flag, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			cfg := cliConfig{ServerURL: "http://127.0.0.1:8640"}
			code := runInstallWith([]string{flag}, &stdout, &stderr, http.DefaultClient, cfg, strings.NewReader(""))

			if code != 0 {
				t.Errorf("expected exit code 0 for %s, got %d", flag, code)
			}
			if !strings.Contains(stdout.String(), "Usage:") {
				t.Errorf("expected stdout to contain \"Usage:\", got %q", stdout.String())
			}
		})
	}
}

// ─── TestPrintInstallHelp ─────────────────────────────────────────────────────

// TestPrintInstallHelp verifies the help text contains required strings.
func TestPrintInstallHelp(t *testing.T) {
	var buf bytes.Buffer
	printInstallHelp(&buf)
	out := buf.String()

	for _, want := range []string{"--yes", "OPENDRAY_SERVER_URL"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected help to contain %q, got:\n%s", want, out)
		}
	}
}

// ─── TestLoadCLIConfig ────────────────────────────────────────────────────────

// TestLoadCLIConfig_EnvOverride verifies env vars take precedence.
func TestLoadCLIConfig_EnvOverride(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "http://test:9999")
	t.Setenv("OPENDRAY_TOKEN", "my-token")

	cfg := loadCLIConfig()

	if cfg.ServerURL != "http://test:9999" {
		t.Errorf("ServerURL: want %q, got %q", "http://test:9999", cfg.ServerURL)
	}
	if cfg.Token != "my-token" {
		t.Errorf("Token: want %q, got %q", "my-token", cfg.Token)
	}
}

// TestLoadCLIConfig_Defaults verifies the fallback default server URL.
func TestLoadCLIConfig_Defaults(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "")
	t.Setenv("OPENDRAY_TOKEN", "")

	cfg := loadCLIConfig()

	if cfg.ServerURL != "http://127.0.0.1:8640" {
		t.Errorf("default ServerURL: want %q, got %q", "http://127.0.0.1:8640", cfg.ServerURL)
	}
}

// ─── TestRunInstall_AuthBearerHeader ─────────────────────────────────────────

// TestRunInstall_AuthBearerHeader verifies the Authorization header is sent when a token is set.
func TestRunInstall_AuthBearerHeader(t *testing.T) {
	var gotAuth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			gotAuth = r.Header.Get("Authorization")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-auth", "x", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "x"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	var stdout, stderr bytes.Buffer
	cfg := cliConfig{ServerURL: ts.URL, Token: "secret-token"}
	code := runInstallWith([]string{"--yes", "./x"}, &stdout, &stderr, http.DefaultClient, cfg, strings.NewReader(""))

	if code != 0 {
		t.Errorf("expected exit code 0, got %d; stderr=%q", code, stderr.String())
	}
	if gotAuth != "Bearer secret-token" {
		t.Errorf("expected Authorization header %q, got %q", "Bearer secret-token", gotAuth)
	}
}

// ─── TestRunInstall_401_ClearError ───────────────────────────────────────────

// TestRunInstall_401_ClearError verifies that a 401 response prints a helpful message
// pointing at the OPENDRAY_TOKEN env var.
func TestRunInstall_401_ClearError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(makeErrorBody("EUNAUTH", "unauthorized"))
	}))
	defer ts.Close()

	code, _, stderr := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if code != 1 {
		t.Errorf("expected exit code 1 for 401, got %d", code)
	}
	if !strings.Contains(stderr, "OPENDRAY_TOKEN") {
		t.Errorf("expected stderr to mention OPENDRAY_TOKEN, got %q", stderr)
	}
}

// ─── TestRunInstall_ContentTypeJSON ──────────────────────────────────────────

// TestRunInstall_ContentTypeJSON verifies the Content-Type header is application/json.
func TestRunInstall_ContentTypeJSON(t *testing.T) {
	var gotContentType string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			gotContentType = r.Header.Get("Content-Type")
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-ct", "x", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "x"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if !strings.Contains(gotContentType, "application/json") {
		t.Errorf("expected Content-Type application/json, got %q", gotContentType)
	}
}

// ─── TestRunInstall_MarketplacePassthrough ────────────────────────────────────

// TestRunInstall_MarketplacePassthrough verifies marketplace:// src is passed through unchanged.
func TestRunInstall_MarketplacePassthrough(t *testing.T) {
	var gotSrc string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			gotSrc = body["src"]
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-mp", "x", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "x"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	const mpURL = "marketplace://example/plugin"
	runWithServer(t, []string{"--yes", mpURL}, ts.URL, "")

	if gotSrc != mpURL {
		t.Errorf("expected src to be unchanged %q, got %q", mpURL, gotSrc)
	}
}

// ─── TestRunInstall_ConsentYESVariants ───────────────────────────────────────

// TestRunInstall_ConsentYESVariants verifies all accepted yes-variants confirm install.
func TestRunInstall_ConsentYESVariants(t *testing.T) {
	for _, input := range []string{"y\n", "Y\n", "yes\n", "YES\n"} {
		t.Run(fmt.Sprintf("input=%q", input), func(t *testing.T) {
			confirmCalled := false
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					w.Write(makeInstallResponse("tok-yes", "x", plugin.PermissionsV1{}))

				case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
					confirmCalled = true
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusOK)
					w.Write(makeConfirmResponse(true, "x"))

				default:
					http.Error(w, "unexpected route", http.StatusBadRequest)
				}
			}))
			defer ts.Close()

			runWithServer(t, []string{"./x"}, ts.URL, input)

			if !confirmCalled {
				t.Errorf("input %q should accept consent, but confirm was not called", input)
			}
		})
	}
}

// ─── TestRunInstall_ConsentNoVariants ────────────────────────────────────────

// TestRunInstall_ConsentNoVariants verifies declined variants (empty, n, N) abort.
func TestRunInstall_ConsentNoVariants(t *testing.T) {
	for _, input := range []string{"\n", "n\n", "N\n", "no\n"} {
		t.Run(fmt.Sprintf("input=%q", input), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
					w.Header().Set("Content-Type", "application/json")
					w.WriteHeader(http.StatusAccepted)
					w.Write(makeInstallResponse("tok-no", "x", plugin.PermissionsV1{}))
				default:
					http.Error(w, "confirm must not be called", http.StatusInternalServerError)
				}
			}))
			defer ts.Close()

			code, _, stderr := runWithServer(t, []string{"./x"}, ts.URL, input)

			if code != 1 {
				t.Errorf("input %q should decline consent, got code %d", input, code)
			}
			if !strings.Contains(stderr, "aborted.") {
				t.Errorf("input %q: expected \"aborted.\" in stderr, got %q", input, stderr)
			}
		})
	}
}

// ─── TestRunInstall_PluginNameInOutput ────────────────────────────────────────

// TestRunInstall_PluginNameInOutput verifies the plugin name from the install
// response appears somewhere in the stdout banner.
func TestRunInstall_PluginNameInOutput(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusAccepted)
			w.Write(makeInstallResponse("tok-name", "my-cool-plugin", plugin.PermissionsV1{}))

		case r.Method == http.MethodPost && r.URL.Path == "/api/plugins/install/confirm":
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write(makeConfirmResponse(true, "my-cool-plugin"))

		default:
			http.Error(w, "unexpected route", http.StatusBadRequest)
		}
	}))
	defer ts.Close()

	_, stdout, _ := runWithServer(t, []string{"--yes", "./x"}, ts.URL, "")

	if !strings.Contains(stdout, "my-cool-plugin") {
		t.Errorf("expected stdout to contain plugin name \"my-cool-plugin\", got %q", stdout)
	}
}

// ─── TestLoadCLIConfigFrom_TOMLFile ──────────────────────────────────────────

// TestLoadCLIConfigFrom_TOMLFile verifies that cli.toml is parsed when env vars are absent.
func TestLoadCLIConfigFrom_TOMLFile(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "")
	t.Setenv("OPENDRAY_TOKEN", "")

	// Create a temporary home directory with a .opendray/cli.toml file.
	home := t.TempDir()
	opendrayDir := home + "/.opendray"
	if err := os.MkdirAll(opendrayDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	toml := `# OpenDray CLI config
server_url = "http://toml-server:1234"
token = "toml-token"
`
	if err := os.WriteFile(opendrayDir+"/cli.toml", []byte(toml), 0o600); err != nil {
		t.Fatalf("write cli.toml: %v", err)
	}

	cfg := loadCLIConfigFrom(home)

	if cfg.ServerURL != "http://toml-server:1234" {
		t.Errorf("ServerURL from TOML: want %q, got %q", "http://toml-server:1234", cfg.ServerURL)
	}
	if cfg.Token != "toml-token" {
		t.Errorf("Token from TOML: want %q, got %q", "toml-token", cfg.Token)
	}
}

// TestLoadCLIConfigFrom_EnvWinsOverTOML verifies env vars take precedence over cli.toml.
func TestLoadCLIConfigFrom_EnvWinsOverTOML(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "http://env-server:9999")
	t.Setenv("OPENDRAY_TOKEN", "env-token")

	// Create a TOML file with different values.
	home := t.TempDir()
	opendrayDir := home + "/.opendray"
	if err := os.MkdirAll(opendrayDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	toml := `server_url = "http://toml-server:1234"
token = "toml-token"
`
	if err := os.WriteFile(opendrayDir+"/cli.toml", []byte(toml), 0o600); err != nil {
		t.Fatalf("write cli.toml: %v", err)
	}

	cfg := loadCLIConfigFrom(home)

	if cfg.ServerURL != "http://env-server:9999" {
		t.Errorf("ServerURL: env should win over TOML, got %q", cfg.ServerURL)
	}
	if cfg.Token != "env-token" {
		t.Errorf("Token: env should win over TOML, got %q", cfg.Token)
	}
}

// TestLoadCLIConfigFrom_MissingTOML verifies defaults are returned when cli.toml is absent.
func TestLoadCLIConfigFrom_MissingTOML(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "")
	t.Setenv("OPENDRAY_TOKEN", "")

	home := t.TempDir() // no .opendray/cli.toml

	cfg := loadCLIConfigFrom(home)

	if cfg.ServerURL != "http://127.0.0.1:8640" {
		t.Errorf("default ServerURL: want %q, got %q", "http://127.0.0.1:8640", cfg.ServerURL)
	}
	if cfg.Token != "" {
		t.Errorf("default Token should be empty, got %q", cfg.Token)
	}
}

// TestLoadCLIConfigFrom_EmptyHome verifies empty homeDir skips file lookup.
func TestLoadCLIConfigFrom_EmptyHome(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "")
	t.Setenv("OPENDRAY_TOKEN", "")

	cfg := loadCLIConfigFrom("")

	if cfg.ServerURL != "http://127.0.0.1:8640" {
		t.Errorf("default ServerURL with empty home: want %q, got %q", "http://127.0.0.1:8640", cfg.ServerURL)
	}
}

// TestLoadCLIConfigFrom_TOMLWithSingleQuotes verifies single-quoted values are parsed.
func TestLoadCLIConfigFrom_TOMLWithSingleQuotes(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "")
	t.Setenv("OPENDRAY_TOKEN", "")

	home := t.TempDir()
	opendrayDir := home + "/.opendray"
	os.MkdirAll(opendrayDir, 0o700)
	toml := "server_url = 'http://single-quoted:5555'\ntoken = 'single-token'\n"
	os.WriteFile(opendrayDir+"/cli.toml", []byte(toml), 0o600)

	cfg := loadCLIConfigFrom(home)

	if cfg.ServerURL != "http://single-quoted:5555" {
		t.Errorf("single-quoted server_url: want %q, got %q", "http://single-quoted:5555", cfg.ServerURL)
	}
}

// TestLoadCLIConfigFrom_TOMLPartialEnv verifies partial env override (only token set).
func TestLoadCLIConfigFrom_TOMLPartialEnv(t *testing.T) {
	t.Setenv("OPENDRAY_SERVER_URL", "")
	t.Setenv("OPENDRAY_TOKEN", "env-only-token")

	home := t.TempDir()
	opendrayDir := home + "/.opendray"
	os.MkdirAll(opendrayDir, 0o700)
	toml := "server_url = \"http://toml-url:1111\"\ntoken = \"toml-token\"\n"
	os.WriteFile(opendrayDir+"/cli.toml", []byte(toml), 0o600)

	cfg := loadCLIConfigFrom(home)

	// server_url should come from TOML (env not set); token from env.
	if cfg.ServerURL != "http://toml-url:1111" {
		t.Errorf("ServerURL should come from TOML when env empty: got %q", cfg.ServerURL)
	}
	if cfg.Token != "env-only-token" {
		t.Errorf("Token should come from env: got %q", cfg.Token)
	}
}

// ─── TestPrintPerms ──────────────────────────────────────────────────────────

// TestPrintPerms_AllFields verifies that all perm fields are rendered.
func TestPrintPerms_AllFields(t *testing.T) {
	perms := plugin.PermissionsV1{
		Fs:        json.RawMessage(`["~/workspace/*"]`),
		Exec:      json.RawMessage(`["git *","npm *"]`),
		HTTP:      json.RawMessage(`["https://api.example.com/*"]`),
		Session:   "read",
		Storage:   true,
		Secret:    true,
		Clipboard: "read",
		Telegram:  true,
		Git:       "read",
		LLM:       true,
		Events:    []string{"onStartup", "onFileChange"},
	}

	var buf bytes.Buffer
	printPerms(&buf, perms)
	out := buf.String()

	checks := []string{
		"Run commands",
		"File access",
		"HTTP calls",
		"Per-plugin storage",
		"Secret storage",
		"Session access",
		"Clipboard",
		"Telegram",
		"Git access",
		"LLM access",
		"Events",
		"onStartup",
	}
	for _, want := range checks {
		if !strings.Contains(out, want) {
			t.Errorf("printPerms: expected output to contain %q\nGot:\n%s", want, out)
		}
	}
}

// TestPrintPerms_NullExec verifies null/false exec produces no exec line.
func TestPrintPerms_NullExec(t *testing.T) {
	perms := plugin.PermissionsV1{
		Exec: json.RawMessage(`null`),
	}
	var buf bytes.Buffer
	printPerms(&buf, perms)
	out := buf.String()

	if strings.Contains(out, "Run commands") {
		t.Errorf("expected no exec line for null exec, got %q", out)
	}
	if !strings.Contains(out, "no special permissions") {
		t.Errorf("expected 'no special permissions' for null exec, got %q", out)
	}
}

// TestPrintPerms_FalseExec verifies false exec produces no exec line.
func TestPrintPerms_FalseExec(t *testing.T) {
	perms := plugin.PermissionsV1{
		Exec: json.RawMessage(`false`),
	}
	var buf bytes.Buffer
	printPerms(&buf, perms)
	out := buf.String()

	if strings.Contains(out, "Run commands") {
		t.Errorf("expected no exec line for false exec, got %q", out)
	}
}

// ─── Ensure unused imports are referenced ─────────────────────────────────────

// Make sure bufio is used in the production code (referenced here to avoid
// import cycle if we declare a consentReader helper). The test uses
// strings.NewReader as io.Reader directly; bufio.NewReader wrapping happens
// inside runInstallWith.
var _ = bufio.NewReader
var _ = io.Discard
