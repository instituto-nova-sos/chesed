package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCanTransition(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{AttendanceStatusScheduled, AttendanceStatusInProgress, true},
		{AttendanceStatusScheduled, AttendanceStatusCancelled, true},
		{AttendanceStatusScheduled, AttendanceStatusCompleted, false},
		{AttendanceStatusInProgress, AttendanceStatusCompleted, true},
		{AttendanceStatusInProgress, AttendanceStatusCancelled, true},
		{AttendanceStatusInProgress, AttendanceStatusScheduled, false},
		{AttendanceStatusCompleted, AttendanceStatusInProgress, false},
		{AttendanceStatusCompleted, AttendanceStatusCancelled, false},
		{AttendanceStatusCancelled, AttendanceStatusCompleted, false},
		{"UNKNOWN", AttendanceStatusInProgress, false},
		{AttendanceStatusScheduled, "FOLLOW_UP", false}, // Phase 2 reserved
	}

	for _, c := range cases {
		t.Run(c.from+"->"+c.to, func(t *testing.T) {
			assert.Equal(t, c.want, CanTransition(c.from, c.to))
		})
	}
}
