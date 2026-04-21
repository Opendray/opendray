package bridge

import (
	"encoding/json"
	"testing"
)

func TestProtocolVersion_Frozen(t *testing.T) {
	// This test exists to catch anyone casually bumping the version —
	// the on-wire envelope is part of the v1 plugin contract, and bumping
	// it is a compatibility-breaking change that requires coordinated
	// migration of every installed plugin. Breaking this test is a
	// signal to update docs/plugin-platform/04-bridge-api.md first.
	if ProtocolVersion != 1 {
		t.Errorf("ProtocolVersion = %d, want 1 (v1 bridge contract)", ProtocolVersion)
	}
}

func TestNewOK_RoundTrip(t *testing.T) {
	env, err := NewOK("req-42", map[string]any{"items": []string{"a", "b"}})
	if err != nil {
		t.Fatalf("NewOK: %v", err)
	}
	if env.V != ProtocolVersion {
		t.Errorf("V = %d, want %d", env.V, ProtocolVersion)
	}
	if env.ID != "req-42" {
		t.Errorf("ID = %q, want req-42", env.ID)
	}
	if env.Error != nil {
		t.Errorf("Error = %+v, want nil", env.Error)
	}

	// Re-decode the Result payload as the concrete shape we passed in.
	var back struct {
		Items []string `json:"items"`
	}
	if err := json.Unmarshal(env.Result, &back); err != nil {
		t.Fatalf("Result unmarshal: %v", err)
	}
	if len(back.Items) != 2 || back.Items[0] != "a" {
		t.Errorf("Result round-trip mismatch: %+v", back)
	}
}

func TestNewErr_Shape(t *testing.T) {
	env := NewErr("req-7", "EPERM", "exec denied by capability gate")
	if env.V != ProtocolVersion || env.ID != "req-7" {
		t.Errorf("meta: V=%d ID=%q", env.V, env.ID)
	}
	if env.Error == nil {
		t.Fatal("Error is nil")
	}
	if env.Error.Code != "EPERM" || env.Error.Message == "" {
		t.Errorf("WireError = %+v", env.Error)
	}
	if env.Result != nil {
		t.Errorf("Result must be absent on error envelope, got %s", env.Result)
	}
}

func TestNewStreamChunk_EndPair(t *testing.T) {
	chunk, err := NewStreamChunk("sub-9", map[string]int{"x": 1})
	if err != nil {
		t.Fatalf("NewStreamChunk: %v", err)
	}
	if chunk.Stream != "chunk" {
		t.Errorf("Stream = %q, want chunk", chunk.Stream)
	}
	if chunk.ID != "sub-9" || len(chunk.Data) == 0 {
		t.Errorf("chunk meta: %+v", chunk)
	}

	end := NewStreamEnd("sub-9")
	if end.Stream != "end" || end.ID != "sub-9" || end.Data != nil {
		t.Errorf("end envelope = %+v", end)
	}
}

func TestEnvelope_JSONRoundTrip(t *testing.T) {
	cases := []struct {
		name string
		env  Envelope
	}{
		{
			"request without result",
			Envelope{V: 1, ID: "r1", NS: "storage", Method: "get", Args: json.RawMessage(`{"key":"x"}`)},
		},
		{
			"success response",
			func() Envelope { e, _ := NewOK("r1", map[string]string{"value": "hi"}); return e }(),
		},
		{
			"error response",
			NewErr("r1", "ENOENT", "key not found"),
		},
		{
			"stream chunk",
			func() Envelope { e, _ := NewStreamChunk("s1", []int{1, 2, 3}); return e }(),
		},
		{
			"stream end",
			NewStreamEnd("s1"),
		},
		{
			"handshake with token",
			Envelope{V: 1, ID: "hs", Method: "hello", Token: "abc123"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.env)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}
			var back Envelope
			if err := json.Unmarshal(raw, &back); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			raw2, err := json.Marshal(back)
			if err != nil {
				t.Fatalf("re-marshal: %v", err)
			}
			// Bit-identical after round-trip guards against accidental
			// omitempty regressions that would flood the wire with noise.
			if string(raw) != string(raw2) {
				t.Errorf("round-trip mismatch\n  first:  %s\n  second: %s", raw, raw2)
			}
		})
	}
}

func TestEnvelope_OmitEmpty(t *testing.T) {
	// A bare, zero-ish envelope — we should NOT see keys like "id":"",
	// "error":null on the wire. Only V is required and always present.
	raw, err := json.Marshal(Envelope{V: 1})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	want := `{"v":1}`
	if string(raw) != want {
		t.Errorf("bare envelope: got %s want %s", raw, want)
	}
}

func TestWireError_Codes(t *testing.T) {
	// The six codes listed in M2-PLAN §5 + docs/plugin-platform/04-bridge-api.md.
	// Kept as a constant-like list so linters don't nudge them around and
	// client code can rely on the exact strings.
	codes := []string{"EPERM", "EINVAL", "ENOENT", "ETIMEOUT", "EUNAVAIL", "EINTERNAL"}
	for _, c := range codes {
		c := c
		t.Run(c, func(t *testing.T) {
			e := NewErr("x", c, "test")
			if e.Error == nil || e.Error.Code != c {
				t.Errorf("NewErr dropped or rewrote code %q: got %+v", c, e.Error)
			}
		})
	}
}

func TestNewOK_UnmarshalableResult(t *testing.T) {
	// channels can't be marshaled — constructor must surface the error
	// rather than silently emit a broken envelope.
	_, err := NewOK("r1", make(chan int))
	if err == nil {
		t.Fatal("want error for unmarshalable result, got nil")
	}
}
