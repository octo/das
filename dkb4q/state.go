package dkb4q

import (
	"context"
	"errors"
	"fmt"
	"image/color"
	"time"
)

// State represents the (desired) state of one key. "Idle" refers to the
// key's normal state, "active" to the keys state after is has been pressed.
type State struct {
	ID           uint8
	IdleEffect   IdleEffect
	IdleColor    color.NRGBA
	ActiveEffect ActiveEffect
	ActiveColor  color.NRGBA
}

// SetState sets the state of one or more LEDs / keys. Passing many states in
// one call is more efficient than calling SetState repeatedly.
func (kb *Keyboard) SetState(ctx context.Context, states ...State) error {
	for _, s := range states {
		if err := kb.stageState(ctx, s); err != nil {
			return err
		}
	}

	return kb.commitState(ctx)
}

func (kb *Keyboard) stageState(ctx context.Context, s State) error {
	msg0 := encodeReport(0xEA, []byte{0x78, 0x03, s.ID, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	if err := kb.setReport(ctx, msg0); err != nil {
		return fmt.Errorf("setReport(msg0 = %#v) = %w", msg0, err)
	}

	res0, err := kb.getReports(ctx)
	if err != nil && !errors.Is(err, errNoReport) {
		return err
	}
	// should return "ED 03 78 00 96"
	fmt.Printf("response 0 = %#v\n", res0)

	msg1 := encodeReport(0xEA, []byte{0x78, 0x08, s.ID, byte(s.IdleEffect),
		s.IdleColor.R, s.IdleColor.G, s.IdleColor.B})
	if err := kb.setReport(ctx, msg1); err != nil {
		return fmt.Errorf("setReport(msg1 = %#v) = %w", msg1, err)
	}

	msg2 := []byte{0x78, 0x04, s.ID, s.ActiveEffect.id,
		s.ActiveColor.R, s.ActiveColor.G, s.ActiveColor.B,
		s.ActiveEffect.arg0,
		s.ActiveEffect.arg1,
		s.ActiveEffect.arg2}
	msg2 = encodeReport(0xEA, msg2)
	if err := kb.setReport(ctx, msg2); err != nil {
		return fmt.Errorf("setReport(msg2 = %#v) = %w", msg2, err)
	}

	res1, err := kb.getReports(ctx)
	if err != nil && !errors.Is(err, errNoReport) {
		return err
	}
	// should return "ED 03 78 00 96"
	fmt.Printf("response 1 = %#v\n", res1)

	return nil
}

func (kb *Keyboard) commitState(ctx context.Context) error {
	msg3 := encodeReport(0xEA, []byte{0x78, 0x0A})
	if err := kb.setReport(ctx, msg3); err != nil {
		return err
	}

	res2, err := kb.getReports(ctx)
	if err != nil {
		return err
	}
	// should return "ED 03 78 00 96"
	fmt.Printf("response 2 = %#v\n", res2)

	return nil
}

// IdleEffect describes the key's behavior when it is inactive.
type IdleEffect uint8

const (
	// SetColor steadily lights the key in a single color.
	SetColor IdleEffect = 0x01
	// Breathe cycles the key's light through continuous phases of high/low intensity.
	Breathe = 0x08
	// Blink turns the key's light on/off at regular intervals.
	Blink = 0x1F
	// ColorCycle continuously cycles the key's light through the colors of the rainbow.
	ColorCycle = 0x14
)

// ActiveEffect describes the key's behavior when it is activated, i.e. pressed.
type ActiveEffect struct {
	id               byte
	arg0, arg1, arg2 byte
}

// None disables an active effect, i.e. the key will not react to key presses.
var None = ActiveEffect{}

// ActiveEffectOption is an option to an active effect. Not all active effects
// support all options – see the option's documentation for the effects they
// support.
type ActiveEffectOption func(*ActiveEffect)

// SetColorActive lights the key in a single color. After some time (default:
// 1.9 seconds) the key reverts to its idle state.
func SetColorActive(opts ...ActiveEffectOption) ActiveEffect {
	ae := ActiveEffect{
		id:   0x1E,
		arg0: 0x07,
		arg1: 0xD0,
	}

	for _, opt := range opts {
		opt(&ae)
	}

	return ae
}

// BlinkActive lets keys blink when pressed. Number of on/off cycles and the
// cycle duration can be controlled with CycleCount and CycleDuration.
func BlinkActive(opts ...ActiveEffectOption) ActiveEffect {
	ae := ActiveEffect{
		id:   byte(Blink),
		arg0: 0x01,
		arg1: 0xF4,
		arg2: 0x03,
	}

	for _, opt := range opts {
		opt(&ae)
	}

	return ae
}

// BreatheActive lets keys "breathe" – smoothly cycle through high/low
// intensity – when pressed. Number of on/off cycles and the cycle duration can
// be controlled with CycleCount and CycleDuration.
func BreatheActive(opts ...ActiveEffectOption) ActiveEffect {
	ae := ActiveEffect{
		id: byte(Breathe),
		// TODO(octo): arg0 and arg1 are not yet understood.
		arg0: 0x03,
		arg1: 0xE8,
		arg2: 0x03,
	}

	for _, opt := range opts {
		opt(&ae)
	}

	return ae
}

// EffectDuration sets how long the "SetColorActive" effect lasts before it
// reverts to the idle state.
func EffectDuration(d time.Duration) ActiveEffectOption {
	return func(ae *ActiveEffect) {
		if ae.id != 0x1E {
			return
		}
		const precision = 270 * time.Millisecond

		d = d.Round(precision)
		value := int(d / precision)
		if value < 1 || value > 255 {
			return
		}

		ae.arg0 = byte(value)
	}
}

// CycleCount sets how often a key blinks with the "BlinkActive" effect or
// "breathes" with the "BreatheActive" effect.
func CycleCount(c uint8) ActiveEffectOption {
	return func(ae *ActiveEffect) {
		if ae.id != byte(Blink) && ae.id != byte(Breathe) {
			return
		}
		ae.arg2 = byte(c)
	}
}

// CycleDuration sets how long each on/off cycle of the "BlinkActive" effect is.
// Defaults to 1.05 seconds.
func CycleDuration(d time.Duration) ActiveEffectOption {
	return func(ae *ActiveEffect) {
		if ae.id != byte(Blink) {
			return
		}

		// the default value 0x01F4 (500) translates into a cycle time of 1.05 seconds.
		const precision = (1050 * time.Millisecond) / 500

		d = d.Round(precision)
		value := uint(d / precision)
		if value < 1 || value > 0xffff {
			return
		}

		ae.arg0 = byte((value >> 8) & 0x00ff)
		ae.arg1 = byte(value & 0x00FF)
	}
}
