package host

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

func TestFramedWriter_RoundTripsThroughFramedReader(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	w := NewFramedWriter(buf)
	cases := []RPC{
		{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: "initialize"},
		{JSONRPC: "2.0", ID: json.RawMessage(`"req-2"`), Result: json.RawMessage(`null`)},
		{JSONRPC: "2.0", Method: "textDocument/didOpen", Params: json.RawMessage(`{"uri":"file://x"}`)},
	}
	for _, c := range cases {
		if err := w.Write(c); err != nil {
			t.Fatalf("Write: %v", err)
		}
	}
	r := NewFramedReader(buf)
	for i, want := range cases {
		got, err := r.Read()
		if err != nil {
			t.Fatalf("Read[%d]: %v", i, err)
		}
		var gotRPC RPC
		if err := json.Unmarshal(got, &gotRPC); err != nil {
			t.Fatalf("Unmarshal[%d]: %v", i, err)
		}
		if gotRPC.JSONRPC != want.JSONRPC || gotRPC.Method != want.Method {
			t.Errorf("round trip[%d]: got %+v, want %+v", i, gotRPC, want)
		}
	}
	if _, err := r.Read(); err != io.EOF {
		t.Errorf("expected EOF after draining, got %v", err)
	}
}

func TestFramedReader_TrailingHeadersTolerated(t *testing.T) {
	t.Parallel()
	body := `{"jsonrpc":"2.0","method":"ping"}`
	frame := "Content-Length: " + itoa(len(body)) + "\r\n" +
		"Content-Type: application/vscode-jsonrpc; charset=utf-8\r\n\r\n" +
		body
	r := NewFramedReader(strings.NewReader(frame))
	got, err := r.Read()
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(got) != body {
		t.Errorf("body mismatch: %q vs %q", got, body)
	}
}

func TestFramedReader_MissingContentLengthRejected(t *testing.T) {
	t.Parallel()
	r := NewFramedReader(strings.NewReader("Content-Type: x\r\n\r\n{}"))
	_, err := r.Read()
	if !errors.Is(err, ErrMalformedHeader) {
		t.Fatalf("expected ErrMalformedHeader, got %v", err)
	}
}

func TestFramedReader_NegativeContentLengthRejected(t *testing.T) {
	t.Parallel()
	r := NewFramedReader(strings.NewReader("Content-Length: -5\r\n\r\n"))
	_, err := r.Read()
	if !errors.Is(err, ErrMalformedHeader) {
		t.Fatalf("expected ErrMalformedHeader, got %v", err)
	}
}

func TestFramedReader_BodyTooLargeDrainedAndRejected(t *testing.T) {
	t.Parallel()
	oversize := MaxBodyBytes + 1
	frame := "Content-Length: " + itoa(oversize) + "\r\n\r\n" +
		strings.Repeat("x", oversize)
	// Follow the oversize frame with a valid small frame so we can
	// confirm the stream stayed aligned after the drain.
	frame += "Content-Length: 2\r\n\r\n{}"
	r := NewFramedReader(strings.NewReader(frame))
	if _, err := r.Read(); !errors.Is(err, ErrBodyTooLarge) {
		t.Fatalf("expected ErrBodyTooLarge, got %v", err)
	}
	body, err := r.Read()
	if err != nil {
		t.Fatalf("Read after oversize: %v", err)
	}
	if string(body) != "{}" {
		t.Errorf("stream not resynced: got %q", body)
	}
}

func TestFramedReader_HeaderTooLargeRejected(t *testing.T) {
	t.Parallel()
	// A pathological header line of > MaxHeaderBytes with no
	// terminator.
	junk := strings.Repeat("X", MaxHeaderBytes+100) + "\r\n\r\n"
	r := NewFramedReader(strings.NewReader(junk))
	_, err := r.Read()
	if !errors.Is(err, ErrHeaderTooLarge) {
		t.Fatalf("expected ErrHeaderTooLarge, got %v", err)
	}
}

func TestFramedWriter_ConcurrentSafe(t *testing.T) {
	t.Parallel()
	buf := &bytes.Buffer{}
	w := NewFramedWriter(buf)
	var wg sync.WaitGroup
	const N = 50
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := w.Write(RPC{JSONRPC: "2.0", ID: json.RawMessage(itoa(id)), Method: "ping"}); err != nil {
				t.Errorf("concurrent Write: %v", err)
			}
		}(i)
	}
	wg.Wait()
	// Drain through a reader — frames must be intact even with
	// concurrent writers.
	r := NewFramedReader(buf)
	seen := 0
	for {
		_, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("Read after concurrent writes: %v", err)
		}
		seen++
	}
	if seen != N {
		t.Errorf("expected %d frames, got %d", N, seen)
	}
}

// itoa without pulling strconv into every test file.
func itoa(n int) string {
	return stringsIndex(n)
}

func stringsIndex(n int) string {
	return intToStr(n)
}

func intToStr(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
