A cryptocurrency trading bot supporting multiple exchanges written in Golang. 

**Please note that this bot is under development and is not ready for production!**

## Exchange Support Table

| Exchange | REST API | Streaming API | FIX API |
|----------|------|-----------|-----|
| Alphapoint | Yes  | Yes        | NA  |
| ANXPRO | Yes  | No        | NA  |
| Bitfinex | Yes  | NA        | NA  |
| Bitstamp | Yes  | Yes       | NA  |
| BTCChina | Yes  | Yes     | No  |
| BTCE     | Yes  | NA        | NA  |
| BTCMarkets | Yes | NA       | NA  |
| Coinbase | Yes | Yes | No|
| Cryptsy | Yes | Yes | NA|
| DWVX | Yes  | Yes        | NA  |
| Huobi | Yes | Yes |No
| ItBit | Yes | NA | NA |
| Kraken | Yes | NA | NA
| LakeBTC | Yes | Yes | NA
|OKCoin (both) | Yes | Yes | No

** NA means not applicable as the Exchange does not support the feature.

## Current Features
+ Support for all Exchange fiat and digital currencies, with the ability to individually toggle them on/off.
+ REST API support for all exchanges.
+ Websocket support for applicable exchanges.
+ Ability to turn off/on certain exchanges.
+ Ability to adjust manual polling timer for exchanges.
+ SMS notification support via SMS Gateway.
+ Basic event trigger system.

## Planned Features
+ WebGUI.
+ FIX support.
+ Expanding event trigger system.
+ TALib.
+ Trade history summary generation for tax purposes.

Please feel free to submit any pull requests or suggest any desired features to be added.
