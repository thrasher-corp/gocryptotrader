package main

import (
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var (
	startTime, endTime, orderingDirection string
	limit                                 int
)

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
	Name:      "enablesubsystem",
	Usage:     "enables an engine subsystem",
	ArgsUsage: "<subsystem>",
	Action:    enableSubsystem,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "subsystem",
			Usage: "the subsystem to enable",
		},
	},
}

func enableSubsystem(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var subsystemName string
	if c.IsSet("subsystem") {
		subsystemName = c.String("subsystem")
	} else {
		subsystemName = c.Args().First()
	}

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
	Name:      "disablesubsystem",
	Usage:     "disables an engine subsystem",
	ArgsUsage: "<subsystem>",
	Action:    disableSubsystem,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "subsystem",
			Usage: "the subsystem to disable",
		},
	},
}

func disableSubsystem(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var subsystemName string
	if c.IsSet("subsystem") {
		subsystemName = c.String("subsystem")
	} else {
		subsystemName = c.Args().First()
	}

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
	Name:      "getexchanges",
	Usage:     "gets a list of enabled or available exchanges",
	ArgsUsage: "<enabled>",
	Action:    getExchanges,
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

	var enabledOnly bool
	if c.IsSet("enabled") {
		enabledOnly = c.Bool("enabled")
	}

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetExchanges(c.Context,
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

var enableExchangeCommand = &cli.Command{
	Name:      "enableexchange",
	Usage:     "enables an exchange",
	ArgsUsage: "<exchange>",
	Action:    enableExchange,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to enable",
		},
	},
}

func enableExchange(c *cli.Context) error {
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
	result, err := client.EnableExchange(c.Context,
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

var disableExchangeCommand = &cli.Command{
	Name:      "disableexchange",
	Usage:     "disables an exchange",
	ArgsUsage: "<exchange>",
	Action:    disableExchange,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to disable",
		},
	},
}

func disableExchange(c *cli.Context) error {
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
	result, err := client.DisableExchange(c.Context,
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

var getExchangeOTPCommand = &cli.Command{
	Name:      "getexchangeotp",
	Usage:     "gets a specific exchange OTP code",
	ArgsUsage: "<exchange>",
	Action:    getExchangeOTPCode,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the OTP code for",
		},
	},
}

func getExchangeOTPCode(c *cli.Context) error {
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
	result, err := client.GetExchangeOTPCode(c.Context,
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
	Name:      "getexchangeinfo",
	Usage:     "gets a specific exchanges info",
	ArgsUsage: "<exchange>",
	Action:    getExchangeInfo,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the info for",
		},
	},
}

func getExchangeInfo(c *cli.Context) error {
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
	result, err := client.GetExchangeInfo(c.Context,
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

var getTickerCommand = &cli.Command{
	Name:      "getticker",
	Usage:     "gets the ticker for a specific currency pair and exchange",
	ArgsUsage: "<exchange> <pair> <asset>",
	Action:    getTicker,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the ticker for",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to get the ticker for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type of the currency pair to get the ticker for",
		},
	},
}

func getTicker(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

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

	if !validPair(currencyPair) {
		return errInvalidPair
	}

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

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetTicker(c.Context,
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
	Name:      "getaccountbalances",
	Usage:     "gets the exchange account balances",
	ArgsUsage: "<exchange> <asset>",
	Action:    getAccountBalances,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "exchange",
			Usage:    "the exchange to get the account balances for",
			Required: true,
		},
		&cli.StringFlag{
			Name:     "asset",
			Usage:    "the asset type to get the account balances for",
			Required: true,
		},
	},
}

func getAccountBalances(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}
	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

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
			Exchange:  exchange,
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
	Name:      "getaccountbalancesstream",
	Usage:     "gets the account balances stream for a specific exchange",
	ArgsUsage: "<exchange> <asset>",
	Action:    getAccountBalancesStream,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the account balances stream from",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type to get the account balances stream for",
		},
	},
}

func getAccountBalancesStream(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

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

		err = clearScreen()
		if err != nil {
			return err
		}

		fmt.Printf("Account balance stream for %s:\n\n", exchangeName)

		fmt.Printf("%+v", resp)
	}
}

var updateAccountBalancesCommand = &cli.Command{
	Name:      "updateaccountbalances",
	Usage:     "updates the exchange account balances",
	ArgsUsage: "<exchange> <asset>",
	Action:    updateAccountBalances,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the account balances for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type to get the account balances for",
		},
	},
}

func updateAccountBalances(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

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
			Exchange:  exchange,
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
	Name:      "addportfolioaddress",
	Usage:     "adds an address to the portfolio",
	ArgsUsage: "<address> <coin_type> <description> <balance> <cold_storage> <supported_exchanges> ",
	Action:    addPortfolioAddress,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "address",
			Usage: "the address to add to the portfolio",
		},
		&cli.StringFlag{
			Name:  "coin_type",
			Usage: "the coin type e.g ('BTC')",
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "description of the address",
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var address string
	var coinType string
	var description string
	var balance float64
	var supportedExchanges string
	var coldstorage bool

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
		description = c.String("description")
	} else {
		description = c.Args().Get(2)
	}
	var err error
	if c.IsSet("balance") {
		balance = c.Float64("balance")
	} else if c.Args().Get(3) != "" {
		balance, err = strconv.ParseFloat(c.Args().Get(3), 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("cold_storage") {
		coldstorage = c.Bool("cold_storage")
	} else {
		tv, errBool := strconv.ParseBool(c.Args().Get(4))
		if errBool == nil {
			coldstorage = tv
		}
	}

	if c.IsSet("supported_exchanges") {
		supportedExchanges = c.String("supported_exchanges")
	} else {
		supportedExchanges = c.Args().Get(5)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.AddPortfolioAddress(c.Context,
		&gctrpc.AddPortfolioAddressRequest{
			Address:            address,
			CoinType:           coinType,
			Description:        description,
			Balance:            balance,
			SupportedExchanges: supportedExchanges,
			ColdStorage:        coldstorage,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var removePortfolioAddressCommand = &cli.Command{
	Name:      "removeportfolioaddress",
	Usage:     "removes an address from the portfolio",
	ArgsUsage: "<address> <coin_type> <description>",
	Action:    removePortfolioAddress,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "address",
			Usage: "the address to add to the portfolio",
		},
		&cli.StringFlag{
			Name:  "coin_type",
			Usage: "the coin type e.g ('BTC')",
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "description of the address",
		},
	},
}

func removePortfolioAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

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
		description = c.String("description")
	} else {
		description = c.Args().Get(2)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.RemovePortfolioAddress(c.Context,
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
	Name:      "getorders",
	Usage:     "gets the open orders",
	ArgsUsage: "<exchange> <asset> <pair> <start> <end>",
	Action:    getOrders,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get orders for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type to get orders for",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to get orders for",
		},
		&cli.StringFlag{
			Name:        "start",
			Usage:       "start date, optional. Will filter any results before this date",
			Value:       time.Now().AddDate(0, -1, 0).Format(time.DateTime),
			Destination: &startTime,
		},
		&cli.StringFlag{
			Name:        "end",
			Usage:       "end date, optional. Will filter any results after this date",
			Value:       time.Now().Format(time.DateTime),
			Destination: &endTime,
		},
	},
}

func getOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var assetType string
	var currencyPair string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if !c.IsSet("start") {
		if c.Args().Get(3) != "" {
			startTime = c.Args().Get(3)
		}
	}

	if !c.IsSet("end") {
		if c.Args().Get(4) != "" {
			endTime = c.Args().Get(4)
		}
	}
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
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
		Exchange:  exchangeName,
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
	Name:      "getmanagedorders",
	Usage:     "gets the current orders from the order manager",
	ArgsUsage: "<exchange> <asset> <pair>",
	Action:    getManagedOrders,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get orders for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type to get orders for",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to get orders for",
		},
	},
}

func getManagedOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var assetType string
	var currencyPair string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}

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

var getOrderCommand = &cli.Command{
	Name:      "getorder",
	Usage:     "gets the specified order info",
	ArgsUsage: "<exchange> <asset> <pair> <order_id>",
	Action:    getOrder,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the order for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "required asset type",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the pair to retrieve",
		},
		&cli.StringFlag{
			Name:  "order_id",
			Usage: "the order id to retrieve",
		},
	},
}

func getOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var orderID string
	var currencyPair string
	var assetType string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}
	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("order_id") {
		orderID = c.String("order_id")
	} else {
		orderID = c.Args().Get(3)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetOrder(c.Context, &gctrpc.GetOrderRequest{
		Exchange: exchangeName,
		OrderId:  orderID,
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
	Name:      "submitorder",
	Usage:     "submit order submits an exchange order",
	ArgsUsage: "<exchange> <pair> <side> <type> <amount> <price> <client_id>",
	Action:    submitOrder,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to submit the order for",
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
			Name:  "type",
			Usage: "the order type (MARKET OR LIMIT)",
		},
		&cli.Float64Flag{
			Name:  "amount",
			Usage: "the amount for the order",
		},
		&cli.Float64Flag{
			Name:  "price",
			Usage: "the price for the order",
		},
		&cli.StringFlag{
			Name:  "client_id",
			Usage: "the optional client order ID",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "required asset type",
		},
		&cli.StringFlag{
			Name:     "margintype",
			Usage:    "required asset type",
			Required: false,
		},
	},
}

func submitOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var currencyPair string
	var orderSide string
	var orderType string
	var amount float64
	var price float64
	var clientID string
	var assetType string
	var marginType string

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

	if c.IsSet("type") {
		orderType = c.String("type")
	} else {
		orderType = c.Args().Get(3)
	}

	if orderType == "" {
		return errors.New("order type must be set")
	}

	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else if c.Args().Get(4) != "" {
		var err error
		amount, err = strconv.ParseFloat(c.Args().Get(4), 64)
		if err != nil {
			return err
		}
	}

	if amount == 0 {
		return errors.New("amount must be set")
	}

	// price is optional for market orders
	if c.IsSet("price") {
		price = c.Float64("price")
	} else if c.Args().Get(5) != "" {
		var err error
		price, err = strconv.ParseFloat(c.Args().Get(5), 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("client_id") {
		clientID = c.String("client_id")
	} else {
		clientID = c.Args().Get(6)
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(7)
	}

	if c.IsSet("margintype") {
		marginType = c.String("margintype")
	} else {
		marginType = c.Args().Get(8)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	marginType = strings.ToLower(marginType)
	if !margin.IsValidString(marginType) {
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
		AssetType: assetType,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var simulateOrderCommand = &cli.Command{
	Name:      "simulateorder",
	Usage:     "simulate order simulates an exchange order",
	ArgsUsage: "<exchange> <pair> <side> <amount>",
	Action:    simulateOrder,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to simulate the order for",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair",
		},
		&cli.StringFlag{
			Name:  "side",
			Usage: "the order side to use (BUY OR SELL)",
		},
		&cli.Float64Flag{
			Name:  "amount",
			Usage: "the amount for the order",
		},
	},
}

func simulateOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var currencyPair string
	var orderSide string
	var amount float64

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
		return errors.New("side must be set")
	}

	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else if c.Args().Get(3) != "" {
		var err error
		amount, err = strconv.ParseFloat(c.Args().Get(3), 64)
		if err != nil {
			return err
		}
	}

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
		Exchange: exchangeName,
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
	Name:      "cancelorder",
	Usage:     "cancel order cancels an exchange order",
	ArgsUsage: "<exchange> <account_id> <order_id> <pair> <asset> <side>",
	Action:    cancelOrder,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to cancel the order for",
		},
		&cli.StringFlag{
			Name:  "account_id",
			Usage: "the account id",
		},
		&cli.StringFlag{
			Name:  "order_id",
			Usage: "the order id",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to cancel the order for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type",
		},
		&cli.StringFlag{
			Name:  "side",
			Usage: "the order side",
		},
	},
}

func cancelOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var accountID string
	var orderID string
	var currencyPair string
	var assetType string
	var orderSide string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("account_id") {
		accountID = c.String("account_id")
	} else {
		accountID = c.Args().Get(1)
	}

	if c.IsSet("order_id") {
		orderID = c.String("order_id")
	} else {
		orderID = c.Args().Get(2)
	}

	if orderID == "" {
		return errors.New("an order ID must be set")
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(3)
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(4)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("side") {
		orderSide = c.String("side")
	} else {
		orderSide = c.Args().Get(5)
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
		Exchange:  exchangeName,
		AccountId: accountID,
		OrderId:   orderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: assetType,
		Side:      orderSide,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var cancelBatchOrdersCommand = &cli.Command{
	Name:      "cancelbatchorders",
	Usage:     "cancel batch orders cancels a list of exchange orders (comma separated)",
	ArgsUsage: "<exchange> <account_id> <order_ids> <pair> <asset> <side>",
	Action:    cancelBatchOrders,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to cancel the order for",
		},
		&cli.StringFlag{
			Name:  "account_id",
			Usage: "the account id",
		},
		&cli.StringFlag{
			Name:  "order_ids",
			Usage: "the comma separated orders id-s",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair to cancel the order for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type",
		},
		&cli.StringFlag{
			Name:  "side",
			Usage: "the order side",
		},
	},
}

func cancelBatchOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var accountID string
	var orderID string
	var currencyPair string
	var assetType string
	var orderSide string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("account_id") {
		accountID = c.String("account_id")
	} else {
		accountID = c.Args().Get(1)
	}

	if c.IsSet("order_ids") {
		orderID = c.String("order_ids")
	} else {
		orderID = c.Args().Get(2)
	}

	if orderID == "" {
		return errors.New("an order ID must be set")
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(3)
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(4)
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("side") {
		orderSide = c.String("side")
	} else {
		orderSide = c.Args().Get(5)
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
		Exchange:  exchangeName,
		AccountId: accountID,
		OrdersId:  orderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: assetType,
		Side:      orderSide,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var cancelAllOrdersCommand = &cli.Command{
	Name:      "cancelallorders",
	Usage:     "cancels all orders (all or by exchange name)",
	ArgsUsage: "<exchange>",
	Action:    cancelAllOrders,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to cancel all orders on",
		},
	},
}

var modifyOrderCommand = &cli.Command{
	Name:      "modifyorder",
	Usage:     "modify price and/or amount of a previously submitted order",
	ArgsUsage: "<exchange> <asset> <pair> <order_id>",
	Action:    modifyOrder,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "exchange this order is submitted to",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "required asset type",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "required trading pair",
		},
		&cli.StringFlag{
			Name:  "order_id",
			Usage: "id of the order to be modified",
		},
		&cli.Float64Flag{
			Name:  "price",
			Usage: "new order price",
		},
		&cli.Float64Flag{
			Name:  "amount",
			Usage: "new order amount",
		},
	},
}

func cancelAllOrders(c *cli.Context) error {
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
	result, err := client.CancelAllOrders(c.Context, &gctrpc.CancelAllOrdersRequest{
		Exchange: exchangeName,
	})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

func modifyOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	// Parse positional arguments.
	var exchangeName string
	var orderID string
	var currencyPair string
	var assetType string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}
	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		currencyPair = c.Args().Get(2)
	}
	if !validPair(currencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if c.IsSet("order_id") {
		orderID = c.String("order_id")
	} else {
		orderID = c.Args().Get(3)
	}

	// Parse optional flags.
	var price float64
	var amount float64

	if c.IsSet("price") {
		price = c.Float64("price")
	}
	if c.IsSet("amount") {
		amount = c.Float64("amount")
	}
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
		Exchange: exchangeName,
		OrderId:  orderID,
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
	Name:      "addevent",
	Usage:     "adds an event",
	ArgsUsage: "<exchange> <item> <condition> <price> <check_bids> <check_bids_and_asks> <orderbook_amount> <pair> <asset> <action>",
	Action:    addEvent,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to add an event for",
		},
		&cli.StringFlag{
			Name:  "item",
			Usage: "the item to trigger the event",
		},
		&cli.StringFlag{
			Name:  "condition",
			Usage: "the condition for the event",
		},
		&cli.Float64Flag{
			Name:  "price",
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
		&cli.StringFlag{
			Name:  "pair",
			Usage: "the currency pair",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type",
		},
		&cli.StringFlag{
			Name:  "action",
			Usage: "the action for the event to perform upon trigger",
		},
	},
}

func addEvent(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var item string
	var condition string
	var price float64
	var checkBids bool
	var checkAsks bool
	var orderbookAmount float64
	var currencyPair string
	var assetType string
	var action string

	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		return errors.New("exchange name is required")
	}

	if c.IsSet("item") {
		item = c.String("item")
	} else {
		return errors.New("item is required")
	}

	if c.IsSet("condition") {
		condition = c.String("condition")
	} else {
		return errors.New("condition is required")
	}

	if c.IsSet("price") {
		price = c.Float64("price")
	}

	if c.IsSet("check_bids") {
		checkBids = c.Bool("check_bids")
	}

	if c.IsSet("check_asks") {
		checkAsks = c.Bool("check_asks")
	}

	if c.IsSet("orderbook_amount") {
		orderbookAmount = c.Float64("orderbook_amount")
	}

	if c.IsSet("pair") {
		currencyPair = c.String("pair")
	} else {
		return errors.New("currency pair is required")
	}

	if !validPair(currencyPair) {
		return errInvalidPair
	}

	if c.IsSet("asset") {
		assetType = c.String("asset")
	}

	assetType = strings.ToLower(assetType)
	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("action") {
		action = c.String("action")
	} else {
		return errors.New("action is required")
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
		Exchange: exchangeName,
		Item:     item,
		ConditionParams: &gctrpc.ConditionParams{
			Condition:       condition,
			Price:           price,
			CheckBids:       checkBids,
			CheckAsks:       checkAsks,
			OrderbookAmount: orderbookAmount,
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

var removeEventCommand = &cli.Command{
	Name:      "removeevent",
	Usage:     "removes an event",
	ArgsUsage: "<event_id>",
	Action:    removeEvent,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "event_id",
			Usage: "the event id to remove",
		},
	},
}

func removeEvent(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var eventID int64
	if c.IsSet("event_id") {
		eventID = c.Int64("event_id")
	} else if c.Args().Get(0) != "" {
		var err error
		eventID, err = strconv.ParseInt(c.Args().Get(0), 10, 64)
		if err != nil {
			return err
		}
	}

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
	Name:      "getcryptocurrencydepositaddresses",
	Usage:     "gets the cryptocurrency deposit addresses for an exchange",
	ArgsUsage: "<exchange>",
	Action:    getCryptocurrencyDepositAddresses,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the cryptocurrency deposit addresses for",
		},
	},
}

func getCryptocurrencyDepositAddresses(c *cli.Context) error {
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
	result, err := client.GetCryptocurrencyDepositAddresses(c.Context,
		&gctrpc.GetCryptocurrencyDepositAddressesRequest{Exchange: exchangeName})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCryptocurrencyDepositAddressCommand = &cli.Command{
	Name:      "getcryptocurrencydepositaddress",
	Usage:     "gets the cryptocurrency deposit address for an exchange and cryptocurrency",
	ArgsUsage: "<exchange> <cryptocurrency> <chain> <bypass>",
	Action:    getCryptocurrencyDepositAddress,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the cryptocurrency deposit address for",
		},
		&cli.StringFlag{
			Name:  "cryptocurrency",
			Usage: "the cryptocurrency to get the deposit address for",
		},
		&cli.StringFlag{
			Name:  "chain",
			Usage: "the chain to use for the deposit",
		},
		&cli.BoolFlag{
			Name:  "bypass",
			Usage: "whether to bypass the deposit address manager cache if enabled",
		},
	},
}

func getCryptocurrencyDepositAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
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
	} else if c.Args().Get(1) != "" {
		cryptocurrency = c.Args().Get(1)
	}

	if cryptocurrency == "" {
		return errors.New("cryptocurrency must be set")
	}

	var chain string
	if c.IsSet("chain") {
		chain = c.String("chain")
	} else if c.Args().Get(2) != "" {
		chain = c.Args().Get(2)
	}

	var bypass bool
	if c.IsSet("bypass") {
		bypass = c.Bool("bypass")
	} else if c.Args().Get(3) != "" {
		b, err := strconv.ParseBool(c.Args().Get(3))
		if err != nil {
			return err
		}
		bypass = b
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCryptocurrencyDepositAddress(c.Context,
		&gctrpc.GetCryptocurrencyDepositAddressRequest{
			Exchange:       exchangeName,
			Cryptocurrency: cryptocurrency,
			Chain:          chain,
			Bypass:         bypass,
		},
	)
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getAvailableTransferChainsCommand = &cli.Command{
	Name:      "getavailabletransferchains",
	Usage:     "gets the available transfer chains (deposits and withdrawals) for the desired exchange and cryptocurrency",
	ArgsUsage: "<exchange> <cryptocurrency>",
	Action:    getAvailableTransferChains,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the available transfer chains",
		},
		&cli.StringFlag{
			Name:  "cryptocurrency",
			Usage: "the cryptocurrency to get the available transfer chains for",
		},
	},
}

func getAvailableTransferChains(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
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
	} else if c.Args().Get(1) != "" {
		cryptocurrency = c.Args().Get(1)
	}

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

var withdrawCryptocurrencyFundsCommand = &cli.Command{
	Name:      "withdrawcryptofunds",
	Usage:     "withdraws cryptocurrency funds from the desired exchange",
	ArgsUsage: "<exchange> <currency> <amount> <address> <addresstag> <fee> <description> <chain>",
	Action:    withdrawCryptocurrencyFunds,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to withdraw from",
		},
		&cli.StringFlag{
			Name:  "currency",
			Usage: "the cryptocurrency to withdraw funds from",
		},
		&cli.StringFlag{
			Name:  "address",
			Usage: "address to withdraw to",
		},
		&cli.StringFlag{
			Name:  "addresstag",
			Usage: "address tag/memo",
		},
		&cli.Float64Flag{
			Name:  "amount",
			Usage: "amount of funds to withdraw",
		},
		&cli.Float64Flag{
			Name:  "fee",
			Usage: "fee to submit with request",
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "description to submit with request",
		},
		&cli.StringFlag{
			Name:  "chain",
			Usage: "chain to use for the withdrawal",
		},
	},
}

func withdrawCryptocurrencyFunds(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange, cur, address, addressTag, chain, description string
	var amount, fee float64

	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else if c.Args().Get(0) != "" {
		exchange = c.Args().Get(0)
	}

	if c.IsSet("currency") {
		cur = c.String("currency")
	} else if c.Args().Get(1) != "" {
		cur = c.Args().Get(1)
	}

	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else if c.Args().Get(2) != "" {
		amountStr, err := strconv.ParseFloat(c.Args().Get(2), 64)
		if err == nil {
			amount = amountStr
		}
	}

	if c.IsSet("address") {
		address = c.String("address")
	} else if c.Args().Get(3) != "" {
		address = c.Args().Get(3)
	}

	if c.IsSet("addresstag") {
		addressTag = c.String("addresstag")
	} else if c.Args().Get(4) != "" {
		addressTag = c.Args().Get(4)
	}

	if c.IsSet("fee") {
		fee = c.Float64("fee")
	} else if c.Args().Get(5) != "" {
		feeStr, err := strconv.ParseFloat(c.Args().Get(5), 64)
		if err == nil {
			fee = feeStr
		}
	}

	if c.IsSet("description") {
		description = c.String("description")
	} else if c.Args().Get(6) != "" {
		description = c.Args().Get(6)
	}

	if c.IsSet("chain") {
		chain = c.String("chain")
	} else if c.Args().Get(7) != "" {
		chain = c.Args().Get(7)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.WithdrawCryptocurrencyFunds(c.Context,
		&gctrpc.WithdrawCryptoRequest{
			Exchange:    exchange,
			Currency:    cur,
			Address:     address,
			AddressTag:  addressTag,
			Amount:      amount,
			Fee:         fee,
			Description: description,
			Chain:       chain,
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

var withdrawFiatFundsCommand = &cli.Command{
	Name:      "withdrawfiatfunds",
	Usage:     "withdraws fiat funds from the desired exchange",
	ArgsUsage: "<exchange> <currency> <amount> <bankaccount id> <description>",
	Action:    withdrawFiatFunds,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to withdraw from",
		},
		&cli.StringFlag{
			Name:  "currency",
			Usage: "the fiat currency to withdraw funds from",
		},
		&cli.Float64Flag{
			Name:  "amount",
			Usage: "amount of funds to withdraw",
		},
		&cli.StringFlag{
			Name:  "bankaccountid",
			Usage: "ID of bank account to use",
		},
		&cli.StringFlag{
			Name:  "description",
			Usage: "description to submit with request",
		},
	},
}

func withdrawFiatFunds(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange, cur, description, bankAccountID string
	var amount float64

	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else if c.Args().Get(0) != "" {
		exchange = c.Args().Get(0)
	}

	if c.IsSet("currency") {
		cur = c.String("currency")
	} else if c.Args().Get(1) != "" {
		cur = c.Args().Get(1)
	}

	if c.IsSet("amount") {
		amount = c.Float64("amount")
	} else if c.Args().Get(2) != "" {
		amountStr, err := strconv.ParseFloat(c.Args().Get(2), 64)
		if err == nil {
			amount = amountStr
		}
	}

	if c.IsSet("bankaccountid") {
		bankAccountID = c.String("bankaccountid")
	} else if c.Args().Get(3) != "" {
		bankAccountID = c.Args().Get(3)
	}

	if c.IsSet("description") {
		description = c.String("description")
	} else if c.Args().Get(4) != "" {
		description = c.Args().Get(4)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WithdrawFiatFunds(c.Context,
		&gctrpc.WithdrawFiatRequest{
			Exchange:      exchange,
			Currency:      cur,
			Amount:        amount,
			Description:   description,
			BankAccountId: bankAccountID,
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
	ArgsUsage: "<type> <args>",
	Subcommands: []*cli.Command{
		{
			Name:      "byid",
			Usage:     "id",
			ArgsUsage: "<id>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "id",
					Usage: "withdrawal id",
				},
			},
			Action: withdrawlRequestByID,
		},
		{
			Name:      "byexchangeid",
			Usage:     "exchange id",
			ArgsUsage: "<id>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "exchange name",
				},
				&cli.StringFlag{
					Name:  "id",
					Usage: "withdrawal id",
				},
			},
			Action: withdrawlRequestByExchangeID,
		},
		{
			Name:      "byexchange",
			Usage:     "exchange limit",
			ArgsUsage: "<id>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "exchange name",
				},
				&cli.Int64Flag{
					Name:  "limit",
					Usage: "max number of withdrawals to return",
				},
				&cli.StringFlag{
					Name:  "currency",
					Usage: "<currency>",
				},
				&cli.StringFlag{
					Name:  "asset",
					Usage: "the asset type of the currency pair",
				},
			},
			Action: withdrawlRequestByExchangeID,
		},
		{
			Name:      "bydate",
			Usage:     "exchange start end limit",
			ArgsUsage: "<exchange> <start> <end> <limit>",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:  "exchange",
					Usage: "the currency used in to withdraw",
				},
				&cli.StringFlag{
					Name:        "start",
					Usage:       "the start date to get withdrawals from. Any withdrawal before this date will be filtered",
					Value:       time.Now().AddDate(0, -1, 0).Format(time.DateTime),
					Destination: &startTime,
				},
				&cli.StringFlag{
					Name:        "end",
					Usage:       "the end date to get withdrawals from. Any withdrawal after this date will be filtered",
					Value:       time.Now().Format(time.DateTime),
					Destination: &endTime,
				},
				&cli.Int64Flag{
					Name:  "limit",
					Usage: "max number of withdrawals to return",
				},
			},
			Action: withdrawlRequestByDate,
		},
	},
}

func withdrawlRequestByID(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var ID string
	if c.IsSet("id") {
		ID = c.String("id")
	} else {
		ID = c.Args().First()
	}

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

func withdrawlRequestByExchangeID(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange, ccy, assetType string
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	var limit, limitStr int64
	var ID string
	var err error
	if c.Command.Name == "byexchangeid" {
		if c.IsSet("id") {
			ID = c.String("id")
		} else {
			ID = c.Args().Get(1)
		}
		if ID == "" {
			return errors.New("an ID must be specified")
		}
		limit = 1
	} else {
		if c.IsSet("limit") {
			limit = c.Int64("limit")
		} else if c.Args().Get(1) != "" {
			limitStr, err = strconv.ParseInt(c.Args().Get(1), 10, 64)
			if err != nil {
				return err
			}
			if limitStr > math.MaxInt32 {
				return fmt.Errorf("limit greater than max size: %v", math.MaxInt32)
			}
			limit = limitStr
		}

		if c.IsSet("currency") {
			ccy = c.String("currency")
		}

		if c.IsSet("asset") {
			assetType = c.String("asset")
		}

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
			Limit:     int32(limit), //nolint:gosec // TODO: SQL boiler's QueryMode limit only accepts the int type
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

func withdrawlRequestByDate(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchange string
	var limit, limitStr int64
	var err error
	if c.IsSet("exchange") {
		exchange = c.String("exchange")
	} else {
		exchange = c.Args().First()
	}

	if !c.IsSet("start") {
		if c.Args().Get(1) != "" {
			startTime = c.Args().Get(1)
		}
	}

	if !c.IsSet("end") {
		if c.Args().Get(2) != "" {
			endTime = c.Args().Get(2)
		}
	}

	if c.IsSet("limit") {
		limit = c.Int64("limit")
	} else if c.Args().Get(3) != "" {
		limitStr, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
		if limitStr > math.MaxInt32 {
			return fmt.Errorf("limit greater than max size: %v", math.MaxInt32)
		}
		limit = limitStr
	}

	s, err := time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
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
			Exchange: exchange,
			Start:    s.Format(common.SimpleTimeFormatWithTimezone),
			End:      e.Format(common.SimpleTimeFormatWithTimezone),
			Limit:    int32(limit), //nolint:gosec // TODO: SQL boiler's QueryMode limit only accepts the int type
		},
	)
	if err != nil {
		return err
	}
	jsonOutput(result)
	return nil
}

var getLoggerDetailsCommand = &cli.Command{
	Name:      "getloggerdetails",
	Usage:     "gets an individual loggers details",
	ArgsUsage: "<logger>",
	Action:    getLoggerDetails,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "logger",
			Usage: "logger to get level details of",
		},
	},
}

func getLoggerDetails(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var logger string
	if c.IsSet("logger") {
		logger = c.String("logger")
	} else {
		logger = c.Args().First()
	}

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
	Name:      "setloggerdetails",
	Usage:     "sets an individual loggers details",
	ArgsUsage: "<logger> <flags>",
	Action:    setLoggerDetails,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "logger",
			Usage: "logger to get level details of",
		},
		&cli.StringFlag{
			Name:  "flags",
			Usage: "pipe separated value of loggers e.g INFO|WARN",
		},
	},
}

func setLoggerDetails(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var logger string
	var level string

	if c.IsSet("logger") {
		logger = c.String("logger")
	} else {
		logger = c.Args().First()
	}

	if logger == "" {
		return errors.New("a logger must be specified")
	}

	if c.IsSet("level") {
		level = c.String("level")
	} else {
		level = c.Args().Get(1)
	}

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
	Name:      "gettickerstream",
	Usage:     "gets the ticker stream for a specific currency pair and exchange",
	ArgsUsage: "<exchange> <pair> <asset>",
	Action:    getTickerStream,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the ticker from",
		},
		&cli.StringFlag{
			Name:  "pair",
			Usage: "currency pair",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type of the currency pair",
		},
	},
}

func getTickerStream(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	var pair string
	var assetType string

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
	result, err := client.GetTickerStream(c.Context,
		&gctrpc.GetTickerStreamRequest{
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

		fmt.Printf("Ticker stream for %s %s:\n", exchangeName,
			resp.Pair.String())
		fmt.Println()

		fmt.Printf("LAST: %f\n HIGH: %f\n LOW: %f\n BID: %f\n ASK: %f\n VOLUME: %f\n PRICEATH: %f\n LASTUPDATED: %d\n",
			resp.Last,
			resp.High,
			resp.Low,
			resp.Bid,
			resp.Ask,
			resp.Volume,
			resp.PriceAth,
			resp.LastUpdated)
	}
}

var getExchangeTickerStreamCommand = &cli.Command{
	Name:      "getexchangetickerstream",
	Usage:     "gets a stream for all tickers associated with an exchange",
	ArgsUsage: "<exchange>",
	Action:    getExchangeTickerStream,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the ticker from",
		},
	},
}

func getExchangeTickerStream(c *cli.Context) error {
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
	result, err := client.GetExchangeTickerStream(c.Context,
		&gctrpc.GetExchangeTickerStreamRequest{
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

		fmt.Printf("Ticker stream for %s %s:\n",
			exchangeName,
			resp.Pair.String())

		fmt.Printf("LAST: %f HIGH: %f LOW: %f BID: %f ASK: %f VOLUME: %f PRICEATH: %f LASTUPDATED: %d\n",
			resp.Last,
			resp.High,
			resp.Low,
			resp.Bid,
			resp.Ask,
			resp.Volume,
			resp.PriceAth,
			resp.LastUpdated)
	}
}

var getAuditEventCommand = &cli.Command{
	Name:      "getauditevent",
	Usage:     "gets audit events matching query parameters",
	ArgsUsage: "<starttime> <endtime> <orderby> <limit>",
	Action:    getAuditEvent,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "start",
			Aliases:     []string{"s"},
			Usage:       "start date to search",
			Value:       time.Now().Add(-time.Hour).Format(time.DateTime),
			Destination: &startTime,
		},
		&cli.StringFlag{
			Name:        "end",
			Aliases:     []string{"e"},
			Usage:       "end time to search",
			Value:       time.Now().Format(time.DateTime),
			Destination: &endTime,
		},
		&cli.StringFlag{
			Name:        "order",
			Aliases:     []string{"o"},
			Usage:       "order results by ascending/descending",
			Value:       "asc",
			Destination: &orderingDirection,
		},
		&cli.IntFlag{
			Name:        "limit",
			Aliases:     []string{"l"},
			Usage:       "how many results to retrieve",
			Value:       100,
			Destination: &limit,
		},
	},
}

func getAuditEvent(c *cli.Context) error {
	if !c.IsSet("start") {
		if c.Args().Get(0) != "" {
			startTime = c.Args().Get(0)
		}
	}

	if !c.IsSet("end") {
		if c.Args().Get(1) != "" {
			endTime = c.Args().Get(1)
		}
	}

	if !c.IsSet("order") {
		if c.Args().Get(2) != "" {
			orderingDirection = c.Args().Get(2)
		}
	}

	if !c.IsSet("limit") {
		if c.Args().Get(3) != "" {
			limitStr, err := strconv.ParseInt(c.Args().Get(3), 10, 64)
			if err == nil {
				limit = int(limitStr)
			}
		}
	}

	s, err := time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}

	e, err := time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
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
			Limit:     int32(limit), //nolint:gosec // TODO: SQL boiler's QueryMode limit only accepts the int type
			OrderBy:   orderingDirection,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var (
	uuid, filename, path string
	gctScriptCommand     = &cli.Command{
		Name:      "script",
		Usage:     "execute scripting management command",
		ArgsUsage: "<command> <args>",
		Subcommands: []*cli.Command{
			{
				Name:      "execute",
				Usage:     "execute script filename",
				ArgsUsage: "<filename> <path>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "filename",
						Usage:       "the script filename",
						Destination: &filename,
					},
					&cli.StringFlag{
						Name:        "path",
						Usage:       "the directory of the script file",
						Destination: &path,
					},
				},
				Action: gctScriptExecute,
			},
			{
				Name:  "query",
				Usage: "query running virtual machine",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "uuid",
						Usage:       "the unique id of the script in memory",
						Destination: &uuid,
					},
				},
				Action: gctScriptQuery,
			},
			{
				Name:  "read",
				Usage: "read script",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "name",
						Usage:       "the script name",
						Destination: &uuid,
					},
				},
				Action: gctScriptRead,
			},
			{
				Name:   "status",
				Usage:  "get status of running scripts",
				Action: gctScriptStatus,
			},
			{
				Name:   "list",
				Usage:  "lists all scripts in default scriptpath",
				Action: gctScriptList,
			},
			{
				Name:  "stop",
				Usage: "terminate running script",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "uuid",
						Usage:       "the unique id of the script in memory",
						Destination: &uuid,
					},
				},
				Action: gctScriptStop,
			},
			{
				Name:   "stopall",
				Usage:  "terminate running script",
				Action: gctScriptStopAll,
			},
			{
				Name:  "upload",
				Usage: "upload a new script/archive",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:        "path",
						Usage:       "<path> to single script or zip collection",
						Destination: &filename,
					},
					&cli.BoolFlag{
						Name:  "overwrite",
						Usage: "<true/false>",
					},
					&cli.BoolFlag{
						Name:  "archived",
						Usage: "<true/false>",
					},
				},
				Action: gctScriptUpload,
			},
			{
				Name:  "autoload",
				Usage: "add or remove script from autoload list",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "command",
						Usage: "<add/remove>",
					},
					&cli.StringFlag{
						Name:  "script",
						Usage: "<script name>",
					},
				},
				Action: gctScriptAutoload,
			},
		},
	}
)

func gctScriptAutoload(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var command, script string
	var status bool
	if !c.IsSet("command") {
		if c.Args().Get(0) != "" {
			command = c.Args().Get(0)
		}
	}

	if !c.IsSet("script") {
		if c.Args().Get(1) != "" {
			script = c.Args().Get(1)
		}
	}

	switch command {
	case "add":
		status = false
	case "remove":
		status = true
	default:
		return cli.ShowSubcommandHelp(c)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptAutoLoadToggle(c.Context,
		&gctrpc.GCTScriptAutoLoadRequest{
			Script: script,
			Status: status,
		})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)
	return nil
}

func gctScriptExecute(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	if !c.IsSet("filename") {
		if c.Args().Get(0) != "" {
			filename = c.Args().Get(0)
		}
	}

	if !c.IsSet("path") {
		if c.Args().Get(1) != "" {
			path = c.Args().Get(1)
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptExecute(c.Context,
		&gctrpc.GCTScriptExecuteRequest{
			Script: &gctrpc.GCTScript{
				Name: filename,
				Path: path,
			},
		})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)

	return nil
}

func gctScriptStatus(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptStatus(c.Context,
		&gctrpc.GCTScriptStatusRequest{})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)
	return nil
}

func gctScriptList(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptListAll(c.Context,
		&gctrpc.GCTScriptListAllRequest{})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)
	return nil
}

func gctScriptStop(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	if !c.IsSet("uuid") {
		if c.Args().Get(0) != "" {
			uuid = c.Args().Get(0)
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptStop(c.Context,
		&gctrpc.GCTScriptStopRequest{
			Script: &gctrpc.GCTScript{Uuid: uuid},
		})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)
	return nil
}

func gctScriptStopAll(c *cli.Context) error {
	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptStopAll(c.Context,
		&gctrpc.GCTScriptStopAllRequest{})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)
	return nil
}

func gctScriptRead(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	if !c.IsSet("name") {
		if c.Args().Get(0) != "" {
			uuid = c.Args().Get(0)
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptReadScript(c.Context,
		&gctrpc.GCTScriptReadScriptRequest{
			Script: &gctrpc.GCTScript{
				Name: uuid,
			},
		})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)
	return nil
}

func gctScriptQuery(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	if !c.IsSet("uuid") {
		if c.Args().Get(0) != "" {
			uuid = c.Args().Get(0)
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	executeCommand, err := client.GCTScriptQuery(c.Context,
		&gctrpc.GCTScriptQueryRequest{
			Script: &gctrpc.GCTScript{
				Uuid: uuid,
			},
		})
	if err != nil {
		return err
	}

	jsonOutput(executeCommand)

	return nil
}

func gctScriptUpload(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var overwrite bool
	var archived bool
	if !c.IsSet("path") {
		if c.Args().Get(0) != "" {
			filename = c.Args().Get(0)
		}
	}

	if c.IsSet("overwrite") {
		overwrite = c.Bool("overwrite")
	} else {
		ow, err := strconv.ParseBool(c.Args().Get(1))
		if err == nil {
			overwrite = ow
		}
	}

	if c.IsSet("archived") {
		archived = c.Bool("archived")
	} else {
		ow, err := strconv.ParseBool(c.Args().Get(1))
		if err == nil {
			archived = ow
		}
	}

	if filepath.Ext(filename) != common.GctExt && filepath.Ext(filename) != ".zip" {
		return errors.New("file type must be gct or zip")
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)
	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	data, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	uploadCommand, err := client.GCTScriptUpload(c.Context,
		&gctrpc.GCTScriptUploadRequest{
			ScriptName: filepath.Base(file.Name()),
			Data:       data,
			Archived:   archived,
			Overwrite:  overwrite,
		})
	if err != nil {
		return err
	}

	jsonOutput(uploadCommand)
	return nil
}

const klineMessage = `interval in seconds. supported values are: 15, 60(1min), 180(3min), 300(5min), 600(10min),
		900(15min) 1800(30min), 3600(1h), 7200(2h), 14400(4h), 21600(6h), 28800(8h), 43200(12h),
		86400(1d), 259200(3d) 604800(1w), 1209600(2w), 1296000(15d), 2592000(1M), 31536000(1Y)`

var (
	candleRangeSize, candleGranularity int64
	getHistoricCandlesCommand          = &cli.Command{
		Name:      "gethistoriccandles",
		Usage:     "gets historical candles for the specified granularity up to range size time from now",
		ArgsUsage: "<exchange> <pair> <asset> <rangesize> <granularity>",
		Action:    getHistoricCandles,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "exchange",
				Aliases: []string{"e"},
				Usage:   "the exchange to get the candles from",
			},
			&cli.StringFlag{
				Name:  "pair",
				Usage: "the currency pair to get the candles for",
			},
			&cli.StringFlag{
				Name:  "asset",
				Usage: "the asset type of the currency pair",
			},
			&cli.Int64Flag{
				Name:        "rangesize",
				Aliases:     []string{"r"},
				Usage:       "the amount of time to go back from now to fetch candles in the given granularity",
				Value:       10,
				Destination: &candleRangeSize,
			},
			&cli.Int64Flag{
				Name:        "granularity",
				Aliases:     []string{"g"},
				Usage:       klineMessage,
				Value:       86400,
				Destination: &candleGranularity,
			},
			&cli.BoolFlag{
				Name:  "fillmissingdatawithtrades, fill",
				Usage: "will create candles for missing intervals using stored trade data <true/false>",
			},
		},
	}
)

func getHistoricCandles(c *cli.Context) error {
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
	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("rangesize") {
		candleRangeSize = c.Int64("rangesize")
	} else if c.Args().Get(3) != "" {
		candleRangeSize, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if c.IsSet("granularity") {
		candleGranularity = c.Int64("granularity")
	} else if c.Args().Get(4) != "" {
		candleGranularity, err = strconv.ParseInt(c.Args().Get(4), 10, 64)
		if err != nil {
			return err
		}
	}

	var fillMissingData bool
	if c.IsSet("fillmissingdatawithtrades") {
		fillMissingData = c.Bool("fillmissingdatawithtrades")
	} else if c.IsSet("fill") {
		fillMissingData = c.Bool("fill")
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	candleInterval := time.Duration(candleGranularity) * time.Second

	e := time.Now().Truncate(candleInterval)
	s := e.Add(-candleInterval * time.Duration(candleRangeSize))

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetHistoricCandles(c.Context,
		&gctrpc.GetHistoricCandlesRequest{
			Exchange: exchangeName,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType:             assetType,
			Start:                 s.Format(common.SimpleTimeFormatWithTimezone),
			End:                   e.Format(common.SimpleTimeFormatWithTimezone),
			TimeInterval:          int64(candleInterval),
			FillMissingWithTrades: fillMissingData,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getHistoricCandlesExtendedCommand = &cli.Command{
	Name:      "gethistoriccandlesextended",
	Usage:     "gets historical candles for the specified pair, asset, interval & date range",
	ArgsUsage: "<exchange> <pair> <asset> <interval> <start> <end>",
	Action:    getHistoricCandlesExtended,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "exchange",
			Aliases: []string{"e"},
			Usage:   "the exchange to get the candles from",
		},
		&cli.StringFlag{
			Name:    "pair",
			Aliases: []string{"p"},
			Usage:   "the currency pair to get the candles for",
		},
		&cli.StringFlag{
			Name:    "asset",
			Aliases: []string{"a"},
			Usage:   "the asset type of the currency pair",
		},
		&cli.Int64Flag{
			Name:        "interval",
			Aliases:     []string{"i"},
			Usage:       klineMessage,
			Value:       86400,
			Destination: &candleGranularity,
		},
		&cli.StringFlag{
			Name:        "start",
			Usage:       "the date to begin retrieving candles. Any candles before this date will be filtered",
			Value:       time.Now().AddDate(0, -1, 0).Format(time.DateTime),
			Destination: &startTime,
		},
		&cli.StringFlag{
			Name:        "end",
			Usage:       "the date to end retrieving candles. Any candles after this date will be filtered",
			Value:       time.Now().Format(time.DateTime),
			Destination: &endTime,
		},
		&cli.BoolFlag{
			Name:  "sync",
			Usage: "<true/false>",
		},
		&cli.BoolFlag{
			Name:  "force",
			Usage: "will overwrite any conflicting candle data on save <true/false>",
		},
		&cli.BoolFlag{
			Name:  "db",
			Usage: "source data from database <true/false>",
		},
		&cli.BoolFlag{
			Name:    "fillmissingdatawithtrades",
			Aliases: []string{"fill"},
			Usage:   "will create candles for missing intervals using stored trade data <true/false>",
		},
	},
}

func getHistoricCandlesExtended(c *cli.Context) error {
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

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("interval") {
		candleGranularity = c.Int64("interval")
	} else if c.Args().Get(3) != "" {
		candleGranularity, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			startTime = c.Args().Get(4)
		}
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			endTime = c.Args().Get(5)
		}
	}

	var sync bool
	if c.IsSet("sync") {
		sync = c.Bool("sync")
	}

	var useDB bool
	if c.IsSet("db") {
		useDB = c.Bool("db")
	}

	var fillMissingData bool
	if c.IsSet("fillmissingdatawithtrades") {
		fillMissingData = c.Bool("fillmissingdatawithtrades")
	} else if c.IsSet("fill") {
		fillMissingData = c.Bool("fill")
	}

	var force bool
	if c.IsSet("force") {
		force = c.Bool("force")
	}

	if force && !sync {
		return errors.New("cannot forcefully overwrite without sync")
	}

	candleInterval := time.Duration(candleGranularity) * time.Second
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
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
			Exchange: exchangeName,
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
			UseDb:                 useDB,
			FillMissingWithTrades: fillMissingData,
			Force:                 force,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var findMissingSavedCandleIntervalsCommand = &cli.Command{
	Name:      "findmissingsavedcandleintervals",
	Usage:     "will highlight any interval that is missing candle data so you can fill that gap",
	ArgsUsage: "<exchange> <pair> <asset> <interval> <start> <end>",
	Action:    findMissingSavedCandleIntervals,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "exchange",
			Aliases: []string{"e"},
			Usage:   "the exchange to find the missing candles",
		},
		&cli.StringFlag{
			Name:    "pair",
			Aliases: []string{"p"},
			Usage:   "the currency pair",
		},
		&cli.StringFlag{
			Name:    "asset",
			Aliases: []string{"a"},
			Usage:   "the asset type of the currency pair",
		},
		&cli.Int64Flag{
			Name:        "interval",
			Aliases:     []string{"i"},
			Usage:       klineMessage,
			Value:       86400,
			Destination: &candleGranularity,
		},
		&cli.StringFlag{
			Name:        "start",
			Usage:       "<start> rounded down to the nearest hour",
			Value:       time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
			Destination: &startTime,
		},
		&cli.StringFlag{
			Name:        "end",
			Usage:       "<end> rounded down to the nearest hour",
			Value:       time.Now().Truncate(time.Hour).Format(time.DateTime),
			Destination: &endTime,
		},
	},
}

func findMissingSavedCandleIntervals(c *cli.Context) error {
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

	p, err := currency.NewPairDelimiter(currencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(2)
	}

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	if c.IsSet("interval") {
		candleGranularity = c.Int64("interval")
	} else if c.Args().Get(3) != "" {
		candleGranularity, err = strconv.ParseInt(c.Args().Get(3), 10, 64)
		if err != nil {
			return err
		}
	}

	if !c.IsSet("start") {
		if c.Args().Get(4) != "" {
			startTime = c.Args().Get(4)
		}
	}

	if !c.IsSet("end") {
		if c.Args().Get(5) != "" {
			endTime = c.Args().Get(5)
		}
	}

	candleInterval := time.Duration(candleGranularity) * time.Second
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
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
			ExchangeName: exchangeName,
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
	Name:      "getmarginrateshistory",
	Usage:     "returns margin lending/borrow rates for a period",
	ArgsUsage: "<exchange> <asset> <currency> <start> <end> <getpredictedrate> <getlendingpayments> <getborrowrates> <getborrowcosts> <includeallrates>",
	Action:    getMarginRatesHistory,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "exchange",
			Aliases: []string{"e"},
			Usage:   "the exchange to retrieve margin rates from",
		},
		&cli.StringFlag{
			Name:    "asset",
			Aliases: []string{"a"},
			Usage:   "the asset type of the currency pair",
		},
		&cli.StringFlag{
			Name:    "currency",
			Aliases: []string{"c"},
			Usage:   "must be an enabled currency",
		},
		&cli.StringFlag{
			Name:        "start",
			Aliases:     []string{"sd"},
			Usage:       "<start>",
			Value:       time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
			Destination: &startTime,
		},
		&cli.StringFlag{
			Name:        "end",
			Aliases:     []string{"ed"},
			Usage:       "<end>",
			Value:       time.Now().Format(time.DateTime),
			Destination: &endTime,
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
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	var curr string
	if c.IsSet("currency") {
		curr = c.String("currency")
	} else {
		curr = c.Args().Get(2)
	}

	if !c.IsSet("start") {
		if c.Args().Get(3) != "" {
			startTime = c.Args().Get(3)
		}
	}

	if !c.IsSet("end") {
		if c.Args().Get(4) != "" {
			endTime = c.Args().Get(4)
		}
	}

	var err error
	var getPredictedRate bool
	if c.IsSet("getpredictedrate") {
		getPredictedRate = c.Bool("getpredictedrate")
	} else if c.Args().Get(5) != "" {
		getPredictedRate, err = strconv.ParseBool(c.Args().Get(5))
		if err != nil {
			return err
		}
	}

	var getLendingPayments bool
	if c.IsSet("getlendingpayments") {
		getLendingPayments = c.Bool("getlendingpayments")
	} else if c.Args().Get(6) != "" {
		getLendingPayments, err = strconv.ParseBool(c.Args().Get(6))
		if err != nil {
			return err
		}
	}

	var getBorrowRates bool
	if c.IsSet("getborrowrates") {
		getBorrowRates = c.Bool("getborrowrates")
	} else if c.Args().Get(7) != "" {
		getBorrowRates, err = strconv.ParseBool(c.Args().Get(7))
		if err != nil {
			return err
		}
	}

	var getBorrowCosts bool
	if c.IsSet("getborrowcosts") {
		getBorrowCosts = c.Bool("getborrowcosts")
	} else if c.Args().Get(8) != "" {
		getBorrowCosts, err = strconv.ParseBool(c.Args().Get(8))
		if err != nil {
			return err
		}
	}

	var includeAllRates bool
	if c.IsSet("includeallrates") {
		includeAllRates = c.Bool("includeallrates")
	} else if c.Args().Get(9) != "" {
		includeAllRates, err = strconv.ParseBool(c.Args().Get(9))
		if err != nil {
			return err
		}
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, startTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, endTime, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for end: %v", err)
	}

	err = common.StartEndTimeCheck(s, e)
	if err != nil {
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
			Exchange:           exchangeName,
			Asset:              assetType,
			Currency:           curr,
			StartDate:          s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:            e.Format(common.SimpleTimeFormatWithTimezone),
			GetPredictedRate:   getPredictedRate,
			GetLendingPayments: getLendingPayments,
			GetBorrowRates:     getBorrowRates,
			GetBorrowCosts:     getBorrowCosts,
			IncludeAllRates:    includeAllRates,
		})
	if err != nil {
		return err
	}

	jsonOutput(result)
	return nil
}

var getCurrencyTradeURLCommand = &cli.Command{
	Name:      "getcurrencytradeurl",
	Usage:     "returns the trading url of the instrument",
	ArgsUsage: "<exchange> <asset> <pair>",
	Action:    getCurrencyTradeURL,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:    "exchange",
			Aliases: []string{"e"},
			Usage:   "the exchange to retrieve margin rates from",
		},
		&cli.StringFlag{
			Name:    "asset",
			Aliases: []string{"a"},
			Usage:   "the asset type of the currency pair",
		},
		&cli.StringFlag{
			Name:    "pair",
			Aliases: []string{"p"},
			Usage:   "the currency pair",
		},
	},
}

func getCurrencyTradeURL(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	var exchangeName string
	if c.IsSet("exchange") {
		exchangeName = c.String("exchange")
	} else {
		exchangeName = c.Args().First()
	}

	var assetType string
	if c.IsSet("asset") {
		assetType = c.String("asset")
	} else {
		assetType = c.Args().Get(1)
	}

	if !validAsset(assetType) {
		return errInvalidAsset
	}

	var cp string
	if c.IsSet("pair") {
		cp = c.String("pair")
	} else {
		cp = c.Args().Get(2)
	}

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
			Exchange: exchangeName,
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
