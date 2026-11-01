// Package templates holds the files the CLI lays down. They are plain
// markdown, so a pack can also be set up by copying them by hand.
package templates

import "embed"

//go:embed root project
var FS embed.FS
