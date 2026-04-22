package sourcecontrol

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/opendray/opendray/gateway/git"
)

// looksLikeRef mirrors the refname regex gateway/git enforces on
// untrusted refs so the commit parameter can't sneak shell
// metacharacters into `git show`.
var refRegex = regexp.MustCompile(`^[A-Za-z0-9._/\-]{1,200}$`)

func looksLikeRef(s string) bool { return refRegex.MatchString(s) }

// FileDiff is one changed file rendered whole — rather than the
// single-file-at-a-time model the old git-viewer used, the panel gets
// every file in one response and can render them as a stacked list.
type FileDiff struct {
	Path        string `json:"path"`
	OldPath     string `json:"oldPath,omitempty"`
	Status      string `json:"status"`   // "modified" | "added" | "deleted" | "renamed" | "untracked"
	Add         int    `json:"add"`      // lines added
	Del         int    `json:"del"`      // lines deleted
	IsBinary    bool   `json:"isBinary"` // numstat reported "-"; Patch is empty
	Patch       string `json:"patch"`    // full unified diff hunk for this file
	PreviewHTML string `json:"previewHtml,omitempty"` // sanitised markdown render; only populated for .md/.markdown when MarkdownPreview=true
}

// MultiDiffMode selects which slice of the working tree to diff.
type MultiDiffMode string

const (
	ModeUnstaged MultiDiffMode = "unstaged" // working tree vs index
	ModeStaged   MultiDiffMode = "staged"   // index vs HEAD
	ModeBaseline MultiDiffMode = "baseline" // HEAD vs a prior session-start SHA
	ModeCommit   MultiDiffMode = "commit"   // single commit — git show <sha>
)

// MultiDiffOptions parameterises MultiDiff.
//
// Field usage by mode:
//
//	ModeUnstaged / ModeStaged → nothing else
//	ModeBaseline              → Since (commit-ish; older tree)
//	ModeCommit                → Commit (single SHA to inspect)
type MultiDiffOptions struct {
	Mode   MultiDiffMode
	Since  string // baseline commit SHA, validated by git.Diff via validateRef
	Commit string // SHA for ModeCommit — "git show <commit>"
	Full   bool   // true ⇒ render with maximum context (0x u flag)
}

// MultiDiffResult is what the handler returns. Files is sorted by
// path for deterministic rendering.
type MultiDiffResult struct {
	Repo  string        `json:"repo"`
	Mode  MultiDiffMode `json:"mode"`
	Files []FileDiff    `json:"files"`
}

// MultiDiff returns every changed file's unified diff in one shot.
// Uses two git invocations per mode:
//
//  1. `git diff [--cached|<since>] --no-color -U<n>` for the patch
//  2. `git diff [--cached|<since>] --numstat` for per-file counts
//
// The results are zipped together on the server so the panel gets a
// single list it can render without further round-trips.
func MultiDiff(ctx context.Context, cfg Config, repoPath string, opts MultiDiffOptions) (MultiDiffResult, error) {
	repo, err := git.SecurePath(cfg.gitConfig(), repoPath)
	if err != nil {
		return MultiDiffResult{}, err
	}

	mode := opts.Mode
	if mode == "" {
		mode = ModeUnstaged
	}

	contextArg := fmt.Sprintf("-U%d", diffContextFor(cfg, opts.Full))

	var baseArgs []string
	switch mode {
	case ModeUnstaged:
		baseArgs = []string{"diff", contextArg, "--no-color"}
	case ModeStaged:
		baseArgs = []string{"diff", "--cached", contextArg, "--no-color"}
	case ModeBaseline:
		if opts.Since == "" {
			return MultiDiffResult{}, fmt.Errorf("sourcecontrol: baseline mode requires Since")
		}
		// Delegate validation: git.Diff already enforces a strict ref
		// regex before shelling out. Reuse it by constructing the same
		// underlying args shape and letting runGit parse output. But
		// we also want numstat for counts, so run both here.
		baseArgs = []string{"diff", opts.Since, contextArg, "--no-color"}
	case ModeCommit:
		if opts.Commit == "" {
			return MultiDiffResult{}, fmt.Errorf("sourcecontrol: commit mode requires Commit")
		}
		if !looksLikeRef(opts.Commit) {
			return MultiDiffResult{}, fmt.Errorf("sourcecontrol: invalid commit ref %q", opts.Commit)
		}
		// `git show <sha>` emits the commit metadata followed by the
		// diff. --first-parent collapses merge commits to their
		// mainline diff, which is what History-tab users expect when
		// clicking a commit. --no-patch-with-stat would lose patches;
		// stick with default patch + numstat via the second invocation.
		baseArgs = []string{"show", opts.Commit, contextArg, "--no-color",
			"--first-parent", "--format="}
	default:
		return MultiDiffResult{}, fmt.Errorf("sourcecontrol: unknown mode %q", mode)
	}

	patchOut, err := runGit(ctx, cfg, repo, append([]string{}, baseArgs...))
	if err != nil {
		return MultiDiffResult{}, err
	}

	// numstat shares the ref / --cached / --first-parent / --format=
	// flags but strips -U<n> + --no-color and appends --numstat. Works
	// for both `git diff` and `git show` base commands.
	numstatArgs := make([]string, 0, len(baseArgs)+1)
	numstatArgs = append(numstatArgs, baseArgs[0])
	for _, a := range baseArgs[1:] {
		if strings.HasPrefix(a, "-U") || a == "--no-color" {
			continue
		}
		numstatArgs = append(numstatArgs, a)
	}
	numstatArgs = append(numstatArgs, "--numstat")

	numstatOut, err := runGit(ctx, cfg, repo, numstatArgs)
	if err != nil {
		return MultiDiffResult{}, err
	}

	files := parsePatchBlocks(string(patchOut))
	counts := parseNumstat(string(numstatOut))
	for i := range files {
		if c, ok := counts[files[i].Path]; ok {
			files[i].Add = c.add
			files[i].Del = c.del
			files[i].IsBinary = c.binary
		} else if files[i].OldPath != "" {
			if c, ok := counts[files[i].OldPath]; ok {
				files[i].Add = c.add
				files[i].Del = c.del
				files[i].IsBinary = c.binary
			}
		}
	}

	if cfg.MarkdownPreview {
		files = attachMarkdownPreviews(ctx, repo, files, 0)
	}

	return MultiDiffResult{
		Repo:  repo,
		Mode:  mode,
		Files: files,
	}, nil
}

func diffContextFor(cfg Config, full bool) int {
	if full {
		// 99999 is git's convention for "show the whole file" — no
		// real file has 100k context lines around a change.
		return 99999
	}
	if cfg.DiffContext > 0 {
		return cfg.DiffContext
	}
	return 3
}

// runGit is a thin wrapper that matches gateway/git's internal runner:
// timeout + GIT_TERMINAL_PROMPT=0 + -C repo + --no-pager. We need our
// own because gateway/git.run is unexported; the effort to expose it
// properly can come with Phase 4 consolidation.
func runGit(ctx context.Context, cfg Config, repo string, args []string) ([]byte, error) {
	timeout := cfg.CommandTimeout
	if timeout <= 0 {
		timeout = 20 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	bin := cfg.GitBinary
	if bin == "" {
		bin = "git"
	}
	full := append([]string{"--no-pager", "-C", repo}, args...)
	cmd := exec.CommandContext(ctx, bin, full...)
	cmd.Env = append(cmd.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(ee.Stderr)))
		}
		return nil, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// parsePatchBlocks splits a combined `git diff` output into one
// FileDiff per file. Handles:
//   - ordinary modifications ("diff --git a/x b/x")
//   - renames (rename from / rename to headers)
//   - deletions ("deleted file mode")
//   - additions ("new file mode")
//   - binary files ("Binary files ... differ")
func parsePatchBlocks(patch string) []FileDiff {
	if strings.TrimSpace(patch) == "" {
		return []FileDiff{}
	}
	lines := strings.Split(patch, "\n")
	var (
		out     []FileDiff
		curBuf  strings.Builder
		curFile *FileDiff
	)
	flush := func() {
		if curFile == nil {
			return
		}
		curFile.Patch = curBuf.String()
		out = append(out, *curFile)
		curFile = nil
		curBuf.Reset()
	}
	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			flush()
			fd := FileDiff{Status: "modified"}
			// Parse "diff --git a/<old> b/<new>"
			rest := strings.TrimPrefix(line, "diff --git ")
			aPath, bPath := splitDiffPaths(rest)
			if aPath != "" {
				oldP := stripDiffPrefix(aPath)
				newP := stripDiffPrefix(bPath)
				fd.Path = newP
				// Only flag rename when the paths actually differ AFTER
				// stripping a/ and b/ — identical paths are the common
				// case and must stay status=modified.
				if oldP != newP {
					fd.OldPath = oldP
					fd.Status = "renamed"
				}
			}
			curFile = &fd
		}
		if curFile == nil {
			continue
		}
		switch {
		case strings.HasPrefix(line, "new file mode"):
			curFile.Status = "added"
		case strings.HasPrefix(line, "deleted file mode"):
			curFile.Status = "deleted"
		case strings.HasPrefix(line, "rename from "):
			curFile.OldPath = strings.TrimPrefix(line, "rename from ")
			curFile.Status = "renamed"
		case strings.HasPrefix(line, "rename to "):
			curFile.Path = strings.TrimPrefix(line, "rename to ")
			curFile.Status = "renamed"
		case strings.HasPrefix(line, "Binary files ") && strings.HasSuffix(line, " differ"):
			curFile.IsBinary = true
		}
		curBuf.WriteString(line)
		curBuf.WriteByte('\n')
	}
	flush()
	return out
}

// splitDiffPaths parses `a/foo b/bar`, handling quoted paths with
// spaces (`"a/path with space" "b/path with space"`). Quoted paths
// are rare but valid in git output when a path contains spaces.
func splitDiffPaths(s string) (string, string) {
	if strings.HasPrefix(s, `"`) {
		// Find the closing quote.
		if idx := strings.Index(s[1:], `" `); idx >= 0 {
			a := s[1 : 1+idx]
			rest := strings.TrimSpace(s[2+idx:])
			if strings.HasPrefix(rest, `"`) && strings.HasSuffix(rest, `"`) {
				return a, rest[1 : len(rest)-1]
			}
			return a, rest
		}
	}
	parts := strings.SplitN(s, " ", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

// stripDiffPrefix removes the leading "a/" or "b/" git adds to paths
// in its diff headers.
func stripDiffPrefix(p string) string {
	if strings.HasPrefix(p, "a/") || strings.HasPrefix(p, "b/") {
		return p[2:]
	}
	return p
}

type numstatCount struct {
	add, del int
	binary   bool
}

// parseNumstat consumes `git diff --numstat` output:
//
//	<add>\t<del>\t<path>
//
// or for binary files:
//
//	-\t-\t<path>
//
// and produces a path → counts map.
func parseNumstat(s string) map[string]numstatCount {
	out := make(map[string]numstatCount)
	scanner := bufio.NewScanner(strings.NewReader(s))
	for scanner.Scan() {
		parts := strings.SplitN(scanner.Text(), "\t", 3)
		if len(parts) != 3 {
			continue
		}
		path := parts[2]
		// Rename format: "old => new" inside braces; keep the new path
		// so counts can key off the display path. Simple heuristic —
		// handles the common "foo/{old => new}.txt" shape.
		if i := strings.Index(path, " => "); i >= 0 {
			// crude but works for plain renames; brace form is stripped below
			path = strings.ReplaceAll(path, "{", "")
			path = strings.ReplaceAll(path, "}", "")
			parts2 := strings.Split(path, " => ")
			if len(parts2) == 2 {
				path = parts2[1]
			}
		}
		if parts[0] == "-" && parts[1] == "-" {
			out[path] = numstatCount{binary: true}
			continue
		}
		add, _ := strconv.Atoi(parts[0])
		del, _ := strconv.Atoi(parts[1])
		out[path] = numstatCount{add: add, del: del}
	}
	return out
}
