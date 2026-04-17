package hub

import (
	"reflect"
	"testing"
)

func TestRewriteModelArg(t *testing.T) {
	cases := []struct {
		name   string
		in     []string
		newVal string
		want   []string
	}{
		{
			name:   "replaces existing pair in the middle",
			in:     []string{"--foo", "--model", "qwen3", "--bar"},
			newVal: "lmstudio/qwen3",
			want:   []string{"--foo", "--model", "lmstudio/qwen3", "--bar"},
		},
		{
			name:   "replaces pair at end",
			in:     []string{"--foo", "--model", "qwen3"},
			newVal: "lmstudio/qwen3",
			want:   []string{"--foo", "--model", "lmstudio/qwen3"},
		},
		{
			name:   "appends when absent",
			in:     []string{"--foo", "--bar"},
			newVal: "lmstudio/qwen3",
			want:   []string{"--foo", "--bar", "--model", "lmstudio/qwen3"},
		},
		{
			name:   "bare --model at end (no value) — treated as flag, append new pair",
			in:     []string{"--model"},
			newVal: "lmstudio/qwen3",
			want:   []string{"--model", "--model", "lmstudio/qwen3"},
		},
		{
			name:   "replaces only first occurrence",
			in:     []string{"--model", "a", "--model", "b"},
			newVal: "x",
			want:   []string{"--model", "x", "--model", "b"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rewriteModelArg(tc.in, tc.newVal)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("rewriteModelArg(%v, %q) = %v; want %v", tc.in, tc.newVal, got, tc.want)
			}
		})
	}
}
