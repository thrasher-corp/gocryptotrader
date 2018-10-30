package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/thrasher-/gocryptotrader/common"
	"github.com/thrasher-/gocryptotrader/core"
	"github.com/thrasher-/gocryptotrader/gctrpc/auth"
	"github.com/urfave/cli"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	host     string
	username string
	password string
)

func jsonOutput(in interface{}) {
	j, err := json.MarshalIndent(in, "", " ")
	if err != nil {
		return
	}
	fmt.Print(string(j))
}

func setupClient() (*grpc.ClientConn, error) {
	targetPath := filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "tls", "cert.pem")
	creds, err := credentials.NewClientTLSFromFile(targetPath, "")
	if err != nil {
		return nil, err
	}

	opts := []grpc.DialOption{grpc.WithTransportCredentials(creds),
		grpc.WithPerRPCCredentials(auth.BasicAuth{
			Username: username,
			Password: password,
		}),
	}
	conn, err := grpc.Dial(host, opts...)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func main() {
	app := cli.NewApp()
	app.Name = "gctcli"
	app.Version = core.Version(true)
	app.Usage = "command line interface for managing the gocryptotrader daemon"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "rpchost",
			Value:       "localhost:9052",
			Usage:       "the gRPC host to connect to",
			Destination: &host,
		},
		cli.StringFlag{
			Name:        "rpcuser",
			Value:       "admin",
			Usage:       "the gRPC username",
			Destination: &username,
		},
		cli.StringFlag{
			Name:        "rpcpassword",
			Value:       "Password",
			Usage:       "the gRPC password",
			Destination: &password,
		},
	}
	app.Commands = []cli.Command{
		getInfoCommand,
		getExchangesCommand,
		enableExchangeCommand,
		disableExchangeCommand,
		getExchangeInfoCommand,
		getTickerCommand,
		getTickersCommand,
		getOrderbookCommand,
		getOrderbooksCommand,
		getAccountInfoCommand,
		getConfigCommand,
		getPortfolioCommand,
		getPortfolioSummaryCommand,
		addPortfolioAddressCommand,
		removePortfolioAddressCommand,
		getForexProvidersCommand,
		getForexRatesCommand,
		getOrdersCommand,
		getOrderCommand,
		submitOrderCommand,
		cancelOrderCommand,
		cancelAllOrdersCommand,
		getEventsCommand,
		addEventCommand,
		removeEventCommand,
		getCryptocurrencyDepositAddressesCommand,
		getCryptocurrencyDepositAddressCommand,
		withdrawCryptocurrencyFundsCommand,
		withdrawFiatFundsCommand,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}
}
