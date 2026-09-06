# Physical replay v03

Built on replay v02.

## Replay transport

- `J` — reverse replay; repeated presses increase speed to 2x, 4x, 8x.
- `K` — enter replay paused / pause replay.
- `L` — forward replay; repeated presses increase speed to 2x, 4x, 8x.
- `Left` / `Right` — previous / next replay frame while replay is active.
- `Home` / `End` — first / last replay frame.
- `Esc` — exit replay and return to the live/result state.

Outside replay, the cursor keys remain the normal paddle controls.

## SVG export

- `V` — export the current frame as a clean vector SVG.
- `B` — export a temporal trail / motion-blur-style vector SVG.

Both `V` and `B` intentionally omit the HUD, status text, replay transport and debug/diagnostic overlays.

`B` keeps the current bricks/background crisp and composites the previous 8 recorded replay samples for moving elements. Older samples fade more strongly; the current frame remains at full opacity. The trail includes balls (including spin-marker orientation), paddle, debris and moving black hole. Zapper remains current-frame-only so it stays crisp.

At the default 30 fps replay capture rate, 8 historical samples cover about 0.27 s before the current frame.

Generated filenames include `-trail.svg` for B exports.

## Detailed vector Zapper export (v03)

The exported Zapper is no longer a single straight SVG line. `V` and `B` now render each active strike as a deterministic 12-segment jagged bolt, matching the live game's electrical shape more closely.

Each bolt is layered entirely as vector geometry:

- broad low-opacity colored aura;
- tighter colored glow;
- bright main bolt;
- thin white-hot core;
- two short branching forks.

Blue/cyan and purple Zapper colors still alternate with the ball color set. The SVG remains fully vector; no raster blur or embedded bitmap is used.

## Replay storage

Replay remains RAM-only. It is not stored in localStorage or IndexedDB.

## Tuning

`replay_config.go` contains:

```go
defaultReplayFPS         = 30.0
defaultReplayMaxSeconds  = 600.0
defaultReplayTrailFrames = 8
```

`defaultReplayShowHUD` still controls the on-screen replay viewer HUD only. It does not affect SVG export; exported SVGs are always clean.

## Validation

- `gofmt` passed.
- Full Go/WASM package build passed using the current main with the existing multiball v02 config and dummy files in the required embedded audio runtime directories.
- `go vet` passed under the same build-check setup.


Update in v04: SVG zapper export now uses several overlaid deterministic zigzag bolt paths, so it more closely matches the layered electricity look of the live game.
