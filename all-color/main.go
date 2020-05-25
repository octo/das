// all-color sets all keys on the keyboard to the same color.
//
// Currently the colors are hard-coded. Eventually, they should be provided by
// a flag.
package main

import (
	"image/color"
	"log"

	"github.com/octo/das/dkb4q"
)

var colors = []color.NRGBA{
	{R: 66, G: 133, B: 244},
	{R: 219, G: 68, B: 55},
	{R: 244, G: 160, B: 0},
	{R: 15, G: 157, B: 88},
}

func main() {
	kb, err := dkb4q.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer kb.Close()

	var states []dkb4q.KeyState
	for i := 0; i < dkb4q.MaxLEDID; i++ {
		c := colors[i%len(colors)]

		states = append(states, dkb4q.KeyState{
			LEDID:         uint8(i),
			PassiveEffect: dkb4q.SetColor,
			PassiveColor:  c,
			ActiveEffect:  dkb4q.SetColorActive,
			ActiveColor:   color.NRGBA{R: 0xFF - c.R, G: 0xFF - c.G, B: 0xFF - c.B},
		})
	}

	if err := kb.State(states...); err != nil {
		log.Fatal(err)
	}
}
