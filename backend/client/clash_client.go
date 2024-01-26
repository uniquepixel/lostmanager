package client

import (
	"errors"
	"strings"
	"time"

	"github.com/aaantiii/goclash"

	"github.com/aaantiii/lostapp/backend/env"
)

func NewClashClient() (*goclash.Client, error) {
	emails := strings.Split(env.COC_API_EMAILS.Value(), ",")
	passwords := strings.Split(env.COC_API_PASSWORDS.Value(), ",")
	if len(emails) != len(passwords) {
		return nil, errors.New("invalid COC-API credentials")
	}

	credentials := make(goclash.Credentials, len(emails))
	for i, email := range emails {
		credentials[email] = passwords[i]
	}

	client, err := goclash.New(credentials)
	if err != nil {
		return nil, err
	}
	client.UseCache(true)
	client.SetCacheTime(time.Minute * 3)
	return client, nil
}
