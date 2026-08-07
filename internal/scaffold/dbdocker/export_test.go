package dbdocker

import (
	"context"
	"time"
)

// SkipReadyDelayForTest removes the post-start warm-up wait for the duration of
// a test. Without it every setup test pays the real 5-10 second delay.
func SkipReadyDelayForTest() func() {
	original := sleep
	sleep = func(context.Context, time.Duration) {}
	return func() { sleep = original }
}
