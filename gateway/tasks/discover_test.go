package tasks

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func makeConfig(root string) Config {
	return Config{
		AllowedRoots:        []string{root},
		DefaultPath:         root,
		IncludeMakefile:     true,
		IncludePackageJSON:  true,
		IncludeShellScripts: true,
	}
}

func names(tasks []Task) []string {
	out := make([]string, 0, len(tasks))
	for _, t := range tasks {
		out = append(out, string(t.Source)+":"+t.Name)
	}
	sort.Strings(out)
	return out
}

func TestDiscover_Makefile(t *testing.T) {
	tests := []struct {
		name     string
		makefile string
		want     []string
	}{
		{
			name: "basic targets",
			makefile: "build:\n\tgo build\n\ntest:\n\tgo test ./...\n",
			want: []string{"makefile:build", "makefile:test"},
		},
		{
			name: "skips PHONY and pattern rules",
			makefile: ".PHONY: build\nbuild:\n\tgo build\n%.o: %.c\n\tcc -c $<\n",
			want: []string{"makefile:build"},
		},
		{
			name: "skips variable assignment",
			makefile: "VERSION := 1.0\nbuild:\n\tgo build\n",
			want: []string{"makefile:build"},
		},
		{
			name:     "no targets",
			makefile: "VERSION := 1.0\n",
			want:     nil,
		},
		{
			name: "deduplicates",
			makefile: "build:\n\tgo build\nbuild:\n\tgo build\n",
			want: []string{"makefile:build"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "Makefile"), tc.makefile)
			got, err := Discover(makeConfig(dir), dir)
			if err != nil {
				t.Fatalf("discover: %v", err)
			}
			gotNames := names(got)
			if len(gotNames) != len(tc.want) {
				t.Fatalf("got %v, want %v", gotNames, tc.want)
			}
			for i, n := range tc.want {
				if gotNames[i] != n {
					t.Errorf("got[%d]=%q want %q", i, gotNames[i], n)
				}
			}
		})
	}
}

func TestDiscover_PackageJSON(t *testing.T) {
	tests := []struct {
		name        string
		pkg         string
		extraFiles  map[string]string
		wantRunner  string
		wantScripts []string
	}{
		{
			name:        "npm default",
			pkg:         `{"scripts":{"build":"tsc","test":"jest"}}`,
			wantRunner:  "npm",
			wantScripts: []string{"build", "test"},
		},
		{
			name:        "pnpm via lockfile",
			pkg:         `{"scripts":{"dev":"vite"}}`,
			extraFiles:  map[string]string{"pnpm-lock.yaml": "lockfile: 1\n"},
			wantRunner:  "pnpm",
			wantScripts: []string{"dev"},
		},
		{
			name:        "yarn via lockfile",
			pkg:         `{"scripts":{"start":"node ."}}`,
			extraFiles:  map[string]string{"yarn.lock": "# yarn lockfile\n"},
			wantRunner:  "yarn",
			wantScripts: []string{"start"},
		},
		{
			name:        "packageManager hint wins",
			pkg:         `{"scripts":{"x":"echo"},"packageManager":"pnpm@8.6.0"}`,
			wantRunner:  "pnpm",
			wantScripts: []string{"x"},
		},
		{
			name: "no scripts",
			pkg:  `{"name":"x"}`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "package.json"), tc.pkg)
			for name, content := range tc.extraFiles {
				writeFile(t, filepath.Join(dir, name), content)
			}
			got, err := Discover(makeConfig(dir), dir)
			if err != nil {
				t.Fatalf("discover: %v", err)
			}
			if len(tc.wantScripts) == 0 {
				if len(got) != 0 {
					t.Fatalf("expected no tasks, got %v", names(got))
				}
				return
			}
			scripts := make(map[string]Task)
			for _, task := range got {
				if task.Source == SourcePackageJSON {
					scripts[task.Name] = task
				}
			}
			if len(scripts) != len(tc.wantScripts) {
				t.Fatalf("got %d scripts, want %d", len(scripts), len(tc.wantScripts))
			}
			for _, name := range tc.wantScripts {
				task, ok := scripts[name]
				if !ok {
					t.Errorf("missing script %q", name)
					continue
				}
				if task.Argv[0] != tc.wantRunner {
					t.Errorf("script %q runner=%q want %q", name, task.Argv[0], tc.wantRunner)
				}
			}
		})
	}
}

func TestDiscover_ShellScripts(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "build_release.sh"), "#!/usr/bin/env bash\necho hi\n")
	writeFile(t, filepath.Join(dir, "scripts", "deploy.sh"), "#!/usr/bin/env bash\necho deploy\n")
	writeFile(t, filepath.Join(dir, "README.md"), "not a script")

	got, err := Discover(makeConfig(dir), dir)
	if err != nil {
		t.Fatalf("discover: %v", err)
	}
	want := map[string]bool{
		"shell:build_release.sh": false,
		"shell:scripts/deploy.sh": false,
	}
	for _, n := range names(got) {
		if _, ok := want[n]; ok {
			want[n] = true
		}
	}
	for k, ok := range want {
		if !ok {
			t.Errorf("missing %s in %v", k, names(got))
		}
	}
}

func TestDiscover_RejectsOutsideAllowedRoots(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	cfg := makeConfig(root)
	if _, err := Discover(cfg, other); err == nil {
		t.Fatalf("expected error for path outside allowed roots")
	}
}

func TestHashIDStable(t *testing.T) {
	a := hashID(SourceMakefile, "/x", "build")
	b := hashID(SourceMakefile, "/x", "build")
	c := hashID(SourceMakefile, "/x", "test")
	if a != b {
		t.Fatalf("hash should be stable, got %s vs %s", a, b)
	}
	if a == c {
		t.Fatalf("different names should hash differently")
	}
}
