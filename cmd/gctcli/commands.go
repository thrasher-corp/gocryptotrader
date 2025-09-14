package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"github.com/urfave/cli/v2"
)

var getInfoCommand = &cli.Command{
	Name:   "getinfo",
	Usage:  "gets GoCryptoTrader info",
	Action: getInfo,
}

// error declarations
var (
	ErrRequiredValueMissing = errors.New("required value missing")
)

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
	Flags:     FlagsFromStruct(&GetTickerParams{}),
}

func getTicker(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &GetTickerParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	arg.Asset = strings.ToLower(arg.Asset)
	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	p, err := currency.NewPairDelimiter(arg.Pair, pairDelimiter)
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
			Exchange: arg.Exchange,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: arg.Asset,
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

var getAccountInfoCommand = &cli.Command{
	Name:      "getaccountinfo",
	Usage:     "gets the exchange account balance info",
	ArgsUsage: "<exchange> <asset>",
	Action:    getAccountInfo,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the account info for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type to get the account info for",
		},
	},
}

func getAccountInfo(c *cli.Context) error {
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
	result, err := client.GetAccountInfo(c.Context,
		&gctrpc.GetAccountInfoRequest{
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

var getAccountInfoStreamCommand = &cli.Command{
	Name:      "getaccountinfostream",
	Usage:     "gets the account info stream for a specific exchange",
	ArgsUsage: "<exchange> <asset>",
	Action:    getAccountInfoStream,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the account info stream from",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type to get the account info stream for",
		},
	},
}

func getAccountInfoStream(c *cli.Context) error {
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
	result, err := client.GetAccountInfoStream(c.Context,
		&gctrpc.GetAccountInfoRequest{Exchange: exchangeName, AssetType: assetType})
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

var updateAccountInfoCommand = &cli.Command{
	Name:      "updateaccountinfo",
	Usage:     "updates the exchange account balance info",
	ArgsUsage: "<exchange> <asset>",
	Action:    updateAccountInfo,
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:  "exchange",
			Usage: "the exchange to get the account info for",
		},
		&cli.StringFlag{
			Name:  "asset",
			Usage: "the asset type to get the account info for",
		},
	},
}

func updateAccountInfo(c *cli.Context) error {
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
	result, err := client.UpdateAccountInfo(c.Context,
		&gctrpc.GetAccountInfoRequest{
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
	Flags:     FlagsFromStruct(&AddPortfolioAddressParams{}),
}

func addPortfolioAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &AddPortfolioAddressParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.AddPortfolioAddress(c.Context,
		&gctrpc.AddPortfolioAddressRequest{
			Address:            arg.Address,
			CoinType:           arg.CoinType,
			Description:        arg.Description,
			Balance:            arg.Balance,
			SupportedExchanges: arg.SupportedExchanges,
			ColdStorage:        arg.ColdStorage,
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
	Flags:     FlagsFromStruct(&RemovePortfolioAddressCommandParam{}),
}

func removePortfolioAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &RemovePortfolioAddressCommandParam{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.RemovePortfolioAddress(c.Context,
		&gctrpc.RemovePortfolioAddressRequest{
			Address:     arg.Address,
			CoinType:    arg.CoinType,
			Description: arg.Description,
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
	Flags: FlagsFromStruct(&GetOrdersCommandParams{
		Start: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
		End:   time.Now().Format(time.DateTime),
	}),
}

func getOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &GetOrdersCommandParams{
		Start: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
		End:   time.Now().Format(time.DateTime),
	}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}
	arg.Asset = strings.ToLower(arg.Asset)
	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(arg.Pair, pairDelimiter)
	if err != nil {
		return err
	}

	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
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
		Exchange:  arg.Exchange,
		AssetType: arg.Asset,
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
	Flags:     FlagsFromStruct(&GetManagedOrdersCommandParams{}),
}

func getManagedOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &GetManagedOrdersCommandParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	arg.Asset = strings.ToLower(arg.Asset)
	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(arg.Pair, pairDelimiter)
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
		Exchange:  arg.Exchange,
		AssetType: arg.Asset,
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
	Flags:     FlagsFromStruct(&GetOrderParams{}),
}

func getOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &GetOrderParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	arg.Asset = strings.ToLower(arg.Asset)
	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	if !validPair(arg.CurrencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(arg.CurrencyPair, pairDelimiter)
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
		Exchange: arg.Exchange,
		OrderId:  arg.OrderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Asset: arg.Asset,
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
	Flags:     FlagsFromStruct(&SubmitOrderParams{}),
}

func submitOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &SubmitOrderParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validPair(arg.CurrencyPair) {
		return errInvalidPair
	}

	if arg.OrderSide == "" {
		return fmt.Errorf("%w: order side must be set", order.ErrSideIsInvalid)
	}

	if arg.OrderType == "" {
		return fmt.Errorf("%w: order type must be set", order.ErrTypeIsInvalid)
	}

	if arg.Amount == 0 {
		return order.ErrAmountMustBeSet
	}

	arg.AssetType = strings.ToLower(arg.AssetType)
	if !validAsset(arg.AssetType) {
		return errInvalidAsset
	}

	arg.MarginType = strings.ToLower(arg.MarginType)
	if arg.MarginType != "" && !margin.IsValidString(arg.MarginType) {
		return margin.ErrInvalidMarginType
	}

	p, err := currency.NewPairDelimiter(arg.CurrencyPair, pairDelimiter)
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
		Exchange: arg.ExchangeName,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Amount:            arg.Amount,
		Price:             arg.Price,
		Leverage:          arg.Leverage,
		Side:              arg.OrderSide,
		OrderType:         arg.OrderType,
		AssetType:         arg.AssetType,
		MarginType:        arg.MarginType,
		ClientId:          arg.ClientID,
		ClientOrderId:     arg.ClientOrderID,
		QuoteAmount:       arg.QuoteAmount,
		TimeInForce:       arg.TimeInForce,
		TriggerPrice:      arg.TriggerPrice,
		TriggerPriceType:  arg.TriggerPriceType,
		TriggerLimitPrice: arg.TriggerLimitPrice,
		StopLoss: &gctrpc.RiskManagement{
			Price:      arg.SlPrice,
			LimitPrice: arg.SlLimitPrice,
			PriceType:  arg.SlPriceType,
		},
		TakeProfit: &gctrpc.RiskManagement{
			Price:      arg.TpPrice,
			LimitPrice: arg.TpLimitPrice,
			PriceType:  arg.TpPriceType,
		},
		Hidden:             arg.Hidden,
		Iceberg:            arg.Iceberg,
		ReduceOnly:         arg.ReduceOnly,
		AutoBorrow:         arg.AutoBorrow,
		RetrieveFees:       arg.RetrieveFees,
		RetrieveFeeDelayMs: arg.RetrieveFeeDelayMs,
		TrackingMode:       arg.TrackingMode,
		TrackingValue:      arg.TrackingValue,
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
	Flags:     FlagsFromStruct(&SimulateOrderCommandParams{}),
}

func simulateOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &SimulateOrderCommandParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	if arg.OrderSide == "" {
		return order.ErrSideIsInvalid
	}

	if arg.Amount == 0 {
		return order.ErrAmountMustBeSet
	}

	p, err := currency.NewPairDelimiter(arg.Pair, pairDelimiter)
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
		Exchange: arg.Exchange,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Side:   arg.OrderSide,
		Amount: arg.Amount,
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
	ArgsUsage: "<exchange> <order_id> <client_order_id> <account_id> <pair> <asset> <side> <type> <client_id> <margin_type> <time_in_force>",
	Action:    cancelOrder,
	Flags:     FlagsFromStruct(&CancelOrderParams{}),
}

func cancelOrder(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &CancelOrderParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if arg.OrderID == "" {
		return errors.New("an order ID must be set")
	}

	arg.AssetType = strings.ToLower(arg.AssetType)
	if !validAsset(arg.AssetType) {
		return errInvalidAsset
	}

	// pair is optional, but if it's set, do a validity check
	var p currency.Pair
	if arg.CurrencyPair != "" {
		if !validPair(arg.CurrencyPair) {
			return errInvalidPair
		}
		var err error
		p, err = currency.NewPairDelimiter(arg.CurrencyPair, pairDelimiter)
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
		Exchange:  arg.Exchange,
		AccountId: arg.AccountID,
		OrderId:   arg.OrderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType:     arg.AssetType,
		Side:          arg.OrderSide,
		Type:          arg.OrderType,
		ClientOrderId: arg.ClientOrderID,
		ClientId:      arg.ClientID,
		MarginType:    arg.MarginType,
		TimeInForce:   arg.TimeInForce,
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
	Flags:     FlagsFromStruct(&CancelOrderParams{}),
}

func cancelBatchOrders(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &CancelOrderParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if arg.OrderID == "" {
		return errors.New("an order ID must be set")
	}

	arg.AssetType = strings.ToLower(arg.AssetType)
	if !validAsset(arg.AssetType) {
		return errInvalidAsset
	}

	// pair is optional, but if it's set, do a validity check
	var p currency.Pair
	if arg.CurrencyPair != "" {
		if !validPair(arg.CurrencyPair) {
			return errInvalidPair
		}
		var err error
		p, err = currency.NewPairDelimiter(arg.CurrencyPair, pairDelimiter)
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
		Exchange:  arg.Exchange,
		AccountId: arg.AccountID,
		OrdersId:  arg.OrderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: arg.AssetType,
		Side:      arg.OrderSide,
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
	Flags:     FlagsFromStruct(&ModifyOrderParams{}),
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

	modifyOrderParams := &ModifyOrderParams{}
	modifyOrderParams.AssetType = strings.ToLower(modifyOrderParams.AssetType)
	if !validAsset(modifyOrderParams.AssetType) {
		return errInvalidAsset
	}

	if !validPair(modifyOrderParams.CurrencyPair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(modifyOrderParams.CurrencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if modifyOrderParams.Price == 0 && modifyOrderParams.Amount == 0 {
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
		Exchange: modifyOrderParams.ExchangeName,
		OrderId:  modifyOrderParams.OrderID,
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		Asset:             modifyOrderParams.AssetType,
		Price:             modifyOrderParams.Price,
		Amount:            modifyOrderParams.Amount,
		Type:              modifyOrderParams.OrderType,
		Side:              modifyOrderParams.OrderSide,
		TimeInForce:       modifyOrderParams.TimeInForce,
		ClientOrderId:     modifyOrderParams.ClientOrderID,
		TriggerPrice:      modifyOrderParams.TriggerPrice,
		TriggerLimitPrice: modifyOrderParams.TriggerLimitPrice,
		TriggerPriceType:  modifyOrderParams.TriggerPriceType,
		StopLoss: &gctrpc.RiskManagement{
			Price:      modifyOrderParams.SlPrice,
			LimitPrice: modifyOrderParams.SlLimitPrice,
			PriceType:  modifyOrderParams.SlPriceType,
		},
		TakeProfit: &gctrpc.RiskManagement{
			Price:      modifyOrderParams.TpPrice,
			LimitPrice: modifyOrderParams.TpLimitPrice,
			PriceType:  modifyOrderParams.TpPriceType,
		},
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
	Flags:     FlagsFromStruct(&AddEventParams{}),
}

func addEvent(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &AddEventParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validPair(arg.CurrencyPair) {
		return errInvalidPair
	}

	arg.AssetType = strings.ToLower(arg.AssetType)
	if !validAsset(arg.AssetType) {
		return errInvalidAsset
	}

	p, err := currency.NewPairDelimiter(arg.CurrencyPair, pairDelimiter)
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
		Exchange: arg.ExchangeName,
		Item:     arg.Item,
		ConditionParams: &gctrpc.ConditionParams{
			Condition:       arg.Condition,
			Price:           arg.Price,
			CheckBids:       arg.CheckBids,
			CheckAsks:       arg.CheckAsks,
			OrderbookAmount: arg.OrderbookAmount,
		},
		Pair: &gctrpc.CurrencyPair{
			Delimiter: p.Delimiter,
			Base:      p.Base.String(),
			Quote:     p.Quote.String(),
		},
		AssetType: arg.AssetType,
		Action:    arg.Action,
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
	Flags:     FlagsFromStruct(&GetCryptoCurrencyDepositAddressCommandParams{}),
}

func getCryptocurrencyDepositAddress(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &GetCryptoCurrencyDepositAddressCommandParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}
	if arg.Currency == "" {
		return fmt.Errorf("%w: cryptocurrency must be set", currency.ErrCurrencyCodeEmpty)
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.GetCryptocurrencyDepositAddress(c.Context,
		&gctrpc.GetCryptocurrencyDepositAddressRequest{
			Exchange:       arg.Exchange,
			Cryptocurrency: arg.Currency,
			Chain:          arg.Chain,
			Bypass:         arg.Bypass,
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

	arg := &GetAvailableTransferChainsParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if arg.Currency == "" {
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
			Exchange:       arg.Exchange,
			Cryptocurrency: arg.Currency,
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
	Flags:     FlagsFromStruct(&WithdrawCryptoCurrencyFundParams{}),
}

func withdrawCryptocurrencyFunds(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}
	arg := &WithdrawCryptoCurrencyFundParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.WithdrawCryptocurrencyFunds(c.Context,
		&gctrpc.WithdrawCryptoRequest{
			Exchange:    arg.Exchange,
			Currency:    arg.CurrencyPair,
			Address:     arg.Address,
			AddressTag:  arg.AddressTag,
			Amount:      arg.Amount,
			Fee:         arg.Fee,
			Description: arg.Description,
			Chain:       arg.Chain,
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
	Flags:     FlagsFromStruct(&WithdrawFiatFundParams{}),
}

func withdrawFiatFunds(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &WithdrawFiatFundParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)
	result, err := client.WithdrawFiatFunds(c.Context,
		&gctrpc.WithdrawFiatRequest{
			Exchange:      arg.Exchange,
			Currency:      arg.Currency,
			Amount:        arg.Amount,
			Description:   arg.Description,
			BankAccountId: arg.BankAccountID,
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
			Flags:     FlagsFromStruct(&WithdrawlRequestByExchangeID{}),
			Action:    withdrawlRequestByExchangeID,
		},
		{
			Name:      "byexchange",
			Usage:     "exchange limit",
			ArgsUsage: "<id>",
			Flags:     FlagsFromStruct(&WithdrawlRequestByExchange{}),
			Action:    withdrawlRequestByExchangeID,
		},
		{
			Name:      "bydate",
			Usage:     "exchange start end limit",
			ArgsUsage: "<exchange> <start> <end> <limit>",
			Flags: FlagsFromStruct(&WithdrawalRequestByDate{
				Start: time.Now().AddDate(0, -1, 0).Format(time.DateTime),
				End:   time.Now().Format(time.DateTime),
			}),
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

	var in *gctrpc.WithdrawalEventsByExchangeRequest
	if c.Command.Name == "byexchangeid" {
		arg := &WithdrawlRequestByExchangeID{}
		err := UnmarshalCLIFields(c, arg)
		if err != nil {
			return err
		}

		in = &gctrpc.WithdrawalEventsByExchangeRequest{
			Exchange: arg.Exchange,
			Id:       arg.ID,
			Limit:    1,
		}
	} else {
		arg := &WithdrawlRequestByExchange{}
		err := UnmarshalCLIFields(c, arg)
		if err != nil {
			return err
		}

		arg.Asset = strings.ToLower(arg.Asset)
		if !validAsset(arg.Asset) {
			return errInvalidAsset
		}

		in = &gctrpc.WithdrawalEventsByExchangeRequest{
			Exchange:  arg.Exchange,
			Limit:     int32(arg.Limit), //nolint:gosec // TODO: SQL boiler's QueryMode limit only accepts the int type
			Currency:  arg.Currency,
			AssetType: arg.Asset,
		}
	}

	conn, cancel, err := setupClient(c)
	if err != nil {
		return err
	}
	defer closeConn(conn, cancel)

	client := gctrpc.NewGoCryptoTraderServiceClient(conn)

	result, err := client.WithdrawalEventsByExchange(c.Context, in)
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

	arg := &WithdrawalRequestByDate{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	s, err := time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err := time.ParseInLocation(time.DateTime, arg.End, time.Local)
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
			Exchange: arg.Exchange,
			Start:    s.Format(common.SimpleTimeFormatWithTimezone),
			End:      e.Format(common.SimpleTimeFormatWithTimezone),
			Limit:    int32(arg.Limit), //nolint:gosec // TODO: SQL boiler's QueryMode limit only accepts the int type
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
	Flags:     FlagsFromStruct(&GetTickerStreamParams{}),
}

func getTickerStream(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &GetTickerStreamParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	arg.Asset = strings.ToLower(arg.Asset)
	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	p, err := currency.NewPairFromString(arg.Pair)
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
			Exchange: arg.Exchange,
			Pair: &gctrpc.CurrencyPair{
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
				Delimiter: p.Delimiter,
			},
			AssetType: arg.Asset,
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

		fmt.Printf("Ticker stream for %s %s:\n", arg.Exchange,
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
	Flags: FlagsFromStruct(&GetAuditEventParam{
		Start: time.Now().Add(-time.Hour).Format(time.DateTime),
		End:   time.Now().Format(time.DateTime),
		Order: "asc",
		Limit: 100,
	}),
}

func getAuditEvent(c *cli.Context) error {
	arg := &GetAuditEventParam{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	s, err := time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}

	e, err := time.ParseInLocation(time.DateTime, arg.End, time.Local)
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
			Limit:     int32(arg.Limit), //nolint:gosec // TODO: SQL boiler's QueryMode limit only accepts the int type
			OrderBy:   arg.Order,
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
		Flags:     FlagsFromStruct(&HistoricCandlesParams{Granularity: 86400, RangeSize: 10}),
	}
)

func getHistoricCandles(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &HistoricCandlesParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validPair(arg.CurrencyPair) {
		return errInvalidPair
	}
	p, err := currency.NewPairDelimiter(arg.CurrencyPair, pairDelimiter)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
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
			Exchange: arg.Exchange,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType:             arg.Asset,
			Start:                 s.Format(common.SimpleTimeFormatWithTimezone),
			End:                   e.Format(common.SimpleTimeFormatWithTimezone),
			TimeInterval:          int64(candleInterval),
			FillMissingWithTrades: arg.FillMissingDataWithTrades,
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
	Flags: FlagsFromStruct(&GetHistoricCandlesParams{
		Interval: 86400,
		Start:    time.Now().AddDate(0, -1, 0).Format(time.DateTime),
		End:      time.Now().Format(time.DateTime),
	}),
}

func getHistoricCandlesExtended(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &GetHistoricCandlesParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}
	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(arg.Pair, pairDelimiter)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}
	if arg.Force && !arg.Sync {
		return errors.New("cannot forcefully overwrite without sync")
	}

	candleInterval := time.Duration(arg.Interval) * time.Second
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
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
			Exchange: arg.Exchange,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType:             arg.Asset,
			Start:                 s.Format(common.SimpleTimeFormatWithTimezone),
			End:                   e.Format(common.SimpleTimeFormatWithTimezone),
			TimeInterval:          int64(candleInterval),
			ExRequest:             true,
			Sync:                  arg.Sync,
			UseDb:                 arg.Database,
			FillMissingWithTrades: arg.FillMissingDataWithTrades,
			Force:                 arg.Force,
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
	Flags: FlagsFromStruct(&FindMissingSavedCandleIntervalsParams{
		Interval: 86400,
		Start:    time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
		End:      time.Now().Truncate(time.Hour).Format(time.DateTime),
	}),
}

func findMissingSavedCandleIntervals(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &FindMissingSavedCandleIntervalsParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}

	p, err := currency.NewPairDelimiter(arg.Pair, pairDelimiter)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	candleInterval := time.Duration(arg.Interval) * time.Second
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
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
			ExchangeName: arg.Exchange,
			Pair: &gctrpc.CurrencyPair{
				Delimiter: p.Delimiter,
				Base:      p.Base.String(),
				Quote:     p.Quote.String(),
			},
			AssetType: arg.Asset,
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
	Flags: FlagsFromStruct(&MarginRateHistoryParam{
		Start: time.Now().AddDate(0, -1, 0).Truncate(time.Hour).Format(time.DateTime),
		End:   time.Now().Format(time.DateTime),
	}),
}

func getMarginRatesHistory(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &MarginRateHistoryParam{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}
	var s, e time.Time
	s, err = time.ParseInLocation(time.DateTime, arg.Start, time.Local)
	if err != nil {
		return fmt.Errorf("invalid time format for start: %v", err)
	}
	e, err = time.ParseInLocation(time.DateTime, arg.End, time.Local)
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
			Exchange:           arg.Exchange,
			Asset:              arg.Asset,
			Currency:           arg.Currency,
			StartDate:          s.Format(common.SimpleTimeFormatWithTimezone),
			EndDate:            e.Format(common.SimpleTimeFormatWithTimezone),
			GetPredictedRate:   arg.GetPredictedRate,
			GetLendingPayments: arg.GetLendingPayments,
			GetBorrowRates:     arg.GetBorrowRates,
			GetBorrowCosts:     arg.GetBorrowCosts,
			IncludeAllRates:    arg.IncludeAllRates,
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
	Flags:     FlagsFromStruct(&CurrencyTradeURLParams{}),
}

func getCurrencyTradeURL(c *cli.Context) error {
	if c.NArg() == 0 && c.NumFlags() == 0 {
		return cli.ShowSubcommandHelp(c)
	}

	arg := &CurrencyTradeURLParams{}
	err := UnmarshalCLIFields(c, arg)
	if err != nil {
		return err
	}

	if !validAsset(arg.Asset) {
		return errInvalidAsset
	}

	if !validPair(arg.Pair) {
		return errInvalidPair
	}
	p, err := currency.NewPairDelimiter(arg.Pair, pairDelimiter)
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
			Exchange: arg.Exchange,
			Asset:    arg.Asset,
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

func UnmarshalCLIFields(c *cli.Context, params any) error {
	val := reflect.ValueOf(params).Elem()
	typ := val.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		flagNames := strings.Split(field.Tag.Get("name"), ",")
		if len(flagNames) == 0 || flagNames[0] == "" {
			continue
		}
		required := slices.Contains([]string{"t", "true"}, strings.ToLower(field.Tag.Get("required")))

		switch field.Type.Kind() {
		case reflect.String:
			var value string
			for n := range flagNames {
				if c.IsSet(flagNames[n]) {
					value = c.String(flagNames[n])
				} else {
					value = c.Args().Get(i)
				}
				if value != "" {
					break
				}
			}
			if value == "" && val.Field(i).String() != "" {
				value = val.Field(i).String()
			} else if required && value == "" {
				return fmt.Errorf("%w for flag %q", ErrRequiredValueMissing, flagNames[0])
			}
			val.Field(i).SetString(value)
		case reflect.Float64:
			var value float64
			for n := range flagNames {
				if c.IsSet(flagNames[n]) {
					value = c.Float64(flagNames[n])
				} else if c.Args().Get(i) != "" {
					var err error
					value, err = strconv.ParseFloat(c.Args().Get(i), 64)
					if err != nil {
						return err
					}
				}
				if value != 0 {
					break
				}
			}
			if value == 0 && val.Field(i).Float() != 0 {
				value = val.Field(i).Float()
			} else if required && value == 0 {
				return fmt.Errorf("%w for flag %q", ErrRequiredValueMissing, flagNames[0])
			}
			val.Field(i).SetFloat(value)
		case reflect.Bool:
			var value bool
			for n := range flagNames {
				if c.IsSet(flagNames[n]) {
					value = c.Bool(flagNames[n])
				} else if c.Args().Get(i) != "" {
					var err error
					value, err = strconv.ParseBool(c.Args().Get(i))
					if required && (err != nil || !value) {
						return fmt.Errorf("%w for flag %q", ErrRequiredValueMissing, flagNames[0])
					}
				}
				if !value {
					break
				}
			}
			if !value && val.Field(i).Bool() {
				value = val.Field(i).Bool()
			}
			val.Field(i).SetBool(value)
		case reflect.Int64:
			var value int64
			for n := range flagNames {
				if c.IsSet(flagNames[n]) {
					value = c.Int64(flagNames[n])
				} else if c.Args().Get(i) != "" {
					var err error
					value, err = strconv.ParseInt(c.Args().Get(i), 10, 64)
					if err != nil {
						return err
					}
				}
				if value != 0 {
					break
				}
			}
			if value == 0 && val.Field(i).Int() != 0 {
				value = val.Field(i).Int()
			} else if required && value == 0 {
				return fmt.Errorf("%w for flag %q", ErrRequiredValueMissing, flagNames[0])
			}
			val.Field(i).SetInt(value)
		}
	}
	return nil
}

// FlagsFromStruct returns list of cli flags from exported flags
func FlagsFromStruct(params any) []cli.Flag {
	var flags []cli.Flag
	val := reflect.ValueOf(params).Elem()
	typ := val.Type()

	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		flagNames := strings.Split(field.Tag.Get("name"), ",")
		if len(flagNames) == 0 || flagNames[0] == "" {
			continue
		}
		flagName := flagNames[0]

		var aliceNames []string
		if len(flagNames) > 1 {
			if !slices.Contains(flagNames[1:], "") {
				aliceNames = flagNames[1:]
			}
		}

		required := slices.Contains([]string{"t", "true"}, strings.ToLower(field.Tag.Get("required")))
		hidden := slices.Contains([]string{"t", "true"}, strings.ToLower(field.Tag.Get("hidden")))

		usage := field.Tag.Get("usage")
		if usage == "" {
			if required {
				usage = "the required '" + flagName + "' for the request"
			} else {
				usage = "the optional '" + flagName + "' for the request"
			}
		}

		switch field.Type.Kind() {
		case reflect.String:
			flags = append(flags, &cli.StringFlag{
				Name:     flagName,
				Usage:    usage,
				Hidden:   hidden,
				Required: required,
				Value:    val.Field(i).String(),
				Aliases:  aliceNames,
			})
		case reflect.Float64:
			flags = append(flags, &cli.Float64Flag{
				Name:     flagName,
				Usage:    usage,
				Hidden:   hidden,
				Required: required,
				Value:    val.Field(i).Float(),
				Aliases:  aliceNames,
			})
		case reflect.Bool:
			flags = append(flags, &cli.BoolFlag{
				Name:     flagName,
				Usage:    usage,
				Hidden:   hidden,
				Required: required,
				Aliases:  aliceNames,
			})
		case reflect.Int64:
			flags = append(flags, &cli.Int64Flag{
				Name:     flagName,
				Usage:    usage,
				Hidden:   hidden,
				Required: required,
				Value:    val.Field(i).Int(),
				Aliases:  aliceNames,
			})
		}
	}
	return flags
}
