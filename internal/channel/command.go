package channel

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// CommandHandler runs a registered command and returns plain text.
// When the reply is non-empty the Hub posts it back to the
// originating channel (with ReplyCtx preserved).
//
// Long-running handlers should respect ctx.Done() and return promptly.
type CommandHandler func(ctx context.Context, cc CommandContext) (reply string, err error)

// CommandCardHandler is the structured-reply alternative to
// CommandHandler. Channels that implement CardSender (Telegram,
// Slack, ...) render the returned Card natively — most usefully,
// CardActions become inline-keyboard buttons so the user can
// trigger follow-up commands by tapping rather than typing an
// opaque session id. Channels without CardSender fall back to
// Card.RenderText().
//
// Mutually exclusive with Handler — set exactly one per Command.
type CommandCardHandler func(ctx context.Context, cc CommandContext) (*Card, error)

// CommandContext is the dispatch envelope passed to each handler.
type CommandContext struct {
	Channel Channel
	Message ChannelMessage
	Hub     *Hub
	Command string   // canonical name, lowercased, leading "/" stripped
	Args    []string // whitespace-split tail after the command name
	Raw     string   // full original text (e.g. "/help arg1 arg2")
}

// Command describes one registered slash command. Exactly one of
// Handler or CardHandler must be set; CardHandler takes precedence
// when both are provided (config error — logged).
type Command struct {
	Name        string             // lowercased, no leading "/"
	Description string             // shown in /help
	Handler     CommandHandler     // plain-text reply
	CardHandler CommandCardHandler // structured reply (cards + buttons)
	Source      string             // "builtin" | "session" | "custom"
}

// CommandRegistry holds the set of available slash commands.
type CommandRegistry struct {
	mu       sync.RWMutex
	commands map[string]Command
}

// NewCommandRegistry returns a registry pre-populated with the
// built-in commands (/help and /notify) and nothing else.
//
// App code that knows how to manipulate sessions wires its own
// commands by calling Register or Hub.RegisterCommand.
func NewCommandRegistry() *CommandRegistry {
	r := &CommandRegistry{commands: make(map[string]Command)}
	r.Register(Command{
		Name:        "help",
		Description: "List available commands",
		Source:      "builtin",
		Handler:     helpHandler(r),
	})
	// /start is what Telegram (and most chat platforms) send when a
	// user first opens the bot, so it must resolve to something
	// friendly instead of "Unknown command". It greets, then prints
	// the same command list as /help.
	r.Register(Command{
		Name:        "start",
		Description: "Welcome message + command list",
		Source:      "builtin",
		Handler:     startHandler(r),
	})
	return r
}

// Register adds (or replaces) a command in the registry.
func (r *CommandRegistry) Register(cmd Command) {
	if cmd.Name == "" {
		return
	}
	cmd.Name = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(cmd.Name), "/"))
	if cmd.Source == "" {
		cmd.Source = "custom"
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.commands[cmd.Name] = cmd
}

// Lookup returns (command, true) when name is registered.
func (r *CommandRegistry) Lookup(name string) (Command, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.commands[strings.ToLower(strings.TrimPrefix(strings.TrimSpace(name), "/"))]
	return c, ok
}

// List returns every registered command, sorted by name.
func (r *CommandRegistry) List() []Command {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Command, 0, len(r.commands))
	for _, c := range r.commands {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// ParseCommand extracts a (name, args) tuple from text. Recognises:
//
//   - Plain text starting with "/" — "/help arg1 arg2"
//   - Button callback values prefixed by EncodeAction — "act:cmd:/help"
//   - Embedded "cmd:" payloads inside button data — "cmd:/help"
//
// Returns ok=false when the input is not a command.
func ParseCommand(text string) (name string, args []string, ok bool) {
	s := strings.TrimSpace(text)
	if s == "" {
		return "", nil, false
	}
	if action, isAction := DecodeAction(s); isAction {
		s = strings.TrimSpace(action)
	}
	s = strings.TrimPrefix(s, "cmd:")
	if !strings.HasPrefix(s, "/") {
		return "", nil, false
	}
	s = strings.TrimPrefix(s, "/")
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return "", nil, false
	}
	return strings.ToLower(fields[0]), fields[1:], true
}

func helpHandler(r *CommandRegistry) CommandHandler {
	return func(_ context.Context, _ CommandContext) (string, error) {
		var b strings.Builder
		b.WriteString("Available commands:\n")
		for _, c := range r.List() {
			fmt.Fprintf(&b, "  /%s — %s\n", c.Name, c.Description)
		}
		return strings.TrimRight(b.String(), "\n"), nil
	}
}

// startHandler greets the user and appends the /help command list.
// Wired to /start so the bot's first-contact command is never an
// "Unknown command" error.
func startHandler(r *CommandRegistry) CommandHandler {
	help := helpHandler(r)
	return func(ctx context.Context, cc CommandContext) (string, error) {
		body, err := help(ctx, cc)
		if err != nil {
			return "", err
		}
		return "👋 opendray is connected. Use /list to see sessions, then tap “💬 Talk to” to pick one.\n\n" + body, nil
	}
}
