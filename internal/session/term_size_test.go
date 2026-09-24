package session

import "testing"

func TestSpawnWinsize(t *testing.T) {
	cases := []struct {
		name             string
		cols, rows       uint16
		wantC, wantR     uint16
	}{
		{"unset falls back to default", 0, 0, defaultPTYCols, defaultPTYRows},
		{"remembered size is used", 144, 59, 144, 59},
		{"partial (no rows) falls back", 144, 0, defaultPTYCols, defaultPTYRows},
		{"partial (no cols) falls back", 0, 59, defaultPTYCols, defaultPTYRows},
		{"absurd cols falls back", 5000, 59, defaultPTYCols, defaultPTYRows},
		{"absurd rows falls back", 144, 5000, defaultPTYCols, defaultPTYRows},
		{"upper bound accepted", maxPTYCols, maxPTYRows, maxPTYCols, maxPTYRows},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := spawnWinsize(tc.cols, tc.rows)
			if ws.Cols != tc.wantC || ws.Rows != tc.wantR {
				t.Errorf("spawnWinsize(%d,%d) = %dx%d, want %dx%d",
					tc.cols, tc.rows, ws.Cols, ws.Rows, tc.wantC, tc.wantR)
			}
		})
	}
}
