package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/signaler"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	host          string
	username      string
	password      string
	pairDelimiter string
	certPath      string
	timeout       time.Duration
)

const defaultTimeout = time.Second * 30

func jsonOutput(in interface{}) {
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

	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: username,
			Password: password,
		}),
	}

	var cancel context.CancelFunc
	c.Context, cancel = context.WithTimeout(c.Context, timeout)
	conn, err := grpc.DialContext(c.Context, host, opts...)
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
		getOrderbookCommand,
		getOrderbooksCommand,
		getAccountInfoCommand,
		getAccountInfoStreamCommand,
		updateAccountInfoCommand,
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
		whaleBombCommand,
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
		getOrderbookStreamCommand,
		getExchangeOrderbookStreamCommand,
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
		exchangeFeeManagementCommand,
		currencyStateManagementCommand,
		getFuturesPositionsCommand,
		getCollateralCommand,
	}

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		// Capture cancel for interrupt
		signaler.WaitForInterrupt()
		cancel()
		fmt.Println("rpc process interrupted")
		os.Exit(1)
	}()

	err := app.RunContext(ctx, os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
