// Package gitactivity summarises recent git history into a markdown
// "Recent activity" doc that ends up in the spawn-time banner. M16c.
//
// Flow:
//
//  1. Reader.Read(cwd, since) shells out to `git log --since=... --stat`,
//     parses commit headers + file change stats. Pure data extraction;
//     no LLM, no judgement.
//  2. Summariser.Summarise(commits) sends the parsed commits to an
//     OpenAI-compatible chat completions endpoint with a "project
//     historian" prompt. Output is 1-3 short paragraphs covering
//     themes / hot areas / notable decisions.
//  3. Service.Run(cwd) wires Reader + Summariser, persists the
//     output as project_docs.kind='recent_activity', updated_by=
//     'scanner'.
//
// Triggered by a periodic scheduler (default 24h) and by an admin
// HTTP endpoint POST /git-activity/run. Spawn-time banner reads
// the stored doc directly through RenderForSpawn — no synchronous
// summarisation at spawn (LLM is too slow + the doc rarely changes
// session-to-session).
package gitactivity

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/opendray/opendray-v2/internal/projectdoc"
)

// Commit is one git log entry with its file change stat.
type Commit struct {
	SHA          string // short SHA
	Author       string
	AuthoredAt   time.Time
	Subject      string // first line of commit message
	FilesChanged int
	Insertions   int
	Deletions    int
	// Files lists up to 10 paths from the commit's --stat line.
	// More than that and we truncate — keeps the prompt under
	// control for high-churn commits.
	Files []string
}

// Reader extracts commits from a git repo. Pure data — no LLM.
type Reader struct {
	log *slog.Logger
}

func NewReader(log *slog.Logger) *Reader {
	if log == nil {
		log = slog.Default()
	}
	return &Reader{log: log.With("component", "gitactivity.reader")}
}

// Read runs `git log` in cwd and parses the output. since clamps
// the history window (e.g. "7 days ago", "2 weeks ago"). limit
// caps the number of commits returned regardless of the window.
//
// Returns an empty slice (not an error) for repos with no commits
// in the window. Returns an error only when the cwd is not a git
// repo or git itself fails.
func (r *Reader) Read(ctx context.Context, cwd, since string, limit int) ([]Commit, error) {
	if strings.TrimSpace(cwd) == "" {
		return nil, errors.New("gitactivity: empty cwd")
	}
	if since == "" {
		since = "7 days ago"
	}
	if limit <= 0 {
		limit = 50
	}

	// --pretty: structured header line we can split on
	// --stat:   per-file change stats + summary line
	// --no-color: don't try to emit ANSI escapes
	args := []string{
		"log",
		"--since=" + since,
		"--max-count=" + strconv.Itoa(limit),
		"--no-color",
		"--no-merges",
		`--pretty=format:%x1ecommit %h%n%an%n%aI%n%s%n%x1f`,
		"--stat",
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		// Many error cases are "not a git repo" — surface to the
		// caller as a structured error so the service layer can
		// no-op gracefully.
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			return nil, fmt.Errorf("gitactivity: git log: %w (stderr=%s)", err, string(ee.Stderr))
		}
		return nil, fmt.Errorf("gitactivity: git log: %w", err)
	}
	return parseGitLog(out), nil
}

// parseGitLog walks the output of our --pretty + --stat format.
// Commit blocks are separated by 0x1e (record separator), the
// header section ends with 0x1f, after which the --stat block
// continues until the next 0x1e or EOF.
func parseGitLog(b []byte) []Commit {
	var commits []Commit
	blocks := bytes.Split(b, []byte{0x1e})
	for _, blk := range blocks {
		blk = bytes.TrimSpace(blk)
		if len(blk) == 0 {
			continue
		}
		parts := bytes.SplitN(blk, []byte{0x1f}, 2)
		if len(parts) == 0 {
			continue
		}
		header := parts[0]
		var stat []byte
		if len(parts) == 2 {
			stat = parts[1]
		}
		c, ok := parseHeader(header)
		if !ok {
			continue
		}
		c.FilesChanged, c.Insertions, c.Deletions, c.Files = parseStat(stat)
		commits = append(commits, c)
	}
	return commits
}

// parseHeader expects: "commit <sha>\n<author>\n<iso8601>\n<subject>"
func parseHeader(b []byte) (Commit, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(b))
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if len(lines) < 4 {
		return Commit{}, false
	}
	shaLine := strings.TrimPrefix(lines[0], "commit ")
	authoredAt, _ := time.Parse(time.RFC3339, lines[2])
	return Commit{
		SHA:        shaLine,
		Author:     lines[1],
		AuthoredAt: authoredAt,
		Subject:    lines[3],
	}, true
}

// parseStat extracts per-file lines + the summary line of
// `git log --stat`. Summary format:
//
//	"N files changed, X insertions(+), Y deletions(-)"
func parseStat(b []byte) (filesChanged, insertions, deletions int, files []string) {
	scanner := bufio.NewScanner(bytes.NewReader(b))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		// Summary line.
		if strings.Contains(line, "file") && strings.Contains(line, "changed") {
			filesChanged = extractInt(line, " file")
			insertions = extractInt(line, " insertion")
			deletions = extractInt(line, " deletion")
			continue
		}
		// Per-file line: "<path> | <count> <+/->"
		if idx := strings.Index(line, " | "); idx > 0 {
			path := strings.TrimSpace(line[:idx])
			if path != "" && len(files) < 10 {
				files = append(files, path)
			}
		}
	}
	return
}

// extractInt finds the integer immediately preceding the given
// suffix in line. Returns 0 when not found.
func extractInt(line, suffix string) int {
	idx := strings.Index(line, suffix)
	if idx <= 0 {
		return 0
	}
	// walk back from idx until non-digit
	end := idx
	start := end
	for start > 0 && line[start-1] >= '0' && line[start-1] <= '9' {
		start--
	}
	if start == end {
		return 0
	}
	v, _ := strconv.Atoi(line[start:end])
	return v
}

// ── Aggregation ────────────────────────────────────────────────

// Summary precomputes lightweight aggregates the LLM prompt needs.
type Summary struct {
	Commits      []Commit
	TotalCommits int
	TotalFiles   int
	TotalInserts int
	TotalDeletes int
	HotPaths     []PathStat // top 10 paths by change count
	WindowSince  string
	GeneratedAt  time.Time
}

// PathStat is one entry in HotPaths.
type PathStat struct {
	Path string
	Hits int
}

// SummariseRaw rolls up a commit slice into a Summary. No LLM —
// just counting. The LLM step happens in the Summariser.
func SummariseRaw(commits []Commit, since string) Summary {
	s := Summary{
		Commits:      commits,
		WindowSince:  since,
		GeneratedAt:  time.Now().UTC(),
		TotalCommits: len(commits),
	}
	hits := map[string]int{}
	for _, c := range commits {
		s.TotalFiles += c.FilesChanged
		s.TotalInserts += c.Insertions
		s.TotalDeletes += c.Deletions
		for _, f := range c.Files {
			hits[f]++
		}
	}
	// Top 10 paths.
	type kv struct {
		k string
		v int
	}
	pairs := make([]kv, 0, len(hits))
	for k, v := range hits {
		pairs = append(pairs, kv{k, v})
	}
	// Bubble-style trim — len(pairs) is small.
	for i := 0; i < len(pairs); i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[j].v > pairs[i].v {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}
	max := 10
	if len(pairs) < max {
		max = len(pairs)
	}
	for i := 0; i < max; i++ {
		s.HotPaths = append(s.HotPaths, PathStat{Path: pairs[i].k, Hits: pairs[i].v})
	}
	return s
}

// RenderRawMarkdown formats a Summary as a fallback markdown body
// for when no LLM is configured. The service uses this when the
// Summariser is nil or its call fails — better to ship the raw
// stats than nothing.
func RenderRawMarkdown(s Summary) string {
	var b strings.Builder
	// M23 — agent-facing: scope line is enough; "auto-generated"
	// disclaimer is operator-only noise (the doc is overwritten on
	// every scan, no human edits it).
	fmt.Fprintf(&b, "_Window: %s · %d commits · %d files · +%d / -%d_\n\n",
		s.WindowSince, s.TotalCommits, s.TotalFiles, s.TotalInserts, s.TotalDeletes)

	if s.TotalCommits == 0 {
		b.WriteString("No recent commits in the window.\n")
		return b.String()
	}

	if len(s.HotPaths) > 0 {
		b.WriteString("## Hot paths\n\n")
		for _, p := range s.HotPaths {
			fmt.Fprintf(&b, "- `%s` — %d commits\n", p.Path, p.Hits)
		}
		b.WriteString("\n")
	}

	b.WriteString("## Commits\n\n")
	for _, c := range s.Commits {
		fmt.Fprintf(&b, "- `%s` — %s _(%s, +%d/-%d)_\n",
			c.SHA, c.Subject, c.AuthoredAt.Format("Jan 02"),
			c.Insertions, c.Deletions)
	}
	return b.String()
}

// ── Service ────────────────────────────────────────────────────

// Service ties Reader + (optional) LLM summariser + persistence.
type Service struct {
	reader       *Reader
	llm          *Client // nil → fall back to raw markdown
	docs         *projectdoc.Service
	defaultSince string
	limit        int
	log          *slog.Logger
}

// ServiceOption customises NewService.
type ServiceOption func(*Service)

// WithLLM installs an LLM summariser. Without it, Service.Run
// persists the raw stats markdown via RenderRawMarkdown.
func WithLLM(c *Client) ServiceOption {
	return func(s *Service) { s.llm = c }
}

// WithWindow sets the default "since" expression. Default "7 days ago".
func WithWindow(since string) ServiceOption {
	return func(s *Service) { s.defaultSince = since }
}

// WithCommitLimit caps how many commits are sent through git log
// regardless of the window. Default 50.
func WithCommitLimit(n int) ServiceOption {
	return func(s *Service) { s.limit = n }
}

// NewService builds a Service.
func NewService(docs *projectdoc.Service, log *slog.Logger, opts ...ServiceOption) *Service {
	if log == nil {
		log = slog.Default()
	}
	s := &Service{
		reader:       NewReader(log),
		docs:         docs,
		defaultSince: "7 days ago",
		limit:        50,
		log:          log.With("component", "gitactivity.service"),
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Run reads recent git history in cwd, summarises (LLM if wired,
// raw stats otherwise), and persists the markdown as
// project_docs.kind='recent_activity'.
//
// Errors from git itself propagate so the caller can decide
// whether to retry. LLM call failures degrade silently to raw
// markdown.
func (s *Service) Run(ctx context.Context, cwd string) (projectdoc.Doc, error) {
	commits, err := s.reader.Read(ctx, cwd, s.defaultSince, s.limit)
	if err != nil {
		return projectdoc.Doc{}, err
	}
	summary := SummariseRaw(commits, s.defaultSince)

	var body string
	if s.llm != nil && len(commits) > 0 {
		// LLM gets its own long-deadline context — reasoning models
		// on LM Studio can take 2-3 minutes on a 50-commit batch.
		// The HTTP request context (60s by default) would otherwise
		// cut the LLM off mid-stream.
		llmCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		llmBody, lerr := s.llm.Summarise(llmCtx, summary)
		cancel()
		if lerr != nil {
			s.log.Warn("gitactivity llm summarise failed; falling back to raw", "err", lerr)
		} else if strings.TrimSpace(llmBody) != "" {
			// Wrap LLM body with our header so operators always see
			// generation metadata.
			body = composeFinalMarkdown(summary, llmBody)
		}
	}
	if body == "" {
		body = RenderRawMarkdown(summary)
	}

	// Persist with a fresh short-deadline context. We do NOT want
	// the LLM call's long timeout (or a slow caller's HTTP request
	// context running out of slack) to make the DB write fail —
	// we already paid for the LLM work, dropping the result is
	// strictly worse than re-using a few more milliseconds.
	persistCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	doc, err := s.docs.PutDoc(persistCtx, cwd, projectdoc.KindRecentActivity, body, projectdoc.AuthorScanner)
	if err != nil {
		return projectdoc.Doc{}, fmt.Errorf("gitactivity: persist: %w", err)
	}
	s.log.Info("gitactivity.scanned",
		"cwd", cwd,
		"window", s.defaultSince,
		"commits", summary.TotalCommits,
		"files", summary.TotalFiles,
		"with_llm", s.llm != nil,
	)
	return doc, nil
}

// IsStale mirrors projectscan.Service.IsStale — the scheduler /
// HTTP layer uses it to decide whether to re-run.
func (s *Service) IsStale(ctx context.Context, cwd string, maxAge time.Duration) bool {
	doc, err := s.docs.GetDoc(ctx, cwd, projectdoc.KindRecentActivity)
	if err != nil {
		return true
	}
	return time.Since(doc.UpdatedAt) > maxAge
}

// RefreshAsync runs Service.Run for cwd in a detached goroutine
// with its own background context. Used by the catalog adapter at
// spawn time so a slow LLM call (~60-150s on reasoning models)
// can't block the agent's PTY allocation.
//
// We do not coalesce concurrent calls for the same cwd — if two
// spawns hit a stale doc at the same time, both fire. Service.Run
// is idempotent (UPSERT) so the worst case is one redundant LLM
// call, not corrupted data.
func (s *Service) RefreshAsync(cwd string) {
	if cwd == "" {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Minute)
		defer cancel()
		if _, err := s.Run(ctx, cwd); err != nil {
			s.log.Warn("gitactivity.refresh_async_failed", "cwd", cwd, "err", err)
		}
	}()
}

// composeFinalMarkdown wraps the LLM-generated narrative with a
// deterministic header (window + stats) so agents always know the
// scope. M23 — dropped the "auto-generated" disclaimer; agent
// consumers don't need it and operators read the doc via UI which
// has its own provenance UI.
func composeFinalMarkdown(s Summary, llmBody string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "_Window: %s · %d commits · %d files · +%d / -%d_\n\n",
		s.WindowSince, s.TotalCommits, s.TotalFiles, s.TotalInserts, s.TotalDeletes)
	b.WriteString(strings.TrimSpace(llmBody))
	b.WriteString("\n\n")
	if len(s.HotPaths) > 0 {
		b.WriteString("## Hot paths\n\n")
		for _, p := range s.HotPaths {
			fmt.Fprintf(&b, "- `%s` — %d commits\n", p.Path, p.Hits)
		}
	}
	return b.String()
}
