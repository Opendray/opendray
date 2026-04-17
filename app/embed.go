package app

import "embed"

//go:embed all:build/web
var DistFS embed.FS
