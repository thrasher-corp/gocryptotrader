package engine

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/shopspring/decimal"
	"github.com/thrasher-corp/gocryptotrader/backtester/common"
	"github.com/thrasher-corp/gocryptotrader/backtester/data"
	"github.com/thrasher-corp/gocryptotrader/backtester/data/kline"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/exchange"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/compliance"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventhandlers/portfolio/holdings"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/event"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/fill"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/order"
	"github.com/thrasher-corp/gocryptotrader/backtester/eventtypes/signal"
	"github.com/thrasher-corp/gocryptotrader/backtester/funding"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	gctkline "github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	gctorder "github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

// Overriding functions
// these are designed to override interface implementations
// so there is less requirement gathering per test as the functions are
// tested in their own package

type fakeFolio struct{}

func (f fakeFolio) GetLatestComplianceSnapshot(string, asset.Item, currency.Pair) (*compliance.Snapshot, error) {
	return &compliance.Snapshot{}, nil
}

func (f fakeFolio) GetPositions(common.Event) ([]futures.Position, error) {
	return nil, nil
}

func (f fakeFolio) SetHoldingsForEvent(funding.IFundReader, common.Event) error {
	return nil
}

func (f fakeFolio) SetHoldingsForTimestamp(*holdings.Holding) error {
	return nil
}

func (f fakeFolio) OnSignal(signal.Event, *exchange.Settings, funding.IFundReserver) (*order.Order, error) {
	return nil, nil
}

func (f fakeFolio) OnFill(fill.Event, funding.IFundReleaser) (fill.Event, error) {
	return nil, nil
}

func (f fakeFolio) GetLatestOrderSnapshotForEvent(common.Event) (compliance.Snapshot, error) {
	return compliance.Snapshot{}, nil
}

func (f fakeFolio) GetLatestOrderSnapshots() ([]compliance.Snapshot, error) {
	return nil, nil
}

func (f fakeFolio) ViewHoldingAtTimePeriod(common.Event) (*holdings.Holding, error) {
	return nil, nil
}

func (f fakeFolio) UpdateHoldings(data.Event, funding.IFundReleaser) error {
	return nil
}

func (f fakeFolio) GetComplianceManager(string, asset.Item, currency.Pair) (*compliance.Manager, error) {
	return nil, nil
}

func (f fakeFolio) TrackFuturesOrder(fill.Event, funding.IFundReleaser) (*portfolio.PNLSummary, error) {
	return &portfolio.PNLSummary{}, nil
}

func (f fakeFolio) UpdatePNL(common.Event, decimal.Decimal) error {
	return nil
}

func (f fakeFolio) GetLatestPNLForEvent(common.Event) (*portfolio.PNLSummary, error) {
	return &portfolio.PNLSummary{}, nil
}

func (f fakeFolio) GetLatestPNLs() []portfolio.PNLSummary {
	return nil
}

func (f fakeFolio) CheckLiquidationStatus(data.Event, funding.ICollateralReader, *portfolio.PNLSummary) error {
	return nil
}

func (f fakeFolio) CreateLiquidationOrdersForExchange(data.Event, funding.IFundingManager) ([]order.Event, error) {
	return nil, nil
}

func (f fakeFolio) GetLatestHoldingsForAllCurrencies() []holdings.Holding {
	return nil
}

func (f fakeFolio) Reset() error {
	return nil
}

type fakeReport struct{}

func (f fakeReport) GenerateReport() error {
	return nil
}

func (f fakeReport) SetKlineData(*gctkline.Item) error {
	return nil
}

func (f fakeReport) UseDarkMode(bool) {}

type fakeStats struct{}

func (f *fakeStats) SetStrategyName(string) {
}

func (f *fakeStats) SetEventForOffset(common.Event) error {
	return nil
}

func (f *fakeStats) AddHoldingsForTime(*holdings.Holding) error {
	return nil
}

func (f *fakeStats) AddComplianceSnapshotForTime(*compliance.Snapshot, common.Event) error {
	return nil
}

func (f *fakeStats) CalculateAllResults() error {
	return nil
}

func (f *fakeStats) Reset() error {
	return nil
}

func (f *fakeStats) Serialise() (string, error) {
	return "", nil
}

func (f *fakeStats) AddPNLForTime(*portfolio.PNLSummary) error {
	return nil
}

func (f *fakeStats) CreateLog(common.Event) (string, error) {
	return "", nil
}

type fakeDataHolder struct{}

func (f fakeDataHolder) Setup() {
}

func (f fakeDataHolder) SetDataForCurrency(string, asset.Item, currency.Pair, data.Handler) error {
	return nil
}

func (f fakeDataHolder) GetAllData() ([]data.Handler, error) {
	cp := currency.NewBTCUSD()
	return []data.Handler{
		&kline.DataFromKline{
			Base: &data.Base{},
			Item: &gctkline.Item{
				Exchange:       testExchange,
				Pair:           cp,
				UnderlyingPair: cp,
				Asset:          asset.Spot,
				Interval:       gctkline.OneMin,
				Candles: []gctkline.Candle{
					{
						Time:   time.Now(),
						Open:   1337,
						High:   1337,
						Low:    1337,
						Close:  1337,
						Volume: 1337,
					},
				},
				SourceJobID:     uuid.UUID{},
				ValidationJobID: uuid.UUID{},
			},
			RangeHolder: &gctkline.IntervalRangeHolder{},
		},
	}, nil
}

func (f fakeDataHolder) GetDataForCurrency(common.Event) (data.Handler, error) {
	return nil, nil
}

func (f fakeDataHolder) Reset() error {
	return nil
}

type fakeFunding struct {
	hasFutures bool
}

func (f fakeFunding) UpdateCollateralForEvent(common.Event, bool) error {
	return nil
}

func (f fakeFunding) UpdateAllCollateral(bool, bool) error {
	return nil
}

func (f fakeFunding) UpdateFundingFromLiveData(bool) error {
	return nil
}

func (f fakeFunding) SetFunding(string, asset.Item, *accounts.Balance, bool) error {
	return nil
}

func (f fakeFunding) Reset() error {
	return nil
}

func (f fakeFunding) IsUsingExchangeLevelFunding() bool {
	return true
}

func (f fakeFunding) GetFundingForEvent(common.Event) (funding.IFundingPair, error) {
	return &funding.SpotPair{}, nil
}

func (f fakeFunding) Transfer(decimal.Decimal, *funding.Item, *funding.Item, bool) error {
	return nil
}

func (f fakeFunding) GenerateReport() (*funding.Report, error) {
	return nil, nil
}

func (f fakeFunding) AddUSDTrackingData(*kline.DataFromKline) error {
	return nil
}

func (f fakeFunding) CreateSnapshot(time.Time) error {
	return nil
}

func (f fakeFunding) USDTrackingDisabled() bool {
	return false
}

func (f fakeFunding) Liquidate(common.Event) error {
	return nil
}

func (f fakeFunding) GetAllFunding() ([]funding.BasicItem, error) {
	return nil, nil
}

func (f fakeFunding) UpdateCollateral() error {
	return nil
}

func (f fakeFunding) HasFutures() bool {
	return f.hasFutures
}

func (f fakeFunding) HasExchangeBeenLiquidated(common.Event) bool {
	return false
}

func (f fakeFunding) RealisePNL(string, asset.Item, currency.Code, decimal.Decimal) error {
	return nil
}

type fakeStrat struct{}

func (f fakeStrat) Name() string {
	return "fake"
}

func (f fakeStrat) Description() string {
	return "fake"
}

func (f fakeStrat) OnSignal(data.Handler, funding.IFundingTransferer, portfolio.Handler) (signal.Event, error) {
	return nil, nil
}

func (f fakeStrat) OnSimultaneousSignals([]data.Handler, funding.IFundingTransferer, portfolio.Handler) ([]signal.Event, error) {
	return nil, nil
}

func (f fakeStrat) UsingSimultaneousProcessing() bool {
	return true
}

func (f fakeStrat) SupportsSimultaneousProcessing() bool {
	return true
}

func (f fakeStrat) SetSimultaneousProcessing(bool) {}

func (f fakeStrat) SetCustomSettings(map[string]any) error {
	return nil
}

func (f fakeStrat) SetDefaults() {}

func (f fakeStrat) CloseAllPositions([]holdings.Holding, []data.Event) ([]signal.Event, error) {
	return []signal.Event{
		&signal.Signal{
			Base: &event.Base{
				Offset:         1,
				Exchange:       testExchange,
				Time:           time.Now(),
				Interval:       gctkline.FifteenSecond,
				CurrencyPair:   currency.NewBTCUSD(),
				UnderlyingPair: currency.NewBTCUSD(),
				AssetType:      asset.Spot,
			},
			OpenPrice:  leet,
			HighPrice:  leet,
			LowPrice:   leet,
			ClosePrice: leet,
			Volume:     leet,
			BuyLimit:   leet,
			SellLimit:  leet,
			Amount:     leet,
			Direction:  gctorder.Buy,
		},
	}, nil
}
