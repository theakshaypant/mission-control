//go:build !ui

package main

import "io/fs"

func uiFiles() fs.FS { return nil }
