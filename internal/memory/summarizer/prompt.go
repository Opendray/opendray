package summarizer

// systemPromptText is the strict-JSON-output instructions every
// provider sends as the system message. Few-shot examples give
// the model concrete patterns for what is and isn't durable.
//
// Keep this text deterministic — every change is a behavioural
// change for every provider. Versions are tracked through git
// history; if we need A/B testing we'll add a version column to
// memory_summarizer_calls.
const systemPromptText = `You are an extraction assistant.

Read the conversation excerpt and extract DURABLE facts the user would
want to remember in future sessions. ONLY extract facts that fit one
of these categories:
- preference: user's stated preference about how things should work
- identifier: a name, ID, URL, or address the user provided
- decision: a project-level decision the user made
- task: an ongoing task the user mentioned wanting to track
- other: durable context that doesn't fit the above

DO NOT extract:
- transient questions or rhetorical statements
- the assistant's own opinions or suggestions
- code snippets unless the user explicitly says "remember this snippet"
- anything you would not bet money on being true tomorrow

OUTPUT FORMAT — STRICT JSON only, no prose, no markdown fences:

{
  "facts": [
    {
      "text": "<one-sentence durable claim, in the user's voice>",
      "category": "preference|identifier|decision|task|other",
      "confidence": <float 0..1>
    }
  ]
}

Empty facts array is valid and expected when nothing durable surfaced.

EXAMPLES:

Conversation:
USER: I prefer pnpm over npm for all my projects.
ASSISTANT: Got it.

Output:
{"facts":[{"text":"User prefers pnpm over npm for package management","category":"preference","confidence":0.95}]}

Conversation:
USER: What time is it?
ASSISTANT: I don't have access to real time.

Output:
{"facts":[]}

Conversation:
USER: My production DB is at 192.168.3.88:5432, please use the dev_user role.
ASSISTANT: Understood, I'll use dev_user for queries.

Output:
{"facts":[
  {"text":"Production DB host is 192.168.3.88:5432","category":"identifier","confidence":0.98},
  {"text":"User wants the dev_user role for DB access","category":"preference","confidence":0.9}
]}

Now extract facts from the following conversation.`

// SystemPrompt returns the raw system prompt text. Exported so
// providers can embed it verbatim in their own message envelope.
func SystemPrompt() string { return systemPromptText }

// FactsToolJSONSchema is the Anthropic tool-use input_schema that
// forces the model to call a record_facts tool with a strict
// shape, sidestepping the "model returns JSON in prose" failure
// mode. ollama uses format:"json" instead which doesn't need
// this schema.
const FactsToolJSONSchema = `{
  "type": "object",
  "properties": {
    "facts": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "text":       {"type": "string"},
          "category":   {"type": "string", "enum": ["preference","identifier","decision","task","other"]},
          "confidence": {"type": "number", "minimum": 0, "maximum": 1}
        },
        "required": ["text","category","confidence"],
        "additionalProperties": false
      }
    }
  },
  "required": ["facts"],
  "additionalProperties": false
}`

// FactsToolName is the tool that providers tell the model to call.
const FactsToolName = "record_facts"

// FactsToolDescription is shown to the model alongside the schema
// so it understands the tool's purpose, not just its shape.
const FactsToolDescription = "Record the durable facts you extracted from the conversation."

// MessagesToTranscriptText flattens a []Message into the plain-text
// transcript form most useful when injected as the user message of
// a Chat Completions call. We keep it deterministic + simple — one
// message per line, role-prefixed, dropping system messages.
func MessagesToTranscriptText(msgs []Message) string {
	var b []byte
	for _, m := range msgs {
		if m.Role == RoleSystem {
			continue
		}
		role := "USER"
		if m.Role == RoleAssistant {
			role = "ASSISTANT"
		}
		b = append(b, role...)
		b = append(b, ": "...)
		b = append(b, m.Text...)
		b = append(b, '\n')
	}
	if len(b) > 0 && b[len(b)-1] == '\n' {
		b = b[:len(b)-1]
	}
	return string(b)
}
