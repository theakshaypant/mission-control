package cmd

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
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
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseDuration(tc.input)
			if tc.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}

func TestParseUntil(t *testing.T) {
	cases := []struct {
		input   string
		wantErr bool
		check   func(t *testing.T, got time.Time)
	}{
		{
			input: "2026-04-01T00:00:00Z",
			check: func(t *testing.T, got time.Time) {
				want, _ := time.Parse(time.RFC3339, "2026-04-01T00:00:00Z")
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
			got, err := parseUntil(tc.input)
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

func TestResolveSnoozeTime_For(t *testing.T) {
	before := time.Now()
	got, err := resolveSnoozeTime("24h", "")
	after := time.Now()

	require.NoError(t, err)
	assert.GreaterOrEqual(t, got, before.Add(24*time.Hour))
	assert.LessOrEqual(t, got, after.Add(24*time.Hour))
}

func TestResolveSnoozeTime_Until(t *testing.T) {
	got, err := resolveSnoozeTime("", "2026-06-01T00:00:00Z")
	require.NoError(t, err)

	want, _ := time.Parse(time.RFC3339, "2026-06-01T00:00:00Z")
	assert.Equal(t, want, got)
}

func TestResolveSnoozeTime_InvalidFor(t *testing.T) {
	_, err := resolveSnoozeTime("bad", "")
	assert.Error(t, err)
}

func TestResolveSnoozeTime_InvalidUntil(t *testing.T) {
	_, err := resolveSnoozeTime("", "not-a-time")
	assert.Error(t, err)
}
