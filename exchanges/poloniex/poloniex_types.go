package poloniex

import (
	"fmt"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

type (
	// TimeInForce wraps order.TimeInForce and implements a custom Text marshaler.
	TimeInForce order.TimeInForce

	// OrderType wraps order.Type and implements a custom Text marshaler.
	OrderType order.Type

	// AccountType wraps asset.Item and implements a custom Text marshaler.
	AccountType asset.Item
)

// MarshalText implements the TextMarshaler interface for TimeInForce
func (t TimeInForce) MarshalText() ([]byte, error) {
	tif := order.TimeInForce(t)
	switch {
	case tif.Is(order.GoodTillCancel):
		return []byte("GTC"), nil
	case tif.Is(order.FillOrKill):
		return []byte("FOK"), nil
	case tif.Is(order.ImmediateOrCancel):
		return []byte("IOC"), nil
	case tif == order.UnknownTIF:
		return []byte(""), nil
	}
	return nil, fmt.Errorf("%w: %v", order.ErrInvalidTimeInForce, t)
}

// MarshalText implements the TextMarshaler interface for OrderType
func (o OrderType) MarshalText() ([]byte, error) {
	t := order.Type(o)
	switch t {
	case order.Market:
		return []byte("MARKET"), nil
	case order.Limit:
		return []byte("LIMIT"), nil
	case order.LimitMaker:
		return []byte("LIMIT_MAKER"), nil
	case order.Stop:
		return []byte("STOP"), nil
	case order.StopLimit:
		return []byte("STOP_LIMIT"), nil
	case order.TrailingStop:
		return []byte("TRAILING_STOP"), nil
	case order.TrailingStopLimit:
		return []byte("TRAILING_STOP_LIMIT"), nil
	case order.AnyType, order.UnknownType:
		return nil, nil
	}
	return nil, fmt.Errorf("%w: %v", order.ErrUnsupportedOrderType, o)
}

// MarshalText implements the TextMarshaler interface for AccountType
func (a AccountType) MarshalText() ([]byte, error) {
	switch asset.Item(a) {
	case asset.Spot:
		return []byte("SPOT"), nil
	case asset.Futures:
		return []byte("FUTURES"), nil
	default:
		return nil, asset.ErrNotSupported
	}
}

// DepositAddresses holds the full address per crypto-currency
type DepositAddresses map[string]string

// WithdrawalFees the large list of predefined withdrawal fees
// Prone to change, using highest value
var WithdrawalFees = map[currency.Code]float64{
	currency.ZRX:   5,
	currency.ARDR:  2,
	currency.REP:   0.1,
	currency.BTC:   0.0005,
	currency.BCH:   0.0001,
	currency.XBC:   0.0001,
	currency.BTCD:  0.01,
	currency.BTM:   0.01,
	currency.BTS:   5,
	currency.BURST: 1,
	currency.BCN:   1,
	currency.CVC:   1,
	currency.CLAM:  0.001,
	currency.XCP:   1,
	currency.DASH:  0.01,
	currency.DCR:   0.1,
	currency.DGB:   0.1,
	currency.DOGE:  5,
	currency.EMC2:  0.01,
	currency.EOS:   0,
	currency.ETH:   0.01,
	currency.ETC:   0.01,
	currency.EXP:   0.01,
	currency.FCT:   0.01,
	currency.GAME:  0.01,
	currency.GAS:   0,
	currency.GNO:   0.015,
	currency.GNT:   1,
	currency.GRC:   0.01,
	currency.HUC:   0.01,
	currency.LBC:   0.05,
	currency.LSK:   0.1,
	currency.LTC:   0.001,
	currency.MAID:  10,
	currency.XMR:   0.015,
	currency.NMC:   0.01,
	currency.NAV:   0.01,
	currency.XEM:   15,
	currency.NEOS:  0.0001,
	currency.NXT:   1,
	currency.OMG:   0.3,
	currency.OMNI:  0.1,
	currency.PASC:  0.01,
	currency.PPC:   0.01,
	currency.POT:   0.01,
	currency.XPM:   0.01,
	currency.XRP:   0.15,
	currency.SC:    10,
	currency.STEEM: 0.01,
	currency.SBD:   0.01,
	currency.XLM:   0.00001,
	currency.STORJ: 1,
	currency.STRAT: 0.01, //nolint:misspell // Not a misspelling
	currency.AMP:   5,
	currency.SYS:   0.01,
	currency.USDT:  10,
	currency.VRC:   0.01,
	currency.VTC:   0.001,
	currency.VIA:   0.01,
	currency.ZEC:   0.001,
}

// SubscriptionResponse represents a subscription response detail
type SubscriptionResponse struct {
	ID      string          `json:"id"`
	Channel string          `json:"channel"`
	Data    json.RawMessage `json:"data"`
	Action  string          `json:"action"`
	Event   string          `json:"event"`
	Message string          `json:"message"`
}

// SymbolTradeLimit holds symbol's trade limit details
type SymbolTradeLimit struct {
	Symbol           currency.Pair `json:"symbol"`
	PriceScale       uint8         `json:"priceScale"`
	QuantityScale    uint8         `json:"quantityScale"`
	QuoteAmountScale uint8         `json:"amountScale"`
	MinQuantity      types.Number  `json:"minQuantity"`
	MinAmount        types.Number  `json:"minAmount"`
	MaxQuantity      types.Number  `json:"maxQuantity"`
	MaxAmount        types.Number  `json:"maxAmount"`
	HighestBid       types.Number  `json:"highestBid"`
	LowestAsk        types.Number  `json:"lowestAsk"`
}

// WsSymbolTradeLimit holds a websocket symbol's trade limit details
type WsSymbolTradeLimit struct {
	Symbol           currency.Pair `json:"symbol"`
	PriceScale       uint8         `json:"priceScale"`
	QuantityScale    uint8         `json:"quantityScale"`
	QuoteAmountScale uint8         `json:"amountScale"`
	MinQuantity      types.Number  `json:"minQuantity"`
	MinAmount        types.Number  `json:"minAmount"`
	HighestBid       types.Number  `json:"highestBid"`
	LowestAsk        types.Number  `json:"lowestAsk"`
}

// SymbolDetails represents a currency symbol
type SymbolDetails struct {
	Symbol            currency.Pair           `json:"symbol"`
	BaseCurrencyName  currency.Code           `json:"baseCurrencyName"`
	QuoteCurrencyName currency.Code           `json:"quoteCurrencyName"`
	DisplayName       string                  `json:"displayName"`
	State             string                  `json:"state"`
	VisibleStartTime  types.Time              `json:"visibleStartTime"`
	TradableStartTime types.Time              `json:"tradableStartTime"`
	SymbolTradeLimit  *SymbolTradeLimit       `json:"symbolTradeLimit"`
	CrossMargin       *CrossMarginSupportInfo `json:"crossMargin"`
}

// Currency represents all supported currencies
type Currency struct {
	ID                uint64                 `json:"id"`
	Coin              currency.Code          `json:"coin"`
	Delisted          bool                   `json:"delisted"`
	TradeEnable       bool                   `json:"tradeEnable"`
	Name              string                 `json:"name"`
	NetworkList       []*CryptoNetworkDetail `json:"networkList"`
	SupportCollateral bool                   `json:"supportCollateral"`
	SupportBorrow     bool                   `json:"supportBorrow"`
}

// CryptoNetworkDetail holds a crypto network detail
type CryptoNetworkDetail struct {
	ID               uint64       `json:"id"`
	Coin             string       `json:"coin"`
	Name             string       `json:"name"`
	CurrencyType     string       `json:"currencyType"`
	Blockchain       string       `json:"blockchain"`
	WithdrawalEnable bool         `json:"withdrawalEnable"`
	DepositEnable    bool         `json:"depositEnable"`
	DepositAddress   string       `json:"depositAddress"`
	Decimals         float64      `json:"decimals"`
	MinConfirm       uint64       `json:"minConfirm"`
	WithdrawMin      types.Number `json:"withdrawMin"`
	WithdrawFee      types.Number `json:"withdrawFee"`
	ContractAddress  string       `json:"contractAddress"`
}

// ServerSystemTime represents a server time.
type ServerSystemTime struct {
	ServerTime types.Time `json:"serverTime"`
}

// MarketPrice represents ticker information.
type MarketPrice struct {
	Symbol        currency.Pair `json:"symbol"`
	DailyChange   types.Number  `json:"dailyChange"`
	Price         types.Number  `json:"price"`
	Timestamp     types.Time    `json:"time"`
	PushTimestamp types.Time    `json:"ts"`
}

// MarkPrice represents latest mark price for all cross margin symbols.
type MarkPrice struct {
	Symbol          currency.Pair `json:"symbol"`
	MarkPrice       types.Number  `json:"markPrice"`
	RecordTimestamp types.Time    `json:"time"`
}

// MarkPriceComponents represents a mark price component instance.
type MarkPriceComponents struct {
	Symbol     currency.Pair         `json:"symbol"`
	Timestamp  types.Time            `json:"ts"`
	MarkPrice  types.Number          `json:"markPrice"`
	Components []*MarkPriceComponent `json:"components"`
}

// MarkPriceComponent holds a mark price detail component
type MarkPriceComponent struct {
	Symbol       string       `json:"symbol"`
	Exchange     string       `json:"exchange"`
	SymbolPrice  types.Number `json:"symbolPrice"`
	Weight       types.Number `json:"weight"`
	ConvertPrice types.Number `json:"convertPrice"`
}

// OrderbookData represents an order book data for a specific symbol.
type OrderbookData struct {
	CreationTime  types.Time     `json:"time"`
	Scale         types.Number   `json:"scale"`
	Asks          []types.Number `json:"asks"`
	Bids          []types.Number `json:"bids"`
	PushTimestamp types.Time     `json:"ts"`
}

// CandlestickData represents a candlestick data for a specific symbol.
type CandlestickData struct {
	Low              types.Number
	High             types.Number
	Open             types.Number
	Close            types.Number
	BaseAmount       types.Number
	QuoteAmount      types.Number
	BuyTakerAmount   types.Number
	BuyTakerQuantity types.Number
	TradeCount       types.Number
	PushTimestamp    types.Time
	WeightedAverage  types.Number
	Interval         string
	StartTime        types.Time
	EndTime          types.Time
}

// UnmarshalJSON deserializes byte data into CandlestickData structure
func (c *CandlestickData) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &[14]any{&c.Low, &c.High, &c.Open, &c.Close, &c.QuoteAmount, &c.BaseAmount, &c.BuyTakerAmount, &c.BuyTakerQuantity, &c.TradeCount, &c.PushTimestamp, &c.WeightedAverage, &c.Interval, &c.StartTime, &c.EndTime})
}

// Trade represents a trade instance.
type Trade struct {
	ID          string       `json:"id"`
	Price       types.Number `json:"price"`
	BaseAmount  types.Number `json:"quantity"`
	QuoteAmount types.Number `json:"amount"`
	TakerSide   string       `json:"takerSide"`
	Timestamp   types.Time   `json:"ts"`
	CreateTime  types.Time   `json:"createTime"`
}

// TickerData represents a price ticker information.
type TickerData struct {
	Symbol      currency.Pair `json:"symbol"`
	Open        types.Number  `json:"open"`
	Low         types.Number  `json:"low"`
	High        types.Number  `json:"high"`
	Close       types.Number  `json:"close"`
	BaseAmount  types.Number  `json:"quantity"`
	QuoteAmount types.Number  `json:"amount"`
	TradeCount  uint64        `json:"tradeCount"`
	StartTime   types.Time    `json:"startTime"`
	CloseTime   types.Time    `json:"closeTime"`
	DisplayName string        `json:"displayName"`
	DailyChange types.Number  `json:"dailyChange"`
	Bid         types.Number  `json:"bid"`
	BidQuantity types.Number  `json:"bidQuantity"`
	Ask         types.Number  `json:"ask"`
	AskQuantity types.Number  `json:"askQuantity"`
	Timestamp   types.Time    `json:"ts"`
	MarkPrice   types.Number  `json:"markPrice"`
}

// CollateralDetails represents collateral information.
type CollateralDetails struct {
	Currency              currency.Code `json:"currency"`
	CollateralRate        types.Number  `json:"collateralRate"`
	InitialMarginRate     types.Number  `json:"initialMarginRate"`
	MaintenanceMarginRate types.Number  `json:"maintenanceMarginRate"`
}

// BorrowRateDetails represents borrow rates information
type BorrowRateDetails struct {
	Tier  string        `json:"tier"`
	Rates []*BorrowRate `json:"rates"`
}

// BorrowRate holds currency borrow detail
type BorrowRate struct {
	Currency         currency.Code `json:"currency"`
	DailyBorrowRate  types.Number  `json:"dailyBorrowRate"`
	HourlyBorrowRate types.Number  `json:"hourlyBorrowRate"`
	BorrowLimit      types.Number  `json:"borrowLimit"`
}

// AccountDetails represents a user account information.
type AccountDetails struct {
	AccountID    string `json:"accountId"`
	AccountType  string `json:"accountType"`
	AccountState string `json:"accountState"`
}

// AccountBalances represents each account’s id, type and balances (assets).
type AccountBalances struct {
	AccountID   string            `json:"accountId"`
	AccountType string            `json:"accountType"`
	Balances    []*AccountBalance `json:"balances"`
}

// AccountBalance holds an account balance of currency
type AccountBalance struct {
	CurrencyID string        `json:"currencyId"`
	Currency   currency.Code `json:"currency"`
	Available  types.Number  `json:"available"`
	Hold       types.Number  `json:"hold"`
}

// AccountActivity represents activities such as airdrop, rebates, staking,
// credit/debit adjustments, and other (historical adjustments).
type AccountActivity struct {
	ID           string        `json:"id"`
	Currency     currency.Code `json:"currency"`
	Amount       types.Number  `json:"amount"`
	State        string        `json:"state"`
	CreateTime   types.Time    `json:"createTime"`
	Description  string        `json:"description"`
	ActivityType uint8         `json:"activityType"` // possible values: 'ALL': 200, 'AIRDROP': 201, 'COMMISSION_REBATE': 202, 'STAKING': 203, 'REFERRAL_REBATE': 204, 'SWAP': 205, 'CREDIT_ADJUSTMENT': 104, 'DEBIT_ADJUSTMENT': 105, 'OTHER': 199
}

// AccountTransferRequest request parameter for account fund transfer.
type AccountTransferRequest struct {
	Currency    currency.Code `json:"currency"`
	Amount      float64       `json:"amount,string"`
	FromAccount string        `json:"fromAccount"`
	ToAccount   string        `json:"toAccount"`
}

// AccountTransferResponse represents an account transfer response.
type AccountTransferResponse struct {
	TransferID string `json:"transferId"`
}

// AccountTransferRecord represents an account transfer record.
type AccountTransferRecord struct {
	ID          string        `json:"id"`
	FromAccount string        `json:"fromAccount"`
	ToAccount   string        `json:"toAccount"`
	Currency    currency.Code `json:"currency"`
	State       string        `json:"state"`
	CreateTime  types.Time    `json:"createTime"`
	Amount      types.Number  `json:"amount"`
}

// FeeInfo represents an account transfer information.
type FeeInfo struct {
	TransactionDiscount bool              `json:"trxDiscount"`
	MakerRate           types.Number      `json:"makerRate"`
	TakerRate           types.Number      `json:"takerRate"`
	Volume30D           types.Number      `json:"volume30D"`
	SpecialFeeRates     []*SpecialFeeRate `json:"specialFeeRates"`
}

// SpecialFeeRate holds special fee rate of a symbol
type SpecialFeeRate struct {
	Symbol    string       `json:"symbol"`
	MakerRate types.Number `json:"makerRate"`
	TakerRate types.Number `json:"takerRate"`
}

// InterestHistory represents an interest history.
type InterestHistory struct {
	ID                  string        `json:"id"`
	Currency            currency.Code `json:"currencyName"`
	Principal           types.Number  `json:"principal"`
	Interest            types.Number  `json:"interest"`
	InterestRate        types.Number  `json:"interestRate"`
	InterestAccuredTime types.Time    `json:"interestAccuredTime"`
}

// SubAccount represents a users account.
type SubAccount struct {
	AccountID    string `json:"accountId"`
	AccountName  string `json:"accountName"`
	AccountState string `json:"accountState"`
	IsPrimary    bool   `json:"isPrimary,string"`
}

// SubAccountBalances represents a users account details and balances
type SubAccountBalances struct {
	AccountID   string               `json:"accountId"`
	AccountName string               `json:"accountName"`
	AccountType string               `json:"accountType"`
	IsPrimary   bool                 `json:"isPrimary,string"`
	Balances    []*SubAccountBalance `json:"balances"`
}

// SubAccountBalance holds a subaccount balance detail
type SubAccountBalance struct {
	Currency              currency.Code `json:"currency"`
	Available             types.Number  `json:"available"`
	Hold                  types.Number  `json:"hold"`
	MaxAvailable          types.Number  `json:"maxAvailable"`
	AccountEquity         types.Number  `json:"accountEquity"`
	UnrealisedPNL         types.Number  `json:"unrealisedPNL"`
	MarginBalance         types.Number  `json:"marginBalance"`
	PositionMargin        types.Number  `json:"positionMargin"`
	OrderMargin           types.Number  `json:"orderMargin"`
	FrozenFunds           types.Number  `json:"frozenFunds"`
	AvailableBalance      types.Number  `json:"availableBalance"`
	RealizedProfitAndLoss types.Number  `json:"pnl"`
}

// SubAccountTransferRequest represents a sub-account transfer request parameters.
type SubAccountTransferRequest struct {
	Currency        currency.Code `json:"currency"`
	Amount          float64       `json:"amount,string"`
	FromAccountID   string        `json:"fromAccountId"`
	FromAccountType string        `json:"fromAccountType"`
	ToAccountID     string        `json:"toAccountId"`
	ToAccountType   string        `json:"toAccountType"`
}

// SubAccountTransfer represents a sub-account transfer record.
type SubAccountTransfer struct {
	ID              string        `json:"id"`
	FromAccountID   string        `json:"fromAccountId"`
	FromAccountName string        `json:"fromAccountName"`
	FromAccountType string        `json:"fromAccountType"`
	ToAccountID     string        `json:"toAccountId"`
	ToAccountName   string        `json:"toAccountName"`
	ToAccountType   string        `json:"toAccountType"`
	Currency        currency.Code `json:"currency"`
	Amount          types.Number  `json:"amount"`
	State           string        `json:"state"`
	CreateTime      types.Time    `json:"createTime"`
}

// WalletActivity holds wallet activity info
type WalletActivity struct {
	Deposits    []*WalletDeposits    `json:"deposits"`
	Withdrawals []*WalletWithdrawals `json:"withdrawals"`
}

// WalletDeposits holds wallet deposit info
type WalletDeposits struct {
	DepositNumber uint64        `json:"depositNumber"`
	Currency      currency.Code `json:"currency"`
	Address       string        `json:"address"`
	Amount        types.Number  `json:"amount"`
	Confirmations uint64        `json:"confirmations"`
	TransactionID string        `json:"txid"`
	Timestamp     types.Time    `json:"timestamp"`
	Status        string        `json:"status"`
}

// WalletWithdrawals holds wallet withdrawal info
type WalletWithdrawals struct {
	WithdrawalRequestsID uint64        `json:"withdrawalRequestsId"`
	Currency             currency.Code `json:"currency"`
	Address              string        `json:"address"`
	Status               string        `json:"status"`
	TransactionID        string        `json:"txid"`
	IPAddress            string        `json:"ipAddress"`
	PaymentID            string        `json:"paymentID"`
	Amount               types.Number  `json:"amount"`
	Fee                  types.Number  `json:"fee"`
	Timestamp            types.Time    `json:"timestamp"`
}

// Withdraw holds withdraw information
type Withdraw struct {
	WithdrawRequestID uint64 `json:"withdrawalRequestsId"`
}

// WithdrawCurrencyRequest represents a V2 currency withdrawal parameter.
type WithdrawCurrencyRequest struct {
	Coin        currency.Code `json:"coin"`
	Network     string        `json:"network"`
	Amount      float64       `json:"amount,string"`
	Address     string        `json:"address"`
	AddressTag  string        `json:"addressTag,omitempty"`
	AllowBorrow bool          `json:"allowBorrow,omitempty"`
}

// AccountMargin represents an account margin response
type AccountMargin struct {
	TotalAccountValue types.Number `json:"totalAccountValue"`
	TotalMargin       types.Number `json:"totalMargin"`
	UsedMargin        types.Number `json:"usedMargin"`
	FreeMargin        types.Number `json:"freeMargin"`
	MaintenanceMargin types.Number `json:"maintenanceMargin"`
	CreationTime      types.Time   `json:"time"`
	MarginRatio       string       `json:"marginRatio"`
}

// BorrowStatus represents currency borrow status.
type BorrowStatus struct {
	Currency         currency.Code `json:"currency"`
	Available        types.Number  `json:"available"`
	Borrowed         types.Number  `json:"borrowed"`
	Hold             types.Number  `json:"hold"`
	MaxAvailable     types.Number  `json:"maxAvailable"`
	HourlyBorrowRate types.Number  `json:"hourlyBorrowRate"`
	Version          string        `json:"version"`
}

// MarginBuySellAmount represents a maximum buy and sell amount.
type MarginBuySellAmount struct {
	Symbol           string       `json:"symbol"`
	MaxLeverage      uint16       `json:"maxLeverage"`
	AvailableBuy     types.Number `json:"availableBuy"`
	MaxAvailableBuy  types.Number `json:"maxAvailableBuy"`
	AvailableSell    types.Number `json:"availableSell"`
	MaxAvailableSell types.Number `json:"maxAvailableSell"`
}

// PlaceOrderRequest represents place order parameters.
type PlaceOrderRequest struct {
	Symbol      currency.Pair `json:"symbol"`
	Side        order.Side    `json:"side"`
	Type        OrderType     `json:"type,omitempty"`
	AccountType AccountType   `json:"accountType,omitempty"`

	// BaseAmount Base units for the order. BaseAmount is required for MARKET SELL or any LIMIT orders
	BaseAmount float64 `json:"quantity,omitempty,string"`

	// QuoteAmount Quote units for the order. QuoteAmount is required for MARKET BUY order
	QuoteAmount float64 `json:"amount,omitempty,string"`

	// Price is required for non-market orders
	Price float64 `json:"price,omitempty,string"`

	TimeInForce   TimeInForce `json:"timeInForce,omitempty"` // GTC, IOC, FOK (Default: GTC)
	ClientOrderID string      `json:"clientOrderId,omitempty"`

	AllowBorrow             bool   `json:"allowBorrow,omitempty"`
	SelfTradePreventionMode string `json:"stpMode,omitempty"` // self-trade prevention. Defaults to EXPIRE_TAKER. None: enable self-trade; EXPIRE_TAKER: Taker order will be canceled when self-trade happens

	SlippageTolerance string `json:"slippageTolerance,omitempty"` // Used to control the maximum slippage ratio, the value range is greater than 0 and less than 1
}

type statusResponse struct {
	Message string `json:"message"`
	Code    int64  `json:"code"`
}

func (s *statusResponse) Error() error {
	if s == nil {
		return common.ErrNoResponse
	}
	if s.Code != 0 && s.Code != 200 {
		return fmt.Errorf("error code: %d; message: %s", s.Code, s.Message)
	}
	return nil
}

// CancelReplaceOrderRequest represents a cancellation and order replacement request parameter.
type CancelReplaceOrderRequest struct {
	OrderID           string      `json:"-"` // used in order path parameter.
	ClientOrderID     string      `json:"clientOrderId"`
	Price             float64     `json:"price,omitempty,string"`
	BaseAmount        float64     `json:"quantity,omitempty,string"`
	QuoteAmount       float64     `json:"amount,omitempty,string"`
	AmendedType       string      `json:"type,omitempty,string"`
	TimeInForce       TimeInForce `json:"timeInForce,omitempty"`
	AllowBorrow       bool        `json:"allowBorrow,omitempty"`
	ProceedOnFailure  bool        `json:"proceedOnFailure,omitempty,string"`
	SlippageTolerance float64     `json:"slippageTolerance,omitempty,string"`
}

// CancelReplaceOrderResponse represents a response parameter for order cancellation and replacement operation.
type CancelReplaceOrderResponse struct {
	ID            string       `json:"id"`
	ClientOrderID string       `json:"clientOrderId"`
	Price         types.Number `json:"price"`
	Quantity      types.Number `json:"quantity"`
	Code          uint64       `json:"code"`
	Message       string       `json:"message"`
}

// OrdersHistoryRequest holds a orders history request parameters
type OrdersHistoryRequest struct {
	Symbol      currency.Pair
	AccountType string
	OrderType   string
	OrderTypes  []string
	Side        order.Side
	Direction   string
	States      string
	From        int64
	Limit       int64
	StartTime   time.Time
	EndTime     time.Time
	HideCancel  bool
}

// OrderHistoryItem represents an order history item
type OrderHistoryItem struct {
	TradeOrder
	ID uint64 `json:"id"`
}

// TradeOrder represents a trade order
type TradeOrder struct {
	ID             string            `json:"id"`
	ClientOrderID  string            `json:"clientOrderId"`
	Symbol         currency.Pair     `json:"symbol"`
	State          string            `json:"state"`
	AccountType    string            `json:"accountType"`
	Side           order.Side        `json:"side"`
	Type           string            `json:"type"`
	TimeInForce    order.TimeInForce `json:"timeInForce"`
	Price          types.Number      `json:"price"`
	AveragePrice   types.Number      `json:"avgPrice"`
	BaseAmount     types.Number      `json:"quantity"`
	QuoteAmount    types.Number      `json:"amount"`
	FilledQuantity types.Number      `json:"filledQuantity"`
	FilledAmount   types.Number      `json:"filledAmount"`
	CreateTime     types.Time        `json:"createTime"`
	UpdateTime     types.Time        `json:"updateTime"`
	OrderSource    string            `json:"orderSource"`
	Loan           bool              `json:"loan"`
	CancelReason   uint64            `json:"cancelReason"`

	statusResponse
}

// SmartOrder represents a smart order detail.
type SmartOrder struct {
	ID              string            `json:"id"`
	ClientOrderID   string            `json:"clientOrderId"`
	Symbol          currency.Pair     `json:"symbol"`
	State           string            `json:"state"`
	AccountType     string            `json:"accountType"`
	Side            order.Side        `json:"side"`
	Type            string            `json:"type"`
	TimeInForce     order.TimeInForce `json:"timeInForce"`
	Price           types.Number      `json:"price"`
	ActivationPrice types.Number      `json:"activationPrice"`
	BaseAmount      types.Number      `json:"quantity"`
	QuoteAmount     types.Number      `json:"amount"`
	StopPrice       types.Number      `json:"stopPrice"`
	CreateTime      types.Time        `json:"createTime"`
	UpdateTime      types.Time        `json:"updateTime"`
	TrailingOffset  string            `json:"trailingOffset"`
	LimitOffset     string            `json:"limitOffset"`
	Operator        string            `json:"operator"`
}

// CancelOrderResponse represents a cancel order response instance.
type CancelOrderResponse struct {
	OrderID       string `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	State         string `json:"state"`
	statusResponse
}

// WsCancelOrderResponse represents a websocket cancel orders instance.
type WsCancelOrderResponse struct {
	OrderID       uint64 `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	State         string `json:"state"`
	statusResponse
}

// CancelOrdersRequest represents cancel spot order request parameters
type CancelOrdersRequest struct {
	OrderIDs       []string `json:"orderIds,omitempty"`
	ClientOrderIDs []string `json:"clientOrderIds,omitempty"`
}

// KillSwitchStatus represents a kill switch response
type KillSwitchStatus struct {
	StartTime        types.Time `json:"startTime"`
	CancellationTime types.Time `json:"cancellationTime"`
}

// SmartOrderRequest represents a smart trade order's parameters
type SmartOrderRequest struct {
	Symbol         currency.Pair `json:"symbol"`
	Side           order.Side    `json:"side"`
	TimeInForce    TimeInForce   `json:"timeInForce,omitempty"`
	AccountType    AccountType   `json:"accountType,omitempty"`
	Type           OrderType     `json:"type,omitempty"`
	Price          float64       `json:"price,omitempty,string"`
	StopPrice      float64       `json:"stopPrice,omitempty,string"`
	BaseAmount     float64       `json:"quantity,omitempty,string"`
	QuoteAmount    float64       `json:"amount,omitempty,string"`
	ClientOrderID  string        `json:"clientOrderId,omitempty"`
	TrailingOffset string        `json:"trailingOffset,omitempty"` // trailing stop offset; Append % to trail by percentage
	LimitOffset    string        `json:"limitOffset,omitempty"`    // When trigger price is reached a limit order is placed. Append % for percentage
	Operator       string        `json:"operator,omitempty"`       // Direction for TRAILING_STOP orders; Allowed values: `GTE` for >= or `LTE` for <=
}

// CancelReplaceSmartOrderRequest represents a cancellation and order replacement request parameter for smart orders.
type CancelReplaceSmartOrderRequest struct {
	OrderID          string      `json:"-"` // will be used in request path
	OldClientOrderID string      `json:"-"`
	NewClientOrderID string      `json:"clientOrderId,omitempty"`
	Price            float64     `json:"price,omitempty,string"`
	StopPrice        float64     `json:"stopPrice,omitempty,string"`
	BaseAmount       float64     `json:"quantity,omitempty,string"`
	QuoteAmount      float64     `json:"amount,omitempty,string"`
	AmendedType      OrderType   `json:"type,omitempty,string"`
	TimeInForce      TimeInForce `json:"timeInForce,omitempty"`
	ProceedOnFailure bool        `json:"proceedOnFailure,omitempty,string"` // proceedOnFailure flag is intended to specify whether to continue with new smart order placement in case cancellation of the existing smart order fails.
}

// WsOrderIDResponse represents order's ID response for websocket create order
type WsOrderIDResponse struct {
	OrderID       int64  `json:"orderId"`
	ClientOrderID string `json:"clientOrderId"`
	statusResponse
}

// OrderIDResponse represents order's ID response structure details
type OrderIDResponse struct {
	ID            string `json:"id"`
	ClientOrderID string `json:"clientOrderId"`
	statusResponse
}

// SmartOrderDetails represents a smart order information and trigger detailed information.
type SmartOrderDetails struct {
	ID              string            `json:"id"`
	ClientOrderID   string            `json:"clientOrderId"`
	Symbol          currency.Pair     `json:"symbol"`
	State           string            `json:"state"`
	AccountType     string            `json:"accountType"`
	Side            order.Side        `json:"side"`
	Type            string            `json:"type"`
	TimeInForce     order.TimeInForce `json:"timeInForce"`
	Price           types.Number      `json:"price"`
	BaseAmount      types.Number      `json:"quantity"`
	QuoteAmount     types.Number      `json:"amount"`
	StopPrice       types.Number      `json:"stopPrice"`
	CreateTime      types.Time        `json:"createTime"`
	UpdateTime      types.Time        `json:"updateTime"`
	TriggeredOrder  TradeOrder        `json:"triggeredOrder"`
	TrailingOffset  string            `json:"trailingOffset"`
	LimitOffset     string            `json:"limitOffset"`
	ActivationPrice types.Number      `json:"activationPrice"`
	Operator        string            `json:"operator"`
}

// TradeHistory represents an order trade history instance.
type TradeHistory struct {
	ID            string        `json:"id"`
	ClientOrderID string        `json:"clientOrderId"`
	Symbol        string        `json:"symbol"`
	AccountType   string        `json:"accountType"`
	OrderID       string        `json:"orderId"`
	Side          order.Side    `json:"side"`
	Type          string        `json:"type"`
	MatchRole     string        `json:"matchRole"`
	Price         types.Number  `json:"price"`
	BaseAmount    types.Number  `json:"quantity"`
	QuoteAmount   types.Number  `json:"amount"`
	FeeCurrency   currency.Code `json:"feeCurrency"`
	FeeAmount     types.Number  `json:"feeAmount"`
	PageID        string        `json:"pageId"`
	CreateTime    types.Time    `json:"createTime"`
}

// SubscriptionPayload represents a subscriptions request instance structure.
type SubscriptionPayload struct {
	Event      string         `json:"event"`
	Channel    []string       `json:"channel"`
	Symbols    []string       `json:"symbols,omitempty"`
	Currencies []string       `json:"currencies,omitempty"`
	Depth      int64          `json:"depth,omitempty"`
	Params     map[string]any `json:"params,omitempty"`
}

// CrossMarginSupportInfo represents information on whether cross margin support is enabled or not, and leverage detail
type CrossMarginSupportInfo struct {
	SupportCrossMargin bool   `json:"supportCrossMargin"`
	MaxLeverage        uint64 `json:"maxLeverage"`
}

// WsCrossMarginSupportInfo represents information returned through the websocket stream on whether cross margin support is enabled or not, and leverage detail
type WsCrossMarginSupportInfo struct {
	SupportCrossMargin bool   `json:"supportCrossMargin"`
	MaxLeverage        uint64 `json:"maxLeverage,string"`
}

// WsSymbol represents a subscription
type WsSymbol struct {
	Symbol            currency.Pair             `json:"symbol"`
	BaseCurrencyName  currency.Code             `json:"baseCurrencyName"`
	QuoteCurrencyName currency.Code             `json:"quoteCurrencyName"`
	DisplayName       string                    `json:"displayName"`
	State             string                    `json:"state"`
	VisibleStartTime  types.Time                `json:"visibleStartTime"`
	TradableStartTime types.Time                `json:"tradableStartTime"`
	CrossMargin       *WsCrossMarginSupportInfo `json:"crossMargin"`
	SymbolTradeLimit  *WsSymbolTradeLimit       `json:"symbolTradeLimit"`
}

// WsCurrency represents a currency instance from websocket stream.
type WsCurrency struct {
	ID                uint64        `json:"id"`
	Name              string        `json:"name"`
	Currency          currency.Code `json:"currency"`
	Description       string        `json:"description"`
	Type              string        `json:"type"`
	WithdrawalFee     types.Number  `json:"withdrawalFee"`
	MinConf           uint64        `json:"minConf"`
	DepositAddress    string        `json:"depositAddress"`
	Blockchain        string        `json:"blockchain"`
	Delisted          bool          `json:"delisted"`
	TradingState      string        `json:"tradingState"`
	WalletState       string        `json:"walletState"`
	ParentChain       string        `json:"parentChain"`
	IsMultiChain      bool          `json:"isMultiChain"`
	IsChildChain      bool          `json:"isChildChain"`
	SupportCollateral bool          `json:"supportCollateral"`
	SupportBorrow     bool          `json:"supportBorrow"`
	ChildChains       []string      `json:"childChains"`
}

// WsExchangeStatus represents websocket exchange status.
// the values for MM and POM are ON and OFF
type WsExchangeStatus struct {
	MaintenanceMode string `json:"MM"`
	PostOnlyMode    string `json:"POM"`
}

// WsCandles represents a candlestick data instance.
type WsCandles struct {
	Symbol      currency.Pair `json:"symbol"`
	Open        types.Number  `json:"open"`
	High        types.Number  `json:"high"`
	Low         types.Number  `json:"low"`
	Close       types.Number  `json:"close"`
	BaseAmount  types.Number  `json:"quantity"`
	QuoteAmount types.Number  `json:"amount"`
	TradeCount  uint64        `json:"tradeCount"`
	StartTime   types.Time    `json:"startTime"`
	CloseTime   types.Time    `json:"closeTime"`
	Timestamp   types.Time    `json:"ts"`
}

// WsTrade represents websocket trade data
type WsTrade struct {
	ID          uint64        `json:"id,string"`
	Symbol      currency.Pair `json:"symbol"`
	BaseAmount  types.Number  `json:"quantity"`
	QuoteAmount types.Number  `json:"amount"`
	TakerSide   order.Side    `json:"takerSide"`
	Price       types.Number  `json:"price"`
	CreateTime  types.Time    `json:"createTime"`
	Timestamp   types.Time    `json:"ts"`
}

// WsTicker represents a websocket ticker information.
type WsTicker struct {
	TradeCount  uint64        `json:"tradeCount"`
	Symbol      currency.Pair `json:"symbol"`
	StartTime   types.Time    `json:"startTime"`
	Open        types.Number  `json:"open"`
	High        types.Number  `json:"high"`
	Low         types.Number  `json:"low"`
	Close       types.Number  `json:"close"`
	BaseAmount  types.Number  `json:"quantity"`
	QuoteAmount types.Number  `json:"amount"`
	DailyChange types.Number  `json:"dailyChange"`
	MarkPrice   types.Number  `json:"markPrice"`
	CloseTime   types.Time    `json:"closeTime"`
	Timestamp   types.Time    `json:"ts"`
}

// WsBook represents an orderbook.
type WsBook struct {
	Symbol     currency.Pair                    `json:"symbol"`
	Asks       orderbook.LevelsArrayPriceAmount `json:"asks"`
	Bids       orderbook.LevelsArrayPriceAmount `json:"bids"`
	ID         int64                            `json:"id"`
	Timestamp  types.Time                       `json:"ts"`
	CreateTime types.Time                       `json:"createTime"`
	LastID     int64                            `json:"lastId"`
}

// AuthRequest represents websocket authentication parameters
type AuthRequest struct {
	Key              string `json:"key"`
	SignTimestamp    int64  `json:"signTimestamp"`
	SignatureMethod  string `json:"signatureMethod,omitempty"`
	SignatureVersion string `json:"signatureVersion,omitempty"`
	Signature        string `json:"signature"`
}

// WebsocketAuthenticationResponse represents websocket authentication response.
type WebsocketAuthenticationResponse struct {
	Success   bool       `json:"success"`
	Message   string     `json:"message"`
	Timestamp types.Time `json:"ts"`
}

// WebsocketTradeOrder represents a websocket trade order.
type WebsocketTradeOrder struct {
	Symbol         currency.Pair `json:"symbol"`
	Type           string        `json:"type"`
	BaseAmount     types.Number  `json:"quantity"`
	OrderID        string        `json:"orderId"`
	TradeFee       types.Number  `json:"tradeFee"`
	ClientOrderID  string        `json:"clientOrderId"`
	AccountType    string        `json:"accountType"`
	FeeCurrency    currency.Code `json:"feeCurrency"`
	EventType      string        `json:"eventType"`
	Source         string        `json:"source"`
	Side           order.Side    `json:"side"`
	FilledQuantity types.Number  `json:"filledQuantity"`
	FilledAmount   types.Number  `json:"filledAmount"`
	MatchRole      string        `json:"matchRole"`
	State          string        `json:"state"`
	TradeTime      types.Time    `json:"tradeTime"`
	TradeAmount    types.Number  `json:"tradeAmount"`
	OrderAmount    types.Number  `json:"orderAmount"`
	CreateTime     types.Time    `json:"createTime"`
	Price          types.Number  `json:"price"`
	TradeQty       types.Number  `json:"tradeQty"`
	TradePrice     types.Number  `json:"tradePrice"`
	TradeID        string        `json:"tradeId"`
	Timestamp      types.Time    `json:"ts"`
}

// WsTradeBalance represents a balance information through the websocket channel
type WsTradeBalance struct {
	ID          uint64        `json:"id"`
	UserID      uint64        `json:"userId"`
	ChangeTime  types.Time    `json:"changeTime"`
	AccountID   string        `json:"accountId"`
	AccountType string        `json:"accountType"`
	EventType   string        `json:"eventType"`
	Available   types.Number  `json:"available"`
	Currency    currency.Code `json:"currency"`
	Hold        types.Number  `json:"hold"`
	Version     uint32        `json:"version"`
	Timestamp   types.Time    `json:"ts"`
}

// WebsocketResponse represents a websocket responses.
type WebsocketResponse struct {
	ID   string `json:"id"`
	Data any    `json:"data"`
}

// SubAccountTransferRecordRequest represents a sub-account transfer record retrieval parameters
type SubAccountTransferRecordRequest struct {
	Currency        currency.Code
	StartTime       time.Time
	EndTime         time.Time
	FromAccountID   string
	ToAccountID     string
	FromAccountType string
	ToAccountType   string
	Direction       string
	From            uint64
	Limit           uint64
}

// V3ResponseWrapper holds a wrapper struct for V3 and smart-order endpoints
type V3ResponseWrapper struct {
	Code    int64  `json:"code"`
	Message string `json:"msg"`
	Data    any    `json:"data"`
}

func (s *V3ResponseWrapper) Error() error {
	if s.Code != 0 && s.Code != 200 {
		return fmt.Errorf("error code: %d; message: %s", s.Code, s.Message)
	}
	if s.Data == nil {
		return common.ErrNoResponse
	}
	return nil
}

// UnmarshalJSON conforms type to the unmarshaler interface
func (s *V3ResponseWrapper) UnmarshalJSON(data []byte) error {
	var aux struct {
		Code    int64           `json:"code"`
		Message string          `json:"message"`
		Data    json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	s.Code = aux.Code
	s.Message = aux.Message
	if (s.Code == 0 || s.Code == 200) && aux.Data != nil {
		return json.Unmarshal(aux.Data, &s.Data)
	}

	return nil
}

type hasError interface{ Error() error }
