package ta

import (
	"github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/gctscript/modules/ta/indicators"
)

// Modules map of all loadable modules
var Modules = map[string]map[string]tengo.Object{
	"indicator/bbands":                 indicators.BBandsModule,
	"indicator/macd":                   indicators.MACDModule,
	"indicator/ema":                    indicators.EMAModule,
	"indicator/sma":                    indicators.SMAModule,
	"indicator/rsi":                    indicators.RsiModule,
	"indicator/obv":                    indicators.ObvModule,
	"indicator/mfi":                    indicators.MfiModule,
	"indicator/atr":                    indicators.AtrModule,
	"indicator/correlationcoefficient": indicators.CorrelationCoefficientModule,
}
