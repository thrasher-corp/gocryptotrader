package gct

import (
	"context"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

const (
	setVerboseFunc    = "set_verbose"
	setAccountFunc    = "set_account"
	setSubAccountFunc = "set_sub_account"
)

var globalModules = map[string]objects.Object{
	setVerboseFunc:    &objects.UserFunction{Name: setVerboseFunc, Value: setVerbose},
	setAccountFunc:    &objects.UserFunction{Name: setAccountFunc, Value: setAccount},
	setSubAccountFunc: &objects.UserFunction{Name: setSubAccountFunc, Value: setSubAccount},
}

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	names := make([]string, 0, len(Modules))
	for name := range Modules {
		names = append(names, name)
	}
	return names
}

// setVerbose specifically sets verbosity for http rest requests for this script
// Params: scriptCTX
func setVerbose(args ...objects.Object) (objects.Object, error) {
	if len(args) != 1 {
		return nil, objects.ErrWrongNumArguments
	}

	ctx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, setVerboseFunc, "*gct.Context", args[0])
	}
	if ctx.Value == nil {
		ctx.Value = make(map[string]objects.Object)
	}
	ctx.Value["verbose"] = objects.TrueValue
	return ctx, nil
}

// setAccount sets account details which overrides default credentials for
// script account management, api key and secret are required.
// Params: scriptCTX, apiKey, apiSecret, subAccount, clientID, PEMKey, OneTimePassword string
func setAccount(args ...objects.Object) (objects.Object, error) {
	if len(args) < 3 || len(args) > 7 {
		return nil, objects.ErrWrongNumArguments
	}

	ctx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, setAccountFunc, "*gct.Context", args[0])
	}

	apikey, ok := objects.ToInterface(args[1]).(string)
	if !ok {
		return nil, constructRuntimeError(2, setAccountFunc, "string", args[1])
	}

	if ctx.Value == nil {
		ctx.Value = make(map[string]objects.Object)
	}

	ctx.Value["apikey"] = &objects.String{Value: apikey}

	apisecret, ok := objects.ToInterface(args[2]).(string)
	if !ok {
		return nil, constructRuntimeError(3, setAccountFunc, "string", args[2])
	}

	ctx.Value["apisecret"] = &objects.String{Value: apisecret}

	if len(args) > 3 {
		var subaccount string
		subaccount, ok = objects.ToInterface(args[3]).(string)
		if !ok {
			return nil, constructRuntimeError(4, setAccountFunc, "string", args[3])
		}

		if subaccount != "" {
			ctx.Value["subaccount"] = &objects.String{Value: subaccount}
		}
	}

	if len(args) > 4 {
		var clientID string
		clientID, ok = objects.ToInterface(args[4]).(string)
		if !ok {
			return nil, constructRuntimeError(5, setAccountFunc, "string", args[4])
		}
		if clientID != "" {
			ctx.Value["clientid"] = &objects.String{Value: clientID}
		}
	}

	if len(args) > 5 {
		var pemKey string
		pemKey, ok = objects.ToInterface(args[5]).(string)
		if !ok {
			return nil, constructRuntimeError(6, setAccountFunc, "string", args[5])
		}
		if pemKey != "" {
			ctx.Value["pemkey"] = &objects.String{Value: pemKey}
		}
	}

	if len(args) > 5 {
		var oneTimePassword string
		oneTimePassword, ok = objects.ToInterface(args[6]).(string)
		if !ok {
			return nil, constructRuntimeError(7, setAccountFunc, "string", args[6])
		}
		if oneTimePassword != "" {
			ctx.Value["otp"] = &objects.String{Value: oneTimePassword}
		}
	}
	return ctx, nil
}

// setSubAccount sets sub account details which overrides default credential
// sub account but uses the same configured api keys.
// Params: scriptCTX, subAccount string
func setSubAccount(args ...objects.Object) (objects.Object, error) {
	if len(args) != 2 {
		return nil, objects.ErrWrongNumArguments
	}

	ctx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, constructRuntimeError(1, setSubAccountFunc, "*gct.Context", args[0])
	}

	sub, ok := objects.ToInterface(args[1]).(string)
	if !ok {
		return nil, constructRuntimeError(2, setSubAccountFunc, "string", args[1])
	}

	if ctx.Value == nil {
		ctx.Value = make(map[string]objects.Object)
	}

	ctx.Value["subaccount"] = &objects.String{Value: sub}
	return ctx, nil
}

func processScriptContext(scriptCtx *Context) context.Context {
	ctx := context.Background()
	if scriptCtx == nil || scriptCtx.Value == nil {
		return ctx
	}
	var object objects.Object
	if object = scriptCtx.Value["verbose"]; object != nil {
		ctx = request.WithVerbose(ctx)
	}

	if object = scriptCtx.Value["apikey"]; object != nil {
		key, _ := objects.ToString(object)

		var secret string
		if object = scriptCtx.Value["apisecret"]; object != nil {
			secret, _ = objects.ToString(object)
		}

		var subAccount string
		if object = scriptCtx.Value["subaccount"]; object != nil {
			subAccount, _ = objects.ToString(object)
		}

		var clientID string
		if object = scriptCtx.Value["clientid"]; object != nil {
			clientID, _ = objects.ToString(object)
		}

		var pemKey string
		if object = scriptCtx.Value["pemkey"]; object != nil {
			pemKey, _ = objects.ToString(object)
		}

		var otp string
		if object = scriptCtx.Value["otp"]; object != nil {
			otp, _ = objects.ToString(object)
		}

		ctx = accounts.DeployCredentialsToContext(ctx, &accounts.Credentials{
			Key:             key,
			Secret:          secret,
			SubAccount:      subAccount,
			ClientID:        clientID,
			PEMKey:          pemKey,
			OneTimePassword: otp,
		})
	} else if object = scriptCtx.Value["subaccount"]; object != nil {
		subAccount, _ := objects.ToString(object)
		ctx = accounts.DeploySubAccountOverrideToContext(ctx, subAccount)
	}
	return ctx
}

// TypeName returns the name of the custom type.
func (c *Context) TypeName() string {
	return "scriptContext"
}
