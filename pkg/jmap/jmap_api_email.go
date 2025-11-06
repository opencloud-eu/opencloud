package jmap

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

type Emails struct {
	Emails []Email `json:"emails,omitempty"`
	Total  uint    `json:"total,omitzero"`
	Limit  uint    `json:"limit,omitzero"`
	Offset uint    `json:"offset,omitzero"`
}

// Retrieve specific Emails by their id.
func (j *Client) GetEmails(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, ids []string, fetchBodies bool, maxBodyValueBytes uint, markAsSeen bool, withThreads bool) ([]Email, SessionState, State, Language, Error) {
	logger = j.logger("GetEmails", session, logger)

	get := EmailGetCommand{AccountId: accountId, Ids: ids, FetchAllBodyValues: fetchBodies}
	if maxBodyValueBytes > 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}
	invokeGet := invocation(CommandEmailGet, get, "1")

	methodCalls := []Invocation{invokeGet}
	if markAsSeen {
		updates := make(map[string]EmailUpdate, len(ids))
		for _, id := range ids {
			updates[id] = EmailUpdate{EmailPropertyKeywords + "/" + JmapKeywordSeen: true}
		}
		mark := EmailSetCommand{AccountId: accountId, Update: updates}
		methodCalls = []Invocation{invocation(CommandEmailSet, mark, "0"), invokeGet}
	}
	if withThreads {
		threads := ThreadGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: "1",
				Name:     CommandEmailGet,
				Path:     "/list/*/" + EmailPropertyThreadId,
			},
		}
		methodCalls = append(methodCalls, invocation(CommandThreadGet, threads, "2"))
	}

	cmd, err := j.request(session, logger, methodCalls...)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, "", "", "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) ([]Email, State, Error) {
		if markAsSeen {
			var markResponse EmailSetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &markResponse)
			if err != nil {
				return nil, "", err
			}
			for _, seterr := range markResponse.NotUpdated {
				// TODO we don't have a way to compose multiple set errors yet
				return nil, "", setErrorError(seterr, EmailType)
			}
		}
		var response EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &response)
		if err != nil {
			return nil, "", err
		}
		if withThreads {
			var threads ThreadGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandThreadGet, "2", &threads)
			if err != nil {
				return nil, "", err
			}
			setThreadSize(&threads, response.List)
		}
		return response.List, response.State, nil
	})
}

func (j *Client) GetEmailBlobId(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, id string) (string, SessionState, State, Language, Error) {
	logger = j.logger("GetEmailBlobId", session, logger)

	get := EmailGetCommand{AccountId: accountId, Ids: []string{id}, FetchAllBodyValues: false, Properties: []string{"blobId"}}
	cmd, err := j.request(session, logger, invocation(CommandEmailGet, get, "0"))
	if err != nil {
		logger.Error().Err(err).Send()
		return "", "", "", "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (string, State, Error) {
		var response EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "0", &response)
		if err != nil {
			return "", "", err
		}
		if len(response.List) != 1 {
			return "", "", nil
		}
		email := response.List[0]
		return email.BlobId, response.State, nil
	})
}

// Retrieve all the Emails in a given Mailbox by its id.
func (j *Client) GetAllEmailsInMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, mailboxId string, offset uint, limit uint, collapseThreads bool, fetchBodies bool, maxBodyValueBytes uint, withThreads bool) (Emails, SessionState, State, Language, Error) {
	logger = j.loggerParams("GetAllEmailsInMailbox", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Uint(logOffset, offset).Uint(logLimit, limit)
	})

	query := EmailQueryCommand{
		AccountId:       accountId,
		Filter:          &EmailFilterCondition{InMailbox: mailboxId},
		Sort:            []EmailComparator{{Property: EmailPropertyReceivedAt, IsAscending: false}},
		CollapseThreads: collapseThreads,
		CalculateTotal:  true,
	}
	if offset > 0 {
		query.Position = offset
	}
	if limit > 0 {
		query.Limit = limit
	}

	get := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdsRef:             &ResultReference{Name: CommandEmailQuery, Path: "/ids/*", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}

	invocations := []Invocation{
		invocation(CommandEmailQuery, query, "0"),
		invocation(CommandEmailGet, get, "1"),
	}

	if withThreads {
		threads := ThreadGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: "1",
				Name:     CommandEmailGet,
				Path:     "/list/*/" + EmailPropertyThreadId,
			},
		}
		invocations = append(invocations, invocation(CommandThreadGet, threads, "2"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return Emails{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (Emails, State, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, "0", &queryResponse)
		if err != nil {
			return Emails{}, "", err
		}
		var getResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &getResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return Emails{}, "", err
		}

		if withThreads {
			var thread ThreadGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandThreadGet, "2", &thread)
			if err != nil {
				return Emails{}, "", err
			}
			setThreadSize(&thread, getResponse.List)
		}

		return Emails{
			Emails: getResponse.List,
			Total:  queryResponse.Total,
			Limit:  queryResponse.Limit,
			Offset: queryResponse.Position,
		}, queryResponse.QueryState, nil
	})
}

// Get all the Emails that have been created, updated or deleted since a given state.
func (j *Client) GetEmailsSince(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, sinceState string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (MailboxChanges, SessionState, State, Language, Error) {
	logger = j.loggerParams("GetEmailsSince", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Str(logSinceState, sinceState)
	})

	changes := EmailChangesCommand{
		AccountId:  accountId,
		SinceState: sinceState,
	}
	if maxChanges > 0 {
		changes.MaxChanges = maxChanges
	}

	getCreated := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdsRef:             &ResultReference{Name: CommandEmailChanges, Path: "/created", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		getCreated.MaxBodyValueBytes = maxBodyValueBytes
	}
	getUpdated := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdsRef:             &ResultReference{Name: CommandEmailChanges, Path: "/updated", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		getUpdated.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailChanges, changes, "0"),
		invocation(CommandEmailGet, getCreated, "1"),
		invocation(CommandEmailGet, getUpdated, "2"),
	)
	if err != nil {
		return MailboxChanges{}, "", "", "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (MailboxChanges, State, Error) {
		var changesResponse EmailChangesResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailChanges, "0", &changesResponse)
		if err != nil {
			return MailboxChanges{}, "", err
		}

		var createdResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, "", err
		}

		var updatedResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, "", err
		}

		return MailboxChanges{
			Destroyed:      changesResponse.Destroyed,
			HasMoreChanges: changesResponse.HasMoreChanges,
			NewState:       changesResponse.NewState,
			Created:        createdResponse.List,
			Updated:        createdResponse.List,
		}, updatedResponse.State, nil
	})
}

type SearchSnippetWithMeta struct {
	ReceivedAt time.Time `json:"receivedAt,omitzero"`
	EmailId    string    `json:"emailId,omitempty"`
	SearchSnippet
}

type EmailSnippetQueryResult struct {
	Snippets   []SearchSnippetWithMeta `json:"snippets,omitempty"`
	Total      uint                    `json:"total"`
	Limit      uint                    `json:"limit,omitzero"`
	Position   uint                    `json:"position,omitzero"`
	QueryState State                   `json:"queryState"`
}

func (j *Client) QueryEmailSnippets(accountIds []string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, offset uint, limit uint) (map[string]EmailSnippetQueryResult, SessionState, State, Language, Error) {
	logger = j.loggerParams("QueryEmailSnippets", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Uint(logLimit, limit).Uint(logOffset, offset)
	})

	uniqueAccountIds := structs.Uniq(accountIds)
	invocations := make([]Invocation, len(uniqueAccountIds)*3)
	for i, accountId := range uniqueAccountIds {
		query := EmailQueryCommand{
			AccountId:       accountId,
			Filter:          filter,
			Sort:            []EmailComparator{{Property: EmailPropertyReceivedAt, IsAscending: false}},
			CollapseThreads: true,
			CalculateTotal:  true,
		}
		if offset > 0 {
			query.Position = offset
		}
		if limit > 0 {
			query.Limit = limit
		}

		mails := EmailGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: mcid(accountId, "0"),
				Name:     CommandEmailQuery,
				Path:     "/ids/*",
			},
			FetchAllBodyValues: false,
			MaxBodyValueBytes:  0,
			Properties:         []string{EmailPropertyId, EmailPropertyReceivedAt, EmailPropertySentAt},
		}

		snippet := SearchSnippetGetRefCommand{
			AccountId: accountId,
			Filter:    filter,
			EmailIdRef: &ResultReference{
				ResultOf: mcid(accountId, "0"),
				Name:     CommandEmailQuery,
				Path:     "/ids/*",
			},
		}

		invocations[i*3+0] = invocation(CommandEmailQuery, query, mcid(accountId, "0"))
		invocations[i*3+1] = invocation(CommandEmailGet, mails, mcid(accountId, "1"))
		invocations[i*3+2] = invocation(CommandSearchSnippetGet, snippet, mcid(accountId, "2"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]EmailSnippetQueryResult, State, Error) {
		results := make(map[string]EmailSnippetQueryResult, len(uniqueAccountIds))
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var mailResponse EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "1"), &mailResponse)
			if err != nil {
				return nil, "", err
			}

			var snippetResponse SearchSnippetGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandSearchSnippetGet, mcid(accountId, "2"), &snippetResponse)
			if err != nil {
				return nil, "", err
			}

			mailResponseById := structs.Index(mailResponse.List, func(e Email) string { return e.Id })

			snippets := make([]SearchSnippetWithMeta, len(queryResponse.Ids))
			if len(queryResponse.Ids) > len(snippetResponse.List) {
				// TODO how do we handle this, if there are more email IDs than snippets?
			}

			i := 0
			for _, id := range queryResponse.Ids {
				if mail, ok := mailResponseById[id]; ok {
					snippets[i] = SearchSnippetWithMeta{
						EmailId:       id,
						ReceivedAt:    mail.ReceivedAt,
						SearchSnippet: snippetResponse.List[i],
					}
				} else {
					// TODO how do we handle this, if there is no email result for that id?
				}
				i++
			}

			results[accountId] = EmailSnippetQueryResult{
				Snippets:   snippets,
				Total:      queryResponse.Total,
				Limit:      queryResponse.Limit,
				Position:   queryResponse.Position,
				QueryState: queryResponse.QueryState,
			}
		}
		return results, squashStateFunc(results, func(r EmailSnippetQueryResult) State { return r.QueryState }), nil
	})
}

type EmailQueryResult struct {
	Emails     []Email `json:"emails"`
	Total      uint    `json:"total"`
	Limit      uint    `json:"limit,omitzero"`
	Position   uint    `json:"position,omitzero"`
	QueryState State   `json:"queryState"`
}

func (j *Client) QueryEmails(accountIds []string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, offset uint, limit uint, fetchBodies bool, maxBodyValueBytes uint) (map[string]EmailQueryResult, SessionState, State, Language, Error) {
	logger = j.loggerParams("QueryEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies)
	})

	uniqueAccountIds := structs.Uniq(accountIds)
	invocations := make([]Invocation, len(uniqueAccountIds)*2)
	for i, accountId := range uniqueAccountIds {
		query := EmailQueryCommand{
			AccountId:       accountId,
			Filter:          filter,
			Sort:            []EmailComparator{{Property: EmailPropertyReceivedAt, IsAscending: false}},
			CollapseThreads: true,
			CalculateTotal:  true,
		}
		if offset > 0 {
			query.Position = offset
		}
		if limit > 0 {
			query.Limit = limit
		}

		mails := EmailGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: mcid(accountId, "0"),
				Name:     CommandEmailQuery,
				Path:     "/ids/*",
			},
			FetchAllBodyValues: fetchBodies,
			MaxBodyValueBytes:  maxBodyValueBytes,
		}

		invocations[i*2+0] = invocation(CommandEmailQuery, query, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(CommandEmailGet, mails, mcid(accountId, "1"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]EmailQueryResult, State, Error) {
		results := make(map[string]EmailQueryResult, len(uniqueAccountIds))
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var emailsResponse EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "1"), &emailsResponse)
			if err != nil {
				return nil, "", err
			}

			results[accountId] = EmailQueryResult{
				Emails:     emailsResponse.List,
				Total:      queryResponse.Total,
				Limit:      queryResponse.Limit,
				Position:   queryResponse.Position,
				QueryState: queryResponse.QueryState,
			}
		}
		return results, squashStateFunc(results, func(r EmailQueryResult) State { return r.QueryState }), nil
	})
}

type EmailWithSnippets struct {
	Email    Email           `json:"email"`
	Snippets []SearchSnippet `json:"snippets,omitempty"`
}

type EmailQueryWithSnippetsResult struct {
	Results    []EmailWithSnippets `json:"results"`
	Total      uint                `json:"total"`
	Limit      uint                `json:"limit,omitzero"`
	Position   uint                `json:"position,omitzero"`
	QueryState State               `json:"queryState"`
}

func (j *Client) QueryEmailsWithSnippets(accountIds []string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, offset uint, limit uint, fetchBodies bool, maxBodyValueBytes uint) (map[string]EmailQueryWithSnippetsResult, SessionState, State, Language, Error) {
	logger = j.loggerParams("QueryEmailsWithSnippets", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies)
	})

	uniqueAccountIds := structs.Uniq(accountIds)
	invocations := make([]Invocation, len(uniqueAccountIds)*3)
	for i, accountId := range uniqueAccountIds {
		query := EmailQueryCommand{
			AccountId:       accountId,
			Filter:          filter,
			Sort:            []EmailComparator{{Property: EmailPropertyReceivedAt, IsAscending: false}},
			CollapseThreads: false,
			CalculateTotal:  true,
		}
		if offset > 0 {
			query.Position = offset
		}
		if limit > 0 {
			query.Limit = limit
		}

		snippet := SearchSnippetGetRefCommand{
			AccountId: accountId,
			Filter:    filter,
			EmailIdRef: &ResultReference{
				ResultOf: mcid(accountId, "0"),
				Name:     CommandEmailQuery,
				Path:     "/ids/*",
			},
		}

		mails := EmailGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: mcid(accountId, "0"),
				Name:     CommandEmailQuery,
				Path:     "/ids/*",
			},
			FetchAllBodyValues: fetchBodies,
			MaxBodyValueBytes:  maxBodyValueBytes,
		}
		invocations[i*3+0] = invocation(CommandEmailQuery, query, mcid(accountId, "0"))
		invocations[i*3+1] = invocation(CommandSearchSnippetGet, snippet, mcid(accountId, "1"))
		invocations[i*3+2] = invocation(CommandEmailGet, mails, mcid(accountId, "2"))
	}

	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		logger.Error().Err(err).Send()
		return nil, "", "", "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]EmailQueryWithSnippetsResult, State, Error) {
		result := make(map[string]EmailQueryWithSnippetsResult, len(uniqueAccountIds))
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var snippetResponse SearchSnippetGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandSearchSnippetGet, mcid(accountId, "1"), &snippetResponse)
			if err != nil {
				return nil, "", err
			}

			var emailsResponse EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "2"), &emailsResponse)
			if err != nil {
				return nil, "", err
			}

			snippetsById := map[string][]SearchSnippet{}
			for _, snippet := range snippetResponse.List {
				list, ok := snippetsById[snippet.EmailId]
				if !ok {
					list = []SearchSnippet{}
				}
				snippetsById[snippet.EmailId] = append(list, snippet)
			}

			results := []EmailWithSnippets{}
			for _, email := range emailsResponse.List {
				snippets, ok := snippetsById[email.Id]
				if !ok {
					snippets = []SearchSnippet{}
				}
				results = append(results, EmailWithSnippets{
					Email:    email,
					Snippets: snippets,
				})
			}

			result[accountId] = EmailQueryWithSnippetsResult{
				Results:    results,
				Total:      queryResponse.Total,
				Limit:      queryResponse.Limit,
				Position:   queryResponse.Position,
				QueryState: queryResponse.QueryState,
			}
		}
		return result, squashStateFunc(result, func(r EmailQueryWithSnippetsResult) State { return r.QueryState }), nil
	})
}

type UploadedEmail struct {
	Id     string `json:"id"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Sha512 string `json:"sha:512"`
}

func (j *Client) ImportEmail(accountId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, data []byte) (UploadedEmail, SessionState, State, Language, Error) {
	encoded := base64.StdEncoding.EncodeToString(data)

	upload := BlobUploadCommand{
		AccountId: accountId,
		Create: map[string]UploadObject{
			"0": {
				Data: []DataSourceObject{{
					DataAsBase64: encoded,
				}},
				Type: EmailMimeType,
			},
		},
	}

	getHash := BlobGetRefCommand{
		AccountId: accountId,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     CommandBlobUpload,
			Path:     "/ids",
		},
		Properties: []string{BlobPropertyDigestSha512},
	}

	cmd, err := j.request(session, logger,
		invocation(CommandBlobUpload, upload, "0"),
		invocation(CommandBlobGet, getHash, "1"),
	)
	if err != nil {
		return UploadedEmail{}, "", "", "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (UploadedEmail, State, Error) {
		var uploadResponse BlobUploadResponse
		err = retrieveResponseMatchParameters(logger, body, CommandBlobUpload, "0", &uploadResponse)
		if err != nil {
			return UploadedEmail{}, "", err
		}

		var getResponse BlobGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandBlobGet, "1", &getResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return UploadedEmail{}, "", err
		}

		if len(uploadResponse.Created) != 1 {
			logger.Error().Msgf("%T.Created has %v elements instead of 1", uploadResponse, len(uploadResponse.Created))
			return UploadedEmail{}, "", simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		upload, ok := uploadResponse.Created["0"]
		if !ok {
			logger.Error().Msgf("%T.Created has no element '0'", uploadResponse)
			return UploadedEmail{}, "", simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		if len(getResponse.List) != 1 {
			logger.Error().Msgf("%T.List has %v elements instead of 1", getResponse, len(getResponse.List))
			return UploadedEmail{}, "", simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		get := getResponse.List[0]

		return UploadedEmail{
			Id:     upload.Id,
			Size:   upload.Size,
			Type:   upload.Type,
			Sha512: get.DigestSha512,
		}, State(get.DigestSha256), nil
	})

}

func (j *Client) CreateEmail(accountId string, email EmailCreate, replaceId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (*Email, SessionState, State, Language, Error) {
	set := EmailSetCommand{
		AccountId: accountId,
		Create: map[string]EmailCreate{
			"c": email,
		},
	}
	if replaceId != "" {
		set.Destroy = []string{replaceId}
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailSet, set, "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (*Email, State, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}

		if len(setResponse.NotCreated) > 0 {
			// error occured
			// TODO(pbleser-oc) handle submission errors
		}

		setErr, notok := setResponse.NotCreated["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return nil, "", setErrorError(setErr, EmailType)
		}

		created, ok := setResponse.Created["c"]
		if !ok {
			berr := fmt.Errorf("failed to find %s in %s response", string(EmailType), string(CommandEmailSet))
			logger.Error().Err(berr)
			return nil, "", simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return created, setResponse.NewState, nil
	})
}

// The Email/set method encompasses:
//   - Changing the keywords of an Email (e.g., unread/flagged status)
//   - Adding/removing an Email to/from Mailboxes (moving a message)
//   - Deleting Emails
//
// To create drafts, use the CreateEmail function instead.
//
// To delete mails, use the DeleteEmails function instead.
func (j *Client) UpdateEmails(accountId string, updates map[string]EmailUpdate, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]*Email, SessionState, State, Language, Error) {
	cmd, err := j.request(session, logger,
		invocation(CommandEmailSet, EmailSetCommand{
			AccountId: accountId,
			Update:    updates,
		}, "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]*Email, State, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}
		if len(setResponse.NotUpdated) > 0 {
			// TODO we don't have composite errors
			for _, notUpdated := range setResponse.NotUpdated {
				return nil, "", setErrorError(notUpdated, EmailType)
			}
		}
		return setResponse.Updated, setResponse.NewState, nil
	})
}

func (j *Client) DeleteEmails(accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (map[string]SetError, SessionState, State, Language, Error) {
	cmd, err := j.request(session, logger,
		invocation(CommandEmailSet, EmailSetCommand{
			AccountId: accountId,
			Destroy:   destroy,
		}, "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]SetError, State, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}
		return setResponse.NotDestroyed, setResponse.NewState, nil
	})
}

type SubmittedEmail struct {
	Id         string                    `json:"id"`
	SendAt     time.Time                 `json:"sendAt,omitzero"`
	ThreadId   string                    `json:"threadId,omitempty"`
	UndoStatus EmailSubmissionUndoStatus `json:"undoStatus,omitempty"`
	Envelope   *Envelope                 `json:"envelope,omitempty"`

	// A list of blob ids for DSNs [RFC3464] received for this submission,
	// in order of receipt, oldest first.
	//
	// The blob is the whole MIME message (with a top-level content-type of multipart/report), as received.
	//
	// [RFC3464]: https://datatracker.ietf.org/doc/html/rfc3464
	DsnBlobIds []string `json:"dsnBlobIds,omitempty"`

	// A list of blob ids for MDNs [RFC8098] received for this submission,
	// in order of receipt, oldest first.
	//
	// The blob is the whole MIME message (with a top-level content-type of multipart/report), as received.
	//
	// [RFC8098]: https://datatracker.ietf.org/doc/html/rfc8098
	MdnBlobIds []string `json:"mdnBlobIds,omitempty"`
}

type MoveMail struct {
	FromMailboxId string
	ToMailboxId   string
}

func (j *Client) SubmitEmail(accountId string, identityId string, emailId string, move *MoveMail, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string) (EmailSubmission, SessionState, State, Language, Error) {
	logger = j.logger("SubmitEmail", session, logger)

	update := map[string]any{
		EmailPropertyKeywords + "/" + JmapKeywordDraft: nil,  // unmark as draft
		EmailPropertyKeywords + "/" + JmapKeywordSeen:  true, // mark as seen (read)
	}
	if move != nil && move.FromMailboxId != "" && move.ToMailboxId != "" && move.FromMailboxId != move.ToMailboxId {
		update[EmailPropertyMailboxIds+"/"+move.FromMailboxId] = nil
		update[EmailPropertyMailboxIds+"/"+move.ToMailboxId] = true
	}

	id := "s0"

	set := EmailSubmissionSetCommand{
		AccountId: accountId,
		Create: map[string]EmailSubmissionCreate{
			id: {
				IdentityId: identityId,
				EmailId:    emailId,
				// leaving Envelope empty
			},
		},
		OnSuccessUpdateEmail: map[string]PatchObject{
			"#" + id: update,
		},
	}

	get := EmailSubmissionGetCommand{
		AccountId: accountId,
		Ids:       []string{"#" + id},
		/*
			IdRef: &ResultReference{
				ResultOf: "0",
				Name:     CommandEmailSubmissionSet,
				Path:     ["#"]"/created/" + "#" + id + "/" + EmailPropertyId,
			},
		*/
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailSubmissionSet, set, "0"),
		invocation(CommandEmailSubmissionGet, get, "1"),
	)
	if err != nil {
		return EmailSubmission{}, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (EmailSubmission, State, Error) {
		var submissionResponse EmailSubmissionSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSubmissionSet, "0", &submissionResponse)
		if err != nil {
			return EmailSubmission{}, "", err
		}

		if len(submissionResponse.NotCreated) > 0 {
			// error occured
			// TODO(pbleser-oc) handle submission errors
		}

		// there is an implicit Email/set response:
		// "After all create/update/destroy items in the EmailSubmission/set invocation have been processed,
		// a single implicit Email/set call MUST be made to perform any changes requested in these two arguments.
		// The response to this MUST be returned after the EmailSubmission/set response."
		// from an example in the spec, it has the same tag as the EmailSubmission/set command ("0" in this case)
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return EmailSubmission{}, "", err
		}

		if emailId := structs.FirstKey(setResponse.Updated); emailId != nil && len(setResponse.Updated) == 1 {
			var getResponse EmailSubmissionGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailSubmissionGet, "1", &getResponse)
			if err != nil {
				return EmailSubmission{}, "", err
			}

			if len(getResponse.List) != 1 {
				// for some reason (error?)...
				// TODO(pbleser-oc) handle absence of emailsubmission
			}

			submission := getResponse.List[0]

			return submission, setResponse.NewState, nil
		} else {
			err = simpleError(fmt.Errorf("failed to submit email: updated is empty"), 0) // TODO proper error handling
			return EmailSubmission{}, "", err
		}
	})
}

func (j *Client) EmailsInThread(accountId string, threadId string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, fetchBodies bool, maxBodyValueBytes uint) ([]Email, SessionState, State, Language, Error) {
	logger = j.loggerParams("EmailsInThread", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Str("threadId", log.SafeString(threadId))
	})

	cmd, err := j.request(session, logger,
		invocation(CommandThreadGet, ThreadGetCommand{
			AccountId: accountId,
			Ids:       []string{threadId},
		}, "0"),
		invocation(CommandEmailGet, EmailGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: "0",
				Name:     CommandThreadGet,
				Path:     "/list/*/emailIds",
			},
			FetchAllBodyValues: fetchBodies,
			MaxBodyValueBytes:  maxBodyValueBytes,
		}, "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) ([]Email, State, Error) {
		var emailsResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &emailsResponse)
		if err != nil {
			return nil, "", err
		}
		return emailsResponse.List, emailsResponse.State, nil
	})
}

type EmailsSummary struct {
	Emails []Email `json:"emails"`
	Total  int     `json:"total"`
	Limit  int     `json:"limit"`
	Offset int     `json:"offset"`
	State  State   `json:"state"`
}

var EmailSummaryProperties = []string{
	EmailPropertyId,
	EmailPropertyThreadId,
	EmailPropertyMailboxIds,
	EmailPropertyKeywords,
	EmailPropertySize,
	EmailPropertyReceivedAt,
	EmailPropertySender,
	EmailPropertyFrom,
	EmailPropertyTo,
	EmailPropertyCc,
	EmailPropertyBcc,
	EmailPropertySubject,
	EmailPropertySentAt,
	EmailPropertyHasAttachment,
	EmailPropertyAttachments,
	EmailPropertyPreview,
}

func (j *Client) QueryEmailSummaries(accountIds []string, session *Session, ctx context.Context, logger *log.Logger, acceptLanguage string, filter EmailFilterElement, limit uint, withThreads bool) (map[string]EmailsSummary, SessionState, State, Language, Error) {
	logger = j.logger("QueryEmailSummaries", session, logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	factor := 2
	if withThreads {
		factor++
	}

	invocations := make([]Invocation, len(uniqueAccountIds)*factor)
	for i, accountId := range uniqueAccountIds {
		invocations[i*factor+0] = invocation(CommandEmailQuery, EmailQueryCommand{
			AccountId: accountId,
			Filter:    filter,
			Sort:      []EmailComparator{{Property: EmailPropertyReceivedAt, IsAscending: false}},
			Limit:     limit,
			//CalculateTotal: false,
		}, mcid(accountId, "0"))
		invocations[i*factor+1] = invocation(CommandEmailGet, EmailGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				Name:     CommandEmailQuery,
				Path:     "/ids/*",
				ResultOf: mcid(accountId, "0"),
			},
			Properties: EmailSummaryProperties,
		}, mcid(accountId, "1"))
		if withThreads {
			invocations[i*factor+2] = invocation(CommandThreadGet, ThreadGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandEmailGet,
					Path:     "/list/*/" + EmailPropertyThreadId,
					ResultOf: mcid(accountId, "1"),
				},
			}, mcid(accountId, "2"))
		}
	}
	cmd, err := j.request(session, logger, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, acceptLanguage, func(body *Response) (map[string]EmailsSummary, State, Error) {
		resp := map[string]EmailsSummary{}
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var response EmailGetResponse
			err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, mcid(accountId, "1"), &response)
			if err != nil {
				return nil, "", err
			}
			if len(response.NotFound) > 0 {
				// TODO what to do when there are not-found emails here? potentially nothing, they could have been deleted between query and get?
			}
			if withThreads {
				var thread ThreadGetResponse
				err = retrieveResponseMatchParameters(logger, body, CommandThreadGet, mcid(accountId, "2"), &thread)
				if err != nil {
					return nil, "", err
				}
				setThreadSize(&thread, response.List)
			}

			resp[accountId] = EmailsSummary{
				Emails: response.List,
				Total:  int(queryResponse.Total),
				Limit:  int(queryResponse.Limit),
				Offset: int(queryResponse.Position),
				State:  response.State,
			}
		}
		return resp, squashStateFunc(resp, func(s EmailsSummary) State { return s.State }), nil
	})
}

func setThreadSize(threads *ThreadGetResponse, emails []Email) {
	threadSizeById := make(map[string]int, len(threads.List))
	for _, thread := range threads.List {
		threadSizeById[thread.Id] = len(thread.EmailIds)
	}
	for i := range len(emails) {
		ts, ok := threadSizeById[emails[i].ThreadId]
		if !ok {
			ts = 1
		}
		emails[i].ThreadSize = ts
	}
}
