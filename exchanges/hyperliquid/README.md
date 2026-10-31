# GoCryptoTrader package Hyperliquid

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/hyperliquid)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This hyperliquid package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Hyperliquid Exchange

### Current Features

+ Spot, perpetual and builder-deployed (HIP-3) perpetual DEX markets
+ REST and websocket tickers, L2 orderbooks, trades and candles
+ Account balances, orders, fills, TWAP history and slice fills, funding payments and ledger history
+ Exchange status, and vault, staking, referral, borrow/lend, outcome market and template, HIP-3\* DEX approval
  state and deployment information
+ Signed orders, cancels, modifications, TWAP and trailing stop orders, scheduled cancels, and leverage and margin
  updates, with orders, cancels and modifications also available over the websocket
+ Transfers between spot, perpetual DEXs and accounts, HyperEVM sends, HIP-3 backstop liquidator transfers, staking,
  and API wallet and builder fee approvals
+ Outcome splits, merges and negations, account abstraction settings, reward claims, request weight reservation,
  pending nonce invalidation, and gossip priority bids and auction status
+ Funding rates, open interest and fee estimates

### How to enable

+ [Enable via configuration](../../config/README.md#enable-exchange-via-config-example)

### Credentials

Hyperliquid accounts are EVM addresses, so the credentials hold addresses and private keys rather than API keys:

+ `Key` is the account address, whose data is queried
+ `Secret` is optional, and is the private key of the account or of one of its approved API wallets
+ `SubAccount` is optional, and is a subaccount or vault the account controls, whose data is then queried and on whose
  behalf trading actions are signed

Account data and account websocket feeds need only `Key`. Signed actions need `Secret`, and an approved API wallet is
recommended for trading. Transfers, withdrawals, staking, approvals and abstraction settings are user-signed, so they
need `Secret` to be the account's own private key, except `AgentSendAsset`, `AgentEnableDEXAbstraction`,
`AgentSetUserAbstraction`, `TransferVaultUSDC` and `TransferHIP3BackstopLiquidator`, which an API wallet can sign.
`SendAsset`, `AgentSendAsset`, `TransferUSDCBetweenSpotAndPerp`, `SetUserDEXAbstraction` and `SetUserAbstraction` act
for a configured subaccount and are refused when `SubAccount` is a vault, while the other user-signed actions,
including `WithdrawCryptocurrencyFunds`, are refused whenever `SubAccount` is set. The remaining actions act for the
account itself. The account, signer and subaccount or vault are checked against Hyperliquid's records before the first
signed action for each combination of them, and again after Hyperliquid rejects the signer; checked combinations are
remembered, so alternating subaccounts or credentials does not repeat the check.

Signed actions use millisecond nonces, which only increase within one exchange instance, so the same signer must not be
used by several instances or processes at once.

### Testnet

Set `useSandbox` to `true` to sign for testnet. Endpoints left at their production defaults then move to testnet, while
custom endpoints are kept, so `useSandbox` must match the environment behind a custom endpoint.

### Orders

+ Orders are sized in the base currency, so `QuoteAmount` is not supported.
+ Market orders are sent as IOC limit orders priced `SlippageTolerance` away from the mid price, so they accept only IOC
  or no time in force. `SlippageTolerance` must be above zero and below one, and defaults to 5%, as in Hyperliquid's
  Python SDK. The price is rounded towards the mid price, so the order never executes beyond its tolerance.
+ Trigger orders are supported on perpetual markets only, accept only GTC or no time in force, as Hyperliquid places a
  triggered order as GTC, and need not be reduce-only. Hyperliquid triggers on the mark price, so trigger orders and
  take-profit or stop-loss children must set `TriggerPriceType` to `order.MarkPrice`. A market trigger order or child
  without a limit price is priced `SlippageTolerance` away from its trigger price, which defaults to 10%, as for
  Hyperliquid's own TP/SL market orders.
+ `ModifyOrder` replaces resting limit orders only, and cannot add a take profit or stop loss. It leaves `always_place`
  unset, so Hyperliquid accepts only a replacement that rests, and cancels the order before placing it: a replacement
  that would match is rejected with the order already cancelled, and an error from `ModifyOrder` does not mean the order
  still rests. A placed replacement rests as ALO, whatever its time in force, and trigger and IOC replacements are not
  supported.
+ Fee estimates follow Hyperliquid's fee formula, from the account's rates and referral discount when credentials are
  set, and the base tier rates otherwise. A spot pair between two spot quote assets, the tokens that quote a listed spot
  market, pays a fifth of the rate. A builder-deployed perpetual market's deployer fee scale raises its fees, and growth
  mode cuts its fees and rebates to a tenth. The referral discount cuts fees but not rebates. Aligned quote asset
  discounts are not applied, as Hyperliquid documents none on mainnet.
+ `GetActiveOrders` reads open orders only from the DEXs of the requested perpetual pairs, or from every DEX when the
  request names no pairs or a pair whose market is not cached. Each DEX's orders come from `frontendOpenOrders`, which
  returns every open order of most accounts but only 100, not necessarily the newest, for an account with very many,
  such as one of HLP's child vaults, so a DEX at that count has the rest added from `openOrders`, without their order
  type, time in force or trigger price. Position TP/SL orders, which Hyperliquid lists without a size, take the size of
  the position they close.
+ HIP-4 outcome orders, which the first perpetual DEX lists beside spot and perpetual orders, are skipped by
  `GetActiveOrders` and `GetOrderHistory` and left open by `CancelAllOrders`, as no supported asset type covers them.
+ `GetOrderHistory` covers the account's 2000 most recent order status changes, and returns each order once, with its
  latest status and, for a triggered order, the trigger price and placement time of its earlier changes. An open order
  keeps the sizes of its latest status change, which can predate its partial fills; `GetActiveOrders` gives its
  remaining size. `GetOrderInfo` reads only the latest status, so a trigger order that has triggered and filled is
  returned without its trigger price and dated when it triggered. `orderStatus` can answer an open order with its status
  record from before its partial fills, so `GetOrderInfo` takes an open order's remaining size from the open orders,
  where it also finds an older open order Hyperliquid reports as unknown to `orderStatus`. It sizes a zero-size open
  position TP/SL order from the position it closes, as `GetActiveOrders` does, and gives a finished order that executed
  something its cost and average price from the account's fills, which `orderStatus` omits, and its fee when they paid
  it in one token; that costs `userFillsByTime` requests, and leaves them unset for an order whose fills Hyperliquid no
  longer keeps, as it keeps an account's 10000 most recent.
+ Hyperliquid cuts a reduce-only order down to the position it closes, when it is placed for more or other orders
  shrink the position, without a fill for the size it removes. An order's executed size is its original size less its
  remaining size, so `GetActiveOrders`, `GetOrderHistory` and `orderUpdates` rows count the size cut from a reduce-only
  order as executed, and fills relayed to the order stream before the order's final row count it as remaining. When a
  finished reduce-only order's fills fall short of its executed size, `GetOrderInfo` takes its executed size and status
  from them, once it has read them without passing over any and another `userFillsByTime` request shows Hyperliquid
  still keeps a fill of the account from before the order was placed, and so all of the order's. Such an order that
  executed something takes its executed and remaining sizes as its size; one that executed nothing keeps its original
  size, as the engine's order store keeps an executed size it holds when a lookup lowers it to zero.
+ Delisted perpetual markets are neither tradable nor subscribable, but their orders and fills still convert.

### Balances

Unified and portfolio margin accounts hold one balance across spot and perpetuals, which is stored under spot; while the
spot asset is disabled, refreshing perpetual balances refreshes it too, and clears it once the account holds separate
balances. Other accounts hold a balance per perpetual DEX, in that DEX's collateral token, stored under a subaccount ID
of the account address and DEX name. Its hold is the margin in use, up to the account value, and its free amount is what
can be withdrawn. Portfolio margin accounts report a spot hold beside a hold that can be negative, so a spot balance's
hold is the larger of the two, from zero up to its total, and the rest of the total is free.

### Websocket

The standard channels use these Hyperliquid feeds: tickers `activeAssetCtx`, orderbooks `l2Book`, trades `trades`,
candles `candle`, account orders `orderUpdates` and account trades `userEvents`. Any other feed, such as `bbo`,
`allMids` or `webData3`, is subscribed by using its name as the channel. Market feeds take one pair, account feeds must
be authenticated, and other feeds take no asset. Subscription parameters set the feed options:

+ `dex` selects a builder-deployed perpetual DEX for `allMids`, `clearinghouseState`, `openOrders` and `twapStates`
+ `nSigFigs` and `mantissa` aggregate an orderbook, and 5 levels select the fast orderbook, which Hyperliquid pushes
  about every half second instead of every 5 seconds
+ `aggregateByTime` combines an order's partial fills on `userFills`, and `ignorePortfolioMargin` omits portfolio margin
  amounts from `spotState`

Subscriptions differing only in these parameters select separate feeds and are held side by side, except that a
market's orderbook and the account's `userFills` take one variant at a time: two orderbook variants would replace each
other's stored book, and two `userFills` variants would relay each fill twice. An aggregated `userFills` is refused
beside `userEvents`, which the standard account trades channel uses, for the same reason.

The websocket manager runs two connections: a public one for market feeds and `WebsocketAPI()` posts, which stays open
without subscriptions, and an account one for account feeds, opened when account feeds are subscribed, authenticated
websocket support is enabled and an account address is configured. Hyperliquid limits an IP address to 1000
subscriptions across both, so the market feeds configured past the room the configured account feeds need are left
out, and held ones are removed when account feeds are added; any other market feed is refused once it would leave too
little room for the configured account feeds not yet held. When the account connection cannot connect or is lost, the
public feeds stay up, and the manager retries it after a connection monitor cycle, then at intervals doubling up to a
minute, or staying at the monitor delay when that is longer, as Hyperliquid accepts 30 new connections a minute from an
IP address; the interval carries over reconnections until the account connection connects.

Hyperliquid closes the connection, dropping every feed on it, when a subscription fails validation, so coins, DEXs and
options are validated first. Subscriptions complete when Hyperliquid acknowledges them, and are rolled back when it
rejects them, the connection drops or they time out. Hyperliquid may still subscribe a feed whose subscribe timed out or
was cancelled once sent, so the feed keeps its slot, and neither it nor a variant exclusive of it can be subscribed,
until Hyperliquid answers or the connection drops. A late acknowledgement leaves the feed sending but unheld until it is
subscribed again, which Hyperliquid answers as already subscribed; until then it is not checked against exclusive
variants, and counts towards the subscription limit only through the room kept for a configured account feed not yet
held. An unsubscribe that times out or is cancelled once sent does not keep its feed's slot but is taken as done, so
subscribing the feed again sends a subscribe, which Hyperliquid answers after the unsubscribe, as it answers a
connection's requests in order, or as already subscribed if it kept the feed. As the feed may send until Hyperliquid
answers the unsubscribe, a variant exclusive of it cannot be subscribed until then or until the connection drops; should
Hyperliquid refuse the unsubscribe, the feed is left sending but unheld, as after a late acknowledgement. When a
`trades`, `userFills` or `userTwapSliceFills` subscription opens, Hyperliquid replays recent entries, and only those
newer than the last entry relayed or skipped from that feed are relayed, so a first subscription relays none and
resubscribing after a reconnect relays what the replay holds of the gap. Account trades use `userEvents`, which includes
TWAP slice fills but replays nothing, so the fills it misses while disconnected are not recovered from the websocket.

`orderUpdates` sends a row only when an order's status changes, so fills from `userEvents`, and from `userFills` when
subscribed, are also relayed to the order stream as trades of their orders; a fill both feeds deliver is relayed there
once. Hyperliquid sends a block's fills before its rows, the fills of an order placed in the block among them, and the
engine stores an order it has not seen from its first update, so while `orderUpdates` is held for the account, a fill of
an order no row has shown waits for the rows of its block; a `userEvents` message names no account, so its fills wait
while `orderUpdates` is held for any account. The order's first row relays its trade; failing that, the fill is relayed
alone once a row or fill of a later block arrives, as for an order that rested before the feed was held, once the
account connection drops, as the rows sent while it is down are not resent, or at the account connection's first pong 2
seconds or more after its order's held fills began to wait, as a quiet account may send nothing else. An order's held
fills begin to wait when the first is held, and again when a later block's arrival releases some of them. TWAP slice
fills go only to the fills feed, as their orders get no rows and are unknown to `orderStatus`. Once an order's fills are
known to include all its executions, because its first row showed it unfilled or a row's executed size matched its
fills, each fill reports the order's executed and remaining size, cost, average price and total fee, and its final row
carries the cost, average price and total fee. The total fee is left out when the fills paid fees in different tokens,
as a spot order's fills can: a spot fill pays its fee in the token it receives but takes a maker rebate in the token it
gives. Otherwise a fill carries the trade alone, so it never lowers an executed size or fee learnt elsewhere. Fills from
a `userFills` feed with `aggregateByTime` go only to the fills feed, as each combines an order's fills within a block
under one of their trade IDs. Orders are remembered for a day after their last row or fill, the least recently touched
forgotten beyond 10000, though not while their fills wait for their first row, and the trade IDs of finished orders'
fills, 10000 at most, those of the orders that finished earliest dropped first, so a fill both feeds deliver after its
order's final row is still relayed once. Outcome (HIP-4) orders and fills are skipped.

`WebsocketAPI()` posts over the websocket instead of REST: its `PostInfo` sends an info request, and its `PlaceOrders`,
`CancelOrders`, `CancelOrdersByClientOrderID`, `ModifySingleOrder` and `ModifyOrders` send the same signed actions as
the exchange's methods of the same names.

### Withdrawals

`WithdrawCryptocurrencyFunds` only sends USDC to another Hyperliquid account, which needs `InternalTransfer` set, and is
refused while `SubAccount` is set. Withdrawals off Hyperliquid are not supported: Hyperliquid has deprecated its
Arbitrum bridge in favour of minting USDC natively through Circle's CCTP, which the package does not wrap. Bridge
deposits are credited to the address that sends them, so there are no deposit addresses, and a deposit below the
bridge's 5 USDC minimum is not credited and is lost.

### Not supported

Token, market and outcome deployment, validator actions and the other platform administration actions are not wrapped.
`GetHistoricCandles` refuses the intervals GCT aligns to calendar months, one, three, six and nine months and a year, as
Hyperliquid's `1M` candle spans 30 days from its own start dates. Its 3-day and weekly candles open on other days than
GCT's too, so those are built from daily candles, as are other intervals of whole days, while `GetCandleSnapshot` returns
Hyperliquid's own.
`SendSignedAction` signs and sends any L1 action, including one the package does not wrap, and its
`L1ActionOptionsRequest` sets the batch length that weighs the action against the rate limit, whether it acts for the
configured subaccount or vault, its expiry and nonce, and whether it is sent over the websocket instead of REST. The
signature hashes an action's fields in declaration order, so an action type the package does not define must declare its
fields, with `json` and `msgpack` tags, in the order Hyperliquid's SDKs send them.

### How to do REST public/private calls

When enabled through configuration, Hyperliquid is available through the `exchange.IBotExchange` interface:

```go
var h exchange.IBotExchange

for i := range bot.Exchanges {
	if bot.Exchanges[i].GetName() == "Hyperliquid" {
		h = bot.Exchanges[i]
	}
}

// Public calls
tick, err := h.UpdateTicker(ctx, pair, asset.Spot)
if err != nil {
	// Handle error
}

book, err := h.UpdateOrderbook(ctx, pair, asset.PerpetualContract)
if err != nil {
	// Handle error
}

trades, err := h.GetRecentTrades(ctx, pair, asset.PerpetualContract)
if err != nil {
	// Handle error
}

// Private calls, which need Key, and Secret for signed actions
balances, err := h.UpdateAccountBalances(ctx, asset.PerpetualContract)
if err != nil {
	// Handle error
}
```

The rest of the API is available through the package's own methods:

```go
hl, ok := h.(*hyperliquid.Exchange)
if !ok {
	// Handle error
}

funding, err := hl.GetUserFunding(ctx, &hyperliquid.UserFundingRequest{
	User:      address,
	StartTime: time.Now().Add(-24 * time.Hour),
})
if err != nil {
	// Handle error
}
```

### How to do Websocket public/private calls

Feeds are subscribed as the Websocket section describes. Orders, cancels and modifications can also be sent over the
websocket:

```go
statuses, err := hl.WebsocketAPI().PlaceOrders(ctx, &hyperliquid.PlaceOrdersRequest{
	Orders: []hyperliquid.OrderRequest{
		{
			Asset: 0, // BTC perpetual on mainnet
			IsBuy: true,
			Price: price,
			Size:  size,
			Limit: &hyperliquid.LimitOrderType{TimeInForce: hyperliquid.TimeInForceGTC},
		},
	},
})
if err != nil {
	// Handle error
}
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
