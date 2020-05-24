// all-color sets all keys on the keyboard to the same color.
//
// Currently the colors are hard-coded. Eventually, they should be provided by
// a flag.
package main

import (
	"fmt"
	"image/color"
	"log"
	"time"

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

	for i := 0; i < dkb4q.MaxLEDID; i++ {
		c := colors[i%len(colors)]

		fmt.Println("=== LED", i, "===")
		err := kb.State(dkb4q.KeyState{
			LEDID:         uint8(i),
			PassiveEffect: dkb4q.SetColor,
			PassiveColor:  c,
			ActiveEffect:  dkb4q.SetColorActive,
			ActiveColor:   color.NRGBA{R: 0xF4, G: 0x05, B: 0x06},
		})
		if err != nil {
			log.Fatal(err)
		}

		fmt.Println("")
		time.Sleep(time.Second)
	}
}
