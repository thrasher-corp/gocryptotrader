package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/core"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc/auth"
	"github.com/thrasher-corp/gocryptotrader/signaler"
	"github.com/urfave/cli/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

const (
	defaultUsername = "rpcuser"
	defaultPassword = "helloImTheDefaultPassword"
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
	c.Context, cancel = context.WithTimeout(c.Context, timeout)
	conn, err := grpc.NewClient(host, opts...)
	return conn, cancel, err
}

func main() {
	version := core.Version(true)
	version = strings.Replace(version, "GoCryptoTrader", "GoCryptoTrader Backtester", 1)
	app := cli.NewApp()
	app.Name = "btcli"
	app.Version = version
	app.EnableBashCompletion = true
	app.Usage = "command line interface for managing the backtester daemon"
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:        "rpchost",
			Value:       "localhost:9054",
			Usage:       "the gRPC host to connect to",
			Destination: &host,
		},
		&cli.StringFlag{
			Name:        "rpcuser",
			Value:       defaultUsername,
			Usage:       "the gRPC username",
			Destination: &username,
		},
		&cli.StringFlag{
			Name:        "rpcpassword",
			Value:       defaultPassword,
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
			Value:       filepath.Join(common.GetDefaultDataDir(runtime.GOOS), "backtester", "tls", "cert.pem"),
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
		executeStrategyFromFileCommand,
		executeStrategyFromConfigCommand,
		listAllTasksCommand,
		startTaskCommand,
		startAllTasksCommand,
		stopTaskCommand,
		stopAllTasksCommand,
		clearTaskCommand,
		clearAllTasksCommand,
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
