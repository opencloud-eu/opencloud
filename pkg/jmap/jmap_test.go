package jmap

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/require"
)

func jsoneq[X any](t *testing.T, expected string, object X) {
	data, err := json.MarshalIndent(object, "", "")
	require.NoError(t, err)
	require.JSONEq(t, expected, string(data))

	var rec X
	err = json.Unmarshal(data, &rec)
	require.NoError(t, err)
	require.Equal(t, object, rec)
}

func TestEmptySessionCapabilitiesMarshalling(t *testing.T) {
	jsoneq(t, `{}`, SessionCapabilities{})
}

func TestSessionCapabilitiesMarshalling(t *testing.T) {
	jsoneq(t, `{
		"urn:ietf:params:jmap:core": {
			"maxSizeUpload": 123,
			"maxConcurrentUpload": 4,
			"maxSizeRequest": 1000,
			"maxConcurrentRequests": 8,
			"maxCallsInRequest": 16,
			"maxObjectsInGet": 32,
			"maxObjectsInSet": 8
		},
		"urn:ietf:params:jmap:tasks": {
		}
	}`, SessionCapabilities{
		Core: &SessionCoreCapabilities{
			MaxSizeUpload:         123,
			MaxConcurrentUpload:   4,
			MaxSizeRequest:        1000,
			MaxConcurrentRequests: 8,
			MaxCallsInRequest:     16,
			MaxObjectsInGet:       32,
			MaxObjectsInSet:       8,
		},
		Tasks: &SessionTasksCapabilities{},
	})
}

type TestJmapWellKnownClient struct {
	t *testing.T
}

func NewTestJmapWellKnownClient(t *testing.T) SessionClient {
	return &TestJmapWellKnownClient{t: t}
}

func (t *TestJmapWellKnownClient) Close() error {
	return nil
}

func (t *TestJmapWellKnownClient) GetSession(sessionUrl *url.URL, username string, logger *log.Logger) (SessionResponse, Error) {
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
		Capabilities: SessionCapabilities{
			Core: &SessionCoreCapabilities{
				MaxCallsInRequest: 64,
			},
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

type TestJmapBlobClient struct {
	t *testing.T
}

func NewTestJmapBlobClient(t *testing.T) BlobClient {
	return &TestJmapBlobClient{t: t}
}

func (t TestJmapBlobClient) UploadBinary(ctx context.Context, logger *log.Logger, session *Session, uploadUrl string, endpoint string, contentType string, acceptLanguage string, body io.Reader) (UploadedBlob, Language, Error) {
	bytes, err := io.ReadAll(body)
	if err != nil {
		return UploadedBlob{}, "", SimpleError{code: 0, err: err}
	}
	hasher := sha512.New()
	hasher.Write(bytes)
	return UploadedBlob{
		BlobId: uuid.NewString(),
		Size:   len(bytes),
		Type:   contentType,
		Sha512: base64.StdEncoding.EncodeToString(hasher.Sum(nil)),
	}, "", nil
}

func (h *TestJmapBlobClient) DownloadBinary(ctx context.Context, logger *log.Logger, session *Session, downloadUrl string, endpoint string, acceptLanguage string) (*BlobDownload, Language, Error) {
	return &BlobDownload{
		Body:               io.NopCloser(strings.NewReader("")),
		Size:               -1,
		Type:               "text/plain",
		ContentDisposition: "attachment; filename=\"file.txt\"",
		CacheControl:       "",
	}, "", nil
}

func (t TestJmapBlobClient) Close() error {
	return nil
}

type TestWsClientFactory struct {
	WsClientFactory
}

var _ WsClientFactory = &TestWsClientFactory{}

func NewTestWsClientFactory(t *testing.T) WsClientFactory {
	return TestWsClientFactory{}
}

func (t TestWsClientFactory) EnableNotifications(pushState string, sessionProvider func() (*Session, error), listener WsPushListener) (WsClient, Error) {
	return nil, nil // TODO
}

func (t TestWsClientFactory) Close() error {
	return nil
}

func serveTestFile(t *testing.T, name string) ([]byte, Language, Error) {
	cwd, _ := os.Getwd()
	p := filepath.Join(cwd, "testdata", name)
	bytes, err := os.ReadFile(p)
	if err != nil {
		return bytes, "", SimpleError{code: 0, err: err}
	}
	// try to parse it first to avoid any deeper issues that are caused by the test tools
	var target map[string]any
	err = json.Unmarshal(bytes, &target)
	if err != nil {
		t.Errorf("failed to parse JSON test data file '%v': %v", p, err)
		return nil, "", SimpleError{code: 0, err: err}
	}
	return bytes, "", nil
}

func (t *TestJmapApiClient) Command(ctx context.Context, logger *log.Logger, session *Session, request Request, acceptLanguage string) ([]byte, Language, Error) {
	command := request.MethodCalls[0].Command
	switch command {
	case CommandMailboxGet:
		return serveTestFile(t.t, "mailboxes1.json")
	case CommandEmailQuery:
		return serveTestFile(t.t, "mails1.json")
	default:
		require.Fail(t.t, "TestJmapApiClient: unsupported jmap command: %v", command)
		return nil, "", SimpleError{code: 0, err: fmt.Errorf("TestJmapApiClient: unsupported jmap command: %v", command)}
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
	blobClient := NewTestJmapBlobClient(t)
	wsClientFactory := NewTestWsClientFactory(t)
	logger := log.NopLogger()
	ctx := context.Background()
	client := NewClient(wkClient, apiClient, blobClient, wsClientFactory)

	jmapUrl, err := url.Parse("http://localhost/jmap")
	require.NoError(err)

	session := Session{
		Username: "user123",
		JmapUrl:  *jmapUrl,
		SessionResponse: SessionResponse{
			Capabilities: SessionCapabilities{
				Core: &SessionCoreCapabilities{
					MaxCallsInRequest: 10,
				},
			},
		},
	}

	foldersByAccountId, sessionState, _, _, err := client.GetAllMailboxes([]string{"a"}, &session, ctx, &logger, "")
	require.NoError(err)
	require.Len(foldersByAccountId, 1)
	require.Contains(foldersByAccountId, "a")
	folders := foldersByAccountId["a"]
	require.Len(folders, 5)
	require.NotEmpty(sessionState)

	emails, sessionState, _, _, err := client.GetAllEmailsInMailbox("a", &session, ctx, &logger, "", "Inbox", 0, 0, false, true, 0, true)
	require.NoError(err)
	require.Len(emails.Emails, 3)
	require.NotEmpty(sessionState)

	{
		email := emails.Emails[0]
		require.Equal("Ornare Senectus Ultrices Elit", email.Subject)
		require.Equal(false, email.HasAttachment)
	}
	{
		email := emails.Emails[1]
		require.Equal("Lorem Tortor Eros Blandit Adipiscing Scelerisque Fermentum", email.Subject)
		require.Equal(false, email.HasAttachment)
	}
}

func TestEmailFilterSerialization(t *testing.T) {
	expectedFilterJson := `
{"operator":"AND","conditions":[{"hasKeyword":"seen","text":"sample"},{"hasKeyword":"draft"}]}
`

	require := require.New(t)

	text := "sample"
	mailboxId := ""
	notInMailboxIds := []string{}
	from := ""
	to := ""
	cc := ""
	bcc := ""
	subject := ""
	body := ""
	before := time.Time{}
	after := time.Time{}
	minSize := 0
	maxSize := 0
	keywords := []string{"seen", "draft"}

	var filter EmailFilterElement

	firstFilter := EmailFilterCondition{
		Text:               text,
		InMailbox:          mailboxId,
		InMailboxOtherThan: notInMailboxIds,
		From:               from,
		To:                 to,
		Cc:                 cc,
		Bcc:                bcc,
		Subject:            subject,
		Body:               body,
		Before:             before,
		After:              after,
		MinSize:            minSize,
		MaxSize:            maxSize,
	}
	filter = &firstFilter

	if len(keywords) > 0 {
		firstFilter.HasKeyword = keywords[0]
		if len(keywords) > 1 {
			firstFilter.HasKeyword = keywords[0]
			filters := make([]EmailFilterElement, len(keywords))
			filters[0] = firstFilter
			for i, keyword := range keywords[1:] {
				filters[i+1] = EmailFilterCondition{
					HasKeyword: keyword,
				}
			}
			filter = &EmailFilterOperator{
				Operator:   And,
				Conditions: filters,
			}
		}
	}

	b, err := json.Marshal(filter)
	require.NoError(err)
	json := string(b)
	require.Equal(strings.TrimSpace(expectedFilterJson), json)
}
