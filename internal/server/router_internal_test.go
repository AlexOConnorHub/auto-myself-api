package server

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestResolveTimeouts(t *testing.T) {
	override := &timeoutDurations{defaultTimeout: 100 * time.Millisecond, uploadTimeout: time.Second}

	tests := []struct {
		name     string
		mode     string
		override *timeoutDurations
		want     *timeoutDurations
	}{
		{
			name: "release uses defaults",
			mode: gin.ReleaseMode,
			want: &timeoutDurations{defaultTimeout: 5 * time.Second, uploadTimeout: 2 * time.Minute},
		},
		{
			name: "test mode disables timeouts",
			mode: gin.TestMode,
			want: nil,
		},
		{
			name:     "override wins in release",
			mode:     gin.ReleaseMode,
			override: override,
			want:     override,
		},
		{
			name:     "override wins in test mode",
			mode:     gin.TestMode,
			override: override,
			want:     override,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := resolveTimeouts(test.mode, test.override)
			switch {
			case test.want == nil && got != nil:
				t.Fatalf("expected nil, got %+v", got)
			case test.want != nil && got == nil:
				t.Fatalf("expected %+v, got nil", test.want)
			case test.want != nil && *got != *test.want:
				t.Fatalf("got %+v, want %+v", *got, *test.want)
			}
		})
	}
}

func TestWithTimeoutsRejectsNonPositiveDurations(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("expected WithTimeouts to panic on a non-positive duration")
		}
	}()

	WithTimeouts(0, time.Second)
}
