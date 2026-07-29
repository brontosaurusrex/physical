//go:build js && wasm

package main

import (
	"embed"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"sort"
	"strconv"
	"strings"
	"syscall/js"
)

// Compressed runtime samples are embedded from dedicated runtime directories.
// Editable WAV masters stay one directory above and are not embedded as long as
// they are kept out of runtime/. Supported compressed files are decoded
// sequentially at game startup so gameplay never waits for first-use decoding.
//
//go:embed sounds/brickHits/runtime sounds/magicBrickHits/runtime
var embeddedRuntimeAudio embed.FS

// ---- Default values (constants) ----
const (
	buildID = "20260729-8f31c6e4a9"

	// Three submix buses feed the master output. Change these values to rebalance
	// complete sound families without editing individual sound definitions.
	audioMixerMaster = 1.00
	audioMixerBricks = 0.30
	audioMixerMagic  = 0.20
	audioMixerSynths = 0.80

	// Level files may select a room with audioRoom=<name> and optionally
	// override its dry amount with audioRoomDry=0..1. Omitted audioRoom means
	// a completely dry signal path with no room effect.
	audioRoomTransitionSeconds = 0.16

	brickHitSampleDirectory = "sounds/brickHits/runtime"
	brickHitPlaybackRateMin = 0.94
	brickHitPlaybackRateMax = 1.07
	brickHitFilterMinHz     = 2400.0
	brickHitFilterMaxHz     = 12900.0
	brickHitGainMin         = 0.16
	brickHitGainMax         = 0.40

	magicFeatureSampleDirectory = "sounds/magicBrickHits/runtime"
	magicFeaturePlaybackRateMin = 0.96
	magicFeaturePlaybackRateMax = 1.04
	magicFeatureFilterMinHz     = 3000.0
	magicFeatureFilterMaxHz     = 16000.0
	magicFeatureHardMaxHz       = 500.0
	magicFeatureGainMin         = 0.15
	magicFeatureGainMax         = 0.32
	magicFeatureMaxActiveVoices = 4
	magicFeatureRetriggerFade   = 0.018

	brickHitMaxActiveVoices    = 8
	brickHitMaxStartsPerFrame  = 3
	brickSoundPanLimit         = 0.75
	embeddedAudioDecodeYieldMS = 12

	// Zapper-destroyed bricks use their normal sample bank, but the sample is
	// shifted upward so an electrical kill is distinct from a ball collision.
	zapperBrickPitchScale = 1.32
	zapperBrickStrength   = 0.72

	// Rendering follows requestAnimationFrame, but simulation always advances in
	// fixed 1/240-second steps. At maxSpeed=1250 this is about 5.2 px per tick.
	physicsStepHz             = 240.0 // 240.0
	physicsStepSeconds        = 1.0 / physicsStepHz
	physicsMaxCatchUpSteps    = 32
	physicsMaxFrameDelta      = physicsStepSeconds * physicsMaxCatchUpSteps
	physicsWarningHoldSeconds = 3.0

	defaultCanvasWidth    = 1800.0
	defaultCanvasHeight   = 900.0
	defaultPaddleWidth    = 220.0
	defaultPaddleHeight   = 30.0
	defaultBallRadius     = 8.0
	defaultBrickRows      = 10
	defaultBrickCols      = 18
	defaultBrickWidth     = 60.0
	defaultBrickHeight    = 20.0
	defaultBrickPadding   = 20.0
	defaultBrickOffsetTop = -1.0

	// Keyboard and two-thumb controls accelerate from a precise low speed
	// to a faster cross-screen speed, then brake quickly when released.
	defaultDigitalPaddleMaxSpeed     = 2800.0
	defaultDigitalPaddleAcceleration = 9000.0
	defaultDigitalPaddleBraking      = 40000.0

	// Mouse input supplies only a target. The fixed-step simulation owns the
	// actual movement, using a stopping-distance controller so the paddle is
	// responsive, does not overshoot, and produces refresh-independent spin.
	defaultMousePaddleMaxSpeed     = 6000.0
	defaultMousePaddleAcceleration = 50000.0
	defaultMousePaddleBraking      = 70000.0
	defaultMousePaddleSnapDistance = 0.35

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
	defaultZapperHitTime        = 0.1
	defaultZapperRange          = 320.0

	// Physics defaults. These are the single runtime physics model.
	defaultPhysicsGravity             = 300.0
	defaultPhysicsRestitution         = 0.90
	defaultPhysicsFrictionCoeff       = 0.14 // 0.10
	defaultPhysicsPaddleBoost         = 900.0
	defaultPhysicsBrickBoost          = 100.0
	defaultPhysicsMaxSpeed            = 1000.0 // 1170.0
	defaultPhysicsMaxSpin             = 1530.0
	defaultPhysicsStuckSpeedThreshold = 85.0
	defaultPhysicsStuckDuration       = 1.2 //3.0
	defaultPhysicsTiltUpSpeed         = 520.0
	defaultPhysicsTiltSideMin         = 180.0
	defaultPhysicsTiltSideMax         = 340.0

	// Contact and flight tuning.
	physicsMagnusCoefficient        = 0.0030 //0.0015
	physicsMagnusAccelerationScale  = 0.25
	physicsSpinDrag                 = 0.04 // 0.10
	physicsAirDrag                  = 0.010
	physicsWallFrictionScale        = 2.00 //0.35
	physicsBrickFrictionScale       = 2.00 //1.00
	physicsUnbreakableFrictionScale = 1.80 //0.35
	physicsPaddleFrictionScale      = 3.20 //2.60
	physicsPaddleSpinTransfer       = 2.80 //2.25
	physicsCollisionSpinCoupling    = 2.00 //2.00
	physicsMinimumCollisionGrip     = 0.08
	physicsMinimumPaddleGrip        = 0.55 // 0.45
	physicsCollisionSlop            = 0.05

	// A short input-memory window makes deliberate paddle spin less dependent on
	// landing on one exact 240 Hz tick. Only spin transfer uses this history.
	paddleSpinGraceSeconds = 0.050

	// Speeds above the level max are allowed when collision spin converts into
	// translation, then their excess decays smoothly back toward maxSpeed.
	physicsOverspeedHalfLife = 0.35

	// Invisible deterministic wall roughness. Nearby impact positions receive
	// smoothly related normals; the same level and position always match.
	wallNoiseCellSize          = 20.0 //120.0
	wallSideTiltDegrees        = 0.15
	wallTopTiltDegrees         = 3.45 //0.45
	wallCornerFadeDistance     = 40.0
	wallNoiseIDLeft        int = 1
	wallNoiseIDRight       int = 2
	wallNoiseIDTop         int = 3

	// Every brick receives a stable, tiny rotation. The same
	// angle is used for drawing and collision normals, breaking exact vertical
	// loops without changing the brick grid or consuming gameplay randomness.
	brickTiltMinDegrees = 0.2
	brickTiltMaxDegrees = 2.0 // 1.0
	drawBrickTilt       = true

	// Detector for fast, nearly axis-aligned loops involving
	// unbreakable bricks. It preserves total speed and holds one escape direction
	// briefly so repeated collisions cannot alternate the correction sign.
	physicsOrbitMinimumSpeed         = 300.0
	physicsOrbitMinimumHitSpeed      = 80.0
	physicsOrbitMinorSpeedRatio      = 0.08
	physicsOrbitMinorSpeedFloor      = 45.0
	physicsOrbitRequiredHits         = 3
	physicsOrbitDetectionWindow      = 4.0
	physicsOrbitMaximumMinorProgress = 40.0
	physicsOrbitHitCooldown          = 0.08
	physicsOrbitEscapeSpeed          = 110.0
	physicsOrbitEscapeDuration       = 0.90
	physicsOrbitMessageDuration      = 1.5

	statusMessageLimit = 10

	enableHighSpinMessage   = true
	highSpinThreshold       = 100.0 // 100.0
	highSpinMessageDuration = 3.5

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

// physicsSettings contains the level-overridable values for the single physics model.
type physicsSettings struct {
	gravity             float64
	restitution         float64
	frictionCoeff       float64
	paddleBoost         float64
	brickBoost          float64
	maxSpeed            float64
	maxSpin             float64
	stuckSpeedThreshold float64
	stuckDuration       float64
	tiltUpSpeed         float64
	tiltSideMin         float64
	tiltSideMax         float64
}

func defaultPhysicsSettings() physicsSettings {
	return physicsSettings{
		gravity: defaultPhysicsGravity, restitution: defaultPhysicsRestitution,
		frictionCoeff: defaultPhysicsFrictionCoeff, paddleBoost: defaultPhysicsPaddleBoost,
		brickBoost: defaultPhysicsBrickBoost, maxSpeed: defaultPhysicsMaxSpeed,
		maxSpin: defaultPhysicsMaxSpin, stuckSpeedThreshold: defaultPhysicsStuckSpeedThreshold,
		stuckDuration: defaultPhysicsStuckDuration, tiltUpSpeed: defaultPhysicsTiltUpSpeed,
		tiltSideMin: defaultPhysicsTiltSideMin, tiltSideMax: defaultPhysicsTiltSideMax,
	}
}

var physicsConfig = defaultPhysicsSettings()

func setPhysicsSetting(settings *physicsSettings, key string, value float64) {
	switch key {
	case "gravity":
		settings.gravity = value
	case "restitution":
		settings.restitution = value
	case "frictionCoeff":
		settings.frictionCoeff = value
	case "paddleBoost":
		settings.paddleBoost = value
	case "brickBoost":
		settings.brickBoost = value
	case "maxSpeed":
		settings.maxSpeed = value
	case "maxSpin":
		settings.maxSpin = value
	case "stuckSpeedThreshold":
		settings.stuckSpeedThreshold = value
	case "stuckDuration":
		settings.stuckDuration = value
	case "tiltUpSpeed":
		settings.tiltUpSpeed = value
	case "tiltSideMin":
		settings.tiltSideMin = value
	case "tiltSideMax":
		settings.tiltSideMax = value
	}
}

// ---- These are variables, reset each level ----
var (
	canvasWidth    = defaultCanvasWidth
	canvasHeight   = defaultCanvasHeight
	paddleWidth    = defaultPaddleWidth
	paddleHeight   = defaultPaddleHeight
	ballRadius     = defaultBallRadius
	brickRows      = defaultBrickRows
	brickCols      = defaultBrickCols
	brickWidth     = defaultBrickWidth
	brickHeight    = defaultBrickHeight
	brickPadding   = defaultBrickPadding
	brickOffsetTop = defaultBrickOffsetTop

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

	// Per-level room settings. A negative dry value means use the preset default.
	currentAudioRoom    = "none"
	currentAudioRoomDry = -1.0

	paused       bool
	leftPressed  bool
	rightPressed bool

	touchControlActive bool
	touchPointerID     int
	touchLastY         float64

	mouseControlActive bool
	mousePaddleTargetX float64

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

	orbitCandidateAxis   int
	orbitCandidateHits   int
	orbitCandidateTimer  float64
	orbitAnchorMinor     float64
	orbitHitCooldown     float64
	orbitEscapeAxis      int
	orbitEscapeDirection float64
	orbitEscapeTimer     float64
}

type statusMessage struct {
	text  string
	timer float64
}

// A feature sample may be long (for example blackhole.wav). Only one voice for
// a given feature is allowed at once; retriggering gently replaces it.
type magicFeatureVoice struct {
	source js.Value
	gain   js.Value
	ended  js.Func
	active bool
}

type paddleVelocitySample struct {
	vx  float64
	age float64
}

// renderSnapshot stores the last completed fixed-step state used for visual
// interpolation. Physics remains authoritative; only drawing is smoothed.
type renderSnapshot struct {
	ballX, ballY, ballAngle                   float64
	secondBallX, secondBallY, secondBallAngle float64
	paddleX                                   float64
	blackHoleX, blackHoleY                    float64
	secondBallActive                          bool
	blackHoleActive                           bool
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

	// Recent fixed-step paddle velocities used only for spin transfer at impact.
	paddleSpinHistory []paddleVelocitySample

	hudLivesValue int
	hudLevelValue int
	hudScoreValue int
	hudLivesText  string
	hudLevelText  string
	hudScoreText  string

	// Per-level measured peaks. Speed records the fastest incoming collision
	// speed, before paddle boost or collision response. Spin records the greatest
	// absolute spin reached. Both survive life loss and reset with the level.
	levelMeasuredMaxSpeed float64
	levelMeasuredMaxSpin  float64

	// ---- Level system ----
	currentLevelIndex    int
	highestUnlockedLevel int
	unlockFrontier       int // Highest level index made available; may equal len(levels) after finishing all current levels.
	levels               []levelData

	devAllLevelsUnlocked  bool
	debugOverlayVisible   bool
	physicsOverlayVisible bool

	physicsAccumulator       float64
	physicsStepRateCurrent   float64
	physicsRealtimePercent   float64
	physicsComputeLoad       float64
	physicsSampleElapsed     float64
	physicsSampleSteps       int
	physicsSampleComputeTime float64
	physicsSampleDroppedTime float64
	physicsDroppedTimeTotal  float64
	physicsWarningTimer      float64
	physicsLastFrameSteps    int
	physicsPeakFrameSteps    int

	previousRenderSnapshot   renderSnapshot
	renderSnapshotReady      bool
	renderInterpolationAlpha float64

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
	audioBrickBus    js.Value
	audioMagicBus    js.Value
	audioSynthBus    js.Value
	audioInitialized bool

	// Reusable per-level room-effects graph. The three mixer buses feed
	// audioRoomInput; its dry and processed wet branches reunite at audioMaster.
	audioRoomInput          js.Value
	audioRoomDryGain        js.Value
	audioRoomWetGain        js.Value
	audioRoomInputFilter    js.Value
	audioRoomOutputFilter   js.Value
	audioRoomDelay1         js.Value
	audioRoomDelay2         js.Value
	audioRoomDelay3         js.Value
	audioRoomTapGain1       js.Value
	audioRoomTapGain2       js.Value
	audioRoomTapGain3       js.Value
	audioRoomWetSum         js.Value
	audioRoomFeedbackFilter js.Value
	audioRoomFeedbackGain   js.Value

	brickHitBuffers   []js.Value
	brickHitLastIndex = -1

	magicFeatureBuffers      = make(map[string]js.Value)
	magicFeatureVoices       = make(map[string]*magicFeatureVoice)
	magicFeatureActiveVoices int

	// Embedded compressed audio is read and decoded one file at a time in the
	// background. Empty/not-yet-ready sample banks use the existing synth
	// fallback immediately, so gameplay never waits for audio decoding.
	embeddedAudioCallbacks      []js.Func
	embeddedAudioPreloadStarted bool
	embeddedAudioQueue          []embeddedAudioSample
	embeddedAudioLoadActive     bool
	embeddedAudioMagicFeatures  = make(map[string]bool)
	embeddedAudioLoadedCount    int
	embeddedAudioFailedCount    int

	brickHitActiveVoices    int
	brickHitStartsThisFrame int
)

type embeddedAudioSample struct {
	path    string
	label   string
	feature string
	magic   bool
}

type brick struct {
	x, y, w, h  float64
	row, col    int
	tiltRadians float64
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

// brickMicroTiltRadians returns a deterministic random-looking angle. It does
// not use math/rand, so brick construction cannot disturb power-up or gameplay
// randomness. Re-entering the same level produces the same brick angles.
func brickMicroTiltRadians(levelIndex, row, col int) float64 {
	hash := uint32(levelIndex+1)*0x9e3779b9 ^
		uint32(row+1)*0x85ebca6b ^
		uint32(col+1)*0xc2b2ae35
	hash ^= hash >> 16
	hash *= 0x7feb352d
	hash ^= hash >> 15
	hash *= 0x846ca68b
	hash ^= hash >> 16

	unit := float64(hash&0x00ffffff) / float64(0x00ffffff)
	magnitudeDegrees := brickTiltMinDegrees +
		unit*(brickTiltMaxDegrees-brickTiltMinDegrees)
	if hash&0x80000000 != 0 {
		magnitudeDegrees = -magnitudeDegrees
	}
	return magnitudeDegrees * math.Pi / 180.0
}

func rotateVector(x, y, angle float64) (float64, float64) {
	cosAngle := math.Cos(angle)
	sinAngle := math.Sin(angle)
	return x*cosAngle - y*sinAngle, x*sinAngle + y*cosAngle
}

func wallNoiseHash(cell, wallID int) float64 {
	hash := uint32(cell+0x40000000)*0x9e3779b9 ^
		uint32(currentLevelIndex+1)*0x85ebca6b ^
		uint32(wallID)*0xc2b2ae35
	hash ^= hash >> 16
	hash *= 0x7feb352d
	hash ^= hash >> 15
	hash *= 0x846ca68b
	hash ^= hash >> 16
	return float64(hash&0x00ffffff)/float64(0x00ffffff)*2 - 1
}

// wallNoise returns smooth deterministic one-dimensional value noise.
func wallNoise(position float64, wallID int) float64 {
	if wallNoiseCellSize <= 0 {
		return 0
	}

	x := position / wallNoiseCellSize
	cell := int(math.Floor(x))
	t := x - float64(cell)
	t = t * t * (3 - 2*t)
	a := wallNoiseHash(cell, wallID)
	b := wallNoiseHash(cell+1, wallID)
	return a + (b-a)*t
}

func wallCornerFade(position, wallLength float64) float64 {
	if wallCornerFadeDistance <= 0 || wallLength <= 0 {
		return 1
	}

	edgeDistance := math.Min(position, wallLength-position)
	t := clampFloat(edgeDistance/wallCornerFadeDistance, 0, 1)
	return t * t * (3 - 2*t)
}

func roughWallNormal(
	nx, ny, position, wallLength float64,
	wallID int,
	maximumTiltDegrees float64,
) (float64, float64) {
	angleDegrees := wallNoise(position, wallID) * maximumTiltDegrees
	angleDegrees *= wallCornerFade(position, wallLength)
	return rotateVector(nx, ny, angleDegrees*math.Pi/180)
}

const (
	orbitAxisNone = iota
	orbitAxisVertical
	orbitAxisHorizontal
)

func resetFastOrbitCandidate(b *Ball) {
	b.orbitCandidateAxis = orbitAxisNone
	b.orbitCandidateHits = 0
	b.orbitCandidateTimer = 0
	b.orbitAnchorMinor = 0
}

func resetFastOrbitState(b *Ball) {
	resetFastOrbitCandidate(b)
	b.orbitHitCooldown = 0
	b.orbitEscapeAxis = orbitAxisNone
	b.orbitEscapeDirection = 0
	b.orbitEscapeTimer = 0
}

func fastOrbitAxis(b *Ball) int {
	speed := math.Hypot(b.vx, b.vy)
	if speed < physicsOrbitMinimumSpeed {
		return orbitAxisNone
	}

	minorLimit := math.Max(physicsOrbitMinorSpeedFloor, speed*physicsOrbitMinorSpeedRatio)
	nearVertical := math.Abs(b.vx) <= minorLimit
	nearHorizontal := math.Abs(b.vy) <= minorLimit
	if nearVertical && !nearHorizontal {
		return orbitAxisVertical
	}
	if nearHorizontal && !nearVertical {
		return orbitAxisHorizontal
	}
	return orbitAxisNone
}

func orbitMinorPosition(b *Ball, axis int) float64 {
	if axis == orbitAxisVertical {
		return b.x
	}
	return b.y
}

func chooseOrbitEscapeDirection(b *Ball, axis int) float64 {
	minorVelocity := b.vy
	if axis == orbitAxisVertical {
		minorVelocity = b.vx
	}
	if math.Abs(minorVelocity) >= 1 {
		return sign(minorVelocity)
	}
	if math.Abs(b.omega) >= 0.01 {
		return sign(b.omega)
	}
	if axis == orbitAxisVertical {
		direction := sign(canvasWidth/2 - b.x)
		if direction != 0 {
			return direction
		}
	} else {
		direction := sign(canvasHeight/2 - b.y)
		if direction != 0 {
			return direction
		}
	}
	return 1
}

func enforceFastOrbitEscape(b *Ball) {
	if b.orbitEscapeTimer <= 0 || b.orbitEscapeAxis == orbitAxisNone {
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	if speed <= 0 {
		return
	}
	minorSpeed := math.Min(physicsOrbitEscapeSpeed, speed*0.35)
	majorSpeed := math.Sqrt(math.Max(0, speed*speed-minorSpeed*minorSpeed))

	if b.orbitEscapeAxis == orbitAxisVertical {
		majorDirection := sign(b.vy)
		if majorDirection == 0 {
			majorDirection = -1
		}
		b.vx = b.orbitEscapeDirection * minorSpeed
		b.vy = majorDirection * majorSpeed
		return
	}

	majorDirection := sign(b.vx)
	if majorDirection == 0 {
		majorDirection = 1
	}
	b.vx = majorDirection * majorSpeed
	b.vy = b.orbitEscapeDirection * minorSpeed
}

func activateFastOrbitEscape(b *Ball, axis int) {
	b.orbitEscapeAxis = axis
	b.orbitEscapeDirection = chooseOrbitEscapeDirection(b, axis)
	b.orbitEscapeTimer = physicsOrbitEscapeDuration
	resetFastOrbitCandidate(b)
	enforceFastOrbitEscape(b)
	showStatusUnique("Orbital tilt!", physicsOrbitMessageDuration)
}

func recordUnbreakableOrbitHit(b *Ball) {
	if b.orbitHitCooldown > 0 || b.orbitEscapeTimer > 0 {
		return
	}

	axis := fastOrbitAxis(b)
	if axis == orbitAxisNone {
		resetFastOrbitCandidate(b)
		return
	}

	minorPosition := orbitMinorPosition(b, axis)
	newCandidate := b.orbitCandidateAxis != axis ||
		b.orbitCandidateHits == 0 ||
		b.orbitCandidateTimer > physicsOrbitDetectionWindow ||
		math.Abs(minorPosition-b.orbitAnchorMinor) > physicsOrbitMaximumMinorProgress

	if newCandidate {
		b.orbitCandidateAxis = axis
		b.orbitCandidateHits = 1
		b.orbitCandidateTimer = 0
		b.orbitAnchorMinor = minorPosition
	} else {
		b.orbitCandidateHits++
	}
	b.orbitHitCooldown = physicsOrbitHitCooldown

	if b.orbitCandidateHits >= physicsOrbitRequiredHits {
		activateFastOrbitEscape(b, axis)
	}
}

func updateFastOrbitDetector(b *Ball, dt float64) {
	if b.orbitHitCooldown > 0 {
		b.orbitHitCooldown = math.Max(0, b.orbitHitCooldown-dt)
	}
	if b.orbitCandidateHits > 0 {
		b.orbitCandidateTimer += dt
		if b.orbitCandidateTimer > physicsOrbitDetectionWindow {
			resetFastOrbitCandidate(b)
		}
	}
	if b.orbitEscapeTimer > 0 {
		b.orbitEscapeTimer = math.Max(0, b.orbitEscapeTimer-dt)
		enforceFastOrbitEscape(b)
		if b.orbitEscapeTimer == 0 {
			b.orbitEscapeAxis = orbitAxisNone
			b.orbitEscapeDirection = 0
		}
	}
}

// preventVerticalLock preserves total
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

func resetPaddleSpinHistory() {
	paddleSpinHistory = paddleSpinHistory[:0]
}

func updatePaddleSpinHistory(dt float64) {
	if dt <= 0 || paddleSpinGraceSeconds <= 0 {
		resetPaddleSpinHistory()
		return
	}

	kept := paddleSpinHistory[:0]
	for _, sample := range paddleSpinHistory {
		sample.age += dt
		if sample.age <= paddleSpinGraceSeconds {
			kept = append(kept, sample)
		}
	}
	paddleSpinHistory = append(kept, paddleVelocitySample{vx: paddle.vx})
}

func effectivePaddleSpinVelocity() float64 {
	best := paddle.vx
	for _, sample := range paddleSpinHistory {
		if sample.age < 0 || sample.age > paddleSpinGraceSeconds {
			continue
		}
		weight := 1 - sample.age/paddleSpinGraceSeconds
		candidate := sample.vx * weight
		if math.Abs(candidate) > math.Abs(best) {
			best = candidate
		}
	}
	return best
}

func showStatus(text string, duration float64) {
	statusMessages = append(statusMessages, statusMessage{text: text, timer: duration})
	if len(statusMessages) > statusMessageLimit {
		statusMessages = statusMessages[len(statusMessages)-statusMessageLimit:]
	}
}

func showStatusUnique(text string, duration float64) {
	for i := range statusMessages {
		if statusMessages[i].text == text {
			statusMessages[i].timer = duration
			return
		}
	}
	showStatus(text, duration)
}

// Keep a single High spin! entry in the existing top-left status stack.
// A new qualifying paddle hit refreshes its timer instead of adding duplicates.
func showHighSpinStatus() {
	if !enableHighSpinMessage {
		return
	}

	showStatusUnique("High spin!", highSpinMessageDuration)
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

func recordMeasuredBallSpin(b *Ball) {
	if b == nil {
		return
	}

	levelMeasuredMaxSpin = math.Max(levelMeasuredMaxSpin, math.Abs(b.omega))
}

func recordIncomingCollisionSpeed(speed float64) {
	levelMeasuredMaxSpeed = math.Max(levelMeasuredMaxSpeed, speed)
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
		unlockAudioFromGesture()
		showStatus("Sound on", 2.0)
	} else {
		stopAllMagicFeatureVoices()
		if audioInitialized && !audioMaster.IsUndefined() && !audioMaster.IsNull() {
			audioMaster.Get("gain").Call("cancelScheduledValues", audioCtx.Get("currentTime").Float())
			audioMaster.Get("gain").Set("value", 0)
		}
		showStatus("Sound off", 2.0)
	}
}

// ---- Audio (non-blocking, scheduled via Web Audio) ----

func supportedCompressedAudioFile(name string) bool {
	name = strings.TrimSpace(name)

	switch strings.ToLower(filepathExtension(name)) {
	case ".mp3", ".ogg", ".oga", ".opus", ".webm", ".m4a", ".aac", ".flac":
		return true
	default:
		// WAV is intentionally excluded. WAV masters belong outside runtime/ and
		// are never read or decoded by this loader.
		return false
	}
}

func filepathExtension(name string) string {
	index := strings.LastIndexByte(name, '.')
	if index < 0 {
		return ""
	}
	return name[index:]
}

func keepEmbeddedAudioCallback(callback js.Func) {
	embeddedAudioCallbacks = append(embeddedAudioCallbacks, callback)
}

func queueEmbeddedAudioDirectory(directory, label string, magic bool) {
	entries, err := fs.ReadDir(embeddedRuntimeAudio, directory)
	if err != nil {
		log("Could not read embedded " + label + " directory " + directory + ": " + err.Error())
		return
	}

	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !supportedCompressedAudioFile(entry.Name()) {
			continue
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)

	for _, fileName := range names {
		sample := embeddedAudioSample{
			path:  directory + "/" + fileName,
			label: label,
			magic: magic,
		}

		if magic {
			sample.feature = magicFeatureNameFromFilename(fileName)
			if sample.feature == "" || embeddedAudioMagicFeatures[sample.feature] {
				continue
			}
			embeddedAudioMagicFeatures[sample.feature] = true
		}

		embeddedAudioQueue = append(embeddedAudioQueue, sample)
	}
}

func finishEmbeddedAudioSampleLoad(sample embeddedAudioSample, loaded bool) {
	if loaded {
		embeddedAudioLoadedCount++
	} else {
		embeddedAudioFailedCount++
	}
	embeddedAudioLoadActive = false
	scheduleNextEmbeddedAudioLoad()
}

func decodeEmbeddedAudio(sample embeddedAudioSample) {
	data, err := embeddedRuntimeAudio.ReadFile(sample.path)
	if err != nil {
		log("Could not read embedded " + sample.label + " sample " + sample.path + ": " + err.Error())
		finishEmbeddedAudioSampleLoad(sample, false)
		return
	}
	if len(data) == 0 {
		log("Embedded " + sample.label + " sample was empty: " + sample.path)
		finishEmbeddedAudioSampleLoad(sample, false)
		return
	}

	byteArray := js.Global().Get("Uint8Array").New(len(data))
	js.CopyBytesToJS(byteArray, data)

	var decodeSuccessCallback js.Func
	var decodeFailureCallback js.Func
	finished := false
	finish := func(loaded bool) {
		if finished {
			return
		}
		finished = true
		finishEmbeddedAudioSampleLoad(sample, loaded)
	}

	decodeSuccessCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if len(args) == 0 || args[0].IsUndefined() || args[0].IsNull() {
			log("Decoded embedded " + sample.label + " sample was empty: " + sample.path)
			finish(false)
			return nil
		}

		if sample.magic {
			magicFeatureBuffers[sample.feature] = args[0]
		} else {
			brickHitBuffers = append(brickHitBuffers, args[0])
		}
		finish(true)
		return nil
	})

	decodeFailureCallback = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		reason := "decode failed"
		if len(args) > 0 {
			reason = fmt.Sprint(args[0])
		}
		log("Could not decode embedded " + sample.label + " sample " + sample.path + ": " + reason)
		finish(false)
		return nil
	})

	keepEmbeddedAudioCallback(decodeSuccessCallback)
	keepEmbeddedAudioCallback(decodeFailureCallback)

	defer func() {
		if recovered := recover(); recovered != nil {
			log("Could not start decoding embedded " + sample.label + " sample " + sample.path + ": " + fmt.Sprint(recovered))
			finish(false)
		}
	}()
	audioCtx.Call("decodeAudioData", byteArray.Get("buffer"), decodeSuccessCallback, decodeFailureCallback)
}

func scheduleNextEmbeddedAudioLoad() {
	if embeddedAudioLoadActive {
		return
	}
	if len(embeddedAudioQueue) == 0 {
		if embeddedAudioPreloadStarted {
			log(fmt.Sprintf(
				"Embedded audio preload finished: %d loaded, %d failed; brick=%d magic=%d",
				embeddedAudioLoadedCount,
				embeddedAudioFailedCount,
				len(brickHitBuffers),
				len(magicFeatureBuffers),
			))
		}
		return
	}

	sample := embeddedAudioQueue[0]
	embeddedAudioQueue = embeddedAudioQueue[1:]
	embeddedAudioLoadActive = true

	timerCallback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if !audioInitialized || audioCtx.IsUndefined() || audioCtx.IsNull() {
			finishEmbeddedAudioSampleLoad(sample, false)
			return nil
		}
		decodeEmbeddedAudio(sample)
		return nil
	})
	keepEmbeddedAudioCallback(timerCallback)
	js.Global().Call("setTimeout", timerCallback, embeddedAudioDecodeYieldMS)
}

func startEmbeddedAudioPreload() {
	if embeddedAudioPreloadStarted || !audioInitialized {
		return
	}

	embeddedAudioPreloadStarted = true
	embeddedAudioLoadedCount = 0
	embeddedAudioFailedCount = 0
	embeddedAudioQueue = embeddedAudioQueue[:0]
	clear(embeddedAudioMagicFeatures)
	brickHitBuffers = brickHitBuffers[:0]
	brickHitLastIndex = -1
	clear(magicFeatureBuffers)

	queueEmbeddedAudioDirectory(brickHitSampleDirectory, "normal-brick hit", false)
	queueEmbeddedAudioDirectory(magicFeatureSampleDirectory, "magic-feature", true)
	sort.Slice(embeddedAudioQueue, func(i, j int) bool {
		return embeddedAudioQueue[i].path < embeddedAudioQueue[j].path
	})

	if len(embeddedAudioQueue) == 0 {
		log("No embedded compressed runtime audio found; using synthesized fallbacks")
		return
	}

	log(fmt.Sprintf("Embedded compressed audio queued: %d file(s)", len(embeddedAudioQueue)))
	scheduleNextEmbeddedAudioLoad()
}

func magicFeatureNameFromFilename(name string) string {
	name = strings.TrimSpace(name)
	if slash := strings.LastIndexByte(name, '/'); slash >= 0 {
		name = name[slash+1:]
	}
	extension := filepathExtension(name)
	return strings.ToLower(strings.TrimSpace(strings.TrimSuffix(name, extension)))
}

type audioRoomPreset struct {
	name       string
	defaultDry float64

	inputFilterType string
	inputFrequency  float64
	inputQ          float64

	outputFilterType string
	outputFrequency  float64
	outputQ          float64

	delay1, delay2, delay3       float64
	tapGain1, tapGain2, tapGain3 float64
	feedback                     float64
}

func normalizeAudioRoomName(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	replacer := strings.NewReplacer("_", "", "-", "", " ", "")
	name = replacer.Replace(name)
	switch name {
	case "", "none", "off", "dry", "disabled":
		return "none"
	case "smallroom", "roomsmall":
		return "smallroom"
	case "bigopenroom", "openroom", "bigroom", "roomopen":
		return "bigopenroom"
	case "smallhall", "hallsmall":
		return "smallhall"
	case "bighall", "largehall", "hallbig":
		return "bighall"
	case "underwater", "water":
		return "underwater"
	case "space":
		return "space"
	case "cave", "cavern":
		return "cave"
	case "matrix", "digital":
		return "matrix"
	default:
		return name
	}
}

func audioRoomPresetForName(name string) (audioRoomPreset, bool) {
	name = normalizeAudioRoomName(name)
	switch name {
	case "none":
		return audioRoomPreset{
			name: "none", defaultDry: 1,
			inputFilterType: "lowpass", inputFrequency: 20000, inputQ: 0.0001,
			outputFilterType: "lowpass", outputFrequency: 20000, outputQ: 0.0001,
			delay1: 0.01, delay2: 0.02, delay3: 0.04,
		}, true
	case "smallroom":
		return audioRoomPreset{
			name: "smallroom", defaultDry: 0.78,
			inputFilterType: "highpass", inputFrequency: 80, inputQ: 0.35,
			outputFilterType: "lowpass", outputFrequency: 9000, outputQ: 0.45,
			delay1: 0.011, delay2: 0.023, delay3: 0.041,
			tapGain1: 0.40, tapGain2: 0.28, tapGain3: 0.20, feedback: 0.08,
		}, true
	case "bigopenroom":
		return audioRoomPreset{
			name: "bigopenroom", defaultDry: 0.82,
			inputFilterType: "highpass", inputFrequency: 70, inputQ: 0.30,
			outputFilterType: "lowpass", outputFrequency: 11000, outputQ: 0.35,
			delay1: 0.070, delay2: 0.150, delay3: 0.310,
			tapGain1: 0.28, tapGain2: 0.20, tapGain3: 0.15, feedback: 0.08,
		}, true
	case "smallhall":
		return audioRoomPreset{
			name: "smallhall", defaultDry: 0.68,
			inputFilterType: "highpass", inputFrequency: 90, inputQ: 0.35,
			outputFilterType: "lowpass", outputFrequency: 7000, outputQ: 0.50,
			delay1: 0.028, delay2: 0.061, delay3: 0.115,
			tapGain1: 0.38, tapGain2: 0.30, tapGain3: 0.24, feedback: 0.22,
		}, true
	case "bighall":
		return audioRoomPreset{
			name: "bighall", defaultDry: 0.55,
			inputFilterType: "highpass", inputFrequency: 80, inputQ: 0.35,
			outputFilterType: "lowpass", outputFrequency: 5400, outputQ: 0.55,
			delay1: 0.055, delay2: 0.125, delay3: 0.260,
			tapGain1: 0.34, tapGain2: 0.28, tapGain3: 0.24, feedback: 0.36,
		}, true
	case "cave":
		return audioRoomPreset{
			name: "cave", defaultDry: 0.42,
			inputFilterType: "highpass", inputFrequency: 65, inputQ: 0.30,
			outputFilterType: "lowpass", outputFrequency: 3000, outputQ: 0.65,
			delay1: 0.075, delay2: 0.190, delay3: 0.420,
			tapGain1: 0.36, tapGain2: 0.30, tapGain3: 0.26, feedback: 0.50,
		}, true
	case "space":
		return audioRoomPreset{
			name: "space", defaultDry: 0.58,
			inputFilterType: "highpass", inputFrequency: 160, inputQ: 0.40,
			outputFilterType: "lowpass", outputFrequency: 7000, outputQ: 0.35,
			delay1: 0.110, delay2: 0.290, delay3: 0.520,
			tapGain1: 0.22, tapGain2: 0.18, tapGain3: 0.14, feedback: 0.26,
		}, true
	case "matrix":
		return audioRoomPreset{
			name: "matrix", defaultDry: 0.60,
			inputFilterType: "bandpass", inputFrequency: 1800, inputQ: 2.80,
			outputFilterType: "lowpass", outputFrequency: 6500, outputQ: 1.20,
			delay1: 0.011, delay2: 0.023, delay3: 0.047,
			tapGain1: 0.32, tapGain2: 0.27, tapGain3: 0.22, feedback: 0.58,
		}, true
	case "underwater":
		return audioRoomPreset{
			name: "underwater", defaultDry: 0.10,
			inputFilterType: "lowpass", inputFrequency: 600, inputQ: 0.80,
			outputFilterType: "lowpass", outputFrequency: 750, outputQ: 0.55,
			delay1: 0.008, delay2: 0.022, delay3: 0.041,
			tapGain1: 0.40, tapGain2: 0.30, tapGain3: 0.23, feedback: 0.18,
		}, true
	default:
		return audioRoomPreset{}, false
	}
}

func smoothAudioParam(param js.Value, value, now float64) {
	if param.IsUndefined() || param.IsNull() {
		return
	}
	defer func() { _ = recover() }()
	current := param.Get("value").Float()
	param.Call("cancelScheduledValues", now)
	param.Call("setValueAtTime", current, now)
	param.Call("linearRampToValueAtTime", value, now+audioRoomTransitionSeconds)
}

func applyAudioRoomSettings() {
	if !audioInitialized || audioCtx.IsUndefined() || audioCtx.IsNull() {
		return
	}

	preset, found := audioRoomPresetForName(currentAudioRoom)
	if !found {
		log("Unknown audioRoom preset: " + currentAudioRoom + "; using none")
		preset, _ = audioRoomPresetForName("none")
		currentAudioRoom = "none"
	}

	dry := preset.defaultDry
	if currentAudioRoomDry >= 0 {
		dry = clampFloat(currentAudioRoomDry, 0, 1)
	}
	if preset.name == "none" {
		dry = 1
	}
	wet := 1 - dry

	now := audioCtx.Get("currentTime").Float()

	// Filter type changes are instantaneous; frequency, Q, delay, gain, and
	// feedback are ramped to avoid clicks while moving between levels.
	audioRoomInputFilter.Set("type", preset.inputFilterType)
	audioRoomOutputFilter.Set("type", preset.outputFilterType)
	audioRoomFeedbackFilter.Set("type", "lowpass")

	smoothAudioParam(audioRoomInputFilter.Get("frequency"), preset.inputFrequency, now)
	smoothAudioParam(audioRoomInputFilter.Get("Q"), preset.inputQ, now)
	smoothAudioParam(audioRoomOutputFilter.Get("frequency"), preset.outputFrequency, now)
	smoothAudioParam(audioRoomOutputFilter.Get("Q"), preset.outputQ, now)
	smoothAudioParam(audioRoomFeedbackFilter.Get("frequency"), math.Min(preset.outputFrequency, 5000), now)
	smoothAudioParam(audioRoomFeedbackFilter.Get("Q"), 0.45, now)

	smoothAudioParam(audioRoomDelay1.Get("delayTime"), preset.delay1, now)
	smoothAudioParam(audioRoomDelay2.Get("delayTime"), preset.delay2, now)
	smoothAudioParam(audioRoomDelay3.Get("delayTime"), preset.delay3, now)
	smoothAudioParam(audioRoomTapGain1.Get("gain"), preset.tapGain1, now)
	smoothAudioParam(audioRoomTapGain2.Get("gain"), preset.tapGain2, now)
	smoothAudioParam(audioRoomTapGain3.Get("gain"), preset.tapGain3, now)
	smoothAudioParam(audioRoomFeedbackGain.Get("gain"), clampFloat(preset.feedback, 0, 0.75), now)
	smoothAudioParam(audioRoomDryGain.Get("gain"), dry, now)
	smoothAudioParam(audioRoomWetGain.Get("gain"), wet, now)

	log(fmt.Sprintf("Audio room: %s (dry %.2f, wet %.2f)", preset.name, dry, wet))
}

func initAudio() {
	if audioInitialized || !enableSounds {
		return
	}

	defer func() {
		if r := recover(); r != nil {
			audioInitialized = false
			audioCtx = js.Undefined()
			audioMaster = js.Undefined()
			audioBrickBus = js.Undefined()
			audioMagicBus = js.Undefined()
			audioSynthBus = js.Undefined()
			audioRoomInput = js.Undefined()
			audioRoomDryGain = js.Undefined()
			audioRoomWetGain = js.Undefined()
			audioRoomInputFilter = js.Undefined()
			audioRoomOutputFilter = js.Undefined()
			audioRoomDelay1 = js.Undefined()
			audioRoomDelay2 = js.Undefined()
			audioRoomDelay3 = js.Undefined()
			audioRoomTapGain1 = js.Undefined()
			audioRoomTapGain2 = js.Undefined()
			audioRoomTapGain3 = js.Undefined()
			audioRoomWetSum = js.Undefined()
			audioRoomFeedbackFilter = js.Undefined()
			audioRoomFeedbackGain = js.Undefined()
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
	audioBrickBus = audioCtx.Call("createGain")
	audioMagicBus = audioCtx.Call("createGain")
	audioSynthBus = audioCtx.Call("createGain")

	audioRoomInput = audioCtx.Call("createGain")
	audioRoomDryGain = audioCtx.Call("createGain")
	audioRoomWetGain = audioCtx.Call("createGain")
	audioRoomInputFilter = audioCtx.Call("createBiquadFilter")
	audioRoomOutputFilter = audioCtx.Call("createBiquadFilter")
	audioRoomDelay1 = audioCtx.Call("createDelay", 2.0)
	audioRoomDelay2 = audioCtx.Call("createDelay", 2.0)
	audioRoomDelay3 = audioCtx.Call("createDelay", 2.0)
	audioRoomTapGain1 = audioCtx.Call("createGain")
	audioRoomTapGain2 = audioCtx.Call("createGain")
	audioRoomTapGain3 = audioCtx.Call("createGain")
	audioRoomWetSum = audioCtx.Call("createGain")
	audioRoomFeedbackFilter = audioCtx.Call("createBiquadFilter")
	audioRoomFeedbackGain = audioCtx.Call("createGain")

	// Start fully dry and silent on the wet branch. The selected level preset
	// then fades in from this safe state, so an omitted audioRoom can never leak
	// the default Biquad/Delay settings during initial audio startup.
	audioRoomDryGain.Get("gain").Set("value", 1)
	audioRoomWetGain.Get("gain").Set("value", 0)
	audioRoomTapGain1.Get("gain").Set("value", 0)
	audioRoomTapGain2.Get("gain").Set("value", 0)
	audioRoomTapGain3.Get("gain").Set("value", 0)
	audioRoomDelay1.Get("delayTime").Set("value", 0.01)
	audioRoomDelay2.Get("delayTime").Set("value", 0.02)
	audioRoomDelay3.Get("delayTime").Set("value", 0.04)
	audioRoomFeedbackGain.Get("gain").Set("value", 0)

	audioBrickBus.Call("connect", audioRoomInput)
	audioMagicBus.Call("connect", audioRoomInput)
	audioSynthBus.Call("connect", audioRoomInput)

	// Completely dry branch.
	audioRoomInput.Call("connect", audioRoomDryGain)
	audioRoomDryGain.Call("connect", audioMaster)

	// Three reflection taps plus a damped feedback tail form the wet branch.
	audioRoomInput.Call("connect", audioRoomInputFilter)
	audioRoomInputFilter.Call("connect", audioRoomDelay1)
	audioRoomInputFilter.Call("connect", audioRoomDelay2)
	audioRoomInputFilter.Call("connect", audioRoomDelay3)
	audioRoomDelay1.Call("connect", audioRoomTapGain1)
	audioRoomDelay2.Call("connect", audioRoomTapGain2)
	audioRoomDelay3.Call("connect", audioRoomTapGain3)
	audioRoomTapGain1.Call("connect", audioRoomWetSum)
	audioRoomTapGain2.Call("connect", audioRoomWetSum)
	audioRoomTapGain3.Call("connect", audioRoomWetSum)
	audioRoomWetSum.Call("connect", audioRoomOutputFilter)
	audioRoomOutputFilter.Call("connect", audioRoomWetGain)
	audioRoomWetGain.Call("connect", audioMaster)

	// DelayNode in the cycle makes this legal Web Audio feedback routing.
	audioRoomDelay3.Call("connect", audioRoomFeedbackFilter)
	audioRoomFeedbackFilter.Call("connect", audioRoomFeedbackGain)
	audioRoomFeedbackGain.Call("connect", audioRoomDelay3)

	audioMaster.Call("connect", audioCtx.Get("destination"))

	audioInitialized = true
	setAudioMixerGains()
	applyAudioRoomSettings()
	startEmbeddedAudioPreload()
	log("Audio graph initialized; embedded compressed samples are preloading")
}

func setAudioMixerGains() {
	if !audioInitialized {
		return
	}

	if !audioMaster.IsUndefined() && !audioMaster.IsNull() {
		audioMaster.Get("gain").Set("value", audioMixerMaster)
	}
	if !audioBrickBus.IsUndefined() && !audioBrickBus.IsNull() {
		audioBrickBus.Get("gain").Set("value", audioMixerBricks)
	}
	if !audioMagicBus.IsUndefined() && !audioMagicBus.IsNull() {
		audioMagicBus.Get("gain").Set("value", audioMixerMagic)
	}
	if !audioSynthBus.IsUndefined() && !audioSynthBus.IsNull() {
		audioSynthBus.Get("gain").Set("value", audioMixerSynths)
	}
}

func ensureAudioRunning() {
	if !enableSounds ||
		!audioInitialized || audioCtx.IsUndefined() || audioCtx.IsNull() {
		return
	}
	setAudioMixerGains()
	state := audioCtx.Get("state")
	if state.Type() == js.TypeString && state.String() == "suspended" {
		audioCtx.Call("resume")
	}
}

func unlockAudioFromGesture() {
	if !enableSounds {
		return
	}
	if !audioInitialized {
		initAudio()
	}
	ensureAudioRunning()
}

func scheduleAudioPreparation() {
	if !enableSounds || audioInitialized {
		return
	}

	callback := js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if enableSounds && !audioInitialized {
			initAudio()
		}
		return nil
	})
	keepEmbeddedAudioCallback(callback)
	js.Global().Call("setTimeout", callback, 0)
}

func brickCenterX(br *brick) float64 {
	if br == nil {
		return canvasWidth / 2
	}
	return br.x + br.w/2
}

func audioPanFromX(hitX float64) float64 {
	if canvasWidth <= 0 {
		return 0
	}
	pan := hitX/canvasWidth*2 - 1
	return clampFloat(pan, -brickSoundPanLimit, brickSoundPanLimit)
}

// Connect an audio node through a StereoPannerNode and then into its mixer bus.
// Browsers without stereo-panner support fall back to a centered bus connection.
func connectAudioNodePanned(node, destination js.Value, hitX, when float64) {
	connected := false
	defer func() {
		if recover() != nil && !connected {
			defer func() { _ = recover() }()
			node.Call("connect", destination)
		}
	}()

	createPanner := audioCtx.Get("createStereoPanner")
	if createPanner.Type() != js.TypeFunction {
		node.Call("connect", destination)
		return
	}

	panner := audioCtx.Call("createStereoPanner")
	panner.Get("pan").Call("setValueAtTime", audioPanFromX(hitX), when)
	node.Call("connect", panner)
	panner.Call("connect", destination)
	connected = true
}

func scheduleTone(
	oscType string,
	startFreq float64,
	endFreq float64,
	duration float64,
	volume float64,
	delay float64,
) {
	scheduleTonePanned(oscType, startFreq, endFreq, duration, volume, delay, canvasWidth/2)
}

func scheduleTonePanned(
	oscType string,
	startFreq float64,
	endFreq float64,
	duration float64,
	volume float64,
	delay float64,
	hitX float64,
) {
	if !audioInitialized || !enableSounds ||
		audioCtx.IsNull() || audioCtx.IsUndefined() ||
		audioMaster.IsNull() || audioMaster.IsUndefined() ||
		audioSynthBus.IsNull() || audioSynthBus.IsUndefined() {
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
	connectAudioNodePanned(gain, audioSynthBus, hitX, start)
	osc.Call("start", start)
	osc.Call("stop", stop+0.02)
}

func scheduleChord(freqs []float64, oscType string, duration, volume, delay float64) {
	scheduleChordPanned(freqs, oscType, duration, volume, delay, canvasWidth/2)
}

func scheduleChordPanned(freqs []float64, oscType string, duration, volume, delay, hitX float64) {
	if len(freqs) == 0 {
		return
	}
	perVoice := volume / float64(len(freqs))
	for i, freq := range freqs {
		detune := 1.0 + float64(i)*0.002
		scheduleTonePanned(oscType, freq*detune, freq*0.98, duration, perVoice, delay, hitX)
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

func playSynthBrickBreakPitched(pitchScale, hitX float64) {
	if pitchScale <= 0 {
		pitchScale = 1
	}
	scheduleTonePanned("square", varyFreq(520*pitchScale, 0.12), varyFreq(360*pitchScale, 0.12), audioRand(0.045, 0.065), audioRand(0.07, 0.11), 0, hitX)
	scheduleTonePanned("triangle", varyFreq(760*pitchScale, 0.10), varyFreq(520*pitchScale, 0.10), audioRand(0.032, 0.05), audioRand(0.035, 0.06), audioRand(0.005, 0.012), hitX)
}

func playSynthBrickBreak(hitX float64) {
	playSynthBrickBreakPitched(1, hitX)
}

func chooseSampleIndex(buffers []js.Value, lastIndex int) int {
	count := len(buffers)
	if count <= 1 {
		return count - 1
	}
	if lastIndex < 0 || lastIndex >= count {
		return rand.Intn(count)
	}

	index := rand.Intn(count - 1)
	if index >= lastIndex {
		index++
	}
	return index
}

func playSampledBrickImpact(
	impactSpeed float64,
	hitX float64,
	buffers []js.Value,
	lastIndex *int,
	playbackRateMin float64,
	playbackRateMax float64,
	filterMinHz float64,
	filterMaxHz float64,
	gainMin float64,
	gainMax float64,
	label string,
	fallback func(),
) {
	if !enableSounds {
		return
	}

	samplePlaybackReady := audioInitialized &&
		!audioCtx.IsNull() && !audioCtx.IsUndefined() &&
		!audioMaster.IsNull() && !audioMaster.IsUndefined() &&
		!audioBrickBus.IsNull() && !audioBrickBus.IsUndefined() &&
		len(buffers) > 0

	// The budget is shared by normal and magic bricks. Extra bricks are still
	// destroyed; only excess sounds in a large burst are skipped.
	if brickHitStartsThisFrame >= brickHitMaxStartsPerFrame {
		return
	}
	if samplePlaybackReady && brickHitActiveVoices >= brickHitMaxActiveVoices {
		return
	}
	brickHitStartsThisFrame++

	if !samplePlaybackReady {
		fallback()
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			log(label + " sample playback failed: " + fmt.Sprint(recovered))
			fallback()
		}
	}()

	ensureAudioRunning()

	index := chooseSampleIndex(buffers, *lastIndex)
	if index < 0 || index >= len(buffers) {
		fallback()
		return
	}
	*lastIndex = index

	referenceSpeed := math.Max(physicsConfig.maxSpeed, minimumCollisionSoundSpeed+1)
	strength := clampFloat(
		(impactSpeed-minimumCollisionSoundSpeed)/(referenceSpeed-minimumCollisionSoundSpeed),
		0, 1,
	)
	strength = math.Sqrt(strength)

	now := audioCtx.Get("currentTime").Float()
	source := audioCtx.Call("createBufferSource")
	filter := audioCtx.Call("createBiquadFilter")
	gain := audioCtx.Call("createGain")

	source.Set("buffer", buffers[index])
	source.Get("playbackRate").Call(
		"setValueAtTime",
		audioRand(playbackRateMin, playbackRateMax),
		now,
	)

	filter.Set("type", "lowpass")
	cutoff := filterMinHz + (filterMaxHz-filterMinHz)*strength
	cutoff *= audioRand(0.86, 1.14)
	nyquistMargin := audioCtx.Get("sampleRate").Float() * 0.45
	filter.Get("frequency").Call("setValueAtTime", clampFloat(cutoff, 800, nyquistMargin), now)
	filter.Get("Q").Call("setValueAtTime", audioRand(0.25, 0.85), now)

	volume := gainMin + (gainMax-gainMin)*strength
	volume *= audioRand(0.90, 1.08)
	gain.Get("gain").Call("setValueAtTime", volume, now)

	source.Call("connect", filter)
	filter.Call("connect", gain)
	connectAudioNodePanned(gain, audioBrickBus, hitX, now)

	// Install the completion callback before starting the source, then release
	// the Go callback as soon as this one-shot voice ends.
	var ended js.Func
	ended = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if brickHitActiveVoices > 0 {
			brickHitActiveVoices--
		}
		source.Set("onended", js.Null())
		ended.Release()
		return nil
	})
	source.Set("onended", ended)
	source.Call("start", now)
	brickHitActiveVoices++
}

func playBrickBreakPitched(impactSpeed, pitchScale, hitX float64) {
	if pitchScale <= 0 {
		pitchScale = 1
	}
	playSampledBrickImpact(
		impactSpeed,
		hitX,
		brickHitBuffers,
		&brickHitLastIndex,
		brickHitPlaybackRateMin*pitchScale,
		brickHitPlaybackRateMax*pitchScale,
		brickHitFilterMinHz,
		brickHitFilterMaxHz,
		brickHitGainMin,
		brickHitGainMax,
		"Normal-brick hit",
		func() { playSynthBrickBreakPitched(pitchScale, hitX) },
	)
}

func playBrickBreak(impactSpeed, hitX float64) {
	playBrickBreakPitched(impactSpeed, 1, hitX)
}

func retireMagicFeatureVoice(voice *magicFeatureVoice, now float64) {
	if voice == nil {
		return
	}
	if voice.active {
		voice.active = false
		if magicFeatureActiveVoices > 0 {
			magicFeatureActiveVoices--
		}
	}

	defer func() { _ = recover() }()
	gainParam := voice.gain.Get("gain")
	currentGain := math.Max(gainParam.Get("value").Float(), 0.0001)
	gainParam.Call("cancelScheduledValues", now)
	gainParam.Call("setValueAtTime", currentGain, now)
	gainParam.Call("linearRampToValueAtTime", 0.0001, now+magicFeatureRetriggerFade)
	voice.source.Call("stop", now+magicFeatureRetriggerFade+0.004)
}

func stopMagicFeatureVoice(feature string) {
	feature = strings.ToLower(strings.TrimSpace(feature))
	voice := magicFeatureVoices[feature]
	if voice == nil {
		return
	}
	delete(magicFeatureVoices, feature)

	now := 0.0
	if audioInitialized && !audioCtx.IsNull() && !audioCtx.IsUndefined() {
		now = audioCtx.Get("currentTime").Float()
	}
	retireMagicFeatureVoice(voice, now)
}

func stopAllMagicFeatureVoices() {
	for feature := range magicFeatureVoices {
		stopMagicFeatureVoice(feature)
	}
}

func playMagicFeature(feature string, impactSpeed, pitchScale, hitX float64) {
	if !enableSounds {
		return
	}
	if pitchScale <= 0 {
		pitchScale = 1
	}

	feature = strings.ToLower(strings.TrimSpace(feature))
	buffer, found := magicFeatureBuffers[feature]
	sampleReady := found && audioInitialized &&
		!audioCtx.IsNull() && !audioCtx.IsUndefined() &&
		!audioMaster.IsNull() && !audioMaster.IsUndefined() &&
		!audioMagicBus.IsNull() && !audioMagicBus.IsUndefined()
	if !sampleReady {
		// This is the established generated feature-unlock cue.
		playPowerup(hitX)
		return
	}

	existing := magicFeatureVoices[feature]
	if existing == nil && magicFeatureActiveVoices >= magicFeatureMaxActiveVoices {
		playPowerup(hitX)
		return
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			log("Magic-feature sample playback failed for " + feature + ": " + fmt.Sprint(recovered))
			playPowerup(hitX)
		}
	}()

	ensureAudioRunning()
	now := audioCtx.Get("currentTime").Float()
	start := now
	if existing != nil {
		// A rapid second blackhole (or any same feature) restarts that feature's
		// cue instead of layering another long copy over it.
		retireMagicFeatureVoice(existing, now)
		start += magicFeatureRetriggerFade * 0.55
	}

	referenceSpeed := math.Max(physicsConfig.maxSpeed, minimumCollisionSoundSpeed+1)
	strength := clampFloat(
		(impactSpeed-minimumCollisionSoundSpeed)/(referenceSpeed-minimumCollisionSoundSpeed),
		0, 1,
	)
	strength = math.Sqrt(strength)

	source := audioCtx.Call("createBufferSource")
	filter := audioCtx.Call("createBiquadFilter")
	gain := audioCtx.Call("createGain")

	source.Set("buffer", buffer)
	source.Get("playbackRate").Call(
		"setValueAtTime",
		audioRand(magicFeaturePlaybackRateMin, magicFeaturePlaybackRateMax)*pitchScale,
		start,
	)

	filter.Set("type", "lowpass")
	cutoff := magicFeatureFilterMinHz + (magicFeatureFilterMaxHz-magicFeatureFilterMinHz)*strength
	cutoff *= audioRand(0.90, 1.10)

	nyquistMargin := audioCtx.Get("sampleRate").Float() * 0.45
	maxCutoff := math.Min(magicFeatureHardMaxHz, nyquistMargin)

	filter.Get("frequency").Call(
		"setValueAtTime",
		clampFloat(cutoff, 800, maxCutoff),
		start,
	)

	filter.Get("Q").Call("setValueAtTime", audioRand(0.20, 0.70), start)

	volume := magicFeatureGainMin + (magicFeatureGainMax-magicFeatureGainMin)*strength
	volume *= audioRand(0.94, 1.06)
	gain.Get("gain").Call("setValueAtTime", volume, start)

	source.Call("connect", filter)
	filter.Call("connect", gain)
	connectAudioNodePanned(gain, audioMagicBus, hitX, start)

	voice := &magicFeatureVoice{source: source, gain: gain, active: true}
	var ended js.Func
	ended = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		if voice.active {
			voice.active = false
			if magicFeatureActiveVoices > 0 {
				magicFeatureActiveVoices--
			}
		}
		if current := magicFeatureVoices[feature]; current == voice {
			delete(magicFeatureVoices, feature)
		}
		source.Set("onended", js.Null())
		ended.Release()
		return nil
	})
	voice.ended = ended
	source.Set("onended", ended)
	magicFeatureVoices[feature] = voice
	magicFeatureActiveVoices++
	source.Call("start", start)
}

func playMagicPitched(impactSpeed, pitchScale, hitX float64) {
	_ = impactSpeed
	if !enableSounds {
		return
	}
	if brickHitStartsThisFrame >= brickHitMaxStartsPerFrame {
		return
	}
	brickHitStartsThisFrame++
	playSynthMagicPitched(pitchScale, hitX)
}

func playMagic(impactSpeed, hitX float64) {
	playMagicPitched(impactSpeed, 1, hitX)
}

func playZapperDestroyedBrick(br *brick) {
	if br == nil {
		return
	}

	referenceSpeed := math.Max(physicsConfig.maxSpeed, minimumCollisionSoundSpeed+1)
	impactSpeed := minimumCollisionSoundSpeed +
		(referenceSpeed-minimumCollisionSoundSpeed)*zapperBrickStrength

	hitX := brickCenterX(br)
	if br.magic {
		playMagicPitched(impactSpeed, zapperBrickPitchScale, hitX)
	} else {
		playBrickBreakPitched(impactSpeed, zapperBrickPitchScale, hitX)
	}
}

func playUnbreakable(hitX float64) {
	// Low, rounded impact with small natural variation.
	base := varyFreq(82, 0.10)
	scheduleTonePanned("sine", base, varyFreq(52, 0.08), audioRand(0.09, 0.14), audioRand(0.12, 0.17), 0, hitX)
	scheduleTonePanned("triangle", varyFreq(46, 0.08), varyFreq(34, 0.08), audioRand(0.11, 0.16), audioRand(0.05, 0.08), 0.004, hitX)
}

func playSynthMagicPitched(pitchScale, hitX float64) {
	if pitchScale <= 0 {
		pitchScale = 1
	}
	scheduleTonePanned("sine", varyFreq(660*pitchScale, 0.06), varyFreq(990*pitchScale, 0.06), audioRand(0.09, 0.13), audioRand(0.08, 0.11), 0, hitX)
	scheduleTonePanned("triangle", varyFreq(990*pitchScale, 0.05), varyFreq(1480*pitchScale, 0.05), audioRand(0.11, 0.15), audioRand(0.055, 0.08), audioRand(0.05, 0.075), hitX)
}

func playSynthMagic(hitX float64) {
	playSynthMagicPitched(1, hitX)
}

func playPowerup(hitX float64) {
	root := varyFreq(330, 0.05)
	scheduleTonePanned("triangle", root, root*4/3, audioRand(0.09, 0.12), audioRand(0.08, 0.11), 0, hitX)
	scheduleTonePanned("triangle", root*4/3, root*2, audioRand(0.10, 0.14), audioRand(0.07, 0.10), audioRand(0.07, 0.10), hitX)
	scheduleChordPanned([]float64{root * 2, root * 2.5, root * 3}, "sine", audioRand(0.18, 0.24), audioRand(0.11, 0.15), audioRand(0.14, 0.19), hitX)
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

func playZapper(hitX float64) {
	scheduleTonePanned("sawtooth", varyFreq(920, 0.10), varyFreq(280, 0.10), 0.08, 0.08, 0, hitX)
	scheduleTonePanned("square", varyFreq(1450, 0.08), varyFreq(520, 0.08), 0.045, 0.045, 0.008, hitX)
}

func playLevelComplete() {
	root := varyFreq(440, 0.04)
	scheduleTone("triangle", root, root*1.5, audioRand(0.085, 0.115), audioRand(0.07, 0.09), 0.08)
	scheduleChord([]float64{root * 1.5, root * 1.875}, "sine", audioRand(0.14, 0.19), audioRand(0.09, 0.12), audioRand(0.16, 0.20))
}

// ---- Reset globals to defaults ----
func resetGlobals() {
	physicsConfig = defaultPhysicsSettings()
	canvasWidth = defaultCanvasWidth
	canvasHeight = defaultCanvasHeight
	paddleWidth = defaultPaddleWidth
	paddleHeight = defaultPaddleHeight
	ballRadius = defaultBallRadius
	brickRows = defaultBrickRows
	brickCols = defaultBrickCols
	brickWidth = defaultBrickWidth
	brickHeight = defaultBrickHeight
	brickPadding = defaultBrickPadding
	brickOffsetTop = defaultBrickOffsetTop
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
	currentAudioRoom = "none"
	currentAudioRoomDry = -1
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
		case "audioRoom":
			currentAudioRoom = normalizeAudioRoomName(val)
		case "audioRoomDry":
			if f, err := strconv.ParseFloat(val, 64); err == nil && f >= 0 && f <= 1 {
				currentAudioRoomDry = f
			} else {
				log("audioRoomDry must be between 0 and 1")
			}
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
		case "gravity", "restitution", "frictionCoeff", "paddleBoost", "brickBoost",
			"maxSpeed", "maxSpin", "stuckSpeedThreshold", "stuckDuration",
			"tiltUpSpeed", "tiltSideMin", "tiltSideMax":
			if f, err := strconv.ParseFloat(val, 64); err == nil {
				valid := true
				switch key {
				case "restitution":
					valid = f >= 0 && f <= 1
				case "frictionCoeff", "stuckSpeedThreshold", "stuckDuration",
					"tiltUpSpeed", "tiltSideMin", "tiltSideMax":
					valid = f >= 0
				case "maxSpeed", "maxSpin":
					valid = f > 0
				}
				if valid {
					setPhysicsSetting(&physicsConfig, key, f)
				}
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

// resolveCollisionBallImproved uses the actual contact point and a solid-disk
// tangential impulse. It is used only while physics is selected.
func resolveCollisionBall(b *Ball, nx, ny, surfVx, surfVy, frictionScale float64) (float64, bool) {
	physics := &physicsConfig
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
	vt := translationVt + spinSurfaceSpeed*physicsCollisionSpinCoupling
	if vn >= 0 {
		return 0, false
	}

	vnNew := -physics.restitution * vn
	deltaVn := vnNew - vn

	// For a solid disk, no-slip correction is -vt/3 because tangential
	// impulse changes both translation and rotation. Different surfaces use
	// different friction scales: paddle strongest, bricks medium, walls weak.
	effectiveFriction := math.Max(
		physics.frictionCoeff*frictionScale,
		physicsMinimumCollisionGrip,
	)
	maxFriction := effectiveFriction * math.Abs(deltaVn)
	desiredDeltaVt := -vt / 3.0
	deltaVt := clampFloat(desiredDeltaVt, -maxFriction, maxFriction)

	b.vx += deltaVn*nx + deltaVt*tx
	b.vy += deltaVn*ny + deltaVt*ty
	b.omega -= 2 * deltaVt / b.r
	b.omega = clampFloat(b.omega, -physics.maxSpin, physics.maxSpin)

	return deltaVt, true
}

// Resolve one collision while measuring how much the ball's spin changed the
// outgoing direction. Diagnostics are recorded only for Ball 1.
func resolveCollisionDebug(
	b *Ball,
	nx, ny, surfVx, surfVy float64,
	frictionScale float64,
	surface string,
	record bool,
) {
	incomingAngle := velocityAngleDegrees(b.vx, b.vy)
	incomingSpeed := math.Hypot(b.vx, b.vy)
	spinBefore := b.omega

	// A zero-spin clone gives a direct A/B measurement of the angle caused by
	// spin at this exact collision, using the same incoming velocity and normal.
	noSpin := *b
	noSpin.omega = 0
	noSpinImpulse := 0.0
	if impulse, collided := resolveCollisionBall(&noSpin, nx, ny, surfVx, surfVy, frictionScale); collided {
		noSpinImpulse = impulse
	}

	tangentialImpulse := 0.0
	if impulse, collided := resolveCollisionBall(b, nx, ny, surfVx, surfVy, frictionScale); collided {
		recordIncomingCollisionSpeed(incomingSpeed)
		tangentialImpulse = impulse
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
func buildBricksFromLevel(lvl levelData, levelIndex int) {
	bricks = nil
	brickGrid = make(map[int][]int)
	remainingBreakableBricks = 0
	lastBrickSoundPlayed = false
	if len(lvl.layout) == 0 {
		initBricksDefault(levelIndex)
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
		initBricksDefault(levelIndex)
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
				x:           gridOffsetLeft + float64(col)*gridCellWidth,
				y:           gridOffsetTop + float64(row)*gridCellHeight,
				w:           scaledWidth,
				h:           scaledHeight,
				row:         row,
				col:         col,
				tiltRadians: brickMicroTiltRadians(levelIndex, row, col),
				alive:       true,
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
func initBricksDefault(levelIndex int) {
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
				row: row, col: col,
				tiltRadians: brickMicroTiltRadians(levelIndex, row, col),
				alive:       true, unbreakable: unbreakable, magic: magic,
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
	gravity := physicsConfig.gravity
	if blackHoleActive {
		currentGravity = 0
	} else if reverseGravityActive {
		currentGravity = -gravity
	} else if lowGravityActive {
		currentGravity = gravity / 3
	} else {
		currentGravity = gravity
	}
}

func clearTimedPowerUps() {
	stopAllMagicFeatureVoices()
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

// Activate a magic-brick feature and play the WAV whose basename exactly
// matches that feature. Missing/failed samples use the established generator.
func activatePowerUpWithBrick(hitBrick *brick, impactSpeed float64) bool {

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
		return false
	}

	p := available[rand.Intn(len(available))]
	feature := ""
	activated := true

	switch p {
	case POWER_LOW_GRAVITY:
		feature = "lowgravity"
		lowGravityActive = true
		lowGravityTimer = powerUpDuration
		refreshCurrentGravity()
		showStatus("Low Gravity!", powerUpDuration)
	case POWER_PASS:
		feature = "passthrough"
		passActive = true
		passTimer = powerUpDuration
		showStatus("Pass Through!", powerUpDuration)
	case POWER_NUKE:
		feature = "nuke"
		nukeBricks(hitBrick)
		showStatus("Nuke!", 2.0)
	case POWER_REVERSE_GRAVITY:
		feature = "reversegravity"
		reverseGravityActive = true
		reverseGravityTimer = powerUpDuration
		refreshCurrentGravity()
		showStatus("Reverse Gravity!", powerUpDuration)
	case POWER_DUAL_BALLS:
		if !secondBallActive {
			feature = "dualballs"
			secondBallActive = true
			secondBall.x = ball.x
			secondBall.y = ball.y
			secondBall.vx = -ball.vx
			secondBall.vy = ball.vy
			secondBall.omega = -ball.omega
			secondBall.angle = ball.angle
			secondBall.stuckTimer = 0
			secondBall.r = ball.r
			resetFastOrbitState(&secondBall)
			showStatus("Dual Balls!", 2.0)
		} else {
			feature = "speedboost"
			ball.vx *= 1.1
			ball.vy *= 1.1
			showStatus("Speed Boost!", 2.0)
		}
	case POWER_BLACKHOLE:
		feature = "blackhole"
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
		refreshCurrentGravity()
	case POWER_MAGNET:
		feature = "magnet"
		magnetPowerActive = true
		magnetPowerTimer = powerUpDuration
		showStatus("Magnets!", powerUpDuration)
	case POWER_INFLUENCER:
		feature = "influencer"
		influencerActive = true
		influencerTimer = powerUpDuration
		showStatus("Influencer!", powerUpDuration)
	case POWER_ZAPPER:
		feature = "zapper"
		zapperPowerActive = true
		zapperPowerTimer = powerUpDuration
		zapperTargetIndex = -1
		zapperHitTimer = 0
		showStatus("Zapper!", powerUpDuration)
	case POWER_BREAK_UNBREAKABLE:
		feature = "breakunbreakable"
		if breakRandomUnbreakable() {
			showStatus("Unbreakable destroyed!", 2.0)
		} else {
			activated = false
		}
	case POWER_BIG_PADDLE:
		feature = "bigpaddle"
		bigPaddleActive = true
		bigPaddleTimer = powerUpDuration
		setPaddleSize(paddleWidth*2, paddleHeight)
		showStatus("Big Paddle!", powerUpDuration)
	}

	if !activated || feature == "" {
		return false
	}
	playMagicFeature(feature, impactSpeed, 1, brickCenterX(hitBrick))
	return true
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
	stopAllMagicFeatureVoices()
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
	mouseControlActive = false
	mousePaddleTargetX = 0
	mobileLeftHeld = false
	mobileRightHeld = false
	mobileLeftPointerID = -1
	mobileRightPointerID = -1
	paddle.vx = 0
	resetPaddleSpinHistory()

	// User toggles never carry into a new level.
	magnetCheat = false
	zapperCheat = false

	// Reset all globals and level-only state before applying config.
	resetGlobals()
	levelMagnetActive = false
	levelZapperActive = false

	// Apply level-specific config. A level can use magnet=true, audioRoom=cave,
	// and audioRoomDry=0.65. If audioRoom is omitted, resetGlobals leaves it dry.
	applyConfig(levels[index].config)
	applyAudioRoomSettings()

	paddle.w = paddleWidth
	paddle.h = paddleHeight

	ball.x, ball.y = startBallX, startBallY
	ball.vx, ball.vy = startBallVx, startBallVy
	ball.omega, ball.angle = 0, 0
	ball.stuckTimer = 0
	ball.soundCooldown = 0
	ball.r = ballRadius
	// Use the real launch speed as the baseline. Later updates are made only
	// from incoming speeds immediately before actual collisions.
	levelMeasuredMaxSpeed = math.Hypot(ball.vx, ball.vy)
	levelMeasuredMaxSpin = math.Abs(ball.omega)
	resetFastOrbitState(&ball)
	resetFastOrbitState(&secondBall)
	secondBallActive = false
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.y = canvasHeight - 40
	paddle.vx = 0
	paddlePreviousX = paddle.x
	mousePaddleTargetX = paddle.x
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
	refreshCurrentGravity()
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

	buildBricksFromLevel(levels[index], index)
	currentLevelIndex = index
	syncRenderInterpolation()
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
func destroyBricksInRadius(ballX, ballY, radius, impactSpeed float64) {
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
			hitX := brickCenterX(br)
			if br.magic {
				playMagic(impactSpeed, hitX)
			} else {
				playBrickBreak(impactSpeed, hitX)
			}
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
	physics := &physicsConfig
	sideMin, sideMax := physics.tiltSideMin, physics.tiltSideMax
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
	b.vy = -physics.tiltUpSpeed
	b.omega += (rand.Float64()*2 - 1) * 8
	b.stuckTimer = 0

	showStatus("TILT!", 1.5)
	playTilt()
}

func updateStuckDetector(b *Ball, dt float64) {
	physics := &physicsConfig
	if b.y+b.r > canvasHeight {
		b.stuckTimer = 0
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	if speed < physics.stuckSpeedThreshold {
		b.stuckTimer += dt
		if b.stuckTimer >= physics.stuckDuration {
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
		playZapperDestroyedBrick(br)
		playZapper(brickCenterX(br))
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

// One brick contact. We still choose a single classic axis
// normal, but group simultaneous overlaps so adjacent bricks do not bounce the
// ball multiple times in one substep.
type brickContact struct {
	index          int
	nx, ny         float64
	axisNX, axisNY float64
	penetration    float64
	impact         float64
	swept          bool
	time           float64
}

func sweptPointAABB(
	previousX, previousY, currentX, currentY,
	minX, minY, maxX, maxY float64,
) (bool, float64, float64, float64) {
	dx := currentX - previousX
	dy := currentY - previousY

	axisTimes := func(position, delta, minimum, maximum float64) (float64, float64) {
		if delta > 0 {
			return (minimum - position) / delta, (maximum - position) / delta
		}
		if delta < 0 {
			return (maximum - position) / delta, (minimum - position) / delta
		}
		if position < minimum || position > maximum {
			return math.Inf(1), math.Inf(-1)
		}
		return math.Inf(-1), math.Inf(1)
	}

	xEntry, xExit := axisTimes(previousX, dx, minX, maxX)
	yEntry, yExit := axisTimes(previousY, dy, minY, maxY)
	entry := math.Max(xEntry, yEntry)
	exit := math.Min(xExit, yExit)
	if entry > exit || exit < 0 || entry < 0 || entry > 1 {
		return false, 0, 0, 0
	}

	nx, ny := 0.0, 0.0
	if math.Abs(xEntry-yEntry) <= 0.000001 {
		if math.Abs(dy) >= math.Abs(dx) {
			ny = -sign(dy)
		} else {
			nx = -sign(dx)
		}
	} else if xEntry > yEntry {
		nx = -sign(dx)
	} else {
		ny = -sign(dy)
	}
	return true, nx, ny, entry
}

func penetrationForBrickNormal(b *Ball, br *brick, nx, ny float64) float64 {
	switch {
	case nx < 0:
		return math.Max(0, b.x-(br.x-b.r))
	case nx > 0:
		return math.Max(0, br.x+br.w+b.r-b.x)
	case ny < 0:
		return math.Max(0, b.y-(br.y-b.r))
	default:
		return math.Max(0, br.y+br.h+b.r-b.y)
	}
}

func findBrickContacts(b *Ball, previousX, previousY float64) []brickContact {
	contacts := make([]brickContact, 0, 4)
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

		axisNX, axisNY := -1.0, 0.0
		penetration := leftPenetration
		if rightPenetration < penetration {
			axisNX, axisNY = 1, 0
			penetration = rightPenetration
		}
		if topPenetration < penetration {
			axisNX, axisNY = 0, -1
			penetration = topPenetration
		}
		if bottomPenetration < penetration {
			axisNX, axisNY = 0, 1
			penetration = bottomPenetration
		}

		swept, sweptNX, sweptNY, hitTime := sweptPointAABB(
			previousX, previousY, b.x, b.y,
			br.x-b.r, br.y-b.r, br.x+br.w+b.r, br.y+br.h+b.r,
		)
		if swept {
			axisNX, axisNY = sweptNX, sweptNY
			penetration = penetrationForBrickNormal(b, br, axisNX, axisNY)
		}

		// The broad-phase geometry remains the existing axis-aligned brick, but
		// the actual collision response follows the brick's tiny visual angle.
		// Correct penetration by the normal's axis component so separation still
		// fully clears the brick boundary.
		nx, ny := rotateVector(axisNX, axisNY, br.tiltRadians)
		axisComponent := math.Abs(nx*axisNX + ny*axisNY)
		if axisComponent > 0.000001 {
			penetration /= axisComponent
		}

		contacts = append(contacts, brickContact{
			index:       i,
			nx:          nx,
			ny:          ny,
			axisNX:      axisNX,
			axisNY:      axisNY,
			penetration: math.Max(0, penetration),
			impact:      math.Max(0, -(b.vx*nx + b.vy*ny)),
			swept:       swept,
			time:        hitTime,
		})
	}
	return contacts
}

func handleBrickCollisions(b *Ball, isPrimary bool, previousX, previousY float64) {
	contacts := findBrickContacts(b, previousX, previousY)
	if len(contacts) == 0 {
		return
	}

	if passActive {
		for _, contact := range contacts {
			br := &bricks[contact.index]
			featureActivated := false
			if br.magic && isPrimary {
				featureActivated = activatePowerUpWithBrick(br, contact.impact)
			}
			if destroyBrick(br) {
				hitX := brickCenterX(br)
				if br.magic {
					if !featureActivated {
						playMagic(contact.impact, hitX)
					}
				} else {
					playBrickBreak(contact.impact, hitX)
				}
				if influencerActive {
					destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier, contact.impact)
				}
			}
		}
		return
	}

	// Swept entry normals win over overlap-only normals. This prevents a ball
	// approaching a flat row from above or below from mistaking the internal
	// seam between two adjacent bricks for alternating left/right walls.
	bestIndex := -1
	for i, contact := range contacts {
		if !contact.swept {
			continue
		}
		if bestIndex < 0 || contact.time < contacts[bestIndex].time-0.000001 ||
			(math.Abs(contact.time-contacts[bestIndex].time) <= 0.000001 && contact.impact > contacts[bestIndex].impact) {
			bestIndex = i
		}
	}

	if bestIndex < 0 {
		hasLeft, hasRight, hasVertical := false, false, false
		for _, contact := range contacts {
			hasLeft = hasLeft || contact.axisNX < 0
			hasRight = hasRight || contact.axisNX > 0
			hasVertical = hasVertical || contact.axisNY != 0
		}
		ignoreInternalHorizontalSeam := hasLeft && hasRight && hasVertical
		for i, contact := range contacts {
			if ignoreInternalHorizontalSeam && contact.axisNX != 0 {
				continue
			}
			if bestIndex < 0 || contact.impact > contacts[bestIndex].impact+0.001 ||
				(math.Abs(contact.impact-contacts[bestIndex].impact) <= 0.001 && contact.penetration > contacts[bestIndex].penetration) {
				bestIndex = i
			}
		}
	}
	if bestIndex < 0 {
		bestIndex = 0
	}
	best := contacts[bestIndex]

	b.x += best.nx * (best.penetration + physicsCollisionSlop)
	b.y += best.ny * (best.penetration + physicsCollisionSlop)
	bestWasUnbreakable := bricks[best.index].unbreakable
	frictionScale := physicsBrickFrictionScale
	if bestWasUnbreakable {
		frictionScale = physicsUnbreakableFrictionScale
	}
	resolveCollisionDebug(b, best.nx, best.ny, 0, 0, frictionScale, "BRICK", isPrimary)
	if bestWasUnbreakable && best.impact >= physicsOrbitMinimumHitSpeed {
		recordUnbreakableOrbitHit(b)
	}

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
			featureActivated := false
			if isPrimary {
				featureActivated = activatePowerUpWithBrick(br, contact.impact)
			}
			if destroyBrick(br) && !featureActivated {
				playMagic(contact.impact, brickCenterX(br))
			}
			if influencerActive {
				destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier, contact.impact)
			}
			continue
		}
		if destroyBrick(br) {
			destroyedNormal = true
			playBrickBreak(contact.impact, brickCenterX(br))
		}
		if influencerActive {
			destroyBricksInRadius(b.x, b.y, b.r*influencerMultiplier, contact.impact)
		}
	}

	if hitUnbreakable {
		hitX := brickCenterX(&bricks[best.index])
		playImpactSound(b, best.impact, func() { playUnbreakable(hitX) })
	}
	if destroyedNormal && physicsConfig.brickBoost != 0 {
		b.vy -= physicsConfig.brickBoost
		if math.Abs(b.vy) > physicsConfig.maxSpeed {
			b.vy = math.Copysign(physicsConfig.maxSpeed, b.vy)
		}
		if math.Abs(b.vx) > physicsConfig.maxSpeed {
			b.vx = math.Copysign(physicsConfig.maxSpeed, b.vx)
		}
	}
}

// ---- Update a single ball ----
func updateBallStep(b *Ball, dt float64, isPrimary bool) {
	physics := &physicsConfig
	if b.soundCooldown > 0 {
		b.soundCooldown = math.Max(0, b.soundCooldown-dt)
	}

	b.vy += currentGravity * dt
	applyBrickMagnetism(b, dt)

	if blackHoleActive {
		dx := blackHoleX - b.x
		dy := blackHoleY - b.y
		dist := math.Hypot(dx, dy)
		if dist > 1.0 {
			b.vx += (dx / dist) * blackHoleStrength * dt
			b.vy += (dy / dist) * blackHoleStrength * dt
		} else {
			b.vx += (rand.Float64() - 0.5) * 10.0
			b.vy += (rand.Float64() - 0.5) * 10.0
		}
	}

	// Magnus effect: spin bends the flight path perpendicular to velocity.
	magnusAx := -b.vy * b.omega * physicsMagnusCoefficient
	magnusAy := b.vx * b.omega * physicsMagnusCoefficient
	magnusMagnitude := math.Hypot(magnusAx, magnusAy)
	maxMagnusAcceleration := math.Max(100.0, physics.maxSpeed*physicsMagnusAccelerationScale)
	if magnusMagnitude > maxMagnusAcceleration {
		scale := maxMagnusAcceleration / magnusMagnitude
		magnusAx *= scale
		magnusAy *= scale
	}
	b.vx += magnusAx * dt
	b.vy += magnusAy * dt

	airDamping := math.Exp(-physicsAirDrag * dt)
	spinDamping := math.Exp(-physicsSpinDrag * dt)
	b.vx *= airDamping
	b.vy *= airDamping
	b.omega *= spinDamping
	applyOverspeedDrag(b, dt, physics)

	previousX, previousY := b.x, b.y
	b.x += b.vx * dt
	b.y += b.vy * dt
	b.angle += b.omega * dt

	// Walls.
	if b.x-b.r < 0 {
		impactSpeed := math.Max(0, -b.vx)
		b.x = b.r + physicsCollisionSlop
		nx, ny := roughWallNormal(1, 0, b.y, canvasHeight, wallNoiseIDLeft, wallSideTiltDegrees)
		resolveCollisionDebug(b, nx, ny, 0, 0, physicsWallFrictionScale, "WALL LEFT", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.x+b.r > canvasWidth {
		impactSpeed := math.Max(0, b.vx)
		b.x = canvasWidth - b.r - physicsCollisionSlop
		nx, ny := roughWallNormal(-1, 0, b.y, canvasHeight, wallNoiseIDRight, wallSideTiltDegrees)
		resolveCollisionDebug(b, nx, ny, 0, 0, physicsWallFrictionScale, "WALL RIGHT", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.y-b.r < 0 {
		impactSpeed := math.Max(0, -b.vy)
		b.y = b.r + physicsCollisionSlop
		nx, ny := roughWallNormal(0, 1, b.x, canvasWidth, wallNoiseIDTop, wallTopTiltDegrees)
		resolveCollisionDebug(b, nx, ny, 0, 0, physicsWallFrictionScale, "WALL TOP", isPrimary)
		playImpactSound(b, impactSpeed, playWallHit)
	}
	if b.y+b.r > canvasHeight {
		return
	}

	// Paddle.
	pLeft, pRight := paddle.x, paddle.x+paddle.w
	pTop, pBottom := paddle.y, paddle.y+paddle.h
	if b.vy > 0 &&
		b.x+b.r > pLeft && b.x-b.r < pRight &&
		b.y+b.r > pTop && b.y+b.r < pBottom {
		recordIncomingCollisionSpeed(math.Hypot(b.vx, b.vy))
		b.y = pTop - b.r - physicsCollisionSlop
		resetFastOrbitState(b)

		spinBeforePaddle := b.omega
		incomingPaddleAngle := velocityAngleDegrees(b.vx, b.vy)
		hitPos := clampFloat((b.x-pLeft)/paddle.w, 0, 1)
		angle := (hitPos - 0.5) * 2.0 * (80.0 * math.Pi / 180.0)
		speed := math.Max(100, math.Hypot(b.vx, b.vy))
		speed = math.Min(speed*physics.restitution+physics.paddleBoost, physics.maxSpeed)

		incomingVx := b.vx
		incomingVy := b.vy
		b.vx = speed*math.Sin(angle) + incomingVx*0.15
		b.vy = -speed * math.Cos(angle)

		effectivePaddleVx := effectivePaddleSpinVelocity()
		relativeSlip := b.vx - b.omega*b.r - effectivePaddleVx
		normalDeltaSpeed := math.Abs(b.vy - incomingVy)
		effectivePaddleFriction := math.Max(
			physics.frictionCoeff*physicsPaddleFrictionScale,
			physicsMinimumPaddleGrip,
		)
		maxFrictionDelta := effectivePaddleFriction * normalDeltaSpeed
		desiredDeltaVx := -relativeSlip / 3.0
		deltaVx := clampFloat(desiredDeltaVx, -maxFrictionDelta, maxFrictionDelta)
		b.vx += deltaVx
		b.omega -= 2 * deltaVx / b.r
		b.omega += -effectivePaddleVx * physicsPaddleSpinTransfer / math.Max(b.r, 1)

		preferredDirection := incomingVx
		if preferredDirection == 0 {
			preferredDirection = paddle.vx
		}
		if preferredDirection == 0 {
			preferredDirection = hitPos - 0.5
		}

		noSpinPaddle := *b
		noSpinPaddle.omega = 0
		noSpinPaddle.vx -= deltaVx
		noSpinRelativeSlip := noSpinPaddle.vx - effectivePaddleVx
		noSpinDesiredDeltaVx := -noSpinRelativeSlip / 3.0
		noSpinDeltaVx := clampFloat(noSpinDesiredDeltaVx, -maxFrictionDelta, maxFrictionDelta)
		noSpinPaddle.vx += noSpinDeltaVx
		preventVerticalLock(&noSpinPaddle, preferredDirection)
		clampBallToPhysicsSettings(&noSpinPaddle, physics)

		preventVerticalLock(b, preferredDirection)
		clampBallToPhysicsSettings(b, physics)
		maybeShowHighSpin(spinBeforePaddle, b.omega)
		if isPrimary {
			recordLastPaddleSpinDebug(true, spinBeforePaddle, b.omega)
			recordLastCollisionDebug(
				"PADDLE", incomingPaddleAngle, velocityAngleDegrees(b.vx, b.vy),
				velocityAngleDegrees(noSpinPaddle.vx, noSpinPaddle.vy),
				spinBeforePaddle, b.omega, deltaVx, noSpinDeltaVx, b.r,
			)
		}
		playImpactSound(b, math.Abs(incomingVy), playPaddleHit)
	}

	handleBrickCollisions(b, isPrimary, previousX, previousY)
	updateFastOrbitDetector(b, dt)
	updateStuckDetector(b, dt)
}

// Physics subdivides fast movement so a ball cannot skip through thin
// bricks between frames.  still performs exactly one step.
// Subdivide exceptionally fast movement so the ball cannot skip thin bricks.
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
		updateBallStep(b, stepDT, isPrimary)
		if b.y+b.r > canvasHeight {
			break
		}
	}
}

func updateBall(b *Ball, dt float64, isPrimary bool) {
	updateBallAdaptive(b, dt, isPrimary)
	recordMeasuredBallSpin(b)
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

// applyMousePaddleControl advances the paddle toward the latest mouse target.
// The target speed is limited by the remaining stopping distance, so the paddle
// brakes before the target instead of crossing it and oscillating. Pointer event
// frequency and monitor refresh rate therefore do not affect the motion model.
func applyMousePaddleControl(dt float64) bool {
	if !mouseControlActive || dt <= 0 {
		return false
	}

	maxX := math.Max(0, canvasWidth-paddle.w)
	mousePaddleTargetX = clampFloat(mousePaddleTargetX, 0, maxX)
	distance := mousePaddleTargetX - paddle.x

	if math.Abs(distance) <= defaultMousePaddleSnapDistance {
		paddle.x = mousePaddleTargetX
		paddle.vx = 0
		return true
	}

	direction := sign(distance)
	stoppingSpeed := math.Sqrt(2 * defaultMousePaddleBraking * math.Abs(distance))
	targetSpeed := direction * math.Min(defaultMousePaddleMaxSpeed, stoppingSpeed)

	changeRate := defaultMousePaddleAcceleration
	if sign(paddle.vx) != 0 && sign(paddle.vx) != direction {
		changeRate = defaultMousePaddleBraking
	} else if math.Abs(targetSpeed) < math.Abs(paddle.vx) {
		changeRate = defaultMousePaddleBraking
	}

	nextVelocity := moveToward(paddle.vx, targetSpeed, changeRate*dt)
	step := nextVelocity * dt
	if sign(step) == direction && math.Abs(step) >= math.Abs(distance) {
		paddle.x = mousePaddleTargetX
		paddle.vx = 0
		return true
	}

	paddle.vx = nextVelocity
	paddle.x += step
	return true
}

// ---- Update (main loop) ----
func update(dt float64) {
	if gameOver || paused || waitingForStart {
		resetPaddleSpinHistory()
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
	} else if applyMousePaddleControl(dt) {
		// Mouse movement is already integrated by the fixed-step controller.
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
	updatePaddleSpinHistory(dt)

	// ---- Independent power-up timers ----
	gravityChanged := false
	if lowGravityActive {
		lowGravityTimer -= dt
		if lowGravityTimer <= 0 {
			stopMagicFeatureVoice("lowgravity")
			lowGravityActive = false
			lowGravityTimer = 0
			gravityChanged = true
		}
	}
	if reverseGravityActive {
		reverseGravityTimer -= dt
		if reverseGravityTimer <= 0 {
			stopMagicFeatureVoice("reversegravity")
			reverseGravityActive = false
			reverseGravityTimer = 0
			gravityChanged = true
		}
	}
	if passActive {
		passTimer -= dt
		if passTimer <= 0 {
			stopMagicFeatureVoice("passthrough")
			passActive = false
			passTimer = 0
		}
	}
	if magnetPowerActive {
		magnetPowerTimer -= dt
		if magnetPowerTimer <= 0 {
			stopMagicFeatureVoice("magnet")
			magnetPowerActive = false
			magnetPowerTimer = 0
		}
	}
	if zapperPowerActive {
		zapperPowerTimer -= dt
		if zapperPowerTimer <= 0 {
			stopMagicFeatureVoice("zapper")
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
			stopMagicFeatureVoice("bigpaddle")
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
			stopMagicFeatureVoice("blackhole")
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
			stopMagicFeatureVoice("influencer")
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
	resetFastOrbitState(&ball)
	resetFastOrbitState(&secondBall)
	secondBallActive = false
	paddle.x = (canvasWidth - paddle.w) / 2
	paddle.vx = 0
	paddlePreviousX = paddle.x
	resetPaddleSpinHistory()
	mouseControlActive = false
	mousePaddleTargetX = paddle.x
	clearLastPaddleSpinDebug()
	clearLastCollisionDebug()
	syncRenderInterpolation()
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
		target.Call("save")
		target.Call("beginPath")
		if drawBrickTilt {
			centerX := br.x + br.w/2
			centerY := br.y + br.h/2
			target.Call("translate", centerX, centerY)
			target.Call("rotate", br.tiltRadians)
			target.Call("roundRect", -br.w/2, -br.h/2, br.w, br.h, brickRadius)
		} else {
			target.Call("roundRect", br.x, br.y, br.w, br.h, brickRadius)
		}
		target.Call("fill")
		target.Call("stroke")
		target.Call("restore")
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

func physicsSettingValue(settings *physicsSettings, key string) float64 {
	switch key {
	case "gravity":
		return settings.gravity
	case "restitution":
		return settings.restitution
	case "frictionCoeff":
		return settings.frictionCoeff
	case "paddleBoost":
		return settings.paddleBoost
	case "brickBoost":
		return settings.brickBoost
	case "maxSpeed":
		return settings.maxSpeed
	case "maxSpin":
		return settings.maxSpin
	case "stuckSpeedThreshold":
		return settings.stuckSpeedThreshold
	case "stuckDuration":
		return settings.stuckDuration
	case "tiltUpSpeed":
		return settings.tiltUpSpeed
	case "tiltSideMin":
		return settings.tiltSideMin
	case "tiltSideMax":
		return settings.tiltSideMax
	}
	return 0
}

func parsePhysicsConfigKey(key string) (field string, ok bool) {
	fields := []string{
		"gravity", "restitution", "frictionCoeff", "paddleBoost", "brickBoost",
		"maxSpeed", "maxSpin", "stuckSpeedThreshold", "stuckDuration",
		"tiltUpSpeed", "tiltSideMin", "tiltSideMax",
	}
	for _, candidate := range fields {
		if key == candidate {
			return candidate, true
		}
	}
	return "", false
}

func configState(key string) (effective, defaultValue, kind string, ok bool) {
	if field, physicsKey := parsePhysicsConfigKey(key); physicsKey {
		defaults := defaultPhysicsSettings()
		return formatConfigFloat(physicsSettingValue(&physicsConfig, field)),
			formatConfigFloat(physicsSettingValue(&defaults, field)), "float", true
	}
	switch key {
	case "audioRoom":
		return currentAudioRoom, "none", "string", true
	case "audioRoomDry":
		return formatConfigFloat(currentAudioRoomDry), "-1", "float", true
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

	lines := []string{
		"BUILD " + buildID,
		"LEVEL " + strconv.Itoa(currentLevelIndex+1) + "/" + strconv.Itoa(len(levels)),
		"UNLOCKED THROUGH " + strconv.Itoa(highestUnlockedLevel+1),
		"DEV ALL LEVELS " + devState,
		"LEVEL OVERRIDES:",
	}

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
func physicsOverlayLines() []string {
	status := "OK"
	if physicsWarningTimer > 0 {
		status = "WARNING: PHYSICS COULD NOT KEEP UP"
	}

	lines := []string{
		"BUILD " + buildID,
		"PHYSICS FIXED STEP " + fmt.Sprintf("%.0f Hz / %.3f ms", physicsStepHz, physicsStepSeconds*1000),
		"MAX TRAVEL / TICK  " + fmt.Sprintf("%.2f px", physicsConfig.maxSpeed*physicsStepSeconds),
		"PHYSICS ACTUAL     " + fmt.Sprintf("%.1f Hz", physicsStepRateCurrent),
		"SIMULATION REALTIME " + fmt.Sprintf("%.1f%%", physicsRealtimePercent),
		"PHYSICS COMPUTE LOAD " + fmt.Sprintf("%.1f%%", physicsComputeLoad),
		"RENDER FPS          " + fmt.Sprintf("%.1f", fpsCurrent),
		"RENDER INTERP       ON / alpha " + fmt.Sprintf("%.3f", renderInterpolationAlpha),
		"STEPS LAST FRAME    " + strconv.Itoa(physicsLastFrameSteps),
		"STEPS PEAK FRAME    " + strconv.Itoa(physicsPeakFrameSteps),
		"CATCH-UP LIMIT      " + strconv.Itoa(physicsMaxCatchUpSteps),
		"DROPPED SIM TIME    " + fmt.Sprintf("%.4f s", physicsDroppedTimeTotal),
		"STATUS " + status,
		"",
		"BALL 1 SPEED " + fmt.Sprintf("%.2f", math.Hypot(ball.vx, ball.vy)) +
			" (max " + strconv.FormatFloat(physicsConfig.maxSpeed, 'f', -1, 64) + ")",
		"BALL 1 SPIN  " + fmt.Sprintf("%+.2f", ball.omega) +
			" (max " + strconv.FormatFloat(physicsConfig.maxSpin, 'f', -1, 64) + ")",
	}
	if secondBallActive {
		lines = append(lines,
			"BALL 2 SPEED "+fmt.Sprintf("%.2f", math.Hypot(secondBall.vx, secondBall.vy))+
				" (max "+strconv.FormatFloat(physicsConfig.maxSpeed, 'f', -1, 64)+")",
			"BALL 2 SPIN  "+fmt.Sprintf("%+.2f", secondBall.omega)+
				" (max "+strconv.FormatFloat(physicsConfig.maxSpin, 'f', -1, 64)+")",
		)
	}
	if lastPaddleSpinValid {
		lines = append(lines,
			"",
			"LAST PADDLE HIT BALL "+strconv.Itoa(lastPaddleSpinBall),
			"SPIN BEFORE "+fmt.Sprintf("%+.2f", lastPaddleSpinBefore),
			"SPIN ADDED  "+fmt.Sprintf("%+.2f", lastPaddleSpinAdded),
			"SPIN AFTER  "+fmt.Sprintf("%+.2f", lastPaddleSpinAfter),
		)
	}
	if lastCollisionValid {
		lines = append(lines,
			"",
			"LAST BALL 1 COLLISION "+lastCollisionSurface,
			"ANGLE IN          "+fmt.Sprintf("%+.2f deg", lastCollisionIncomingAngle),
			"ANGLE OUT         "+fmt.Sprintf("%+.2f deg", lastCollisionOutgoingAngle),
			"ANGLE CHANGE      "+fmt.Sprintf("%+.2f deg", lastCollisionAngleChange),
			"SPIN ANGLE EFFECT "+fmt.Sprintf("%+.2f deg", lastCollisionSpinAngleEffect),
			"HIT SPIN BEFORE   "+fmt.Sprintf("%+.2f", lastCollisionSpinBefore),
			"HIT SPIN CHANGE   "+fmt.Sprintf("%+.2f", lastCollisionSpinChange),
			"HIT SPIN AFTER    "+fmt.Sprintf("%+.2f", lastCollisionSpinAfter),
			"CONTACT SPIN SPEED "+fmt.Sprintf("%+.2f", lastCollisionContactSpinSurfaceSpeed),
			"IMPULSE WITH SPIN "+fmt.Sprintf("%+.2f", lastCollisionTangentialImpulse),
			"IMPULSE NO SPIN   "+fmt.Sprintf("%+.2f", lastCollisionNoSpinImpulse),
			"SPIN IMPULSE EFFECT "+fmt.Sprintf("%+.2f", lastCollisionSpinImpulseEffect),
		)
	}
	return lines
}

func physicsOverlayReport() string {
	return strings.Join(physicsOverlayLines(), "\n")
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

func physicsOverlayGeometry(lineCount int) (panelX, panelY, panelWidth, panelHeight float64, maxRows int) {
	panelX, panelY, panelWidth, panelHeight, maxRows = debugOverlayGeometry(lineCount)
	panelX = 10
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

func overlayReportAtPointer(e js.Value) (string, bool) {
	x, y, ok := pointerCanvasPosition(e)
	if !ok {
		return "", false
	}
	if debugOverlayVisible {
		panelX, panelY, panelWidth, panelHeight, _ := debugOverlayGeometry(len(debugOverlayLines()))
		if x >= panelX && x <= panelX+panelWidth && y >= panelY && y <= panelY+panelHeight {
			return debugOverlayReport(), true
		}
	}
	if physicsOverlayVisible {
		panelX, panelY, panelWidth, panelHeight, _ := physicsOverlayGeometry(len(physicsOverlayLines()))
		if x >= panelX && x <= panelX+panelWidth && y >= panelY && y <= panelY+panelHeight {
			return physicsOverlayReport(), true
		}
	}
	return "", false
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

func drawOverlayLines(lines []string) {
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

func drawDebugOverlay() {
	if debugOverlayVisible {
		drawOverlayLines(debugOverlayLines())
	}
}

func drawPhysicsOverlayLines(lines []string) {
	panelX, panelY, _, _, maxRows := physicsOverlayGeometry(len(lines))

	const lineHeight = 18.0
	const padding = 12.0
	const columnWidth = 420.0

	ctx.Call("save")
	ctx.Set("fillStyle", palette[4])
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

func drawPhysicsOverlay() {
	if physicsOverlayVisible {
		drawPhysicsOverlayLines(physicsOverlayLines())
	}
}

func drawCenteredOverlay() {
	ctx.Call("save")
	ctx.Set("fillStyle", "rgba(0, 0, 0, 0.5)")
	ctx.Call("fillRect", 0, 0, canvasWidth, canvasHeight)
	ctx.Call("restore")
}

func draw(alpha float64) {
	renderState := interpolatedRenderSnapshot(alpha)
	ctx.Set("fillStyle", palette[0])
	ctx.Call("fillRect", 0, 0, canvasWidth, canvasHeight)

	if bricksDirty {
		rebuildBrickCanvas()
	}
	ctx.Call("drawImage", brickCanvas, 0, 0)
	drawZapperBolts()

	if renderState.blackHoleActive && showBlackHole {
		ctx.Set("fillStyle", "#000000")
		ctx.Set("strokeStyle", "#9d4edd")
		ctx.Set("lineWidth", 5)
		ctx.Call("beginPath")
		ctx.Call("arc", renderState.blackHoleX, renderState.blackHoleY, 28, 0, 2*math.Pi)
		ctx.Call("fill")
		ctx.Call("stroke")

		ctx.Set("strokeStyle", "#c77dff")
		ctx.Set("lineWidth", 2)
		ctx.Call("beginPath")
		ctx.Call("arc", renderState.blackHoleX, renderState.blackHoleY, 42, 0, 2*math.Pi)
		ctx.Call("stroke")
	}

	drawBall(renderState.ballX, renderState.ballY, ball.r, renderState.ballAngle, palette[3], palette[5])
	if renderState.secondBallActive {
		drawBall(renderState.secondBallX, renderState.secondBallY, secondBall.r, renderState.secondBallAngle, palette[7], palette[8])
	}

	ctx.Set("fillStyle", palette[1])
	ctx.Call("beginPath")
	ctx.Call("roundRect", renderState.paddleX, paddle.y, paddle.w, paddle.h, paddleRadius)
	ctx.Call("fill")

	ctx.Set("fillStyle", palette[4])
	ctx.Set(
		"font",
		"18px GameFont, monospace",
	)
	// Match the P display: 10 units from the canvas edge plus 12 units of
	// internal text padding. Its first-line baseline would be 37 at the top.
	const (
		hudTextX      = 22.0
		hudFirstLineY = 37.0
		hudLineStep   = 30.0
	)
	updateHUDCache()
	ctx.Call("fillText", hudLivesText, hudTextX, hudFirstLineY)
	ctx.Call("fillText", hudLevelText, hudTextX, hudFirstLineY+hudLineStep)
	ctx.Call("fillText", hudScoreText, hudTextX, hudFirstLineY+2*hudLineStep)

	currentSpeed := math.Hypot(ball.vx, ball.vy)
	ctx.Call("fillText", "Speed: "+fmt.Sprintf("%.0f", currentSpeed)+
		" ("+fmt.Sprintf("%.0f", levelMeasuredMaxSpeed)+")", hudTextX, hudFirstLineY+3*hudLineStep)

	if math.Abs(ball.omega) > 100 {
		ctx.Set("fillStyle", "#ff0000")
	}
	ctx.Call("fillText", "Spin:  "+fmt.Sprintf("%+.0f", ball.omega)+
		" ("+fmt.Sprintf("%.0f", levelMeasuredMaxSpin)+")", hudTextX, hudFirstLineY+4*hudLineStep)
	ctx.Set("fillStyle", palette[4])

	for i, message := range statusMessages {
		y := hudFirstLineY + 160.0 + float64(i)*hudLineStep
		ctx.Call("fillText", message.text, hudTextX, y)
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
	drawPhysicsOverlay()
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
	// This budget spans all fixed physics steps executed by one visual frame.
	brickHitStartsThisFrame = 0

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

	frameDt := rawDt
	if frameDt < 0 {
		frameDt = 0
	}
	if frameDt > physicsMaxFrameDelta {
		physicsSampleDroppedTime += frameDt - physicsMaxFrameDelta
		physicsDroppedTimeTotal += frameDt - physicsMaxFrameDelta
		physicsWarningTimer = physicsWarningHoldSeconds
		frameDt = physicsMaxFrameDelta
	}
	physicsAccumulator += frameDt

	computeStart := js.Global().Get("performance").Call("now").Float()
	steps := 0
	for physicsAccumulator+1e-12 >= physicsStepSeconds && steps < physicsMaxCatchUpSteps {
		beginPhysicsStepForRendering()
		update(physicsStepSeconds)
		physicsAccumulator -= physicsStepSeconds
		steps++
	}
	if physicsAccumulator >= physicsStepSeconds {
		dropped := physicsAccumulator - math.Mod(physicsAccumulator, physicsStepSeconds)
		physicsAccumulator = math.Mod(physicsAccumulator, physicsStepSeconds)
		physicsSampleDroppedTime += dropped
		physicsDroppedTimeTotal += dropped
		physicsWarningTimer = physicsWarningHoldSeconds
	}
	computeSeconds := (js.Global().Get("performance").Call("now").Float() - computeStart) / 1000.0

	physicsLastFrameSteps = steps
	if steps > physicsPeakFrameSteps {
		physicsPeakFrameSteps = steps
	}
	if rawDt > 0 && rawDt < 1.0 {
		physicsSampleElapsed += rawDt
		physicsSampleSteps += steps
		physicsSampleComputeTime += computeSeconds
		if physicsSampleElapsed >= 0.5 {
			physicsStepRateCurrent = float64(physicsSampleSteps) / physicsSampleElapsed
			physicsRealtimePercent = float64(physicsSampleSteps) * physicsStepSeconds / physicsSampleElapsed * 100
			physicsComputeLoad = physicsSampleComputeTime / physicsSampleElapsed * 100
			if physicsSampleDroppedTime > 0 || physicsComputeLoad >= 90 {
				physicsWarningTimer = physicsWarningHoldSeconds
			}
			physicsSampleElapsed = 0
			physicsSampleSteps = 0
			physicsSampleComputeTime = 0
			physicsSampleDroppedTime = 0
		}
	}
	if physicsWarningTimer > 0 {
		physicsWarningTimer = math.Max(0, physicsWarningTimer-rawDt)
	}

	renderInterpolationAlpha = clampFloat(physicsAccumulator/physicsStepSeconds, 0, 1)
	draw(renderInterpolationAlpha)
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

func captureRenderSnapshot() renderSnapshot {
	return renderSnapshot{
		ballX: ball.x, ballY: ball.y, ballAngle: ball.angle,
		secondBallX: secondBall.x, secondBallY: secondBall.y, secondBallAngle: secondBall.angle,
		paddleX:    paddle.x,
		blackHoleX: blackHoleX, blackHoleY: blackHoleY,
		secondBallActive: secondBallActive,
		blackHoleActive:  blackHoleActive,
	}
}

func syncRenderInterpolation() {
	previousRenderSnapshot = captureRenderSnapshot()
	renderSnapshotReady = true
	renderInterpolationAlpha = 0
}

func beginPhysicsStepForRendering() {
	previousRenderSnapshot = captureRenderSnapshot()
	renderSnapshotReady = true
}

func lerpFloat(a, b, alpha float64) float64 {
	return a + (b-a)*alpha
}

// TODO(render): If uneven browser/compositor pacing remains visible, consider
// bounded visual-only extrapolation of moving objects by no more than one physics
// step, suppressed around collisions. Keep it disabled unless interpolation alone
// proves insufficient; it must never modify the authoritative physics state.
func interpolatedRenderSnapshot(alpha float64) renderSnapshot {
	current := captureRenderSnapshot()
	if !renderSnapshotReady {
		previousRenderSnapshot = current
		renderSnapshotReady = true
		return current
	}

	alpha = clampFloat(alpha, 0, 1)
	result := renderSnapshot{
		ballX:            lerpFloat(previousRenderSnapshot.ballX, current.ballX, alpha),
		ballY:            lerpFloat(previousRenderSnapshot.ballY, current.ballY, alpha),
		ballAngle:        lerpFloat(previousRenderSnapshot.ballAngle, current.ballAngle, alpha),
		paddleX:          lerpFloat(previousRenderSnapshot.paddleX, current.paddleX, alpha),
		blackHoleX:       lerpFloat(previousRenderSnapshot.blackHoleX, current.blackHoleX, alpha),
		blackHoleY:       lerpFloat(previousRenderSnapshot.blackHoleY, current.blackHoleY, alpha),
		secondBallActive: current.secondBallActive,
		blackHoleActive:  current.blackHoleActive,
	}

	// New or removed transient objects must not interpolate from stale positions.
	if previousRenderSnapshot.secondBallActive == current.secondBallActive {
		result.secondBallX = lerpFloat(previousRenderSnapshot.secondBallX, current.secondBallX, alpha)
		result.secondBallY = lerpFloat(previousRenderSnapshot.secondBallY, current.secondBallY, alpha)
		result.secondBallAngle = lerpFloat(previousRenderSnapshot.secondBallAngle, current.secondBallAngle, alpha)
	} else {
		result.secondBallX = current.secondBallX
		result.secondBallY = current.secondBallY
		result.secondBallAngle = current.secondBallAngle
	}
	if previousRenderSnapshot.blackHoleActive != current.blackHoleActive {
		result.blackHoleX = current.blackHoleX
		result.blackHoleY = current.blackHoleY
	}

	return result
}

func applyOverspeedDrag(b *Ball, dt float64, settings *physicsSettings) {
	if b == nil || settings == nil || dt <= 0 || settings.maxSpeed <= 0 || physicsOverspeedHalfLife <= 0 {
		return
	}

	speed := math.Hypot(b.vx, b.vy)
	if speed <= settings.maxSpeed || speed <= 0 {
		return
	}

	excess := speed - settings.maxSpeed
	excess *= math.Exp(-math.Ln2 * dt / physicsOverspeedHalfLife)
	targetSpeed := settings.maxSpeed + excess
	scale := targetSpeed / speed
	b.vx *= scale
	b.vy *= scale
}

func clampBallToPhysicsSettings(b *Ball, settings *physicsSettings) {
	// Linear overspeed is intentionally not hard-clamped. It is allowed as a
	// temporary result of spin-to-speed transfer and decays in flight.
	b.omega = clampFloat(b.omega, -settings.maxSpin, settings.maxSpin)
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
func pointerPaddleX(e js.Value) (float64, bool) {
	rect := canvas.Call("getBoundingClientRect")
	displayWidth := rect.Get("width").Float()
	if displayWidth <= 0 {
		return 0, false
	}

	scaleX := canvasWidth / displayWidth
	pointerX := (e.Get("clientX").Float() - rect.Get("left").Float()) * scaleX
	x := pointerX - paddle.w/2
	x = clampFloat(x, 0, math.Max(0, canvasWidth-paddle.w))
	return x, true
}

func setMousePaddleTarget(e js.Value) {
	x, ok := pointerPaddleX(e)
	if !ok {
		return
	}
	mousePaddleTargetX = x
	mouseControlActive = true
}

// Touch-follow mode intentionally remains direct: its physics update branch does
// not integrate paddle.vx again, unlike the old desktop mouse path.
func movePaddleToPointer(e js.Value) {
	x, ok := pointerPaddleX(e)
	if !ok {
		return
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
		unlockAudioFromGesture()
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

		// Unlock/resume audio on the first keyboard gesture, including the key
		// that leaves the waiting screen. Embedded sample decoding is already
		// asynchronous, so this path never waits for a sample.
		unlockAudioFromGesture()

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

		// I: level/palette overrides. P: physics diagnostics.
		if (key == "i" || key == "I") && !e.Get("repeat").Bool() {
			debugOverlayVisible = !debugOverlayVisible
			if debugOverlayVisible {
				physicsOverlayVisible = false
			}
			return nil
		}
		if (key == "p" || key == "P") && !e.Get("repeat").Bool() {
			physicsOverlayVisible = !physicsOverlayVisible
			if physicsOverlayVisible {
				debugOverlayVisible = false
			}
			return nil
		}

		// Page Up / Page Down navigate levels in every game state, including the
		// win and game-over screens. Check both key and code for browser/keyboard
		// compatibility.
		pageDownPressed := key == "PageDown" || code == "PageDown"
		pageUpPressed := key == "PageUp" || code == "PageUp"
		if pageDownPressed && !e.Get("repeat").Bool() {
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
		if pageUpPressed && !e.Get("repeat").Bool() {
			limit := activeUnlockedLimit()
			if currentLevelIndex > 0 {
				jumpToLevel(currentLevelIndex - 1)
			} else if limit > 0 {
				jumpToLevel(limit)
			}
			return nil
		}

		// Start a new game after winning.
		if gameOver && win {
			if (key == " " || key == "Enter") && !e.Get("repeat").Bool() {
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
				resetFastOrbitState(&secondBall)
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
			mouseControlActive = false
			leftPressed = true
		} else if key == "ArrowRight" || key == "d" || key == "D" || code == "AltRight" {
			mouseControlActive = false
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
		if report, inside := overlayReportAtPointer(e); inside {
			if copyTextToClipboard(report) {
				showStatus("Debug info copied", 1.5)
			} else {
				showStatus("Clipboard unavailable", 1.5)
			}
			return nil
		}

		unlockAudioFromGesture()

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
			setMousePaddleTarget(e)

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
			setMousePaddleTarget(e)
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
		mouseControlActive = false
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
	syncRenderInterpolation()

	// Build the audio graph and begin embedded compressed-sample decoding after
	// the current call stack, before normal gameplay. Browsers may keep the
	// context suspended until the first user gesture, but decoding can proceed.
	scheduleAudioPreparation()

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
