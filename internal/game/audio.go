package game

import (
	"github.com/olivierh59500/democonstructionkit/sound"
	playback "github.com/olivierh59500/democonstructionkit/sound/ebiten"
	"github.com/olivierh59500/go-rick32/assets"
	"strings"
)

type soundtrack struct {
	music   *playback.Player
	effects map[string]*playback.Player
	name    string
	muted   bool
}

func newSoundtrack() (*soundtrack, error) {
	a := &soundtrack{effects: map[string]*playback.Player{}}
	for _, name := range []string{"bonus", "jump", "walk", "crawl", "box"} {
		raw, err := assets.Files.ReadFile(name + ".ym")
		if err != nil {
			return nil, err
		}
		p, err := playback.Open(nil, name+".ym", raw, sound.Options{Gain: 1})
		if err != nil {
			a.Close()
			return nil, err
		}
		a.effects[name] = p
	}
	return a, nil
}
func (a *soundtrack) track(name string) error {
	if a == nil || a.name == name {
		return nil
	}
	raw, err := assets.Files.ReadFile(name)
	if err != nil {
		return err
	}
	p, err := playback.Open(nil, name, raw, sound.Options{Loop: true, Gain: 1})
	if err != nil {
		return err
	}
	if a.music != nil {
		a.music.Close()
	}
	a.music = p
	a.name = name
	if a.muted {
		p.SetVolume(0)
	} else {
		p.SetVolume(.6)
	}
	p.Play()
	return nil
}
func (a *soundtrack) effect(event string) {
	if a == nil {
		return
	}
	name := strings.TrimSuffix(event, ".wav")
	switch name {
	case "pad":
		name = "jump"
	case "explode", "die", "sbonus1", "sbonus2", "bullet", "bombshht", "stick":
		return
	}
	p := a.effects[name]
	if p == nil {
		return
	}
	if err := p.Seek(0); err == nil {
		if a.muted {
			p.SetVolume(0)
		} else {
			p.SetVolume(.5)
		}
		p.Play()
	}
}
func (a *soundtrack) pause(paused bool) {
	if a == nil {
		return
	}
	if a.music != nil {
		if paused {
			a.music.Pause()
		} else {
			a.music.Play()
		}
	}
}
func (a *soundtrack) mute() {
	if a == nil {
		return
	}
	a.muted = !a.muted
	volume := .6
	if a.muted {
		volume = 0
	}
	if a.music != nil {
		a.music.SetVolume(volume)
	}
	for _, p := range a.effects {
		p.SetVolume(volume)
	}
}
func (a *soundtrack) Close() {
	if a == nil {
		return
	}
	if a.music != nil {
		a.music.Close()
	}
	for _, p := range a.effects {
		p.Close()
	}
}

func (a *soundtrack) restart(name string) error {
	if a == nil {
		return nil
	}
	if a.name != name {
		return a.track(name)
	}
	if err := a.music.Seek(0); err != nil {
		return err
	}
	a.music.Play()
	return nil
}
