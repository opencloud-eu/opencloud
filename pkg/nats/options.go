package nats

import (
	"crypto/tls"

	"github.com/nats-io/nats.go"
)

func Secure(enableTLS, insecure bool, rootCA string) nats.Option {
	if enableTLS {
		if rootCA != "" {
			return nats.RootCAs(rootCA)
		}
		return nats.Secure(&tls.Config{
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: insecure,
		})
	}
	return nil
}

// Auth returns the option to authenticate with the given credentials, nil when no username is set.
func Auth(username, password string) nats.Option {
	if username != "" {
		return nats.UserInfo(username, password)
	}
	return nil
}
