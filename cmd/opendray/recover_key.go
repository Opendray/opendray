// recover-key subcommand: reconstruct the backup passphrase from a
// Recovery Kit (see internal/backup/recovery.go).
//
// Disaster-recovery flow on a fresh host where the original backup
// passphrase/keyfile is gone but the operator kept their Recovery Kit
// and recovery passphrase:
//
//	OPENDRAY_RECOVERY_PASSPHRASE=… opendray recover-key --kit kit.json --install
//
// --install writes the recovered passphrase to ~/.opendray/secrets/
// backup.key so the next start can decrypt existing backups; without it
// the passphrase is printed to stdout so the operator can set
// OPENDRAY_BACKUP_KEY themselves.

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/opendray/opendray-v2/internal/backup"
)

// recoveryPassphraseEnv is read first; if unset, recover-key prompts on
// stdin (the input is echoed — this runs in a recovery shell, not a
// service).
const recoveryPassphraseEnv = "OPENDRAY_RECOVERY_PASSPHRASE"

func runRecoverKey(args []string) int {
	fs := flag.NewFlagSet("recover-key", flag.ExitOnError)
	kitPath := fs.String("kit", "", "path to the Recovery Kit JSON file (required)")
	install := fs.Bool("install", false, "write the recovered passphrase to the default keyfile (~/.opendray/secrets/backup.key)")
	overwrite := fs.Bool("overwrite", false, "with --install, replace an existing keyfile")
	fs.Parse(args)

	if *kitPath == "" {
		fmt.Fprintln(os.Stderr, "recover-key: --kit is required")
		return 2
	}
	kit, err := os.ReadFile(*kitPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "recover-key: read kit: %v\n", err)
		return 1
	}

	// Show which backup key this kit unlocks before asking for anything.
	if fp, ferr := backup.RecoveryKitFingerprint(kit); ferr == nil && fp != "" {
		fmt.Fprintf(os.Stderr, "Recovery Kit unlocks backup key fingerprint: %s\n", fp)
	}

	pass := os.Getenv(recoveryPassphraseEnv)
	if pass == "" {
		fmt.Fprintf(os.Stderr, "Recovery passphrase (or set %s): ", recoveryPassphraseEnv)
		sc := bufio.NewScanner(os.Stdin)
		if !sc.Scan() {
			if err := sc.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "recover-key: read passphrase: %v\n", err)
			} else {
				fmt.Fprintln(os.Stderr, "recover-key: no passphrase provided")
			}
			return 1
		}
		pass = strings.TrimRight(sc.Text(), "\r")
	}
	if pass == "" {
		fmt.Fprintln(os.Stderr, "recover-key: empty recovery passphrase")
		return 1
	}

	backupPass, err := backup.ImportRecoveryKit(kit, pass)
	if err != nil {
		fmt.Fprintf(os.Stderr, "recover-key: %v\n", err)
		return 1
	}

	if *install {
		path, werr := backup.WriteKeyFile(backupPass, *overwrite)
		if werr != nil {
			fmt.Fprintf(os.Stderr, "recover-key: install keyfile: %v\n", werr)
			return 1
		}
		fmt.Fprintf(os.Stderr, "OK: backup passphrase recovered and written to %s\n", path)
		fmt.Fprintln(os.Stderr, "  restart opendray to decrypt existing backups.")
		return 0
	}

	// Print only the passphrase on stdout so it can be captured:
	//   export OPENDRAY_BACKUP_KEY="$(opendray recover-key --kit kit.json)"
	fmt.Println(backupPass)
	fmt.Fprintln(os.Stderr, "OK: backup passphrase recovered (printed to stdout).")
	fmt.Fprintln(os.Stderr, "  set OPENDRAY_BACKUP_KEY to it, or re-run with --install.")
	return 0
}
