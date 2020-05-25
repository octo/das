package dkb4q

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/octo/das/dkb4q/fake"
)

func TestGetReports(t *testing.T) {
	cases := []struct {
		title     string
		responses [][]byte
		want      [][]byte
		wantErr   bool
	}{
		{
			title:     "base",
			responses: [][]byte{{0xED, 0x03, 0x78, 0x00, 0x96, 0x00, 0x00, 0x00}},
			want:      [][]byte{{0xED, 0x03, 0x78, 0x00, 0x96}},
		},
		{
			title: "batch",
			responses: [][]byte{
				{0xED, 0x03, 0x78, 0x00, 0x96, 0xED, 0x03, 0x78},
				{0x00, 0x96, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			want: [][]byte{
				{0xED, 0x03, 0x78, 0x00, 0x96},
				{0xED, 0x03, 0x78, 0x00, 0x96},
			},
		},
		{
			title: "initial zero response",
			responses: [][]byte{
				{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				{0xED, 0x03, 0x78, 0x00, 0x96, 0x00, 0x00, 0x00},
			},
			want: [][]byte{{0xED, 0x03, 0x78, 0x00, 0x96}},
		},
		{
			title: "intermittent zero response",
			responses: [][]byte{
				{0xED, 0x03, 0x78, 0x00, 0x96, 0xED, 0x03, 0x78},
				{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
				{0x00, 0x96, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			},
			want: [][]byte{
				{0xED, 0x03, 0x78, 0x00, 0x96},
				{0xED, 0x03, 0x78, 0x00, 0x96},
			},
		},
		{
			title:     "checksum mismatch",
			responses: [][]byte{{0xED, 0x03, 0x78, 0x00, 0xEE, 0x00, 0x00, 0x00}},
			wantErr:   true,
		},
		{
			title:     "invalid length",
			responses: [][]byte{{0xED, 0x00, 0x78, 0x00, 0x95, 0x00, 0x00, 0x00}},
			wantErr:   true,
		},
	}

	for _, tc := range cases {
		var (
			ctx = context.Background()
			hid fake.HID
		)

		for _, res := range tc.responses {
			hid.WantGetReport = append(hid.WantGetReport, fake.Report{
				ID:   1,
				Data: res,
			})
		}
		kb := &Keyboard{
			dev: &hid,
		}

		got, err := kb.getReports(ctx)
		if gotErr := err != nil; gotErr != tc.wantErr {
			t.Errorf("getReports() = %v, want error %v", err, tc.wantErr)
		}
		if tc.wantErr {
			continue
		}

		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Errorf("getReports() differs (+got/-want):\n%s", diff)
		}
	}
}

type fakeReportGetter struct {
	responses [][]byte
}

func (r *fakeReportGetter) GetReport(reportID int) ([]byte, error) {
	if reportID != 1 {
		return nil, fmt.Errorf("reportID: got %d, want 1", reportID)
	}

	if len(r.responses) == 0 {
		return nil, fmt.Errorf("unexpected GetReport call")
	}

	var res []byte
	res, r.responses = r.responses[0], r.responses[1:]
	return res, nil
}
