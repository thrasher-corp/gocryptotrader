package bank

import (
	"errors"
	"testing"
)

func TestString(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		Value  Transfer
		Return string
	}{
		{NotApplicable, "NotApplicable"},
		{WireTransfer, "WireTransfer"},
		{PerfectMoney, "PerfectMoney"},
		{Neteller, "Neteller"},
		{AdvCash, "AdvCash"},
		{Payeer, "Payeer"},
		{Skrill, "Skrill"},
		{Simplex, "Simplex"},
		{SEPA, "SEPA"},
		{Swift, "Swift"},
		{RapidTransfer, "RapidTransfer"},
		{MisterTangoSEPA, "MisterTangoSEPA"},
		{Qiwi, "Qiwi"},
		{VisaMastercard, "VisaMastercard"},
		{WebMoney, "WebMoney"},
		{Capitalist, "Capitalist"},
		{WesternUnion, "WesternUnion"},
		{MoneyGram, "MoneyGram"},
		{Contact, "Contact"},
		{ExpressWireTransfer, "ExpressWireTransfer"},
		{PayIDOsko, "PayID/Osko"},
		{BankCardVisa, "BankCard Visa"},
		{BankCardMastercard, "BankCard Mastercard"},
		{BankCardMIR, "BankCard MIR"},
		{CreditCardMastercard, "CreditCard Mastercard"},
		{Sofort, "Sofort"},
		{P2P, "P2P"},
		{Etana, "Etana"},
		{FasterPaymentService, "FasterPaymentService(FPS)"},
		{MobileMoney, "MobileMoney"},
		{CashTransfer, "CashTransfer"},
		{YandexMoney, "YandexMoney"},
		{GEOPay, "GEOPay"},
		{SettlePay, "SettlePay"},
		{ExchangeFiatDWChannelSignetUSD, "ExchangeFiatDWChannelSignetUSD"},
		{ExchangeFiatDWChannelSwiftSignatureBar, "ExchangeFiatDWChannelSignetUSD"},
		{AutomaticClearingHouse, "AutomaticClearingHouse"},
		{FedWire, "FedWire"},
		{TelegraphicTransfer, "TelegraphicTransfer"},
		{SDDomesticCheque, "SDDomesticCheque"},
		{Xfers, "Xfers"},
		{ExmoGiftCard, "ExmoGiftCard"},
		{Terminal, "Terminal"},
		{255, ""},
	}

	for _, tt := range testCases {
		tt := tt
		t.Run("", func(t *testing.T) {
			t.Parallel()
			if tt.Value.String() != tt.Return {
				t.Fatalf("expected: %s but received: %s", tt.Value, tt.Return)
			}
		})
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	err := Transfer(255).Validate()
	if !errors.Is(err, ErrUnknownTransfer) {
		t.Fatalf("received: %v but expected: %v", err, ErrUnknownTransfer)
	}
	err = NotApplicable.Validate()
	if !errors.Is(err, nil) {
		t.Fatalf("received: %v but expected: %v", err, nil)
	}
	err = Transfer(0).Validate()
	if !errors.Is(err, ErrTransferTypeUnset) {
		t.Fatalf("received: %v but expected: %v", err, ErrTransferTypeUnset)
	}
}
