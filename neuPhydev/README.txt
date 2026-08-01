Breakout WASM — gamepad + physics tuner + debris + ball rescue (v57)
Build ID: 20260801-57e8b63f20

BUILD

All compile-time defaults and tuning values live in config.go. Build the package
so both Go files are included:

  GOOS=js GOARCH=wasm go build -o main.wasm .

Equivalent explicit-file build:

  GOOS=js GOARCH=wasm go build -o main.wasm main.go config.go

IN-GAME PHYSICS TUNER

Press E to open the physics tuner. While it is open:

  the game is completely paused
  sliders change the active physics settings
  keyboard and canvas game controls are ignored
  E closes the panel

Closing the panel restores the pause state from before it was opened and copies
the selected export format to the clipboard. Choose in the panel:

  Level-file lines
  config.go defaults
  Both formats

The panel includes the most useful motion, spin, contact, boost, drag, wall and
brick-geometry controls. "Opening values" restores the values present when the
panel was opened. "Built-in defaults" restores defaultPhysicsSettings().

See PHYSICS_TUNER.txt for the complete slider list and clipboard behavior.

MOUSE PADDLE

Desktop mouse movement is direct: the paddle immediately takes the cursor's
horizontal position instead of accelerating toward a target. Render
interpolation is bypassed for the mouse-driven paddle.

Spin is preserved. Each fixed physics step still measures the real paddle
position delta and stores that velocity in the existing spin-history system.
Very large cursor jumps are capped only for collision/spin calculations by the
level-overridable physics value:

  mousePaddleSpinVelocityLimit=6000

The visible paddle position itself is not speed-limited. This value is also in
the E-key physics tuner.

CONFIG.GO

config.go contains compile-time defaults, tunable values and the physics-tuner
slider definitions, including:

  audio mixer and sample tuning
  room preset parameters
  fixed-step engine settings
  dimensions and gameplay defaults
  keyboard/mouse/mobile control tuning
  physics defaults
  physics-tuner slider ranges
  colors and palette
  power-up defaults

Runtime state and behavior remain in main.go.

PER-LEVEL PHYSICS

The existing physics level keys remain supported. Contact, flight, spin,
mouse-spin limiting, wall-roughness, brick-tilt, overspeed and orbit-escape
values are level settings. See LEVEL_PHYSICS_SETTINGS.txt for the complete list.

Example level header:

  gravity=300
  magnusCoefficient=0.003
  spinDrag=0.04
  wallTopTiltDegrees=3.45
  paddleSpinTransfer=2.8
  mousePaddleSpinVelocityLimit=3200
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

LEVEL START TITLE
-----------------
Put this before the --- separator in a level file:

    title="S L A Y"

Quoted and unquoted values are accepted. If title is absent or empty, the
start overlay displays READY.

JOYSTICK / GAMEPAD
------------------
The browser Gamepad API is polled automatically. For the Vivanco
0663-9807 USB Game Device (raw mapping):

  Axis 0   paddle left/right
  Button 0 start/pause/unpause (one action per press)

The center dead zone is configured in config.go as
defaultGamepadDeadZone. Keyboard, mouse, touch and phone tilt remain available.


Joystick: axis 0 moves left/right, B0 starts/pauses, B1 toggles fullscreen.

DYNAMIC BRICK DEBRIS
--------------------
Every destroyed normal, magic, or explicitly broken unbreakable brick creates
a configurable number of irregular physical fragments. Each fragment stores the
source brick's single main fill color plus an independently randomized
starting opacity, then fades during the end of its lifetime.

Fragments respond to gravity, low/reverse gravity, black holes and magnets. They
collide with the ball, paddle, top/side walls, and—after the configured delay—
living bricks without damaging them. They do not collide with one another.
Pieces are removed when their lifetime ends or they fall outside the playfield.

The attached config defaults to a cap of 100 active pieces. Mass destruction
removes the oldest pieces rather than allowing the physics workload to grow
without bound. See BRICK_DEBRIS.txt for all config.go defaults and level-file
overrides.


V55: debris fragments now use only the source brick main fill color; secondary stroke colors are not rendered.


V56: deliberate slivers reduced to 1.5%, normal chunks are aspect-ratio limited,
and 18% of debris pieces are true circles by default. All three values are
config.go defaults and level-file overrides.


V57: PERFORMANCE-AWARE BALL RESCUE
----------------------------------
Normal TILT! and Orbital tilt! recovery attempts are now checked after a
configurable interval. A normal tilt must move the ball by the configured total
distance. An orbital tilt must make that progress along the orbit's minor axis,
so simply travelling around the same loop does not count as an escape.

After four consecutive healthy-simulation failures, the ball is moved to the
first safe opening found in the upper third of the playfield. If the upper third
is blocked by living bricks, the search expands downward while remaining above
the paddle. The ball receives a fresh downward-biased velocity.

The automatic failure counter is gated by the existing fixed-step performance
measurements. Slow/throttled samples, dropped simulation time, or excessive
physics compute load are discarded and cannot trigger a teleport. Press P to see
RESCUE PERF GATE and failure counts.

Press T during active gameplay for a manual teleport using the same safe-position
search. The manual cheat intentionally does not require the performance gate.

V57: DEBRIS FLASH AND IMPACT ENERGY
-----------------------------------
New debris begins with a short configurable opacity flash, default 0.10 seconds,
then settles smoothly into each piece's randomized normal opacity. Brick impact
speed now contributes to debris launch speed through debrisImpactSpeedFactor.
The existing debrisMaxSpeed remains the final safety cap.
