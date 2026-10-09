package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var orderbookCommonFlags = []cli.Flag{
	&cli.StringFlag{
		Name:     exchangeFlag,
		Required: true,
		Usage:    "the exchange to get the orderbook for",
	},
	&cli.StringFlag{
		Name:     pairFlag,
		Required: true,
		Usage:    "the currency pair to get the orderbook for",
	},
	&cli.StringFlag{
		Name:     assetFlag,
		Required: true,
		Usage:    "the asset type of the currency pair to get the orderbook for",
	},
}

var orderbookCommand = &cli.Command{
	Name:      "orderbook",
	Usage:     "orderbook system simulations and analytics command",
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:        "sell",
			Usage:       "simulates sell to derive orderbook liquidity impact information",
			ArgsUsage:   commandArgsUsage,
			Subcommands: []*cli.Command{nominal, impact, base, quoteRequired},
			Flags:       []cli.Flag{&cli.BoolFlag{Name: "sell", Hidden: true, Value: true}},
		},
		{
			Name:        "buy",
			Usage:       "simulates buy to derive orderbook liquidity impact information",
			ArgsUsage:   commandArgsUsage,
			Subcommands: []*cli.Command{nominal, impact, quote, baseRequired},
		},
		getOrderbookCommand,
		getOrderbooksCommand,
		getOrderbookStreamCommand,
		getExchangeOrderbookStreamCommand,
		whaleBombCommand,
	},
}

var nominal = &cli.Command{
	Name:   "nominal",
	Usage:  "simulates a buy or sell based off the percentage between the reference price and the average order cost",
	Action: getNominal,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:  "percent",
		Usage: "the max percentage slip you wish to occur e.g. 1 = 1% and 100 = 100%. Note: If selling base/hitting the bids you can only have a max value of 100%",
	}),
}

func getNominal(c *cli.Context) error {
	isSelling := c.Bool("sell")
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	var currencyPair string
	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	var assetType string
	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var percentage float64
	if c.IsSet("percent") {
		percentage = c.Float64("percent")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrderbookAmountByNominal(c.Context,
		&gctrpc.GetOrderbookAmountByNominalRequest{
			Exchange: exchangeName,
			Pair: &gctrpc.CurrencyPair{
				Base:  p.Base.String(),
				Quote: p.Quote.String(),
			},
			Asset:             assetType,
			Sell:              isSelling,
			NominalPercentage: percentage,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var impact = &cli.Command{
	Name:   "impact",
	Usage:  "simulates a buy or sell based off the reference price and the orderbook impact slippage",
	Action: getImpact,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     "percent",
		Required: true,
		Usage:    "the max percentage slip you wish to occur e.g. 1 = 1% and 100 = 100%. Note: If selling base/hitting the bids you can only have a max value of 100%",
	}),
}

func getImpact(c *cli.Context) error {
	isSelling := c.Bool("sell")
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	var currencyPair string
	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	var assetType string
	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var percentage float64
	if c.IsSet("percent") {
		percentage = c.Float64("percent")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrderbookAmountByImpact(c.Context,
		&gctrpc.GetOrderbookAmountByImpactRequest{
			Exchange: exchangeName,
			Pair: &gctrpc.CurrencyPair{
				Base:  p.Base.String(),
				Quote: p.Quote.String(),
			},
			Asset:            assetType,
			Sell:             isSelling,
			ImpactPercentage: percentage,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var purchase = &cli.BoolFlag{
	Name:   "purchase",
	Hidden: true,
	Value:  true,
}

var quote = &cli.Command{
	Name:   "quote",
	Usage:  "simulates a buy using quotation amount",
	Action: getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     amountFlag,
		Required: true,
		Usage:    "the amount of quotation currency lifting the asks",
	}),
}

var baseRequired = &cli.Command{
	Name:   "baserequired",
	Usage:  "simulates a buy with a required base amount to be purchased",
	Action: getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     amountFlag,
		Required: true,
		Usage:    "the amount of base currency required to be purchased when lifting the asks",
	}, purchase),
}

var base = &cli.Command{
	Name:   "base",
	Usage:  "simulates a sell using base amount",
	Action: getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     amountFlag,
		Required: true,
		Usage:    "the amount of base currency hitting the bids",
	}),
}

var quoteRequired = &cli.Command{
	Name:   "quoterequired",
	Usage:  "simulates a sell with a required quote amount to be purchased",
	Action: getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     amountFlag,
		Required: true,
		Usage:    "the amount of quotation currency required to be purchased when hitting the bids",
	}, purchase),
}

func getMovement(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	var currencyPair string
	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	var assetType string
	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var amount float64
	if c.IsSet(amountFlag) {
		amount = c.Float64(amountFlag)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrderbookMovement(c.Context, &gctrpc.GetOrderbookMovementRequest{
		Exchange: exchangeName,
		Pair: &gctrpc.CurrencyPair{
			Base:  p.Base.String(),
			Quote: p.Quote.String(),
		},
		Asset:    assetType,
		Sell:     c.Bool("sell"),
		Amount:   amount,
		Purchase: c.Bool("purchase"),
	})
	if err != nil {
		return err
	}

	jsonOutput(result)

	return nil
}

var getOrderbookCommand = &cli.Command{
	Name:   "getorderbook",
	Usage:  "gets the orderbook for a specific currency pair and exchange",
	Action: getOrderbook,
	Flags: append(orderbookCommonFlags,
		&cli.BoolFlag{
			Name:  "exchangestyle",
			Usage: "optional - renders the books like on an exchange website",
		},
		&cli.Int64Flag{
			Name:  "depthlimit",
			Usage: "optional - limit how deep the book rendering is, max 100 - only works if exchangestyle is true",
		}),
}

func getOrderbook(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var (
		exchangeName, pair, assetType string
		exchangeStyle                 bool
		err                           error
	)

	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(pairFlag) {
		pair = c.String(pairFlag)
	}

	if !validPair(pair) {
		return errInvalidPair
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	if c.IsSet("exchangestyle") {
		exchangeStyle = c.Bool("exchangestyle")
	}

	const depthCeiling = 100 // The maximum the depth can be regardless of user entry
	depthLimit := depthCeiling
	if d := c.Int64("depthlimit"); d > 0 && d < depthCeiling {
		depthLimit = int(d)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	p, err := currency.NewPairDelimiter(pair, pairDelimiter)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrderbook(c.Context,
		&gctrpc.GetOrderbookRequest{
			Exchange: exchangeName,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: assetType,
		},
	)
	if err != nil {
		return err
	}

	if exchangeStyle {
		return renderOrderbookExchangeStyle(os.Stdout, result, exchangeName, assetType, depthLimit)
	}
	jsonOutput(result)
	return nil
}

var getOrderbooksCommand = &cli.Command{
	Name:   "getorderbooks",
	Usage:  "gets all orderbooks for all enabled exchanges and currency pairs",
	Action: getOrderbooks,
}

func getOrderbooks(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrderbooks(c.Context, &gctrpc.GetOrderbooksRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getOrderbookStreamCommand = &cli.Command{
	Name:   "getorderbookstream",
	Usage:  "gets the orderbook stream for a specific currency pair and exchange",
	Action: getOrderbookStream,
	Flags: append(orderbookCommonFlags,
		&cli.BoolFlag{
			Name:  "exchangestyle",
			Usage: "optional - renders the books like on an exchange website",
		},
		&cli.Int64Flag{
			Name:  "depthlimit",
			Usage: "optional - limit how deep the book rendering is, max 50",
		}),
}

func getOrderbookStream(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var (
		exchangeName, pair, assetType string
		exchangeStyle                 bool
		err                           error
	)

	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(pairFlag) {
		pair = c.String(pairFlag)
	}

	if !validPair(pair) {
		return errInvalidPair
	}

	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	if c.IsSet("exchangestyle") {
		exchangeStyle = c.Bool("exchangestyle")
	}

	const depthCeiling = 50 // The maximum the depth can be regardless of user entry
	depthLimit := depthCeiling
	if d := c.Int64("depthlimit"); d > 0 && d < depthCeiling {
		depthLimit = int(d)
	}

	assetType = strings.ToLower(assetType)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	p, err := currency.NewPairDelimiter(pair, pairDelimiter)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrderbookStream(c.Context,
		&gctrpc.GetOrderbookStreamRequest{
			Exchange: exchangeName,
			Pair: &gctrpc.CurrencyPair{
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
				Delimiter: p.Delimiter,
			},
			AssetType: assetType,
		},
	)
	if err != nil {
		return err
	}

	for {
		resp, err := result.Recv()
		if err != nil {
			return err
		}

		err = clearScreen()
		if err != nil {
			return err
		}

		if resp.Error != "" {
			fmt.Printf("%s\n", resp.Error)
			continue
		}

		if exchangeStyle {
			err = renderOrderbookExchangeStyle(os.Stdout, resp, exchangeName, assetType, depthLimit)
		} else {
			err = renderOrderbookStream(os.Stdout, resp, exchangeName, depthLimit)
		}
		if err != nil {
			return err
		}
	}
}

type orderbookDisplayLevel struct {
	price, amount string
}

func orderbookDisplayLevels(items []*gctrpc.OrderbookItem, depthLimit int) []orderbookDisplayLevel {
	levels := make([]orderbookDisplayLevel, min(len(items), max(0, depthLimit)))
	for i := range levels {
		levels[i] = orderbookDisplayLevel{
			price:  items[i].StrPrice,
			amount: items[i].StrAmount,
		}
		if levels[i].price == "" {
			levels[i].price = strconv.FormatFloat(items[i].Price, 'f', -1, 64)
		}
		if levels[i].amount == "" {
			levels[i].amount = strconv.FormatFloat(items[i].Amount, 'f', -1, 64)
		}
	}
	return levels
}

func orderbookColumnWidths(bids, asks []orderbookDisplayLevel, priceHeader, amountHeader string) (priceWidth, amountWidth int) {
	priceWidth, amountWidth = len(priceHeader), len(amountHeader)
	for _, side := range [][]orderbookDisplayLevel{bids, asks} {
		for _, level := range side {
			priceWidth = max(priceWidth, len(level.price))
			amountWidth = max(amountWidth, len(level.amount))
		}
	}
	return priceWidth, amountWidth
}

func orderbookDecimalPlaces(value string) int {
	if value == "" {
		return -1
	}
	for _, char := range value {
		if (char < '0' || char > '9') && char != '.' && char != '-' && char != '+' {
			return -1
		}
	}
	if dot := strings.IndexByte(value, '.'); dot >= 0 {
		return len(value) - dot - 1
	}
	return 0
}

func padOrderbookDecimals(value string, precision int) string {
	places := orderbookDecimalPlaces(value)
	// Preserve missing values and non-fixed-point representations rather than
	// appending zeros that would change their meaning.
	if places < 0 || places >= precision {
		return value
	}
	if !strings.Contains(value, ".") {
		value += "."
	}
	return value + strings.Repeat("0", precision-places)
}

func normaliseOrderbookPrecision(bids, asks []orderbookDisplayLevel) {
	var pricePrecision, amountPrecision int
	for _, side := range [][]orderbookDisplayLevel{bids, asks} {
		for _, level := range side {
			pricePrecision = max(pricePrecision, orderbookDecimalPlaces(level.price))
			amountPrecision = max(amountPrecision, orderbookDecimalPlaces(level.amount))
		}
	}
	for _, side := range [][]orderbookDisplayLevel{bids, asks} {
		for i := range side {
			side[i].price = padOrderbookDecimals(side[i].price, pricePrecision)
			side[i].amount = padOrderbookDecimals(side[i].amount, amountPrecision)
		}
	}
}

func writeOrderbookColumns(frame *bytes.Buffer, level orderbookDisplayLevel, priceWidth, amountWidth int) {
	for range priceWidth - len(level.price) {
		frame.WriteByte(' ')
	}
	frame.WriteString(level.price)
	frame.WriteString("  ")
	for range amountWidth - len(level.amount) {
		frame.WriteByte(' ')
	}
	frame.WriteString(level.amount)
}

func writeOrderbookFrame(w io.Writer, frame *bytes.Buffer) error {
	n, err := w.Write(frame.Bytes())
	if err != nil {
		return err
	}
	if n != frame.Len() {
		return io.ErrShortWrite
	}
	return nil
}

func renderOrderbookStream(w io.Writer, resp *gctrpc.OrderbookResponse, exchangeName string, depthLimit int) error {
	bids, asks := orderbookDisplayLevels(resp.Bids, depthLimit), orderbookDisplayLevels(resp.Asks, depthLimit)
	normaliseOrderbookPrecision(bids, asks)
	priceHeader, amountHeader := "Price("+strings.ToUpper(resp.Pair.Quote)+")", "Amount("+strings.ToUpper(resp.Pair.Base)+")"
	priceWidth, amountWidth := orderbookColumnWidths(bids, asks, priceHeader, amountHeader)
	var frame bytes.Buffer
	frame.Grow(256 + (2*(priceWidth+amountWidth+2)+4)*max(len(bids), len(asks)))
	fmt.Fprintf(&frame, "Orderbook stream for %s %s:\n\n", exchangeName, resp.Pair)
	fmt.Fprintf(&frame, "%-*s | Asks\n", priceWidth+amountWidth+2, "Bids")
	fmt.Fprintf(&frame, "%*s  %*s | %*s  %*s\n", priceWidth, priceHeader, amountWidth, amountHeader, priceWidth, priceHeader, amountWidth, amountHeader)
	for i := range max(len(bids), len(asks)) {
		var bid, ask orderbookDisplayLevel
		if i < len(bids) {
			bid = bids[i]
		}
		if i < len(asks) {
			ask = asks[i]
		}
		writeOrderbookColumns(&frame, bid, priceWidth, amountWidth)
		frame.WriteString(" | ")
		writeOrderbookColumns(&frame, ask, priceWidth, amountWidth)
		frame.WriteByte('\n')
	}
	return writeOrderbookFrame(w, &frame)
}

func renderOrderbookExchangeStyle(w io.Writer, resp *gctrpc.OrderbookResponse, exchangeName, assetType string, depthLimit int) error {
	bids, asks := orderbookDisplayLevels(resp.Bids, depthLimit), orderbookDisplayLevels(resp.Asks, depthLimit)
	normaliseOrderbookPrecision(bids, asks)
	upperBase, upperQuote := strings.ToUpper(resp.Pair.Base), strings.ToUpper(resp.Pair.Quote)
	priceHeader, amountHeader := "Price("+upperQuote+")", "Amount("+upperBase+")"
	priceWidth, amountWidth := orderbookColumnWidths(bids, asks, priceHeader, amountHeader)
	var frame bytes.Buffer
	frame.Grow(256 + (len(redText)+priceWidth+amountWidth+3)*(len(bids)+len(asks)))
	fmt.Fprintf(&frame, "%sOrderbook stream for %v %v %v - Last updated %v\n",
		whiteText, strings.ToUpper(exchangeName), assetType, upperBase+"-"+upperQuote, time.UnixMicro(resp.LastUpdated).Format(common.SimpleTimeFormatWithTimezone))
	fmt.Fprintf(&frame, "%s%*s  %*s\n", grayText, priceWidth, priceHeader, amountWidth, amountHeader)
	for i := len(asks); i > 0; i-- {
		frame.WriteString(redText)
		writeOrderbookColumns(&frame, asks[i-1], priceWidth, amountWidth)
		frame.WriteByte('\n')
	}
	fmt.Fprintln(&frame)
	for _, bid := range bids {
		frame.WriteString(greenText)
		writeOrderbookColumns(&frame, bid, priceWidth, amountWidth)
		frame.WriteByte('\n')
	}
	fmt.Fprintln(&frame, defaultText)
	return writeOrderbookFrame(w, &frame)
}

var getExchangeOrderbookStreamCommand = &cli.Command{
	Name:   "getexchangeorderbookstream",
	Usage:  "gets a stream for all orderbooks associated with an exchange",
	Action: getExchangeOrderbookStream,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     exchangeFlag,
			Required: true,
			Usage:    "the exchange to get the orderbook from",
		},
	},
}

func getExchangeOrderbookStream(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangeOrderbookStream(c.Context,
		&gctrpc.GetExchangeOrderbookStreamRequest{
			Exchange: exchangeName,
		})
	if err != nil {
		return err
	}

	for {
		resp, err := result.Recv()
		if err != nil {
			return err
		}

		err = clearScreen()
		if err != nil {
			return err
		}

		fmt.Printf("Orderbook streamed for %s %s at %s", exchangeName, resp.Pair, time.UnixMicro(resp.LastUpdated).Format(common.SimpleTimeFormatWithTimezone))
		if resp.Error != "" {
			fmt.Printf("%s\n", resp.Error)
		}
	}
}

var whaleBombCommand = &cli.Command{
	Name:   "whalebomb",
	Usage:  "whale bomb finds the amount required to reach a price target",
	Action: whaleBomb,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     exchangeFlag,
			Required: true,
			Usage:    "the exchange to whale bomb",
		},
		&cli.StringFlag{
			Name:     pairFlag,
			Required: true,
			Usage:    pairUsage,
		},
		&cli.StringFlag{
			Name:     sideFlag,
			Required: true,
			Usage:    "the order side to use (BUY OR SELL)",
		},
		&cli.StringFlag{
			Name:     assetFlag,
			Required: true,
			Usage:    "the asset type of the currency pair to get the orderbook for",
		},
		&cli.Float64Flag{
			Name:  "price",
			Usage: "the price target",
		},
	},
}

func whaleBomb(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var currencyPair string
	var orderSide string
	var price float64

	if c.IsSet(exchangeFlag) {
		exchangeName = c.String(exchangeFlag)
	}

	if c.IsSet(pairFlag) {
		currencyPair = c.String(pairFlag)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	if c.IsSet(sideFlag) {
		orderSide = c.String(sideFlag)
	}

	if orderSide == "" {
		return errors.New("order side must be set")
	}

	var assetType string
	if c.IsSet(assetFlag) {
		assetType = c.String(assetFlag)
	}

	if c.IsSet("price") {
		price = c.Float64("price")
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WhaleBomb(c.Context, &gctrpc.WhaleBombRequest{
		Exchange: exchangeName,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:        orderSide,
		PriceTarget: price,
		AssetType:   assetType,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
