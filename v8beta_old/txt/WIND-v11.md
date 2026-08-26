# Physical geometry-aware wind v11 — slipstream

Wind is level physics, not a power-up.

## Control

`W` cycles:

1. WIND ON
2. WIND PATHS (same physics, flow visualization visible)
3. WIND OFF

If a level is configured with wind enabled, it starts at step 1. Changing whether wind physics is active during a level makes that record run assisted; merely showing/hiding PATHS does not change physics.

## Global defaults in config.go

The supplied defaults leave wind globally disabled:

```go
defaultWindEnabled = false
```

Set it to `true` to enable wind for every level unless a level overrides it.

Direction convention:

- `0` degrees = right
- `-90` degrees = up
- `+90` degrees = down
- `180` / `-180` = left

The wind smoothly chooses a new random direction and speed target every few seconds. The supplied default uses `defaultWindDirectionVariationDegrees = 180`, so direction may wander through the full circle; lower it for a prevailing wind.

## Ball coupling

Wind accelerates the ball toward the local air velocity:

```text
acceleration = (localWindVelocity - ballVelocity) * windBallCoupling
```

The result is capped by `windMaxAcceleration`.

With the supplied defaults:

```text
windBallCoupling    = 0.25
windMaxAcceleration = 220 px/s²
physics step        = 240 Hz
```

So the absolute maximum wind-only velocity change in one physics tick is about `220 / 240 = 0.917 px/s`.

Physics diagnostics panel 1 includes `WIND BALL EFFECT`, showing the primary ball's current local air speed, actual post-cap wind acceleration, and velocity change per 240 Hz physics tick.

## Slipstream

v11 adds coherent fast shear/slipstream ribbons along the two edges of the slower rotor wake. The center of the lee wake stays slow and turbulent; the boundary bands retain or gain downwind speed and decay with distance.

Defaults:

```go
defaultWindSlipstreamStrength = 0.65
defaultWindSlipstreamLength   = 320.0
defaultWindSlipstreamWidth    = 55.0
```

Set strength to `0` to disable the slipstream while keeping ordinary wake/rotor behavior.

All three can be overridden per level:

```ini
windSlipstreamStrength=0.65
windSlipstreamLength=320
windSlipstreamWidth=55
```

The existing `windMaxLocalSpeedMultiplier` remains the final safety cap on local air velocity, so the slipstream cannot create unbounded wind speed.

## Per-level examples

Enable with defaults:

```ini
wind=true
```

A mostly left-to-right variable wind:

```ini
wind=true
windDirection=0
windDirectionVariation=35
windSpeedMin=80
windSpeedMax=260
windChangeSecondsMin=5
windChangeSecondsMax=14
```

A stronger lee/slipstream level:

```ini
wind=true
windDirection=0
windDirectionVariation=20
windSpeedMin=100
windSpeedMax=300
windWakeStrength=0.80
windSlipstreamStrength=0.90
windSlipstreamLength=420
windSlipstreamWidth=60
windTurbulence=0.45
```

Additional level overrides:

```ini
windResponseSeconds=2.2
windBallCoupling=0.25
windMaxAcceleration=220
windWakeStrength=0.75
windSlipstreamStrength=0.65
windSlipstreamLength=320
windSlipstreamWidth=55
windLiftStrength=0.70
windTurbulence=0.40
windRecalcInterval=0.35
windFieldBlendSeconds=0.55
windGridColumns=60
windGridRows=30
```

## Geometry response

The coarse flow grid is rebuilt periodically from the current living-brick geometry. It includes:

- deflection around brick faces
- accelerated/compressed flow around close geometry and gaps
- upward windward lift
- lower-speed downwind wake cores
- deterministic rotor-like wake turbulence
- coherent fast slipstream/shear bands around wake edges
- smooth blending from the old field to the rebuilt field

Destroying bricks marks the wind geometry dirty. As the living-brick silhouette changes, wake, rotor, and slipstream positions move with it on subsequent field rebuilds.

Wind evolution freezes during READY, pause, and level-complete screens.

## Performance measurements

Physics panel 1 includes:

- `WIND BALL EFFECT` — current local air speed / actual wind acceleration / velocity change per physics tick
- `WIND SLIPSTREAM` — configured strength / length / width
- `WIND REBUILD` — last/average field rebuild time in milliseconds
- `WIND PERF OFF` — observed physics compute load / physics Hz with wind off
- `WIND PERF ON` — observed physics compute load / physics Hz with wind on
- `WIND PERF DELTA` — extra compute-load percentage points and observed Hz loss

The ON/OFF comparison is an observed runtime comparison, so debris count and other changing game state can add some noise. `WIND REBUILD` is the direct timing of geometry-field regeneration itself.

The PATHS visualization is render work rather than physics work, so its cost is reflected mainly in `RENDER FPS`, not `PHYSICS COMPUTE LOAD`.
