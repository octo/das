// cpu-meter colors the F1–F12 keys according to current CPU usage.
package main

import (
	"bufio"
	"bytes"
	"context"
	"image/color"
	"io/ioutil"
	"log"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/octo/das/dkb4q"
)

var keys = []uint8{
	0x11, 0x17, 0x1D, 0x23, // F1–F4
	0x29, 0x2F, 0x35, 0x3b, // F5–F8
	0x41, 0x47, 0x4D, 0x53, // F9–F12
}

func main() {
	ctx := context.Background()

	kb, err := dkb4q.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer kb.Close()

	var state cpuState
	if err := state.update(); err != nil {
		log.Fatal(err)
	}

	ticker := time.Tick(5 * time.Second)
	for range ticker {
		if err := state.update(); err != nil {
			log.Fatal(err)
		}
		system, user, idle := state.rates()
		total := system + user + idle

		systemKeys := float64(len(keys)) * system / total
		userKeys := float64(len(keys)) * user / total

		var keyState []dkb4q.State
		for i, id := range keys {
			ks := dkb4q.State{
				ID:         id,
				IdleEffect: dkb4q.SetColor,
				IdleColor:  color.NRGBA{}, // black
			}

			switch {
			case float64(i+1) <= systemKeys:
				ks.IdleColor = color.NRGBA{R: 0xFF} // red
			case float64(i) < systemKeys:
				// Transition from system to user CPU. Mix red
				// and blue according to their respective
				// weights.
				weightRed := systemKeys - math.Floor(systemKeys)
				weightBlue := math.Min(1.0-weightRed, userKeys)
				// value may be <1 if userKeys<1.
				value := weightRed + weightBlue

				var red, blue float64
				if weightRed < weightBlue {
					red = value * weightRed / weightBlue
					blue = value
				} else {
					red = value
					blue = value * weightBlue / weightRed
				}

				ks.IdleColor = color.NRGBA{
					R: uint8(255.0*red + .5),
					B: uint8(255.0*blue + .5),
				}
			case float64(i+1) <= (systemKeys + userKeys):
				ks.IdleColor = color.NRGBA{B: 0xFF} // blue
			case float64(i) < (systemKeys + userKeys):
				blue := (systemKeys + userKeys) - math.Floor(systemKeys+userKeys)
				ks.IdleColor = color.NRGBA{
					B: uint8(255.0*blue + .5),
				}
			}

			keyState = append(keyState, ks)
		}

		if err := kb.SetState(ctx, keyState...); err != nil {
			log.Fatal(err)
		}
	}
}

type cpuState struct {
	counter []uint64
	rate    []float64
}

func (s *cpuState) update() error {
	data, err := ioutil.ReadFile("/proc/stat")
	if err != nil {
		return err
	}

	scan := bufio.NewScanner(bytes.NewReader(data))
	for scan.Scan() {
		line := scan.Text()

		fields := strings.Fields(line)
		if len(fields) < 10 || fields[0] != "cpu" {
			continue
		}

		init := false
		if len(s.counter) < 9 {
			init = true

			s.counter = make([]uint64, 9, 9)
			s.rate = make([]float64, 9, 9)
		}

		for i := 0; i < 9; i++ {
			v, err := strconv.ParseUint(fields[i+1], 10, 64)
			if err != nil {
				return err
			}

			if init {
				s.rate[i] = math.NaN()
			} else {
				s.rate[i] = float64(v-s.counter[i]) / 10.0
			}
			s.counter[i] = v
		}

		break
	}

	return nil
}

const (
	fieldUser = 0
	fieldNice = 1
	fieldIdle = 3
)

func (s *cpuState) rates() (system, user, idle float64) {
	for i := 0; i < 9; i++ {
		if math.IsNaN(s.rate[i]) {
			continue
		}
		switch i {
		case fieldUser, fieldNice:
			user += s.rate[i]
		case fieldIdle:
			idle += s.rate[i]
		default:
			system += s.rate[i]
		}
	}

	return system, user, idle
}
