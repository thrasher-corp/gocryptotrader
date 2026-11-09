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
	"unicode"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
	"golang.org/x/text/width"
)

var orderbookCommonFlags = []cli.Flag{
	requiredExchangeFlag,
	requiredPairFlag,
	requiredAssetFlag,
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
			Flags: []cli.Flag{&cli.BoolFlag{
				Name: "sell", Hidden: true, Value: true,
			}},
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
		Name:  percentFlagName,
		Usage: percentUsage,
	}),
}

func getNominal(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
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
	result, err := client.GetOrderbookAmountByNominal(c.Context,
		&gctrpc.GetOrderbookAmountByNominalRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Base:  p.Base.String(),
				Quote: p.Quote.String(),
			},
			Asset:             assetType,
			Sell:              c.Bool("sell"),
			NominalPercentage: c.Float64(percentFlagName),
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
		Name:     percentFlagName,
		Required: true,
		Usage:    percentUsage,
	}),
}

func getImpact(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
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
	result, err := client.GetOrderbookAmountByImpact(c.Context,
		&gctrpc.GetOrderbookAmountByImpactRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Base:  p.Base.String(),
				Quote: p.Quote.String(),
			},
			Asset:            assetType,
			Sell:             c.Bool("sell"),
			ImpactPercentage: c.Float64(percentFlagName),
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
		Name:     amountFlagName,
		Required: true,
		Usage:    "the amount of quotation currency lifting the asks",
	}),
}

var baseRequired = &cli.Command{
	Name:   "baserequired",
	Usage:  "simulates a buy with a required base amount to be purchased",
	Action: getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     amountFlagName,
		Required: true,
		Usage:    "the amount of base currency required to be purchased when lifting the asks",
	}, purchase),
}

var base = &cli.Command{
	Name:   "base",
	Usage:  "simulates a sell using base amount",
	Action: getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     amountFlagName,
		Required: true,
		Usage:    "the amount of base currency hitting the bids",
	}),
}

var quoteRequired = &cli.Command{
	Name:   "quoterequired",
	Usage:  "simulates a sell with a required quote amount to be purchased",
	Action: getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:     amountFlagName,
		Required: true,
		Usage:    "the amount of quotation currency required to be purchased when hitting the bids",
	}, purchase),
}

func getMovement(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
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
	result, err := client.GetOrderbookMovement(c.Context, &gctrpc.GetOrderbookMovementRequest{
		Exchange: c.String(exchangeFlagName),
		Pair: &gctrpc.CurrencyPair{
			Base:  p.Base.String(),
			Quote: p.Quote.String(),
		},
		Asset:    assetType,
		Sell:     c.Bool("sell"),
		Amount:   c.Float64(amountFlagName),
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
			Name:  exchangeStyleFlagName,
			Usage: exchangeStyleUsage,
		},
		&cli.UintFlag{
			Name:  depthLimitFlagName,
			Usage: "optional - limit how deep the book rendering is, max 100 - only works if exchangestyle is true",
		}),
}

func getOrderbook(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	exchangeName := c.String(exchangeFlagName)
	pair := c.String(pairFlagName)

	if !validPair(pair) {
		return errInvalidPair
	}

	assetType := c.String(assetFlagName)

	const depthCeiling = 100                                // The maximum the depth can be regardless of user entry
	depthLimit := int(orderbookDepthLimit(c, depthCeiling)) // Clamped to 100 before converting for slice-based rendering.

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

	if c.Bool(exchangeStyleFlagName) {
		return renderOrderbookExchangeStyle(os.Stdout, result, exchangeName, assetType, depthLimit, false)
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
			Name:  exchangeStyleFlagName,
			Usage: exchangeStyleUsage,
		},
		&cli.UintFlag{
			Name:  depthLimitFlagName,
			Usage: "optional - limit how deep the book rendering is, max 50",
		}),
}

func getOrderbookStream(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	exchangeName := c.String(exchangeFlagName)
	pair := c.String(pairFlagName)

	if !validPair(pair) {
		return errInvalidPair
	}

	assetType := c.String(assetFlagName)

	const depthCeiling = 50                                 // The maximum the depth can be regardless of user entry
	depthLimit := int(orderbookDepthLimit(c, depthCeiling)) // Clamped to 50 before converting for slice-based rendering.

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

		if err := clearScreen(); err != nil {
			return err
		}

		if resp.Error != "" {
			fmt.Printf("%s\n", resp.Error)
			continue
		}

		if c.Bool(exchangeStyleFlagName) {
			err = renderOrderbookExchangeStyle(os.Stdout, resp, exchangeName, assetType, depthLimit, true)
		} else {
			err = renderOrderbookStream(os.Stdout, resp, exchangeName, depthLimit)
		}
		if err != nil {
			return err
		}
	}
}

func orderbookDepthLimit(c *cli.Context, ceiling uint) uint {
	if d := c.Uint(depthLimitFlagName); d > 0 && d < ceiling {
		return d
	}
	return ceiling
}

// orderbookTextWidth measures terminal columns rather than bytes or runes.
// Exchange currency codes can contain wide glyphs and combining marks.
func orderbookTextWidth(text string) int {
	var columns int
	for _, char := range text {
		if char < 128 {
			columns++
			continue
		}
		if unicode.Is(unicode.M, char) || unicode.Is(unicode.Cf, char) {
			continue
		}
		switch width.LookupRune(char).Kind() {
		case width.EastAsianWide, width.EastAsianFullwidth:
			columns += 2
		default:
			columns++
		}
	}
	return columns
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
	priceWidth, amountWidth = orderbookTextWidth(priceHeader), orderbookTextWidth(amountHeader)
	for _, side := range [][]orderbookDisplayLevel{bids, asks} {
		for _, level := range side {
			priceWidth = max(priceWidth, orderbookTextWidth(level.price))
			amountWidth = max(amountWidth, orderbookTextWidth(level.amount))
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
	for range priceWidth - orderbookTextWidth(level.price) {
		frame.WriteByte(' ')
	}
	frame.WriteString(level.price)
	frame.WriteString("  ")
	for range amountWidth - orderbookTextWidth(level.amount) {
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
	fmt.Fprintf(&frame, "Orderbook stream for %s %s-%s:\n\n", exchangeName, strings.ToUpper(resp.Pair.Base), strings.ToUpper(resp.Pair.Quote))
	fmt.Fprintf(&frame, "%-*s | Asks\n", priceWidth+amountWidth+2, "Bids")
	headings := orderbookDisplayLevel{price: priceHeader, amount: amountHeader}
	writeOrderbookColumns(&frame, headings, priceWidth, amountWidth)
	frame.WriteString(" | ")
	writeOrderbookColumns(&frame, headings, priceWidth, amountWidth)
	frame.WriteByte('\n')
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

func renderOrderbookExchangeStyle(w io.Writer, resp *gctrpc.OrderbookResponse, exchangeName, assetType string, depthLimit int, includeLastUpdated bool) error {
	bids, asks := orderbookDisplayLevels(resp.Bids, depthLimit), orderbookDisplayLevels(resp.Asks, depthLimit)
	normaliseOrderbookPrecision(bids, asks)
	upperBase, upperQuote := strings.ToUpper(resp.Pair.Base), strings.ToUpper(resp.Pair.Quote)
	priceHeader, amountHeader := "Price("+upperQuote+")", "Amount("+upperBase+")"
	priceWidth, amountWidth := orderbookColumnWidths(bids, asks, priceHeader, amountHeader)
	var frame bytes.Buffer
	frame.Grow(256 + (len(redText)+priceWidth+amountWidth+3)*(len(bids)+len(asks)))
	fmt.Fprintf(&frame, "%sOrderbook stream for %v %v %v", whiteText, strings.ToUpper(exchangeName), assetType, upperBase+"-"+upperQuote)
	// Only stream RPCs guarantee microseconds; snapshots use daemon-configured units.
	if includeLastUpdated {
		fmt.Fprintf(&frame, " - Last updated %s", time.UnixMicro(resp.LastUpdated).Format(common.SimpleTimeFormatWithTimezone))
	}
	frame.WriteByte('\n')
	frame.WriteString(grayText)
	writeOrderbookColumns(&frame, orderbookDisplayLevel{price: priceHeader, amount: amountHeader}, priceWidth, amountWidth)
	frame.WriteByte('\n')
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

func renderExchangeOrderbookStream(w io.Writer, resp *gctrpc.OrderbookResponse, exchangeName string) error {
	// Exchange stream errors carry no pair; the caller prints resp.Error.
	if resp.Pair == nil {
		return nil
	}
	_, err := fmt.Fprintf(w, "Orderbook streamed for %s %s-%s at %s",
		exchangeName, strings.ToUpper(resp.Pair.Base), strings.ToUpper(resp.Pair.Quote), time.UnixMicro(resp.LastUpdated).Format(common.SimpleTimeFormatWithTimezone))
	return err
}

var getExchangeOrderbookStreamCommand = &cli.Command{
	Name:   "getexchangeorderbookstream",
	Usage:  "gets a stream for all orderbooks associated with an exchange",
	Action: getExchangeOrderbookStream,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

func getExchangeOrderbookStream(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	exchangeName := c.String(exchangeFlagName)

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

		if err := clearScreen(); err != nil {
			return err
		}

		if err := renderExchangeOrderbookStream(os.Stdout, resp, exchangeName); err != nil {
			return err
		}
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
		requiredExchangeFlag,
		requiredPairFlag,
		requiredSideFlag,
		requiredAssetFlag,
		&cli.Float64Flag{
			Name:  priceFlagName,
			Usage: "the price target",
		},
	},
}

func whaleBomb(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	orderSide := c.String(sideFlagName)

	if orderSide == "" {
		return errors.New("order side must be set")
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
		Exchange: c.String(exchangeFlagName),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:        orderSide,
		PriceTarget: c.Float64(priceFlagName),
		AssetType:   c.String(assetFlagName),
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}
