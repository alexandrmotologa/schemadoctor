package web

import (
	"embed"
	"io/fs"
)

//go:embed all:dist
var distFS embed.FS

// GetFS returns the sub-filesystem pointing directly to the compiled web assets.
func GetFS() (fs.FS, error) {
	return fs.Sub(distFS, "dist")
}
