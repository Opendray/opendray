package llmproxy

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ── Anthropic Messages API (subset we parse) ────────────────────────

// AnthropicRequest is the body Claude CLI POSTs to /v1/messages.
//
// We keep extra fields in a generic map so future CLI additions (e.g.
// metadata, top_k) pass through without breaking the translator — only
// fields we actually map are strongly typed.
type AnthropicRequest struct {
	Model       string           `json:"model"`
	MaxTokens   int              `json:"max_tokens"`
	Temperature *float64         `json:"temperature,omitempty"`
	TopP        *float64         `json:"top_p,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
	System      json.RawMessage  `json:"system,omitempty"` // string OR []ContentBlock
	Messages    []AnthropicMsg   `json:"messages"`
	Tools       []AnthropicTool  `json:"tools,omitempty"`
	ToolChoice  json.RawMessage  `json:"tool_choice,omitempty"`
	StopSeqs    []string         `json:"stop_sequences,omitempty"`
}

type AnthropicMsg struct {
	Role    string          `json:"role"`    // "user" | "assistant"
	Content json.RawMessage `json:"content"` // string OR []ContentBlock
}

// ContentBlock covers the five variants the CLI actually sends.
type ContentBlock struct {
	Type string `json:"type"`

	// text
	Text string `json:"text,omitempty"`

	// tool_use
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`

	// tool_result
	ToolUseID    string          `json:"tool_use_id,omitempty"`
	ToolContent  json.RawMessage `json:"content,omitempty"` // string or []block
	ToolIsError  bool            `json:"is_error,omitempty"`

	// image (not yet forwarded — local models rarely accept images
	// in a portable way. We stringify with a placeholder.)
	Source json.RawMessage `json:"source,omitempty"`
}

type AnthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	InputSchema json.RawMessage `json:"input_schema"`
}

// AnthropicResponse is the non-streaming reply shape we emit.
type AnthropicResponse struct {
	ID           string         `json:"id"`
	Type         string         `json:"type"`
	Role         string         `json:"role"`
	Model        string         `json:"model"`
	Content      []ContentBlock `json:"content"`
	StopReason   string         `json:"stop_reason"`
	StopSequence *string        `json:"stop_sequence"`
	Usage        AnthropicUsage `json:"usage"`
}

type AnthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ── OpenAI Chat Completions (subset) ────────────────────────────────

type OpenAIRequest struct {
	Model       string          `json:"model"`
	Messages    []OpenAIMessage `json:"messages"`
	MaxTokens   int             `json:"max_tokens,omitempty"`
	Temperature *float64        `json:"temperature,omitempty"`
	TopP        *float64        `json:"top_p,omitempty"`
	Stream      bool            `json:"stream,omitempty"`
	Tools       []OpenAITool    `json:"tools,omitempty"`
	ToolChoice  json.RawMessage `json:"tool_choice,omitempty"`
	Stop        []string        `json:"stop,omitempty"`
}

type OpenAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	Name       string           `json:"name,omitempty"`
}

type OpenAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON-encoded string
	} `json:"function"`
}

type OpenAITool struct {
	Type     string            `json:"type"` // always "function"
	Function OpenAIToolFuncDef `json:"function"`
}

type OpenAIToolFuncDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type OpenAIResponse struct {
	ID      string         `json:"id"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   OpenAIUsage    `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

type OpenAIUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ── Request translation ─────────────────────────────────────────────

func anthropicToOpenAI(in AnthropicRequest, model string) (OpenAIRequest, error) {
	out := OpenAIRequest{
		Model:       model,
		MaxTokens:   in.MaxTokens,
		Temperature: in.Temperature,
		TopP:        in.TopP,
		Stream:      in.Stream,
		Stop:        in.StopSeqs,
	}

	// System prompt: Anthropic accepts either a string or a list of
	// content blocks ("cache" blocks for prompt caching). We flatten
	// to a single system message.
	if sys := flattenTextMaybeBlocks(in.System); sys != "" {
		out.Messages = append(out.Messages, OpenAIMessage{Role: "system", Content: sys})
	}

	for _, m := range in.Messages {
		blocks, asText, err := parseContent(m.Content)
		if err != nil {
			return OpenAIRequest{}, fmt.Errorf("message content: %w", err)
		}

		switch m.Role {
		case "user":
			// User messages can carry text AND tool_result blocks in
			// the same message. OpenAI splits these: tool results go
			// in separate role=tool messages, free text stays as a
			// user message.
			var userText strings.Builder
			if asText != "" {
				userText.WriteString(asText)
			}
			var tailToolMsgs []OpenAIMessage
			for _, b := range blocks {
				switch b.Type {
				case "text":
					if userText.Len() > 0 {
						userText.WriteString("\n")
					}
					userText.WriteString(b.Text)
				case "tool_result":
					tailToolMsgs = append(tailToolMsgs, OpenAIMessage{
						Role:       "tool",
						ToolCallID: b.ToolUseID,
						Content:    toolResultText(b),
					})
				case "image":
					if userText.Len() > 0 {
						userText.WriteString("\n")
					}
					userText.WriteString("[image omitted — proxy does not forward images to local models]")
				}
			}
			if userText.Len() > 0 {
				out.Messages = append(out.Messages, OpenAIMessage{Role: "user", Content: userText.String()})
			}
			out.Messages = append(out.Messages, tailToolMsgs...)

		case "assistant":
			msg := OpenAIMessage{Role: "assistant"}
			var textBuf strings.Builder
			if asText != "" {
				textBuf.WriteString(asText)
			}
			for _, b := range blocks {
				switch b.Type {
				case "text":
					if textBuf.Len() > 0 {
						textBuf.WriteString("\n")
					}
					textBuf.WriteString(b.Text)
				case "tool_use":
					args := string(b.Input)
					if args == "" {
						args = "{}"
					}
					msg.ToolCalls = append(msg.ToolCalls, OpenAIToolCall{
						ID:   b.ID,
						Type: "function",
						Function: struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						}{Name: b.Name, Arguments: args},
					})
				}
			}
			msg.Content = textBuf.String()
			out.Messages = append(out.Messages, msg)
		}
	}

	for _, t := range in.Tools {
		out.Tools = append(out.Tools, OpenAITool{
			Type: "function",
			Function: OpenAIToolFuncDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	if len(in.ToolChoice) > 0 {
		out.ToolChoice = in.ToolChoice // OpenAI accepts similar shape; passthrough
	}

	return out, nil
}

// parseContent accepts the Anthropic content field in either form:
//   - a bare string: "hello"
//   - a list of content blocks
//
// asText is non-empty only for the string form; blocks is non-empty
// only for the list form.
func parseContent(raw json.RawMessage) (blocks []ContentBlock, asText string, err error) {
	if len(raw) == 0 {
		return nil, "", nil
	}
	// String shortcut
	if raw[0] == '"' {
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil, "", err
		}
		return nil, s, nil
	}
	// Array of blocks
	var bs []ContentBlock
	if err := json.Unmarshal(raw, &bs); err != nil {
		return nil, "", err
	}
	return bs, "", nil
}

// flattenTextMaybeBlocks flattens a system field that may be a string
// or a list of {type:"text", text:"..."} blocks into a single string.
func flattenTextMaybeBlocks(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if raw[0] == '"' {
		var s string
		_ = json.Unmarshal(raw, &s)
		return s
	}
	var bs []ContentBlock
	if err := json.Unmarshal(raw, &bs); err != nil {
		return ""
	}
	var b strings.Builder
	for _, blk := range bs {
		if blk.Type == "text" && blk.Text != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(blk.Text)
		}
	}
	return b.String()
}

// toolResultText coerces the tool_result content (which may be a
// string or a list of {type:"text", text:"..."} blocks per the
// Anthropic spec) into the plain-string shape OpenAI expects.
func toolResultText(b ContentBlock) string {
	if len(b.ToolContent) == 0 {
		return ""
	}
	if b.ToolContent[0] == '"' {
		var s string
		_ = json.Unmarshal(b.ToolContent, &s)
		if b.ToolIsError {
			return "ERROR: " + s
		}
		return s
	}
	var bs []ContentBlock
	if err := json.Unmarshal(b.ToolContent, &bs); err != nil {
		return string(b.ToolContent)
	}
	var sb strings.Builder
	for _, inner := range bs {
		if inner.Type == "text" {
			if sb.Len() > 0 {
				sb.WriteString("\n")
			}
			sb.WriteString(inner.Text)
		}
	}
	out := sb.String()
	if b.ToolIsError {
		return "ERROR: " + out
	}
	return out
}

// ── Response translation (non-streaming) ────────────────────────────

func openAIToAnthropicResponse(in OpenAIResponse, model string) AnthropicResponse {
	out := AnthropicResponse{
		ID:    "msg_" + randID(),
		Type:  "message",
		Role:  "assistant",
		Model: model,
		Usage: AnthropicUsage{
			InputTokens:  in.Usage.PromptTokens,
			OutputTokens: in.Usage.CompletionTokens,
		},
	}
	if len(in.Choices) == 0 {
		out.StopReason = "end_turn"
		return out
	}
	msg := in.Choices[0].Message
	if strings.TrimSpace(msg.Content) != "" {
		out.Content = append(out.Content, ContentBlock{Type: "text", Text: msg.Content})
	}
	for _, tc := range msg.ToolCalls {
		input := json.RawMessage(tc.Function.Arguments)
		if len(input) == 0 {
			input = json.RawMessage("{}")
		}
		id := tc.ID
		if id == "" {
			id = "toolu_" + randID()
		}
		out.Content = append(out.Content, ContentBlock{
			Type:  "tool_use",
			ID:    id,
			Name:  tc.Function.Name,
			Input: input,
		})
	}
	out.StopReason = mapFinishReason(in.Choices[0].FinishReason)
	return out
}

func mapFinishReason(openaiFinish string) string {
	switch openaiFinish {
	case "tool_calls", "function_call":
		return "tool_use"
	case "length":
		return "max_tokens"
	case "stop", "":
		return "end_turn"
	default:
		return "end_turn"
	}
}
