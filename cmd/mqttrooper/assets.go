// mqttrooper is the main package for the mqttrooper application.
package main

import "embed"

//go:embed templates/*.tmpl
var templates embed.FS
