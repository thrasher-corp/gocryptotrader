package kucoin

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
)

// GetCrossMarginSymbols returns the configuration of every cross margin symbol, or of the symbol given (Get Symbols -
// Cross Margin)
func (e *Exchange) GetCrossMarginSymbols(ctx context.Context, symbol string) (*CrossMarginSymbolsResponse, error) {
	params := url.Values{}
	if symbol != "" {
		params.Set("symbol", symbol)
	}
	var resp *CrossMarginSymbolsResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getCrossMarginSymbolsEPL, common.EncodeURLValues("/v3/margin/symbols", params), &resp)
}

// GetIsolatedMarginSymbols returns the configuration of every isolated margin symbol (Get Symbols - Isolated Margin)
func (e *Exchange) GetIsolatedMarginSymbols(ctx context.Context) ([]IsolatedMarginSymbolResponse, error) {
	var resp []IsolatedMarginSymbolResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getIsolatedMarginSymbolsEPL, "/v1/isolated/symbols", &resp)
}

// GetMarkPrice returns a margin symbol's current mark price; KuCoin prices margin currencies against BTC, such as
// USDT-BTC (Get Mark Price Detail)
func (e *Exchange) GetMarkPrice(ctx context.Context, symbol string) (*MarginMarkPriceResponse, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	var resp *MarginMarkPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMarkPriceEPL, "/v1/mark-price/"+symbol+"/current", &resp)
}

// GetMarginConfiguration returns the cross margin configuration (Get Margin Config)
func (e *Exchange) GetMarginConfiguration(ctx context.Context) (*MarginConfigurationResponse, error) {
	var resp *MarginConfigurationResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMarginConfigurationEPL, "/v1/margin/config", &resp)
}

// GetAllMarginTradingPairsMarkPrices returns the current mark price of every margin symbol (Get Mark Price List)
func (e *Exchange) GetAllMarginTradingPairsMarkPrices(ctx context.Context) ([]MarginMarkPriceResponse, error) {
	var resp []MarginMarkPriceResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getAllMarginTradingPairsMarkPricesEPL, "/v3/mark-price/all-symbols", &resp)
}

// GetMarginCollateralRatio returns the margin collateral ratio tiers of every currency, or of the currencies given
// (Get Margin Collateral Ratio)
func (e *Exchange) GetMarginCollateralRatio(ctx context.Context, currencies []currency.Code) ([]MarginCollateralRatioResponse, error) {
	params := url.Values{}
	if len(currencies) > 0 {
		list, err := currencyListParam(currencies)
		if err != nil {
			return nil, err
		}
		params.Set("currencyList", list)
	}
	var resp []MarginCollateralRatioResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMarginCollateralRatioEPL, common.EncodeURLValues("/v3/margin/collateralRatio", params), &resp)
}

// GetMarketAvailableInventory returns the amount the platform has available to lend of every currency, or of the
// currency given (Get Market Available Inventory)
func (e *Exchange) GetMarketAvailableInventory(ctx context.Context, ccy currency.Code) ([]MarketAvailableInventoryResponse, error) {
	params := url.Values{}
	if !ccy.IsEmpty() {
		params.Set("currency", ccy.Upper().String())
	}
	var resp []MarketAvailableInventoryResponse
	return resp, e.SendHTTPRequest(ctx, exchange.RestSpot, getMarketAvailableInventoryEPL, common.EncodeURLValues("/v3/margin/available-inventory", params), &resp)
}

// currencyListParam joins currencies into the comma separated list KuCoin's multi-currency filters take
func currencyListParam(currencies []currency.Code) (string, error) {
	list := make([]string, len(currencies))
	for i := range currencies {
		if currencies[i].IsEmpty() {
			return "", fmt.Errorf("%w: currency %d", currency.ErrCurrencyCodeEmpty, i)
		}
		list[i] = currencies[i].Upper().String()
	}
	return strings.Join(list, ","), nil
}
