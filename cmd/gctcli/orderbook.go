package main

import (
	"errors"
	"fmt"
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
		Name:  "exchange",
		Usage: "the exchange to get the orderbook for",
	},
	&cli.StringFlag{
		Name:  "pair",
		Usage: "the currency pair to get the orderbook for",
	},
	&cli.StringFlag{
		Name:  "asset",
		Usage: "the asset type of the currency pair to get the orderbook for",
	},
}

var orderbookCommand = &cli.Command{
	Name:      "orderbook",
	Usage:     "orderbook system simulations and analytics command",
	ArgsUsage: "<command> <args>",
	Subcommands: []*cli.Command{
		{
			Name:        "sell",
			Usage:       "simulates sell to derive orderbook liquidity impact information",
			ArgsUsage:   "<command> <args>",
			Subcommands: []*cli.Command{nominal, impact, base, quoteRequired},
			Flags:       []cli.Flag{&cli.BoolFlag{Name: "sell", Hidden: true, Value: true}},
		},
		{
			Name:        "buy",
			Usage:       "simulates buy to derive orderbook liquidity impact information",
			ArgsUsage:   "<command> <args>",
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
	Name:      "nominal",
	Usage:     "simulates a buy or sell based off the percentage between the reference price and the average order cost",
	ArgsUsage: "<exchange> <pair> <asset> <percent>",
	Action:    getNominal,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:  "percent",
		Usage: "the max percentage slip you wish to occur e.g. 1 = 1% and 100 = 100%. Note: If selling base/hitting the bids you can only have a max value of 100%",
	}),
}

func getNominal(c *cli.Context) error {
	isSelling := c.Bool("sell")
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var currencyPair string
	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(1)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
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
	if c.IsSet("asset") {
		percentage = c.Float64("percent")
	} else {
		percentage, _ = strconv.ParseFloat(c.Args().Get(3), 64)
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
	Name:      "impact",
	Usage:     "simulates a buy or sell based off the reference price and the orderbook impact slippage",
	ArgsUsage: "<exchange> <pair> <asset> <percent>",
	Action:    getImpact,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:  "percent",
		Usage: "the max percentage slip you wish to occur e.g. 1 = 1% and 100 = 100%. Note: If selling base/hitting the bids you can only have a max value of 100%",
	}),
}

func getImpact(c *cli.Context) error {
	isSelling := c.Bool("sell")
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var currencyPair string
	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(1)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
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
	if c.IsSet("asset") {
		percentage = c.Float64("percent")
	} else {
		percentage, _ = strconv.ParseFloat(c.Args().Get(3), 64)
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
	Name:      "quote",
	Usage:     "simulates a buy using quotation amount",
	ArgsUsage: "<exchange> <pair> <asset> <amount>",
	Action:    getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:  "amount",
		Usage: "the amount of quotation currency lifting the asks",
	}),
}

var baseRequired = &cli.Command{
	Name:      "baserequired",
	Usage:     "simulates a buy with a required base amount to be purchased",
	ArgsUsage: "<exchange> <pair> <asset> <amount>",
	Action:    getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:  "amount",
		Usage: "the amount of base currency required to be purchased when lifting the asks",
	}, purchase),
}

var base = &cli.Command{
	Name:      "base",
	Usage:     "simulates a sell using base amount",
	ArgsUsage: "<exchange> <pair> <asset> <amount>",
	Action:    getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:  "amount",
		Usage: "the amount of base currency hitting the bids",
	}),
}

var quoteRequired = &cli.Command{
	Name:      "quoterequired",
	Usage:     "simulates a sell with a required quote amount to be purchased",
	ArgsUsage: "<exchange> <pair> <asset> <amount>",
	Action:    getMovement,
	Flags: append(orderbookCommonFlags, &cli.Float64Flag{
		Name:  "amount",
		Usage: "the amount of quotation currency required to be purchased when hitting the bids",
	}, purchase),
}

func getMovement(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var currencyPair string
	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(1)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
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
	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else {
		amount, _ = strconv.ParseFloat(c.Args().Get(3), 64)
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
	Name:      "getorderbook",
	Usage:     "gets the orderbook for a specific currency pair and exchange",
	ArgsUsage: "<exchange> <pair> <asset> <exchangestyle> <depthlimit>",
	Action:    getOrderbook,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var (
		exchangeName, pair, assetType string
		exchangeStyle                 bool
		err                           error
	)

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("pair") {
		pair = c.String("pair")
	} else {
		pair = c.Args().Get(1)
	}

	if !validPair(pair) {
		return errInvalidPair
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	if c.IsSet("exchangestyle") {
		exchangeStyle = c.Bool("exchangestyle")
	} else if c.Args().Get(3) != "" {
		exchangeStyle, err = strconv.ParseBool(c.Args().Get(3))
		if err != nil {
			return err
		}
	}

	const depthCeiling uint64 = 100 // The maximum the depth can be regardless of user entry
	depthLimit := depthCeiling
	if d := c.Uint64("depthlimit"); d > 0 && d < depthCeiling {
		depthLimit = d
	} else if d := c.Args().Get(4); d != "" {
		if du, err := strconv.ParseUint(d, 10, 64); err != nil {
			return err
		} else if du > 0 && du < depthCeiling {
			depthLimit = du
		}
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
		bidLen := uint64(len(result.Bids) - 1) //nolint:gosec // Can fit in uint64
		askLen := uint64(len(result.Asks) - 1) //nolint:gosec // Can fit in uint64
		maxLen := min(max(bidLen, askLen), depthLimit)
		renderOrderbookExchangeStyle(result, exchangeName, assetType, maxLen, askLen, bidLen)
	} else {
		jsonOutput(result)
	}
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
	Name:      "getorderbookstream",
	Usage:     "gets the orderbook stream for a specific currency pair and exchange",
	ArgsUsage: "<exchange> <pair> <asset> <exchangestyle> <depthlimit>",
	Action:    getOrderbookStream,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var (
		exchangeName, pair, assetType string
		exchangeStyle                 bool
		err                           error
	)

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("pair") {
		pair = c.String("pair")
	} else {
		pair = c.Args().Get(1)
	}

	if !validPair(pair) {
		return errInvalidPair
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	if c.IsSet("exchangestyle") {
		exchangeStyle = c.Bool("exchangestyle")
	} else if c.Args().Get(3) != "" {
		exchangeStyle, err = strconv.ParseBool(c.Args().Get(3))
		if err != nil {
			return err
		}
	}

	const depthCeiling uint64 = 50 // The maximum the depth can be regardless of user entry
	depthLimit := depthCeiling
	if d := c.Uint64("depthlimit"); d > 0 && d < depthCeiling {
		depthLimit = d
	} else if d := c.Args().Get(4); d != "" {
		if du, err := strconv.ParseUint(d, 10, 64); err != nil {
			return err
		} else if du > 0 && du < depthCeiling {
			depthLimit = du
		}
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

		bidLen := uint64(len(resp.Bids) - 1) //nolint:gosec // Can fit in uint64
		askLen := uint64(len(resp.Asks) - 1) //nolint:gosec // Can fit in uint64
		maxLen := min(max(bidLen, askLen), depthLimit)

		if exchangeStyle {
			renderOrderbookExchangeStyle(resp, exchangeName, assetType, maxLen, askLen, bidLen)
		} else {
			fmt.Printf("Orderbook stream for %s %s:\n\n", exchangeName, resp.Pair)
			fmt.Println("\t\tBids\t\t\t\tAsks")
			fmt.Println()

			for i := range maxLen {
				var bidAmount, bidPrice float64
				if i <= bidLen {
					bidAmount = resp.Bids[i].Amount
					bidPrice = resp.Bids[i].Price
				}

				var askAmount, askPrice float64
				if i <= askLen {
					askAmount = resp.Asks[i].Amount
					askPrice = resp.Asks[i].Price
				}

				fmt.Printf("%.8f %s @ %.8f %s\t\t%.8f %s @ %.8f %s\n",
					bidAmount,
					resp.Pair.Base,
					bidPrice,
					resp.Pair.Quote,
					askAmount,
					resp.Pair.Base,
					askPrice,
					resp.Pair.Quote)
			}
		}
	}
}

func renderOrderbookExchangeStyle(resp *gctrpc.OrderbookResponse, exchangeName, assetType string, maxLen, askLen, bidLen uint64) {
	maxLen-- // ensure we get the 0 index at the correct max length
	upperBase := strings.ToUpper(resp.Pair.Base)
	upperQuote := strings.ToUpper(resp.Pair.Quote)
	printFmt := "%s%.8f\t\t%.8f\n"
	fmt.Printf("%sOrderbook stream for %v %v %v - Last updated %v\n",
		whiteText, strings.ToUpper(exchangeName), assetType, upperBase+"-"+upperQuote, time.UnixMicro(resp.LastUpdated).Format(common.SimpleTimeFormatWithTimezone))

	fmt.Printf("%sPrice(%v)\t\tAmount(%s)\n",
		grayText, upperQuote, upperBase)
	for i := uint64(0); i <= maxLen; i++ {
		j := maxLen - i
		var askAmount, askPrice float64
		if j <= askLen {
			askAmount = resp.Asks[j].Amount
			askPrice = resp.Asks[j].Price
		}
		fmt.Printf(printFmt, redText, askPrice, askAmount)
	}
	fmt.Println()
	for i := uint64(0); i <= maxLen; i++ {
		var bidAmount, bidPrice float64
		if i <= bidLen {
			bidAmount = resp.Bids[i].Amount
			bidPrice = resp.Bids[i].Price
		}
		fmt.Printf(printFmt, greenText, bidPrice, bidAmount)
	}
	fmt.Println(defaultText)
}

var getExchangeOrderbookStreamCommand = &cli.Command{
	Name:      "getexchangeorderbookstream",
	Usage:     "gets a stream for all orderbooks associated with an exchange",
	ArgsUsage: "<exchange>",
	Action:    getExchangeOrderbookStream,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the orderbook from",
		},
	},
}

func getExchangeOrderbookStream(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
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
	Name:      "whalebomb",
	Usage:     "whale bomb finds the amount required to reach a price target",
	ArgsUsage: "<exchange> <pair> <side> <asset> <price>",
	Action:    whaleBomb,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to whale bomb",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair",
		},
		&cli.StringFlag{
			Name:  "side",
			Usage: "the order side to use (BUY OR SELL)",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type of the currency pair to get the orderbook for",
		},
		&cli.Float64Flag{
			Name:  "price",
			Usage: "the price target",
		},
	},
}

func whaleBomb(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var currencyPair string
	var orderSide string
	var price float64

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(1)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	if c.IsSet("side") {
		orderSide = c.String("side")
	} else {
		orderSide = c.Args().Get(2)
	}

	if orderSide == "" {
		return errors.New("order side must be set")
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(3)
	}

	if c.IsSet("price") {
		price = c.Float64("price")
	} else if c.Args().Get(4) != "" {
		var err error
		price, err = strconv.ParseFloat(c.Args().Get(4), 64)
		if err != nil {
			return err
		}
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
