# GoCryptoTrader File Hierarchy

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

## Default data directory

By default, GoCryptoTrader uses the following data directories:

Operating System | Path | Translated
--- | --- | ----
| Windows | %APPDATA%\GoCryptoTrader | C:\Users\User\AppData\Roaming\GoCryptoTrader
| Linux | ~/.gocryptotrader | /home/user/.gocryptotrader
| macOS | ~/.gocryptotrader | /Users/User/.gocryptotrader

This can be overridden by running GoCryptoTrader with the `-datadir` command line
parameter.

## Subdirectories

Depending on the features enabled, you'll see the following directories created
inside the data directory:

Directory | Reason
--- | ---
| database | Used to store the database file (if using SQLite3) and sqlboiler config files
| logs | Used to store the debug log file (`log.txt` by default), if file output and logging is enabled
| tls | Used to store the generated self-signed certificate and key for gRPC authentication

## Files

File | Reason
--- | ---
config.json or config.dat (encrypted config) | Config file which GoCryptoTrader loads from (can be overridden by the `-config` command line parameter).
currency.json | Cached list of fiat and digital currencies
