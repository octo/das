package dkb4q

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/octo/retry"
)

func encodeReport(reportType byte, data []byte) []byte {
	encLen := 2 + len(data) + 1

	// The output buffer size must be divisible by 7.
	bufSize := encLen
	if rem := bufSize % 7; rem != 0 {
		bufSize += 7 - rem
	}

	enc := make([]byte, bufSize, bufSize)
	enc[0] = reportType
	enc[1] = byte(encLen - 2)
	copy(enc[2:encLen-1], data)
	enc[encLen-1] = xorAll(enc[:encLen-1])

	return enc
}

func xorAll(data []byte) byte {
	var ret byte
	for _, d := range data {
		ret ^= d
	}
	return ret
}

func isZero(data []byte) bool {
	for _, b := range data {
		if b != 0x00 {
			return false
		}
	}
	return true
}

type reportGetter interface {
	GetReport(reportID int) ([]byte, error)
}

func getReports(hid reportGetter) ([][]byte, error) {
	var (
		buf     bytes.Buffer
		reports [][]byte
	)

	for {
		for buf.Len() < 2 {
			data, err := getReport(hid)
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
			data, err := getReport(hid)
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

func getReport(hid reportGetter) ([]byte, error) {
	var ret []byte
	cb := func(_ context.Context) error {
		fmt.Printf("<- GetReport(1) = ")
		data, err := hid.GetReport(1)
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

	err := retry.Do(context.TODO(), cb)
	return ret, err
}
