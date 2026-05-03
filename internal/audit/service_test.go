package audit

import (
	"strings"
	"testing"
	"time"
)

// buildQuery is the audit query builder. We exercise it via table
// tests rather than spinning up a live DB — the SQL string + args
// are the public contract the Postgres planner ultimately consumes.
func TestBuildQuery(t *testing.T) {
	since := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		opts      QueryOpts
		wantWhere []string // substrings that must appear in the WHERE clause
		wantArgs  int      // expected total $N args (incl. trailing LIMIT)
	}{
		{
			name:      "no_filters",
			opts:      QueryOpts{Limit: 50},
			wantWhere: nil,
			wantArgs:  1, // just LIMIT
		},
		{
			name: "subject_kind_and_id",
			opts: QueryOpts{
				SubjectKind: "session",
				SubjectID:   "ses_abc",
				Limit:       20,
			},
			wantWhere: []string{"subject_kind = $1", "subject_id = $2"},
			wantArgs:  3,
		},
		{
			name: "action_exact",
			opts: QueryOpts{
				Action: "session.idle",
				Limit:  10,
			},
			wantWhere: []string{"action = $1"},
			wantArgs:  2,
		},
		{
			name: "action_prefix",
			opts: QueryOpts{
				Action: "integration.*",
				Limit:  10,
			},
			wantWhere: []string{"action LIKE $1"},
			wantArgs:  2,
		},
		{
			name: "time_range",
			opts: QueryOpts{
				Since: since,
				Until: until,
				Limit: 100,
			},
			wantWhere: []string{"ts >= $1", "ts < $2"},
			wantArgs:  3,
		},
		{
			name: "cursor",
			opts: QueryOpts{
				Cursor: 12345,
				Limit:  100,
			},
			wantWhere: []string{"id < $1"},
			wantArgs:  2,
		},
		{
			name: "all_filters",
			opts: QueryOpts{
				SubjectKind: "channel",
				SubjectID:   "ch_x",
				Action:      "channel.*",
				Since:       since,
				Until:       until,
				Cursor:      999,
				Limit:       50,
			},
			wantWhere: []string{
				"subject_kind = $1",
				"subject_id = $2",
				"action LIKE $3",
				"ts >= $4",
				"ts < $5",
				"id < $6",
			},
			wantArgs: 7,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, args := buildQuery(tc.opts)

			if len(args) != tc.wantArgs {
				t.Errorf("got %d args, want %d (q=%s)", len(args), tc.wantArgs, q)
			}
			for _, snippet := range tc.wantWhere {
				if !strings.Contains(q, snippet) {
					t.Errorf("missing %q in: %s", snippet, q)
				}
			}
			if !strings.Contains(q, "ORDER BY id DESC") {
				t.Errorf("missing ORDER BY id DESC in: %s", q)
			}
			if !strings.HasSuffix(strings.TrimSpace(q), " LIMIT $"+itoa(tc.wantArgs)) {
				t.Errorf("LIMIT placeholder is wrong; q=%s", q)
			}
		})
	}
}

func TestBuildQuery_PrefixStripsStarOnly(t *testing.T) {
	_, args := buildQuery(QueryOpts{Action: "session.*", Limit: 10})
	if len(args) != 2 {
		t.Fatalf("want 2 args, got %d", len(args))
	}
	if got, ok := args[0].(string); !ok || got != "session.%" {
		t.Errorf("LIKE pattern = %v, want session.%%", args[0])
	}
}

func TestBuildQuery_LimitDefaultsApplyOutside(t *testing.T) {
	// buildQuery itself doesn't enforce defaults; that's Service.Query's
	// job. We just verify it threads Limit faithfully.
	q, args := buildQuery(QueryOpts{Limit: 7})
	if got, ok := args[len(args)-1].(int); !ok || got != 7 {
		t.Errorf("LIMIT arg = %v, want 7", args[len(args)-1])
	}
	if !strings.Contains(q, "LIMIT") {
		t.Errorf("missing LIMIT in: %s", q)
	}
}

// tiny itoa to avoid pulling strconv just for the test
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	return string(b)
}
