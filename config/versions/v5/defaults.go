package v5

import (
	"strconv"
	"time"
)

// DefaultFuturesTrackingSeekDuration contains the default futures tracking seek duration
// Note: Do not be tempted to use an external package constant for Duration; This is the value at v5 only
var DefaultFuturesTrackingSeekDuration = strconv.FormatInt(int64(time.Hour)*24*365, 10)

// DefaultOrderbookConfig contains the stateless V5 representation of orderbookManager
var DefaultOrderbookConfig = []byte(`{
  "enabled": true,
  "verbose": false,
  "activelyTrackFuturesPositions": true,
  "futuresTrackingSeekDuration": ` + DefaultFuturesTrackingSeekDuration + `,
  "cancelOrdersOnShutdown": false,
  "respectOrderHistoryLimits": true
 }`)
