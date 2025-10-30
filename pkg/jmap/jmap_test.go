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
	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
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

func TestUtcDateUnmarshalling(t *testing.T) {
	require := require.New(t)
	r := struct {
		Ts UTCDate `json:"ts"`
	}{}
	err := json.Unmarshal([]byte(`{"ts":"2025-10-30T14:15:16.987Z"}`), &r)
	require.NoError(err)
	require.Equal(2025, r.Ts.Year())
	require.Equal(time.Month(10), r.Ts.Month())
	require.Equal(30, r.Ts.Day())
	require.Equal(14, r.Ts.Hour())
	require.Equal(15, r.Ts.Minute())
	require.Equal(16, r.Ts.Second())
	require.Equal(987000000, r.Ts.Nanosecond())
}

func TestUtcDateMarshalling(t *testing.T) {
	require := require.New(t)
	r := struct {
		Ts UTCDate `json:"ts"`
	}{}
	ts, err := time.Parse(time.RFC3339, "2025-10-30T14:15:16.987Z")
	require.NoError(err)
	r.Ts = UTCDate{ts}

	jsoneq(t, `{"ts":"2025-10-30T14:15:16.987Z"}`, r)
}

func TestUtcDateUnmarshallingWithWeirdDate(t *testing.T) {
	require := require.New(t)
	r := struct {
		Ts UTCDate `json:"ts"`
	}{}
	err := json.Unmarshal([]byte(`{"ts":"65534-12-31T23:59:59Z"}`), &r)
	require.NoError(err)
	require.Equal(65534, r.Ts.Year())
	require.Equal(time.Month(12), r.Ts.Month())
	require.Equal(31, r.Ts.Day())
	require.Equal(23, r.Ts.Hour())
	require.Equal(59, r.Ts.Minute())
	require.Equal(59, r.Ts.Second())
	require.Equal(0, r.Ts.Nanosecond())
}

func TestUnmarshallingCalendarEvent(t *testing.T) {
	payload := `
{
   "locale" : "en-US",
   "description" : "Internal meeting about the grand strategy for the future",
   "locations" : {
      "ux1uokie" : {
         "links" : {
            "eefe2pax" : {
               "@type" : "Link",
               "href" : "https://example.com/office"
            }
         },
         "iCalComponent" : {
            "name" : "vlocation"
         },
         "@type" : "Location",
         "description" : "Office meeting room upstairs",
         "name" : "Office",
         "locationTypes" : {
            "office" : true
         },
         "coordinates" : "geo:52.5334956,13.4079872",
         "relativeTo" : "start",
         "timeZone" : "CEST"
      }
   },
   "replyTo" : {
      "imip" : "mailto:organizer@example.com"
   },
   "links" : {
      "cai0thoh" : {
         "href" : "https://example.com/9a7ab91a-edca-4988-886f-25e00743430d",
         "rel" : "about",
         "contentType" : "text/html",
         "@type" : "Link"
      }
   },
   "prodId" : "Mock 0.0",
   "@type" : "Event",
   "keywords" : {
      "secret" : true,
      "meeting" : true
   },
   "status" : "confirmed",
   "freeBusyStatus" : "busy",
   "categories" : {
      "internal" : true,
      "secret" : true
   },
   "duration" : "PT30M",
   "calendarIds" : {
      "b" : true
   },
   "alerts" : {
      "ahqu4xi0" : {
         "@type" : "Alert"
      }
   },
   "start" : "2025-09-30T12:00:00",
   "privacy" : "public",
   "isDraft" : false,
   "id" : "c",
   "isOrigin" : true,
   "sentBy" : "organizer@example.com",
   "descriptionContentType" : "text/plain",
   "updated" : "2025-09-29T16:17:18Z",
   "created" : "2025-09-29T16:17:18Z",
   "color" : "purple",
   "recurrenceRule" : {
      "skip" : "omit",
      "count" : 4,
      "firstDayOfWeek" : "monday",
      "rscale" : "iso8601",
      "interval" : 1,
      "frequency" : "weekly"
   },
   "timeZone" : "Etc/UTC",
   "title" : "Meeting of the Minds",
   "participants" : {
      "xeikie9p" : {
         "name" : "Klaes Ashford",
         "locationId" : "em4eal0o",
         "language" : "en-GB",
         "description" : "As the first officer on the Behemoth",
         "invitedBy" : "eegh7uph",
         "@type" : "Participant",
         "links" : {
            "oifooj6g" : {
               "@type" : "Link",
               "contentType" : "image/png",
               "display" : "badge",
               "href" : "https://static.wikia.nocookie.net/expanse/images/0/02/Klaes_Ashford_-_Expanse_season_4_promotional_2.png/revision/latest?cb=20191206012007",
               "rel" : "icon",
               "title" : "Ashford on Medina Station"
            }
         },
         "iCalComponent" : {
            "name" : "participant"
         },
         "scheduleAgent" : "server",
         "scheduleId" : "mailto:ashford@opa.org"
      },
      "eegh7uph" : {
         "description" : "Called the meeting",
         "language" : "en-GB",
         "locationId" : "ux1uokie",
         "name" : "Anderson Dawes",
         "scheduleAgent" : "server",
         "scheduleUpdated" : "2025-10-01T11:59:12Z",
         "links" : {
            "ieni5eiw" : {
               "href" : "https://static.wikia.nocookie.net/expanse/images/1/1e/OPA_leader.png/revision/latest?cb=20250121103410",
               "display" : "badge",
               "rel" : "icon",
               "title" : "Anderson Dawes",
               "contentType" : "image/png",
               "@type" : "Link"
            }
         },
         "iCalComponent" : {
            "name" : "participant"
         },
         "@type" : "Participant",
         "invitedBy" : "eegh7uph",
         "scheduleSequence" : 1,
         "sendTo" : {
            "imip" : "mailto:adawes@opa.org"
         },
         "scheduleStatus" : [
            "1.0"
         ],
         "scheduleId" : "mailto:adawes@opa.org"
      }
   },
   "uid" : "9a7ab91a-edca-4988-886f-25e00743430d",
   "virtualLocations" : {
      "em4eal0o" : {
         "@type" : "VirtualLocation",
         "description" : "The opentalk Conference Room",
         "uri" : "https://meet.opentalk.eu",
         "features" : {
            "audio" : true,
            "screen" : true,
            "chat" : true,
            "video" : true
         },
         "name" : "opentalk"
      }
   }
}
   `
	require := require.New(t)
	var result CalendarEvent
	err := json.Unmarshal([]byte(payload), &result)
	require.NoError(err)

	require.Len(result.VirtualLocations, 1)
	require.Len(result.Locations, 1)
	require.Equal("9a7ab91a-edca-4988-886f-25e00743430d", result.Uid)
	require.Equal(jscalendar.PrivacyPublic, result.Privacy)
}

func TestUnmarshallingCalendarEventGetResponse(t *testing.T) {
	payload := `
{
   "sessionState" : "7d3cae5b",
   "methodResponses" : [
      [
         "CalendarEvent/query",
         {
            "position" : 0,
            "queryState" : "s2yba",
            "accountId" : "b",
            "canCalculateChanges" : true,
            "ids" : [
               "c"
            ]
         },
         "b:0"
      ],
      [
         "CalendarEvent/get",
         {
            "state" : "s2yba",
            "list" : [
               {
                  "links" : {
                     "cai0thoh" : {
                        "contentType" : "text/html",
                        "href" : "https://example.com/9a7ab91a-edca-4988-886f-25e00743430d",
                        "@type" : "Link",
                        "rel" : "about"
                     }
                  },
                  "freeBusyStatus" : "busy",
                  "color" : "purple",
                  "isDraft" : false,
                  "calendarIds" : {
                     "b" : true
                  },
                  "updated" : "2025-09-29T16:17:18Z",
                  "locations" : {
                     "ux1uokie" : {
                        "relativeTo" : "start",
                        "description" : "Office meeting room upstairs",
                        "coordinates" : "geo:52.5334956,13.4079872",
                        "name" : "Office",
                        "locationTypes" : {
                           "office" : true
                        },
                        "links" : {
                           "eefe2pax" : {
                              "href" : "https://example.com/office",
                              "@type" : "Link"
                           }
                        },
                        "iCalComponent" : {
                           "name" : "vlocation"
                        },
                        "@type" : "Location",
                        "timeZone" : "CEST"
                     }
                  },
                  "virtualLocations" : {
                     "em4eal0o" : {
                        "name" : "opentalk",
                        "@type" : "VirtualLocation",
                        "features" : {
                           "screen" : true,
                           "chat" : true,
                           "audio" : true,
                           "video" : true
                        },
                        "uri" : "https://meet.opentalk.eu",
                        "description" : "The opentalk Conference Room"
                     }
                  },
                  "uid" : "9a7ab91a-edca-4988-886f-25e00743430d",
                  "categories" : {
                     "secret" : true,
                     "internal" : true
                  },
                  "keywords" : {
                     "secret" : true,
                     "meeting" : true
                  },
                  "replyTo" : {
                     "imip" : "mailto:organizer@example.com"
                  },
                  "duration" : "PT30M",
                  "created" : "2025-09-29T16:17:18Z",
                  "start" : "2025-09-30T12:00:00",
                  "id" : "c",
                  "sentBy" : "organizer@example.com",
                  "timeZone" : "Etc/UTC",
                  "@type" : "Event",
                  "title" : "Meeting of the Minds",
                  "alerts" : {
                     "ahqu4xi0" : {
                        "@type" : "Alert"
                     }
                  },
                  "participants" : {
                     "xeikie9p" : {
                        "@type" : "Participant",
                        "scheduleId" : "mailto:ashford@opa.org",
                        "invitedBy" : "eegh7uph",
                        "language" : "en-GB",
                        "links" : {
                           "oifooj6g" : {
                              "contentType" : "image/png",
                              "display" : "badge",
                              "title" : "Ashford on Medina Station",
                              "@type" : "Link",
                              "href" : "https://static.wikia.nocookie.net/expanse/images/0/02/Klaes_Ashford_-_Expanse_season_4_promotional_2.png/revision/latest?cb=20191206012007",
                              "rel" : "icon"
                           }
                        },
                        "iCalComponent" : {
                           "name" : "participant"
                        },
                        "scheduleAgent" : "server",
                        "name" : "Klaes Ashford",
                        "locationId" : "em4eal0o",
                        "description" : "As the first officer on the Behemoth"
                     },
                     "eegh7uph" : {
                        "description" : "Called the meeting",
                        "locationId" : "ux1uokie",
                        "scheduleUpdated" : "2025-10-01T11:59:12Z",
                        "sendTo" : {
                           "imip" : "mailto:adawes@opa.org"
                        },
                        "scheduleAgent" : "server",
                        "scheduleStatus" : [
                           "1.0"
                        ],
                        "name" : "Anderson Dawes",
                        "invitedBy" : "eegh7uph",
                        "language" : "en-GB",
                        "links" : {
                           "ieni5eiw" : {
                              "rel" : "icon",
                              "display" : "badge",
                              "@type" : "Link",
                              "title" : "Anderson Dawes",
                              "href" : "https://static.wikia.nocookie.net/expanse/images/1/1e/OPA_leader.png/revision/latest?cb=20250121103410",
                              "contentType" : "image/png"
                           }
                        },
                        "iCalComponent" : {
                           "name" : "participant"
                        },
                        "scheduleSequence" : 1,
                        "scheduleId" : "mailto:adawes@opa.org",
                        "@type" : "Participant"
                     }
                  },
                  "status" : "confirmed",
                  "description" : "Internal meeting about the grand strategy for the future",
                  "locale" : "en-US",
                  "recurrenceRule" : {
                     "count" : 4,
                     "rscale" : "iso8601",
                     "frequency" : "weekly",
                     "interval" : 1,
                     "firstDayOfWeek" : "monday",
                     "skip" : "omit"
                  },
                  "descriptionContentType" : "text/plain",
                  "isOrigin" : true,
                  "prodId" : "Mock 0.0",
                  "privacy" : "public"
               }
            ],
            "accountId" : "b",
            "notFound" : []
         },
         "b:1"
      ]
   ]
}
	`

	require := require.New(t)
	var response Response
	err := json.Unmarshal([]byte(payload), &response)
	require.NoError(err)
	r1 := response.MethodResponses[1]
	require.Equal(CommandCalendarEventGet, r1.Command)
	get := r1.Parameters.(CalendarEventGetResponse)
	require.Len(get.List, 1)
	result := get.List[0]
	require.Len(result.VirtualLocations, 1)
	require.Len(result.Locations, 1)
	require.Equal("9a7ab91a-edca-4988-886f-25e00743430d", result.Uid)
	require.Equal(jscalendar.PrivacyPublic, result.Privacy)
}
