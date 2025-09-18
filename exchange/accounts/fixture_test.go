package accounts

import (
	"context"

	"github.com/thrasher-corp/gocryptotrader/common"
)

type mockEx struct {
	name string
}

func (m *mockEx) GetName() string {
	return "mocky"
}

func (m *mockEx) GetCredentials(ctx context.Context) (*Credentials, error) {
	if value := ctx.Value(ContextCredentialsFlag); value != nil {
		if s, ok := value.(*ContextCredentialsStore); ok {
			return s.Get(), nil
		}
		return nil, common.GetTypeAssertError("*accounts.ContextCredentialsStore", value)
	}
	return nil, nil
}

type mockExBase struct {
	base exchange
}

func (m *mockExBase) GetBase() exchange {
	return m.base
}

func (m *mockExBase) GetCredentials(ctx context.Context) (*Credentials, error) {
	return m.base.GetCredentials(ctx)
}

func (m *mockExBase) GetName() string {
	return m.base.GetName()
}
