package engine

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/thrasher-corp/gocryptotrader/backtester/btrpc"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/config"
)

func TestExecuteStrategyFromFile(t *testing.T) {
	t.Parallel()
	s := &GRPCServer{}
	_, err := s.ExecuteStrategyFromFile(context.Background(), nil)
	if !errors.Is(err, common.ErrNilArguments) {
		t.Errorf("received '%v' expecting '%v'", err, common.ErrNilArguments)
	}

	_, err = s.ExecuteStrategyFromFile(context.Background(), &btrpc.ExecuteStrategyFromFileRequest{})
	if !errors.Is(err, config.ErrFileNotFound) {
		t.Errorf("received '%v' expecting '%v'", err, config.ErrFileNotFound)
	}

	_, err = s.ExecuteStrategyFromFile(context.Background(), &btrpc.ExecuteStrategyFromFileRequest{
		StrategyFilePath: filepath.Join("..", "config", "strategyexamples", "dca-api-candles.strat"),
	})
	if err != nil {
		t.Error(err)
	}
}

func TestExecuteStrategyFromConfig(t *testing.T) {
	t.Parallel()
}
