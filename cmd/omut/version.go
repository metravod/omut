package main

import (
	"runtime/debug"
	"strings"
)

// version is set by goreleaser (-X main.version=0.4.1). A binary from
// `go install ...@v0.4.1` has no ldflags, but the module version is in its
// build info.
var version = "dev"

func init() {
	if version != "dev" {
		return
	}
	if bi, ok := debug.ReadBuildInfo(); ok {
		if v := bi.Main.Version; v != "" && v != "(devel)" {
			version = strings.TrimPrefix(v, "v")
		}
	}
}
