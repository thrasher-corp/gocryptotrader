package main

import (
	"net/url"
	"strings"
	"log"
)

const (
	SMSGLOBAL_API_URL = "http://www.smsglobal.com/http-api.php"
)

func SMSSendToAll(message string) {
	for _, contact := range bot.config.SMSContacts {
		if contact.Enabled {
			err := SMSNotify(contact.Number, message)

			if err != nil {
				log.Println(err)
			}
		}
	}
}

func SMSGetNumberByName(name string) (string) {
	for _, contact := range bot.config.SMSContacts {
		if contact.Name == name {
			return contact.Number
		}
	}
	return ""
}

func SMSNotify(to, message string) (error) {
	values := url.Values{}
	values.Set("action", "sendsms")
	values.Set("user", bot.config.SMSGlobalUsername)
	values.Set("password", bot.config.SMSGlobalPassword)
	values.Set("from", bot.config.Name)
	values.Set("to", to)
	values.Set("text", message)

	headers := make(map[string]string)
	headers["Content-Type"] = "application/x-www-form-urlencoded"

	resp, err := SendHTTPRequest("POST", SMSGLOBAL_API_URL, headers, strings.NewReader(values.Encode()))

	if err != nil {
		return err
	}

	log.Printf("Recieved raw: %s\n", resp)
	return nil
}