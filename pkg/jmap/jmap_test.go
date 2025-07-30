package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/require"
)

type TestJmapWellKnownClient struct {
	t *testing.T
}

func NewTestJmapWellKnownClient(t *testing.T) SessionClient {
	return &TestJmapWellKnownClient{t: t}
}

func (t *TestJmapWellKnownClient) Close() error {
	return nil
}

func (t *TestJmapWellKnownClient) GetSession(username string, logger *log.Logger) (SessionResponse, Error) {
	pa := generateRandomString(2 + seededRand.Intn(10))
	return SessionResponse{
		Username: generateRandomString(8),
		ApiUrl:   "test://",
		PrimaryAccounts: SessionPrimaryAccounts{
			Core:             pa,
			Mail:             pa,
			Submission:       pa,
			VacationResponse: pa,
			Sieve:            pa,
			Blob:             pa,
			Quota:            pa,
			Websocket:        pa,
		},
	}, nil
}

type TestJmapApiClient struct {
	t *testing.T
}

func NewTestJmapApiClient(t *testing.T) ApiClient {
	return &TestJmapApiClient{t: t}
}

func (t TestJmapApiClient) Close() error {
	return nil
}

func serveTestFile(t *testing.T, name string) ([]byte, Error) {
	cwd, _ := os.Getwd()
	p := filepath.Join(cwd, "testdata", name)
	bytes, err := os.ReadFile(p)
	if err != nil {
		return bytes, SimpleError{code: 0, err: err}
	}
	// try to parse it first to avoid any deeper issues that are caused by the test tools
	var target map[string]any
	err = json.Unmarshal(bytes, &target)
	if err != nil {
		t.Errorf("failed to parse JSON test data file '%v': %v", p, err)
	}
	return bytes, SimpleError{code: 0, err: err}
}

func (t *TestJmapApiClient) Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, Error) {
	command := request.MethodCalls[0].Command
	switch command {
	case MailboxGet:
		return serveTestFile(t.t, "mailboxes1.json")
	case EmailQuery:
		return serveTestFile(t.t, "mails1.json")
	default:
		require.Fail(t.t, "TestJmapApiClient: unsupported jmap command: %v", command)
		return nil, SimpleError{code: 0, err: fmt.Errorf("TestJmapApiClient: unsupported jmap command: %v", command)}
	}
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func generateRandomString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func TestRequests(t *testing.T) {
	require := require.New(t)
	apiClient := NewTestJmapApiClient(t)
	wkClient := NewTestJmapWellKnownClient(t)
	logger := log.NopLogger()
	ctx := context.Background()
	client := NewClient(wkClient, apiClient)

	jmapUrl, err := url.Parse("http://localhost/jmap")
	require.NoError(err)

	session := Session{MailAccountId: "123", Username: "user123", JmapUrl: *jmapUrl}

	folders, err := client.GetAllMailboxes(&session, ctx, &logger)
	require.NoError(err)
	require.Len(folders.List, 5)

	emails, err := client.GetEmails(&session, ctx, &logger, "Inbox", 0, 0, true, 0)
	require.NoError(err)
	require.Len(emails.Emails, 3)

	{
		email := emails.Emails[0]
		require.Equal("Ornare Senectus Ultrices Elit", email.Subject)
		require.Equal(false, email.HasAttachments)
	}
	{
		email := emails.Emails[1]
		require.Equal("Lorem Tortor Eros Blandit Adipiscing Scelerisque Fermentum", email.Subject)
		require.Equal(false, email.HasAttachments)
	}
}
