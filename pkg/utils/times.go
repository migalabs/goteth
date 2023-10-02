package utils

import "time"

func DurationToFloat64Millis(d time.Duration) float64 {
	return float64(d) / float64(time.Millisecond)
}
