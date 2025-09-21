package internal

import "embed"

//go:embed templates/index.html.tmpl
var templates embed.FS
