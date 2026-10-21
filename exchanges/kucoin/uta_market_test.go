package kucoin

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// utaTestStart and utaTestEnd bound the fixed window most UTA market tests query, so live and mock requests match
var (
	utaTestStart = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	utaTestEnd   = time.Date(2026, 10, 2, 0, 0, 0, 0, time.UTC)
)

func TestGetUTAAnnouncements(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAAnnouncements(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAAnnouncements must reject a nil request")
	_, err = e.GetUTAAnnouncements(t.Context(), &UTAAnnouncementsRequest{StartTime: utaTestEnd, EndTime: utaTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAAnnouncements must reject a start after the end")

	result, err := e.GetUTAAnnouncements(t.Context(), &UTAAnnouncementsRequest{
		Language:   "en_US",
		Type:       "latest-announcements",
		PageNumber: 1,
		PageSize:   2,
		StartTime:  time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err, "GetUTAAnnouncements must not error")
	if mockTests {
		exp := &UTAAnnouncementsResponse{
			TotalNumber: 31,
			TotalPage:   16,
			PageNumber:  1,
			PageSize:    2,
			List: []UTAAnnouncement{
				{
					ID:    330351,
					Title: "USDG Flexible Promotion, Enjoy an APR of 6%!",
					Type: []string{
						"latest-announcements",
					},
					Description: "USDG Flexible Promotion, Enjoy an APR of 6%!",
					ReleaseTime: types.Time(time.Unix(1788782400, 0)),
					Language:    "en_US",
					URL:         "https://www.kucoin.com/announcement/USDG_Flexible_0907?lang=en_US",
				},
				{
					ID:    329762,
					Title: "KuCoin AEON Creator Task Rewards Distribution Notice",
					Type: []string{
						"latest-announcements",
					},
					Description: "The KuCoin AEON Creator Task has officially concluded, and rewards have now been distributed to eligible creators.",
					ReleaseTime: types.Time(time.Unix(1788776340, 0)),
					Language:    "en_US",
					URL:         "https://www.kucoin.com/announcement/en-kucoin-aeon-creator-task-rewards-distribution-notice?lang=en_US",
				},
			},
		}
		assert.Equal(t, exp, result, "GetUTAAnnouncements should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetUTAAnnouncements should return announcements")
}

func TestGetUTACurrency(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTACurrency(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUTACurrency must reject an empty currency")

	for _, tc := range []struct {
		name  string
		ccy   currency.Code
		chain string
		exp   *UTACurrencyResponse
	}{
		{name: "every chain", ccy: currency.USDT, exp: &UTACurrencyResponse{
			Currency:        currency.USDT,
			Name:            "USDT",
			FullName:        "Tether",
			Precision:       8,
			Confirmations:   64,
			ContractAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			IsMarginEnabled: true,
			IsDebitEnabled:  true,
			List: []UTACurrencyChain{
				{
					Chain:             "eth",
					ChainName:         "ERC20",
					Confirmations:     64,
					PreConfirmations:  12,
					ContractAddress:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
					IsDepositEnabled:  true,
					IsWithdrawEnabled: true,
					AddressRegex:      "^(0x)[0-9A-Fa-f]{40}$",
					DepositFeeRate:    0.001,
					DepositTierFee:    "0.5",
					FixedDepositFee:   1,
					MaxDepositFee:     5,
					MinDepositSize:    1,
					MaxDepositSize:    10000000,
					WithdrawFeeRate:   0.002,
					WithdrawPrecision: 6,
					MinWithdrawSize:   30,
					MaxWithdrawSize:   5000000,
					MinWithdrawFee:    5.5,
					MaxWithdrawFee:    36,
				},
			},
		}},
		{name: "one chain", ccy: currency.XRP, chain: "xrp", exp: &UTACurrencyResponse{
			Currency:        currency.XRP,
			Name:            "XRP",
			FullName:        "Ripple",
			Precision:       8,
			IsMarginEnabled: true,
			IsDebitEnabled:  true,
			List: []UTACurrencyChain{
				{
					Chain:             "xrp",
					ChainName:         "XRP",
					Confirmations:     12,
					PreConfirmations:  12,
					IsDepositEnabled:  true,
					IsWithdrawEnabled: true,
					IsMemoRequired:    true,
					AddressRegex:      "^r[1-9A-HJ-NP-Za-km-z]{25,34}$",
					MemoRegex:         "^(\\d{0,100})$",
					MinDepositSize:    0.1,
					WithdrawPrecision: 6,
					MinWithdrawSize:   14,
					MinWithdrawFee:    0.3,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTACurrency(t.Context(), tc.ccy, tc.chain)
			require.NoError(t, err, "GetUTACurrency must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTACurrency should decode every field")
				return
			}
			assert.Equal(t, tc.ccy, result.Currency, "GetUTACurrency should return the currency asked for")
			assert.NotEmpty(t, result.List, "GetUTACurrency should return chains")
		})
	}
}

func TestGetUTACurrencies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		chain      string
		currencies []currency.Code
		exp        []UTACurrencyResponse
	}{
		{name: "every chain", currencies: []currency.Code{currency.USDT, currency.XRP}, exp: []UTACurrencyResponse{
			{
				Currency:        currency.USDT,
				Name:            "USDT",
				FullName:        "Tether",
				Precision:       8,
				Confirmations:   64,
				ContractAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
				IsMarginEnabled: true,
				IsDebitEnabled:  true,
				List: []UTACurrencyChain{
					{
						Chain:             "eth",
						ChainName:         "ERC20",
						Confirmations:     64,
						PreConfirmations:  12,
						ContractAddress:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
						IsDepositEnabled:  true,
						IsWithdrawEnabled: true,
						AddressRegex:      "^(0x)[0-9A-Fa-f]{40}$",
						DepositFeeRate:    0.001,
						DepositTierFee:    "0.5",
						FixedDepositFee:   1,
						MaxDepositFee:     5,
						MinDepositSize:    1,
						MaxDepositSize:    10000000,
						WithdrawFeeRate:   0.002,
						WithdrawPrecision: 6,
						MinWithdrawSize:   30,
						MaxWithdrawSize:   5000000,
						MinWithdrawFee:    5.5,
						MaxWithdrawFee:    36,
					},
				},
			},
			{
				Currency:        currency.XRP,
				Name:            "XRP",
				FullName:        "Ripple",
				Precision:       8,
				IsMarginEnabled: true,
				IsDebitEnabled:  true,
				List: []UTACurrencyChain{
					{
						Chain:             "xrp",
						ChainName:         "XRP",
						Confirmations:     12,
						PreConfirmations:  12,
						IsDepositEnabled:  true,
						IsWithdrawEnabled: true,
						IsMemoRequired:    true,
						AddressRegex:      "^r[1-9A-HJ-NP-Za-km-z]{25,34}$",
						MemoRegex:         "^(\\d{0,100})$",
						MinDepositSize:    0.1,
						WithdrawPrecision: 6,
						MinWithdrawSize:   14,
						MinWithdrawFee:    0.3,
					},
				},
			},
		}},
		{name: "one chain", chain: "trx", currencies: []currency.Code{currency.USDT}, exp: []UTACurrencyResponse{
			{
				Currency:        currency.USDT,
				Name:            "USDT",
				FullName:        "Tether",
				Precision:       8,
				IsMarginEnabled: true,
				IsDebitEnabled:  true,
				List: []UTACurrencyChain{
					{
						Chain:             "trx",
						ChainName:         "TRC20",
						Confirmations:     3,
						PreConfirmations:  3,
						ContractAddress:   "TR7NHqjeKQxGTCi8q8ZY4pL8otSzgjLj6t",
						IsDepositEnabled:  true,
						IsWithdrawEnabled: true,
						AddressRegex:      "^T[1-9A-HJ-NP-Za-km-z]{33}$",
						MinDepositSize:    1,
						WithdrawPrecision: 6,
						MinWithdrawSize:   4,
						MinWithdrawFee:    1.99,
					},
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTACurrencies(t.Context(), tc.chain, tc.currencies)
			require.NoError(t, err, "GetUTACurrencies must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTACurrencies should decode every field")
				return
			}
			assert.Len(t, result, len(tc.currencies), "GetUTACurrencies should return each currency asked for")
		})
	}
}

func TestGetUTATicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTATicker(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTATicker must reject an empty trade type")

	for _, tc := range []struct {
		tradeType, symbol string
		exp               *UTATickerResponse
	}{
		{tradeType: "SPOT", symbol: "BTC-USDT", exp: &UTATickerResponse{
			TradeType: "SPOT",
			Timestamp: types.Time(time.UnixMilli(1791898254567)),
			List: []UTATicker{
				{
					Symbol:             "BTC-USDT",
					Name:               "BTC-USDT",
					BestBidPrice:       83494.7,
					BestBidSize:        0.69202077,
					BestAskPrice:       83494.8,
					BestAskSize:        0.004,
					Open:               82558.1,
					High:               84149,
					Low:                81035.5,
					BaseVolume:         6188.159957745228,
					QuoteVolume:        512843954.4099426,
					LastPrice:          83494.7,
					Size:               0.00163544,
					PriceChange:        936.6,
					PriceChangePercent: 0.0113,
				},
			},
		}},
		{tradeType: "FUTURES", symbol: "XBTUSDTM", exp: &UTATickerResponse{
			TradeType: "FUTURES",
			Timestamp: types.Time(time.UnixMilli(1791898254839)),
			List: []UTATicker{
				{
					Symbol:             "XBTUSDTM",
					BestBidPrice:       83477.6,
					BestBidSize:        423,
					BestAskPrice:       83477.7,
					BestAskSize:        525,
					Open:               82526.7,
					High:               84133.2,
					Low:                80985.9,
					BaseVolume:         4956.14,
					QuoteVolume:        410000152.3762,
					LastPrice:          83473.7,
					Size:               7,
					PriceChange:        947,
					PriceChangePercent: 1.1475,
					IndexPrice:         83507.08,
					MarkPrice:          83473.7,
					Sequence:           1748391384877,
					Side:               "sell",
					TradeID:            "1947309277512",
					Timestamp:          types.Time(time.UnixMilli(1791898253295)),
				},
			},
		}},
	} {
		t.Run(tc.tradeType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTATicker(t.Context(), tc.tradeType, tc.symbol)
			require.NoError(t, err, "GetUTATicker must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTATicker should decode every field")
				return
			}
			require.Len(t, result.List, 1, "GetUTATicker must return the symbol's ticker")
			assert.Positive(t, result.List[0].LastPrice.Float64(), "GetUTATicker should return a last price")
		})
	}
}

func TestGetUTASymbol(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTASymbol(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTASymbol must reject an empty trade type")

	for _, tc := range []struct {
		name              string
		tradeType, symbol string
		exp               *UTASymbolResponse
	}{
		{name: "every spot symbol", tradeType: "SPOT", exp: &UTASymbolResponse{
			TradeType: "SPOT",
			List: []UTASymbol{
				{
					Symbol:              "BTC-USDT",
					Name:                "BTC-USDT",
					BaseCurrency:        currency.BTC,
					QuoteCurrency:       currency.USDT,
					Market:              "USDS",
					FeeCurrency:         currency.USDT,
					TradingStatus:       "TradingEnabled",
					PriceLimitRatio:     0.01,
					FeeCategory:         "classA",
					MakerFeeCoefficient: 1,
					TakerFeeCoefficient: 1,
					MinBaseOrderSize:    0.00001,
					MaxBaseOrderSize:    10000000000,
					BaseOrderStep:       0.00000001,
					MinQuoteOrderSize:   0.1,
					MaxQuoteOrderSize:   99999999,
					QuoteOrderStep:      0.000001,
					TickSize:            0.1,
					MinFunds:            0.1,
				},
				{
					Symbol:              "POLS-USDT",
					Name:                "POLS-USDT",
					BaseCurrency:        currency.NewCode("POLS"),
					QuoteCurrency:       currency.USDT,
					Market:              "USDS",
					FeeCurrency:         currency.USDT,
					TradingStatus:       "TradingEnabled",
					PriceLimitRatio:     0.08,
					FeeCategory:         "classB",
					MakerFeeCoefficient: 1,
					TakerFeeCoefficient: 1,
					ST:                  true,
					MinBaseOrderSize:    0.1,
					MaxBaseOrderSize:    10000000000,
					BaseOrderStep:       0.0001,
					MinQuoteOrderSize:   0.1,
					MaxQuoteOrderSize:   99999999,
					QuoteOrderStep:      0.0001,
					TickSize:            0.0001,
					MinFunds:            0.1,
					TradingStartTime:    types.Time(time.Unix(1622800800, 0)),
				},
				{
					Symbol:                          "VAYLA-USDT",
					Name:                            "VAYLA-USDT",
					BaseCurrency:                    currency.NewCode("VAYLA"),
					QuoteCurrency:                   currency.USDT,
					Market:                          "USDS",
					FeeCurrency:                     currency.USDT,
					TradingStatus:                   "TradingEnabled",
					PriceLimitRatio:                 0.05,
					FeeCategory:                     "classC",
					MakerFeeCoefficient:             1,
					TakerFeeCoefficient:             1,
					MinBaseOrderSize:                10,
					MaxBaseOrderSize:                10000000000,
					BaseOrderStep:                   0.1,
					MinQuoteOrderSize:               0.1,
					MaxQuoteOrderSize:               99999999,
					QuoteOrderStep:                  0.01,
					TickSize:                        0.00001,
					MinFunds:                        0.1,
					CallAuctionIsEnabled:            true,
					CallAuctionPriceFloor:           0.00001,
					CallAuctionPriceCeiling:         1,
					CallAuctionFirstStageStartTime:  types.Time(time.Unix(1791882000, 0)),
					CallAuctionSecondStageStartTime: types.Time(time.Unix(1791885000, 0)),
					CallAuctionThirdStageStartTime:  types.Time(time.Unix(1791885300, 0)),
					TradingStartTime:                types.Time(time.Unix(1791885600, 0)),
				},
			},
		}},
		{name: "one futures contract", tradeType: "FUTURES", symbol: "XBTUSDTM", exp: &UTASymbolResponse{
			TradeType: "FUTURES",
			List: []UTASymbol{
				{
					Symbol:              "XBTUSDTM",
					Name:                "XBTUSDTM",
					BaseCurrency:        currency.NewCode("XBT"),
					QuoteCurrency:       currency.USDT,
					TradingStatus:       "Open",
					PriceLimitRatio:     0.05,
					MaxBaseOrderSize:    1000000,
					TickSize:            0.1,
					TradingStartTime:    types.Time(time.Unix(1585555200, 0)),
					MaxLeverage:         125,
					BuyLimit:            87651.6,
					DisplayBaseCurrency: currency.NewCode("XBT"),
					LotSize:             1,
					MaxMarketOrderSize:  1000000,
					MakerFeeRate:        0.0002,
					MarketStage:         "NORMAL",
					MarketType:          "CRYPTO",
					AssetClass:          "CRYPTO",
					MaxPrice:            1000000,
					UnitSize:            0.001,
					IndexSourceExchanges: []string{
						"okex",
						"binance",
						"kucoin",
						"bybit",
						"gateio",
					},
					TakerFeeRate:        0.0006,
					SettlementCurrency:  currency.USDT,
					SellLimit:           79303.8,
					IndexPriceTickSize:  0.01,
					ContractType:        "0",
					K:                   490,
					M:                   300,
					F:                   1.3,
					MMRLimit:            0.3,
					MMRLeverageConstant: 125,
				},
			},
		}},
		{name: "every futures contract", tradeType: "FUTURES", exp: &UTASymbolResponse{
			TradeType: "FUTURES",
			List: []UTASymbol{
				{
					Symbol:              "XBTMZ26",
					Name:                "XBTMZ26",
					BaseCurrency:        currency.NewCode("XBT"),
					QuoteCurrency:       currency.USD,
					TradingStatus:       "Open",
					PriceLimitRatio:     0.1,
					MaxBaseOrderSize:    100000,
					TickSize:            0.1,
					TradingStartTime:    types.Time(time.Unix(1790071200, 0)),
					MaxLeverage:         20,
					BuyLimit:            92608.3,
					DisplayBaseCurrency: currency.NewCode("XBT"),
					ExpiryTime:          types.Time(time.Unix(1798185600, 0)),
					IsInverse:           true,
					LotSize:             1,
					MaxMarketOrderSize:  100000,
					MakerFeeRate:        0.0002,
					MarketStage:         "NORMAL",
					MarketType:          "CRYPTO",
					AssetClass:          "CRYPTO",
					MaxPrice:            1000000,
					UnitSize:            1,
					IndexSourceExchanges: []string{
						"kraken",
						"bitstamp",
						"crypto",
						"coinbase",
						"binance",
					},
					TakerFeeRate:        0.0006,
					SettlementFeeRate:   0.00025,
					SettlementTime:      types.Time(time.Unix(1798185600, 0)),
					SettlementCurrency:  currency.NewCode("XBT"),
					SellLimit:           75770.5,
					IndexPriceTickSize:  0.1,
					ContractType:        "1",
					K:                   531041,
					M:                   327200,
					F:                   1.3,
					MMRLimit:            0.3,
					MMRLeverageConstant: 20,
				},
				{
					Symbol:              "AAPLUSDTM",
					Name:                "AAPLUSDTM",
					BaseCurrency:        currency.NewCode("AAPL"),
					QuoteCurrency:       currency.USDT,
					TradingStatus:       "Open",
					PriceLimitRatio:     0.02,
					MaxBaseOrderSize:    10000,
					TickSize:            0.01,
					TradingStartTime:    types.Time(time.Unix(1775484600, 0)),
					MaxLeverage:         20,
					BuyLimit:            352.37,
					DisplayBaseCurrency: currency.NewCode("AAPL"),
					LotSize:             1,
					MaxMarketOrderSize:  10000,
					MakerFeeRate:        0.0002,
					MarketStage:         "NORMAL",
					MarketType:          "NASDAQ",
					AssetClass:          "STOCK",
					SubMarketType:       "US.STOCK",
					MaxPrice:            1000000,
					UnitSize:            0.01,
					IndexSourceExchanges: []string{
						"binance_index",
						"binance_futures",
						"okx_index",
						"finnhub",
					},
					TakerFeeRate:        0.0006,
					SettlementCurrency:  currency.USDT,
					SellLimit:           338.55,
					IndexPriceTickSize:  0.01,
					ContractType:        "0",
					K:                   1130,
					M:                   700,
					F:                   1.3,
					MMRLimit:            0.3,
					MMRLeverageConstant: 20,
				},
				{
					Symbol:              "ANTHROPICUSDTM",
					Name:                "ANTHROPICUSDTM",
					BaseCurrency:        currency.NewCode("ANTHROPIC"),
					QuoteCurrency:       currency.USDT,
					TradingStatus:       "Open",
					PriceLimitRatio:     0.02,
					MaxBaseOrderSize:    100000,
					TickSize:            0.01,
					TradingStartTime:    types.Time(time.Unix(1780380000, 0)),
					MaxLeverage:         20,
					BuyLimit:            2050.79,
					DisplayBaseCurrency: currency.NewCode("ANTHROPIC"),
					LotSize:             1,
					MaxMarketOrderSize:  100000,
					MakerFeeRate:        0.0002,
					MarketStage:         "PRE_MARKET",
					MarketType:          "NASDAQ",
					AssetClass:          "STOCK",
					SubMarketType:       "US.STOCK",
					MaxPrice:            1000000,
					PreMarketToPerpDate: types.Time(time.Unix(1793433600, 0)),
					UnitSize:            0.001,
					IndexSourceExchanges: []string{
						"binance_mark_price",
						"binance_index",
						"kucoin_futures",
					},
					TakerFeeRate:        0.0006,
					SettlementCurrency:  currency.USDT,
					SellLimit:           1970.37,
					IndexPriceTickSize:  0.01,
					ContractType:        "0",
					K:                   77.4,
					M:                   50,
					F:                   1.3,
					MMRLimit:            0.3,
					MMRLeverageConstant: 20,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTASymbol(t.Context(), tc.tradeType, tc.symbol)
			require.NoError(t, err, "GetUTASymbol must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTASymbol should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetUTASymbol should return symbols")
		})
	}
}

func TestGetUTAKlines(t *testing.T) {
	t.Parallel()
	start, end := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 10, 1, 2, 0, 0, 0, time.UTC)
	_, err := e.GetUTAKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAKlines must reject a nil request")
	_, err = e.GetUTAKlines(t.Context(), &UTAKlinesRequest{TradeType: "SPOT", Interval: "1hour"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTAKlines must reject an empty symbol")
	_, err = e.GetUTAKlines(t.Context(), &UTAKlinesRequest{Symbol: "BTC-USDT", Interval: "1hour"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTAKlines must reject an empty trade type")
	_, err = e.GetUTAKlines(t.Context(), &UTAKlinesRequest{Symbol: "BTC-USDT", TradeType: "SPOT"})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetUTAKlines must reject an empty interval")
	_, err = e.GetUTAKlines(t.Context(), &UTAKlinesRequest{Symbol: "BTC-USDT", TradeType: "SPOT", Interval: "1hour", StartAt: end, EndAt: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAKlines must reject a start after the end")

	for _, tc := range []struct {
		name string
		req  *UTAKlinesRequest
		exp  *UTAKlinesResponse
	}{
		{
			name: "spot",
			req:  &UTAKlinesRequest{Symbol: "BTC-USDT", TradeType: "SPOT", Interval: "1hour", StartAt: start, EndAt: end},
			exp: &UTAKlinesResponse{
				TradeType: "SPOT",
				Symbol:    "BTC-USDT",
				List: []UTAKline{
					{
						StartTime:  types.Time(time.Unix(1790816400, 0)),
						OpenPrice:  83514.3,
						HighPrice:  83680.6,
						LowPrice:   83491.7,
						ClosePrice: 83520.4,
						Volume:     97.25781432,
						Amount:     8127663.165194624,
					},
					{
						StartTime:  types.Time(time.Unix(1790812800, 0)),
						OpenPrice:  83609.9,
						HighPrice:  83639.1,
						LowPrice:   83437.3,
						ClosePrice: 83514.4,
						Volume:     116.52778808,
						Amount:     9733398.873544538,
					},
				},
			},
		},
		{
			name: "futures mark price",
			req:  &UTAKlinesRequest{Symbol: "XBTUSDTM", TradeType: "FUTURES", KlineType: "MARK_PRICE", Interval: "1hour", StartAt: start, EndAt: end},
			exp: &UTAKlinesResponse{
				TradeType: "FUTURES",
				KlineType: "MARK_PRICE",
				Symbol:    "XBTUSDTM",
				List: []UTAKline{
					{
						StartTime:  types.Time(time.Unix(1790820000, 0)),
						OpenPrice:  83474.05,
						HighPrice:  83628.04,
						LowPrice:   83375.8,
						ClosePrice: 83467.3,
					},
					{
						StartTime:  types.Time(time.Unix(1790816400, 0)),
						OpenPrice:  83480,
						HighPrice:  83656.2,
						LowPrice:   83454,
						ClosePrice: 83474.03,
					},
					{
						StartTime:  types.Time(time.Unix(1790812800, 0)),
						OpenPrice:  83580.48,
						HighPrice:  83613.67,
						LowPrice:   83387.78,
						ClosePrice: 83480,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAKlines(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAKlines must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAKlines should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetUTAKlines should return candles")
		})
	}
}

func TestUTAKlineUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var k UTAKline
	require.NoError(t, k.UnmarshalJSON([]byte(`[1790816400,"83514.3","83680.6","83491.7","83520.4","97.25781432","8127663.165194624"]`)), "UnmarshalJSON must not error")
	exp := UTAKline{
		StartTime:  types.Time(time.Unix(1790816400, 0)),
		OpenPrice:  83514.3,
		HighPrice:  83680.6,
		LowPrice:   83491.7,
		ClosePrice: 83520.4,
		Volume:     97.25781432,
		Amount:     8127663.165194624,
	}
	assert.Equal(t, exp, k, "UnmarshalJSON should decode every element")
	assert.Error(t, k.UnmarshalJSON([]byte(`{"startTime":1790816400}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, k.UnmarshalJSON([]byte(`[1790816400,"bad"]`)), "UnmarshalJSON should reject an invalid number")
}

func TestGetUTAOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAOrderbook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAOrderbook must reject a nil request")
	_, err = e.GetUTAOrderbook(t.Context(), &UTAOrderbookRequest{Symbol: "BTC-USDT", Limit: "20"})
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTAOrderbook must reject an empty trade type")
	_, err = e.GetUTAOrderbook(t.Context(), &UTAOrderbookRequest{TradeType: "SPOT", Limit: "20"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTAOrderbook must reject an empty symbol")
	_, err = e.GetUTAOrderbook(t.Context(), &UTAOrderbookRequest{TradeType: "SPOT", Symbol: "BTC-USDT"})
	require.ErrorIs(t, err, errInvalidLimit, "GetUTAOrderbook must reject an empty limit")
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}

	for _, tc := range []struct {
		name string
		req  *UTAOrderbookRequest
		exp  *UTAOrderbookResponse
	}{
		{name: "spot", req: &UTAOrderbookRequest{TradeType: "SPOT", Symbol: "BTC-USDT", Limit: "20"}, exp: &UTAOrderbookResponse{
			TradeType: "SPOT",
			Symbol:    "BTC-USDT",
			Sequence:  27275344291,
			Bids: []UTAOrderbookLevel{
				{
					Price: 92671,
					Size:  0.82674146,
				},
				{
					Price: 92670.9,
					Size:  0.57100305,
				},
			},
			Asks: []UTAOrderbookLevel{
				{
					Price: 92671.1,
					Size:  0.13163929,
				},
				{
					Price: 92671.2,
					Size:  0.0004,
				},
			},
		}},
		{name: "futures RPI", req: &UTAOrderbookRequest{TradeType: "FUTURES", Symbol: "XBTUSDTM", Limit: "20", RPIFilter: true}, exp: &UTAOrderbookResponse{
			TradeType: "FUTURES",
			Symbol:    "XBTUSDTM",
			Sequence:  1748391384877,
			Bids: []UTAOrderbookLevel{
				{
					Price:   83477.6,
					Size:    420,
					RPISize: 3,
				},
				{
					Price: 83477.5,
					Size:  25,
				},
			},
			Asks: []UTAOrderbookLevel{
				{
					Price:   83477.7,
					Size:    520,
					RPISize: 5,
				},
				{
					Price: 83477.8,
					Size:  12,
				},
			},
			Timestamp: types.Time(time.Unix(0, 1768802285598000000)),
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAOrderbook(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAOrderbook must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAOrderbook should decode every field")
				return
			}
			assert.NotEmpty(t, result.Bids, "GetUTAOrderbook should return bids")
			assert.NotEmpty(t, result.Asks, "GetUTAOrderbook should return asks")
		})
	}
}

func TestUTAOrderbookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var l UTAOrderbookLevel
	require.NoError(t, l.UnmarshalJSON([]byte(`["83477.6","420","3"]`)), "UnmarshalJSON must not error on an RPI level")
	assert.Equal(t, UTAOrderbookLevel{Price: 83477.6, Size: 420, RPISize: 3}, l, "UnmarshalJSON should decode every element of an RPI level")
	require.NoError(t, l.UnmarshalJSON([]byte(`["92671","0.82674146"]`)), "UnmarshalJSON must not error on a plain level")
	assert.Equal(t, UTAOrderbookLevel{Price: 92671, Size: 0.82674146}, l, "UnmarshalJSON should decode a plain level and clear the RPI size")
	assert.Error(t, l.UnmarshalJSON([]byte(`{"price":"92671"}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, l.UnmarshalJSON([]byte(`["92671","bad"]`)), "UnmarshalJSON should reject an invalid number")
	assert.ErrorIs(t, l.UnmarshalJSON([]byte(`["92671"]`)), errUTAOrderbookLevelLength, "UnmarshalJSON should reject a level without a size")
}

func TestGetUTAIndexPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAIndexPrice(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAIndexPrice must reject a nil request")
	_, err = e.GetUTAIndexPrice(t.Context(), &UTAIndexPriceRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTAIndexPrice must reject an empty symbol")
	_, err = e.GetUTAIndexPrice(t.Context(), &UTAIndexPriceRequest{Symbol: "XBTUSDTM", StartAt: utaTestEnd, EndAt: utaTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAIndexPrice must reject a start after the end")

	t.Run("latest", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetUTAIndexPrice(t.Context(), &UTAIndexPriceRequest{Symbol: "XBTUSDTM", PageSize: 2})
		require.NoError(t, err, "GetUTAIndexPrice must not error")
		if mockTests {
			exp := &UTAIndexPriceResponse{
				Items: []UTAIndexPrice{
					{
						Symbol:      "XBTUSDTM",
						Granularity: 1000,
						Timestamp:   types.Time(time.Unix(1791898338, 0)),
						IndexPrice:  83419.5,
						DecompositionList: []UTAIndexPriceDecomposition{
							{
								Exchange: "binance",
								Price:    83410,
								Weight:   0.4115,
							},
							{
								Exchange: "bybit",
								Price:    83424.6,
								Weight:   0.0864,
							},
						},
					},
					{
						Symbol:      "XBTUSDTM",
						Granularity: 1000,
						Timestamp:   types.Time(time.Unix(1791898337, 0)),
						IndexPrice:  83410.34,
						DecompositionList: []UTAIndexPriceDecomposition{
							{
								Exchange: "binance",
								Price:    83410,
								Weight:   0.4115,
							},
							{
								Exchange: "bybit",
								Price:    83413.9,
								Weight:   0.0864,
							},
						},
					},
				},
				LastID: 1791898337000,
			}
			assert.Equal(t, exp, result, "GetUTAIndexPrice should decode every field")
			return
		}
		assert.NotEmpty(t, result.Items, "GetUTAIndexPrice should return index prices")
	})
	t.Run("window", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetUTAIndexPrice(t.Context(), &UTAIndexPriceRequest{
			Symbol:   "XBTUSDTM",
			StartAt:  utaTestStart,
			EndAt:    utaTestStart.Add(10 * time.Second),
			PageSize: 2,
			LastID:   1790812809000,
		})
		require.NoError(t, err, "GetUTAIndexPrice must not error")
		if mockTests {
			exp := &UTAIndexPriceResponse{
				Items: []UTAIndexPrice{
					{
						Symbol:      "XBTUSDTM",
						Granularity: 1000,
						Timestamp:   types.Time(time.Unix(1790812808, 0)),
						IndexPrice:  83620.83,
						DecompositionList: []UTAIndexPriceDecomposition{
							{
								Exchange: "binance",
								Price:    83620.95,
								Weight:   0.4115,
							},
							{
								Exchange: "bybit",
								Price:    83619.6,
								Weight:   0.0864,
							},
						},
					},
					{
						Symbol:      "XBTUSDTM",
						Granularity: 1000,
						Timestamp:   types.Time(time.Unix(1790812807, 0)),
						IndexPrice:  83626.57,
						DecompositionList: []UTAIndexPriceDecomposition{
							{
								Exchange: "binance",
								Price:    83629.06,
								Weight:   0.4115,
							},
							{
								Exchange: "bybit",
								Price:    83619.6,
								Weight:   0.0864,
							},
						},
					},
				},
				LastID: 1790812807000,
			}
			assert.Equal(t, exp, result, "GetUTAIndexPrice should decode every field")
		}
		// KuCoin prunes old index prices, after which the window returns no items, so live only checks the call
	})
}

func TestGetUTATrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTATrades(t.Context(), "", "BTC-USDT")
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTATrades must reject an empty trade type")
	_, err = e.GetUTATrades(t.Context(), "SPOT", "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTATrades must reject an empty symbol")

	for _, tc := range []struct {
		tradeType, symbol string
		exp               *UTATradesResponse
	}{
		{tradeType: "SPOT", symbol: "BTC-USDT", exp: &UTATradesResponse{
			TradeType: "SPOT",
			List: []UTATrade{
				{
					Sequence:  24806572116426759,
					TradeID:   "24806572116426759",
					Price:     83464.4,
					Size:      0.00004335,
					Side:      "SELL",
					Timestamp: types.Time(time.UnixMilli(1791898330115)),
				},
				{
					Sequence:  24806572119310336,
					TradeID:   "24806572119310336",
					Price:     83464.3,
					Size:      0.01437479,
					Side:      "SELL",
					Timestamp: types.Time(time.UnixMilli(1791898330116)),
				},
			},
		}},
		{tradeType: "FUTURES", symbol: "XBTUSDTM", exp: &UTATradesResponse{
			TradeType: "FUTURES",
			List: []UTATrade{
				{
					Sequence:   1947309340686,
					TradeID:    "1947309340686",
					Price:      83454.8,
					Size:       6,
					Side:       "BUY",
					Timestamp:  types.Time(time.UnixMilli(1791898322588)),
					IsRPITrade: true,
				},
				{
					Sequence:  1947309338772,
					TradeID:   "1947309338772",
					Price:     83465.2,
					Size:      8,
					Side:      "SELL",
					Timestamp: types.Time(time.UnixMilli(1791898321039)),
				},
			},
		}},
	} {
		t.Run(tc.tradeType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTATrades(t.Context(), tc.tradeType, tc.symbol)
			require.NoError(t, err, "GetUTATrades must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTATrades should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetUTATrades should return trades")
		})
	}
}

func TestGetUTAPositionTiers(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAPositionTiers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAPositionTiers must reject a nil request")
	for _, tc := range []struct {
		name string
		req  *UTAPositionTiersRequest
		err  error
	}{
		{name: "no trade type", req: &UTAPositionTiersRequest{MarginMode: "CROSS", Data: "BORROW", AccountType: "UNIFIED"}, err: errTradeTypeMissing},
		{name: "no margin mode", req: &UTAPositionTiersRequest{TradeType: "MARGIN", Data: "BORROW", AccountType: "UNIFIED"}, err: margin.ErrInvalidMarginType},
		{name: "no data type", req: &UTAPositionTiersRequest{TradeType: "MARGIN", MarginMode: "CROSS", AccountType: "UNIFIED"}, err: errPositionTierDataRequired},
		{name: "no account type", req: &UTAPositionTiersRequest{TradeType: "MARGIN", MarginMode: "CROSS", Data: "BORROW"}, err: errAccountTypeMissing},
		{name: "borrow without currencies", req: &UTAPositionTiersRequest{TradeType: "MARGIN", MarginMode: "CROSS", Data: "BORROW", AccountType: "UNIFIED"}, err: currency.ErrCurrencyCodesEmpty},
		{name: "risk limit without symbols", req: &UTAPositionTiersRequest{TradeType: "FUTURES", MarginMode: "CROSS", Data: "RISK_LIMIT", AccountType: "UNIFIED"}, err: currency.ErrSymbolStringEmpty},
	} {
		_, err := e.GetUTAPositionTiers(t.Context(), tc.req)
		require.ErrorIsf(t, err, tc.err, "GetUTAPositionTiers must reject a request with %s", tc.name)
	}

	for _, tc := range []struct {
		name string
		req  *UTAPositionTiersRequest
		exp  []UTAPositionTierResponse
	}{
		{
			name: "futures risk limit",
			req:  &UTAPositionTiersRequest{TradeType: "FUTURES", MarginMode: "CROSS", Data: "RISK_LIMIT", AccountType: "UNIFIED", Symbols: []string{"XBTUSDTM"}},
			exp: []UTAPositionTierResponse{
				{
					Symbol:             "XBTUSDTM",
					Tier:               1,
					MaxSize:            250000,
					MaxLeverage:        125,
					InitialMarginRate:  0.008,
					MaintainMarginRate: 0.004,
				},
				{
					Symbol:                 "XBTUSDTM",
					Tier:                   2,
					MinSize:                250000,
					MaxSize:                600000,
					MaxLeverage:            100,
					InitialMarginRate:      0.01,
					MaintainMarginRate:     0.005,
					QuickCalculationOffset: 250,
				},
			},
		},
		{
			name: "margin borrow",
			req:  &UTAPositionTiersRequest{TradeType: "MARGIN", Currencies: []currency.Code{currency.BTC}, MarginMode: "CROSS", Data: "BORROW", AccountType: "UNIFIED"},
			exp: []UTAPositionTierResponse{
				{
					Currency:           currency.BTC,
					Tier:               1,
					MaxSize:            20,
					MaxLeverage:        10,
					MaintainMarginRate: 0.02,
				},
				{
					Currency:           currency.BTC,
					Tier:               2,
					MinSize:            20,
					MaxSize:            22,
					MaxLeverage:        9,
					MaintainMarginRate: 0.022,
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAPositionTiers(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAPositionTiers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAPositionTiers should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetUTAPositionTiers should return tiers")
		})
	}
}

func TestGetUTACollateralRatio(t *testing.T) {
	t.Parallel()
	result, err := e.GetUTACollateralRatio(t.Context())
	require.NoError(t, err, "GetUTACollateralRatio must not error")
	if mockTests {
		exp := []UTACollateralRatioResponse{
			{
				Currency: currency.USDT,
				CDRConfigurations: []UTACollateralRatioTier{
					{
						Tier:      1,
						Max:       40000000,
						CDR:       1,
						CDREquity: 40000000,
					},
					{
						Tier:      2,
						Min:       40000000,
						Max:       50000000,
						CDR:       0.9999,
						CDREquity: 49999000,
					},
				},
			},
			{
				Currency: currency.BTC,
				CDRConfigurations: []UTACollateralRatioTier{
					{
						Tier:      1,
						Max:       5000000,
						CDR:       0.98,
						CDREquity: 4900000,
					},
					{
						Tier:      2,
						Min:       5000000,
						Max:       7500000,
						CDR:       0.975,
						CDREquity: 7337500,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetUTACollateralRatio should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUTACollateralRatio should return currencies")
}

func TestGetUTABorrowableCurrencies(t *testing.T) {
	t.Parallel()
	result, err := e.GetUTABorrowableCurrencies(t.Context())
	require.NoError(t, err, "GetUTABorrowableCurrencies must not error")
	if mockTests {
		exp := []UTABorrowableCurrencyResponse{
			{
				Currency: currency.USDC,
			},
			{
				Currency: currency.USDT,
			},
			{
				Currency: currency.BTC,
			},
		}
		assert.Equal(t, exp, result, "GetUTABorrowableCurrencies should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUTABorrowableCurrencies should return currencies")
}

func TestGetUTACurrentFundingRate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, symbol, productType string
		exp                       []UTACurrentFundingRateResponse
	}{
		{name: "symbol", symbol: "XBTUSDTM", exp: []UTACurrentFundingRateResponse{
			{
				Symbol:                  "XBTUSDTM",
				NextFundingRate:         0.000053,
				FundingTime:             types.Time(time.Unix(1791907200, 0)),
				FundingRateCap:          0.003,
				FundingRateFloor:        -0.003,
				CurrentGranularity:      28800000,
				NewGranularity:          28800000,
				NewGranularityStartTime: types.Time(time.Unix(1750147200, 0)),
			},
		}},
		{name: "product type", productType: "COIN-FUTURES", exp: []UTACurrentFundingRateResponse{
			{
				Symbol:                  "XBTUSDM",
				NextFundingRate:         -0.000624,
				FundingTime:             types.Time(time.Unix(1791907200, 0)),
				FundingRateCap:          0.00525,
				FundingRateFloor:        -0.00525,
				CurrentGranularity:      28800000,
				NewGranularity:          28800000,
				NewGranularityStartTime: types.Time(time.Unix(1750147200, 0)),
			},
			{
				Symbol:                  "ETHUSDM",
				NextFundingRate:         0.0001,
				FundingTime:             types.Time(time.Unix(1791907200, 0)),
				FundingRateCap:          0.0075,
				FundingRateFloor:        -0.0075,
				CurrentGranularity:      28800000,
				NewGranularity:          28800000,
				NewGranularityStartTime: types.Time(time.Unix(1750147200, 0)),
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTACurrentFundingRate(t.Context(), tc.symbol, tc.productType)
			require.NoError(t, err, "GetUTACurrentFundingRate must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTACurrentFundingRate should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetUTACurrentFundingRate should return funding rates")
		})
	}
}

func TestGetUTAFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAFundingRateHistory(t.Context(), "", utaTestStart, utaTestEnd)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTAFundingRateHistory must reject an empty symbol")
	_, err = e.GetUTAFundingRateHistory(t.Context(), "XBTUSDTM", time.Time{}, utaTestEnd)
	require.ErrorIs(t, err, common.ErrDateUnset, "GetUTAFundingRateHistory must reject an unset start")
	_, err = e.GetUTAFundingRateHistory(t.Context(), "XBTUSDTM", utaTestEnd, utaTestStart)
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAFundingRateHistory must reject a start after the end")

	result, err := e.GetUTAFundingRateHistory(t.Context(), "XBTUSDTM", utaTestStart, utaTestEnd)
	require.NoError(t, err, "GetUTAFundingRateHistory must not error")
	if mockTests {
		exp := &UTAFundingRateHistoryResponse{
			Symbol: "XBTUSDTM",
			List: []UTAHistoricFundingRate{
				{
					FundingRate: -0.000034,
					Timestamp:   types.Time(time.Unix(1790899200, 0)),
				},
				{
					FundingRate: 0.000042,
					Timestamp:   types.Time(time.Unix(1790870400, 0)),
				},
				{
					FundingRate: 0.00003,
					Timestamp:   types.Time(time.Unix(1790841600, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetUTAFundingRateHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetUTAFundingRateHistory should return funding rates")
}

func TestGetUTAFuturesOpenInterest(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAFuturesOpenInterest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAFuturesOpenInterest must reject a nil request")
	_, err = e.GetUTAFuturesOpenInterest(t.Context(), &UTAFuturesOpenInterestRequest{Symbols: []string{"XBTUSDTM", "ETHUSDTM"}, Interval: "1day"})
	require.ErrorIs(t, err, errHistoricOpenInterestSymbols, "GetUTAFuturesOpenInterest must reject historic open interest for two symbols")
	_, err = e.GetUTAFuturesOpenInterest(t.Context(), &UTAFuturesOpenInterestRequest{Symbols: []string{"XBTUSDTM"}, Interval: "1day", StartAt: utaTestEnd, EndAt: utaTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAFuturesOpenInterest must reject a start after the end")

	for _, tc := range []struct {
		name string
		req  *UTAFuturesOpenInterestRequest
		exp  []UTAFuturesOpenInterestResponse
	}{
		{name: "real time", req: &UTAFuturesOpenInterestRequest{Symbols: []string{"XBTUSDTM", "ETHUSDTM"}}, exp: []UTAFuturesOpenInterestResponse{
			{
				Symbol:       "ETHUSDTM",
				OpenInterest: 15305501,
				Timestamp:    types.Time(time.UnixMilli(1791898386440)),
			},
			{
				Symbol:       "XBTUSDTM",
				OpenInterest: 9499008,
				Timestamp:    types.Time(time.UnixMilli(1791898386440)),
			},
		}},
		{name: "historic", req: &UTAFuturesOpenInterestRequest{Symbols: []string{"XBTUSDTM"}, Interval: "1day", PageSize: 2}, exp: []UTAFuturesOpenInterestResponse{
			{
				OpenInterest: 10045949,
				Timestamp:    types.Time(time.Unix(1791849600, 0)),
			},
			{
				OpenInterest: 10000455,
				Timestamp:    types.Time(time.Unix(1791763200, 0)),
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAFuturesOpenInterest(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAFuturesOpenInterest must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAFuturesOpenInterest should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetUTAFuturesOpenInterest should return open interest")
		})
	}
}

func TestGetUTAInterestRateIndex(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAInterestRateIndex(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUTAInterestRateIndex must reject a nil request")
	_, err = e.GetUTAInterestRateIndex(t.Context(), &UTAInterestRateIndexRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTAInterestRateIndex must reject an empty symbol")
	_, err = e.GetUTAInterestRateIndex(t.Context(), &UTAInterestRateIndexRequest{Symbol: "XBTUSDTM", StartAt: utaTestEnd, EndAt: utaTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUTAInterestRateIndex must reject a start after the end")

	for _, tc := range []struct {
		name string
		req  *UTAInterestRateIndexRequest
		exp  *UTAInterestRateIndexResponse
	}{
		{name: "latest", req: &UTAInterestRateIndexRequest{Symbol: "XBTUSDTM", PageSize: 2}, exp: &UTAInterestRateIndexResponse{
			Items: []UTAInterestRate{
				{
					Symbol:       "XBTUSDTM",
					InterestRate: 0.0003,
					Timestamp:    types.Time(time.Unix(1791898380, 0)),
				},
				{
					Symbol:       "XBTUSDTM",
					InterestRate: 0.0003,
					Timestamp:    types.Time(time.Unix(1791898320, 0)),
				},
			},
			LastID: 1791898320000,
		}},
		{
			name: "window",
			req:  &UTAInterestRateIndexRequest{Symbol: "XBTUSDTM", StartAt: utaTestStart, EndAt: utaTestEnd, LastID: 1790899200000, PageSize: 2},
			exp: &UTAInterestRateIndexResponse{
				Items: []UTAInterestRate{
					{
						Symbol:       "XBTUSDTM",
						InterestRate: 0.0003,
						Timestamp:    types.Time(time.Unix(1790899140, 0)),
					},
					{
						Symbol:       "XBTUSDTM",
						InterestRate: 0.0003,
						Timestamp:    types.Time(time.Unix(1790899080, 0)),
					},
				},
				LastID: 1790899080000,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAInterestRateIndex(t.Context(), tc.req)
			require.NoError(t, err, "GetUTAInterestRateIndex must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAInterestRateIndex should decode every field")
				return
			}
			assert.NotEmpty(t, result.Items, "GetUTAInterestRateIndex should return interest rates")
		})
	}
}

func TestGetUTAPlatform24HourMarketStatistics(t *testing.T) {
	t.Parallel()
	result, err := e.GetUTAPlatform24HourMarketStatistics(t.Context())
	require.NoError(t, err, "GetUTAPlatform24HourMarketStatistics must not error")
	if mockTests {
		exp := &UTAPlatform24HourMarketStatisticsResponse{
			Spot: UTAMarketTurnover{
				TurnoverOf24Hour: 1443439487.4,
			},
			Futures: UTAMarketTurnover{
				TurnoverOf24Hour: 2534166517.68,
			},
		}
		assert.Equal(t, exp, result, "GetUTAPlatform24HourMarketStatistics should decode every field")
		return
	}
	assert.Positive(t, result.Spot.TurnoverOf24Hour.Float64(), "GetUTAPlatform24HourMarketStatistics should return a spot turnover")
	assert.Positive(t, result.Futures.TurnoverOf24Hour.Float64(), "GetUTAPlatform24HourMarketStatistics should return a futures turnover")
}

func TestGetUTACallAuctionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTACallAuctionInfo(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetUTACallAuctionInfo must reject an empty symbol")

	result, err := e.GetUTACallAuctionInfo(t.Context(), "BTC-USDT")
	require.NoError(t, err, "GetUTACallAuctionInfo must not error")
	if mockTests {
		exp := &UTACallAuctionInfoResponse{
			Symbol:                  "BTC-USDT",
			EstimatedPrice:          83500.5,
			EstimatedSize:           12.5,
			SellOrderRangeLowPrice:  83500.5,
			SellOrderRangeHighPrice: 91850,
			BuyOrderRangeLowPrice:   75150,
			BuyOrderRangeHighPrice:  83500.4,
			Time:                    types.Time(time.Unix(1791898400, 0)),
		}
		assert.Equal(t, exp, result, "GetUTACallAuctionInfo should decode every field")
		return
	}
	// BTC-USDT is in no call auction, so KuCoin answers with every field null
	assert.NotNil(t, result, "GetUTACallAuctionInfo should return a response")
}

func TestGetUTAFiatPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAFiatPrice(t.Context(), currency.EMPTYCODE, nil)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetUTAFiatPrice must reject an empty base")

	result, err := e.GetUTAFiatPrice(t.Context(), currency.USD, []currency.Code{currency.BTC, currency.ETH})
	require.NoError(t, err, "GetUTAFiatPrice must not error")
	if mockTests {
		exp := map[currency.Code]types.Number{
			currency.BTC: 83229.7604697214,
			currency.ETH: 2503.9606140003098,
		}
		assert.Equal(t, exp, result, "GetUTAFiatPrice should decode every price")
		return
	}
	assert.Positive(t, result[currency.BTC].Float64(), "GetUTAFiatPrice should return a BTC price")
	assert.Positive(t, result[currency.ETH].Float64(), "GetUTAFiatPrice should return an ETH price")
}

func TestGetUTAThirdPartyCustodyCurrencies(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name      string
		custodian string
		ccy       currency.Code
		exp       []UTAThirdPartyCustodyCurrencyResponse
	}{
		{name: "one currency", custodian: "BITGO_SG", ccy: currency.BTC, exp: []UTAThirdPartyCustodyCurrencyResponse{
			{
				Custodian: "BITGO_SG",
				Currency:  currency.BTC,
				Precision: 8,
			},
		}},
		{name: "every custodian", exp: []UTAThirdPartyCustodyCurrencyResponse{
			{
				Custodian: "BITGO_SG",
				Currency:  currency.NewCode("XAUT"),
				Precision: 6,
			},
			{
				Custodian: "BITGO_SG",
				Currency:  currency.NewCode("SUI"),
				Precision: 8,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAThirdPartyCustodyCurrencies(t.Context(), tc.custodian, tc.ccy)
			require.NoError(t, err, "GetUTAThirdPartyCustodyCurrencies must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAThirdPartyCustodyCurrencies should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetUTAThirdPartyCustodyCurrencies should return currencies")
		})
	}
}

func TestGetUTAServiceStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetUTAServiceStatus(t.Context(), "")
	require.ErrorIs(t, err, errTradeTypeMissing, "GetUTAServiceStatus must reject an empty trade type")

	for _, tc := range []struct {
		tradeType string
		exp       *UTAServiceStatusResponse
	}{
		{tradeType: "SPOT", exp: &UTAServiceStatusResponse{
			TradeType:    "SPOT",
			ServerStatus: "open",
			Message:      "Scheduled maintenance at 2026-10-20 02:00 UTC",
		}},
		{tradeType: "FUTURES", exp: &UTAServiceStatusResponse{
			TradeType:    "FUTURES",
			ServerStatus: "open",
		}},
	} {
		t.Run(tc.tradeType, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetUTAServiceStatus(t.Context(), tc.tradeType)
			require.NoError(t, err, "GetUTAServiceStatus must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetUTAServiceStatus should decode every field")
				return
			}
			assert.Equal(t, tc.tradeType, result.TradeType, "GetUTAServiceStatus should return the trade type asked for")
			assert.NotEmpty(t, result.ServerStatus, "GetUTAServiceStatus should return a status")
		})
	}
}

func TestGetUTAKYCRegions(t *testing.T) {
	t.Parallel()
	result, err := e.GetUTAKYCRegions(t.Context())
	require.NoError(t, err, "GetUTAKYCRegions must not error")
	if mockTests {
		exp := []UTAKYCRegionResponse{
			{
				Code:        "AD",
				EnglishName: "Andorra",
			},
			{
				Code:        "AE",
				EnglishName: "United Arab Emirates",
			},
			{
				Code:        "AF",
				EnglishName: "Afghanistan",
			},
		}
		assert.Equal(t, exp, result, "GetUTAKYCRegions should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetUTAKYCRegions should return regions")
}

func TestGetUTAClientIPAddress(t *testing.T) {
	t.Parallel()
	result, err := e.GetUTAClientIPAddress(t.Context())
	require.NoError(t, err, "GetUTAClientIPAddress must not error")
	if mockTests {
		assert.Equal(t, "192.0.2.1", result, "GetUTAClientIPAddress should decode the address")
		return
	}
	// The address is not logged, so the test never records the machine's IP
	assert.NotNil(t, net.ParseIP(result), "GetUTAClientIPAddress should return an IP address")
}
