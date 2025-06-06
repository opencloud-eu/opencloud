package jmap

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
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

func NewTestJmapWellKnownClient(t *testing.T) WellKnownClient {
	return &TestJmapWellKnownClient{t: t}
}

func (t *TestJmapWellKnownClient) GetWellKnown(username string, logger *log.Logger) (WellKnownResponse, error) {
	return WellKnownResponse{
		Username:        generateRandomString(8),
		ApiUrl:          "test://",
		PrimaryAccounts: map[string]string{JmapMail: generateRandomString(2 + seededRand.Intn(10))},
	}, nil
}

type TestJmapApiClient struct {
	t *testing.T
}

func NewTestJmapApiClient(t *testing.T) ApiClient {
	return &TestJmapApiClient{t: t}
}

func serveTestFile(t *testing.T, name string) ([]byte, error) {
	cwd, _ := os.Getwd()
	p := filepath.Join(cwd, "testdata", name)
	bytes, err := os.ReadFile(p)
	if err != nil {
		return bytes, err
	}
	// try to parse it first to avoid any deeper issues that are caused by the test tools
	var target map[string]any
	err = json.Unmarshal(bytes, &target)
	if err != nil {
		t.Errorf("failed to parse JSON test data file '%v': %v", p, err)
	}
	return bytes, err
}

func (t *TestJmapApiClient) Command(ctx context.Context, logger *log.Logger, session *Session, request Request) ([]byte, error) {
	command := request.MethodCalls[0].Command
	switch command {
	case MailboxGet:
		return serveTestFile(t.t, "mailboxes1.json")
	case EmailQuery:
		return serveTestFile(t.t, "mails1.json")
	default:
		require.Fail(t.t, "TestJmapApiClient: unsupported jmap command: %v", command)
		return nil, fmt.Errorf("TestJmapApiClient: unsupported jmap command: %v", command)
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

	session := Session{AccountId: "123", JmapUrl: "test://"}

	folders, err := client.GetMailboxes(&session, ctx, &logger)
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
