package bybit

import (
	"cmp"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"maps"
	"math"
	"math/rand/v2"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

func TestEndpointBudgets(t *testing.T) {
	t.Parallel()
	assert.Equal(t, []rateLimitBudget{{5 * time.Second, 570}, {100 * time.Millisecond, 12}}, ipBudgets, "the HTTP IP limit should allow 570 of Bybit's 600 requests in 5 seconds, and 12 in any 100ms")
	rows := []struct {
		epl      request.EndpointLimit
		interval time.Duration
		requests int
		endpoint string
	}{
		// Trade
		{createOrderEPL, time.Second, 10, "POST /v5/order/create, linear and inverse"},
		{createOptionOrderEPL, time.Second, 10, "POST /v5/order/create, options"},
		{createSpotOrderEPL, time.Second, 20, "POST /v5/order/create, spot"},
		{amendOrderEPL, time.Second, 10, "POST /v5/order/amend, linear and inverse"},
		{amendOptionOrderEPL, time.Second, 10, "POST /v5/order/amend, options"},
		{amendSpotOrderEPL, time.Second, 10, "POST /v5/order/amend, spot"},
		{cancelOrderEPL, time.Second, 10, "POST /v5/order/cancel, linear and inverse"},
		{cancelOptionOrderEPL, time.Second, 10, "POST /v5/order/cancel, options"},
		{cancelSpotOrderEPL, time.Second, 20, "POST /v5/order/cancel, spot"},
		{cancelAllOrdersEPL, time.Second, 10, "POST /v5/order/cancel-all, linear and inverse"},
		{cancelAllOptionOrdersEPL, time.Second, 1, "POST /v5/order/cancel-all, options"},
		{cancelAllSpotOrdersEPL, time.Second, 20, "POST /v5/order/cancel-all, spot"},
		{createBatchOrderEPL, time.Second, 10, "POST /v5/order/create-batch, linear and inverse"},
		{createOptionBatchOrderEPL, time.Second, 10, "POST /v5/order/create-batch, options"},
		{createSpotBatchOrderEPL, time.Second, 20, "POST /v5/order/create-batch, spot"},
		{amendBatchOrderEPL, time.Second, 10, "POST /v5/order/amend-batch, linear and inverse"},
		{amendOptionBatchOrderEPL, time.Second, 10, "POST /v5/order/amend-batch, options"},
		{amendSpotBatchOrderEPL, time.Second, 20, "POST /v5/order/amend-batch, spot"},
		{cancelBatchOrderEPL, time.Second, 10, "POST /v5/order/cancel-batch, linear and inverse"},
		{cancelOptionBatchOrderEPL, time.Second, 10, "POST /v5/order/cancel-batch, options"},
		{cancelSpotBatchOrderEPL, time.Second, 20, "POST /v5/order/cancel-batch, spot"},
		{setDisconnectCancelAllEPL, time.Second, 5, "POST /v5/order/disconnected-cancel-all"},
		{preCheckOrderEPL, time.Second, 10, "POST /v5/order/pre-check, linear and inverse"},
		{preCheckOptionOrderEPL, time.Second, 10, "POST /v5/order/pre-check, options"},
		{preCheckSpotOrderEPL, time.Second, 20, "POST /v5/order/pre-check, spot"},
		{getOpenOrdersEPL, time.Second, 50, "GET /v5/order/realtime"},
		{getTradeOrderHistoryEPL, time.Second, 50, "GET /v5/order/history"},
		{getExecutionEPL, time.Second, 50, "GET /v5/execution/list"},
		{getBorrowQuotaEPL, time.Second, 50, "GET /v5/order/spot-borrow-check"},
		{getFuturesComboBotLimitEPL, time.Second, 10, "POST /v5/fcombobot/getlimit"},
		{createFuturesComboBotEPL, time.Second, 10, "POST /v5/fcombobot/create"},
		{closeFuturesComboBotEPL, time.Second, 10, "POST /v5/fcombobot/close"},
		{getFuturesComboBotDetailEPL, time.Second, 10, "POST /v5/fcombobot/detail"},
		{validateFuturesGridInputEPL, time.Second, 10, "POST /v5/fgridbot/validate"},
		{createFuturesGridBotEPL, time.Second, 10, "POST /v5/fgridbot/create"},
		{closeFuturesGridBotEPL, time.Second, 10, "POST /v5/fgridbot/close"},
		{getFuturesGridBotDetailEPL, time.Second, 10, "POST /v5/fgridbot/detail"},
		{getFuturesMartingaleBotLimitEPL, time.Second, 10, "POST /v5/fmartingalebot/getlimit"},
		{createFuturesMartingaleBotEPL, time.Second, 100, "POST /v5/fmartingalebot/create"},
		{closeFuturesMartingaleBotEPL, time.Second, 10, "POST /v5/fmartingalebot/close"},
		{getFuturesMartingaleBotDetailEPL, time.Second, 10, "POST /v5/fmartingalebot/detail"},
		{validateSpotGridInputEPL, time.Second, 100, "POST /v5/grid/validate-input"},
		{createSpotGridBotEPL, time.Second, 3, "POST /v5/grid/create-grid"},
		{closeSpotGridBotEPL, time.Second, 3, "POST /v5/grid/close-grid"},
		{getSpotGridBotDetailEPL, time.Second, 10, "POST /v5/grid/query-grid-detail"},
		{createDCABotEPL, time.Second, 3, "POST /v5/dca/create-bot"},
		{closeDCABotEPL, time.Second, 3, "POST /v5/dca/close-bot"},
		{createStrategyOrderEPL, time.Second, 100, "POST /v5/strategy/create"},
		{getStrategyListEPL, time.Second, 200, "GET /v5/strategy/list"},
		{getStrategyOrderListEPL, time.Second, 200, "GET /v5/strategy/order-list"},
		{stopStrategyEPL, time.Second, 100, "POST /v5/strategy/stop"},
		// Position
		{getPositionInfoEPL, time.Second, 50, "GET /v5/position/list"},
		{getClosedPNLEPL, time.Second, 50, "GET /v5/position/closed-pnl"},
		{getClosedOptionsPositionsEPL, time.Second, 50, "GET /v5/position/get-closed-positions"},
		{getMovePositionHistoryEPL, time.Second, 10, "GET /v5/position/move-history"},
		{setLeverageLevelEPL, time.Second, 10, "POST /v5/position/set-leverage"},
		{switchPositionModeEPL, time.Second, 10, "POST /v5/position/switch-mode"},
		{setTradingStopEPL, time.Second, 10, "POST /v5/position/trading-stop"},
		{setAutoAddMarginEPL, time.Second, 10, "POST /v5/position/set-auto-add-margin"},
		{addOrReduceMarginEPL, time.Second, 10, "POST /v5/position/add-margin"},
		{confirmNewRiskLimitEPL, time.Second, 10, "POST /v5/position/confirm-pending-mmr"},
		{movePositionsEPL, time.Second, 10, "POST /v5/position/move-positions"},
		// Account
		{getWalletBalanceEPL, time.Second, 50, "GET /v5/account/wallet-balance"},
		{getTransferableAmountEPL, time.Second, 50, "GET /v5/account/withdrawal"},
		{getBorrowHistoryEPL, time.Second, 50, "GET /v5/account/borrow-history"},
		{manualBorrowEPL, time.Second, 1, "POST /v5/account/borrow"},
		{manualRepayEPL, time.Second, 1, "POST /v5/account/repay"},
		{manualRepayWithoutAssetConversionEPL, time.Second, 1, "POST /v5/account/no-convert-repay"},
		{getCollateralInfoEPL, time.Second, 50, "GET /v5/account/collateral-info"},
		{getCoinGreeksEPL, time.Second, 50, "GET /v5/asset/coin-greeks"},
		{getTransactionLogEPL, time.Second, 25, "GET /v5/account/transaction-log"},
		{getFeeRateEPL, time.Second, 5, "GET /v5/account/fee-rate"},
		{getAccountInfoEPL, time.Second, 50, "GET /v5/account/info"},
		{getAccountInstrumentsInfoEPL, time.Second, 10, "GET /v5/account/instruments-info"},
		{getPayInfoEPL, time.Second, 50, "GET /v5/account/pay-info"},
		{getDCPInfoEPL, time.Second, 5, "GET /v5/account/query-dcp-info"},
		{getSMPGroupIDEPL, time.Second, 5, "GET /v5/account/smp-group"},
		{getTradeInfoForAnalysisEPL, time.Second, 50, "GET /v5/account/trade-info-for-analysis"},
		{getTradeBehaviourConfigEPL, time.Second, 50, "GET /v5/account/user-setting-config"},
		{setMMPEPL, time.Second, 5, "POST /v5/account/mmp-modify"},
		{resetMMPEPL, time.Second, 5, "POST /v5/account/mmp-reset"},
		{repayLiabilityEPL, time.Second, 1, "POST /v5/account/quick-repayment"},
		{batchSetCollateralCoinEPL, time.Second, 5, "POST /v5/account/set-collateral-switch-batch"},
		{setPriceLimitBehaviourEPL, time.Second, 10, "POST /v5/account/set-limit-px-action"},
		{setMarginModeEPL, time.Second, 5, "POST /v5/account/set-margin-mode"},
		{upgradeToUnifiedAccountEPL, time.Second, 1, "POST /v5/account/upgrade-to-uta"},
		// Asset
		{getTransferableCoinEPL, time.Minute, 60, "GET /v5/asset/transfer/query-transfer-coin-list"},
		{getInternalTransferRecordsEPL, time.Minute, 60, "GET /v5/asset/transfer/query-inter-transfer-list"},
		{getSubUIDEPL, time.Minute, 60, "GET /v5/asset/transfer/query-sub-member-list"},
		{getUniversalTransferRecordsEPL, time.Second, 5, "GET /v5/asset/transfer/query-universal-transfer-list"},
		{getAllCoinsBalanceEPL, time.Second, 5, "GET /v5/asset/transfer/query-account-coins-balance"},
		{getSingleCoinBalanceEPL, time.Second, 450, "GET /v5/asset/transfer/query-account-coin-balance"},
		{getAssetOverviewEPL, time.Second, 50, "GET /v5/asset/asset-overview"},
		{getWithdrawableAmountEPL, time.Second, 300, "GET /v5/asset/withdraw/withdrawable-amount"},
		{getDepositRecordsEPL, time.Minute, 100, "GET /v5/asset/deposit/query-record"},
		{getSubDepositRecordsEPL, time.Minute, 300, "GET /v5/asset/deposit/query-sub-member-record"},
		{getMasterDepositAddressEPL, time.Minute, 300, "GET /v5/asset/deposit/query-address"},
		{getSubDepositAddressEPL, time.Minute, 300, "GET /v5/asset/deposit/query-sub-member-address"},
		{getWithdrawalRecordsEPL, time.Minute, 300, "GET /v5/asset/withdraw/query-record"},
		{getCoinInfoEPL, time.Second, 5, "GET /v5/asset/coin/query-info"},
		{getCoinExchangeRecordsEPL, time.Minute, 600, "GET /v5/asset/exchange/order-record"},
		{getConvertCoinListEPL, time.Second, 30, "GET /v5/asset/exchange/query-coin-list"},
		{getSmallBalanceExchangeHistoryEPL, time.Second, 5, "GET /v5/asset/covert/small-balance-history"},
		{getSmallBalanceCoinsEPL, time.Second, 10, "GET /v5/asset/covert/small-balance-list"},
		{getDeliveryRecordEPL, time.Second, 50, "GET /v5/asset/delivery-record"},
		{getInternalDepositRecordsOffChainEPL, time.Second, 300, "GET /v5/asset/deposit/query-internal-record"},
		{getFiatBalanceEPL, time.Second, 1000, "GET /v5/fiat/balance-query"},
		{getFiatTradingPairListEPL, time.Second, 1000, "GET /v5/fiat/query-coin-list"},
		{getFiatConvertStatusEPL, time.Second, 1000, "GET /v5/fiat/trade-query"},
		{getFiatConvertHistoryEPL, time.Second, 1000, "GET /v5/fiat/query-trade-history"},
		{getFundingAccountTransactionHistoryEPL, time.Second, 30, "GET /v5/asset/fundinghistory"},
		{getPortfolioMarginInfoEPL, time.Second, 50, "GET /v5/asset/portfolio-margin"},
		{getUSDCSessionSettlementEPL, time.Second, 50, "GET /v5/asset/settlement-record"},
		{getTotalMembersAssetsEPL, time.Second, 50, "GET /v5/asset/total-members-assets"},
		{getAvailableVASPsEPL, time.Second, 1, "GET /v5/asset/withdraw/vasp/list"},
		{getWithdrawalAddressListEPL, time.Second, 300, "GET /v5/asset/withdraw/query-address"},
		{createInternalTransferEPL, time.Minute, 60, "POST /v5/asset/transfer/inter-transfer"},
		{createUniversalTransferEPL, time.Second, 5, "POST /v5/asset/transfer/universal-transfer"},
		{withdrawCurrencyEPL, time.Second, 5, "POST /v5/asset/withdraw/create"},
		{cancelWithdrawalEPL, time.Minute, 60, "POST /v5/asset/withdraw/cancel"},
		{requestQuoteEPL, time.Second, 20, "POST /v5/asset/exchange/quote-apply"},
		{confirmQuoteEPL, time.Second, 20, "POST /v5/asset/exchange/convert-execute"},
		{confirmSmallBalanceQuoteEPL, time.Second, 5, "POST /v5/asset/covert/small-balance-execute"},
		{requestSmallBalanceQuoteEPL, time.Second, 5, "POST /v5/asset/covert/get-quote"},
		{setDepositAccountEPL, time.Second, 300, "POST /v5/asset/deposit/deposit-to-account"},
		{confirmFiatQuoteEPL, time.Second, 100, "POST /v5/fiat/trade-execute"},
		{requestFiatQuoteEPL, time.Second, 1000, "POST /v5/fiat/quote-apply"},
		{getConvertStatusEPL, time.Second, 50, "GET /v5/asset/exchange/convert-result-query"},
		{getConvertHistoryEPL, time.Second, 50, "GET /v5/asset/exchange/query-convert-history"},
		{getFiatReferencePriceEPL, time.Second, 1000, "GET /v5/fiat/reference-price"},
		// User
		{createNewSubUserIDEPL, time.Second, 1, "POST /v5/user/create-sub-member"},
		{createSubUIDAPIKeyEPL, time.Second, 1, "POST /v5/user/create-sub-api"},
		{freezeSubUIDEPL, time.Second, 5, "POST /v5/user/frozen-sub-member"},
		{modifyMasterAPIKeyEPL, time.Second, 5, "POST /v5/user/update-api"},
		{modifySubAPIKeyEPL, time.Second, 5, "POST /v5/user/update-sub-api"},
		{deleteMasterAPIKeyEPL, time.Second, 5, "POST /v5/user/delete-api"},
		{deleteSubAccountAPIKeyEPL, time.Second, 5, "POST /v5/user/delete-sub-api"},
		{getSubUIDListEPL, time.Second, 10, "GET /v5/user/query-sub-members"},
		{getAPIKeyInformationEPL, time.Second, 10, "GET /v5/user/query-api"},
		{getAffiliateUserInfoEPL, time.Second, 10, "GET /v5/user/aff-customer-info"},
		{signAgreementEPL, time.Second, 20, "POST /v5/user/agreement"},
		{getSubUIDListUnlimitedEPL, time.Second, 5, "GET /v5/user/submembers"},
		{getFundCustodialSubAccountsEPL, time.Second, 5, "GET /v5/user/escrow_sub_members"},
		{getSubAccountAllAPIKeysEPL, time.Second, 10, "GET /v5/user/sub-apikeys"},
		{getUIDWalletTypeEPL, time.Second, 10, "GET /v5/user/get-member-type"},
		{deleteSubUIDEPL, time.Second, 5, "POST /v5/user/del-submember"},
		{getFriendReferralsEPL, time.Second, 10, "GET /v5/user/invitation/referrals"},
		// Spot margin trade
		{toggleMarginTradeEPL, time.Second, 5, "POST /v5/spot-margin-trade/switch-mode"},
		{setSpotMarginTradeLeverageEPL, time.Second, 5, "POST /v5/spot-margin-trade/set-leverage"},
		{getSpotMarginStatusAndLeverageEPL, time.Second, 50, "GET /v5/spot-margin-trade/state"},
		{getMaxBorrowableAmountEPL, time.Second, 50, "GET /v5/spot-margin-trade/max-borrowable"},
		{getSpotMarginCoinStateEPL, time.Second, 50, "GET /v5/spot-margin-trade/coinstate"},
		{getSpotMarginAvailableAmountToRepayEPL, time.Second, 50, "GET /v5/spot-margin-trade/repayment-available-amount"},
		{setSpotMarginAutoRepayModeEPL, time.Second, 5, "POST /v5/spot-margin-trade/set-auto-repay-mode"},
		{getSpotMarginAutoRepayModeEPL, time.Second, 50, "GET /v5/spot-margin-trade/get-auto-repay-mode"},
		{getSpotMarginFixedRateBorrowOrderQuoteEPL, time.Second, 5, "GET /v5/spot-margin-trade/fixedborrow-order-quote"},
		{spotMarginFixedRateBorrowEPL, time.Second, 1, "POST /v5/spot-margin-trade/fixedborrow"},
		{renewSpotMarginFixedRateBorrowEPL, time.Second, 1, "POST /v5/spot-margin-trade/fixedborrow-renew"},
		{getSpotMarginFixedRateBorrowOrderInfoEPL, time.Second, 5, "GET /v5/spot-margin-trade/fixedborrow-order-info"},
		{getSpotMarginFixedRateBorrowContractInfoEPL, time.Second, 5, "GET /v5/spot-margin-trade/fixedborrow-contract-info"},
		{getSpotMarginLiabilityEPL, time.Second, 5, "GET /v5/spot-margin-trade/liability"},
		// Spread trading
		{createSpreadOrderEPL, time.Second, 20, "POST /v5/spread/order/create"},
		{amendSpreadOrderEPL, time.Second, 20, "POST /v5/spread/order/amend"},
		{cancelSpreadOrderEPL, time.Second, 20, "POST /v5/spread/order/cancel"},
		{cancelAllSpreadOrdersEPL, time.Second, 5, "POST /v5/spread/order/cancel-all"},
		{getSpreadOpenOrdersEPL, time.Second, 50, "GET /v5/spread/order/realtime"},
		{getSpreadOrderHistoryEPL, time.Second, 50, "GET /v5/spread/order/history"},
		{getSpreadTradeHistoryEPL, time.Second, 50, "GET /v5/spread/execution/list"},
		{getSpreadMaxQuantityEPL, time.Second, 50, "GET /v5/spread/max-qty"},
		// RFQ
		{cancelAllRFQQuotesEPL, time.Second, 50, "POST /v5/rfq/cancel-all-quotes"},
		{cancelRFQQuoteEPL, time.Second, 50, "POST /v5/rfq/cancel-quote"},
		{acceptNonLPQuoteEPL, time.Second, 50, "POST /v5/rfq/accept-other-quote"},
		{cancelAllRFQsEPL, time.Second, 50, "POST /v5/rfq/cancel-all-rfq"},
		{createRFQQuoteEPL, time.Second, 50, "POST /v5/rfq/create-quote"},
		{createRFQEPL, time.Second, 50, "POST /v5/rfq/create-rfq"},
		{executeRFQQuoteEPL, time.Second, 50, "POST /v5/rfq/execute-quote"},
		{cancelRFQEPL, time.Second, 50, "POST /v5/rfq/cancel-rfq"},
		{getRFQPublicTradesEPL, time.Second, 50, "GET /v5/rfq/public-trades"},
		{getRFQQuotesEPL, time.Second, 50, "GET /v5/rfq/quote-list"},
		{getRFQQuotesRealtimeEPL, time.Second, 50, "GET /v5/rfq/quote-realtime"},
		{getRFQConfigEPL, time.Second, 50, "GET /v5/rfq/config"},
		{getRFQsEPL, time.Second, 50, "GET /v5/rfq/rfq-list"},
		{getRFQsRealtimeEPL, time.Second, 50, "GET /v5/rfq/rfq-realtime"},
		{getRFQTradeHistoryEPL, time.Second, 50, "GET /v5/rfq/trade-list"},
		{getRFQDetailsEPL, time.Second, 50, "GET /v5/rfq/rfq-detail-list"},
		// Institutional loan
		{getInstitutionalLoanProductInfoEPL, time.Second, 100, "GET /v5/ins-loan/product-infos"},
		{getInstitutionalLoanMarginCoinInfoEPL, time.Second, 100, "GET /v5/ins-loan/ensure-tokens-convert"},
		{getInstitutionalLoanOrdersEPL, time.Second, 20, "GET /v5/ins-loan/loan-order"},
		{getInstitutionalLoanRepaymentOrdersEPL, time.Second, 20, "GET /v5/ins-loan/repaid-history"},
		{getInstitutionalLoanLTVEPL, time.Second, 20, "GET /v5/ins-loan/ltv-convert"},
		{bindOrUnbindUIDEPL, time.Second, 1, "POST /v5/ins-loan/association-uid"},
		{repayInstitutionalLoanEPL, time.Second, 1, "POST /v5/ins-loan/repay-loan"},
		{getInstitutionalLoanDelayLiquidationStatusEPL, time.Second, 2, "GET /v5/ins-loan/delay-liq-status"},
		// Endpoints the rate limit table leaves out
		// Demo trading
		{requestDemoTradingFundsEPL, time.Minute, 1, "POST /v5/account/demo-apply-money"},
		{createDemoAccountEPL, time.Second, 5, "POST /v5/user/create-demo-member"},
		// Tax
		{batchCreateTaxReportsEPL, time.Second, 5, "POST /v5/fht/compliance/tax/private/batch_create"},
		{getBatchTaxReportsStatusEPL, time.Second, 20, "GET /v5/fht/compliance/tax/private/batch_query"},
		// Crypto loan
		{getCryptoLoanMaxCollateralReductionAmountEPL, time.Second, 5, "GET /v5/crypto-loan-common/max-collateral-amount"},
		{adjustCryptoLoanCollateralAmountEPL, time.Second, 1, "POST /v5/crypto-loan-common/adjust-ltv"},
		{getCryptoLoanCollateralAdjustmentHistoryEPL, time.Second, 5, "GET /v5/crypto-loan-common/adjustment-history"},
		{getCryptoLoanPositionEPL, time.Second, 5, "GET /v5/crypto-loan-common/position"},
		{getCryptoLoanMaxLoanAmountEPL, time.Second, 5, "POST /v5/crypto-loan-common/max-loan"},
		{getFlexibleLoanAvailableInventoryEPL, time.Second, 5, "GET /v5/crypto-loan-flexible/available-inventory"},
		{borrowFlexibleLoanEPL, time.Second, 1, "POST /v5/crypto-loan-flexible/borrow"},
		{repayFlexibleLoanEPL, time.Second, 1, "POST /v5/crypto-loan-flexible/repay"},
		{repayFlexibleLoanWithCollateralEPL, time.Second, 1, "POST /v5/crypto-loan-flexible/repay-collateral"},
		{getFlexibleLoansEPL, time.Second, 5, "GET /v5/crypto-loan-flexible/ongoing-coin"},
		{getFlexibleLoanBorrowingHistoryEPL, time.Second, 5, "GET /v5/crypto-loan-flexible/borrow-history"},
		{getFlexibleLoanRepaymentHistoryEPL, time.Second, 5, "GET /v5/crypto-loan-flexible/repayment-history"},
		{getFixedLoanAvailableInventoryEPL, time.Second, 5, "GET /v5/crypto-loan-fixed/available-inventory"},
		{createFixedLoanBorrowOrderEPL, time.Second, 1, "POST /v5/crypto-loan-fixed/borrow"},
		{renewFixedLoanBorrowOrderEPL, time.Second, 1, "POST /v5/crypto-loan-fixed/renew"},
		{createFixedLoanSupplyOrderEPL, time.Second, 1, "POST /v5/crypto-loan-fixed/supply"},
		{cancelFixedLoanBorrowOrderEPL, time.Second, 1, "POST /v5/crypto-loan-fixed/borrow-order-cancel"},
		{cancelFixedLoanSupplyOrderEPL, time.Second, 1, "POST /v5/crypto-loan-fixed/supply-order-cancel"},
		{getFixedLoanBorrowContractInfoEPL, time.Second, 5, "GET /v5/crypto-loan-fixed/borrow-contract-info"},
		{getFixedLoanSupplyContractInfoEPL, time.Second, 5, "GET /v5/crypto-loan-fixed/supply-contract-info"},
		{getFixedLoanBorrowOrderInfoEPL, time.Second, 5, "GET /v5/crypto-loan-fixed/borrow-order-info"},
		{getFixedLoanRenewOrderInfoEPL, time.Second, 5, "GET /v5/crypto-loan-fixed/renew-info"},
		{getFixedLoanSupplyOrderInfoEPL, time.Second, 5, "GET /v5/crypto-loan-fixed/supply-order-info"},
		{repayFixedLoanEPL, time.Second, 1, "POST /v5/crypto-loan-fixed/fully-repay"},
		{repayFixedLoanWithCollateralEPL, time.Second, 1, "POST /v5/crypto-loan-fixed/repay-collateral"},
		{getFixedLoanRepaymentHistoryEPL, time.Second, 5, "GET /v5/crypto-loan-fixed/repayment-history"},
		// Exchange broker
		{getExchangeBrokerEarningEPL, time.Second, 10, "GET /v5/broker/earnings-info"},
		{getExchangeBrokerAccountInfoEPL, time.Second, 10, "GET /v5/broker/account-info"},
		{getBrokerSubAccountDepositRecordsEPL, time.Minute, 300, "GET /v5/broker/asset/query-sub-member-deposit-record"},
		{setBrokerRateLimitEPL, time.Second, 1, "POST /v5/broker/apilimit/set"},
		{getBrokerRateLimitCapEPL, time.Second, 5, "GET /v5/broker/apilimit/query-cap"},
		{getBrokerAllRateLimitsEPL, time.Second, 1, "GET /v5/broker/apilimit/query-all"},
		// Earn
		{getEarnCouponListEPL, time.Second, 10, "GET /v5/earn/coupons"},
		{getFixedSavingProductInfoEPL, time.Second, 50, "GET /v5/earn/fixed-term/product"},
		{placeFixedSavingOrderEPL, time.Second, 5, "POST /v5/earn/fixed-term/place-order"},
		{redeemFixedSavingEPL, time.Second, 5, "POST /v5/earn/fixed-term/redeem"},
		{getFixedSavingPositionInfoEPL, time.Second, 10, "GET /v5/earn/fixed-term/position"},
		{getFixedSavingOrderListEPL, time.Second, 10, "GET /v5/earn/fixed-term/order"},
		{setFixedSavingAutoInvestEPL, time.Second, 5, "POST /v5/earn/fixed-term/position/auto-invest"},
		{getFlexibleSavingAutoSavingsEPL, time.Second, 10, "GET /v5/earn/flexible-saving/auto-savings"},
		{setFlexibleSavingAutoSavingsEPL, time.Second, 5, "POST /v5/earn/flexible-saving/auto-savings"},
		// Advanced earn
		{getAdvancedEarnProductInfoEPL, time.Second, 50, "GET /v5/earn/advance/product"},
		{getAdvancedEarnProductQuoteEPL, time.Second, 50, "GET /v5/earn/advance/product-extra-info"},
		{placeAdvancedEarnOrderEPL, time.Second, 5, "POST /v5/earn/advance/place-order"},
		{getAdvancedEarnPositionInfoEPL, time.Second, 10, "GET /v5/earn/advance/position"},
		{getAdvancedEarnOrderInfoEPL, time.Second, 10, "GET /v5/earn/advance/order"},
		{getAdvancedEarnRedeemEstimatedAmountEPL, time.Second, 10, "GET /v5/earn/advance/get-redeem-est-amount-list"},
		{getLiquidityMiningProductInfoEPL, time.Second, 50, "GET /v5/earn/liquidity-mining/product"},
		{addLiquidityEPL, time.Second, 5, "POST /v5/earn/liquidity-mining/add-liquidity"},
		{removeLiquidityEPL, time.Second, 5, "POST /v5/earn/liquidity-mining/remove-liquidity"},
		{reinvestLiquidityMiningEPL, time.Second, 5, "POST /v5/earn/liquidity-mining/reinvest"},
		{addLiquidityMiningMarginEPL, time.Second, 5, "POST /v5/earn/liquidity-mining/add-margin"},
		{claimLiquidityMiningInterestEPL, time.Second, 5, "POST /v5/earn/liquidity-mining/claim-interest"},
		{getLiquidityMiningPositionInfoEPL, time.Second, 10, "GET /v5/earn/liquidity-mining/position"},
		{getLiquidityMiningOrderInfoEPL, time.Second, 10, "GET /v5/earn/liquidity-mining/order"},
		{getLiquidityMiningYieldRecordsEPL, time.Second, 10, "GET /v5/earn/liquidity-mining/yield-records"},
		{getLiquidityMiningLiquidationRecordsEPL, time.Second, 10, "GET /v5/earn/liquidity-mining/liquidation-records"},
		// Real world assets
		{getRWAProductListEPL, time.Second, 20, "GET /v5/earn/rwa/product"},
		{placeRWAOrderEPL, time.Second, 5, "POST /v5/earn/rwa/place-order"},
		{getRWAPositionListEPL, time.Second, 10, "GET /v5/earn/rwa/position"},
		{getRWAOrderListEPL, time.Second, 10, "GET /v5/earn/rwa/order"},
		{getRWANAVChartEPL, time.Second, 20, "GET /v5/earn/rwa/nav-chart"},
		// Web3 liquidity pools
		{executeWeb3LPStakeEPL, time.Second, 1, "POST /v5/alpha/lp/stake"},
		{executeWeb3LPRedeemEPL, time.Second, 1, "POST /v5/alpha/lp/redeem"},
		{getWeb3LPOrderListEPL, time.Second, 3, "POST /v5/alpha/lp/order-list"},
		{getWeb3LPPayTokenListEPL, time.Second, 5, "POST /v5/alpha/lp/pay-token-list"},
		{getWeb3LPPayTokenPriceEPL, time.Second, 5, "POST /v5/alpha/lp/pay-token-price"},
		{getWeb3LPPoolInfoEPL, time.Second, 5, "POST /v5/alpha/lp/pool-info"},
		{getWeb3LPPoolListEPL, time.Second, 5, "POST /v5/alpha/lp/pool-list"},
		{getWeb3LPPositionListEPL, time.Second, 3, "POST /v5/alpha/lp/position-list"},
		// Rate limits for pro accounts
		{setAPIRateLimitEPL, time.Second, 50, "POST /v5/apilimit/set"},
		{getAPIRateLimitEPL, time.Second, 50, "GET /v5/apilimit/query"},
		{getAPIRateLimitCapEPL, time.Second, 50, "GET /v5/apilimit/query-cap"},
		{getAllAPIRateLimitsEPL, time.Second, 50, "GET /v5/apilimit/query-all"},
		// P2P
		{getP2POnlineAdsEPL, time.Second, 10, "POST /v5/p2p/item/online"},
		{postP2PAdEPL, time.Second, 5, "POST /v5/p2p/item/create"},
		{removeP2PAdEPL, time.Second, 5, "POST /v5/p2p/item/cancel"},
		{updateP2PAdEPL, time.Second, 5, "POST /v5/p2p/item/update"},
		{getP2PMyAdsEPL, time.Second, 10, "POST /v5/p2p/item/personal/list"},
		{getP2PMyAdDetailsEPL, time.Second, 10, "POST /v5/p2p/item/info"},
		{getAllP2POrdersEPL, time.Second, 10, "POST /v5/p2p/order/simplifyList"},
		{getP2POrderDetailEPL, time.Second, 10, "POST /v5/p2p/order/info"},
		{getP2PPendingOrdersEPL, time.Second, 10, "POST /v5/p2p/order/pending/simplifyList"},
		{markP2POrderAsPaidEPL, time.Second, 5, "POST /v5/p2p/order/pay"},
		{releaseP2PAssetsEPL, time.Second, 5, "POST /v5/p2p/order/finish"},
		{reviewP2PSellerCancelOrderApplyEPL, time.Second, 5, "POST /v5/p2p/order/buyer/examine/sellerCancelOrderApply"},
		{getP2PAccountInformationEPL, time.Second, 10, "POST /v5/p2p/user/personal/info"},
		{getP2PCounterpartyUserInfoEPL, time.Second, 10, "POST /v5/p2p/user/order/personal/info"},
		{getP2PUserPaymentEPL, time.Second, 10, "POST /v5/p2p/user/payment/list"},
		{getP2PChatSessionListEPL, time.Second, 10, "POST /v5/p2p/chat/session/list_v1"},
		{getP2PChatSessionIDEPL, time.Second, 10, "POST /v5/p2p/chat/session/getSessionId"},
		{uploadP2PChatFileEPL, time.Second, 5, "POST /v5/p2p/oss/upload_file"},
		{sendP2PChatMessageEPL, time.Second, 5, "POST /v5/p2p/chat/message/send_v1"},
		{getP2PChatMessageListEPL, time.Second, 10, "POST /v5/p2p/chat/message/listpage_v1"},
	}
	for _, tc := range rows {
		assert.Equalf(t, rateLimitBudget{tc.interval, tc.requests}, endpointBudgets[tc.epl], "%s should have its documented budget", tc.endpoint)
	}
	assert.Len(t, endpointBudgets, len(rows), "endpointBudgets should hold exactly the documented budgets")
}

// endpointLimitNames returns the names of the endpoint limit constants in declaration order
func endpointLimitNames(t *testing.T) []string {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), "ratelimit.go", nil, 0)
	require.NoError(t, err, "ratelimit.go must parse")
	for _, decl := range f.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.CONST || len(gen.Specs) == 0 {
			continue
		}
		first, ok := gen.Specs[0].(*ast.ValueSpec)
		if !ok {
			continue
		}
		if sel, ok := first.Type.(*ast.SelectorExpr); !ok || sel.Sel.Name != "EndpointLimit" {
			continue
		}
		names := make([]string, 0, len(gen.Specs))
		for _, spec := range gen.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			require.True(t, ok, "every endpoint limit declaration must be a value spec")
			for _, name := range valueSpec.Names {
				names = append(names, name.Name)
			}
		}
		return names
	}
	require.FailNow(t, "ratelimit.go must declare the endpoint limits")
	return nil
}

// websocketOrderLimits returns the set of websocket order entry limits
func websocketOrderLimits() map[request.EndpointLimit]bool {
	limits := make(map[request.EndpointLimit]bool, 3*len(websocketOrderBudgets))
	for _, budgets := range websocketOrderBudgets {
		for _, budget := range []orderBudget{budgets.derivatives, budgets.option, budgets.spot} {
			limits[budget.websocket] = true
		}
	}
	return limits
}

func TestEndpointLimitsComplete(t *testing.T) {
	t.Parallel()
	names := endpointLimitNames(t)
	require.Equal(t, "defaultEPL", names[0], "defaultEPL must be the first endpoint limit")
	require.Equal(t, "wsSubscriptionEPL", names[len(names)-1], "wsSubscriptionEPL must be the last endpoint limit")
	require.Equal(t, int(wsSubscriptionEPL-defaultEPL)+1, len(names), "the endpoint limits must run from defaultEPL to wsSubscriptionEPL")
	assert.Greater(t, defaultEPL, request.UnAuth, "endpoint limits should not alias the request package's generic limits")

	websocketLimits := websocketOrderLimits()
	definitions := rateLimitDefinitions()
	for i, name := range names {
		epl := defaultEPL + request.EndpointLimit(i)
		limiter, defined := definitions[epl]
		if !assert.Truef(t, defined, "%s should be defined", name) {
			continue
		}
		_, hasBudget := endpointBudgets[epl]
		isWebsocket := websocketLimits[epl] || epl == wsSubscriptionEPL
		assert.Equalf(t, isWebsocket, strings.HasPrefix(name, "ws"), "%s should be named for whether it limits websocket requests", name)
		switch epl {
		case wsSubscriptionEPL:
			assert.Nil(t, limiter, "wsSubscriptionEPL should leave messages to the connection's limiter")
			continue
		case defaultEPL:
			assert.False(t, hasBudget, "defaultEPL should draw on the HTTP IP limit alone")
		default:
			assert.NotEqualf(t, hasBudget, isWebsocket, "%s should either have a budget of its own or be a websocket order entry limit", name)
		}
		if assert.NotNilf(t, limiter, "%s should have a limiter", name) {
			assert.Equalf(t, request.Weight(1), limiter.Weight(), "%s should take one request", name)
		}
	}
	assert.Len(t, definitions, len(names), "rateLimitDefinitions should define every endpoint limit and nothing else")
	assert.Len(t, rateLimits, len(names), "rateLimits should define every endpoint limit")
}

func TestRateLimitDefinitionsOwnBudgets(t *testing.T) {
	t.Parallel()
	definitions := rateLimitDefinitions()
	twins := make(map[request.EndpointLimit]request.EndpointLimit, 6*len(websocketOrderBudgets))
	for operation, budgets := range websocketOrderBudgets {
		for category, budget := range map[string]orderBudget{"linear and inverse": budgets.derivatives, cOption: budgets.option, cSpot: budgets.spot} {
			require.NotZerof(t, budget.websocket, "%s %s must have a websocket limit", operation, category)
			twins[budget.rest], twins[budget.websocket] = budget.websocket, budget.rest
		}
	}
	assert.Len(t, twins, 36, "each of the six websocket order entry operations should share its three category budgets with REST")
	for epl, budget := range endpointBudgets {
		exp := request.NewRateLimit(budget.interval, budget.requests).Limit()
		if twin, ok := twins[epl]; ok {
			// Order entry budgets keep a margin beyond Bybit's rolling second
			exp = request.NewRateLimit(budget.interval+orderBudgetMargin, budget.requests).Limit()
			assert.Equalf(t, exp, definitions[twin].Limit(), "websocket endpoint limit %d should allow its REST budget", twin)
		}
		assert.Equalf(t, exp, definitions[epl].Limit(), "endpoint limit %d should allow its budget", epl)
	}
	for key, limiter := range definitions {
		epl, ok := key.(request.EndpointLimit)
		require.Truef(t, ok, "definition key %v must be an endpoint limit", key)
		for otherKey, other := range definitions {
			exp := key == otherKey || twins[epl] == otherKey
			assert.Equalf(t, exp, limiter.SharesBudgetWith(other), "endpoint limit %d should share a budget of its own only with itself and its websocket or REST twin, not with %v", epl, otherKey)
		}
	}
}

// fillIPBudget sends on defaultEPL until both windows of the HTTP IP limit are full, the five second one over the 100ms
// ones it takes to fill, and returns when the first of those requests was sent
func fillIPBudget(t *testing.T, definitions request.RateLimitDefinitions) time.Time {
	t.Helper()
	start := time.Now()
	for range ipBudgets[0].requests {
		require.NoError(t, definitions[defaultEPL].RateLimit(t.Context()), "defaultEPL must take a request from the HTTP IP limit")
	}
	return start
}

// ipLimitedSends returns the most of the sends in any five seconds and in any 100ms, the windows of the HTTP IP limit
func ipLimitedSends(sends []time.Duration) (fiveSeconds, hundredMilliseconds int) {
	slices.Sort(sends)
	busiest := func(window time.Duration) int {
		var most int
		for i, first := range sends {
			end, _ := slices.BinarySearch(sends[i:], first+window)
			most = max(most, end)
		}
		return most
	}
	return busiest(ipBudgets[0].interval), busiest(ipBudgets[1].interval)
}

func TestRateLimitDefinitionsIPBudget(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		fill func(*testing.T, request.RateLimitDefinitions) time.Duration
	}{
		{"REST limits wait for the IP limit's 100ms window", func(t *testing.T, definitions request.RateLimitDefinitions) time.Duration {
			t.Helper()
			for range ipBudgets[1].requests {
				require.NoError(t, definitions[defaultEPL].RateLimit(request.WithDelayNotAllowed(t.Context())), "defaultEPL must take a request from the HTTP IP limit at once")
			}
			return ipBudgets[1].interval
		}},
		{"REST limits wait for the IP limit's five second window", func(t *testing.T, definitions request.RateLimitDefinitions) time.Duration {
			t.Helper()
			return ipBudgets[0].interval - time.Since(fillIPBudget(t, definitions))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
				definitions := rateLimitDefinitions()
				websocketLimits := websocketOrderLimits()
				noDelay := request.WithDelayNotAllowed(t.Context())
				untilRoom := tc.fill(t, definitions)
				require.Positive(t, untilRoom, "the IP limit must be full")
				// The bubble's clock stands still, so the IP limit stays full while every endpoint limit is checked
				for key, limiter := range definitions {
					epl, ok := key.(request.EndpointLimit)
					require.Truef(t, ok, "definition key %v must be an endpoint limit", key)
					if epl == wsSubscriptionEPL || websocketLimits[epl] {
						continue
					}
					assert.ErrorIsf(t, limiter.RateLimit(noDelay), request.ErrDelayNotAllowed, "REST endpoint limit %d should wait for the HTTP IP limit while its own budget is idle", epl)
				}
				// The refused REST checks reserved nothing, so the websocket limits find their shared budgets idle
				for epl := range websocketLimits {
					assert.NoErrorf(t, definitions[epl].RateLimit(noDelay), "websocket endpoint limit %d should not wait for the HTTP IP limit", epl)
				}
				time.Sleep(untilRoom)
				assert.NoError(t, definitions[getOpenOrdersEPL].RateLimit(noDelay), "a REST request should go once the HTTP IP limit's window has passed")
			})
		})
	}

	t.Run("a rate limit barrier admits REST requests the IP limit has room for", func(t *testing.T) {
		t.Parallel()
		restLimits := slices.Sorted(maps.Keys(endpointBudgets))
		for _, tc := range []struct {
			name         string
			participants uint
			alreadySent  int
			err          error
		}{
			{"2 from idle budgets", 2, 0, nil},
			{"12 from idle budgets", 12, 0, nil},
			{"13 from idle budgets", 13, 0, request.ErrDelayNotAllowed},
			{"2 with room for 1", 2, ipBudgets[1].requests - 1, request.ErrDelayNotAllowed},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
					definitions := rateLimitDefinitions()
					noDelay := request.WithDelayNotAllowed(t.Context())
					for range tc.alreadySent {
						require.NoError(t, definitions[defaultEPL].RateLimit(noDelay), "defaultEPL must take a request from the HTTP IP limit")
					}
					contexts, err := request.NewRateLimitBarrierContexts(t.Context(), tc.participants)
					require.NoError(t, err, "barrier contexts must be created")
					errs := make(chan error, tc.participants)
					for i, ctx := range contexts {
						go func() { errs <- definitions[restLimits[i]].RateLimit(ctx) }()
					}
					for range tc.participants {
						assert.ErrorIs(t, <-errs, tc.err, "every participant of the barrier should share its outcome")
					}
					if tc.err == nil {
						return
					}
					for range ipBudgets[1].requests - tc.alreadySent {
						assert.NoError(t, definitions[defaultEPL].RateLimit(noDelay), "a refused barrier should book nothing in the HTTP IP limit")
					}
					assert.ErrorIs(t, definitions[defaultEPL].RateLimit(noDelay), request.ErrDelayNotAllowed, "the HTTP IP limit should then be full")
				})
			})
		}
	})

	t.Run("no window holds more than the IP budget under a backlog", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			const group = 12
			definitions := rateLimitDefinitions()
			contexts, err := request.NewRateLimitBarrierContexts(t.Context(), group)
			require.NoError(t, err, "barrier contexts must be created")
			restLimits := slices.Sorted(maps.Keys(endpointBudgets))
			start := time.Now()
			var mu sync.Mutex
			sends := make([]time.Duration, 0, group+1300)
			record := func() {
				mu.Lock()
				sends = append(sends, time.Since(start))
				mu.Unlock()
			}
			var barrier common.ErrorCollector
			for i, ctx := range contexts {
				barrier.Go(func() error {
					if err := definitions[restLimits[i]].RateLimit(ctx); err != nil {
						return err
					}
					record()
					return nil
				})
			}
			require.NoError(t, barrier.Collect(), "a barrier the IP limit has room for must be admitted")
			var backlog common.ErrorCollector
			for i := range 1300 {
				epl := defaultEPL
				if i%10 == 0 {
					epl = getOpenOrdersEPL
				}
				backlog.Go(func() error {
					if err := definitions[epl].RateLimit(t.Context()); err != nil {
						return err
					}
					record()
					return nil
				})
			}
			require.NoError(t, backlog.Collect(), "a backlog of requests on the HTTP IP limit must not error")
			fiveSeconds, hundredMilliseconds := ipLimitedSends(sends)
			assert.Equal(t, ipBudgets[0].requests, fiveSeconds, "the busiest five seconds should fill the IP budget without exceeding it")
			assert.Equal(t, ipBudgets[1].requests, hundredMilliseconds, "the busiest 100ms should fill the burst budget without exceeding it")
		})
	})
}

func TestRateLimitDefinitionsWebsocketOrderBudgets(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		definitions := rateLimitDefinitions()
		noDelay := request.WithDelayNotAllowed(t.Context())
		for operation, budgets := range websocketOrderBudgets {
			for _, budget := range []orderBudget{budgets.derivatives, budgets.option, budgets.spot} {
				// Bybit counts each second as it rolls, so the whole budget can go at once
				for range endpointBudgets[budget.rest].requests {
					require.NoErrorf(t, definitions[budget.websocket].RateLimit(noDelay), "%s websocket limit %d must take its whole budget at once", operation, budget.websocket)
				}
				assert.ErrorIsf(t, definitions[budget.websocket].RateLimit(noDelay), request.ErrDelayNotAllowed, "%s websocket limit %d should wait once its budget is full", operation, budget.websocket)
				assert.ErrorIsf(t, definitions[budget.rest].RateLimit(noDelay), request.ErrDelayNotAllowed, "%s REST limit %d should wait for the requests its websocket limit took", operation, budget.rest)
			}
		}
		// A second is not enough: the budgets keep a margin, so that orders sent a window apart reach Bybit a second apart
		time.Sleep(time.Second)
		for operation, budgets := range websocketOrderBudgets {
			for _, budget := range []orderBudget{budgets.derivatives, budgets.option, budgets.spot} {
				assert.ErrorIsf(t, definitions[budget.websocket].RateLimit(noDelay), request.ErrDelayNotAllowed, "%s websocket limit %d should still wait a second after its budget filled", operation, budget.websocket)
			}
		}
		time.Sleep(orderBudgetMargin)
		for operation, budgets := range websocketOrderBudgets {
			for _, budget := range []orderBudget{budgets.derivatives, budgets.option, budgets.spot} {
				for range endpointBudgets[budget.rest].requests {
					assert.NoErrorf(t, definitions[budget.websocket].RateLimit(noDelay), "%s websocket limit %d should have its whole budget again once its window has passed", operation, budget.websocket)
				}
			}
		}
	})
}

func TestRateLimitDefinitionsWebsocketOrderAheadOfRESTOrder(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		fill func(*testing.T, request.RateLimitDefinitions) time.Duration
	}{
		{"the IP limit's 100ms window", func(t *testing.T, definitions request.RateLimitDefinitions) time.Duration {
			t.Helper()
			for range ipBudgets[1].requests {
				require.NoError(t, definitions[defaultEPL].RateLimit(t.Context()), "defaultEPL must take a request from the HTTP IP limit")
			}
			return ipBudgets[1].interval
		}},
		{"the IP limit's five second window", func(t *testing.T, definitions request.RateLimitDefinitions) time.Duration {
			t.Helper()
			return ipBudgets[0].interval - time.Since(fillIPBudget(t, definitions))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
				definitions := rateLimitDefinitions()
				untilRoom := tc.fill(t, definitions)
				start := time.Now()
				restSent := make(chan time.Duration, 1)
				go func() {
					assert.NoError(t, definitions[createOrderEPL].RateLimit(t.Context()), "the REST order should go once the HTTP IP limit has room")
					restSent <- time.Since(start)
				}()
				synctest.Wait()
				time.Sleep(10 * time.Millisecond)
				assert.NoError(t, definitions[wsCreateOrderEPL].RateLimit(request.WithDelayNotAllowed(t.Context())), "a websocket order should not wait for a REST order the HTTP IP limit holds back")
				assert.Equal(t, untilRoom, <-restSent, "the REST order should go when the HTTP IP limit's window has passed")
			})
		})
	}
}

func TestOrderEntryBudgets(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name                      string
		budgets                   orderEntryBudgets
		derivatives, option, spot orderBudget
	}{
		{"create", createOrderBudgets, orderBudget{createOrderEPL, wsCreateOrderEPL}, orderBudget{createOptionOrderEPL, wsCreateOptionOrderEPL}, orderBudget{createSpotOrderEPL, wsCreateSpotOrderEPL}},
		{"amend", amendOrderBudgets, orderBudget{amendOrderEPL, wsAmendOrderEPL}, orderBudget{amendOptionOrderEPL, wsAmendOptionOrderEPL}, orderBudget{amendSpotOrderEPL, wsAmendSpotOrderEPL}},
		{"cancel", cancelOrderBudgets, orderBudget{cancelOrderEPL, wsCancelOrderEPL}, orderBudget{cancelOptionOrderEPL, wsCancelOptionOrderEPL}, orderBudget{cancelSpotOrderEPL, wsCancelSpotOrderEPL}},
		{"cancel all", cancelAllOrdersBudgets, orderBudget{rest: cancelAllOrdersEPL}, orderBudget{rest: cancelAllOptionOrdersEPL}, orderBudget{rest: cancelAllSpotOrdersEPL}},
		{"batch create", createBatchOrderBudgets, orderBudget{createBatchOrderEPL, wsCreateBatchOrderEPL}, orderBudget{createOptionBatchOrderEPL, wsCreateOptionBatchOrderEPL}, orderBudget{createSpotBatchOrderEPL, wsCreateSpotBatchOrderEPL}},
		{"batch amend", amendBatchOrderBudgets, orderBudget{amendBatchOrderEPL, wsAmendBatchOrderEPL}, orderBudget{amendOptionBatchOrderEPL, wsAmendOptionBatchOrderEPL}, orderBudget{amendSpotBatchOrderEPL, wsAmendSpotBatchOrderEPL}},
		{"batch cancel", cancelBatchOrderBudgets, orderBudget{cancelBatchOrderEPL, wsCancelBatchOrderEPL}, orderBudget{cancelOptionBatchOrderEPL, wsCancelOptionBatchOrderEPL}, orderBudget{cancelSpotBatchOrderEPL, wsCancelSpotBatchOrderEPL}},
		{"pre-check", preCheckOrderBudgets, orderBudget{rest: preCheckOrderEPL}, orderBudget{rest: preCheckOptionOrderEPL}, orderBudget{rest: preCheckSpotOrderEPL}},
	} {
		for category, exp := range map[string]orderBudget{cLinear: tc.derivatives, cInverse: tc.derivatives, cOption: tc.option, cSpot: tc.spot, "spread": tc.derivatives} {
			assert.Equalf(t, exp, tc.budgets.forCategory(category), "%s orders in category %q should draw on their documented budget", tc.name, category)
		}
	}
	assert.Equal(t, map[string]orderEntryBudgets{
		"order.create":       createOrderBudgets,
		"order.amend":        amendOrderBudgets,
		"order.cancel":       cancelOrderBudgets,
		"order.create-batch": createBatchOrderBudgets,
		"order.amend-batch":  amendBatchOrderBudgets,
		"order.cancel-batch": cancelBatchOrderBudgets,
	}, websocketOrderBudgets, "each websocket order entry operation should draw on its REST operation's budgets")
}

func TestGetWSRateLimitEPLByCategory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		operation, category string
		exp                 request.EndpointLimit
		err                 error
	}{
		{"order.create", cLinear, wsCreateOrderEPL, nil},
		{"order.create", cInverse, wsCreateOrderEPL, nil},
		{"order.create", cOption, wsCreateOptionOrderEPL, nil},
		{"order.create", cSpot, wsCreateSpotOrderEPL, nil},
		{"order.amend", cLinear, wsAmendOrderEPL, nil},
		{"order.amend", cInverse, wsAmendOrderEPL, nil},
		{"order.amend", cOption, wsAmendOptionOrderEPL, nil},
		{"order.amend", cSpot, wsAmendSpotOrderEPL, nil},
		{"order.cancel", cLinear, wsCancelOrderEPL, nil},
		{"order.cancel", cInverse, wsCancelOrderEPL, nil},
		{"order.cancel", cOption, wsCancelOptionOrderEPL, nil},
		{"order.cancel", cSpot, wsCancelSpotOrderEPL, nil},
		{"order.create-batch", cLinear, wsCreateBatchOrderEPL, nil},
		{"order.create-batch", cInverse, wsCreateBatchOrderEPL, nil},
		{"order.create-batch", cOption, wsCreateOptionBatchOrderEPL, nil},
		{"order.create-batch", cSpot, wsCreateSpotBatchOrderEPL, nil},
		{"order.amend-batch", cLinear, wsAmendBatchOrderEPL, nil},
		{"order.amend-batch", cInverse, wsAmendBatchOrderEPL, nil},
		{"order.amend-batch", cOption, wsAmendOptionBatchOrderEPL, nil},
		{"order.amend-batch", cSpot, wsAmendSpotBatchOrderEPL, nil},
		{"order.cancel-batch", cLinear, wsCancelBatchOrderEPL, nil},
		{"order.cancel-batch", cInverse, wsCancelBatchOrderEPL, nil},
		{"order.cancel-batch", cOption, wsCancelOptionBatchOrderEPL, nil},
		{"order.cancel-batch", cSpot, wsCancelSpotBatchOrderEPL, nil},
		{"order.create", "", 0, errUnknownCategory},
		{"order.create", "spread", 0, errUnknownCategory},
		{"order.cancel-all", cLinear, 0, errUnknownOrderOperation},
		{"", cSpot, 0, errUnknownOrderOperation},
	} {
		epl, err := getWSRateLimitEPLByCategory(tc.operation, tc.category)
		if tc.err != nil {
			assert.ErrorIsf(t, err, tc.err, "getWSRateLimitEPLByCategory should refuse operation %q in category %q", tc.operation, tc.category)
			continue
		}
		require.NoErrorf(t, err, "getWSRateLimitEPLByCategory must not error for operation %q in category %q", tc.operation, tc.category)
		assert.Equalf(t, tc.exp, epl, "getWSRateLimitEPLByCategory should return the limit of operation %q in category %q", tc.operation, tc.category)
	}
}

// weightOf returns the weight a request sent with ctx takes from its endpoint's own budget: how much of an idle budget
// of the largest weight it leaves no room for
func weightOf(ctx context.Context) (int, error) {
	budget := request.GetRateLimiterWithWindowBudget(request.NewWindowBudget(time.Hour, math.MaxUint8), 1)
	noDelay := request.WithDelayNotAllowed(ctx)
	if err := budget.RateLimit(noDelay); err != nil {
		return 0, err
	}
	single := request.WithWeight(noDelay, 1)
	weight := math.MaxUint8
	for budget.RateLimit(single) == nil {
		weight--
	}
	return weight, nil
}

func TestBatchOrderLimit(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		budgets  orderEntryBudgets
		category string
		orders   int
		epl      request.EndpointLimit
		weight   int
	}{
		{createBatchOrderBudgets, cSpot, 2, createSpotBatchOrderEPL, 2},
		{createBatchOrderBudgets, cLinear, 10, createBatchOrderEPL, 10},
		{amendBatchOrderBudgets, cInverse, 3, amendBatchOrderEPL, 3},
		{amendBatchOrderBudgets, cOption, 5, amendOptionBatchOrderEPL, 1},
		{cancelBatchOrderBudgets, cSpot, 1, cancelSpotBatchOrderEPL, 1},
	} {
		ctx, epl := batchOrderLimit(t.Context(), tc.budgets, tc.category, tc.orders)
		assert.Equalf(t, tc.epl, epl, "a %s batch should be sent with its category's batch limit", tc.category)
		weight, err := weightOf(ctx)
		require.NoErrorf(t, err, "a %s batch of %d orders must fit an idle budget", tc.category, tc.orders)
		assert.Equalf(t, tc.weight, weight, "a %s batch of %d orders should be booked as Bybit counts it", tc.category, tc.orders)
	}
}

func TestWithBatchOrderWeight(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		category string
		orders   int
		exp      int
	}{
		{cLinear, 1, 1},
		{cLinear, 10, 10},
		{cInverse, 7, 7},
		{cSpot, 10, 10},
		{cOption, 5, 1},
		{cOption, 11, 1},
		{cLinear, math.MaxUint8 - 1, math.MaxUint8 - 1},
		{cLinear, math.MaxUint8, math.MaxUint8},
		{cLinear, math.MaxUint8 + 1, math.MaxUint8},
		{cLinear, -1, math.MaxUint8},
	} {
		weight, err := weightOf(withBatchOrderWeight(t.Context(), tc.category, tc.orders))
		require.NoErrorf(t, err, "a %s batch of %d orders must fit an idle budget of the largest weight", tc.category, tc.orders)
		assert.Equalf(t, tc.exp, weight, "a %s batch of %d orders should weigh %d", tc.category, tc.orders, tc.exp)
	}
	_, err := weightOf(withBatchOrderWeight(t.Context(), cLinear, 0))
	assert.ErrorContains(t, err, "equal-or-greater than 1", "a batch of no orders should be refused for its weight")
	// A batch weighs what Bybit counts it as, whatever weight the caller's context carried
	weight, err := weightOf(withBatchOrderWeight(request.WithWeight(t.Context(), 11), cOption, 2))
	require.NoError(t, err, "an options batch must fit its budget whatever weight its context carried")
	assert.Equal(t, 1, weight, "an options batch should weigh one whatever weight its context carried")
}

// batchSend is when a batch order request reached the transport and how many orders it carried
type batchSend struct {
	at     time.Duration
	orders int
}

// batchRequestBody is the part of a batch order request's body a test transport reads
type batchRequestBody struct {
	Request []json.RawMessage `json:"request"`
}

// batchTransport answers every batch order request with an outcome for each of its orders, after recording when the
// request was sent and how many orders it carried
type batchTransport struct {
	start time.Time
	mu    sync.Mutex
	sends []batchSend
}

// RoundTrip records a batch order request and answers it
func (b *batchTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	var batch batchRequestBody
	if err := json.Unmarshal(body, &batch); err != nil {
		return nil, err
	}
	b.mu.Lock()
	b.sends = append(b.sends, batchSend{time.Since(b.start), len(batch.Request)})
	b.mu.Unlock()
	orders := strings.TrimSuffix(strings.Repeat(`{"orderId":"1"},`, len(batch.Request)), ",")
	outcomes := strings.TrimSuffix(strings.Repeat(`{"code":0,"msg":"OK"},`, len(batch.Request)), ",")
	reply := `{"retCode":0,"retMsg":"OK","result":{"list":[` + orders + `]},"retExtInfo":{"list":[` + outcomes + `]},"time":1760000000000}`
	return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(reply)), Request: r}, nil
}

// recorded returns the batch order requests sent so far, in the order they were sent
func (b *batchTransport) recorded() []batchSend {
	b.mu.Lock()
	defer b.mu.Unlock()
	return slices.SortedFunc(slices.Values(b.sends), func(x, y batchSend) int { return cmp.Compare(x.at, y.at) })
}

// batchTransportExchange returns an exchange with Bybit's rate limits whose REST requests reach transport
func batchTransportExchange(t *testing.T) (*Exchange, *batchTransport) {
	t.Helper()
	ex := withMockCredentials(websocketTestInstance(t))
	ex.Name = t.Name()
	transport := &batchTransport{start: time.Now()}
	var err error
	ex.Requester, err = request.New(t.Name(), &http.Client{Transport: transport}, request.WithLimiter(rateLimitDefinitions()))
	require.NoError(t, err, "request.New must not error")
	return ex, transport
}

// linearCancellations returns a batch cancellation of n linear orders
func linearCancellations(n int) *BatchCancelOrderRequest {
	r := &BatchCancelOrderRequest{Category: cLinear, Request: make([]BatchCancelOrderItem, n)}
	for i := range r.Request {
		r.Request[i] = BatchCancelOrderItem{Symbol: "BTCUSDT", OrderID: strconv.Itoa(i + 1)}
	}
	return r
}

func TestBatchOrderSendTimes(t *testing.T) {
	// orderWindow is an order entry budget's window, Bybit's rolling second and the margin kept in hand
	const orderWindow = time.Second + orderBudgetMargin
	t.Parallel()
	for _, tc := range []struct {
		name       string
		sizes      []int
		concurrent bool
		exp        []batchSend
	}{
		{"a batch after one of the whole budget waits a window", []int{10, 1}, false, []batchSend{{0, 10}, {orderWindow, 1}}},
		{"concurrent batches of the whole budget go a window apart", []int{10, 10}, true, []batchSend{{0, 10}, {orderWindow, 10}}},
		{"a batch fitting the room the second has left goes at once", []int{1, 1, 1, 1, 1, 1, 1, 3}, false, []batchSend{{0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 3}}},
		{"a batch larger than the room the second has left waits for room for all of it", []int{1, 1, 1, 1, 1, 1, 1, 1, 3}, false, []batchSend{{0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 1}, {0, 1}, {orderWindow, 3}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
				ex, transport := batchTransportExchange(t)
				var errs common.ErrorCollector
				for _, orders := range tc.sizes {
					if tc.concurrent {
						errs.Go(func() error {
							_, err := ex.CancelBatchOrder(t.Context(), linearCancellations(orders))
							return err
						})
						continue
					}
					_, err := ex.CancelBatchOrder(t.Context(), linearCancellations(orders))
					require.NoError(t, err, "CancelBatchOrder must not error")
				}
				require.NoError(t, errs.Collect(), "CancelBatchOrder must not error")
				assert.Equal(t, tc.exp, transport.recorded(), "each batch should be sent once no second holding it holds more than the budget's 10 orders")
			})
		})
	}

	t.Run("a batch larger than its budget is refused before it is sent", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			ex, transport := batchTransportExchange(t)
			_, err := ex.CancelBatchOrder(t.Context(), linearCancellations(11))
			require.ErrorIs(t, err, request.ErrWeightExceedsBudget, "CancelBatchOrder must refuse a batch its budget never admits")
			assert.Empty(t, transport.recorded(), "a refused batch should not be sent")
			_, err = ex.CancelBatchOrder(request.WithDelayNotAllowed(t.Context()), linearCancellations(10))
			assert.NoError(t, err, "a refused batch should book nothing")
		})
	})

	t.Run("cancelling more orders than the budget admits sends batches it admits", func(t *testing.T) {
		t.Parallel()
		synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
			ex, transport := batchTransportExchange(t)
			cancels := make([]order.Cancel, 25)
			for i := range cancels {
				cancels[i] = order.Cancel{OrderID: strconv.Itoa(i + 1), Pair: currency.NewBTCUSDT(), AssetType: asset.USDTMarginedFutures}
			}
			resp, err := ex.CancelBatchOrders(t.Context(), cancels)
			require.NoError(t, err, "CancelBatchOrders must not error")
			assert.Len(t, resp.Status, 25, "CancelBatchOrders should report every order's outcome")
			assert.Equal(t, []batchSend{{0, 10}, {orderWindow, 10}, {2 * orderWindow, 5}}, transport.recorded(), "CancelBatchOrders should send batches of the 10 orders a window the budget admits")
		})
	})
}

func TestBatchOrderBudgetsUnderLoad(t *testing.T) {
	t.Parallel()
	synctest.Test(t, func(t *testing.T) { //nolint:thelper,nolintlint // false positive
		definitions := rateLimitDefinitions()
		random := rand.New(rand.NewPCG(5, 6)) //nolint:gosec // A fixed seed keeps the load reproducible
		start := time.Now()
		var mu sync.Mutex
		var orders, httpRequests []time.Duration
		var errs common.ErrorCollector
		for range 150 {
			arrival, size, epl := time.Duration(random.IntN(5000))*time.Millisecond, 1+random.IntN(10), createBatchOrderEPL
			if random.IntN(2) == 0 {
				epl = wsCreateBatchOrderEPL
			}
			errs.Go(func() error {
				time.Sleep(arrival)
				if err := definitions[epl].RateLimit(withBatchOrderWeight(t.Context(), cLinear, size)); err != nil {
					return err
				}
				sent := time.Since(start)
				mu.Lock()
				defer mu.Unlock()
				for range size {
					orders = append(orders, sent)
				}
				if epl == createBatchOrderEPL {
					httpRequests = append(httpRequests, sent)
				}
				return nil
			})
		}
		require.NoError(t, errs.Collect(), "every batch must be sent")
		slices.Sort(orders)
		var busiest int
		for i, first := range orders {
			end, _ := slices.BinarySearch(orders[i:], first+time.Second)
			busiest = max(busiest, end)
		}
		assert.Equal(t, endpointBudgets[createBatchOrderEPL].requests, busiest, "the busiest second should fill the batch budget's orders without exceeding it, over REST and websocket together")
		_, hundredMilliseconds := ipLimitedSends(httpRequests)
		assert.LessOrEqual(t, hundredMilliseconds, ipBudgets[1].requests, "the HTTP IP limit should count each REST batch once")
	})
}

func TestRateLimitsWithRequester(t *testing.T) {
	t.Parallel()
	r, err := request.New(t.Name(), &http.Client{}, request.WithLimiter(rateLimitDefinitions()))
	require.NoError(t, err, "request.New must not error")
	for _, epl := range []request.EndpointLimit{request.Unset, request.Auth, request.UnAuth} {
		assert.ErrorIsf(t, r.InitiateRateLimit(t.Context(), epl), common.ErrNilPointer, "generic endpoint limit %d should find no Bybit limiter", epl)
	}
	assert.NoError(t, r.InitiateRateLimit(t.Context(), getOpenOrdersEPL), "a request should not wait on fresh budgets")
}

// senderLimitArguments holds the argument index of the endpoint limit of every sender, split by the requests it sends
var senderLimitArguments = map[bool]map[string]int{
	false: {"SendHTTPRequest": 3, "SendAuthHTTPRequest": 7, "sendRequest": 1, "sendBatchOrderRequest": 3},
	true: {
		"SendMessageReturnResponse": 1, "SendMessageReturnResponses": 1, "SendMessageReturnResponsesWithInspector": 1,
		"SendRawMessage": 1, "SendJSONMessage": 1, "SetupPingHandler": 0,
		"sendWebsocketTradeRequest": 5, "sendWebsocketOrderRequest": 4, "sendWebsocketBatchOrderRequest": 3,
	},
}

func TestEndpointLimitArguments(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err, "the package's files must be listed")
	fset := token.NewFileSet()
	for _, file := range files {
		// ratelimit.go numbers the endpoint limits from request.UnAuth and sends nothing
		if strings.HasSuffix(file, "_test.go") || file == "ratelimit.go" {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoErrorf(t, err, "%s must parse", file)
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.SelectorExpr:
				if pkg, ok := n.X.(*ast.Ident); ok && pkg.Name == "request" {
					switch n.Sel.Name {
					case "Unset", "Auth", "UnAuth":
						assert.Failf(t, "generic endpoint limit", "%s should use a Bybit endpoint limit instead of request.%s", fset.Position(n.Pos()), n.Sel.Name)
					}
				}
			case *ast.CallExpr:
				var name string
				switch fun := n.Fun.(type) {
				case *ast.SelectorExpr:
					name = fun.Sel.Name
				case *ast.IndexExpr:
					if ident, ok := fun.X.(*ast.Ident); ok {
						name = ident.Name
					}
				}
				for websocket, senders := range senderLimitArguments {
					i, ok := senders[name]
					if !ok || i >= len(n.Args) {
						continue
					}
					arg, ok := n.Args[i].(*ast.Ident)
					if !ok || !strings.HasSuffix(arg.Name, "EPL") {
						continue
					}
					assert.Equalf(t, websocket, strings.HasPrefix(arg.Name, "ws"), "%s should pass %s a limit for the kind of request it sends, not %s", fset.Position(n.Pos()), name, arg.Name)
				}
			}
			return true
		})
	}
}

func TestEndpointLimitsUsed(t *testing.T) {
	t.Parallel()
	files, err := filepath.Glob("*.go")
	require.NoError(t, err, "the package's files must be listed")
	used := make(map[string]bool)
	fset := token.NewFileSet()
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, 0)
		require.NoErrorf(t, err, "%s must parse", file)
		for _, decl := range f.Decls {
			if gen, ok := decl.(*ast.GenDecl); ok && file == "ratelimit.go" && (gen.Tok == token.CONST || declares(gen, "endpointBudgets")) {
				// Declaring an endpoint limit and its budget is not a use
				continue
			}
			ast.Inspect(decl, func(n ast.Node) bool {
				if ident, ok := n.(*ast.Ident); ok {
					used[ident.Name] = true
				}
				return true
			})
		}
	}
	for _, name := range endpointLimitNames(t) {
		assert.Truef(t, used[name], "%s should be used by an endpoint", name)
	}
}

// declares reports whether gen declares name
func declares(gen *ast.GenDecl, name string) bool {
	for _, spec := range gen.Specs {
		if valueSpec, ok := spec.(*ast.ValueSpec); ok {
			for _, ident := range valueSpec.Names {
				if ident.Name == name {
					return true
				}
			}
		}
	}
	return false
}
