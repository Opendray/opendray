package backup

import (
	"reflect"
	"testing"
)

func TestDedupeStrings(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", nil, []string{}},
		{"single", []string{"a"}, []string{"a"}},
		{"drops empties", []string{"", "a", ""}, []string{"a"}},
		{"removes later dups, keeps order", []string{"a", "b", "a", "c", "b"}, []string{"a", "b", "c"}},
		{"all dups", []string{"x", "x", "x"}, []string{"x"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := dedupeStrings(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("dedupeStrings(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

func TestRunBackupRequest_TargetIDs(t *testing.T) {
	tests := []struct {
		name string
		req  RunBackupRequest
		want []string
	}{
		{"explicit list wins", RunBackupRequest{TargetID: "x", TargetIDs: []string{"a", "b"}}, []string{"a", "b"}},
		{"falls back to single", RunBackupRequest{TargetID: "x"}, []string{"x"}},
		{"defaults to local", RunBackupRequest{}, []string{"local"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.req.targetIDs()
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("targetIDs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestScheduleTargetIDs(t *testing.T) {
	tests := []struct {
		name string
		sc   Schedule
		want []string
	}{
		{"explicit list", Schedule{TargetID: "a", TargetIDs: []string{"a", "b"}}, []string{"a", "b"}},
		{"fallback from single", Schedule{TargetID: "a"}, []string{"a"}},
		{"empty both", Schedule{}, []string{}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := scheduleTargetIDs(tc.sc)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("scheduleTargetIDs() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNewGroupID(t *testing.T) {
	a, b := NewGroupID(), NewGroupID()
	if a == b {
		t.Error("NewGroupID returned duplicate IDs")
	}
	if len(a) < 4 || a[:3] != "bg_" {
		t.Errorf("group id %q lacks bg_ prefix", a)
	}
}
