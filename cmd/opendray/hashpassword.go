package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"
)

// runHashPassword implements `opendray hash-password`: read a password
// from stdin, bcrypt-hash it, and print the hash to stdout. Output is
// one line suitable for pasting into config.toml as:
//
//	[admin]
//	password_hash = "$2a$12$..."
//
// Reads from stdin so passwords don't end up in shell history. Two
// flows:
//
//	# interactive — stdin is a TTY
//	$ opendray hash-password
//	enter password (no echo): <type, press enter>
//	$2a$10$…
//
//	# piped — stdin is a pipe
//	$ echo 'mypw' | opendray hash-password
//	$2a$10$…
//
// Exit codes: 0 on success, 1 on read or hash error, 2 on usage error.
func runHashPassword(args []string) int {
	fs := flag.NewFlagSet("hash-password", flag.ExitOnError)
	cost := fs.Int("cost", bcrypt.DefaultCost, "bcrypt cost factor (4..31; default = bcrypt.DefaultCost)")
	fs.Usage = func() {
		fmt.Fprintln(os.Stderr, `opendray hash-password — generate a bcrypt hash for the admin password

usage:
  opendray hash-password                    # interactive (reads stdin)
  echo 'mypw' | opendray hash-password      # piped

flags:
  -cost   bcrypt cost (default: package default)`)
	}
	_ = fs.Parse(args)

	if *cost < bcrypt.MinCost || *cost > bcrypt.MaxCost {
		fmt.Fprintf(os.Stderr, "hash-password: cost must be in [%d, %d]\n", bcrypt.MinCost, bcrypt.MaxCost)
		return 2
	}

	if isTerminal(os.Stdin) {
		// Best-effort prompt for interactive use. Echo cannot be
		// suppressed without pulling in golang.org/x/term as a new
		// dependency; skipping that for now to keep the change
		// minimal. Operators concerned about shoulder-surfing can
		// pipe the password in instead.
		fmt.Fprintln(os.Stderr, "enter password and press enter (visible while typing):")
	}

	password, err := readLine(os.Stdin)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash-password: %v\n", err)
		return 1
	}
	if password == "" {
		fmt.Fprintln(os.Stderr, "hash-password: empty password")
		return 1
	}
	if !utf8.ValidString(password) {
		fmt.Fprintln(os.Stderr, "hash-password: password is not valid UTF-8")
		return 1
	}
	// bcrypt has a 72-byte input cap; longer is silently truncated.
	// Refuse up front so an operator with a 100-char password isn't
	// surprised that only the first 72 bytes determine the hash.
	if len(password) > 72 {
		fmt.Fprintln(os.Stderr, "hash-password: bcrypt has a 72-byte input limit; choose a shorter password")
		return 1
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), *cost)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash-password: %v\n", err)
		return 1
	}
	fmt.Println(string(hash))
	return 0
}

func readLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	line, err := br.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read stdin: %w", err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

// isTerminal reports whether the file is a terminal. Implemented via
// os.Stat + ModeCharDevice to avoid pulling golang.org/x/term as a
// new module dep just for this one check.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
