package account

import (
	"errors"
	"testing"
)

func TestLoadAccount(t *testing.T) {
	a := Accounts{}
	err := a.LoadAccount("", false)
	if !errors.Is(err, ErrAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", ErrAccountNameUnset, err)
	}

	err = a.LoadAccount("testAccount", true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = a.LoadAccount("testAccOunt", false)
	if !errors.Is(err, errAccountAlreadyLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountAlreadyLoaded, err)
	}

	err = a.LoadAccount("testAccOunt2", true)
	if !errors.Is(err, errMainAccountAlreadyLoaded) {
		t.Fatalf("expected: %v but received: %v", errMainAccountAlreadyLoaded, err)
	}

	if len(a.available) != 1 {
		t.Fatal("unexpected account count")
	}
}

func TestLoadAccounts(t *testing.T) {
	a := Accounts{}
	err := a.LoadAccounts("")
	if !errors.Is(err, errMainAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", errMainAccountNameUnset, err)
	}

	err = a.LoadAccounts("exchange")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	// This call flushes the entire list of accounts
	err = a.LoadAccounts("exchange")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = a.LoadAccounts("exchange", "exchange")
	if !errors.Is(err, errAccountAlreadyLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountAlreadyLoaded, err)
	}
}

func TestGetAccounts(t *testing.T) {
	a := Accounts{}
	_, err := a.GetAccounts()
	if !errors.Is(err, errAccountsNotLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountsNotLoaded, err)
	}

	err = a.LoadAccount("testAccount", true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	accs, err := a.GetAccounts()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if len(accs) != 1 {
		t.Fatal("unexpected amount received")
	}

	if accs[0] != "testaccount" {
		t.Fatalf("unexpected value %s received", accs[0])
	}
}

func TestAccountValid(t *testing.T) {
	a := Accounts{}
	err := a.AccountValid("")
	if !errors.Is(err, ErrAccountNameUnset) {
		t.Fatalf("expected: %v but received: %v", ErrAccountNameUnset, err)
	}

	err = a.AccountValid("test")
	if !errors.Is(err, errAccountsNotLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountsNotLoaded, err)
	}

	err = a.LoadAccount("testAccount", true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = a.AccountValid("tEsTAccOuNt")
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	err = a.AccountValid("test1")
	if !errors.Is(err, errAccountNotFound) {
		t.Fatalf("expected: %v but received: %v", errAccountNotFound, err)
	}
}

func TestGetMainAccount(t *testing.T) {
	a := Accounts{}
	_, err := a.GetMainAccount()
	if !errors.Is(err, errAccountsNotLoaded) {
		t.Fatalf("expected: %v but received: %v", errAccountsNotLoaded, err)
	}

	err = a.LoadAccount("testAccount", false)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	_, err = a.GetMainAccount()
	if !errors.Is(err, errMainAccountNotLoaded) {
		t.Fatalf("expected: %v but received: %v", errMainAccountNotLoaded, err)
	}

	err = a.LoadAccount("main", true)
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	m, err := a.GetMainAccount()
	if !errors.Is(err, nil) {
		t.Fatalf("expected: %v but received: %v", nil, err)
	}

	if m != "main" {
		t.Fatal("unexpected main account")
	}
}
