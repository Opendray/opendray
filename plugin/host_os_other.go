//go:build !ios

package plugin

// HostFormAllowed is true on every non-iOS build. Android is additionally
// gated at installer level in M3 (Google Play §4.4 review) — see
// plugin/install/install.go for the runtime check.
const HostFormAllowed = true
