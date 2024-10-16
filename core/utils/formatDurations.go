package utils

import (
	"fmt"
	"time"
)

func FormatDuration(d time.Duration) string {
	if d >= time.Second {
		return fmt.Sprintf("%.2f s", d.Seconds())
	} else if d >= time.Millisecond {
		return fmt.Sprintf("%.2f ms", float64(d.Microseconds())/1000)
	} else {
		return fmt.Sprintf("%d μs", d.Microseconds())
	}
}
