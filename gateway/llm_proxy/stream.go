package llmproxy

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
)

// streamOpenAIToAnthropic reads an OpenAI SSE stream from upstream and
// emits Anthropic-format SSE events to the client.
//
// Anthropic's SSE protocol is event-oriented (one named event per
// content block lifecycle stage) whereas OpenAI sends a flat sequence
// of delta chunks. We buffer deltas per content block and emit:
//
//	event: message_start          {message: {id, role, model, usage}}
//	event: content_block_start    {index, content_block: {type, ...}}
//	event: content_block_delta    {index, delta: {...}}    × N
//	event: content_block_stop     {index}
//	(repeat start/delta/stop per tool call)
//	event: message_delta          {delta: {stop_reason}, usage}
//	event: message_stop           {}
//
// Text and tool_use are emitted as separate content blocks with
// monotonically-increasing indexes. We start the text block only on
// the first non-empty content delta so we don't emit an empty block
// when the model immediately calls a tool.
func streamOpenAIToAnthropic(w http.ResponseWriter, body io.Reader, model string, logger *slog.Logger) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.WriteHeader(http.StatusOK)

	msgID := "msg_" + randID()

	// message_start
	writeEvent(w, flusher, "message_start", map[string]any{
		"type": "message_start",
		"message": map[string]any{
			"id":            msgID,
			"type":          "message",
			"role":          "assistant",
			"model":         model,
			"content":       []any{},
			"stop_reason":   nil,
			"stop_sequence": nil,
			"usage":         map[string]any{"input_tokens": 0, "output_tokens": 0},
		},
	})

	// Per-content-block state.
	textIndex := -1
	textOpen := false

	// tool_use blocks keyed by OpenAI tool-call index.
	type toolState struct {
		index    int
		id       string
		name     string
		args     strings.Builder
		opened   bool
	}
	toolStates := map[int]*toolState{}
	nextIndex := 0 // next content-block index to allocate

	finishReason := ""
	var usage OpenAIUsage

	scanner := bufio.NewScanner(body)
	// Some providers emit very large single SSE frames (tool argument
	// chunks for large JSON). Raise the line buffer.
	scanner.Buffer(make([]byte, 64*1024), 4*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chunk struct {
			Choices []struct {
				Index int `json:"index"`
				Delta struct {
					Content   string `json:"content"`
					ToolCalls []struct {
						Index    int    `json:"index"`
						ID       string `json:"id,omitempty"`
						Type     string `json:"type,omitempty"`
						Function struct {
							Name      string `json:"name,omitempty"`
							Arguments string `json:"arguments,omitempty"`
						} `json:"function"`
					} `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason,omitempty"`
			} `json:"choices"`
			Usage *OpenAIUsage `json:"usage,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			logger.Warn("llm proxy: bad sse chunk", "err", err, "data", data)
			continue
		}
		if chunk.Usage != nil {
			usage = *chunk.Usage
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		ch := chunk.Choices[0]
		if ch.FinishReason != "" {
			finishReason = ch.FinishReason
		}

		// Text delta.
		if ch.Delta.Content != "" {
			if !textOpen {
				textIndex = nextIndex
				nextIndex++
				textOpen = true
				writeEvent(w, flusher, "content_block_start", map[string]any{
					"type":          "content_block_start",
					"index":         textIndex,
					"content_block": map[string]any{"type": "text", "text": ""},
				})
			}
			writeEvent(w, flusher, "content_block_delta", map[string]any{
				"type":  "content_block_delta",
				"index": textIndex,
				"delta": map[string]any{"type": "text_delta", "text": ch.Delta.Content},
			})
		}

		// Tool-call deltas.
		for _, tc := range ch.Delta.ToolCalls {
			st, exists := toolStates[tc.Index]
			if !exists {
				// First time we see this tool call — close any open
				// text block, open a new tool_use block.
				if textOpen {
					writeEvent(w, flusher, "content_block_stop", map[string]any{
						"type":  "content_block_stop",
						"index": textIndex,
					})
					textOpen = false
				}
				st = &toolState{index: nextIndex, id: tc.ID, name: tc.Function.Name}
				nextIndex++
				toolStates[tc.Index] = st
			}
			if tc.ID != "" && st.id == "" {
				st.id = tc.ID
			}
			if tc.Function.Name != "" && st.name == "" {
				st.name = tc.Function.Name
			}
			// OpenAI may send id/name before the first arguments
			// chunk. Emit content_block_start only once we've seen
			// enough to populate the header.
			if !st.opened && st.name != "" {
				id := st.id
				if id == "" {
					id = "toolu_" + randID()
					st.id = id
				}
				writeEvent(w, flusher, "content_block_start", map[string]any{
					"type":  "content_block_start",
					"index": st.index,
					"content_block": map[string]any{
						"type":  "tool_use",
						"id":    id,
						"name":  st.name,
						"input": map[string]any{},
					},
				})
				st.opened = true
			}
			if tc.Function.Arguments != "" && st.opened {
				st.args.WriteString(tc.Function.Arguments)
				writeEvent(w, flusher, "content_block_delta", map[string]any{
					"type":  "content_block_delta",
					"index": st.index,
					"delta": map[string]any{"type": "input_json_delta", "partial_json": tc.Function.Arguments},
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		logger.Warn("llm proxy: stream scan error", "err", err)
	}

	// Close any still-open blocks.
	if textOpen {
		writeEvent(w, flusher, "content_block_stop", map[string]any{
			"type":  "content_block_stop",
			"index": textIndex,
		})
	}
	for _, st := range toolStates {
		if st.opened {
			writeEvent(w, flusher, "content_block_stop", map[string]any{
				"type":  "content_block_stop",
				"index": st.index,
			})
		}
	}

	// message_delta with final stop_reason + usage, then message_stop.
	writeEvent(w, flusher, "message_delta", map[string]any{
		"type": "message_delta",
		"delta": map[string]any{
			"stop_reason":   mapFinishReason(finishReason),
			"stop_sequence": nil,
		},
		"usage": map[string]any{
			"input_tokens":  usage.PromptTokens,
			"output_tokens": usage.CompletionTokens,
		},
	})
	writeEvent(w, flusher, "message_stop", map[string]any{"type": "message_stop"})
}

func writeEvent(w io.Writer, f http.Flusher, event string, payload any) {
	buf, err := json.Marshal(payload)
	if err != nil {
		return
	}
	fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, buf)
	f.Flush()
}
