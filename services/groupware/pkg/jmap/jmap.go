package jmap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

const (
	JmapCore = "urn:ietf:params:jmap:core"
	JmapMail = "urn:ietf:params:jmap:mail"
)

type JmapClient struct {
	wellKnown JmapWellKnownClient
	api       JmapApiClient
}

func NewJmapClient(wellKnown JmapWellKnownClient, api JmapApiClient) JmapClient {
	return JmapClient{
		wellKnown: wellKnown,
		api:       api,
	}
}

type JmapContext struct {
	AccountId string
	JmapUrl   string
}

func NewJmapContext(wellKnown WellKnownJmap) (JmapContext, error) {
	// TODO validate
	return JmapContext{
		AccountId: wellKnown.PrimaryAccounts[JmapMail],
		JmapUrl:   wellKnown.ApiUrl,
	}, nil
}

func (j *JmapClient) FetchJmapContext(username string, logger *log.Logger) (JmapContext, error) {
	wk, err := j.wellKnown.GetWellKnown(username, logger)
	if err != nil {
		return JmapContext{}, err
	}
	return NewJmapContext(wk)
}

type ContextKey int

const (
	ContextAccountId ContextKey = iota
	ContextOperationId
)

func (j *JmapClient) validate(jmapContext JmapContext) error {
	if jmapContext.AccountId == "" {
		return fmt.Errorf("AccountId not set")
	}
	return nil
}

func (j *JmapClient) GetMailboxes(jc JmapContext, ctx context.Context, logger *log.Logger) (JmapFolders, error) {
	if err := j.validate(jc); err != nil {
		return JmapFolders{}, err
	}

	logger.Info().Str("command", "Mailbox/get").Str("accountId", jc.AccountId).Msg("GetMailboxes")
	cmd := simpleCommand("Mailbox/get", map[string]any{"accountId": jc.AccountId})
	commandCtx := context.WithValue(ctx, ContextOperationId, "GetMailboxes")
	return command(j.api, logger, commandCtx, &cmd, func(body *[]byte) (JmapFolders, error) {
		var data JmapCommandResponse
		err := json.Unmarshal(*body, &data)
		if err != nil {
			logger.Error().Err(err).Msg("failed to deserialize body JSON payload")
			var zero JmapFolders
			return zero, err
		}
		return parseMailboxGetResponse(data)
	})
}

func (j *JmapClient) EmailQuery(jc JmapContext, ctx context.Context, logger *log.Logger, mailboxId string) (Emails, error) {
	if err := j.validate(jc); err != nil {
		return Emails{}, err
	}

	cmd := make([][]any, 4)
	cmd[0] = []any{
		"Email/query",
		map[string]any{
			"accountId": jc.AccountId,
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
			"accountId": jc.AccountId,
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
			"accountId": jc.AccountId,
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
			"accountId": jc.AccountId,
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

	commandCtx := context.WithValue(ctx, ContextOperationId, "EmailQuery")
	return command(j.api, logger, commandCtx, &cmd, func(body *[]byte) (Emails, error) {
		var data JmapCommandResponse
		err := json.Unmarshal(*body, &data)
		if err != nil {
			logger.Error().Err(err).Msg("failed to unmarshal response payload")
			return Emails{}, err
		}
		first := retrieveResponseMatch(&data, 3, "Email/get", "3")
		if first == nil {
			return Emails{Emails: []Email{}, State: data.SessionState}, nil
		}
		if len(first) != 3 {
			return Emails{}, fmt.Errorf("wrong Email/get response payload size, expecting a length of 3 but it is %v", len(first))
		}

		payload := first[1].(map[string]any)
		list, listExists := payload["list"].([]any)
		if !listExists {
			return Emails{}, fmt.Errorf("wrong Email/get response payload size, expecting a length of 3 but it is %v", len(first))
		}

		emails := make([]Email, 0, len(list))
		for _, elem := range list {
			email, err := mapEmail(elem.(map[string]any))
			if err != nil {
				return Emails{}, err
			}
			emails = append(emails, email)
		}
		return Emails{Emails: emails, State: data.SessionState}, nil
	})
}
