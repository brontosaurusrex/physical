Breakout WASM — optimized debris and merged diagnostics (v66)
Build ID: 20260803-66d4f9a2c7

BUILD

All compile-time defaults and tuning values live in config.go. Build the package
so both Go files are included:

  GOOS=js GOARCH=wasm go build -o main.wasm .

Equivalent explicit-file build:

  GOOS=js GOARCH=wasm go build -o main.wasm main.go config.go

IN-GAME PHYSICS + DEBRIS TUNER

Press E to open the right-docked live tuner. When opened during play, the game
keeps running. The panel includes Live simulation, Auto paddle (O), and Debris
enabled checkboxes plus the existing ball-physics controls and a complete debris
section. Closing restores the pause state from before E was opened and copies
physics + debris settings in level, config.go, or combined format.

See PHYSICS_TUNER.txt for the complete control list and live/new-piece behavior.

AUTOMATIC INSPECTION PADDLE

Press O to toggle automatic gameplay. It starts from READY, follows the predicted
paddle-line crossing with bounded acceleration, prioritizes the earliest-arriving
ball in dual-ball play, and stays enabled across levels. It is intended for visual
inspection while tuning effects rather than competitive play.

See AUTO_PADDLE_v60.txt for details and compile-time tuning.

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

The attached config defaults to a cap of 150 active pieces. Mass destruction
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


V58: LARGER DEBRIS + ROTATION SETTLING
--------------------------------------
Debris size/settling support includes debrisSizeScale, debrisAngularDrag, and
debrisAngularStopSpeed=0.10. Size scaling affects drawing, collision radius and
mass. Angular drag gradually slows rotation, and the stop threshold snaps very
slow rotation to zero. Collisions can naturally start a piece rotating again.

The attached custom config.go is the source of truth in this build. Its 2..16
piece range, 9-second lifetime, 0.35 +/- 0.25 opacity, 6 rad/s angular maximum,
1.35 size scale, 0.25-second brick-collision delay, 1.00 debris friction and
150-piece cap are preserved.


V59: LIVE DEBRIS TUNER + AUTOMATIC INSPECTION
---------------------------------------------
The E panel is now live and docked to the right. It exposes debris on/off and the
main debris settings while the game runs. O toggles a predictive automatic paddle
that starts READY screens and continues across levels, allowing hands-off visual
inspection.


V60: VARIABLE AUTO HITS + DEBRIS LIFETIME VARIATION
----------------------------------------------------
The automatic paddle now chooses a stable randomized hit position for each
return instead of centering every contact. The E panel exposes the maximum hit
offset. Debris lifetime variation is now percentage-based and independently
randomized per new piece; 16.7% around the attached 9-second default produces
approximately 7.5..10.5-second lifetimes.


V61: DEBRIS LAYERING, OPAQUE COLOR AND SHAPE VARIETY
----------------------------------------------------
Debris is rendered behind living bricks. Its opacity appearance is baked into
opaque colors blended against the level background, preventing overlap from
becoming increasingly transparent. New live-tunable shape families include
triangles, randomized stars and sharp seven-to-sixteen-corner glass polygons.
All continue to use bounded-circle collision physics.


V62: BELOW-FLOOR BALL GRACE ZONE
--------------------------------
The ball is no longer lost the instant its bottom edge crosses the visible floor.
It continues receiving full physics below the canvas for a configurable number of
pixels, allowing reverse gravity or black-hole attraction to pull it back.

Default:
  defaultBallBelowFloorGracePixels = 120.0

Level override:
  ballBelowFloorGracePixels=120

The threshold is measured from the ball's bottom edge. Set it to 0 to restore the
old immediate floor boundary. The P overlay shows FLOOR GRACE.

V64: SPEED-DEPENDENT DEBRIS DEPTH

Debris is split into two visual layers using debrisFrontLayerSpeed. Shards below
the threshold are drawn behind living bricks; shards at or above it are drawn
after the brick cache and therefore pass visibly over intact bricks. The default
is 600 px/s. Each shard is drawn exactly once; the renderer only performs one
extra velocity-squared comparison per active shard per frame.

V66 DIAGNOSTICS AND FPS
-----------------------

P cycles physics diagnostics, level/config diagnostics, then off. I is an alias
for the same sequence. Both full views are transparent and bottom-right. F
toggles a compact current/lowest FPS display. Lowest FPS uses visible-page
half-second samples after warm-up and ignores hidden-tab/resume gaps.

V65 RENDER OPTIMIZATIONS RETAINED
---------------------------------

Cached Path2D shard outlines, cached opaque color palettes, reduced Canvas state
changes, and adaptive drawing of older slow debris remain enabled. Debris physics
and collisions are never skipped. See DEBRIS_RENDER_OPTIMIZATION_v65.txt.


V68 SELECTABLE DEBRIS SHAPE MODES
--------------------------------

config.go now contains:

  defaultDebrisShapeMode = "mixed"

Levels may override it with debrisShapeMode=mixed, triangles, or circles.
Triangles mode uses only sharp three-edge cached Path2D fragments; circles mode
uses only round chips. The existing mixed probabilities are unchanged and become
active again when the mode returns to mixed. The P performance view displays the
current mode. Debris defaults remain 6 minimum, 18 maximum, and cap 150. See
DEBRIS_SHAPE_MODES_v68.txt.
