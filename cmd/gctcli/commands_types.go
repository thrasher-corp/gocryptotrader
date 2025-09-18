package main

// SubmitOrderParams defines command-line flags for exchange order submission and unmarshal their values.
type SubmitOrderParams struct {
	ExchangeName       string  `name:"exchange"              required:"t"`
	CurrencyPair       string  `name:"pair"                  required:"t"                                               usage:"the currency pair"`
	OrderSide          string  `name:"side"                  required:"t"                                               usage:"the order side to use (BUY OR SELL)"`
	OrderType          string  `name:"type"                  required:"t"                                               usage:"the order type (MARKET OR LIMIT)"`
	Amount             float64 `name:"amount"                required:"t"`
	AssetType          string  `name:"asset"                 required:"t"`
	Price              float64 `name:"price"`
	Leverage           float64 `name:"leverage"`
	ClientOrderID      string  `name:"client_order_id"`
	MarginType         string  `name:"margin_type"`
	TimeInForce        string  `name:"time_in_force"`
	QuoteAmount        float64 `name:"quote_amount"`
	ClientID           string  `name:"client_id"`
	TriggerPrice       float64 `name:"trigger_price"`
	TriggerLimitPrice  float64 `name:"trigger_limit_price"`
	TriggerPriceType   string  `name:"trigger_price_type"`
	TpPrice            float64 `name:"tp_price"              usage:"the optional take-profit price for the order"`
	TpLimitPrice       float64 `name:"tp_limit_price"        usage:"the optional take-profit limit price for the order"`
	TpPriceType        string  `name:"tp_price_type"         usage:"the optional take-profit price type for the order"`
	SlPrice            float64 `name:"sl_price"              usage:"the optional stop-loss price for the order"`
	SlLimitPrice       float64 `name:"sl_limit_price"        usage:"the optional stop-loss limit price for the order"`
	SlPriceType        string  `name:"sl_price_type"         usage:"the optional stop-loss price type for the order"`
	TrackingMode       string  `name:"tracking_mode"`
	TrackingValue      float64 `name:"tracking_value"`
	Hidden             bool    `name:"hidden"`
	Iceberg            bool    `name:"iceberg"`
	AutoBorrow         bool    `name:"auto_borrow"`
	ReduceOnly         bool    `name:"reduce_only"`
	RetrieveFees       bool    `name:"retrieve_fees"`
	RetrieveFeeDelayMs int64   `name:"retrieve_fee_delay_ms"`
}

// GetOrderParams defines command-line flags for exchange order retrieval and unmarshal their values.
type GetOrderParams struct {
	Exchange     string `name:"exchange"`
	Asset        string `name:"asset"    required:"t"`
	CurrencyPair string `name:"pair"`
	OrderID      string `name:"order_id"`
}

// ModifyOrderParams defines command-line flags for exchange order modification and unmarshal their values.
type ModifyOrderParams struct {
	ExchangeName      string  `name:"exchange"            required:"t"`
	AssetType         string  `name:"asset"               required:"t"`
	CurrencyPair      string  `name:"pair"                required:"t"                                                              usage:"the currency pair"`
	OrderID           string  `name:"order_id"`
	OrderType         string  `name:"type"                usage:"the order type (MARKET OR LIMIT)"`
	OrderSide         string  `name:"side"                usage:"the order side of the order to be modified"`
	Price             float64 `name:"price"`
	Amount            float64 `name:"amount"`
	ClientOrderID     string  `name:"client_order_id"`
	TimeInForce       string  `name:"time_in_force"`
	TriggerPrice      float64 `name:"trigger_price"`
	TriggerLimitPrice float64 `name:"trigger_limit_price"`
	TriggerPriceType  string  `name:"trigger_price_type"`
	TpPrice           float64 `name:"tp_price"            usage:"the optional take-profit price for the order to be modified"`
	TpLimitPrice      float64 `name:"tp_limit_price"      usage:"the optional take-profit limit price for the order to be modified"`
	TpPriceType       string  `name:"tp_price_type"       usage:"the optional take-profit price type for the order to be modified"`
	SlPrice           float64 `name:"sl_price"            usage:"the optional stop-loss price for the order to be modified"`
	SlLimitPrice      float64 `name:"sl_limit_price"      usage:"the optional stop-loss limit price for the order to be modified"`
	SlPriceType       string  `name:"sl_price_type"       usage:"the optional stop-loss price type for the order to be modified"`
}

// CancelOrderParams defines command-line flags for exchange order cancellation and unmarshal their values.
type CancelOrderParams struct {
	Exchange      string `name:"exchange"        required:"t"                                      usage:"the exchange to cancel the order for"`
	OrderID       string `name:"order_id"        required:"t"`
	ClientOrderID string `name:"client_order_id"`
	AccountID     string `name:"account_id"`
	ClientID      string `name:"client_id"`
	OrderType     string `name:"type"            usage:"the order type (MARKET OR LIMIT)"`
	OrderSide     string `name:"side"`
	AssetType     string `name:"asset"`
	CurrencyPair  string `name:"pair"            usage:"the currency pair to cancel the order for"`
	MarginType    string `name:"margin_type"`
	TimeInForce   string `name:"time_in_force"`
}

// WithdrawCryptoCurrencyFundParams defines command-line flags for exchange cryptocurrency withdrawal and unmarshal their values.
type WithdrawCryptoCurrencyFundParams struct {
	Exchange     string  `name:"exchange"    usage:"the exchange to withdraw from"`
	CurrencyPair string  `name:"currency"    usage:"the cryptocurrency to withdraw funds from"`
	Amount       float64 `name:"amount"      usage:"amount of funds to withdraw"`
	Address      string  `name:"address"     usage:"address to withdraw to"`
	AddressTag   string  `name:"addresstag"  usage:"address tag/memo"`
	Fee          float64 `name:"fee"`
	Description  string  `name:"description"`
	Chain        string  `name:"chain"       usage:"chain to use for the withdrawal"`
}

// GetAvailableTransferChainsParams defines command-line flags for exchange currency available transfer chains retrieval and unmarshal their values.
type GetAvailableTransferChainsParams struct {
	Exchange string `name:"exchange"`
	Currency string `name:"cryptocurrency"`
}

// GetCryptoCurrencyDepositAddressCommandParams defines command-line flags for exchange currency deposit address retrieval and unmarshal their values.
type GetCryptoCurrencyDepositAddressCommandParams struct {
	Exchange string `name:"exchange,e"       required:"t"                                                           usage:"the exchange to get the cryptocurrency deposit address for"`
	Currency string `name:"cryptocurrency,c" required:"t"                                                           usage:"the cryptocurrency to get the deposit address for"`
	Chain    string `name:"chain"            usage:"the chain to use for the deposit"`
	Bypass   bool   `name:"bypass"           usage:"whether to bypass the deposit address manager cache if enabled"`
}

// WithdrawFiatFundParams defines command-line flags for exchange fiat fund withdrawal and unmarshal their values.
type WithdrawFiatFundParams struct {
	Exchange      string  `name:"exchange,e"    required:"t"                               usage:"the exchange to withdraw from"`
	Currency      string  `name:"currency"      required:"t"                               usage:"the fiat currency to withdraw funds from"`
	Amount        float64 `name:"amount"        usage:"amount of funds to withdraw"`
	BankAccountID string  `name:"bankaccountid" usage:"ID of bank account to use"`
	Description   string  `name:"description"   usage:"description to submit with request"`
}

// AddEventParams defines command-line flags for exchange event adding and unmarshal their values.
type AddEventParams struct {
	ExchangeName    string  `name:"exchange"         required:"t"                                      usage:"the exchange to add an event for"`
	Item            string  `name:"item"             usage:"the item to trigger the event"`
	Condition       string  `name:"condition"        required:"t"                                      usage:"the condition for the event"`
	Price           float64 `name:"price"            usage:"the price to trigger the event"`
	CheckBids       bool    `name:"check_bids"       usage:"whether to check the bids"`
	CheckAsks       bool    `name:"check_asks"       usage:"whether to check the asks"`
	OrderbookAmount float64 `name:"orderbook_amount" usage:"the orderbook amount to trigger the event"`
	CurrencyPair    string  `name:"pair"             required:"t"`
	AssetType       string  `name:"asset"`
	Action          string  `name:"action"           required:"t"                                      usage:"the action for the event to perform upon trigger"`
}

// GetTickerStreamParams defines command-line flags for exchange tickers stream retrieval and unmarshal their values
type GetTickerStreamParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to get the ticker from"`
	Pair     string `name:"pair,p"     required:"t" usage:"currency pair"`
	Asset    string `name:"asset"      required:"t" usage:"the asset type of the currency pair"`
}

// GetAuditEventParam defines command-line flags for exchange event audit retrieval and unmarshal their values.
type GetAuditEventParam struct {
	Start string `name:"start,s" required:"t" usage:"start date to search"`
	End   string `name:"end,e"   required:"t" usage:"end time to search"`
	Order string `name:"order"   required:"t" usage:"order results by ascending/descending"`
	Limit int64  `name:"limit"   required:"t" usage:"how many results to retrieve"`
}

// HistoricCandlesParams defines command-line flags for exchange historical candles retrieval and unmarshal their values.
type HistoricCandlesParams struct {
	Exchange                  string `name:"exchange,e"                     usage:"the exchange to get the candles from"`
	CurrencyPair              string `name:"pair"                           required:"t"                                                                                                                                                                                                                                                                     usage:"the currency pair to get the candles for"`
	Asset                     string `name:"asset"                          required:"t"                                                                                                                                                                                                                                                                     usage:"the asset type of the currency pair"`
	RangeSize                 int64  `name:"rangesize,r"                    usage:"the amount of time to go back from now to fetch candles in the given granularity"`
	Granularity               int64  `name:"granularity,g"                  usage:"interval in seconds. supported values are: 15, 60(1min), 180(3min), 300(5min), 600(10min),900(15min) 1800(30min), 3600(1h), 7200(2h), 14400(4h), 21600(6h), 28800(8h), 43200(12h),86400(1d), 259200(3d) 604800(1w), 1209600(2w), 1296000(15d), 2592000(1M), 31536000(1Y)"`
	FillMissingDataWithTrades bool   `name:"fillmissingdatawithtrades,fill" usage:"will create candles for missing intervals using stored trade data <true/false>"`
}

// GetTickerParams defines command-line flags for exchange price tickers retrieval and unmarshal their values.
type GetTickerParams struct {
	Exchange string `name:"exchange,e" usage:"the exchange to get the ticker for"`
	Pair     string `name:"pair"       usage:"the currency pair to get the ticker for"`
	Asset    string `name:"asset"      usage:"the asset type of the currency pair to get the ticker for"`
}

// GetHistoricCandlesParams defines command-line flags for exchange historic candles retrieval and unmarshal their values.
type GetHistoricCandlesParams struct {
	Exchange                  string `name:"exchange,e"                     required:"t"                                                                                                                                                                                                                                                                     usage:"the exchange to get the candles from"`
	Pair                      string `name:"pair,p"                         required:"t"                                                                                                                                                                                                                                                                     usage:"the currency pair to get the candles for"`
	Asset                     string `name:"asset,a"                        usage:"the asset type of the currency pair"`
	Interval                  int64  `name:"interval,i"                     usage:"interval in seconds. supported values are: 15, 60(1min), 180(3min), 300(5min), 600(10min),900(15min) 1800(30min), 3600(1h), 7200(2h), 14400(4h), 21600(6h), 28800(8h), 43200(12h),86400(1d), 259200(3d) 604800(1w), 1209600(2w), 1296000(15d), 2592000(1M), 31536000(1Y)"`
	Start                     string `name:"start"                          usage:"the date to begin retrieving candles. Any candles before this date will be filtered"`
	End                       string `name:"end"                            usage:"the date to end retrieving candles. Any candles after this date will be filtered"`
	Sync                      bool   `name:"sync"                           usage:"<true/false>"`
	Force                     bool   `name:"force"                          usage:"will overwrite any conflicting candle data on save <true/false>"`
	Database                  bool   `name:"db"                             usage:"source data from database <true/false>"`
	FillMissingDataWithTrades bool   `name:"fillmissingdatawithtrades,fill" usage:"will create candles for missing intervals using stored trade data <true/false>"`
}

// FindMissingSavedCandleIntervalsParams defines command-line flags for finding exchange missing saved candle intervals and unmarshal their values.
type FindMissingSavedCandleIntervalsParams struct {
	Exchange string `name:"exchange,e" required:"t"                                                                                                                                                                                                                                                                     usage:"the exchange to get the candles from"`
	Pair     string `name:"pair,p"     required:"t"                                                                                                                                                                                                                                                                     usage:"the currency pair"`
	Asset    string `name:"asset,a"    usage:"the asset type of the currency pair"`
	Interval int64  `name:"interval,i" usage:"interval in seconds. supported values are: 15, 60(1min), 180(3min), 300(5min), 600(10min),900(15min) 1800(30min), 3600(1h), 7200(2h), 14400(4h), 21600(6h), 28800(8h), 43200(12h),86400(1d), 259200(3d) 604800(1w), 1209600(2w), 1296000(15d), 2592000(1M), 31536000(1Y)"`
	Start    string `name:"start"      usage:"<start> rounded down to the nearest hour"`
	End      string `name:"end"        usage:"<end> rounded down to the nearest hour"`
}

// MarginRateHistoryParams defines command-line flags for exchange currency margin rate history retrieval and unmarshal their values.
type MarginRateHistoryParams struct {
	Exchange           string `name:"exchange,e"                   required:"t"                                                                         usage:"the exchange to get the candles from"`
	Asset              string `name:"asset,a"                      required:"t"                                                                         usage:"the asset type of the currency pair"`
	Currency           string `name:"currency,c"                   required:"t"                                                                         usage:"must be an enabled currency"`
	Start              string `name:"start,sd"                     usage:"<start>"`
	End                string `name:"end,ed"                       usage:"<end>"`
	GetPredictedRate   bool   `name:"getpredictedrate,p"           usage:"include the predicted upcoming rate in the response"`
	GetLendingPayments bool   `name:"getlendingpayments,lp"        usage:"retrieve and summarise your lending payments over the time period"`
	GetBorrowRates     bool   `name:"getborrowrates,br"            usage:"retrieve borrowing rates"`
	GetBorrowCosts     bool   `name:"getborrowcosts,bc"            usage:"retrieve and summarise your borrowing costs over the time period"`
	IncludeAllRates    bool   `name:"includeallrates,ar,v,verbose" usage:"include a detailed slice of all lending/borrowing rates over the time period"`
}

// CurrencyTradeURLParams defines command-line flags for exchange currency trade URL retrieval and unmarshal their values.
type CurrencyTradeURLParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to retrieve margin rates from"`
	Asset    string `name:"asset,a"    require:"t"  usage:"the asset type of the currency pair"`
	Pair     string `name:"pair,p"     require:"t"  usage:"the currency pair"`
}

// AddPortfolioAddressParams defines command-line flags for adding portfolio address and unmarshal their values.
type AddPortfolioAddressParams struct {
	Address            string  `name:"address"             usage:"the address to add to the portfolio"`
	Balance            float64 `name:"balance"             usage:"balance of the address"`
	CoinType           string  `name:"coin_type"           usage:"the coin type e.g ('BTC')"`
	Description        string  `name:"description"         usage:"description of the address"`
	ColdStorage        bool    `name:"cold_storage"        usage:"true/false if address is cold storage"`
	SupportedExchanges string  `name:"supported_exchanges" usage:"common separated list of exchanges supported by this address for withdrawals"`
}

// GetOrdersCommandParams defines command-line flags for exchange orders retrieval and unmarshal their values.
type GetOrdersCommandParams struct {
	Exchange string `name:"exchange,e" required:"t"                                                           usage:"the exchange to get orders for"`
	Asset    string `name:"asset"      required:"t"                                                           usage:"the asset type to get orders for"`
	Pair     string `name:"pair,p"     required:"t"                                                           usage:"the currency pair to get orders for"`
	Start    string `name:"start"      usage:"start date, optional. Will filter any results before this date"`
	End      string `name:"end"        usage:"end date, optional. Will filter any results after this date"`
}

// SimulateOrderParams defines command-line flags for simulate exchange order and unmarshal their values.
type SimulateOrderParams struct {
	Exchange  string  `name:"exchange,e" required:"t" usage:"the exchange to simulate the order for"`
	Pair      string  `name:"pair,p"     required:"t" usage:"the currency pair"`
	OrderSide string  `name:"side"       required:"t" usage:"the order side to use (BUY OR SELL)"`
	Amount    float64 `name:"amount"     required:"t" usage:"the amount for the order"`
}

// RemovePortfolioAddressParams defines command-line flags for simulate exchange order and unmarshal their values.
type RemovePortfolioAddressParams struct {
	Address     string `name:"address"     usage:"the address to add to the portfolio"`
	CoinType    string `name:"coin_type"   usage:"the coin type e.g ('BTC')"`
	Description string `name:"description" usage:"description of the address"`
}

// GetManagedOrdersParams defines command-line flags for managed orders retrieval and unmarshal their values.
type GetManagedOrdersParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to get orders for"`
	Asset    string `name:"asset"      required:"t" usage:"the asset type to get orders for"`
	Pair     string `name:"pair,p"     required:"t" usage:"the currency pair to get orders for"`
}

// WithdrawalRequestByDateParams defines command-line flags for withdrawal request by date and unmarshal their values.
type WithdrawalRequestByDateParams struct {
	Exchange string `name:"exchange,e" required:"t"                                                                                     usage:"the currency used in to withdraw"`
	Start    string `name:"start"      usage:"the start date to get withdrawals from. Any withdrawal before this date will be filtered"`
	End      string `name:"end"        usage:"the end date to get withdrawals from. Any withdrawal after this date will be filtered"`
	Limit    int64  `name:"limit"      usage:"max number of withdrawals to return"`
}

// WithdrawlRequestByExchangeIDParams defines command-line flags for withdrawal request by exchange id and unmarshal their values.
type WithdrawlRequestByExchangeIDParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"exchange name"`
	ID       string `name:"id"         required:"t" usage:"withdrawal id"`
}

// WithdrawlRequestByExchangeParams defines command-line flags for withdrawal request by exchange and unmarshal their values.
type WithdrawlRequestByExchangeParams struct {
	Exchange string `name:"exchange,e" required:"t"                                usage:"exchange name"`
	Asset    string `name:"asset"      usage:"the asset type of the currency pair"`
	Limit    int64  `name:"limit"      usage:"max number of withdrawals to return"`
	Currency string `name:"currency"   usage:"<currency>"`
}

// SetExchangeTradeProcessingParams defines command-line flags for setting exchange trade processing and unmarshal their values.
type SetExchangeTradeProcessingParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to change the status of"`
	Status   bool   `name:"status"     required:"t" usage:"<true>/<false>"`
}

// GetRecentTradesParams defines command-line flags for exchange trades retrieval and unmarshal their values.
type GetRecentTradesParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to get the trades from"`
	Pair     string `name:"pair,p"     required:"t" usage:"the currency pair to get the trades for"`
	Asset    string `name:"asset,a"    required:"t" usage:"the asset type of the currency pair"`
}

// GetTradesParams defines command-line flags for exchange trades retrieval and unmarshal their values.
type GetTradesParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to get the trades from"`
	Pair     string `name:"pair,p"     required:"t" usage:"the currency pair to get the trades for"`
	Asset    string `name:"asset,a"    required:"t" usage:"the asset type of the currency pair"`
	Start    string `name:"start,s"    required:"t" usage:"<start>"`
	End      string `name:"end"        required:"t" usage:"<end> WARNING: large date ranges may take considerable time"`
}

// FindMisingSavedTradeIntervalsParams defines command-line flags for exchange missing saved trade intervals retrieval and unmarshal their values.
type FindMisingSavedTradeIntervalsParams struct {
	Exchange string `name:"exchange,e" required:"t" usage:"the exchange to find the missing trades"`
	Pair     string `name:"pair,p"     required:"t" usage:"the currency pair"`
	Asset    string `name:"asset,a"    required:"t" usage:"the asset type of the currency pair"`
	Start    string `name:"start,s"    required:"t" usage:"<start> rounded down to the nearest hour"`
	End      string `name:"end"        required:"t" usage:"<end> rounded down to the nearest hour"`
}

// ConvertSavedTradesToCandlesParams defines command-line flags for exchange saved trades to candles conversion and unmarshal their values.
type ConvertSavedTradesToCandlesParams struct {
	Exchange string `name:"exchange,e" required:"t"                                                                                                                                                                                                                                                                     usage:"the exchange to find the missing trades"`
	Pair     string `name:"pair,p"     required:"t"                                                                                                                                                                                                                                                                     usage:"the currency pair"`
	Asset    string `name:"asset,a"    required:"t"                                                                                                                                                                                                                                                                     usage:"the asset type of the currency pair"`
	Interval int64  `name:"interval,i" usage:"interval in seconds. supported values are: 15, 60(1min), 180(3min), 300(5min), 600(10min),900(15min) 1800(30min), 3600(1h), 7200(2h), 14400(4h), 21600(6h), 28800(8h), 43200(12h),86400(1d), 259200(3d) 604800(1w), 1209600(2w), 1296000(15d), 2592000(1M), 31536000(1Y)"`
	Start    string `name:"start,s"    required:"t"                                                                                                                                                                                                                                                                     usage:"<start>"`
	End      string `name:"end,e"      required:"t"                                                                                                                                                                                                                                                                     usage:"<end>"`
	Sync     bool   `name:"sync,s"     usage:"will sync the resulting candles to the database <true/false>"`
	Force    bool   `name:"force,f"    usage:"will overwrite any conflicting candle data on save <true/false>"`
}

// WhaleBombParams defines command-line flags for exchange whale bomb amount required to reach a price target and unmarshal their values.
type WhaleBombParams struct {
	Exchange string  `name:"exchange,e" required:"t"             usage:"the exchange to whale bomb"`
	Pair     string  `name:"pair,p"     required:"t"             usage:"the currency pair"`
	Side     string  `name:"side,s"     required:"t"             usage:"the order side to use (BUY OR SELL)"`
	Asset    string  `name:"asset,a"    required:"t"             usage:"the asset type of the currency pair to get the orderbook for"`
	Price    float64 `name:"price"      usage:"the price target"`
}

// EnableDisableExchangePairParams defines command-line flags for exchange pair toggling and unmarshal their values.
type EnableDisableExchangePairParams struct {
	Exchange string `name:"exchange,e" required:"t"  usage:"the exchange to act on"`
	Pairs    string `name:"pairs,p"    required:"t"  usage:"either a single currency pair string or comma delimiter string of pairs e.g. \"BTC-USD,XRP-USD\""`
	Asset    string `name:"asset"      required:"t"  usage:"asset"`
	Enable   bool   `hidden:"t"        name:"enable"`
}

// EnableDisableExchangeAssetParams defines command-line flags for exchange asset toggling and unmarshal their values.
type EnableDisableExchangeAssetParams struct {
	Exchange string `name:"exchange,e" required:"t"  usage:"the exchange to act on"`
	Asset    string `name:"asset"      required:"t"  usage:"asset"`
	Enable   bool   `hidden:"t"        name:"enable"`
}
