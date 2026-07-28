# Breakout Physics and Level File Reference

Source basis: `main(2).go`, build `20260727-63fa838581`.

This reference separates:

1. values accepted inside `levels/levelN.txt`;
2. physics constants that can only be changed in the Go source and then rebuilt;
3. the current collision, spin, anti-stuck, fixed-step, and rendering behavior.

---

## `physicsCollisionSlop = 0.05`

`physicsCollisionSlop` is a tiny **positional separation margin**, measured in canvas coordinate units (effectively pixels in the game's 1800 × 900 coordinate system).

After a collision is detected, the ball is not left exactly touching the surface. It is moved an additional `0.05` units away from it.

For a brick collision, the correction is:

```go
ball position += collision normal * (penetration + physicsCollisionSlop)
```

For walls and the paddle, the ball is placed `0.05` units inside the valid play area:

```go
left wall:  ball.x = ball.radius + 0.05
right wall: ball.x = canvasWidth - ball.radius - 0.05
top wall:   ball.y = ball.radius + 0.05
paddle:     ball.y = paddleTop - ball.radius - 0.05
```

### What it does

- Clears any measured overlap.
- Adds a microscopic gap after separation.
- Prevents floating-point rounding from leaving the ball exactly on or slightly inside a surface.
- Reduces repeated collision detection, double impulses, buzzing, sticking, and corner jitter.

### What it does not do

It does **not** directly change:

- velocity;
- bounce angle;
- restitution;
- friction;
- spin;
- collision strength.

### Tuning

- Smaller values may allow repeated contacts or sticking.
- Larger values can create visible gaps or small position jumps.
- `0.05` is extremely small and normally visually invisible.
- It is currently **not a level-file key**. Changing it requires editing the Go source and rebuilding.

---

# 1. Level file format

The game loads contiguous files:

```text
levels/level1.txt
levels/level2.txt
levels/level3.txt
...
```

Loading stops at the first missing file.

Each file contains optional `key=value` settings, followed by a line containing exactly `---`, followed by the brick layout.

```text
# Comments are allowed before the separator.

gravity=300
restitution=0.90
maxSpeed=1000
textColor=#f1faee

---
###M###U###
#   X   X #
###########
```

Rules:

- Blank lines before `---` are ignored.
- Lines beginning with `#` before `---` are comments.
- Settings use `key=value`.
- Whitespace around the key and value is trimmed.
- Duplicate keys are allowed; the final occurrence wins.
- Unknown keys are logged as `Unknown level variable`.
- Invalid numeric or Boolean values are ignored.
- Every level begins by resetting to the built-in defaults, then applying that level's settings.
- Settings do not leak from one level into the next.

## Layout characters

| Character | Meaning |
|---|---|
| space | Empty cell |
| `#` | Normal breakable brick |
| `U` | Unbreakable brick |
| `M` | Magic brick |
| `X` | Random brick |
| any other non-space character | Normal breakable brick |

For `X`, the current source first tests the compile-time `unbreakableChance` of `0.15`. If that fails, it performs a second random test using `magicChance = 0.30`. Therefore the approximate absolute results are:

- 15% unbreakable;
- 25.5% magic;
- 59.5% normal.

Those two chances are not currently level-file keys.

---

# 2. Physics values accepted in level files

These are the only deeper physics settings currently accepted per level.

## Complete example

```text
gravity=300
restitution=0.90
frictionCoeff=0.14
paddleBoost=900
brickBoost=100
maxSpeed=1000
maxSpin=1530
stuckSpeedThreshold=85
stuckDuration=3
tiltUpSpeed=520
tiltSideMin=180
tiltSideMax=340
```

All are reset to their defaults at the start of every level.

## `gravity`

Default:

```text
gravity=300
```

Vertical acceleration in canvas units per second squared.

- Positive values accelerate the ball downward.
- `0` disables ordinary gravity.
- Negative values accelerate upward.
- Low-gravity power-up uses `gravity / 3`.
- Reverse-gravity power-up uses `-gravity`.
- An active black hole temporarily replaces ordinary gravity with `0`.

No range validation is applied by the level parser.

## `restitution`

Default:

```text
restitution=0.90
```

Controls retained normal speed during wall and brick rebounds:

```text
new normal velocity = -restitution × incoming normal velocity
```

- `1.0`: ideally retains normal collision speed.
- `0.90`: retains 90%.
- `0.0`: removes the incoming normal component.

It also participates in paddle speed calculation:

```text
new paddle-bounce speed =
    current speed × restitution + paddleBoost
```

The result is capped by `maxSpeed`.

Accepted range:

```text
0.0 to 1.0
```

## `frictionCoeff`

Default:

```text
frictionCoeff=0.14
```

Base tangential collision friction.

For walls and bricks:

```text
effective friction =
    max(frictionCoeff × surface friction scale,
        minimum collision grip)
```

For the paddle:

```text
effective paddle friction =
    max(frictionCoeff × paddle friction scale,
        minimum paddle grip)
```

Larger values allow a stronger tangential impulse, so:

- spin can change the outgoing direction more;
- translation can transfer more strongly into spin;
- surfaces feel grippier.

It does not by itself create spin; it controls how much slip correction is permitted.

Accepted values:

```text
0 or greater
```

## `paddleBoost`

Default:

```text
paddleBoost=900
```

Additive speed applied on a paddle bounce after restitution:

```text
speed = min(currentSpeed × restitution + paddleBoost, maxSpeed)
```

Larger values make each paddle return accelerate the ball more strongly.

The parser applies no range restriction, so negative values are technically accepted, although they may produce undesirable behavior.

## `brickBoost`

Default:

```text
brickBoost=100
```

After a collision destroys at least one ordinary non-magic breakable brick:

```text
ball.vy -= brickBoost
```

Positive values add an upward kick.

Important details:

- It applies after destroying an ordinary brick.
- It does not trigger merely from hitting an unbreakable brick.
- The magic-brick branch does not set the ordinary-brick boost flag.
- It is applied once for the collision batch, not once per destroyed brick.
- Each velocity component is then limited to `±maxSpeed`.

The parser applies no range restriction. A negative value would kick downward.

## `maxSpeed`

Default:

```text
maxSpeed=1000
```

The main speed ceiling used by several systems:

- caps paddle-bounce speed;
- clamps the complete ball velocity after a paddle collision;
- limits velocity components after `brickBoost`;
- contributes to the Magnus acceleration cap;
- is shown in the P diagnostics as maximum travel per fixed tick.

Important limitation:

`maxSpeed` is not applied as a universal clamp after every ordinary wall or brick collision. It is primarily enforced in the paddle and brick-boost paths.

Accepted values:

```text
greater than 0
```

## `maxSpin`

Default:

```text
maxSpin=1530
```

Maximum absolute angular velocity in radians per second.

Spin is clamped to:

```text
-maxSpin ... +maxSpin
```

after generic contact resolution and after paddle collision processing.

Higher values permit more stored spin, stronger Magnus curvature, and potentially stronger spin-dependent collision steering.

Accepted values:

```text
greater than 0
```

## `stuckSpeedThreshold`

Default:

```text
stuckSpeedThreshold=85
```

The ball is considered slow enough to be stuck when:

```text
hypot(vx, vy) < stuckSpeedThreshold
```

The timer resets whenever speed rises back above the threshold or the ball falls below the play area.

Accepted values:

```text
0 or greater
```

## `stuckDuration`

Default:

```text
stuckDuration=3
```

How many continuous seconds the ball must remain below `stuckSpeedThreshold` before the automatic `TILT!` rescue is applied.

Accepted values:

```text
0 or greater
```

## `tiltUpSpeed`

Default:

```text
tiltUpSpeed=520
```

Vertical speed assigned by the automatic low-speed rescue:

```text
ball.vy = -tiltUpSpeed
```

Positive values launch upward because negative Y is upward in the game.

Accepted values:

```text
0 or greater
```

## `tiltSideMin`

Default:

```text
tiltSideMin=180
```

Minimum absolute horizontal speed selected by `TILT!`.

## `tiltSideMax`

Default:

```text
tiltSideMax=340
```

Maximum absolute horizontal speed selected by `TILT!`.

The rescue chooses a random magnitude between the two values, then randomly chooses left or right.

If `tiltSideMax` is lower than `tiltSideMin`, the code swaps them internally.

Both accept:

```text
0 or greater
```

---

# 3. How the current physics works

## Fixed-step simulation

The simulation runs at:

```go
physicsStepHz = 240.0
physicsStepSeconds = 1.0 / 240.0
```

Rendering still follows `requestAnimationFrame`.

Consequences:

- physics speed is independent of monitor refresh rate;
- a 60 Hz display normally renders after about four physics ticks;
- changing 240 Hz to 480 Hz does not make the game run twice as fast;
- 480 Hz halves the simulated time per tick and approximately doubles physics work.

The loop permits up to 32 catch-up physics steps in one rendered frame. Excess accumulated simulation time is dropped and shown as a warning in the P display.

## Render interpolation

Drawing interpolates between the previous and current completed physics states.

It smooths:

- primary ball position and angle;
- secondary ball position and angle;
- paddle position;
- moving black-hole position.

Interpolation changes only presentation. It does not modify collision state or physics.

Bounded render extrapolation is only a TODO and is currently disabled.

## Adaptive ball substeps

Each 240 Hz ball update can be subdivided further when the ball is moving unusually fast.

```text
maximum desired travel per substep = max(ball radius, 6)
substeps = ceil(speed × dt / maximum travel)
substeps are limited to 1 ... 6
```

This reduces tunneling through thin bricks.

## Gravity

Each ball substep applies:

```text
vy += currentGravity × dt
```

Current gravity is selected in this order:

1. black hole active → `0`;
2. reverse gravity active → `-gravity`;
3. low gravity active → `gravity / 3`;
4. otherwise → `gravity`.

## Magnus effect

Spin curves the flight path perpendicular to velocity:

```text
magnusAx = -vy × omega × physicsMagnusCoefficient
magnusAy =  vx × omega × physicsMagnusCoefficient
```

The Magnus acceleration magnitude is capped at:

```text
max(100, maxSpeed × physicsMagnusAccelerationScale)
```

This is flight curvature between impacts. It is different from spin changing the bounce direction during a collision.

## Air and spin drag

The model uses exponential damping:

```text
velocity multiplier = exp(-physicsAirDrag × dt)
spin multiplier     = exp(-physicsSpinDrag × dt)
```

Larger constants make translation or spin decay faster.

## Wall and brick collision response

The collision resolver calculates the velocity of the ball's actual contact point, including rotation.

It splits relative motion into:

- normal velocity: motion into or away from the surface;
- tangential velocity: sideways surface slip.

Normal response uses `restitution`.

Tangential response attempts a solid-disk no-slip correction:

```text
desired tangential velocity change = -slip / 3
```

The result is limited by the available friction impulse.

That tangential impulse changes both:

- linear velocity;
- angular velocity.

## Spin-to-direction coupling

The spin surface contribution is:

```text
spin surface speed = -omega × radius
```

It enters collision slip as:

```text
combined tangential slip =
    translational slip +
    spin surface speed × physicsCollisionSpinCoupling
```

Therefore larger `physicsCollisionSpinCoupling` makes existing spin more prominent in the outgoing collision direction.

## Paddle collision

The base departure angle depends on where the ball touches the paddle:

```text
paddle centre = 0 degrees from vertical
paddle edges  = up to ±80 degrees
```

The paddle first constructs a base outgoing velocity from:

- hit position;
- current speed;
- restitution;
- paddle boost;
- 15% of incoming horizontal velocity.

It then applies tangential slip correction using:

- existing ball spin;
- paddle horizontal speed;
- paddle grip/friction.

Finally, moving-paddle spin is added:

```text
omega += -paddle.vx × physicsPaddleSpinTransfer / ball.radius
```

A right-moving paddle adds negative spin; a left-moving paddle adds positive spin.

## Near-vertical paddle lock prevention

After a paddle collision, if total speed is sufficient but absolute horizontal speed is below `60`, the game inserts a deterministic horizontal component while preserving total speed.

This prevents a nearly perfect vertical paddle loop.

## Brick collision selection

The brick system uses:

- candidate-grid broad phase;
- swept point-versus-expanded-AABB testing;
- overlap fallback;
- earliest swept contact preference;
- seam rejection for adjacent bricks;
- penetration correction;
- a tiny rotated collision normal matching each brick's visual micro-tilt.

This reduces tunneling and false left/right bounces at internal brick seams.

## Brick micro-tilt

Every brick receives a deterministic rotation based on:

- level index;
- row;
- column.

Current range:

```text
0.2 to 2.0 degrees
```

The sign is positive or negative.

The same angle is used for drawing and collision response. Its purpose is to break exact vertical or horizontal loops without consuming gameplay random numbers.

## Fast unbreakable-orbit detector

The detector watches repeated fast, nearly axis-aligned impacts on unbreakable bricks.

When the configured pattern is detected, it preserves total speed but inserts a controlled minor-axis velocity for a short period. This prevents repeated collisions from alternating the correction direction and falling back into the same orbit.

## Low-speed `TILT!`

If speed remains below the configured threshold for the configured duration:

```text
vx = random left/right value between tiltSideMin and tiltSideMax
vy = -tiltUpSpeed
```

It also adds a tiny random spin change and resets the stuck timer.

---

# 4. Compile-time physics constants

These values cannot currently be placed in a level file. Change them in the Go source and rebuild.

## Flight and collision constants

| Constant | Current value | Exact role |
|---|---:|---|
| `physicsMagnusCoefficient` | `0.0030` | Multiplies `velocity × spin` to produce sideways Magnus acceleration. |
| `physicsMagnusAccelerationScale` | `0.25` | Magnus acceleration cap uses `max(100, maxSpeed × this value)`. |
| `physicsSpinDrag` | `0.04` | Exponential angular-velocity damping rate. |
| `physicsAirDrag` | `0.010` | Exponential linear-velocity damping rate. |
| `physicsWallFrictionScale` | `0.35` | Multiplies `frictionCoeff` for wall collisions. |
| `physicsBrickFrictionScale` | `1.00` | Multiplies `frictionCoeff` for normal and magic brick collision response. |
| `physicsUnbreakableFrictionScale` | `0.80` | Multiplies `frictionCoeff` for unbreakable brick collision response. |
| `physicsPaddleFrictionScale` | `2.60` | Multiplies `frictionCoeff` for paddle slip correction. |
| `physicsPaddleSpinTransfer` | `2.80` | Converts paddle horizontal velocity into new ball spin. |
| `physicsCollisionSpinCoupling` | `2.00` | Multiplies existing spin's contribution to tangential collision slip. |
| `physicsMinimumCollisionGrip` | `0.08` | Minimum effective friction for walls and bricks. |
| `physicsMinimumPaddleGrip` | `0.45` | Minimum effective paddle friction. |
| `physicsCollisionSlop` | `0.05` | Extra positional separation after wall, brick, and paddle contacts. |

## Micro-tilt constants

| Constant | Current value | Role |
|---|---:|---|
| `brickTiltMinDegrees` | `0.2` | Minimum absolute brick tilt. |
| `brickTiltMaxDegrees` | `2.0` | Maximum absolute brick tilt. |
| `drawBrickTilt` | `true` | Draw bricks using the same tilt used by collision normals. |

## Fast-orbit constants

| Constant | Current value | Role |
|---|---:|---|
| `physicsOrbitMinimumSpeed` | `300` | Detector ignores slower balls. |
| `physicsOrbitMinimumHitSpeed` | `80` | Minimum impact speed on an unbreakable brick before recording a hit. |
| `physicsOrbitMinorSpeedRatio` | `0.08` | Axis is considered nearly locked when its minor component is below this fraction of total speed. |
| `physicsOrbitMinorSpeedFloor` | `45` | Absolute floor for the minor-speed test. |
| `physicsOrbitRequiredHits` | `3` | Number of qualifying unbreakable hits required. |
| `physicsOrbitDetectionWindow` | `4.0` | Maximum seconds allowed for the candidate sequence. |
| `physicsOrbitMaximumMinorProgress` | `40` | Maximum movement along the minor axis before the candidate is reset. |
| `physicsOrbitHitCooldown` | `0.08` | Prevents one physical contact from being counted repeatedly. |
| `physicsOrbitEscapeSpeed` | `110` | Desired minor-axis escape speed, limited by total ball speed. |
| `physicsOrbitEscapeDuration` | `0.90` | How long the chosen escape direction is enforced. |
| `physicsOrbitMessageDuration` | `1.5` | Duration of the `Orbital tilt!` message. |

## Fixed-step and control constants

| Constant | Current value | Role |
|---|---:|---|
| `physicsStepHz` | `240` | Fixed simulation frequency. |
| `physicsMaxCatchUpSteps` | `32` | Maximum physics ticks processed during one render frame. |
| `defaultDigitalPaddleMaxSpeed` | `2800` | Keyboard/two-thumb paddle speed cap. |
| `defaultDigitalPaddleAcceleration` | `9000` | Keyboard/two-thumb acceleration. |
| `defaultDigitalPaddleBraking` | `40000` | Keyboard braking rate. |
| `defaultMousePaddleMaxSpeed` | `6000` | Mouse-controlled paddle speed cap. |
| `defaultMousePaddleAcceleration` | `50000` | Mouse-controlled paddle acceleration. |
| `defaultMousePaddleBraking` | `70000` | Mouse-controlled stopping and reversal rate. |
| `defaultMousePaddleSnapDistance` | `0.35` | Distance at which the paddle snaps exactly to the mouse target and stops. |

---

# 5. Other values accepted in level files

## Ball start and paddle geometry

| Key | Default | Meaning |
|---|---:|---|
| `ballX` | `1000` | Starting ball-centre X position. |
| `ballY` | `600` | Starting ball-centre Y position. |
| `ballVx` | `180` | Starting horizontal velocity. Negative is left. |
| `ballVy` | `-350` | Starting vertical velocity. Negative is upward. |
| `ballRadius` | `8` | Ball radius. The parser accepts any float; use a positive value. |
| `paddleWidth` | `220` | Base paddle width. Must be greater than zero. |
| `paddleHeight` | `30` | Base paddle height. Must be greater than zero. |

## Brick geometry

| Key | Default | Meaning |
|---|---:|---|
| `brickWidth` | `60` | Nominal brick width before automatic layout scaling. |
| `defaultBrickWidth` | `60` | Alias for `brickWidth`. |
| `brickHeight` | `20` | Nominal brick height before automatic layout scaling. |
| `defaultBrickHeight` | `20` | Alias for `brickHeight`. |
| `brickPadding` | `20` | Nominal gap between cells before automatic scaling. |
| `defaultBrickPadding` | `20` | Alias for `brickPadding`. |
| `brickOffsetTop` | `-1` | Negative means vertically centre the complete layout. Nonnegative requests a top position, clamped to a 20-unit margin and adjusted to fit. |
| `brickRows` | `10` | Used by the built-in fallback grid, not to override the number of rows in an explicit text layout. |
| `brickCols` | `18` | Used by the built-in fallback grid, not to override the number of columns in an explicit text layout. |

For an explicit layout, row and column counts are derived from the layout text itself.

The grid is automatically scaled down when necessary to fit the canvas. It is never scaled above its nominal size.

## Permanent level mechanics

| Key | Default | Meaning |
|---|---:|---|
| `magnet` | `false` | Starts this level with breakable-brick magnetism continuously active. |
| `zapper` | `false` | Starts this level with the zapper continuously active. |
| `magnetStrength` | `600` | Maximum magnetic acceleration near a breakable brick. |
| `magnetRange` | `300` | Search radius for the nearest breakable brick. Magnetic strength falls linearly to zero at this distance. |
| `influencerMultiplier` | `5` | Influencer destruction radius equals `ballRadius × influencerMultiplier`. Must be greater than zero. |
| `zapperRange` | `320` | Maximum distance to a breakable zapper target. Must be greater than zero. |
| `powerUpDuration` | `10` | Duration in seconds for timed power-ups. The parser does not reject negative values, but positive values are sensible. |

The zapper can also activate automatically when the remaining breakable bricks are at or below:

```text
ceil(initial breakable brick count × 0.03)
```

with a minimum threshold of one brick.

## Magic-brick power-up availability

These Boolean keys decide which effects may be randomly selected when a magic brick is broken.

| Key | Default | Effect made available |
|---|---|---|
| `enableLowGravity` | `true` | Low gravity (`gravity / 3`) |
| `enablePassThrough` | `true` | Ball destroys breakable bricks without bouncing |
| `enableNuke` | `true` | Destroys bricks around the hit brick |
| `enableReverseGravity` | `true` | Uses `-gravity` |
| `enableDualBalls` | `true` | Creates a mirrored second ball; if already active, gives a speed boost |
| `enableBlackHole` | `true` | Moving central attraction field; ordinary gravity becomes zero while active |
| `enableMagnet` | `true` | Timed attraction toward nearest breakable brick |
| `enableInfluencer` | `true` | Area destruction around ball impacts |
| `enableZapper` | `false` | Timed automatic destruction of nearby breakable bricks |
| `enableBreakUnbreakable` | `true` | Destroys one random living unbreakable brick, when one exists |
| `enableBigPaddle` | `true` | Temporarily doubles paddle width |

Use `true` or `false`.

If every available power-up is disabled, breaking a magic brick produces no selected effect.

## Palette and colours

All colour values are passed to the browser canvas as strings. Standard CSS colours such as hexadecimal, `rgb(...)`, or named colours are appropriate.

| Key | Default | Used for |
|---|---|---|
| `backgroundColor` | `#1d3557` | Canvas background |
| `paddleColor` | `#f1faee` | Paddle |
| `brickColor` | `#a8dadc` | Normal bricks |
| `ballColor` | `#e63946` | Primary ball |
| `textColor` | `#f1faee` | HUD, messages, and text-only P diagnostics |
| `spinColor` | `#e63946` | Primary-ball spin marker |
| `unbreakableColor` | `#f4a261` | Unbreakable brick fill |
| `secondBallColor` | `#e63946` | Secondary ball |
| `secondBallSpinColor` | `#e63946` | Secondary-ball spin marker |
| `magicColor` | `#f1faee` | Magic brick fill |
| `magicStrokeColor` | `#ffd700` | Magic brick outline |
| `unbreakableStrokeColor` | `#e76f51` | Unbreakable brick outline |
| `brickStrokeColor` | `#27ae60` | Normal brick outline |

---

# 6. Values that are not level-file keys

The following may look configurable because constants exist in the source, but `applyConfig` does not currently accept them:

- `physicsStepHz`;
- all `physics...Scale`, Magnus, drag, grip, spin-transfer, spin-coupling, and collision-slop constants;
- brick micro-tilt range and enable switch;
- fast-orbit detector settings;
- mouse and keyboard controller speeds;
- `unbreakableChance`;
- `magicChance`;
- `blackHoleStrength`;
- `blackHoleRange`;
- canvas width and height;
- starting lives;
- paddle and brick corner radii;
- zapper hit interval;
- high-spin message threshold and duration;
- black-hole visibility switch.

Putting any of these names in a level file currently produces an unknown-variable console message.

---

# 7. Useful tuning recipes

## Make spin alter collision direction more clearly

Compile-time source changes:

```go
physicsCollisionSpinCoupling = 3.00 // from 2.00
```

If the effect reaches the friction cap, also raise the relevant surface grip:

```go
physicsWallFrictionScale        = 0.55
physicsBrickFrictionScale       = 1.25
physicsUnbreakableFrictionScale = 1.05
physicsMinimumCollisionGrip     = 0.10
```

For stronger existing-spin response on the paddle:

```go
physicsPaddleFrictionScale = 3.20
physicsMinimumPaddleGrip   = 0.55
```

`physicsPaddleSpinTransfer` mainly changes how much **new** spin a moving paddle creates.

## Increase curved flight without changing impact grip

```go
physicsMagnusCoefficient = 0.004
```

Do not use this when the goal is only stronger direction change at the instant of collision.

## Preserve spin longer

Lower:

```go
physicsSpinDrag
```

For example, `0.02` preserves spin longer than `0.04`.

## Reduce speed loss in normal collisions

Raise the per-level value:

```text
restitution=0.96
```

## Make the paddle accelerate the ball faster

Raise:

```text
paddleBoost=1050
maxSpeed=1200
```

## Disable ordinary gravity for one level

```text
gravity=0
```

## Create an upward-gravity level

```text
gravity=-250
```

Be aware that the reverse-gravity power-up negates this value and therefore makes gravity point downward.

---

# 8. Full level example

```text
# Faster, spin-heavy level.

gravity=260
restitution=0.94
frictionCoeff=0.18
paddleBoost=980
brickBoost=130
maxSpeed=1200
maxSpin=1800

stuckSpeedThreshold=90
stuckDuration=2.5
tiltUpSpeed=560
tiltSideMin=210
tiltSideMax=380

ballX=900
ballY=650
ballVx=220
ballVy=-390
ballRadius=8

paddleWidth=220
paddleHeight=30

brickWidth=64
brickHeight=22
brickPadding=16
brickOffsetTop=80

powerUpDuration=10
magnet=false
zapper=false
magnetStrength=600
magnetRange=300
influencerMultiplier=5
zapperRange=320

enableLowGravity=true
enablePassThrough=true
enableNuke=true
enableReverseGravity=true
enableDualBalls=true
enableBlackHole=true
enableMagnet=true
enableInfluencer=true
enableZapper=false
enableBreakUnbreakable=true
enableBigPaddle=true

backgroundColor=#1d3557
paddleColor=#f1faee
brickColor=#a8dadc
ballColor=#e63946
textColor=#f1faee
spinColor=#e63946
unbreakableColor=#f4a261
secondBallColor=#e63946
secondBallSpinColor=#e63946
magicColor=#f1faee
magicStrokeColor=#ffd700
unbreakableStrokeColor=#e76f51
brickStrokeColor=#27ae60

---
    UUUUUUUUUU
  ###MMMMMM###
 ##X##X##X##X##
################
###   ###   ###
```

Only include settings that differ from the defaults; omitted settings automatically use the built-in values.
