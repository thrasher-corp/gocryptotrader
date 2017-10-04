package smsglobal

import (
	"log"
	"testing"
)

func TestNew(t *testing.T) {
	result := New("", "", "", nil)
	if result != nil {
		t.Error("Test failed. New: Expected nil result")
	}

	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)

	result = New("bob", "pw", "Skynet", contacts)
	if !result.ContactExists(contact) {
		t.Error("Test failed. New: Expected contact not found")
	}
}

func TestGetEnabledContacts(t *testing.T) {
	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)
	result := New("bob", "pw", "Skynet", contacts)

	expected := 1
	actual := result.GetEnabledContacts()
	if expected != actual {
		t.Errorf("Test failed. TestGetEnabledContacts expected %d, got %d",
			expected, actual)
	}
}

func TestGetContactByNumber(t *testing.T) {
	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)
	result := New("bob", "pw", "Skynet", contacts)

	actual, err := result.GetContactByNumber(contact.Number)
	if err != nil {
		t.Fatalf("Test failed. TestGetContactByNumber: %s", err)
	}

	if actual.Name != contact.Name && actual.Number != contact.Number && actual.Enabled != contact.Enabled {
		t.Fatal("Test failed. TestGetContactByNumber: Incorrect values")
	}

	_, err = result.GetContactByNumber("ASDASDASD")
	if err == nil {
		t.Fatal("Test failed. TestGetContactByNumber: Returned nil err on non-existent number")
	}
}

func TestGetContactByName(t *testing.T) {
	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)
	result := New("bob", "pw", "Skynet", contacts)

	actual, err := result.GetContactByName(contact.Name)
	if err != nil {
		t.Fatalf("Test failed. TestGetContactByName: %s", err)
	}

	if actual.Name != contact.Name && actual.Number != contact.Number && actual.Enabled != contact.Enabled {
		t.Fatal("Test failed. TestGetContactByName: Incorrect values")
	}

	_, err = result.GetContactByName("ASDASDASD")
	if err == nil {
		t.Fatal("Test failed. TestGetContactByName: Returned nil err on non-existent number")
	}
}

func TestAddContact(t *testing.T) {
	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)
	result := New("bob", "pw", "Skynet", contacts)

	// Test adding same contact
	result.AddContact(contact)
	if result.GetEnabledContacts() > 1 {
		t.Fatal("Test failed. TestAddContact: Incorrect values")
	}

	invalidContact := Contact{Name: "", Number: "", Enabled: true}
	result.AddContact(invalidContact)
	if result.GetEnabledContacts() > 1 {
		t.Fatal("Test failed. TestAddContact: Incorrect values")
	}

	newContact := Contact{Name: "newContact", Number: "12345", Enabled: true}
	result.AddContact(newContact)
	if result.GetEnabledContacts() != 2 {
		t.Fatal("Test failed. TestAddContact: Incorrect values")
	}
}

func TestRemoveContact(t *testing.T) {
	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)
	result := New("bob", "pw", "Skynet", contacts)

	result.RemoveContact(Contact{Name: "blah", Number: "1234"})
	if result.GetEnabledContacts() != 1 {
		t.Fatal("Test failed. TestRemoveContact: Incorrect values")
	}

	result.RemoveContact(contact)
	if result.GetEnabledContacts() != 0 {
		t.Fatal("Test failed. TestRemoveContact: Incorrect values")
	}
}

func TestSendMessageToAll(t *testing.T) {
	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)
	result := New("bob", "pw", "Skynet", contacts)
	result.SendMessageToAll("hello world")
}

func TestSendMessage(t *testing.T) {
	contact := Contact{Name: "bob", Number: "1234", Enabled: true}
	var contacts []Contact
	contacts = append(contacts, contact)
	result := New("bob", "pw", "Skynet", contacts)
	err := result.SendMessage(contact.Number, "hello world")
	log.Println(err)
	t.Log(err)
}
