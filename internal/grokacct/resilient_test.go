package grokacct

import (
	"errors"
	"testing"
)

// pickSpawnHome keeps a grok session working across any available account:
// prefer the pinned account, else the default home, else any other
// logged-in account; error only when nothing is usable.
func TestPickSpawnHome(t *testing.T) {
	dflt := homeCandidate{name: "default", home: "/h/.grok", usable: true}
	acctA := homeCandidate{name: "acct-a", home: "/h/a", usable: true}
	acctBdead := homeCandidate{name: "acct-b", home: "/h/b", usable: false}

	t.Run("pinned account used when logged in", func(t *testing.T) {
		got, err := pickSpawnHome(acctA, dflt, []homeCandidate{acctBdead})
		if err != nil || got.name != "acct-a" {
			t.Fatalf("got %+v err %v, want acct-a", got, err)
		}
	})

	t.Run("dead pinned falls back to default", func(t *testing.T) {
		got, err := pickSpawnHome(acctBdead, dflt, []homeCandidate{acctA})
		if err != nil || got.name != "default" {
			t.Fatalf("got %+v err %v, want default fallback", got, err)
		}
	})

	t.Run("dead pinned + dead default falls back to any usable account", func(t *testing.T) {
		deadDflt := homeCandidate{name: "default", home: "/h/.grok", usable: false}
		got, err := pickSpawnHome(acctBdead, deadDflt, []homeCandidate{acctBdead, acctA})
		if err != nil || got.name != "acct-a" {
			t.Fatalf("got %+v err %v, want acct-a fallback", got, err)
		}
	})

	t.Run("nothing usable errors", func(t *testing.T) {
		deadDflt := homeCandidate{name: "default", usable: false}
		_, err := pickSpawnHome(acctBdead, deadDflt, []homeCandidate{acctBdead})
		if !errors.Is(err, ErrNoUsableGrok) {
			t.Fatalf("want ErrNoUsableGrok, got %v", err)
		}
	})
}
