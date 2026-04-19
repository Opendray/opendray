// Package templates embeds the declarative plugin scaffold templates.
// The embedded FS is used by [cmd/opendray.writeScaffold] at runtime so
// the binary carries its own templates with no external file dependencies.
package templates

import "embed"

//go:embed all:declarative
var DeclarativeFS embed.FS
