package bybit

import (
	"errors"
	"math"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetAdvancedEarnProductInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetAdvancedEarnProductInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAdvancedEarnProductInfo must reject a nil request")
	_, err = e.GetAdvancedEarnProductInfo(t.Context(), &AdvancedEarnProductInfoRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetAdvancedEarnProductInfo must require a category")

	for _, tc := range []struct {
		category string
		exp      *AdvancedEarnProductInfoResponse
	}{
		{category: AdvancedEarnCategoryDualAssets, exp: &AdvancedEarnProductInfoResponse{
			Category: "DualAssets",
			DualAssets: []DualAssetsProduct{
				{
					Category:                   "DualAssets",
					ProductID:                  "155525",
					BaseCoin:                   currency.BTC,
					QuoteCoin:                  currency.USDT,
					ExpectReceiveAt:            types.Time(time.Unix(1791620399, 0)),
					Duration:                   "8h",
					Status:                     "Available",
					SubscribeStartAt:           types.Time(time.Unix(1791532800, 0)),
					SubscribeEndAt:             types.Time(time.Unix(1791590399, 0)),
					ApplyStartAt:               types.Time(time.Unix(1791532800, 0)),
					SettlementTime:             types.Time(time.Unix(1791619199, 0)),
					MinPurchaseQuoteAmount:     20,
					MinPurchaseBaseAmount:      0.0002,
					RemainingAmountQuote:       4000000,
					RemainingAmountBase:        48.4564,
					OrderPrecisionDigitalQuote: 4,
					OrderPrecisionDigitalBase:  8,
				},
				{
					Category:                   "DualAssets",
					ProductID:                  "155555",
					BaseCoin:                   currency.BTC,
					QuoteCoin:                  currency.USDT,
					ExpectReceiveAt:            types.Time(time.Unix(1792138799, 0)),
					Duration:                   "6d",
					Status:                     "Available",
					IsVIPProduct:               true,
					SubscribeStartAt:           types.Time(time.Unix(1791532800, 0)),
					SubscribeEndAt:             types.Time(time.Unix(1791619199, 0)),
					ApplyStartAt:               types.Time(time.Unix(1791532800, 0)),
					SettlementTime:             types.Time(time.Unix(1792137599, 0)),
					MinPurchaseQuoteAmount:     20,
					MinPurchaseBaseAmount:      0.0002,
					RemainingAmountQuote:       4000000,
					RemainingAmountBase:        48.4564,
					OrderPrecisionDigitalQuote: 4,
					OrderPrecisionDigitalBase:  8,
				},
			},
		}},
		{category: AdvancedEarnCategorySmartLeverage, exp: &AdvancedEarnProductInfoResponse{
			Category: "SmartLeverage",
			SmartLeverage: []SmartLeverageProduct{
				{
					Category:              "SmartLeverage",
					ProductID:             "69221",
					InvestCoin:            currency.USDT,
					UnderlyingAsset:       currency.BTC,
					Direction:             "Long",
					Leverage:              50,
					Duration:              "1d",
					SubscribeStartAt:      types.Time(time.Unix(1791504000, 0)),
					SubscribeEndAt:        types.Time(time.Unix(1791561600, 0)),
					SettlementTime:        types.Time(time.Unix(1791619200, 0)),
					ExpectReceiveAt:       types.Time(time.Unix(1791619800, 0)),
					MinPurchaseAmount:     10,
					RemainingAmount:       3962051.4499999997,
					OrderPrecisionDigital: 4,
				},
				{
					Category:              "SmartLeverage",
					ProductID:             "69222",
					InvestCoin:            currency.USDT,
					UnderlyingAsset:       currency.BTC,
					Direction:             "Short",
					Leverage:              50,
					Duration:              "1d",
					SubscribeStartAt:      types.Time(time.Unix(1791504000, 0)),
					SubscribeEndAt:        types.Time(time.Unix(1791561600, 0)),
					SettlementTime:        types.Time(time.Unix(1791619200, 0)),
					ExpectReceiveAt:       types.Time(time.Unix(1791619800, 0)),
					MinPurchaseAmount:     10,
					RemainingAmount:       3962051.4499999997,
					OrderPrecisionDigital: 4,
				},
			},
		}},
		{category: AdvancedEarnCategoryDoubleWin, exp: &AdvancedEarnProductInfoResponse{
			Category: "DoubleWin",
			DoubleWin: []DoubleWinProduct{
				{
					Category:              "DoubleWin",
					ProductID:             "45904",
					InvestCoin:            currency.USDT,
					UnderlyingAsset:       currency.BTC,
					Duration:              "3d",
					SubscribeStartAt:      types.Time(time.Unix(1791504000, 0)),
					SubscribeEndAt:        types.Time(time.Unix(1791590399, 0)),
					SettlementTime:        types.Time(time.Unix(1791792000, 0)),
					ExpectReceiveAt:       types.Time(time.Unix(1791792900, 0)),
					MinPurchaseAmount:     10,
					OrderPrecisionDigital: 4,
					LowerPriceBuffer:      3000,
					UpperPriceBuffer:      3000,
				},
				{
					Category:              "DoubleWin",
					ProductID:             "45881",
					InvestCoin:            currency.USDT,
					UnderlyingAsset:       currency.BTC,
					Duration:              "1d",
					SubscribeStartAt:      types.Time(time.Unix(1791504000, 0)),
					SubscribeEndAt:        types.Time(time.Unix(1791590399, 0)),
					SettlementTime:        types.Time(time.Unix(1791619200, 0)),
					ExpectReceiveAt:       types.Time(time.Unix(1791620100, 0)),
					MinPurchaseAmount:     10,
					OrderPrecisionDigital: 4,
					IsRFQProduct:          true,
					MinDeviationRatio:     0.007,
					MaxDeviationRatio:     0.06,
					PriceTickSize:         500,
				},
			},
		}},
		{category: AdvancedEarnCategoryDiscountBuy, exp: &AdvancedEarnProductInfoResponse{
			Category: "DiscountBuy",
			DiscountBuy: []DiscountBuyProduct{
				{
					Category:              "DiscountBuy",
					ProductID:             "15929",
					Coin:                  currency.USDT,
					UnderlyingAsset:       currency.BTC,
					SettlementTime:        types.Time(time.Unix(1792137600, 0)),
					Duration:              "7d",
					SubscribeStartAt:      types.Time(time.Unix(1791504000, 0)),
					SubscribeEndAt:        types.Time(time.Unix(1791590399, 0)),
					MinPurchaseAmount:     100,
					RemainingAmount:       2000000,
					OrderPrecisionDigital: 4,
					ExpectReceiveAt:       types.Time(time.Unix(1792138500, 0)),
				},
				{
					Category:              "DiscountBuy",
					ProductID:             "15935",
					Coin:                  currency.USDT,
					UnderlyingAsset:       currency.BTC,
					SettlementTime:        types.Time(time.Unix(1792742400, 0)),
					Duration:              "14d",
					IsVIPProduct:          true,
					SubscribeStartAt:      types.Time(time.Unix(1791504000, 0)),
					SubscribeEndAt:        types.Time(time.Unix(1791590399, 0)),
					MinPurchaseAmount:     100,
					RemainingAmount:       2000000,
					OrderPrecisionDigital: 4,
					ExpectReceiveAt:       types.Time(time.Unix(1792743300, 0)),
				},
			},
		}},
	} {
		t.Run(tc.category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAdvancedEarnProductInfo(t.Context(), &AdvancedEarnProductInfoRequest{Category: tc.category, Coin: currency.BTC})
			require.NoError(t, err, "GetAdvancedEarnProductInfo must not error")
			require.NotNil(t, result, "GetAdvancedEarnProductInfo must return a result")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAdvancedEarnProductInfo should decode every field")
				return
			}
			assert.NotZero(t, len(result.DualAssets)+len(result.SmartLeverage)+len(result.DoubleWin)+len(result.DiscountBuy), "GetAdvancedEarnProductInfo should return products")
		})
	}

	t.Run("duration", func(t *testing.T) {
		t.Parallel()
		duration := "8h"
		if !mockTests {
			// Bybit lists different durations over time, so filter live on one it lists now
			products, err := e.GetAdvancedEarnProductInfo(t.Context(), &AdvancedEarnProductInfoRequest{Category: AdvancedEarnCategoryDualAssets, Coin: currency.BTC})
			require.NoError(t, err, "GetAdvancedEarnProductInfo must not error")
			require.NotEmpty(t, products.DualAssets, "GetAdvancedEarnProductInfo must return DualAssets products")
			duration = products.DualAssets[0].Duration
		}
		result, err := e.GetAdvancedEarnProductInfo(t.Context(), &AdvancedEarnProductInfoRequest{Category: AdvancedEarnCategoryDualAssets, Coin: currency.BTC, Duration: duration})
		require.NoError(t, err, "GetAdvancedEarnProductInfo must not error")
		require.NotNil(t, result, "GetAdvancedEarnProductInfo must return a result")
		if mockTests {
			exp := &AdvancedEarnProductInfoResponse{
				Category: "DualAssets",
				DualAssets: []DualAssetsProduct{
					{
						Category:                   "DualAssets",
						ProductID:                  "156215",
						BaseCoin:                   currency.BTC,
						QuoteCoin:                  currency.USDT,
						ExpectReceiveAt:            types.Time(time.Unix(1791965999, 0)),
						Duration:                   "8h",
						Status:                     "Available",
						SubscribeStartAt:           types.Time(time.Unix(1791878400, 0)),
						SubscribeEndAt:             types.Time(time.Unix(1791935999, 0)),
						ApplyStartAt:               types.Time(time.Unix(1791878400, 0)),
						SettlementTime:             types.Time(time.Unix(1791964799, 0)),
						MinPurchaseQuoteAmount:     20,
						MinPurchaseBaseAmount:      0.0002,
						RemainingAmountQuote:       4000000,
						RemainingAmountBase:        48.0581,
						OrderPrecisionDigitalQuote: 4,
						OrderPrecisionDigitalBase:  8,
					},
					{
						Category:                   "DualAssets",
						ProductID:                  "156195",
						BaseCoin:                   currency.BTC,
						QuoteCoin:                  currency.USDT,
						ExpectReceiveAt:            types.Time(time.Unix(1791965999, 0)),
						Duration:                   "8h",
						Status:                     "Available",
						IsVIPProduct:               true,
						SubscribeStartAt:           types.Time(time.Unix(1791878400, 0)),
						SubscribeEndAt:             types.Time(time.Unix(1791935999, 0)),
						ApplyStartAt:               types.Time(time.Unix(1791878400, 0)),
						SettlementTime:             types.Time(time.Unix(1791964799, 0)),
						MinPurchaseQuoteAmount:     20,
						MinPurchaseBaseAmount:      0.0002,
						RemainingAmountQuote:       4000000,
						RemainingAmountBase:        48.0581,
						OrderPrecisionDigitalQuote: 4,
						OrderPrecisionDigitalBase:  8,
					},
				},
			}
			assert.Equal(t, exp, result, "GetAdvancedEarnProductInfo should decode the products of the requested duration")
			return
		}
		require.NotEmpty(t, result.DualAssets, "GetAdvancedEarnProductInfo must return products of a listed duration")
		for i := range result.DualAssets {
			assert.Equalf(t, duration, result.DualAssets[i].Duration, "GetAdvancedEarnProductInfo should only return products of duration %s", duration)
		}
	})
}

func TestAdvancedEarnProductInfoResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const inp = `{"category":"DualAssets","list":[{"category":"DualAssets","productId":"1"},{"category":"SmartLeverage","productId":"2"},{"category":"DoubleWin","productId":"3"},{"category":"DiscountBuy","productId":"4"},{"productId":"5"}]}`
	var result AdvancedEarnProductInfoResponse
	require.NoError(t, json.Unmarshal([]byte(inp), &result), "Unmarshal must not error")
	exp := AdvancedEarnProductInfoResponse{
		Category:      "DualAssets",
		DualAssets:    []DualAssetsProduct{{Category: "DualAssets", ProductID: "1"}, {ProductID: "5"}},
		SmartLeverage: []SmartLeverageProduct{{Category: "SmartLeverage", ProductID: "2"}},
		DoubleWin:     []DoubleWinProduct{{Category: "DoubleWin", ProductID: "3"}},
		DiscountBuy:   []DiscountBuyProduct{{Category: "DiscountBuy", ProductID: "4"}},
	}
	assert.Equal(t, exp, result, "AdvancedEarnProductInfoResponse should decode each product into its category's type")

	err := json.Unmarshal([]byte(`{"category":"SharkFin","list":[{"category":"SharkFin","productId":"6"}]}`), &result)
	assert.ErrorIs(t, err, errUnsupportedAdvancedEarnCategory, "Unmarshal should reject a category without a type")
	err = json.Unmarshal([]byte(`{"list":[{"productId":"7"}]}`), &result)
	assert.ErrorIs(t, err, errUnsupportedAdvancedEarnCategory, "Unmarshal should reject a product without a category")
	assert.Error(t, json.Unmarshal([]byte(`{"list":[{"category":"DualAssets","productId":6}]}`), &result), "Unmarshal should error on a product that does not decode")
	assert.Error(t, json.Unmarshal([]byte(`{"list":[6]}`), &result), "Unmarshal should error on a list element that is not an object")
	assert.Error(t, json.Unmarshal([]byte(`{"list":6}`), &result), "Unmarshal should error on a list that is not an array")
}

func TestDecodeAdvancedEarnElements(t *testing.T) {
	t.Parallel()
	errDecode := errors.New("decode failed")
	for _, tc := range []struct {
		name           string
		resultCategory string
		elements       []string
		exp            []string
		err            error
	}{
		{
			name:           "element category before the result's",
			resultCategory: AdvancedEarnCategoryDualAssets,
			elements:       []string{`{"category":"DiscountBuy","productId":"1"}`, `{"productId":"2"}`, `{"category":"DualAssets","productId":"3"}`},
			exp:            []string{`DiscountBuy {"category":"DiscountBuy","productId":"1"}`, `DualAssets {"productId":"2"}`, `DualAssets {"category":"DualAssets","productId":"3"}`},
		},
		{
			name:     "element category without the result's",
			elements: []string{`{"category":"DiscountBuy","productId":"4"}`},
			exp:      []string{`DiscountBuy {"category":"DiscountBuy","productId":"4"}`},
		},
		{name: "no elements", resultCategory: AdvancedEarnCategoryDualAssets},
		{
			name:           "category without a decoder",
			resultCategory: AdvancedEarnCategoryDualAssets,
			elements:       []string{`{"productId":"5"}`, `{"category":"SharkFin","productId":"6"}`, `{"productId":"7"}`},
			exp:            []string{`DualAssets {"productId":"5"}`},
			err:            errUnsupportedAdvancedEarnCategory,
		},
		{name: "no category", elements: []string{`{"productId":"8"}`}, err: errUnsupportedAdvancedEarnCategory},
		{
			name:           "decoder error",
			resultCategory: AdvancedEarnCategoryDualAssets,
			elements:       []string{`{"category":"Failing","productId":"9"}`, `{"productId":"10"}`},
			err:            errDecode,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var decoded []string
			record := func(category string) func(json.RawMessage) error {
				return func(element json.RawMessage) error {
					decoded = append(decoded, category+" "+string(element))
					return nil
				}
			}
			decoders := map[string]func(json.RawMessage) error{
				AdvancedEarnCategoryDualAssets:  record(AdvancedEarnCategoryDualAssets),
				AdvancedEarnCategoryDiscountBuy: record(AdvancedEarnCategoryDiscountBuy),
				"Failing":                       func(json.RawMessage) error { return errDecode },
			}
			elements := make([]json.RawMessage, len(tc.elements))
			for i := range tc.elements {
				elements[i] = json.RawMessage(tc.elements[i])
			}
			err := decodeAdvancedEarnElements(tc.resultCategory, elements, decoders)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "decodeAdvancedEarnElements should return the expected error")
			} else {
				assert.NoError(t, err, "decodeAdvancedEarnElements should not error")
			}
			assert.Equal(t, tc.exp, decoded, "decodeAdvancedEarnElements should decode each element in order with its category's decoder, stopping at the first error")
		})
	}

	err := decodeAdvancedEarnElements("", []json.RawMessage{json.RawMessage(`{"category":"SharkFin"}`)}, nil)
	assert.ErrorContains(t, err, `"SharkFin"`, "decodeAdvancedEarnElements should name a category it cannot decode")
	err = decodeAdvancedEarnElements(AdvancedEarnCategoryDualAssets, []json.RawMessage{json.RawMessage(`6`)}, map[string]func(json.RawMessage) error{
		AdvancedEarnCategoryDualAssets: func(json.RawMessage) error { return nil },
	})
	assert.Error(t, err, "decodeAdvancedEarnElements should reject an element that is not an object")
}

func TestAdvancedEarnElementAppender(t *testing.T) {
	t.Parallel()
	products := []DualAssetsProduct{{Category: "DualAssets", ProductID: "155525"}}
	appendProduct := advancedEarnElementAppender(&products)
	require.NoError(t, appendProduct(json.RawMessage(`{"category":"DualAssets","productId":"155555","duration":"6d","isVipProduct":true}`)), "the appender must decode a product")
	require.NoError(t, appendProduct(json.RawMessage(`{"category":"DualAssets","productId":"155556","orderPrecisionDigitalBase":8}`)), "the appender must decode a product")
	assert.Error(t, appendProduct(json.RawMessage(`{"category":"DualAssets","productId":155557}`)), "the appender should reject a product that does not decode")
	exp := []DualAssetsProduct{
		{Category: "DualAssets", ProductID: "155525"},
		{Category: "DualAssets", ProductID: "155555", Duration: "6d", IsVIPProduct: true},
		{Category: "DualAssets", ProductID: "155556", OrderPrecisionDigitalBase: 8},
	}
	assert.Equal(t, exp, products, "the appender should append each product it decodes after the existing ones, and none it cannot decode")
}

func TestGetAdvancedEarnProductQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetAdvancedEarnProductQuote(t.Context(), "", "155525")
	require.ErrorIs(t, err, errCategoryNotSet, "GetAdvancedEarnProductQuote must require a category")
	for _, category := range []string{AdvancedEarnCategoryDualAssets, AdvancedEarnCategorySmartLeverage, AdvancedEarnCategoryDoubleWin} {
		_, err = e.GetAdvancedEarnProductQuote(t.Context(), category, "")
		require.ErrorIsf(t, err, errAdvancedEarnProductIDNotSet, "GetAdvancedEarnProductQuote must require a %s product ID", category)
	}

	for _, tc := range []struct {
		name      string
		category  string
		productID string
		exp       *AdvancedEarnProductQuoteResponse
	}{
		{name: AdvancedEarnCategoryDualAssets, category: AdvancedEarnCategoryDualAssets, productID: "155525", exp: &AdvancedEarnProductQuoteResponse{
			Category: "DualAssets",
			DualAssets: []DualAssetsProductQuote{
				{
					ProductID:    "155525",
					CurrentPrice: 82636.86,
					BuyLowPrice: []DualAssetsQuotePrice{
						{
							SelectPrice:         82645.6,
							APYE8:               535749565,
							MaxInvestmentAmount: 796045.3087,
							ExpiredAt:           types.Time(time.UnixMilli(1791534009508)),
						},
						{
							SelectPrice:         82000,
							APYE8:               275275838,
							MaxInvestmentAmount: 364842,
							ExpiredAt:           types.Time(time.UnixMilli(1791534023788)),
						},
					},
					SellHighPrice: []DualAssetsQuotePrice{
						{
							SelectPrice:         82645.8,
							APYE8:               531718557,
							MaxInvestmentAmount: 1.68,
							ExpiredAt:           types.Time(time.UnixMilli(1791534009508)),
						},
						{
							SelectPrice:         83000,
							APYE8:               367822498,
							MaxInvestmentAmount: 7.83132369,
							ExpiredAt:           types.Time(time.UnixMilli(1791534023788)),
						},
					},
				},
			},
		}},
		{name: AdvancedEarnCategorySmartLeverage, category: AdvancedEarnCategorySmartLeverage, productID: "69221", exp: &AdvancedEarnProductQuoteResponse{
			Category: "SmartLeverage",
			SmartLeverage: &SmartLeverageProductQuote{
				Category:            "SmartLeverage",
				ProductID:           "69221",
				BreakevenPrice:      82968.68774554,
				CurrentPrice:        82547.15,
				ExpireAt:            types.Time(time.UnixMilli(1791535319476)),
				MaxInvestmentAmount: 9500,
			},
		}},
		{name: AdvancedEarnCategoryDoubleWin, category: AdvancedEarnCategoryDoubleWin, productID: "45904", exp: &AdvancedEarnProductQuoteResponse{
			Category: "DoubleWin",
			DoubleWin: &DoubleWinProductQuote{
				Category:            "DoubleWin",
				ProductID:           "45904",
				Leverage:            72.76,
				ExpireTime:          types.Time(time.UnixMilli(1791535309229)),
				MaxInvestmentAmount: 5000,
				CurrentPrice:        82547.13,
			},
		}},
		{name: AdvancedEarnCategoryDiscountBuy, category: AdvancedEarnCategoryDiscountBuy, productID: "15929", exp: &AdvancedEarnProductQuoteResponse{
			DiscountBuy: []DiscountBuyProductQuote{
				{
					ProductID:           "15929",
					CurrentPrice:        82652.4,
					PurchasePrice:       81740,
					KnockoutPrice:       84300,
					KnockoutCouponE8:    10000000,
					MaxInvestmentAmount: 950000,
					InstitutionUID:      "103489231",
					ExpiredAt:           types.Time(time.UnixMilli(1791534012022)),
					Category:            "DiscountBuy",
				},
			},
		}},
		{name: "DiscountBuy every product", category: AdvancedEarnCategoryDiscountBuy, exp: &AdvancedEarnProductQuoteResponse{
			DiscountBuy: []DiscountBuyProductQuote{
				{
					ProductID:           "15927",
					CurrentPrice:        82652.97,
					PurchasePrice:       82509,
					KnockoutPrice:       83500,
					KnockoutCouponE8:    10000000,
					MaxInvestmentAmount: 950000,
					InstitutionUID:      "103489231",
					ExpiredAt:           types.Time(time.UnixMilli(1791533986410)),
					Category:            "DiscountBuy",
				},
				{
					ProductID:           "15928",
					CurrentPrice:        2503.78,
					PurchasePrice:       2494,
					KnockoutPrice:       2530,
					KnockoutCouponE8:    10000000,
					MaxInvestmentAmount: 950000,
					InstitutionUID:      "103489231",
					ExpiredAt:           types.Time(time.UnixMilli(1791533986573)),
					Category:            "DiscountBuy",
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			productID := tc.productID
			if productID != "" {
				productID = advancedEarnTestProductID(t, tc.category, productID)
			}
			result, err := e.GetAdvancedEarnProductQuote(t.Context(), tc.category, productID)
			require.NoError(t, err, "GetAdvancedEarnProductQuote must not error")
			require.NotNil(t, result, "GetAdvancedEarnProductQuote must return a result")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAdvancedEarnProductQuote should decode every field")
				return
			}
			if tc.category == AdvancedEarnCategoryDiscountBuy {
				// Bybit's DiscountBuy offers come and go between requests, so a listed product's quote can hold none
				assert.NotNil(t, result.DiscountBuy, "GetAdvancedEarnProductQuote should return the DiscountBuy offers")
				return
			}
			assert.True(t, len(result.DualAssets) != 0 || result.SmartLeverage != nil || result.DoubleWin != nil, "GetAdvancedEarnProductQuote should return a quote")
		})
	}
}

// advancedEarnTestProductID returns the product a test asks for: the fixture's product when mocked, and a listed BTC
// product of the category when live, since Advanced Earn products expire. DoubleWin quotes need a fixed range product
func advancedEarnTestProductID(t *testing.T, category, mockProductID string) string {
	t.Helper()
	if mockTests {
		return mockProductID
	}
	products, err := e.GetAdvancedEarnProductInfo(t.Context(), &AdvancedEarnProductInfoRequest{Category: category, Coin: currency.BTC})
	require.NoError(t, err, "GetAdvancedEarnProductInfo must not error")
	switch category {
	case AdvancedEarnCategoryDualAssets:
		require.NotEmpty(t, products.DualAssets, "GetAdvancedEarnProductInfo must return DualAssets products")
		return products.DualAssets[0].ProductID
	case AdvancedEarnCategorySmartLeverage:
		require.NotEmpty(t, products.SmartLeverage, "GetAdvancedEarnProductInfo must return SmartLeverage products")
		return products.SmartLeverage[0].ProductID
	case AdvancedEarnCategoryDoubleWin:
		for i := range products.DoubleWin {
			if products.DoubleWin[i].IsRFQProduct {
				continue
			}
			_, err := e.GetAdvancedEarnProductQuote(t.Context(), category, products.DoubleWin[i].ProductID)
			// Bybit lists fixed range products whose quotes it refuses with 180007 Product not available. A type
			// assertion, unlike errors.As, matches only that bare refusal, so any other error still fails
			if apiErr, ok := err.(*APIError); ok && apiErr.Code == 180007 {
				continue
			}
			require.NoErrorf(t, err, "GetAdvancedEarnProductQuote must not error for DoubleWin product %s", products.DoubleWin[i].ProductID)
			return products.DoubleWin[i].ProductID
		}
		require.Fail(t, "GetAdvancedEarnProductInfo must return a quoted fixed range DoubleWin product")
	case AdvancedEarnCategoryDiscountBuy:
		require.NotEmpty(t, products.DiscountBuy, "GetAdvancedEarnProductInfo must return DiscountBuy products")
		return products.DiscountBuy[0].ProductID
	}
	require.Failf(t, "unhandled category", "advancedEarnTestProductID must know category %s", category)
	return ""
}

func TestAdvancedEarnProductQuoteResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		inp  string
		exp  AdvancedEarnProductQuoteResponse
	}{
		{
			name: "DualAssets",
			inp:  `{"category":"DualAssets","list":[{"productId":"1","buyLowPrice":[{"selectPrice":"1.5"}]}]}`,
			exp:  AdvancedEarnProductQuoteResponse{Category: "DualAssets", DualAssets: []DualAssetsProductQuote{{ProductID: "1", BuyLowPrice: []DualAssetsQuotePrice{{SelectPrice: 1.5}}}}},
		},
		{
			name: "DualAssets without quotes",
			inp:  `{"category":"DualAssets"}`,
			exp:  AdvancedEarnProductQuoteResponse{Category: "DualAssets"},
		},
		{
			name: "SmartLeverage",
			inp:  `{"category":"SmartLeverage","productId":"2","breakevenPrice":"68650.62"}`,
			exp:  AdvancedEarnProductQuoteResponse{Category: "SmartLeverage", SmartLeverage: &SmartLeverageProductQuote{Category: "SmartLeverage", ProductID: "2", BreakevenPrice: 68650.62}},
		},
		{
			name: "DoubleWin",
			inp:  `{"category":"DoubleWin","productId":"3","leverage":"9.3"}`,
			exp:  AdvancedEarnProductQuoteResponse{Category: "DoubleWin", DoubleWin: &DoubleWinProductQuote{Category: "DoubleWin", ProductID: "3", Leverage: 9.3}},
		},
		{
			name: "DiscountBuy",
			inp:  `{"offers":[{"productId":"4","category":"DiscountBuy"}]}`,
			exp:  AdvancedEarnProductQuoteResponse{DiscountBuy: []DiscountBuyProductQuote{{ProductID: "4", Category: "DiscountBuy"}}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result AdvancedEarnProductQuoteResponse
			require.NoError(t, json.Unmarshal([]byte(tc.inp), &result), "Unmarshal must not error")
			assert.Equal(t, tc.exp, result, "AdvancedEarnProductQuoteResponse should decode the quote into its category's field")
		})
	}

	var result AdvancedEarnProductQuoteResponse
	err := json.Unmarshal([]byte(`{"category":"SharkFin","productId":"5"}`), &result)
	assert.ErrorIs(t, err, errUnsupportedAdvancedEarnCategory, "Unmarshal should reject a category without a type")
	assert.Error(t, json.Unmarshal([]byte(`{"category":6}`), &result), "Unmarshal should error on a result that does not decode")
}

func TestPlaceAdvancedEarnOrder(t *testing.T) {
	t.Parallel()
	_, err := e.PlaceAdvancedEarnOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "PlaceAdvancedEarnOrder must reject a nil request")
	_, err = e.PlaceAdvancedEarnOrder(t.Context(), &AdvancedEarnPlaceOrderRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "PlaceAdvancedEarnOrder must reject an invalid request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AdvancedEarnPlaceOrderRequest
		exp  *AdvancedEarnOrderResponse
	}{
		{
			name: "DualAssets stake",
			req: &AdvancedEarnPlaceOrderRequest{
				Category: AdvancedEarnCategoryDualAssets, ProductID: "36320", OrderType: AdvancedEarnOrderTypeStake, Amount: 20,
				AccountType: "UNIFIED", Coin: currency.USDT, OrderLinkID: "54b3589b-da55-4b17-acdd-aa75912c9eb",
				DualAssetsExtra: &DualAssetsOrderExtra{OrderDirection: "BuyLow", SelectPrice: 2325, APYE8: 857565000},
				InterestCard:    &AdvancedEarnInterestCard{AwardID: 1234567, SpecCode: "DUAL_ASSETS_APR_BOOST"},
			},
			exp: &AdvancedEarnOrderResponse{OrderID: "97f198e9-b14b-4703-b4a6-a4aa06ba1499", OrderLinkID: "54b3589b-da55-4b17-acdd-aa75912c9eb"},
		},
		{
			name: "SmartLeverage stake",
			req: &AdvancedEarnPlaceOrderRequest{
				Category: AdvancedEarnCategorySmartLeverage, ProductID: "12999", OrderType: AdvancedEarnOrderTypeStake, Amount: 100,
				AccountType: "FUND", Coin: currency.USDT, OrderLinkID: "usdt-earn-007",
				SmartLeverageStakeExtra: &SmartLeverageStakeExtra{InitialPrice: 68403, BreakevenPrice: 68650},
			},
			exp: &AdvancedEarnOrderResponse{OrderID: "b3a2c4d1-5e6f-4a7b-8c9d-0e1f2a3b4c5d", OrderLinkID: "usdt-earn-007"},
		},
		{
			name: "SmartLeverage redeem",
			req: &AdvancedEarnPlaceOrderRequest{
				Category: AdvancedEarnCategorySmartLeverage, ProductID: "12999", OrderType: AdvancedEarnOrderTypeRedeem,
				AccountType: "FUND", OrderLinkID: "usdt-earn-008",
				SmartLeverageRedeemExtra: &AdvancedEarnRedeemExtra{PositionID: "1277", EstimatedRedeemAmount: 77.8469, IsSlippageProtected: true},
			},
			exp: &AdvancedEarnOrderResponse{OrderID: "3e7d23fb-acc4-4853-9e32-ab0c0eb43985", OrderLinkID: "usdt-earn-008"},
		},
		{
			name: "DoubleWin fixed range stake",
			req: &AdvancedEarnPlaceOrderRequest{
				Category: AdvancedEarnCategoryDoubleWin, ProductID: "14084", OrderType: AdvancedEarnOrderTypeStake, Amount: 150,
				AccountType: "FUND", Coin: currency.USDT, OrderLinkID: "usdt-earn-009",
				DoubleWinStakeExtra: &DoubleWinStakeExtra{Leverage: 9, InitialPrice: 66445.69},
			},
			exp: &AdvancedEarnOrderResponse{OrderID: "05184c23-8a98-456c-a2af-0ef1c45116cc", OrderLinkID: "usdt-earn-009"},
		},
		{
			name: "DoubleWin RFQ stake",
			req: &AdvancedEarnPlaceOrderRequest{
				Category: AdvancedEarnCategoryDoubleWin, ProductID: "14092", OrderType: AdvancedEarnOrderTypeStake, Amount: 200,
				AccountType: "FUND", Coin: currency.USDT, OrderLinkID: "usdt-earn-010",
				DoubleWinStakeExtra: &DoubleWinStakeExtra{Leverage: 245.18, InitialPrice: 66333.94, LowerPrice: 63000, UpperPrice: 70000},
			},
			exp: &AdvancedEarnOrderResponse{OrderID: "7c1d2e3f-4a5b-4c6d-9e8f-1a2b3c4d5e6f", OrderLinkID: "usdt-earn-010"},
		},
		{
			name: "DoubleWin redeem",
			req: &AdvancedEarnPlaceOrderRequest{
				Category: AdvancedEarnCategoryDoubleWin, ProductID: "14084", OrderType: AdvancedEarnOrderTypeRedeem,
				AccountType: "FUND", OrderLinkID: "usdt-earn-011",
				DoubleWinRedeemExtra: &AdvancedEarnRedeemExtra{PositionID: "2848", EstimatedRedeemAmount: 148.21, IsSlippageProtected: true},
			},
			exp: &AdvancedEarnOrderResponse{OrderID: "e5f6a7b8-c9d0-4e1f-a2b3-c4d5e6f7a8b9", OrderLinkID: "usdt-earn-011"},
		},
		{
			name: "DiscountBuy stake",
			req: &AdvancedEarnPlaceOrderRequest{
				Category: AdvancedEarnCategoryDiscountBuy, ProductID: "7037", OrderType: AdvancedEarnOrderTypeStake, Amount: 200,
				AccountType: "FUND", Coin: currency.USDT, OrderLinkID: "my-order-001",
				DiscountBuyExtra: &DiscountBuyOrderExtra{
					InitialPrice: 74571.32, PurchasePrice: 74019, KnockoutPrice: 76050, KnockoutCouponE8: 1000000,
					SettleType: "Base", InstitutionUID: "100307526",
				},
			},
			exp: &AdvancedEarnOrderResponse{OrderID: "38f6f5ce-57e2-4d69-b4d3-c39464389ccb", OrderLinkID: "my-order-001"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.PlaceAdvancedEarnOrder(t.Context(), tc.req)
			require.NoError(t, err, "PlaceAdvancedEarnOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "PlaceAdvancedEarnOrder should decode the order IDs")
				return
			}
			assert.NotEmpty(t, result.OrderID, "PlaceAdvancedEarnOrder should return an order ID")
		})
	}
}

func TestAdvancedEarnPlaceOrderRequestPayload(t *testing.T) {
	t.Parallel()
	_, err := (*AdvancedEarnPlaceOrderRequest)(nil).Payload()
	require.ErrorIs(t, err, common.ErrNilPointer, "Payload must reject a nil request")

	newRequest := func() *AdvancedEarnPlaceOrderRequest {
		return &AdvancedEarnPlaceOrderRequest{
			Category: AdvancedEarnCategoryDualAssets, ProductID: "36320", OrderType: AdvancedEarnOrderTypeStake, Amount: 20.5,
			AccountType: "UNIFIED", Coin: currency.NewCode("usdt"), OrderLinkID: "link-1",
			DualAssetsExtra:          &DualAssetsOrderExtra{OrderDirection: "BuyLow", SelectPrice: 2325.5, APYE8: 857565000},
			SmartLeverageStakeExtra:  &SmartLeverageStakeExtra{InitialPrice: 68403.1, BreakevenPrice: 68650.2},
			SmartLeverageRedeemExtra: &AdvancedEarnRedeemExtra{PositionID: "1277", EstimatedRedeemAmount: 77.8469, IsSlippageProtected: true},
			DoubleWinStakeExtra:      &DoubleWinStakeExtra{Leverage: 245.18, InitialPrice: 66333.94, LowerPrice: 63000, UpperPrice: 70000},
			DoubleWinRedeemExtra:     &AdvancedEarnRedeemExtra{PositionID: "2848", EstimatedRedeemAmount: 148.21},
			DiscountBuyExtra: &DiscountBuyOrderExtra{
				InitialPrice: 74571.32, PurchasePrice: 74019, KnockoutPrice: 76050, KnockoutCouponE8: 1000000, SettleType: "Base",
				InstitutionUID: "100307526",
			},
			InterestCard: &AdvancedEarnInterestCard{AwardID: 1234567, SpecCode: "DUAL_ASSETS_APR_BOOST"},
		}
	}
	body, err := newRequest().Payload()
	require.NoError(t, err, "Payload must not error")
	exp := map[string]any{
		"category":    "DualAssets",
		"productId":   "36320",
		"orderType":   "Stake",
		"orderLinkId": "link-1",
		"amount":      "20.5",
		"accountType": "UNIFIED",
		"coin":        "USDT",
		"dualAssetsExtra": map[string]any{
			"orderDirection": "BuyLow",
			"selectPrice":    "2325.5",
			"apyE8":          uint64(857565000),
		},
		"smartLeverageStakeExtra": map[string]any{
			"initialPrice":   "68403.1",
			"breakevenPrice": "68650.2",
		},
		"smartLeverageRedeemExtra": map[string]any{
			"positionId":          "1277",
			"estRedeemAmount":     "77.8469",
			"isSlippageProtected": true,
		},
		"doubleWinStakeExtra": map[string]any{
			"leverage":     "245.18",
			"initialPrice": "66333.94",
			"lowerPrice":   "63000",
			"upperPrice":   "70000",
		},
		"doubleWinRedeemExtra": map[string]any{
			"positionId":      "2848",
			"estRedeemAmount": "148.21",
		},
		"discountBuyExtra": map[string]any{
			"initialPrice":     "74571.32",
			"purchasePrice":    "74019",
			"knockoutPrice":    "76050",
			"knockoutCouponE8": "1000000",
			"settleType":       "Base",
			"instUid":          "100307526",
		},
		"interestCard": map[string]any{
			"awardId":  uint64(1234567),
			"specCode": "DUAL_ASSETS_APR_BOOST",
		},
	}
	assert.Equal(t, exp, body, "Payload should build every parameter")

	smartLeverageRedeem := &AdvancedEarnPlaceOrderRequest{
		Category: AdvancedEarnCategorySmartLeverage, ProductID: "12999", OrderType: AdvancedEarnOrderTypeRedeem, OrderLinkID: "link-2",
		SmartLeverageRedeemExtra: &AdvancedEarnRedeemExtra{PositionID: "1277", EstimatedRedeemAmount: 77.8469},
	}
	_, err = smartLeverageRedeem.Payload()
	require.ErrorIs(t, err, errMissingAccountType, "Payload must require the account type of a SmartLeverage redemption")
	_, err = (&AdvancedEarnPlaceOrderRequest{
		Category: AdvancedEarnCategoryDoubleWin, ProductID: "14084", OrderType: AdvancedEarnOrderTypeRedeem, OrderLinkID: "link-3",
		DoubleWinRedeemExtra: &AdvancedEarnRedeemExtra{PositionID: "2848", EstimatedRedeemAmount: 148.21},
	}).Payload()
	require.NoError(t, err, "Payload must not require the account type of a DoubleWin redemption")
	smartLeverageRedeem.AccountType = "FUND"
	body, err = smartLeverageRedeem.Payload()
	require.NoError(t, err, "Payload must not require a stake's amount, coin or extra parameters for a redemption")
	exp = map[string]any{
		"category":    "SmartLeverage",
		"productId":   "12999",
		"orderType":   "Redeem",
		"orderLinkId": "link-2",
		"accountType": "FUND",
		"smartLeverageRedeemExtra": map[string]any{
			"positionId":      "1277",
			"estRedeemAmount": "77.8469",
		},
	}
	assert.Equal(t, exp, body, "Payload should omit unset parameters")

	for _, tc := range []struct {
		category, orderType, extra string
		unset                      func(*AdvancedEarnPlaceOrderRequest)
	}{
		{category: AdvancedEarnCategoryDualAssets, orderType: AdvancedEarnOrderTypeStake, extra: "dualAssetsExtra", unset: func(r *AdvancedEarnPlaceOrderRequest) { r.DualAssetsExtra = nil }},
		{category: AdvancedEarnCategoryDiscountBuy, orderType: AdvancedEarnOrderTypeStake, extra: "discountBuyExtra", unset: func(r *AdvancedEarnPlaceOrderRequest) { r.DiscountBuyExtra = nil }},
		{category: AdvancedEarnCategorySmartLeverage, orderType: AdvancedEarnOrderTypeStake, extra: "smartLeverageStakeExtra", unset: func(r *AdvancedEarnPlaceOrderRequest) { r.SmartLeverageStakeExtra = nil }},
		{category: AdvancedEarnCategorySmartLeverage, orderType: AdvancedEarnOrderTypeRedeem, extra: "smartLeverageRedeemExtra", unset: func(r *AdvancedEarnPlaceOrderRequest) { r.SmartLeverageRedeemExtra = nil }},
		{category: AdvancedEarnCategoryDoubleWin, orderType: AdvancedEarnOrderTypeStake, extra: "doubleWinStakeExtra", unset: func(r *AdvancedEarnPlaceOrderRequest) { r.DoubleWinStakeExtra = nil }},
		{category: AdvancedEarnCategoryDoubleWin, orderType: AdvancedEarnOrderTypeRedeem, extra: "doubleWinRedeemExtra", unset: func(r *AdvancedEarnPlaceOrderRequest) { r.DoubleWinRedeemExtra = nil }},
	} {
		t.Run(tc.category+" "+tc.orderType+" without "+tc.extra, func(t *testing.T) {
			t.Parallel()
			r := newRequest()
			r.Category, r.OrderType = tc.category, tc.orderType
			_, err := r.Payload()
			require.NoError(t, err, "Payload must accept a request with every extra set")
			// Every other category's and order type's extra stays set, so none of them may stand in for this one
			tc.unset(r)
			_, err = r.Payload()
			require.ErrorIs(t, err, errAdvancedEarnExtraNotSet, "Payload must require the extra parameters of the category and order type")
			assert.ErrorContains(t, err, tc.extra, "Payload should name the missing extra parameters")
		})
	}

	for _, tc := range []struct {
		name   string
		modify func(*AdvancedEarnPlaceOrderRequest)
		err    error
	}{
		{name: "category", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.Category = "" }, err: errCategoryNotSet},
		{name: "productId", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.ProductID = "" }, err: errAdvancedEarnProductIDNotSet},
		{name: "orderType", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.OrderType = "" }, err: errAdvancedEarnOrderTypeNotSet},
		{name: "orderLinkId", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.OrderLinkID = "" }, err: errOrderLinkIDMissing},
		{name: "stake amount", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.Amount = 0 }, err: order.ErrAmountIsInvalid},
		{name: "stake coin", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.Coin = currency.EMPTYCODE }, err: currency.ErrCurrencyCodeEmpty},
		{name: "stake accountType", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.AccountType = "" }, err: errMissingAccountType},
		{name: "dualAssetsExtra orderDirection", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DualAssetsExtra.OrderDirection = "" }, err: errAdvancedEarnDirectionNotSet},
		{name: "dualAssetsExtra selectPrice", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DualAssetsExtra.SelectPrice = 0 }, err: errAdvancedEarnPriceNotSet},
		{name: "dualAssetsExtra apyE8", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DualAssetsExtra.APYE8 = 0 }, err: errAdvancedEarnAPYNotSet},
		{name: "smartLeverageStakeExtra initialPrice", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.SmartLeverageStakeExtra.InitialPrice = 0 }, err: errAdvancedEarnPriceNotSet},
		{name: "smartLeverageStakeExtra breakevenPrice", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.SmartLeverageStakeExtra.BreakevenPrice = 0 }, err: errAdvancedEarnPriceNotSet},
		{name: "smartLeverageRedeemExtra positionId", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.SmartLeverageRedeemExtra.PositionID = "" }, err: errAdvancedEarnPositionIDNotSet},
		{name: "smartLeverageRedeemExtra estRedeemAmount", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.SmartLeverageRedeemExtra.EstimatedRedeemAmount = 0 }, err: errAdvancedEarnRedeemAmountNotSet},
		{name: "doubleWinStakeExtra leverage", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DoubleWinStakeExtra.Leverage = 0 }, err: errAdvancedEarnLeverageNotSet},
		{name: "doubleWinStakeExtra initialPrice", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DoubleWinStakeExtra.InitialPrice = 0 }, err: errAdvancedEarnPriceNotSet},
		{name: "doubleWinRedeemExtra positionId", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DoubleWinRedeemExtra.PositionID = "" }, err: errAdvancedEarnPositionIDNotSet},
		{name: "discountBuyExtra initialPrice", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DiscountBuyExtra.InitialPrice = 0 }, err: errAdvancedEarnPriceNotSet},
		{name: "discountBuyExtra purchasePrice", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DiscountBuyExtra.PurchasePrice = 0 }, err: errAdvancedEarnPriceNotSet},
		{name: "discountBuyExtra knockoutPrice", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DiscountBuyExtra.KnockoutPrice = 0 }, err: errAdvancedEarnPriceNotSet},
		{name: "discountBuyExtra knockoutCouponE8", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DiscountBuyExtra.KnockoutCouponE8 = 0 }, err: errAdvancedEarnKnockoutCouponNotSet},
		{name: "discountBuyExtra settleType", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DiscountBuyExtra.SettleType = "" }, err: errAdvancedEarnSettleTypeNotSet},
		{name: "discountBuyExtra instUid", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.DiscountBuyExtra.InstitutionUID = "" }, err: errAdvancedEarnInstitutionUIDNotSet},
		{name: "interestCard awardId", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.InterestCard.AwardID = 0 }, err: errAdvancedEarnAwardIDNotSet},
		{name: "interestCard specCode", modify: func(r *AdvancedEarnPlaceOrderRequest) { r.InterestCard.SpecCode = "" }, err: errAdvancedEarnSpecCodeNotSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r := newRequest()
			tc.modify(r)
			_, err := r.Payload()
			assert.ErrorIs(t, err, tc.err, "Payload should reject the request")
		})
	}
}

func TestGetAdvancedEarnPositionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetAdvancedEarnPositionInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAdvancedEarnPositionInfo must reject a nil request")
	_, err = e.GetAdvancedEarnPositionInfo(t.Context(), &AdvancedEarnPositionInfoRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetAdvancedEarnPositionInfo must require a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *AdvancedEarnPositionInfoRequest
		exp *AdvancedEarnPositionInfoResponse
	}{
		{req: &AdvancedEarnPositionInfoRequest{Category: AdvancedEarnCategoryDualAssets, Coin: currency.USDT, Limit: 20}, exp: &AdvancedEarnPositionInfoResponse{
			Category:       "DualAssets",
			NextPageCursor: "eyJwb3NpdGlvbklkIjoxOTAzNX0=",
			DualAssets: []DualAssetsPosition{
				{
					PositionID:         "19035",
					ProductID:          "36320",
					Category:           "DualAssets",
					BaseCoin:           currency.ETH,
					QuoteCoin:          currency.USDT,
					InvestCoin:         currency.USDT,
					Amount:             20,
					APYE8:              902700000,
					Direction:          "BuyLow",
					TargetPrice:        2325,
					SettlementTime:     types.Time(time.Unix(1774079999, 0)),
					Status:             "Active",
					OrderID:            "ef9644c2-0c98-40e5-ae33-8aa5b6d85058",
					Duration:           "3d",
					ExpectReturnCoin:   currency.USDT,
					ExpectReturnAmount: 21.4838,
					AccountType:        "UNIFIED",
					ToAccountType:      "FUND",
					YieldStartAt:       types.Time(time.Unix(1773820799, 0)),
					YieldEndAt:         types.Time(time.Unix(1774079999, 0)),
				},
			},
		}},
		{req: &AdvancedEarnPositionInfoRequest{Category: AdvancedEarnCategorySmartLeverage, ProductID: "12959"}, exp: &AdvancedEarnPositionInfoResponse{
			Category: "SmartLeverage",
			SmartLeverage: []SmartLeveragePosition{
				{
					PositionID:      "1252",
					ProductID:       "12959",
					Category:        "SmartLeverage",
					InvestCoin:      currency.USDT,
					Amount:          1000,
					Duration:        "1d",
					SettlementTime:  types.Time(time.Unix(1774944000, 0)),
					AccountType:     "FUND",
					OrderID:         "d7be0f06-af7b-4ae8-bd2f-37000d67edf2",
					UnderlyingAsset: currency.BTC,
					Direction:       "Long",
					Leverage:        50,
					BreakevenPrice:  67367.16,
					InitialPrice:    67243.31,
					Status:          "Active",
					CreatedTime:     types.Time(time.Unix(1774838832, 0)),
					Redeemable:      true,
					OrderLinkID:     "usdt-earn-005",
				},
			},
		}},
		{req: &AdvancedEarnPositionInfoRequest{Category: AdvancedEarnCategoryDoubleWin, ProductID: "14084"}, exp: &AdvancedEarnPositionInfoResponse{
			Category: "DoubleWin",
			DoubleWin: []DoubleWinPosition{
				{
					PositionID:      "2848",
					ProductID:       "14084",
					Category:        "DoubleWin",
					InvestCoin:      currency.USDT,
					UnderlyingAsset: currency.BTC,
					Amount:          150,
					Leverage:        9.330979317884342,
					InitialPrice:    66445.69,
					LowerPrice:      66245.69,
					UpperPrice:      66645.69,
					Duration:        "1d",
					SettlementTime:  types.Time(time.Unix(1775203200, 0)),
					CreatedTime:     types.Time(time.Unix(1775107014, 0)),
					Status:          "Active",
					Redeemable:      true,
					AccountType:     "FUND",
					OrderID:         "05184c23-8a98-456c-a2af-0ef1c45116cc",
				},
			},
		}},
		{req: &AdvancedEarnPositionInfoRequest{Category: AdvancedEarnCategoryDiscountBuy, ProductID: "7037", Cursor: "eyJwb3NpdGlvbklkIjoxMTk2MH0="}, exp: &AdvancedEarnPositionInfoResponse{
			DiscountBuy: []DiscountBuyPosition{
				{
					PositionID:       "11959",
					ProductID:        "7037",
					Category:         "DiscountBuy",
					Coin:             currency.USDT,
					UnderlyingAsset:  currency.BTC,
					Amount:           200,
					PurchasePrice:    74019,
					KnockoutPrice:    76050,
					KnockoutCouponE8: 1000000,
					Status:           "Active",
					OrderID:          "38f6f5ce-57e2-4d69-b4d3-c39464389ccb",
					Duration:         "1d",
					SettlementTime:   types.Time(time.Unix(1776240000, 0)),
					AccountType:      "FUND",
					ToAccountType:    "FUND",
					SettleType:       "Base",
					ExpectReceiveAt:  types.Time(time.Unix(1776240900, 0)),
				},
			},
		}},
	} {
		t.Run(tc.req.Category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAdvancedEarnPositionInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetAdvancedEarnPositionInfo must not error")
			require.NotNil(t, result, "GetAdvancedEarnPositionInfo must return a result")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAdvancedEarnPositionInfo should decode every field")
			}
		})
	}
}

func TestAdvancedEarnPositionInfoResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const inp = `{"category":"","nextPageCursor":"abc","list":[{"category":"DualAssets","positionId":"1"},{"category":"SmartLeverage","positionId":"2"},{"category":"DoubleWin","positionId":"3"},{"category":"DiscountBuy","positionId":"4"}]}`
	var result AdvancedEarnPositionInfoResponse
	require.NoError(t, json.Unmarshal([]byte(inp), &result), "Unmarshal must not error")
	exp := AdvancedEarnPositionInfoResponse{
		NextPageCursor: "abc",
		DualAssets:     []DualAssetsPosition{{Category: "DualAssets", PositionID: "1"}},
		SmartLeverage:  []SmartLeveragePosition{{Category: "SmartLeverage", PositionID: "2"}},
		DoubleWin:      []DoubleWinPosition{{Category: "DoubleWin", PositionID: "3"}},
		DiscountBuy:    []DiscountBuyPosition{{Category: "DiscountBuy", PositionID: "4"}},
	}
	assert.Equal(t, exp, result, "AdvancedEarnPositionInfoResponse should decode each position into its category's type")

	err := json.Unmarshal([]byte(`{"list":[{"category":"SharkFin","positionId":"5"}]}`), &result)
	assert.ErrorIs(t, err, errUnsupportedAdvancedEarnCategory, "Unmarshal should reject a category without a type")
	assert.Error(t, json.Unmarshal([]byte(`{"list":6}`), &result), "Unmarshal should error on a list that is not an array")
}

func TestGetAdvancedEarnOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetAdvancedEarnOrderInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAdvancedEarnOrderInfo must reject a nil request")
	_, err = e.GetAdvancedEarnOrderInfo(t.Context(), &AdvancedEarnOrderInfoRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetAdvancedEarnOrderInfo must require a category")
	_, err = e.GetAdvancedEarnOrderInfo(t.Context(), &AdvancedEarnOrderInfoRequest{
		Category: AdvancedEarnCategoryDualAssets, StartTime: time.UnixMilli(1773900000000), EndTime: time.UnixMilli(1773800000000),
	})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAdvancedEarnOrderInfo must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		req *AdvancedEarnOrderInfoRequest
		exp *AdvancedEarnOrderInfoResponse
	}{
		{req: &AdvancedEarnOrderInfoRequest{Category: AdvancedEarnCategoryDualAssets, StartTime: time.UnixMilli(1773800000000), EndTime: time.UnixMilli(1773900000000), Limit: 2}, exp: &AdvancedEarnOrderInfoResponse{
			Category:       "DualAssets",
			NextPageCursor: "eyJvcmRlcklkIjoiYTFiMmMzZDQifQ==",
			DualAssets: []DualAssetsOrder{
				{
					OrderID:          "ef9644c2-0c98-40e5-ae33-8aa5b6d85058",
					OrderLinkID:      "54b3589b-da55-4b17-acdd-aa75912c9eb9",
					ProductID:        "36320",
					Category:         "DualAssets",
					OrderType:        "Stake",
					Amount:           20,
					Coin:             currency.USDT,
					BaseCoin:         currency.ETH,
					QuoteCoin:        currency.USDT,
					Status:           "Settled",
					CreatedTime:      types.Time(time.Unix(1773815357, 0)),
					UpdatedTime:      types.Time(time.Unix(1774080005, 0)),
					Direction:        "BuyLow",
					TargetPrice:      2325,
					SettlementTime:   types.Time(time.Unix(1774079999, 0)),
					EstimateAPYE8:    771808500,
					Duration:         "3d",
					AccountType:      "UNIFIED",
					ToAccountType:    "FUND",
					SelectAPYE8:      857565000,
					IsVIP:            true,
					SettlementCoin:   currency.ETH,
					SettlementAmount: 0.00924034,
					OrderMode:        "Normal",
					SettlementPrice:  2310.5,
					TrialBonusAmount: 5,
					TrialBonusPNL:    0.0427,
				},
				{
					OrderID:       "a1b2c3d4-e5f6-4a7b-8c9d-0e1f2a3b4c5d",
					OrderLinkID:   "54b3589b-da55-4b17-acdd-aa75912c9ec0",
					ProductID:     "36321",
					Category:      "DualAssets",
					OrderType:     "Stake",
					Amount:        0.01,
					Coin:          currency.ETH,
					BaseCoin:      currency.ETH,
					QuoteCoin:     currency.USDT,
					Status:        "Fail",
					CreatedTime:   types.Time(time.Unix(1773815360, 0)),
					UpdatedTime:   types.Time(time.Unix(1773815361, 0)),
					Direction:     "SellHigh",
					TargetPrice:   2400,
					EstimateAPYE8: 650000000,
					Duration:      "1d",
					AccountType:   "FUND",
					ToAccountType: "FUND",
					SelectAPYE8:   700000000,
					OrderMode:     "RFQ",
					RefundStatus:  "Processed",
				},
			},
		}},
		{req: &AdvancedEarnOrderInfoRequest{Category: AdvancedEarnCategorySmartLeverage, OrderID: "3e7d23fb-acc4-4853-9e32-ab0c0eb43985"}, exp: &AdvancedEarnOrderInfoResponse{
			Category: "SmartLeverage",
			SmartLeverage: []SmartLeverageOrder{
				{
					OrderID:         "3e7d23fb-acc4-4853-9e32-ab0c0eb43985",
					OrderLinkID:     "usdt-earn-008",
					ProductID:       "12999",
					Category:        "SmartLeverage",
					OrderType:       "Redeem",
					Amount:          100,
					CreatedTime:     types.Time(time.Unix(1775038326, 0)),
					Status:          "Settled",
					Duration:        "2d",
					AccountType:     "FUND",
					SettlementTime:  types.Time(time.Unix(1775038327, 0)),
					InvestCoin:      currency.USDT,
					UnderlyingAsset: currency.BTC,
					Direction:       "Long",
					Leverage:        200,
					BreakevenPrice:  68622.69,
					InitialPrice:    68403,
					SettlementPrice: 68566.72,
					PNL:             83.6876,
					ToAccountType:   "FUND",
				},
				{
					OrderID:         "4f8e34fc-bdd5-4964-af43-bc1d1fc54096",
					OrderLinkID:     "usdt-earn-009",
					ProductID:       "12999",
					Category:        "SmartLeverage",
					OrderType:       "Stake",
					Amount:          100,
					CreatedTime:     types.Time(time.Unix(1775038330, 0)),
					Status:          "Fail",
					Duration:        "2d",
					AccountType:     "FUND",
					SettlementTime:  types.Time(time.Unix(1775203200, 0)),
					RefundStatus:    "Processed",
					InvestCoin:      currency.USDT,
					UnderlyingAsset: currency.BTC,
					Direction:       "Short",
					Leverage:        200,
					BreakevenPrice:  68183.31,
					InitialPrice:    68403,
					ToAccountType:   "FUND",
				},
			},
		}},
		{req: &AdvancedEarnOrderInfoRequest{Category: AdvancedEarnCategoryDoubleWin, OrderID: "05184c23-8a98-456c-a2af-0ef1c45116cc"}, exp: &AdvancedEarnOrderInfoResponse{
			Category: "DoubleWin",
			DoubleWin: []DoubleWinOrder{
				{
					OrderID:         "05184c23-8a98-456c-a2af-0ef1c45116cc",
					OrderLinkID:     "usdt-earn-009",
					ProductID:       "14084",
					Category:        "DoubleWin",
					OrderType:       "Stake",
					InvestCoin:      currency.USDT,
					Amount:          150,
					UnderlyingAsset: currency.BTC,
					Leverage:        9.330979317884342,
					InitialPrice:    66445.69,
					LowerPrice:      66245.69,
					UpperPrice:      66645.69,
					Duration:        "1d",
					SettlementTime:  types.Time(time.Unix(1775203200, 0)),
					CreatedTime:     types.Time(time.Unix(1775107011, 0)),
					UpdatedTime:     types.Time(time.Unix(1775203260, 0)),
					Status:          "Settled",
					SettlementPrice: 66702.15,
					PNL:             79.6512,
					AccountType:     "FUND",
					ToAccountType:   "FUND",
				},
				{
					OrderID:         "16295d34-9ba9-467d-b3b0-1fd2d56227dd",
					OrderLinkID:     "usdt-earn-012",
					ProductID:       "14084",
					Category:        "DoubleWin",
					OrderType:       "Stake",
					InvestCoin:      currency.USDT,
					Amount:          150,
					UnderlyingAsset: currency.BTC,
					Leverage:        9.33,
					InitialPrice:    66445.69,
					LowerPrice:      66245.69,
					UpperPrice:      66645.69,
					Duration:        "1d",
					SettlementTime:  types.Time(time.Unix(1775203200, 0)),
					CreatedTime:     types.Time(time.Unix(1775107020, 0)),
					UpdatedTime:     types.Time(time.Unix(1775107021, 0)),
					Status:          "Fail",
					RefundStatus:    "Processing",
					AccountType:     "UNIFIED",
					ToAccountType:   "UNIFIED",
				},
			},
		}},
		{req: &AdvancedEarnOrderInfoRequest{Category: AdvancedEarnCategoryDiscountBuy, OrderLinkID: "my-order-001"}, exp: &AdvancedEarnOrderInfoResponse{
			DiscountBuy: []DiscountBuyOrder{
				{
					OrderID:          "38f6f5ce-57e2-4d69-b4d3-c39464389ccb",
					OrderLinkID:      "my-order-001",
					ProductID:        "7037",
					Category:         "DiscountBuy",
					OrderType:        "Stake",
					Amount:           200,
					Coin:             currency.USDT,
					UnderlyingAsset:  currency.BTC,
					Status:           "Settled",
					CreatedTime:      types.Time(time.Unix(1776154116, 0)),
					PurchasePrice:    74019,
					KnockoutPrice:    76050,
					KnockoutCouponE8: 1000000,
					Duration:         "1d",
					SettlementTime:   types.Time(time.Unix(1776240000, 0)),
					AccountType:      "FUND",
					ToAccountType:    "FUND",
					SettleType:       "Base",
					SettlementPrice:  73650.4,
					SettlementCoin:   currency.BTC,
					SettlementAmount: 0.00270199,
					IsVIP:            true,
				},
				{
					OrderID:          "49a7a6df-68f3-4e7a-c5e4-d4a575490ddc",
					OrderLinkID:      "my-order-001",
					ProductID:        "7037",
					Category:         "DiscountBuy",
					OrderType:        "Stake",
					Amount:           200,
					Coin:             currency.USDT,
					UnderlyingAsset:  currency.BTC,
					Status:           "Fail",
					CreatedTime:      types.Time(time.Unix(1776154120, 0)),
					PurchasePrice:    74019,
					KnockoutPrice:    76050,
					KnockoutCouponE8: 1000000,
					Duration:         "1d",
					SettlementTime:   types.Time(time.Unix(1776240000, 0)),
					AccountType:      "FUND",
					ToAccountType:    "FUND",
					SettleType:       "Quote",
					RefundStatus:     "REFUND_DONE",
				},
			},
		}},
	} {
		t.Run(tc.req.Category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAdvancedEarnOrderInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetAdvancedEarnOrderInfo must not error")
			require.NotNil(t, result, "GetAdvancedEarnOrderInfo must return a result")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAdvancedEarnOrderInfo should decode every field")
			}
		})
	}

	t.Run("product pages", func(t *testing.T) {
		t.Parallel()
		req := &AdvancedEarnOrderInfoRequest{Category: AdvancedEarnCategoryDualAssets, ProductID: "36320", Limit: 1}
		first, err := e.GetAdvancedEarnOrderInfo(t.Context(), req)
		require.NoError(t, err, "GetAdvancedEarnOrderInfo must not error")
		require.NotNil(t, first, "GetAdvancedEarnOrderInfo must return a result")
		if mockTests {
			exp := &AdvancedEarnOrderInfoResponse{
				Category:       "DualAssets",
				NextPageCursor: "eyJvcmRlcklkIjoiZWY5NjQ0YzIifQ==",
				DualAssets: []DualAssetsOrder{
					{
						OrderID:          "ef9644c2-0c98-40e5-ae33-8aa5b6d85058",
						OrderLinkID:      "54b3589b-da55-4b17-acdd-aa75912c9eb9",
						ProductID:        "36320",
						Category:         "DualAssets",
						OrderType:        "Stake",
						Amount:           20,
						Coin:             currency.USDT,
						BaseCoin:         currency.ETH,
						QuoteCoin:        currency.USDT,
						Status:           "Settled",
						CreatedTime:      types.Time(time.Unix(1773815357, 0)),
						UpdatedTime:      types.Time(time.Unix(1774080005, 0)),
						Direction:        "BuyLow",
						TargetPrice:      2325,
						SettlementTime:   types.Time(time.Unix(1774079999, 0)),
						EstimateAPYE8:    771808500,
						Duration:         "3d",
						AccountType:      "UNIFIED",
						ToAccountType:    "FUND",
						SelectAPYE8:      857565000,
						IsVIP:            true,
						SettlementCoin:   currency.ETH,
						SettlementAmount: 0.00924034,
						OrderMode:        "Normal",
						SettlementPrice:  2310.5,
						TrialBonusAmount: 5,
						TrialBonusPNL:    0.0427,
					},
				},
			}
			assert.Equal(t, exp, first, "GetAdvancedEarnOrderInfo should decode the first page of the product's orders")
		} else if first.NextPageCursor == "" {
			return // The account's orders of the product fit on one page
		}
		req.Cursor = first.NextPageCursor
		next, err := e.GetAdvancedEarnOrderInfo(t.Context(), req)
		require.NoError(t, err, "GetAdvancedEarnOrderInfo must not error")
		require.NotNil(t, next, "GetAdvancedEarnOrderInfo must return a result")
		if mockTests {
			exp := &AdvancedEarnOrderInfoResponse{
				Category: "DualAssets",
				DualAssets: []DualAssetsOrder{
					{
						OrderID:       "6c2d7e8f-1a2b-4c3d-9e8f-7a6b5c4d3e2f",
						OrderLinkID:   "54b3589b-da55-4b17-acdd-aa75912c9ec1",
						ProductID:     "36320",
						Category:      "DualAssets",
						OrderType:     "Stake",
						Amount:        50,
						Coin:          currency.USDT,
						BaseCoin:      currency.ETH,
						QuoteCoin:     currency.USDT,
						Status:        "Success",
						CreatedTime:   types.Time(time.Unix(1773815390, 0)),
						UpdatedTime:   types.Time(time.Unix(1773815391, 0)),
						Direction:     "BuyLow",
						TargetPrice:   2325,
						EstimateAPYE8: 771808500,
						Duration:      "3d",
						AccountType:   "FUND",
						ToAccountType: "FUND",
						SelectAPYE8:   857565000,
						OrderMode:     "Normal",
					},
				},
			}
			assert.Equal(t, exp, next, "GetAdvancedEarnOrderInfo should decode the last page of the product's orders")
			return
		}
		for i := range next.DualAssets {
			assert.Equal(t, req.ProductID, next.DualAssets[i].ProductID, "GetAdvancedEarnOrderInfo should only return orders of the requested product")
		}
	})
}

func TestAdvancedEarnOrderInfoResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	const inp = `{"category":"","nextPageCursor":"abc","list":[{"category":"DualAssets","orderId":"1"},{"category":"SmartLeverage","orderId":"2"},{"category":"DoubleWin","orderId":"3"},{"category":"DiscountBuy","orderId":"4"}]}`
	var result AdvancedEarnOrderInfoResponse
	require.NoError(t, json.Unmarshal([]byte(inp), &result), "Unmarshal must not error")
	exp := AdvancedEarnOrderInfoResponse{
		NextPageCursor: "abc",
		DualAssets:     []DualAssetsOrder{{Category: "DualAssets", OrderID: "1"}},
		SmartLeverage:  []SmartLeverageOrder{{Category: "SmartLeverage", OrderID: "2"}},
		DoubleWin:      []DoubleWinOrder{{Category: "DoubleWin", OrderID: "3"}},
		DiscountBuy:    []DiscountBuyOrder{{Category: "DiscountBuy", OrderID: "4"}},
	}
	assert.Equal(t, exp, result, "AdvancedEarnOrderInfoResponse should decode each order into its category's type")

	err := json.Unmarshal([]byte(`{"list":[{"category":"SharkFin","orderId":"5"}]}`), &result)
	assert.ErrorIs(t, err, errUnsupportedAdvancedEarnCategory, "Unmarshal should reject a category without a type")
	assert.Error(t, json.Unmarshal([]byte(`{"list":6}`), &result), "Unmarshal should error on a list that is not an array")
}

func TestSetAdvancedEarnTimeWindow(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1775100000000), time.UnixMilli(1775200000000)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		exp        url.Values
		err        error
	}{
		{name: "no window", exp: url.Values{}},
		{name: "start only", start: start, exp: url.Values{"startTime": {"1775100000000"}}},
		{name: "end only", end: end, exp: url.Values{"endTime": {"1775200000000"}}},
		{name: "both bounds", start: start, end: end, exp: url.Values{"startTime": {"1775100000000"}, "endTime": {"1775200000000"}}},
		{name: "start after end", start: end, end: start, exp: url.Values{}, err: common.ErrStartAfterEnd},
		{name: "start equals end", start: start, end: start, exp: url.Values{}, err: common.ErrStartEqualsEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			err := setAdvancedEarnTimeWindow(params, tc.start, tc.end)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "setAdvancedEarnTimeWindow should reject the window")
			} else {
				assert.NoError(t, err, "setAdvancedEarnTimeWindow should accept the window")
			}
			assert.Equal(t, tc.exp, params, "setAdvancedEarnTimeWindow should set each given bound in milliseconds, and nothing for a rejected window")
		})
	}
}

func TestGetAdvancedEarnRedeemEstimatedAmount(t *testing.T) {
	t.Parallel()
	_, err := e.GetAdvancedEarnRedeemEstimatedAmount(t.Context(), "", []string{"1277"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetAdvancedEarnRedeemEstimatedAmount must require a category")
	_, err = e.GetAdvancedEarnRedeemEstimatedAmount(t.Context(), AdvancedEarnCategorySmartLeverage, nil)
	require.ErrorIs(t, err, errAdvancedEarnPositionIDNotSet, "GetAdvancedEarnRedeemEstimatedAmount must require a position ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		category    string
		positionIDs []string
		exp         *AdvancedEarnRedeemEstimateResponse
	}{
		{category: AdvancedEarnCategorySmartLeverage, positionIDs: []string{"1277", "1260"}, exp: &AdvancedEarnRedeemEstimateResponse{
			List: []AdvancedEarnRedeemEstimate{
				{
					Success:               true,
					PositionID:            "1277",
					EstimatedRedeemAmount: 76.8356,
					EstimatedRedeemTime:   types.Time(time.UnixMilli(1775038305615)),
					SlippageRate:          0.97,
				},
				{
					PositionID:   "1260",
					SlippageRate: 0.97,
				},
			},
		}},
		{category: AdvancedEarnCategoryDoubleWin, positionIDs: []string{"2848"}, exp: &AdvancedEarnRedeemEstimateResponse{
			List: []AdvancedEarnRedeemEstimate{
				{
					Success:               true,
					PositionID:            "2848",
					EstimatedRedeemAmount: 148.21,
					EstimatedRedeemTime:   types.Time(time.Unix(1775107977, 0)),
					SlippageRate:          0.9,
				},
			},
		}},
	} {
		t.Run(tc.category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAdvancedEarnRedeemEstimatedAmount(t.Context(), tc.category, tc.positionIDs)
			require.NoError(t, err, "GetAdvancedEarnRedeemEstimatedAmount must not error")
			require.NotNil(t, result, "GetAdvancedEarnRedeemEstimatedAmount must return a result")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetAdvancedEarnRedeemEstimatedAmount should decode every field")
				return
			}
			assert.Len(t, result.List, len(tc.positionIDs), "GetAdvancedEarnRedeemEstimatedAmount should estimate each position")
		})
	}
}

func TestGetDoubleWinCustomProductQuote(t *testing.T) {
	t.Parallel()
	_, err := e.GetDoubleWinCustomProductQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDoubleWinCustomProductQuote must reject a nil request")
	req := &DoubleWinCustomQuoteRequest{}
	_, err = e.GetDoubleWinCustomProductQuote(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnProductIDNotSet, "GetDoubleWinCustomProductQuote must require a product ID")
	req.ProductID = "14092"
	_, err = e.GetDoubleWinCustomProductQuote(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnPriceNotSet, "GetDoubleWinCustomProductQuote must require an initial price")
	req.InitialPrice = 66333.94
	_, err = e.GetDoubleWinCustomProductQuote(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnPriceNotSet, "GetDoubleWinCustomProductQuote must require a lower price")
	req.LowerPrice = 63000
	_, err = e.GetDoubleWinCustomProductQuote(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnPriceNotSet, "GetDoubleWinCustomProductQuote must require an upper price")
	req.UpperPrice = 70000

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		req = liveDoubleWinCustomQuoteRequest(t)
	}
	result, err := e.GetDoubleWinCustomProductQuote(t.Context(), req)
	require.NoError(t, err, "GetDoubleWinCustomProductQuote must not error")
	require.NotNil(t, result, "GetDoubleWinCustomProductQuote must return a result")
	if mockTests {
		exp := &DoubleWinCustomQuoteResponse{
			ProductID:           "14092",
			InitialPrice:        66333.94,
			LowerPrice:          63000,
			UpperPrice:          70000,
			Leverage:            241.15,
			ExpireTime:          types.Time(time.Unix(1775106748, 0)),
			MaxInvestmentAmount: 10000,
		}
		assert.Equal(t, exp, result, "GetDoubleWinCustomProductQuote should decode every field")
		return
	}
	assert.Positive(t, result.Leverage.Float64(), "GetDoubleWinCustomProductQuote should quote a leverage")
}

// liveDoubleWinCustomQuoteRequest asks for a quote on a listed RFQ Double Win product, around the index price a fixed
// range product's quote reports and within the RFQ product's allowed deviation
func liveDoubleWinCustomQuoteRequest(t *testing.T) *DoubleWinCustomQuoteRequest {
	t.Helper()
	products, err := e.GetAdvancedEarnProductInfo(t.Context(), &AdvancedEarnProductInfoRequest{Category: AdvancedEarnCategoryDoubleWin, Coin: currency.BTC})
	require.NoError(t, err, "GetAdvancedEarnProductInfo must not error")
	var fixed, rfq *DoubleWinProduct
	for i := range products.DoubleWin {
		if products.DoubleWin[i].IsRFQProduct {
			rfq = &products.DoubleWin[i]
		} else {
			fixed = &products.DoubleWin[i]
		}
	}
	require.NotNil(t, fixed, "GetAdvancedEarnProductInfo must return a fixed range DoubleWin product")
	require.NotNil(t, rfq, "GetAdvancedEarnProductInfo must return an RFQ DoubleWin product")
	quote, err := e.GetAdvancedEarnProductQuote(t.Context(), AdvancedEarnCategoryDoubleWin, fixed.ProductID)
	require.NoError(t, err, "GetAdvancedEarnProductQuote must not error")
	require.NotNil(t, quote.DoubleWin, "GetAdvancedEarnProductQuote must return a DoubleWin quote")
	price, tick := quote.DoubleWin.CurrentPrice.Float64(), rfq.PriceTickSize.Float64()
	deviation := (rfq.MinDeviationRatio.Float64() + rfq.MaxDeviationRatio.Float64()) / 2
	return &DoubleWinCustomQuoteRequest{
		ProductID:    rfq.ProductID,
		InitialPrice: price,
		LowerPrice:   math.Floor(price*(1-deviation)/tick) * tick,
		UpperPrice:   math.Ceil(price*(1+deviation)/tick) * tick,
	}
}

func TestGetLiquidityMiningProductInfo(t *testing.T) {
	t.Parallel()
	result, err := e.GetLiquidityMiningProductInfo(t.Context(), currency.BTC, currency.USDT)
	require.NoError(t, err, "GetLiquidityMiningProductInfo must not error")
	require.NotNil(t, result, "GetLiquidityMiningProductInfo must return a result")
	if mockTests {
		exp := &LiquidityMiningProductInfoResponse{
			Products: []LiquidityMiningProduct{
				{
					ProductID:           "1",
					BaseCoin:            currency.BTC,
					QuoteCoin:           currency.USDT,
					Status:              "Available",
					MaxLeverage:         10,
					MinInvestmentQuote:  50,
					MinInvestmentBase:   0.001,
					MaxInvestmentQuote:  150000,
					MaxInvestmentBase:   5,
					MinWithdrawalAmount: 20,
					MinReinvestAmount:   5,
					BaseCoinPrecision:   8,
					QuoteCoinPrecision:  4,
					YieldCoins: []currency.Code{
						currency.USDT,
					},
					APYE8:              1407795,
					APY7DayE8:          1163143,
					PoolLiquidityValue: 26530058.37,
					DailyYield:         1153.2908,
					SlippageLevels: []types.Number{
						0,
						1000,
						20000,
						100000,
						150000,
					},
					SlippageRateE8List: []types.Number{
						0,
						50000,
						120000,
						350000,
						500000,
					},
					APYBreakdown: []LiquidityMiningAPYBreakdown{
						{
							Coin: currency.USDT,
							APY:  0.01407795,
						},
					},
					APY7DayBreakdown: []LiquidityMiningAPYBreakdown{
						{
							Coin: currency.USDT,
							APY:  0.01163143,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetLiquidityMiningProductInfo should decode every field")
		return
	}
	assert.NotEmpty(t, result.Products, "GetLiquidityMiningProductInfo should return products")
}

func TestAddLiquidity(t *testing.T) {
	t.Parallel()
	_, err := e.AddLiquidity(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddLiquidity must reject a nil request")
	req := &AddLiquidityRequest{}
	_, err = e.AddLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnProductIDNotSet, "AddLiquidity must require a product ID")
	req.ProductID = "36"
	_, err = e.AddLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "AddLiquidity must require an order link ID")
	req.OrderLinkID = "lm-001"
	_, err = e.AddLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errLiquidityMiningAmountNotSet, "AddLiquidity must require an amount")
	req.QuoteAmount = 200
	_, err = e.AddLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errMissingAccountType, "AddLiquidity must require the quote amount's account type")
	req.QuoteAccountType = "FUND"
	req.BaseAmount = 0.5
	_, err = e.AddLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errMissingAccountType, "AddLiquidity must require the base amount's account type")
	req.BaseAccountType = "UNIFIED"
	req.Leverage = 2

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddLiquidity(t.Context(), req)
	require.NoError(t, err, "AddLiquidity must not error")
	if mockTests {
		assert.Equal(t, &AdvancedEarnOrderResponse{OrderID: "5e651d09-6169-4f72-a609-8622ff421d19", OrderLinkID: "lm-001"}, result, "AddLiquidity should decode the order IDs")
		return
	}
	assert.NotEmpty(t, result.OrderID, "AddLiquidity should return an order ID")
}

func TestRemoveLiquidity(t *testing.T) {
	t.Parallel()
	_, err := e.RemoveLiquidity(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RemoveLiquidity must reject a nil request")
	req := &RemoveLiquidityRequest{}
	_, err = e.RemoveLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnProductIDNotSet, "RemoveLiquidity must require a product ID")
	req.ProductID = "1001"
	_, err = e.RemoveLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "RemoveLiquidity must require an order link ID")
	req.OrderLinkID = "lm-remove-001-20260312"
	_, err = e.RemoveLiquidity(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnPositionIDNotSet, "RemoveLiquidity must require a position ID")
	req.PositionID = "5001"
	req.RemoveRate = 50
	req.RemoveType = "Normal"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RemoveLiquidity(t.Context(), req)
	require.NoError(t, err, "RemoveLiquidity must not error")
	if mockTests {
		assert.Equal(t, &AdvancedEarnOrderResponse{OrderID: "b2c3d4e5-f6a7-8901-bcde-f12345678901", OrderLinkID: "lm-remove-001-20260312"}, result, "RemoveLiquidity should decode the order IDs")
		return
	}
	assert.NotEmpty(t, result.OrderID, "RemoveLiquidity should return an order ID")
}

func TestReinvestLiquidityMining(t *testing.T) {
	t.Parallel()
	_, err := e.ReinvestLiquidityMining(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ReinvestLiquidityMining must reject a nil request")
	req := &ReinvestLiquidityMiningRequest{}
	_, err = e.ReinvestLiquidityMining(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnProductIDNotSet, "ReinvestLiquidityMining must require a product ID")
	req.ProductID = "5"
	_, err = e.ReinvestLiquidityMining(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "ReinvestLiquidityMining must require an order link ID")
	req.OrderLinkID = "lm-reinvest-01"
	_, err = e.ReinvestLiquidityMining(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnPositionIDNotSet, "ReinvestLiquidityMining must require a position ID")
	req.PositionID = "1498"
	req.Leverage = 2

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ReinvestLiquidityMining(t.Context(), req)
	require.NoError(t, err, "ReinvestLiquidityMining must not error")
	if mockTests {
		assert.Equal(t, &AdvancedEarnOrderResponse{OrderID: "c3d4e5f6-a7b8-9012-cdef-123456789012", OrderLinkID: "lm-reinvest-01"}, result, "ReinvestLiquidityMining should decode the order IDs")
		return
	}
	assert.NotEmpty(t, result.OrderID, "ReinvestLiquidityMining should return an order ID")
}

func TestAddLiquidityMiningMargin(t *testing.T) {
	t.Parallel()
	_, err := e.AddLiquidityMiningMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddLiquidityMiningMargin must reject a nil request")
	req := &AddLiquidityMiningMarginRequest{}
	_, err = e.AddLiquidityMiningMargin(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnProductIDNotSet, "AddLiquidityMiningMargin must require a product ID")
	req.ProductID = "5"
	_, err = e.AddLiquidityMiningMargin(t.Context(), req)
	require.ErrorIs(t, err, errOrderLinkIDMissing, "AddLiquidityMiningMargin must require an order link ID")
	req.OrderLinkID = "lm-add-m-01"
	_, err = e.AddLiquidityMiningMargin(t.Context(), req)
	require.ErrorIs(t, err, errAdvancedEarnPositionIDNotSet, "AddLiquidityMiningMargin must require a position ID")
	req.PositionID = "1498"
	_, err = e.AddLiquidityMiningMargin(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "AddLiquidityMiningMargin must require an amount")
	req.Amount = 1000
	_, err = e.AddLiquidityMiningMargin(t.Context(), req)
	require.ErrorIs(t, err, errMissingAccountType, "AddLiquidityMiningMargin must require an account type")
	req.QuoteAccountType = "FUND"

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddLiquidityMiningMargin(t.Context(), req)
	require.NoError(t, err, "AddLiquidityMiningMargin must not error")
	if mockTests {
		assert.Equal(t, &AdvancedEarnOrderResponse{OrderID: "91720e72-86f8-4ee7-a22c-47f01a9a14e9", OrderLinkID: "lm-add-m-01"}, result, "AddLiquidityMiningMargin should decode the order IDs")
		return
	}
	assert.NotEmpty(t, result.OrderID, "AddLiquidityMiningMargin should return an order ID")
}

func TestLiquidityMiningPositionOrderBody(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                               string
		productID, orderLinkID, positionID string
		err                                error
	}{
		{name: "a product ID", orderLinkID: "lm-reinvest-01", positionID: "1498", err: errAdvancedEarnProductIDNotSet},
		{name: "an order link ID", productID: "5", positionID: "1498", err: errOrderLinkIDMissing},
		{name: "a position ID", productID: "5", orderLinkID: "lm-reinvest-01", err: errAdvancedEarnPositionIDNotSet},
	} {
		body, err := liquidityMiningPositionOrderBody(tc.productID, tc.orderLinkID, tc.positionID)
		assert.ErrorIsf(t, err, tc.err, "liquidityMiningPositionOrderBody should require %s", tc.name)
		assert.Nilf(t, body, "liquidityMiningPositionOrderBody should return no body without %s", tc.name)
	}

	body, err := liquidityMiningPositionOrderBody("5", "lm-reinvest-01", "1498")
	require.NoError(t, err, "liquidityMiningPositionOrderBody must not error")
	exp := map[string]any{"productId": "5", "orderLinkId": "lm-reinvest-01", "positionId": "1498"}
	assert.Equal(t, exp, body, "liquidityMiningPositionOrderBody should start the body with the position's identifiers")
}

func TestClaimLiquidityMiningInterest(t *testing.T) {
	t.Parallel()
	err := e.ClaimLiquidityMiningInterest(t.Context(), "")
	require.ErrorIs(t, err, errAdvancedEarnProductIDNotSet, "ClaimLiquidityMiningInterest must require a product ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.ClaimLiquidityMiningInterest(t.Context(), "-1"), "ClaimLiquidityMiningInterest should not error")
}

func TestGetLiquidityMiningPositionInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLiquidityMiningPositionInfo(t.Context(), "5", currency.ETH)
	require.NoError(t, err, "GetLiquidityMiningPositionInfo must not error")
	require.NotNil(t, result, "GetLiquidityMiningPositionInfo must return a result")
	if mockTests {
		exp := &LiquidityMiningPositionInfoResponse{
			Positions: []LiquidityMiningPosition{
				{
					PositionID:              "1498",
					ProductID:               "5",
					BaseCoin:                currency.ETH,
					QuoteCoin:               currency.USDT,
					QuoteAmount:             1637.7537,
					BaseAmount:              0.80131605,
					PrincipalQuoteAmount:    817.5297,
					PrincipalBaseAmount:     0.39999891,
					PrincipalLiquidityValue: 1635.0595,
					LeveragedValue:          3275.5075,
					Loan:                    1640.448,
					ClaimableYield:          0.0412,
					CurrentAPR:              27856097,
					Leverage:                2.003295552626343,
					Margin:                  25,
					LiquidationPrice:        538.272,
					CurrentPriceY:           2043.83,
					Status:                  "Active",
					CreatedTime:             types.Time(time.Unix(1775116860, 0)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetLiquidityMiningPositionInfo should decode every field")
	}
}

func TestGetLiquidityMiningOrderInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidityMiningOrderInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLiquidityMiningOrderInfo must reject a nil request")
	_, err = e.GetLiquidityMiningOrderInfo(t.Context(), &LiquidityMiningOrderInfoRequest{OrderID: "5e651d09-6169-4f72-a609-8622ff421d19", OrderLinkID: "lm-001"})
	require.ErrorIs(t, err, errLiquidityMiningOrderIDsExclusive, "GetLiquidityMiningOrderInfo must reject both an order ID and an order link ID")
	_, err = e.GetLiquidityMiningOrderInfo(t.Context(), &LiquidityMiningOrderInfoRequest{StartTime: time.UnixMilli(1775200000000), EndTime: time.UnixMilli(1775100000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLiquidityMiningOrderInfo must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *LiquidityMiningOrderInfoRequest
		exp  *LiquidityMiningOrderInfoResponse
	}{
		{name: "order link ID", req: &LiquidityMiningOrderInfoRequest{OrderLinkID: "lm-001"}, exp: &LiquidityMiningOrderInfoResponse{
			Orders: []LiquidityMiningOrder{
				{
					OrderID:       "5e651d09-6169-4f72-a609-8622ff421d19",
					OrderLinkID:   "lm-001",
					ProductID:     "36",
					OrderType:     "AddLiquidity",
					BaseCoin:      currency.NewCode("TON"),
					QuoteCoin:     currency.USDT,
					QuoteAmount:   200,
					Status:        "Success",
					Leverage:      2,
					SlippageValue: 0.02,
					CreatedTime:   types.Time(time.Unix(1775123507, 0)),
				},
			},
		}},
		{name: "order ID", req: &LiquidityMiningOrderInfoRequest{OrderID: "c3d4e5f6-a7b8-9012-cdef-123456789012"}, exp: &LiquidityMiningOrderInfoResponse{
			Orders: []LiquidityMiningOrder{
				{
					OrderID:     "c3d4e5f6-a7b8-9012-cdef-123456789012",
					OrderLinkID: "lm-reinvest-01",
					ProductID:   "5",
					OrderType:   "Reinvest",
					BaseCoin:    currency.ETH,
					QuoteCoin:   currency.USDT,
					QuoteAmount: 0.0412,
					Status:      "Processing",
					CreatedTime: types.Time(time.Unix(1775125900, 0)),
				},
			},
		}},
		{
			name: "filters",
			req: &LiquidityMiningOrderInfoRequest{
				ProductID: "1001", OrderType: "RemoveLiquidity", StartTime: time.UnixMilli(1775100000000), EndTime: time.UnixMilli(1775200000000),
				Status: "Success", Limit: 50, Cursor: "eyJpZCI6NTAwMX0=",
			},
			exp: &LiquidityMiningOrderInfoResponse{
				Orders: []LiquidityMiningOrder{
					{
						OrderID:     "b2c3d4e5-f6a7-8901-bcde-f12345678901",
						OrderLinkID: "lm-remove-001-20260312",
						ProductID:   "1001",
						OrderType:   "RemoveLiquidity",
						BaseCoin:    currency.BTC,
						QuoteCoin:   currency.USDT,
						QuoteAmount: 1580.21,
						BaseAmount:  0.01712,
						Status:      "Success",
						RemoveType:  "Normal",
						RemoveRate:  50,
						CreatedTime: types.Time(time.Unix(1775123600, 0)),
					},
				},
				NextPageCursor: "eyJpZCI6NDk5OX0=",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLiquidityMiningOrderInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetLiquidityMiningOrderInfo must not error")
			require.NotNil(t, result, "GetLiquidityMiningOrderInfo must return a result")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetLiquidityMiningOrderInfo should decode every field")
			}
		})
	}
}

func TestGetLiquidityMiningYieldRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidityMiningYieldRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLiquidityMiningYieldRecords must reject a nil request")
	_, err = e.GetLiquidityMiningYieldRecords(t.Context(), &LiquidityMiningRecordsRequest{StartTime: time.UnixMilli(1775200000000), EndTime: time.UnixMilli(1775100000000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLiquidityMiningYieldRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLiquidityMiningYieldRecords(t.Context(), &LiquidityMiningRecordsRequest{
		BaseCoin: currency.ETH, QuoteCoin: currency.USDT, StartTime: time.UnixMilli(1775100000000), EndTime: time.UnixMilli(1775200000000),
		Limit: 20, Cursor: "eyJpZCI6MTJ9",
	})
	require.NoError(t, err, "GetLiquidityMiningYieldRecords must not error")
	require.NotNil(t, result, "GetLiquidityMiningYieldRecords must return a result")
	if mockTests {
		exp := &LiquidityMiningYieldRecordsResponse{
			Records: []LiquidityMiningYieldRecord{
				{
					Coin:        currency.USDT,
					Amount:      0.0098,
					BaseCoin:    currency.ETH,
					QuoteCoin:   currency.USDT,
					Type:        "Manual",
					Status:      "Complete",
					CreatedTime: types.Time(time.Unix(1775125850, 0)),
				},
			},
			NextPageCursor: "eyJpZCI6MTF9",
		}
		assert.Equal(t, exp, result, "GetLiquidityMiningYieldRecords should decode every field")
	}
}

func TestLiquidityMiningRecordsRequestParams(t *testing.T) {
	t.Parallel()
	_, err := (*LiquidityMiningRecordsRequest)(nil).params()
	require.ErrorIs(t, err, common.ErrNilPointer, "params must reject a nil request")
	_, err = (&LiquidityMiningRecordsRequest{StartTime: time.UnixMilli(1775200000000), EndTime: time.UnixMilli(1775100000000)}).params()
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "params must reject a start time after the end time")

	params, err := (&LiquidityMiningRecordsRequest{}).params()
	require.NoError(t, err, "params must not error")
	assert.Empty(t, params, "params should send no filter for an empty request")

	params, err = (&LiquidityMiningRecordsRequest{
		BaseCoin: currency.NewCode("eth"), QuoteCoin: currency.NewCode("usdt"), StartTime: time.UnixMilli(1775100000000),
		EndTime: time.UnixMilli(1775200000000), Limit: 20, Cursor: "eyJpZCI6MTJ9",
	}).params()
	require.NoError(t, err, "params must not error")
	exp := url.Values{
		"baseCoin":  {"ETH"},
		"quoteCoin": {"USDT"},
		"startTime": {"1775100000000"},
		"endTime":   {"1775200000000"},
		"limit":     {"20"},
		"cursor":    {"eyJpZCI6MTJ9"},
	}
	assert.Equal(t, exp, params, "params should send every filter, with uppercase coins and millisecond times")
}

func TestGetLiquidityMiningLiquidationRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetLiquidityMiningLiquidationRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLiquidityMiningLiquidationRecords must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetLiquidityMiningLiquidationRecords(t.Context(), &LiquidityMiningRecordsRequest{BaseCoin: currency.BTC, Limit: 20})
	require.NoError(t, err, "GetLiquidityMiningLiquidationRecords must not error")
	require.NotNil(t, result, "GetLiquidityMiningLiquidationRecords must return a result")
	if mockTests {
		exp := &LiquidityMiningLiquidationRecordsResponse{
			Records: []LiquidityMiningLiquidationRecord{
				{
					BaseCoin:         currency.BTC,
					QuoteCoin:        currency.USDT,
					BaseAmount:       0.05234,
					QuoteAmount:      1580.21,
					LiquidationPrice: 30200,
					LiquidationTime:  types.Time(time.Unix(1741824000, 0)),
				},
			},
			NextPageCursor: "eyJpZCI6MzN9",
		}
		assert.Equal(t, exp, result, "GetLiquidityMiningLiquidationRecords should decode every field")
	}
}
