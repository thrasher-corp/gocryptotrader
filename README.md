<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

A cryptocurrency trading bot supporting multiple exchanges written in Golang.

**Please note that this bot is under development and is not ready for production!**

## Community

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Exchange Support Table

| Exchange | REST API | Websocket API | FIX API |
|----------|------|-----------|-----|
| Binance.US| Yes  | Yes        | NA  |
| Binance| Yes  | Yes        | NA  |
| Bitfinex | Yes  | Yes        | NA  |
| Bitflyer | Yes  | No      | NA  |
| Bithumb | Yes  | Yes       | NA  |
| BitMEX | Yes | Yes | NA |
| Bitstamp | Yes  | Yes       | No  |
| BTCMarkets | Yes | Yes       | NA  |
| BTSE | Yes | Yes | NA |
| Bybit | Yes | Yes | NA |
| Coinbase | Yes | Yes | No|
| COINUT | Yes | Yes | NA |
| Deribit | Yes | Yes | No |
| Exmo | Yes | NA | NA |
| GateIO | Yes | Yes | NA |
| Gemini | Yes | Yes | No |
| HitBTC | Yes | Yes | No |
| Huobi.Pro | Yes | Yes | NA |
| Kraken | Yes | Yes | NA |
| Kucoin | Yes | Yes | NA |
| Lbank | Yes | No | NA |
| Okx | Yes | Yes | NA |
| Poloniex | Yes | Yes | NA |
| Yobit | Yes | NA | NA |

We are aiming to support the top 30 exchanges sorted by average liquidity as [ranked by CoinMarketCap](https://coinmarketcap.com/rankings/exchanges/). 
However, we welcome pull requests for any exchange which does not match this criterion. If you need help with this, please join us on [Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g).

** NA means not applicable as the exchange does not support the feature.

## Current Features

+ Support for all exchange fiat and digital currencies, with the ability to individually toggle them on/off.
+ AES256 encrypted config file.
+ REST API support for all exchanges.
+ Websocket support for applicable exchanges.
+ Ability to turn off/on certain exchanges.
+ Communication packages (Slack, SMS via SMSGlobal, Telegram and SMTP).
+ HTTP rate limiter package.
+ Unified API for exchange usage.
+ Customisation of HTTP client features including setting a proxy, user agent and adjusting transport settings.
+ NTP client package.
+ Database support (Postgres and SQLite3). See [database](/database/README.md).
+ OTP generation tool. See [gen otp](/cmd/gen_otp).
+ Connection monitor package.
+ gRPC service and JSON RPC proxy. See [gRPC service](/gctrpc/README.md).
+ gRPC client. See [gctcli](/cmd/gctcli/README.md).
+ Forex currency converter packages (CurrencyConverterAPI, CurrencyLayer, Exchange Rates, Fixer.io, OpenExchangeRates, Exchange Rate Host).
+ Packages for handling currency pairs, tickers and orderbooks.
+ Portfolio management tool; fetches balances from supported exchanges and allows for custom address tracking.
+ Basic event trigger system.
+ OHLCV/Candle retrieval support. See [OHLCV](/docs/OHLCV.md).
+ Scripting support. See [gctscript](/gctscript/README.md).
+ Recent and historic trade processing. See [trades](/exchanges/trade/README.md).
+ Backtesting application. An event-driven backtesting tool to test and iterate trading strategies using historical or custom data. See [backtester](/backtester/README.md).
+ Exchange HTTP mock testing. See [mock](/exchanges/mock/README.md).
+ Exchange multichain deposits and withdrawals for specific exchanges. See [multichain transfer support](/docs/MULTICHAIN_TRANSFER_SUPPORT.md).

## Development Tracking

Our [Kanban board](https://github.com/orgs/thrasher-corp/projects/3) provides updates on:

+ New feature development
+ Bug fixes in progress
+ Recently completed work
+ Contribution opportunities

Follow our progress as we continuously improve GoCryptoTrader.

## Contribution

Please feel free to submit any pull requests or suggest any desired features to be added.

When submitting a PR, please abide by our [coding guidelines](/docs/CODING_GUIDELINES.md).

## Compiling and Running instructions

Download and install Go from [Go Downloads](https://golang.org/dl/) for your platform.

### Linux/macOS

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader
go build
mkdir ~/.gocryptotrader
cp config_example.json ~/.gocryptotrader/config.json
```

### Windows

```bash
git clone https://github.com/thrasher-corp/gocryptotrader.git
cd gocryptotrader
go build
mkdir %AppData%\GoCryptoTrader
copy config_example.json %APPDATA%\GoCryptoTrader\config.json
```

+ Make any necessary changes to the `config.json` file.
+ Run the `gocryptotrader` binary file.

### Sonic JSON handling

GoCryptoTrader can optionally use the [Sonic](https://github.com/bytedance/sonic) JSON library for improved performance, as a drop in replacement for golang.org/encoding/json.
Please see sonic [Requirements](https://github.com/bytedance/sonic/#requirement) for supported platforms.

To enable sonic, build with the sonic_on tag:

```bash
go build -tags=sonic_on
```

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***

## Binaries

Binaries will be published once the codebase reaches a stable condition.

## Contributor List

### A very special thank you to all who have contributed to this program:

|User|Contribution Amount|
|--|--|
| [thrasher-](https://github.com/thrasher-) | 737 |
| [dependabot[bot]](https://github.com/apps/dependabot) | 415 |
| [shazbert](https://github.com/shazbert) | 389 |
| [gloriousCode](https://github.com/gloriousCode) | 240 |
| [gbjk](https://github.com/gbjk) | 143 |
| [dependabot-preview[bot]](https://github.com/apps/dependabot-preview) | 88 |
| [xtda](https://github.com/xtda) | 47 |
| [lrascao](https://github.com/lrascao) | 27 |
| [Beadko](https://github.com/Beadko) | 24 |
| [samuael](https://github.com/samuael) | 16 |
| [vazha](https://github.com/vazha) | 15 |
| [ydm](https://github.com/ydm) | 15 |
| [Rots](https://github.com/Rots) | 15 |
| [ermalguni](https://github.com/ermalguni) | 14 |
| [MadCozBadd](https://github.com/MadCozBadd) | 13 |
| [Copilot](https://github.com/apps/copilot-swe-agent) | 13 |
| [vadimzhukck](https://github.com/vadimzhukck) | 10 |
| [junnplus](https://github.com/junnplus) | 9 |
| [geseq](https://github.com/geseq) | 8 |
| [marcofranssen](https://github.com/marcofranssen) | 8 |
| [140am](https://github.com/140am) | 8 |
| [cranktakular](https://github.com/cranktakular) | 7 |
| [TaltaM](https://github.com/TaltaM) | 6 |
| [dackroyd](https://github.com/dackroyd) | 5 |
| [khcchiu](https://github.com/khcchiu) | 5 |
| [yangrq1018](https://github.com/yangrq1018) | 4 |
| [woshidama323](https://github.com/woshidama323) | 3 |
| [romanornr](https://github.com/romanornr) | 3 |
| [crackcomm](https://github.com/crackcomm) | 3 |
| [azhang](https://github.com/azhang) | 2 |
| [if1live](https://github.com/if1live) | 2 |
| [lozdog245](https://github.com/lozdog245) | 2 |
| [Asalei](https://github.com/Asalei) | 2 |
| [soxipy](https://github.com/soxipy) | 2 |
| [tk42](https://github.com/tk42) | 2 |
| [herenow](https://github.com/herenow) | 2 |
| [mshogin](https://github.com/mshogin) | 2 |
| [andreygrehov](https://github.com/andreygrehov) | 2 |
| [bretep](https://github.com/bretep) | 2 |
| [Christian-Achilli](https://github.com/Christian-Achilli) | 2 |
| [dsinuela-taurus](https://github.com/dsinuela-taurus) | 2 |
| [cornelk](https://github.com/cornelk) | 2 |
| [gam-phon](https://github.com/gam-phon) | 2 |
| [MarkDzulko](https://github.com/MarkDzulko) | 2 |
| [MathieuCesbron](https://github.com/MathieuCesbron) | 2 |
| [aidan-bailey](https://github.com/aidan-bailey) | 1 |
| [tongxiaofeng](https://github.com/tongxiaofeng) | 1 |
| [tonywangcn](https://github.com/tonywangcn) | 1 |
| [varunbhat](https://github.com/varunbhat) | 1 |
| [idealhack](https://github.com/idealhack) | 1 |
| [hannut91](https://github.com/hannut91) | 1 |
| [vyloy](https://github.com/vyloy) | 1 |
| [arttobe](https://github.com/arttobe) | 1 |
| [shoman4eg](https://github.com/shoman4eg) | 1 |
| [cangqiaoyuzhuo](https://github.com/cangqiaoyuzhuo) | 1 |
| [dazi005](https://github.com/dazi005) | 1 |
| [gcmutator](https://github.com/gcmutator) | 1 |
| [gopherorg](https://github.com/gopherorg) | 1 |
| [whilei](https://github.com/whilei) | 1 |
| [yuhangcangqian](https://github.com/yuhangcangqian) | 1 |
| [keeghcet](https://github.com/keeghcet) | 1 |
| [mickychang9](https://github.com/mickychang9) | 1 |
| [phieudu241](https://github.com/phieudu241) | 1 |
| [quantpoet](https://github.com/quantpoet) | 1 |
| [snipesjr](https://github.com/snipesjr) | 1 |
| [snussik](https://github.com/snussik) | 1 |
| [taewdy](https://github.com/taewdy) | 1 |
| [threehonor](https://github.com/threehonor) | 1 |
| [xiiiew](https://github.com/xiiiew) | 1 |
| [youzichuan](https://github.com/youzichuan) | 1 |
| [antonzhukov](https://github.com/antonzhukov) | 1 |
| [blombard](https://github.com/blombard) | 1 |
| [CodeLingoBot](https://github.com/CodeLingoBot) | 1 |
| [CodeLingoTeam](https://github.com/CodeLingoTeam) | 1 |
| [Daanikus](https://github.com/Daanikus) | 1 |
| [daniel-cohen-deltatre](https://github.com/daniel-cohen-deltatre) | 1 |
| [merkeld](https://github.com/merkeld) | 1 |
| [shanhuhai5739](https://github.com/shanhuhai5739) | 1 |
| [DirectX](https://github.com/DirectX) | 1 |
| [dnldd](https://github.com/dnldd) | 1 |
| [Juneezee](https://github.com/Juneezee) | 1 |
| [fclairamb](https://github.com/fclairamb) | 1 |
| [frankzougc](https://github.com/frankzougc) | 1 |
| [gemscng](https://github.com/gemscng) | 1 |
| [Jdpurohit](https://github.com/Jdpurohit) | 1 |
| [jimexist](https://github.com/jimexist) | 1 |
| [lookfirst](https://github.com/lookfirst) | 1 |
| [zeldrinn](https://github.com/zeldrinn) | 1 |
| [roskee](https://github.com/roskee) | 1 |
| [mattkanwisher](https://github.com/mattkanwisher) | 1 |
| [mgravitt](https://github.com/mgravitt) | 1 |
| [mKurrels](https://github.com/mKurrels) | 1 |
| [m1kola](https://github.com/m1kola) | 1 |
| [mortensorensen](https://github.com/mortensorensen) | 1 |
| [Polizo96](https://github.com/Polizo96) | 1 |
| [cavapoo2](https://github.com/cavapoo2) | 1 |
| [idoall](https://github.com/idoall) | 1 |
| [starit](https://github.com/starit) | 1 |
