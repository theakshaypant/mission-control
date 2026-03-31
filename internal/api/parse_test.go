package api

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAPIDuration(t *testing.T) {
	cases := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"24h", 24 * time.Hour, false},
		{"2h30m", 2*time.Hour + 30*time.Minute, false},
		{"30m", 30 * time.Minute, false},
		{"7d", 7 * 24 * time.Hour, false},
		{"1d", 24 * time.Hour, false},
		{"bad", 0, true},
		{"xd", 0, true},
		{"", 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseAPIDuration(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseAPIUntil(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			input: "2026-04-01T12:00:00Z",
			check: func(t *testing.T, got time.Time) {
				want, _ := time.Parse(time.RFC3339, "2026-04-01T12:00:00Z")
				assert.Equal(t, want, got)
			},
		},
		{
			input: "2026-04-01",
			check: func(t *testing.T, got time.Time) {
				assert.Equal(t, 2026, got.Year())
				assert.Equal(t, time.April, got.Month())
				assert.Equal(t, 1, got.Day())
				assert.Equal(t, 0, got.Hour())
				assert.Equal(t, 0, got.Minute())
			},
		},
		{
			input: "14:30",
			check: func(t *testing.T, got time.Time) {
				now := time.Now()
				assert.Equal(t, 14, got.Hour())
				assert.Equal(t, 30, got.Minute())
				assert.Equal(t, now.Year(), got.Year())
				assert.Equal(t, now.Month(), got.Month())
				assert.Equal(t, now.Day(), got.Day())
			},
		},
		{input: "not-a-time", wantErr: true},
		{input: "99:99", wantErr: true},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseAPIUntil(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			if tc.check != nil {
				tc.check(t, got)
			}
		})
	}
}

func TestParseSnoozeRequest_Validation(t *testing.T) {
	cases := []struct {
		name    string
		req     snoozeRequest
		wantErr bool
	}{
		{"both set", snoozeRequest{For: "24h", Until: "14:30"}, true},
		{"neither set", snoozeRequest{}, true},
		{"for only — valid", snoozeRequest{For: "24h"}, false},
		{"for only — days", snoozeRequest{For: "7d"}, false},
		{"for only — invalid", snoozeRequest{For: "bad"}, true},
		{"until only — RFC3339", snoozeRequest{Until: "2026-04-01T00:00:00Z"}, false},
		{"until only — date", snoozeRequest{Until: "2026-04-01"}, false},
		{"until only — HH:MM", snoozeRequest{Until: "14:30"}, false},
		{"until only — invalid", snoozeRequest{Until: "bad"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseSnoozeRequest(tc.req)
			if tc.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
