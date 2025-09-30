# GoCryptoTrader gRPC client

<img src="/docs/assets/page-logo.png" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)

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
visit our [GoCryptoTrader API reference.](https://api.gocryptotrader.app/)

## Autocomplete

Bash/ZSH autocomplete entries can be found [here](/contrib).
