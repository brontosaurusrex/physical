# Physical geometry-aware wind v10

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

## Per-level examples

Enable with defaults:

```ini
wind=true
```

A mostly left-to-right variable wind:

```ini
wind=true
windDirection=0
windDirectionVariation=55
windSpeedMin=70
windSpeedMax=260
windChangeSecondsMin=5
windChangeSecondsMax=14
```

A fully wandering wind:

```ini
wind=true
windDirection=0
windDirectionVariation=180
windSpeedMin=40
windSpeedMax=320
```

Additional level overrides:

```ini
windResponseSeconds=2.2
windBallCoupling=0.25
windMaxAcceleration=220
windWakeStrength=0.75
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
- lower-speed downwind wakes
- deterministic rotor-like wake turbulence
- smooth blending from the old field to the rebuilt field

Destroying bricks marks the wind geometry dirty. Because the field is rebuilt at the configured interval, mass destruction does not trigger one full rebuild per brick.

Wind evolution freezes during READY, pause, and level-complete screens.

## Performance measurements

Physics panel 1 now includes:

- `WIND REBUILD` — last/average field rebuild time in milliseconds
- `WIND PERF OFF` — observed physics compute load / physics Hz with wind off
- `WIND PERF ON` — observed physics compute load / physics Hz with wind on
- `WIND PERF DELTA` — extra compute-load percentage points and observed Hz loss

The ON/OFF comparison is an observed runtime comparison, so debris count and other changing game state can add some noise. `WIND REBUILD` is the direct timing of the expensive geometry-field regeneration itself.

The PATHS visualization is render work rather than physics work, so its cost is reflected mainly in `RENDER FPS`, not `PHYSICS COMPUTE LOAD`.
