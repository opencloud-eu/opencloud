package jmap

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/stretchr/testify/require"
)

const mails1 = `{"methodResponses": [
["Email/query",{
  "accountId":"j",
  "queryState":"sqcakzewfqdk7oay",
  "canCalculateChanges":true,
  "position":0,
  "ids":["fmaaaabh"],
  "total":1
},"0"],
["Email/get",{
  "accountId":"j",
  "state":"sqcakzewfqdk7oay",
  "list":[
    {"threadId":"bl","id":"fmaaaabh"}
  ],"notFound":[]
},"1"],
["Thread/get",{
  "accountId":"j",
  "state":"sqcakzewfqdk7oay",
  "list":[
    {"id":"bl","emailIds":["fmaaaabh"]}
  ],"notFound":[]
},"2"],
["Email/get",{
  "accountId":"j",
  "state":"sqcakzewfqdk7oay",
  "list":[
    {"threadId":"bl","mailboxIds":{"a":true},"keywords":{},"hasAttachment":false,"from":[{"name":"current generally","email":"current.generally"}],"subject":"eros auctor proin","receivedAt":"2025-04-30T09:47:44Z","size":15423,"preview":"Lorem ipsum dolor sit amet consectetur adipiscing elit sed urna tristique himenaeos eu a mattis laoreet aliquet enim. Magnis est facilisis nibh nisl vitae nisi mauris nostra velit donec erat pellentesque sagittis ligula turpis suscipit ultricies. Morbi ...","id":"fmaaaabh"}
  ],"notFound":[]
},"3"]
],"sessionState":"3e25b2a0"
}`

const mailboxes = `{"methodResponses": [
	["Mailbox/get", {
		"accountId":"cs",
		"state":"n",
		"list": [
			{
				"id":"a",
				"name":"Inbox",
				"parentId":null,
				"role":"inbox",
				"sortOrder":0,
				"isSubscribed":true,
				"totalEmails":0,
				"unreadEmails":0,
				"totalThreads":0,
				"unreadThreads":0,
				"myRights":{
					"mayReadItems":true,
					"mayAddItems":true,
					"mayRemoveItems":true,
					"maySetSeen":true,
					"maySetKeywords":true,
					"mayCreateChild":true,
					"mayRename":true,
					"mayDelete":true,
					"maySubmit":true
				}
			},{
				"id":"b",
				"name":"Deleted Items",
				"parentId":null,
				"role":"trash",
				"sortOrder":0,
				"isSubscribed":true,
				"totalEmails":0,
				"unreadEmails":0,
				"totalThreads":0,
				"unreadThreads":0,
				"myRights":{
					"mayReadItems":true,
					"mayAddItems":true,
					"mayRemoveItems":true,
					"maySetSeen":true,
					"maySetKeywords":true,
					"mayCreateChild":true,
					"mayRename":true,
					"mayDelete":true,
					"maySubmit":true
				}
			},{
				"id":"c",
				"name":"Junk Mail",
				"parentId":null,
				"role":"junk",
				"sortOrder":0,
				"isSubscribed":true,
				"totalEmails":0,
				"unreadEmails":0,
				"totalThreads":0,
				"unreadThreads":0,
				"myRights":{
					"mayReadItems":true,
					"mayAddItems":true,
					"mayRemoveItems":true,
					"maySetSeen":true,
					"maySetKeywords":true,
					"mayCreateChild":true,
					"mayRename":true,
					"mayDelete":true,
					"maySubmit":true
				}
			},{
				"id":"d",
				"name":"Drafts",
				"parentId":null,
				"role":"drafts",
				"sortOrder":0,
				"isSubscribed":true,
				"totalEmails":0,
				"unreadEmails":0,
				"totalThreads":0,
				"unreadThreads":0,
				"myRights":{
					"mayReadItems":true,
					"mayAddItems":true,
					"mayRemoveItems":true,
					"maySetSeen":true,
					"maySetKeywords":true,
					"mayCreateChild":true,
					"mayRename":true,
					"mayDelete":true,
					"maySubmit":true
				}
			},{
				"id":"e",
				"name":"Sent Items",
				"parentId":null,
				"role":"sent",
				"sortOrder":0,
				"isSubscribed":true,
				"totalEmails":0,
				"unreadEmails":0,
				"totalThreads":0,
				"unreadThreads":0,
				"myRights":{
					"mayReadItems":true,
					"mayAddItems":true,
					"mayRemoveItems":true,
					"maySetSeen":true,
					"maySetKeywords":true,
					"mayCreateChild":true,
					"mayRename":true,
					"mayDelete":true,
					"maySubmit":true
				}
			}
		],
		"notFound":[]
	},"0"]
], "sessionState":"3e25b2a0"
}`
const mails2 = `{"methodResponses":[
   ["Email/query",{
     "accountId":"j",
     "queryState":"sqcakzewfqdk7oay",
     "canCalculateChanges":true,
     "position":0,
     "ids":["fmaaaabh"],
     "total":1
   }, "0"],
   ["Email/get", {
     "accountId":"j",
     "state":"sqcakzewfqdk7oay",
     "list":[
       {
         "threadId":"bl",
         "id":"fmaaaabh"
       }
     ],
     "notFound":[]
   }, "1"],
   ["Thread/get",{
     "accountId":"j",
     "state":"sqcakzewfqdk7oay",
     "list":[
       {
         "id":"bl",
         "emailIds":["fmaaaabh"]
       }
     ],
     "notFound":[]
   }, "2 "],
   ["Email/get",{
     "accountId":"j",
     "state":"sqcakzewfqdk7oay",
     "list":[
       {
         "threadId":"bl",
         "mailboxIds":{"a":true},
         "keywords":{},
         "hasAttachment":false,
         "from":[
           {"name":"current generally", "email":"current.generally@example.com"}
         ],
         "subject":"eros auctor proin",
         "receivedAt":"2025-04-30T09:47:44Z",
         "size":15423,
         "preview":"Lorem ipsum dolor sit amet consectetur adipiscing elit sed urna tristique himenaeos eu a mattis laoreet aliquet enim. Magnis est facilisis nibh nisl vitae nisi mauris nostra velit donec erat pellentesque sagittis ligula turpis suscipit ultricies. Morbi ...",
         "id":"fmaaaabh"
       }
     ],
     "notFound":[]
   }, "3"]
   ], "sessionState":"3e25b2a0"
 }`

type TestJmapWellKnownClient struct {
	t *testing.T
}

func NewTestJmapWellKnownClient(t *testing.T) JmapWellKnownClient {
	return &TestJmapWellKnownClient{t: t}
}

func (t *TestJmapWellKnownClient) GetWellKnown(username string, logger *log.Logger) (WellKnownJmap, error) {
	return WellKnownJmap{
		ApiUrl:          "test://",
		PrimaryAccounts: map[string]string{JmapMail: generateRandomString(2 + seededRand.Intn(10))},
	}, nil
}

type TestJmapApiClient struct {
	t *testing.T
}

func NewTestJmapApiClient(t *testing.T) JmapApiClient {
	return &TestJmapApiClient{t: t}
}

func (t *TestJmapApiClient) Command(ctx context.Context, logger *log.Logger, request map[string]any) ([]byte, error) {
	methodCalls := request["methodCalls"].(*[][]any)
	command := (*methodCalls)[0][0].(string)
	switch command {
	case "Mailbox/get":
		return []byte(mailboxes), nil
	case "Email/query":
		return []byte(mails1), nil
	default:
		require.Fail(t.t, "unsupported jmap command: %v", command)
		return nil, fmt.Errorf("unsupported jmap command: %v", command)
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
	client := NewJmapClient(wkClient, apiClient)

	jc := JmapContext{AccountId: "123", JmapUrl: "test://"}

	folders, err := client.GetMailboxes(jc, ctx, &logger)
	require.NoError(err)
	require.Len(folders.Folders, 5)

	emails, err := client.EmailQuery(jc, ctx, &logger, "Inbox")
	require.NoError(err)
	require.Len(emails.Emails, 1)

	email := emails.Emails[0]
	require.Equal("eros auctor proin", email.Subject)
	require.Equal(false, email.HasAttachments)
}
