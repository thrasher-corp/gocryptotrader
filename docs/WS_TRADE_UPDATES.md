# Websocket trade events

GoCryptoTrader unifies trading and order-updating events by composing
an order.Detail object.

This is the full list of order.Detail fields that exchange
implementations should populate on streamed trade events.

Every once in a while an exchange changes its API.  This list is a
guide for developers to keep GCT reporting unified across exchanges.

Some fields are mandatory and are expected from all exchange
implementations.

Some fields are optional as not all exchanges report them.

```
| order.Detail field   | Description                                                       | Condition                                               | Presence  |
|----------------------+-------------------------------------------------------------------+---------------------------------------------------------+-----------|
| Price                | Original price assigned to order                                  | Depends on order type (e.g. limit orders have prices)   | Mandatory |
| Amount               | Original quantity assigned to order                               |                                                         | Mandatory |
| AverageExecutedPrice | Average price of what's traded thus far                           | Order is filled, partially filled or partially canceled | Mandatory |
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
```
