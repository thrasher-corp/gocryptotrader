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
| AverageExecutedPrice | Average price of what's traded thus far                           | Order is filled, partially filled or partially canceled | Desirable |
| ExecutedAmount       | How much of the original order quantity is filled                 | Order is filled, partially filled or partially canceled | Mandatory |
| RemainingAmount      | Amount - ExecutedAmount                                           |                                                         | Mandatory |
| Cost                 | How much is spent thus far (cumulative transacted quote currency) | Order is filled, partially filled or partially canceled | Mandatory |
| CostAsset            | Deprecated, cost currency is always pair.Quote                    |                                                         | -         |
| Fee                  | How much last trade was charged by the exchange                   | Reported event is a trade                               | Optional  |
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
