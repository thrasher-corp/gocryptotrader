package kucoin

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAnnouncements(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 9, 8, 0, 0, 0, 0, time.UTC)
	_, err := e.GetAnnouncements(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAnnouncements must reject a nil request")
	_, err = e.GetAnnouncements(t.Context(), &AnnouncementsRequest{StartTime: end, EndTime: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAnnouncements must reject a start after the end")

	result, err := e.GetAnnouncements(t.Context(), &AnnouncementsRequest{CurrentPage: 1, PageSize: 2, AnnouncementType: "delistings", Language: "en_US", StartTime: start, EndTime: end})
	require.NoError(t, err, "GetAnnouncements must not error")
	if mockTests {
		exp := &AnnouncementsResponse{
			TotalNumber: 2,
			Items: []Announcement{
				{
					AnnouncementID:          329319,
					AnnouncementTitle:       "KuCoin Earn Will Delist the WAXP Products",
					AnnouncementType:        []string{"latest-announcements", "delistings"},
					AnnouncementDescription: "KuCoin Earn Will Delist the WAXP Products",
					CreateTime:              types.Time(time.UnixMilli(1788438300000)),
					Language:                "en_US",
					AnnouncementURL:         "https://www.kucoin.com/announcement/en-kucoin-earn-will-delist-the-waxp-products?lang=en_US",
				},
				{
					AnnouncementID:          328939,
					AnnouncementTitle:       "ST: KuCoin Will Delist Multiple Tokens on 2026-09-07",
					AnnouncementType:        []string{"latest-announcements", "delistings"},
					AnnouncementDescription: "In accordance with the Special Treatment Rules of KuCoin, the following projects will be delisted, and its associated token will be removed from the platform at 8:00 on September 7, 2026 (UTC). ",
					CreateTime:              types.Time(time.UnixMilli(1788422280000)),
					Language:                "en_US",
					AnnouncementURL:         "https://www.kucoin.com/announcement/en-st-kucoin-will-delist-multiple-tokens-on-2026-09-07-1?lang=en_US",
				},
			},
			CurrentPage: 1,
			PageSize:    2,
			TotalPage:   1,
		}
		assert.Equal(t, exp, result, "GetAnnouncements should decode every field")
		return
	}
	assert.NotEmpty(t, result.Items, "GetAnnouncements should return announcements")
}

func TestGetCurrencyDetailV3(t *testing.T) {
	t.Parallel()
	_, err := e.GetCurrencyDetailV3(t.Context(), currency.EMPTYCODE, "")
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetCurrencyDetailV3 must reject an empty currency")

	btcChain := CurrencyChain{
		ChainName:         "BTC",
		WithdrawalMinSize: 0.00018,
		WithdrawMinSize:   0.00018,
		DepositMinSize:    0.00003,
		WithdrawalMinFee:  0.00009,
		WithdrawMinFee:    0.00009,
		IsWithdrawEnabled: true,
		IsDepositEnabled:  true,
		Confirms:          3,
		PreConfirms:       1,
		WithdrawPrecision: 8,
		MaxWithdraw:       20,
		ChainID:           "btc",
		AddressRegex:      "^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$|^(bc1)[0-9A-Za-z]{39,59}$",
	}
	bep20Chain := CurrencyChain{
		ChainName:         "BEP20",
		WithdrawalMinSize: 0.000008,
		WithdrawMinSize:   0.000008,
		DepositMinSize:    0.0000001,
		WithdrawalMinFee:  0.000004,
		WithdrawMinFee:    0.000004,
		IsWithdrawEnabled: true,
		IsDepositEnabled:  true,
		Confirms:          60,
		PreConfirms:       15,
		ContractAddress:   "0x7130d2a12b9bcbfae4f2634d864a1ee1ce3ead9c",
		WithdrawPrecision: 8,
		ChainID:           "bsc",
		AddressRegex:      "^(0x)[0-9A-Fa-f]{40}$",
	}
	lightningChain := CurrencyChain{
		ChainName:         "Lightning Network",
		WithdrawalMinSize: 0.00004,
		WithdrawMinSize:   0.00004,
		DepositMinSize:    0.00001,
		WithdrawalMinFee:  0.00002,
		WithdrawMinFee:    0.00002,
		Confirms:          1,
		PreConfirms:       1,
		WithdrawPrecision: 8,
		MaxDeposit:        0.03,
		ChainID:           "btcln",
		AddressRegex:      "^(lnbc)[0-9a-hj-np-z]{4,639}$",
	}
	for _, tc := range []struct {
		name  string
		ccy   currency.Code
		chain string
		exp   *CurrencyResponse
	}{
		{name: "BTC", ccy: currency.BTC, exp: &CurrencyResponse{
			Currency:        currency.BTC,
			Name:            "BTC",
			FullName:        "Bitcoin",
			Precision:       8,
			IsMarginEnabled: true,
			IsDebitEnabled:  true,
			Chains:          []CurrencyChain{btcChain, bep20Chain, lightningChain},
		}},
		{name: "BTC on bsc", ccy: currency.BTC, chain: "bsc", exp: &CurrencyResponse{
			Currency:        currency.BTC,
			Name:            "BTC",
			FullName:        "Bitcoin",
			Precision:       8,
			IsMarginEnabled: true,
			IsDebitEnabled:  true,
			Chains:          []CurrencyChain{bep20Chain},
		}},
		{name: "USDT", ccy: currency.USDT, exp: &CurrencyResponse{
			Currency:        currency.USDT,
			Name:            "USDT",
			FullName:        "Tether",
			Precision:       8,
			Confirms:        64,
			ContractAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			IsMarginEnabled: true,
			IsDebitEnabled:  true,
			Chains: []CurrencyChain{
				{
					ChainName:         "TON",
					WithdrawalMinSize: 1,
					WithdrawMinSize:   1,
					DepositMinSize:    0.1,
					WithdrawalMinFee:  0.5,
					WithdrawMinFee:    0.5,
					IsWithdrawEnabled: true,
					IsDepositEnabled:  true,
					Confirms:          1,
					PreConfirms:       1,
					ContractAddress:   "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs",
					WithdrawPrecision: 6,
					NeedTag:           true,
					ChainID:           "ton",
					AddressRegex:      `^[UE]Q[0-9a-zA-Z\_\-\+\/]{46,50}$`, // codespell:ignore ue
					MemoRegex:         "^[a-zA-Z0-9]{0,32}$",
				},
				{
					ChainName:         "ERC20",
					WithdrawalMinSize: 30,
					WithdrawMinSize:   30,
					DepositMinSize:    1,
					WithdrawFeeRate:   0.002,
					WithdrawalMinFee:  5.5,
					WithdrawMinFee:    5.5,
					IsWithdrawEnabled: true,
					IsDepositEnabled:  true,
					Confirms:          64,
					PreConfirms:       12,
					ContractAddress:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
					WithdrawPrecision: 6,
					ChainID:           "eth",
					DepositFeeRate:    0.001,
					WithdrawMaxFee:    36,
					DepositTierFee:    "0.0001",
					AddressRegex:      "^(0x)[0-9A-Fa-f]{40}$",
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCurrencyDetailV3(t.Context(), tc.ccy, tc.chain)
			require.NoError(t, err, "GetCurrencyDetailV3 must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCurrencyDetailV3 should decode every field")
				return
			}
			assert.True(t, tc.ccy.Equal(result.Currency), "GetCurrencyDetailV3 should return the requested currency")
			assert.NotEmpty(t, result.Chains, "GetCurrencyDetailV3 should return chains")
			if tc.chain != "" {
				for i := range result.Chains {
					assert.Equal(t, tc.chain, result.Chains[i].ChainID, "GetCurrencyDetailV3 should return only the requested chain")
				}
			}
		})
	}
}

func TestGetCurrenciesV3(t *testing.T) {
	t.Parallel()
	result, err := e.GetCurrenciesV3(t.Context())
	require.NoError(t, err, "GetCurrenciesV3 must not error")
	if mockTests {
		exp := []CurrencyResponse{
			{
				Currency:        currency.BTC,
				Name:            "BTC",
				FullName:        "Bitcoin",
				Precision:       8,
				IsMarginEnabled: true,
				IsDebitEnabled:  true,
				Chains: []CurrencyChain{
					{
						ChainName:         "BTC",
						WithdrawalMinSize: 0.00018,
						WithdrawMinSize:   0.00018,
						DepositMinSize:    0.00003,
						WithdrawalMinFee:  0.00009,
						WithdrawMinFee:    0.00009,
						IsWithdrawEnabled: true,
						IsDepositEnabled:  true,
						Confirms:          3,
						PreConfirms:       1,
						WithdrawPrecision: 8,
						MaxWithdraw:       20,
						ChainID:           "btc",
						AddressRegex:      "^[13][a-km-zA-HJ-NP-Z1-9]{25,34}$|^(bc1)[0-9A-Za-z]{39,59}$",
					},
					{
						ChainName:         "Lightning Network",
						WithdrawalMinSize: 0.00004,
						WithdrawMinSize:   0.00004,
						DepositMinSize:    0.00001,
						WithdrawalMinFee:  0.00002,
						WithdrawMinFee:    0.00002,
						Confirms:          1,
						PreConfirms:       1,
						WithdrawPrecision: 8,
						MaxDeposit:        0.03,
						ChainID:           "btcln",
						AddressRegex:      "^(lnbc)[0-9a-hj-np-z]{4,639}$",
					},
				},
			},
			{
				Currency:        currency.USDT,
				Name:            "USDT",
				FullName:        "Tether",
				Precision:       8,
				Confirms:        64,
				ContractAddress: "0xdac17f958d2ee523a2206206994597c13d831ec7",
				IsMarginEnabled: true,
				IsDebitEnabled:  true,
				Chains: []CurrencyChain{
					{
						ChainName:         "TON",
						WithdrawalMinSize: 1,
						WithdrawMinSize:   1,
						DepositMinSize:    0.1,
						WithdrawalMinFee:  0.5,
						WithdrawMinFee:    0.5,
						IsWithdrawEnabled: true,
						IsDepositEnabled:  true,
						Confirms:          1,
						PreConfirms:       1,
						ContractAddress:   "EQCxE6mUtQJKFnGfaROTKOt1lZbDiiX1kCixRv7Nw2Id_sDs",
						WithdrawPrecision: 6,
						NeedTag:           true,
						ChainID:           "ton",
						AddressRegex:      `^[UE]Q[0-9a-zA-Z\_\-\+\/]{46,50}$`, // codespell:ignore ue
						MemoRegex:         "^[a-zA-Z0-9]{0,32}$",
					},
					{
						ChainName:         "ERC20",
						WithdrawalMinSize: 30,
						WithdrawMinSize:   30,
						DepositMinSize:    1,
						WithdrawFeeRate:   0.002,
						WithdrawalMinFee:  5.5,
						WithdrawMinFee:    5.5,
						IsWithdrawEnabled: true,
						IsDepositEnabled:  true,
						Confirms:          64,
						PreConfirms:       12,
						ContractAddress:   "0xdac17f958d2ee523a2206206994597c13d831ec7",
						WithdrawPrecision: 6,
						ChainID:           "eth",
						DepositFeeRate:    0.001,
						WithdrawMaxFee:    36,
						DepositTierFee:    "0.0001",
						AddressRegex:      "^(0x)[0-9A-Fa-f]{40}$",
					},
				},
			},
			{
				Currency:  currency.USD,
				Name:      "USD",
				FullName:  "US Dollar",
				Precision: 2,
			},
		}
		assert.Equal(t, exp, result, "GetCurrenciesV3 should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCurrenciesV3 should return currencies")
}

func TestGetSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.GetSymbol(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetSymbol must reject an empty symbol")

	result, err := e.GetSymbol(t.Context(), "BTC-USDT")
	require.NoError(t, err, "GetSymbol must not error")
	if mockTests {
		exp := &SymbolResponse{
			Symbol:                          "BTC-USDT",
			Name:                            "BTC-USDT",
			BaseCurrency:                    currency.BTC,
			QuoteCurrency:                   currency.USDT,
			FeeCurrency:                     currency.USDT,
			Market:                          "USDS",
			BaseMinSize:                     0.00001,
			QuoteMinSize:                    0.1,
			BaseMaxSize:                     10000000000,
			QuoteMaxSize:                    99999999,
			BaseIncrement:                   0.00000001,
			QuoteIncrement:                  0.000001,
			PriceIncrement:                  0.1,
			PriceLimitRate:                  0.01,
			MinFunds:                        0.1,
			IsMarginEnabled:                 true,
			EnableTrading:                   true,
			FeeCategory:                     1,
			MakerFeeCoefficient:             1,
			TakerFeeCoefficient:             1,
			SpecialTreatment:                true,
			CallAuctionIsEnabled:            true,
			CallAuctionPriceFloor:           41000,
			CallAuctionPriceCeiling:         125000,
			CallAuctionFirstStageStartTime:  types.Time(time.UnixMilli(1760000400000)),
			CallAuctionSecondStageStartTime: types.Time(time.UnixMilli(1760003400000)),
			CallAuctionThirdStageStartTime:  types.Time(time.UnixMilli(1760003700000)),
			TradingStartTime:                types.Time(time.UnixMilli(1760004000000)),
		}
		assert.Equal(t, exp, result, "GetSymbol should decode every field")
		return
	}
	assert.Equal(t, "BTC-USDT", result.Symbol, "GetSymbol should return the requested symbol")
}

func TestGetSymbols(t *testing.T) {
	t.Parallel()
	btcUSDT := SymbolResponse{
		Symbol:              "BTC-USDT",
		Name:                "BTC-USDT",
		BaseCurrency:        currency.BTC,
		QuoteCurrency:       currency.USDT,
		FeeCurrency:         currency.USDT,
		Market:              "USDS",
		BaseMinSize:         0.00001,
		QuoteMinSize:        0.1,
		BaseMaxSize:         10000000000,
		QuoteMaxSize:        99999999,
		BaseIncrement:       0.00000001,
		QuoteIncrement:      0.000001,
		PriceIncrement:      0.1,
		PriceLimitRate:      0.01,
		MinFunds:            0.1,
		IsMarginEnabled:     true,
		EnableTrading:       true,
		FeeCategory:         1,
		MakerFeeCoefficient: 1,
		TakerFeeCoefficient: 1,
	}
	doloUSDT := SymbolResponse{
		Symbol:                          "DOLO-USDT",
		Name:                            "DOLO-USDT",
		BaseCurrency:                    currency.NewCode("DOLO"),
		QuoteCurrency:                   currency.USDT,
		FeeCurrency:                     currency.USDT,
		Market:                          "USDS",
		BaseMinSize:                     10,
		QuoteMinSize:                    0.1,
		BaseMaxSize:                     10000000000,
		QuoteMaxSize:                    99999999,
		BaseIncrement:                   0.1,
		QuoteIncrement:                  0.00001,
		PriceIncrement:                  0.00001,
		PriceLimitRate:                  0.08,
		MinFunds:                        0.1,
		EnableTrading:                   true,
		FeeCategory:                     1,
		MakerFeeCoefficient:             1,
		TakerFeeCoefficient:             1,
		SpecialTreatment:                true,
		CallAuctionIsEnabled:            true,
		CallAuctionPriceFloor:           0.00001,
		CallAuctionPriceCeiling:         3,
		CallAuctionFirstStageStartTime:  types.Time(time.UnixMilli(1745496000000)),
		CallAuctionSecondStageStartTime: types.Time(time.UnixMilli(1745499000000)),
		CallAuctionThirdStageStartTime:  types.Time(time.UnixMilli(1745499300000)),
		TradingStartTime:                types.Time(time.UnixMilli(1745499600000)),
	}
	for _, tc := range []struct {
		name   string
		market string
		exp    []SymbolResponse
	}{
		{name: "all markets", exp: []SymbolResponse{btcUSDT, doloUSDT, {
			Symbol:              "BTC-EUR",
			Name:                "BTC-EUR",
			BaseCurrency:        currency.BTC,
			QuoteCurrency:       currency.EUR,
			FeeCurrency:         currency.EUR,
			Market:              "FIAT",
			BaseMinSize:         0.00001,
			QuoteMinSize:        0.1,
			BaseMaxSize:         10000000000,
			QuoteMaxSize:        99999999,
			BaseIncrement:       0.00001,
			QuoteIncrement:      0.01,
			PriceIncrement:      0.01,
			PriceLimitRate:      0.05,
			EnableTrading:       true,
			FeeCategory:         1,
			MakerFeeCoefficient: 1,
			TakerFeeCoefficient: 1,
			TradingStartTime:    types.Time(time.UnixMilli(1677826800000)),
		}}},
		{name: "USDS market", market: "USDS", exp: []SymbolResponse{btcUSDT, doloUSDT}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetSymbols(t.Context(), tc.market)
			require.NoError(t, err, "GetSymbols must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetSymbols should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetSymbols should return symbols")
		})
	}
}

func TestGetTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTicker(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetTicker must reject an empty symbol")
	_, err = e.GetTicker(t.Context(), "NOTLISTED-USDT")
	require.ErrorIs(t, err, common.ErrNoResponse, "GetTicker must reject the null data KuCoin sends for an unknown symbol")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = e.GetTicker(ctx, "BTC-USDT")
	require.ErrorIs(t, err, context.Canceled, "GetTicker must return the request error")

	result, err := e.GetTicker(t.Context(), "BTC-USDT")
	require.NoError(t, err, "GetTicker must not error")
	if mockTests {
		exp := &TickerResponse{
			Time:        types.Time(time.UnixMilli(1791898219339)),
			Sequence:    38624856889,
			Price:       83481.4,
			Size:        0.00324118,
			BestBid:     83465.2,
			BestBidSize: 0.51594271,
			BestAsk:     83465.3,
			BestAskSize: 0.19624045,
		}
		assert.Equal(t, exp, result, "GetTicker should decode every field")
		return
	}
	assert.Positive(t, result.Price.Float64(), "GetTicker should return a price")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	result, err := e.GetTickers(t.Context())
	require.NoError(t, err, "GetTickers must not error")
	if mockTests {
		exp := &TickersResponse{
			Time: types.Time(time.UnixMilli(1791898249918)),
			Ticker: []TickerInfo{
				{
					Symbol:             "BTC-USDT",
					SymbolName:         "BTC-USDT",
					Buy:                83493.8,
					BestBidSize:        0.33258019,
					Sell:               83493.9,
					BestAskSize:        0.00549083,
					ChangeRate:         0.0113,
					ChangePrice:        935.8,
					Open:               82558.1,
					High:               84149,
					Low:                81035.5,
					Volume:             6187.000634905228,
					VolumeValue:        512747165.9412732,
					Last:               83493.9,
					LastSize:           0.00001941,
					AveragePrice:       83541.67284915,
					TakerFeeRate:       0.001,
					MakerFeeRate:       0.001,
					TakerCoefficient:   1,
					MakerCoefficient:   1,
					PriceChange:        935.8,
					PriceChangePercent: 0.0113,
				},
				{
					Symbol:             "ETH-USDT",
					SymbolName:         "ETH-USDT",
					Buy:                2512.16,
					BestBidSize:        1.3236729,
					Sell:               2512.17,
					BestAskSize:        0.0819805,
					ChangeRate:         -0.0059,
					ChangePrice:        -15.16,
					Open:               2526.76,
					High:               2544.63,
					Low:                2409.42,
					Volume:             37647.42377968,
					VolumeValue:        93872638.19838758,
					Last:               2511.6,
					LastSize:           0.014,
					AveragePrice:       2544.48174935,
					TakerFeeRate:       0.001,
					MakerFeeRate:       0.001,
					TakerCoefficient:   1,
					MakerCoefficient:   1,
					PriceChange:        -15.16,
					PriceChangePercent: -0.0059,
				},
			},
		}
		assert.Equal(t, exp, result, "GetTickers should decode every field")
		return
	}
	assert.NotEmpty(t, result.Ticker, "GetTickers should return tickers")
}

func TestGetTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradeHistory(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetTradeHistory must reject an empty symbol")

	result, err := e.GetTradeHistory(t.Context(), "BTC-USDT")
	require.NoError(t, err, "GetTradeHistory must not error")
	if mockTests {
		exp := []TradeResponse{
			{
				Sequence: "24806547213402117",
				TradeID:  "24806547213402117",
				Price:    83490.5,
				Size:     0.0015,
				Side:     "buy",
				Time:     types.Time(time.UnixMilli(1791898248229)),
			},
			{
				Sequence: "24806548332756992",
				TradeID:  "24806548332756992",
				Price:    83494.3,
				Size:     0.00024787,
				Side:     "sell",
				Time:     types.Time(time.UnixMilli(1791898253238)),
			},
		}
		assert.Equal(t, exp, result, "GetTradeHistory should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetTradeHistory should return trades")
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 1, 0, 15, 0, 0, time.UTC)
	_, err := e.GetKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetKlines must reject a nil request")
	_, err = e.GetKlines(t.Context(), &KlinesRequest{Type: "1week"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetKlines must reject an empty symbol")
	_, err = e.GetKlines(t.Context(), &KlinesRequest{Symbol: "BTC-USDT"})
	require.ErrorIs(t, err, errInvalidPeriod, "GetKlines must reject an empty type")
	_, err = e.GetKlines(t.Context(), &KlinesRequest{Symbol: "BTC-USDT", Type: "5min", StartAt: end, EndAt: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetKlines must reject a start after the end")

	for _, tc := range []struct {
		req *KlinesRequest
		exp []KlineResponse
	}{
		{req: &KlinesRequest{Symbol: "BTC-USDT", Type: "1week"}, exp: []KlineResponse{
			{
				StartTime: types.Time(time.Unix(1791417600, 0)),
				Open:      83321.7,
				Close:     83530.5,
				High:      84334.1,
				Low:       80401,
				Volume:    14499.096956535048,
				Amount:    1198518816.9491231,
			},
			{
				StartTime: types.Time(time.Unix(1790812800, 0)),
				Open:      83609.9,
				Close:     83321.6,
				High:      87220,
				Low:       82784.6,
				Volume:    11918.082024522886,
				Amount:    1011375055.9435077,
			},
		}},
		{req: &KlinesRequest{Symbol: "BTC-USDT", Type: "5min", StartAt: start, EndAt: end}, exp: []KlineResponse{
			{
				StartTime: types.Time(time.Unix(1790813400, 0)),
				Open:      83497.4,
				Close:     83515.6,
				High:      83541.2,
				Low:       83468.7,
				Volume:    5.44817871,
				Amount:    454968.496499014,
			},
			{
				StartTime: types.Time(time.Unix(1790813100, 0)),
				Open:      83535.5,
				Close:     83497.5,
				High:      83545.5,
				Low:       83497.4,
				Volume:    3.98071408,
				Amount:    332474.234429911,
			},
			{
				StartTime: types.Time(time.Unix(1790812800, 0)),
				Open:      83609.9,
				Close:     83535.5,
				High:      83639.1,
				Low:       83535.5,
				Volume:    14.62208182,
				Amount:    1222334.11095265,
			},
		}},
	} {
		t.Run(tc.req.Type, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetKlines(t.Context(), tc.req)
			require.NoError(t, err, "GetKlines must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetKlines should decode every field")
				return
			}
			assert.NotEmpty(t, result, "GetKlines should return candles")
		})
	}
}

func TestKlineUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var k KlineResponse
	require.NoError(t, k.UnmarshalJSON([]byte(`["1746645900","96248.3","96060.4","96248.3","95991.1","7.30387554","701787.956631596"]`)), "UnmarshalJSON must not error")
	exp := KlineResponse{
		StartTime: types.Time(time.Unix(1746645900, 0)),
		Open:      96248.3,
		Close:     96060.4,
		High:      96248.3,
		Low:       95991.1,
		Volume:    7.30387554,
		Amount:    701787.956631596,
	}
	assert.Equal(t, exp, k, "UnmarshalJSON should decode every element")
	assert.Error(t, k.UnmarshalJSON([]byte(`{"time":"1746645900"}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, k.UnmarshalJSON([]byte(`["1746645900","bad"]`)), "UnmarshalJSON should reject an invalid number")
}

func TestGetPartOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetPartOrderbook(t.Context(), "", 20)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetPartOrderbook must reject an empty symbol")
	_, err = e.GetPartOrderbook(t.Context(), "BTC-USDT", 0)
	require.ErrorIs(t, err, errOrderbookSizeRequired, "GetPartOrderbook must reject a zero size")
	ex, sent := newSigningTestInstance(t, `{"code":"200000","data":null}`)
	for _, size := range []uint64{1, 21, 50} {
		_, err = ex.GetPartOrderbook(t.Context(), "BTC-USDT", size)
		require.ErrorIsf(t, err, errInvalidOrderbookSize, "GetPartOrderbook must reject size %d, which KuCoin does not serve", size)
	}
	assert.Empty(t, *sent, "GetPartOrderbook should not send a request for a size KuCoin does not serve")
	_, err = e.GetPartOrderbook(t.Context(), "NOTLISTED-USDT", 20)
	require.ErrorIs(t, err, common.ErrNoResponse, "GetPartOrderbook must reject the null data KuCoin sends for an unknown symbol")
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = e.GetPartOrderbook(ctx, "BTC-USDT", 20)
	require.ErrorIs(t, err, context.Canceled, "GetPartOrderbook must return the request error")

	for _, tc := range []struct {
		size uint64
		exp  *OrderbookResponse
	}{
		{size: 20, exp: &OrderbookResponse{
			Time:     types.Time(time.UnixMilli(1791898290081)),
			Sequence: 38624942045,
			Bids:     []orderbook.Level{{Price: 83489.1, Amount: 0.005}, {Price: 83489, Amount: 0.005}},
			Asks:     []orderbook.Level{{Price: 83489.2, Amount: 1.04605643}, {Price: 83489.4, Amount: 0.0332425}},
		}},
		{size: 100, exp: &OrderbookResponse{
			Time:     types.Time(time.UnixMilli(1791898291420)),
			Sequence: 38624943962,
			Bids:     []orderbook.Level{{Price: 83475.5, Amount: 0.01}, {Price: 83475.4, Amount: 0.01}},
			Asks:     []orderbook.Level{{Price: 83475.6, Amount: 2.03945572}, {Price: 83476, Amount: 0.03136865}},
		}},
	} {
		t.Run(strconv.FormatUint(tc.size, 10), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPartOrderbook(t.Context(), "BTC-USDT", tc.size)
			require.NoError(t, err, "GetPartOrderbook must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPartOrderbook should decode every field")
				return
			}
			assert.NotEmpty(t, result.Bids, "GetPartOrderbook should return bids")
			assert.NotEmpty(t, result.Asks, "GetPartOrderbook should return asks")
		})
	}
}

func TestGetOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderbook(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetOrderbook must reject an empty symbol")

	if mockTests {
		_, err = e.GetOrderbook(t.Context(), "NOTLISTED-USDT")
		require.ErrorIs(t, err, common.ErrNoResponse, "GetOrderbook must reject null data")
	} else {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = e.GetOrderbook(ctx, "BTC-USDT")
	require.ErrorIs(t, err, context.Canceled, "GetOrderbook must return the request error")

	result, err := e.GetOrderbook(t.Context(), "BTC-USDT")
	require.NoError(t, err, "GetOrderbook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			Time:     types.Time(time.UnixMilli(1729176273859)),
			Sequence: 14610502970,
			Bids:     []orderbook.Level{{Price: 66976.4, Amount: 0.69109872}, {Price: 66976.3, Amount: 0.14377}},
			Asks:     []orderbook.Level{{Price: 66976.5, Amount: 0.05408199}, {Price: 66976.8, Amount: 0.0005}},
		}
		assert.Equal(t, exp, result, "GetOrderbook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetOrderbook should return bids")
	assert.NotEmpty(t, result.Asks, "GetOrderbook should return asks")
}

func TestGetCallAuctionPartOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetCallAuctionPartOrderbook(t.Context(), "", 20)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCallAuctionPartOrderbook must reject an empty symbol")
	_, err = e.GetCallAuctionPartOrderbook(t.Context(), "BTC-USDT", 0)
	require.ErrorIs(t, err, errOrderbookSizeRequired, "GetCallAuctionPartOrderbook must reject a zero size")
	ex, sent := newSigningTestInstance(t, `{"code":"200000","data":null}`)
	for _, size := range []uint64{1, 21, 50} {
		_, err = ex.GetCallAuctionPartOrderbook(t.Context(), "BTC-USDT", size)
		require.ErrorIsf(t, err, errInvalidOrderbookSize, "GetCallAuctionPartOrderbook must reject size %d, which KuCoin does not serve", size)
	}
	assert.Empty(t, *sent, "GetCallAuctionPartOrderbook should not send a request for a size KuCoin does not serve")

	for _, tc := range []struct {
		size uint64
		exp  *OrderbookResponse
	}{
		{size: 20, exp: &OrderbookResponse{
			Time:     types.Time(time.UnixMilli(1791898316476)),
			Sequence: 38624942051,
			Bids:     []orderbook.Level{{Price: 83480.1, Amount: 0.25}, {Price: 83480, Amount: 0.5}},
			Asks:     []orderbook.Level{{Price: 83480.1, Amount: 0.10234}, {Price: 83481.2, Amount: 0.0021}},
		}},
		{size: 100, exp: &OrderbookResponse{
			Time:     types.Time(time.UnixMilli(1791899187834)),
			Sequence: -1,
			Bids:     []orderbook.Level{},
			Asks:     []orderbook.Level{},
		}},
	} {
		t.Run(strconv.FormatUint(tc.size, 10), func(t *testing.T) {
			t.Parallel()
			result, err := e.GetCallAuctionPartOrderbook(t.Context(), "BTC-USDT", tc.size)
			require.NoError(t, err, "GetCallAuctionPartOrderbook must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetCallAuctionPartOrderbook should decode every field")
				return
			}
			assert.NotZero(t, result.Time.Time(), "GetCallAuctionPartOrderbook should return a time")
		})
	}
}

func TestGetCallAuctionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetCallAuctionInfo(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCallAuctionInfo must reject an empty symbol")

	result, err := e.GetCallAuctionInfo(t.Context(), "BTC-USDT")
	require.NoError(t, err, "GetCallAuctionInfo must not error")
	if mockTests {
		exp := &CallAuctionInfoResponse{
			Symbol:                  "BTC-USDT",
			EstimatedPrice:          83480.1,
			EstimatedSize:           0.35234,
			SellOrderRangeLowPrice:  75132.1,
			SellOrderRangeHighPrice: 91828.1,
			BuyOrderRangeLowPrice:   75132.1,
			BuyOrderRangeHighPrice:  91828.1,
			Time:                    types.Time(time.UnixMilli(1791898316476)),
		}
		assert.Equal(t, exp, result, "GetCallAuctionInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCallAuctionInfo should return call auction data")
}

func TestGetFiatPrice(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		base       currency.Code
		currencies currency.Currencies
		exp        map[currency.Code]types.Number
	}{
		{name: "every currency in USD", exp: map[currency.Code]types.Number{
			currency.BTC:  83372.4746997041,
			currency.ETH:  2509.683164999837,
			currency.USDT: 0.9987,
		}},
		{name: "ETH and BTC in EUR", base: currency.EUR, currencies: currency.Currencies{currency.ETH, currency.BTC}, exp: map[currency.Code]types.Number{
			currency.BTC: 74440.99004315505,
			currency.ETH: 2240.27221557448,
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFiatPrice(t.Context(), tc.base, tc.currencies)
			require.NoError(t, err, "GetFiatPrice must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFiatPrice should decode every price")
				return
			}
			assert.Positive(t, result[currency.BTC].Float64(), "GetFiatPrice should return a BTC price")
		})
	}
}

func TestGet24HourStats(t *testing.T) {
	t.Parallel()
	_, err := e.Get24HourStats(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "Get24HourStats must reject an empty symbol")

	result, err := e.Get24HourStats(t.Context(), "BTC-USDT")
	require.NoError(t, err, "Get24HourStats must not error")
	if mockTests {
		exp := &Stats24HourResponse{
			Time:             types.Time(time.UnixMilli(1791898338200)),
			Symbol:           "BTC-USDT",
			Buy:              83425.2,
			Sell:             83425.3,
			ChangeRate:       0.0107,
			ChangePrice:      891.1,
			High:             84149,
			Low:              81035.5,
			Volume:           6204.8008343452275,
			VolumeValue:      514240541.6235756,
			Last:             83425.3,
			AveragePrice:     83541.67284915,
			TakerFeeRate:     0.001,
			MakerFeeRate:     0.001,
			TakerCoefficient: 1,
			MakerCoefficient: 1,
		}
		assert.Equal(t, exp, result, "Get24HourStats should decode every field")
		return
	}
	assert.Equal(t, "BTC-USDT", result.Symbol, "Get24HourStats should return the requested symbol")
}

func TestGetMarketList(t *testing.T) {
	t.Parallel()
	result, err := e.GetMarketList(t.Context())
	require.NoError(t, err, "GetMarketList must not error")
	if mockTests {
		exp := []string{"Majors", "New", "Stocks", "TON", "Meme", "Solana", "AI", "DeFi", "RWA", "Layer1", "Layer2", "NFT", "DePIN", "PoW", "GameFi", "Storage", "Payment", "USDS", "KCS", "FIAT", "VR-AR", "BTC", "ALTS"}
		assert.Equal(t, exp, result, "GetMarketList should decode every market")
		return
	}
	assert.Contains(t, result, "USDS", "GetMarketList should return the USDS market")
}

func TestGetClientIPAddress(t *testing.T) {
	t.Parallel()
	result, err := e.GetClientIPAddress(t.Context())
	require.NoError(t, err, "GetClientIPAddress must not error")
	if mockTests {
		assert.Equal(t, "192.0.2.1", result, "GetClientIPAddress should decode the address")
		return
	}
	assert.NotNil(t, net.ParseIP(result), "GetClientIPAddress should return an IP address")
}

func TestGetCurrentServerTime(t *testing.T) {
	t.Parallel()
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := e.GetCurrentServerTime(ctx)
	require.ErrorIs(t, err, context.Canceled, "GetCurrentServerTime must return the request error")

	result, err := e.GetCurrentServerTime(t.Context())
	require.NoError(t, err, "GetCurrentServerTime must not error")
	if mockTests {
		// The mock serves the spot and futures hosts from one fixture file, so this shares the futures server time
		// fixture, which has the same shape
		assert.Equal(t, time.UnixMilli(1791898075306), result, "GetCurrentServerTime should decode the time")
		return
	}
	assert.WithinDuration(t, time.Now(), result, time.Minute, "GetCurrentServerTime should return the current time")
}

func TestGetServiceStatus(t *testing.T) {
	t.Parallel()
	result, err := e.GetServiceStatus(t.Context())
	require.NoError(t, err, "GetServiceStatus must not error")
	if mockTests {
		// The mock serves the spot and futures hosts from one fixture file, so this shares the futures service status
		// fixture, which has the same shape
		exp := &ServiceStatusResponse{
			Status:  "open",
			Message: "Futures trading is running normally",
		}
		assert.Equal(t, exp, result, "GetServiceStatus should decode every field")
		return
	}
	assert.NotEmpty(t, result.Status, "GetServiceStatus should return a status")
}

func TestGetKYCRegions(t *testing.T) {
	t.Parallel()
	result, err := e.GetKYCRegions(t.Context())
	require.NoError(t, err, "GetKYCRegions must not error")
	if mockTests {
		exp := []KYCRegionResponse{
			{Code: "AD", EnglishName: "Andorra"},
			{Code: "CI", EnglishName: "Côte d'Ivoire"},
			{Code: "GB", EnglishName: "United Kingdom of Great Britain and Northern Ireland"},
		}
		assert.Equal(t, exp, result, "GetKYCRegions should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetKYCRegions should return regions")
}
