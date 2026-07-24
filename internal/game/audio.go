//go:build window

package game

// Procedural chiptune sound effects for window mode. Everything is
// synthesized at startup into PCM buffers: square waves with pitch
// sweeps for actions, xorshift noise for impacts and weather. No asset
// files.

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const sfxSampleRate = 44100

type sfxBank struct {
	ctx  *audio.Context
	data [SfxCount][]byte
}

func newSfxBank() *sfxBank {
	b := &sfxBank{ctx: audio.NewContext(sfxSampleRate)}

	b.data[SfxJump] = synth(0.12, func(s *synthState, t, u float64) float64 {
		return s.square(250+300*u) * 0.5 * (1 - u)
	})
	b.data[SfxDash] = synth(0.16, func(s *synthState, t, u float64) float64 {
		return s.smoothNoise() * 0.6 * (1 - u)
	})
	b.data[SfxShoot] = synth(0.08, func(s *synthState, t, u float64) float64 {
		return s.square(950-650*u) * 0.4 * (1 - u) * (1 - u)
	})
	b.data[SfxLand] = synth(0.05, func(s *synthState, t, u float64) float64 {
		return math.Sin(2*math.Pi*140*t) * 0.4 * (1 - u)
	})
	b.data[SfxHurt] = synth(0.2, func(s *synthState, t, u float64) float64 {
		return (s.noise()*0.4 + s.square(110)*0.4) * (1 - u)
	})
	b.data[SfxKill] = synth(0.25, func(s *synthState, t, u float64) float64 {
		return (s.square(480-380*u)*0.4 + s.noise()*0.25) * (1 - u)
	})
	b.data[SfxHeart] = synth(0.16, func(s *synthState, t, u float64) float64 {
		f := 660.0
		if u > 0.5 {
			f = 880
		}
		return s.square(f) * 0.35 * (1 - u)
	})
	b.data[SfxPart] = synth(0.42, func(s *synthState, t, u float64) float64 {
		notes := []float64{523, 659, 784, 1046}
		i := min(int(u*4), 3)
		seg := u*4 - float64(i)
		return s.square(notes[i]) * 0.4 * (1 - seg*0.6)
	})
	b.data[SfxBossRoar] = synth(0.7, func(s *synthState, t, u float64) float64 {
		return s.square(58+16*math.Sin(u*22)) * 0.5 * (1 - u*u)
	})
	b.data[SfxExplosion] = synth(0.5, func(s *synthState, t, u float64) float64 {
		e := math.Pow(1-u, 1.6)
		return (s.smoothNoise()*0.7 + s.square(60)*0.25) * e
	})
	b.data[SfxThunder] = synth(0.9, func(s *synthState, t, u float64) float64 {
		e := 0.4 + 0.6*math.Exp(-u*4)
		return (s.smoothNoise()*0.55 + math.Sin(2*math.Pi*48*t)*0.25) * e * (1 - u)
	})
	return b
}

func (b *sfxBank) Play(s Sfx) {
	if d := b.data[s]; len(d) > 0 {
		p := b.ctx.NewPlayerFromBytes(d)
		p.SetVolume(0.4)
		p.Play()
	}
}

// synthState carries oscillator phase and noise state across samples,
// so pitch sweeps stay continuous.
type synthState struct {
	phase float64
	rng   uint32
	last  float64
}

func (s *synthState) square(freq float64) float64 {
	s.phase += 2 * math.Pi * freq / sfxSampleRate
	if math.Sin(s.phase) >= 0 {
		return 1
	}
	return -1
}

func (s *synthState) noise() float64 {
	s.rng ^= s.rng << 13
	s.rng ^= s.rng >> 17
	s.rng ^= s.rng << 5
	return float64(int32(s.rng))/math.MaxInt32*2 - 1
}

// smoothNoise is noise put through a crude one-pole low pass, for
// whooshes and rumbles instead of hiss.
func (s *synthState) smoothNoise() float64 {
	s.last += (s.noise() - s.last) * 0.18
	return s.last * 3
}

// synth renders dur seconds of a mono generator into 16-bit stereo PCM.
func synth(dur float64, gen func(s *synthState, t, u float64) float64) []byte {
	n := int(dur * sfxSampleRate)
	out := make([]byte, n*4)
	s := &synthState{rng: 0x9d2c5680}
	for i := 0; i < n; i++ {
		t := float64(i) / sfxSampleRate
		v := gen(s, t, t/dur)
		v = math.Max(-1, math.Min(1, v))
		p := int16(v * 32767)
		out[i*4] = byte(p)
		out[i*4+1] = byte(p >> 8)
		out[i*4+2] = byte(p)
		out[i*4+3] = byte(p >> 8)
	}
	return out
}
