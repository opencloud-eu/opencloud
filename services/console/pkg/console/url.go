package console

import (
	"fmt"
	"net/url"
	"slices"
)

type URLBuilder struct {
	apiURL       url.URL
	websocketURL url.URL
}

func NewURLBuilder(claims *Claims) (URLBuilder, error) {
	apiURL, err := url.Parse(claims.Issuer)
	switch {
	case err != nil:
		return URLBuilder{}, err
	case !slices.Contains([]string{"http", "https"}, apiURL.Scheme):
		return URLBuilder{}, fmt.Errorf("invalid scheme: %s", apiURL.Scheme)
	}

	websocketURL, err := url.Parse(claims.WebsocketURL)
	switch {
	case err != nil:
		return URLBuilder{}, err
	case !slices.Contains([]string{"ws", "wss"}, websocketURL.Scheme):
		return URLBuilder{}, fmt.Errorf("invalid scheme: %s", apiURL.Scheme)
	}

	return URLBuilder{
		apiURL:       *apiURL.JoinPath("api", claims.APIVersion),
		websocketURL: *websocketURL,
	}, nil
}

func (b URLBuilder) APIUrl(elem ...string) *url.URL {
	return b.apiURL.JoinPath(elem...)
}

func (b URLBuilder) SubscriptionUrl() *url.URL {
	return &b.websocketURL
}
