//go:build darwin

package keepawake

import (
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
)

// The caffeinate flags are the whole feature, so pin them: -s must not
// drift to -i (that would keep a laptop awake on battery) and -w must
// always carry our pid (without it a killed gateway strands the host
// awake forever).
func TestNewInhibitCmd_Flags(t *testing.T) {
	self := strconv.Itoa(os.Getpid())
	tests := []struct {
		name     string
		mode     Mode
		wantFlag string
	}{
		{"ac asserts only on wall power", ModeAC, "-s"},
		{"always asserts on any power source", ModeAlways, "-i"},
		{"on-demand holds assert on any power source", ModeOnDemand, "-i"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newInhibitCmd(context.Background(), tt.mode)
			if cmd.Path != caffeinatePath {
				t.Fatalf("path = %q, want %q", cmd.Path, caffeinatePath)
			}
			args := strings.Join(cmd.Args[1:], " ")
			if want := tt.wantFlag + " -w " + self; args != want {
				t.Fatalf("args = %q, want %q", args, want)
			}
		})
	}
}
