package bitfinex

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"hash/crc32"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/buger/jsonparser"
	gws "github.com/gorilla/websocket"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/common/crypto"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/subscription"
	"github.com/thrasher-corp/gocryptotrader/exchanges/ticker"
	"github.com/thrasher-corp/gocryptotrader/exchanges/trade"
	"github.com/thrasher-corp/gocryptotrader/log"
	"github.com/thrasher-corp/gocryptotrader/types"
)

const (
	authenticatedBitfinexWebsocketEndpoint = "wss://api.bitfinex.com/ws/2"
	publicBitfinexWebsocketEndpoint        = "wss://api-pub.bitfinex.com/ws/2"
	pong                                   = "pong"
	wsHeartbeat                            = "hb"
	wsChecksum                             = "cs"
	wsPositionSnapshot                     = "ps"
	wsPositionNew                          = "pn"
	wsPositionUpdate                       = "pu"
	wsPositionClose                        = "pc"
	wsWalletSnapshot                       = "ws"
	wsWalletUpdate                         = "wu"
	wsTradeUpdated                         = "tu"
	wsTradeExecuted                        = "te"
	wsFundingCreditSnapshot                = "fcs"
	wsFundingCreditNew                     = "fcn"
	wsFundingCreditUpdate                  = "fcu"
	wsFundingCreditCancel                  = "fcc"
	wsFundingLoanSnapshot                  = "fls"
	wsFundingLoanNew                       = "fln"
	wsFundingLoanUpdate                    = "flu"
	wsFundingLoanCancel                    = "flc"
	wsFundingTradeExecuted                 = "fte"
	wsFundingTradeUpdated                  = "ftu"
	wsFundingInfoUpdate                    = "fiu"
	wsBalanceUpdate                        = "bu"
	wsMarginInfoUpdate                     = "miu"
	wsNotification                         = "n"
	wsOrderSnapshot                        = "os"
	wsOrderNew                             = "on"
	wsOrderUpdate                          = "ou"
	wsOrderCancel                          = "oc"
	wsRequest                              = "-req"
	wsOrderNewRequest                      = wsOrderNew + wsRequest
	wsOrderUpdateRequest                   = wsOrderUpdate + wsRequest
	wsOrderCancelRequest                   = wsOrderCancel + wsRequest
	wsFundingOfferSnapshot                 = "fos"
	wsFundingOfferNew                      = "fon"
	wsFundingOfferUpdate                   = "fou"
	wsFundingOfferCancel                   = "foc"
	wsFundingOfferNewRequest               = wsFundingOfferNew + wsRequest
	wsFundingOfferUpdateRequest            = wsFundingOfferUpdate + wsRequest
	wsFundingOfferCancelRequest            = wsFundingOfferCancel + wsRequest
	wsCancelMultipleOrders                 = "oc_multi"
	wsBookChannel                          = "book"
	wsCandlesChannel                       = "candles"
	wsTickerChannel                        = "ticker"
	wsTradesChannel                        = "trades"
	wsError                                = "error"
	wsEventSubscribed                      = "subscribed"
	wsEventUnsubscribed                    = "unsubscribed"
	wsEventAuth                            = "auth"
	wsEventError                           = "error"
	wsEventConf                            = "conf"
	wsEventInfo                            = "info"
)

var defaultSubscriptions = subscription.List{
	{Enabled: true, Channel: subscription.TickerChannel, Asset: asset.All},
	{Enabled: true, Channel: subscription.AllTradesChannel, Asset: asset.All},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Spot, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.Margin, Interval: kline.OneMin},
	{Enabled: true, Channel: subscription.CandlesChannel, Asset: asset.MarginFunding, Interval: kline.OneMin, Params: map[string]any{CandlesPeriodKey: "p30"}},
	{Enabled: true, Channel: subscription.OrderbookChannel, Asset: asset.All, Levels: 100, Params: map[string]any{"prec": "R0"}},
}

var comms = make(chan websocket.Response)

type checksum struct {
	Token    uint32
	Sequence int64
}

// checksumStore quick global for now
var (
	checksumStore = make(map[int]*checksum)
	cMtx          sync.Mutex
)

var subscriptionNames = map[string]string{
	subscription.TickerChannel:    wsTickerChannel,
	subscription.OrderbookChannel: wsBookChannel,
	subscription.CandlesChannel:   wsCandlesChannel,
	subscription.AllTradesChannel: wsTradesChannel,
}

// WsConnect starts a new websocket connection
func (b *Bitfinex) WsConnect() error {
	ctx := context.TODO()
	if !b.Websocket.IsEnabled() || !b.IsEnabled() {
		return websocket.ErrWebsocketNotEnabled
	}
	var dialer gws.Dialer
	err := b.Websocket.Conn.Dial(ctx, &dialer, http.Header{})
	if err != nil {
		return fmt.Errorf("%v unable to connect to Websocket. Error: %s",
			b.Name,
			err)
	}

	b.Websocket.Wg.Add(1)
	go b.wsReadData(b.Websocket.Conn)
	if b.Websocket.CanUseAuthenticatedEndpoints() {
		err = b.Websocket.AuthConn.Dial(ctx, &dialer, http.Header{})
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v unable to connect to authenticated Websocket. Error: %s",
				b.Name,
				err)
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
		b.Websocket.Wg.Add(1)
		go b.wsReadData(b.Websocket.AuthConn)
		err = b.WsSendAuth(ctx)
		if err != nil {
			log.Errorf(log.ExchangeSys,
				"%v - authentication failed: %v\n",
				b.Name,
				err)
			b.Websocket.SetCanUseAuthenticatedEndpoints(false)
		}
	}

	b.Websocket.Wg.Add(1)
	go b.WsDataHandler(ctx)
	return b.ConfigureWS(ctx)
}

// wsReadData receives and passes on websocket messages for processing
func (b *Bitfinex) wsReadData(ws websocket.Connection) {
	defer b.Websocket.Wg.Done()
	for {
		resp := ws.ReadMessage()
		if resp.Raw == nil {
			return
		}
		comms <- resp
	}
}

// WsDataHandler handles data from wsReadData
func (b *Bitfinex) WsDataHandler(ctx context.Context) {
	defer b.Websocket.Wg.Done()
	for {
		select {
		case <-b.Websocket.ShutdownC:
			select {
			case resp := <-comms:
				err := b.wsHandleData(ctx, resp.Raw)
				if err != nil {
					select {
					case b.Websocket.DataHandler <- err:
					default:
						log.Errorf(log.WebsocketMgr, "%s websocket handle data error: %v", b.Name, err)
					}
				}
			default:
			}
			return
		case resp := <-comms:
			if resp.Type != gws.TextMessage {
				continue
			}
			err := b.wsHandleData(ctx, resp.Raw)
			if err != nil {
				b.Websocket.DataHandler <- err
			}
		}
	}
}

func (b *Bitfinex) wsHandleData(_ context.Context, respRaw []byte) error {
	var result any
	if err := json.Unmarshal(respRaw, &result); err != nil {
		return err
	}
	switch d := result.(type) {
	case map[string]any:
		return b.handleWSEvent(respRaw)
	case []any:
		chanIDFloat, ok := d[0].(float64)
		if !ok {
			return common.GetTypeAssertError("float64", d[0], "chanID")
		}
		chanID := int(chanIDFloat)

		eventType, hasEventType := d[1].(string)

		if chanID != 0 {
			if s := b.Websocket.GetSubscription(chanID); s != nil {
				return b.handleWSChannelUpdate(s, respRaw, eventType, d)
			}
			if b.Verbose {
				log.Warnf(log.ExchangeSys, "%s %s; dropped WS message: %s", b.Name, subscription.ErrNotFound, respRaw)
			}
			// We didn't have a mapping for this chanID; This probably means we have unsubscribed OR
			// received our first message before processing the sub chanID
			// In either case it's okay. No point in erroring because there's nothing we can do about it, and it happens often
			return nil
		}

		if !hasEventType {
			return errors.New("WS message without eventType")
		}

		switch eventType {
		case wsHeartbeat, pong:
			return nil
		case wsNotification:
			return b.handleWSNotification(d, respRaw)
		case wsOrderSnapshot:
			if snapBundle, ok := d[2].([]any); ok && len(snapBundle) > 0 {
				if _, ok := snapBundle[0].([]any); ok {
					for i := range snapBundle {
						if positionData, ok := snapBundle[i].([]any); ok {
							b.wsHandleOrder(positionData)
						}
					}
				}
			}
		case wsOrderCancel, wsOrderNew, wsOrderUpdate:
			if oData, ok := d[2].([]any); ok && len(oData) > 0 {
				b.wsHandleOrder(oData)
			}
		case wsPositionSnapshot:
			return b.handleWSPositionSnapshot(d)
		case wsPositionNew, wsPositionUpdate, wsPositionClose:
			return b.handleWSPositionUpdate(d)
		case wsTradeExecuted, wsTradeUpdated:
			return b.handleWSMyTradeUpdate(d, eventType)
		case wsFundingOfferSnapshot:
			if snapBundle, ok := d[2].([]any); ok && len(snapBundle) > 0 {
				if _, ok := snapBundle[0].([]any); ok {
					snapshot := make([]*WsFundingOffer, len(snapBundle))
					for i := range snapBundle {
						data, ok := snapBundle[i].([]any)
						if !ok {
							return errors.New("unable to type assert wsFundingOrderSnapshot snapBundle data")
						}
						offer, err := wsHandleFundingOffer(data, false /* include rate real */)
						if err != nil {
							return err
						}
						snapshot[i] = offer
					}
					b.Websocket.DataHandler <- snapshot
				}
			}
		case wsFundingOfferNew, wsFundingOfferUpdate, wsFundingOfferCancel:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				offer, err := wsHandleFundingOffer(data, true /* include rate real */)
				if err != nil {
					return err
				}
				b.Websocket.DataHandler <- offer
			}
		case wsFundingCreditSnapshot:
			if snapBundle, ok := d[2].([]any); ok && len(snapBundle) > 0 {
				if _, ok := snapBundle[0].([]any); ok {
					snapshot := make([]*WsCredit, len(snapBundle))
					for i := range snapBundle {
						data, ok := snapBundle[i].([]any)
						if !ok {
							return errors.New("unable to type assert wsFundingCreditSnapshot snapBundle data")
						}
						fundingCredit, err := wsHandleFundingCreditLoanData(data, true /* include position pair */)
						if err != nil {
							return err
						}
						snapshot[i] = fundingCredit
					}
					b.Websocket.DataHandler <- snapshot
				}
			}
		case wsFundingCreditNew, wsFundingCreditUpdate, wsFundingCreditCancel:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				fundingCredit, err := wsHandleFundingCreditLoanData(data, true /* include position pair */)
				if err != nil {
					return err
				}
				b.Websocket.DataHandler <- fundingCredit
			}
		case wsFundingLoanSnapshot:
			if snapBundle, ok := d[2].([]any); ok && len(snapBundle) > 0 {
				if _, ok := snapBundle[0].([]any); ok {
					snapshot := make([]*WsCredit, len(snapBundle))
					for i := range snapBundle {
						data, ok := snapBundle[i].([]any)
						if !ok {
							return errors.New("unable to type assert wsFundingLoanSnapshot snapBundle data")
						}
						fundingLoanSnapshot, err := wsHandleFundingCreditLoanData(data, false /* include position pair */)
						if err != nil {
							return err
						}
						snapshot[i] = fundingLoanSnapshot
					}
					b.Websocket.DataHandler <- snapshot
				}
			}
		case wsFundingLoanNew, wsFundingLoanUpdate, wsFundingLoanCancel:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				fundingData, err := wsHandleFundingCreditLoanData(data, false /* include position pair */)
				if err != nil {
					return err
				}
				b.Websocket.DataHandler <- fundingData
			}
		case wsWalletSnapshot:
			if snapBundle, ok := d[2].([]any); ok && len(snapBundle) > 0 {
				if _, ok := snapBundle[0].([]any); ok {
					snapshot := make([]WsWallet, len(snapBundle))
					for i := range snapBundle {
						data, ok := snapBundle[i].([]any)
						if !ok {
							return errors.New("unable to type assert wsWalletSnapshot snapBundle data")
						}
						var wallet WsWallet
						if wallet.Type, ok = data[0].(string); !ok {
							return errors.New("unable to type assert wallet snapshot type")
						}
						if wallet.Currency, ok = data[1].(string); !ok {
							return errors.New("unable to type assert wallet snapshot currency")
						}
						if wallet.Balance, ok = data[2].(float64); !ok {
							return errors.New("unable to type assert wallet snapshot balance")
						}
						if wallet.UnsettledInterest, ok = data[3].(float64); !ok {
							return errors.New("unable to type assert wallet snapshot unsettled interest")
						}
						if data[4] != nil {
							if wallet.BalanceAvailable, ok = data[4].(float64); !ok {
								return errors.New("unable to type assert wallet snapshot balance available")
							}
						}
						snapshot[i] = wallet
					}
					b.Websocket.DataHandler <- snapshot
				}
			}
		case wsWalletUpdate:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				var wallet WsWallet
				if wallet.Type, ok = data[0].(string); !ok {
					return errors.New("unable to type assert wallet snapshot type")
				}
				if wallet.Currency, ok = data[1].(string); !ok {
					return errors.New("unable to type assert wallet snapshot currency")
				}
				if wallet.Balance, ok = data[2].(float64); !ok {
					return errors.New("unable to type assert wallet snapshot balance")
				}
				if wallet.UnsettledInterest, ok = data[3].(float64); !ok {
					return errors.New("unable to type assert wallet snapshot unsettled interest")
				}
				if data[4] != nil {
					if wallet.BalanceAvailable, ok = data[4].(float64); !ok {
						return errors.New("unable to type assert wallet snapshot balance available")
					}
				}
				b.Websocket.DataHandler <- wallet
			}
		case wsBalanceUpdate:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				var balance WsBalanceInfo
				if balance.TotalAssetsUnderManagement, ok = data[0].(float64); !ok {
					return errors.New("unable to type assert balance total assets under management")
				}
				if balance.NetAssetsUnderManagement, ok = data[1].(float64); !ok {
					return errors.New("unable to type assert balance net assets under management")
				}
				b.Websocket.DataHandler <- balance
			}
		case wsMarginInfoUpdate:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				if eventType, ok := data[0].(string); ok && eventType == "base" {
					baseData, ok := data[1].([]any)
					if !ok {
						return errors.New("unable to type assert wsMarginInfoUpdate baseData")
					}
					var marginInfoBase WsMarginInfoBase
					if marginInfoBase.UserProfitLoss, ok = baseData[0].(float64); !ok {
						return errors.New("unable to type assert margin info user profit loss")
					}
					if marginInfoBase.UserSwaps, ok = baseData[1].(float64); !ok {
						return errors.New("unable to type assert margin info user swaps")
					}
					if marginInfoBase.MarginBalance, ok = baseData[2].(float64); !ok {
						return errors.New("unable to type assert margin info balance")
					}
					if marginInfoBase.MarginNet, ok = baseData[3].(float64); !ok {
						return errors.New("unable to type assert margin info net")
					}
					if marginInfoBase.MarginRequired, ok = baseData[4].(float64); !ok {
						return errors.New("unable to type assert margin info required")
					}
					b.Websocket.DataHandler <- marginInfoBase
				}
			}
		case wsFundingInfoUpdate:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				if fundingType, ok := data[0].(string); ok && fundingType == "sym" {
					symbolData, ok := data[2].([]any)
					if !ok {
						return errors.New("unable to type assert wsFundingInfoUpdate symbolData")
					}
					var fundingInfo WsFundingInfo
					if fundingInfo.Symbol, ok = data[1].(string); !ok {
						return errors.New("unable to type assert symbol")
					}
					if fundingInfo.YieldLoan, ok = symbolData[0].(float64); !ok {
						return errors.New("unable to type assert funding info update yield loan")
					}
					if fundingInfo.YieldLend, ok = symbolData[1].(float64); !ok {
						return errors.New("unable to type assert funding info update yield lend")
					}
					if fundingInfo.DurationLoan, ok = symbolData[2].(float64); !ok {
						return errors.New("unable to type assert funding info update duration loan")
					}
					if fundingInfo.DurationLend, ok = symbolData[3].(float64); !ok {
						return errors.New("unable to type assert funding info update duration lend")
					}
					b.Websocket.DataHandler <- fundingInfo
				}
			}
		case wsFundingTradeExecuted, wsFundingTradeUpdated:
			if data, ok := d[2].([]any); ok && len(data) > 0 {
				var wsFundingTrade WsFundingTrade
				tradeID, ok := data[0].(float64)
				if !ok {
					return errors.New("unable to type assert funding trade ID")
				}
				wsFundingTrade.ID = int64(tradeID)
				if wsFundingTrade.Symbol, ok = data[1].(string); !ok {
					return errors.New("unable to type assert funding trade symbol")
				}
				created, ok := data[2].(float64)
				if !ok {
					return errors.New("unable to type assert funding trade created")
				}
				wsFundingTrade.MTSCreated = time.UnixMilli(int64(created))
				offerID, ok := data[3].(float64)
				if !ok {
					return errors.New("unable to type assert funding trade offer ID")
				}
				wsFundingTrade.OfferID = int64(offerID)
				if wsFundingTrade.Amount, ok = data[4].(float64); !ok {
					return errors.New("unable to type assert funding trade amount")
				}
				if wsFundingTrade.Rate, ok = data[5].(float64); !ok {
					return errors.New("unable to type assert funding trade rate")
				}
				period, ok := data[6].(float64)
				if !ok {
					return errors.New("unable to type assert funding trade period")
				}
				wsFundingTrade.Period = int64(period)
				wsFundingTrade.Maker = data[7] != nil
				b.Websocket.DataHandler <- wsFundingTrade
			}
		default:
			b.Websocket.DataHandler <- websocket.UnhandledMessageWarning{
				Message: b.Name + websocket.UnhandledMessage + string(respRaw),
			}
			return nil
		}
	}
	return nil
}

func (b *Bitfinex) handleWSEvent(respRaw []byte) error {
	event, err := jsonparser.GetUnsafeString(respRaw, "event")
	if err != nil {
		return fmt.Errorf("%w 'event': %w from message: %s", common.ErrParsingWSField, err, respRaw)
	}
	switch event {
	case wsEventSubscribed:
		return b.handleWSSubscribed(respRaw)
	case wsEventUnsubscribed:
		chanID, err := jsonparser.GetUnsafeString(respRaw, "chanId")
		if err != nil {
			return fmt.Errorf("%w 'chanId': %w from message: %s", common.ErrParsingWSField, err, respRaw)
		}
		err = b.Websocket.Match.RequireMatchWithData("unsubscribe:"+chanID, respRaw)
		if err != nil {
			return fmt.Errorf("%w: unsubscribe:%v", err, chanID)
		}
	case wsEventError:
		if subID, err := jsonparser.GetUnsafeString(respRaw, "subId"); err == nil {
			err = b.Websocket.Match.RequireMatchWithData("subscribe:"+subID, respRaw)
			if err != nil {
				return fmt.Errorf("%w: subscribe:%v", err, subID)
			}
		} else if chanID, err := jsonparser.GetUnsafeString(respRaw, "chanId"); err == nil {
			err = b.Websocket.Match.RequireMatchWithData("unsubscribe:"+chanID, respRaw)
			if err != nil {
				return fmt.Errorf("%w: unsubscribe:%v", err, chanID)
			}
		} else {
			return fmt.Errorf("unknown channel error; Message: %s", respRaw)
		}
	case wsEventAuth:
		status, err := jsonparser.GetUnsafeString(respRaw, "status")
		if err != nil {
			return fmt.Errorf("%w 'status': %w from message: %s", common.ErrParsingWSField, err, respRaw)
		}
		if status == "OK" {
			var glob map[string]any
			if err := json.Unmarshal(respRaw, &glob); err != nil {
				return fmt.Errorf("unable to Unmarshal auth resp; Error: %w Msg: %v", err, respRaw)
			}
			// TODO - Send a better value down the channel
			b.Websocket.DataHandler <- glob
		} else {
			errCode, err := jsonparser.GetInt(respRaw, "code")
			if err != nil {
				log.Errorf(log.ExchangeSys, "%s %s 'code': %s from message: %s", b.Name, common.ErrParsingWSField, err, respRaw)
			}
			return fmt.Errorf("WS auth subscription error; Status: %s Error Code: %d", status, errCode)
		}
	case wsEventInfo:
		// Nothing to do with info for now.
		// version or platform.status might be useful in the future.
	case wsEventConf:
		status, err := jsonparser.GetUnsafeString(respRaw, "status")
		if err != nil {
			return fmt.Errorf("%w 'status': %w from message: %s", common.ErrParsingWSField, err, respRaw)
		}
		if status != "OK" {
			return fmt.Errorf("WS configure channel error; Status: %s", status)
		}
	default:
		return fmt.Errorf("unknown WS event msg: %s", respRaw)
	}

	return nil
}

// handleWSSubscribed parses a subscription response and registers the chanID key immediately, before updating subscribeToChan via IncomingWithData chan
// wsHandleData happens sequentially, so by rekeying on chanID immediately we ensure the first message is not dropped
func (b *Bitfinex) handleWSSubscribed(respRaw []byte) error {
	subID, err := jsonparser.GetUnsafeString(respRaw, "subId")
	if err != nil {
		return fmt.Errorf("%w 'subId': %w from message: %s", common.ErrParsingWSField, err, respRaw)
	}

	c := b.Websocket.GetSubscription(subID)
	if c == nil {
		return fmt.Errorf("%w: %w subID: %s", websocket.ErrSubscriptionFailure, subscription.ErrNotFound, subID)
	}

	chanID, err := jsonparser.GetInt(respRaw, "chanId")
	if err != nil {
		return fmt.Errorf("%w: %w 'chanId': %w; Channel: %s Pair: %s", websocket.ErrSubscriptionFailure, common.ErrParsingWSField, err, c.Channel, c.Pairs)
	}

	// Note: chanID's int type avoids conflicts with the string type subID key because of the type difference
	c = c.Clone()
	c.Key = int(chanID)

	// subscribeToChan removes the old subID keyed Subscription
	err = b.Websocket.AddSuccessfulSubscriptions(b.Websocket.Conn, c)
	if err != nil {
		return fmt.Errorf("%w: %w subID: %s", websocket.ErrSubscriptionFailure, err, subID)
	}

	if b.Verbose {
		log.Debugf(log.ExchangeSys, "%s Subscribed to Channel: %s Pair: %s ChannelID: %d\n", b.Name, c.Channel, c.Pairs, chanID)
	}

	return b.Websocket.Match.RequireMatchWithData("subscribe:"+subID, respRaw)
}

func (b *Bitfinex) handleWSChannelUpdate(s *subscription.Subscription, respRaw []byte, eventType string, d []any) error {
	if s == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}

	switch eventType {
	case wsChecksum:
		return b.handleWSChecksum(s, d)
	case wsHeartbeat:
		return nil
	}

	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}

	switch s.Channel {
	case subscription.OrderbookChannel:
		return b.handleWSBookUpdate(s, d)
	case subscription.CandlesChannel:
		return b.handleWSAllCandleUpdates(s, respRaw)
	case subscription.TickerChannel:
		return b.handleWSTickerUpdate(s, d)
	case subscription.AllTradesChannel:
		return b.handleWSAllTrades(s, respRaw)
	}

	return fmt.Errorf("%s unhandled channel update: %s", b.Name, s.Channel)
}

func (b *Bitfinex) handleWSChecksum(c *subscription.Subscription, d []any) error {
	if c == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	var token uint32
	if f, ok := d[2].(float64); !ok {
		return common.GetTypeAssertError("float64", d[2], "checksum")
	} else { //nolint:revive // using lexical variable requires else statement
		token = uint32(f)
	}
	if len(d) < 4 {
		return errNoSeqNo
	}
	var seqNo int64
	if f, ok := d[3].(float64); !ok {
		return common.GetTypeAssertError("float64", d[3], "seqNo")
	} else { //nolint:revive // using lexical variable requires else statement
		seqNo = int64(f)
	}

	chanID, ok := c.Key.(int)
	if !ok {
		return common.GetTypeAssertError("int", c.Key, "ChanID") // Should be impossible
	}

	cMtx.Lock()
	checksumStore[chanID] = &checksum{
		Token:    token,
		Sequence: seqNo,
	}
	cMtx.Unlock()
	return nil
}

func (b *Bitfinex) handleWSBookUpdate(c *subscription.Subscription, d []any) error {
	if c == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	if len(c.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	var newOrderbook []WebsocketBook
	obSnapBundle, ok := d[1].([]any)
	if !ok {
		return errors.New("orderbook interface cast failed")
	}
	if len(obSnapBundle) == 0 {
		return errors.New("no data within orderbook snapshot")
	}
	if len(d) < 3 {
		return errNoSeqNo
	}
	sequenceNo, ok := d[2].(float64)
	if !ok {
		return errors.New("type assertion failure")
	}
	var fundingRate bool
	switch id := obSnapBundle[0].(type) {
	case []any:
		for i := range obSnapBundle {
			data, ok := obSnapBundle[i].([]any)
			if !ok {
				return errors.New("type assertion failed for orderbok item data")
			}
			id, okAssert := data[0].(float64)
			if !okAssert {
				return errors.New("type assertion failed for orderbook id data")
			}
			pricePeriod, okAssert := data[1].(float64)
			if !okAssert {
				return errors.New("type assertion failed for orderbook price data")
			}
			rateAmount, okAssert := data[2].(float64)
			if !okAssert {
				return errors.New("type assertion failed for orderbook rate data")
			}
			if len(data) == 4 {
				fundingRate = true
				amount, okFunding := data[3].(float64)
				if !okFunding {
					return errors.New("type assertion failed for orderbook funding data")
				}
				newOrderbook = append(newOrderbook, WebsocketBook{
					ID:     int64(id),
					Period: int64(pricePeriod),
					Price:  rateAmount,
					Amount: amount,
				})
			} else {
				newOrderbook = append(newOrderbook, WebsocketBook{
					ID:     int64(id),
					Price:  pricePeriod,
					Amount: rateAmount,
				})
			}
		}
		if err := b.WsInsertSnapshot(c.Pairs[0], c.Asset, newOrderbook, fundingRate); err != nil {
			return fmt.Errorf("inserting snapshot error: %s",
				err)
		}
	case float64:
		pricePeriod, okSnap := obSnapBundle[1].(float64)
		if !okSnap {
			return errors.New("type assertion failed for orderbook price snapshot data")
		}
		amountRate, okSnap := obSnapBundle[2].(float64)
		if !okSnap {
			return errors.New("type assertion failed for orderbook amount snapshot data")
		}
		if len(obSnapBundle) == 4 {
			fundingRate = true
			var amount float64
			amount, okSnap = obSnapBundle[3].(float64)
			if !okSnap {
				return errors.New("type assertion failed for orderbook amount snapshot data")
			}
			newOrderbook = append(newOrderbook, WebsocketBook{
				ID:     int64(id),
				Period: int64(pricePeriod),
				Price:  amountRate,
				Amount: amount,
			})
		} else {
			newOrderbook = append(newOrderbook, WebsocketBook{
				ID:     int64(id),
				Price:  pricePeriod,
				Amount: amountRate,
			})
		}

		if err := b.WsUpdateOrderbook(c, c.Pairs[0], c.Asset, newOrderbook, int64(sequenceNo), fundingRate); err != nil {
			return fmt.Errorf("updating orderbook error: %s",
				err)
		}
	}

	return nil
}

func (b *Bitfinex) handleWSAllCandleUpdates(c *subscription.Subscription, respRaw []byte) error {
	if c == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	if len(c.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	v, valueType, _, err := jsonparser.Get(respRaw, "[1]")
	if err != nil {
		return fmt.Errorf("%w `candlesUpdate[1]`: %w", common.ErrParsingWSField, err)
	}
	if valueType != jsonparser.Array {
		return fmt.Errorf("%w `candlesUpdate[1]`: %w %q", common.ErrParsingWSField, jsonparser.UnknownValueTypeError, valueType)
	}
	var wsCandles []Candle
	if bytes.HasPrefix(v, []byte("[[")) {
		if err := json.Unmarshal(v, &wsCandles); err != nil {
			return fmt.Errorf("error unmarshalling candle snapshot: %w", err)
		}
	} else {
		var wsCandle Candle
		if err := json.Unmarshal(v, &wsCandle); err != nil {
			return fmt.Errorf("error unmarshalling candle update: %w", err)
		}
		wsCandles = []Candle{wsCandle}
	}

	klines := make([]websocket.KlineData, len(wsCandles))
	for i := range wsCandles {
		klines[i] = websocket.KlineData{
			Exchange:   b.Name,
			AssetType:  c.Asset,
			Pair:       c.Pairs[0],
			Timestamp:  wsCandles[i].Timestamp.Time(),
			OpenPrice:  wsCandles[i].Open.Float64(),
			ClosePrice: wsCandles[i].Close.Float64(),
			HighPrice:  wsCandles[i].High.Float64(),
			LowPrice:   wsCandles[i].Low.Float64(),
			Volume:     wsCandles[i].Volume.Float64(),
		}
	}
	b.Websocket.DataHandler <- klines
	return nil
}

func (b *Bitfinex) handleWSTickerUpdate(c *subscription.Subscription, d []any) error {
	if c == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	if len(c.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	tickerData, ok := d[1].([]any)
	if !ok {
		return errors.New("type assertion for tickerData")
	}

	t := &ticker.Price{
		AssetType:    c.Asset,
		Pair:         c.Pairs[0],
		ExchangeName: b.Name,
	}

	if len(tickerData) == 10 {
		if t.Bid, ok = tickerData[0].(float64); !ok {
			return errors.New("unable to type assert ticker bid")
		}
		if t.Ask, ok = tickerData[2].(float64); !ok {
			return errors.New("unable to type assert ticker ask")
		}
		if t.Last, ok = tickerData[6].(float64); !ok {
			return errors.New("unable to type assert ticker last")
		}
		if t.Volume, ok = tickerData[7].(float64); !ok {
			return errors.New("unable to type assert ticker volume")
		}
		if t.High, ok = tickerData[8].(float64); !ok {
			return errors.New("unable to type assert  ticker high")
		}
		if t.Low, ok = tickerData[9].(float64); !ok {
			return errors.New("unable to type assert ticker low")
		}
	} else {
		if t.FlashReturnRate, ok = tickerData[0].(float64); !ok {
			return errors.New("unable to type assert ticker flash return rate")
		}
		if t.Bid, ok = tickerData[1].(float64); !ok {
			return errors.New("unable to type assert ticker bid")
		}
		if t.BidPeriod, ok = tickerData[2].(float64); !ok {
			return errors.New("unable to type assert ticker bid period")
		}
		if t.BidSize, ok = tickerData[3].(float64); !ok {
			return errors.New("unable to type assert ticker bid size")
		}
		if t.Ask, ok = tickerData[4].(float64); !ok {
			return errors.New("unable to type assert ticker ask")
		}
		if t.AskPeriod, ok = tickerData[5].(float64); !ok {
			return errors.New("unable to type assert ticker ask period")
		}
		if t.AskSize, ok = tickerData[6].(float64); !ok {
			return errors.New("unable to type assert ticker ask size")
		}
		if t.Last, ok = tickerData[9].(float64); !ok {
			return errors.New("unable to type assert ticker last")
		}
		if t.Volume, ok = tickerData[10].(float64); !ok {
			return errors.New("unable to type assert ticker volume")
		}
		if t.High, ok = tickerData[11].(float64); !ok {
			return errors.New("unable to type assert ticker high")
		}
		if t.Low, ok = tickerData[12].(float64); !ok {
			return errors.New("unable to type assert ticker low")
		}
		if t.FlashReturnRateAmount, ok = tickerData[15].(float64); !ok {
			return errors.New("unable to type assert ticker flash return rate")
		}
	}
	b.Websocket.DataHandler <- t
	return nil
}

func (b *Bitfinex) handleWSAllTrades(s *subscription.Subscription, respRaw []byte) error {
	feedEnabled := b.IsTradeFeedEnabled()
	if !feedEnabled && !b.IsSaveTradeDataEnabled() {
		return nil
	}
	if s == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	if len(s.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	v, valueType, _, err := jsonparser.Get(respRaw, "[1]")
	if err != nil {
		return fmt.Errorf("%w `tradesUpdate[1]`: %w", common.ErrParsingWSField, err)
	}
	var wsTrades []*Trade
	switch valueType {
	case jsonparser.String:
		t, err := b.handleWSPublicTradeUpdate(respRaw)
		if err != nil {
			return fmt.Errorf("%w `tradesUpdate[2]`: %w", common.ErrParsingWSField, err)
		}
		wsTrades = []*Trade{t}
	case jsonparser.Array:
		if wsTrades, err = b.handleWSPublicTradesSnapshot(v); err != nil {
			return fmt.Errorf("%w `tradesSnapshot`: %w", common.ErrParsingWSField, err)
		}
	default:
		return fmt.Errorf("%w `tradesUpdate[1]`: %w %q", common.ErrParsingWSField, jsonparser.UnknownValueTypeError, valueType)
	}
	trades := make([]trade.Data, len(wsTrades))
	for _, w := range wsTrades {
		t := trade.Data{
			Exchange:     b.Name,
			AssetType:    s.Asset,
			CurrencyPair: s.Pairs[0],
			TID:          strconv.FormatInt(w.TID, 10),
			Timestamp:    w.Timestamp.Time().UTC(),
			Side:         w.Side,
			Amount:       w.Amount,
			Price:        w.Price,
		}
		if w.Period != 0 {
			t.AssetType = asset.MarginFunding
			t.Price = w.Rate
		}
		if feedEnabled {
			b.Websocket.DataHandler <- t
		}
	}
	if b.IsSaveTradeDataEnabled() {
		err = trade.AddTradesToBuffer(trades...)
	}
	return err
}

func (b *Bitfinex) handleWSPublicTradesSnapshot(v []byte) ([]*Trade, error) {
	var trades []*Trade
	return trades, json.Unmarshal(v, &trades)
}

func (b *Bitfinex) handleWSPublicTradeUpdate(respRaw []byte) (*Trade, error) {
	v, _, _, err := jsonparser.Get(respRaw, "[2]")
	if err != nil {
		return nil, err
	}
	t := &Trade{}
	return t, json.Unmarshal(v, t)
}

func (b *Bitfinex) handleWSNotification(d []any, respRaw []byte) error {
	notification, ok := d[2].([]any)
	if !ok {
		return errors.New("unable to type assert notification data")
	}
	if data, ok := notification[4].([]any); ok {
		channelName, ok := notification[1].(string)
		if !ok {
			return errors.New("unable to type assert channelName")
		}
		switch {
		case strings.Contains(channelName, wsFundingOfferNewRequest),
			strings.Contains(channelName, wsFundingOfferUpdateRequest),
			strings.Contains(channelName, wsFundingOfferCancelRequest):
			if data[0] != nil {
				if id, ok := data[0].(float64); ok && id > 0 {
					if b.Websocket.Match.IncomingWithData(int64(id), respRaw) {
						return nil
					}
					offer, err := wsHandleFundingOffer(data, true /* include rate real */)
					if err != nil {
						return err
					}
					b.Websocket.DataHandler <- offer
				}
			}
		case strings.Contains(channelName, wsOrderNewRequest):
			if data[2] != nil {
				if cid, ok := data[2].(float64); !ok {
					return common.GetTypeAssertError("float64", data[2], channelName+" cid")
				} else if cid > 0 {
					if b.Websocket.Match.IncomingWithData(int64(cid), respRaw) {
						return nil
					}
					b.wsHandleOrder(data)
				}
			}
		case strings.Contains(channelName, wsOrderUpdateRequest),
			strings.Contains(channelName, wsOrderCancelRequest):
			if data[0] != nil {
				if id, ok := data[0].(float64); !ok {
					return common.GetTypeAssertError("float64", data[0], channelName+" id")
				} else if id > 0 {
					if b.Websocket.Match.IncomingWithData(int64(id), respRaw) {
						return nil
					}
					b.wsHandleOrder(data)
				}
			}
		default:
			return fmt.Errorf("%s - Unexpected data returned %s",
				b.Name,
				respRaw)
		}
	}
	if notification[5] != nil {
		if wsErr, ok := notification[5].(string); ok {
			if strings.EqualFold(wsErr, wsError) {
				if errMsg, ok := notification[6].(string); ok {
					return fmt.Errorf("%s - Error %s",
						b.Name,
						errMsg)
				}
				return fmt.Errorf("%s - unhandled error message: %v", b.Name,
					notification[6])
			}
		}
	}
	return nil
}

func (b *Bitfinex) handleWSPositionSnapshot(d []any) error {
	snapBundle, ok := d[2].([]any)
	if !ok {
		return common.GetTypeAssertError("[]any", d[2], "positionSnapshotBundle")
	}
	if len(snapBundle) == 0 {
		return nil
	}
	snapshot := make([]WebsocketPosition, len(snapBundle))
	for i := range snapBundle {
		positionData, ok := snapBundle[i].([]any)
		if !ok {
			return common.GetTypeAssertError("[]any", snapBundle[i], "positionSnapshot")
		}
		var position WebsocketPosition
		if position.Pair, ok = positionData[0].(string); !ok {
			return errors.New("unable to type assert position snapshot pair")
		}
		if position.Status, ok = positionData[1].(string); !ok {
			return errors.New("unable to type assert position snapshot status")
		}
		if position.Amount, ok = positionData[2].(float64); !ok {
			return errors.New("unable to type assert position snapshot amount")
		}
		if position.Price, ok = positionData[3].(float64); !ok {
			return errors.New("unable to type assert position snapshot price")
		}
		if position.MarginFunding, ok = positionData[4].(float64); !ok {
			return errors.New("unable to type assert position snapshot margin funding")
		}
		marginFundingType, ok := positionData[5].(float64)
		if !ok {
			return errors.New("unable to type assert position snapshot margin funding type")
		}
		position.MarginFundingType = int64(marginFundingType)
		if position.ProfitLoss, ok = positionData[6].(float64); !ok {
			return errors.New("unable to type assert position snapshot profit loss")
		}
		if position.ProfitLossPercent, ok = positionData[7].(float64); !ok {
			return errors.New("unable to type assert position snapshot profit loss percent")
		}
		if position.LiquidationPrice, ok = positionData[8].(float64); !ok {
			return errors.New("unable to type assert position snapshot liquidation price")
		}
		if position.Leverage, ok = positionData[9].(float64); !ok {
			return errors.New("unable to type assert position snapshot leverage")
		}
		snapshot[i] = position
	}
	b.Websocket.DataHandler <- snapshot
	return nil
}

func (b *Bitfinex) handleWSPositionUpdate(d []any) error {
	positionData, ok := d[2].([]any)
	if !ok {
		return common.GetTypeAssertError("[]any", d[2], "positionUpdate")
	}
	if len(positionData) == 0 {
		return nil
	}
	var position WebsocketPosition
	if position.Pair, ok = positionData[0].(string); !ok {
		return errors.New("unable to type assert position pair")
	}
	if position.Status, ok = positionData[1].(string); !ok {
		return errors.New("unable to type assert position status")
	}
	if position.Amount, ok = positionData[2].(float64); !ok {
		return errors.New("unable to type assert position amount")
	}
	if position.Price, ok = positionData[3].(float64); !ok {
		return errors.New("unable to type assert position price")
	}
	if position.MarginFunding, ok = positionData[4].(float64); !ok {
		return errors.New("unable to type assert margin position funding")
	}
	marginFundingType, ok := positionData[5].(float64)
	if !ok {
		return errors.New("unable to type assert position margin funding type")
	}
	position.MarginFundingType = int64(marginFundingType)
	if position.ProfitLoss, ok = positionData[6].(float64); !ok {
		return errors.New("unable to type assert position profit loss")
	}
	if position.ProfitLossPercent, ok = positionData[7].(float64); !ok {
		return errors.New("unable to type assert position profit loss percent")
	}
	if position.LiquidationPrice, ok = positionData[8].(float64); !ok {
		return errors.New("unable to type assert position liquidation price")
	}
	if position.Leverage, ok = positionData[9].(float64); !ok {
		return errors.New("unable to type assert position leverage")
	}
	b.Websocket.DataHandler <- position
	return nil
}

func (b *Bitfinex) handleWSMyTradeUpdate(d []any, eventType string) error {
	tradeData, ok := d[2].([]any)
	if !ok {
		return common.GetTypeAssertError("[]any", d[2], "tradeUpdate")
	}
	if len(tradeData) <= 4 {
		return nil
	}
	var tData WebsocketTradeData
	var tradeID float64
	if tradeID, ok = tradeData[0].(float64); !ok {
		return errors.New("unable to type assert trade ID")
	}
	tData.TradeID = int64(tradeID)
	if tData.Pair, ok = tradeData[1].(string); !ok {
		return errors.New("unable to type assert trade pair")
	}
	var timestamp float64
	if timestamp, ok = tradeData[2].(float64); !ok {
		return errors.New("unable to type assert trade timestamp")
	}
	tData.Timestamp = types.Time(time.UnixMilli(int64(timestamp)))
	var orderID float64
	if orderID, ok = tradeData[3].(float64); !ok {
		return errors.New("unable to type assert trade order ID")
	}
	tData.OrderID = int64(orderID)
	if tData.AmountExecuted, ok = tradeData[4].(float64); !ok {
		return errors.New("unable to type assert trade amount executed")
	}
	if tData.PriceExecuted, ok = tradeData[5].(float64); !ok {
		return errors.New("unable to type assert trade price executed")
	}
	if tData.OrderType, ok = tradeData[6].(string); !ok {
		return errors.New("unable to type assert trade order type")
	}
	if tData.OrderPrice, ok = tradeData[7].(float64); !ok {
		return errors.New("unable to type assert trade order type")
	}
	var maker float64
	if maker, ok = tradeData[8].(float64); !ok {
		return errors.New("unable to type assert trade maker")
	}
	tData.Maker = maker == 1
	if eventType == "tu" {
		if tData.Fee, ok = tradeData[9].(float64); !ok {
			return errors.New("unable to type assert trade fee")
		}
		if tData.FeeCurrency, ok = tradeData[10].(string); !ok {
			return errors.New("unable to type assert trade fee currency")
		}
	}
	b.Websocket.DataHandler <- tData
	return nil
}

func wsHandleFundingOffer(data []any, includeRateReal bool) (*WsFundingOffer, error) {
	var offer WsFundingOffer
	var ok bool
	if data[0] != nil {
		var offerID float64
		if offerID, ok = data[0].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer ID")
		}
		offer.ID = int64(offerID)
	}
	if data[1] != nil {
		if offer.Symbol, ok = data[1].(string); !ok {
			return nil, errors.New("unable to type assert funding offer symbol")
		}
	}
	if data[2] != nil {
		var created float64
		if created, ok = data[2].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer created")
		}
		offer.Created = time.UnixMilli(int64(created))
	}
	if data[3] != nil {
		var updated float64
		if updated, ok = data[3].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer updated")
		}
		offer.Updated = time.UnixMilli(int64(updated))
	}
	if data[4] != nil {
		if offer.Amount, ok = data[4].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer amount")
		}
	}
	if data[5] != nil {
		if offer.OriginalAmount, ok = data[5].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer original amount")
		}
	}
	if data[6] != nil {
		if offer.Type, ok = data[6].(string); !ok {
			return nil, errors.New("unable to type assert funding offer type")
		}
	}
	if data[9] != nil {
		if offer.Flags, ok = data[9].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer flags")
		}
	}
	if data[10] != nil {
		if offer.Status, ok = data[10].(string); !ok {
			return nil, errors.New("unable to type assert funding offer status")
		}
	}
	if data[14] != nil {
		if offer.Rate, ok = data[14].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer rate")
		}
	}
	if data[15] != nil {
		var period float64
		if period, ok = data[15].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer period")
		}
		offer.Period = int64(period)
	}
	if data[16] != nil {
		var notify float64
		if notify, ok = data[16].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer notify")
		}
		offer.Notify = notify == 1
	}
	if data[17] != nil {
		var hidden float64
		if hidden, ok = data[17].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer hidden")
		}
		offer.Hidden = hidden == 1
	}
	if data[19] != nil {
		var renew float64
		if renew, ok = data[19].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer renew")
		}
		offer.Renew = renew == 1
	}
	if includeRateReal && data[20] != nil {
		if offer.RateReal, ok = data[20].(float64); !ok {
			return nil, errors.New("unable to type assert funding offer rate real")
		}
	}
	return &offer, nil
}

func wsHandleFundingCreditLoanData(data []any, includePositionPair bool) (*WsCredit, error) {
	var credit WsCredit
	var ok bool
	if data[0] != nil {
		var id float64
		if id, ok = data[0].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit ID")
		}
		credit.ID = int64(id)
	}
	if data[1] != nil {
		if credit.Symbol, ok = data[1].(string); !ok {
			return nil, errors.New("unable to type assert funding credit symbol")
		}
	}
	if data[2] != nil {
		var side float64
		if side, ok = data[2].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit side")
		}
		credit.Side = int8(side)
	}
	if data[3] != nil {
		var created float64
		if created, ok = data[3].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit created")
		}
		credit.Created = time.UnixMilli(int64(created))
	}
	if data[4] != nil {
		var updated float64
		if updated, ok = data[4].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit updated")
		}
		credit.Updated = time.UnixMilli(int64(updated))
	}
	if data[5] != nil {
		if credit.Amount, ok = data[5].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit amount")
		}
	}
	if data[6] != nil {
		credit.Flags = data[6]
	}
	if data[7] != nil {
		if credit.Status, ok = data[7].(string); !ok {
			return nil, errors.New("unable to type assert funding credit status")
		}
	}
	if data[11] != nil {
		if credit.Rate, ok = data[11].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit rate")
		}
	}
	if data[12] != nil {
		var period float64
		if period, ok = data[12].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit period")
		}
		credit.Period = int64(period)
	}
	if data[13] != nil {
		var opened float64
		if opened, ok = data[13].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit opened")
		}
		credit.Opened = time.UnixMilli(int64(opened))
	}
	if data[14] != nil {
		var lastPayout float64
		if lastPayout, ok = data[14].(float64); !ok {
			return nil, errors.New("unable to type assert last funding credit payout")
		}
		credit.LastPayout = time.UnixMilli(int64(lastPayout))
	}
	if data[15] != nil {
		var notify float64
		if notify, ok = data[15].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit notify")
		}
		credit.Notify = notify == 1
	}
	if data[16] != nil {
		var hidden float64
		if hidden, ok = data[16].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit hidden")
		}
		credit.Hidden = hidden == 1
	}
	if data[18] != nil {
		var renew float64
		if renew, ok = data[18].(float64); !ok {
			return nil, errors.New("unable to type assert funding credit renew")
		}
		credit.Renew = renew == 1
	}
	if data[19] != nil {
		if credit.RateReal, ok = data[19].(float64); !ok {
			return nil, errors.New("unable to type assert rate funding credit real")
		}
	}
	if data[20] != nil {
		var noClose float64
		if noClose, ok = data[20].(float64); !ok {
			return nil, errors.New("unable to type assert no funding credit close")
		}
		credit.NoClose = noClose == 1
	}
	if includePositionPair {
		if data[21] != nil {
			if credit.PositionPair, ok = data[21].(string); !ok {
				return nil, errors.New("unable to type assert funding credit position pair")
			}
		}
	}
	return &credit, nil
}

func (b *Bitfinex) wsHandleOrder(data []any) {
	var od order.Detail
	var err error
	od.Exchange = b.Name
	if data[0] != nil {
		if id, ok := data[0].(float64); ok {
			od.OrderID = strconv.FormatFloat(id, 'f', -1, 64)
		}
	}
	if data[16] != nil {
		if price, ok := data[16].(float64); ok {
			od.Price = price
		}
	}
	if data[7] != nil {
		if amount, ok := data[7].(float64); ok {
			od.Amount = amount
		}
	}
	if data[6] != nil {
		if remainingAmount, ok := data[6].(float64); ok {
			od.RemainingAmount = remainingAmount
		}
	}
	if data[7] != nil && data[6] != nil {
		if executedAmount, ok := data[7].(float64); ok {
			od.ExecutedAmount = executedAmount - od.RemainingAmount
		}
	}
	if data[4] != nil {
		if date, ok := data[4].(float64); ok {
			od.Date = time.Unix(int64(date)*1000, 0)
		}
	}
	if data[5] != nil {
		if lastUpdated, ok := data[5].(float64); ok {
			od.LastUpdated = time.Unix(int64(lastUpdated)*1000, 0)
		}
	}
	if data[2] != nil {
		if p, ok := data[3].(string); ok {
			od.Pair, od.AssetType, err = b.GetRequestFormattedPairAndAssetType(p[1:])
			if err != nil {
				b.Websocket.DataHandler <- err
				return
			}
		}
	}
	if data[8] != nil {
		if ordType, ok := data[8].(string); ok {
			oType, err := order.StringToOrderType(ordType)
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  od.OrderID,
					Err:      err,
				}
			}
			od.Type = oType
		}
	}
	if data[13] != nil {
		if combinedStatus, ok := data[13].(string); ok {
			statusParts := strings.Split(combinedStatus, " @ ")
			oStatus, err := order.StringToOrderStatus(statusParts[0])
			if err != nil {
				b.Websocket.DataHandler <- order.ClassificationError{
					Exchange: b.Name,
					OrderID:  od.OrderID,
					Err:      err,
				}
			}
			od.Status = oStatus
		}
	}
	b.Websocket.DataHandler <- &od
}

// WsInsertSnapshot add the initial orderbook snapshot when subscribed to a channel
func (b *Bitfinex) WsInsertSnapshot(p currency.Pair, assetType asset.Item, books []WebsocketBook, fundingRate bool) error {
	if len(books) == 0 {
		return errors.New("no orderbooks submitted")
	}
	var book orderbook.Book
	book.Bids = make(orderbook.Levels, 0, len(books))
	book.Asks = make(orderbook.Levels, 0, len(books))
	for i := range books {
		item := orderbook.Level{
			ID:     books[i].ID,
			Amount: books[i].Amount,
			Price:  books[i].Price,
			Period: books[i].Period,
		}
		if fundingRate {
			if item.Amount < 0 {
				item.Amount *= -1
				book.Bids = append(book.Bids, item)
			} else {
				book.Asks = append(book.Asks, item)
			}
		} else {
			if books[i].Amount > 0 {
				book.Bids = append(book.Bids, item)
			} else {
				item.Amount *= -1
				book.Asks = append(book.Asks, item)
			}
		}
	}

	book.Asset = assetType
	book.Pair = p
	book.Exchange = b.Name
	book.PriceDuplication = true
	book.IsFundingRate = fundingRate
	book.ValidateOrderbook = b.ValidateOrderbook
	book.LastUpdated = time.Now() // Not included in snapshot
	return b.Websocket.Orderbook.LoadSnapshot(&book)
}

// WsUpdateOrderbook updates the orderbook list, removing and adding to the
// orderbook sides
func (b *Bitfinex) WsUpdateOrderbook(c *subscription.Subscription, p currency.Pair, assetType asset.Item, book []WebsocketBook, sequenceNo int64, fundingRate bool) error {
	if c == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	if len(c.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	orderbookUpdate := orderbook.Update{
		Asset:      assetType,
		Pair:       p,
		Bids:       make([]orderbook.Level, 0, len(book)),
		Asks:       make([]orderbook.Level, 0, len(book)),
		UpdateTime: time.Now(), // Not included in update
	}

	for i := range book {
		item := orderbook.Level{
			ID:     book[i].ID,
			Amount: book[i].Amount,
			Price:  book[i].Price,
			Period: book[i].Period,
		}

		if book[i].Price > 0 {
			orderbookUpdate.Action = orderbook.UpdateOrInsertAction
			if fundingRate {
				if book[i].Amount < 0 {
					item.Amount *= -1
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				} else {
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				}
			} else {
				if book[i].Amount > 0 {
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				} else {
					item.Amount *= -1
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				}
			}
		} else {
			orderbookUpdate.Action = orderbook.DeleteAction
			if fundingRate {
				if book[i].Amount == 1 {
					// delete bid
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				} else {
					// delete ask
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				}
			} else {
				if book[i].Amount == 1 {
					// delete bid
					orderbookUpdate.Bids = append(orderbookUpdate.Bids, item)
				} else {
					// delete ask
					orderbookUpdate.Asks = append(orderbookUpdate.Asks, item)
				}
			}
		}
	}

	chanID, ok := c.Key.(int)
	if !ok {
		return common.GetTypeAssertError("int", c.Key, "ChanID") // Should be impossible
	}

	cMtx.Lock()
	checkme := checksumStore[chanID]
	if checkme == nil {
		cMtx.Unlock()
		return b.Websocket.Orderbook.Update(&orderbookUpdate)
	}
	checksumStore[chanID] = nil
	cMtx.Unlock()

	if checkme.Sequence+1 == sequenceNo {
		// Sequence numbers get dropped, if checksum is not in line with
		// sequence, do not check.
		ob, err := b.Websocket.Orderbook.GetOrderbook(p, assetType)
		if err != nil {
			return fmt.Errorf("cannot calculate websocket checksum: book not found for %s %s %w",
				p,
				assetType,
				err)
		}

		if err = validateCRC32(ob, checkme.Token); err != nil {
			log.Errorf(log.WebsocketMgr, "%s websocket orderbook update error, will resubscribe orderbook: %v", b.Name, err)
			if e2 := b.resubOrderbook(c); e2 != nil {
				log.Errorf(log.WebsocketMgr, "%s error resubscribing orderbook: %v", b.Name, e2)
			}
			return err
		}
	}

	return b.Websocket.Orderbook.Update(&orderbookUpdate)
}

// resubOrderbook resubscribes the orderbook after a consistency error, probably a failed checksum,
// which forces a fresh snapshot. If we don't do this the orderbook will keep erroring and drifting.
// Flushing the orderbook happens immediately, but the ReSub itself is a go routine to avoid blocking the WS data channel
func (b *Bitfinex) resubOrderbook(c *subscription.Subscription) error {
	if c == nil {
		return fmt.Errorf("%w: Subscription param", common.ErrNilPointer)
	}
	if len(c.Pairs) != 1 {
		return subscription.ErrNotSinglePair
	}
	if err := b.Websocket.Orderbook.InvalidateOrderbook(c.Pairs[0], c.Asset); err != nil {
		// Non-fatal error
		log.Errorf(log.ExchangeSys, "%s error flushing orderbook: %v", b.Name, err)
	}

	// Resub will block so we have to do this in a goro
	go func() {
		if err := b.Websocket.ResubscribeToChannel(b.Websocket.Conn, c); err != nil {
			log.Errorf(log.ExchangeSys, "%s error resubscribing orderbook: %v", b.Name, err)
		}
	}()

	return nil
}

// generateSubscriptions returns a list of subscriptions from the configured subscriptions feature
func (b *Bitfinex) generateSubscriptions() (subscription.List, error) {
	return b.Features.Subscriptions.ExpandTemplates(b)
}

// GetSubscriptionTemplate returns a subscription channel template
func (b *Bitfinex) GetSubscriptionTemplate(_ *subscription.Subscription) (*template.Template, error) {
	return template.New("master.tmpl").Funcs(sprig.FuncMap()).Funcs(template.FuncMap{
		"subToMap": subToMap,
		"removeSpotFromMargin": func(ap map[asset.Item]currency.Pairs) string {
			spotPairs, _ := b.GetEnabledPairs(asset.Spot)
			return removeSpotFromMargin(ap, spotPairs)
		},
	}).Parse(subTplText)
}

// ConfigureWS to send checksums and sequence numbers
func (b *Bitfinex) ConfigureWS(ctx context.Context) error {
	return b.Websocket.Conn.SendJSONMessage(ctx, request.Unset, map[string]any{
		"event": "conf",
		"flags": bitfinexChecksumFlag + bitfinexWsSequenceFlag,
	})
}

// Subscribe sends a websocket message to receive data from channels
func (b *Bitfinex) Subscribe(subs subscription.List) error {
	ctx := context.TODO()
	var err error
	if subs, err = subs.ExpandTemplates(b); err != nil {
		return err
	}
	return b.ParallelChanOp(ctx, subs, b.subscribeToChan, 1)
}

// Unsubscribe sends a websocket message to stop receiving data from channels
func (b *Bitfinex) Unsubscribe(subs subscription.List) error {
	ctx := context.TODO()
	var err error
	if subs, err = subs.ExpandTemplates(b); err != nil {
		return err
	}
	return b.ParallelChanOp(ctx, subs, b.unsubscribeFromChan, 1)
}

// subscribeToChan handles a single subscription and parses the result
// on success it adds the subscription to the websocket
func (b *Bitfinex) subscribeToChan(ctx context.Context, subs subscription.List) error {
	if len(subs) != 1 {
		return subscription.ErrNotSinglePair
	}

	s := subs[0]
	req := map[string]any{
		"event": "subscribe",
	}
	if err := json.Unmarshal([]byte(s.QualifiedChannel), &req); err != nil {
		return err
	}

	// subId is a single round-trip identifier that provides linking sub requests to chanIDs
	// Although docs only mention subId for wsBookChannel, it works for all chans
	subID := strconv.FormatInt(b.Websocket.Conn.GenerateMessageID(false), 10)
	req["subId"] = subID

	// Add a temporary Key so we can find this Sub when we get the resp without delay or context switch
	// Otherwise we might drop the first messages after the subscribed resp
	s.Key = subID // Note subID string type avoids conflicts with later chanID key
	if err := b.Websocket.AddSubscriptions(b.Websocket.Conn, s); err != nil {
		return fmt.Errorf("%w Channel: %s Pair: %s", err, s.Channel, s.Pairs)
	}

	// Always remove the temporary subscription keyed by subID
	defer func() {
		_ = b.Websocket.RemoveSubscriptions(b.Websocket.Conn, s)
	}()

	respRaw, err := b.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, "subscribe:"+subID, req)
	if err != nil {
		return fmt.Errorf("%w: Channel: %s Pair: %s", err, s.Channel, s.Pairs)
	}

	if err = b.getErrResp(respRaw); err != nil {
		wErr := fmt.Errorf("%w: Channel: %s Pair: %s", err, s.Channel, s.Pairs)
		b.Websocket.DataHandler <- wErr
		return wErr
	}

	return nil
}

// unsubscribeFromChan sends a websocket message to stop receiving data from a channel
func (b *Bitfinex) unsubscribeFromChan(ctx context.Context, subs subscription.List) error {
	if len(subs) != 1 {
		return errors.New("subscription batching limited to 1")
	}
	s := subs[0]
	chanID, ok := s.Key.(int)
	if !ok {
		return common.GetTypeAssertError("int", s.Key, "subscription.Key")
	}

	req := map[string]any{
		"event":  "unsubscribe",
		"chanId": chanID,
	}

	respRaw, err := b.Websocket.Conn.SendMessageReturnResponse(ctx, request.Unset, "unsubscribe:"+strconv.Itoa(chanID), req)
	if err != nil {
		return err
	}

	if err := b.getErrResp(respRaw); err != nil {
		wErr := fmt.Errorf("%w: ChanId: %v", err, chanID)
		b.Websocket.DataHandler <- wErr
		return wErr
	}

	return b.Websocket.RemoveSubscriptions(b.Websocket.Conn, s)
}

// getErrResp takes a json response string and looks for an error event type
// If found it parses the error code and message as a wrapped error and returns it
// It might log parsing errors about the nature of the error
// If the error message is not defined it will return a wrapped common.ErrUnknownError
func (b *Bitfinex) getErrResp(resp []byte) error {
	event, err := jsonparser.GetUnsafeString(resp, "event")
	if err != nil {
		return fmt.Errorf("%w 'event': %w from message: %s", common.ErrParsingWSField, err, resp)
	}
	if event != "error" {
		return nil
	}
	errCode, err := jsonparser.GetInt(resp, "code")
	if err != nil {
		log.Errorf(log.ExchangeSys, "%s %s 'code': %s from message: %s", b.Name, common.ErrParsingWSField, err, resp)
	}

	var apiErr error
	if msg, e2 := jsonparser.GetString(resp, "msg"); e2 != nil {
		log.Errorf(log.ExchangeSys, "%s %s 'msg': %s from message: %s", b.Name, common.ErrParsingWSField, e2, resp)
		apiErr = common.ErrUnknownError
	} else {
		apiErr = errors.New(msg)
	}
	return fmt.Errorf("%w (code: %d)", apiErr, errCode)
}

// WsSendAuth sends a authenticated event payload
func (b *Bitfinex) WsSendAuth(ctx context.Context) error {
	creds, err := b.GetCredentials(ctx)
	if err != nil {
		return err
	}

	nonce := strconv.FormatInt(time.Now().Unix(), 10)
	payload := "AUTH" + nonce

	hmac, err := crypto.GetHMAC(crypto.HashSHA512_384, []byte(payload), []byte(creds.Secret))
	if err != nil {
		return err
	}

	return b.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, WsAuthRequest{
		Event:         "auth",
		APIKey:        creds.Key,
		AuthPayload:   payload,
		AuthSig:       hex.EncodeToString(hmac),
		AuthNonce:     nonce,
		DeadManSwitch: 0,
	})
}

// WsNewOrder authenticated new order request
func (b *Bitfinex) WsNewOrder(ctx context.Context, data *WsNewOrderRequest) (string, error) {
	data.CustomID = b.Websocket.AuthConn.GenerateMessageID(false)
	req := makeRequestInterface(wsOrderNew, data)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, data.CustomID, req)
	if err != nil {
		return "", err
	}
	if resp == nil {
		return "", errors.New(b.Name + " - Order message not returned")
	}
	var respData []any
	err = json.Unmarshal(resp, &respData)
	if err != nil {
		return "", err
	}

	if len(respData) < 3 {
		return "", errors.New("unexpected respData length")
	}
	responseDataDetail, ok := respData[2].([]any)
	if !ok {
		return "", errors.New("unable to type assert respData")
	}

	if len(responseDataDetail) < 4 {
		return "", errors.New("invalid responseDataDetail length")
	}

	responseOrderDetail, ok := responseDataDetail[4].([]any)
	if !ok {
		return "", errors.New("unable to type assert responseOrderDetail")
	}
	var orderID string
	if responseOrderDetail[0] != nil {
		if ordID, ordOK := responseOrderDetail[0].(float64); ordOK && ordID > 0 {
			orderID = strconv.FormatFloat(ordID, 'f', -1, 64)
		}
	}
	var errorMessage, errCode string
	if len(responseDataDetail) > 6 {
		errCode, ok = responseDataDetail[6].(string)
		if !ok {
			return "", errors.New("unable to type assert errCode")
		}
	}
	if len(responseDataDetail) > 7 {
		errorMessage, ok = responseDataDetail[7].(string)
		if !ok {
			return "", errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return orderID, errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}
	return orderID, nil
}

// WsModifyOrder authenticated modify order request
func (b *Bitfinex) WsModifyOrder(ctx context.Context, data *WsUpdateOrderRequest) error {
	req := makeRequestInterface(wsOrderUpdate, data)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, data.OrderID, req)
	if err != nil {
		return err
	}
	if resp == nil {
		return errors.New(b.Name + " - Order message not returned")
	}

	var responseData []any
	err = json.Unmarshal(resp, &responseData)
	if err != nil {
		return err
	}
	if len(responseData) < 3 {
		return errors.New("unexpected responseData length")
	}
	responseOrderData, ok := responseData[2].([]any)
	if !ok {
		return errors.New("unable to type assert responseOrderData")
	}
	var errorMessage, errCode string
	if len(responseOrderData) > 6 {
		errCode, ok = responseOrderData[6].(string)
		if !ok {
			return errors.New("unable to type assert errCode")
		}
	}
	if len(responseOrderData) > 7 {
		errorMessage, ok = responseOrderData[7].(string)
		if !ok {
			return errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}
	return nil
}

// WsCancelMultiOrders authenticated cancel multi order request
func (b *Bitfinex) WsCancelMultiOrders(ctx context.Context, orderIDs []int64) error {
	cancel := WsCancelGroupOrdersRequest{
		OrderID: orderIDs,
	}
	req := makeRequestInterface(wsCancelMultipleOrders, cancel)
	return b.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, req)
}

// WsCancelOrder authenticated cancel order request
func (b *Bitfinex) WsCancelOrder(ctx context.Context, orderID int64) error {
	cancel := WsCancelOrderRequest{
		OrderID: orderID,
	}
	req := makeRequestInterface(wsOrderCancel, cancel)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, orderID, req)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%v - Order %v failed to cancel", b.Name, orderID)
	}
	var responseData []any
	err = json.Unmarshal(resp, &responseData)
	if err != nil {
		return err
	}
	if len(responseData) < 3 {
		return errors.New("unexpected responseData length")
	}
	responseOrderData, ok := responseData[2].([]any)
	if !ok {
		return errors.New("unable to type assert responseOrderData")
	}
	var errorMessage, errCode string
	if len(responseOrderData) > 6 {
		errCode, ok = responseOrderData[6].(string)
		if !ok {
			return errors.New("unable to type assert errCode")
		}
	}
	if len(responseOrderData) > 7 {
		errorMessage, ok = responseOrderData[7].(string)
		if !ok {
			return errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}
	return nil
}

// WsCancelAllOrders authenticated cancel all orders request
func (b *Bitfinex) WsCancelAllOrders(ctx context.Context) error {
	cancelAll := WsCancelAllOrdersRequest{All: 1}
	req := makeRequestInterface(wsCancelMultipleOrders, cancelAll)
	return b.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, req)
}

// WsNewOffer authenticated new offer request
func (b *Bitfinex) WsNewOffer(ctx context.Context, data *WsNewOfferRequest) error {
	req := makeRequestInterface(wsFundingOfferNew, data)
	return b.Websocket.AuthConn.SendJSONMessage(ctx, request.Unset, req)
}

// WsCancelOffer authenticated cancel offer request
func (b *Bitfinex) WsCancelOffer(ctx context.Context, orderID int64) error {
	cancel := WsCancelOrderRequest{
		OrderID: orderID,
	}
	req := makeRequestInterface(wsFundingOfferCancel, cancel)
	resp, err := b.Websocket.AuthConn.SendMessageReturnResponse(ctx, request.Unset, orderID, req)
	if err != nil {
		return err
	}
	if resp == nil {
		return fmt.Errorf("%v - Order %v failed to cancel", b.Name, orderID)
	}
	var responseData []any
	err = json.Unmarshal(resp, &responseData)
	if err != nil {
		return err
	}
	if len(responseData) < 3 {
		return errors.New("unexpected responseData length")
	}
	responseOrderData, ok := responseData[2].([]any)
	if !ok {
		return errors.New("unable to type assert responseOrderData")
	}
	var errorMessage, errCode string
	if len(responseOrderData) > 6 {
		errCode, ok = responseOrderData[6].(string)
		if !ok {
			return errors.New("unable to type assert errCode")
		}
	}
	if len(responseOrderData) > 7 {
		errorMessage, ok = responseOrderData[7].(string)
		if !ok {
			return errors.New("unable to type assert errorMessage")
		}
	}
	if strings.EqualFold(errCode, wsError) {
		return errors.New(b.Name + " - " + errCode + ": " + errorMessage)
	}

	return nil
}

func makeRequestInterface(channelName string, data any) []any {
	return []any{0, channelName, nil, data}
}

func validateCRC32(book *orderbook.Book, token uint32) error {
	// Order ID's need to be sub-sorted in ascending order, this needs to be
	// done on the main book to ensure that we do not cut price levels out below
	reOrderByID(book.Bids)
	reOrderByID(book.Asks)

	// R0 precision calculation is based on order ID's and amount values
	var bids, asks []orderbook.Level
	for i := range 25 {
		if i < len(book.Bids) {
			bids = append(bids, book.Bids[i])
		}
		if i < len(book.Asks) {
			asks = append(asks, book.Asks[i])
		}
	}

	// ensure '-' (negative amount) is passed back to string buffer as
	// this is needed for calcs - These get swapped if funding rate
	bidmod := float64(1)
	if book.IsFundingRate {
		bidmod = -1
	}

	askMod := float64(-1)
	if book.IsFundingRate {
		askMod = 1
	}

	var check strings.Builder
	for i := range 25 {
		if i < len(bids) {
			check.WriteString(strconv.FormatInt(bids[i].ID, 10))
			check.WriteString(":")
			check.WriteString(strconv.FormatFloat(bidmod*bids[i].Amount, 'f', -1, 64))
			check.WriteString(":")
		}

		if i < len(asks) {
			check.WriteString(strconv.FormatInt(asks[i].ID, 10))
			check.WriteString(":")
			check.WriteString(strconv.FormatFloat(askMod*asks[i].Amount, 'f', -1, 64))
			check.WriteString(":")
		}
	}

	checksumStr := strings.TrimSuffix(check.String(), ":")
	checksum := crc32.ChecksumIEEE([]byte(checksumStr))
	if checksum == token {
		return nil
	}
	return fmt.Errorf("invalid checksum for %s %s: calculated [%d] does not match [%d]",
		book.Asset,
		book.Pair,
		checksum,
		token)
}

// reOrderByID sub sorts orderbook items by its corresponding ID when price
// levels are the same. TODO: Deprecate and shift to buffer level insertion
// based off ascending ID.
func reOrderByID(depth []orderbook.Level) {
subSort:
	for x := 0; x < len(depth); {
		var subset []orderbook.Level
		// Traverse forward elements
		for y := x + 1; y < len(depth); y++ {
			if depth[x].Price == depth[y].Price &&
				// Period matching is for funding rates, this was undocumented
				// but these need to be matched with price for the correct ID
				// alignment
				depth[x].Period == depth[y].Period {
				// Append element to subset when price match occurs
				subset = append(subset, depth[y])
				// Traverse next
				continue
			}
			if len(subset) != 0 {
				// Append root element
				subset = append(subset, depth[x])
				// Sort IDs by ascending
				sort.Slice(subset, func(i, j int) bool {
					return subset[i].ID < subset[j].ID
				})
				// Re-align elements with sorted ID subset
				for z := range subset {
					depth[x+z] = subset[z]
				}
			}
			// When price is not matching change checked element to root
			x = y
			continue subSort
		}
		break
	}
}

// subToMap returns a json object of request params for subscriptions
func subToMap(s *subscription.Subscription, a asset.Item, p currency.Pair) map[string]any {
	c := s.Channel
	if name, ok := subscriptionNames[s.Channel]; ok {
		c = name
	}
	req := map[string]any{
		"channel": c,
	}

	var fundingPeriod string
	for k, v := range s.Params {
		switch k {
		case CandlesPeriodKey:
			s, ok := v.(string)
			if !ok {
				panic(common.GetTypeAssertError("string", v, "subscription.CandlesPeriodKey"))
			}
			fundingPeriod = ":" + s
		case "key", "symbol", "len":
			panic(fmt.Errorf("%w: %s", errParamNotAllowed, k)) // Ensure user's Params aren't silently overwritten
		default:
			req[k] = v
		}
	}

	if s.Levels != 0 {
		req["len"] = s.Levels
	}

	prefix := "t"
	if a == asset.MarginFunding {
		prefix = "f"
	}

	pairFmt := currency.PairFormat{Uppercase: true}
	if needsDelimiter := p.Len() > 6; needsDelimiter {
		pairFmt.Delimiter = ":"
	}
	symbol := p.Format(pairFmt).String()
	if c == wsCandlesChannel {
		req["key"] = "trade:" + s.Interval.Short() + ":" + prefix + symbol + fundingPeriod
	} else {
		req["symbol"] = prefix + symbol
	}

	return req
}

// removeSpotFromMargin removes spot pairs from margin pairs in the supplied AssetPairs map to avoid duplicate subscriptions
func removeSpotFromMargin(ap map[asset.Item]currency.Pairs, spotPairs currency.Pairs) string {
	if p, ok := ap[asset.Margin]; ok {
		ap[asset.Margin] = p.Remove(spotPairs...)
	}
	return ""
}

const subTplText = `
{{- removeSpotFromMargin $.AssetPairs -}}
{{ range $asset, $pairs := $.AssetPairs }}
	{{- range $p := $pairs  -}}
		{{- subToMap $.S $asset $p | mustToJson }}
		{{- $.PairSeparator }}
	{{- end -}}
	{{ $.AssetSeparator }}
{{- end -}}
`
