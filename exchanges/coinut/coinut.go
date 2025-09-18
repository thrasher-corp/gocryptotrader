package coinut

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	coinutAPIURL          = "https://api.coinut.com"
	tradeBaseURL          = "https://coinut.com/spot/"
	coinutAPIVersion      = "1"
	coinutInstruments     = "inst_list"
	coinutTicker          = "inst_tick"
	coinutOrderbook       = "inst_order_book"
	coinutTrades          = "inst_trade"
	coinutBalance         = "user_balance"
	coinutOrder           = "new_order"
	coinutOrders          = "new_orders"
	coinutOrdersOpen      = "user_open_orders"
	coinutOrderCancel     = "cancel_order"
	coinutOrdersCancel    = "cancel_orders"
	coinutTradeHistory    = "trade_history"
	coinutIndexTicker     = "index_tick"
	coinutOptionChain     = "option_chain"
	coinutPositionHistory = "position_history"
	coinutPositionOpen    = "user_open_positions"

	coinutStatusOK = "OK"
	coinutMaxNonce = 16777215 // See https://github.com/coinut/api/wiki/Websocket-API#nonce

)

var errLookupInstrumentID = errors.New("unable to lookup instrument ID")

// Exchange implements exchange.IBotExchange and contains additional specific api methods for interacting with COINUT
type Exchange struct {
	exchange.Base
	instrumentMap instrumentMap
}

// SeedInstruments seeds the instrument map
func (e *Exchange) SeedInstruments(ctx context.Context) error {
	i, err := e.GetInstruments(ctx)
	if err != nil {
		return err
	}

	for _, y := range i.Instruments {
		e.instrumentMap.Seed(y[0].Base+y[0].Quote, y[0].InstrumentID)
	}
	return nil
}

// GetInstruments returns instruments
func (e *Exchange) GetInstruments(ctx context.Context) (Instruments, error) {
	var result Instruments
	params := make(map[string]any)
	params["sec_type"] = strings.ToUpper(asset.Spot.String())
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutInstruments, params, false, &result)
}

// GetInstrumentTicker returns a ticker for a specific instrument
func (e *Exchange) GetInstrumentTicker(ctx context.Context, instrumentID int64) (Ticker, error) {
	var result Ticker
	params := make(map[string]any)
	params["inst_id"] = instrumentID
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutTicker, params, false, &result)
}

// GetInstrumentOrderbook returns the orderbooks for a specific instrument
func (e *Exchange) GetInstrumentOrderbook(ctx context.Context, instrumentID, limit int64) (*Orderbook, error) {
	var result Orderbook
	params := make(map[string]any)
	params["inst_id"] = instrumentID
	if limit > 0 {
		params["top_n"] = limit
	}

	return &result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutOrderbook, params, false, &result)
}

// GetTrades returns trade information
func (e *Exchange) GetTrades(ctx context.Context, instrumentID int64) (Trades, error) {
	var result Trades
	params := make(map[string]any)
	params["inst_id"] = instrumentID

	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutTrades, params, false, &result)
}

// GetUserBalance returns the full user balance
func (e *Exchange) GetUserBalance(ctx context.Context) (*UserBalance, error) {
	var result *UserBalance
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutBalance, nil, true, &result)
}

// NewOrder places a new order on the exchange
func (e *Exchange) NewOrder(ctx context.Context, instrumentID int64, quantity, price float64, buy bool, orderID uint32) (any, error) {
	var result any
	params := make(map[string]any)
	params["inst_id"] = instrumentID
	if price > 0 {
		params["price"] = strconv.FormatFloat(price, 'f', -1, 64)
	}
	params["qty"] = strconv.FormatFloat(quantity, 'f', -1, 64)
	params["side"] = order.Buy.String()
	if !buy {
		params["side"] = order.Sell.String()
	}
	params["client_ord_id"] = orderID

	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutOrder, params, true, &result)
}

// NewOrders places multiple orders on the exchange
func (e *Exchange) NewOrders(ctx context.Context, orders []Order) ([]OrdersBase, error) {
	var result OrdersResponse
	params := make(map[string]any)
	params["orders"] = orders

	return result.Data, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutOrders, params, true, &result.Data)
}

// GetOpenOrders returns a list of open order and relevant information
func (e *Exchange) GetOpenOrders(ctx context.Context, instrumentID int64) (GetOpenOrdersResponse, error) {
	var result GetOpenOrdersResponse
	params := make(map[string]any)
	params["inst_id"] = instrumentID
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutOrdersOpen, params, true, &result)
}

// CancelExistingOrder cancels a specific order and returns if it was actioned
func (e *Exchange) CancelExistingOrder(ctx context.Context, instrumentID, orderID int64) (bool, error) {
	var result GenericResponse
	params := make(map[string]any)
	type Request struct {
		InstrumentID int64 `json:"inst_id"`
		OrderID      int64 `json:"order_id"`
	}

	entry := Request{
		InstrumentID: instrumentID,
		OrderID:      orderID,
	}

	entries := []Request{entry}
	params["entries"] = entries

	err := e.SendHTTPRequest(ctx, exchange.RestSpot, coinutOrdersCancel, params, true, &result)
	if err != nil {
		return false, err
	}
	return true, nil
}

// CancelOrders cancels multiple orders
func (e *Exchange) CancelOrders(ctx context.Context, orders []CancelOrders) (CancelOrdersResponse, error) {
	var result CancelOrdersResponse
	params := make(map[string]any)
	var entries []CancelOrders
	entries = append(entries, orders...)
	params["entries"] = entries

	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutOrdersCancel, params, true, &result)
}

// GetTradeHistory returns trade history for a specific instrument.
func (e *Exchange) GetTradeHistory(ctx context.Context, instrumentID, start, limit int64) (TradeHistory, error) {
	var result TradeHistory
	params := make(map[string]any)
	params["inst_id"] = instrumentID
	if start >= 0 && start <= 100 {
		params["start"] = start
	}
	if limit >= 0 && start <= 100 {
		params["limit"] = limit
	}

	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutTradeHistory, params, true, &result)
}

// GetIndexTicker returns the index ticker for an asset
func (e *Exchange) GetIndexTicker(ctx context.Context, a string) (IndexTicker, error) {
	var result IndexTicker
	params := make(map[string]any)
	params["asset"] = a
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutIndexTicker, params, false, &result)
}

// GetDerivativeInstruments returns a list of derivative instruments
func (e *Exchange) GetDerivativeInstruments(ctx context.Context, secType string) (any, error) {
	var result any // TODO: Make this a concrete type
	params := make(map[string]any)
	params["sec_type"] = secType

	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutInstruments, params, false, &result)
}

// GetOptionChain returns option chain
func (e *Exchange) GetOptionChain(ctx context.Context, a, secType string) (OptionChainResponse, error) {
	var result OptionChainResponse
	params := make(map[string]any)
	params["asset"] = a
	params["sec_type"] = secType
	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutOptionChain, params, false, &result)
}

// GetPositionHistory returns position history
func (e *Exchange) GetPositionHistory(ctx context.Context, secType string, start, limit int) (PositionHistory, error) {
	var result PositionHistory
	params := make(map[string]any)
	params["sec_type"] = secType
	if start >= 0 {
		params["start"] = start
	}
	if limit >= 0 {
		params["limit"] = limit
	}

	return result, e.SendHTTPRequest(ctx, exchange.RestSpot, coinutPositionHistory, params, true, &result)
}

// GetOpenPositionsForInstrument returns all your current opened positions
func (e *Exchange) GetOpenPositionsForInstrument(ctx context.Context, instrumentID int) ([]OpenPosition, error) {
	type Response struct {
		Positions []OpenPosition `json:"positions"`
	}
	var result Response
	params := make(map[string]any)
	params["inst_id"] = instrumentID

	return result.Positions,
		e.SendHTTPRequest(ctx, exchange.RestSpot, coinutPositionOpen, params, true, &result)
}

// SendHTTPRequest sends either an authenticated or unauthenticated HTTP request
func (e *Exchange) SendHTTPRequest(ctx context.Context, ep exchange.URL, apiRequest string, params map[string]any, authenticated bool, result any) (err error) {
	endpoint, err := e.API.Endpoints.GetURL(ep)
	if err != nil {
		return err
	}

	if params == nil {
		params = make(map[string]any)
	}

	requestType := request.AuthType(request.UnauthenticatedRequest)
	if authenticated {
		requestType = request.AuthenticatedRequest
	}
	var rawMsg json.RawMessage
	err = e.SendPayload(ctx, request.Unset, func() (*request.Item, error) {
		params["nonce"] = getNonce()
		params["request"] = apiRequest

		var payload []byte
		payload, err = json.Marshal(params)
		if err != nil {
			return nil, err
		}

		headers := make(map[string]string)
		if authenticated {
			creds, err := e.GetCredentials(ctx)
			if err != nil {
				return nil, err
			}
			headers["X-USER"] = creds.ClientID
			var hmac []byte
			hmac, err = crypto.GetHMAC(crypto.HashSHA256, payload, []byte(creds.Key))
			if err != nil {
				return nil, err
			}
			headers["X-SIGNATURE"] = hex.EncodeToString(hmac)
		}
		headers["Content-Type"] = "application/json"

		return &request.Item{
			Method:                 http.MethodPost,
			Path:                   endpoint,
			Headers:                headers,
			Body:                   bytes.NewBuffer(payload),
			Result:                 &rawMsg,
			NonceEnabled:           true,
			Verbose:                e.Verbose,
			HTTPDebugging:          e.HTTPDebugging,
			HTTPRecording:          e.HTTPRecording,
			HTTPMockDataSliceLimit: e.HTTPMockDataSliceLimit,
		}, nil
	}, requestType)
	if err != nil {
		return err
	}

	var genResp GenericResponse
	err = json.Unmarshal(rawMsg, &genResp)
	if err != nil {
		return err
	}

	if genResp.Status[0] != coinutStatusOK {
		if authenticated {
			return fmt.Errorf("%w %v", request.ErrAuthRequestFailed, genResp.Status[0])
		}
		return fmt.Errorf("%s SendHTTPRequest error: %s", e.Name, genResp.Status[0])
	}

	return json.Unmarshal(rawMsg, result)
}

// GetFee returns an estimate of fee based on type of transaction
func (e *Exchange) GetFee(feeBuilder *exchange.FeeBuilder) (float64, error) {
	var fee float64
	switch feeBuilder.FeeType {
	case exchange.CryptocurrencyTradeFee:
		fee = e.calculateTradingFee(feeBuilder.Pair.Base,
			feeBuilder.Pair.Quote,
			feeBuilder.PurchasePrice,
			feeBuilder.Amount,
			feeBuilder.IsMaker)
	case exchange.InternationalBankWithdrawalFee:
		fee = getInternationalBankWithdrawalFee(feeBuilder.FiatCurrency,
			feeBuilder.Amount)
	case exchange.InternationalBankDepositFee:
		fee = getInternationalBankDepositFee(feeBuilder.FiatCurrency,
			feeBuilder.Amount)
	case exchange.OfflineTradeFee:
		fee = getOfflineTradeFee(feeBuilder.Pair, feeBuilder.PurchasePrice, feeBuilder.Amount)
	}

	if fee < 0 {
		fee = 0
	}

	return fee, nil
}

// getOfflineTradeFee calculates the worst case-scenario trading fee
func getOfflineTradeFee(c currency.Pair, price, amount float64) float64 {
	if c.IsCryptoFiatPair() {
		return 0.0035 * price * amount
	}
	return 0.002 * price * amount
}

func (e *Exchange) calculateTradingFee(base, quote currency.Code, purchasePrice, amount float64, isMaker bool) float64 {
	var fee float64

	switch {
	case isMaker:
		fee = 0
	case currency.NewPair(base, quote).IsCryptoFiatPair():
		fee = 0.002
	default:
		fee = 0.001
	}

	return fee * amount * purchasePrice
}

func getInternationalBankWithdrawalFee(c currency.Code, amount float64) float64 {
	switch c.Upper() {
	case currency.USD:
		return max(amount*0.001, 10.0)
	case currency.CAD:
		return max(amount*0.005, 2.0)
	case currency.SGD:
		return 2.0
	default:
		return 0 // Handle unknown currencies
	}
}

func getInternationalBankDepositFee(c currency.Code, amount float64) float64 {
	switch c.Upper() {
	case currency.USD:
		return max(amount*0.001, 10.0)
	case currency.CAD:
		return max(amount*0.005, 2.0)
	default:
		return 0
	}
}

// IsLoaded returns whether or not the instrument map has been seeded
func (i *instrumentMap) IsLoaded() bool {
	i.m.Lock()
	isLoaded := i.Loaded
	i.m.Unlock()
	return isLoaded
}

// Seed seeds the instrument map
func (i *instrumentMap) Seed(curr string, id int64) {
	i.m.Lock()
	defer i.m.Unlock()

	if !i.Loaded {
		i.Instruments = make(map[string]int64)
	}

	// check to see if the instrument already exists
	if _, ok := i.Instruments[curr]; ok {
		return
	}

	i.Instruments[curr] = id
	i.Loaded = true
}

// LookupInstrument looks up an instrument based on an id
func (i *instrumentMap) LookupInstrument(id int64) string {
	i.m.Lock()
	defer i.m.Unlock()

	if !i.Loaded {
		return ""
	}

	for k, v := range i.Instruments {
		if v == id {
			return k
		}
	}
	return ""
}

// LookupID looks up an ID based on a string
func (i *instrumentMap) LookupID(curr string) int64 {
	i.m.Lock()
	defer i.m.Unlock()

	if !i.Loaded {
		return 0
	}

	if ic, ok := i.Instruments[curr]; ok {
		return ic
	}
	return 0
}

// GetInstrumentIDs returns a list of IDs
func (i *instrumentMap) GetInstrumentIDs() []int64 {
	i.m.Lock()
	defer i.m.Unlock()

	if !i.Loaded {
		return nil
	}

	instruments := make([]int64, 0, len(i.Instruments))
	for _, x := range i.Instruments {
		instruments = append(instruments, x)
	}
	return instruments
}

func getNonce() int64 {
	return rand.Int63n(coinutMaxNonce-1) + 1 //nolint:gosec // basic number generation required, no need for crypo/rand
}
