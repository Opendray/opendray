package session

import (
	"bytes"
	"testing"
)

func TestRingBuffer_PartialFill(t *testing.T) {
	r := NewRing(10)
	n, err := r.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("Write = %d,%v", n, err)
	}
	got := r.Snapshot()
	if !bytes.Equal(got, []byte("hello")) {
		t.Errorf("snapshot = %q", got)
	}
	if r.BytesWritten() != 5 {
		t.Errorf("written = %d", r.BytesWritten())
	}
}

func TestRingBuffer_ExactFill(t *testing.T) {
	r := NewRing(5)
	_, _ = r.Write([]byte("hello"))
	got := r.Snapshot()
	if !bytes.Equal(got, []byte("hello")) {
		t.Errorf("snapshot = %q", got)
	}
}

func TestRingBuffer_Wrap(t *testing.T) {
	r := NewRing(5)
	_, _ = r.Write([]byte("abcde"))
	_, _ = r.Write([]byte("fg"))
	got := r.Snapshot()
	if !bytes.Equal(got, []byte("cdefg")) {
		t.Errorf("wrapped snapshot = %q, want cdefg", got)
	}
	if r.BytesWritten() != 7 {
		t.Errorf("written = %d", r.BytesWritten())
	}
}

func TestRingBuffer_SingleLargeWrite(t *testing.T) {
	r := NewRing(5)
	_, _ = r.Write([]byte("abcdefghij"))
	got := r.Snapshot()
	if !bytes.Equal(got, []byte("fghij")) {
		t.Errorf("snapshot = %q, want fghij", got)
	}
	if r.BytesWritten() != 10 {
		t.Errorf("written = %d", r.BytesWritten())
	}
}

func TestRingBuffer_MultiWriteWrap(t *testing.T) {
	r := NewRing(4)
	_, _ = r.Write([]byte("ab"))
	_, _ = r.Write([]byte("cd"))
	_, _ = r.Write([]byte("ef"))
	got := r.Snapshot()
	if !bytes.Equal(got, []byte("cdef")) {
		t.Errorf("snapshot = %q, want cdef", got)
	}
}

func TestRingBuffer_Empty(t *testing.T) {
	r := NewRing(10)
	if got := r.Snapshot(); len(got) != 0 {
		t.Errorf("empty snapshot = %q", got)
	}
}

func TestRingBuffer_SnapshotSince_NoLoss(t *testing.T) {
	r := NewRing(10)
	_, _ = r.Write([]byte("abcde"))
	rep := r.SnapshotSince(2)
	if rep.Start != 2 || rep.Written != 5 || !bytes.Equal(rep.Bytes, []byte("cde")) {
		t.Errorf("rep=%+v", rep)
	}
}

func TestRingBuffer_SnapshotSince_LaggedClient(t *testing.T) {
	r := NewRing(5)
	_, _ = r.Write([]byte("abcdefgh")) // written=8, ring=[d,e,f,g,h]
	rep := r.SnapshotSince(0)
	if rep.Start != 3 || rep.Written != 8 || !bytes.Equal(rep.Bytes, []byte("defgh")) {
		t.Errorf("rep=%+v (want Start=3 Written=8 Bytes=defgh)", rep)
	}
}

func TestRingBuffer_SnapshotSince_AlreadyCaughtUp(t *testing.T) {
	r := NewRing(10)
	_, _ = r.Write([]byte("abc"))
	rep := r.SnapshotSince(3)
	if len(rep.Bytes) != 0 || rep.Start != 3 || rep.Written != 3 {
		t.Errorf("rep=%+v", rep)
	}
}

func TestRingBuffer_SnapshotSince_FutureCursor(t *testing.T) {
	r := NewRing(10)
	_, _ = r.Write([]byte("abc"))
	rep := r.SnapshotSince(100)
	if len(rep.Bytes) != 0 || rep.Written != 3 {
		t.Errorf("rep=%+v", rep)
	}
}

func TestRingBuffer_SnapshotSince_PartialOverlap(t *testing.T) {
	r := NewRing(5)
	_, _ = r.Write([]byte("abcdefgh")) // ring=[d,e,f,g,h], minAvail=3
	rep := r.SnapshotSince(5)
	if rep.Start != 5 || rep.Written != 8 || !bytes.Equal(rep.Bytes, []byte("fgh")) {
		t.Errorf("rep=%+v", rep)
	}
}

func TestRingBuffer_NewIDUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := newID()
		if seen[id] {
			t.Fatalf("collision at %d: %s", i, id)
		}
		seen[id] = true
	}
}
