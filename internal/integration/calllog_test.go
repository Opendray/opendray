package integration

import (
	"strings"
	"testing"
	"time"
)

// buildCallQuery is the SQL builder for /integrations/_calls. Test
// the parameterization rather than spinning up a live DB — the SQL
// string + arg slice is the contract we ship to Postgres.
func TestBuildCallQuery(t *testing.T) {
	since := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 5, 2, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name      string
		opts      CallQueryOpts
		wantWhere []string
		wantArgs  int
	}{
		{
			name:      "no_filters",
			opts:      CallQueryOpts{Limit: 50},
			wantWhere: nil,
			wantArgs:  1,
		},
		{
			name: "by_integration",
			opts: CallQueryOpts{
				IntegrationID: "int_abc",
				Limit:         100,
			},
			wantWhere: []string{"integration_id = $1"},
			wantArgs:  2,
		},
		{
			name: "by_direction_inbound",
			opts: CallQueryOpts{
				Direction: "inbound",
				Limit:     100,
			},
			wantWhere: []string{"direction = $1"},
			wantArgs:  2,
		},
		{
			name: "errors_only",
			opts: CallQueryOpts{
				StatusClass: 5,
				Limit:       100,
			},
			wantWhere: []string{"status_code BETWEEN 500 AND 599"},
			wantArgs:  1, // status_class doesn't bind a $; just LIMIT
		},
		{
			name: "4xx",
			opts: CallQueryOpts{
				StatusClass: 4,
				Limit:       100,
			},
			wantWhere: []string{"status_code BETWEEN 400 AND 499"},
			wantArgs:  1,
		},
		{
			name: "time_window",
			opts: CallQueryOpts{
				Since: since,
				Until: until,
				Limit: 100,
			},
			wantWhere: []string{"ts >= $1", "ts < $2"},
			wantArgs:  3,
		},
		{
			name: "cursor",
			opts: CallQueryOpts{
				Cursor: 999,
				Limit:  100,
			},
			wantWhere: []string{"id < $1"},
			wantArgs:  2,
		},
		{
			name: "all_filters",
			opts: CallQueryOpts{
				IntegrationID: "int_x",
				Direction:     "outbound",
				StatusClass:   4,
				Since:         since,
				Until:         until,
				Cursor:        12345,
				Limit:         50,
			},
			wantWhere: []string{
				"integration_id = $1",
				"direction = $2",
				"status_code BETWEEN 400 AND 499",
				"ts >= $3",
				"ts < $4",
				"id < $5",
			},
			wantArgs: 6,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			q, args := buildCallQuery(tc.opts)

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
		})
	}
}

// TestBuildCallQuery_StatusClassUnknown — values outside 2/3/4/5 are
// silently ignored; this guards the switch from accidentally adding
// a wildcard branch later.
func TestBuildCallQuery_StatusClassUnknown(t *testing.T) {
	q, _ := buildCallQuery(CallQueryOpts{StatusClass: 9, Limit: 10})
	if strings.Contains(q, "BETWEEN") {
		t.Errorf("unexpected status range in: %s", q)
	}
	if strings.Contains(q, "WHERE") {
		t.Errorf("status_class=9 should produce no WHERE clause; got: %s", q)
	}
}

// TestCallEntry_MarshalJSON_TimestampRFC3339Nano — the JSON shape is
// the contract with the frontend. Stamp must be RFC3339 with nanos.
func TestCallEntry_MarshalJSON_TimestampRFC3339Nano(t *testing.T) {
	ts := time.Date(2026, 5, 3, 12, 34, 56, 789012345, time.UTC)
	e := CallEntry{
		ID:            42,
		Time:          ts,
		IntegrationID: "int_x",
		Direction:     "inbound",
		Method:        "POST",
		Path:          "/api/v1/sessions",
		StatusCode:    201,
		DurationMS:    43,
	}
	b, err := e.MarshalJSON()
	if err != nil {
		t.Fatal(err)
	}
	got := string(b)
	if !strings.Contains(got, `"ts":"2026-05-03T12:34:56`) {
		t.Errorf("missing RFC3339 ts in: %s", got)
	}
	if !strings.Contains(got, `"id":42`) {
		t.Errorf("missing id in: %s", got)
	}
}
