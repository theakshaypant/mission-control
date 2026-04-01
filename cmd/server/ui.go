//go:build ui

package main

import (
	"embed"
	"io/fs"
)

//go:embed ui

var rawUI embed.FS

func uiFiles() fs.FS {
	sub, _ := fs.Sub(rawUI, "ui")
	return sub
}
