//go:build js && wasm

package main

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"strings"
	"syscall/js"
)

// ---- Default values (constants) ----
const (
	defaultCanvasWidth          = 1800.0
	defaultCanvasHeight         = 900.0
	defaultGravity              = 300.0
	defaultRestitution          = 0.85
	defaultFrictionCoeff        = 0.1 //0.1
	defaultPaddleWidth          = 220.0
	defaultPaddleHeight         = 30.0
	defaultBallRadius           = 8.0
	defaultBrickRows            = 10
	defaultBrickCols            = 18
	defaultBrickWidth           = 60.0
	defaultBrickHeight          = 20.0
	defaultBrickPadding         = 20.0
	defaultBrickOffsetTop       = -1.0
	defaultPaddleBoost          = 700.0
	defaultBrickBoost           = 100.0
	defaultMaxSpeed             = 1000.0
	defaultMaxSpin              = 100.0
	defaultPaddleRadius         = 12.0
	defaultBrickRadius          = 6.0
	defaultUnbreakableChance    = 0.15
	defaultMagicChance          = 0.3
	defaultPowerUpDuration      = 10.0
	defaultBlackHoleStrength    = 800.0
	defaultBlackHoleRange       = 200.0
	defaultMagnetStrength       = 600.0
	defaultMagnetRange          = 300.0
	defaultInfluencerMultiplier = 5.0
	defaultLives                = 7
	defaultStuckSpeedThreshold  = 85.0
	defaultStuckDuration        = 10.0
	defaultTiltUpSpeed          = 520.0
	defaultTiltSideMin          = 180.0
	defaultTiltSideMax          = 340.0
	defaultZapperHitTime        = 0.1
	defaultZapperRange          = 320.0

	defaultPhoneTiltDeadZone  = 1.0
	defaultPhoneTiltMaxAngle  = 10.0
	defaultPhoneTiltSmoothing = 18.0

	defaultStartBallX  = 1000.0
	defaultStartBallY  = 600.0
	defaultStartBallVx = 180.0
	defaultStartBallVy = -350.0

	defaultEnableLowGravity       = true
	defaultEnablePassThrough      = true
	defaultEnableNuke             = true
	defaultEnableReverseGravity   = true
	defaultEnableDualBalls        = true
	defaultEnableBlackHole        = true
	defaultEnableMagnet           = true
	defaultEnableInfluencer       = true
	defaultEnableZapper           = false
	defaultEnableBreakUnbreakable = true

	showBlackHole = false // Set true to draw the moving black hole.
)

// ---- These are variables, reset each level ----
var (
	canvasWidth    = defaultCanvasWidth
	canvasHeight   = defaultCanvasHeight
	baseGravity    = defaultGravity
	restitution    = defaultRestitution
	frictionCoeff  = defaultFrictionCoeff
	paddleWidth    = defaultPaddleWidth
	paddleHeight   = defaultPaddleHeight
	ballRadius     = defaultBallRadius
	brickRows      = defaultBrickRows
	brickCols      = defaultBrickCols
	brickWidth     = defaultBrickWidth
	brickHeight    = defaultBrickHeight
	brickPadding   = defaultBrickPadding
	brickOffsetTop = defaultBrickOffsetTop

	paddleBoost = defaultPaddleBoost
	brickBoost  = defaultBrickBoost
	maxSpeed    = defaultMaxSpeed
	maxSpin     = defaultMaxSpin

	paddleRadius = defaultPaddleRadius
	brickRadius  = defaultBrickRadius

	unbreakableChance = defaultUnbreakableChance
	magicChance       = defaultMagicChance
	powerUpDuration   = defaultPowerUpDuration

	blackHoleStrength    = defaultBlackHoleStrength
	blackHoleRange       = defaultBlackHoleRange
	magnetStrength       = defaultMagnetStrength
	magnetRange          = defaultMagnetRange
	influencerMultiplier = defaultInfluencerMultiplier
	zapperRange          = defaultZapperRange

	startBallX  = defaultStartBallX
	startBallY  = defaultStartBallY
	startBallVx = defaultStartBallVx
	startBallVy = defaultStartBallVy

	stuckSpeedThreshold = defaultStuckSpeedThreshold
	stuckDuration       = defaultStuckDuration
	tiltUpSpeed         = defaultTiltUpSpeed
	tiltSideMin         = defaultTiltSideMin
	tiltSideMax         = defaultTiltSideMax

	enableLowGravity       = defaultEnableLowGravity
	enablePassThrough      = defaultEnablePassThrough
	enableNuke             = defaultEnableNuke
	enableReverseGravity   = defaultEnableReverseGravity
	enableDualBalls        = defaultEnableDualBalls
	enableBlackHole        = defaultEnableBlackHole
	enableMagnet           = defaultEnableMagnet
	enableInfluencer       = defaultEnableInfluencer
	enableZapper           = defaultEnableZapper
	enableBreakUnbreakable = defaultEnableBreakUnbreakable

	enableSounds = true
	paused       bool
	leftPressed  bool
	rightPressed bool

	touchControlActive bool
	touchPointerID     int
	touchLastY         float64

	mobileControlsEnabled  bool
	mobileControlMode      = "vertical"
	mobileSelectorVisible  bool
	mobileLeftHeld         bool
	mobileRightHeld        bool
	mobileLeftPointerID    = -1
	mobileRightPointerID   = -1
	mobileControlCallbacks []js.Func

	phoneTiltAvailable       bool
	phoneTiltCalibrated      bool
	phoneTiltPermissionAsked bool
	phoneTiltListenerSet     bool
	phoneTiltNeutral         float64
	phoneTiltValue           float64
	phoneTiltTargetX         float64

	deviceOrientationCallback js.Func
	nextLevelCallback         js.Func
	previousLevelCallback     js.Func
)

// ---- Color palette ----
var defaultPalette = []string{
	"#1d3557", // background (0)
	"#f1faee", // paddle (1) & magic bricks
	"#a8dadc", // normal bricks (2)
	"#e63946", // ball (3)
	"#f1faee", // text (4)
	"#e63946", // spin marker (5)
	"#f4a261", // unbreakable bricks (6)
	"#e63946", // second ball (7)
	"#e63946", // second ball spin marker (8)
}

var palette = append([]string(nil), defaultPalette...)

var (
	defaultMagicColor             = "#f1faee"
	defaultMagicStrokeColor       = "#ffd700"
	defaultUnbreakableStrokeColor = "#e76f51"
	defaultBrickStrokeColor       = "#27ae60"

	magicColor             = defaultMagicColor
	magicStrokeColor       = defaultMagicStrokeColor
	unbreakableStrokeColor = defaultUnbreakableStrokeColor
	brickStrokeColor       = defaultBrickStrokeColor
)

// ---- Powerup types ----
const (
	POWER_NONE = iota
	POWER_LOW_GRAVITY
	POWER_PASS
	POWER_NUKE
	POWER_REVERSE_GRAVITY
	POWER_DUAL_BALLS
	POWER_BLACKHOLE
	POWER_MAGNET
	POWER_INFLUENCER
	POWER_ZAPPER
	POWER_BREAK_UNBREAKABLE
)

// ---- Ball struct ----
type Ball struct {
	x, y, r      float64
	vx, vy       float64
	omega, angle float64
	stuckTimer   float64
}

type statusMessage struct {
	text  string
	timer float64
}

// ---- Global state ----
var (
	doc    = js.Global().Get("document")
	canvas js.Value
	ctx    js.Value

	brickCanvas js.Value
	brickCtx    js.Value
	bricksDirty bool

	ball             Ball
	secondBall       Ball
	secondBallActive bool

	paddle = struct {
		x, y, w, h float64
		vx         float64
	}{
		x:  0,
		y:  0,
		w:  defaultPaddleWidth,
		h:  defaultPaddleHeight,
		vx: 0,
	}

	bricks                   []brick
	brickGrid                map[int][]int
	remainingBreakableBricks int
	initialBreakableBricks   int
	lastBrickSoundPlayed     bool
	score                    int
	lives                    int
	gameOver                 bool
	win                      bool
	waitingForStart          bool
	levelCompleteTimer       float64
	levelAdvancePending      bool

	lastTime float64

	loopFunc      js.Func
	keyDown       js.Func
	keyUp         js.Func
	pointerMove   js.Func
	pointerDown   js.Func
	pointerUp     js.Func
	pointerCancel js.Func
	mouseLeave    js.Func

	// Powerup states
	activePowerUp  int
	powerUpTimer   float64
	currentGravity float64

	blackHoleActive    bool
	blackHoleTimer     float64
	blackHoleX         float64
	blackHoleY         float64
	blackHoleDirection float64
	blackHoleSpeed     float64

	magnetCheat       bool
	zapperCheat       bool
	levelMagnetActive bool
	levelZapperActive bool

	influencerActive bool
	influencerTimer  float64

	zapperTargetIndex       int
	zapperHitTimer          float64
	secondZapperTargetIndex int
	secondZapperHitTimer    float64

	statusMessages []statusMessage

	gridOffsetLeft float64
	gridOffsetTop  float64
	gridCellWidth  float64
	gridCellHeight float64
	gridRows       int
	gridCols       int

	paddlePreviousX float64

	hudLivesValue int
	hudLevelValue int
	hudScoreValue int
	hudLivesText  string
	hudLevelText  string
	hudScoreText  string

	// ---- Level system ----
	currentLevelIndex int
	levels            []levelData

	// ---- Audio ----
	audioCtx         js.Value
	audioMaster      js.Value
	audioInitialized bool
)

type brick struct {
	x, y, w, h  float64
	row, col    int
	alive       bool
	unbreakable bool
	magic       bool
}

// ---- Level data ----
type levelData struct {
	layout []string
	config map[string]string
}

// ---- Helpers ----
func log(msg string) {
	js.Global().Get("console").Call("log", msg)
}

func sign(x float64) float64 {
	if x > 0 {
		return 1
	}
	if x < 0 {
		return -1
	}
	return 0
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func showStatus(text string, duration float64) {
	statusMessages = append(statusMessages, statusMessage{text: text, timer: duration})
	if len(statusMessages) > 3 {
		statusMessages = statusMessages[len(statusMessages)-3:]
	}
}

func gridKey(row, col int) int {
	return row*gridCols + col
}

func registerBrick(index int) {
	br := &bricks[index]
	brickGrid[gridKey(br.row, br.col)] = append(brickGrid[gridKey(br.row, br.col)], index)
	if !br.unbreakable {
		remainingBreakableBricks++
	}
}

func destroyBrick(br *brick) bool {
	if br == nil || !br.alive || br.unbreakable {
		return false
	}
	br.alive = false
	score++
	remainingBreakableBricks--
	if remainingBreakableBricks < 0 {
		remainingBreakableBricks = 0
	}
	bricksDirty = true
	maybePlayLastBrickSound()
	return true
}

func destroyAnyBrick(br *brick) bool {
	if br == nil || !br.alive {
		return false
	}
	br.alive = false
	score++
	if !br.unbreakable {
		remainingBreakableBricks--
		if remainingBreakableBricks < 0 {
			remainingBreakableBricks = 0
		}
	}
	bricksDirty = true
	maybePlayLastBrickSound()
	return true
}

func maybePlayLastBrickSound() {
	if remainingBreakableBricks == 0 && !lastBrickSoundPlayed {
		lastBrickSoundPlayed = true
		playLastBrick()
	}
}

func updateHUDCache() {
	level := currentLevelIndex + 1
	if hudLivesValue != lives {
		hudLivesValue = lives
		hudLivesText = "Lives: " + strconv.Itoa(lives)
	}
	if hudLevelValue != level {
		hudLevelValue = level
		hudLevelText = "Level: " + strconv.Itoa(level)
	}
	if hudScoreValue != score {
		hudScoreValue = score
		hudScoreText = "Score: " + strconv.Itoa(score)
	}
}

func toggleSound() {
	enableSounds = !enableSounds

	if enableSounds {
		if !audioInitialized {
			initAudio()
		} else {
			ensureAudioRunning()
		}
		showStatus("Sound on", 2.0)
	} else {
		if audioInitialized && !audioMaster.IsUndefined() && !audioMaster.IsNull() {
			audioMaster.Get("gain").Call("cancelScheduledValues", audioCtx.Get("currentTime").Float())
			audioMaster.Get("gain").Set("value", 0)
		}
		showStatus("Sound off", 2.0)
	}
}

// ---- Audio (non-blocking, scheduled via Web Audio) ----
func initAudio() {
	if audioInitialized || !enableSounds {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			audioInitialized = false
			audioCtx = js.Undefined()
			audioMaster = js.Undefined()
			js.Global().Get("console").Call("error", "Audio init panic:", fmt.Sprint(r))
		}
	}()

	window := js.Global().Get("window")
	ctor := window.Get("AudioContext")
	if ctor.IsUndefined() || ctor.IsNull() {
		ctor = window.Get("webkitAudioContext")
	}
	if ctor.IsUndefined() || ctor.IsNull() {
		log("Web Audio API is not supported by this browser")
		return
	}

	audioCtx = ctor.New()
	if audioCtx.IsUndefined() || audioCtx.IsNull() {
		log("Failed to create AudioContext")
		return
	}

	audioMaster = audioCtx.Call("createGain")
	audioMaster.Get("gain").Set("value", 0.33)
	audioMaster.Call("connect", audioCtx.Get("destination"))

	audioInitialized = true
	audioMaster.Get("gain").Set("value", 0.33)
	ensureAudioRunning()
	log("Audio initialized")
}

func ensureAudioRunning() {
	if !audioInitialized || audioCtx.IsUndefined() || audioCtx.IsNull() {
		return
	}
	if enableSounds && !audioMaster.IsUndefined() && !audioMaster.IsNull() {
		audioMaster.Get("gain").Set("value", 0.33)
	}
	state := audioCtx.Get("state")
	if state.Type() == js.TypeString && state.String() == "suspended" {
		audioCtx.Call("resume")
	}
}

func scheduleTone(
	oscType string,
	startFreq float64,
	endFreq float64,
	duration float64,
	volume float64,
	delay float64,
) {
	if !audioInitialized || !enableSounds ||
		audioCtx.IsNull() || audioCtx.IsUndefined() ||
		audioMaster.IsNull() || audioMaster.IsUndefined() {
		return
	}

	defer func() {
		_ = recover()
	}()

	ensureAudioRunning()

	start := audioCtx.Get("currentTime").Float() + delay
	stop := start + duration

	osc := audioCtx.Call("createOscillator")
	gain := audioCtx.Call("createGain")

	osc.Set("type", oscType)
	osc.Get("frequency").Call("setValueAtTime", math.Max(startFreq, 1), start)
	osc.Get("frequency").Call("exponentialRampToValueAtTime", math.Max(endFreq, 1), stop)

	gain.Get("gain").Call("setValueAtTime", 0.0001, start)
	gain.Get("gain").Call("exponentialRampToValueAtTime", math.Max(volume, 0.0001), start+0.008)
	gain.Get("gain").Call("exponentialRampToValueAtTime", 0.0001, stop)

	osc.Call("connect", gain)
	gain.Call("connect", audioMaster)
	osc.Call("start", start)
	osc.Call("stop", stop+0.02)
}

func scheduleChord(freqs []float64, oscType string, duration, volume, delay float64) {
	if len(freqs) == 0 {
		return
	}
	perVoice := volume / float64(len(freqs))
	for i, freq := range freqs {
		detune := 1.0 + float64(i)*0.002
		scheduleTone(oscType, freq*detune, freq*0.98, duration, perVoice, delay)
	}
}

func audioRand(min, max float64) float64 {
	return min + rand.Float64()*(max-min)
}

func varyFreq(freq float64, amount float64) float64 {
	return freq * audioRand(1-amount, 1+amount)
}

func playWallHit() {
	start := varyFreq(230, 0.10)
	end := varyFreq(175, 0.10)
	scheduleTone("triangle", start, end, audioRand(0.038, 0.055), audioRand(0.11, 0.16), 0)
}

func playPaddleHit() {
	start := varyFreq(180, 0.08)
	end := varyFreq(320, 0.08)
	scheduleTone("triangle", start, end, audioRand(0.065, 0.085), audioRand(0.16, 0.22), 0)
	scheduleTone("sine", varyFreq(90, 0.06), varyFreq(70, 0.06), audioRand(0.08, 0.11), audioRand(0.05, 0.08), 0)
}

func playBrickBreak() {
	scheduleTone("square", varyFreq(520, 0.12), varyFreq(360, 0.12), audioRand(0.045, 0.065), audioRand(0.07, 0.11), 0)
	scheduleTone("triangle", varyFreq(760, 0.10), varyFreq(520, 0.10), audioRand(0.032, 0.05), audioRand(0.035, 0.06), audioRand(0.005, 0.012))
}

func playUnbreakable() {
	// Low, rounded impact with small natural variation.
	base := varyFreq(82, 0.10)
	scheduleTone("sine", base, varyFreq(52, 0.08), audioRand(0.09, 0.14), audioRand(0.12, 0.17), 0)
	scheduleTone("triangle", varyFreq(46, 0.08), varyFreq(34, 0.08), audioRand(0.11, 0.16), audioRand(0.05, 0.08), 0.004)
}

func playMagic() {
	scheduleTone("sine", varyFreq(660, 0.06), varyFreq(990, 0.06), audioRand(0.09, 0.13), audioRand(0.08, 0.11), 0)
	scheduleTone("triangle", varyFreq(990, 0.05), varyFreq(1480, 0.05), audioRand(0.11, 0.15), audioRand(0.055, 0.08), audioRand(0.05, 0.075))
}

func playPowerup() {
	root := varyFreq(330, 0.05)
	scheduleTone("triangle", root, root*4/3, audioRand(0.09, 0.12), audioRand(0.08, 0.11), 0)
	scheduleTone("triangle", root*4/3, root*2, audioRand(0.10, 0.14), audioRand(0.07, 0.10), audioRand(0.07, 0.10))
	scheduleChord([]float64{root * 2, root * 2.5, root * 3}, "sine", audioRand(0.18, 0.24), audioRand(0.11, 0.15), audioRand(0.14, 0.19))
}

func playDie() {
	// Audible life-loss cue: short falling mid tone plus low body.
	scheduleTone(
		"triangle",
		varyFreq(420, 0.06),
		varyFreq(150, 0.06),
		audioRand(0.20, 0.26),
		audioRand(0.14, 0.18),
		0,
	)
	scheduleTone(
		"sine",
		varyFreq(150, 0.05),
		varyFreq(58, 0.05),
		audioRand(0.24, 0.32),
		audioRand(0.08, 0.11),
		0.025,
	)
}

func playGameOver() {
	scheduleTone("sawtooth", varyFreq(260, 0.03), varyFreq(120, 0.03), audioRand(0.24, 0.30), audioRand(0.08, 0.11), 0)
	scheduleTone("triangle", varyFreq(170, 0.03), varyFreq(62, 0.03), audioRand(0.36, 0.44), audioRand(0.08, 0.11), 0.20)
	scheduleChord([]float64{55, 65.4, 82.4}, "sine", audioRand(0.42, 0.52), audioRand(0.10, 0.14), 0.42)
}

func playLastBrick() {
	root := varyFreq(520, 0.04)
	scheduleTone("triangle", root, root*1.5, audioRand(0.07, 0.10), audioRand(0.08, 0.11), 0)
	scheduleTone("sine", root*1.5, root*2, audioRand(0.11, 0.15), audioRand(0.07, 0.10), 0.055)
	scheduleChord([]float64{root * 2, root * 2.5}, "sine", audioRand(0.16, 0.22), audioRand(0.10, 0.14), 0.14)
}

func playYouWin() {
	root := varyFreq(392, 0.02)
	scheduleTone("triangle", root, root*4/3, 0.14, 0.09, 0)
	scheduleTone("triangle", root*4/3, root*5/3, 0.15, 0.09, 0.13)
	scheduleTone("triangle", root*5/3, root*2, 0.17, 0.09, 0.27)
	scheduleChord([]float64{root, root * 5 / 4, root * 3 / 2, root * 2}, "sine", 0.48, 0.18, 0.43)
}

func playZapper() {
	scheduleTone("sawtooth", varyFreq(920, 0.10), varyFreq(280, 0.10), 0.08, 0.08, 0)
	scheduleTone("square", varyFreq(1450, 0.08), varyFreq(520, 0.08), 0.045, 0.045, 0.008)
}

func playLevelComplete() {
	root := varyFreq(440, 0.04)
	scheduleTone("triangle", root, root*1.5, audioRand(0.085, 0.115), audioRand(0.07, 0.09), 0.08)
	scheduleChord([]float64{root * 1.5, root * 1.875}, "sine", audioRand(0.14, 0.19), audioRand(0.09, 0.12), audioRand(0.16, 0.20))
}

// ---- Reset globals to defaults ----
func resetGlobals() {
	canvasWidth = defaultCanvasWidth
	canvasHeight = defaultCanvasHeight
	baseGravity = defaultGravity
	restitution = defaultRestitution
	frictionCoeff = defaultFrictionCoeff
	paddleWidth = defaultPaddleWidth
	paddleHeight = defaultPaddleHeight
	ballRadius = defaultBallRadius
	brickRows = defaultBrickRows
	brickCols = defaultBrickCols
	brickWidth = defaultBrickWidth
	brickHeight = defaultBrickHeight
	brickPadding = defaultBrickPadding
	brickOffsetTop = defaultBrickOffsetTop
	paddleBoost = defaultPaddleBoost
	brickBoost = defaultBrickBoost
	maxSpeed = defaultMaxSpeed
	maxSpin = defaultMaxSpin
	paddleRadius = defaultPaddleRadius
	brickRadius = defaultBrickRadius
	unbreakableChance = defaultUnbreakableChance
	magicChance = defaultMagicChance
	powerUpDuration = defaultPowerUpDuration
	blackHoleStrength = defaultBlackHoleStrength
	blackHoleRange = defaultBlackHoleRange
	magnetStrength = defaultMagnetStrength
	magnetRange = defaultMagnetRange
	influencerMultiplier = defaultInfluencerMultiplier
	zapperRange = defaultZapperRange
	startBallX = defaultStartBallX
	startBallY = defaultStartBallY
	startBallVx = defaultStartBallVx
	startBallVy = defaultStartBallVy
	stuckSpeedThreshold = defaultStuckSpeedThreshold
	stuckDuration = defaultStuckDuration
	tiltUpSpeed = defaultTiltUpSpeed
	tiltSideMin = defaultTiltSideMin
	tiltSideMax = defaultTiltSideMax
	enableLowGravity = defaultEnableLowGravity
	enablePassThrough = defaultEnablePassThrough
	enableNuke = defaultEnableNuke
	enableReverseGravity = defaultEnableReverseGravity
	enableDualBalls = defaultEnableDualBalls
	enableBlackHole = defaultEnableBlackHole
	enableMagnet = defaultEnableMagnet
	enableInfluencer = defaultEnableInfluencer
	enableZapper = defaultEnableZapper
	enableBreakUnbreakable = defaultEnableBreakUnbreakable
	palette = append([]string(nil), defaultPalette...)
	magicColor = defaultMagicColor
	magicStrokeColor = defaultMagicStrokeColor
	unbreakableStrokeColor = defaultUnbreakableStrokeColor
	brickStrokeColor = defaultBrickStrokeColor
}

// ---- Apply config overrides from map ----
func applyConfig(config map[string]string) {
	for key, val := range config {
		switch key {
		case "backgroundColor":
			palette[0] = val
		case "paddleColor":
			palette[1] = val
		case "brickColor":
			palette[2] = val
		case "ballColor":
			palette[3] = val
		case "textColor":
			palette[4] = val
		case "spinColor":
			palette[5] = val
		case "unbreakableColor":
			palette[6] = val
		case "secondBallColor":
			palette[7] = val
		case "secondBallSpinColor":
			palette[8] = val
		case "magicColor":
			magicColor = val
		case "magicStrokeColor":
			magicStrokeColor = val
		case "unbreakableStrokeColor":
			unbreakableStrokeColor = val
		case "brickStrokeColor":
			brickStrokeColor = val
		case "gravity":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				baseGravity = f
			}
		case "restitution":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				restitution = f
			}
		case "frictionCoeff":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				frictionCoeff = f
			}
		case "maxSpin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				maxSpin = f
			}
		case "paddleBoost":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				paddleBoost = f
			}
		case "brickBoost":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickBoost = f
			}
		case "maxSpeed":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				maxSpeed = f
			}
		case "stuckSpeedThreshold":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				stuckSpeedThreshold = f
			}
		case "stuckDuration":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				stuckDuration = f
			}
		case "tiltUpSpeed":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				tiltUpSpeed = f
			}
		case "tiltSideMin":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				tiltSideMin = f
			}
		case "tiltSideMax":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 {
				tiltSideMax = f
			}
		case "powerUpDuration":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				powerUpDuration = f
			}
		case "magnet":
			if b, err := strconv.ParseBool(val); err == nil {
				levelMagnetActive = b
			}
		case "zapper":
			if b, err := strconv.ParseBool(val); err == nil {
				levelZapperActive = b
			}
		case "magnetStrength":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				magnetStrength = f
			}
		case "magnetRange":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				magnetRange = f
			}
		case "influencerMultiplier":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				influencerMultiplier = f
			}
		case "zapperRange":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				zapperRange = f
			}
		case "brickOffsetTop":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickOffsetTop = f
			}
		case "brickWidth", "defaultBrickWidth":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickWidth = f
			}
		case "brickHeight", "defaultBrickHeight":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickHeight = f
			}
		case "brickPadding", "defaultBrickPadding":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				brickPadding = f
			}
		case "brickRows":
			if i, err := strconv.Atoi(val); err == nil {
				brickRows = i
			}
		case "brickCols":
			if i, err := strconv.Atoi(val); err == nil {
				brickCols = i
			}
		case "ballX":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallX = f
			}
		case "ballY":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallY = f
			}
		case "ballVx":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallVx = f
			}
		case "ballVy":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				startBallVy = f
			}
		case "ballRadius":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				ballRadius = f
			}
		case "enableLowGravity":
			if b, err := strconv.ParseBool(val); err == nil {
				enableLowGravity = b
			}
		case "enablePassThrough":
			if b, err := strconv.ParseBool(val); err == nil {
				enablePassThrough = b
			}
		case "enableNuke":
			if b, err := strconv.ParseBool(val); err == nil {
				enableNuke = b
			}
		case "enableReverseGravity":
			if b, err := strconv.ParseBool(val); err == nil {
				enableReverseGravity = b
			}
		case "enableDualBalls":
			if b, err := strconv.ParseBool(val); err == nil {
				enableDualBalls = b
			}
		case "enableBlackHole":
			if b, err := strconv.ParseBool(val); err == nil {
				enableBlackHole = b
			}
		case "enableMagnet":
			if b, err := strconv.ParseBool(val); err == nil {
				enableMagnet = b
			}
		case "enableInfluencer":
			if b, err := strconv.ParseBool(val); err == nil {
				enableInfluencer = b
			}
		case "enableZapper":
			if b, err := strconv.ParseBool(val); err == nil {
				enableZapper = b
			}
		case "enableBreakUnbreakable":
			if b, err := strconv.ParseBool(val); err == nil {
				enableBreakUnbreakable = b
			}
		case "paddleWidth":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				paddleWidth = f
			}
		case "paddleHeight":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f > 0 {
				paddleHeight = f
			}
		default:
			log("Unknown level variable: " + key)
		}
	}
}

// ---- Physics ----
func resolveCollisionBall(b *Ball, nx, ny, surfVx, surfVy float64) {
	// nx,ny point from the surface toward the ball, so the point touching
	// the surface is on the opposite side of the ball.
	cx := b.x - nx*b.r
	cy := b.y - ny*b.r

	contactVx := b.vx - b.omega*(cy-b.y)
	contactVy := b.vy + b.omega*(cx-b.x)
	relVx := contactVx - surfVx
	relVy := contactVy - surfVy

	vn := relVx*nx + relVy*ny
	vt := relVx*(-ny) + relVy*nx
	if vn >= 0 {
		return
	}

	vnNew := -restitution * vn
	deltaVn := vnNew - vn

	// For a solid disk, a tangential impulse changes contact velocity three
	// times as much as it changes centre velocity: once through translation
	// and twice through rotation. Therefore no-slip correction is -vt/3.
	maxFriction := frictionCoeff * math.Abs(deltaVn)
	desiredDeltaVt := -vt / 3.0
	deltaVt := math.Max(
		-maxFriction,
		math.Min(maxFriction, desiredDeltaVt),
	)

	tx := -ny
	ty := nx
	b.vx += deltaVn*nx + deltaVt*tx
	b.vy += deltaVn*ny + deltaVt*ty
	b.omega -= 2 * deltaVt / b.r
	b.omega = clampFloat(b.omega, -maxSpin, maxSpin)
}

// ---- Build bricks from level layout (with auto-scaling) ----
func buildBricksFromLevel(lvl levelData) {
	bricks = nil
	brickGrid = make(map[int][]int)
	remainingBreakableBricks = 0
	lastBrickSoundPlayed = false
	if len(lvl.layout) == 0 {
		initBricksDefault()
		return
	}

	rows := len(lvl.layout)
	cols := 0
	for _, line := range lvl.layout {
		trimmed := strings.TrimRight(line, " ")
		if len(trimmed) > cols {
			cols = len(trimmed)
		}
	}
	if rows == 0 || cols == 0 {
		initBricksDefault()
		return
	}

	totalWidth := float64(cols)*(brickWidth+brickPadding) - brickPadding
	totalHeight := float64(rows)*(brickHeight+brickPadding) - brickPadding
	margin := 20.0
	scale := math.Min((canvasWidth-2*margin)/totalWidth, (canvasHeight-2*margin)/totalHeight)
	if scale > 1 {
		scale = 1
	}

	scaledWidth := brickWidth * scale
	scaledHeight := brickHeight * scale
	scaledPadding := brickPadding * scale
	scaledTotalWidth := float64(cols)*(scaledWidth+scaledPadding) - scaledPadding
	scaledTotalHeight := float64(rows)*(scaledHeight+scaledPadding) - scaledPadding
	gridOffsetLeft = (canvasWidth - scaledTotalWidth) / 2

	if brickOffsetTop >= 0 {
		gridOffsetTop = math.Max(brickOffsetTop, margin)
		if gridOffsetTop+scaledTotalHeight > canvasHeight-margin {
			gridOffsetTop = canvasHeight - margin - scaledTotalHeight
		}
	} else {
		gridOffsetTop = (canvasHeight - scaledTotalHeight) / 2
	}
	gridCellWidth = scaledWidth + scaledPadding
	gridCellHeight = scaledHeight + scaledPadding
	gridRows, gridCols = rows, cols

	for row := 0; row < rows; row++ {
		line := strings.TrimRight(lvl.layout[row], " ")
		for col := 0; col < cols; col++ {
			ch := byte(' ')
			if col < len(line) {
				ch = line[col]
			}
			if ch == 'X' {
				if rand.Float64() < unbreakableChance {
					ch = 'U'
				} else if rand.Float64() < magicChance {
					ch = 'M'
				} else {
					ch = '#'
				}
			}
			if ch == ' ' {
				continue
			}
			br := brick{
				x:     gridOffsetLeft + float64(col)*gridCellWidth,
				y:     gridOffsetTop + float64(row)*gridCellHeight,
				w:     scaledWidth,
				h:     scaledHeight,
				row:   row,
				col:   col,
				alive: true,
			}
			br.unbreakable = ch == 'U'
			br.magic = ch == 'M'
			bricks = append(bricks, br)
			registerBrick(len(bricks) - 1)
		}
	}
	initialBreakableBricks = remainingBreakableBricks
	bricksDirty = true
}

// ---- Fallback default bricks (also scaled) ----
func initBricksDefault() {
	bricks = nil
	brickGrid = make(map[int][]int)
	remainingBreakableBricks = 0
	lastBrickSoundPlayed = false

	// The built-in fallback behaves as if it has empty rows below its
	// visible bricks. The complete imaginary layout is still vertically
	// centered, which moves the actual brick grid upward.
	const extraEmptyRowsBelow = 5

	totalWidth := float64(brickCols)*(brickWidth+brickPadding) - brickPadding
	layoutRows := brickRows + extraEmptyRowsBelow
	totalHeight := float64(layoutRows)*(brickHeight+brickPadding) - brickPadding
	margin := 20.0
	scale := math.Min((canvasWidth-2*margin)/totalWidth, (canvasHeight-2*margin)/totalHeight)
	if scale > 1 {
		scale = 1
	}

	scaledWidth := brickWidth * scale
	scaledHeight := brickHeight * scale
	scaledPadding := brickPadding * scale
	scaledTotalWidth := float64(brickCols)*(scaledWidth+scaledPadding) - scaledPadding
	scaledTotalHeight := float64(layoutRows)*(scaledHeight+scaledPadding) - scaledPadding
	gridOffsetLeft = (canvasWidth - scaledTotalWidth) / 2
	if brickOffsetTop >= 0 {
		gridOffsetTop = math.Max(brickOffsetTop, margin)
		if gridOffsetTop+scaledTotalHeight > canvasHeight-margin {
			gridOffsetTop = canvasHeight - margin - scaledTotalHeight
		}
	} else {
		gridOffsetTop = (canvasHeight - scaledTotalHeight) / 2
	}
	gridCellWidth = scaledWidth + scaledPadding
	gridCellHeight = scaledHeight + scaledPadding
	gridRows, gridCols = brickRows, brickCols

	rand.Seed(1)
	for row := 0; row < brickRows; row++ {
		for col := 0; col < brickCols; col++ {
			unbreakable := rand.Float64() < unbreakableChance
			magic := !unbreakable && rand.Float64() < magicChance
			bricks = append(bricks, brick{
				x: gridOffsetLeft + float64(col)*gridCellWidth,
				y: gridOffsetTop + float64(row)*gridCellHeight,
				w: scaledWidth, h: scaledHeight,
				row: row, col: col, alive: true,
				unbreakable: unbreakable, magic: magic,
			})
			registerBrick(len(bricks) - 1)
		}
	}
	initialBreakableBricks = remainingBreakableBricks
	bricksDirty = true
}

// ---- Nuke ----
func nukeBricks(hitBrick *brick) {
	if hitBrick == nil {
		return
	}
	for i := range bricks {
		br := &bricks[i]
		if !br.alive {
			continue
		}
		if (br.row == hitBrick.row && abs(br.col-hitBrick.col) <= 2) ||
			(br.col == hitBrick.col && abs(br.row-hitBrick.row) <= 2) {
			destroyAnyBrick(br)
		}
	}
}

// Destroy one random living unbreakable brick.
func breakRandomUnbreakable() bool {
	var candidates []int
	for i := range bricks {
		if bricks[i].alive && bricks[i].unbreakable {
			candidates = append(candidates, i)
		}
	}
	if len(candidates) == 0 {
		return false
	}

	index := candidates[rand.Intn(len(candidates))]
	return destroyAnyBrick(&bricks[index])
}

// Activate powerup
func activatePowerUpWithBrick(hitBrick *brick) {
	if activePowerUp == POWER_LOW_GRAVITY || activePowerUp == POWER_REVERSE_GRAVITY {
		currentGravity = baseGravity
	}
	// Black Hole persists independently.

	var available []int
	if enableLowGravity {
		available = append(available, POWER_LOW_GRAVITY)
	}
	if enablePassThrough {
		available = append(available, POWER_PASS)
	}
	if enableNuke {
		available = append(available, POWER_NUKE)
	}
	if enableReverseGravity {
		available = append(available, POWER_REVERSE_GRAVITY)
	}
	if enableDualBalls {
		available = append(available, POWER_DUAL_BALLS)
	}
	if enableBlackHole {
		available = append(available, POWER_BLACKHOLE)
	}
	if enableMagnet {
		available = append(available, POWER_MAGNET)
	}
	if enableInfluencer {
		available = append(available, POWER_INFLUENCER)
	}
	if enableZapper {
		available = append(available, POWER_ZAPPER)
	}
	if enableBreakUnbreakable {
		for i := range bricks {
			if bricks[i].alive && bricks[i].unbreakable {
				available = append(available, POWER_BREAK_UNBREAKABLE)
				break
			}
		}
	}

	if len(available) == 0 {
		return
	}

	p := available[rand.Intn(len(available))]

	if blackHoleActive && (p == POWER_LOW_GRAVITY || p == POWER_REVERSE_GRAVITY) {
		return
	}

	switch p {
	case POWER_LOW_GRAVITY:
		activePowerUp = POWER_LOW_GRAVITY
		powerUpTimer = powerUpDuration
		currentGravity = baseGravity / 3
		showStatus("Low Gravity!", powerUpDuration)
		playPowerup()
	case POWER_PASS:
		activePowerUp = POWER_PASS
		powerUpTimer = powerUpDuration
		showStatus("Pass Through!", powerUpDuration)
		playPowerup()
	case POWER_NUKE:
		nukeBricks(hitBrick)
		showStatus("Nuke!", 2.0)
		playPowerup()
	case POWER_REVERSE_GRAVITY:
		activePowerUp = POWER_REVERSE_GRAVITY
		powerUpTimer = powerUpDuration
		currentGravity = -baseGravity
		showStatus("Reverse Gravity!", powerUpDuration)
		playPowerup()
	case POWER_DUAL_BALLS:
		if !secondBallActive {
			secondBallActive = true
			secondBall.x = ball.x
			secondBall.y = ball.y
			secondBall.vx = -ball.vx
			secondBall.vy = ball.vy
			secondBall.omega = -ball.omega
			secondBall.angle = ball.angle
			secondBall.stuckTimer = 0
			secondBall.r = ball.r
			showStatus("Dual Balls!", 2.0)
			playPowerup()
		} else {
			ball.vx *= 1.1
			ball.vy *= 1.1
			showStatus("Speed Boost!", 2.0)
			playPowerup()
		}
	case POWER_BLACKHOLE:
		blackHoleActive = true
		blackHoleTimer = powerUpDuration
		currentGravity = 0

		// Move back and forth within a limited range around screen center.
		blackHoleX = canvasWidth / 2
		blackHoleY = canvasHeight / 2
		blackHoleSpeed = (4 * blackHoleRange) / math.Max(powerUpDuration, 0.001)
		if rand.Intn(2) == 0 {
			blackHoleDirection = 1
		} else {
			blackHoleDirection = -1
		}

		showStatus("Black Hole!", powerUpDuration)
		playPowerup()
		if activePowerUp == POWER_LOW_GRAVITY || activePowerUp == POWER_REVERSE_GRAVITY {
			activePowerUp = POWER_NONE
			powerUpTimer = 0
		}
	case POWER_MAGNET:
		activePowerUp = POWER_MAGNET
		powerUpTimer = powerUpDuration
		showStatus("Magnets!", powerUpDuration)
		playPowerup()
	case POWER_INFLUENCER:
		influencerActive = true
		influencerTimer = powerUpDuration
		showStatus("Influencer!", powerUpDuration)
		playPowerup()
	case POWER_ZAPPER:
		activePowerUp = POWER_ZAPPER
		powerUpTimer = powerUpDuration
		zapperTargetIndex = -1
		zapperHitTimer = 0
		showStatus("Zapper!", powerUpDuration)
		playPowerup()
	case POWER_BREAK_UNBREAKABLE:
		if breakRandomUnbreakable() {
			showStatus("Unbreakable destroyed!", 2.0)
			playPowerup()
		}
	}
}

// ---- Load levels ----
func loadLevels() {
	levelIndex := 1
	for {
		// Cache-bust each request while developing so edited level files
		// are always fetched fresh instead of reused from browser cache.
		timestamp := int64(js.Global().Get("Date").Call("now").Float())
		url := "levels/level" + strconv.Itoa(levelIndex) + ".txt?t=" + strconv.FormatInt(timestamp, 10)

		xhr := js.Global().Get("XMLHttpRequest").New()
		xhr.Call("open", "GET", url, false)
		xhr.Call("setRequestHeader", "Cache-Control", "no-cache")
		xhr.Call("setRequestHeader", "Pragma", "no-cache")
		xhr.Call("send")
		status := xhr.Get("status").Int()
		if status == 404 {
			break
		}
		if status != 200 {
			log("Error loading level " + strconv.Itoa(levelIndex) + ": status " + strconv.Itoa(status))
			break
		}
		text := xhr.Get("responseText").String()
		lines := strings.Split(text, "\n")
		var configLines []string
		var layoutLines []string
		inLayout := false
		for _, rawLine := range lines {
			line := strings.TrimSpace(rawLine)
			if line == "---" {
				inLayout = true
				continue
			}
			if !inLayout {
				if line != "" && !strings.HasPrefix(line, "#") {
					configLines = append(configLines, line)
				}
			} else {
				layoutLine := strings.TrimRight(rawLine, "\r\n ")
				layoutLines = append(layoutLines, layoutLine)
			}
		}
		configMap := make(map[string]string)
		for _, cfg := range configLines {
			parts := strings.SplitN(cfg, "=", 2)
			if len(parts) != 2 {
				continue
			}
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			configMap[key] = val
		}
		levels = append(levels, levelData{layout: layoutLines, config: configMap})
		levelIndex++
	}
	if len(levels) == 0 {
		log("No level files found; using default layout")
		defaultLayout := []string{
			"##################",
			"#  #  #  #  #  #",
			"#  #  #  #  #  #",
			"##################",
		}
		levels = append(levels, levelData{layout: defaultLayout, config: make(map[string]string)})
	}
}

// ---- Remember current level in browser storage ----
const savedLevelKey = "breakout.currentLevel"

func saveCurrentLevel() {
	defer func() {
		if r := recover(); r != nil {
			js.Global().Get("console").Call("warn", "Could not save current level:", fmt.Sprint(r))
		}
	}()

	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	storage.Call("setItem", savedLevelKey, strconv.Itoa(currentLevelIndex))
}

func loadSavedLevel() int {
	if len(levels) == 0 {
		return 0
	}

	savedIndex := 0
	func() {
		defer func() {
			if r := recover(); r != nil {
				js.Global().Get("console").Call("warn", "Could not load saved level:", fmt.Sprint(r))
			}
		}()

		storage := js.Global().Get("localStorage")
		if storage.IsUndefined() || storage.IsNull() {
			return
		}

		value := storage.Call("getItem", savedLevelKey)
		if value.IsUndefined() || value.IsNull() {
			return
		}

		index, err := strconv.Atoi(value.String())
		if err == nil {
			savedIndex = index
		}
	}()

	if savedIndex < 0 {
		return 0
	}
	if savedIndex >= len(levels) {
		return len(levels) - 1
	}
	return savedIndex
}

// ---- Start a level ----
func startLevel(index int) {
	if index >= len(levels) {
		gameOver = true
		win = true
		return
	}
	paused = false
	waitingForStart = true
	levelCompleteTimer = 0
	levelAdvancePending = false
	leftPressed = false
	rightPressed = false
	touchControlActive = false
	mobileLeftHeld = false
	mobileRightHeld = false
	mobileLeftPointerID = -1
	mobileRightPointerID = -1
	paddle.vx = 0

	// User toggles never carry into a new level.
	magnetCheat = false
	zapperCheat = false

	// Reset all globals and level-only state before applying config.
	resetGlobals()
	levelMagnetActive = false
	levelZapperActive = false

	// Apply level-specific config. A level can use magnet=true.
	applyConfig(levels[index].config)

	paddle.w = paddleWidth
	paddle.h = paddleHeight

	ball.x, ball.y = startBallX, startBallY
	ball.vx, ball.vy = startBallVx, startBallVy
	ball.omega, ball.angle = 0, 0
	ball.stuckTimer = 0
	ball.r = ballRadius
	secondBallActive = false
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.y = canvasHeight - 40
	paddle.vx = 0
	paddlePreviousX = paddle.x

	activePowerUp = POWER_NONE
	powerUpTimer = 0
	blackHoleActive = false
	blackHoleTimer = 0
	blackHoleX = canvasWidth / 2
	blackHoleY = canvasHeight / 2
	blackHoleDirection = 0
	blackHoleSpeed = 0
	currentGravity = baseGravity
	influencerActive = false
	influencerTimer = 0
	zapperTargetIndex = -1
	zapperHitTimer = 0
	secondZapperTargetIndex = -1
	secondZapperHitTimer = 0
	statusMessages = nil
	if levelMagnetActive {
		showStatus("Magnets enabled by level", 2.0)
	}
	if levelZapperActive {
		showStatus("Zapper enabled by level", 2.0)
	}

	buildBricksFromLevel(levels[index])
	currentLevelIndex = index
	saveCurrentLevel()
	log(fmt.Sprintf(
		"Level %d: restitution=%.3f friction=%.3f maxSpin=%.1f paddle=%.1fx%.1f boost=%.1f",
		index+1, restitution, frictionCoeff, maxSpin, paddle.w, paddle.h, paddleBoost,
	))
}

// ---- Jump to a specific level (cheat) ----
func jumpToLevel(index int) {
	if index < 0 {
		index = 0
	}
	if index >= len(levels) {
		index = len(levels) - 1
	}

	lives = defaultLives
	score = 0

	gameOver = false
	win = false
	startLevel(index)
}

// ---- Reset game ----
func resetGame() {
	lives = defaultLives
	score = 0
	gameOver = false
	win = false
	paused = false
	leftPressed = false
	rightPressed = false
	startLevel(loadSavedLevel())
}

// ---- Retry the level after losing all lives ----
func retryCurrentLevel() {
	lives = defaultLives
	score = 0
	gameOver = false
	win = false
	paused = false
	leftPressed = false
	rightPressed = false
	paddle.vx = 0
	startLevel(currentLevelIndex)
}

// ---- Magnetic breakable bricks ----
func magnetIsActive() bool {
	return activePowerUp == POWER_MAGNET || (levelMagnetActive != magnetCheat)
}

func applyBrickMagnetism(b *Ball, dt float64) {
	if !magnetIsActive() {
		return
	}

	nearestIndex := -1
	nearestDistanceSquared := magnetRange * magnetRange
	for i := range bricks {
		br := &bricks[i]
		if !br.alive || br.unbreakable {
			continue
		}
		dx := br.x + br.w/2 - b.x
		dy := br.y + br.h/2 - b.y
		distanceSquared := dx*dx + dy*dy
		if distanceSquared > 1 && distanceSquared < nearestDistanceSquared {
			nearestIndex = i
			nearestDistanceSquared = distanceSquared
		}
	}
	if nearestIndex < 0 {
		return
	}

	nearest := &bricks[nearestIndex]
	dx := nearest.x + nearest.w/2 - b.x
	dy := nearest.y + nearest.h/2 - b.y
	nearestDistance := math.Sqrt(nearestDistanceSquared)
	falloff := 1 - nearestDistance/magnetRange
	acceleration := magnetStrength * falloff
	b.vx += dx / nearestDistance * acceleration * dt
	b.vy += dy / nearestDistance * acceleration * dt
}

// ---- Influencer area damage ----
func destroyBricksInRadius(ballX, ballY, radius float64) {
	radiusSquared := radius * radius
	for i := range bricks {
		br := &bricks[i]
		if !br.alive || br.unbreakable {
			continue
		}
		closestX := math.Max(br.x, math.Min(ballX, br.x+br.w))
		closestY := math.Max(br.y, math.Min(ballY, br.y+br.h))
		dx := ballX - closestX
		dy := ballY - closestY
		if dx*dx+dy*dy <= radiusSquared && destroyBrick(br) {
			playBrickBreak()
		}
	}
}

func candidateBrickIndices(b *Ball) []int {
	if gridRows <= 0 || gridCols <= 0 || gridCellWidth <= 0 || gridCellHeight <= 0 {
		indices := make([]int, len(bricks))
		for i := range bricks {
			indices[i] = i
		}
		return indices
	}

	minCol := int(math.Floor((b.x - b.r - gridOffsetLeft) / gridCellWidth))
	maxCol := int(math.Floor((b.x + b.r - gridOffsetLeft) / gridCellWidth))
	minRow := int(math.Floor((b.y - b.r - gridOffsetTop) / gridCellHeight))
	maxRow := int(math.Floor((b.y + b.r - gridOffsetTop) / gridCellHeight))
	minCol, maxCol = minCol-1, maxCol+1
	minRow, maxRow = minRow-1, maxRow+1
	if minCol < 0 {
		minCol = 0
	}
	if minRow < 0 {
		minRow = 0
	}
	if maxCol >= gridCols {
		maxCol = gridCols - 1
	}
	if maxRow >= gridRows {
		maxRow = gridRows - 1
	}

	indices := make([]int, 0, 9)
	for row := minRow; row <= maxRow; row++ {
		for col := minCol; col <= maxCol; col++ {
			indices = append(indices, brickGrid[gridKey(row, col)]...)
		}
	}
	return indices
}

func playTilt() {
	scheduleTone("triangle", varyFreq(130, 0.05), varyFreq(360, 0.05), 0.12, 0.14, 0)
	scheduleTone("sine", varyFreq(70, 0.04), varyFreq(115, 0.04), 0.18, 0.08, 0.02)
}

func applyTilt(b *Ball) {
	sideMin, sideMax := tiltSideMin, tiltSideMax
	if sideMax < sideMin {
		sideMin, sideMax = sideMax, sideMin
	}
	sideSpeed := sideMin
	if sideMax > sideMin {
		sideSpeed += rand.Float64() * (sideMax - sideMin)
	}

	if rand.Intn(2) == 0 {
		sideSpeed = -sideSpeed
	}

	b.vx = sideSpeed
	b.vy = -tiltUpSpeed
	b.omega += (rand.Float64()*2 - 1) * 8
	b.stuckTimer = 0

	showStatus("TILT!", 1.5)
	playTilt()
}

func updateStuckDetector(b *Ball, dt float64) {
	if b.y+b.r > canvasHeight {
		b.stuckTimer = 0
		return
	}

	speed := math.Sqrt(b.vx*b.vx + b.vy*b.vy)

	if speed < stuckSpeedThreshold {
		b.stuckTimer += dt
		if b.stuckTimer >= stuckDuration {
			applyTilt(b)
		}
	} else {
		b.stuckTimer = 0
	}
}

func nearestBreakableBrickIndex(x, y float64) int {
	nearest := -1
	nearestDistanceSquared := zapperRange * zapperRange

	for i := range bricks {
		br := &bricks[i]
		if !br.alive || br.unbreakable {
			continue
		}

		dx := br.x + br.w/2 - x
		dy := br.y + br.h/2 - y
		distanceSquared := dx*dx + dy*dy

		if distanceSquared < nearestDistanceSquared {
			nearest = i
			nearestDistanceSquared = distanceSquared
		}
	}

	return nearest
}

func zapperIsActive() bool {
	threshold := zapperThreshold()
	autoActive := threshold > 0 &&
		remainingBreakableBricks > 0 &&
		remainingBreakableBricks <= threshold

	return autoActive ||
		activePowerUp == POWER_ZAPPER ||
		(levelZapperActive != zapperCheat)
}

func zapperThreshold() int {
	if initialBreakableBricks <= 0 {
		return 0
	}

	threshold := int(math.Ceil(float64(initialBreakableBricks) * 0.03))
	if threshold < 1 {
		threshold = 1
	}
	return threshold
}

func updateOneZapper(
	b *Ball,
	active bool,
	targetIndex *int,
	hitTimer *float64,
	dt float64,
) {
	if !active {
		*targetIndex = -1
		*hitTimer = 0
		return
	}

	target := nearestBreakableBrickIndex(b.x, b.y)
	if target < 0 {
		*targetIndex = -1
		*hitTimer = 0
		return
	}

	if target != *targetIndex {
		*targetIndex = target
		*hitTimer = 0
	}

	*hitTimer += dt
	if *hitTimer < defaultZapperHitTime {
		return
	}

	br := &bricks[*targetIndex]
	if destroyBrick(br) {
		playZapper()
	}

	*targetIndex = -1
	*hitTimer = 0
}

func updateZapper(dt float64) {
	if !zapperIsActive() {
		zapperTargetIndex = -1
		zapperHitTimer = 0
		secondZapperTargetIndex = -1
		secondZapperHitTimer = 0
		return
	}

	updateOneZapper(
		&ball,
		true,
		&zapperTargetIndex,
		&zapperHitTimer,
		dt,
	)

	updateOneZapper(
		&secondBall,
		secondBallActive,
		&secondZapperTargetIndex,
		&secondZapperHitTimer,
		dt,
	)
}

// ---- Update a single ball ----
func updateBall(b *Ball, dt float64, isPrimary bool) {
	b.vy += currentGravity * dt
	applyBrickMagnetism(b, dt)

	if blackHoleActive {
		dx := blackHoleX - b.x
		dy := blackHoleY - b.y
		dist := math.Sqrt(dx*dx + dy*dy)
		if dist > 1.0 {
			accX := (dx / dist) * blackHoleStrength
			accY := (dy / dist) * blackHoleStrength
			b.vx += accX * dt
			b.vy += accY * dt
		} else {
			b.vx += (rand.Float64() - 0.5) * 10.0
			b.vy += (rand.Float64() - 0.5) * 10.0
		}
	}

	b.x += b.vx * dt
	b.y += b.vy * dt
	b.angle += b.omega * dt

	// Walls
	if b.x-b.r < 0 {
		b.x = b.r
		resolveCollisionBall(b, 1, 0, 0, 0)
		playWallHit()
	}
	if b.x+b.r > canvasWidth {
		b.x = canvasWidth - b.r
		resolveCollisionBall(b, -1, 0, 0, 0)
		playWallHit()
	}
	if b.y-b.r < 0 {
		b.y = b.r
		resolveCollisionBall(b, 0, 1, 0, 0)
		playWallHit()
	}
	if b.y+b.r > canvasHeight {
		// Ball lost handled by caller
		return
	}

	// Paddle
	pLeft, pRight := paddle.x, paddle.x+paddle.w
	pTop, pBottom := paddle.y, paddle.y+paddle.h
	if b.vy > 0 &&
		b.x+b.r > pLeft && b.x-b.r < pRight &&
		b.y+b.r > pTop && b.y+b.r < pBottom {
		b.y = pTop - b.r

		hitPos := (b.x - pLeft) / paddle.w
		if hitPos < 0 {
			hitPos = 0
		}
		if hitPos > 1 {
			hitPos = 1
		}

		angle := (hitPos - 0.5) * 2.0 * (80.0 * math.Pi / 180.0)

		speed := math.Sqrt(b.vx*b.vx + b.vy*b.vy)
		if speed < 100 {
			speed = 100
		}

		// Restitution controls retained speed; paddleBoost represents energy
		// actively supplied by the paddle.
		speed = speed*restitution + paddleBoost
		if speed > maxSpeed {
			speed = maxSpeed
		}

		incomingVy := b.vy

		b.vx = speed * math.Sin(angle)
		b.vy = -speed * math.Cos(angle)

		// The moving paddle drags the bottom contact point of the ball.
		// Positive omega is clockwise on canvas, so a paddle moving right
		// normally creates negative (counter-clockwise) omega.
		relativeSlip := b.vx - b.omega*b.r - paddle.vx
		normalDeltaSpeed := math.Abs(b.vy - incomingVy)
		maxFrictionDelta := frictionCoeff * normalDeltaSpeed
		desiredDeltaVx := -relativeSlip / 3.0
		deltaVx := math.Max(
			-maxFrictionDelta,
			math.Min(maxFrictionDelta, desiredDeltaVx),
		)

		b.vx += deltaVx
		b.omega -= 2 * deltaVx / b.r

		// Keep the game's configured speed limit after paddle friction.
		postCollisionSpeed := math.Hypot(b.vx, b.vy)
		if postCollisionSpeed > maxSpeed {
			scale := maxSpeed / postCollisionSpeed
			b.vx *= scale
			b.vy *= scale
		}

		b.omega = clampFloat(b.omega, -maxSpin, maxSpin)
		playPaddleHit()
	}

	// ---- Bricks ----
	for _, i := range candidateBrickIndices(b) {
		brickPtr := &bricks[i]
		if !brickPtr.alive {
			continue
		}
		if b.x+b.r > brickPtr.x && b.x-b.r < brickPtr.x+brickPtr.w &&
			b.y+b.r > brickPtr.y && b.y-b.r < brickPtr.y+brickPtr.h {

			if activePowerUp == POWER_PASS {
				if destroyBrick(brickPtr) {
					playBrickBreak()
					if influencerActive {
						destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier)
					}
				}
				continue
			}

			// Proper circle-vs-AABB collision. The closest point gives a
			// diagonal normal at corners instead of forcing an axis-only bounce.
			closestX := clampFloat(b.x, brickPtr.x, brickPtr.x+brickPtr.w)
			closestY := clampFloat(b.y, brickPtr.y, brickPtr.y+brickPtr.h)
			dx := b.x - closestX
			dy := b.y - closestY
			distanceSquared := dx*dx + dy*dy

			var nx, ny float64
			if distanceSquared > 1e-12 {
				distance := math.Sqrt(distanceSquared)
				nx = dx / distance
				ny = dy / distance
				penetration := b.r - distance
				if penetration > 0 {
					b.x += nx * penetration
					b.y += ny * penetration
				}
			} else {
				// The centre is inside the rectangle. Push it through the nearest
				// face; adaptive substeps make this fallback uncommon.
				left := b.x - brickPtr.x
				right := brickPtr.x + brickPtr.w - b.x
				top := b.y - brickPtr.y
				bottom := brickPtr.y + brickPtr.h - b.y

				minimum := left
				nx, ny = -1, 0
				push := left + b.r
				if right < minimum {
					minimum = right
					nx, ny = 1, 0
					push = right + b.r
				}
				if top < minimum {
					minimum = top
					nx, ny = 0, -1
					push = top + b.r
				}
				if bottom < minimum {
					nx, ny = 0, 1
					push = bottom + b.r
				}
				b.x += nx * push
				b.y += ny * push
			}

			resolveCollisionBall(b, nx, ny, 0, 0)

			if brickPtr.unbreakable {
				playUnbreakable()
				continue
			}

			if brickPtr.magic {
				if isPrimary {
					activatePowerUpWithBrick(brickPtr)
				}
				destroyBrick(brickPtr)
				playMagic()
				if influencerActive {
					destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier)
				}
				continue
			}

			// Normal brick
			if brickBoost != 0 {
				b.vy -= brickBoost
				if math.Abs(b.vy) > maxSpeed {
					b.vy = math.Copysign(maxSpeed, b.vy)
				}
				if math.Abs(b.vx) > maxSpeed {
					b.vx = math.Copysign(maxSpeed, b.vx)
				}
			}
			if destroyBrick(brickPtr) {
				playBrickBreak()
			}
			if influencerActive {
				destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier)
			}
		}
	}
	updateStuckDetector(b, dt)
}

func loseLife() {
	if gameOver || levelAdvancePending || remainingBreakableBricks == 0 || lives <= 0 {
		lives = 0
		return
	}

	lives--
	if lives < 0 {
		lives = 0
	}

	playDie()

	if lives == 0 {
		gameOver = true
		win = false
		playGameOver()
	} else {
		resetBalls()
	}
}

const mobileControlStorageKey = "breakout.mobileControlMode"

func validMobileControlMode(mode string) bool {
	switch mode {
	case "vertical", "tilt", "follow", "two-thumb":
		return true
	default:
		return false
	}
}

func resetMobileControlState() {
	touchControlActive = false
	touchPointerID = -1
	mobileLeftHeld = false
	mobileRightHeld = false
	mobileLeftPointerID = -1
	mobileRightPointerID = -1
	leftPressed = false
	rightPressed = false
	paddle.vx = 0
}

func saveMobileControlMode() {
	defer func() { _ = recover() }()
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	storage.Call("setItem", mobileControlStorageKey, mobileControlMode)
}

func selectMobileControlMode(mode string) {
	if !validMobileControlMode(mode) {
		return
	}

	mobileControlMode = mode
	resetMobileControlState()
	saveMobileControlMode()

	if mode == "tilt" {
		phoneTiltAvailable = false
		recalibratePhoneTilt()
		requestPhoneTiltPermission()
	}
}

func hideMobileControlSelector() {
	overlay := doc.Call("getElementById", "mobileControlSelector")
	if !overlay.IsUndefined() && !overlay.IsNull() {
		overlay.Get("style").Set("display", "none")
	}
	mobileSelectorVisible = false
}

func showMobileControlSelector() {
	if !mobileControlsEnabled {
		return
	}

	overlay := doc.Call("getElementById", "mobileControlSelector")
	if overlay.IsUndefined() || overlay.IsNull() {
		return
	}

	resetMobileControlState()
	mobileSelectorVisible = true
	overlay.Get("style").Set("display", "flex")
}

func setupMobileControlSelector() {
	window := js.Global().Get("window")
	navigator := js.Global().Get("navigator")

	maxTouchPoints := 0
	if !navigator.IsUndefined() && !navigator.IsNull() {
		value := navigator.Get("maxTouchPoints")
		if value.Type() == js.TypeNumber {
			maxTouchPoints = value.Int()
		}
	}

	coarsePointer := false
	matchMedia := window.Get("matchMedia")
	if matchMedia.Type() == js.TypeFunction {
		coarsePointer = window.Call("matchMedia", "(pointer: coarse)").Get("matches").Bool()
	}

	width := window.Get("innerWidth").Int()
	height := window.Get("innerHeight").Int()
	mobileControlsEnabled = (maxTouchPoints > 0 || coarsePointer) &&
		(width <= 1200 || height <= 800)

	if !mobileControlsEnabled {
		return
	}

	style := doc.Call("createElement", "style")
	style.Set("textContent", `
#mobileControlSelector {
	position: fixed;
	inset: 0;
	z-index: 1000;
	display: none;
	align-items: center;
	justify-content: center;
	padding: 18px;
	background: rgba(8, 10, 24, 0.94);
	font-family: GameFont, monospace;
}
#mobileControlPanel {
	width: min(620px, 96vw);
	padding: 20px;
	border: 1px solid rgba(255,255,255,0.22);
	border-radius: 12px;
	background: #16213e;
	box-shadow: 0 12px 40px rgba(0,0,0,0.45);
	color: white;
}
#mobileControlPanel h2 {
	margin: 0 0 8px;
	text-align: center;
	font-size: 24px;
}
#mobileControlPanel p {
	margin: 0 0 16px;
	text-align: center;
	color: rgba(255,255,255,0.68);
	font-size: 14px;
}
.mobileControlChoice {
	display: block;
	width: 100%;
	margin: 9px 0;
	padding: 12px 14px;
	border: 1px solid rgba(255,255,255,0.18);
	border-radius: 8px;
	background: rgba(255,255,255,0.07);
	color: white;
	text-align: left;
	font: 700 16px/1.25 GameFont, monospace;
	touch-action: manipulation;
}
.mobileControlChoice small {
	display: block;
	margin-top: 4px;
	color: rgba(255,255,255,0.60);
	font: 700 12px/1.3 GameFont, monospace;
}
.mobileControlChoice:active {
	background: rgba(255,255,255,0.18);
}
`)
	doc.Get("head").Call("appendChild", style)

	overlay := doc.Call("createElement", "div")
	overlay.Set("id", "mobileControlSelector")
	overlay.Set("innerHTML", `
<div id="mobileControlPanel">
	<h2>Choose controls</h2>
	<p>You can change this later with the Controls button.</p>
	<button class="mobileControlChoice" data-mode="vertical">
		Up / down drag
		<small>Slide one finger vertically. Down moves right; up moves left.</small>
	</button>
	<button class="mobileControlChoice" data-mode="tilt">
		Phone tilt
		<small>Rotate the phone left or right. Selecting this also calibrates it.</small>
	</button>
	<button class="mobileControlChoice" data-mode="follow">
		Follow finger
		<small>The paddle follows one finger's horizontal position.</small>
	</button>
	<button class="mobileControlChoice" data-mode="two-thumb">
		Two thumbs
		<small>Hold the left or right half of the screen to move.</small>
	</button>
</div>
`)
	doc.Get("body").Call("appendChild", overlay)

	settingsButton := doc.Call("getElementById", "mobileControlModeButton")
	if !settingsButton.IsUndefined() && !settingsButton.IsNull() {
		settingsCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			if len(args) > 0 {
				args[0].Call("preventDefault")
				args[0].Call("stopPropagation")
			}
			showMobileControlSelector()
			return nil
		})
		mobileControlCallbacks = append(mobileControlCallbacks, settingsCallback)
		settingsButton.Call("addEventListener", "pointerdown", settingsCallback)
	}

	buttons := overlay.Call("querySelectorAll", ".mobileControlChoice")
	for i := 0; i < buttons.Get("length").Int(); i++ {
		button := buttons.Index(i)
		mode := button.Get("dataset").Get("mode").String()

		callback := js.FuncOf(func(selectedMode string) func(js.Value, []js.Value) interface{} {
			return func(this js.Value, args []js.Value) interface{} {
				if len(args) > 0 {
					args[0].Call("preventDefault")
					args[0].Call("stopPropagation")
				}
				selectMobileControlMode(selectedMode)
				hideMobileControlSelector()
				return nil
			}
		}(mode))

		mobileControlCallbacks = append(mobileControlCallbacks, callback)
		button.Call("addEventListener", "pointerdown", callback)
	}

	savedMode := ""
	func() {
		defer func() { _ = recover() }()
		storage := js.Global().Get("localStorage")
		if storage.IsUndefined() || storage.IsNull() {
			return
		}
		value := storage.Call("getItem", mobileControlStorageKey)
		if !value.IsUndefined() && !value.IsNull() {
			savedMode = value.String()
		}
	}()

	if validMobileControlMode(savedMode) {
		selectMobileControlMode(savedMode)
	} else {
		showMobileControlSelector()
	}
}

// ---- Phone/tablet tilt paddle control ----
func screenOrientationAngle() int {
	screen := js.Global().Get("screen")
	if screen.IsUndefined() || screen.IsNull() {
		return 0
	}

	orientation := screen.Get("orientation")
	if !orientation.IsUndefined() && !orientation.IsNull() {
		angle := orientation.Get("angle")
		if angle.Type() == js.TypeNumber {
			return angle.Int()
		}
	}

	legacy := js.Global().Get("orientation")
	if legacy.Type() == js.TypeNumber {
		return legacy.Int()
	}

	return 0
}

func orientationTiltValue(event js.Value) (float64, bool) {
	betaValue := event.Get("beta")
	gammaValue := event.Get("gamma")
	if betaValue.Type() != js.TypeNumber || gammaValue.Type() != js.TypeNumber {
		return 0, false
	}

	beta := betaValue.Float()
	gamma := gammaValue.Float()

	switch ((screenOrientationAngle() % 360) + 360) % 360 {
	case 90:
		return beta, true
	case 270:
		return -beta, true
	default:
		return gamma, true
	}
}

func installPhoneTiltListener() {
	if phoneTiltListenerSet {
		return
	}
	phoneTiltListenerSet = true

	deviceOrientationCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 {
			return nil
		}

		value, ok := orientationTiltValue(args[0])
		if !ok || math.IsNaN(value) || math.IsInf(value, 0) {
			return nil
		}

		if !phoneTiltCalibrated {
			phoneTiltNeutral = value
			phoneTiltValue = 0
			phoneTiltCalibrated = true
		} else {
			phoneTiltValue = value - phoneTiltNeutral
		}

		phoneTiltAvailable = true
		return nil
	})

	js.Global().Call("addEventListener", "deviceorientation", deviceOrientationCallback)
}

func requestPhoneTiltPermission() {
	window := js.Global().Get("window")
	orientationEvent := window.Get("DeviceOrientationEvent")
	if orientationEvent.IsUndefined() || orientationEvent.IsNull() {
		orientationEvent = js.Global().Get("DeviceOrientationEvent")
	}
	if orientationEvent.IsUndefined() || orientationEvent.IsNull() {
		showStatus("Tilt unavailable", 2.0)
		return
	}

	requestPermission := orientationEvent.Get("requestPermission")
	if requestPermission.Type() != js.TypeFunction {
		installPhoneTiltListener()
		recalibratePhoneTilt()
		return
	}

	if phoneTiltPermissionAsked && phoneTiltListenerSet {
		recalibratePhoneTilt()
		return
	}

	phoneTiltPermissionAsked = true
	promise := orientationEvent.Call("requestPermission")

	granted := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 && args[0].String() == "granted" {
			installPhoneTiltListener()
			recalibratePhoneTilt()
			showStatus("Tilt ready", 1.5)
		} else {
			phoneTiltPermissionAsked = false
			showStatus("Tilt permission denied", 2.0)
		}
		return nil
	})
	rejected := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		phoneTiltPermissionAsked = false
		showStatus("Tilt unavailable", 2.0)
		return nil
	})

	js.Global().Set("__breakoutTiltGranted", granted)
	js.Global().Set("__breakoutTiltRejected", rejected)
	promise.Call("then", granted).Call("catch", rejected)
}

func recalibratePhoneTilt() {
	phoneTiltCalibrated = false
	phoneTiltValue = 0
}

func applyPhoneTiltControl(dt float64) bool {
	if !phoneTiltAvailable || !phoneTiltCalibrated || dt <= 0 {
		return false
	}

	tilt := phoneTiltValue
	if math.Abs(tilt) <= defaultPhoneTiltDeadZone {
		tilt = 0
	} else {
		tilt = math.Copysign(math.Abs(tilt)-defaultPhoneTiltDeadZone, tilt)
	}

	usableRange := defaultPhoneTiltMaxAngle - defaultPhoneTiltDeadZone
	if usableRange <= 0 {
		return false
	}

	normalized := tilt / usableRange
	if normalized < -1 {
		normalized = -1
	}
	if normalized > 1 {
		normalized = 1
	}

	availableWidth := canvasWidth - paddle.w
	phoneTiltTargetX = (normalized + 1) * 0.5 * availableWidth

	oldX := paddle.x
	smoothing := 1 - math.Exp(-defaultPhoneTiltSmoothing*dt)
	paddle.x += (phoneTiltTargetX - paddle.x) * smoothing
	paddle.vx = (paddle.x - oldX) / dt

	return true
}

// Update a ball with adaptive substeps so it cannot cross a thin brick in
// one large frame. The cap prevents pathological slowdown.
func updateBallAdaptive(b *Ball, dt float64, isPrimary bool) {
	if dt <= 0 {
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	maxTravelPerStep := math.Max(b.r, 6.0)
	steps := int(math.Ceil(speed * dt / maxTravelPerStep))
	if steps < 1 {
		steps = 1
	}
	if steps > 6 {
		steps = 6
	}

	stepDT := dt / float64(steps)
	for step := 0; step < steps; step++ {
		updateBall(b, stepDT, isPrimary)
		if b.y+b.r > canvasHeight {
			break
		}
	}
}

// ---- Update (main loop) ----
func update(dt float64) {
	if gameOver || paused || waitingForStart {
		return
	}

	if levelAdvancePending {
		levelCompleteTimer -= dt
		if levelCompleteTimer <= 0 {
			levelCompleteTimer = 0
			levelAdvancePending = false

			nextLevel := currentLevelIndex + 1
			if nextLevel >= len(levels) {
				gameOver = true
				win = true
				playYouWin()
			} else {
				startLevel(nextLevel)
			}
		}
		return
	}

	const keyboardPaddleSpeed = 700.0
	const twoThumbPaddleSpeed = 1500.0

	if leftPressed && !rightPressed {
		paddle.vx = -keyboardPaddleSpeed
		paddle.x += paddle.vx * dt
	} else if rightPressed && !leftPressed {
		paddle.vx = keyboardPaddleSpeed
		paddle.x += paddle.vx * dt
	} else if mobileControlsEnabled {
		switch mobileControlMode {
		case "vertical", "follow":
			if !touchControlActive {
				paddle.vx = 0
			}
		case "two-thumb":
			if mobileLeftHeld && !mobileRightHeld {
				paddle.vx = -twoThumbPaddleSpeed
				paddle.x += paddle.vx * dt
			} else if mobileRightHeld && !mobileLeftHeld {
				paddle.vx = twoThumbPaddleSpeed
				paddle.x += paddle.vx * dt
			} else {
				paddle.vx = 0
			}
		case "tilt":
			if !applyPhoneTiltControl(dt) {
				paddle.vx = 0
			}
		default:
			paddle.vx = 0
		}
	} else {
		paddle.vx = 0
	}
	if paddle.x < 0 {
		paddle.x = 0
	}
	if paddle.x+paddle.w > canvasWidth {
		paddle.x = canvasWidth - paddle.w
	}

	// Use actual frame-to-frame movement for paddle collision friction.
	// This works consistently for keyboard, mouse, follow, vertical drag,
	// two-thumb control, and tilt.
	if dt > 0 {
		paddle.vx = (paddle.x - paddlePreviousX) / dt
	} else {
		paddle.vx = 0
	}
	paddlePreviousX = paddle.x

	// ---- Power-up timers ----
	if activePowerUp != POWER_NONE {
		powerUpTimer -= dt
		if powerUpTimer <= 0 {
			if activePowerUp == POWER_LOW_GRAVITY || activePowerUp == POWER_REVERSE_GRAVITY {
				if !blackHoleActive {
					currentGravity = baseGravity
				}
			}
			activePowerUp = POWER_NONE
		}
	}

	if blackHoleActive {
		blackHoleX += blackHoleDirection * blackHoleSpeed * dt

		leftLimit := canvasWidth/2 - blackHoleRange
		rightLimit := canvasWidth/2 + blackHoleRange

		if blackHoleX <= leftLimit {
			blackHoleX = leftLimit
			blackHoleDirection = 1
		}
		if blackHoleX >= rightLimit {
			blackHoleX = rightLimit
			blackHoleDirection = -1
		}

		blackHoleTimer -= dt
		if blackHoleTimer <= 0 {
			blackHoleActive = false
			blackHoleDirection = 0
			blackHoleSpeed = 0
			if activePowerUp != POWER_LOW_GRAVITY && activePowerUp != POWER_REVERSE_GRAVITY {
				currentGravity = baseGravity
			}
		}
	}

	if influencerActive {
		influencerTimer -= dt
		if influencerTimer <= 0 {
			influencerActive = false
			influencerTimer = 0
		}
	}

	updateZapper(dt)

	for i := len(statusMessages) - 1; i >= 0; i-- {
		statusMessages[i].timer -= dt
		if statusMessages[i].timer <= 0 {
			statusMessages = append(statusMessages[:i], statusMessages[i+1:]...)
		}
	}

	// Primary ball
	updateBallAdaptive(&ball, dt, true)

	primaryLost := ball.y+ball.r > canvasHeight
	secondLost := secondBallActive &&
		secondBall.y+secondBall.r > canvasHeight

	if secondBallActive {
		updateBallAdaptive(&secondBall, dt, false)

		if primaryLost && !secondLost {
			// Keep playing with the second ball. No life was lost.
			ball = secondBall
			secondBallActive = false
		} else if secondLost && !primaryLost {
			// Primary ball is still alive. No life was lost.
			secondBallActive = false
		} else if primaryLost && secondLost {
			// Both balls were lost.
			secondBallActive = false
			loseLife()
		}
	} else if primaryLost {
		// Only one ball was active.
		loseLife()
	}
	// Hold the cleared level on screen for three seconds.
	if !gameOver && !levelAdvancePending && remainingBreakableBricks == 0 {
		levelAdvancePending = true
		levelCompleteTimer = 3.0
		leftPressed = false
		rightPressed = false
		touchControlActive = false
		paddle.vx = 0
		zapperTargetIndex = -1
		zapperHitTimer = 0
		secondZapperTargetIndex = -1
		secondZapperHitTimer = 0
		playLevelComplete()
		showStatus("Level complete!", 3.0)
	}
}

// ---- Reset balls on life lost ----
func resetBalls() {
	ball.x, ball.y = startBallX, startBallY
	ball.vx, ball.vy = startBallVx, startBallVy
	ball.omega, ball.angle = 0, 0
	ball.stuckTimer = 0
	secondBall.stuckTimer = 0
	secondBallActive = false
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.vx = 0
	paddlePreviousX = paddle.x
}

func ensureBrickCanvas() {
	if brickCanvas.IsUndefined() || brickCanvas.IsNull() {
		brickCanvas = doc.Call("createElement", "canvas")
		brickCtx = brickCanvas.Call("getContext", "2d")
	}
	if brickCanvas.Get("width").Int() != int(canvasWidth) {
		brickCanvas.Set("width", int(canvasWidth))
	}
	if brickCanvas.Get("height").Int() != int(canvasHeight) {
		brickCanvas.Set("height", int(canvasHeight))
	}
}

func drawBrickGroup(target js.Value, kind int) {
	switch kind {
	case 0: // normal
		target.Set("fillStyle", palette[2])
		target.Set("strokeStyle", brickStrokeColor)
		target.Set("lineWidth", 1)
		target.Call("setLineDash", []interface{}{})
	case 1: // magic
		target.Set("fillStyle", magicColor)
		target.Set("strokeStyle", magicStrokeColor)
		target.Set("lineWidth", 2)
		target.Call("setLineDash", []interface{}{})
	case 2: // unbreakable
		target.Set("fillStyle", palette[6])
		target.Set("strokeStyle", unbreakableStrokeColor)
		target.Set("lineWidth", 2)
		target.Call("setLineDash", []interface{}{})
	}

	for i := range bricks {
		br := &bricks[i]
		if !br.alive {
			continue
		}
		matches := (kind == 0 && !br.magic && !br.unbreakable) ||
			(kind == 1 && br.magic) || (kind == 2 && br.unbreakable)
		if !matches {
			continue
		}
		target.Call("beginPath")
		target.Call("roundRect", br.x, br.y, br.w, br.h, brickRadius)
		target.Call("fill")
		target.Call("stroke")
	}
	target.Call("setLineDash", []interface{}{})
}

func rebuildBrickCanvas() {
	ensureBrickCanvas()
	brickCtx.Call("clearRect", 0, 0, canvasWidth, canvasHeight)
	drawBrickGroup(brickCtx, 0)
	drawBrickGroup(brickCtx, 1)
	drawBrickGroup(brickCtx, 2)
	bricksDirty = false
}

// ---- Draw ----
func drawOneZapperBolt(
	b *Ball,
	targetIndex int,
	strokeColor string,
	shadowColor string,
) {
	if targetIndex < 0 || targetIndex >= len(bricks) {
		return
	}

	br := &bricks[targetIndex]
	if !br.alive || br.unbreakable {
		return
	}

	startX, startY := b.x, b.y
	endX := br.x + br.w/2
	endY := br.y + br.h/2

	const segments = 12

	ctx.Call("save")
	ctx.Set("strokeStyle", strokeColor)
	ctx.Set("lineWidth", 3)
	ctx.Set("shadowColor", shadowColor)
	ctx.Set("shadowBlur", 12)
	ctx.Call("beginPath")
	ctx.Call("moveTo", startX, startY)

	for i := 1; i < segments; i++ {
		t := float64(i) / float64(segments)
		x := startX + (endX-startX)*t
		y := startY + (endY-startY)*t

		dx := endX - startX
		dy := endY - startY
		length := math.Sqrt(dx*dx + dy*dy)
		if length > 0 {
			nx := -dy / length
			ny := dx / length
			jitter := (rand.Float64()*2 - 1) * 14
			x += nx * jitter
			y += ny * jitter
		}

		ctx.Call("lineTo", x, y)
	}

	ctx.Call("lineTo", endX, endY)
	ctx.Call("stroke")
	ctx.Call("restore")
}

func drawZapperBolts() {
	if !zapperIsActive() {
		return
	}

	// Primary ball: blue-white electricity.
	drawOneZapperBolt(
		&ball,
		zapperTargetIndex,
		"#e8fbff",
		"#58d9ff",
	)

	// Second ball: slightly different violet-cyan electricity.
	if secondBallActive {
		drawOneZapperBolt(
			&secondBall,
			secondZapperTargetIndex,
			"#f3e8ff",
			"#b56cff",
		)
	}
}

func drawCenteredOverlay() {
	ctx.Call("save")
	ctx.Set("fillStyle", "rgba(0, 0, 0, 0.5)")
	ctx.Call("fillRect", 0, 0, canvasWidth, canvasHeight)
	ctx.Call("restore")
}

func draw() {
	ctx.Set("fillStyle", palette[0])
	ctx.Call("fillRect", 0, 0, canvasWidth, canvasHeight)

	if bricksDirty {
		rebuildBrickCanvas()
	}
	ctx.Call("drawImage", brickCanvas, 0, 0)
	drawZapperBolts()

	if blackHoleActive && showBlackHole {
		ctx.Set("fillStyle", "#000000")
		ctx.Set("strokeStyle", "#9d4edd")
		ctx.Set("lineWidth", 5)
		ctx.Call("beginPath")
		ctx.Call("arc", blackHoleX, blackHoleY, 28, 0, 2*math.Pi)
		ctx.Call("fill")
		ctx.Call("stroke")

		ctx.Set("strokeStyle", "#c77dff")
		ctx.Set("lineWidth", 2)
		ctx.Call("beginPath")
		ctx.Call("arc", blackHoleX, blackHoleY, 42, 0, 2*math.Pi)
		ctx.Call("stroke")
	}

	drawBall(ball.x, ball.y, ball.r, ball.angle, palette[3], palette[5])
	if secondBallActive {
		drawBall(secondBall.x, secondBall.y, secondBall.r, secondBall.angle, palette[7], palette[8])
	}

	ctx.Set("fillStyle", palette[1])
	ctx.Call("beginPath")
	ctx.Call("roundRect", paddle.x, paddle.y, paddle.w, paddle.h, paddleRadius)
	ctx.Call("fill")

	ctx.Set("fillStyle", palette[4])
	ctx.Set(
		"font",
		"18px GameFont, monospace",
	)
	updateHUDCache()
	ctx.Call("fillText", hudLivesText, 10, 30)
	ctx.Call("fillText", hudLevelText, 10, 60)
	ctx.Call("fillText", hudScoreText, 10, 90)

	for i, message := range statusMessages {
		y := 120.0 + float64(i)*30.0
		ctx.Call("fillText", message.text, 10, y)
	}

	if waitingForStart && !gameOver {
		drawCenteredOverlay()
		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")
		ctx.Set("font", "72px GameFont, monospace")
		ctx.Call("fillText", "READY", canvasWidth/2, canvasHeight/2)
		ctx.Set("font", "28px GameFont, monospace")
		//ctx.Call("fillText", "Press Space, Enter, click, or touch", canvasWidth/2, canvasHeight/2+70)
		ctx.Set("textAlign", "start")
	}

	if levelAdvancePending && !gameOver {
		drawCenteredOverlay()
		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")
		ctx.Set("font", "64px GameFont, monospace")
		ctx.Call("fillText", "LEVEL COMPLETE", canvasWidth/2, canvasHeight/2)
		ctx.Set("textAlign", "start")
	}

	if paused && !gameOver && !waitingForStart && !levelAdvancePending {
		drawCenteredOverlay()
		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")
		ctx.Set("font", "72px GameFont, monospace")
		ctx.Call("fillText", "PAUSED", canvasWidth/2, canvasHeight/2)
		ctx.Set("font", "28px GameFont, monospace")
		//ctx.Call("fillText", "Press Space, Escape, click, or touch", canvasWidth/2, canvasHeight/2+70)
		ctx.Set("textAlign", "start")
	}

	if gameOver {
		drawCenteredOverlay()
		ctx.Set("fillStyle", palette[4])
		ctx.Set("textAlign", "center")

		if win {
			// Deliberately the largest title used anywhere in the game.
			ctx.Set("font", "144px GameFont, monospace")
			ctx.Call("fillText", "YOU WIN!", canvasWidth/2, canvasHeight/2-10)
			ctx.Set("font", "28px GameFont, monospace")
			ctx.Call(
				"fillText",
				"Press Space or touch screen to restart",
				canvasWidth/2,
				canvasHeight/2+75,
			)
		} else {
			ctx.Set("font", "72px GameFont, monospace")
			ctx.Call("fillText", "GAME OVER", canvasWidth/2, canvasHeight/2)
			ctx.Set("font", "24px GameFont, monospace")
			ctx.Call(
				"fillText",
				"Press Space, Enter, or left-click to retry",
				canvasWidth/2,
				canvasHeight/2+60,
			)
		}

		ctx.Set("textAlign", "start")
	}
}

func drawBall(x, y, radius, angle float64, fillColor, strokeColor string) {
	ctx.Call("save")
	ctx.Call("translate", x, y)
	ctx.Call("rotate", angle)
	ctx.Set("fillStyle", fillColor)
	ctx.Call("beginPath")
	ctx.Call("arc", 0, 0, radius, 0, 2*math.Pi)
	ctx.Call("fill")
	ctx.Set("strokeStyle", strokeColor)
	ctx.Set("lineWidth", 2)
	ctx.Call("beginPath")
	ctx.Call("moveTo", 0, -radius)
	ctx.Call("lineTo", 0, radius)
	ctx.Call("stroke")
	ctx.Set("fillStyle", strokeColor)
	ctx.Call("beginPath")
	ctx.Call("arc", 0, 0, 2, 0, 2*math.Pi)
	ctx.Call("fill")
	ctx.Call("restore")
}

// ---- Game loop ----
func gameLoop(this js.Value, args []js.Value) interface{} {
	defer func() {
		if r := recover(); r != nil {
			js.Global().Get("console").Call("error", "Panic in gameLoop:", r)
		}
	}()
	now := js.Global().Get("performance").Call("now").Float()
	if lastTime == 0 {
		lastTime = now
	}
	dt := (now - lastTime) / 1000.0
	if dt > 0.05 {
		dt = 0.05
	}
	lastTime = now
	update(dt)
	draw()
	js.Global().Call("requestAnimationFrame", loopFunc)
	return nil
}

func clampFloat(x, min, max float64) float64 {
	if x < min {
		return min
	}
	if x > max {
		return max
	}
	return x
}

func verticalDragToHorizontalDelta(deltaY float64) float64 {
	// Finger down moves paddle right; finger up moves paddle left.
	rect := canvas.Call("getBoundingClientRect")
	height := rect.Get("height").Float()
	if height <= 0 {
		return 0
	}

	scaleY := canvasHeight / height
	return deltaY * scaleY * 2.2
}

func goToNextLevel() {
	if currentLevelIndex < len(levels)-1 {
		jumpToLevel(currentLevelIndex + 1)
	}
}

func goToPreviousLevel() {
	if currentLevelIndex > 0 {
		jumpToLevel(currentLevelIndex - 1)
	}
}

func setupLevelNavigationBridge() {
	nextLevelCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		goToNextLevel()
		return nil
	})

	previousLevelCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		goToPreviousLevel()
		return nil
	})

	js.Global().Set("breakoutNextLevel", nextLevelCallback)
	js.Global().Set("breakoutPreviousLevel", previousLevelCallback)
}

// ---- Input ----
func movePaddleToPointer(e js.Value) {
	rect := canvas.Call("getBoundingClientRect")
	displayWidth := rect.Get("width").Float()
	if displayWidth <= 0 {
		return
	}

	scaleX := canvasWidth / displayWidth
	pointerX := (e.Get("clientX").Float() - rect.Get("left").Float()) * scaleX
	x := pointerX - paddle.w/2

	if x < 0 {
		x = 0
	}
	if x > canvasWidth-paddle.w {
		x = canvasWidth - paddle.w
	}

	paddle.vx = (x - paddle.x) * 12
	paddle.x = x
}

func bindMobileButton(id string, handler func()) {
	button := doc.Call("getElementById", id)
	if button.IsUndefined() || button.IsNull() {
		return
	}

	fn := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			args[0].Call("preventDefault")
		}
		if !audioInitialized {
			initAudio()
		} else {
			ensureAudioRunning()
		}
		handler()
		return nil
	})

	js.Global().Set("__breakout_"+id, fn)
	button.Call("addEventListener", "pointerdown", fn)
}

func setupInput() {
	keyDown = js.FuncOf(func(this js.Value, args []js.Value) (ret interface{}) {
		defer func() {
			if r := recover(); r != nil {
				js.Global().Get("console").Call("error", "Panic in keydown handler:", fmt.Sprint(r))
				ret = nil
			}
		}()

		if len(args) == 0 {
			return nil
		}
		e := args[0]
		e.Call("preventDefault")
		key := e.Get("key").String()

		// Cheat keys must work even while a newly selected level is waiting
		// for its first launch input.
		if key == "n" || key == "N" {
			goToNextLevel()
			return nil
		}
		if key == "p" || key == "P" {
			goToPreviousLevel()
			return nil
		}

		// Restart from level one after winning.
		if gameOver && win {
			if key == " " && !e.Get("repeat").Bool() {
				jumpToLevel(0)
			}
			return nil
		}

		// Retry the same level after losing all lives. This must come
		// before pause handling so Space retries instead of toggling pause.
		if gameOver && !win {
			if (key == " " || key == "Enter") && !e.Get("repeat").Bool() {
				retryCurrentLevel()
			}
			return nil
		}

		if waitingForStart {
			if (key == " " || key == "Enter" ||
				key == "ArrowLeft" || key == "ArrowRight") &&
				!e.Get("repeat").Bool() {
				waitingForStart = false
				leftPressed = false
				rightPressed = false
				paddle.vx = 0
				showStatus("Go!", 1.0)
			}
			return nil
		}

		if levelAdvancePending {
			return nil
		}

		// Browser audio must be unlocked from a user gesture.
		if !audioInitialized {
			initAudio()
		} else {
			ensureAudioRunning()
		}

		// Sound toggle.
		if (key == "s" || key == "S") && !e.Get("repeat").Bool() {
			toggleSound()
			return nil
		}

		// Toggle magnetism relative to the level default.
		if (key == "m" || key == "M") && !e.Get("repeat").Bool() {
			magnetCheat = !magnetCheat
			if magnetIsActive() {
				showStatus("Magnets on", 2.0)
			} else {
				showStatus("Magnets off", 2.0)
			}
			return nil
		}

		// Toggle Zapper relative to the level default.
		if (key == "z" || key == "Z") && !e.Get("repeat").Bool() {
			zapperCheat = !zapperCheat
			zapperTargetIndex = -1
			zapperHitTimer = 0
			if zapperIsActive() {
				showStatus("Zapper on", 2.0)
			} else {
				showStatus("Zapper off", 2.0)
			}
			return nil
		}

		if key == "2" && !e.Get("repeat").Bool() {
			if !secondBallActive {
				secondBallActive = true
				secondBall = ball
				secondBall.vx = -ball.vx
				secondBall.omega = -ball.omega
				secondBall.x += secondBall.r * 2
				if secondBall.x+secondBall.r > canvasWidth {
					secondBall.x = ball.x - secondBall.r*2
				}
				secondBall.stuckTimer = 0
			}
			return nil
		}

		// Pause/resume. Ignore key-repeat so holding the key toggles only once.
		if (key == " " || key == "Escape") && !e.Get("repeat").Bool() {
			paused = !paused
			leftPressed = false
			rightPressed = false
			paddle.vx = 0
			return nil
		}

		// Paddle controls
		if key == "ArrowLeft" {
			leftPressed = true
		} else if key == "ArrowRight" {
			rightPressed = true
		}
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keydown", keyDown)

	keyUp = js.FuncOf(func(this js.Value, args []js.Value) (ret interface{}) {
		defer func() {
			if r := recover(); r != nil {
				js.Global().Get("console").Call("error", "Panic in keyup handler:", fmt.Sprint(r))
				ret = nil
			}
		}()

		if len(args) == 0 {
			return nil
		}
		e := args[0]
		e.Call("preventDefault")
		key := e.Get("key").String()
		if key == "ArrowLeft" {
			leftPressed = false
		} else if key == "ArrowRight" {
			rightPressed = false
		}
		return nil
	})
	js.Global().Get("document").Call("addEventListener", "keyup", keyUp)

	orientationChange := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		recalibratePhoneTilt()
		return nil
	})
	js.Global().Set("__breakoutOrientationChange", orientationChange)
	js.Global().Call("addEventListener", "orientationchange", orientationChange)

	// Pointer controls work with mouse, touch, and pen.
	pointerDown = js.FuncOf(func(this js.Value, args []js.Value) (ret interface{}) {
		defer func() {
			if r := recover(); r != nil {
				js.Global().Get("console").Call("error", "Panic in pointerdown handler:", fmt.Sprint(r))
				ret = nil
			}
		}()

		if len(args) == 0 {
			return nil
		}

		e := args[0]
		e.Call("preventDefault")

		if !audioInitialized {
			initAudio()
		} else {
			ensureAudioRunning()
		}

		if gameOver && win {
			jumpToLevel(0)
			return nil
		}

		pointerType := e.Get("pointerType").String()

		if waitingForStart {
			waitingForStart = false
			leftPressed = false
			rightPressed = false
			paddle.vx = 0
			showStatus("Go!", 1.0)
			return nil
		}

		if paused {
			paused = false
			leftPressed = false
			rightPressed = false
			touchControlActive = false
			paddle.vx = 0
			showStatus("Go!", 1.0)
			return nil
		}

		if levelAdvancePending {
			return nil
		}

		if pointerType == "touch" || pointerType == "pen" {
			pointerID := e.Get("pointerId").Int()

			if gameOver && !win {
				retryCurrentLevel()
			}

			if mobileControlsEnabled {
				switch mobileControlMode {
				case "vertical":
					if !touchControlActive {
						touchControlActive = true
						touchPointerID = pointerID
						touchLastY = e.Get("clientY").Float()
						paddle.vx = 0
						canvas.Call("setPointerCapture", e.Get("pointerId"))
					}
				case "follow":
					if !touchControlActive {
						touchControlActive = true
						touchPointerID = pointerID
						canvas.Call("setPointerCapture", e.Get("pointerId"))
					}
					movePaddleToPointer(e)
				case "two-thumb":
					rect := canvas.Call("getBoundingClientRect")
					midX := rect.Get("left").Float() + rect.Get("width").Float()/2
					if e.Get("clientX").Float() < midX {
						if mobileLeftPointerID < 0 {
							mobileLeftPointerID = pointerID
							mobileLeftHeld = true
						}
					} else if mobileRightPointerID < 0 {
						mobileRightPointerID = pointerID
						mobileRightHeld = true
					}
					touchControlActive = mobileLeftHeld || mobileRightHeld
					canvas.Call("setPointerCapture", e.Get("pointerId"))
				case "tilt":
					requestPhoneTiltPermission()
				}
			}
			return nil
		}

		if pointerType == "mouse" {
			leftPressed = false
			rightPressed = false
			movePaddleToPointer(e)

			if gameOver && !win {
				retryCurrentLevel()
			}
		}

		return nil
	})
	canvas.Call("addEventListener", "pointerdown", pointerDown)

	pointerMove = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if paused || len(args) == 0 {
			return nil
		}

		e := args[0]
		pointerType := e.Get("pointerType").String()
		pointerID := e.Get("pointerId").Int()

		if (pointerType == "touch" || pointerType == "pen") &&
			mobileControlsEnabled {
			switch mobileControlMode {
			case "vertical":
				if touchControlActive && pointerID == touchPointerID {
					y := e.Get("clientY").Float()
					deltaY := y - touchLastY
					touchLastY = y

					deltaX := verticalDragToHorizontalDelta(deltaY)
					paddle.x += deltaX
					paddle.vx = deltaX * 60

					if paddle.x < 0 {
						paddle.x = 0
					}
					if paddle.x+paddle.w > canvasWidth {
						paddle.x = canvasWidth - paddle.w
					}
				}
			case "follow":
				if touchControlActive && pointerID == touchPointerID {
					movePaddleToPointer(e)
				}
			}
			e.Call("preventDefault")
			return nil
		}

		if pointerType == "mouse" {
			e.Call("preventDefault")
			leftPressed = false
			rightPressed = false
			movePaddleToPointer(e)
		}

		return nil
	})
	canvas.Call("addEventListener", "pointermove", pointerMove)

	pointerUp = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			e := args[0]
			pointerID := e.Get("pointerId").Int()

			if mobileControlMode == "two-thumb" {
				if pointerID == mobileLeftPointerID {
					mobileLeftPointerID = -1
					mobileLeftHeld = false
				}
				if pointerID == mobileRightPointerID {
					mobileRightPointerID = -1
					mobileRightHeld = false
				}
				touchControlActive = mobileLeftHeld || mobileRightHeld
				if !touchControlActive {
					paddle.vx = 0
				}
			} else if touchControlActive && pointerID == touchPointerID {
				touchControlActive = false
				touchPointerID = -1
				paddle.vx = 0
			}

			pointerValue := e.Get("pointerId")
			if !pointerValue.IsUndefined() &&
				canvas.Call("hasPointerCapture", pointerValue).Bool() {
				canvas.Call("releasePointerCapture", pointerValue)
			}
		}
		return nil
	})
	canvas.Call("addEventListener", "pointerup", pointerUp)

	pointerCancel = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) > 0 {
			pointerID := args[0].Get("pointerId").Int()
			if mobileControlMode == "two-thumb" {
				if pointerID == mobileLeftPointerID {
					mobileLeftPointerID = -1
					mobileLeftHeld = false
				}
				if pointerID == mobileRightPointerID {
					mobileRightPointerID = -1
					mobileRightHeld = false
				}
				touchControlActive = mobileLeftHeld || mobileRightHeld
			} else if touchControlActive && pointerID == touchPointerID {
				touchControlActive = false
				touchPointerID = -1
			}
			if !touchControlActive {
				paddle.vx = 0
			}
		}
		return nil
	})
	canvas.Call("addEventListener", "pointercancel", pointerCancel)

	mouseLeave = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		paddle.vx = 0
		return nil
	})
	canvas.Call("addEventListener", "mouseleave", mouseLeave)

	bindMobileButton("pauseButton", func() {
		if gameOver {
			return
		}
		paused = !paused
		leftPressed = false
		rightPressed = false
		paddle.vx = 0
	})

	bindMobileButton("soundButton", func() {
		toggleSound()
	})

	bindMobileButton("previousLevelButton", func() {
		goToPreviousLevel()
	})

	bindMobileButton("nextLevelButton", func() {
		goToNextLevel()
	})

	bindMobileButton("magnetsButton", func() {
		magnetCheat = !magnetCheat
		if magnetIsActive() {
			showStatus("Magnets on", 2.0)
		} else {
			showStatus("Magnets off", 2.0)
		}
	})

	bindMobileButton("zapperButton", func() {
		zapperCheat = !zapperCheat
		zapperTargetIndex = -1
		zapperHitTimer = 0
		if zapperIsActive() {
			showStatus("Zapper on", 2.0)
		} else {
			showStatus("Zapper off", 2.0)
		}
	})
}

// ---- main ----
func main() {
	defer func() {
		if r := recover(); r != nil {
			js.Global().Get("console").Call("error", "Panic in main:", r)
		}
	}()

	log("main: starting")
	canvas = doc.Call("getElementById", "gameCanvas")
	if canvas.IsNull() {
		log("canvas not found!")
		return
	}
	ctx = canvas.Call("getContext", "2d")
	if ctx.IsNull() {
		log("context not found!")
		return
	}
	log("canvas and context obtained")
	hudLivesValue = -1
	hudLevelValue = -1
	hudScoreValue = -1
	ensureBrickCanvas()

	loadLevels()
	setupInput()
	setupMobileControlSelector()
	setupLevelNavigationBridge()
	resetGame()

	loopFunc = js.FuncOf(gameLoop)
	js.Global().Call("requestAnimationFrame", loopFunc)
	log("main: requestAnimationFrame called")

	select {}
}
