// One-shot verification: decrypt a backup blob via the cipher,
// untar the contents, and run `pg_restore --list` against the dump
// to prove it's structurally valid without needing a target server.
package main

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"

	"github.com/opendray/opendray-v2/internal/backup"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatalf("usage: %s <bundle.enc> <passphrase> <pg_restore_path>", os.Args[0])
	}
	bundlePath, passphrase, restorePath := os.Args[1], os.Args[2], os.Args[3]

	c, err := backup.NewCipher(passphrase)
	if err != nil {
		log.Fatalf("cipher: %v", err)
	}
	fmt.Printf("cipher fingerprint: %s\n", c.Fingerprint())

	f, err := os.Open(bundlePath)
	if err != nil {
		log.Fatalf("open: %v", err)
	}
	defer f.Close()

	plain := c.Open(f)
	gzr, err := gzip.NewReader(plain)
	if err != nil {
		log.Fatalf("gzip: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	var manifestBody []byte
	dumpPath := "/tmp/extracted-dump.bin"
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("tar: %v", err)
		}
		fmt.Printf("entry: %-20s  %d bytes\n", h.Name, h.Size)
		switch h.Name {
		case "manifest.json":
			manifestBody, _ = io.ReadAll(tr)
		case "dump.bin":
			out, _ := os.Create(dumpPath)
			io.Copy(out, tr)
			out.Close()
		default:
			io.Copy(io.Discard, tr)
		}
	}

	var mf map[string]any
	json.Unmarshal(manifestBody, &mf)
	fmt.Printf("\nmanifest fingerprint: %v\nbackup_id: %v\npg_version: %v\nversion: %v\n",
		mf["encryption"].(map[string]any)["fingerprint"], mf["backup_id"], mf["pg_version"], mf["version"])

	fmt.Printf("\n--- pg_restore --list output (header only) ---\n")
	cmd := exec.Command(restorePath, "--list", dumpPath)
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("pg_restore --list error: %v\noutput: %s\n", err, out)
		os.Exit(1)
	}
	// print first 30 lines
	lines := splitLines(string(out))
	for i, l := range lines {
		if i >= 30 {
			fmt.Printf("...(%d more lines)\n", len(lines)-i)
			break
		}
		fmt.Println(l)
	}
}

func splitLines(s string) []string {
	out := []string{""}
	for _, r := range s {
		if r == '\n' {
			out = append(out, "")
			continue
		}
		out[len(out)-1] += string(r)
	}
	return out
}
