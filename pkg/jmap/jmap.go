package jmap

import (
	"context"
	"fmt"
	"io"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

type Client struct {
	wellKnown WellKnownClient
	api       ApiClient
	io.Closer
}

func (j *Client) Close() error {
	return j.api.Close()
}

func NewClient(wellKnown WellKnownClient, api ApiClient) Client {
	return Client{
		wellKnown: wellKnown,
		api:       api,
	}
}

// Cached user related information
//
// This information is typically retrieved once (or at least for a certain period of time) from the
// JMAP well-known endpoint of Stalwart and then kept in cache to avoid the performance cost of
// retrieving it over and over again.
//
// This is really only needed due to the Graph API limitations, since ideally, the account ID should
// be passed as a request parameter by the UI, in order to support a user having multiple accounts.
//
// Keeping track of the JMAP URL might be useful though, in case of Stalwart sharding strategies making
// use of that, by providing different URLs for JMAP on a per-user basis, and that is not something
// we would want to query before every single JMAP request. On the other hand, that then also creates
// a risk of going out-of-sync, e.g. if a node is down and the user is reassigned to a different node.
// There might be webhooks to subscribe to in Stalwart to be notified of such situations, in which case
// the Session needs to be removed from the cache.
//
// The Username is only here for convenience, it could just as well be passed as a separate parameter
// instead of being part of the Session, since the username is always part of the request (typically in
// the authentication token payload.)
type Session struct {
	// The name of the user to use to authenticate against Stalwart
	Username string
	// The identifier of the account to use when performing JMAP operations with Stalwart
	AccountId string
	// The base URL to use for JMAP operations towards Stalwart
	JmapUrl string
}

const (
	logOperation   = "operation"
	logUsername    = "username"
	logAccountId   = "account-id"
	logMailboxId   = "mailbox-id"
	logFetchBodies = "fetch-bodies"
	logOffset      = "offset"
	logLimit       = "limit"

	emailSortByReceivedAt              = "receivedAt"
	emailSortBySize                    = "size"
	emailSortByFrom                    = "from"
	emailSortByTo                      = "to"
	emailSortBySubject                 = "subject"
	emailSortBySentAt                  = "sentAt"
	emailSortByHasKeyword              = "hasKeyword"
	emailSortByAllInThreadHaveKeyword  = "allInThreadHaveKeyword"
	emailSortBySomeInThreadHaveKeyword = "someInThreadHaveKeyword"
)

// Create a new log.Logger that is decorated with fields containing information about the Session.
func (s Session) DecorateLogger(l log.Logger) log.Logger {
	return log.Logger{
		Logger: l.With().Str(logUsername, s.Username).Str(logAccountId, s.AccountId).Logger(),
	}
}

var (
	errWellKnownResponseHasNoUsername             = fmt.Errorf("well-known response has no username")
	errWellKnownResponseHasJmapMailPrimaryAccount = fmt.Errorf("PrimaryAccounts in well-known response has no entry for %v", JmapMail)
	errWellKnownResponseHasNoApiUrl               = fmt.Errorf("well-known response has no API URL")
)

// Create a new Session from a WellKnownResponse.
func NewSession(wellKnownResponse WellKnownResponse) (Session, error) {
	username := wellKnownResponse.Username
	if username == "" {
		return Session{}, errWellKnownResponseHasNoUsername
	}
	accountId := wellKnownResponse.PrimaryAccounts[JmapMail]
	if accountId == "" {
		return Session{}, errWellKnownResponseHasJmapMailPrimaryAccount
	}
	apiUrl := wellKnownResponse.ApiUrl
	if apiUrl == "" {
		return Session{}, errWellKnownResponseHasNoApiUrl
	}
	return Session{
		Username:  username,
		AccountId: accountId,
		JmapUrl:   apiUrl,
	}, nil
}

// Retrieve JMAP well-known data from the Stalwart server and create a Session from that.
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

// https://jmap.io/spec-mail.html#identityget
func (j *Client) GetIdentity(session *Session, ctx context.Context, logger *log.Logger) (IdentityGetResponse, error) {
	logger = j.logger("GetIdentity", session, logger)
	cmd, err := request(invocation(IdentityGet, IdentityGetCommand{AccountId: session.AccountId}, "0"))
	if err != nil {
		return IdentityGetResponse{}, err
	}
	return command(j.api, logger, ctx, session, cmd, func(body *Response) (IdentityGetResponse, error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(body, IdentityGet, "0", &response)
		return response, err
	})
}

// https://jmap.io/spec-mail.html#vacationresponseget
func (j *Client) GetVacationResponse(session *Session, ctx context.Context, logger *log.Logger) (VacationResponseGetResponse, error) {
	logger = j.logger("GetVacationResponse", session, logger)
	cmd, err := request(invocation(VacationResponseGet, VacationResponseGetCommand{AccountId: session.AccountId}, "0"))
	if err != nil {
		return VacationResponseGetResponse{}, err
	}
	return command(j.api, logger, ctx, session, cmd, func(body *Response) (VacationResponseGetResponse, error) {
		var response VacationResponseGetResponse
		err = retrieveResponseMatchParameters(body, VacationResponseGet, "0", &response)
		return response, err
	})
}

// https://jmap.io/spec-mail.html#mailboxget
func (j *Client) GetMailbox(session *Session, ctx context.Context, logger *log.Logger, ids []string) (MailboxGetResponse, error) {
	logger = j.logger("GetMailbox", session, logger)
	cmd, err := request(invocation(MailboxGet, MailboxGetCommand{AccountId: session.AccountId, Ids: ids}, "0"))
	if err != nil {
		return MailboxGetResponse{}, err
	}
	return command(j.api, logger, ctx, session, cmd, func(body *Response) (MailboxGetResponse, error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, MailboxGet, "0", &response)
		return response, err
	})
}

func (j *Client) GetAllMailboxes(session *Session, ctx context.Context, logger *log.Logger) (MailboxGetResponse, error) {
	return j.GetMailbox(session, ctx, logger, nil)
}

type Emails struct {
	Emails []Email
	State  string
}

func (j *Client) GetEmails(session *Session, ctx context.Context, logger *log.Logger, mailboxId string, offset int, limit int, fetchBodies bool, maxBodyValueBytes int) (Emails, error) {
	logger = j.loggerParams("GetEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Int(logOffset, offset).Int(logLimit, limit)
	})
	cmd, err := request(
		invocation(EmailQuery, EmailQueryCommand{
			AccountId:       session.AccountId,
			Filter:          &Filter{InMailbox: mailboxId},
			Sort:            []Sort{{Property: emailSortByReceivedAt, IsAscending: false}},
			CollapseThreads: true,
			Position:        offset,
			Limit:           limit,
			CalculateTotal:  false,
		}, "0"),
		invocation(EmailGet, EmailGetCommand{
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
