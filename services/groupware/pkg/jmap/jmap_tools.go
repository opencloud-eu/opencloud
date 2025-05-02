package jmap

import (
	"context"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

func command[T any](api JmapApiClient,
	logger *log.Logger,
	ctx context.Context,
	methodCalls *[][]any,
	mapper func(body *[]byte) (T, error)) (T, error) {
	body := map[string]any{
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

	responseBody, err := api.Command(ctx, logger, body)
	if err != nil {
		var zero T
		return zero, err
	}
	return mapper(&responseBody)
}

func simpleCommand(cmd string, params map[string]any) [][]any {
	jmap := make([][]any, 1)
	jmap[0] = make([]any, 3)
	jmap[0][0] = cmd
	jmap[0][1] = params
	jmap[0][2] = "0"
	return jmap
}

func mapFolder(item map[string]any) JmapFolder {
	return JmapFolder{
		Id:            item["id"].(string),
		Name:          item["name"].(string),
		Role:          item["role"].(string),
		TotalEmails:   int(item["totalEmails"].(float64)),
		UnreadEmails:  int(item["unreadEmails"].(float64)),
		TotalThreads:  int(item["totalThreads"].(float64)),
		UnreadThreads: int(item["unreadThreads"].(float64)),
	}
}

func parseMailboxGetResponse(data JmapCommandResponse) (JmapFolders, error) {
	first := data.MethodResponses[0]
	params := first[1]
	payload := params.(map[string]any)
	state := payload["state"].(string)
	list := payload["list"].([]any)
	folders := make([]JmapFolder, 0, len(list))
	for _, a := range list {
		item := a.(map[string]any)
		folder := mapFolder(item)
		folders = append(folders, folder)
	}
	return JmapFolders{Folders: folders, state: state}, nil
}

func mapEmail(elem map[string]any) (Email, error) {
	fromList := elem["from"].([]any)
	from := fromList[0].(map[string]any)
	var subject string
	var value any = elem["subject"]
	if value != nil {
		subject = value.(string)
	} else {
		subject = ""
	}
	var hasAttachments bool
	hasAttachmentsAny := elem["hasAttachments"]
	if hasAttachmentsAny != nil {
		hasAttachments = hasAttachmentsAny.(bool)
	} else {
		hasAttachments = false
	}

	received, err := time.ParseInLocation(time.RFC3339, elem["receivedAt"].(string), time.UTC)
	if err != nil {
		return Email{}, err
	}

	return Email{
		From:           from["email"].(string),
		Subject:        subject,
		HasAttachments: hasAttachments,
		Received:       received,
	}, nil
}

func retrieveResponseMatch(data *JmapCommandResponse, length int, operation string, tag string) []any {
	for _, elem := range data.MethodResponses {
		if len(elem) == length && elem[0] == operation && elem[2] == tag {
			return elem
		}
	}
	return nil
}
