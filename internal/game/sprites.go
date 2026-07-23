package game

import "github.com/AgustinBanchio/terminal-cappy/internal/gfx"

// All art is authored facing right; left-facing variants are mirrored
// at startup. Colours are xterm-256 palette indices.

// Cappy: a bipedal border collie in a red astronaut suit with a glass
// helmet, holding a laser revolver. 10x12 pixels, hitbox 6x10.
var cappyPal = map[rune]uint8{
	'G': 152, // helmet glass rim (pale blue)
	'*': 231, // glass shine
	'W': 255, // white fur (blaze, muzzle)
	'k': 235, // black fur patches
	'E': 16,  // eye
	'N': 16,  // nose
	'R': 160, // suit red
	'r': 88,  // suit shade
	'O': 252, // suit trim / belt
	'g': 246, // revolver metal
	'd': 240, // revolver barrel
	'B': 59,  // boots
}

// cappyBody is rows 0-9 shared by every frame: helmet + head, torso and
// the gun arm. Frames only differ in the two leg rows appended below.
var cappyBody = []string{
	"..GGGGG...",
	".G*kk..G..",
	".GkWWWEG..",
	".GWWWWNG..",
	".G.WWW.G..",
	"..GGGGG...",
	"..RRRR....",
	".RRRRRggd.",
	"WRRRRRr...",
	"..ORRO....",
}

func cappyFrame(legs ...string) gfx.Frames {
	rows := append(append([]string{}, cappyBody...), legs...)
	return gfx.NewFrames(gfx.MustSprite(cappyPal, rows...))
}

var (
	sprCappyIdle = cappyFrame(
		"..R..R....",
		"..B..B....")
	sprCappyRun1 = cappyFrame(
		".R....R...",
		".B....B...")
	sprCappyRun2 = cappyFrame(
		"...RR.....",
		"...BB.....")
	sprCappyJump = cappyFrame(
		"..R.R.....",
		".B...B....")
	sprCappyFall = cappyFrame(
		"..R..R....",
		".B....B...")
	sprCappySlide = cappyFrame(
		"..RR.R....",
		"..BB.B....")
)

// sprPortrait is Cappy's helmet-and-head, doubled in size, used as the
// avatar in dialogue boxes.
var sprPortrait = gfx.MustSprite(cappyPal, cappyBody[:6]...).Scale(2)

// Walker alien: a grumpy green blob on stubby feet. 8x5.
var walkerPal = map[rune]uint8{
	'A': 40,  // body green
	'v': 28,  // body shade
	'e': 231, // eyes
	'm': 16,  // mouth
}

var (
	sprWalker1 = gfx.NewFrames(gfx.MustSprite(walkerPal,
		"..vAAv..",
		".vAAAAv.",
		"vAeAAeAv",
		"vAAmmAAv",
		".vv..vv."))
	sprWalker2 = gfx.NewFrames(gfx.MustSprite(walkerPal,
		"..vAAv..",
		".vAAAAv.",
		"vAeAAeAv",
		"vAAmmAAv",
		"..vv.vv."))
)

// Flyer alien: a drifting jellyfish thing. 7x6.
var flyerPal = map[rune]uint8{
	'M': 165, // bell magenta
	'm': 126, // bell shade
	'e': 231, // eyes
	't': 133, // tentacles
}

var (
	sprFlyer1 = gfx.NewFrames(gfx.MustSprite(flyerPal,
		"..MMM..",
		".MeMeM.",
		"MMMMMMM",
		".mMmMm.",
		"..t.t..",
		".t...t."))
	sprFlyer2 = gfx.NewFrames(gfx.MustSprite(flyerPal,
		"..MMM..",
		".MeMeM.",
		"MMMMMMM",
		".mMmMm.",
		".t.t.t.",
		"..t.t.."))
)

// Cappy's crashed ship: a red and white rocket lying on its side,
// scorched near the engine. 28x12.
var shipPal = map[rune]uint8{
	'R': 160, // hull red
	's': 88,  // hull shade
	'H': 254, // white band
	'w': 117, // cockpit glass
	'W': 231, // glass shine
	'g': 246, // engine metal
	'k': 236, // scorch marks
	'f': 88,  // fins
}

var sprShip = gfx.MustSprite(shipPal,
	"...................ff.......",
	"..................fRRs......",
	"........RRRRRRRRRRRRRss.....",
	"......RRHHHHHHHHHHRRRRss....",
	"....RRHHwwHHHHHHHHHHRRRgg...",
	"...RHHHwWwwHHHHHHHHHRRsgg...",
	"..RRHHHwwwHHHHkkHHHRRRsgg...",
	"..RRRHHHHHHHHHkkkHRRRRsgg...",
	"...RRRRRRRRRRRRRkkRRRss.....",
	"....RRRRRRRRRRRRRRRRss......",
	"..................fRRs......",
	"...................ff.......")

// Ship part pickup: a glowing gear. 5x5.
var partPal = map[rune]uint8{'Y': 220, 'y': 178, 'w': 231}

var sprPart = gfx.MustSprite(partPal,
	".y.y.",
	"yYYYy",
	".YwY.",
	"yYYYy",
	".y.y.")

// Heart pickup / HUD heart. 5x4.
var heartPal = map[rune]uint8{'h': 196, 'H': 210}
var heartEmptyPal = map[rune]uint8{'h': 238, 'H': 238}

var heartArt = []string{
	".h.h.",
	"hHhhh",
	".hhh.",
	"..h..",
}

var (
	sprHeart      = gfx.MustSprite(heartPal, heartArt...)
	sprHeartEmpty = gfx.MustSprite(heartEmptyPal, heartArt...)
)

// Muzzle flash. 3x3.
var muzzlePal = map[rune]uint8{'Y': 226, 'W': 231}

var sprMuzzle = gfx.MustSprite(muzzlePal,
	".Y.",
	"YWY",
	".Y.")
