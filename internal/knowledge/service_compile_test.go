package knowledge

import (
	"context"
	"log/slog"
	"testing"
)

type fakeSkillSink struct {
	skills map[string]string
	assets map[string]string // "<id>/<name>" -> content
}

func newFakeSkillSink() *fakeSkillSink {
	return &fakeSkillSink{skills: map[string]string{}, assets: map[string]string{}}
}

func (f *fakeSkillSink) WriteSkill(_ context.Context, id, markdown string) error {
	f.skills[id] = markdown
	return nil
}

func (f *fakeSkillSink) WriteSkillAsset(_ context.Context, id, name, content string) error {
	f.assets[id+"/"+name] = content
	return nil
}

func (f *fakeSkillSink) DeleteSkill(_ context.Context, id string) error {
	delete(f.skills, id)
	return nil
}

type fakeTaskSink struct {
	tasks map[string]string // slug -> cwd
}

func (f *fakeTaskSink) EnsureSkillTask(_ context.Context, slug, _, _, cwd string) error {
	f.tasks[slug] = cwd
	return nil
}

func TestMaterialiseCompiledForm(t *testing.T) {
	sink := newFakeSkillSink()
	tasks := &fakeTaskSink{tasks: map[string]string{}}
	svc := &Service{skillSink: sink, taskSink: tasks, log: slog.Default()}

	// A compiled, project-scoped skill ships run.sh and a cwd-scoped task.
	svc.materialiseCompiledForm(context.Background(), "deploy-api", Node{
		Title:    "Deploy the API",
		Scope:    ScopeProject,
		ScopeKey: "/work/api",
		Provenance: map[string]any{
			"script": "#!/usr/bin/env bash\nset -euo pipefail\n\nmake deploy\ncurl -fsS http://127.0.0.1:8080/healthz\n",
		},
	})
	if _, ok := sink.assets["deploy-api/run.sh"]; !ok {
		t.Fatal("run.sh asset not written")
	}
	if cwd := tasks.tasks["deploy-api"]; cwd != "/work/api" {
		t.Fatalf("task cwd = %q, want /work/api", cwd)
	}

	// A global skill registers a global task.
	svc.materialiseCompiledForm(context.Background(), "global-skill", Node{
		Title:      "Global thing",
		Scope:      ScopeGlobal,
		Provenance: map[string]any{"script": "echo hi"},
	})
	if cwd, ok := tasks.tasks["global-skill"]; !ok || cwd != "" {
		t.Fatalf("global task cwd = %q, want empty", cwd)
	}

	// No script → nothing happens (prose skills stay prose).
	before := len(sink.assets)
	svc.materialiseCompiledForm(context.Background(), "prose-skill", Node{Title: "Prose"})
	if len(sink.assets) != before {
		t.Fatal("prose skill must not produce assets")
	}
	if _, ok := tasks.tasks["prose-skill"]; ok {
		t.Fatal("prose skill must not register a task")
	}
}
