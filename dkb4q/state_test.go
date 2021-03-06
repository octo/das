package dkb4q

import (
	"context"
	"image/color"
	"testing"
	"time"

	"github.com/octo/das/dkb4q/fake"
)

func TestKeyboard_State(t *testing.T) {
	cases := []struct {
		title         string
		states        []State
		wantSetReport [][]byte
		wantGetReport [][]byte
		wantErr       bool
	}{
		{
			title: "set_color and none",
			states: []State{
				{
					ID:           0x05,
					IdleEffect:   SetColor,
					IdleColor:    color.NRGBA{R: 0xFB, G: 0x02, B: 0x03},
					ActiveEffect: None,
				},
			},
			wantSetReport: [][]byte{
				{1, 0xEA, 0x0B, 0x78, 0x03, 0x05, 0x00, 0x00}, // msg 0
				{1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9F, 0},    // msg 1
				{1, 0xEA, 0x08, 0x78, 0x08, 0x05, 0x01, 0xFB}, // msg 3
				{1, 0x02, 0x03, 0x6C, 0, 0, 0, 0},             // msg 4
				{1, 0xEA, 0x0B, 0x78, 0x04, 0x05, 0x00, 0x00}, // msg 5
				{1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x98, 0},    // msg 6
				{1, 0xEA, 0x03, 0x78, 0x0A, 0x9B, 0, 0},       // msg 8
			},
			wantGetReport: [][]byte{
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 2
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 7
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 9
			},
		},
		{
			title: "set_color and set_color",
			states: []State{
				{
					ID:           0x05,
					IdleEffect:   SetColor,
					IdleColor:    color.NRGBA{R: 0xFB, G: 0x02, B: 0x03},
					ActiveEffect: SetColorActive(EffectDuration(4 * time.Second)),
					ActiveColor:  color.NRGBA{R: 0xFC, G: 0xFD, B: 0xFE},
				},
			},
			wantSetReport: [][]byte{
				{1, 0xEA, 0x0B, 0x78, 0x03, 0x05, 0x00, 0x00}, // msg 0
				{1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9F, 0},    // msg 1
				{1, 0xEA, 0x08, 0x78, 0x08, 0x05, 0x01, 0xFB}, // msg 3
				{1, 0x02, 0x03, 0x6C, 0, 0, 0, 0},             // msg 4
				{1, 0xEA, 0x0B, 0x78, 0x04, 0x05, 0x1E, 0xFC}, // msg 5
				{1, 0xFD, 0xFE, 0x0F, 0xD0, 0x00, 0xA6, 0},    // msg 6
				{1, 0xEA, 0x03, 0x78, 0x0A, 0x9B, 0, 0},       // msg 8
			},
			wantGetReport: [][]byte{
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 2
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 7
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 9
			},
		},
		{
			title: "set_color and blink",
			states: []State{
				{
					ID:           0x05,
					IdleEffect:   SetColor,
					IdleColor:    color.NRGBA{R: 0xFB, G: 0x02, B: 0x03},
					ActiveEffect: BlinkActive(CycleCount(2), CycleDuration(2*time.Second)),
					ActiveColor:  color.NRGBA{R: 0xFC, G: 0xFD, B: 0xFE},
				},
			},
			wantSetReport: [][]byte{
				{1, 0xEA, 0x0B, 0x78, 0x03, 0x05, 0x00, 0x00}, // msg 0
				{1, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9F, 0},    // msg 1
				{1, 0xEA, 0x08, 0x78, 0x08, 0x05, 0x01, 0xFB}, // msg 3
				{1, 0x02, 0x03, 0x6C, 0, 0, 0, 0},             // msg 4
				{1, 0xEA, 0x0B, 0x78, 0x04, 0x05, 0x1F, 0xFC}, // msg 5
				{1, 0xFD, 0xFE, 0x03, 0xB8, 0x02, 0xC1, 0},    // msg 6
				{1, 0xEA, 0x03, 0x78, 0x0A, 0x9B, 0, 0},       // msg 8
			},
			wantGetReport: [][]byte{
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 2
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 7
				{0xED, 0x03, 0x78, 0x00, 0x96, 0, 0, 0}, // msg 9
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.title, func(t *testing.T) {
			var (
				ctx = context.Background()
				hid fake.HID
			)

			for _, data := range tc.wantSetReport {
				hid.WantSetReport = append(hid.WantSetReport, fake.Report{
					ID:   1,
					Data: data,
				})
			}
			for _, data := range tc.wantGetReport {
				hid.WantGetReport = append(hid.WantGetReport, fake.Report{
					ID:   1,
					Data: data,
				})
			}

			kb := &Keyboard{
				dev: &hid,
			}
			defer kb.Close()

			err := kb.SetState(ctx, tc.states...)
			if gotErr := err != nil; gotErr != tc.wantErr {
				t.Errorf("Keyboard.SetState() = %v, want error %v", err, tc.wantErr)
			}
			if tc.wantErr {
				return
			}
		})
	}
}
