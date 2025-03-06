package binance

import (
	"strconv"
	"time"
)

// timeString gets the time as Binance timestamp
func timeString(t time.Time) string {
	return strconv.FormatInt(t.UnixMilli(), 10)
}
