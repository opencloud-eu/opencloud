package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	userpb "github.com/cs3org/go-cs3apis/cs3/identity/user/v1beta1"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/config"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/router"
	"github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend"
	backendmocks "github.com/opencloud-eu/opencloud/services/proxy/pkg/user/backend/mocks"
	revactx "github.com/opencloud-eu/reva/v2/pkg/ctx"
	"github.com/opencloud-eu/reva/v2/pkg/signedurl"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go-micro.dev/v4/store"
)

func TestSignedURLAuthLegacy_shouldServe(t *testing.T) {
	pua := SignedURLAuthenticator{}
	tests := []struct {
		url      string
		enabled  bool
		expected bool
	}{
		{"https://example.com/example.jpg", true, false},
		{"https://example.com/example.jpg?OC-Signature=something", true, true},
		{"https://example.com/example.jpg", false, false},
		{"https://example.com/example.jpg?OC-Signature=something", false, false},
	}

	for _, tt := range tests {
		pua.PreSignedURLConfig.Enabled = tt.enabled
		r := httptest.NewRequest("", tt.url, nil)
		result := pua.shouldServeLegacy(r)

		if result != tt.expected {
			t.Errorf("with %s expected %t got %t", tt.url, tt.expected, result)
		}
	}
}

func TestSignedURLAuth_shouldServe(t *testing.T) {
	tests := []struct {
		url      string
		secret   string
		enabled  bool
		expected bool
	}{
		{"https://example.com/example.jpg", "", true, false},
		{"https://example.com/example.jpg", "", false, false},
		{"https://example.com/example.jpg?oc-jwt-sig=something1", "secret", true, true},
		{"https://example.com/example.jpg?oc-jwt-sig=something2", "", true, false},
		{"https://example.com/example.jpg?oc-jwt-sig=something3", "secret", false, true},
	}

	for _, tt := range tests {
		pua := SignedURLAuthenticator{}
		pua.PreSignedURLConfig.Enabled = tt.enabled
		if tt.secret != "" {
			signURLVerifier, err := signedurl.NewJWTSignedURL(signedurl.WithSecret(tt.secret))
			if err != nil {
				t.Fatalf("failed to create signed URL verifier: %v", err)
			}
			pua.URLVerifier = signURLVerifier
		}
		r := httptest.NewRequest("", tt.url, nil)
		result := pua.shouldServe(r)

		if result != tt.expected {
			t.Errorf("with %s expected %t got %t", tt.url, tt.expected, result)
		}
	}
}

func TestSignedURLAuth_authenticateRejectsDisallowedMethods(t *testing.T) {
	signer, err := signedurl.NewJWTSignedURL(signedurl.WithSecret("secret"))
	if err != nil {
		t.Fatalf("failed to create signer: %v", err)
	}
	signed, err := signer.Sign("https://example.com/dav/spaces/file.txt", "userid", time.Minute)
	if err != nil {
		t.Fatalf("failed to sign url: %v", err)
	}

	pua := SignedURLAuthenticator{Logger: log.NewLogger(), URLVerifier: signer}
	pua.PreSignedURLConfig.AllowedHTTPMethods = []string{"GET"}

	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodDelete, "PROPFIND", "MOVE"} {
		r := httptest.NewRequest(method, signed, nil)
		if _, ok := pua.authenticate(r); ok {
			t.Errorf("expected %s with a signed url to be rejected", method)
		}
	}
}

func TestSignedURLAuthFailureSuppressesAuthenticationChallenge(t *testing.T) {
	oldStrategies := SupportedAuthStrategies
	SupportedAuthStrategies = nil
	t.Cleanup(func() { SupportedAuthStrategies = oldStrategies })

	verifier, err := signedurl.NewJWTSignedURL(signedurl.WithSecret("secret"))
	if err != nil {
		t.Fatalf("failed to create signed URL verifier: %v", err)
	}
	userProvider := &backendmocks.UserBackend{}
	userProvider.On("GetUserByClaims", mock.Anything, "username", "").Return(nil, "", backend.ErrAccountNotFound)

	tests := []struct {
		name              string
		url               string
		authenticator     SignedURLAuthenticator
		expectedChallenge bool
	}{
		{
			name: "invalid signed URL",
			url:  "https://example.com/file?oc-jwt-sig=invalid",
			authenticator: SignedURLAuthenticator{
				Logger:             log.NewLogger(),
				PreSignedURLConfig: config.PreSignedURL{AllowedHTTPMethods: []string{http.MethodGet}},
				URLVerifier:        verifier,
			},
			expectedChallenge: false,
		},
		{
			name: "invalid legacy signed URL",
			url:  "https://example.com/file?OC-Signature=invalid",
			authenticator: SignedURLAuthenticator{
				Logger:             log.NewLogger(),
				PreSignedURLConfig: config.PreSignedURL{Enabled: true},
				UserProvider:       userProvider,
			},
			expectedChallenge: false,
		},
		{
			name:              "legacy signed URLs disabled",
			url:               "https://example.com/file?OC-Signature=invalid",
			authenticator:     SignedURLAuthenticator{},
			expectedChallenge: true,
		},
		{
			name:              "signed URL verifier disabled",
			url:               "https://example.com/file?oc-jwt-sig=invalid",
			authenticator:     SignedURLAuthenticator{},
			expectedChallenge: true,
		},
		{
			name:              "unsigned URL",
			url:               "https://example.com/file",
			authenticator:     SignedURLAuthenticator{URLVerifier: verifier},
			expectedChallenge: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SupportedAuthStrategies = nil
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			req = req.WithContext(router.SetRoutingInfo(req.Context(), router.RoutingInfo{}))
			rr := httptest.NewRecorder()
			nextCalled := false
			handler := Authentication([]Authenticator{tt.authenticator}, EnableBasicAuth(true))(
				http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true }),
			)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code)
			assert.False(t, nextCalled)
			if tt.expectedChallenge {
				assert.NotEmpty(t, rr.Header().Values(WwwAuthenticate))
			} else {
				assert.Empty(t, rr.Header().Values(WwwAuthenticate))
			}
		})
	}
}

func TestSignedURLAuth_allRequiredParametersPresent(t *testing.T) {
	pua := SignedURLAuthenticator{}
	baseURL := "https://example.com/example.jpg?"
	tests := []struct {
		params       string
		errorMessage string
	}{
		{"OC-Signature=something&OC-Credential=something&OC-Date=something&OC-Expires=something&OC-Verb=something", ""},
		{"OC-Credential=something&OC-Date=something&OC-Expires=something&OC-Verb=something", "required OC-Signature parameter not found"},
		{"OC-Signature=something&OC-Date=something&OC-Expires=something&OC-Verb=something", "required OC-Credential parameter not found"},
		{"OC-Signature=something&OC-Credential=something&OC-Expires=something&OC-Verb=something", "required OC-Date parameter not found"},
		{"OC-Signature=something&OC-Credential=something&OC-Date=something&OC-Verb=something", "required OC-Expires parameter not found"},
		{"OC-Signature=something&OC-Credential=something&OC-Date=something&OC-Expires=something", "required OC-Verb parameter not found"},
	}
	for _, tt := range tests {
		r := httptest.NewRequest("", baseURL+tt.params, nil)
		err := pua.allRequiredParametersArePresent(r.URL.Query())
		if tt.errorMessage != "" {
			assert.EqualError(t, err, tt.errorMessage, tt.params)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestSignedURLAuth_requestMethodMatches(t *testing.T) {
	pua := SignedURLAuthenticator{}
	tests := []struct {
		method       string
		url          string
		errorMessage string
	}{
		{"GET", "https://example.com/example.jpg?OC-Verb=GET", ""},
		{"GET", "https://example.com/example.jpg?OC-Verb=get", ""},
		{"POST", "https://example.com/example.jpg?OC-Verb=GET", "required OC-Verb parameter did not match request method"},
	}

	for _, tt := range tests {
		r := httptest.NewRequest(tt.method, tt.url, nil)
		err := pua.requestMethodMatches(r.Method, r.URL.Query())
		if tt.errorMessage != "" {
			assert.EqualError(t, err, tt.errorMessage, tt.url)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestSignedURLAuth_requestMethodIsAllowed(t *testing.T) {
	pua := SignedURLAuthenticator{}
	tests := []struct {
		method       string
		allowed      []string
		errorMessage string
	}{
		{"GET", []string{}, "request method is not listed in PreSignedURLConfig AllowedHTTPMethods"},
		{"GET", []string{"POST"}, "request method is not listed in PreSignedURLConfig AllowedHTTPMethods"},
		{"GET", []string{"GET"}, ""},
		{"GET", []string{"get"}, ""},
		{"GET", []string{"POST", "GET"}, ""},
	}

	for _, tt := range tests {
		pua.PreSignedURLConfig.AllowedHTTPMethods = tt.allowed
		err := pua.requestMethodIsAllowed(tt.method)
		if tt.errorMessage != "" {
			assert.EqualError(t, err, tt.errorMessage)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestSignedURLAuth_urlIsExpired(t *testing.T) {
	nowFunc := func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2020-02-02T12:30:00.000Z")
		return t
	}
	pua := SignedURLAuthenticator{
		Now: nowFunc,
	}

	tests := []struct {
		url          string
		errorMessage string
	}{
		// a valid signed url
		{"http://example.com/example.jpg?OC-Date=2020-02-02T12:29:00.000Z&OC-Expires=61", ""},
		// invalid expiry
		{"http://example.com/example.jpg?OC-Date=2020-02-02T12:29:00.000Z&OC-Expires=invalid", "time: invalid duration \"invalids\""},
		// wrong date format on OC-Date
		{"http://example.com/example.jpg?OC-Date=2020-02-02TTT12:29:00.000Z&OC-Expires=5", "parsing time \"2020-02-02TTT12:29:00.000Z\" as \"2006-01-02T15:04:05Z07:00\": cannot parse \"TT12:29:00.000Z\" as \"15\""},
		// expired - 12:29:00 + 59s < 12:30
		{"http://example.com/example.jpg?OC-Date=2020-02-02T12:29:00.000Z&OC-Expires=59", "URL is expired"},
		// expired - basically url was created yesterday
		{"http://example.com/example.jpg?OC-Date=2020-02-01T12:29:00.000Z&OC-Expires=59", "URL is expired"},
		// future OC-Date - also valid now
		{"http://example.com/example.jpg?OC-Date=2020-02-03T12:29:00.000Z&OC-Expires=59", ""},
	}

	for _, tt := range tests {
		r := httptest.NewRequest("", tt.url, nil)
		err := pua.urlIsExpired(r.URL.Query())
		if tt.errorMessage != "" {
			assert.EqualError(t, err, tt.errorMessage, tt.url)
		} else {
			assert.Nil(t, err)
		}
	}
}

func TestSignedURLAuth_createSignature(t *testing.T) {
	pua := SignedURLAuthenticator{}
	expected := "27d2ebea381384af3179235114801dcd00f91e46f99fca72575301cf3948101d"
	s := pua.createSignature("something", []byte("somerandomkey"))

	if s != expected {
		t.Fail()
	}
}

func TestSignedURLAuth_validate(t *testing.T) {
	nowFunc := func() time.Time {
		t, _ := time.Parse(time.RFC3339, "2020-02-02T12:30:00.000Z")
		return t
	}
	cfg := config.PreSignedURL{
		AllowedHTTPMethods: []string{"get"},
		Enabled:            true,
	}
	pua := SignedURLAuthenticator{
		PreSignedURLConfig: cfg,
		Store:              store.NewMemoryStore(),
		Now:                nowFunc,
	}

	pua.Store.Write(&store.Record{
		Key:      "useri",
		Value:    []byte("1234567890"),
		Metadata: nil,
	})

	tests := []struct {
		now          string
		url          string
		errorMessage string
	}{
		{"2020-02-02T12:30:00.000Z", "http://example.com/example.jpg?OC-Date=2020-02-02T12:29:00.000Z&OC-Expires=invalid", "required OC-Signature parameter not found"},
		{"2020-02-02T12:30:00.000Z", "http://cloud.example.net/?OC-Credential=alice&OC-Date=2019-05-14T11%3A01%3A58.135Z&OC-Expires=1200&OC-Verb=GET&OC-Signature=f9e53a1ee23caef10f72ec392c1b537317491b687bfdd224c782be197d9ca2b6", "URL is expired"},
		{"2019-05-14T11:02:00.000Z", "http://cloud.example.net/?OC-Credential=alice&OC-Date=2019-05-14T11%3A01%3A58.135Z&OC-Expires=1200&OC-Verb=GET&OC-Signature=f9e53a1ee23caef10f72ec392c1b537317491b687bfdd224c782be197d9ca2b", "signature mismatch: expected f9e53a1ee23caef10f72ec392c1b537317491b687bfdd224c782be197d9ca2b6 != actual f9e53a1ee23caef10f72ec392c1b537317491b687bfdd224c782be197d9ca2b"},
		{"2019-05-14T11:02:00.000Z", "http://cloud.example.net/?OC-Credential=alice&OC-Date=2019-05-14T11%3A01%3A58.135Z&OC-Expires=1200&OC-Verb=GET&OC-Signature=f9e53a1ee23caef10f72ec392c1b537317491b687bfdd224c782be197d9ca2b6", ""},
		{"2019-05-14T11:02:00.000Z", "http://cloud.example.net/?OC-Date=2019-05-14T11%3A01%3A58.135Z&OC-Expires=1200&OC-Verb=GET&OC-Credential=alice&OC-Signature=f9e53a1ee23caef10f72ec392c1b537317491b687bfdd224c782be197d9ca2b6", ""},
		{"2019-05-14T11:02:00.000Z", "http://cloud.example.net/?OC-Algo=PBKDF2%2F10000-SHA512&OC-Date=2019-05-14T11%3A01%3A58.135Z&OC-Expires=1200&OC-Verb=GET&OC-Credential=alice&OC-Signature=f9e53a1ee23caef10f72ec392c1b537317491b687bfdd224c782be197d9ca2b6", ""},
		{"2024-02-07T12:03:11.966Z", "http://localhost:33001/try?id=1&id=2&OC-Credential=user&OC-Date=2024-02-07T12%3A03%3A11.966Z&OC-Expires=2&OC-Verb=GET&OC-Algo=PBKDF2%2F10000-SHA512&OC-Signature=86e21a1efbf0be989a206109cfedf70a22f338dc8995e849ce002032bc6741c5", ""},
	}

	for _, tt := range tests {
		u := userpb.User{
			Id:          &userpb.UserId{OpaqueId: "useri"},
			DisplayName: "Test User",
		}
		ctx := revactx.ContextSetUser(context.Background(), &u)

		pua.Now = func() time.Time {
			t, _ := time.Parse(time.RFC3339, tt.now)
			return t
		}

		r := httptest.NewRequest("", tt.url, nil).WithContext(ctx)
		err := pua.validate(r)
		if tt.errorMessage == "" {
			assert.Nil(t, err)
		} else {
			assert.EqualError(t, err, tt.errorMessage)
		}
	}
}
