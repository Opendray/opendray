package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
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

// CheckExpanded is Check with per-call path-variable context. For fs.*
// caps it expands BOTH the Target and every declared grant glob via
// ExpandPathVars before running MatchFSPath. Non-fs caps ignore vars
// entirely and fall back to Check.
//
// Keeping this a new method (rather than mutating Check's signature) lets
// M2 callers continue using Check unchanged.
//
// Grant expansion is the "matcher tap" called out in the M3 plan §T9:
// a plugin declaring `fs.read = ["${workspace}/**"]` with vars
// `{Workspace:"/home/kev/proj"}` and target "/home/kev/proj/README.md"
// resolves the grant to "/home/kev/proj/**" before matching.
func (g *Gate) CheckExpanded(ctx context.Context, plugin string, need Need, vars PathVarCtx) error {
	// Non-fs caps — path vars irrelevant, delegate to plain Check.
	if !strings.HasPrefix(need.Cap, "fs.") {
		return g.Check(ctx, plugin, need)
	}
	// fs.* cap but no vars supplied — nothing to expand, delegate.
	if vars == (PathVarCtx{}) {
		return g.Check(ctx, plugin, need)
	}
	// fs.* with vars: load consent, expand grants in-place, then evaluate
	// against the expanded target. We replay the audit shape from Check so
	// downstream audit consumers see consistent records.
	return g.checkFSExpanded(ctx, plugin, need, vars)
}

// checkFSExpanded is the fs.*-specific path of CheckExpanded. It mirrors
// Check's consent-load / parse / evaluate / audit flow but applies
// ExpandPathVars to every declared fs grant before the MatchFSPath call,
// and expands the Target too. The rest of the shape — load errors,
// missing-consent denial, audit write — matches Check byte-for-byte so
// no audit consumer needs to special-case this path.
func (g *Gate) checkFSExpanded(ctx context.Context, plugin string, need Need, vars PathVarCtx) error {
	start := time.Now()

	rawPerms, found, loadErr := g.consents.Load(ctx, plugin)
	if loadErr != nil {
		elapsed := int(time.Since(start).Milliseconds())
		g.appendAudit(ctx, AuditEvent{
			PluginName: plugin,
			Ns:         needNs(need.Cap),
			Method:     need.Cap,
			Caps:       []string{need.Cap},
			Result:     "error",
			DurationMs: elapsed,
			Message:    loadErr.Error(),
		})
		return fmt.Errorf("bridge: load consent for %q: %w", plugin, loadErr)
	}
	if !found {
		elapsed := int(time.Since(start).Milliseconds())
		msg := fmt.Sprintf("no consent record for plugin %q; install the plugin first", plugin)
		g.appendAudit(ctx, AuditEvent{
			PluginName: plugin,
			Ns:         needNs(need.Cap),
			Method:     need.Cap,
			Caps:       []string{need.Cap},
			Result:     "denied",
			DurationMs: elapsed,
			Message:    msg,
		})
		return &PermError{Code: "EPERM", Msg: msg}
	}

	var perms permissionsV1Wire
	if len(rawPerms) > 0 {
		if err := json.Unmarshal(rawPerms, &perms); err != nil {
			elapsed := int(time.Since(start).Milliseconds())
			g.appendAudit(ctx, AuditEvent{
				PluginName: plugin,
				Ns:         needNs(need.Cap),
				Method:     need.Cap,
				Caps:       []string{need.Cap},
				Result:     "error",
				DurationMs: elapsed,
				Message:    err.Error(),
			})
			return fmt.Errorf("bridge: parse consent JSON for %q: %w", plugin, err)
		}
	}

	readRaw, writeRaw := parseFSPerm(perms.Fs)
	readGrants := expandGlobList(readRaw, vars)
	writeGrants := expandGlobList(writeRaw, vars)
	expandedTarget := ExpandPathVars(need.Target, vars)

	var (
		allowed bool
		denyMsg string
	)
	switch need.Cap {
	case "fs.read":
		if MatchFSPath(readGrants, expandedTarget) {
			allowed = true
		} else {
			denyMsg = fmt.Sprintf("fs.read not granted for: %s", expandedTarget)
		}
	case "fs.write":
		if MatchFSPath(writeGrants, expandedTarget) {
			allowed = true
		} else {
			denyMsg = fmt.Sprintf("fs.write not granted for: %s", expandedTarget)
		}
	default:
		// Other fs.* caps (e.g. fs.watch in T10) fall back to Check's
		// evaluate — they don't use grant-expansion yet.
		return g.Check(ctx, plugin, Need{Cap: need.Cap, Target: expandedTarget})
	}

	elapsed := int(time.Since(start).Milliseconds())
	if allowed {
		g.appendAudit(ctx, AuditEvent{
			PluginName: plugin,
			Ns:         needNs(need.Cap),
			Method:     need.Cap,
			Caps:       []string{need.Cap},
			Result:     "ok",
			DurationMs: elapsed,
		})
		return nil
	}
	g.appendAudit(ctx, AuditEvent{
		PluginName: plugin,
		Ns:         needNs(need.Cap),
		Method:     need.Cap,
		Caps:       []string{need.Cap},
		Result:     "denied",
		DurationMs: elapsed,
		Message:    denyMsg,
	})
	return &PermError{Code: "EPERM", Msg: denyMsg}
}

// expandGlobList returns a copy of src with every entry passed through
// ExpandPathVars. Returns nil for a nil input so MatchFSPath's "no
// grants → deny" fast path still fires.
func expandGlobList(src []string, vars PathVarCtx) []string {
	if len(src) == 0 {
		return nil
	}
	out := make([]string, len(src))
	for i, glob := range src {
		out[i] = ExpandPathVars(glob, vars)
	}
	return out
}
