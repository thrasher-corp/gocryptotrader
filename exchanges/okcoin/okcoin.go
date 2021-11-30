package okcoin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/okgroup"
)

const (
	okCoinRateInterval        = time.Second
	okCoinStandardRequestRate = 6
	okCoinAPIPath             = "api/"
	okCoinAPIURL              = "https://www.okcoin.com/" + okCoinAPIPath
	okCoinAPIVersion          = "/v3/"
	okCoinExchangeName        = "OKCOIN International"
	okCoinWebsocketURL        = "wss://real.okcoin.com:8443/ws/v3"

	okgroupTradeFee = "trade_fee"
)

// OKCoin bases all methods off okgroup implementation
type OKCoin struct {
	okgroup.OKGroup
}

// FeeInfo defines the current trading fees for your associated api keys
type FeeInfo struct {
	Category  int       `json:"category,string"`
	Maker     float64   `json:"maker,string"`
	Taker     float64   `json:"taker,string"`
	Timestamp time.Time `json:"timestamp"`
}

// GetTradingFee returns trading fee based on the asset item and pair, pair can
// be ommited.
func (o *OKCoin) GetTradingFee(ctx context.Context, a asset.Item, instrumentID currency.Pair, category string) (FeeInfo, error) {
	vals := url.Values{}

	var requestType string
	switch a {
	case asset.Spot:
		requestType = "spot"
		if !instrumentID.IsEmpty() {
			vals.Set("instId", instrumentID.String())
		}
	case asset.Margin:
		requestType = "margin"
		if !instrumentID.IsEmpty() {
			vals.Set("instId", instrumentID.String())
		}
	default:
		return FeeInfo{}, fmt.Errorf("%s %w", a, asset.ErrNotSupported)
	}

	if category != "" {
		vals.Set("category", category)
	}

	var resp FeeInfo
	return resp, o.SendHTTPRequest(ctx,
		exchange.RestSpot,
		http.MethodGet,
		okgroup.Version3,
		requestType,
		common.EncodeURLValues(okgroupTradeFee, vals),
		nil,
		&resp,
		true)
}
