package main

import (
	"embed"
	"io/fs"
)

//go:embed all:internal/web-dist
var frontendDist embed.FS

func getFrontendFS() (fs.FS, error) {
	return fs.Sub(frontendDist, "internal/web-dist")
}
