# Live-play optimization check — 2026-10-11

The collision and debris changes reduce allocation and copying while retaining
the existing physics calculations, traversal order, and rendering.

- Collision candidates and contacts use stack buffers, with normal slice growth
  when a query exceeds their capacity.
- Debris updates modify fragments in place and copy only survivors that need to
  move during compaction. Removed references are cleared.
- Linear and angular debris damping are calculated once per physics step, using
  the same expressions and values as before.
- `physical183.wasm` is rebuilt, and its fetch URL includes a new build marker.

## Firefox comparison

Tested with headless Firefox 155.0.1 on this machine at a 1280 × 800 viewport.
Before and after builds used the same Go compiler and bundled WASM runtime.
Temporary instrumentation and browser automation stayed in `/tmp`.

The controlled browser workload used the automatic paddle, one- and five-ball
starts, and four unchanged 240 Hz physics steps per frame. Each run covered
1,980 frames (33 simulated seconds); frame timings excluded the first 180 frames.
Replay recording remained enabled. Each case ran twice.

| Controlled workload | Mean frame work, before → after | Allocated during run, before → after | Go GC cycles, before → after |
| --- | --- | --- | --- |
| One-ball start | 9.16 → 9.12 ms | 8.91 → 5.85 MiB | 3 → 2 |
| Five-ball start | 8.12 → 7.79 ms | 16.58 → 5.14 MiB | 6 → 2 |

A separate physics-only workload simulated 30 seconds without drawing, twice
per case. Mean execution time fell from 2.38 to 2.28 seconds for one ball and
from 0.923 to 0.712 seconds for a five-ball start: approximately 4% and 23% less
simulation work. These are workload measurements, not promised FPS gains.

Before/after hashes of the final ball, paddle, brick, debris, and run states
matched for every controlled frame-workload run and physics-only case.
Additional physics checks also matched. Browser measurements reported no
console errors.

The main benefit is lower allocation and garbage-collection pressure. Drawing
still accounts for most frame work in these Firefox workloads. Headless results
do not guarantee identical timing in a normal Firefox profile or on a phone.

Two 30-second normal-play measurements per version averaged 59.28 FPS before
and 59.40 FPS after. Run-to-run timing varied; this does not establish a
meaningful FPS increase. The optimized binary also received a Firefox smoke
check without temporary Go instrumentation.

## Regression checks

WASM build, `go vet .`, formatting, and package tests passed. The tests cover
buffer reset and growth, dense contacts without truncation or reordering,
allocation-free common contact queries, debris survivor order, interpolation
history, and clearing removed references.

Run the tracked physics-buffer tests from `dev` with Go and Node installed:

```sh
GOOS=js GOARCH=wasm go test -exec "$(go env GOROOT)/lib/wasm/go_js_wasm_exec" .
```

Raw results from this session are in `/tmp/physical-firefox-*-results.json`.

## Brick rendering optimization

Brick-cache rebuilds now pack the live brick geometry into reused buffers and
call a JavaScript Canvas loop once. The loop retains the original draw order,
styles, save/restore, fill/stroke, and tilt operations. The Go renderer remains
a fallback if the helper is unavailable. Physics and replay are unchanged by
this rendering change. The rebuilt WASM and helper have updated cache markers.

Firefox 155.0.1 headless comparison against the preceding physics optimization:

| Workload | Before | After |
| --- | --- | --- |
| 160-brick cache rebuild, normal layout | 3.84 ms | 2.47 ms |
| 160-brick cache rebuild, tilted layout | 5.77 ms | 4.79 ms |
| Full-game drawing, mean per frame | 6.46 ms | 6.25 ms |
| Full-game frame work, excluding warmup | 9.88 ms | 9.30 ms |
| Full-game observed FPS | 59.23 | 58.58 |

Cache timings average four runs of 100 rebuilds each, using the same instrumented
binary with the helper enabled or disabled. Exact brick-canvas pixel hashes
matched for both layouts. Full-game timings average two 1,980-frame controlled
runs per version with the same one-ball start. Final gameplay-state hashes and
whole-canvas pixel hashes matched in every run. The separate one- and five-ball
physics checks also matched. No browser errors were reported.

The cache test shows approximately 36% and 17% less rebuild work. The full-game
results show a smaller average CPU gain, with variable frame pacing and no
established FPS improvement. This targets brick-change redraw cost; headless
tests cannot establish an improvement in desktop display/compositor refresh.
Build, package tests, vet, JavaScript syntax, and diff whitespace checks passed.
Temporary instrumentation was excluded from the production binary.

Raw rendering results are in `/tmp/physical-render-*-results.json` and
`/tmp/physical-render-bricks.json` for this session.

## Browser frame clock and magnetic debris — 2026-10-25

The animation loop now uses the timestamp supplied to `requestAnimationFrame`.
Previously it sampled `performance.now()` after entering the callback and polling
gamepads, so callback-delivery delays could change the measured movement interval.
Manual calls without a valid timestamp retain a `performance.now()` fallback.
The first frame is tracked explicitly, including a valid timestamp of zero.
This is the browser equivalent of using a frame-associated clock in the native
port; Qt's submission callbacks and process-pipe flush are not part of this port.
See the [animation-frame standard](https://html.spec.whatwg.org/multipage/imagebitmap-and-animations.html#animation-frames).

Magnetic debris now queries a direct cell array alongside the existing brick
map. Both normal and fallback level builders populate it in the same order.
Targets still use the same distance calculations and alive/unbreakable filters;
the original map remains the fallback. Physics constants, fixed 240 Hz stepping,
collision rules, randomness and rendering formulas are unchanged.

The WASM regression tests compare 3,000 queries before and after brick destruction
and check rebuilding both normal and fallback layouts. All targets and distances
matched exactly. Existing collision-buffer and debris-compaction tests also pass,
as do the callback-clock tests and `go vet`.

Node/WASM query benchmarks used the same sparse 15-by-44 grid and 128 query
positions for both paths, three runs each:

| Magnetic-debris query | Mean time | Allocations |
| --- | --- | --- |
| Original map lookup | 37.9 microseconds | 0 |
| Direct cell array | 12.9 microseconds | 0 |

This is about 66% less query work, or 2.9 times as many queries per second in this
workload. It does not imply a corresponding whole-game FPS gain.

The Firefox check injects a 6 ms delay before every third animation callback.
For 179 measured intervals per clock, the 95th-percentile discrepancy between
the movement clock interval and browser frame-timestamp interval was 15.2 ms
with the old callback-entry clock and zero with the supplied frame timestamps.
Zero here means agreement with the supplied browser clock, not zero screen
jitter or a measurement of physical presentation. Headless tests do not establish
desktop compositor smoothness.

Both browser clock modes passed active play, assisted paddle, pause/resume and
replay/exit checks without browser errors. A separate check loaded the rebuilt
production WASM, played and paused it, and confirmed that test hooks were absent.
Debug overlays remained off by default. The production fetch URL has a new build
marker, and `build_wasm` now writes `physical184.wasm`, renamed from
`physical183.wasm` to avoid reusing the cached binary. The page and browser tests
both reference the new filename.

Additional frame-clock, magnetic-query and browser checks were run with local
test tooling kept in ignored `test/`. That tooling and its temporary overlays,
screenshots and raw measurements are excluded from this web publication. The
existing tracked physics-buffer tests remain available in the source folder.
Normal game builds contain no added test instrumentation.
