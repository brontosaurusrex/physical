Breakout WASM — instant mouse + config.go + per-level physics (v47)
Build ID: 20260730-47f0c2a91d

BUILD

All compile-time defaults and tuning values now live in config.go. Build the
package so both Go files are included:

  GOOS=js GOARCH=wasm go build -o main.wasm .

Equivalent explicit-file build:

  GOOS=js GOARCH=wasm go build -o main.wasm main.go config.go

MOUSE PADDLE

Desktop mouse movement is direct: the paddle immediately takes the cursor's
horizontal position instead of accelerating toward a target. Render
interpolation is bypassed for the mouse-driven paddle.

Spin is preserved. Each fixed physics step still measures the real paddle
position delta and stores that velocity in the existing spin-history system.
Very large cursor jumps are capped only for collision/spin calculations by:

  mousePaddleSpinVelocityLimit = 6000.0

The visible paddle position itself is not speed-limited.

CONFIG.GO

config.go contains the compile-time defaults and tunable values that were
previously concentrated at the top of main.go, including:

  audio mixer and sample tuning
  room preset parameters
  fixed-step engine settings
  dimensions and gameplay defaults
  keyboard/mouse/mobile control tuning
  physics defaults
  colors and palette
  power-up defaults

Runtime state and behavior remain in main.go.

PER-LEVEL PHYSICS

The existing physics level keys remain supported. The newer contact, flight,
spin, wall-roughness, brick-tilt, overspeed, and orbit-escape values are now
level settings too. See LEVEL_PHYSICS_SETTINGS.txt for the complete list.

Example level header:

  gravity=300
  magnusCoefficient=0.003
  spinDrag=0.04
  wallTopTiltDegrees=3.45
  paddleSpinTransfer=2.8
  overspeedHalfLife=0.35
  drawBrickTilt=true
  orbitRequiredHits=3
  ---
  ##################

The fixed-step rate and catch-up limits intentionally remain compile-time-only;
changing those per level would change engine stability and timing rather than
level physics.

AUDIO LAYOUT

Editable WAV masters stay here and are NOT embedded:

  sounds/brickHits/*.wav
  sounds/magicBrickHits/*.wav

Put compressed files that should be embedded here:

  sounds/brickHits/runtime/
  sounds/magicBrickHits/runtime/

Supported runtime extensions:

  .opus .ogg .oga .mp3 .webm .m4a .aac .flac

No manifests are used. Runtime samples are decoded sequentially at startup.
