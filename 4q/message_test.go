package dkb4q

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestEncodeReport(t *testing.T) {
	cases := []struct {
		inType       byte
		inData, want []byte
	}{
		{
			inType: 0xEA,
			inData: []byte{0x78, 0x03, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
			want:   []byte{0xEA, 0x0B, 0x78, 0x03, 0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x9F, 0x00},
		},
		{
			inType: 0xEA,
			inData: []byte{0x78, 0x08, 0x05, 0x01, 0x60, 0x61, 0x62},
			want:   []byte{0xEA, 0x08, 0x78, 0x08, 0x05, 0x01, 0x60, 0x61, 0x62, 0xF5, 0x00, 0x00, 0x00, 0x00},
		},
		{
			inType: 0xEA,
			inData: []byte{0x78, 0x04, 0x05, 0x1E, 0xFE, 0x01, 0x02, 0x07, 0xD0, 0x00},
			want:   []byte{0xEA, 0x0B, 0x78, 0x04, 0x05, 0x1E, 0xFE, 0x01, 0x02, 0x07, 0xD0, 0x00, 0xAC, 0x00},
		},
		{
			inType: 0xEA,
			inData: []byte{0x78, 0x0A},
			want:   []byte{0xEA, 0x03, 0x78, 0x0A, 0x9B, 0x00, 0x00},
		},
	}

	for _, tc := range cases {
		got := encodeReport(tc.inType, tc.inData)
		if diff := cmp.Diff(tc.want, got); diff != "" {
			t.Errorf("encodeReport(%#x, %#v) differs (+got/-want):\n%s", tc.inType, tc.inData, diff)
		}
	}
}

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
			want:      [][]byte{{0xED, 0x03, 0x78, 0x00, 0x96}},
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
			wantErr: true,
		},
		{
			title:     "invalid length",
			responses: [][]byte{{0xED, 0x00, 0x78, 0x00, 0x95, 0x00, 0x00, 0x00}},
			wantErr: true,
		},
	}

	for _, tc := range cases {
		r := &fakeReportGetter{
			responses: tc.responses,
		}

		got, err := getReports(r)
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
