// Package das implements the low-level protocol of Das Keyboard.
//
// This is a very early draft and very much work in progress. The goal is to
// support "Das Keyboard 4Q", because that's the one I happen to own.
package dkb4q

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/octo/retry"
	"github.com/zserge/hid"
)

const (
	MaxLEDID = 124
)

// Keyboard represents the connection to a keyboard.
type Keyboard struct {
	dev interface {
		Close()
		SetReport(int, []byte) error
		GetReport(int) ([]byte, error)
	}
}

// ErrNotFound is returned by Open if no matching device is found.
var ErrNotFound = errors.New("no Das Keyboard device found")

// Open scans USB devices for a "Das Keyboard" by looking for the vendor ID
// 0x24F0. It returns a Keyboard talking to the first device successfully
// opened. If no device could be opened, the last error is returned, or
// ErrNotFound if no matching device was found.
//
// The connection to the keyboard should be closed with Close().
func Open() (Keyboard, error) {
	const vendorID = 0x24F0

	var (
		device  hid.Device
		lastErr error
	)
	hid.UsbWalk(func(dev hid.Device) {
		if device != nil || dev.Info().Vendor != vendorID {
			return
		}

		fmt.Println("dev.Info() =", dev.Info())
		if dev.Info().Interface != 1 {
			return
		}

		if err := dev.Open(); err != nil {
			lastErr = err
			return
		}

		device = dev
		return
	})

	if device == nil {
		if lastErr != nil {
			return Keyboard{}, lastErr
		}
		return Keyboard{}, ErrNotFound
	}

	kb := Keyboard{
		dev: device,
	}

	return kb, nil
}

// Close closes the connection to the keyboard.
func (kb *Keyboard) Close() error {
	defer func() {
		kb.dev = nil
	}()

	if kb.dev == nil {
		return fmt.Errorf("connection to keyboard not open")
	}

	kb.dev.Close()
	return nil
}

func (kb *Keyboard) setReport(ctx context.Context, data []byte) error {
	if len(data)%7 != 0 {
		return fmt.Errorf("invalid message length %d; use encodeReport to generate a correct encoding", len(data))
	}

	for i := 0; i < len(data); i += 7 {
		payload := append([]byte{0x01}, data[i:i+7]...)
		err := retry.Do(ctx, func(_ context.Context) error {
			fmt.Printf("-> SetReport(1, %#v)\n", payload)
			return kb.dev.SetReport(1, payload)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func (kb *Keyboard) getReports(ctx context.Context) ([][]byte, error) {
	var (
		buf     bytes.Buffer
		reports [][]byte
	)

	for {
		for buf.Len() < 2 {
			data, err := kb.getReport(ctx)
			if err != nil {
				return nil, err
			}
			buf.Write(data)
		}

		reportType, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}

		reportLen, err := buf.ReadByte()
		if err != nil {
			return nil, err
		}
		if reportLen < 2 {
			return nil, fmt.Errorf("invalid length %d, want at least 2", reportLen)
		}

		for buf.Len() < int(reportLen) {
			data, err := kb.getReport(ctx)
			if err != nil {
				return nil, err
			}
			buf.Write(data)
		}

		payload := buf.Next(int(reportLen))
		report := append([]byte{reportType, reportLen}, payload...)

		gotParity := payload[len(payload)-1]
		wantParity := xorAll(report[:len(report)-1])
		if gotParity != wantParity {
			return nil, fmt.Errorf("parity mismatch: got %#x, want %#x", gotParity, wantParity)
		}

		reports = append(reports, report)

		if isZero(buf.Bytes()) {
			break
		}
	}

	return reports, nil
}

var errNoReport = errors.New("no report available")

func (kb *Keyboard) getReport(ctx context.Context) ([]byte, error) {
	var ret []byte
	cb := func(_ context.Context) error {
		fmt.Printf("<- GetReport(1) = ")
		data, err := kb.dev.GetReport(1)
		if err != nil {
			fmt.Println(err)
			return retry.Abort(err)
		}
		if isZero(data) {
			fmt.Println(errNoReport)
			return errNoReport
		}
		fmt.Printf("%#v\n", data)
		ret = data
		return nil
	}

	err := retry.Do(ctx, cb)
	return ret, err
}
