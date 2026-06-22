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
