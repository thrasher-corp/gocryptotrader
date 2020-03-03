package ta

import (
	"github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/ta/indicators"
)

// Modules map of all loadable modules
var Modules = map[string]map[string]tengo.Object{
	"rsi":            indicators.RsiModule,
	"moving_average": indicators.MovingAverageModule,
	"index":          indicators.IndexModule,
}
