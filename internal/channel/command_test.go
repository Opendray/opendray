package channel

import (
	"context"
	"strings"
	"testing"
)

func TestParseCommand(t *testing.T) {
	cases := []struct {
		text     string
		wantOK   bool
		wantName string
		wantArgs []string
	}{
		{"/help", true, "help", []string{}},
		{"/cancel abc-123", true, "cancel", []string{"abc-123"}},
		{"  /Cancel  abc  ", true, "cancel", []string{"abc"}},
		{"act:/help", true, "help", []string{}},
		{"act:cmd:/cancel abc", true, "cancel", []string{"abc"}},
		{"cmd:/notify off", true, "notify", []string{"off"}},
		{"hello world", false, "", nil},
		{"", false, "", nil},
		{"/", false, "", nil},
	}
	for _, c := range cases {
		name, args, ok := ParseCommand(c.text)
		if ok != c.wantOK || name != c.wantName {
			t.Errorf("ParseCommand(%q) = (%q,%v,%v), want (%q,%v,%v)",
				c.text, name, args, ok, c.wantName, c.wantArgs, c.wantOK)
			continue
		}
		if !c.wantOK {
			continue
		}
		if len(args) != len(c.wantArgs) {
			t.Errorf("ParseCommand(%q) args=%v, want %v", c.text, args, c.wantArgs)
			continue
		}
		for i := range args {
			if args[i] != c.wantArgs[i] {
				t.Errorf("ParseCommand(%q) args[%d]=%q, want %q", c.text, i, args[i], c.wantArgs[i])
			}
		}
	}
}

func TestRegistry_RegisterLookupList(t *testing.T) {
	r := NewCommandRegistry()
	if _, ok := r.Lookup("help"); !ok {
		t.Fatal("builtin /help missing")
	}
	r.Register(Command{
		Name:        "/Cancel",
		Description: "End a session",
		Handler:     func(_ context.Context, _ CommandContext) (string, error) { return "ok", nil },
	})
	if _, ok := r.Lookup("/cancel"); !ok {
		t.Fatal("/cancel not registered")
	}
	if c, ok := r.Lookup("CANCEL"); !ok || c.Description != "End a session" {
		t.Errorf("lookup case-insensitive: %+v ok=%v", c, ok)
	}
	names := make([]string, 0)
	for _, c := range r.List() {
		names = append(names, c.Name)
	}
	got := strings.Join(names, ",")
	if got != "cancel,help,start" {
		t.Errorf("List sorted: %s", got)
	}
}

func TestHelpHandler_ListsCommands(t *testing.T) {
	r := NewCommandRegistry()
	r.Register(Command{Name: "cancel", Description: "End a session", Handler: noopHandler})
	r.Register(Command{Name: "status", Description: "Show session status", Handler: noopHandler})

	cmd, _ := r.Lookup("help")
	out, err := cmd.Handler(context.Background(), CommandContext{})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"/help", "/cancel — End a session", "/status — Show session status"} {
		if !strings.Contains(out, want) {
			t.Errorf("help output missing %q\n--full--\n%s", want, out)
		}
	}
}

func TestStartHandler_GreetsAndListsCommands(t *testing.T) {
	r := NewCommandRegistry()
	cmd, ok := r.Lookup("start")
	if !ok {
		t.Fatal("/start should be registered by default")
	}
	out, err := cmd.Handler(context.Background(), CommandContext{})
	if err != nil {
		t.Fatal(err)
	}
	// Greeting up top, then the same command list as /help (incl. itself).
	for _, want := range []string{"opendray is connected", "/help", "/start"} {
		if !strings.Contains(out, want) {
			t.Errorf("/start output missing %q\n--full--\n%s", want, out)
		}
	}
}

func noopHandler(_ context.Context, _ CommandContext) (string, error) { return "", nil }
