Feature-named magic samples
===========================

Place at most one WAV per feature in this directory, then rebuild main.wasm.
The WAV basename is the exact lowercase feature key:

    lowgravity.wav
    passthrough.wav
    nuke.wav
    reversegravity.wav
    dualballs.wav
    speedboost.wav
    blackhole.wav
    magnet.wav
    influencer.wav
    zapper.wav
    breakunbreakable.wav
    bigpaddle.wav

Small playback-rate, filter, and gain variation is applied on every trigger.
When a sample is absent or cannot be decoded, the established synthesized
power-up cue is used automatically.

Only one instance of the same feature sample plays at once. If blackhole is
triggered again while blackhole.wav is still playing, the old copy fades out
very briefly and the sample restarts. Different feature samples may overlap.
Timed feature audio stops when its effect ends, and all long feature voices stop
when sound is disabled or a new level starts.
