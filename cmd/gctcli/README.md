# GoCryptoTrader gRPC client

<img src="../../docs/assets/page-logo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/cmd/gctcli)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

A cryptocurrency trading bot supporting multiple exchanges written in Golang.

**Please note that this bot is under development and is not ready for production!**

## Community

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Background

GoCryptoTrader utilises gRPC for client/server interaction. Authentication is done
by a self signed TLS cert, which only supports connections from localhost and also
through basic authorisation specified by the users config file.

## Usage

GoCryptoTrader must be running with gRPC enabled in order to use the client features.

```bash
go build or go run .
```

For a full list of commands, you can run `gctcli --help`. Alternatively, you can also
visit our [GoCryptoTrader API reference](https://api.gocryptotrader.app/).

Supply command parameters either entirely as positional arguments or entirely as
named flags. Mixing the two forms is rejected, including flags placed after
positional arguments. This applies to all commands and subcommands.

```bash
gctcli getticker Binance BTC-USDT spot
gctcli getticker --exchange Binance --pair BTC-USDT --asset spot
```

Global options are separate from command parameters. **You can use global options
with positional command arguments or with named command flags.** Place global
options, such as `--rpchost`, before the command name:

```bash
# Allowed: global option with positional command arguments
gctcli --rpchost localhost:9052 getticker Binance BTC-USDT spot

# Allowed: global option with named command flags
gctcli --rpchost localhost:9052 getticker --exchange Binance --pair BTC-USDT --asset spot

# Rejected: a named command flag mixed with positional command arguments
gctcli --rpchost localhost:9052 getticker --exchange Binance BTC-USDT spot
```

Negative numeric positional values remain supported. Use `--` before positional
values that begin with a dash and should be treated literally.

If `cancelallorders` completes some cancellations before a later batch fails, the
CLI prints the retained order statuses as JSON and still exits with an error.
Inspect those results before retrying the request.

## Autocomplete

Bash/ZSH autocomplete entries are available in the [contrib directory](../../contrib).
