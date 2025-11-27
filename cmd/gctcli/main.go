package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/signaler"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/metadata"
)

var (
	host          string
	username      string
	password      string
	pairDelimiter string
	certPath      string
	timeout       time.Duration
	exchangeCreds accounts.Credentials
	verbose       bool
	ignoreTimeout bool
)

const defaultTimeout = time.Second * 30

func jsonOutput(in any) {
	j, err := json.MarshalIndent(in, "", " ")
	if err != nil {
		return
	}
	fmt.Print(string(j))
}

func setupClient(c *cli.Context) (*grpc.ClientConn, context.CancelFunc, error) {
	creds, err := credentials.NewClientTLSFromFile(certPath, "")
	if err != nil {
		return nil, nil, err
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: username,
			Password: password,
		}),
	}

	var cancel context.CancelFunc
	if !ignoreTimeout {
		c.Context, cancel = context.WithTimeout(c.Context, timeout)
	}
	if !exchangeCreds.IsEmpty() {
		flag, values := exchangeCreds.GetMetaData()
		c.Context = metadata.AppendToOutgoingContext(c.Context, flag, values)
	}
	if verbose {
		c.Context = metadata.AppendToOutgoingContext(c.Context, "verbose", "true")
	}
	conn, err := grpc.NewClient(host, opts...)
	return conn, cancel, err
}

func main() {
	app := cli.NewApp()
	app.Name = "gctcli"
	app.Version = core.Version(true)
	app.EnableBashCompletion = true
	app.Usage = "command line interface for managing the gocryptotrader daemon"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "rpchost",
			Value:       "localhost:9052",
			Usage:       "the gRPC host to connect to",
			Destination: &host,
		},
		&cli.StringFlag{
			Name:        "rpcuser",
			Value:       "admin",
			Usage:       "the gRPC username",
			Destination: &username,
		},
		&cli.StringFlag{
			Name:        "rpcpassword",
			Value:       "Password",
			Usage:       "the gRPC password",
			Destination: &password,
		},
		&cli.StringFlag{
			Name:        "delimiter",
			Value:       "-",
			Usage:       "the default currency pair delimiter used to standardise currency pair input",
			Destination: &pairDelimiter,
		},
		&cli.StringFlag{
			Name:        "cert",
			Value:       filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "tls", "cert.pem"),
			Usage:       "the path to TLS cert of the gRPC server",
			Destination: &certPath,
		},
		&cli.DurationFlag{
			Name:        "timeout",
			Value:       defaultTimeout,
			Usage:       "the default context timeout value for requests",
			Destination: &timeout,
		},
		&cli.StringFlag{
			Name:        "apikey",
			Usage:       "override config API key for request",
			Destination: &exchangeCreds.Key,
		},
		&cli.StringFlag{
			Name:        "apisecret",
			Usage:       "override config API Secret for request",
			Destination: &exchangeCreds.Secret,
		},
		&cli.StringFlag{
			Name:        "apisubaccount",
			Usage:       "override config API sub account for request",
			Destination: &exchangeCreds.SubAccount,
		},
		&cli.StringFlag{
			Name:        "apiclientid",
			Usage:       "override config API client ID for request",
			Destination: &exchangeCreds.ClientID,
		},
		&cli.StringFlag{
			Name:        "apipemkey",
			Usage:       "override config API PEM key for request",
			Destination: &exchangeCreds.PEMKey,
		},
		&cli.StringFlag{
			Name:        "apionetimepassword",
			Usage:       "override config API One Time Password (OTP) for request",
			Destination: &exchangeCreds.OneTimePassword,
		},
		&cli.BoolFlag{
			Name:        "verbose",
			Usage:       "allows the request to generate a more verbose outputs server side",
			Destination: &verbose,
		},
		&cli.BoolFlag{
			Name:        "ignoretimeout",
			Aliases:     []string{"it"},
			Usage:       "ignores the context timeout for requests",
			Destination: &ignoreTimeout,
		},
	}
	app.Commands = []*cli.Command{
		getInfoCommand,
		getSubsystemsCommand,
		enableSubsystemCommand,
		disableSubsystemCommand,
		getRPCEndpointsCommand,
		getCommunicationRelayersCommand,
		getExchangesCommand,
		enableExchangeCommand,
		disableExchangeCommand,
		getExchangeOTPCommand,
		getExchangeOTPsCommand,
		getExchangeInfoCommand,
		getTickerCommand,
		getTickersCommand,
		getAccountBalancesCommand,
		getAccountBalancesStreamCommand,
		updateAccountBalancesCommand,
		getConfigCommand,
		getPortfolioCommand,
		getPortfolioSummaryCommand,
		addPortfolioAddressCommand,
		removePortfolioAddressCommand,
		getForexProvidersCommand,
		getForexRatesCommand,
		getOrdersCommand,
		getManagedOrdersCommand,
		getOrderCommand,
		submitOrderCommand,
		simulateOrderCommand,
		cancelOrderCommand,
		cancelBatchOrdersCommand,
		cancelAllOrdersCommand,
		modifyOrderCommand,
		getEventsCommand,
		addEventCommand,
		removeEventCommand,
		getCryptocurrencyDepositAddressesCommand,
		getCryptocurrencyDepositAddressCommand,
		getAvailableTransferChainsCommand,
		withdrawCryptocurrencyFundsCommand,
		withdrawFiatFundsCommand,
		withdrawalRequestCommand,
		getLoggerDetailsCommand,
		setLoggerDetailsCommand,
		exchangePairManagerCommand,
		getTickerStreamCommand,
		getExchangeTickerStreamCommand,
		getAuditEventCommand,
		getHistoricCandlesCommand,
		getHistoricCandlesExtendedCommand,
		findMissingSavedCandleIntervalsCommand,
		gctScriptCommand,
		websocketManagerCommand,
		tradeCommand,
		dataHistoryCommands,
		currencyStateManagementCommand,
		futuresCommands,
		shutdownCommand,
		technicalAnalysisCommand,
		getMarginRatesHistoryCommand,
		orderbookCommand,
		getCurrencyTradeURLCommand,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Capture cancel for interrupt
		<-signaler.WaitForInterrupt()
		cancel()
		fmt.Println("rpc process interrupted")
		os.Exit(1)
	}()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
