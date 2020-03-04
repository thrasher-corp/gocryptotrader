package ta

import (
	"github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/ta/indicators"
)

// Modules map of all loadable modules
var Modules = map[string]map[string]tengo.Object{
	"indicator/rsi":            indicators.RsiModule,
	"indicator/moving-average": indicators.MovingAverageModule,
	"indicator/index":          indicators.IndexModule,
	"indicator/volume":		 	indicators.VolumeModule,
	"indicator/range":			indicators.RangeModule,
}
