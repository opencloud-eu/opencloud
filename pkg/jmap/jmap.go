package jmap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type Client struct {
	wellKnown WellKnownClient
	api       ApiClient
}

func NewClient(wellKnown WellKnownClient, api ApiClient) Client {
	return Client{
		wellKnown: wellKnown,
		api:       api,
	}
}

type Session struct {
	Username  string
	AccountId string
	JmapUrl   string
}

type ContextKey int

const (
	ContextAccountId ContextKey = iota
	ContextOperationId
	ContextUsername
)

func (s Session) DecorateSession(ctx context.Context) context.Context {
	ctx = context.WithValue(ctx, ContextUsername, s.Username)
	ctx = context.WithValue(ctx, ContextAccountId, s.AccountId)
	return ctx
}

const (
	logUsername  = "username"
	logAccountId = "account-id"
)

func (s Session) DecorateLogger(l log.Logger) log.Logger {
	return log.Logger{
		Logger: l.With().Str(logUsername, s.Username).Str(logAccountId, s.AccountId).Logger(),
	}
}

func NewSession(wellKnownResponse WellKnownResponse) (Session, error) {
	username := wellKnownResponse.Username
	if username == "" {
		return Session{}, fmt.Errorf("well-known response has no username")
	}
	accountId := wellKnownResponse.PrimaryAccounts[JmapMail]
	if accountId == "" {
		return Session{}, fmt.Errorf("PrimaryAccounts in well-known response has no entry for %v", JmapMail)
	}
	apiUrl := wellKnownResponse.ApiUrl
	if apiUrl == "" {
		return Session{}, fmt.Errorf("well-known response has no API URL")
	}
	return Session{
		Username:  username,
		AccountId: accountId,
		JmapUrl:   apiUrl,
	}, nil
}

func (j *Client) FetchSession(username string, logger *log.Logger) (Session, error) {
	wk, err := j.wellKnown.GetWellKnown(username, logger)
	if err != nil {
		return Session{}, err
	}
	return NewSession(wk)
}

func (j *Client) GetMailboxes(session Session, ctx context.Context, logger *log.Logger) (Folders, error) {
	logger.Info().Str("command", "Mailbox/get").Str("accountId", session.AccountId).Msg("GetMailboxes")
	cmd := simpleCommand("Mailbox/get", map[string]any{"accountId": session.AccountId})
	commandCtx := context.WithValue(ctx, ContextOperationId, "GetMailboxes")
	return command(j.api, logger, commandCtx, &cmd, func(body *[]byte) (Folders, error) {
		var data JmapCommandResponse
		err := json.Unmarshal(*body, &data)
		if err != nil {
			logger.Error().Err(err).Msg("failed to deserialize body JSON payload")
			var zero Folders
			return zero, err
		}
		return parseMailboxGetResponse(data)
	})
}

func (j *Client) GetEmails(session Session, ctx context.Context, logger *log.Logger, mailboxId string, offset int, limit int, fetchBodies bool, maxBodyValueBytes int) (Emails, error) {
	cmd := make([][]any, 2)
	cmd[0] = []any{
		"Email/query",
		map[string]any{
			"accountId": session.AccountId,
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
			"position":        offset,
			"limit":           limit,
			"calculateTotal":  true,
		},
		"0",
	}
	cmd[1] = []any{
		"Email/get",
		map[string]any{
			"accountId":          session.AccountId,
			"fetchAllBodyValues": fetchBodies,
			"maxBodyValueBytes":  maxBodyValueBytes,
			"#ids": map[string]any{
				"name":     "Email/query",
				"path":     "/ids/*",
				"resultOf": "0",
			},
		},
		"1",
	}
	commandCtx := context.WithValue(ctx, ContextOperationId, "GetEmails")

	logger = &log.Logger{Logger: logger.With().Str("mailboxId", mailboxId).Bool("fetchBodies", fetchBodies).Int("offset", offset).Int("limit", limit).Logger()}

	return command(j.api, logger, commandCtx, &cmd, func(body *[]byte) (Emails, error) {
		var data JmapCommandResponse
		err := json.Unmarshal(*body, &data)
		if err != nil {
			logger.Error().Err(err).Msg("failed to unmarshal response payload")
			return Emails{}, err
		}
		first := retrieveResponseMatch(&data, 3, "Email/get", "1")
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
			email, err := mapEmail(elem.(map[string]any), fetchBodies, logger)
			if err != nil {
				return Emails{}, err
			}
			emails = append(emails, email)
		}
		return Emails{Emails: emails, State: data.SessionState}, nil
	})
}

func (j *Client) EmailThreadsQuery(session Session, ctx context.Context, logger *log.Logger, mailboxId string) (Emails, error) {
	cmd := make([][]any, 4)
	cmd[0] = []any{
		"Email/query",
		map[string]any{
			"accountId": session.AccountId,
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
			"accountId": session.AccountId,
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
			"accountId": session.AccountId,
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
			"accountId": session.AccountId,
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

	commandCtx := context.WithValue(ctx, ContextOperationId, "EmailThreadsQuery")
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
			email, err := mapEmail(elem.(map[string]any), false, logger)
			if err != nil {
				return Emails{}, err
			}
			emails = append(emails, email)
		}
		return Emails{Emails: emails, State: data.SessionState}, nil
	})
}
