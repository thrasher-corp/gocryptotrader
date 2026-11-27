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
| AverageExecutedPrice | Average price of what's traded thus far                           | Order is filled, partially filled or partially cancelled | Desirable |
| ExecutedAmount       | How much of the original order quantity is filled                 | Order is filled, partially filled or partially cancelled | Mandatory |
| RemainingAmount      | Amount - ExecutedAmount                                           |                                                         | Mandatory |
| Cost                 | How much is spent thus far (cumulative value, in CostAsset)       | Order is filled, partially filled or partially cancelled | Mandatory |
| CostAsset            | Currency of Cost, pair.Quote or a derivative's settle coin        |                                                         | Optional  |
| Fee                  | Fee charged; follows Fees once reported, see [Fees](#fees)        | Reported event is a trade, or Fees is reported          | Optional  |
| FeeAsset             | Asset of the taken fee                                            |                                                         | Optional  |
| Fees                 | Cumulative fees per currency, see [Fees](#fees)                   | Exchange reports the order's cumulative fees            | Desirable |
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

## Fees

`Fees` holds an order's cumulative fees with one amount per currency, as the
exchange reports them. Amounts in different currencies are never summed or
converted, a negative amount is a rebate and a zero amount is kept. Build it
with `order.NewFees` and set it with `Detail.SetFees`, which also sets `Fee`
and `FeeAsset`. These rules apply to every `order.Detail` an exchange returns,
whether streamed or fetched over REST.

- Leave `Fees` unset when the event or response does not carry the order's
  cumulative fees, such as a fill event. The order manager then keeps the fees
  it has stored.
- Set a snapshot when it does. A snapshot replaces the stored fees, so a
  repeated snapshot never adds to them. Set an empty snapshot,
  `order.NewFees()`, when the exchange reports that the order has incurred no
  fees.
- Put each fill's own fees on its `order.TradeHistory` with
  `TradeHistory.SetFees`. Fills merge by trade ID, and their fees never change
  the order's `Fees`.
- Once `Fees` is reported, `Fee` and `FeeAsset` hold its only amount and
  currency, or zero and an empty currency when it holds none or several,
  because no single amount can stand for fees in several currencies. An update
  that sets only `Fee` and `FeeAsset` then leaves them alone. Exchanges that do
  not report `Fees` keep setting `Fee` and `FeeAsset` as before.
