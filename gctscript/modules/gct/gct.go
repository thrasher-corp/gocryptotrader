package gct

import (
	"context"

	objects "github.com/d5/tengo/v2"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/exchanges/account"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
)

// AllModuleNames returns a list of all default module names.
func AllModuleNames() []string {
	names := make([]string, 0, len(Modules))
	for name := range Modules {
		names = append(names, name)
	}
	return names
}

var globalModules = map[string]objects.Object{
	"set_verbose": &objects.UserFunction{Name: "set_verbose", Value: setVerbose},
	"set_account": &objects.UserFunction{Name: "set_account", Value: setAccount},
}

// setVerbose specifically sets verbosity for http rest requests for this script
// Params: scriptCTX
func setVerbose(args ...objects.Object) (objects.Object, error) {
	if len(args) != 1 {
		return nil, objects.ErrWrongNumArguments
	}

	ctx, ok := objects.ToInterface(args[0]).(*Context)
	if !ok {
		return nil, common.GetAssertError("*gct.Convert for arg 1", args[0])
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
		return nil, common.GetAssertError("*gct.Context for arg 1", args[0])
	}

	apikey, ok := objects.ToInterface(args[1]).(string)
	if !ok {
		return nil, common.GetAssertError("string for arg 2", args[1])
	}

	if ctx.Value == nil {
		ctx.Value = make(map[string]objects.Object)
	}

	ctx.Value["apikey"] = &objects.String{Value: apikey}

	apisecret, ok := objects.ToInterface(args[2]).(string)
	if !ok {
		return nil, common.GetAssertError("string for arg 3", args[2])
	}

	ctx.Value["apisecret"] = &objects.String{Value: apisecret}

	if len(args) > 3 {
		var subaccount string
		subaccount, ok = objects.ToInterface(args[3]).(string)
		if !ok {
			return nil, common.GetAssertError("string for arg 4", args[3])
		}

		if subaccount != "" {
			ctx.Value["subaccount"] = &objects.String{Value: subaccount}
		}
	}

	if len(args) > 4 {
		var clientID string
		clientID, ok = objects.ToInterface(args[4]).(string)
		if !ok {
			return nil, common.GetAssertError("string for arg 5", args[4])
		}
		if clientID != "" {
			ctx.Value["clientid"] = &objects.String{Value: clientID}
		}
	}

	if len(args) > 5 {
		var pemKey string
		pemKey, ok = objects.ToInterface(args[5]).(string)
		if !ok {
			return nil, common.GetAssertError("string for arg 6", args[5])
		}
		if pemKey != "" {
			ctx.Value["pemkey"] = &objects.String{Value: pemKey}
		}
	}

	if len(args) > 5 {
		var oneTimePassword string
		oneTimePassword, ok = objects.ToInterface(args[6]).(string)
		if !ok {
			return nil, common.GetAssertError("string for arg 7", args[6])
		}
		if oneTimePassword != "" {
			ctx.Value["otp"] = &objects.String{Value: oneTimePassword}
		}
	}

	return ctx, nil
}

func processScriptContext(scriptCtx *Context) context.Context {
	ctx := context.Background()
	if scriptCtx == nil {
		return ctx
	}
	if _, ok := scriptCtx.Value["verbose"]; ok {
		ctx = request.WithVerbose(ctx)
	}
	if apikeyobj, ok := scriptCtx.Value["apikey"]; ok {
		key, _ := objects.ToString(apikeyobj)

		var secret string
		if secretObj, ok := scriptCtx.Value["apisecret"]; ok {
			secret, _ = objects.ToString(secretObj)
		}

		var subaccount string
		if subaccountObj, ok := scriptCtx.Value["subaccount"]; ok {
			subaccount, _ = objects.ToString(subaccountObj)
		}

		var clientID string
		if clientidObj, ok := scriptCtx.Value["clientid"]; ok {
			clientID, _ = objects.ToString(clientidObj)
		}

		var pemKey string
		if pemkeyObj, ok := scriptCtx.Value["pemkey"]; ok {
			pemKey, _ = objects.ToString(pemkeyObj)
		}

		var otp string
		if otpObj, ok := scriptCtx.Value["otp"]; ok {
			otp, _ = objects.ToString(otpObj)
		}

		creds := &account.Credentials{
			Key:             key,
			Secret:          secret,
			SubAccount:      subaccount,
			ClientID:        clientID,
			PEMKey:          pemKey,
			OneTimePassword: otp,
		}
		ctx = account.DeployCredentialsToContext(ctx, creds)
	}
	return ctx
}

// TypeName returns the name of the custom type.
func (c *Context) TypeName() string {
	return "scriptContext"
}
