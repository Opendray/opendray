package host

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"testing"
	"time"
)

// pipeMux wires two Mux instances back-to-back via in-memory pipes
// so tests exercise the full call/response shape without real IO.
func pipeMux(t *testing.T, serverHandler RPCHandler) (*Mux, *Mux, func()) {
	t.Helper()
	// ai→bi: messages from A to B. bi→ai: messages from B to A.
	aReader, bWriter := io.Pipe()
	bReader, aWriter := io.Pipe()

	client := NewMux(NewFramedReader(aReader), NewFramedWriter(aWriter), nil, nil)
	server := NewMux(NewFramedReader(bReader), NewFramedWriter(bWriter), serverHandler, nil)

	ctx, cancel := context.WithCancel(context.Background())
	client.Start(ctx)
	server.Start(ctx)

	cleanup := func() {
		cancel()
		_ = aReader.Close()
		_ = bReader.Close()
		_ = aWriter.Close()
		_ = bWriter.Close()
	}
	return client, server, cleanup
}

func TestMux_CallEchoRoundTrip(t *testing.T) {
	t.Parallel()
	handler := RPCHandlerFunc(func(_ context.Context, method string, params json.RawMessage) (any, error) {
		if method != "echo" {
			return nil, &RPCError{Code: RPCErrMethodNotFound, Message: method}
		}
		return map[string]any{"echo": string(params)}, nil
	})
	client, _, cleanup := pipeMux(t, handler)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := client.Call(ctx, "echo", map[string]string{"hello": "world"})
	if err != nil {
		t.Fatalf("Call: %v", err)
	}
	var resp struct{ Echo string }
	if err := json.Unmarshal(got, &resp); err != nil {
		t.Fatalf("Unmarshal response: %v", err)
	}
	if resp.Echo == "" {
		t.Errorf("expected non-empty echo, got %+v", resp)
	}
}

func TestMux_CallHandlerErrorRoutedAsRPCError(t *testing.T) {
	t.Parallel()
	handler := RPCHandlerFunc(func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, &RPCError{Code: RPCErrEPERM, Message: "denied"}
	})
	client, _, cleanup := pipeMux(t, handler)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := client.Call(ctx, "anything", nil)
	var rerr *RPCError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RPCError, got %v", err)
	}
	if rerr.Code != RPCErrEPERM {
		t.Errorf("code = %d, want %d", rerr.Code, RPCErrEPERM)
	}
}

func TestMux_CallNonRPCErrorBecomesInternal(t *testing.T) {
	t.Parallel()
	handler := RPCHandlerFunc(func(_ context.Context, _ string, _ json.RawMessage) (any, error) {
		return nil, errors.New("boom")
	})
	client, _, cleanup := pipeMux(t, handler)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_, err := client.Call(ctx, "any", nil)
	var rerr *RPCError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RPCError, got %v", err)
	}
	if rerr.Code != RPCErrInternal {
		t.Errorf("code = %d, want %d", rerr.Code, RPCErrInternal)
	}
}

func TestMux_ConcurrentCalls(t *testing.T) {
	t.Parallel()
	// Echo + small delay so multiple calls are in flight at once.
	handler := RPCHandlerFunc(func(_ context.Context, _ string, params json.RawMessage) (any, error) {
		time.Sleep(5 * time.Millisecond)
		return json.RawMessage(params), nil
	})
	client, _, cleanup := pipeMux(t, handler)
	defer cleanup()

	const N = 50
	var wg sync.WaitGroup
	errs := make(chan error, N)
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			_, err := client.Call(ctx, "echo", map[string]int{"i": i})
			if err != nil {
				errs <- err
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	var failed int
	for e := range errs {
		t.Error("concurrent Call:", e)
		failed++
	}
	if failed > 0 {
		t.Fatalf("%d of %d concurrent calls failed", failed, N)
	}
}

func TestMux_NotifyDoesNotBlock(t *testing.T) {
	t.Parallel()
	// Handler records notifications seen.
	var mu sync.Mutex
	var seen []string
	handler := RPCHandlerFunc(func(_ context.Context, m string, _ json.RawMessage) (any, error) {
		mu.Lock()
		seen = append(seen, m)
		mu.Unlock()
		return nil, nil
	})
	// Use pipeMux with a handler that the CLIENT will deliver via the
	// SERVER's Notifications() channel.
	client, server, cleanup := pipeMux(t, handler)
	defer cleanup()

	// Notifications aren't requests — they go through server.Notifications().
	if err := client.Notify("ping", map[string]int{"n": 1}); err != nil {
		t.Fatalf("Notify: %v", err)
	}

	select {
	case n := <-server.Notifications():
		if n.Method != "ping" {
			t.Errorf("notification method = %q, want ping", n.Method)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no notification received")
	}
}

func TestMux_CallCancelledOnContextDone(t *testing.T) {
	t.Parallel()
	handler := RPCHandlerFunc(func(ctx context.Context, _ string, _ json.RawMessage) (any, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	client, _, cleanup := pipeMux(t, handler)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := client.Call(ctx, "slow", nil)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("expected DeadlineExceeded, got %v", err)
	}
}
