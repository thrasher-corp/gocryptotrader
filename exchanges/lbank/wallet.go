package lbank

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
)

// GetSystemStatus returns whether LBank is operating normally or is under maintenance
func (e *Exchange) GetSystemStatus(ctx context.Context) (*SystemStatusResponse, error) {
	var resp *SystemStatusResponse
	if err := e.SendHTTPRequest(ctx, exchange.RestSpot, defaultEPL, http.MethodPost, "/v2/supplement/system_status.do", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAllCoinInformation returns the account's balance of each coin it holds, with the chains each coin can be
// deposited and withdrawn on
func (e *Exchange) GetAllCoinInformation(ctx context.Context) ([]CoinInformationResponse, error) {
	var resp []CoinInformationResponse
	return resp, e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/user_info.do", nil, &resp)
}

// Withdraw withdraws a coin to an address, or transfers it to another LBank account when InternalTransfer is set
func (e *Exchange) Withdraw(ctx context.Context, arg *WithdrawRequest) (*WithdrawResponse, error) {
	if arg == nil {
		return nil, common.ErrNilPointer
	}
	if arg.Address == "" {
		return nil, errAddressRequired
	}
	if arg.Coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if arg.Amount <= 0 {
		return nil, limits.ErrAmountBelowMin
	}
	params := url.Values{}
	params.Set("address", arg.Address)
	params.Set("coin", arg.Coin.Lower().String())
	params.Set("amount", strconv.FormatFloat(arg.Amount, 'f', -1, 64))
	// LBank requires the fee, which must match the chain's withdrawal fee
	params.Set("fee", strconv.FormatFloat(arg.Fee, 'f', -1, 64))
	if arg.NetworkName != "" {
		params.Set("networkName", arg.NetworkName)
	}
	if arg.Memo != "" {
		params.Set("memo", arg.Memo)
	}
	if arg.Mark != "" {
		params.Set("mark", arg.Mark)
	}
	if arg.AddressName != "" {
		params.Set("name", arg.AddressName)
	}
	if arg.WithdrawOrderID != "" {
		params.Set("withdrawOrderId", arg.WithdrawOrderID)
	}
	if arg.InternalTransfer {
		params.Set("type", "1")
	}
	var resp *WithdrawResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/spot/wallet/withdraw.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetDepositHistory returns the account's deposits. LBank defaults to the last 90 days when no time range is given
func (e *Exchange) GetDepositHistory(ctx context.Context, arg *DepositHistoryRequest) (*DepositHistoryResponse, error) {
	params := url.Values{}
	if arg != nil {
		if err := setMillisecondTimeRange(params, arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
		if arg.Status != 0 {
			params.Set("status", strconv.FormatUint(arg.Status, 10))
		}
		if !arg.Coin.IsEmpty() {
			params.Set("coin", arg.Coin.Lower().String())
		}
	}
	var resp *DepositHistoryResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/spot/wallet/deposit_history.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetWithdrawalHistory returns the account's withdrawals. LBank defaults to the last 90 days when no time range is
// given
func (e *Exchange) GetWithdrawalHistory(ctx context.Context, arg *WithdrawalHistoryRequest) (*WithdrawalHistoryResponse, error) {
	params := url.Values{}
	if arg != nil {
		if err := setMillisecondTimeRange(params, arg.StartTime, arg.EndTime); err != nil {
			return nil, err
		}
		if arg.Status != 0 {
			params.Set("status", strconv.FormatUint(arg.Status, 10))
		}
		if !arg.Coin.IsEmpty() {
			params.Set("coin", arg.Coin.Lower().String())
		}
		if arg.WithdrawOrderID != "" {
			params.Set("withdrawOrderId", arg.WithdrawOrderID)
		}
	}
	var resp *WithdrawalHistoryResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/spot/wallet/withdraws.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetCoinDepositAddress returns the account's deposit address for a coin, on the coin's default chain when the network
// name is empty
func (e *Exchange) GetCoinDepositAddress(ctx context.Context, coin currency.Code, networkName string) (*DepositAddressResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	params := url.Values{}
	params.Set("coin", coin.Lower().String())
	if networkName != "" {
		params.Set("networkName", networkName)
	}
	var resp *DepositAddressResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/get_deposit_address.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// AddDepositAddress creates a new deposit address for a coin on a chain. Every successful call creates another address,
// up to LBank's limit, so a retried call can create more than one
func (e *Exchange) AddDepositAddress(ctx context.Context, coin currency.Code, networkName string) (*DepositAddressResponse, error) {
	if coin.IsEmpty() {
		return nil, currency.ErrCurrencyCodeEmpty
	}
	if networkName == "" {
		return nil, errNetworkRequired
	}
	params := url.Values{}
	params.Set("coin", coin.Lower().String())
	params.Set("networkName", networkName)
	var resp *DepositAddressResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/add_deposit_address.do", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// GetAssetDetails returns each coin's deposit and withdrawal status and withdrawal fee, or one coin's when the coin is
// set
func (e *Exchange) GetAssetDetails(ctx context.Context, coin currency.Code) (map[currency.Code]AssetDetailResponse, error) {
	params := url.Values{}
	if !coin.IsEmpty() {
		params.Set("coin", coin.Lower().String())
	}
	var resp map[currency.Code]AssetDetailResponse
	return resp, e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/asset_detail.do", params, &resp)
}

// GetTradingFeeRates returns the account's maker and taker fee rates on each pair, or on one pair when the pair is set
func (e *Exchange) GetTradingFeeRates(ctx context.Context, pair currency.Pair) ([]TradingFeeRateResponse, error) {
	params := url.Values{}
	if !pair.IsEmpty() {
		symbol, err := e.FormatSymbol(pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		params.Set("category", symbol)
	}
	var resp []TradingFeeRateResponse
	return resp, e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/customer_trade_fee.do", params, &resp)
}

// GetAPIRestrictions returns the permissions of the API key the request is signed with
func (e *Exchange) GetAPIRestrictions(ctx context.Context) (*APIRestrictionsResponse, error) {
	var resp *APIRestrictionsResponse
	if err := e.SendAuthHTTPRequest(ctx, defaultEPL, "/v2/supplement/api_Restrictions.do", nil, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// setMillisecondTimeRange sets the startTime and endTime parameters in milliseconds, checking the range when both are
// set
func setMillisecondTimeRange(params url.Values, start, end time.Time) error {
	if !start.IsZero() && !end.IsZero() {
		if err := common.StartEndTimeCheck(start, end); err != nil {
			return err
		}
	}
	if !start.IsZero() {
		params.Set("startTime", strconv.FormatInt(start.UnixMilli(), 10))
	}
	if !end.IsZero() {
		params.Set("endTime", strconv.FormatInt(end.UnixMilli(), 10))
	}
	return nil
}
