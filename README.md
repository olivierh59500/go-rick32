# Rick32 Go

A native Go/Ebitengine conversion of Leonard/Oxygene’s **Rick Dangerous — 32Kb revenge** (2002). It contains the nine rooms of the South American level, the original tiles and sprites, recorded attract-mode inputs, animated text pages and YM music.

Gameplay follows the original 25 Hz clock. Rendering and text animation run at 60 Hz. Demo Construction Kit v1.0.13 supplies cached sprite atlases, batched drawing, bitmap-font layout, per-character deformation, YM playback and synchronized video capture. The game engine is written entirely in Go.

```sh
go run .
```

The game starts in attract mode, with its recorded route and animated credits. Press **Control**, **Space** or **Enter** to play. Arrow keys move Rick; **Control + Up** fires a bullet and **Control + Down** places a bomb. **Up** jumps or climbs; **Down** crouches or climbs down.

| Key | Action |
| --- | --- |
| F | Switch linear/nearest texture filtering |
| W | Show geometry outlines |
| P | Pause/resume |
| R | Reset to attract mode |
| D | Return to attract mode |
| M | Mute/unmute |
| F1 | Toggle fullscreen |
| Escape | Return to attract mode, or quit from attract mode |

`go run . -mute` disables sound; `-fullscreen` starts fullscreen.

## Android

```sh
./scripts/run-android.sh
```

Add `--build-only` to compile without installing. The ARM64 APK is generated at `android/app/build/outputs/apk/debug/app-debug.apk`. The landscape interface provides a directional pad, Fire, Play, Demo, Pause, Filter, Wire and Reset. Two simultaneous touches combine Fire with Up or Down. The app keeps the screen awake and follows the Android pause/resume lifecycle.

## Video

```sh
go run ./cmd/video -output recordings/rick32.mp4
```

The default recording lasts three minutes at 640 × 480 and 60 frames/s. DCK captures the game canvas and its YM soundtrack together. `-duration` and `-poster-at` select the recording length and PNG thumbnail time.

## Verification and credits

`go test ./...` checks the compact native data, all nine rooms, jumping/reset, and the 2,272-step recorded route against positions, animation frames and player flags obtained by running the original x86 routines. The gameplay ending and final score also match that recording. These checks cover the simulation independently of the graphics backend.

[Data formats](reference/FORMAT.md) describe the native tables. [Rendering and audio](reference/RENDERING.md) cover the Ebitengine/DCK composition.

Rick Dangerous was created by Core Design. Rick32 was programmed by Arnaud Carré (Leonard/Oxygene), using the XRick gameplay work of Stéphane (Bigorno) Gay. Music is by Jochen Hippel. The original artwork and in-game credits are retained.
