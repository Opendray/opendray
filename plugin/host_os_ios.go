//go:build ios

package plugin

// HostFormAllowed is false on iOS builds. The App Store §2.5.2 clause
// forbids executable code outside the app bundle / WebView, so the
// manifest validator (plugin/manifest_validate.go) and the installer
// (plugin/install/install.go) both refuse form:"host" plugins on iOS.
const HostFormAllowed = false
