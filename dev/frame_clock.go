//go:build js && wasm

package main

import (
	"math"
	"syscall/js"
)

// Use the browser's animation-frame clock even if other callbacks or gamepad
// polling delayed entry into this callback. Manual calls retain a clock fallback.
func animationFrameTime(args []js.Value) float64 {
	if len(args) > 0 && args[0].Type() == js.TypeNumber {
		timestamp := args[0].Float()
		if timestamp >= 0 && !math.IsNaN(timestamp) && !math.IsInf(timestamp, 0) {
			return timestamp
		}
	}
	return js.Global().Get("performance").Call("now").Float()
}
