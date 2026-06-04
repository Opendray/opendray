// macOS code-signing + privacy (TCC) helpers.
//
//	opendray doctor        cross-platform read-only health check; on macOS it
//	                       flags an unstable (ad-hoc) signature and a config
//	                       living in a TCC-protected folder.
//	opendray setup-macos   give this binary a STABLE per-machine self-signed
//	                       identity and re-sign it, so a one-time Full Disk
//	                       Access grant survives every future rebuild/update.
//
// Why this exists: a Go binary is ad-hoc signed, so its TCC identity is its
// content hash (cdhash) — every rebuild/update changes it, and macOS
// re-prompts (and the gateway can't read a config under ~/Documents until
// the operator clicks "Allow", blocking startup). Signing with a stable
// self-signed cert keys the grant to the identity instead of the cdhash, so
// it persists. No Apple Developer account required; everything is per-machine
// and local. Official releases can additionally be Developer ID-signed +
// notarized in CI (a separate, optional track) — that is NOT done here.
//
// Everything macOS-specific is guarded by runtime.GOOS so the subcommands
// are safe no-ops elsewhere; the file compiles on every platform.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Stable signing identity. The cert (Common Name) lives in the login
// keychain; the identifier is baked into the signature. Keeping BOTH
// constant across re-signs is what makes the TCC grant survive rebuilds.
const (
	signingCertCN     = "opendray-codesign"
	signingIdentifier = "online.opendray.gateway"
)

// ── opendray doctor ────────────────────────────────────────────────

func runDoctor(_ []string) int {
	fmt.Println("opendray doctor")
	fmt.Println(strings.Repeat("─", 40))

	home, _ := os.UserHomeDir()
	cfg := activeConfigPath()
	fmt.Printf("config:   %s\n", orNone(cfg))
	if cfg != "" && isTCCProtectedPath(home, cfg) {
		fmt.Println("  ⚠ config is inside a macOS privacy-protected folder (Documents/Desktop/Downloads/a")
		fmt.Println("    mounted volume). On launch the gateway can't read it until you grant access —")
		fmt.Println("    which is what blocks startup. Move it under ~/.opendray/, or grant Full Disk")
		fmt.Println("    Access via `opendray setup-macos`.")
	}

	if runtime.GOOS != "darwin" {
		fmt.Printf("platform: %s — macOS code-signing checks not applicable.\n", runtime.GOOS)
		return 0
	}

	bin, err := os.Executable()
	if err == nil {
		bin, _ = filepath.EvalSymlinks(bin)
	}
	fmt.Printf("binary:   %s\n", orNone(bin))

	out, _ := exec.Command("codesign", "-dvvv", bin).CombinedOutput()
	switch {
	case signatureIsAdhoc(string(out)):
		fmt.Println("signing:  ad-hoc (UNSTABLE) — every rebuild/update changes the binary's TCC")
		fmt.Println("          identity, so macOS re-prompts and a Full Disk Access grant won't stick.")
		fmt.Println("          Fix: run `opendray setup-macos` once.")
	case signatureHasIdentifier(string(out), signingIdentifier):
		fmt.Printf("signing:  stable (identifier %s) ✓ — TCC grants persist across rebuilds.\n", signingIdentifier)
	default:
		fmt.Println("signing:  signed, but not by opendray's stable identity. If macOS keeps")
		fmt.Println("          re-prompting on update, run `opendray setup-macos`.")
	}

	fmt.Println(strings.Repeat("─", 40))
	fmt.Println("Note: macOS won't let any tool read whether Full Disk Access is granted")
	fmt.Println("(the TCC database is SIP-protected) — grant it once in System Settings.")
	return 0
}

// ── opendray setup-macos ───────────────────────────────────────────

func runSetupMacos(args []string) int {
	fs := flag.NewFlagSet("setup-macos", flag.ExitOnError)
	resignOnly := fs.Bool("resign", false,
		"ensure the identity + re-sign silently, skipping the Full Disk Access guidance "+
			"(for build/install automation — the FDA grant is one-time and persists)")
	_ = fs.Parse(args)

	if runtime.GOOS != "darwin" {
		if !*resignOnly {
			fmt.Printf("`opendray setup-macos` is macOS-only; nothing to do on %s.\n", runtime.GOOS)
		}
		return 0
	}
	for _, tool := range []string{"codesign", "security"} {
		if _, err := exec.LookPath(tool); err != nil {
			fmt.Fprintf(os.Stderr, "✗ required tool %q not found in PATH.\n", tool)
			return 1
		}
	}

	bin, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "✗ can't locate this binary: %v\n", err)
		return 1
	}
	bin, _ = filepath.EvalSymlinks(bin)

	if !*resignOnly {
		fmt.Printf("Stabilising the code signature of:\n  %s\n\n", bin)
	}

	if !signingIdentityExists() {
		if !*resignOnly {
			fmt.Printf("Creating a one-time self-signed signing identity %q in your login keychain…\n", signingCertCN)
		}
		if err := createSigningIdentity(); err != nil {
			fmt.Fprintf(os.Stderr, "✗ couldn't create the signing identity automatically: %v\n", err)
			if !*resignOnly {
				fmt.Fprintln(os.Stderr)
				printManualCertSteps()
			}
			return 1
		}
		if !*resignOnly {
			fmt.Println("  identity created ✓")
		}
	} else if !*resignOnly {
		fmt.Printf("Reusing existing signing identity %q.\n", signingCertCN)
	}

	if err := resignBinary(bin); err != nil {
		fmt.Fprintf(os.Stderr, "✗ codesign failed: %v\n", err)
		return 1
	}
	if *resignOnly {
		// Quiet success line so build logs show it happened.
		fmt.Printf("opendray: re-signed %s with stable identity %s\n", bin, signingIdentifier)
		return 0
	}
	fmt.Println("Re-signed with the stable identity ✓")

	fmt.Println()
	fmt.Println("Done. One manual step remains (only needed once — it now survives rebuilds):")
	fmt.Println("  1. System Settings → Privacy & Security → Full Disk Access")
	fmt.Printf("  2. Add this binary:  %s\n", bin)
	fmt.Println("  3. Restart the gateway:  opendray restart")
	fmt.Println()
	fmt.Println("Opening the Full Disk Access settings pane…")
	_ = exec.Command("open", "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles").Run()
	return 0
}

// signingKeychainPath is a DEDICATED keychain opendray owns, with a known
// (empty) password. Using our own keychain — rather than the login one — is
// what makes the whole flow non-interactive: we can unlock it and run
// set-key-partition-list with the known password, so codesign never pops a
// "allow access to key" prompt. It carries only this self-signed code-
// signing key, nothing else.
func signingKeychainPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".opendray", "opendray-signing.keychain-db")
}

// signingIdentityExists reports whether our code-signing identity is already
// in opendray's signing keychain (so re-runs just re-sign). No -v: a self-
// signed cert is "invalid"/untrusted and hidden by the valid-only filter,
// yet codesign signs with it fine once the key is accessible.
func signingIdentityExists() bool {
	kc := signingKeychainPath()
	if kc == "" {
		return false
	}
	if _, err := os.Stat(kc); err != nil {
		return false
	}
	out, _ := exec.Command("security", "find-identity", "-p", "codesigning", kc).CombinedOutput()
	return strings.Contains(string(out), signingCertCN)
}

// createSigningIdentity mints a self-signed code-signing cert (openssl, via a
// config file so it works on LibreSSL too) and installs it into opendray's
// dedicated keychain so codesign can use it with zero prompts. The key never
// leaves this machine and grants no Apple trust.
func createSigningIdentity() error {
	kc := signingKeychainPath()
	if kc == "" {
		return fmt.Errorf("can't resolve home dir")
	}
	if err := os.MkdirAll(filepath.Dir(kc), 0o700); err != nil {
		return err
	}

	tmp, err := os.MkdirTemp("", "opendray-codesign-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	cfgPath := filepath.Join(tmp, "openssl.cnf")
	keyPath := filepath.Join(tmp, "key.pem")
	crtPath := filepath.Join(tmp, "cert.pem")
	p12Path := filepath.Join(tmp, "id.p12")

	cnf := "[req]\ndistinguished_name=dn\nx509_extensions=ext\nprompt=no\n" +
		"[dn]\nCN=" + signingCertCN + "\n" +
		"[ext]\nkeyUsage=critical,digitalSignature\n" +
		"extendedKeyUsage=critical,codeSigning\nbasicConstraints=critical,CA:FALSE\n"
	if err := os.WriteFile(cfgPath, []byte(cnf), 0o600); err != nil {
		return err
	}

	if out, err := exec.Command("openssl", "req", "-x509", "-newkey", "rsa:2048",
		"-nodes", "-keyout", keyPath, "-out", crtPath, "-days", "3650",
		"-config", cfgPath).CombinedOutput(); err != nil {
		return fmt.Errorf("openssl req: %v: %s", err, strings.TrimSpace(string(out)))
	}

	// PKCS12 export. OpenSSL 3.x defaults to AES-256/SHA-256 PKCS12 MACs
	// that Apple's Security framework can't import ("MAC verification
	// failed"); -legacy forces the SHA1/3DES algorithms it accepts.
	// LibreSSL (/usr/bin/openssl) and OpenSSL 1.x don't know -legacy, so
	// fall back to a plain export there (their default is already legacy).
	p12 := func(legacy bool) ([]byte, error) {
		args := []string{"pkcs12", "-export"}
		if legacy {
			args = append(args, "-legacy")
		}
		args = append(args, "-inkey", keyPath, "-in", crtPath,
			"-out", p12Path, "-passout", "pass:opendray", "-name", signingCertCN)
		return exec.Command("openssl", args...).CombinedOutput()
	}
	if _, err := p12(true); err != nil {
		if out2, err2 := p12(false); err2 != nil {
			return fmt.Errorf("openssl pkcs12: %v: %s", err2, strings.TrimSpace(string(out2)))
		}
	}

	// Create (idempotently) + unlock the dedicated keychain with an empty
	// password, and disable the inactivity auto-lock so re-signs after a
	// reboot stay non-interactive.
	if _, err := os.Stat(kc); err != nil {
		if out, err := exec.Command("security", "create-keychain", "-p", "", kc).CombinedOutput(); err != nil {
			return fmt.Errorf("create-keychain: %v: %s", err, strings.TrimSpace(string(out)))
		}
	}
	_ = exec.Command("security", "set-keychain-settings", kc).Run()
	if out, err := exec.Command("security", "unlock-keychain", "-p", "", kc).CombinedOutput(); err != nil {
		return fmt.Errorf("unlock-keychain: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if err := ensureKeychainInSearchList(kc); err != nil {
		return err
	}
	if out, err := exec.Command("security", "import", p12Path, "-k", kc,
		"-P", "opendray", "-T", "/usr/bin/codesign", "-A").CombinedOutput(); err != nil {
		return fmt.Errorf("security import: %v: %s", err, strings.TrimSpace(string(out)))
	}
	// Pre-authorise codesign (and apple tools) to use the key without a
	// prompt. Needs the keychain password — which we set to empty.
	if out, err := exec.Command("security", "set-key-partition-list",
		"-S", "apple-tool:,apple:", "-s", "-k", "", kc).CombinedOutput(); err != nil {
		return fmt.Errorf("set-key-partition-list: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// ensureKeychainInSearchList adds kc to the user keychain search list if
// absent, preserving every existing entry (so codesign/Security can find the
// identity). Additive only — never drops the login keychain.
func ensureKeychainInSearchList(kc string) error {
	out, err := exec.Command("security", "list-keychains", "-d", "user").Output()
	if err != nil {
		return fmt.Errorf("list-keychains: %w", err)
	}
	var list []string
	for _, ln := range strings.Split(string(out), "\n") {
		p := strings.TrimSpace(ln)
		p = strings.Trim(p, "\"")
		if p == "" {
			continue
		}
		if p == kc {
			return nil // already present
		}
		list = append(list, p)
	}
	list = append(list, kc)
	args := append([]string{"list-keychains", "-d", "user", "-s"}, list...)
	if out, err := exec.Command("security", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("set search list: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func resignBinary(bin string) error {
	kc := signingKeychainPath()
	// Unlock first: the dedicated keychain re-locks on reboot, and an
	// automation re-sign must stay non-interactive (empty password).
	_ = exec.Command("security", "unlock-keychain", "-p", "", kc).Run()
	cmd := exec.Command("codesign", "--force", "--options", "runtime",
		"--keychain", kc, "--sign", signingCertCN, "--identifier", signingIdentifier, bin)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func printManualCertSteps() {
	fmt.Fprintln(os.Stderr, "Create the signing identity by hand, then re-run `opendray setup-macos`:")
	fmt.Fprintln(os.Stderr, "  1. Open Keychain Access → menu: Keychain Access → Certificate Assistant →")
	fmt.Fprintln(os.Stderr, "     Create a Certificate…")
	fmt.Fprintf(os.Stderr, "  2. Name: %s   Identity Type: Self Signed Root   Certificate Type: Code Signing\n", signingCertCN)
	fmt.Fprintln(os.Stderr, "  3. Create it (login keychain), then re-run this command.")
}

// ── pure helpers (unit-tested) ─────────────────────────────────────

// isTCCProtectedPath reports whether p sits inside a macOS privacy-gated
// location: ~/Documents, ~/Desktop, ~/Downloads, or any mounted volume
// under /Volumes. These are the folders that trigger a launch-time
// "allow access?" prompt for a launchd-spawned daemon.
func isTCCProtectedPath(home, p string) bool {
	if p == "" {
		return false
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		abs = p
	}
	if strings.HasPrefix(abs, "/Volumes/") {
		return true
	}
	if home == "" {
		return false
	}
	for _, sub := range []string{"Documents", "Desktop", "Downloads"} {
		base := filepath.Join(home, sub)
		if abs == base || strings.HasPrefix(abs, base+string(os.PathSeparator)) {
			return true
		}
	}
	return false
}

// signatureIsAdhoc parses `codesign -dvvv` output for the ad-hoc marker.
func signatureIsAdhoc(codesignOutput string) bool {
	return strings.Contains(codesignOutput, "Signature=adhoc") ||
		strings.Contains(codesignOutput, "flags=0x2(adhoc)") ||
		strings.Contains(codesignOutput, "adhoc")
}

// signatureHasIdentifier reports whether the signature carries our stable
// identifier (i.e. it was signed by `opendray setup-macos`).
func signatureHasIdentifier(codesignOutput, identifier string) bool {
	return strings.Contains(codesignOutput, "Identifier="+identifier)
}

func orNone(s string) string {
	if strings.TrimSpace(s) == "" {
		return "(none)"
	}
	return s
}

// defaultConfigPath is opendray's preferred config location: ~/.opendray/
// config.toml — deliberately OUTSIDE the TCC-protected folders so a fresh
// install's gateway starts without a privacy prompt (Layer 0). Used by
// `run` as the fallback when no -config is given.
func defaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return ""
	}
	return filepath.Join(home, ".opendray", "config.toml")
}

// activeConfigPath reports the config the gateway would actually use, for
// diagnostics: the launchd plist's -config argument on macOS (the real
// configured path), else the ~/.opendray default when that file exists.
func activeConfigPath() string {
	if runtime.GOOS == "darwin" {
		if p := configFromLaunchdPlist(); p != "" {
			return p
		}
	}
	if d := defaultConfigPath(); d != "" {
		if _, err := os.Stat(d); err == nil {
			return d
		}
	}
	return ""
}

// configFromLaunchdPlist extracts the -config argument from the user
// LaunchAgent plist via a small line scan (avoids a plist parser dep).
func configFromLaunchdPlist() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	data, err := os.ReadFile(filepath.Join(home, "Library", "LaunchAgents", "com.opendray.opendray.plist"))
	if err != nil {
		return ""
	}
	return configArgFromPlistXML(string(data))
}

// configArgFromPlistXML pulls the value of the <string> immediately after
// a `-config` <string> entry in a ProgramArguments array. Pure for tests.
func configArgFromPlistXML(xml string) string {
	const marker = "<string>-config</string>"
	i := strings.Index(xml, marker)
	if i < 0 {
		return ""
	}
	rest := xml[i+len(marker):]
	open := strings.Index(rest, "<string>")
	if open < 0 {
		return ""
	}
	rest = rest[open+len("<string>"):]
	closeIdx := strings.Index(rest, "</string>")
	if closeIdx < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:closeIdx])
}
