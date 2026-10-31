package cryptodotcom

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

const (
	// tradingBotMaximumPageSize is the largest page of bots or executions a trading bot query returns
	tradingBotMaximumPageSize = 500
	// tradingBotGridMinimumLevels is the fewest levels a grid bot's grid has
	tradingBotGridMinimumLevels = 2
)

var (
	errTradingBotIDRequired                     = errors.New("trading bot ID required")
	errInvalidTradingBotID                      = errors.New("invalid trading bot ID")
	errInvalidTradingBotType                    = errors.New("invalid trading bot type")
	errInvalidTradingBotState                   = errors.New("invalid trading bot state")
	errTradingBotTypesRequired                  = errors.New("trading bot types required")
	errTradingBotStatesRequired                 = errors.New("trading bot states required without a bot ID")
	errTradingBotSettingsRequired               = errors.New("trading bot settings required")
	errTradingBotSettingsMismatch               = errors.New("trading bot settings are for another bot type")
	errTradingBotAllocationsRequired            = errors.New("trading bot allocations required")
	errInvalidTradingBotAllocation              = errors.New("trading bot allocated percentage must be greater than zero")
	errTradingBotQuantityOrInvestmentRequired   = errors.New("trading bot quantity or total investment required")
	errInvalidTradingBotInvestmentFrequencyMode = errors.New("invalid trading bot investment frequency mode")
	errInvalidTradingBotInvestmentFrequency     = errors.New("trading bot investment frequency must be a positive number of milliseconds")
	errInvalidTradingBotGridMode                = errors.New("invalid trading bot grid mode")
	errInvalidTradingBotGridLevels              = errors.New("trading bot grid levels below the minimum")
	errInvalidTradingBotPriceRange              = errors.New("trading bot upper price must be above its lower price")
)

// CreateTradingBot creates a trading bot of a type with that type's settings, and returns its ID
func (e *Exchange) CreateTradingBot(ctx context.Context, req *CreateTradingBotRequest) (*CreateTradingBotResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	if err := validateTradingBotType(req.BotType); err != nil {
		return nil, err
	}
	if req.Settings == nil {
		return nil, errTradingBotSettingsRequired
	}
	if botType := req.Settings.BotType(); botType != req.BotType {
		return nil, fmt.Errorf("%w: %s settings for a %s bot", errTradingBotSettingsMismatch, botType, req.BotType)
	}
	if err := req.Settings.validateSettings(); err != nil {
		return nil, err
	}
	var resp *CreateTradingBotResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/bot/create-trading-bot", req, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}

// UpdateTradingBot updates a running or paused trading bot's notifications and display name, and a DCA or grid bot's
// settings
func (e *Exchange) UpdateTradingBot(ctx context.Context, req *UpdateTradingBotRequest) error {
	if req == nil {
		return fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	botID, err := tradingBotIDParam(req.BotID)
	if err != nil {
		return err
	}
	if err := validateTradingBotType(req.BotType); err != nil {
		return err
	}
	params := &updateTradingBotParams{
		BotID:             botID,
		BotType:           req.BotType,
		NotifyOnBotChange: req.NotifyOnBotChange,
		DisplayName:       req.DisplayName,
	}
	if req.Settings != nil {
		if botType := req.Settings.BotType(); botType != req.BotType {
			return fmt.Errorf("%w: %s settings for a %s bot", errTradingBotSettingsMismatch, botType, req.BotType)
		}
		if params.Settings, err = req.Settings.updateSettingsParams(); err != nil {
			return err
		}
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/bot/update-trading-bot", params, nil)
}

// TerminateTradingBot terminates a trading bot, cancelling its open orders
func (e *Exchange) TerminateTradingBot(ctx context.Context, req *TerminateTradingBotRequest) error {
	if req == nil {
		return fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	botID, err := tradingBotIDParam(req.BotID)
	if err != nil {
		return err
	}
	if err := validateTradingBotType(req.BotType); err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/bot/terminate-trading-bot", &terminateTradingBotParams{
		BotID:         botID,
		BotType:       req.BotType,
		ClosePosition: req.ClosePosition,
	}, nil)
}

// PauseTradingBot pauses a running trading bot, cancelling its open orders and placing none until it is resumed
func (e *Exchange) PauseTradingBot(ctx context.Context, req *TradingBotRequest) error {
	params, err := newTradingBotParams(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/bot/pause-trading-bot", params, nil)
}

// ResumeTradingBot resumes a paused trading bot, which places orders by its settings again
func (e *Exchange) ResumeTradingBot(ctx context.Context, req *TradingBotRequest) error {
	params, err := newTradingBotParams(req)
	if err != nil {
		return err
	}
	return e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/bot/resume-trading-bot", params, nil)
}

// GetTradingBots returns the trading bots of some types, by ID or by state, latest updated first
func (e *Exchange) GetTradingBots(ctx context.Context, req *TradingBotsRequest) ([]TradingBotResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	var botID uint64
	if req.BotID != "" {
		id, err := tradingBotIDParam(req.BotID)
		if err != nil {
			return nil, err
		}
		botID = id
	}
	if len(req.BotTypes) == 0 {
		return nil, errTradingBotTypesRequired
	}
	for _, botType := range req.BotTypes {
		if err := validateTradingBotType(botType); err != nil {
			return nil, err
		}
	}
	if req.BotID == "" && len(req.States) == 0 {
		return nil, errTradingBotStatesRequired
	}
	for _, state := range req.States {
		switch state {
		case TradingBotStateRunning, TradingBotStatePaused, TradingBotStateTerminated:
		default:
			return nil, fmt.Errorf("%w: %q", errInvalidTradingBotState, state)
		}
	}
	if req.PageSize > tradingBotMaximumPageSize {
		return nil, fmt.Errorf("%w: page size %d is above %d", errInvalidCount, req.PageSize, tradingBotMaximumPageSize)
	}
	var resp *tradingBotListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/bot/get-trading-bots", &tradingBotsParams{
		BotID:    botID,
		BotTypes: req.BotTypes,
		States:   req.States,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.BotList, nil
}

// GetTradingBotExecutions returns the orders a trading bot placed
func (e *Exchange) GetTradingBotExecutions(ctx context.Context, req *TradingBotExecutionsRequest) ([]TradingBotExecutionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	botID, err := tradingBotIDParam(req.BotID)
	if err != nil {
		return nil, err
	}
	if req.PageSize > tradingBotMaximumPageSize {
		return nil, fmt.Errorf("%w: page size %d is above %d", errInvalidCount, req.PageSize, tradingBotMaximumPageSize)
	}
	params := &tradingBotExecutionsParams{BotID: botID, OrderStatuses: req.OrderStatuses, Page: req.Page, PageSize: req.PageSize}
	if len(req.BasketInstrumentNames) > 0 {
		params.BasketBotFilter = &tradingBotBasketFilter{IncludeChildSymbols: req.BasketInstrumentNames}
	}
	var resp *tradingBotExecutionListResponse
	if err := e.SendAuthenticatedHTTPRequest(ctx, privateEPL, "private/bot/get-trading-bot-executions", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.ExecutionList, nil
}

// validateSettings checks a DCA bot's settings
func (s *TradingBotDCASettings) validateSettings() error {
	if s == nil {
		return fmt.Errorf("%w: %T", common.ErrNilPointer, s)
	}
	if len(s.Allocations) == 0 {
		return errTradingBotAllocationsRequired
	}
	if err := validateTradingBotAllocations(s.Allocations); err != nil {
		return err
	}
	if s.InvestmentCurrency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if !s.IsBasket {
		if err := validateTradingBotSide(s.Side); err != nil {
			return err
		}
		if err := validateTradingBotOrderType(s.OrderType); err != nil {
			return err
		}
	}
	if s.OrderType == OrderTypeLimit && s.Price <= 0 {
		return errInvalidPrice
	}
	if s.Quantity < 0 {
		return errInvalidQuantity
	}
	if s.TotalInvestment < 0 {
		return errInvalidAmount
	}
	if s.Quantity == 0 && s.TotalInvestment == 0 {
		return errTradingBotQuantityOrInvestmentRequired
	}
	switch s.InvestmentFrequencyMode {
	case TradingBotInvestmentFrequencyModeFixedDuration:
		if s.InvestmentFrequency < time.Millisecond {
			return fmt.Errorf("%w: %s", errInvalidTradingBotInvestmentFrequency, s.InvestmentFrequency)
		}
	case TradingBotInvestmentFrequencyModeMonthly:
		if s.InvestmentFrequency < 0 {
			return fmt.Errorf("%w: %s", errInvalidTradingBotInvestmentFrequency, s.InvestmentFrequency)
		}
	default:
		return fmt.Errorf("%w: %q", errInvalidTradingBotInvestmentFrequencyMode, s.InvestmentFrequencyMode)
	}
	_, err := unixMilliParam(s.FirstInvestmentTime)
	return err
}

// validateSettings checks a TWAP bot's settings; its start may be in the future, so only its order is checked
func (s *TradingBotTWAPSettings) validateSettings() error {
	if s == nil {
		return fmt.Errorf("%w: %T", common.ErrNilPointer, s)
	}
	if s.InstrumentName == "" {
		return errInstrumentNameRequired
	}
	if err := validateTradingBotSide(s.Side); err != nil {
		return err
	}
	if err := validateTradingBotOrderType(s.OrderType); err != nil {
		return err
	}
	if s.OrderType == OrderTypeLimit && s.Price <= 0 {
		return errInvalidPrice
	}
	if s.Quantity <= 0 {
		return errInvalidQuantity
	}
	if s.StartTime.IsZero() {
		return fmt.Errorf("start %w", common.ErrDateUnset)
	}
	if s.EndTime.IsZero() {
		return fmt.Errorf("end %w", common.ErrDateUnset)
	}
	if s.StartTime.After(s.EndTime) {
		return common.ErrStartAfterEnd
	}
	if s.StartTime.Equal(s.EndTime) {
		return common.ErrStartEqualsEnd
	}
	_, err := unixMilliParam(s.StartTime)
	return err
}

// validateSettings checks a grid bot's settings
func (s *TradingBotGridSettings) validateSettings() error {
	if s == nil {
		return fmt.Errorf("%w: %T", common.ErrNilPointer, s)
	}
	if s.InstrumentName == "" {
		return errInstrumentNameRequired
	}
	if s.TotalInvestment <= 0 {
		return errInvalidAmount
	}
	if s.LowerPrice <= 0 {
		return errInvalidPrice
	}
	if s.UpperPrice <= s.LowerPrice {
		return errInvalidTradingBotPriceRange
	}
	switch s.GridMode {
	case TradingBotGridModeArithmetic, TradingBotGridModeGeometric:
	default:
		return fmt.Errorf("%w: %q", errInvalidTradingBotGridMode, s.GridMode)
	}
	if s.GridLevels < tradingBotGridMinimumLevels {
		return fmt.Errorf("%w: %d levels, at least %d", errInvalidTradingBotGridLevels, s.GridLevels, tradingBotGridMinimumLevels)
	}
	return nil
}

// validateSettings checks a funding arbitrage bot's settings
func (s *TradingBotFundingArbitrageSettings) validateSettings() error {
	if s == nil {
		return fmt.Errorf("%w: %T", common.ErrNilPointer, s)
	}
	if s.TotalInvestment <= 0 {
		return errInvalidAmount
	}
	if s.InvestmentCurrency.IsEmpty() {
		return currency.ErrCurrencyCodeEmpty
	}
	if err := s.SpotLeg.validate(); err != nil {
		return fmt.Errorf("spot leg: %w", err)
	}
	if err := s.PerpetualLeg.validate(); err != nil {
		return fmt.Errorf("perpetual leg: %w", err)
	}
	return nil
}

// validate checks a funding arbitrage bot's leg
func (l *TradingBotLeg) validate() error {
	if l.InstrumentName == "" {
		return errInstrumentNameRequired
	}
	return validateTradingBotSide(l.Side)
}

// updateSettingsParams checks a DCA bot's settings update and returns it as Crypto.com takes it, the frequency in
// milliseconds
func (s *TradingBotDCAUpdateSettings) updateSettingsParams() (any, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, s)
	}
	if err := validateTradingBotAllocations(s.Allocations); err != nil {
		return nil, err
	}
	if s.OrderType != "" {
		if err := validateTradingBotOrderType(s.OrderType); err != nil {
			return nil, err
		}
	}
	if s.Price < 0 || s.StopPrice < 0 || s.TakeProfitPrice < 0 {
		return nil, errInvalidPrice
	}
	if s.Quantity < 0 {
		return nil, errInvalidQuantity
	}
	frequency, err := investmentFrequencyParam(s.InvestmentFrequency)
	if err != nil {
		return nil, err
	}
	return &tradingBotDCAUpdateSettingsWire{
		Allocations:                     s.Allocations,
		OrderType:                       s.OrderType,
		Price:                           s.Price,
		MaxOrders:                       s.MaxOrders,
		StopPrice:                       s.StopPrice,
		TakeProfitPrice:                 s.TakeProfitPrice,
		AutoStake:                       s.AutoStake,
		InvestmentDuration:              s.InvestmentDuration,
		InvestmentFrequencyMilliseconds: frequency,
		Quantity:                        s.Quantity,
	}, nil
}

// updateSettingsParams checks a grid bot's settings update and returns it, which Crypto.com takes as it is
func (s *TradingBotGridUpdateSettings) updateSettingsParams() (any, error) {
	if s == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, s)
	}
	if s.StopPrice < 0 || s.TakeProfitPrice < 0 || s.TrailingUpPriceCap < 0 {
		return nil, errInvalidPrice
	}
	return s, nil
}

// validateTradingBotAllocations checks a DCA bot's allocations each name an instrument and a positive percentage
func validateTradingBotAllocations(allocations []TradingBotAllocation) error {
	var total decimal.Decimal
	for i := range allocations {
		if allocations[i].InstrumentName == "" {
			return fmt.Errorf("allocation %d: %w", i, errInstrumentNameRequired)
		}
		if allocations[i].AllocatedPercentage <= 0 {
			return fmt.Errorf("allocation %d: %w", i, errInvalidTradingBotAllocation)
		}
		total = total.Add(allocations[i].AllocatedPercentage.Decimal())
	}
	if len(allocations) > 0 && !total.Equal(decimal.NewFromInt(100)) {
		return fmt.Errorf("%w: the percentages total %s, not 100", errInvalidTradingBotAllocation, total)
	}
	return nil
}

// validateTradingBotType checks a bot type is one Crypto.com offers
func validateTradingBotType(botType string) error {
	switch botType {
	case TradingBotTypeDCA, TradingBotTypeTWAP, TradingBotTypeGrid, TradingBotTypeFundingArbitrage:
		return nil
	default:
		return fmt.Errorf("%w: %q", errInvalidTradingBotType, botType)
	}
}

// validateTradingBotSide checks a bot's side is buy or sell
func validateTradingBotSide(side string) error {
	if side != SideBuy && side != SideSell {
		return fmt.Errorf("%w: %q", errInvalidSide, side)
	}
	return nil
}

// validateTradingBotOrderType checks a bot's order type is market or limit, the only types bots place
func validateTradingBotOrderType(orderType string) error {
	if orderType != OrderTypeMarket && orderType != OrderTypeLimit {
		return fmt.Errorf("%w: %q", errInvalidOrderType, orderType)
	}
	return nil
}

// newTradingBotParams checks a pause or resume request and returns its params
func newTradingBotParams(req *TradingBotRequest) (*tradingBotParams, error) {
	if req == nil {
		return nil, fmt.Errorf("%w: %T", common.ErrNilPointer, req)
	}
	botID, err := tradingBotIDParam(req.BotID)
	if err != nil {
		return nil, err
	}
	if err := validateTradingBotType(req.BotType); err != nil {
		return nil, err
	}
	return &tradingBotParams{BotID: botID, BotType: req.BotType}, nil
}

// tradingBotIDParam returns a bot's ID, which Crypto.com reports as a string, as the integer the bot methods take
func tradingBotIDParam(botID string) (uint64, error) {
	if botID == "" {
		return 0, errTradingBotIDRequired
	}
	id, err := strconv.ParseUint(botID, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w %q: %w", errInvalidTradingBotID, botID, err)
	}
	return id, nil
}

// investmentFrequencyParam returns a DCA bot's investment frequency as the whole milliseconds Crypto.com takes, zero,
// which is left unsent, for none, and an error for a negative frequency
func investmentFrequencyParam(frequency time.Duration) (uint64, error) {
	ms := frequency.Milliseconds()
	if ms < 0 {
		return 0, fmt.Errorf("%w: %s", errInvalidTradingBotInvestmentFrequency, frequency)
	}
	return uint64(ms), nil
}
