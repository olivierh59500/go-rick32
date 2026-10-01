// Package assets embeds the compact artwork and authored game data.
package assets

import "embed"

//go:embed *.bin *.json *.ym *.png
var Files embed.FS
