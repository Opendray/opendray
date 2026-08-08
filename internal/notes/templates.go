package notes

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Templates exist so a project's docs come out looking like each other.
// The alternative — every doc starting from an empty file — is how a
// vault ends up with five different ideas of what a feature note is.
//
// Rendering happens HERE, not in the clients. The frontmatter carries a
// date and a title derived from the path, and asking web and mobile to
// each substitute those correctly is asking them to drift; this repo
// has watched that happen before. One endpoint, one implementation.
//
// Built-ins are a starting point, not a policy: dropping
// `_templates/<id>.md` in the vault overrides one, and adding a new
// file there adds a template. A project whose docs want a different
// shape shouldn't need a gateway release.

// TemplateDir is the vault folder scanned for user-authored templates.
// Leading underscore keeps it sorting away from real notes and hints
// that it isn't documentation itself.
const TemplateDir = "_templates"

// Template is one starting shape for a new note.
type Template struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// Source is "builtin" or "vault", so the UI can show which
	// templates the operator has taken ownership of.
	Source string `json:"source"`
	// Body is the raw, unrendered content. Exposed so a client can
	// preview a template without creating a file.
	Body string `json:"body"`
	// Kind is the document kind this template produces. Built-ins are
	// markdown; a vault template's kind comes from its own extension,
	// so `_templates/report.html` starts an HTML document.
	Kind Kind `json:"kind"`
}

// Placeholders substituted at render time. Deliberately few: a template
// nobody can predict the output of is worse than no template.
//
//	{{title}}  Title-cased filename, e.g. "host-power.md" → "Host power"
//	{{slug}}   Bare filename without extension, e.g. "host-power"
//	{{date}}   Today, YYYY-MM-DD, host-local
//	{{path}}   Vault-relative path of the note being created
const (
	phTitle = "{{title}}"
	phSlug  = "{{slug}}"
	phDate  = "{{date}}"
	phPath  = "{{path}}"
)

// builtinTemplates are the shapes opendray's own project docs use.
var builtinTemplates = []Template{
	{
		ID:     "blank",
		Name:   "Blank",
		Source: "builtin",
		Kind:   KindMarkdown,
		Body:   "# {{title}}\n\n",
	},
	{
		ID:     "feature",
		Name:   "Feature",
		Source: "builtin",
		Kind:   KindMarkdown,
		Body: `---
type: feature
status: draft
created: {{date}}
---

# {{title}}

<!-- What problem does this solve, for whom? Lead with the problem,
     not the mechanism. -->

## How it works

## Configuration

## Gotchas

<!-- What will surprise the next person? Prefer writing the trap down
     over trusting they won't fall in it. -->
`,
	},
	{
		ID:     "decision",
		Name:   "Decision (ADR)",
		Source: "builtin",
		Kind:   KindMarkdown,
		Body: `---
type: decision
status: proposed
created: {{date}}
---

# {{title}}

## Context

<!-- What forced a decision? What was true at the time? -->

## Decision

## Consequences

<!-- Including the ones you don't like. A decision record that only
     lists upsides isn't one. -->

## Alternatives considered
`,
	},
	{
		ID:     "runbook",
		Name:   "Runbook",
		Source: "builtin",
		Kind:   KindMarkdown,
		Body: `---
type: runbook
created: {{date}}
---

# {{title}}

## When to use this

## Steps

1.

## Verification

<!-- How do you know it worked? An unverifiable runbook is a wish. -->

## If it goes wrong
`,
	},
}

// Templates returns the available templates, vault overrides merged
// over the built-ins and sorted so "blank" leads.
func (v *Vault) Templates() []Template {
	byID := map[string]Template{}
	order := []string{}
	for _, t := range builtinTemplates {
		byID[t.ID] = t
		order = append(order, t.ID)
	}

	dir := filepath.Join(v.root, TemplateDir)
	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, e := range entries {
			if e.IsDir() || !IsDocument(e.Name()) {
				continue
			}
			body, err := os.ReadFile(filepath.Join(dir, e.Name()))
			if err != nil {
				continue
			}
			id := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			if _, seen := byID[id]; !seen {
				order = append(order, id)
			}
			byID[id] = Template{
				ID:     id,
				Name:   titleCase(id),
				Source: "vault",
				Body:   string(body),
				Kind:   KindOf(e.Name()),
			}
		}
	}

	out := make([]Template, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	// Built-in order is meaningful (blank first); vault-only additions
	// trail it alphabetically rather than in readdir order.
	builtinCount := 0
	for _, t := range out {
		if t.Source == "builtin" {
			builtinCount++
		}
	}
	tail := out[builtinCount:]
	sort.Slice(tail, func(i, j int) bool { return tail[i].ID < tail[j].ID })
	return out
}

// NewFromTemplate creates rel from the named template. It refuses to
// overwrite: "new" that silently replaces an existing doc is a data
// loss bug waiting for someone to retype a filename.
func (v *Vault) NewFromTemplate(rel, templateID string) (Note, error) {
	if err := requireDocument(rel); err != nil {
		return Note{}, err
	}
	full, err := v.resolve(rel)
	if err != nil {
		return Note{}, err
	}
	if _, err := os.Stat(full); err == nil {
		return Note{}, ErrAlreadyExists
	} else if !os.IsNotExist(err) {
		return Note{}, err
	}

	if templateID == "" {
		templateID = "blank"
	}
	var tpl *Template
	for _, t := range v.Templates() {
		if t.ID == templateID {
			tpl = &t
			break
		}
	}
	if tpl == nil {
		return Note{}, fmt.Errorf("%w: unknown template %q", ErrInvalidPath, templateID)
	}

	body := tpl.Body
	// A markdown template inside an .html file produces a document that
	// renders as literal `# Title` in a browser. When the two disagree,
	// the FILENAME wins — the operator chose the extension, the
	// template was very likely just the default.
	if KindOf(rel) == KindHTML && tpl.Kind != KindHTML {
		body = htmlSkeleton
	}
	return v.Write(rel, renderTemplate(body, rel))
}

// htmlSkeleton is what an HTML document starts from when the chosen
// template is markdown. Minimal on purpose: an HTML doc in the vault
// usually arrives from an export tool, and anything opendray adds here
// is something the operator has to delete.
const htmlSkeleton = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>` + phTitle + `</title>
</head>
<body>
<h1>` + phTitle + `</h1>
</body>
</html>
`

// renderTemplate substitutes the placeholders for one target path.
func renderTemplate(body, rel string) string {
	base := filepath.Base(rel)
	slug := strings.TrimSuffix(base, filepath.Ext(base))
	r := strings.NewReplacer(
		phTitle, titleCase(slug),
		phSlug, slug,
		phDate, time.Now().Format("2006-01-02"),
		phPath, rel,
	)
	return r.Replace(body)
}

// A folder's index page (README.md / index.md) is resolved client-side:
// both surfaces already hold the full listing for the folder they are
// rendering, so asking the gateway "does this folder have an index"
// would be a round trip to re-derive what the caller can already see.

// titleCase turns "host-power" into "Host power" — readable as a
// heading without pretending to know English capitalisation rules.
func titleCase(slug string) string {
	s := strings.TrimSpace(strings.NewReplacer("-", " ", "_", " ").Replace(slug))
	if s == "" {
		return slug
	}
	runes := []rune(s)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}
