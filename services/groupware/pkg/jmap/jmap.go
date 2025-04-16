package jmap

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http"
)

type WellKnownJmap struct {
	ApiUrl          string            `json:"apiUrl"`
	PrimaryAccounts map[string]string `json:"primaryAccounts"`
}

/*
func bearer(req *http.Request, token string) {
	req.Header.Add("Authorization", "Bearer "+base64.StdEncoding.EncodeToString([]byte(token)))
}
*/

func fetch[T any](client *http.Client, url string, username string, password string, mapper func(body *[]byte) T) T {
	req, reqErr := http.NewRequest(http.MethodGet, url, nil)
	if reqErr != nil {
		panic(reqErr)
	}
	req.SetBasicAuth(username, password)

	res, getErr := client.Do(req)
	if getErr != nil {
		panic(getErr)
	}
	if res.StatusCode != 200 {
		panic(fmt.Sprintf("HTTP status code not 200: %d", res.StatusCode))
	}
	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(res.Body)
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	return mapper(&body)
}

func simpleCommand(cmd string, params map[string]any) [][]any {
	jmap := make([][]any, 1)
	jmap[0] = make([]any, 3)
	jmap[0][0] = cmd
	jmap[0][1] = params
	jmap[0][2] = "0"
	return jmap
}

const (
	JmapCore = "urn:ietf:params:jmap:core"
	JmapMail = "urn:ietf:params:jmap:mail"
)

func command[T any](client *http.Client, ctx context.Context, url string, username string, password string, methodCalls *[][]any, mapper func(body *[]byte) T) T {
	jmapWrapper := map[string]any{
		"using":       []string{JmapCore, JmapMail},
		"methodCalls": methodCalls,
	}

	/*
		{
		"using":[
		  "urn:ietf:params:jmap:core",
		  "urn:ietf:params:jmap:mail"
		],
		"methodCalls":[
		  [
		    "Identity/get", {
		      "accountId": "cp"
		    }, "0"
		  ]
		]
		}
	*/

	bodyBytes, marshalErr := json.Marshal(jmapWrapper)
	if marshalErr != nil {
		panic(marshalErr)
	}

	req, reqErr := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(bodyBytes))
	if reqErr != nil {
		panic(reqErr)
	}
	req.SetBasicAuth(username, password)
	req.Header.Add("Content-Type", "application/json")

	slog.Info("jmap", "url", url, "username", username)
	res, postErr := client.Do(req)
	if postErr != nil {
		panic(postErr)
	}
	if res.StatusCode != 200 {
		panic(fmt.Sprintf("HTTP status code not 200: %d", res.StatusCode))
	}
	if res.Body != nil {
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(res.Body)
	}

	body, readErr := io.ReadAll(res.Body)
	if readErr != nil {
		log.Fatal(readErr)
	}

	if slog.Default().Enabled(ctx, slog.LevelDebug) {
		slog.Debug(ctx.Value("operation").(string) + " response: " + string(body))
	}

	return mapper(&body)
}

type JmapFolder struct {
	Id            string
	Name          string
	Role          string
	TotalEmails   int
	UnreadEmails  int
	TotalThreads  int
	UnreadThreads int
}
type JmapFolders struct {
	Folders []JmapFolder
	state   string
}

type JmapCommandResponse struct {
	MethodResponses [][]any `json:"methodResponses"`
	SessionState    string  `json:"sessionState"`
}

type JmapClient struct {
	client    *http.Client
	username  string
	password  string
	url       string
	accountId string
	ctx       context.Context
}

func New(client *http.Client, ctx context.Context, username string, password string, url string, accountId string) JmapClient {
	return JmapClient{
		client:    client,
		ctx:       ctx,
		username:  username,
		password:  password,
		url:       url,
		accountId: accountId,
	}
}

func (jmap *JmapClient) FetchWellKnown() WellKnownJmap {
	return fetch(jmap.client, jmap.url+"/.well-known/jmap", jmap.username, jmap.password, func(body *[]byte) WellKnownJmap {
		var data WellKnownJmap
		jsonErr := json.Unmarshal(*body, &data)
		if jsonErr != nil {
			panic(jsonErr)
		}

		/*
			u, urlErr := url.Parse(data.ApiUrl)
			if urlErr != nil {
				panic(urlErr)
			}
			jmap.url = jmap.url + u.Path
		*/
		jmap.accountId = data.PrimaryAccounts[JmapMail]
		return data
	})
}

func (jmap *JmapClient) GetMailboxes() JmapFolders {
	/*
		{"methodResponses":
		[["Mailbox/get",
		{"accountId":"cs","state":"n","list":
		[{"id":"a","name":"Inbox","parentId":null,"role":"inbox","sortOrder":0,"isSubscribed":true,"totalEmails":0,"unreadEmails":0,"totalThreads":0,"unreadThreads":0,"myRights":{"mayReadItems":true,"mayAddItems":true,"mayRemoveItems":true,"maySetSeen":true,"maySetKeywords":true,"mayCreateChild":true,"mayRename":true,"mayDelete":true,"maySubmit":true}},{"id":"b","name":"Deleted Items","parentId":null,"role":"trash","sortOrder":0,"isSubscribed":true,"totalEmails":0,"unreadEmails":0,"totalThreads":0,"unreadThreads":0,"myRights":{"mayReadItems":true,"mayAddItems":true,"mayRemoveItems":true,"maySetSeen":true,"maySetKeywords":true,"mayCreateChild":true,"mayRename":true,"mayDelete":true,"maySubmit":true}},{"id":"c","name":"Junk Mail","parentId":null,"role":"junk","sortOrder":0,"isSubscribed":true,"totalEmails":0,"unreadEmails":0,"totalThreads":0,"unreadThreads":0,"myRights":{"mayReadItems":true,"mayAddItems":true,"mayRemoveItems":true,"maySetSeen":true,"maySetKeywords":true,"mayCreateChild":true,"mayRename":true,"mayDelete":true,"maySubmit":true}},{"id":"d","name":"Drafts","parentId":null,"role":"drafts","sortOrder":0,"isSubscribed":true,"totalEmails":0,"unreadEmails":0,"totalThreads":0,"unreadThreads":0,"myRights":{"mayReadItems":true,"mayAddItems":true,"mayRemoveItems":true,"maySetSeen":true,"maySetKeywords":true,"mayCreateChild":true,"mayRename":true,"mayDelete":true,"maySubmit":true}},{"id":"e","name":"Sent Items","parentId":null,"role":"sent","sortOrder":0,"isSubscribed":true,"totalEmails":0,"unreadEmails":0,"totalThreads":0,"unreadThreads":0,"myRights":{"mayReadItems":true,"mayAddItems":true,"mayRemoveItems":true,"maySetSeen":true,"maySetKeywords":true,"mayCreateChild":true,"mayRename":true,"mayDelete":true,"maySubmit":true}}],"notFound":[]},"0"]],"sessionState":"3e25b2a0"}

	*/
	cmd := simpleCommand("Mailbox/get", map[string]any{"accountId": jmap.accountId})
	commandCtx := context.WithValue(jmap.ctx, "operation", "GetMailboxes")
	return command(jmap.client, commandCtx, jmap.url, jmap.username, jmap.password, &cmd, func(body *[]byte) JmapFolders {
		var data JmapCommandResponse
		jsonErr := json.Unmarshal(*body, &data)
		if jsonErr != nil {
			panic(jsonErr)
		}
		first := data.MethodResponses[0]
		params := first[1]
		payload := params.(map[string]any)
		state := payload["state"].(string)
		list := payload["list"].([]any)
		folders := make([]JmapFolder, len(list))
		for i, a := range list {
			item := a.(map[string]any)
			folders[i] = JmapFolder{
				Id:            item["id"].(string),
				Name:          item["name"].(string),
				Role:          item["role"].(string),
				TotalEmails:   int(item["totalEmails"].(float64)),
				UnreadEmails:  int(item["unreadEmails"].(float64)),
				TotalThreads:  int(item["totalThreads"].(float64)),
				UnreadThreads: int(item["unreadThreads"].(float64)),
			}
		}
		return JmapFolders{Folders: folders, state: state}
	})
}

type Emails struct {
	Emails []Email
	State  string
}

func (jmap *JmapClient) EmailQuery(mailboxId string) Emails {
	cmd := make([][]any, 4)
	cmd[0] = []any{
		"Email/query",
		map[string]any{
			"accountId": jmap.accountId,
			"filter": map[string]any{
				"inMailbox": mailboxId,
			},
			"sort": []map[string]any{
				{
					"isAscending": false,
					"property":    "receivedAt",
				},
			},
			"collapseThreads": true,
			"position":        0,
			"limit":           30,
			"calculateTotal":  true,
		},
		"0",
	}
	cmd[1] = []any{
		"Email/get",
		map[string]any{
			"accountId": jmap.accountId,
			"#ids": map[string]any{
				"resultOf": "0",
				"name":     "Email/query",
				"path":     "/ids",
			},
			"properties": []string{"threadId"},
		},
		"1",
	}
	cmd[2] = []any{
		"Thread/get",
		map[string]any{
			"accountId": jmap.accountId,
			"#ids": map[string]any{
				"resultOf": "1",
				"name":     "Email/get",
				"path":     "/list/*/threadId",
			},
		},
		"2",
	}
	cmd[3] = []any{
		"Email/get",
		map[string]any{
			"accountId": jmap.accountId,
			"#ids": map[string]any{
				"resultOf": "2",
				"name":     "Thread/get",
				"path":     "/list/*/emailIds",
			},
			"properties": []string{
				"threadId",
				"mailboxIds",
				"keywords",
				"hasAttachment",
				"from",
				"subject",
				"receivedAt",
				"size",
				"preview",
			},
		},
		"3",
	}

	commandCtx := context.WithValue(jmap.ctx, "operation", "GetMailboxes")
	return command(jmap.client, commandCtx, jmap.url, jmap.username, jmap.password, &cmd, func(body *[]byte) Emails {
		var data JmapCommandResponse
		jsonErr := json.Unmarshal(*body, &data)
		if jsonErr != nil {
			panic(jsonErr)
		}
		matches := make([][]any, 1)
		for _, elem := range data.MethodResponses {
			if elem[0] == "Email/get" && elem[2] == "3" {
				matches = append(matches, elem)
			}
		}
		/*
			matches := lo.Filter(data.MethodResponses, func(elem []any, index int) bool {
				return elem[0] == "Email/get" && elem[2] == "3"
			})
		*/
		payload := matches[0][1].(map[string]any)
		list := payload["list"].([]any)

		/*
			{
			            "threadId": "cc",
			            "mailboxIds": {
			              "a": true
			            },
			            "keywords": {},
			            "hasAttachment": false,
			            "from": [
			              {
			                "name": null,
			                "email": "root@nsa.gov"
			              }
			            ],
			            "subject": "Hello 5",
			            "receivedAt": "2025-04-10T13:07:27Z",
			            "size": 47,
			            "preview": "Hi <3",
			            "id": "iiaaaaaa"
			          },
		*/

		emails := make([]Email, len(list))
		for i, elem := range list {
			emails[i] = NewEmail(elem.(map[string]any))
		}
		/*
			emails := lo.Map(list, func(elem any, _ int) Email {
				return NewEmail(elem.(map[string]any))
			})
		*/
		return Emails{Emails: emails, State: data.SessionState}
	})
}
