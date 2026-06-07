package knowledge

import "testing"

func TestParseExtracted(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []ExtractedEntity
	}{
		{
			name: "clean json",
			raw:  `{"entities":[{"name":"PostgreSQL","type":"service"},{"name":"kv01","type":"host"}]}`,
			want: []ExtractedEntity{{"PostgreSQL", EntityService}, {"kv01", EntityHost}},
		},
		{
			name: "fenced + prose around json",
			raw:  "Sure! Here you go:\n```json\n{\"entities\":[{\"name\":\"pnpm\",\"type\":\"tool\"}]}\n```",
			want: []ExtractedEntity{{"pnpm", EntityTool}},
		},
		{
			name: "drops unknown types and blanks, dedupes",
			raw:  `{"entities":[{"name":"X","type":"widget"},{"name":"","type":"service"},{"name":"Go","type":"tech"},{"name":"go","type":"tech"}]}`,
			want: []ExtractedEntity{{"Go", EntityTech}},
		},
		{name: "garbage", raw: "not json at all", want: nil},
		{name: "empty entities", raw: `{"entities":[]}`, want: []ExtractedEntity{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseExtracted(tt.raw)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d entities %v, want %d %v", len(got), got, len(tt.want), tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("entity %d = %+v, want %+v", i, got[i], tt.want[i])
				}
			}
		})
	}
}
