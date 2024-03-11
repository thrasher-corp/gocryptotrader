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

// P2POrdersResp holds information on P2P orders
type P2POrdersResp struct {
	Data struct {
		OrderList []struct {
			OrderID        int64         `json:"orderId,string"`
			OrderNum       int64         `json:"orderNo,string"`
			AdvNum         int64         `json:"advNo,string"`
			Side           string        `json:"side"`
			Count          float64       `json:"count,string"`
			FiatCurrency   string        `json:"fiat"`
			CryptoCurrency string        `json:"coin"`
			Price          float64       `json:"price,string"`
			WithdrawTime   UnixTimestamp `json:"withdrawTime"`
			RepresentTime  UnixTimestamp `json:"representTime"`
			ReleaseTime    UnixTimestamp `json:"releaseTime"`
			PaymentTime    UnixTimestamp `json:"paymentTime"`
			Amount         float64       `json:"amount,string"`
			Status         string        `json:"status"`
			BuyerRealName  string        `json:"buyerRealName"`
			SellerRealName string        `json:"sellerRealName"`
			CreationTime   UnixTimestamp `json:"ctime"`
			UpdateTime     UnixTimestamp `json:"utime"`
			PaymentInfo    struct {
				PayMethodName string `json:"paymethodName"`
				PayMethodID   string `json:"paymethodId"`
				PayMethodInfo []struct {
					Name     string    `json:"name"`
					Required YesNoBool `json:"required"`
					Type     string    `json:"type"`
					Value    string    `json:"value"`
				} `json:"paymethodInfo"`
			} `json:"paymentInfo"`
		} `json:"orderList"`
		MinOrderID int64 `json:"minOrderId,string"`
	} `json:"data"`
}

// P2PAdListResp holds information on P2P advertisements
type P2PAdListResp struct {
	Data struct {
		AdList []struct {
			AdID            int64         `json:"adId,string"`
			AdvNum          int64         `json:"advNo,string"`
			Side            string        `json:"side"`
			AdSize          float64       `json:"adSize,string"`
			Size            float64       `json:"size,string"`
			CryptoCurrency  string        `json:"coin"`
			Price           float64       `json:"price,string"`
			CryptoPrecision int64         `json:"coinPrecision,string"`
			FiatCurrency    string        `json:"fiat"`
			FiatPrecision   int64         `json:"fiatPrecision,string"`
			FiatSymbol      string        `json:"fiatSymbol"`
			Status          string        `json:"status"`
			Hide            YesNoBool     `json:"hide"`
			MaxTradeAmount  float64       `json:"maxTradeAmount,string"`
			MinTradeAmount  float64       `json:"minTradeAmount,string"`
			PayDuration     int64         `json:"payDuration,string"`
			TurnoverNum     int64         `json:"turnoverNum,string"`
			TurnoverRate    float64       `json:"turnoverRate,string"`
			Label           string        `json:"label"`
			CreationTime    UnixTimestamp `json:"ctime"`
			UpdateTime      UnixTimestamp `json:"utime"`
			UserLimitList   struct {
				MinCompleteNum     int64     `json:"minCompleteNum,string"`
				MaxCompleteNum     int64     `json:"maxCompleteNum,string"`
				PlaceOrderNum      int64     `json:"placeOrderNum,string"`
				AllowMerchantPlace YesNoBool `json:"allowMerchantPlace"`
				CompleteRate30D    float64   `json:"completeRate30d,string"`
				Country            string    `json:"country"`
			} `json:"userLimitList"`
			PaymentMethodList []struct {
				PaymentMethod string `json:"paymentMethod"`
				PaymentID     int64  `json:"paymentId,string"`
				PaymentInfo   []struct {
					Required bool   `json:"required"`
					Name     string `json:"name"`
					Type     string `json:"type"`
				} `json:"paymentInfo"`
			} `json:"paymentMethodList"`
			MerchantCertifiedList []struct {
				ImageURL string `json:"imageUrl"`
				Desc     string `json:"desc"`
			} `json:"merchantCertifiedList"`
		} `json:"advList"`
	} `json:"data"`
}

// CrVirSubResp contains information returned when creating virtual sub-accounts
type CrVirSubResp struct {
	Data struct {
		FailureList []struct {
			SubAccountName string `json:"subaAccountName"`
		} `json:"failureList"`
		SuccessList []struct {
			SubAccountUID  string        `json:"subAccountUid"`
			SubAccountName string        `json:"subaAccountName"`
			Status         string        `json:"status"`
			PermList       []string      `json:"permList"`
			Label          string        `json:"label"`
			CreationTime   UnixTimestamp `json:"cTime"`
			UpdateTime     UnixTimestamp `json:"uTime"`
		} `json:"successList"`
	} `json:"data"`
}

// SuccessBool is a type used to unmarshal strings that are either "success" or "failure" into bools
type SuccessBool bool

// ModVirSubResp contains information returned when modifying virtual sub-accounts
type ModVirSubResp struct {
	Data struct {
		Success SuccessBool `json:"result"`
	} `json:"data"`
}

// CrSubAccAPIKeyResp contains information returned when simultaneously creating a sub-account and
// an API key
type CrSubAccAPIKeyResp struct {
	Data []struct {
		SubAccountUID    string   `json:"subAccountUid"`
		SubAccountName   string   `json:"subAccountName"`
		Label            string   `json:"label"`
		SubAccountAPIKey string   `json:"subAccountApiKey"`
		SecretKey        string   `json:"secretKey"`
		PermList         []string `json:"permList"`
		IPList           []string `json:"ipList"`
	} `json:"data"`
}

// GetVirSubResp contains information on the user's virtual sub-accounts
type GetVirSubResp struct {
	Data struct {
		SubAccountList []struct {
			SubAccountUID  string        `json:"subAccountUid"`
			SubAccountName string        `json:"subAccountName"`
			Label          string        `json:"label"`
			Status         string        `json:"status"`
			PermList       []string      `json:"permList"`
			CreationTime   UnixTimestamp `json:"cTime"`
			UpdateTime     UnixTimestamp `json:"uTime"`
		} `json:"subAccountList"`
		EndID int64 `json:"endId,string"`
	} `json:"data"`
}

// AlterAPIKeyResp contains information returned when creating or modifying an API key
type AlterAPIKeyResp struct {
	Data struct {
		SubAccountUID    string   `json:"subAccountUid"`
		SubAccountApiKey string   `json:"subAccountApiKey"`
		SecretKey        string   `json:"secretKey"`
		PermList         []string `json:"permList"`
		Label            string   `json:"label"`
		IPList           []string `json:"ipList"`
	} `json:"data"`
}

// GetAPIKeyResp contains information on the user's API keys
type GetAPIKeyResp struct {
	Data []struct {
		SubAccountUID    string   `json:"subAccountUid"`
		SubAccountApiKey string   `json:"subAccountApiKey"`
		IPList           []string `json:"ipList"`
		PermList         []string `json:"permList"`
		Label            string   `json:"label"`
	} `json:"data"`
}

// ConvertCoinsResp
type ConvertCoinsResp struct {
	Data []struct {
		Coin      string  `json:"coin"`
		Available float64 `json:"available,string"`
		MaxAmount float64 `json:"maxAmount,string"`
		MinAmount float64 `json:"minAmount,string"`
	} `json:"data"`
}

// QuotedPriceResp
type QuotedPriceResp struct {
}
