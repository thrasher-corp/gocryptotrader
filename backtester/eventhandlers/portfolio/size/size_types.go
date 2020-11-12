package size

type Size struct {
	MinimumBuySize  float64
	MaximumBuySize  float64
	DefaultBuySize  float64
	MinimumSellSize float64
	MaximumSellSize float64
	DefaultSellSize float64
	CanUseLeverage  bool
	MaximumLeverage float64
}
