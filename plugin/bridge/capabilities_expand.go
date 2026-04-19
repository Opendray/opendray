package bridge

import (
	"context"
	"strings"
)

// PathVarCtx holds the base-variable values used to expand declared
// capability patterns at call time. Populated per request by the
// gateway's PathVarResolver (M3 T24) from the active session's
// workspace + the plugin's install data dir.
//
// A field left empty means the corresponding ${var} stays literal in
// the expanded pattern — which makes MatchFSPath fail to match as a
// safe default.
type PathVarCtx struct {
	Workspace string // ${workspace}
	Home      string // ${home}
	DataDir   string // ${dataDir}
	Tmp       string // ${tmp}
}

// ExpandPathVars substitutes ${workspace}, ${home}, ${dataDir}, ${tmp}
// inside a pattern using ctx. Unknown variables are left literal so
// the downstream matcher decides: a typo like ${worksapce} fails to
// match any real path — the fail-closed default M3 wants.
//
// Path-traversal strings (".." inside the substituted value, or
// around it) are NOT sanitised here — MatchFSPath runs filepath.Clean
// on the final candidate before the glob compare, so traversal
// attempts that cross the pattern's anchor collapse safely.
func ExpandPathVars(pattern string, ctx PathVarCtx) string {
	if !strings.Contains(pattern, "${") {
		return pattern
	}
	replaced := pattern
	if ctx.Workspace != "" {
		replaced = strings.ReplaceAll(replaced, "${workspace}", ctx.Workspace)
	}
	if ctx.Home != "" {
		replaced = strings.ReplaceAll(replaced, "${home}", ctx.Home)
	}
	if ctx.DataDir != "" {
		replaced = strings.ReplaceAll(replaced, "${dataDir}", ctx.DataDir)
	}
	if ctx.Tmp != "" {
		replaced = strings.ReplaceAll(replaced, "${tmp}", ctx.Tmp)
	}
	return replaced
}

// CheckExpanded is Check with per-call path-variable context. Every fs.*
// grant glob is expanded via ExpandPathVars before the MatchFSPath
// compare. Non-fs caps ignore vars entirely and fall back to Check.
//
// Keeping this a new method (rather than mutating Check's signature) lets
// M2 callers continue using Check unchanged.
func (g *Gate) CheckExpanded(ctx context.Context, plugin string, need Need, vars PathVarCtx) error {
	// Nothing to expand → identical to Check.
	if !strings.HasPrefix(need.Cap, "fs.") || !strings.Contains(need.Target, "") || vars == (PathVarCtx{}) {
		return g.Check(ctx, plugin, need)
	}
	// Expand the target before it reaches MatchFSPath. Note we only
	// expand the Target — the stored grants are expanded inside evaluate()
	// via a future hook (T9 wires this through by calling ExpandPathVars
	// on each grant). For T3 we only build the plumbing; the matcher tap
	// lands in T9 when api_fs.go is introduced.
	expanded := ExpandPathVars(need.Target, vars)
	return g.Check(ctx, plugin, Need{Cap: need.Cap, Target: expanded})
}
