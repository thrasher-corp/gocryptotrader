package bitget

import (
	"net/url"
	"time"
)

// Params is used within functions to make the setting of parameters easier
type Params struct {
	url.Values
}

// UnixTimestamp is a type used to unmarshal unix millisecond timestamps returned from the exchange
type UnixTimestamp time.Time

// AnnResp holds information on announcements
type AnnResp struct {
	Data []struct {
		AnnID    string        `json:"annId"`
		AnnTitle string        `json:"annTitle"`
		AnnDesc  string        `json:"annDesc"`
		CTime    UnixTimestamp `json:"cTime"`
		Language string        `json:"language"`
		AnnURL   string        `json:"annUrl"`
	} `json:"data"`
}

// TimeResp holds information on the current server time
type TimeResp struct {
	Data struct {
		ServerTime UnixTimestamp `json:"serverTime"`
	} `json:"data"`
}

// TradeRateResp holds information on the current maker and taker fee rates
type TradeRateResp struct {
	Data struct {
		MakerFeeRate float64 `json:"makerFeeRate,string"`
		TakerFeeRate float64 `json:"takerFeeRate,string"`
	} `json:"data"`
}

// SpotTrResp holds information on spot transactions
type SpotTrResp struct {
	Data []struct {
		ID          int64         `json:"id,string"`
		Coin        string        `json:"coin"`
		SpotTaxType string        `json:"spotTaxType"`
		Amount      float64       `json:"amount,string"`
		Fee         float64       `json:"fee,string"`
		Balance     float64       `json:"balance,string"`
		Timestamp   UnixTimestamp `json:"ts"`
	} `json:"data"`
}

// FutureTrResp holds information on futures transactions
type FutureTrResp struct {
	Data []struct {
		ID            int64         `json:"id,string"`
		Symbol        string        `json:"symbol"`
		MarginCoin    string        `json:"marginCoin"`
		FutureTaxType string        `json:"futureTaxType"`
		Amount        float64       `json:"amount,string"`
		Fee           float64       `json:"fee,string"`
		Timestamp     UnixTimestamp `json:"ts"`
	} `json:"data"`
}

// MarginTrResp holds information on margin transactions
type MarginTrResp struct {
	Data []struct {
		ID            int64         `json:"id,string"`
		Coin          string        `json:"coin"`
		Symbol        string        `json:"symbol"`
		MarginTaxType string        `json:"marginTaxType"`
		Amount        float64       `json:"amount,string"`
		Fee           float64       `json:"fee,string"`
		Total         float64       `json:"total,string"`
		Timestamp     UnixTimestamp `json:"ts"`
	} `json:"data"`
}

// P2PTrResp holds information on P2P transactions
type P2PTrResp struct {
	Data []struct {
		ID         int64         `json:"id,string"`
		Coin       string        `json:"coin"`
		P2PTaxType string        `json:"p2pTaxType"`
		Total      float64       `json:"total,string"`
		Timestamp  UnixTimestamp `json:"ts"`
	} `json:"data"`
}

// P2PMerResp holds information on P2P merchant lists
type P2PMerListResp struct {
	Data struct {
		MerchantList []struct {
			RegisterTime        UnixTimestamp `json:"registerTime"`
			NickName            string        `json:"nickName"`
			IsOnline            string        `json:"isOnline"`
			MerchantID          int64         `json:"merchantId,string"`
			AvgPaymentTime      int64         `json:"avgPaymentTime,string"`
			AvgReleaseTime      int64         `json:"avgReleaseTime,string"`
			TotalTrades         int64         `json:"totalTrades,string"`
			TotalBuy            int64         `json:"totalBuy,string"`
			TotalSell           int64         `json:"totalSell,string"`
			TotalCompletionRate float64       `json:"totalCompletionRate,string"`
			Trades30D           int64         `json:"trades30d,string"`
			Sell30D             float64       `json:"sell30d,string"`
			Buy30D              float64       `json:"buy30d,string"`
			CompletionRate30D   float64       `json:"completionRate30d,string"`
		} `json:"merchantList"`
		MinMerchantID int64 `json:"minMerchantId,string"`
	} `json:"data"`
}

// YesNoBool is a type used to unmarshal strings that are either "yes" or "no" into bools
type YesNoBool bool

// P2PMerInfoResp holds information on P2P merchant information
type P2PMerInfoResp struct {
	Data struct {
		RegisterTime        UnixTimestamp `json:"registerTime"`
		NickName            string        `json:"nickName"`
		MerchantID          int64         `json:"merchantId,string"`
		AvgPaymentTime      int64         `json:"avgPaymentTime,string"`
		AvgReleaseTime      int64         `json:"avgReleaseTime,string"`
		TotalTrades         int64         `json:"totalTrades,string"`
		TotalBuy            int64         `json:"totalBuy,string"`
		TotalSell           int64         `json:"totalSell,string"`
		TotalCompletionRate float64       `json:"totalCompletionRate,string"`
		Trades30D           int64         `json:"trades30d,string"`
		Sell30D             float64       `json:"sell30d,string"`
		Buy30D              float64       `json:"buy30d,string"`
		CompletionRate30D   float64       `json:"completionRate30d,string"`
		KYCStatus           YesNoBool     `json:"kycStatus"`
		EmailBindStatus     YesNoBool     `json:"emailBindStatus"`
		MobileBindStatus    YesNoBool     `json:"mobileBindStatus"`
		Email               string        `json:"email"`
		Mobile              string        `json:"mobile"`
	} `json:"data"`
}
