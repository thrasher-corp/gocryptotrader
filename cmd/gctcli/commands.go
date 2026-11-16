package main

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var errWithdrawalLimitOutOfRange = errors.New("withdrawal limit is outside the int32 range")

var getInfoCommand = &cli.Command{
	Name:   "getinfo",
	Usage:  "gets GoCryptoTrader info",
	Action: getInfo,
}

func getInfo(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetInfo(c.Context,
		&gctrpc.GetInfoRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getSubsystemsCommand = &cli.Command{
	Name:   "getsubsystems",
	Usage:  "gets GoCryptoTrader subsystems and their status",
	Action: getSubsystems,
}

func getSubsystems(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetSubsystems(c.Context,
		&gctrpc.GetSubsystemsRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var enableSubsystemCommand = &cli.Command{
	Name:   "enablesubsystem",
	Usage:  "enables an engine subsystem",
	Action: enableSubsystem,
	Flags: []cli.Flag{
		requiredSubsystemFlag,
	},
}

func enableSubsystem(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	subsystemName := c.String(subsystemFlagName)

	if subsystemName == "" {
		return errors.New("invalid subsystem supplied")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.EnableSubsystem(c.Context,
		&gctrpc.GenericSubsystemRequest{
			Subsystem: subsystemName,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var disableSubsystemCommand = &cli.Command{
	Name:   "disablesubsystem",
	Usage:  "disables an engine subsystem",
	Action: disableSubsystem,
	Flags: []cli.Flag{
		requiredSubsystemFlag,
	},
}

func disableSubsystem(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	subsystemName := c.String(subsystemFlagName)

	if subsystemName == "" {
		return errors.New("invalid subsystem supplied")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.DisableSubsystem(c.Context,
		&gctrpc.GenericSubsystemRequest{
			Subsystem: subsystemName,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getRPCEndpointsCommand = &cli.Command{
	Name:   "getrpcendpoints",
	Usage:  "gets GoCryptoTrader endpoints info",
	Action: getRPCEndpoints,
}

func getRPCEndpoints(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetRPCEndpoints(c.Context,
		&gctrpc.GetRPCEndpointsRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCommunicationRelayersCommand = &cli.Command{
	Name:   "getcommsrelayers",
	Usage:  "gets GoCryptoTrader communication relayers",
	Action: getCommunicationRelayers,
}

func getCommunicationRelayers(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCommunicationRelayers(c.Context,
		&gctrpc.GetCommunicationRelayersRequest{},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getExchangesCommand = &cli.Command{
	Name:   "getexchanges",
	Usage:  "gets a list of enabled or available exchanges",
	Action: getExchanges,
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "enabled",
			Usage: "whether to list enabled exchanges or not",
		},
	},
}

func getExchanges(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchanges(c.Context,
		&gctrpc.GetExchangesRequest{
			Enabled: c.Bool("enabled"),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var enableExchangeCommand = &cli.Command{
	Name:   "enableexchange",
	Usage:  "enables an exchange",
	Action: enableExchange,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

func enableExchange(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.EnableExchange(c.Context,
		&gctrpc.GenericExchangeNameRequest{
			Exchange: c.String(exchangeFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var disableExchangeCommand = &cli.Command{
	Name:   "disableexchange",
	Usage:  "disables an exchange",
	Action: disableExchange,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

func disableExchange(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.DisableExchange(c.Context,
		&gctrpc.GenericExchangeNameRequest{
			Exchange: c.String(exchangeFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getExchangeOTPCommand = &cli.Command{
	Name:   "getexchangeotp",
	Usage:  "gets a specific exchange OTP code",
	Action: getExchangeOTPCode,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

func getExchangeOTPCode(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangeOTPCode(c.Context,
		&gctrpc.GenericExchangeNameRequest{
			Exchange: c.String(exchangeFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getExchangeOTPsCommand = &cli.Command{
	Name:   "getexchangeotps",
	Usage:  "gets all exchange OTP codes",
	Action: getExchangeOTPCodes,
}

func getExchangeOTPCodes(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangeOTPCodes(c.Context,
		&gctrpc.GetExchangeOTPsRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getExchangeInfoCommand = &cli.Command{
	Name:   "getexchangeinfo",
	Usage:  "gets a specific exchanges info",
	Action: getExchangeInfo,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

func getExchangeInfo(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangeInfo(c.Context,
		&gctrpc.GenericExchangeNameRequest{
			Exchange: c.String(exchangeFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getTickerCommand = &cli.Command{
	Name:   "getticker",
	Usage:  "gets the ticker for a specific currency pair and exchange",
	Action: getTicker,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredPairFlag,
		requiredAssetFlag,
	},
}

func getTicker(c *cli.Context) error {
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
	result, err := client.GetTicker(c.Context,
		&gctrpc.GetTickerRequest{
			Exchange: c.String(exchangeFlagName),
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

var getTickersCommand = &cli.Command{
	Name:   "gettickers",
	Usage:  "gets all tickers for all enabled exchanges and currency pairs",
	Action: getTickers,
}

func getTickers(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetTickers(c.Context, &gctrpc.GetTickersRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getAccountBalancesCommand = &cli.Command{
	Name:   "getaccountbalances",
	Usage:  "gets the exchange account balances",
	Action: getAccountBalances,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredAssetFlag,
	},
}

func getAccountBalances(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAccountBalances(c.Context,
		&gctrpc.GetAccountBalancesRequest{
			Exchange:  c.String(exchangeFlagName),
			AssetType: assetType,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getAccountBalancesStreamCommand = &cli.Command{
	Name:   "getaccountbalancesstream",
	Usage:  "gets the account balances stream for a specific exchange",
	Action: getAccountBalancesStream,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredAssetFlag,
	},
}

func getAccountBalancesStream(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	exchangeName := c.String(exchangeFlagName)
	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAccountBalancesStream(c.Context,
		&gctrpc.GetAccountBalancesRequest{Exchange: exchangeName, AssetType: assetType})
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

		fmt.Printf("Account balance stream for %s:\n\n", exchangeName)

		fmt.Printf("%+v", resp)
	}
}

var updateAccountBalancesCommand = &cli.Command{
	Name:   "updateaccountbalances",
	Usage:  "updates the exchange account balances",
	Action: updateAccountBalances,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredAssetFlag,
	},
}

func updateAccountBalances(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.UpdateAccountBalances(c.Context,
		&gctrpc.GetAccountBalancesRequest{
			Exchange:  c.String(exchangeFlagName),
			AssetType: assetType,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getConfigCommand = &cli.Command{
	Name:   "getconfig",
	Usage:  "gets the config",
	Action: getConfig,
}

func getConfig(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetConfig(c.Context, &gctrpc.GetConfigRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getPortfolioCommand = &cli.Command{
	Name:   "getportfolio",
	Usage:  "gets the portfolio",
	Action: getPortfolio,
}

func getPortfolio(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetPortfolio(c.Context, &gctrpc.GetPortfolioRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getPortfolioSummaryCommand = &cli.Command{
	Name:   "getportfoliosummary",
	Usage:  "gets the portfolio summary",
	Action: getPortfolioSummary,
}

func getPortfolioSummary(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetPortfolioSummary(c.Context, &gctrpc.GetPortfolioSummaryRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var addPortfolioAddressCommand = &cli.Command{
	Name:   "addportfolioaddress",
	Usage:  "adds an address to the portfolio",
	Action: addPortfolioAddress,
	Flags: []cli.Flag{
		requiredPortfolioAddressFlag,
		requiredCoinTypeFlag,
		&cli.StringFlag{
			Name:  descriptionFlagName,
			Usage: portfolioDescriptionUsage,
		},
		&cli.Float64Flag{
			Name:  "balance",
			Usage: "balance of the address",
		},
		&cli.BoolFlag{
			Name:  "cold_storage",
			Usage: "true/false if address is cold storage",
		},
		&cli.StringFlag{
			Name:  "supported_exchanges",
			Usage: "common separated list of exchanges supported by this address for withdrawals",
		},
	},
}

func addPortfolioAddress(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.AddPortfolioAddress(c.Context,
		&gctrpc.AddPortfolioAddressRequest{
			Address:            c.String(addressFlagName),
			CoinType:           c.String(coinTypeFlagName),
			Description:        c.String(descriptionFlagName),
			Balance:            c.Float64("balance"),
			SupportedExchanges: c.String("supported_exchanges"),
			ColdStorage:        c.Bool("cold_storage"),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var removePortfolioAddressCommand = &cli.Command{
	Name:   "removeportfolioaddress",
	Usage:  "removes an address from the portfolio",
	Action: removePortfolioAddress,
	Flags: []cli.Flag{
		requiredPortfolioAddressFlag,
		requiredCoinTypeFlag,
		&cli.StringFlag{
			Name:  descriptionFlagName,
			Usage: portfolioDescriptionUsage,
		},
	},
}

func removePortfolioAddress(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.RemovePortfolioAddress(c.Context,
		&gctrpc.RemovePortfolioAddressRequest{
			Address:     c.String(addressFlagName),
			CoinType:    c.String(coinTypeFlagName),
			Description: c.String(descriptionFlagName),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getForexProvidersCommand = &cli.Command{
	Name:   "getforexproviders",
	Usage:  "gets the available forex providers",
	Action: getForexProviders,
}

func getForexProviders(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetForexProviders(c.Context, &gctrpc.GetForexProvidersRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getForexRatesCommand = &cli.Command{
	Name:   "getforexrates",
	Usage:  "gets forex rates",
	Action: getForexRates,
}

func getForexRates(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetForexRates(c.Context, &gctrpc.GetForexRatesRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getOrdersCommand = &cli.Command{
	Name:   "getorders",
	Usage:  "gets the open orders",
	Action: getOrders,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredAssetFlag,
		requiredPairFlag,
		&cli.StringFlag{
			Name:  startFlagName,
			Usage: startUsage,
			Value: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
		},
		&cli.StringFlag{
			Name:  endFlagName,
			Usage: endUsage,
			Value: time.Now().Format(time.DateTime),
		},
	},
}

func getOrders(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	currencyPair := c.String(pairFlagName)

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err = time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if e.Before(s) {
		return common.ErrStartAfterEnd
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrders(c.Context, &gctrpc.GetOrdersRequest{
		Exchange:  c.String(exchangeFlagName),
		AssetType: assetType,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		StartDate: s.Format(common.SimpleTimeFormatWithTimezone),
		EndDate:   e.Format(common.SimpleTimeFormatWithTimezone),
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getManagedOrdersCommand = &cli.Command{
	Name:   "getmanagedorders",
	Usage:  "gets the current orders from the order manager",
	Action: getManagedOrders,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredAssetFlag,
		requiredPairFlag,
	},
}

func getManagedOrders(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	currencyPair := c.String(pairFlagName)

	if !validPair(currencyPair) {
		return errInvalidPair
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
	result, err := client.GetManagedOrders(c.Context, &gctrpc.GetOrdersRequest{
		Exchange:  c.String(exchangeFlagName),
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

var getOrderCommand = &cli.Command{
	Name:   "getorder",
	Usage:  "gets the specified order info",
	Action: getOrder,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredAssetFlag,
		requiredPairFlag,
		requiredOrderIDFlag,
	},
}

func getOrder(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
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
	result, err := client.GetOrder(c.Context, &gctrpc.GetOrderRequest{
		Exchange: c.String(exchangeFlagName),
		OrderId:  c.String(orderIDFlagName),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Asset: assetType,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var submitOrderCommand = &cli.Command{
	Name:   "submitorder",
	Usage:  "submit order submits an exchange order",
	Action: submitOrder,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredPairFlag,
		requiredSideFlag,
		&cli.StringFlag{
			Name:     "type",
			Required: true,
			Usage:    "the order type (MARKET OR LIMIT)",
		},
		requiredOrderAmountFlag,
		&cli.Float64Flag{
			Name:  priceFlagName,
			Usage: orderPriceUsage,
		},
		&cli.StringFlag{
			Name:  "client_id",
			Usage: "the optional client order ID",
		},
		requiredAssetFlag,
		&cli.StringFlag{
			Name:  marginTypeFlagName,
			Usage: marginTypeUsage,
		},
	},
}

func submitOrder(c *cli.Context) error {
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

	orderType := c.String("type")

	if orderType == "" {
		return errors.New("order type must be set")
	}

	amount := c.Float64(amountFlagName)

	if amount == 0 {
		return errors.New("amount must be set")
	}

	assetType := c.String(assetFlagName)
	marginType := c.String(marginTypeFlagName)

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	marginType = strings.ToLower(marginType)
	if marginType != "" && !margin.IsValidString(marginType) {
		return margin.ErrInvalidMarginType
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
	result, err := client.SubmitOrder(c.Context, &gctrpc.SubmitOrderRequest{
		Exchange: c.String(exchangeFlagName),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:       orderSide,
		OrderType:  orderType,
		Amount:     amount,
		Price:      c.Float64(priceFlagName),
		ClientId:   c.String("client_id"),
		AssetType:  assetType,
		MarginType: marginType,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var simulateOrderCommand = &cli.Command{
	Name:   "simulateorder",
	Usage:  "simulate order simulates an exchange order",
	Action: simulateOrder,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredPairFlag,
		requiredSideFlag,
		requiredOrderAmountFlag,
	},
}

func simulateOrder(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	orderSide := c.String(sideFlagName)

	if orderSide == "" {
		return errors.New("side must be set")
	}

	amount := c.Float64(amountFlagName)

	if amount == 0 {
		return errors.New("amount must be set")
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
	result, err := client.SimulateOrder(c.Context, &gctrpc.SimulateOrderRequest{
		Exchange: c.String(exchangeFlagName),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:   orderSide,
		Amount: amount,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var cancelOrderCommand = &cli.Command{
	Name:   "cancelorder",
	Usage:  "cancel order cancels an exchange order",
	Action: cancelOrder,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		&cli.StringFlag{
			Name:  accountIDFlagName,
			Usage: accountIDUsage,
		},
		requiredOrderIDFlag,
		&cli.StringFlag{
			Name:  pairFlagName,
			Usage: pairUsage,
		},
		requiredAssetFlag,
		&cli.StringFlag{
			Name:  sideFlagName,
			Usage: sideUsage,
		},
	},
}

func cancelOrder(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	orderID := c.String(orderIDFlagName)

	if orderID == "" {
		return errors.New("an order ID must be set")
	}

	currencyPair := c.String(pairFlagName)

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	// pair is optional, but if it's set, do a validity check
	var p currency.Pair
	if currencyPair != "" {
		if !validPair(currencyPair) {
			return errInvalidPair
		}
		var err error
		p, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
		if err != nil {
			return err
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CancelOrder(c.Context, &gctrpc.CancelOrderRequest{
		Exchange:  c.String(exchangeFlagName),
		AccountId: c.String(accountIDFlagName),
		OrderId:   orderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: assetType,
		Side:      c.String(sideFlagName),
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var cancelBatchOrdersCommand = &cli.Command{
	Name:   "cancelbatchorders",
	Usage:  "cancel batch orders cancels a list of exchange orders (comma separated)",
	Action: cancelBatchOrders,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		&cli.StringFlag{
			Name:  accountIDFlagName,
			Usage: accountIDUsage,
		},
		&cli.StringFlag{
			Name:     "order_ids",
			Required: true,
			Usage:    "the comma separated orders id-s",
		},
		&cli.StringFlag{
			Name:  pairFlagName,
			Usage: pairUsage,
		},
		requiredAssetFlag,
		&cli.StringFlag{
			Name:  sideFlagName,
			Usage: sideUsage,
		},
	},
}

func cancelBatchOrders(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	orderID := c.String("order_ids")

	if orderID == "" {
		return errors.New("an order ID must be set")
	}

	currencyPair := c.String(pairFlagName)

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	// pair is optional, but if it's set, do a validity check
	var p currency.Pair
	if currencyPair != "" {
		if !validPair(currencyPair) {
			return errInvalidPair
		}
		var err error
		p, err = currency.NewPairDelimiter(currencyPair, pairDelimiter)
		if err != nil {
			return err
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CancelBatchOrders(c.Context, &gctrpc.CancelBatchOrdersRequest{
		Exchange:  c.String(exchangeFlagName),
		AccountId: c.String(accountIDFlagName),
		OrdersId:  orderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: assetType,
		Side:      c.String(sideFlagName),
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var cancelAllOrdersCommand = &cli.Command{
	Name:   "cancelallorders",
	Usage:  "cancels all orders for an exchange",
	Action: cancelAllOrders,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

var modifyOrderCommand = &cli.Command{
	Name:   "modifyorder",
	Usage:  "modify price and/or amount of a previously submitted order",
	Action: modifyOrder,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredAssetFlag,
		requiredPairFlag,
		requiredOrderIDFlag,
		&cli.Float64Flag{
			Name:  priceFlagName,
			Usage: orderPriceUsage,
		},
		&cli.Float64Flag{
			Name:  amountFlagName,
			Usage: orderAmountUsage,
		},
	},
}

func cancelAllOrders(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.CancelAllOrders(c.Context, &gctrpc.CancelAllOrdersRequest{
		Exchange: c.String(exchangeFlagName),
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func modifyOrder(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := strings.ToLower(c.String(assetFlagName))
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	// Parse optional flags.

	price := c.Float64(priceFlagName)
	amount := c.Float64(amountFlagName)
	if price == 0 && amount == 0 {
		return errors.New("either --price or --amount should be present")
	}

	// Setup gRPC, make a request and display response.
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.ModifyOrder(c.Context, &gctrpc.ModifyOrderRequest{
		Exchange: c.String(exchangeFlagName),
		OrderId:  c.String(orderIDFlagName),
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Asset:  assetType,
		Price:  price,
		Amount: amount,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getEventsCommand = &cli.Command{
	Name:   "getevents",
	Usage:  "gets all events",
	Action: getEvents,
}

func getEvents(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetEvents(c.Context, &gctrpc.GetEventsRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var addEventCommand = &cli.Command{
	Name:   "addevent",
	Usage:  "adds an event",
	Action: addEvent,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		&cli.StringFlag{
			Name:     "item",
			Required: true,
			Usage:    "the item to trigger the event",
		},
		&cli.StringFlag{
			Name:     "condition",
			Required: true,
			Usage:    "the condition for the event",
		},
		&cli.Float64Flag{
			Name:  priceFlagName,
			Usage: "the price to trigger the event",
		},
		&cli.BoolFlag{
			Name:  "check_bids",
			Usage: "whether to check the bids",
		},
		&cli.BoolFlag{
			Name:  "check_asks",
			Usage: "whether to check the asks",
		},
		&cli.Float64Flag{
			Name:  "orderbook_amount",
			Usage: "the orderbook amount to trigger the event",
		},
		requiredPairFlag,
		requiredAssetFlag,
		&cli.StringFlag{
			Name:     "action",
			Required: true,
			Usage:    "the action for the event to perform upon trigger",
		},
	},
}

func addEvent(c *cli.Context) error {
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
	result, err := client.AddEvent(c.Context, &gctrpc.AddEventRequest{
		Exchange: c.String(exchangeFlagName),
		Item:     c.String("item"),
		ConditionParams: &gctrpc.ConditionParams{
			Condition:       c.String("condition"),
			Price:           c.Float64(priceFlagName),
			CheckBids:       c.Bool("check_bids"),
			CheckAsks:       c.Bool("check_asks"),
			OrderbookAmount: c.Float64("orderbook_amount"),
		},
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: assetType,
		Action:    c.String("action"),
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var removeEventCommand = &cli.Command{
	Name:   "removeevent",
	Usage:  "removes an event",
	Action: removeEvent,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "event_id",
			Required: true,
			Usage:    "the event id to remove",
		},
	},
}

func removeEvent(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	eventID := c.Int64("event_id")

	if eventID == 0 {
		return errors.New("event id must be specified")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.RemoveEvent(c.Context,
		&gctrpc.RemoveEventRequest{Id: eventID})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCryptocurrencyDepositAddressesCommand = &cli.Command{
	Name:   "getcryptocurrencydepositaddresses",
	Usage:  "gets the cryptocurrency deposit addresses for an exchange",
	Action: getCryptocurrencyDepositAddresses,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

func getCryptocurrencyDepositAddresses(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCryptocurrencyDepositAddresses(c.Context,
		&gctrpc.GetCryptocurrencyDepositAddressesRequest{Exchange: c.String(exchangeFlagName)})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCryptocurrencyDepositAddressCommand = &cli.Command{
	Name:   "getcryptocurrencydepositaddress",
	Usage:  "gets the cryptocurrency deposit address for an exchange and cryptocurrency",
	Action: getCryptocurrencyDepositAddress,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredCryptocurrencyFlag,
		&cli.StringFlag{
			Name:  chainFlagName,
			Usage: chainUsage,
		},
		&cli.BoolFlag{
			Name:  "bypass",
			Usage: "whether to bypass the deposit address manager cache if enabled",
		},
	},
}

func getCryptocurrencyDepositAddress(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	cryptocurrency := c.String(cryptocurrencyFlagName)

	if cryptocurrency == "" {
		return errors.New("cryptocurrency must be set")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCryptocurrencyDepositAddress(c.Context,
		&gctrpc.GetCryptocurrencyDepositAddressRequest{
			Exchange:       c.String(exchangeFlagName),
			Cryptocurrency: cryptocurrency,
			Chain:          c.String(chainFlagName),
			Bypass:         c.Bool("bypass"),
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getAvailableTransferChainsCommand = &cli.Command{
	Name:   "getavailabletransferchains",
	Usage:  "gets the available transfer chains (deposits and withdrawals) for the desired exchange and cryptocurrency",
	Action: getAvailableTransferChains,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredCryptocurrencyFlag,
	},
}

func getAvailableTransferChains(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	cryptocurrency := c.String(cryptocurrencyFlagName)

	if cryptocurrency == "" {
		return errors.New("cryptocurrency must be set")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetAvailableTransferChains(c.Context,
		&gctrpc.GetAvailableTransferChainsRequest{
			Exchange:       c.String(exchangeFlagName),
			Cryptocurrency: cryptocurrency,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var withdrawCryptocurrencyFundsCommand = &cli.Command{
	Name:   "withdrawcryptofunds",
	Usage:  "withdraws cryptocurrency funds from the desired exchange",
	Action: withdrawCryptocurrencyFunds,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredCurrencyFlag,
		&cli.StringFlag{
			Name:     addressFlagName,
			Required: true,
			Usage:    "address to withdraw to",
		},
		&cli.StringFlag{
			Name:  "addresstag",
			Usage: "address tag/memo",
		},
		requiredWithdrawalAmountFlag,
		&cli.Float64Flag{
			Name:  "fee",
			Usage: "fee to submit with request",
		},
		&cli.StringFlag{
			Name:  descriptionFlagName,
			Usage: withdrawalDescriptionUsage,
		},
		&cli.StringFlag{
			Name:  chainFlagName,
			Usage: chainUsage,
		},
	},
}

func withdrawCryptocurrencyFunds(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.WithdrawCryptocurrencyFunds(c.Context,
		&gctrpc.WithdrawCryptoRequest{
			Exchange:    c.String(exchangeFlagName),
			Currency:    c.String(currencyFlagName),
			Address:     c.String(addressFlagName),
			AddressTag:  c.String("addresstag"),
			Amount:      c.Float64(amountFlagName),
			Fee:         c.Float64("fee"),
			Description: c.String(descriptionFlagName),
			Chain:       c.String(chainFlagName),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

var withdrawFiatFundsCommand = &cli.Command{
	Name:   "withdrawfiatfunds",
	Usage:  "withdraws fiat funds from the desired exchange",
	Action: withdrawFiatFunds,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredCurrencyFlag,
		requiredWithdrawalAmountFlag,
		&cli.StringFlag{
			Name:  "bankaccountid",
			Usage: "optional ID of the bank account to use; omit for an account without an ID",
		},
		&cli.StringFlag{
			Name:  descriptionFlagName,
			Usage: withdrawalDescriptionUsage,
		},
	},
}

func withdrawFiatFunds(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WithdrawFiatFunds(c.Context,
		&gctrpc.WithdrawFiatRequest{
			Exchange:      c.String(exchangeFlagName),
			Currency:      c.String(currencyFlagName),
			Amount:        c.Float64(amountFlagName),
			Description:   c.String(descriptionFlagName),
			BankAccountId: c.String("bankaccountid"),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

var withdrawalRequestCommand = &cli.Command{
	Name:      "withdrawalrequesthistory",
	Usage:     "retrieve previous withdrawal request details",
	ArgsUsage: commandArgsUsage,
	Subcommands: []*cli.Command{
		{
			Name:  "byid",
			Usage: "id",
			Flags: []cli.Flag{
				requiredWithdrawalIDFlag,
			},
			Action: withdrawalRequestByID,
		},
		{
			Name:  "byexchangeid",
			Usage: "exchange id",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				requiredWithdrawalIDFlag,
			},
			Action: withdrawalRequestByExchangeID,
		},
		{
			Name:  "byexchange",
			Usage: "exchange limit",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				&cli.Int64Flag{
					Name:  limitFlagName,
					Usage: limitUsage,
				},
				&cli.StringFlag{
					Name:  currencyFlagName,
					Usage: currencyUsage,
				},
				requiredAssetFlag,
			},
			Action: withdrawalRequestByExchangeID,
		},
		{
			Name:  "bydate",
			Usage: "exchange start end limit",
			Flags: []cli.Flag{
				requiredExchangeFlag,
				&cli.StringFlag{
					Name:  startFlagName,
					Usage: startUsage,
					Value: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
				},
				&cli.StringFlag{
					Name:  endFlagName,
					Usage: endUsage,
					Value: time.Now().Format(time.DateTime),
				},
				&cli.Int64Flag{
					Name:  limitFlagName,
					Usage: limitUsage,
				},
			},
			Action: withdrawalRequestByDate,
		},
	},
}

func withdrawalRequestByID(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	ID := c.String(idFlagName)

	if ID == "" {
		return errors.New("an ID must be specified")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.WithdrawalEventByID(c.Context,
		&gctrpc.WithdrawalEventByIDRequest{
			Id: ID,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func withdrawalRequestByExchangeID(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange, ccy, assetType string
	exchange = c.String(exchangeFlagName)

	ID := c.String(idFlagName)
	limit := c.Int64(limitFlagName)
	if c.Command.Name == "byexchangeid" {
		if ID == "" {
			return errors.New("an ID must be specified")
		}
		limit = 1
	} else {
		if limit < math.MinInt32 || limit > math.MaxInt32 {
			return errWithdrawalLimitOutOfRange
		}

		ccy = c.String(currencyFlagName)

		assetType = c.String(assetFlagName)

		assetType = strings.ToLower(assetType)
		if !validAsset(assetType) {
			return errInvalidAsset
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.WithdrawalEventsByExchange(c.Context,
		&gctrpc.WithdrawalEventsByExchangeRequest{
			Exchange:  exchange,
			Id:        ID,
			Limit:     int32(limit),
			Currency:  ccy,
			AssetType: assetType,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

func withdrawalRequestByDate(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	limit := c.Int64(limitFlagName)
	if limit < math.MinInt32 || limit > math.MaxInt32 {
		return errWithdrawalLimitOutOfRange
	}

	s, err := time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err := time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if e.Before(s) {
		return common.ErrStartAfterEnd
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WithdrawalEventsByDate(c.Context,
		&gctrpc.WithdrawalEventsByDateRequest{
			Exchange: c.String(exchangeFlagName),
			Start:    s.Format(common.SimpleTimeFormatWithTimezone),
			End:      e.Format(common.SimpleTimeFormatWithTimezone),
			Limit:    int32(limit),
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

var getLoggerDetailsCommand = &cli.Command{
	Name:   "getloggerdetails",
	Usage:  "gets an individual loggers details",
	Action: getLoggerDetails,
	Flags: []cli.Flag{
		requiredLoggerFlag,
	},
}

func getLoggerDetails(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	logger := c.String(loggerFlagName)

	if logger == "" {
		return errors.New("a logger must be specified")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.GetLoggerDetails(c.Context,
		&gctrpc.GetLoggerDetailsRequest{
			Logger: logger,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

var setLoggerDetailsCommand = &cli.Command{
	Name:   "setloggerdetails",
	Usage:  "sets an individual loggers details",
	Action: setLoggerDetails,
	Flags: []cli.Flag{
		requiredLoggerFlag,
		&cli.StringFlag{
			Name:     "level",
			Aliases:  []string{"flags"},
			Required: true,
			Usage:    "pipe separated value of loggers e.g INFO|WARN",
		},
	},
}

func setLoggerDetails(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	logger := c.String(loggerFlagName)

	if logger == "" {
		return errors.New("a logger must be specified")
	}

	level := c.String("level")

	if level == "" {
		return errors.New("level must be specified")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.SetLoggerDetails(c.Context,
		&gctrpc.SetLoggerDetailsRequest{
			Logger: logger,
			Level:  level,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

var getTickerStreamCommand = &cli.Command{
	Name:   "gettickerstream",
	Usage:  "gets the ticker stream for a specific currency pair and exchange",
	Action: getTickerStream,
	Flags: []cli.Flag{
		requiredExchangeFlag,
		requiredPairFlag,
		requiredAssetFlag,
	},
}

func getTickerStream(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	pair := c.String(pairFlagName)

	if !validPair(pair) {
		return errInvalidPair
	}

	assetType := strings.ToLower(c.String(assetFlagName))

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
	result, err := client.GetTickerStream(c.Context,
		&gctrpc.GetTickerStreamRequest{
			Exchange: c.String(exchangeFlagName),
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

		jsonOutput(resp)
		fmt.Println()
	}
}

var getExchangeTickerStreamCommand = &cli.Command{
	Name:   "getexchangetickerstream",
	Usage:  "gets a stream for all tickers associated with an exchange",
	Action: getExchangeTickerStream,
	Flags: []cli.Flag{
		requiredExchangeFlag,
	},
}

func getExchangeTickerStream(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchangeTickerStream(c.Context,
		&gctrpc.GetExchangeTickerStreamRequest{
			Exchange: c.String(exchangeFlagName),
		})
	if err != nil {
		return err
	}

	for {
		resp, err := result.Recv()
		if err != nil {
			return err
		}

		jsonOutput(resp)
		fmt.Println()
	}
}

var getAuditEventCommand = &cli.Command{
	Name:   "getauditevent",
	Usage:  "gets audit events matching query parameters",
	Action: getAuditEvent,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    startFlagName,
			Aliases: []string{"s"},
			Usage:   startUsage,
			Value:   time.Now().Add(-time.Hour).Format(time.DateTime),
		},
		&cli.StringFlag{
			Name:    endFlagName,
			Aliases: []string{"e"},
			Usage:   endUsage,
			Value:   time.Now().Format(time.DateTime),
		},
		&cli.StringFlag{
			Name:    "order",
			Aliases: []string{"o"},
			Usage:   "order results by ascending/descending",
			Value:   "asc",
		},
		&cli.IntFlag{
			Name:    limitFlagName,
			Aliases: []string{"l"},
			Usage:   "how many results to retrieve",
			Value:   100,
		},
	},
}

func getAuditEvent(c *cli.Context) error {
	s, err := time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}

	e, err := time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if e.Before(s) {
		return common.ErrStartAfterEnd
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}

	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.GetAuditEvent(c.Context,
		&gctrpc.GetAuditEventRequest{
			StartDate: s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:   e.Format(common.SimpleTimeFormatWithTimezone),
			Limit:     int32(c.Int(limitFlagName)), //nolint:gosec // TODO: SQL boiler's QueryMode limit only accepts the int type
			OrderBy:   c.String("order"),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

const klineMessage = `interval in seconds. supported values are: 15, 60(1min), 180(3min), 300(5min), 600(10min),
		900(15min) 1800(30min), 3600(1h), 7200(2h), 14400(4h), 21600(6h), 28800(8h), 43200(12h),
		86400(1d), 259200(3d) 604800(1w), 1209600(2w), 1296000(15d), 2592000(1M), 31536000(1Y)`

var getHistoricCandlesCommand = &cli.Command{
	Name:   "gethistoriccandles",
	Usage:  "gets historical candles for the specified granularity up to range size time from now",
	Action: getHistoricCandles,
	Flags: []cli.Flag{
		requiredExchangeAliasFlag,
		requiredPairFlag,
		requiredAssetFlag,
		&cli.Int64Flag{
			Name:    "rangesize",
			Aliases: []string{"r"},
			Usage:   "the amount of time to go back from now to fetch candles in the given granularity",
			Value:   10,
		},
		&cli.Int64Flag{
			Name:    granularityFlagName,
			Aliases: []string{"g"},
			Usage:   klineMessage,
			Value:   86400,
		},
		&cli.BoolFlag{
			Name:    fillMissingDataWithTradesFlagName,
			Aliases: []string{"fill"},
			Usage:   fillMissingDataWithTradesUsage,
		},
	},
}

func getHistoricCandles(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}
	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	candleInterval := time.Duration(c.Int64(granularityFlagName)) * time.Second

	e := time.Now().Truncate(candleInterval)
	s := e.Add(-candleInterval * time.Duration(c.Int64("rangesize")))

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetHistoricCandles(c.Context,
		&gctrpc.GetHistoricCandlesRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType:             assetType,
			Start:                 s.Format(common.SimpleTimeFormatWithTimezone),
			End:                   e.Format(common.SimpleTimeFormatWithTimezone),
			TimeInterval:          int64(candleInterval),
			FillMissingWithTrades: c.Bool(fillMissingDataWithTradesFlagName),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getHistoricCandlesExtendedCommand = &cli.Command{
	Name:   "gethistoriccandlesextended",
	Usage:  "gets historical candles for the specified pair, asset, interval & date range",
	Action: getHistoricCandlesExtended,
	Flags: []cli.Flag{
		requiredExchangeAliasFlag,
		requiredPairAliasFlag,
		requiredAssetAliasFlag,
		&cli.Int64Flag{
			Name:    intervalFlagName,
			Aliases: []string{"i"},
			Usage:   klineMessage,
			Value:   86400,
		},
		&cli.StringFlag{
			Name:  startFlagName,
			Usage: startUsage,
			Value: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
		},
		&cli.StringFlag{
			Name:  endFlagName,
			Usage: endUsage,
			Value: time.Now().Format(time.DateTime),
		},
		&cli.BoolFlag{
			Name:  syncFlagName,
			Usage: "<true/false>",
		},
		&cli.BoolFlag{
			Name:  forceFlagName,
			Usage: forceUsage,
		},
		&cli.BoolFlag{
			Name:  "db",
			Usage: "source data from database <true/false>",
		},
		&cli.BoolFlag{
			Name:    fillMissingDataWithTradesFlagName,
			Aliases: []string{"fill"},
			Usage:   fillMissingDataWithTradesUsage,
		},
	},
}

func getHistoricCandlesExtended(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	sync := c.Bool(syncFlagName)
	force := c.Bool(forceFlagName)

	if force && !sync {
		return errors.New("cannot forcefully overwrite without sync")
	}

	candleInterval := time.Duration(c.Int64(intervalFlagName)) * time.Second
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err = time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if e.Before(s) {
		return common.ErrStartAfterEnd
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetHistoricCandles(c.Context,
		&gctrpc.GetHistoricCandlesRequest{
			Exchange: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType:             assetType,
			Start:                 s.Format(common.SimpleTimeFormatWithTimezone),
			End:                   e.Format(common.SimpleTimeFormatWithTimezone),
			TimeInterval:          int64(candleInterval),
			ExRequest:             true,
			Sync:                  sync,
			UseDb:                 c.Bool("db"),
			FillMissingWithTrades: c.Bool(fillMissingDataWithTradesFlagName),
			Force:                 force,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var findMissingSavedCandleIntervalsCommand = &cli.Command{
	Name:   "findmissingsavedcandleintervals",
	Usage:  "will highlight any interval that is missing candle data so you can fill that gap",
	Action: findMissingSavedCandleIntervals,
	Flags: []cli.Flag{
		requiredExchangeAliasFlag,
		requiredPairAliasFlag,
		requiredAssetAliasFlag,
		&cli.Int64Flag{
			Name:    intervalFlagName,
			Aliases: []string{"i"},
			Usage:   klineMessage,
			Value:   86400,
		},
		&cli.StringFlag{
			Name:  startFlagName,
			Usage: startRoundedUsage,
			Value: time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
		},
		&cli.StringFlag{
			Name:  endFlagName,
			Usage: endRoundedUsage,
			Value: time.Now().Truncate(time.Hour).Format(time.DateTime),
		},
	},
}

func findMissingSavedCandleIntervals(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	currencyPair := c.String(pairFlagName)
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	candleInterval := time.Duration(c.Int64(intervalFlagName)) * time.Second
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err = time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if e.Before(s) {
		return common.ErrStartAfterEnd
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.FindMissingSavedCandleIntervals(c.Context,
		&gctrpc.FindMissingCandlePeriodsRequest{
			ExchangeName: c.String(exchangeFlagName),
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: assetType,
			Start:     s.Format(common.SimpleTimeFormatWithTimezone),
			End:       e.Format(common.SimpleTimeFormatWithTimezone),
			Interval:  int64(candleInterval),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var shutdownCommand = &cli.Command{
	Name:   "shutdown",
	Usage:  "shuts down bot instance",
	Action: shutdown,
}

func shutdown(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.Shutdown(c.Context, &gctrpc.ShutdownRequest{})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getMarginRatesHistoryCommand = &cli.Command{
	Name:   "getmarginrateshistory",
	Usage:  "returns margin lending/borrow rates for a period",
	Action: getMarginRatesHistory,
	Flags: []cli.Flag{
		requiredExchangeAliasFlag,
		requiredAssetAliasFlag,
		&cli.StringFlag{
			Name:     currencyFlagName,
			Aliases:  []string{"c"},
			Required: true,
			Usage:    "must be an enabled currency",
		},
		&cli.StringFlag{
			Name:    startFlagName,
			Aliases: []string{"sd"},
			Usage:   startUsage,
			Value:   time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
		},
		&cli.StringFlag{
			Name:    endFlagName,
			Aliases: []string{"ed"},
			Usage:   endUsage,
			Value:   time.Now().Format(time.DateTime),
		},
		&cli.BoolFlag{
			Name:    "getpredictedrate",
			Aliases: []string{"p"},
			Usage:   "include the predicted upcoming rate in the response",
		},
		&cli.BoolFlag{
			Name:    "getlendingpayments",
			Aliases: []string{"lp"},
			Usage:   "retrieve and summarise your lending payments over the time period",
		},
		&cli.BoolFlag{
			Name:    "getborrowrates",
			Aliases: []string{"br"},
			Usage:   "retrieve borrowing rates",
		},
		&cli.BoolFlag{
			Name:    "getborrowcosts",
			Aliases: []string{"bc"},
			Usage:   "retrieve and summarise your borrowing costs over the time period",
		},
		&cli.BoolFlag{
			Name:    "includeallrates",
			Aliases: []string{"ar", "v", "verbose"},
			Usage:   "include a detailed slice of all lending/borrowing rates over the time period",
		},
	},
}

func getMarginRatesHistory(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	var err error

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, c.String(startFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %w", err)
	}
	e, err = time.ParseInLocation(time.DateTime, c.String(endFlagName), time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %w", err)
	}

	if err := common.StartEndTimeCheck(s, e); err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetMarginRatesHistory(c.Context,
		&gctrpc.GetMarginRatesHistoryRequest{
			Exchange:           c.String(exchangeFlagName),
			Asset:              assetType,
			Currency:           c.String(currencyFlagName),
			StartDate:          s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:            e.Format(common.SimpleTimeFormatWithTimezone),
			GetPredictedRate:   c.Bool("getpredictedrate"),
			GetLendingPayments: c.Bool("getlendingpayments"),
			GetBorrowRates:     c.Bool("getborrowrates"),
			GetBorrowCosts:     c.Bool("getborrowcosts"),
			IncludeAllRates:    c.Bool("includeallrates"),
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCurrencyTradeURLCommand = &cli.Command{
	Name:   "getcurrencytradeurl",
	Usage:  "returns the trading url of the instrument",
	Action: getCurrencyTradeURL,
	Flags: []cli.Flag{
		requiredExchangeAliasFlag,
		requiredAssetAliasFlag,
		requiredPairAliasFlag,
	},
}

func getCurrencyTradeURL(c *cli.Context) error {
	if c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	assetType := c.String(assetFlagName)

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	cp := c.String(pairFlagName)

	if !validPair(cp) {
		return errInvalidPair
	}
	p, err := currency.NewPairDelimiter(cp, pairDelimiter)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCurrencyTradeURL(c.Context,
		&gctrpc.GetCurrencyTradeURLRequest{
			Exchange: c.String(exchangeFlagName),
			Asset:    assetType,
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
