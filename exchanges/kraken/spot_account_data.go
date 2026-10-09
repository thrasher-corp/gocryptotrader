package kraken

import (
	"context"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// GetAccountBalance calls Get Account Balance, returning each asset's cash balance, net of pending withdrawals, keyed
// by asset name. Earn balances appear under read only names such as USDT.F. A nil request returns the default
// wallet's balances
func (e *Exchange) GetAccountBalance(ctx context.Context, req *BalanceRequest) (map[string]types.Number, error) {
	if req == nil {
		req = new(BalanceRequest)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	var resp map[string]types.Number
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Balance", query, body, &resp)
}

// GetExtendedBalance calls Get Extended Balance, returning each asset's balance with its credit and held amounts,
// keyed by asset name. A nil request returns the default wallet's balances
func (e *Exchange) GetExtendedBalance(ctx context.Context, req *BalanceRequest) (map[string]ExtendedBalance, error) {
	if req == nil {
		req = new(BalanceRequest)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	var resp map[string]ExtendedBalance
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/BalanceEx", query, body, &resp)
}

// GetCreditLines calls Get Credit Lines, returning the credit lines Kraken extends to some VIP accounts. The null
// result the spec allows is returned as common.ErrNoResponse. A nil request returns the default wallet's credit lines
func (e *Exchange) GetCreditLines(ctx context.Context, req *BalanceRequest) (*CreditLinesResponse, error) {
	if req == nil {
		req = new(BalanceRequest)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	var resp *CreditLinesResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/CreditLines", query, body, &resp)
}

// GetTradeBalance calls Get Trade Balance, returning collateral balances, margin position valuations, equity and
// margin level. A nil request values the default wallet in ZUSD
func (e *Exchange) GetTradeBalance(ctx context.Context, req *TradeBalanceRequest) (*TradeBalanceResponse, error) {
	if req == nil {
		req = new(TradeBalanceRequest)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	if !req.Asset.IsEmpty() {
		body["asset"] = req.Asset.Upper().String()
	}
	var resp *TradeBalanceResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/TradeBalance", query, body, &resp)
}

// GetOpenOrders calls Get Open Orders, returning every open order unless Limit is set, or a page of them with
// WithCursor. A nil request returns the default wallet's open orders
func (e *Exchange) GetOpenOrders(ctx context.Context, req *OpenOrdersRequest) (*OpenOrdersResponse, error) {
	if req == nil {
		req = new(OpenOrdersRequest)
	}
	if req.WithCursor && req.Limit > 100 {
		return nil, fmt.Errorf("%w: %d exceeds 100 with cursor pagination", errInvalidLimit, req.Limit)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	if err := accountDataPageParams(body, req.WithCursor, req.Cursor, 0); err != nil {
		return nil, err
	}
	accountDataOrderParams(body, req.IncludeTrades, req.UserReference, req.ClientOrderID, req.WithoutTakerConsolidation)
	if req.Limit != 0 {
		body["limit"] = req.Limit
	}
	var resp *OpenOrdersResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/OpenOrders", query, body, &resp)
}

// GetClosedOrders calls Get Closed Orders, returning a page of up to 50 filled or cancelled orders, the most recent
// first. A nil request returns the default wallet's most recent page
func (e *Exchange) GetClosedOrders(ctx context.Context, req *ClosedOrdersRequest) (*ClosedOrdersResponse, error) {
	if req == nil {
		req = new(ClosedOrdersRequest)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	if err := accountDataRangeParams(body, req.Start, req.End, req.StartOrderID, req.EndOrderID); err != nil {
		return nil, err
	}
	if err := accountDataPageParams(body, req.WithCursor, req.Cursor, req.Offset); err != nil {
		return nil, err
	}
	accountDataOrderParams(body, req.IncludeTrades, req.UserReference, req.ClientOrderID, req.WithoutTakerConsolidation)
	if req.TimeFilter != "" {
		body["closetime"] = req.TimeFilter
	}
	if req.WithoutCount {
		body["without_count"] = true
	}
	var resp *ClosedOrdersResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/ClosedOrders", query, body, &resp)
}

// QueryOrdersInfo calls Query Orders Info, returning up to 50 open or closed orders keyed by order ID
func (e *Exchange) QueryOrdersInfo(ctx context.Context, req *QueryOrdersRequest) (map[string]OrderInfo, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	orderIDs, err := accountDataIDList(req.OrderIDs, 50, order.ErrOrderIDNotSet)
	if err != nil {
		return nil, err
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	body["txid"] = orderIDs
	accountDataOrderParams(body, req.IncludeTrades, req.UserReference, req.ClientOrderID, req.WithoutTakerConsolidation)
	var resp map[string]OrderInfo
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/QueryOrders", query, body, &resp)
}

// GetOrderAmends calls Get Order Amends, returning an order's amends in ascending time order, the order as entered
// first
func (e *Exchange) GetOrderAmends(ctx context.Context, req *OrderAmendsRequest) (*OrderAmendsResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.OrderID == "" {
		return nil, order.ErrOrderIDNotSet
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	body["order_id"] = req.OrderID
	var resp *OrderAmendsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/OrderAmends", query, body, &resp)
}

// GetTradesHistory calls Get Trades History, returning a page of trades, the most recent first. A nil request returns
// the default wallet's most recent page
func (e *Exchange) GetTradesHistory(ctx context.Context, req *TradesHistoryRequest) (*TradesHistoryResponse, error) {
	if req == nil {
		req = new(TradesHistoryRequest)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	if err := accountDataRangeParams(body, req.Start, req.End, req.StartTradeID, req.EndTradeID); err != nil {
		return nil, err
	}
	if err := accountDataPageParams(body, req.WithCursor, req.Cursor, req.Offset); err != nil {
		return nil, err
	}
	if !req.Pair.IsEmpty() {
		pair, err := e.FormatSymbol(req.Pair, asset.Spot)
		if err != nil {
			return nil, err
		}
		body["pair"] = pair
	}
	if req.AssetClass != "" {
		body["aclass"] = req.AssetClass
	}
	if req.TradeType != "" {
		body["type"] = req.TradeType
	}
	if req.IncludeTrades {
		body["trades"] = true
	}
	if req.WithoutCount {
		body["without_count"] = true
	}
	if req.WithoutTakerConsolidation {
		body["consolidate_taker"] = false
	}
	if req.IncludeLedgers {
		body["ledgers"] = true
	}
	if req.Limit != 0 {
		body["limit"] = req.Limit
	}
	var resp *TradesHistoryResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/TradesHistory", query, body, &resp)
}

// QueryTradesInfo calls Query Trades Info, returning up to 20 trades keyed by trade ID
func (e *Exchange) QueryTradesInfo(ctx context.Context, req *QueryTradesRequest) (map[string]TradeInfo, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	tradeIDs, err := accountDataIDList(req.TradeIDs, 20, errTradeIDEmpty)
	if err != nil {
		return nil, err
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	body["txid"] = tradeIDs
	if req.IncludeTrades {
		body["trades"] = true
	}
	if req.IncludeLedgers {
		body["ledgers"] = true
	}
	var resp map[string]TradeInfo
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/QueryTrades", query, body, &resp)
}

// GetOpenPositions calls Get Open Positions, returning open margin positions keyed by the ID of the trade that opened
// each. A nil request returns the default wallet's positions
func (e *Exchange) GetOpenPositions(ctx context.Context, req *OpenPositionsRequest) (map[string]OpenPosition, error) {
	query, body, err := accountDataPositionParams(req)
	if err != nil {
		return nil, err
	}
	var resp map[string]OpenPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/OpenPositions", query, body, &resp)
}

// GetConsolidatedOpenPositions calls Get Open Positions consolidated by market, returning open margin positions
// combined by pair. A nil request returns the default wallet's positions
func (e *Exchange) GetConsolidatedOpenPositions(ctx context.Context, req *OpenPositionsRequest) ([]ConsolidatedPosition, error) {
	query, body, err := accountDataPositionParams(req)
	if err != nil {
		return nil, err
	}
	body["consolidation"] = "market"
	var resp []ConsolidatedPosition
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/OpenPositions", query, body, &resp)
}

// GetLedgers calls Get Ledgers Info, returning a page of up to 50 ledger entries, the most recent first. A nil request
// returns the default wallet's most recent page
func (e *Exchange) GetLedgers(ctx context.Context, req *LedgersRequest) (*LedgersResponse, error) {
	if req == nil {
		req = new(LedgersRequest)
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	switch {
	case len(req.Assets) != 0 && len(req.ClassifiedAssets) != 0:
		return nil, errFilterFormConflict
	case len(req.Assets) != 0:
		codes := make([]string, len(req.Assets))
		for i := range req.Assets {
			if req.Assets[i].IsEmpty() {
				return nil, currency.ErrCurrencyCodeEmpty
			}
			codes[i] = req.Assets[i].Upper().String()
		}
		body["asset"] = strings.Join(codes, ",")
	case len(req.ClassifiedAssets) != 0:
		if err := accountDataCheckClassified(req.ClassifiedAssets, currency.ErrCurrencyCodeEmpty); err != nil {
			return nil, err
		}
		body["asset"] = req.ClassifiedAssets
	}
	if err := accountDataRangeParams(body, req.Start, req.End, req.StartLedgerID, req.EndLedgerID); err != nil {
		return nil, err
	}
	if err := accountDataPageParams(body, req.WithCursor, req.Cursor, req.Offset); err != nil {
		return nil, err
	}
	if req.AssetClass != "" {
		body["aclass"] = req.AssetClass
	}
	if req.LedgerType != "" {
		body["type"] = req.LedgerType
	}
	if req.WithoutCount {
		body["without_count"] = true
	}
	var resp *LedgersResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/Ledgers", query, body, &resp)
}

// QueryLedgers calls Query Ledgers, returning up to 20 ledger entries keyed by ledger ID
func (e *Exchange) QueryLedgers(ctx context.Context, req *QueryLedgersRequest) (map[string]LedgerEntry, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	ledgerIDs, err := accountDataIDList(req.LedgerIDs, 20, errLedgerIDEmpty)
	if err != nil {
		return nil, err
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	body["id"] = ledgerIDs
	if req.IncludeTrades {
		body["trades"] = true
	}
	var resp map[string]LedgerEntry
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/QueryLedgers", query, body, &resp)
}

// GetTradeVolume calls Get Trade Volume, returning the account's 30 day trading volume, with fees only for the pairs
// requested. A nil request returns the volume alone
func (e *Exchange) GetTradeVolume(ctx context.Context, req *TradeVolumeRequest) (*TradeVolumeResponse, error) {
	if req == nil {
		req = new(TradeVolumeRequest)
	}
	body := map[string]any{}
	if req.RebaseMultiplier != "" {
		body["rebase_multiplier"] = req.RebaseMultiplier
	}
	switch {
	case len(req.Pairs) != 0 && len(req.ClassifiedPairs) != 0:
		return nil, errFilterFormConflict
	case len(req.Pairs) != 0:
		pairs, err := e.formatSpotPairs(req.Pairs)
		if err != nil {
			return nil, err
		}
		body["pair"] = pairs
	case len(req.ClassifiedPairs) != 0:
		if err := accountDataCheckClassified(req.ClassifiedPairs, currency.ErrCurrencyPairEmpty); err != nil {
			return nil, err
		}
		body["pair"] = req.ClassifiedPairs
	}
	if req.FeeInfo {
		body["fee-info"] = true
	}
	if req.FeeSchedule {
		body["fee_schedule"] = true
	}
	var resp *TradeVolumeResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/TradeVolume", nil, body, &resp)
}

// RequestExportReport calls Request Export Report, queueing an export of trades or ledgers whose progress
// GetExportReportStatus reports
func (e *Exchange) RequestExportReport(ctx context.Context, req *ExportReportRequest) (*ExportReportResponse, error) {
	if err := common.NilGuard(req); err != nil {
		return nil, err
	}
	if req.Report == "" {
		return nil, errExportReportTypeEmpty
	}
	if req.Description == "" {
		return nil, errExportDescriptionEmpty
	}
	if !req.Start.IsZero() && !req.End.IsZero() && req.Start.After(req.End) {
		return nil, common.ErrStartAfterEnd
	}
	body := map[string]any{
		"report":      req.Report,
		"description": req.Description,
	}
	if req.Format != "" {
		body["format"] = req.Format
	}
	if len(req.Fields) != 0 {
		body["fields"] = strings.Join(req.Fields, ",")
	}
	if !req.Start.IsZero() {
		body["starttm"] = req.Start.Unix()
	}
	if !req.End.IsZero() {
		body["endtm"] = req.End.Unix()
	}
	var resp *ExportReportResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/AddExport", nil, body, &resp)
}

// GetExportReportStatus calls Get Export Report Status, returning the status of every export of a report type, trades
// or ledgers
func (e *Exchange) GetExportReportStatus(ctx context.Context, report string) ([]ExportReport, error) {
	if report == "" {
		return nil, errExportReportTypeEmpty
	}
	var resp []ExportReport
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/ExportStatus", nil, map[string]any{"report": report}, &resp)
}

// RetrieveDataExport calls Retrieve Data Export, returning a processed report as the zip archive Kraken serves
func (e *Exchange) RetrieveDataExport(ctx context.Context, id string) ([]byte, error) {
	if id == "" {
		return nil, errExportIDEmpty
	}
	var archive []byte
	sendErr := e.sendSpotPrivateRequest(ctx, "/0/private/RetrieveExport", nil, map[string]any{"id": id}, &archive)
	// Kraken reports a failure in its JSON envelope rather than serving an archive
	if sendErr != nil || (len(archive) != 0 && archive[0] == '{') {
		if err := e.decodeSpotResponse(archive, sendErr, nil); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %s", errExportNotArchive, archive)
	}
	if len(archive) == 0 {
		return nil, common.ErrNoResponse
	}
	return archive, nil
}

// DeleteExportReport calls Delete Export Report, removing an export report. removalType is delete for a processed
// report, or cancel for a queued or processing one
func (e *Exchange) DeleteExportReport(ctx context.Context, id, removalType string) (*DeleteExportReportResponse, error) {
	if id == "" {
		return nil, errExportIDEmpty
	}
	if removalType == "" {
		return nil, errExportRemovalTypeEmpty
	}
	var resp *DeleteExportReportResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/RemoveExport", nil, map[string]any{"id": id, "type": removalType}, &resp)
}

// GetAPIKeyInfo calls Get API Key Info, returning the name, permissions, restrictions and usage of the API key that
// signs the request. A nil request suits a key without two-factor authentication
func (e *Exchange) GetAPIKeyInfo(ctx context.Context, req *APIKeyInfoRequest) (*APIKeyInfoResponse, error) {
	if req == nil {
		req = new(APIKeyInfoRequest)
	}
	body := map[string]any{}
	if req.OneTimePassword != "" {
		body["otp"] = req.OneTimePassword
	}
	var resp *APIKeyInfoResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/GetApiKeyInfo", nil, body, &resp)
}

// ListWalletAccounts calls List Wallet Accounts, returning the wallets whose account IDs select them on the account
// data endpoints and Funding (Beta)
func (e *Exchange) ListWalletAccounts(ctx context.Context) (*WalletAccountsResponse, error) {
	var resp *WalletAccountsResponse
	return resp, e.SendAuthenticatedHTTPRequest(ctx, "/0/private/ListWalletAccounts", nil, nil, &resp)
}

// accountDataParams returns the account_id query parameter and a body holding rebase_multiplier, which most account
// data endpoints take
func accountDataParams(accountID, rebaseMultiplier string) (query url.Values, body map[string]any) {
	query = url.Values{}
	if accountID != "" {
		query.Set("account_id", accountID)
	}
	body = map[string]any{}
	if rebaseMultiplier != "" {
		body["rebase_multiplier"] = rebaseMultiplier
	}
	return query, body
}

// accountDataPageParams adds the pagination parameters, rejecting a cursor without cursor pagination and the
// deprecated offset alongside it
func accountDataPageParams(body map[string]any, withCursor bool, cursor string, offset uint64) error {
	if cursor != "" && !withCursor {
		return errCursorWithoutPaging
	}
	if offset != 0 && withCursor {
		return errOffsetWithCursor
	}
	if withCursor {
		body["with_cursor"] = true
	}
	if cursor != "" {
		body["cursor"] = cursor
	}
	if offset != 0 {
		body["ofs"] = offset
	}
	return nil
}

// accountDataRangeParams adds the start and end of a history request, each a unix time or the ID of an order, trade
// or ledger entry
func accountDataRangeParams(body map[string]any, start, end time.Time, startID, endID string) error {
	if !start.IsZero() && startID != "" {
		return fmt.Errorf("%w: start", errTimeAndIDBound)
	}
	if !end.IsZero() && endID != "" {
		return fmt.Errorf("%w: end", errTimeAndIDBound)
	}
	if !start.IsZero() && !end.IsZero() && start.After(end) {
		return common.ErrStartAfterEnd
	}
	switch {
	case !start.IsZero():
		body["start"] = start.Unix()
	case startID != "":
		body["start"] = startID
	}
	switch {
	case !end.IsZero():
		body["end"] = end.Unix()
	case endID != "":
		body["end"] = endID
	}
	return nil
}

// accountDataOrderParams adds the parameters Get Open Orders, Get Closed Orders and Query Orders Info share
func accountDataOrderParams(body map[string]any, includeTrades bool, userReference int32, clientOrderID string, withoutTakerConsolidation bool) {
	if includeTrades {
		body["trades"] = true
	}
	if userReference != 0 {
		body["userref"] = userReference
	}
	if clientOrderID != "" {
		body["cl_ord_id"] = clientOrderID
	}
	if withoutTakerConsolidation {
		body["consolidate_taker"] = false
	}
}

// accountDataIDList returns ids as the comma separated list Kraken takes, rejecting an empty list or ID with errEmpty,
// and more than limit IDs
func accountDataIDList(ids []string, limit int, errEmpty error) (string, error) {
	if len(ids) == 0 || slices.Contains(ids, "") {
		return "", errEmpty
	}
	if len(ids) > limit {
		return "", fmt.Errorf("%w: %d exceeds %d", errTooManyIDs, len(ids), limit)
	}
	return strings.Join(ids, ","), nil
}

// accountDataCheckClassified rejects an entry without a name, with errEmpty, or without an asset class
func accountDataCheckClassified(names []ClassifiedAsset, errEmpty error) error {
	for i := range names {
		if names[i].Name == "" {
			return errEmpty
		}
		if names[i].AssetClass == "" {
			return fmt.Errorf("%w: %s", errAssetClassEmpty, names[i].Name)
		}
	}
	return nil
}

// accountDataPositionParams returns the parameters both forms of Get Open Positions take
func accountDataPositionParams(req *OpenPositionsRequest) (url.Values, map[string]any, error) {
	if req == nil {
		req = new(OpenPositionsRequest)
	}
	if slices.Contains(req.TradeIDs, "") {
		return nil, nil, errTradeIDEmpty
	}
	query, body := accountDataParams(req.AccountID, req.RebaseMultiplier)
	if len(req.TradeIDs) != 0 {
		body["txid"] = strings.Join(req.TradeIDs, ",")
	}
	if req.IncludeCalculations {
		body["docalcs"] = true
	}
	return query, body, nil
}
