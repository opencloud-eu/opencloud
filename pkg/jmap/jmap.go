package jmap

import (
	"context"
	"fmt"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
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

const (
	logOperation   = "operation"
	logUsername    = "username"
	logAccountId   = "account-id"
	logMailboxId   = "mailbox-id"
	logFetchBodies = "fetch-bodies"
	logOffset      = "offset"
	logLimit       = "limit"
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

func (j *Client) logger(operation string, session *Session, logger *log.Logger) *log.Logger {
	return &log.Logger{Logger: logger.With().Str(logOperation, operation).Str(logUsername, session.Username).Str(logAccountId, session.AccountId).Logger()}
}

func (j *Client) loggerParams(operation string, session *Session, logger *log.Logger, params func(zerolog.Context) zerolog.Context) *log.Logger {
	base := logger.With().Str(logOperation, operation).Str(logUsername, session.Username).Str(logAccountId, session.AccountId)
	return &log.Logger{Logger: params(base).Logger()}
}

func (j *Client) GetIdentity(session *Session, ctx context.Context, logger *log.Logger) (IdentityGetResponse, error) {
	logger = j.logger("GetIdentity", session, logger)
	cmd, err := NewRequest(NewInvocation(IdentityGet, IdentityGetCommand{AccountId: session.AccountId}, "0"))
	if err != nil {
		return IdentityGetResponse{}, err
	}
	return command(j.api, logger, ctx, session, cmd, func(body *Response) (IdentityGetResponse, error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(body, IdentityGet, "0", &response)
		return response, err
	})
}

func (j *Client) GetVacation(session *Session, ctx context.Context, logger *log.Logger) (VacationResponseGetResponse, error) {
	logger = j.logger("GetVacation", session, logger)
	cmd, err := NewRequest(NewInvocation(VacationResponseGet, VacationResponseGetCommand{AccountId: session.AccountId}, "0"))
	if err != nil {
		return VacationResponseGetResponse{}, err
	}
	return command(j.api, logger, ctx, session, cmd, func(body *Response) (VacationResponseGetResponse, error) {
		var response VacationResponseGetResponse
		err = retrieveResponseMatchParameters(body, VacationResponseGet, "0", &response)
		return response, err
	})
}

func (j *Client) GetMailboxes(session *Session, ctx context.Context, logger *log.Logger) (MailboxGetResponse, error) {
	logger = j.logger("GetMailboxes", session, logger)
	cmd, err := NewRequest(NewInvocation(MailboxGet, MailboxGetCommand{AccountId: session.AccountId}, "0"))
	if err != nil {
		return MailboxGetResponse{}, err
	}
	return command(j.api, logger, ctx, session, cmd, func(body *Response) (MailboxGetResponse, error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, MailboxGet, "0", &response)
		return response, err
	})
}

type Emails struct {
	Emails []Email
	State  string
}

func (j *Client) GetEmails(session *Session, ctx context.Context, logger *log.Logger, mailboxId string, offset int, limit int, fetchBodies bool, maxBodyValueBytes int) (Emails, error) {
	logger = j.loggerParams("GetEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Int(logOffset, offset).Int(logLimit, limit)
	})
	cmd, err := NewRequest(
		NewInvocation(EmailQuery, EmailQueryCommand{
			AccountId:       session.AccountId,
			Filter:          &Filter{InMailbox: mailboxId},
			Sort:            []Sort{{Property: "receivedAt", IsAscending: false}},
			CollapseThreads: true,
			Position:        offset,
			Limit:           limit,
			CalculateTotal:  false,
		}, "0"),
		NewInvocation(EmailGet, EmailGetCommand{
			AccountId:          session.AccountId,
			FetchAllBodyValues: fetchBodies,
			MaxBodyValueBytes:  maxBodyValueBytes,
			IdRef:              &Ref{Name: EmailQuery, Path: "/ids/*", ResultOf: "0"},
		}, "1"),
	)
	if err != nil {
		return Emails{}, err
	}

	return command(j.api, logger, ctx, session, cmd, func(body *Response) (Emails, error) {
		var response EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "1", &response)
		if err != nil {
			return Emails{}, err
		}
		return Emails{Emails: response.List, State: body.SessionState}, nil
	})
}
