# Multichain transfer support

Several exchanges support deposits and withdrawals by other blockchain networks. An example would be Tether (USDT) which supports the ERC20 (Ethereum), TRC20 (Tron) and Omni (BTC) networks.

GoCryptoTrader contains a `GetAvailableTransferChains` exchange method for supported exchanges which returns a list of the supported transfer chains specified by a cryptocurrency.

A simple demonstration using `gctcli` is as follows:

## Obtaining a list of supported transfer chains

```sh
$ ./gctcli getavailabletransferchains --exchange=binance --cryptocurrency=usdt
{
 "chains": [
  "erc20",
  "trx",
  "sol",
  "omni"
 ]
}
```

## Obtaining a deposit address based on a specific cryptocurrency and chain

```sh
$ ./gctcli getcryptocurrencydepositaddress --exchange=binance --cryptocurrency=usdt --chain=sol
{
 "address": "GW3oT9JpFyTkCWPnt6Yw9ugppSQwDv4ZMG1vabC8WmHS"
}
```

## Withdrawing

```sh
$ ./gctcli withdrawcryptofunds --exchange=binance --currency=USDT --address=TJU9piX2WA8WTvxVKMqpvTzZGhvXQAZKSY --amount=10 --chain=trx
{
 "id": "01234567-0000-0000-0000-000000000000",
}
```

## Exchange multichain transfer support table

| Exchange | Deposits | Withdrawals | Notes|
|----------|----------|-------------|------|
| Binance.US | Yes  | Yes        | | 
| Binance | Yes | Yes | |
| Bitfinex | Yes | Yes | Only supports USDT |
| Bitflyer | No | No | |
| Bithumb | No | No | |
| BitMEX | No | No | Supports BTC only |
| Bitstamp | No | No | |
| BTCMarkets | No | No| NA  |
| BTSE | No | No | Only through website |
| Bybit | Yes | Yes | |
| Coinbase | No | No | No|
| COINUT | No | No | NA |
| Deribit | Yes | Yes | |
| Exmo | Yes | Yes | Addresses must be created via their website first |
| GateIO | Yes | Yes | |
| Gemini | No | No | |
| HitBTC | No | No | |
| Huobi.Pro | Yes | Yes | |
| Kraken | Yes | Yes | Front-end and API don't match total available transfer chains |
| Kucoin |  Yes | Yes | |
| Lbank | No | No | |
| Okx | Yes | Yes | |
| Poloniex | Yes | Yes | |
| Yobit | No | No | |
