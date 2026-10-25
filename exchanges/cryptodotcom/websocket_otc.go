package cryptodotcom

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
)

// RequestOTCQuote requests quotes on one or more legs over the websocket's trade connection, as Exchange.RequestOTCQuote
// does over REST, and returns the quote request's IDs
func (w WebsocketAPI) RequestOTCQuote(ctx context.Context, req *OTCQuoteRequest) ([]OTCQuoteRequestResponse, error) {
	params, err := newOTCQuoteRequestParams(req)
	if err != nil {
		return nil, err
	}
	var resp *dataResponse[OTCQuoteRequestResponse]
	if err := w.e.SendWebsocketRequest(ctx, "private/otc/request-quote", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp.Data, nil
}

// RequestOTCDeal requests a deal on a quote over the websocket's trade connection, as Exchange.RequestOTCDeal does over
// REST
func (w WebsocketAPI) RequestOTCDeal(ctx context.Context, req *OTCDealRequest) (*OTCDealResponse, error) {
	params, err := newOTCDealParams(req)
	if err != nil {
		return nil, err
	}
	var resp *OTCDealResponse
	if err := w.e.SendWebsocketRequest(ctx, "private/otc/request-deal", params, &resp); err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, common.ErrNoResponse
	}
	return resp, nil
}
