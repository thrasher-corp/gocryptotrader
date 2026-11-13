# GoCryptoTrader package Adanos

<img src="../../common/gctlogo.png" alt="GoCryptoTrader logo" width="350px" height="350px" hspace="70">

[![Build Status](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml/badge.svg?branch=master)](https://github.com/thrasher-corp/gocryptotrader/actions/workflows/tests.yml)
[![Software License](https://img.shields.io/badge/License-MIT-orange.svg?style=flat-square)](https://github.com/thrasher-corp/gocryptotrader/blob/master/LICENSE)
[![GoDoc](https://godoc.org/github.com/thrasher-corp/gocryptotrader?status.svg)](https://godoc.org/github.com/thrasher-corp/gocryptotrader/currency/adanos)
[![Coverage Status](https://codecov.io/gh/thrasher-corp/gocryptotrader/graph/badge.svg?token=41784B23TS)](https://codecov.io/gh/thrasher-corp/gocryptotrader)

This adanos package is part of the GoCryptoTrader codebase.

## This is still in active development

You can track ideas, planned features and what's in progress on our [GoCryptoTrader Kanban board](https://github.com/orgs/thrasher-corp/projects/3).

Join our slack to discuss all things related to GoCryptoTrader! [GoCryptoTrader Slack](https://join.slack.com/t/gocryptotrader/shared_invite/zt-38z8abs3l-gH8AAOk8XND6DP5NfCiG_g)

## Library-only Adanos client

[Adanos](https://adanos.org/) supplies Reddit crypto sentiment and discussion
metrics through a commercial API. This optional client requires an authorised
API key. A limited free plan is available. See the [API documentation](https://api.adanos.com/docs).

This package is library-only, not an engine integration. It does not register a
forex provider, start polling, supply market prices, or change trading behaviour.
Constructing a client performs no network I/O. Only `GetToken` sends a request.

### Fetch a token report

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/thrasher-corp/gocryptotrader/currency/adanos"
)

func main() {
	client, err := adanos.New(os.Getenv("ADANOS_API_KEY"))
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		if err := client.Shutdown(); err != nil {
			log.Print(err)
		}
	}()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	report, err := client.GetToken(ctx, &adanos.TokenRequest{
		Symbol: "BTC",
	})
	if err != nil {
		log.Print(err)
		return
	}
	fmt.Println(report.Symbol, report.Found)
	if report.SentimentScore != nil {
		fmt.Println(*report.SentimentScore)
	}
}
```

`TokenRequest.From` and `TokenRequest.To` are optional `time.Time` values. Each
non-zero bound becomes an inclusive UTC `YYYY-MM-DD` date. Equal UTC dates are
valid. Zero bounds are omitted and leave the API's default window in effect.
Symbols accept an optional leading `$` and one to twenty ASCII letters or digits.

`GetToken` calls only `GET /reddit/crypto/v1/token/{symbol}` at
`https://api.adanos.com`. Reuse the client across calls. Each call makes at most
one request, including on HTTP 429. Requests have a 30-second timeout and honour
context cancellation and earlier deadlines. Redirects are rejected to protect
the key. The client does not manage account quotas or schedule retries.
The client's private transport uses HTTP/1 only with keep-alives disabled. This
prevents transparent GET replay after a reused connection closes or an HTTP/2
stream is refused, at the cost of HTTP/2 support and connection reuse. Each
request opens a new connection.

After all in-flight requests finish and no callers are using the client, call
`Shutdown` once to close idle connections and release its native requester
registration. Do not use the client during or after shutdown. Shutdown does not
wait for active calls or cancel them.

### Interpret a report

The typed response retains every field in the selected v1 token schema. Nullable
scalars use pointers. Nil does not mean zero or neutral sentiment. Daily dates
remain date-only strings, and mention timestamps use `time.Time`. Reddit net
upvotes are signed and may be negative. `Trend` describes discussion activity,
not price movement.

The deprecated v1 `Found` field remains available. HTTP 200 with `Found == false`
means the symbol is supported but has no qualifying data for the requested
window. An unsupported-symbol HTTP 404 is an error. Deprecated v1 count fields
and the subreddit `Count` alias also remain available for compatibility.

HTTP failures preserve `request.ErrBadStatus` for `errors.Is`. Context and
transport errors retain their causes. Surrounding API key whitespace is trimmed
after empty-key and CR/LF validation. Returned error text redacts the normalised
key, including a key echoed by the server. Validation errors are package
sentinels. Live endpoint compatibility has not been verified. The fixtures are
synthetic and contain no redistributed provider data. This client makes no
point-in-time availability, investment return, or performance guarantees.

### Testing

The package tests use the native requester and local HTTP and TLS servers. They need no
credentials or provider access.

```sh
go test -race -count=1 ./currency/adanos
go test -race -count=1 -tags=sonic_on ./currency/adanos
```

## Donations

<img src="../../docs/assets/donate.png" alt="Donate to GoCryptoTrader" hspace="70">

If this framework helped you in any way, or you would like to support the developers working on it, please donate Bitcoin to:

`bc1qk0jareu4jytc0cfrhr5wgshsq8282awpavfahc`
