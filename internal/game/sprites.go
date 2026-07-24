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

// Bat: a cave dweller that roosts, then dive-dashes. 8x4.
var batPal = map[rune]uint8{
	'k': 235, // body
	'w': 59,  // wings
	'E': 196, // eyes
}

var (
	sprBat1 = gfx.NewFrames(gfx.MustSprite(batPal,
		"ww....ww",
		".wwkkww.",
		"..kEEk..",
		"........"))
	sprBat2 = gfx.NewFrames(gfx.MustSprite(batPal,
		"........",
		"..wkkw..",
		".wwEEww.",
		"ww....ww"))
)

// Lurker: a pale cave spider that hangs from ceilings and drops on
// passers-by. Symmetric, so it reads right hanging or walking. 7x5.
var lurkerPal = map[rune]uint8{
	'l': 244, // legs
	'B': 238, // body
	'E': 226, // eyes
}

var (
	sprLurker1 = gfx.NewFrames(gfx.MustSprite(lurkerPal,
		"l.l.l.l",
		".BBBBB.",
		"lBEBEBl",
		".BBBBB.",
		"l.l.l.l"))
	sprLurker2 = gfx.NewFrames(gfx.MustSprite(lurkerPal,
		".l.l.l.",
		".BBBBB.",
		"lBEBEBl",
		".BBBBB.",
		".l.l.l."))
)

// Shardling: a floating crystal sentinel that fires aimed shards. 5x7.
var shardPal = map[rune]uint8{
	'C': 51,  // bright facets
	'c': 38,  // dim facets
	'P': 183, // core
	'E': 201, // eye
}

var (
	sprShard1 = gfx.NewFrames(gfx.MustSprite(shardPal,
		"..C..",
		".cPc.",
		"CPEPC",
		".cPc.",
		"..C..",
		".c.c.",
		"c...c"))
	sprShard2 = gfx.NewFrames(gfx.MustSprite(shardPal,
		"..c..",
		".CPC.",
		"cPEPc",
		".CPC.",
		"..c..",
		".C.C.",
		"C...C"))
)

// Magling: a molten hopper from the deep fire. 7x5.
var maglingPal = map[rune]uint8{
	'K': 52,  // basalt hide
	'O': 202, // molten cracks
	'E': 231, // eyes
}

var (
	sprMagling1 = gfx.NewFrames(gfx.MustSprite(maglingPal,
		"..KKK..",
		".KKKKK.",
		"KOEKEOK",
		"KKKOKKK",
		".KK.KK."))
	sprMagling2 = gfx.NewFrames(gfx.MustSprite(maglingPal,
		".......",
		".KKKKK.",
		"KOEKEOK",
		"KKKOKKK",
		"KK...KK"))
)

// Boss: Dimi, warden of the ruins. A hulking spike-backed beast with
// glowing eyes. 26x14.
var dimiPal = map[rune]uint8{
	'V': 54,  // hide, dark violet
	'v': 53,  // hide shade / spikes
	'B': 96,  // belly
	'E': 196, // glowing eyes
	'T': 231, // teeth
	'm': 16,  // jaw shadow
	'C': 246, // claws
}

var sprDimi = gfx.NewFrames(gfx.MustSprite(dimiPal,
	".....v....v....v..........",
	"....vVv..vVv..vVv.........",
	"...vVVVvvVVVvvVVVv........",
	"..vVVVVVVVVVVVVVVvv.......",
	".vVVVVVVVVVVVVVVVVVvv.....",
	".vVVVVVVVVVVVVVVVVVVVv....",
	"vVVVVVVVVVVVVVVVEEVVVVv...",
	"vVBBVVVVVVVVVVVVVVVVVVv...",
	"vVBBVVVVVVVVVVVVTTTTTv....",
	".vVVVVVVVVVVVVVVmmmmmv....",
	".vVVVv..vVVVv..vVVVv......",
	".vVVv...vVVv...vVVv.......",
	".vVVv...vVVv...vVVv.......",
	".CCC....CCC....CCC........"))

// Boss: Prisma, the crystal queen. A floating faceted shard with a
// glowing core and orbiting fragments. 18x14.
var prismaPal = map[rune]uint8{
	'C': 51,  // facet edge, bright cyan
	'c': 38,  // facet shade
	'P': 183, // body
	'p': 97,  // body shade
	'E': 201, // eyes
	'W': 231, // crown shine
}

var sprPrisma = gfx.NewFrames(gfx.MustSprite(prismaPal,
	"........cc........",
	".......cCCc.......",
	"......cCWWCc......",
	".....cCPPPPCc.....",
	"....cCPEPPEPCc....",
	"...cCPPPPPPPPCc...",
	"...cCPppppppPCc...",
	"....cCPPPPPPCc....",
	".....cCPPPPCc.....",
	"......cCPPCc......",
	".......cCCc.......",
	"........cc........",
	"..c.....c.....c...",
	".c.......c......c."))

// Boss: Magmaw, lord of the deep fire. A huge molten-cracked maw that
// hops and spits lava. 22x14.
var magmawPal = map[rune]uint8{
	'K': 52,  // basalt hide
	'k': 88,  // hide rim
	'O': 202, // molten cracks
	'E': 231, // eyes
	'Y': 220, // mouth glow
	'T': 254, // teeth
}

var sprMagmaw = gfx.NewFrames(gfx.MustSprite(magmawPal,
	"......kkkkkkkkk.......",
	"....kkKKKKKKKKKkk.....",
	"...kKKEKKKKKKKEKKk....",
	"..kKKKKKKKKKKKKKKKk...",
	".kKKOKKKKKOKKKKOKKKk..",
	".kKKKKKKKKKKKKKKKKKk..",
	".kTYYYYYYYYYYYYYYYTk..",
	".kYYYYYYYYYYYYYYYYYk..",
	".kTYYYYYYYYYYYYYYYTk..",
	".kKKKKKKKKKKKKKKKKKk..",
	"..kKKOKKKKKOKKKKKKk...",
	"..kKKKKKKKKKKKKKKKk...",
	"...kkKKKKkkkKKKKkk....",
	"...OO......OO........."))

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

// Ship part pickups: each of the seven parts is a different piece of
// the rocket, so finding one tells you what you salvaged.
var partsPal = map[rune]uint8{
	'g': 246, // metal
	'G': 252, // bright metal
	'd': 240, // dark metal
	'Y': 220, // brass
	'y': 178, // brass shade
	'w': 117, // glass
	'W': 231, // shine
	'c': 208, // copper
	'C': 130, // copper shade
	'e': 40,  // circuit traces
	'E': 22,  // circuit board
}

var sprParts = []*gfx.Sprite{
	gfx.MustSprite(partsPal, // gear
		".y.y.",
		"yYYYy",
		".YWY.",
		"yYYYy",
		".y.y."),
	gfx.MustSprite(partsPal, // thruster nozzle
		".gGGg.",
		".dggd.",
		".dggd.",
		"dGggGd",
		"dg..gd",
		".c..c."),
	gfx.MustSprite(partsPal, // fuel cell
		".ggg.",
		"gGGGg",
		"gYYYg",
		"gGGGg",
		"gGGGg",
		".ddd."),
	gfx.MustSprite(partsPal, // antenna dish
		"W.gg..",
		".gGGg.",
		"gGGGGg",
		"..dd..",
		"..dd..",
		".dddd."),
	gfx.MustSprite(partsPal, // copper coil
		"cCCCc",
		".ccc.",
		"cCCCc",
		".ccc.",
		"cCCCc",
		".ddd."),
	gfx.MustSprite(partsPal, // circuit board
		"y.yy.y",
		"eEEEEe",
		"eEWeEe",
		"eEEEEe",
		"y.yy.y"),
	gfx.MustSprite(partsPal, // porthole window
		".gggg.",
		"gwwWwg",
		"gwWwwg",
		"gwwwwg",
		"gwwwwg",
		".gggg."),
}

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
