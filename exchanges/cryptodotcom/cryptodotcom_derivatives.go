package cryptodotcom

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	restURL            = "https://deriv-api.crypto.com/v1/"
	websocketUserURL   = "wss://deriv-stream.crypto.com/v1/user"
	websocketMarketURL = "wss://deriv-stream.crypto.com/v1/market"
)

// ChangeAccountLeverage changes the maximum leverage used by the account. Please note, each instrument has its own maximum leverage. Whichever leverage (account or instrument) is lower will be used.
func (cr *Cryptodotcom) ChangeAccountLeverage(ctx context.Context, accountID string, leverage int64) error {
	if accountID == "" {
		return errAccountIDMissing
	}
	if leverage <= 0 {
		return order.ErrSubmitLeverageNotSupported
	}
	params := make(map[string]any)
	params["account_id"] = accountID
	params["leverage"] = leverage
	return cr.SendAuthHTTPRequest(ctx, exchange.RestFutures, request.Auth, "private/change-account-leverage", params, nil)
}

// GetAllExecutableTradesForInstrument returns all executable trades for a particular instrument
func (cr *Cryptodotcom) GetAllExecutableTradesForInstrument(ctx context.Context, symbol string, startTime, endTime time.Time, limit int64) (*InstrumentTrades, error) {
	params := make(map[string]any)
	if symbol != "" {
		params["instrument_name"] = symbol
	}
	if !startTime.IsZero() && !endTime.IsZero() {
		err := common.StartEndTimeCheck(startTime, endTime)
		if err != nil {
			return nil, err
		}
		params["start_time"] = startTime.UnixNano()
		params["end_time"] = endTime.UnixNano()
	}
	if limit > 0 {
		params["limit"] = limit
	}
	var resp *InstrumentTrades
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestFutures, request.Auth, "private/get-trades", params, &resp)
}

// ClosePosition cancels position for a particular instrument/pair (asynchronous).
func (cr *Cryptodotcom) ClosePosition(ctx context.Context, symbol, orderType string, price float64) (*OrderIDsDetail, error) {
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	orderType = strings.ToUpper(orderType)
	if !slices.Contains([]string{"LIMIT", "MARKET"}, orderType) {
		return nil, fmt.Errorf("%w: LIMIT or MARKET order types are supported", order.ErrUnsupportedOrderType)
	}
	if orderType == "LIMIT" && price <= 0 {
		return nil, order.ErrPriceBelowMin
	}
	params := make(map[string]any)
	params["instrument_name"] = symbol
	params["type"] = orderType
	params["price"] = price
	var resp *OrderIDsDetail
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestFutures, request.Auth, "private/close-position", params, &resp)
}

// GetFuturesOrderList gets the details of an outstanding (not executed) contingency order on Exchange.
// contingency type possible value OCO
func (cr *Cryptodotcom) GetFuturesOrderList(ctx context.Context, contingencyType, listID, symbol string) (*OrdersDetail, error) {
	if contingencyType == "" {
		return nil, errContingencyTypeRequired
	}
	if listID == "" {
		return nil, fmt.Errorf("%w: contingency order ID is required", order.ErrOrderIDNotSet)
	}
	if symbol == "" {
		return nil, currency.ErrSymbolStringEmpty
	}
	params := make(map[string]any)
	params["contingency_type"] = contingencyType
	params["list_id"] = listID
	params["instrument_name"] = symbol
	var resp *OrdersDetail
	return resp, cr.SendAuthHTTPRequest(ctx, exchange.RestFutures, request.Auth, "private/get-order-list", params, &resp)
}
