package kucoin

import (
	"context"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFuturesContract(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesContract(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesContract must reject an empty symbol")

	for symbol, exp := range map[string]*FuturesContractResponse{
		"XBTUSDTM": {
			Symbol:                             "XBTUSDTM",
			DisplaySymbol:                      "XBTUSDTM",
			RootSymbol:                         currency.USDT,
			Type:                               "FFWCSX",
			FirstOpenDate:                      types.Time(time.Unix(1585555200, 0)),
			ExpireDate:                         types.Time(time.Unix(1798185600, 0)),
			SettleDate:                         types.Time(time.Unix(1798185600, 0)),
			BaseCurrency:                       currency.XBT,
			DisplayBaseCurrency:                "XBT",
			QuoteCurrency:                      currency.USDT,
			SettleCurrency:                     currency.USDT,
			MaxOrderQuantity:                   1000000,
			MarketMaxOrderQuantity:             1000000,
			MaxPrice:                           1000000,
			LotSize:                            1,
			TickSize:                           0.1,
			IndexPriceTickSize:                 0.01,
			Multiplier:                         0.001,
			InitialMargin:                      0.008,
			MaintainMargin:                     0.004,
			MaxRiskLimit:                       250000,
			MinRiskLimit:                       250000,
			RiskStep:                           125000,
			MakerFeeRate:                       0.0002,
			TakerFeeRate:                       0.0006,
			TakerFixFee:                        0.0006,
			MakerFixFee:                        0.0002,
			SettlementFee:                      0.00025,
			IsDeleverage:                       true,
			IsQuanto:                           true,
			MarkMethod:                         "FairPrice",
			FairMethod:                         "FundingRate",
			FundingBaseSymbol:                  ".XBTINT8H",
			FundingQuoteSymbol:                 ".USDTINT8H",
			FundingRateSymbol:                  ".XBTUSDTMFPI8H",
			IndexSymbol:                        ".KXBTUSDT",
			SettlementSymbol:                   ".BXBT30M",
			Status:                             "Open",
			FundingFeeRate:                     0.000051,
			PredictedFundingFeeRate:            0.000061,
			DailyInterestRate:                  0.0003,
			FundingRateGranularity:             28800000,
			FundingRateCap:                     0.003,
			FundingRateFloor:                   -0.003,
			Period:                             1,
			EffectiveFundingRateCycleStartTime: types.Time(time.Unix(1750147200, 0)),
			CurrentFundingRateGranularity:      28800000,
			OpenInterest:                       9510596,
			TurnoverOf24Hour:                   422891571.0466,
			VolumeOf24Hour:                     5112.416,
			MarkPrice:                          83404.8,
			IndexPrice:                         83436.78,
			LastTradePrice:                     83404.8,
			NextFundingRateTime:                9125018,
			NextFundingRateDateTime:            types.Time(time.Unix(1791907200, 0)),
			MaxLeverage:                        125,
			SourceExchanges: []string{
				"okex",
				"binance",
				"kucoin",
				"bybit",
				"gateio",
			},
			PremiumsSymbol1Minute:     ".XBTUSDTMPI",
			PremiumsSymbol8Hour:       ".XBTUSDTMPI8H",
			FundingBaseSymbol1Minute:  ".XBTINT",
			FundingQuoteSymbol1Minute: ".USDTINT",
			LowPrice:                  80985.9,
			HighPrice:                 84133.2,
			PriceChangePercentage:     0.011,
			PriceChange:               915,
			K:                         490,
			M:                         300,
			F:                         1.3,
			MMRLimit:                  0.3,
			MMRLeverageConstant:       125,
			SupportCross:              true,
			BuyLimit:                  87575,
			SellLimit:                 79234.6,
			AdjustK:                   520,
			AdjustM:                   320,
			AdjustMMRLeverageConstant: 100,
			AdjustActiveTime:          types.Time(time.Unix(1792022400, 0)),
			CrossRiskLimit:            1500000000,
			MarketStage:               "NORMAL",
			OrderPriceRange:           0.05,
			MarketType:                "CRYPTO",
			AssetClass:                "CRYPTO",
			LastTimeFundingRate:       0.000072,
		},
		"XBTUSDM": {
			Symbol:                             "XBTUSDM",
			DisplaySymbol:                      "XBTUSDM",
			RootSymbol:                         currency.XBT,
			Type:                               "FFWCSX",
			FirstOpenDate:                      types.Time(time.Unix(1552638575, 0)),
			BaseCurrency:                       currency.XBT,
			DisplayBaseCurrency:                "XBT",
			QuoteCurrency:                      currency.USD,
			SettleCurrency:                     currency.XBT,
			MaxOrderQuantity:                   1000000,
			MarketMaxOrderQuantity:             1000000,
			MaxPrice:                           1000000,
			LotSize:                            1,
			TickSize:                           0.1,
			IndexPriceTickSize:                 0.1,
			Multiplier:                         -1,
			InitialMargin:                      0.014,
			MaintainMargin:                     0.007,
			MaxRiskLimit:                       5,
			MinRiskLimit:                       5,
			RiskStep:                           2,
			MakerFeeRate:                       0.0002,
			TakerFeeRate:                       0.0006,
			IsDeleverage:                       true,
			IsInverse:                          true,
			MarkMethod:                         "FairPrice",
			FairMethod:                         "FundingRate",
			FundingBaseSymbol:                  ".XBTINT8H",
			FundingQuoteSymbol:                 ".USDINT8H",
			FundingRateSymbol:                  ".XBTUSDMFPI8H",
			IndexSymbol:                        ".BXBT",
			Status:                             "Open",
			FundingFeeRate:                     -0.000684,
			DailyInterestRate:                  0.0003,
			FundingRateGranularity:             28800000,
			FundingRateCap:                     0.00525,
			FundingRateFloor:                   -0.00525,
			Period:                             1,
			EffectiveFundingRateCycleStartTime: types.Time(time.Unix(1750147200, 0)),
			CurrentFundingRateGranularity:      28800000,
			OpenInterest:                       52705229,
			TurnoverOf24Hour:                   133.94739609,
			VolumeOf24Hour:                     11107010,
			MarkPrice:                          83001.6,
			IndexPrice:                         83311.49,
			LastTradePrice:                     82927.6,
			NextFundingRateTime:                8002492,
			NextFundingRateDateTime:            types.Time(time.Unix(1791907200, 0)),
			MaxLeverage:                        75,
			SourceExchanges: []string{
				"kraken",
				"bitstamp",
				"crypto",
				"coinbase",
				"binance",
			},
			PremiumsSymbol1Minute:     ".XBTUSDMPI",
			PremiumsSymbol8Hour:       ".XBTUSDMPI8H",
			FundingBaseSymbol1Minute:  ".XBTINT",
			FundingQuoteSymbol1Minute: ".USDINT",
			LowPrice:                  80888,
			HighPrice:                 84027.9,
			PriceChangePercentage:     0.008,
			PriceChange:               664.8,
			K:                         2645000,
			M:                         1640000,
			F:                         1.3,
			MMRLimit:                  0.3,
			MMRLeverageConstant:       75,
			SupportCross:              true,
			BuyLimit:                  87148.5,
			SellLimit:                 78848.7,
			CrossRiskLimit:            100,
			MarketStage:               "NORMAL",
			OrderPriceRange:           0.05,
			MarketType:                "CRYPTO",
			AssetClass:                "CRYPTO",
			LastTimeFundingRate:       0.0001,
		},
		"AAPLUSDTM": {
			Symbol:                             "AAPLUSDTM",
			DisplaySymbol:                      "AAPLUSDTM",
			RootSymbol:                         currency.USDT,
			Type:                               "FFWCSX",
			FirstOpenDate:                      types.Time(time.Unix(1775484600, 0)),
			BaseCurrency:                       currency.NewCode("AAPL"),
			DisplayBaseCurrency:                "AAPL",
			QuoteCurrency:                      currency.USDT,
			SettleCurrency:                     currency.USDT,
			MaxOrderQuantity:                   10000,
			MarketMaxOrderQuantity:             10000,
			MaxPrice:                           1000000,
			LotSize:                            1,
			TickSize:                           0.01,
			IndexPriceTickSize:                 0.01,
			Multiplier:                         0.01,
			InitialMargin:                      0.05,
			MaintainMargin:                     0.025,
			MaxRiskLimit:                       10000,
			MinRiskLimit:                       10000,
			RiskStep:                           5000,
			MakerFeeRate:                       0.0002,
			TakerFeeRate:                       0.0006,
			IsDeleverage:                       true,
			MarkMethod:                         "FairPrice",
			FairMethod:                         "FundingRate",
			FundingBaseSymbol:                  ".AAPLINT8H",
			FundingQuoteSymbol:                 ".USDTINT8H",
			FundingRateSymbol:                  ".AAPLUSDTMFPI8H",
			IndexSymbol:                        ".KAAPLUSDT",
			Status:                             "Open",
			FundingRateGranularity:             28800000,
			FundingRateCap:                     0.02,
			FundingRateFloor:                   -0.02,
			Period:                             1,
			EffectiveFundingRateCycleStartTime: types.Time(time.Unix(1790265600, 0)),
			CurrentFundingRateGranularity:      28800000,
			OpenInterest:                       515376,
			TurnoverOf24Hour:                   484414.7302,
			VolumeOf24Hour:                     1415.22,
			MarkPrice:                          345.87,
			IndexPrice:                         345.97,
			LastTradePrice:                     345.87,
			NextFundingRateTime:                6104249,
			NextFundingRateDateTime:            types.Time(time.Unix(1791907200, 0)),
			MaxLeverage:                        20,
			SourceExchanges: []string{
				"binance_index",
				"binance_futures",
				"okx_index",
				"finnhub",
			},
			PremiumsSymbol1Minute:     ".AAPLUSDTMPI",
			PremiumsSymbol8Hour:       ".AAPLUSDTMPI8H",
			FundingBaseSymbol1Minute:  ".AAPLINT",
			FundingQuoteSymbol1Minute: ".USDTINT",
			LowPrice:                  339.16,
			HighPrice:                 348.94,
			PriceChangePercentage:     0.0102,
			PriceChange:               3.49,
			K:                         1130,
			M:                         700,
			F:                         1.3,
			MMRLimit:                  0.3,
			MMRLeverageConstant:       20,
			SupportCross:              true,
			BuyLimit:                  352.72,
			SellLimit:                 338.88,
			CrossRiskLimit:            1000000,
			MarketStage:               "NORMAL",
			PreMarketToPerpDate:       types.Time(time.Unix(1776089400, 0)),
			OrderPriceRange:           0.02,
			MarketType:                "NASDAQ",
			AssetClass:                "STOCK",
			SubMarketType:             "US.STOCK",
			LastTimeFundingRate:       -0.000126,
		},
	} {
		t.Run(symbol, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesContract(t.Context(), symbol)
			require.NoError(t, err, "GetFuturesContract must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetFuturesContract should decode every field")
				return
			}
			require.NotNil(t, result, "GetFuturesContract must return a contract")
			assert.Equal(t, symbol, result.Symbol, "GetFuturesContract should return the requested contract")
			assert.Equal(t, exp.IsInverse, result.IsInverse, "GetFuturesContract should report whether the contract is inverse")
			assert.Positive(t, result.MarkPrice, "GetFuturesContract should return a mark price")
		})
	}
}

func TestGetFuturesOpenContracts(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesOpenContracts(t.Context())
	require.NoError(t, err, "GetFuturesOpenContracts must not error")
	if mockTests {
		exp := []Contract{
			{
				Symbol:                             "XBTUSDTM",
				DisplaySymbol:                      "XBTUSDTM",
				RootSymbol:                         currency.USDT,
				Type:                               "FFWCSX",
				FirstOpenDate:                      types.Time(time.Unix(1585555200, 0)),
				BaseCurrency:                       currency.XBT,
				DisplayBaseCurrency:                "XBT",
				QuoteCurrency:                      currency.USDT,
				SettleCurrency:                     currency.USDT,
				MaxOrderQuantity:                   1000000,
				MarketMaxOrderQuantity:             1000000,
				MaxPrice:                           1000000,
				LotSize:                            1,
				TickSize:                           0.1,
				IndexPriceTickSize:                 0.01,
				Multiplier:                         0.001,
				InitialMargin:                      0.008,
				MaintainMargin:                     0.004,
				MaxRiskLimit:                       250000,
				MinRiskLimit:                       250000,
				RiskStep:                           125000,
				MakerFeeRate:                       0.0002,
				TakerFeeRate:                       0.0006,
				IsDeleverage:                       true,
				IsQuanto:                           true,
				MarkMethod:                         "FairPrice",
				FairMethod:                         "FundingRate",
				FundingBaseSymbol:                  ".XBTINT8H",
				FundingQuoteSymbol:                 ".USDTINT8H",
				FundingRateSymbol:                  ".XBTUSDTMFPI8H",
				IndexSymbol:                        ".KXBTUSDT",
				Status:                             "Open",
				FundingFeeRate:                     0.000051,
				DailyInterestRate:                  0.0003,
				FundingRateGranularity:             28800000,
				FundingRateCap:                     0.003,
				FundingRateFloor:                   -0.003,
				Period:                             1,
				EffectiveFundingRateCycleStartTime: types.Time(time.Unix(1750147200, 0)),
				CurrentFundingRateGranularity:      28800000,
				OpenInterest:                       9510596,
				TurnoverOf24Hour:                   414990992.737,
				VolumeOf24Hour:                     5016.614,
				MarkPrice:                          83403.06,
				IndexPrice:                         83437.84,
				LastTradePrice:                     83395.7,
				NextFundingRateTime:                9116021,
				NextFundingRateDateTime:            types.Time(time.Unix(1791907200, 0)),
				MaxLeverage:                        125,
				SourceExchanges: []string{
					"okex",
					"binance",
					"kucoin",
					"bybit",
					"gateio",
				},
				PremiumsSymbol1Minute:     ".XBTUSDTMPI",
				PremiumsSymbol8Hour:       ".XBTUSDTMPI8H",
				FundingBaseSymbol1Minute:  ".XBTINT",
				FundingQuoteSymbol1Minute: ".USDTINT",
				LowPrice:                  80985.9,
				HighPrice:                 84133.2,
				PriceChangePercentage:     0.0115,
				PriceChange:               954.9,
				K:                         490,
				M:                         300,
				F:                         1.3,
				MMRLimit:                  0.3,
				MMRLeverageConstant:       125,
				SupportCross:              true,
				BuyLimit:                  87573.2,
				SellLimit:                 79232.9,
				CrossRiskLimit:            1500000000,
				MarketStage:               "NORMAL",
				OrderPriceRange:           0.05,
				MarketType:                "CRYPTO",
				AssetClass:                "CRYPTO",
				LastTimeFundingRate:       0.000072,
			},
			{
				Symbol:                             "ANTHROPICUSDTM",
				DisplaySymbol:                      "ANTHROPICUSDTM",
				RootSymbol:                         currency.USDT,
				Type:                               "FFWCSX",
				FirstOpenDate:                      types.Time(time.Unix(1780380000, 0)),
				BaseCurrency:                       currency.NewCode("ANTHROPIC"),
				DisplayBaseCurrency:                "ANTHROPIC",
				QuoteCurrency:                      currency.USDT,
				SettleCurrency:                     currency.USDT,
				MaxOrderQuantity:                   100000,
				MarketMaxOrderQuantity:             100000,
				MaxPrice:                           1000000,
				LotSize:                            1,
				TickSize:                           0.01,
				IndexPriceTickSize:                 0.01,
				Multiplier:                         0.001,
				InitialMargin:                      0.05,
				MaintainMargin:                     0.025,
				MaxRiskLimit:                       10000,
				MinRiskLimit:                       10000,
				RiskStep:                           5000,
				MakerFeeRate:                       0.0002,
				TakerFeeRate:                       0.0006,
				TakerFixFee:                        0.0006,
				MakerFixFee:                        0.0002,
				SettlementFee:                      0.00025,
				IsDeleverage:                       true,
				MarkMethod:                         "FairPrice",
				FairMethod:                         "FundingRate",
				FundingBaseSymbol:                  ".ANTHROPICINT8H",
				FundingQuoteSymbol:                 ".USDTINT8H",
				FundingRateSymbol:                  ".ANTHROPICUSDTMFPI8H",
				IndexSymbol:                        ".KANTHROPICUSDT",
				Status:                             "Open",
				FundingFeeRate:                     0.00005,
				PredictedFundingFeeRate:            0.000048,
				DailyInterestRate:                  0.00015,
				FundingRateGranularity:             28800000,
				FundingRateCap:                     0.02,
				FundingRateFloor:                   -0.02,
				Period:                             1,
				EffectiveFundingRateCycleStartTime: types.Time(time.Unix(1784131200, 0)),
				CurrentFundingRateGranularity:      28800000,
				OpenInterest:                       291418,
				TurnoverOf24Hour:                   1158193.7749,
				VolumeOf24Hour:                     572.232,
				MarkPrice:                          2010.95,
				IndexPrice:                         2010.6,
				LastTradePrice:                     2010.6,
				NextFundingRateTime:                9116021,
				NextFundingRateDateTime:            types.Time(time.Unix(1791907200, 0)),
				MaxLeverage:                        20,
				SourceExchanges: []string{
					"binance_mark_price",
					"binance_index",
					"kucoin_futures",
				},
				PremiumsSymbol1Minute:     ".ANTHROPICUSDTMPI",
				PremiumsSymbol8Hour:       ".ANTHROPICUSDTMPI8H",
				FundingBaseSymbol1Minute:  ".ANTHROPICINT",
				FundingQuoteSymbol1Minute: ".USDTINT",
				LowPrice:                  2009.65,
				HighPrice:                 2054.49,
				PriceChangePercentage:     -0.0146,
				PriceChange:               -29.81,
				K:                         77.4,
				M:                         50,
				F:                         1.3,
				MMRLimit:                  0.3,
				MMRLeverageConstant:       20,
				SupportCross:              true,
				BuyLimit:                  2051.15,
				SellLimit:                 1970.71,
				AdjustK:                   85,
				AdjustM:                   55,
				AdjustMMRLeverageConstant: 20,
				AdjustActiveTime:          types.Time(time.Unix(1792022400, 0)),
				CrossRiskLimit:            1000000,
				MarketStage:               "PRE_MARKET",
				PreMarketToPerpDate:       types.Time(time.Unix(1793577600, 0)),
				OrderPriceRange:           0.02,
				MarketType:                "NASDAQ",
				AssetClass:                "STOCK",
				SubMarketType:             "US.STOCK",
				LastTimeFundingRate:       0.00005,
			},
			{
				Symbol:                 "XBTMZ26",
				DisplaySymbol:          "XBTMZ26",
				RootSymbol:             currency.XBT,
				Type:                   "FFICSX",
				FirstOpenDate:          types.Time(time.Unix(1790071200, 0)),
				ExpireDate:             types.Time(time.Unix(1798185600, 0)),
				SettleDate:             types.Time(time.Unix(1798185600, 0)),
				BaseCurrency:           currency.XBT,
				DisplayBaseCurrency:    "XBT",
				QuoteCurrency:          currency.USD,
				SettleCurrency:         currency.XBT,
				MaxOrderQuantity:       100000,
				MarketMaxOrderQuantity: 100000,
				MaxPrice:               1000000,
				LotSize:                1,
				TickSize:               0.1,
				IndexPriceTickSize:     0.1,
				Multiplier:             -1,
				InitialMargin:          0.05,
				MaintainMargin:         0.025,
				MaxRiskLimit:           4,
				MinRiskLimit:           4,
				RiskStep:               2,
				MakerFeeRate:           0.0002,
				TakerFeeRate:           0.0006,
				IsDeleverage:           true,
				IsInverse:              true,
				MarkMethod:             "FairPrice",
				IndexSymbol:            ".BXBT",
				SettlementSymbol:       ".BXBT30M",
				Status:                 "Open",
				OpenInterest:           17480,
				TurnoverOf24Hour:       0.00033732,
				VolumeOf24Hour:         28,
				MarkPrice:              84127.8,
				IndexPrice:             83330.69,
				LastTradePrice:         82990,
				MaxLeverage:            20,
				SourceExchanges: []string{
					"kraken",
					"bitstamp",
					"crypto",
					"coinbase",
					"binance",
				},
				PremiumsSymbol1Minute: ".XBTUSDMPI",
				PremiumsSymbol8Hour:   ".XBTUSDMPI8H",
				LowPrice:              82990,
				HighPrice:             83006.8,
				PriceChangePercentage: -0.0002,
				PriceChange:           -16.8,
				K:                     531041,
				M:                     327200,
				F:                     1.3,
				MMRLimit:              0.3,
				MMRLeverageConstant:   20,
				SupportCross:          true,
				BuyLimit:              92501.1,
				SellLimit:             75682.7,
				CrossRiskLimit:        45,
				MarketStage:           "NORMAL",
				OrderPriceRange:       0.1,
				MarketType:            "CRYPTO",
				AssetClass:            "CRYPTO",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesOpenContracts should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesOpenContracts should return contracts")
}

func TestGetFuturesTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTicker(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesTicker must reject an empty symbol")

	result, err := e.GetFuturesTicker(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesTicker must not error")
	if mockTests {
		exp := &FuturesTickerResponse{
			Sequence:     1748391334916,
			Symbol:       "XBTUSDTM",
			Side:         "sell",
			Size:         50,
			TradeID:      "1947309184440",
			Price:        83395.7,
			BestBidPrice: 83406.1,
			BestBidSize:  524,
			BestAskPrice: 83406.2,
			BestAskSize:  467,
			Timestamp:    types.Time(time.UnixMilli(1791898067404)),
		}
		assert.Equal(t, exp, result, "GetFuturesTicker should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesTicker must return a ticker")
	assert.Positive(t, result.Sequence, "Sequence should be positive")
	assert.Equal(t, "XBTUSDTM", result.Symbol, "GetFuturesTicker should return the requested contract")
	assert.Contains(t, []string{"buy", "sell"}, result.Side, "Side should be buy or sell")
	assert.Positive(t, result.Size, "Size should be positive")
	assert.Positive(t, result.Price.Float64(), "Price should be positive")
	assert.Positive(t, result.BestBidPrice.Float64(), "BestBidPrice should be positive")
	assert.Positive(t, result.BestBidSize, "BestBidSize should be positive")
	assert.Positive(t, result.BestAskPrice.Float64(), "BestAskPrice should be positive")
	assert.Positive(t, result.BestAskSize, "BestAskSize should be positive")
	assert.NotEmpty(t, result.TradeID, "TradeID should not be empty")
	assert.WithinRange(t, result.Timestamp.Time(), time.Now().Add(-24*time.Hour), time.Now().Add(time.Hour), "Timestamp should be recent")
}

func TestGetLatestTickersForAllContracts(t *testing.T) {
	t.Parallel()
	result, err := e.GetLatestTickersForAllContracts(t.Context())
	require.NoError(t, err, "GetLatestTickersForAllContracts must not error")
	if mockTests {
		exp := []FuturesTicker{
			{
				Sequence:     1748391351604,
				Symbol:       "XBTUSDTM",
				Side:         "sell",
				Size:         1,
				TradeID:      "1947309219236",
				Price:        83396.1,
				BestBidPrice: 83396.1,
				BestBidSize:  3742,
				BestAskPrice: 83396.2,
				BestAskSize:  1,
				Timestamp:    types.Time(time.UnixMilli(1791898156711)),
			},
			{
				Sequence:     1705052453555,
				Symbol:       "XBTUSDM",
				Side:         "buy",
				Size:         1274,
				TradeID:      "1705952082754",
				Price:        83083.7,
				BestBidPrice: 82949.4,
				BestBidSize:  337,
				BestAskPrice: 83083.7,
				BestAskSize:  328,
				Timestamp:    types.Time(time.UnixMilli(1791898127706)),
			},
			{
				Sequence:     1737004411066,
				Symbol:       "ETHUSDTM",
				Side:         "buy",
				Size:         1,
				TradeID:      "1835165412834",
				Price:        2509.61,
				BestBidPrice: 2509.35,
				BestBidSize:  567,
				BestAskPrice: 2509.36,
				BestAskSize:  340,
				Timestamp:    types.Time(time.UnixMilli(1791898161017)),
			},
		}
		assert.Equal(t, exp, result, "GetLatestTickersForAllContracts should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetLatestTickersForAllContracts should return tickers")
}

func TestGetFuturesOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderbook(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesOrderbook must reject an empty symbol")

	result, err := e.GetFuturesOrderbook(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesOrderbook must not error")
	if mockTests {
		exp := &FuturesOrderbookResponse{
			Sequence: 1748391355096,
			Symbol:   "XBTUSDTM",
			Bids: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 128,
					Price:  83390.3,
				},
				{
					Amount: 109,
					Price:  83390.1,
				},
				{
					Amount: 5,
					Price:  83383.3,
				},
			},
			Asks: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 327,
					Price:  83390.4,
				},
				{
					Amount: 19,
					Price:  83392.1,
				},
				{
					Amount: 46,
					Price:  83395.9,
				},
			},
			Timestamp: types.Time(time.UnixMilli(1791898178522)),
		}
		assert.Equal(t, exp, result, "GetFuturesOrderbook should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesOrderbook must return an order book")
	assert.Equal(t, "XBTUSDTM", result.Symbol, "GetFuturesOrderbook should return the requested contract")
	assert.NotEmpty(t, result.Bids, "GetFuturesOrderbook should return bids")
	assert.NotEmpty(t, result.Asks, "GetFuturesOrderbook should return asks")
}

func TestGetFuturesPartOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPartOrderbook(t.Context(), "", 20)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesPartOrderbook must reject an empty symbol")
	_, err = e.GetFuturesPartOrderbook(t.Context(), "XBTUSDTM", 0)
	require.ErrorIs(t, err, errOrderbookSizeRequired, "GetFuturesPartOrderbook must reject a zero size")

	for size, exp := range map[uint64]*FuturesOrderbookResponse{
		20: {
			Sequence: 1748391355167,
			Symbol:   "XBTUSDTM",
			Bids: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 247,
					Price:  83390.3,
				},
				{
					Amount: 109,
					Price:  83390.1,
				},
			},
			Asks: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 244,
					Price:  83390.4,
				},
				{
					Amount: 19,
					Price:  83392.1,
				},
			},
			Timestamp: types.Time(time.UnixMilli(1791898179032)),
		},
		100: {
			Sequence: 1748391355155,
			Symbol:   "XBTUSDTM",
			Bids: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 347,
					Price:  83390.3,
				},
				{
					Amount: 109,
					Price:  83390.1,
				},
			},
			Asks: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 198,
					Price:  83390.4,
				},
				{
					Amount: 19,
					Price:  83392.1,
				},
			},
			Timestamp: types.Time(time.UnixMilli(1791898178983)),
		},
	} {
		t.Run(strconv.FormatUint(size, 10), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesPartOrderbook(t.Context(), "XBTUSDTM", size)
			require.NoError(t, err, "GetFuturesPartOrderbook must not error")
			if mockTests {
				assert.Equal(t, exp, result, "GetFuturesPartOrderbook should decode every field")
				return
			}
			require.NotNil(t, result, "GetFuturesPartOrderbook must return an order book")
			assert.NotEmpty(t, result.Bids, "GetFuturesPartOrderbook should return bids")
			assert.LessOrEqual(t, uint64(len(result.Bids)), size, "GetFuturesPartOrderbook should return at most size bids")
			assert.NotEmpty(t, result.Asks, "GetFuturesPartOrderbook should return asks")
			assert.LessOrEqual(t, uint64(len(result.Asks)), size, "GetFuturesPartOrderbook should return at most size asks")
		})
	}
}

func TestGetFuturesTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTradeHistory(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesTradeHistory must reject an empty symbol")

	result, err := e.GetFuturesTradeHistory(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesTradeHistory must not error")
	if mockTests {
		exp := []FuturesTrade{
			{
				Sequence:     1947309222386,
				ContractID:   101,
				TradeID:      "1947309222386",
				MakerOrderID: "499478386083643392",
				TakerOrderID: "499478550462717954",
				Timestamp:    types.Time(time.UnixMilli(1791898168297)),
				Size:         113,
				Price:        83400,
				Side:         "buy",
			},
			{
				Sequence:     1947309222207,
				ContractID:   101,
				TradeID:      "1947309222207",
				MakerOrderID: "499478462655025152",
				TakerOrderID: "499478550416580609",
				Timestamp:    types.Time(time.UnixMilli(1791898168284)),
				Size:         1,
				Price:        83396.2,
				Side:         "buy",
			},
			{
				Sequence:     1947309219236,
				ContractID:   101,
				TradeID:      "1947309219236",
				MakerOrderID: "499478437057019904",
				TakerOrderID: "499478501766885376",
				Timestamp:    types.Time(time.UnixMilli(1791898156711)),
				Size:         1,
				Price:        83396.1,
				Side:         "sell",
			},
		}
		assert.Equal(t, exp, result, "GetFuturesTradeHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesTradeHistory should return trades")
}

func TestGetFuturesKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesKline(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesKline must reject a nil request")
	_, err = e.GetFuturesKline(t.Context(), &FuturesKlineRequest{Granularity: kline.ThirtyMin})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesKline must reject an empty symbol")
	_, err = e.GetFuturesKline(t.Context(), &FuturesKlineRequest{Symbol: "XBTUSDTM"})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetFuturesKline must reject an unset granularity")
	for _, granularity := range []kline.Interval{kline.FifteenSecond, kline.Interval(90 * time.Second)} {
		_, err = e.GetFuturesKline(t.Context(), &FuturesKlineRequest{Symbol: "XBTUSDTM", Granularity: granularity})
		require.ErrorIsf(t, err, kline.ErrUnsupportedInterval, "GetFuturesKline must reject granularity %s, which is not in whole minutes", granularity)
	}
	_, err = e.GetFuturesKline(t.Context(), &FuturesKlineRequest{Symbol: "XBTUSDTM", Granularity: kline.ThirtyMin, From: time.UnixMilli(1791853200000), To: time.UnixMilli(1791849600000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesKline must reject a start after the end")

	result, err := e.GetFuturesKline(t.Context(), &FuturesKlineRequest{Symbol: "XBTUSDTM", Granularity: kline.ThirtyMin, From: time.UnixMilli(1791849600000), To: time.UnixMilli(1791853200000)})
	require.NoError(t, err, "GetFuturesKline must not error")
	if mockTests {
		exp := []FuturesKline{
			{
				StartTime: types.Time(time.Unix(1791849600, 0)),
				Open:      81797.9,
				High:      81913,
				Low:       81257.9,
				Close:     81426.3,
				Volume:    143150,
				Turnover:  11660871.255,
			},
			{
				StartTime: types.Time(time.Unix(1791851400, 0)),
				Open:      81426.7,
				High:      81579.4,
				Low:       80985.9,
				Close:     81393.5,
				Volume:    145927,
				Turnover:  11855905.4132,
			},
			{
				StartTime: types.Time(time.Unix(1791853200, 0)),
				Open:      81406.1,
				High:      82169.2,
				Low:       81393.6,
				Close:     82066.2,
				Volume:    96707,
				Turnover:  7916098.3128,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesKline should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetFuturesKline should return candles")
}

func TestFuturesKlineUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result FuturesKline
	require.NoError(t, json.Unmarshal([]byte(`[1757659080000,115328.4,115328.5,115284.2,115284.3,602,69424.9528]`), &result), "Unmarshal must not error")
	exp := FuturesKline{
		StartTime: types.Time(time.UnixMilli(1757659080000)),
		Open:      115328.4,
		High:      115328.5,
		Low:       115284.2,
		Close:     115284.3,
		Volume:    602,
		Turnover:  69424.9528,
	}
	assert.Equal(t, exp, result, "UnmarshalJSON should decode every element in KuCoin's order")
	assert.Error(t, json.Unmarshal([]byte(`[1757659080000,"115328.4"]`), &result), "UnmarshalJSON should reject a quoted price")
}

func TestGetFuturesCurrentMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCurrentMarkPrice(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesCurrentMarkPrice must reject an empty symbol")

	result, err := e.GetFuturesCurrentMarkPrice(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesCurrentMarkPrice must not error")
	if mockTests {
		exp := &FuturesMarkPriceResponse{
			Symbol:      "XBTUSDTM",
			Granularity: 1000,
			TimePoint:   types.Time(time.Unix(1791898263, 0)),
			Value:       83474.6,
			IndexPrice:  83502.25,
		}
		assert.Equal(t, exp, result, "GetFuturesCurrentMarkPrice should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesCurrentMarkPrice must return a mark price")
	assert.Positive(t, result.Value, "GetFuturesCurrentMarkPrice should return a mark price")
}

func TestGetFuturesIndexList(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesIndexList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesIndexList must reject a nil request")

	result, err := e.GetFuturesIndexList(t.Context(), &FuturesHistoryRequest{Symbol: ".KXBTUSDT", Reverse: new(false), Forward: new(false), MaxCount: 10})
	require.NoError(t, err, "GetFuturesIndexList must not error")
	if mockTests {
		exp := &FuturesSpotIndexPriceResponse{
			DataList: []FuturesSpotIndexPrice{
				{
					Symbol:      ".KXBTUSDT",
					Granularity: 1000,
					TimePoint:   types.Time(time.Unix(1791812450, 0)),
					Value:       82766.47,
					DecomposionList: []FuturesSpotIndexComponent{
						{
							Exchange:     "okex",
							ExchangeName: "OKX",
							Price:        82767.7,
							Weight:       0.3293,
						},
						{
							Exchange:     "binance",
							ExchangeName: "Binance",
							Price:        82768.01,
							Weight:       0.4115,
						},
						{
							Exchange:     "kucoin",
							ExchangeName: "KuCoin",
							Price:        82764,
							Weight:       0.0864,
						},
					},
				},
				{
					Symbol:      ".KXBTUSDT",
					Granularity: 1000,
					TimePoint:   types.Time(time.Unix(1791812451, 0)),
					Value:       82767.35,
					DecomposionList: []FuturesSpotIndexComponent{
						{
							Exchange:     "okex",
							ExchangeName: "OKX",
							Price:        82767.7,
							Weight:       0.3293,
						},
						{
							Exchange:     "binance",
							ExchangeName: "Binance",
							Price:        82768.01,
							Weight:       0.4115,
						},
						{
							Exchange:     "kucoin",
							ExchangeName: "KuCoin",
							Price:        82764,
							Weight:       0.0864,
						},
					},
				},
			},
			HasMore: true,
		}
		assert.Equal(t, exp, result, "GetFuturesIndexList should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesIndexList must return a page")
	assert.NotEmpty(t, result.DataList, "GetFuturesIndexList should return index prices")
}

func TestGetFuturesInterestRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesInterestRate(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesInterestRate must reject a nil request")

	result, err := e.GetFuturesInterestRate(t.Context(), &FuturesHistoryRequest{Symbol: ".XBTINT", Reverse: new(false), Forward: new(false)})
	require.NoError(t, err, "GetFuturesInterestRate must not error")
	if mockTests {
		exp := &FuturesIndexHistoryResponse{
			DataList: []FuturesIndexValue{
				{
					Symbol:      ".XBTINT",
					Granularity: 60000,
					TimePoint:   types.Time(time.Unix(1551369600, 0)),
					Value:       0.0003,
				},
				{
					Symbol:      ".XBTINT",
					Granularity: 60000,
					TimePoint:   types.Time(time.Unix(1551369660, 0)),
					Value:       0.0003,
				},
			},
			HasMore: true,
		}
		assert.Equal(t, exp, result, "GetFuturesInterestRate should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesInterestRate must return a page")
	assert.NotEmpty(t, result.DataList, "GetFuturesInterestRate should return interest rates")
}

func TestGetFuturesPremiumIndex(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPremiumIndex(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPremiumIndex must reject a nil request")

	result, err := e.GetFuturesPremiumIndex(t.Context(), &FuturesHistoryRequest{Symbol: ".XBTUSDTMPI", Reverse: new(false), Forward: new(false)})
	require.NoError(t, err, "GetFuturesPremiumIndex must not error")
	if mockTests {
		exp := &FuturesIndexHistoryResponse{
			DataList: []FuturesIndexValue{
				{
					Symbol:      ".XBTUSDTMPI",
					Granularity: 60000,
					TimePoint:   types.Time(time.Unix(1585210560, 0)),
					Value:       0.00006,
				},
				{
					Symbol:      ".XBTUSDTMPI",
					Granularity: 60000,
					TimePoint:   types.Time(time.Unix(1585210620, 0)),
					Value:       -0.000025,
				},
			},
			HasMore: true,
		}
		assert.Equal(t, exp, result, "GetFuturesPremiumIndex should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesPremiumIndex must return a page")
	assert.NotEmpty(t, result.DataList, "GetFuturesPremiumIndex should return premium index values")
}

func TestFuturesHistoryRequestParams(t *testing.T) {
	t.Parallel()
	_, err := (*FuturesHistoryRequest)(nil).params()
	require.ErrorIs(t, err, common.ErrNilPointer, "params must reject a nil request")
	_, err = (&FuturesHistoryRequest{}).params()
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "params must reject an empty symbol")
	_, err = (&FuturesHistoryRequest{Symbol: ".XBTINT", StartAt: time.UnixMilli(1791853200000), EndAt: time.UnixMilli(1791849600000)}).params()
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "params must reject a start after the end")

	result, err := (&FuturesHistoryRequest{Symbol: ".XBTINT"}).params()
	require.NoError(t, err, "params must not error")
	assert.Equal(t, url.Values{"symbol": {".XBTINT"}}, result, "params should leave unset parameters out, including the booleans the server defaults to true")

	result, err = (&FuturesHistoryRequest{
		Symbol:   ".XBTINT",
		StartAt:  time.UnixMilli(1791849600000),
		EndAt:    time.UnixMilli(1791853200000),
		Reverse:  new(false),
		Offset:   1791849660000,
		Forward:  new(true),
		MaxCount: 100,
	}).params()
	require.NoError(t, err, "params must not error")
	exp := url.Values{
		"symbol":   {".XBTINT"},
		"startAt":  {"1791849600000"},
		"endAt":    {"1791853200000"},
		"reverse":  {"false"},
		"offset":   {"1791849660000"},
		"forward":  {"true"},
		"maxCount": {"100"},
	}
	assert.Equal(t, exp, result, "params should send every set parameter")
}

func TestGet24HourFuturesTransactionVolume(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.Get24HourFuturesTransactionVolume(t.Context())
	require.NoError(t, err, "Get24HourFuturesTransactionVolume must not error")
	if mockTests {
		exp := &Futures24HourStatsResponse{
			TurnoverOf24Hour: 1115573341.3273683,
		}
		assert.Equal(t, exp, result, "Get24HourFuturesTransactionVolume should decode every field")
		return
	}
	require.NotNil(t, result, "Get24HourFuturesTransactionVolume must return statistics")
	assert.Positive(t, result.TurnoverOf24Hour, "Get24HourFuturesTransactionVolume should return a turnover")
}

func TestGetFuturesServerTime(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := e.GetFuturesServerTime(ctx)
	require.ErrorIs(t, err, context.Canceled, "GetFuturesServerTime must return the request's error")

	result, err := e.GetFuturesServerTime(t.Context())
	require.NoError(t, err, "GetFuturesServerTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1791898075306), result, "GetFuturesServerTime should decode the server time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "GetFuturesServerTime should return the current time")
}

func TestGetFuturesServiceStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetFuturesServiceStatus(t.Context())
	require.NoError(t, err, "GetFuturesServiceStatus must not error")
	if mockTests {
		exp := &FuturesServiceStatusResponse{
			Message: "Futures trading is running normally",
			Status:  "open",
		}
		assert.Equal(t, exp, result, "GetFuturesServiceStatus should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesServiceStatus must return a status")
	assert.NotEmpty(t, result.Status, "GetFuturesServiceStatus should return a status")
}

func TestGetFuturesTickers(t *testing.T) {
	t.Parallel()
	ex := testInstance(t)
	require.NoError(t, ex.CurrencyPairs.SetAssetEnabled(asset.Futures, false), "SetAssetEnabled must not error")
	_, err := ex.GetFuturesTickers(t.Context())
	require.ErrorIs(t, err, asset.ErrNotEnabled, "GetFuturesTickers must reject a disabled futures asset")
	_, err = e.GetFuturesTickers(cancelledContext(t))
	require.ErrorIs(t, err, context.Canceled, "GetFuturesTickers must return the failed tickers request's error")

	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/allTickers":       `{"code":"200000","data":[]}`,
		"/api/v1/contracts/active": `{"code":"400100","msg":"Request failed"}`,
	})
	_, err = ex.GetFuturesTickers(t.Context())
	require.ErrorContains(t, err, "400100", "GetFuturesTickers must return the failed contracts request's error")

	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/allTickers":       `{"code":"200000","data":[{"symbol":""}]}`,
		"/api/v1/contracts/active": `{"code":"200000","data":[]}`,
	})
	_, err = ex.GetFuturesTickers(t.Context())
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesTickers must reject a ticker without a symbol")

	ex = marketTestServerExchange(t, map[string]string{
		"/api/v1/allTickers": `{"code":"200000","data":[
			{"sequence":1705052453555,"symbol":"XBTUSDM","side":"buy","size":1274,"tradeId":"1705952082754","price":"83083.7","bestBidPrice":"82949.4","bestBidSize":337,"bestAskPrice":"83083.7","bestAskSize":328,"ts":1791898127706000000},
			{"sequence":1737004411066,"symbol":"ETHUSDTM","side":"buy","size":1,"tradeId":"1835165412834","price":"2509.61","bestBidPrice":"2509.35","bestBidSize":567,"bestAskPrice":"2509.36","bestAskSize":340,"ts":1791898161017000000},
			{"sequence":1748391351604,"symbol":"NOSUCHUSDTM","side":"sell","size":1,"tradeId":"1947309219236","price":"1.1","bestBidPrice":"1.1","bestBidSize":1,"bestAskPrice":"1.2","bestAskSize":1,"ts":1791898156711000000}
		]}`,
		"/api/v1/contracts/active": `{"code":"200000","data":[
			{"symbol":"XBTUSDM","baseCurrency":"XBT","isInverse":true,"openInterest":"6874401","turnoverOf24h":32.7393,"volumeOf24h":2564398,"markPrice":83079.33,"indexPrice":83093.28,"lastTradePrice":83083.7,"lowPrice":80923.4,"highPrice":84089.6}
		]}`,
	})
	enabled := currency.Pairs{currency.NewPairWithDelimiter("XBT", "USDM", "_"), currency.NewPairWithDelimiter("ETH", "USDTM", "_")}
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, enabled, false), "StorePairs must not error for available pairs")
	require.NoError(t, ex.CurrencyPairs.StorePairs(asset.Futures, enabled, true), "StorePairs must not error for enabled pairs")
	result, err := ex.GetFuturesTickers(t.Context())
	require.ErrorIs(t, err, errFuturesContractNotFound, "GetFuturesTickers must report a ticker whose contract Get All Symbols does not list")
	exp := []*ticker.Price{{
		Last:         83083.7,
		LastSize:     1274,
		High:         84089.6,
		Low:          80923.4,
		Bid:          82949.4,
		BidSize:      337,
		Ask:          83083.7,
		AskSize:      328,
		BaseVolume:   32.7393, // An inverse contract's turnover is in its base currency
		QuoteVolume:  2564398,
		OpenInterest: 6874401,
		MarkPrice:    83079.33,
		IndexPrice:   83093.28,
		Pair:         currency.NewPairWithDelimiter("XBT", "USDM", "_"),
		ExchangeName: ex.Name,
		AssetType:    asset.Futures,
		LastUpdated:  time.Unix(0, 1791898127706000000),
	}}
	assert.Equal(t, exp, result, "GetFuturesTickers should return the tickers of enabled pairs with their contracts")

	ensureTradablePairs(t)
	ex = testInstance(t)
	result, err = ex.GetFuturesTickers(t.Context())
	require.NoError(t, err, "GetFuturesTickers must not error")
	if mockTests {
		exp = []*ticker.Price{{
			Last:         83396.1,
			LastSize:     1,
			High:         84133.2,
			Low:          80985.9,
			Bid:          83396.1,
			BidSize:      3742,
			Ask:          83396.2,
			AskSize:      1,
			BaseVolume:   5016.614,
			QuoteVolume:  414990992.737,
			OpenInterest: 9510596,
			MarkPrice:    83403.06,
			IndexPrice:   83437.84,
			Pair:         currency.NewPairWithDelimiter("XBT", "USDTM", "_"),
			ExchangeName: ex.Name,
			AssetType:    asset.Futures,
			LastUpdated:  time.UnixMilli(1791898156711),
		}}
		assert.Equal(t, exp, result, "GetFuturesTickers should combine the enabled pair's ticker with its contract's statistics")
		return
	}
	require.NotEmpty(t, result, "GetFuturesTickers must return the enabled pairs' tickers")
	for _, tick := range result {
		assert.Positivef(t, tick.Last, "Last should be positive for %s", tick.Pair)
		assert.Positivef(t, tick.Bid, "Bid should be positive for %s", tick.Pair)
		assert.Positivef(t, tick.Ask, "Ask should be positive for %s", tick.Pair)
		assert.Positivef(t, tick.High, "High should be positive for %s", tick.Pair)
		assert.Positivef(t, tick.MarkPrice, "MarkPrice should be positive for %s", tick.Pair)
	}
}

// TestContractVolumes pins which currency each of KuCoin's two futures volume figures carries. An
// inverse contract is worth one unit of its quote currency, so volumeOf24h counts that currency
// there while turnoverOf24h carries the base, the reverse of a linear contract
func TestContractVolumes(t *testing.T) {
	t.Parallel()
	t.Run("inverse reports the quote currency in volumeOf24h", func(t *testing.T) {
		t.Parallel()
		// XBTUSDM: 2,564,398 USD against 32.7393 XBT at 79,118
		baseVolume, quoteVolume := contractVolumes(&Contract{IsInverse: true, VolumeOf24Hour: 2564398, TurnoverOf24Hour: 32.7393})
		assert.Equal(t, 32.7393, baseVolume, "contractVolumes should return the base currency volume")
		assert.Equal(t, 2564398.0, quoteVolume, "contractVolumes should return the quote currency volume")
	})
	t.Run("linear reports the base currency in volumeOf24h", func(t *testing.T) {
		t.Parallel()
		// XBTUSDTM: 4,837.85 XBT against 379,665,862.5284 USDT
		baseVolume, quoteVolume := contractVolumes(&Contract{VolumeOf24Hour: 4837.85, TurnoverOf24Hour: 379665862.5284})
		assert.Equal(t, 4837.85, baseVolume, "contractVolumes should return the base currency volume")
		assert.Equal(t, 379665862.5284, quoteVolume, "contractVolumes should return the quote currency volume")
	})
}
