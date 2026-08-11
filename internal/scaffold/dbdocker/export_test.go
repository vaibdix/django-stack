package dbdocker

import (
	"context"
	"time"
)

func SkipReadyDelayForTest() func() {
	original := sleep
	sleep = func(context.Context, time.Duration) {}
	return func() { sleep = original }
}
