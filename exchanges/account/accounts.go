package account

import (
	"errors"
	"fmt"
	"strings"
	"sync"
)

// Designation identifies a sub account or exchange balance segregation
// associated with the supplied api-key set.
type Designation string

// Main defines a default string for the main account, used if there is no
// need for differentiation.
const Main Designation = "main"

// Accounts defines multiple labelled segrations on balances
type Accounts struct {
	available []account
	m         sync.RWMutex
}

// account is a sub type to differentiate between the main account and sub
// accounts
type account struct {
	Name Designation
	Main bool
}

var (
	// ErrAccountNameUnset defines an error for when an account name is unset
	ErrAccountNameUnset         = errors.New("account name unset")
	errMainAccountNameUnset     = errors.New("main account name unset")
	errAccountsNotLoaded        = errors.New("accounts not loaded")
	errAccountAlreadyLoaded     = errors.New("account already loaded")
	errMainAccountAlreadyLoaded = errors.New("main account already loaded")
	errMainAccountNotLoaded     = errors.New("main account not loaded")
)

// LoadAccount loads an account for future checking
func (a *Accounts) LoadAccount(accountName string, main bool) error {
	if accountName == "" {
		return ErrAccountNameUnset
	}

	accD := Designation(strings.ToLower(accountName))

	a.m.Lock()
	defer a.m.Unlock()
	for x := range a.available {
		if a.available[x].Name == accD {
			return errAccountAlreadyLoaded
		}

		if main && a.available[x].Main {
			return errMainAccountAlreadyLoaded
		}
	}
	a.available = append(a.available, account{Name: accD, Main: main})
	return nil
}

// LoadAccount loads a main account and subsequent sub accounts
func (a *Accounts) LoadAccounts(main string, subAccount ...string) error {
	if main == "" {
		return errMainAccountNameUnset
	}

	a.m.Lock()
	a.available = nil // Purge prior loaded accounts
	a.m.Unlock()

	err := a.LoadAccount(main, true)
	if err != nil {
		return err
	}

	for x := range subAccount {
		err = a.LoadAccount(subAccount[x], false)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAccounts returns the loaded accounts associated with the current global
// API credentials
func (a *Accounts) GetAccounts() ([]Designation, error) {
	a.m.RLock()
	defer a.m.RUnlock()
	amount := len(a.available)
	if amount == 0 {
		return nil, errAccountsNotLoaded
	}
	accounts := make([]Designation, amount)
	for x := range a.available {
		accounts[x] = a.available[x].Name
	}
	return accounts, nil
}

// AccountValid cross references account with available accounts list. Used by
// external calls GRPC and/or strategies to ensure availability before locking
// core systems.
func (a *Accounts) AccountValid(account string) error {
	if account == "" {
		return ErrAccountNameUnset
	}

	account = strings.ToLower(account)

	a.m.RLock()
	defer a.m.RUnlock()
	if len(a.available) == 0 {
		return errAccountsNotLoaded
	}

	for x := range a.available {
		if string(a.available[x].Name) == account {
			return nil
		}
	}
	return fmt.Errorf("%s %w: available accounts [%+v]",
		account,
		errAccountNotFound,
		a.available)
}

// GetMainAccount returns the main account for the exchange holdings
func (a *Accounts) GetMainAccount() (Designation, error) {
	a.m.RLock()
	defer a.m.RUnlock()
	if len(a.available) == 0 {
		return "", errAccountsNotLoaded
	}
	for x := range a.available {
		if a.available[x].Main {
			return a.available[x].Name, nil
		}
	}
	return "", errMainAccountNotLoaded
}
