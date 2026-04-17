package llmproxy

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestAnthropicToOpenAI_BasicText(t *testing.T) {
	in := AnthropicRequest{
		Model:     "claude-sonnet-4",
		MaxTokens: 512,
		System:    json.RawMessage(`"you are a helpful assistant"`),
		Messages: []AnthropicMsg{
			{Role: "user", Content: json.RawMessage(`"hello"`)},
		},
	}
	out, err := anthropicToOpenAI(in, "qwen3-coder:30b")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if out.Model != "qwen3-coder:30b" {
		t.Errorf("want model override; got %q", out.Model)
	}
	if len(out.Messages) != 2 {
		t.Fatalf("want 2 messages (system+user); got %d", len(out.Messages))
	}
	if out.Messages[0].Role != "system" || out.Messages[0].Content != "you are a helpful assistant" {
		t.Errorf("system message off: %+v", out.Messages[0])
	}
	if out.Messages[1].Role != "user" || out.Messages[1].Content != "hello" {
		t.Errorf("user message off: %+v", out.Messages[1])
	}
}

func TestAnthropicToOpenAI_ToolUseRoundTrip(t *testing.T) {
	// Simulate a second-turn request: assistant called a tool, user sent
	// a tool_result. Both should translate into the OpenAI tool protocol.
	in := AnthropicRequest{
		Model:     "claude-sonnet-4",
		MaxTokens: 512,
		Messages: []AnthropicMsg{
			{Role: "user", Content: json.RawMessage(`"run ls"`)},
			{Role: "assistant", Content: json.RawMessage(`[
				{"type":"text","text":"I will run ls."},
				{"type":"tool_use","id":"toolu_abc","name":"bash","input":{"cmd":"ls"}}
			]`)},
			{Role: "user", Content: json.RawMessage(`[
				{"type":"tool_result","tool_use_id":"toolu_abc","content":"file1\nfile2"}
			]`)},
		},
		Tools: []AnthropicTool{{
			Name:        "bash",
			Description: "run a shell command",
			InputSchema: json.RawMessage(`{"type":"object","properties":{"cmd":{"type":"string"}}}`),
		}},
	}
	out, err := anthropicToOpenAI(in, "qwen3-coder:30b")
	if err != nil {
		t.Fatalf("translate: %v", err)
	}
	// Expect: user(run ls), assistant(text + 1 tool_call), tool(result)
	if len(out.Messages) != 3 {
		t.Fatalf("want 3 messages; got %d: %+v", len(out.Messages), out.Messages)
	}
	assistant := out.Messages[1]
	if assistant.Role != "assistant" {
		t.Errorf("want assistant, got %q", assistant.Role)
	}
	if !strings.Contains(assistant.Content, "I will run ls") {
		t.Errorf("assistant text lost: %q", assistant.Content)
	}
	if len(assistant.ToolCalls) != 1 || assistant.ToolCalls[0].Function.Name != "bash" {
		t.Errorf("assistant tool_calls off: %+v", assistant.ToolCalls)
	}
	if assistant.ToolCalls[0].ID != "toolu_abc" {
		t.Errorf("tool call id should round-trip; got %q", assistant.ToolCalls[0].ID)
	}
	tool := out.Messages[2]
	if tool.Role != "tool" || tool.ToolCallID != "toolu_abc" || tool.Content != "file1\nfile2" {
		t.Errorf("tool result off: %+v", tool)
	}
	if len(out.Tools) != 1 || out.Tools[0].Function.Name != "bash" {
		t.Errorf("tool definition not forwarded: %+v", out.Tools)
	}
}

func TestOpenAIToAnthropicResponse_ToolCall(t *testing.T) {
	in := OpenAIResponse{
		ID:    "chatcmpl-1",
		Model: "qwen3-coder:30b",
		Choices: []OpenAIChoice{{
			Index: 0,
			Message: OpenAIMessage{
				Role:    "assistant",
				Content: "I'll run the command.",
				ToolCalls: []OpenAIToolCall{{
					ID:   "call_x",
					Type: "function",
					Function: struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					}{Name: "bash", Arguments: `{"cmd":"ls"}`},
				}},
			},
			FinishReason: "tool_calls",
		}},
		Usage: OpenAIUsage{PromptTokens: 10, CompletionTokens: 5},
	}
	out := openAIToAnthropicResponse(in, "qwen3-coder:30b")
	if out.StopReason != "tool_use" {
		t.Errorf("want stop_reason=tool_use; got %q", out.StopReason)
	}
	if len(out.Content) != 2 {
		t.Fatalf("want 2 content blocks (text+tool_use); got %d", len(out.Content))
	}
	if out.Content[0].Type != "text" || !strings.Contains(out.Content[0].Text, "run the command") {
		t.Errorf("text block off: %+v", out.Content[0])
	}
	if out.Content[1].Type != "tool_use" || out.Content[1].Name != "bash" {
		t.Errorf("tool_use block off: %+v", out.Content[1])
	}
	if string(out.Content[1].Input) != `{"cmd":"ls"}` {
		t.Errorf("tool input should round-trip; got %s", string(out.Content[1].Input))
	}
}

func TestFlattenSystemAsBlocks(t *testing.T) {
	// The CLI often sends system as a list of cache-control blocks.
	raw := json.RawMessage(`[
		{"type":"text","text":"part one"},
		{"type":"text","text":"part two"}
	]`)
	got := flattenTextMaybeBlocks(raw)
	if got != "part one\npart two" {
		t.Errorf("want joined system text; got %q", got)
	}
}

func TestMapFinishReason(t *testing.T) {
	cases := map[string]string{
		"stop":          "end_turn",
		"length":        "max_tokens",
		"tool_calls":    "tool_use",
		"function_call": "tool_use",
		"":              "end_turn",
		"weird":         "end_turn",
	}
	for in, want := range cases {
		if got := mapFinishReason(in); got != want {
			t.Errorf("%q: want %q, got %q", in, want, got)
		}
	}
}
