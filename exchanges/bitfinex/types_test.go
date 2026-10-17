package bitfinex

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// testTupleDecode decodes data into a new T and compares it with exp, so a test case states a record's decode in one
// expected value
func testTupleDecode[T any](t *testing.T, data string, exp T) {
	t.Helper()
	var got T
	require.NoErrorf(t, json.Unmarshal([]byte(data), &got), "Unmarshal must not error for %T", got)
	assert.Equalf(t, exp, got, "Unmarshal should decode every field of %T", got)
}

// TestSharedRecordsEveryPosition decodes each shared record from an array with a distinct value at every tagged
// position, so a field tagged with the wrong position or type cannot pass on a null or zero element
func TestFundingFlagsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		data string
		exp  FundingFlags
	}{
		{data: `64`, exp: 64},
		{data: `0`},
		{data: `{}`},
		{data: ` {"future":1}`},
	} {
		var got FundingFlags
		require.NoErrorf(t, got.UnmarshalJSON([]byte(tc.data)), "UnmarshalJSON must not error for %s", tc.data)
		assert.Equalf(t, tc.exp, got, "UnmarshalJSON should decode %s", tc.data)
	}
	var got FundingFlags
	assert.Error(t, got.UnmarshalJSON([]byte(`"x"`)), "UnmarshalJSON should error for a string")
	testTupleDecode(t, `[1000,"fUSD",null,null,1,1,"LIMIT",null,null,{},"ACTIVE"]`,
		FundingOffer{ID: 1000, Symbol: "fUSD", Amount: 1, OriginalAmount: 1, Type: FundingOfferTypeLimit, Status: "ACTIVE"})
}

func TestSharedRecordsEveryPosition(t *testing.T) {
	t.Parallel()
	testTupleDecode(t, `[1000,1001,1002,"s3",1700000000004,1700000000005,6.5,7.5,"s8","s9",1700000000010,null,1012,"s13",null,null,16.5,17.5,18.5,19.5,null,null,null,1,1,1025,null,null,"s28",null,null,{"k":31}]`,
		Order{ID: 1000, GroupID: 1001, ClientOrderID: 1002, Symbol: "s3", CreatedAt: types.Time(time.UnixMilli(1700000000004)), UpdatedAt: types.Time(time.UnixMilli(1700000000005)), Amount: 6.5, OriginalAmount: 7.5, Type: "s8", PreviousType: "s9", TimeInForce: types.Time(time.UnixMilli(1700000000010)), Flags: 1012, Status: "s13", Price: 16.5, AveragePrice: 17.5, TrailingPrice: 18.5, AuxiliaryLimitPrice: 19.5, Notify: true, Hidden: true, PlacedID: 1025, Routing: "s28", Meta: json.RawMessage(`{"k":31}`)})
	testTupleDecode(t, `[1000,"s1",1700000000002,1003,4.5,5.5,"s6",7.5,1008,9.5,"UST",1011]`,
		AccountTrade{ID: 1000, Symbol: "s1", ExecutedAt: types.Time(time.UnixMilli(1700000000002)), OrderID: 1003, ExecutedAmount: 4.5, ExecutedPrice: 5.5, OrderType: "s6", OrderPrice: 7.5, Maker: 1008, Fee: 9.5, FeeCurrency: currency.UST, ClientOrderID: 1011})
	testTupleDecode(t, `["s0","s1",2.5,3.5,4.5,1005,6.5,7.5,8.5,9.5,1010,1011,1700000000012,1700000000013,null,1015,null,17.5,18.5,{"k":19}]`,
		Position{Symbol: "s0", Status: "s1", Amount: 2.5, BasePrice: 3.5, MarginFunding: 4.5, MarginFundingType: 1005, ProfitLoss: 6.5, ProfitLossPercentage: 7.5, LiquidationPrice: 8.5, Leverage: 9.5, Flags: 1010, ID: 1011, CreatedAt: types.Time(time.UnixMilli(1700000000012)), UpdatedAt: types.Time(time.UnixMilli(1700000000013)), Type: 1015, Collateral: 17.5, MinimumCollateral: 18.5, Meta: json.RawMessage(`{"k":19}`)})
	testTupleDecode(t, `[1000,"s1",1700000000002,1700000000003,4.5,5.5,"s6",null,null,1009,"s10",null,null,null,14.5,1015,1,1,null,1]`,
		FundingOffer{ID: 1000, Symbol: "s1", CreatedAt: types.Time(time.UnixMilli(1700000000002)), UpdatedAt: types.Time(time.UnixMilli(1700000000003)), Amount: 4.5, OriginalAmount: 5.5, Type: "s6", Flags: 1009, Status: "s10", Rate: 14.5, Period: 1015, Notify: true, Hidden: true, Renew: true})
	testTupleDecode(t, `[1000,"s1",1002,1700000000003,1700000000004,5.5,1006,"s7","s8",null,null,11.5,1012,1700000000013,1700000000014,1,1,null,1,19.5,1,"s21"]`,
		FundingCredit{ID: 1000, Symbol: "s1", Side: 1002, CreatedAt: types.Time(time.UnixMilli(1700000000003)), UpdatedAt: types.Time(time.UnixMilli(1700000000004)), Amount: 5.5, Flags: 1006, Status: "s7", RateType: "s8", Rate: 11.5, Period: 1012, OpenedAt: types.Time(time.UnixMilli(1700000000013)), LastPayoutAt: types.Time(time.UnixMilli(1700000000014)), Notify: true, Hidden: true, Renew: true, RealRate: 19.5, NoClose: true, PositionPair: "s21"})
	testTupleDecode(t, `[1000,"s1",1700000000002,1003,4.5,5.5,1006,1]`,
		AccountFundingTrade{ID: 1000, Symbol: "s1", CreatedAt: types.Time(time.UnixMilli(1700000000002)), OfferID: 1003, Amount: 4.5, Rate: 5.5, Period: 1006, Maker: true})
	testTupleDecode(t, `["s0",[0.5,1.5,2.5,3.5,4.5]]`,
		MarginInfoBaseResponse{Key: "s0", Info: MarginInfoBase{UserProfitLoss: 0.5, UserSwaps: 1.5, MarginBalance: 2.5, MarginNet: 3.5, MarginMinimum: 4.5}})
	testTupleDecode(t, `["s0","s1",[0.5,1.5,2.5,3.5]]`,
		MarginInfoSymbolResponse{Key: "s0", Symbol: "s1", Info: MarginInfoSymbol{TradableBalance: 0.5, GrossBalance: 1.5, BuyAmount: 2.5, SellAmount: 3.5}})
	testTupleDecode(t, `["s0","s1",[0.5,1.5,2.5,3.5]]`,
		FundingInfoResponse{Key: "s0", Symbol: "s1", Info: FundingInfo{YieldLoan: 0.5, YieldLend: 1.5, DurationLoan: 2.5, DurationLend: 3.5}})
}

// TestSharedRecordsDocumentedExamples decodes the docs' REST and websocket examples of each shared record
func TestSharedRecordsDocumentedExamples(t *testing.T) {
	t.Parallel()
	// POST auth/r/orders, and the websocket order cancel event's order
	testTupleDecode(t, `[33950998275,null,1573476747887,"tETHUSD",1573476748000,1573476748000,-0.5,-0.5,"LIMIT",null,null,null,0,"ACTIVE",null,null,220,0,0,0,null,null,null,0,0,null,null,null,"BFX",null,null,null]`,
		Order{ID: 33950998275, ClientOrderID: 1573476747887, Symbol: "tETHUSD", CreatedAt: types.Time(time.UnixMilli(1573476748000)), UpdatedAt: types.Time(time.UnixMilli(1573476748000)), Amount: -0.5, OriginalAmount: -0.5, Type: OrderTypeLimit, Status: "ACTIVE", Price: 220, Routing: "BFX"})
	testTupleDecode(t, `[34930659963,null,1574955083558,"tETHUSD",1574955083558,1574955354487,0.201104,0.201104,"EXCHANGE LIMIT",null,null,null,0,"CANCELED",null,null,120,0,0,0,null,null,null,0,0,null,null,null,"BFX",null,null,null]`,
		Order{ID: 34930659963, ClientOrderID: 1574955083558, Symbol: "tETHUSD", CreatedAt: types.Time(time.UnixMilli(1574955083558)), UpdatedAt: types.Time(time.UnixMilli(1574955354487)), Amount: 0.201104, OriginalAmount: 0.201104, Type: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "CANCELED", Price: 120, Routing: "BFX"})

	// POST auth/r/trades/hist, POST auth/r/order/{symbol}:{id}/trades, which leaves out the order price, and the
	// websocket trade executed and execution update events
	testTupleDecode(t, `[402088407,"tETHUST",1574963975602,34938060782,-0.2,153.57,"MARKET",0,-1,-0.061668,"USD",1234]`,
		AccountTrade{ID: 402088407, Symbol: "tETHUST", ExecutedAt: types.Time(time.UnixMilli(1574963975602)), OrderID: 34938060782, ExecutedAmount: -0.2, ExecutedPrice: 153.57, OrderType: OrderTypeMarket, Maker: -1, Fee: -0.061668, FeeCurrency: currency.USD, ClientOrderID: 1234})
	testTupleDecode(t, `[399251013,"tETHUSD",1573485493000,33963608932,0.26334268,187.37,"EXCHANGE MARKET",null,-1,-0.09868591,"USD",1234]`,
		AccountTrade{ID: 399251013, Symbol: "tETHUSD", ExecutedAt: types.Time(time.UnixMilli(1573485493000)), OrderID: 33963608932, ExecutedAmount: 0.26334268, ExecutedPrice: 187.37, OrderType: ExchangeOrderTypePrefix + OrderTypeMarket, Maker: -1, Fee: -0.09868591, FeeCurrency: currency.USD, ClientOrderID: 1234})
	testTupleDecode(t, `[402088407,"tETHUST",1574963975602,34938060782,-0.2,153.57,"MARKET",0,-1,null,null,0]`,
		AccountTrade{ID: 402088407, Symbol: "tETHUST", ExecutedAt: types.Time(time.UnixMilli(1574963975602)), OrderID: 34938060782, ExecutedAmount: -0.2, ExecutedPrice: 153.57, OrderType: OrderTypeMarket, Maker: -1})
	testTupleDecode(t, `[402088407,"tETHUST",1574963975602,34938060782,-0.2,153.57,"MARKET",0,-1,-0.061668,"USD",1714466193700]`,
		AccountTrade{ID: 402088407, Symbol: "tETHUST", ExecutedAt: types.Time(time.UnixMilli(1574963975602)), OrderID: 34938060782, ExecutedAmount: -0.2, ExecutedPrice: 153.57, OrderType: OrderTypeMarket, Maker: -1, Fee: -0.061668, FeeCurrency: currency.USD, ClientOrderID: 1714466193700})

	// POST auth/r/positions, and the websocket position update event, which sends the flags
	positionMeta := json.RawMessage(`{"reason":"TRADE","order_id":34271018124,"liq_stage":null,"trade_price":"8565.0267019","trade_amount":"0.0195","order_id_oppo":34277498022}`)
	testTupleDecode(t, `["tBTCUSD","ACTIVE",0.0195,8565.0267019,0,0,-0.33455568705000516,-0.0003117550117425625,7045.876419249083,3.0673001895895604,null,142355652,1574002216000,1574002216000,null,0,null,0,0,`+string(positionMeta)+`]`,
		Position{Symbol: "tBTCUSD", Status: "ACTIVE", Amount: 0.0195, BasePrice: 8565.0267019, ProfitLoss: -0.33455568705000516, ProfitLossPercentage: -0.0003117550117425625, LiquidationPrice: 7045.876419249083, Leverage: 3.0673001895895604, ID: 142355652, CreatedAt: types.Time(time.UnixMilli(1574002216000)), UpdatedAt: types.Time(time.UnixMilli(1574002216000)), Meta: positionMeta})
	positionMeta = json.RawMessage(`{"reason":"TRADE","order_id":34934099168,"order_id_oppo":34934090814,"liq_stage":null,"trade_price":"153.71","trade_amount":"0.2"}`)
	testTupleDecode(t, `["tETHUST","ACTIVE",0.2,153.71,0,0,-0.07944800000000068,-0.05855181835925015,67.52755254906451,1.409288545397275,null,142420429,null,null,null,0,null,0,0,`+string(positionMeta)+`]`,
		Position{Symbol: "tETHUST", Status: "ACTIVE", Amount: 0.2, BasePrice: 153.71, ProfitLoss: -0.07944800000000068, ProfitLossPercentage: -0.05855181835925015, LiquidationPrice: 67.52755254906451, Leverage: 1.409288545397275, ID: 142420429, Meta: positionMeta})

	// POST auth/r/funding/offers, and the websocket funding offer new event
	testTupleDecode(t, `[652606505,"fETH",1574000611000,1574000611000,0.29797676,0.29797676,"LIMIT",null,null,0,"ACTIVE",null,null,null,0.0002,2,0,null,null,0,null]`,
		FundingOffer{ID: 652606505, Symbol: "fETH", CreatedAt: types.Time(time.UnixMilli(1574000611000)), UpdatedAt: types.Time(time.UnixMilli(1574000611000)), Amount: 0.29797676, OriginalAmount: 0.29797676, Type: FundingOfferTypeLimit, Status: "ACTIVE", Rate: 0.0002, Period: 2})
	testTupleDecode(t, `[41238747,"fUST",1575026670000,1575026670000,5000,5000,"LIMIT",null,null,0,"ACTIVE",null,null,null,0.006000000000000001,30,0,0,null,0,null]`,
		FundingOffer{ID: 41238747, Symbol: "fUST", CreatedAt: types.Time(time.UnixMilli(1575026670000)), UpdatedAt: types.Time(time.UnixMilli(1575026670000)), Amount: 5000, OriginalAmount: 5000, Type: FundingOfferTypeLimit, Status: "ACTIVE", Rate: 0.006000000000000001, Period: 30})

	// POST auth/r/funding/credits, POST auth/r/funding/loans, which sends no position pair, and the websocket funding
	// credit update event
	testTupleDecode(t, `[26222883,"fUST",1,1574013661000,1574079687000,350,null,"ACTIVE","FIXED",null,null,0.0024,2,1574013661000,1574078487000,0,null,null,0,null,0,"tBTCUST"]`,
		FundingCredit{ID: 26222883, Symbol: "fUST", Side: 1, CreatedAt: types.Time(time.UnixMilli(1574013661000)), UpdatedAt: types.Time(time.UnixMilli(1574079687000)), Amount: 350, Status: "ACTIVE", RateType: "FIXED", Rate: 0.0024, Period: 2, OpenedAt: types.Time(time.UnixMilli(1574013661000)), LastPayoutAt: types.Time(time.UnixMilli(1574078487000)), PositionPair: "tBTCUST"})
	testTupleDecode(t, `[2995368,"fUST",0,1574077517000,1574077517000,100,null,"ACTIVE","FIXED",null,null,0.0024,2,1574077517000,1574077517000,0,null,null,0,null,0]`,
		FundingCredit{ID: 2995368, Symbol: "fUST", CreatedAt: types.Time(time.UnixMilli(1574077517000)), UpdatedAt: types.Time(time.UnixMilli(1574077517000)), Amount: 100, Status: "ACTIVE", RateType: "FIXED", Rate: 0.0024, Period: 2, OpenedAt: types.Time(time.UnixMilli(1574077517000)), LastPayoutAt: types.Time(time.UnixMilli(1574077517000))})
	testTupleDecode(t, `[26223578,"fUST",1,1575052261000,1575296787000,350,0,"ACTIVE",null,null,null,0,30,1575052261000,1575293487000,0,0,null,0,null,0,"tBTCUST"]`,
		FundingCredit{ID: 26223578, Symbol: "fUST", Side: 1, CreatedAt: types.Time(time.UnixMilli(1575052261000)), UpdatedAt: types.Time(time.UnixMilli(1575296787000)), Amount: 350, Status: "ACTIVE", Period: 30, OpenedAt: types.Time(time.UnixMilli(1575052261000)), LastPayoutAt: types.Time(time.UnixMilli(1575293487000)), PositionPair: "tBTCUST"})

	// POST auth/r/funding/trades/hist, and the websocket funding trade executed event
	testTupleDecode(t, `[636040,"fUST",1574077528000,41237922,-100,0.0024,2,null]`,
		AccountFundingTrade{ID: 636040, Symbol: "fUST", CreatedAt: types.Time(time.UnixMilli(1574077528000)), OfferID: 41237922, Amount: -100, Rate: 0.0024, Period: 2})
	testTupleDecode(t, `[636854,"fUSD",1575282446000,41238905,-1000,0.002,7,null]`,
		AccountFundingTrade{ID: 636854, Symbol: "fUSD", CreatedAt: types.Time(time.UnixMilli(1575282446000)), OfferID: 41238905, Amount: -1000, Rate: 0.002, Period: 7})

	// POST auth/r/info/margin/base and auth/r/info/margin/tBTCUSD, whose arrays carry four more nulls, and POST
	// auth/r/info/funding/fUST
	testTupleDecode(t, `["base",[-3631.798995959999,5.6376,299933.09380832,296301.29481236,4975.291049394]]`,
		MarginInfoBaseResponse{Key: "base", Info: MarginInfoBase{UserProfitLoss: -3631.798995959999, UserSwaps: 5.6376, MarginBalance: 299933.09380832, MarginNet: 296301.29481236, MarginMinimum: 4975.291049394}})
	testTupleDecode(t, `["sym","tBTCUSD",[854774.7939165855,909999.9796645334,52.08334097688924,62.676711921295684,null,null,null,null]]`,
		MarginInfoSymbolResponse{Key: "sym", Symbol: "tBTCUSD", Info: MarginInfoSymbol{TradableBalance: 854774.7939165855, GrossBalance: 909999.9796645334, BuyAmount: 52.08334097688924, SellAmount: 62.676711921295684}})
	testTupleDecode(t, `["sym","fUST",[0.0024,0.0024,1.9522164351851852,1.4818560606060607]]`,
		FundingInfoResponse{Key: "sym", Symbol: "fUST", Info: FundingInfo{YieldLoan: 0.0024, YieldLend: 0.0024, DurationLoan: 1.9522164351851852, DurationLend: 1.4818560606060607}})
}

func TestNotificationUnmarshalJSON(t *testing.T) {
	t.Parallel()
	// POST auth/w/order/submit
	testTupleDecode(t, `[1678987199,"on-req",null,null,[[1747566428,null,1678987199446,"tBTCUSD",1678987199446,1678987199448,0.1,0.1,"EXCHANGE LIMIT",null,null,null,0,"ACTIVE",null,null,10000,0,0,0,null,null,null,0,0,null,null,null,"API>BFX",null,null,null]],null,"SUCCESS","Submitting exchange limit buy order for 0.1 BTC."]`,
		Notification[[]Order]{
			Time: types.Time(time.Unix(1678987199, 0)),
			Type: "on-req",
			Data: []Order{{
				ID: 1747566428, ClientOrderID: 1678987199446, Symbol: "tBTCUSD", CreatedAt: types.Time(time.UnixMilli(1678987199446)), UpdatedAt: types.Time(time.UnixMilli(1678987199448)),
				Amount: 0.1, OriginalAmount: 0.1, Type: ExchangeOrderTypePrefix + OrderTypeLimit, Status: "ACTIVE", Price: 10000, Routing: "API>BFX",
			}},
			Status: NotificationStatusSuccess,
			Text:   "Submitting exchange limit buy order for 0.1 BTC.",
		})
	// A websocket notification event's notification, whose time has fractional seconds
	testTupleDecode(t, `[1575287438.515,"on-req",7,null,[1185815098,null,1575287436979,"tETHUSD"],null,"ERROR","Invalid order: not enough exchange balance"]`,
		Notification[Order]{Time: types.Time(time.UnixMilli(1575287438515)), Type: "on-req", MessageID: 7, Data: Order{ID: 1185815098, ClientOrderID: 1575287436979, Symbol: "tETHUSD"}, Status: NotificationStatusError, Text: "Invalid order: not enough exchange balance"})
}

func TestNotificationErr(t *testing.T) {
	t.Parallel()
	for status, failed := range map[string]bool{NotificationStatusSuccess: false, "INFO": false, "": false, NotificationStatusError: true, NotificationStatusFailure: true} {
		n := &Notification[Order]{Type: "on-req", Status: status, Text: "Invalid order"}
		if !failed {
			assert.NoErrorf(t, n.Err(), "Err should not error for status %q", status)
			continue
		}
		err := n.Err()
		assert.ErrorIsf(t, err, errNotificationFailed, "Err should error for status %q", status)
		assert.ErrorContainsf(t, err, "on-req "+status+": Invalid order", "Err should give the notification's type, status and text for status %q", status)
	}
}
