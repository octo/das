package fake

import (
	"errors"
	"fmt"

	"github.com/google/go-cmp/cmp"
	"github.com/octo/retry"
)

type Report struct {
	ID   int
	Data []byte
}

// HID is a fake implementation of the HID object used by dkb4q.Keyboard.
type HID struct {
	WantSetReport []Report
	WantGetReport []Report
	closed        bool
}

func (d *HID) Close() {
	d.closed = true
}

var (
	errClosed     = retry.Abort(errors.New("device is closed"))
	errUnexpected = retry.Abort(errors.New("unexpected call"))
)

func (d *HID) SetReport(id int, data []byte) error {
	if d.closed {
		return errClosed
	}
	if len(d.WantSetReport) == 0 {
		return errUnexpected
	}

	got := Report{
		ID:   id,
		Data: data,
	}

	var want Report
	want, d.WantSetReport = d.WantSetReport[0], d.WantSetReport[1:]

	if diff := cmp.Diff(want, got); diff != "" {
		return retry.Abort(fmt.Errorf("report differs (+got/-want):\n%s", diff))
	}

	return nil
}

func (d *HID) GetReport(id int) ([]byte, error) {
	if d.closed {
		return nil, errClosed
	}
	if len(d.WantGetReport) == 0 {
		return nil, errUnexpected
	}

	var res Report
	res, d.WantGetReport = d.WantGetReport[0], d.WantGetReport[1:]

	if id != res.ID {
		return nil, retry.Abort(fmt.Errorf("report ID = %d, want %d", id, res.ID))
	}

	return res.Data, nil
}
