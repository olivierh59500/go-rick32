# Rendering and audio

The game uses a 640 × 480 canvas. The native playfield occupies x = 64..575 and y = 56..439. The status row sits at y = 40. A room is drawn from cached 32 × 32 blocks at twice their source size; sprites retain their original 32 × 21 source geometry and independent horizontal mirroring. The A1R5G5B5 palette is expanded once during initialization.

DCK's `sprites.Atlas` caches source crops; `render.Batch` submits the backdrop, sprites and HUD in bounded batches. Gameplay retains integer positions. Vertical transitions change the simulated room by 64 pixels immediately and preserve the visible position with a 0.7-second cubic camera offset. Rendering does not advance the simulation.

Text pages use an embedded raster atlas generated from Arial Bold, with independent glyph advances. DCK `scrolling.New` owns layout and glyph drawing. The per-character mapper applies the native 6.1 and 6.5 radian/s oscillators, with 0.1 and 0.2 radian character/line offsets and 16-pixel amplitudes. Pages retain their original strings, tint, coordinates and dwell durations, with cubic entrances/exits. The attract-mode prompts blink separately.

All playback goes through DCK's sound API. The five short sound effects and the Turrican 2 game-over register stream were recovered from the executable. The Sowatt victory track was converted to YM by running its native music replay. Enchanted Lands accompanies attract mode; the Wings of Death level-five YM accompanies play. The latter two recordings come from the Atari YM collection. No recorded audio format is required by the game.

The native attract sequence visits four rooms before Rick's scripted final fall at logical step 2,238. All 2,272 player positions, animation indices and flags match execution of the original x86 routines. The final score is 4,395. One intermediate 50-point enemy award is reported one logical update earlier than in the native replay; it does not change the recorded player path or final score. Desktop font rasterization/filtering and the geometry-outline view may differ slightly from the DirectX renderer.

## Desktop, Android and media checks

On 1 October 2026, the complete Go test suite and `go vet ./...` passed. The ARM64 Android package was installed on a Pixel 10a; its native load segments and APK placement passed 16 KiB alignment. Normal attract playback recorded about 59.8–60.2 displayed frames/s, with no observed crash. Automated touch checks remained incomplete when the USB device disconnected.

The portfolio export lasts 180 seconds: 10,800 H.264 frames at 60 frames/s with AAC stereo audio. The web preview contains VP9 at 50 frames/s and Opus stereo audio; both retain the same three-minute timing and 640 × 480 canvas. The final WebM is about 39 MB. The PNG thumbnail is a game frame near the first page exit. Browser checks covered French/English descriptions, game filtering, responsive layout, image loading, seeking and decoded video frames.

Rick32 0.1.1 replaces the separate direction buttons with a captured, eight-way touch joystick. Its circular thumb follows the finger, clamps at the travel radius and returns to neutral on release. A central dead zone and a small sector margin stabilize movement; another touch can operate Fire independently. Unit checks cover cardinal/diagonal directions, contact ownership, dragging beyond the base, release, layout changes and sector-boundary stability. The ARM64 update was installed on the Pixel 10a; a live rightward drag and release were checked in gameplay, with display cadence remaining near 60 frames/s.
