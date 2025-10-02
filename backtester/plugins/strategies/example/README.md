# GoCryptoTrader Backtester: Example package

<img src="/backtester/common/backtester.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/backtester/plugins/strategies/example)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This example package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Example package overview

This is a custom strategy for the GoCryptoTrader Backtester. It is a simple example of a strategy that trades a pair of assets and is used to highlight how strategies can be loaded from external sources.

### Designing a strategy
- File must contain `main` package.
- Custom strategy plugins must adhere to the strategy.Handler interface. See the [strategy.Handler interface documentation](./backtester/eventhandlers/strategies/README.md) for more information.
- Must contain function `func GetStrategies() []strategy.Handler` to return a slice of implemented `strategy.Handler`.
   - If only using one custom strategy, can simply `return []strategy.Handler{&customStrategy{}}`.

### Building
See [here](./backtester/plugins/README.md) for details on how to build the plugin file.

### Running
Plugins can only be loaded via Linux, macOS and WSL. Windows itself is not supported.

To run this strategy you will need to use the following flags when running the GoCryptoTrader Backtester:

```bash
./backtester -strategypluginpath="path/to/strategy/example.so"
```

To run this specific example strategy, use:

```bash
./backtester --strategypluginpath="./plugins/strategies/example/example.so"
```

Upon startup, the GoCryptoTrader Backtester will load the strategy and run it for all events.

## Donations

<img src="/docs/assets/donate.png" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
