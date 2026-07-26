//go:build js && wasm

package main

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"syscall/js"
)

// ---- Default values (constants) ----
const (
	buildID = "20260726-c16d4a0e6b"

	defaultCanvasWidth        = 1800.0
	defaultCanvasHeight       = 900.0
	defaultGravity            = 300.0
	defaultRestitution        = 0.85
	defaultFrictionCoeff      = 0.1 //0.1
	defaultPaddleWidth        = 220.0
	defaultPaddleHeight       = 30.0
	defaultBallRadius         = 8.0
	defaultBrickRows          = 10
	defaultBrickCols          = 18
	defaultBrickWidth         = 60.0
	defaultBrickHeight        = 20.0
	defaultBrickPadding       = 20.0
	defaultBrickOffsetTop     = -1.0
	defaultPaddleBoost        = 700.0
	defaultBrickBoost         = 100.0
	defaultMaxSpeed           = 1200.0 // was 1000 for original physics
	defaultMaxSpin            = 530.0
	defaultUseImprovedPhysics = true

	// Keyboard and two-thumb controls accelerate from a precise low speed
	// to a faster cross-screen speed, then brake quickly when released.
	defaultDigitalPaddleMaxSpeed     = 2800.0
	defaultDigitalPaddleAcceleration = 9000.0
	defaultDigitalPaddleBraking      = 40000.0
	defaultPaddleRadius              = 12.0
	defaultBrickRadius               = 6.0
	defaultUnbreakableChance         = 0.15
	defaultMagicChance               = 0.3
	defaultPowerUpDuration           = 10.0
	defaultBlackHoleStrength         = 800.0
	defaultBlackHoleRange            = 200.0
	defaultMagnetStrength            = 600.0
	defaultMagnetRange               = 300.0
	defaultInfluencerMultiplier      = 5.0
	defaultLives                     = 7
	defaultStuckSpeedThreshold       = 85.0
	defaultStuckDuration             = 10.0
	defaultTiltUpSpeed               = 520.0
	defaultTiltSideMin               = 180.0
	defaultTiltSideMax               = 340.0
	defaultZapperHitTime             = 0.1
	defaultZapperRange               = 320.0

	// Improved-mode tuning. Original physics does not read these values.
	improvedMagnusCoefficient       = 0.015 // 0.0015
	improvedMagnusAccelerationScale = 0.30
	improvedSpinDrag                = 0.10
	improvedAirDrag                 = 0.010
	improvedWallFrictionScale       = 1.00
	improvedBrickFrictionScale      = 1.40
	improvedPaddleFrictionScale     = 1.35 // 1.35
	improvedPaddleSpinTransfer      = 2.25 // 1.25
	improvedCollisionSpinCoupling   = 2.50
	improvedMinimumCollisionGrip    = 0.18
	improvedMinimumPaddleGrip       = 0.25
	improvedCollisionSlop           = 0.05

	enableHighSpinMessage   = true
	highSpinThreshold       = 100.0 // 100.0
	highSpinMessageDuration = 1.5

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
	defaultEnableBigPaddle        = true

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
	enableBigPaddle        = defaultEnableBigPaddle

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
	setPausedCallback         js.Func
	getPausedCallback         js.Func
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
	POWER_BIG_PADDLE
)

// ---- Ball struct ----
type Ball struct {
	x, y, r       float64
	vx, vy        float64
	omega, angle  float64
	stuckTimer    float64
	soundCooldown float64
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

	fpsCurrent       float64
	fpsSampleElapsed float64
	fpsSampleFrames  int

	loopFunc      js.Func
	keyDown       js.Func
	keyUp         js.Func
	pointerMove   js.Func
	pointerDown   js.Func
	pointerUp     js.Func
	pointerCancel js.Func
	mouseLeave    js.Func

	// Independent power-up states. Timed effects can coexist.
	lowGravityActive     bool
	lowGravityTimer      float64
	passActive           bool
	passTimer            float64
	reverseGravityActive bool
	reverseGravityTimer  float64
	magnetPowerActive    bool
	magnetPowerTimer     float64
	zapperPowerActive    bool
	zapperPowerTimer     float64
	bigPaddleActive      bool
	bigPaddleTimer       float64
	currentGravity       float64

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
	currentLevelIndex    int
	highestUnlockedLevel int
	unlockFrontier       int // Highest level index made available; may equal len(levels) after finishing all current levels.
	levels               []levelData

	devAllLevelsUnlocked bool
	debugOverlayVisible  bool
	useImprovedPhysics   = defaultUseImprovedPhysics // Runtime A/B toggle; press F to switch modes.

	lastPaddleSpinValid  bool
	lastPaddleSpinBall   int
	lastPaddleSpinBefore float64
	lastPaddleSpinAdded  float64
	lastPaddleSpinAfter  float64

	lastCollisionValid                   bool
	lastCollisionSurface                 string
	lastCollisionIncomingAngle           float64
	lastCollisionOutgoingAngle           float64
	lastCollisionAngleChange             float64
	lastCollisionSpinAngleEffect         float64
	lastCollisionSpinBefore              float64
	lastCollisionSpinAfter               float64
	lastCollisionSpinChange              float64
	lastCollisionTangentialImpulse       float64
	lastCollisionNoSpinImpulse           float64
	lastCollisionSpinImpulseEffect       float64
	lastCollisionContactSpinSurfaceSpeed float64

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

// preventVerticalLock is used only by improved physics. It preserves total
// speed while giving nearly vertical trajectories a small deterministic
// horizontal component.
func preventVerticalLock(b *Ball, preferredDirection float64) {
	const minimumHorizontalSpeed = 60.0

	speed := math.Hypot(b.vx, b.vy)
	if speed < minimumHorizontalSpeed || math.Abs(b.vx) >= minimumHorizontalSpeed {
		return
	}

	direction := sign(preferredDirection)
	if direction == 0 {
		direction = sign(b.omega)
	}
	if direction == 0 {
		direction = sign(b.x - canvasWidth/2)
	}
	if direction == 0 {
		direction = 1
	}

	targetVx := direction * math.Min(minimumHorizontalSpeed, speed*0.35)
	vySign := sign(b.vy)
	if vySign == 0 {
		vySign = -1
	}

	remainingVySquared := math.Max(0, speed*speed-targetVx*targetVx)
	b.vx = targetVx
	b.vy = vySign * math.Sqrt(remainingVySquared)
}

func clearLastPaddleSpinDebug() {
	lastPaddleSpinValid = false
	lastPaddleSpinBall = 0
	lastPaddleSpinBefore = 0
	lastPaddleSpinAdded = 0
	lastPaddleSpinAfter = 0
}

func recordLastPaddleSpinDebug(isPrimary bool, before, after float64) {
	lastPaddleSpinValid = true
	if isPrimary {
		lastPaddleSpinBall = 1
	} else {
		lastPaddleSpinBall = 2
	}
	lastPaddleSpinBefore = before
	lastPaddleSpinAfter = after
	lastPaddleSpinAdded = after - before
}

func velocityAngleDegrees(vx, vy float64) float64 {
	return math.Atan2(vy, vx) * 180 / math.Pi
}

func normalizeAngleDegrees(angle float64) float64 {
	for angle > 180 {
		angle -= 360
	}
	for angle <= -180 {
		angle += 360
	}
	return angle
}

func clearLastCollisionDebug() {
	lastCollisionValid = false
	lastCollisionSurface = ""
	lastCollisionIncomingAngle = 0
	lastCollisionOutgoingAngle = 0
	lastCollisionAngleChange = 0
	lastCollisionSpinAngleEffect = 0
	lastCollisionSpinBefore = 0
	lastCollisionSpinAfter = 0
	lastCollisionSpinChange = 0
	lastCollisionTangentialImpulse = 0
	lastCollisionNoSpinImpulse = 0
	lastCollisionSpinImpulseEffect = 0
	lastCollisionContactSpinSurfaceSpeed = 0
}

func recordLastCollisionDebug(
	surface string,
	incomingAngle, outgoingAngle, noSpinOutgoingAngle float64,
	spinBefore, spinAfter, tangentialImpulse, noSpinImpulse, ballRadius float64,
) {
	lastCollisionValid = true
	lastCollisionSurface = surface
	lastCollisionIncomingAngle = incomingAngle
	lastCollisionOutgoingAngle = outgoingAngle
	lastCollisionAngleChange = normalizeAngleDegrees(outgoingAngle - incomingAngle)
	lastCollisionSpinAngleEffect = normalizeAngleDegrees(outgoingAngle - noSpinOutgoingAngle)
	lastCollisionSpinBefore = spinBefore
	lastCollisionSpinAfter = spinAfter
	lastCollisionSpinChange = spinAfter - spinBefore
	lastCollisionTangentialImpulse = tangentialImpulse
	lastCollisionNoSpinImpulse = noSpinImpulse
	lastCollisionSpinImpulseEffect = tangentialImpulse - noSpinImpulse
	lastCollisionContactSpinSurfaceSpeed = -spinBefore * ballRadius
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func moveToward(current, target, maxDelta float64) float64 {
	if current < target {
		return math.Min(current+maxDelta, target)
	}
	if current > target {
		return math.Max(current-maxDelta, target)
	}
	return target
}

func showStatus(text string, duration float64) {
	statusMessages = append(statusMessages, statusMessage{text: text, timer: duration})
	if len(statusMessages) > 3 {
		statusMessages = statusMessages[len(statusMessages)-3:]
	}
}

// Keep a single High spin! entry in the existing top-left status stack.
// A new qualifying paddle hit refreshes its timer instead of adding duplicates.
func showHighSpinStatus() {
	if !enableHighSpinMessage {
		return
	}

	for i := range statusMessages {
		if statusMessages[i].text == "High spin!" {
			statusMessages[i].timer = highSpinMessageDuration
			return
		}
	}

	showStatus("High spin!", highSpinMessageDuration)
}

func maybeShowHighSpin(before, after float64) {
	// Use the real post-clamp spin change caused by this paddle hit.
	if math.Abs(after-before) >= highSpinThreshold {
		showHighSpinStatus()
	}
}

const (
	minimumCollisionSoundSpeed = 35.0
	collisionSoundCooldown     = 0.045
)

func playImpactSound(b *Ball, impactSpeed float64, play func()) {
	if impactSpeed < minimumCollisionSoundSpeed || b.soundCooldown > 0 {
		return
	}
	play()
	b.soundCooldown = collisionSoundCooldown
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
	enableBigPaddle = defaultEnableBigPaddle
	palette = append([]string(nil), defaultPalette...)
	magicColor = defaultMagicColor
	magicStrokeColor = defaultMagicStrokeColor
	unbreakableStrokeColor = defaultUnbreakableStrokeColor
	brickStrokeColor = defaultBrickStrokeColor
}

// ---- Apply config overrides from map ----
func applyConfig(config map[string]string) {
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		val := config[key]
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
		case "enableBigPaddle":
			if b, err := strconv.ParseBool(val); err == nil {
				enableBigPaddle = b
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
	cx := b.x + nx*b.r
	cy := b.y + ny*b.r
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
	maxFriction := frictionCoeff * math.Abs(deltaVn)
	var deltaVt float64
	if math.Abs(vt) < 0.001 {
		deltaVt = -vt
	} else {
		friction := math.Min(math.Abs(vt), maxFriction)
		deltaVt = -sign(vt) * friction
	}
	tx := -ny
	ty := nx
	b.vx += deltaVn*nx + deltaVt*tx
	b.vy += deltaVn*ny + deltaVt*ty
	b.omega -= 2 * deltaVt / b.r
}

// resolveCollisionBallImproved uses the actual contact point and a solid-disk
// tangential impulse. It is used only while improved physics is selected.
func resolveCollisionBallImproved(b *Ball, nx, ny, surfVx, surfVy, frictionScale float64) (float64, bool) {
	// nx,ny point from the surface toward the ball, so the contact point is
	// on the opposite side of the ball centre.
	cx := b.x - nx*b.r
	cy := b.y - ny*b.r

	contactVx := b.vx - b.omega*(cy-b.y)
	contactVy := b.vy + b.omega*(cx-b.x)
	relVx := contactVx - surfVx
	relVy := contactVy - surfVy

	vn := relVx*nx + relVy*ny
	tx := -ny
	ty := nx
	translationVt := (b.vx-surfVx)*tx + (b.vy-surfVy)*ty
	spinSurfaceSpeed := -b.omega * b.r
	vt := translationVt + spinSurfaceSpeed*improvedCollisionSpinCoupling
	if vn >= 0 {
		return 0, false
	}

	vnNew := -restitution * vn
	deltaVn := vnNew - vn

	// For a solid disk, no-slip correction is -vt/3 because tangential
	// impulse changes both translation and rotation. Different surfaces use
	// different friction scales: paddle strongest, bricks medium, walls weak.
	effectiveFriction := math.Max(
		frictionCoeff*frictionScale,
		improvedMinimumCollisionGrip,
	)
	maxFriction := effectiveFriction * math.Abs(deltaVn)
	desiredDeltaVt := -vt / 3.0
	deltaVt := clampFloat(desiredDeltaVt, -maxFriction, maxFriction)

	b.vx += deltaVn*nx + deltaVt*tx
	b.vy += deltaVn*ny + deltaVt*ty
	b.omega -= 2 * deltaVt / b.r
	b.omega = clampFloat(b.omega, -maxSpin, maxSpin)

	preventVerticalLock(b, deltaVt*tx+b.omega)
	return deltaVt, true
}

func resolveSelectedCollision(b *Ball, nx, ny, surfVx, surfVy float64, improved bool, frictionScale float64) {
	if improved {
		_, _ = resolveCollisionBallImproved(b, nx, ny, surfVx, surfVy, frictionScale)
		return
	}
	resolveCollisionBall(b, nx, ny, surfVx, surfVy)
}

// Resolve one collision while measuring how much the ball's spin changed the
// outgoing direction. Diagnostics are recorded only for Ball 1.
func resolveSelectedCollisionDebug(
	b *Ball,
	nx, ny, surfVx, surfVy float64,
	improved bool,
	frictionScale float64,
	surface string,
	record bool,
) {
	incomingAngle := velocityAngleDegrees(b.vx, b.vy)
	spinBefore := b.omega
	beforeVx, beforeVy := b.vx, b.vy

	// A zero-spin clone gives a direct A/B measurement of the angle caused by
	// spin at this exact collision, using the same incoming velocity and normal.
	noSpin := *b
	noSpin.omega = 0
	noSpinImpulse := 0.0
	if improved {
		if impulse, collided := resolveCollisionBallImproved(&noSpin, nx, ny, surfVx, surfVy, frictionScale); collided {
			noSpinImpulse = impulse
		}
	} else {
		noSpinBeforeVx, noSpinBeforeVy := noSpin.vx, noSpin.vy
		resolveCollisionBall(&noSpin, nx, ny, surfVx, surfVy)
		tx, ty := -ny, nx
		noSpinImpulse = (noSpin.vx-noSpinBeforeVx)*tx + (noSpin.vy-noSpinBeforeVy)*ty
	}

	tangentialImpulse := 0.0
	if improved {
		if impulse, collided := resolveCollisionBallImproved(b, nx, ny, surfVx, surfVy, frictionScale); collided {
			tangentialImpulse = impulse
		}
	} else {
		resolveCollisionBall(b, nx, ny, surfVx, surfVy)
		tx, ty := -ny, nx
		tangentialImpulse = (b.vx-beforeVx)*tx + (b.vy-beforeVy)*ty
	}

	if record {
		recordLastCollisionDebug(
			surface,
			incomingAngle,
			velocityAngleDegrees(b.vx, b.vy),
			velocityAngleDegrees(noSpin.vx, noSpin.vy),
			spinBefore,
			b.omega,
			tangentialImpulse,
			noSpinImpulse,
			b.r,
		)
	}
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
			// X is a randomized block using the level's existing chances:
			// unbreakable first, then magic, otherwise a normal brick.
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

// Resize the paddle while preserving its center and keeping it on-screen.
func setPaddleSize(width, height float64) {
	center := paddle.x + paddle.w/2
	paddle.w = width
	paddle.h = height
	paddle.x = center - paddle.w/2
	paddle.x = math.Max(0, math.Min(paddle.x, canvasWidth-paddle.w))
	paddlePreviousX = paddle.x
}

// Gravity effects coexist. Black hole overrides both; otherwise reverse
// gravity has priority over low gravity.
func refreshCurrentGravity() {
	if blackHoleActive {
		currentGravity = 0
	} else if reverseGravityActive {
		currentGravity = -baseGravity
	} else if lowGravityActive {
		currentGravity = baseGravity / 3
	} else {
		currentGravity = baseGravity
	}
}

func clearTimedPowerUps() {
	lowGravityActive = false
	lowGravityTimer = 0
	passActive = false
	passTimer = 0
	reverseGravityActive = false
	reverseGravityTimer = 0
	magnetPowerActive = false
	magnetPowerTimer = 0
	zapperPowerActive = false
	zapperPowerTimer = 0
	bigPaddleActive = false
	bigPaddleTimer = 0
	blackHoleActive = false
	blackHoleTimer = 0
	influencerActive = false
	influencerTimer = 0
	secondBallActive = false
	setPaddleSize(paddleWidth, paddleHeight)
	refreshCurrentGravity()
}

// Activate powerup
func activatePowerUpWithBrick(hitBrick *brick) {

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
	if enableBigPaddle {
		available = append(available, POWER_BIG_PADDLE)
	}

	if len(available) == 0 {
		return
	}

	p := available[rand.Intn(len(available))]

	switch p {
	case POWER_LOW_GRAVITY:
		lowGravityActive = true
		lowGravityTimer = powerUpDuration
		refreshCurrentGravity()
		showStatus("Low Gravity!", powerUpDuration)
		playPowerup()
	case POWER_PASS:
		passActive = true
		passTimer = powerUpDuration
		showStatus("Pass Through!", powerUpDuration)
		playPowerup()
	case POWER_NUKE:
		nukeBricks(hitBrick)
		showStatus("Nuke!", 2.0)
		playPowerup()
	case POWER_REVERSE_GRAVITY:
		reverseGravityActive = true
		reverseGravityTimer = powerUpDuration
		refreshCurrentGravity()
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
		refreshCurrentGravity()
	case POWER_MAGNET:
		magnetPowerActive = true
		magnetPowerTimer = powerUpDuration
		showStatus("Magnets!", powerUpDuration)
		playPowerup()
	case POWER_INFLUENCER:
		influencerActive = true
		influencerTimer = powerUpDuration
		showStatus("Influencer!", powerUpDuration)
		playPowerup()
	case POWER_ZAPPER:
		zapperPowerActive = true
		zapperPowerTimer = powerUpDuration
		zapperTargetIndex = -1
		zapperHitTimer = 0
		showStatus("Zapper!", powerUpDuration)
		playPowerup()
	case POWER_BREAK_UNBREAKABLE:
		if breakRandomUnbreakable() {
			showStatus("Unbreakable destroyed!", 2.0)
			playPowerup()
		}
	case POWER_BIG_PADDLE:
		bigPaddleActive = true
		bigPaddleTimer = powerUpDuration
		setPaddleSize(paddleWidth*2, paddleHeight)
		showStatus("Big Paddle!", powerUpDuration)
		playPowerup()
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

// ---- Remember progression in browser storage ----
const (
	savedLevelKey           = "breakout.currentLevel"
	highestUnlockedLevelKey = "breakout.highestUnlockedLevel" // Legacy key.
	unlockFrontierKey       = "breakout.unlockFrontier"
)

func readStoredInt(key string) (int, bool) {
	value := 0
	found := false
	func() {
		defer func() { _ = recover() }()
		storage := js.Global().Get("localStorage")
		if storage.IsUndefined() || storage.IsNull() {
			return
		}
		stored := storage.Call("getItem", key)
		if stored.IsUndefined() || stored.IsNull() {
			return
		}
		parsed, err := strconv.Atoi(stored.String())
		if err != nil {
			return
		}
		value = parsed
		found = true
	}()
	return value, found
}

func writeStoredInt(key string, value int) {
	defer func() {
		if r := recover(); r != nil {
			js.Global().Get("console").Call("warn", "Could not save progression:", fmt.Sprint(r))
		}
	}()
	storage := js.Global().Get("localStorage")
	if storage.IsUndefined() || storage.IsNull() {
		return
	}
	storage.Call("setItem", key, strconv.Itoa(value))
}

func saveCurrentLevel() {
	writeStoredInt(savedLevelKey, currentLevelIndex)
}

func permanentUnlockedLimit() int {
	if len(levels) == 0 {
		return 0
	}
	limit := unlockFrontier
	if limit < 0 {
		limit = 0
	}
	if limit >= len(levels) {
		limit = len(levels) - 1
	}
	return limit
}

func activeUnlockedLimit() int {
	if len(levels) == 0 {
		return 0
	}
	if devAllLevelsUnlocked {
		return len(levels) - 1
	}
	return permanentUnlockedLimit()
}

func refreshHighestUnlockedLevel() {
	highestUnlockedLevel = permanentUnlockedLimit()
}

func saveUnlockFrontier() {
	writeStoredInt(unlockFrontierKey, unlockFrontier)
	// Keep the old key updated so older builds still see sensible progress.
	writeStoredInt(highestUnlockedLevelKey, permanentUnlockedLimit())
}

func loadUnlockFrontier() int {
	if len(levels) == 0 {
		return 0
	}

	if frontier, ok := readStoredInt(unlockFrontierKey); ok {
		if frontier < 0 {
			return 0
		}
		return frontier
	}

	legacyHighest, legacyFound := readStoredInt(highestUnlockedLevelKey)
	if !legacyFound {
		return 0
	}
	if legacyHighest < 0 {
		legacyHighest = 0
	}

	frontier := legacyHighest

	// Legacy builds could not store one-past-the-final-level. When more levels
	// were later added, both stored values still pointed at the old final level,
	// forcing it to be replayed. Treat that exact legacy state as completed.
	if saved, ok := readStoredInt(savedLevelKey); ok &&
		saved == legacyHighest && legacyHighest < len(levels)-1 {
		frontier = legacyHighest + 1
		log("Migrated legacy final-level progress to level " + strconv.Itoa(frontier+1))
	}

	unlockFrontier = frontier
	saveUnlockFrontier()
	return frontier
}

func unlockNextLevel() {
	// Visiting a locked level through the temporary developer toggle must not
	// permanently skip normal progression.
	if devAllLevelsUnlocked && currentLevelIndex > permanentUnlockedLimit() {
		return
	}

	next := currentLevelIndex + 1
	if next > unlockFrontier {
		unlockFrontier = next
		refreshHighestUnlockedLevel()
		saveUnlockFrontier()
		if next < len(levels) {
			showStatus("Level "+strconv.Itoa(next+1)+" unlocked!", 3.0)
		}
	}
}

func loadSavedLevel() int {
	if len(levels) == 0 {
		return 0
	}

	savedIndex, ok := readStoredInt(savedLevelKey)
	if !ok {
		savedIndex = 0
	}
	if savedIndex < 0 {
		savedIndex = 0
	}
	if savedIndex >= len(levels) {
		savedIndex = len(levels) - 1
	}

	// Resume the last level actually played. Progression is stored separately
	// in unlockFrontier; clamp only if the saved level is no longer available
	// under normal progression (for example, it was visited with the U toggle).
	if savedIndex > permanentUnlockedLimit() {
		savedIndex = permanentUnlockedLimit()
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
	ball.soundCooldown = 0
	ball.r = ballRadius
	secondBallActive = false
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.y = canvasHeight - 40
	paddle.vx = 0
	paddlePreviousX = paddle.x
	clearLastPaddleSpinDebug()
	clearLastCollisionDebug()

	lowGravityActive = false
	lowGravityTimer = 0
	passActive = false
	passTimer = 0
	reverseGravityActive = false
	reverseGravityTimer = 0
	magnetPowerActive = false
	magnetPowerTimer = 0
	zapperPowerActive = false
	zapperPowerTimer = 0
	bigPaddleActive = false
	bigPaddleTimer = 0
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
	log("Level " + strconv.Itoa(index+1) + " started")
}

// ---- Jump to a specific level (cheat) ----
func jumpToLevel(index int) {
	if index < 0 {
		index = 0
	}
	if index >= len(levels) {
		index = len(levels) - 1
	}
	if index > activeUnlockedLimit() {
		showStatus("Level locked", 2.0)
		return
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
	unlockFrontier = loadUnlockFrontier()
	refreshHighestUnlockedLevel()
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
	return magnetPowerActive || (levelMagnetActive != magnetCheat)
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
		zapperPowerActive ||
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

// One improved-mode brick contact. We still choose a single classic axis
// normal, but group simultaneous overlaps so adjacent bricks do not bounce the
// ball multiple times in one substep.
type improvedBrickContact struct {
	index       int
	nx, ny      float64
	penetration float64
	impact      float64
}

func findImprovedBrickContacts(b *Ball) []improvedBrickContact {
	contacts := make([]improvedBrickContact, 0, 4)
	for _, i := range candidateBrickIndices(b) {
		br := &bricks[i]
		if !br.alive ||
			b.x+b.r <= br.x || b.x-b.r >= br.x+br.w ||
			b.y+b.r <= br.y || b.y-b.r >= br.y+br.h {
			continue
		}

		leftPenetration := b.x + b.r - br.x
		rightPenetration := br.x + br.w - (b.x - b.r)
		topPenetration := b.y + b.r - br.y
		bottomPenetration := br.y + br.h - (b.y - b.r)

		nx, ny := -1.0, 0.0
		penetration := leftPenetration
		if rightPenetration < penetration {
			nx, ny = 1, 0
			penetration = rightPenetration
		}
		if topPenetration < penetration {
			nx, ny = 0, -1
			penetration = topPenetration
		}
		if bottomPenetration < penetration {
			nx, ny = 0, 1
			penetration = bottomPenetration
		}

		contacts = append(contacts, improvedBrickContact{
			index:       i,
			nx:          nx,
			ny:          ny,
			penetration: math.Max(0, penetration),
			impact:      math.Max(0, -(b.vx*nx + b.vy*ny)),
		})
	}
	return contacts
}

func handleImprovedBrickCollisions(b *Ball, isPrimary bool) {
	contacts := findImprovedBrickContacts(b)
	if len(contacts) == 0 {
		return
	}

	if passActive {
		for _, contact := range contacts {
			br := &bricks[contact.index]
			if destroyBrick(br) {
				playBrickBreak()
				if influencerActive {
					destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier)
				}
			}
		}
		return
	}

	// Prefer the contact the ball is moving into most strongly. Penetration is
	// a stable tie-breaker. This produces one axis response for the whole group.
	best := contacts[0]
	for _, contact := range contacts[1:] {
		if contact.impact > best.impact+0.001 ||
			(math.Abs(contact.impact-best.impact) <= 0.001 && contact.penetration > best.penetration) {
			best = contact
		}
	}

	b.x += best.nx * (best.penetration + improvedCollisionSlop)
	b.y += best.ny * (best.penetration + improvedCollisionSlop)
	resolveSelectedCollisionDebug(b, best.nx, best.ny, 0, 0, true, improvedBrickFrictionScale, "BRICK", isPrimary)

	hitUnbreakable := false
	destroyedNormal := false
	for _, contact := range contacts {
		br := &bricks[contact.index]
		if !br.alive {
			continue
		}
		if br.unbreakable {
			hitUnbreakable = true
			continue
		}
		if br.magic {
			if isPrimary {
				activatePowerUpWithBrick(br)
			}
			if destroyBrick(br) {
				playMagic()
			}
			if influencerActive {
				destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier)
			}
			continue
		}
		if destroyBrick(br) {
			destroyedNormal = true
			playBrickBreak()
		}
		if influencerActive {
			destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier)
		}
	}

	if hitUnbreakable {
		playImpactSound(b, best.impact, playUnbreakable)
	}
	if destroyedNormal && brickBoost != 0 {
		b.vy -= brickBoost
		if math.Abs(b.vy) > maxSpeed {
			b.vy = math.Copysign(maxSpeed, b.vy)
		}
		if math.Abs(b.vx) > maxSpeed {
			b.vx = math.Copysign(maxSpeed, b.vx)
		}
	}
}

// ---- Update a single ball ----
func updateBallStep(b *Ball, dt float64, isPrimary bool, improved bool) {
	if b.soundCooldown > 0 {
		b.soundCooldown = math.Max(0, b.soundCooldown-dt)
	}

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

	if improved {
		// Magnus effect: spin bends the flight path perpendicular to velocity.
		// Cap it so high-spin custom levels cannot overwhelm ordinary play.
		magnusAx := -b.vy * b.omega * improvedMagnusCoefficient
		magnusAy := b.vx * b.omega * improvedMagnusCoefficient
		magnusMagnitude := math.Hypot(magnusAx, magnusAy)
		maxMagnusAcceleration := math.Max(100.0, maxSpeed*improvedMagnusAccelerationScale)
		if magnusMagnitude > maxMagnusAcceleration {
			scale := maxMagnusAcceleration / magnusMagnitude
			magnusAx *= scale
			magnusAy *= scale
		}
		b.vx += magnusAx * dt
		b.vy += magnusAy * dt

		// Mild air resistance and angular damping keep energy and curve from
		// accumulating forever after one strong paddle strike.
		airDamping := math.Exp(-improvedAirDrag * dt)
		spinDamping := math.Exp(-improvedSpinDrag * dt)
		b.vx *= airDamping
		b.vy *= airDamping
		b.omega *= spinDamping
	}

	b.x += b.vx * dt
	b.y += b.vy * dt
	b.angle += b.omega * dt

	if improved && math.Abs(b.vx) >= 60 {
		b.stuckTimer = 0
	}

	// Walls
	if b.x-b.r < 0 {
		impactSpeed := math.Max(0, -b.vx)
		b.x = b.r
		if improved {
			b.x += improvedCollisionSlop
		}
		resolveSelectedCollisionDebug(b, 1, 0, 0, 0, improved, improvedWallFrictionScale, "WALL LEFT", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.x+b.r > canvasWidth {
		impactSpeed := math.Max(0, b.vx)
		b.x = canvasWidth - b.r
		if improved {
			b.x -= improvedCollisionSlop
		}
		resolveSelectedCollisionDebug(b, -1, 0, 0, 0, improved, improvedWallFrictionScale, "WALL RIGHT", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.y-b.r < 0 {
		impactSpeed := math.Max(0, -b.vy)
		b.y = b.r
		if improved {
			b.y += improvedCollisionSlop
		}
		resolveSelectedCollisionDebug(b, 0, 1, 0, 0, improved, improvedWallFrictionScale, "WALL TOP", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.y+b.r > canvasHeight {
		// Ball lost handled by caller.
		return
	}

	// Paddle
	pLeft, pRight := paddle.x, paddle.x+paddle.w
	pTop, pBottom := paddle.y, paddle.y+paddle.h
	if b.vy > 0 &&
		b.x+b.r > pLeft && b.x-b.r < pRight &&
		b.y+b.r > pTop && b.y+b.r < pBottom {
		b.y = pTop - b.r
		if improved {
			b.y -= improvedCollisionSlop
		}

		spinBeforePaddle := b.omega
		incomingPaddleAngle := velocityAngleDegrees(b.vx, b.vy)
		hitPos := (b.x - pLeft) / paddle.w
		if hitPos < 0 {
			hitPos = 0
		}
		if hitPos > 1 {
			hitPos = 1
		}

		angle := (hitPos - 0.5) * 2.0 * (80.0 * math.Pi / 180.0)
		speed := math.Hypot(b.vx, b.vy)
		if speed < 100 {
			speed = 100
		}

		if improved {
			// Retain incoming energy according to restitution, then add the
			// paddle's configured boost.
			speed = speed*restitution + paddleBoost
			if speed > maxSpeed {
				speed = maxSpeed
			}

			incomingVx := b.vx
			incomingVy := b.vy
			incomingAngle := incomingPaddleAngle
			b.vx = speed*math.Sin(angle) + incomingVx*0.15
			b.vy = -speed * math.Cos(angle)

			// Paddle motion drags the ball's bottom contact point. A minimum grip
			// keeps paddle-controlled spin useful even when a level sets ordinary
			// surface friction very low. The direct paddle term makes spin clearly
			// proportional to the player's actual paddle velocity at impact.
			relativeSlip := b.vx - b.omega*b.r - paddle.vx
			normalDeltaSpeed := math.Abs(b.vy - incomingVy)
			effectivePaddleFriction := math.Max(
				frictionCoeff*improvedPaddleFrictionScale,
				improvedMinimumPaddleGrip,
			)
			maxFrictionDelta := effectivePaddleFriction * normalDeltaSpeed
			desiredDeltaVx := -relativeSlip / 3.0
			deltaVx := clampFloat(desiredDeltaVx, -maxFrictionDelta, maxFrictionDelta)
			b.vx += deltaVx
			b.omega -= 2 * deltaVx / b.r
			b.omega += -paddle.vx * improvedPaddleSpinTransfer / math.Max(b.r, 1)

			preferredDirection := incomingVx
			if preferredDirection == 0 {
				preferredDirection = paddle.vx
			}
			if preferredDirection == 0 {
				preferredDirection = hitPos - 0.5
			}

			// Zero-spin counterfactual for the debug panel. It receives the same
			// paddle movement and impact geometry, but starts with omega=0.
			noSpinPaddle := *b
			noSpinPaddle.omega = 0
			noSpinPaddle.vx -= deltaVx
			noSpinRelativeSlip := noSpinPaddle.vx - paddle.vx
			noSpinDesiredDeltaVx := -noSpinRelativeSlip / 3.0
			noSpinDeltaVx := clampFloat(noSpinDesiredDeltaVx, -maxFrictionDelta, maxFrictionDelta)
			noSpinPaddle.vx += noSpinDeltaVx
			preventVerticalLock(&noSpinPaddle, preferredDirection)
			noSpinSpeed := math.Hypot(noSpinPaddle.vx, noSpinPaddle.vy)
			if noSpinSpeed > maxSpeed {
				scale := maxSpeed / noSpinSpeed
				noSpinPaddle.vx *= scale
				noSpinPaddle.vy *= scale
			}

			preventVerticalLock(b, preferredDirection)

			postCollisionSpeed := math.Hypot(b.vx, b.vy)
			if postCollisionSpeed > maxSpeed {
				scale := maxSpeed / postCollisionSpeed
				b.vx *= scale
				b.vy *= scale
			}
			b.omega = clampFloat(b.omega, -maxSpin, maxSpin)
			maybeShowHighSpin(spinBeforePaddle, b.omega)
			if isPrimary {
				recordLastPaddleSpinDebug(true, spinBeforePaddle, b.omega)
				recordLastCollisionDebug(
					"PADDLE",
					incomingAngle,
					velocityAngleDegrees(b.vx, b.vy),
					velocityAngleDegrees(noSpinPaddle.vx, noSpinPaddle.vy),
					spinBeforePaddle,
					b.omega,
					deltaVx,
					noSpinDeltaVx,
					b.r,
				)
			}
			playImpactSound(b, math.Abs(incomingVy), playPaddleHit)
		} else {
			// Original paddle response, deliberately preserved unchanged.
			speed += paddleBoost
			if speed > maxSpeed {
				speed = maxSpeed
			}
			b.vx = speed * math.Sin(angle)
			b.vy = -speed * math.Cos(angle)
			b.omega += paddle.vx * 0.1 * (hitPos - 0.5)
			if math.Abs(b.omega) > maxSpin {
				b.omega = math.Copysign(maxSpin, b.omega)
			}
			maybeShowHighSpin(spinBeforePaddle, b.omega)
			if isPrimary {
				recordLastPaddleSpinDebug(true, spinBeforePaddle, b.omega)
				recordLastCollisionDebug(
					"PADDLE",
					incomingPaddleAngle,
					velocityAngleDegrees(b.vx, b.vy),
					velocityAngleDegrees(b.vx, b.vy),
					spinBeforePaddle,
					b.omega,
					0,
					0,
					b.r,
				)
			}
			playImpactSound(b, math.Abs(b.vy), playPaddleHit)
		}
	}

	if improved {
		handleImprovedBrickCollisions(b, isPrimary)
	} else {
		// Original brick response, deliberately preserved unchanged.
		for _, i := range candidateBrickIndices(b) {
			brickPtr := &bricks[i]
			if !brickPtr.alive {
				continue
			}
			if b.x+b.r > brickPtr.x && b.x-b.r < brickPtr.x+brickPtr.w &&
				b.y+b.r > brickPtr.y && b.y-b.r < brickPtr.y+brickPtr.h {

				if passActive {
					if destroyBrick(brickPtr) {
						playBrickBreak()
						if influencerActive {
							destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier)
						}
					}
					continue
				}

				overlapX := 0.0
				overlapY := 0.0
				if b.x < brickPtr.x+brickPtr.w/2 {
					overlapX = (b.x + b.r) - brickPtr.x
				} else {
					overlapX = brickPtr.x + brickPtr.w - (b.x - b.r)
				}
				if b.y < brickPtr.y+brickPtr.h/2 {
					overlapY = (b.y + b.r) - brickPtr.y
				} else {
					overlapY = brickPtr.y + brickPtr.h - (b.y - b.r)
				}

				var nx, ny float64
				if overlapX < overlapY {
					if b.x < brickPtr.x+brickPtr.w/2 {
						nx = -1
						b.x = brickPtr.x - b.r
					} else {
						nx = 1
						b.x = brickPtr.x + brickPtr.w + b.r
					}
				} else {
					if b.y < brickPtr.y+brickPtr.h/2 {
						ny = -1
						b.y = brickPtr.y - b.r
					} else {
						ny = 1
						b.y = brickPtr.y + brickPtr.h + b.r
					}
				}

				impactSpeed := math.Max(0, -(b.vx*nx + b.vy*ny))
				resolveSelectedCollisionDebug(b, nx, ny, 0, 0, false, improvedBrickFrictionScale, "BRICK", isPrimary)

				if brickPtr.unbreakable {
					playImpactSound(b, impactSpeed, playUnbreakable)
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
	}
	updateStuckDetector(b, dt)
}

// Improved mode subdivides fast movement so a ball cannot skip through thin
// bricks between frames. Original mode still performs exactly one step.
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
		updateBallStep(b, stepDT, isPrimary, true)
		if b.y+b.r > canvasHeight {
			break
		}
	}
}

func updateBall(b *Ball, dt float64, isPrimary bool) {
	if useImprovedPhysics {
		updateBallAdaptive(b, dt, isPrimary)
		return
	}
	updateBallStep(b, dt, isPrimary, false)
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

	// Let the HTML page own the Screen Wake Lock lifecycle.
	wakeLockCallback := js.Global().Get("breakoutSetTiltWakeLock")
	if wakeLockCallback.Type() == js.TypeFunction {
		wakeLockCallback.Invoke(mode == "tilt")
	}

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
	z-index: 100000;
	display: none;
	align-items: flex-start;
	justify-content: center;
	box-sizing: border-box;
	padding:
		max(18px, env(safe-area-inset-top))
		max(18px, env(safe-area-inset-right))
		max(18px, env(safe-area-inset-bottom))
		max(18px, env(safe-area-inset-left));
	overflow-x: hidden;
	overflow-y: auto;
	-webkit-overflow-scrolling: touch;
	overscroll-behavior: contain;
	background: rgba(8, 10, 24, 0.94);
	font-family: GameFont, monospace;
}
#mobileControlPanel {
	box-sizing: border-box;
	width: min(620px, 96vw);
	max-width: 100%;
	margin: auto 0;
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

	// Keyboard and two-thumb controls use acceleration rather than jumping
	// immediately to one fixed speed. Short taps make small corrections;
	// holding a direction quickly reaches the higher cross-screen speed.
	digitalControl := false
	digitalDirection := 0.0

	if leftPressed || rightPressed {
		digitalControl = true
		if leftPressed && !rightPressed {
			digitalDirection = -1
		} else if rightPressed && !leftPressed {
			digitalDirection = 1
		}
	} else if mobileControlsEnabled && mobileControlMode == "two-thumb" {
		digitalControl = true
		if mobileLeftHeld && !mobileRightHeld {
			digitalDirection = -1
		} else if mobileRightHeld && !mobileLeftHeld {
			digitalDirection = 1
		}
	}

	if digitalControl {
		targetSpeed := digitalDirection * defaultDigitalPaddleMaxSpeed
		changeRate := defaultDigitalPaddleAcceleration
		if digitalDirection == 0 {
			changeRate = defaultDigitalPaddleBraking
		}
		paddle.vx = moveToward(paddle.vx, targetSpeed, changeRate*dt)
		paddle.x += paddle.vx * dt
	} else if mobileControlsEnabled {
		switch mobileControlMode {
		case "vertical", "follow":
			// These modes position the paddle directly from pointer events.
			if !touchControlActive {
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
		// Brake keyboard movement quickly after the key is released.
		paddle.vx = moveToward(paddle.vx, 0, defaultDigitalPaddleBraking*dt)
		paddle.x += paddle.vx * dt
	}
	if paddle.x < 0 {
		paddle.x = 0
	}
	if paddle.x+paddle.w > canvasWidth {
		paddle.x = canvasWidth - paddle.w
	}

	if dt > 0 {
		paddle.vx = (paddle.x - paddlePreviousX) / dt
	} else {
		paddle.vx = 0
	}
	paddlePreviousX = paddle.x

	// ---- Independent power-up timers ----
	gravityChanged := false
	if lowGravityActive {
		lowGravityTimer -= dt
		if lowGravityTimer <= 0 {
			lowGravityActive = false
			lowGravityTimer = 0
			gravityChanged = true
		}
	}
	if reverseGravityActive {
		reverseGravityTimer -= dt
		if reverseGravityTimer <= 0 {
			reverseGravityActive = false
			reverseGravityTimer = 0
			gravityChanged = true
		}
	}
	if passActive {
		passTimer -= dt
		if passTimer <= 0 {
			passActive = false
			passTimer = 0
		}
	}
	if magnetPowerActive {
		magnetPowerTimer -= dt
		if magnetPowerTimer <= 0 {
			magnetPowerActive = false
			magnetPowerTimer = 0
		}
	}
	if zapperPowerActive {
		zapperPowerTimer -= dt
		if zapperPowerTimer <= 0 {
			zapperPowerActive = false
			zapperPowerTimer = 0
			zapperTargetIndex = -1
			zapperHitTimer = 0
			secondZapperTargetIndex = -1
			secondZapperHitTimer = 0
		}
	}
	if bigPaddleActive {
		bigPaddleTimer -= dt
		if bigPaddleTimer <= 0 {
			bigPaddleActive = false
			bigPaddleTimer = 0
			setPaddleSize(paddleWidth, paddleHeight)
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
			blackHoleTimer = 0
			blackHoleDirection = 0
			blackHoleSpeed = 0
			gravityChanged = true
		}
	}

	if influencerActive {
		influencerTimer -= dt
		if influencerTimer <= 0 {
			influencerActive = false
			influencerTimer = 0
		}
	}

	if gravityChanged {
		refreshCurrentGravity()
	}

	updateZapper(dt)

	for i := len(statusMessages) - 1; i >= 0; i-- {
		statusMessages[i].timer -= dt
		if statusMessages[i].timer <= 0 {
			statusMessages = append(statusMessages[:i], statusMessages[i+1:]...)
		}
	}

	// Primary ball
	updateBall(&ball, dt, true)

	primaryLost := ball.y+ball.r > canvasHeight
	secondLost := secondBallActive &&
		secondBall.y+secondBall.r > canvasHeight

	if secondBallActive {
		updateBall(&secondBall, dt, false)

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
		lives++
		unlockNextLevel()
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
		showStatus("Level complete! +1 life", 3.0)
	}
}

// ---- Reset balls on life lost ----
func resetBalls() {
	ball.x, ball.y = startBallX, startBallY
	ball.vx, ball.vy = startBallVx, startBallVy
	ball.omega, ball.angle = 0, 0
	ball.stuckTimer = 0
	ball.soundCooldown = 0
	secondBall.stuckTimer = 0
	secondBall.soundCooldown = 0
	secondBallActive = false
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.vx = 0
	paddlePreviousX = paddle.x
	clearLastPaddleSpinDebug()
	clearLastCollisionDebug()
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

func configState(key string) (effective, defaultValue, kind string, ok bool) {
	switch key {
	case "backgroundColor":
		return palette[0], defaultPalette[0], "string", true
	case "paddleColor":
		return palette[1], defaultPalette[1], "string", true
	case "brickColor":
		return palette[2], defaultPalette[2], "string", true
	case "ballColor":
		return palette[3], defaultPalette[3], "string", true
	case "textColor":
		return palette[4], defaultPalette[4], "string", true
	case "spinColor":
		return palette[5], defaultPalette[5], "string", true
	case "unbreakableColor":
		return palette[6], defaultPalette[6], "string", true
	case "secondBallColor":
		return palette[7], defaultPalette[7], "string", true
	case "secondBallSpinColor":
		return palette[8], defaultPalette[8], "string", true
	case "magicColor":
		return magicColor, defaultMagicColor, "string", true
	case "magicStrokeColor":
		return magicStrokeColor, defaultMagicStrokeColor, "string", true
	case "unbreakableStrokeColor":
		return unbreakableStrokeColor, defaultUnbreakableStrokeColor, "string", true
	case "brickStrokeColor":
		return brickStrokeColor, defaultBrickStrokeColor, "string", true
	case "gravity":
		return formatConfigFloat(baseGravity), formatConfigFloat(defaultGravity), "float", true
	case "restitution":
		return formatConfigFloat(restitution), formatConfigFloat(defaultRestitution), "float", true
	case "frictionCoeff":
		return formatConfigFloat(frictionCoeff), formatConfigFloat(defaultFrictionCoeff), "float", true
	case "maxSpin":
		return formatConfigFloat(maxSpin), formatConfigFloat(defaultMaxSpin), "float", true
	case "paddleBoost":
		return formatConfigFloat(paddleBoost), formatConfigFloat(defaultPaddleBoost), "float", true
	case "brickBoost":
		return formatConfigFloat(brickBoost), formatConfigFloat(defaultBrickBoost), "float", true
	case "maxSpeed":
		return formatConfigFloat(maxSpeed), formatConfigFloat(defaultMaxSpeed), "float", true
	case "stuckSpeedThreshold":
		return formatConfigFloat(stuckSpeedThreshold), formatConfigFloat(defaultStuckSpeedThreshold), "float", true
	case "stuckDuration":
		return formatConfigFloat(stuckDuration), formatConfigFloat(defaultStuckDuration), "float", true
	case "tiltUpSpeed":
		return formatConfigFloat(tiltUpSpeed), formatConfigFloat(defaultTiltUpSpeed), "float", true
	case "tiltSideMin":
		return formatConfigFloat(tiltSideMin), formatConfigFloat(defaultTiltSideMin), "float", true
	case "tiltSideMax":
		return formatConfigFloat(tiltSideMax), formatConfigFloat(defaultTiltSideMax), "float", true
	case "powerUpDuration":
		return formatConfigFloat(powerUpDuration), formatConfigFloat(defaultPowerUpDuration), "float", true
	case "magnet":
		return strconv.FormatBool(levelMagnetActive), "false", "bool", true
	case "zapper":
		return strconv.FormatBool(levelZapperActive), "false", "bool", true
	case "magnetStrength":
		return formatConfigFloat(magnetStrength), formatConfigFloat(defaultMagnetStrength), "float", true
	case "magnetRange":
		return formatConfigFloat(magnetRange), formatConfigFloat(defaultMagnetRange), "float", true
	case "influencerMultiplier":
		return formatConfigFloat(influencerMultiplier), formatConfigFloat(defaultInfluencerMultiplier), "float", true
	case "zapperRange":
		return formatConfigFloat(zapperRange), formatConfigFloat(defaultZapperRange), "float", true
	case "brickOffsetTop":
		return formatConfigFloat(brickOffsetTop), formatConfigFloat(defaultBrickOffsetTop), "float", true
	case "brickWidth", "defaultBrickWidth":
		return formatConfigFloat(brickWidth), formatConfigFloat(defaultBrickWidth), "float", true
	case "brickHeight", "defaultBrickHeight":
		return formatConfigFloat(brickHeight), formatConfigFloat(defaultBrickHeight), "float", true
	case "brickPadding", "defaultBrickPadding":
		return formatConfigFloat(brickPadding), formatConfigFloat(defaultBrickPadding), "float", true
	case "brickRows":
		return strconv.Itoa(brickRows), strconv.Itoa(defaultBrickRows), "int", true
	case "brickCols":
		return strconv.Itoa(brickCols), strconv.Itoa(defaultBrickCols), "int", true
	case "ballX":
		return formatConfigFloat(startBallX), formatConfigFloat(defaultStartBallX), "float", true
	case "ballY":
		return formatConfigFloat(startBallY), formatConfigFloat(defaultStartBallY), "float", true
	case "ballVx":
		return formatConfigFloat(startBallVx), formatConfigFloat(defaultStartBallVx), "float", true
	case "ballVy":
		return formatConfigFloat(startBallVy), formatConfigFloat(defaultStartBallVy), "float", true
	case "ballRadius":
		return formatConfigFloat(ballRadius), formatConfigFloat(defaultBallRadius), "float", true
	case "enableLowGravity":
		return strconv.FormatBool(enableLowGravity), strconv.FormatBool(defaultEnableLowGravity), "bool", true
	case "enablePassThrough":
		return strconv.FormatBool(enablePassThrough), strconv.FormatBool(defaultEnablePassThrough), "bool", true
	case "enableNuke":
		return strconv.FormatBool(enableNuke), strconv.FormatBool(defaultEnableNuke), "bool", true
	case "enableReverseGravity":
		return strconv.FormatBool(enableReverseGravity), strconv.FormatBool(defaultEnableReverseGravity), "bool", true
	case "enableDualBalls":
		return strconv.FormatBool(enableDualBalls), strconv.FormatBool(defaultEnableDualBalls), "bool", true
	case "enableBlackHole":
		return strconv.FormatBool(enableBlackHole), strconv.FormatBool(defaultEnableBlackHole), "bool", true
	case "enableMagnet":
		return strconv.FormatBool(enableMagnet), strconv.FormatBool(defaultEnableMagnet), "bool", true
	case "enableInfluencer":
		return strconv.FormatBool(enableInfluencer), strconv.FormatBool(defaultEnableInfluencer), "bool", true
	case "enableZapper":
		return strconv.FormatBool(enableZapper), strconv.FormatBool(defaultEnableZapper), "bool", true
	case "enableBreakUnbreakable":
		return strconv.FormatBool(enableBreakUnbreakable), strconv.FormatBool(defaultEnableBreakUnbreakable), "bool", true
	case "enableBigPaddle":
		return strconv.FormatBool(enableBigPaddle), strconv.FormatBool(defaultEnableBigPaddle), "bool", true
	case "paddleWidth":
		return formatConfigFloat(paddleWidth), formatConfigFloat(defaultPaddleWidth), "float", true
	case "paddleHeight":
		return formatConfigFloat(paddleHeight), formatConfigFloat(defaultPaddleHeight), "float", true
	default:
		return "", "", "", false
	}
}

func formatConfigFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func configValuesEqual(kind, a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	switch kind {
	case "float":
		af, errA := strconv.ParseFloat(a, 64)
		bf, errB := strconv.ParseFloat(b, 64)
		return errA == nil && errB == nil && math.Abs(af-bf) <= 1e-9
	case "int":
		ai, errA := strconv.Atoi(a)
		bi, errB := strconv.Atoi(b)
		return errA == nil && errB == nil && ai == bi
	case "bool":
		ab, errA := strconv.ParseBool(a)
		bb, errB := strconv.ParseBool(b)
		return errA == nil && errB == nil && ab == bb
	default:
		return a == b
	}
}

func isPaletteConfigKey(key string) bool {
	return strings.HasSuffix(key, "Color")
}

func currentLevelOverrideSections() (otherLines, paletteLines []string) {
	if currentLevelIndex < 0 || currentLevelIndex >= len(levels) {
		return nil, nil
	}
	config := levels[currentLevelIndex].config
	keys := make([]string, 0, len(config))
	for key := range config {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		effective, defaultValue, kind, accepted := configState(key)
		if !accepted {
			continue
		}
		// Only list values that were successfully applied and differ from the
		// corresponding built-in default.
		if !configValuesEqual(kind, config[key], effective) {
			continue
		}
		if configValuesEqual(kind, effective, defaultValue) {
			continue
		}

		line := key + "=" + effective
		if isPaletteConfigKey(key) {
			paletteLines = append(paletteLines, line)
		} else {
			otherLines = append(otherLines, line)
		}
	}
	return otherLines, paletteLines
}

func debugOverlayLines() []string {
	devState := "OFF"
	if devAllLevelsUnlocked {
		devState = "ON"
	}

	physicsMode := "ORIGINAL"
	if useImprovedPhysics {
		physicsMode = "IMPROVED"
	}

	lines := []string{
		"BUILD " + buildID,
		"PHYSICS " + physicsMode,
		"FPS CURRENT " + fmt.Sprintf("%.1f", fpsCurrent),
		"BALL 1 SPEED CURRENT " + fmt.Sprintf("%.2f", math.Hypot(ball.vx, ball.vy)),
		"BALL 1 SPIN CURRENT " + fmt.Sprintf("%+.2f", ball.omega),
	}
	if lastPaddleSpinValid {
		lines = append(lines,
			"LAST PADDLE HIT BALL "+strconv.Itoa(lastPaddleSpinBall),
			"BALL SPIN BEFORE "+fmt.Sprintf("%+.2f", lastPaddleSpinBefore),
			"BALL SPIN ADDED  "+fmt.Sprintf("%+.2f", lastPaddleSpinAdded),
			"BALL SPIN AFTER  "+fmt.Sprintf("%+.2f", lastPaddleSpinAfter),
		)
	} else {
		lines = append(lines, "LAST PADDLE HIT (none)")
	}
	if lastCollisionValid {
		lines = append(lines,
			"LAST BALL 1 COLLISION "+lastCollisionSurface,
			"ANGLE IN          "+fmt.Sprintf("%+.2f deg", lastCollisionIncomingAngle),
			"ANGLE OUT         "+fmt.Sprintf("%+.2f deg", lastCollisionOutgoingAngle),
			"ANGLE CHANGE      "+fmt.Sprintf("%+.2f deg", lastCollisionAngleChange),
			"SPIN ANGLE EFFECT "+fmt.Sprintf("%+.2f deg", lastCollisionSpinAngleEffect),
			"HIT SPIN BEFORE   "+fmt.Sprintf("%+.2f", lastCollisionSpinBefore),
			"HIT SPIN CHANGE   "+fmt.Sprintf("%+.2f", lastCollisionSpinChange),
			"HIT SPIN AFTER     "+fmt.Sprintf("%+.2f", lastCollisionSpinAfter),
			"CONTACT SPIN SPEED "+fmt.Sprintf("%+.2f", lastCollisionContactSpinSurfaceSpeed),
			"IMPULSE WITH SPIN  "+fmt.Sprintf("%+.2f", lastCollisionTangentialImpulse),
			"IMPULSE NO SPIN    "+fmt.Sprintf("%+.2f", lastCollisionNoSpinImpulse),
			"SPIN IMPULSE EFFECT "+fmt.Sprintf("%+.2f", lastCollisionSpinImpulseEffect),
		)
	} else {
		lines = append(lines, "LAST BALL 1 COLLISION (none)")
	}
	lines = append(lines,
		"LEVEL "+strconv.Itoa(currentLevelIndex+1)+"/"+strconv.Itoa(len(levels)),
		"UNLOCKED THROUGH "+strconv.Itoa(highestUnlockedLevel+1),
		"DEV ALL LEVELS "+devState,
		"LEVEL OVERRIDES:",
	)

	otherLines, paletteLines := currentLevelOverrideSections()
	if len(otherLines) == 0 && len(paletteLines) == 0 {
		return append(lines, "(none)")
	}

	lines = append(lines, otherLines...)
	if len(paletteLines) > 0 {
		lines = append(lines, "", "PALETTE / COLORS:")
		lines = append(lines, paletteLines...)
	}
	return lines
}

func debugOverlayGeometry(lineCount int) (panelX, panelY, panelWidth, panelHeight float64, maxRows int) {
	const lineHeight = 18.0
	const padding = 12.0
	const columnWidth = 420.0

	maxRows = int((canvasHeight - padding*4) / lineHeight)
	if maxRows < 1 {
		maxRows = 1
	}
	columns := (lineCount + maxRows - 1) / maxRows
	if columns < 1 {
		columns = 1
	}
	panelWidth = float64(columns)*columnWidth + padding*2
	rows := lineCount
	if rows > maxRows {
		rows = maxRows
	}
	panelHeight = float64(rows)*lineHeight + padding*2
	panelX = canvasWidth - panelWidth - 10
	panelY = canvasHeight - panelHeight - 10
	if panelX < 10 {
		panelX = 10
	}
	if panelY < 10 {
		panelY = 10
	}
	return panelX, panelY, panelWidth, panelHeight, maxRows
}

func debugOverlayReport() string {
	return strings.Join(debugOverlayLines(), "\n")
}

func pointerCanvasPosition(e js.Value) (x, y float64, ok bool) {
	rect := canvas.Call("getBoundingClientRect")
	width := rect.Get("width").Float()
	height := rect.Get("height").Float()
	if width <= 0 || height <= 0 {
		return 0, 0, false
	}

	x = (e.Get("clientX").Float() - rect.Get("left").Float()) * canvasWidth / width
	y = (e.Get("clientY").Float() - rect.Get("top").Float()) * canvasHeight / height
	return x, y, true
}

func pointerInsideDebugOverlay(e js.Value) bool {
	if !debugOverlayVisible {
		return false
	}
	x, y, ok := pointerCanvasPosition(e)
	if !ok {
		return false
	}
	panelX, panelY, panelWidth, panelHeight, _ := debugOverlayGeometry(len(debugOverlayLines()))
	return x >= panelX && x <= panelX+panelWidth &&
		y >= panelY && y <= panelY+panelHeight
}

func copyTextToClipboard(text string) bool {
	navigator := js.Global().Get("navigator")
	if !navigator.IsUndefined() && !navigator.IsNull() {
		clipboard := navigator.Get("clipboard")
		if !clipboard.IsUndefined() && !clipboard.IsNull() {
			clipboard.Call("writeText", text)
			return true
		}
	}

	// Fallback for older browsers and non-secure local-network pages.
	document := js.Global().Get("document")
	body := document.Get("body")
	if body.IsUndefined() || body.IsNull() {
		return false
	}
	textarea := document.Call("createElement", "textarea")
	textarea.Set("value", text)
	textarea.Call("setAttribute", "readonly", "")
	style := textarea.Get("style")
	style.Set("position", "fixed")
	style.Set("left", "-9999px")
	style.Set("top", "0")
	style.Set("opacity", "0")
	body.Call("appendChild", textarea)
	textarea.Call("focus")
	textarea.Call("select")
	success := document.Call("execCommand", "copy").Bool()
	body.Call("removeChild", textarea)
	return success
}

func drawDebugOverlay() {
	if !debugOverlayVisible {
		return
	}

	lines := debugOverlayLines()
	panelX, panelY, panelWidth, panelHeight, maxRows := debugOverlayGeometry(len(lines))

	const lineHeight = 18.0
	const padding = 12.0
	const columnWidth = 420.0

	ctx.Call("save")
	ctx.Set("fillStyle", "rgba(0, 0, 0, 0.82)")
	ctx.Call("fillRect", panelX, panelY, panelWidth, panelHeight)
	ctx.Set("strokeStyle", "rgba(255, 255, 255, 0.35)")
	ctx.Set("lineWidth", 1)
	ctx.Call("strokeRect", panelX, panelY, panelWidth, panelHeight)
	ctx.Set("fillStyle", "#ffffff")
	ctx.Set("font", "14px GameFont, monospace")
	ctx.Set("textAlign", "left")

	for i, line := range lines {
		column := i / maxRows
		row := i % maxRows
		x := panelX + padding + float64(column)*columnWidth
		y := panelY + padding + lineHeight*float64(row+1) - 3
		ctx.Call("fillText", line, x, y)
	}
	ctx.Call("restore")
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

		msg := "GAME OVER"
		fontSize := "88px GameFont, monospace"
		instruction := "Press Space, Enter, click, or touch to retry"
		if win {
			msg = "YOU WIN!"
			fontSize = "112px GameFont, monospace"
			instruction = "Press Space, Enter, click, or touch to start again"
		}

		ctx.Set("font", fontSize)
		ctx.Call("fillText", msg, canvasWidth/2, canvasHeight/2-10)

		ctx.Set("font", "28px GameFont, monospace")
		ctx.Call("fillText", instruction, canvasWidth/2, canvasHeight/2+70)

		ctx.Set("textAlign", "start")
	}

	drawDebugOverlay()
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
	rawDt := 0.0
	if lastTime != 0 {
		rawDt = (now - lastTime) / 1000.0
	}
	lastTime = now

	// Measure the browser's actual requestAnimationFrame cadence. This is not
	// capped at 60; high-refresh displays can report higher values. Ignore only
	// long background-tab gaps and publish a smoothed half-second sample.
	if rawDt > 0 && rawDt < 1.0 {
		fpsSampleElapsed += rawDt
		fpsSampleFrames++
		if fpsSampleElapsed >= 0.5 {
			sample := float64(fpsSampleFrames) / fpsSampleElapsed
			if fpsCurrent == 0 {
				fpsCurrent = sample
			} else {
				fpsCurrent = fpsCurrent*0.65 + sample*0.35
			}
			fpsSampleElapsed = 0
			fpsSampleFrames = 0
		}
	}

	dt := rawDt
	if dt > 0.05 {
		dt = 0.05
	}
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
		code := e.Get("code").String()

		// Temporary development unlock. It is deliberately not stored.
		if (key == "u" || key == "U") && !e.Get("repeat").Bool() {
			devAllLevelsUnlocked = !devAllLevelsUnlocked
			if devAllLevelsUnlocked {
				showStatus("Developer level unlock ON", 2.0)
			} else {
				permanentLimit := permanentUnlockedLimit()
				if currentLevelIndex > permanentLimit {
					jumpToLevel(permanentLimit)
				}
				showStatus("Developer level unlock OFF", 2.0)
			}
			return nil
		}

		// Toggle the build/config debug panel.
		if (key == "i" || key == "I") && !e.Get("repeat").Bool() {
			debugOverlayVisible = !debugOverlayVisible
			return nil
		}

		// Runtime A/B physics switch. Startup mode is controlled by defaultUseImprovedPhysics.
		if (key == "f" || key == "F") && !e.Get("repeat").Bool() {
			useImprovedPhysics = !useImprovedPhysics
			ball.stuckTimer = 0
			clearLastPaddleSpinDebug()
			clearLastCollisionDebug()
			secondBall.stuckTimer = 0
			if useImprovedPhysics {
				showStatus("Physics: improved", 2.0)
			} else {
				showStatus("Physics: original", 2.0)
			}
			return nil
		}

		// Start a new game after winning. N/P remain level-navigation cheats.
		if gameOver && win {
			if (key == " " || key == "Enter") && !e.Get("repeat").Bool() {
				jumpToLevel(0)
			}
			return nil
		}

		// Cheat keys must work even while waiting to launch, paused, or on
		// a game-over/win screen.
		if (key == "n" || key == "N") && !e.Get("repeat").Bool() {
			limit := activeUnlockedLimit()
			if currentLevelIndex < limit {
				jumpToLevel(currentLevelIndex + 1)
			} else if limit == len(levels)-1 {
				jumpToLevel(0)
			} else {
				showStatus("Finish this level to unlock the next", 2.0)
			}
			return nil
		}
		if (key == "p" || key == "P") && !e.Get("repeat").Bool() {
			limit := activeUnlockedLimit()
			if currentLevelIndex > 0 {
				jumpToLevel(currentLevelIndex - 1)
			} else if limit > 0 {
				jumpToLevel(limit)
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
				key == "ArrowLeft" || key == "ArrowRight" ||
				key == "a" || key == "A" || key == "d" || key == "D" ||
				code == "AltLeft" || code == "AltRight") &&
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

		// Paddle controls: arrows, A/D, and separate left/right Alt keys.
		if key == "ArrowLeft" || key == "a" || key == "A" || code == "AltLeft" {
			leftPressed = true
		} else if key == "ArrowRight" || key == "d" || key == "D" || code == "AltRight" {
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
		code := e.Get("code").String()
		if key == "ArrowLeft" || key == "a" || key == "A" || code == "AltLeft" {
			leftPressed = false
		} else if key == "ArrowRight" || key == "d" || key == "D" || code == "AltRight" {
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

		// Clicking or tapping the visible debug panel copies its complete text.
		// Consume the event so it does not also launch, unpause, or move the paddle.
		if pointerInsideDebugOverlay(e) {
			if copyTextToClipboard(debugOverlayReport()) {
				showStatus("Debug info copied", 1.5)
			} else {
				showStatus("Clipboard unavailable", 1.5)
			}
			return nil
		}

		if !audioInitialized {
			initAudio()
		} else {
			ensureAudioRunning()
		}

		pointerType := e.Get("pointerType").String()

		if gameOver && win {
			jumpToLevel(0)
			return nil
		}

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
		limit := activeUnlockedLimit()
		if currentLevelIndex > 0 {
			jumpToLevel(currentLevelIndex - 1)
		} else if limit > 0 {
			jumpToLevel(limit)
		}
	})

	bindMobileButton("nextLevelButton", func() {
		limit := activeUnlockedLimit()
		if currentLevelIndex < limit {
			jumpToLevel(currentLevelIndex + 1)
		} else if limit == len(levels)-1 {
			jumpToLevel(0)
		} else {
			showStatus("Finish this level to unlock the next", 2.0)
		}
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
	resetGame()

	setPausedCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 || gameOver {
			return nil
		}

		paused = args[0].Bool()
		leftPressed = false
		rightPressed = false
		mobileLeftHeld = false
		mobileRightHeld = false
		touchControlActive = false
		paddle.vx = 0
		return nil
	})
	js.Global().Set("breakoutSetPaused", setPausedCallback)

	getPausedCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		return paused
	})
	js.Global().Set("breakoutIsPaused", getPausedCallback)

	loopFunc = js.FuncOf(gameLoop)
	js.Global().Call("requestAnimationFrame", loopFunc)
	log("main: requestAnimationFrame called")

	select {}
}
