package knowledge

import (
	"context"
	"strings"
)

// SkillSink persists a rendered SKILL.md so the skills loader (internal/skills)
// can pick it up as a vault skill (<skills>/<id>/SKILL.md). WriteSkillAsset
// places an extra file next to SKILL.md — the experience compiler ships a
// skill's executable form as <skills>/<id>/run.sh. The app wires a filesystem
// impl; knowledge owns the interface so it never imports internal/skills
// (the one-way dependency rule).
type SkillSink interface {
	WriteSkill(ctx context.Context, id, markdown string) error
	WriteSkillAsset(ctx context.Context, id, name, content string) error
	DeleteSkill(ctx context.Context, id string) error
}

// TaskSink registers a compiled skill's runnable form as an opendray custom
// task so the operator (and integrations) can run it directly. cwd="" means
// global. The app adapts internal/customtask; best-effort by design.
type TaskSink interface {
	EnsureSkillTask(ctx context.Context, slug, title, description, cwd string) error
}

// skillSlug turns a title into a skills-loader id / dirname (lowercase,
// hyphenated, alphanumeric).
func skillSlug(title string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(title)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash && b.Len() > 0 {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	s := strings.Trim(b.String(), "-")
	if s == "" {
		return "skill"
	}
	if len(s) > 60 {
		s = strings.Trim(s[:60], "-")
	}
	return s
}

// renderSkillMarkdown builds a SKILL.md (frontmatter + body) for the loader.
// description is quoted + sanitised so it stays valid YAML on one line.
func renderSkillMarkdown(name, description, body string) string {
	desc := strings.ReplaceAll(strings.TrimSpace(description), "\n", " ")
	desc = strings.ReplaceAll(desc, `"`, `'`)
	if len(desc) > 200 {
		desc = strings.TrimSpace(desc[:200])
	}
	var b strings.Builder
	b.WriteString("---\nname: ")
	b.WriteString(name)
	b.WriteString("\ndescription: \"")
	b.WriteString(desc)
	b.WriteString("\"\n---\n\n# ")
	b.WriteString(name)
	b.WriteString("\n\n")
	b.WriteString(strings.TrimSpace(body))
	b.WriteByte('\n')
	return b.String()
}

// skillDescription derives a one-line description from a playbook body,
// falling back to the title.
func skillDescription(title, body string) string {
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		t = strings.TrimSpace(strings.TrimPrefix(t, "**Applies when:**"))
		t = strings.TrimSpace(strings.Trim(t, "*#-"))
		if t != "" && !strings.HasPrefix(strings.TrimSpace(line), "##") {
			return t
		}
	}
	return title
}
