package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/currency"
	"github.com/thrasher-/gocryptotrader/gctrpc"
	"github.com/urfave/cli"
)

var getInfoCommand = cli.Command{
	Name:   "getinfo",
	Usage:  "gets GoCryptoTrader info",
	Action: getInfo,
}

func getInfo(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetInfo(context.Background(),
		&gctrpc.GetInfoRequest{},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getExchangesCommand = cli.Command{
	Name:      "getexchanges",
	Usage:     "gets a list of enabled or available exchanges",
	ArgsUsage: "<enabled>",
	Action:    getExchanges,
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "enabled",
			Usage: "whether to list enabled exchanges or not",
		},
	},
}

func getExchanges(c *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var enabledOnly bool
	if c.IsSet("enabled") {
		enabledOnly = c.Bool("enabled")
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetExchanges(context.Background(),
		&gctrpc.GetExchangesRequest{
			Enabled: enabledOnly,
		},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var enableExchangeCommand = cli.Command{
	Name:      "enableexchange",
	Usage:     "enables an exchange",
	ArgsUsage: "<exchange>",
	Action:    enableExchange,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to enable",
		},
	},
}

func enableExchange(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "enableexchange")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.EnableExchange(context.Background(),
		&gctrpc.GenericExchangeNameRequest{
			Exchange: exchangeName,
		},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var disableExchangeCommand = cli.Command{
	Name:      "disableexchange",
	Usage:     "disables an exchange",
	ArgsUsage: "<exchange>",
	Action:    disableExchange,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to disable",
		},
	},
}

func disableExchange(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "disableexchange")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.DisableExchange(context.Background(),
		&gctrpc.GenericExchangeNameRequest{
			Exchange: exchangeName,
		},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getExchangeInfoCommand = cli.Command{
	Name:      "getexchangeinfo",
	Usage:     "gets a specific exchanges info",
	ArgsUsage: "<exchange>",
	Action:    getExchangeInfo,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the info for",
		},
	},
}

func getExchangeInfo(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "getexchangeinfo")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetExchangeInfo(context.Background(),
		&gctrpc.GenericExchangeNameRequest{
			Exchange: exchangeName,
		},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getTickerCommand = cli.Command{
	Name:      "getticker",
	Usage:     "gets the ticker for a specific currency pair and exchange",
	ArgsUsage: "<exchange> <pair> <asset>",
	Action:    getTicker,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the ticker for",
		},
		cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to get the ticker for",
		},
		cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type of the currency pair to get the ticker for",
		},
	},
}

func getTicker(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "getticker")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	var currencyPair string
	var assetType string

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

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	p := currency.NewPairFromString(currencyPair)
	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetTicker(context.Background(),
		&gctrpc.GetTickerRequest{
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

	jsonOutput(result)
	return nil
}

var getTickersCommand = cli.Command{
	Name:   "gettickers",
	Usage:  "gets all tickers for all enabled exchanes and currency pairs",
	Action: getTickers,
}

func getTickers(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetTickers(context.Background(), &gctrpc.GetTickersRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getOrderbookCommand = cli.Command{
	Name:      "getorderbook",
	Usage:     "gets the orderbook for a specific currency pair and exchange",
	ArgsUsage: "<exchange> <pair> <asset>",
	Action:    getOrderbook,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the orderbook for",
		},
		cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to get the orderbook for",
		},
		cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type of the currency pair to get the orderbook for",
		},
	},
}

func getOrderbook(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "getorderbook")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	var currencyPair string
	var assetType string

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

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	p := currency.NewPairFromString(currencyPair)
	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetOrderbook(context.Background(),
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

	jsonOutput(result)
	return nil
}

var getOrderbooksCommand = cli.Command{
	Name:   "getorderbooks",
	Usage:  "gets all orderbooks for all enabled exchanes and currency pairs",
	Action: getOrderbooks,
}

func getOrderbooks(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetOrderbooks(context.Background(), &gctrpc.GetOrderbooksRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getAccountInfoCommand = cli.Command{
	Name:      "getaccountinfo",
	Usage:     "gets the exchange account balance info",
	ArgsUsage: "<exchange>",
	Action:    getAccountInfo,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the account info for",
		},
	},
}

func getAccountInfo(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "getaccountinfo")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetAccountInfo(context.Background(),
		&gctrpc.GetAccountInfoRequest{
			Exchange: exchange,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getConfigCommand = cli.Command{
	Name:   "getconfig",
	Usage:  "gets the config",
	Action: getConfig,
}

func getConfig(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetConfig(context.Background(), &gctrpc.GetConfigRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getPortfolioCommand = cli.Command{
	Name:   "getportfolio",
	Usage:  "gets the portfolio",
	Action: getPortfolio,
}

func getPortfolio(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetPortfolio(context.Background(), &gctrpc.GetPortfolioRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getPortfolioSummaryCommand = cli.Command{
	Name:   "getportfoliosummary",
	Usage:  "gets the portfolio summary",
	Action: getPortfolioSummary,
}

func getPortfolioSummary(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetPortfolioSummary(context.Background(), &gctrpc.GetPortfolioSummaryRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var addPortfolioAddressCommand = cli.Command{
	Name:      "addportfolioaddress",
	Usage:     "adds an address to the portfolio",
	ArgsUsage: "<address> <coin_type> <description> <balance>",
	Action:    addPortfolioAddress,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "address",
			Usage: "the address to add to the portfolio",
		},
		cli.StringFlag{
			Name:  "coin_type",
			Usage: "the coin type e.g ('BTC')",
		},
		cli.StringFlag{
			Name:  "description",
			Usage: "description of the address",
		},
		cli.Float64Flag{
			Name:  "balance",
			Usage: "balance of the address",
		},
	},
}

func addPortfolioAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "addportfolioaddress")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var address string
	var coinType string
	var description string
	var balance float64

	if c.IsSet("address") {
		address = c.String("address")
	} else {
		address = c.Args().First()
	}

	if c.IsSet("coin_type") {
		coinType = c.String("coin_type")
	} else {
		coinType = c.Args().Get(1)
	}

	if c.IsSet("description") {
		description = c.String("asset")
	} else {
		description = c.Args().Get(2)
	}

	if c.IsSet("balance") {
		balance = c.Float64("balance")
	} else {
		balance, _ = strconv.ParseFloat(c.Args().Get(3), 64)
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.AddPortfolioAddress(context.Background(),
		&gctrpc.AddPortfolioAddressRequest{
			Address:     address,
			CoinType:    coinType,
			Description: description,
			Balance:     balance,
		},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var removePortfolioAddressCommand = cli.Command{
	Name:      "removeportfolioaddress",
	Usage:     "removes an address from the portfolio",
	ArgsUsage: "<address> <coin_type> <description>",
	Action:    removePortfolioAddress,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "address",
			Usage: "the address to add to the portfolio",
		},
		cli.StringFlag{
			Name:  "coin_type",
			Usage: "the coin type e.g ('BTC')",
		},
		cli.StringFlag{
			Name:  "description",
			Usage: "description of the address",
		},
	},
}

func removePortfolioAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "removeportfolioaddress")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var address string
	var coinType string
	var description string

	if c.IsSet("address") {
		address = c.String("address")
	} else {
		address = c.Args().First()
	}

	if c.IsSet("coin_type") {
		coinType = c.String("coin_type")
	} else {
		coinType = c.Args().Get(1)
	}

	if c.IsSet("description") {
		description = c.String("asset")
	} else {
		description = c.Args().Get(2)
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.RemovePortfolioAddress(context.Background(),
		&gctrpc.RemovePortfolioAddressRequest{
			Address:     address,
			CoinType:    coinType,
			Description: description,
		},
	)

	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getForexProvidersCommand = cli.Command{
	Name:   "getforexproviders",
	Usage:  "gets the available forex providers",
	Action: getForexProviders,
}

func getForexProviders(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetForexProviders(context.Background(), &gctrpc.GetForexProvidersRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getForexRatesCommand = cli.Command{
	Name:   "getforexrates",
	Usage:  "gets forex rates",
	Action: getForexRates,
}

func getForexRates(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetForexRates(context.Background(), &gctrpc.GetForexRatesRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getOrdersCommand = cli.Command{
	Name:      "getorders",
	Usage:     "gets the open orders",
	ArgsUsage: "<exchange> <asset_type> <pair>",
	Action:    getOrders,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get orders for",
		},
		cli.StringFlag{
			Name:  "asset_type",
			Usage: "the asset type to get orders for",
		},
		cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to get orders for",
		},
	},
}

func getOrders(c *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	var assetType string
	var currencyPair string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset_type") {
		assetType = c.String("asset_type")
	} else {
		assetType = c.Args().Get(1)
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}

	p := currency.NewPairFromString(currencyPair)
	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetOrders(context.Background(), &gctrpc.GetOrdersRequest{
		Exchange:  exchangeName,
		AssetType: assetType,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getOrderCommand = cli.Command{
	Name:      "getorder",
	Usage:     "gets the specified order info",
	ArgsUsage: "<exchange> <order_id>",
	Action:    getOrder,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the order for",
		},
		cli.StringFlag{
			Name:  "order_id",
			Usage: "the order id to retrieve",
		},
	},
}

func getOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "getorder")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	var orderID string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("order_id") {
		orderID = c.String("order_id")
	} else {
		orderID = c.Args().Get(1)
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetOrder(context.Background(), &gctrpc.GetOrderRequest{
		Exchange: exchangeName,
		OrderId:  orderID,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var submitOrderCommand = cli.Command{
	Name:      "submitorder",
	Usage:     "submit order submits an exchange order",
	ArgsUsage: "<exchange> <currency_pair> <side> <order_type> <amount> <price> <client_id>",
	Action:    submitOrder,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to submit the order for",
		},
		cli.StringFlag{
			Name:  "currency_pair",
			Usage: "the currency pair",
		},
		cli.StringFlag{
			Name:  "side",
			Usage: "the order side to use (BUY OR SELL)",
		},
		cli.StringFlag{
			Name:  "order_type",
			Usage: "the order type (MARKET OR LIMIT)",
		},
		cli.Float64Flag{
			Name:  "amount",
			Usage: "the amount for the order",
		},
		cli.Float64Flag{
			Name:  "price",
			Usage: "the price for the order",
		},
		cli.StringFlag{
			Name:  "client_id",
			Usage: "the optional client order ID",
		},
	},
}

func submitOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "submitorder")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	var currencyPair string
	var orderSide string
	var orderType string
	var amount float64
	var price float64
	var clientID string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("currency_pair") {
		currencyPair = c.String("currency_pair")
	} else {
		currencyPair = c.Args().Get(1)
	}

	if c.IsSet("side") {
		orderSide = c.String("side")
	} else {
		orderSide = c.Args().Get(2)
	}

	if c.IsSet("order_type") {
		orderType = c.String("order_type")
	} else {
		orderType = c.Args().Get(3)
	}

	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else {
		amount, _ = strconv.ParseFloat(c.Args().Get(4), 64)
	}

	if c.IsSet("price") {
		price = c.Float64("price")
	} else {
		price, _ = strconv.ParseFloat(c.Args().Get(5), 64)
	}

	if c.IsSet("client_id") {
		clientID = c.String("client_id")
	} else {
		clientID = c.Args().Get(6)
	}

	p := currency.NewPairFromString(currencyPair)
	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.SubmitOrder(context.Background(), &gctrpc.SubmitOrderRequest{
		Exchange: exchangeName,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:      orderSide,
		OrderType: orderType,
		Amount:    amount,
		Price:     price,
		ClientId:  clientID,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var cancelOrderCommand = cli.Command{
	Name:      "cancelorder",
	Usage:     "cancel order cancels an exchange order",
	ArgsUsage: "<exchange> <account_id> <order_id> <currency_pair> <asset_type> <wallet_address> <side>",
	Action:    cancelOrder,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to cancel the order for",
		},
		cli.StringFlag{
			Name:  "account_id",
			Usage: "the account id",
		},
		cli.StringFlag{
			Name:  "order_id",
			Usage: "the order id",
		},
		cli.StringFlag{
			Name:  "currency_pair",
			Usage: "the currency pair to cancel the order for",
		},
		cli.StringFlag{
			Name:  "asset_type",
			Usage: "the asset type",
		},
		cli.Float64Flag{
			Name:  "wallet_address",
			Usage: "the wallet address",
		},
		cli.StringFlag{
			Name:  "side",
			Usage: "the order side",
		},
	},
}

func cancelOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "cancelorder")
		return nil
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	var accountID string
	var orderID string
	var currencyPair string
	var assetType string
	var walletAddress string
	var orderSide string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("order_id") {
		orderID = c.String("order_id")
	} else {
		orderID = c.Args().Get(2)
	}

	if c.IsSet("account_id") {
		accountID = c.String("account_id")
	}

	if c.IsSet("currency_pair") {
		currencyPair = c.String("currency_pair")
	}

	if c.IsSet("asset_type") {
		assetType = c.String("asset_type")
	}

	if c.IsSet("wallet_address") {
		walletAddress = c.String("wallet_address")
	}

	if c.IsSet("order_side") {
		orderSide = c.String("order_side")
	}

	var p currency.Pair
	if len(currencyPair) > 0 {
		p = currency.NewPairFromString(currencyPair)
	}
	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.CancelOrder(context.Background(), &gctrpc.CancelOrderRequest{
		Exchange:  exchangeName,
		AccountId: accountID,
		OrderId:   orderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType:     assetType,
		WalletAddress: walletAddress,
		Side:          orderSide,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var cancelAllOrdersCommand = cli.Command{
	Name:      "cancelallorders",
	Usage:     "cancels all orders (all or by exchange name)",
	ArgsUsage: "<exchange>",
	Action:    cancelAllOrders,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to cancel all orders on",
		},
	},
}

func cancelAllOrders(c *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.CancelAllOrders(context.Background(), &gctrpc.CancelAllOrdersRequest{
		Exchange: exchangeName,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getEventsCommand = cli.Command{
	Name:   "getevents",
	Usage:  "gets all events",
	Action: getEvents,
}

func getEvents(_ *cli.Context) error {
	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetEvents(context.Background(), &gctrpc.GetEventsRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var addEventCommand = cli.Command{
	Name:      "addevent",
	Usage:     "adds an event",
	ArgsUsage: "<exchange> <item> <condition> <price> <check_bids> <check_bids_and_asks> <orderbook_amount> <currency_pair> <asset_type> <action>",
	Action:    addEvent,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to add an event for",
		},
		cli.StringFlag{
			Name:  "item",
			Usage: "the item to trigger the event",
		},
		cli.StringFlag{
			Name:  "condition",
			Usage: "the condition for the event",
		},
		cli.Float64Flag{
			Name:  "price",
			Usage: "the price to trigger the event",
		},
		cli.BoolFlag{
			Name:  "check_bids",
			Usage: "whether to check the bids (if false, asks will be used)",
		},
		cli.BoolFlag{
			Name:  "check_bids_and_asks",
			Usage: "the wallet address",
		},
		cli.Float64Flag{
			Name:  "orderbook_amount",
			Usage: "the orderbook amount to trigger the event",
		},
		cli.StringFlag{
			Name:  "currency_pair",
			Usage: "the currency pair",
		},
		cli.StringFlag{
			Name:  "asset_type",
			Usage: "the asset type",
		},
		cli.StringFlag{
			Name:  "action",
			Usage: "the action for the event to perform upon trigger",
		},
	},
}

func addEvent(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "addevent")
		return nil
	}

	var exchangeName string
	var item string
	var condition string
	var price float64
	var checkBids bool
	var checkBidsAndAsks bool
	var orderbookAmount float64
	var currencyPair string
	var assetType string
	var action string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		return fmt.Errorf("exchange name is required")
	}

	if c.IsSet("item") {
		item = c.String("item")
	} else {
		return fmt.Errorf("item is required")
	}

	if c.IsSet("condition") {
		condition = c.String("condition")
	} else {
		return fmt.Errorf("condition is required")
	}

	if c.IsSet("price") {
		price = c.Float64("price")
	}

	if c.IsSet("check_bids") {
		checkBids = c.Bool("check_bids")
	}

	if c.IsSet("check_bids_and_asks") {
		checkBids = c.Bool("check_bids_and_asks")
	}

	if c.IsSet("orderbook_amount") {
		orderbookAmount = c.Float64("orderbook_amount")
	}

	if c.IsSet("currency_pair") {
		currencyPair = c.String("currency_pair")
	} else {
		return fmt.Errorf("currency pair is required")
	}

	if c.IsSet("asset_type") {
		assetType = c.String("asset_type")
	}

	if c.IsSet("action") {
		action = c.String("action")
	} else {
		return fmt.Errorf("action is required")
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	p := currency.NewPairFromString(currencyPair)
	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.AddEvent(context.Background(), &gctrpc.AddEventRequest{
		Exchange: exchangeName,
		Item:     item,
		ConditionParams: &gctrpc.ConditionParams{
			Condition:        condition,
			Price:            price,
			CheckBids:        checkBids,
			CheckBidsAndAsks: checkBidsAndAsks,
			OrderbookAmount:  orderbookAmount,
		},
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: assetType,
		Action:    action,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var removeEventCommand = cli.Command{
	Name:      "removeevent",
	Usage:     "removes an event",
	ArgsUsage: "<event_id>",
	Action:    removeEvent,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "event_id",
			Usage: "the event id to remove",
		},
	},
}

func removeEvent(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "removeevent")
		return nil
	}

	var eventID int64
	if c.IsSet("event_id") {
		eventID = c.Int64("event_id")
	} else {
		evtID, err := strconv.Atoi(c.Args().Get(0))
		if err != nil {
			return fmt.Errorf("unable to strconv input to int. Err: %s", err)
		}
		eventID = int64(evtID)
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.RemoveEvent(context.Background(),
		&gctrpc.RemoveEventRequest{Id: eventID})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCryptocurrencyDepositAddressesCommand = cli.Command{
	Name:      "getcryptocurrencydepositaddresses",
	Usage:     "gets the cryptocurrency deposit addresses for an exchange",
	ArgsUsage: "<exchange>",
	Action:    getCryptocurrencyDepositAddresses,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the cryptocurrency deposit addresses for",
		},
	},
}

func getCryptocurrencyDepositAddresses(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "getcryptocurrencydepositaddresses")
		return nil
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetCryptocurrencyDepositAddresses(context.Background(),
		&gctrpc.GetCryptocurrencyDepositAddressesRequest{Exchange: exchangeName})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCryptocurrencyDepositAddressCommand = cli.Command{
	Name:      "getcryptocurrencydepositaddress",
	Usage:     "gets the cryptocurrency deposit address for an exchange and cryptocurrency",
	ArgsUsage: "<exchange> <cryptocurrency>",
	Action:    getCryptocurrencyDepositAddress,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the cryptocurrency deposit address for",
		},
		cli.StringFlag{
			Name:  "cryptocurrency",
			Usage: "the cryptocurrency to get the deposit address for",
		},
	},
}

func getCryptocurrencyDepositAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		cli.ShowCommandHelp(c, "getcryptocurrencydepositaddresses")
		return nil
	}

	var exchangeName string
	var cryptocurrency string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("cryptocurrency") {
		cryptocurrency = c.String("cryptocurrency")
	} else {
		cryptocurrency = c.Args().Get(1)
	}

	conn, err := setupClient()
	if err != nil {
		return err
	}
	defer conn.Close()

	client := gctrpc.NewGoCryptoTraderClient(conn)
	result, err := client.GetCryptocurrencyDepositAddress(context.Background(),
		&gctrpc.GetCryptocurrencyDepositAddressRequest{
			Exchange:       exchangeName,
			Cryptocurrency: cryptocurrency,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var withdrawCryptocurrencyFundsCommand = cli.Command{
	Name:      "withdrawcryptocurrencyfunds",
	Usage:     "withdraws cryptocurrency funds from the desired exchange",
	ArgsUsage: "<exchange> <cryptocurrency>",
	Action:    withdrawCryptocurrencyFunds,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to withdraw from",
		},
		cli.StringFlag{
			Name:  "cryptocurrency",
			Usage: "the cryptocurrency to withdraw funds from",
		},
	},
}

func withdrawCryptocurrencyFunds(_ *cli.Context) error {
	return common.ErrNotYetImplemented
}

var withdrawFiatFundsCommand = cli.Command{
	Name:      "withdrawfiatfunds",
	Usage:     "withdraws fiat funds from the desired exchange",
	ArgsUsage: "<exchange> <fiat_currency>",
	Action:    withdrawFiatFunds,
	Flags: []cli.Flag{
		cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to withdraw from",
		},
		cli.StringFlag{
			Name:  "fiat_currency",
			Usage: "the fiat currency to withdraw funds from",
		},
	},
}

func withdrawFiatFunds(_ *cli.Context) error {
	return common.ErrNotYetImplemented
}
