package selfupdate

import (
	"testing"
	"time"
)

func TestNormalizeVersion(t *testing.T) {
	cases := map[string]string{
		"v2.1.1":           "2.1.1",
		"2.1.1":            "2.1.1",
		"  v2.1.1 ":        "2.1.1",
		"2.1.1+ct128.abc1": "2.1.1", // custom build compares against base
		"v2.1.0+build":     "2.1.0",
	}
	for in, want := range cases {
		if got := NormalizeVersion(in); got != want {
			t.Errorf("NormalizeVersion(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestRequestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	if PendingRequest(dir) {
		t.Fatal("PendingRequest = true on empty dir")
	}
	in := Request{Version: "2.1.2", RequestedBy: "admin", RequestedAt: time.Now().UTC().Truncate(time.Second)}
	if err := WriteRequest(dir, in); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	if !PendingRequest(dir) {
		t.Fatal("PendingRequest = false after write")
	}
	got, err := ReadRequest(RequestPath(dir))
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if got.Version != in.Version || got.RequestedBy != in.RequestedBy || !got.RequestedAt.Equal(in.RequestedAt) {
		t.Errorf("round-trip mismatch: got %+v want %+v", got, in)
	}
}

func TestWriteRequestDefaultsTimestamp(t *testing.T) {
	dir := t.TempDir()
	if err := WriteRequest(dir, Request{Version: "2.1.2", RequestedBy: "admin"}); err != nil {
		t.Fatalf("WriteRequest: %v", err)
	}
	got, err := ReadRequest(RequestPath(dir))
	if err != nil {
		t.Fatalf("ReadRequest: %v", err)
	}
	if got.RequestedAt.IsZero() {
		t.Error("RequestedAt was not defaulted")
	}
}
