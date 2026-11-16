package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

//go:embed assets/app.png
var appIconPNG []byte

var appIcon = fyne.NewStaticResource("codex-command-trace.png", appIconPNG)
