package main

import (
	"flag"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/go-rick32/internal/game"
	"log"
)

func main() {
	mute := flag.Bool("mute", false, "disable YM playback")
	fullscreen := flag.Bool("fullscreen", false, "start in fullscreen")
	flag.Parse()
	g, err := game.New(game.Config{Mute: *mute})
	if err != nil {
		log.Fatal(err)
	}
	defer g.Close()
	ebiten.SetTPS(game.FPS)
	ebiten.SetWindowSize(960, 720)
	ebiten.SetWindowTitle("Rick32 Go")
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(*fullscreen)
	if err := ebiten.RunGame(g); err != nil {
		log.Fatal(err)
	}
}
