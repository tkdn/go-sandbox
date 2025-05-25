package clock_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/tkdn/go-sandbox/clock"
)

func TestNewClockEquality(t *testing.T) {
	testCases := []struct {
		name     string
		testTime time.Time
	}{
		{
			name:     "time.Date",
			testTime: time.Date(2025, 12, 15, 17, 00, 30, 123456789, time.UTC),
		},
		{
			name:     "time.Now",
			testTime: time.Now(),
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			got, want := clock.NewClock(tc.testTime), &clock.Clock{
				Hour:   tc.testTime.Hour(),
				Minute: tc.testTime.Minute(),
				Second: tc.testTime.Second(),
				Time:   tc.testTime,
			}

			if err := cmp.Diff(got, want); err != "" {
				t.Errorf("\ngot: %+v\nwant: %+v\n", got, want)
			}

		})
	}
}
