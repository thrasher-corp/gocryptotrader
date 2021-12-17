# GoCryptoTrader package Bank

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/exchanges/bank)
[![Coverage Status](http://codecov.io/github/thrasher-corp/gocryptotrader/coverage.svg?branch=master)](http://codecov.io/github/thrasher-corp/gocryptotrader?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This bank package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on this Trello board: [https://trello.com/b/ZAhMhpOy/gocryptotrader](https://trello.com/b/ZAhMhpOy/gocryptotrader).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/enQtNTQ5NDAxMjA2Mjc5LTc5ZDE1ZTNiOGM3ZGMyMmY1NTAxYWZhODE0MWM5N2JlZDk1NDU0YTViYzk4NTk3OTRiMDQzNGQ1YTc4YmRlMTk)

## Bank

### Current Features

+ Provides a transfer type for different bank transfers.
+ Provides methods on the type for validation and to implement the stringer interface.


## Transfer Types
### Via gRPC when setting a new bank transfer these integers refer to the bank transfer types:

|Bank Transfer Name|Integer|Notes|
|-------|------|------|
| "NotApplicable" | 1 | --- |
| "WireTransfer" | 2 | --- |
| "ExpressWireTransfer" | 3 | --- |
| "PerfectMoney" | 4 | --- |
| "Neteller" | 5 | --- |
| "AdvCash" | 6 | --- |
| "Payeer" | 7 | --- |
| "Skrill" | 8 | --- |
| "Simplex" | 9 | --- |
| "SEPA" | 10 | --- |
| "Swift" | 11 | --- |
| "RapidTransfer" | 12 | --- |
| "MisterTangoSEPA" | 13 | --- |
| "Qiwi" | 14 | --- |
| "VisaMastercard" | 15 | --- |
| "WebMoney" | 16 | --- |
| "Capitalist" | 17 | --- |
| "WesternUnion" | 18 | --- |
| "MoneyGram" | 19 | --- |
| "Contact" | 20 | --- |
| "PayID/Osko" | 21 | --- |
| "BankCard Visa" | 22 | --- |
| "BankCard Mastercard" | 23 | --- |
| "BankCard MIR" | 24 | --- |
| "CreditCard Mastercard" | 25 | --- |
| "Sofort" | 26 | --- |
| "P2P" | 27 | --- |
| "Etana" | 28 | --- |
| "FasterPaymentService(FPS)" | 29 | --- |
| "MobileMoney" | 30 | --- |
| "CashTransfer" | 31 | --- |
| "YandexMoney" | 32 | --- |
| "GEOPay" | 33 | --- |
| "SettlePay" | 34 | --- |
| "ExchangeFiatDWChannelSignetUSD" | 35 | --- |
| "ExchangeFiatDWChannelSignetUSD" | 36 | --- |
| "AutomaticClearingHouse" | 37 | --- |
| "FedWire" | 38 | --- |
| "TelegraphicTransfer" | 39 | --- |
| "SDDomesticCheque" | 40 | --- |
| "Xfers" | 41 | --- |
| "ExmoGiftCard" | 42 | --- |
| "Terminal" | 43 | --- |

### Please click GoDocs chevron above to view current GoDoc information for this package

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our coding guidelines:

+ Code must adhere to the official Go [formatting](https://golang.org/doc/effective_go.html#formatting) guidelines (i.e. uses [gofmt](https://golang.org/cmd/gofmt/)).
+ Code must be documented adhering to the official Go [commentary](https://golang.org/doc/effective_go.html#commentary) guidelines.
+ Code must adhere to our [coding style](https://github.com/thrasher-corp/gocryptotrader/blob/master/doc/coding_style.md).
+ Pull requests need to be based on and opened against the `master` branch.

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
