# Websocket trade events

GoCryptoTrader unifies trades and order update events by composing an
order.Detail object.  This is the full list of order.Detail fields
that exchange implementations should populate on streamed
trade/order-update events.  As exchanges provide different APIs, not
all fields are mandatory.

Note to developers: a special mention is the AverageExecutedPrice,
which is not always provided, but its presence is important and highly
desirable.  Even if not reported, effort should be made to compute it
out of reported trades.

| order.Detail field   | Description                                                       | Condition                                               | Presence  |
|----------------------|-------------------------------------------------------------------|---------------------------------------------------------|-----------|
| Price                | Original price assigned to order                                  | Depends on order type (e.g. limit orders have prices)   | Mandatory |
| Amount               | Original quantity assigned to order                               |                                                         | Mandatory |
| QuoteAmount          | Requested quantity reported in the quote currency                     | Exchange reports a requested quote quantity             | Desirable |
| AverageExecutedPrice | Average price of what's traded thus far                           | Order is filled, partially filled or partially cancelled | Desirable |
| ExecutedAmount       | How much of the original order quantity is filled                 | Order is filled, partially filled or partially cancelled | Mandatory |
| RemainingAmount      | Unfilled quantity in the same unit as Amount and ExecutedAmount       | Exchange reports it in that unit, or both inputs to subtraction are known in that unit | Desirable |
| ExecutedQuoteAmount  | Cumulative executed value in the quote currency, before fees      | Order is filled, partially filled or partially cancelled | Desirable |
| Fee                  | Fee amount reported for the order or event                        | Keep cumulative and per-fill fees distinct              | Optional  |
| FeeAsset             | Asset of the taken fee                                            |                                                         | Optional  |
| Exchange             | String name of concerned exchange                                 |                                                         | Mandatory |
| ID                   | Order ID (on the exchange)                                        |                                                         | Mandatory |
| ClientOrderID        | Client order ID (submitted by user)                               |                                                         | Mandatory |
| Type                 | e.g. MARKET or LIMIT, see exchanges/order/order_types.go          |                                                         | Mandatory |
| Side                 | e.g. BUY or SELL, see exchanges/order/order_types.go              |                                                         | Mandatory |
| Status               | e.g. FILLED or CANCELLED, see exchanges/order/order_types.go      |                                                         | Mandatory |
| AssetType            | e.g. asset.Spot or asset.Futures                                  |                                                         | Mandatory |
| Date                 | Time of order creation (as reported by the exchange)              |                                                         | Optional  |
| LastUpdated          | Time of last order event (as reported by the exchange)            |                                                         | Optional  |
| Pair                 | Tradable pair                                                     |                                                         | Mandatory |

Leave `RemainingAmount` unknown when the exchange reports the remainder only
in another unit. For example, if a market buy requests 10 USDT, fills 0.000199
BTC and has 0.05 USDT left, keep `QuoteAmount` as 10 and the executed base
amount as 0.000199; do not put 0.05 USDT into a base-quantity field or subtract
BTC from USDT. A zero remainder means nothing remains only when the response
establishes that fact; otherwise zero can mean unavailable.
