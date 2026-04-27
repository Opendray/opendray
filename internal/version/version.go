// Package version exposes build-time identification of the opendray binary.
//
// Values are injected via -ldflags at build time. Defaults make local
// `go run` produce a usable banner without requiring a build script.
package version

// These vars are set with `-ldflags "-X github.com/opendray/opendray-v2/internal/version.Version=..."`.
var (
	Version = "0.0.0-dev"
	Commit  = "unknown"
	Date    = "unknown"
)

type Info struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func Current() Info {
	return Info{Version: Version, Commit: Commit, Date: Date}
}
