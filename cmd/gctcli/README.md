# GoCryptoTrader gRPC client

## Background

GoCryptoTrader utilises gRPC for client/server interaction. Authentication is done
by a self signed TLS cert, which only supports connections from localhost and also
through basic authorisation specified by the users config file.

## Usage

GoCryptoTrader must be running with gRPC enabled in order to use  the client features.

```bash
go build or go run main.go
```

For a full list of commands, you can run `gctcli --help`. Alternatively, you can also
visit our [GoCryptoTrader API reference.](https://api.gocryptotrader.app/)

Bash/ZSH autocomplete entries can be found [here](/contrib).
