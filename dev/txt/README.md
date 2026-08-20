# Physical multiball v02 — ball/ball collision sound

Adds a subtle synthesized stereo tick for real ball-to-ball impacts.

## Behaviour

- Sound is emitted only when a real closing ball/ball collision produces an impulse.
- Loudness and pitch scale with the relative speed along the collision normal.
- Stereo pan uses the collision midpoint.
- A dedicated per-ball cooldown prevents repeated overlap/contact from producing machine-gun chatter.
- This cooldown is independent from the existing wall/paddle/brick impact cooldown.
- Global sound mute still applies because the tone scheduler uses the existing audio master/synth path.

## Config defaults

```go
defaultBallBallSoundEnabled  = true
defaultBallBallSoundMinSpeed = 45.0
defaultBallBallSoundCooldown = 0.060
defaultBallBallSoundGainMin  = 0.025
defaultBallBallSoundGainMax  = 0.110
```

The sound uses synthesized sine + triangle components, so no new audio asset is required.
