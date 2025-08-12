# GoCryptoTrader package Apiserver

<img src="/common/gctlogo.png?raw=true" width="350px" height="350px" hspace="70">


[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/engine/apiserver)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)
[![Go Report Card](https://goreportcard.com/badge/github.com/thrasher-corp/gocryptotrader)](https://goreportcard.com/report/github.com/thrasher-corp/gocryptotrader)


This apiserver package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Current Features for Apiserver
+ The API server subsystem is a deprecated service used to host a REST or websocket server to interact with some functions of GoCryptoTrader
+ This subsystem is no longer maintained and it is highly encouraged to interact with GRPC endpoints directly where possible
+ In order to modify the behaviour of the API server subsystem, you can edit the following inside your config file:

### deprecatedRPC

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | If enabled will create a REST server which will listen to commands on the listen address | `true` |
| listenAddress | If enabled will listen for REST requests on this address and return a JSON response | `localhost:9050` |

### websocketRPC

| Config | Description | Example |
| ------ | ----------- | ------- |
| enabled | If enabled will create a REST server which will listen to commands on the listen address | `true` |
| listenAddress | If enabled will listen for requests on this address and return a JSON response | `localhost:9051` |
| connectionLimit | Defines how many connections the websocket RPC server can handle simultanesoly | `1` |
| maxAuthFailures | For authenticated endpoints, the amount of failed attempts allowed before disconnection | `3` |
| allowInsecureOrigin | Allows use of insecure connections | `true` |

## Donations

<img src="https://github.com/thrasher-corp/gocryptotrader/blob/master/web/src/assets/donate.png?raw=true" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

***bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc***
