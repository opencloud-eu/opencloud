package jmap

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

type SessionEventListener interface {
	OnSessionOutdated(session *Session)
}

type Client struct {
	wellKnown             SessionClient
	api                   ApiClient
	sessionEventListeners *eventListeners[SessionEventListener]
	io.Closer
}

func (j *Client) Close() error {
	return j.api.Close()
}

func NewClient(wellKnown SessionClient, api ApiClient) Client {
	return Client{
		wellKnown:             wellKnown,
		api:                   api,
		sessionEventListeners: newEventListeners[SessionEventListener](),
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
	// The base URL to use for JMAP operations towards Stalwart
	JmapUrl url.URL
	// TODO
	DefaultMailAccountId string

	SessionResponse
}

func (s *Session) MailAccountId(accountId string) string {
	if accountId != "" && accountId != defaultAccountId {
		return accountId
	}
	// TODO(pbleser-oc) handle case where there is no default mail account
	return s.DefaultMailAccountId
}

const (
	logOperation    = "operation"
	logUsername     = "username"
	logAccountId    = "account-id"
	logMailboxId    = "mailbox-id"
	logFetchBodies  = "fetch-bodies"
	logOffset       = "offset"
	logLimit        = "limit"
	logApiUrl       = "apiurl"
	logSessionState = "session-state"

	defaultAccountId = "*"

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
	return log.Logger{Logger: l.With().
		Str(logUsername, s.Username).
		Str(logApiUrl, s.ApiUrl).
		Str(logSessionState, s.State).
		Logger()}
}

func (j *Client) AddSessionEventListener(listener SessionEventListener) {
	j.sessionEventListeners.add(listener)
}

func (j *Client) onSessionOutdated(session *Session) {
	j.sessionEventListeners.signal(func(listener SessionEventListener) {
		listener.OnSessionOutdated(session)
	})
}

// Create a new Session from a WellKnownResponse.
func newSession(sessionResponse SessionResponse) (Session, Error) {
	username := sessionResponse.Username
	if username == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide a username")}
	}
	mailAccountId := sessionResponse.PrimaryAccounts.Mail
	if mailAccountId == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide a primary mail account")}
	}
	apiStr := sessionResponse.ApiUrl
	if apiStr == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide an API URL")}
	}
	apiUrl, err := url.Parse(apiStr)
	if err != nil {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response provides an invalid API URL")}
	}
	return Session{
		Username:             username,
		DefaultMailAccountId: mailAccountId,
		JmapUrl:              *apiUrl,
		SessionResponse:      sessionResponse,
	}, nil
}

// Retrieve JMAP well-known data from the Stalwart server and create a Session from that.
func (j *Client) FetchSession(username string, logger *log.Logger) (Session, Error) {
	wk, err := j.wellKnown.GetSession(username, logger)
	if err != nil {
		return Session{}, err
	}
	return newSession(wk)
}

func (j *Client) logger(accountId string, operation string, session *Session, logger *log.Logger) *log.Logger {
	zc := logger.With().Str(logOperation, operation).Str(logUsername, session.Username)
	if accountId != "" {
		zc = zc.Str(logAccountId, accountId)
	}
	return &log.Logger{Logger: zc.Logger()}
}

func (j *Client) loggerParams(accountId string, operation string, session *Session, logger *log.Logger, params func(zerolog.Context) zerolog.Context) *log.Logger {
	zc := logger.With().Str(logOperation, operation).Str(logUsername, session.Username)
	if accountId != "" {
		zc = zc.Str(logAccountId, accountId)
	}
	return &log.Logger{Logger: params(zc).Logger()}
}

// https://jmap.io/spec-mail.html#identityget
func (j *Client) GetIdentity(accountId string, session *Session, ctx context.Context, logger *log.Logger) (IdentityGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetIdentity", session, logger)
	cmd, err := request(invocation(IdentityGet, IdentityGetCommand{AccountId: aid}, "0"))
	if err != nil {
		return IdentityGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (IdentityGetResponse, Error) {
		var response IdentityGetResponse
		err = retrieveResponseMatchParameters(body, IdentityGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

// https://jmap.io/spec-mail.html#vacationresponseget
func (j *Client) GetVacationResponse(accountId string, session *Session, ctx context.Context, logger *log.Logger) (VacationResponseGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetVacationResponse", session, logger)
	cmd, err := request(invocation(VacationResponseGet, VacationResponseGetCommand{AccountId: aid}, "0"))
	if err != nil {
		return VacationResponseGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (VacationResponseGetResponse, Error) {
		var response VacationResponseGetResponse
		err = retrieveResponseMatchParameters(body, VacationResponseGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

// https://jmap.io/spec-mail.html#mailboxget
func (j *Client) GetMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, ids []string) (MailboxGetResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetMailbox", session, logger)
	cmd, err := request(invocation(MailboxGet, MailboxGetCommand{AccountId: aid, Ids: ids}, "0"))
	if err != nil {
		return MailboxGetResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxGetResponse, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, MailboxGet, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

func (j *Client) GetAllMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger) (MailboxGetResponse, Error) {
	return j.GetMailbox(accountId, session, ctx, logger, nil)
}

// https://jmap.io/spec-mail.html#mailboxquery
func (j *Client) QueryMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, filter MailboxFilterCondition) (MailboxQueryResponse, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "QueryMailbox", session, logger)
	cmd, err := request(invocation(MailboxQuery, SimpleMailboxQueryCommand{AccountId: aid, Filter: filter}, "0"))
	if err != nil {
		return MailboxQueryResponse{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxQueryResponse, Error) {
		var response MailboxQueryResponse
		err = retrieveResponseMatchParameters(body, MailboxQuery, "0", &response)
		return response, simpleError(err, JmapErrorInvalidJmapResponsePayload)
	})
}

type Mailboxes struct {
	Mailboxes []Mailbox `json:"mailboxes,omitempty"`
	State     string    `json:"state,omitempty"`
}

func (j *Client) SearchMailboxes(accountId string, session *Session, ctx context.Context, logger *log.Logger, filter MailboxFilterCondition) (Mailboxes, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "SearchMailboxes", session, logger)

	cmd, err := request(
		invocation(MailboxQuery, SimpleMailboxQueryCommand{AccountId: aid, Filter: filter}, "0"),
		invocation(MailboxGet, MailboxGetRefCommand{
			AccountId: aid,
			IdRef:     &Ref{Name: MailboxQuery, Path: "/ids/*", ResultOf: "0"},
		}, "1"),
	)
	if err != nil {
		return Mailboxes{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Mailboxes, Error) {
		var response MailboxGetResponse
		err = retrieveResponseMatchParameters(body, MailboxGet, "1", &response)
		if err != nil {
			return Mailboxes{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}
		return Mailboxes{Mailboxes: response.List, State: body.SessionState}, nil
	})
}

type Emails struct {
	Emails []Email `json:"emails,omitempty"`
	State  string  `json:"state,omitempty"`
}

func (j *Client) GetEmails(accountId string, session *Session, ctx context.Context, logger *log.Logger, ids []string, fetchBodies bool, maxBodyValueBytes int) (Emails, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.logger(aid, "GetEmails", session, logger)

	get := EmailGetCommand{AccountId: aid, Ids: ids, FetchAllBodyValues: fetchBodies}
	if maxBodyValueBytes >= 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := request(invocation(EmailGet, get, "0"))
	if err != nil {
		return Emails{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Emails, Error) {
		var response EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "0", &response)
		if err != nil {
			return Emails{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}
		return Emails{Emails: response.List, State: body.SessionState}, nil
	})
}

func (j *Client) GetAllEmails(accountId string, session *Session, ctx context.Context, logger *log.Logger, mailboxId string, offset int, limit int, fetchBodies bool, maxBodyValueBytes int) (Emails, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.loggerParams(aid, "GetAllEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Int(logOffset, offset).Int(logLimit, limit)
	})

	query := EmailQueryCommand{
		AccountId:       aid,
		Filter:          &MessageFilter{InMailbox: mailboxId},
		Sort:            []Sort{{Property: emailSortByReceivedAt, IsAscending: false}},
		CollapseThreads: true,
		CalculateTotal:  false,
	}
	if offset >= 0 {
		query.Position = offset
	}
	if limit >= 0 {
		query.Limit = limit
	}

	get := EmailGetRefCommand{
		AccountId:          aid,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &Ref{Name: EmailQuery, Path: "/ids/*", ResultOf: "0"},
	}
	if maxBodyValueBytes >= 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := request(
		invocation(EmailQuery, query, "0"),
		invocation(EmailGet, get, "1"),
	)
	if err != nil {
		return Emails{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Emails, Error) {
		var response EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "1", &response)
		if err != nil {
			return Emails{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}
		return Emails{Emails: response.List, State: body.SessionState}, nil
	})
}
