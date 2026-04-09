package jmap

import (
	"encoding/base64"
	"fmt"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/rs/zerolog"
)

var NS_MAIL = ns(JmapMail)
var NS_MAIL_SUBMISSION = ns(JmapMail, JmapSubmission)

type Emails struct {
	Emails []Email `json:"emails,omitempty"`
	Total  uint    `json:"total,omitzero"`
	Limit  uint    `json:"limit,omitzero"`
	Offset uint    `json:"offset,omitzero"`
}

type getEmailsResult struct {
	emails   []Email
	notFound []string
}

// Retrieve specific Emails by their id.
func (j *Client) GetEmails(accountId string, ids []string, //NOSONAR
	fetchBodies bool, maxBodyValueBytes uint, markAsSeen bool, withThreads bool,
	ctx Context) ([]Email, []string, SessionState, State, Language, Error) {
	logger := j.logger("GetEmails", ctx)
	ctx = ctx.WithLogger(logger)

	get := EmailGetCommand{AccountId: accountId, Ids: ids, FetchAllBodyValues: fetchBodies}
	if maxBodyValueBytes > 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}
	invokeGet := invocation(get, "1")

	methodCalls := []Invocation{invokeGet}
	if markAsSeen {
		updates := make(map[string]EmailUpdate, len(ids))
		for _, id := range ids {
			updates[id] = EmailUpdate{EmailPropertyKeywords + "/" + JmapKeywordSeen: true}
		}
		mark := EmailSetCommand{AccountId: accountId, Update: updates}
		methodCalls = []Invocation{invocation(mark, "0"), invokeGet}
	}
	if withThreads {
		threads := ThreadGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: "1",
				Name:     CommandEmailGet,
				Path:     "/list/*/" + EmailPropertyThreadId, //NOSONAR
			},
		}
		methodCalls = append(methodCalls, invocation(threads, "2"))
	}

	cmd, err := j.request(ctx, NS_MAIL, methodCalls...)
	if err != nil {
		return nil, nil, "", "", "", err
	}
	result, sessionState, state, language, gwerr := command(j, ctx, cmd, func(body *Response) (getEmailsResult, State, Error) {
		if markAsSeen {
			var markResponse EmailSetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailSet, "0", &markResponse)
			if err != nil {
				return getEmailsResult{}, "", err
			}
			for _, seterr := range markResponse.NotUpdated {
				// TODO we don't have a way to compose multiple set errors yet
				return getEmailsResult{}, "", setErrorError(seterr, EmailType)
			}
		}
		var response EmailGetResponse
		err = retrieveResponseMatchParameters(ctx, body, CommandEmailGet, "1", &response)
		if err != nil {
			return getEmailsResult{}, "", err
		}
		if withThreads {
			var threads ThreadGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandThreadGet, "2", &threads)
			if err != nil {
				return getEmailsResult{}, "", err
			}
			setThreadSize(&threads, response.List)
		}
		return getEmailsResult{emails: response.List, notFound: response.NotFound}, response.State, nil
	})
	return result.emails, result.notFound, sessionState, state, language, gwerr
}

func (j *Client) GetEmailBlobId(accountId string, id string, ctx Context) (string, SessionState, State, Language, Error) {
	logger := j.logger("GetEmailBlobId", ctx)
	ctx = ctx.WithLogger(logger)

	get := EmailGetCommand{AccountId: accountId, Ids: []string{id}, FetchAllBodyValues: false, Properties: []string{"blobId"}}
	cmd, err := j.request(ctx, NS_MAIL, invocation(get, "0"))
	if err != nil {
		return "", "", "", "", err
	}
	return command(j, ctx, cmd, func(body *Response) (string, State, Error) {
		var response EmailGetResponse
		err = retrieveGet(ctx, body, get, "0", &response)
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
func (j *Client) GetAllEmailsInMailbox(accountId string, mailboxId string, //NOSONAR
	offset int, limit uint, collapseThreads bool, fetchBodies bool, maxBodyValueBytes uint, withThreads bool,
	ctx Context) (Emails, SessionState, State, Language, Error) {
	logger := j.loggerParams("GetAllEmailsInMailbox", ctx, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Int(logOffset, offset).Uint(logLimit, limit)
	})
	ctx = ctx.WithLogger(logger)

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
		query.Limit = &limit
	}

	get := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdsRef:             &ResultReference{Name: CommandEmailQuery, Path: "/ids/*", ResultOf: "0"}, //NOSONAR
	}
	if maxBodyValueBytes > 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}

	invocations := []Invocation{
		invocation(query, "0"),
		invocation(get, "1"),
	}

	threads := ThreadGetRefCommand{}
	if withThreads {
		threads = ThreadGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				ResultOf: "1",
				Name:     CommandEmailGet,
				Path:     "/list/*/" + EmailPropertyThreadId,
			},
		}
		invocations = append(invocations, invocation(threads, "2"))
	}

	cmd, err := j.request(ctx, NS_MAIL, invocations...)
	if err != nil {
		return Emails{}, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (Emails, State, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveQuery(ctx, body, query, "0", &queryResponse)
		if err != nil {
			return Emails{}, "", err
		}
		var getResponse EmailGetResponse
		err = retrieveGet(ctx, body, get, "1", &getResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return Emails{}, "", err
		}

		if withThreads {
			var thread ThreadGetResponse
			err = retrieveGet(ctx, body, threads, "2", &thread)
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

type EmailChanges struct {
	HasMoreChanges bool     `json:"hasMoreChanges"`
	OldState       State    `json:"oldState,omitempty"`
	NewState       State    `json:"newState"`
	Created        []Email  `json:"created,omitempty"`
	Updated        []Email  `json:"updated,omitempty"`
	Destroyed      []string `json:"destroyed,omitempty"`
}

// Retrieve the changes in Emails since a given State.
// @api:tags email,changes
func (j *Client) GetEmailChanges(accountId string,
	sinceState State, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint,
	ctx Context) (EmailChanges, SessionState, State, Language, Error) { //NOSONAR
	logger := j.loggerParams("GetEmailChanges", ctx, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Str(logSinceState, string(sinceState))
	})
	ctx = ctx.WithLogger(logger)

	changes := EmailChangesCommand{
		AccountId:  accountId,
		SinceState: sinceState,
	}
	if maxChanges > 0 {
		changes.MaxChanges = &maxChanges
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

	cmd, err := j.request(ctx, NS_MAIL,
		invocation(changes, "0"),
		invocation(getCreated, "1"),
		invocation(getUpdated, "2"),
	)
	if err != nil {
		return EmailChanges{}, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (EmailChanges, State, Error) {
		var changesResponse EmailChangesResponse
		err = retrieveChanges(ctx, body, changes, "0", &changesResponse)
		if err != nil {
			return EmailChanges{}, "", err
		}

		var createdResponse EmailGetResponse
		err = retrieveGet(ctx, body, getCreated, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return EmailChanges{}, "", err
		}

		var updatedResponse EmailGetResponse
		err = retrieveGet(ctx, body, getUpdated, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return EmailChanges{}, "", err
		}

		return EmailChanges{
			Destroyed:      changesResponse.Destroyed,
			HasMoreChanges: changesResponse.HasMoreChanges,
			OldState:       changesResponse.OldState,
			NewState:       changesResponse.NewState,
			Created:        createdResponse.List,
			Updated:        updatedResponse.List,
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

func (j *Client) QueryEmailSnippets(accountIds []string, //NOSONAR
	filter EmailFilterElement, offset int, limit uint,
	ctx Context) (map[string]EmailSnippetQueryResult, SessionState, State, Language, Error) {
	logger := j.loggerParams("QueryEmailSnippets", ctx, func(z zerolog.Context) zerolog.Context {
		return z.Uint(logLimit, limit).Int(logOffset, offset)
	})
	ctx = ctx.WithLogger(logger)

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
			query.Limit = &limit
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

		invocations[i*3+0] = invocation(query, mcid(accountId, "0"))
		invocations[i*3+1] = invocation(mails, mcid(accountId, "1"))
		invocations[i*3+2] = invocation(snippet, mcid(accountId, "2"))
	}

	cmd, err := j.request(ctx, NS_MAIL, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]EmailSnippetQueryResult, State, Error) {
		results := make(map[string]EmailSnippetQueryResult, len(uniqueAccountIds))
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var mailResponse EmailGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailGet, mcid(accountId, "1"), &mailResponse)
			if err != nil {
				return nil, "", err
			}

			var snippetResponse SearchSnippetGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandSearchSnippetGet, mcid(accountId, "2"), &snippetResponse)
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

func (j *Client) QueryEmails(accountIds []string,
	filter EmailFilterElement, offset int, limit uint, fetchBodies bool, maxBodyValueBytes uint,
	ctx Context) (map[string]EmailQueryResult, SessionState, State, Language, Error) { //NOSONAR
	logger := j.loggerParams("QueryEmails", ctx, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies)
	})
	ctx = ctx.WithLogger(logger)

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
			query.Limit = &limit
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

		invocations[i*2+0] = invocation(query, mcid(accountId, "0"))
		invocations[i*2+1] = invocation(mails, mcid(accountId, "1"))
	}

	cmd, err := j.request(ctx, NS_MAIL, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]EmailQueryResult, State, Error) {
		results := make(map[string]EmailQueryResult, len(uniqueAccountIds))
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var emailsResponse EmailGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailGet, mcid(accountId, "1"), &emailsResponse)
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

func (j *Client) QueryEmailsWithSnippets(accountIds []string, //NOSONAR
	filter EmailFilterElement, offset int, limit uint, fetchBodies bool, maxBodyValueBytes uint,
	ctx Context) (map[string]EmailQueryWithSnippetsResult, SessionState, State, Language, Error) {
	logger := j.loggerParams("QueryEmailsWithSnippets", ctx, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies)
	})
	ctx = ctx.WithLogger(logger)

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
			query.Limit = &limit
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
		invocations[i*3+0] = invocation(query, mcid(accountId, "0"))
		invocations[i*3+1] = invocation(snippet, mcid(accountId, "1"))
		invocations[i*3+2] = invocation(mails, mcid(accountId, "2"))
	}

	cmd, err := j.request(ctx, NS_MAIL, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]EmailQueryWithSnippetsResult, State, Error) {
		result := make(map[string]EmailQueryWithSnippetsResult, len(uniqueAccountIds))
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var snippetResponse SearchSnippetGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandSearchSnippetGet, mcid(accountId, "1"), &snippetResponse)
			if err != nil {
				return nil, "", err
			}

			var emailsResponse EmailGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailGet, mcid(accountId, "2"), &emailsResponse)
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

func (j *Client) ImportEmail(accountId string, data []byte, ctx Context) (UploadedEmail, SessionState, State, Language, Error) {
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

	cmd, err := j.request(ctx, NS_MAIL,
		invocation(upload, "0"),
		invocation(getHash, "1"),
	)
	if err != nil {
		return UploadedEmail{}, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (UploadedEmail, State, Error) {
		var uploadResponse BlobUploadResponse
		err = retrieveResponseMatchParameters(ctx, body, CommandBlobUpload, "0", &uploadResponse)
		if err != nil {
			return UploadedEmail{}, "", err
		}

		var getResponse BlobGetResponse
		err = retrieveResponseMatchParameters(ctx, body, CommandBlobGet, "1", &getResponse)
		if err != nil {
			ctx.Logger.Error().Err(err).Send()
			return UploadedEmail{}, "", err
		}

		if len(uploadResponse.Created) != 1 {
			ctx.Logger.Error().Msgf("%T.Created has %v elements instead of 1", uploadResponse, len(uploadResponse.Created))
			return UploadedEmail{}, "", jmapError(err, JmapErrorInvalidJmapResponsePayload)
		}
		upload, ok := uploadResponse.Created["0"]
		if !ok {
			ctx.Logger.Error().Msgf("%T.Created has no element '0'", uploadResponse)
			return UploadedEmail{}, "", jmapError(err, JmapErrorInvalidJmapResponsePayload)
		}

		if len(getResponse.List) != 1 {
			ctx.Logger.Error().Msgf("%T.List has %v elements instead of 1", getResponse, len(getResponse.List))
			return UploadedEmail{}, "", jmapError(err, JmapErrorInvalidJmapResponsePayload)
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

func (j *Client) CreateEmail(accountId string, email EmailCreate, replaceId string, ctx Context) (*Email, SessionState, State, Language, Error) {
	set := EmailSetCommand{
		AccountId: accountId,
		Create: map[string]EmailCreate{
			"c": email,
		},
	}
	if replaceId != "" {
		set.Destroy = []string{replaceId}
	}

	cmd, err := j.request(ctx, NS_MAIL,
		invocation(set, "0"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (*Email, State, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(ctx, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return nil, "", err
		}

		if len(setResponse.NotCreated) > 0 {
			// error occured
			// TODO(pbleser-oc) handle submission errors
		}

		setErr, notok := setResponse.NotCreated["c"]
		if notok {
			ctx.Logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return nil, "", setErrorError(setErr, EmailType)
		}

		created, ok := setResponse.Created["c"]
		if !ok {
			berr := fmt.Errorf("failed to find %s in %s response", EmailType, string(CommandEmailSet))
			ctx.Logger.Error().Err(berr)
			return nil, "", jmapError(berr, JmapErrorInvalidJmapResponsePayload)
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
func (j *Client) UpdateEmails(accountId string, updates map[string]EmailUpdate, ctx Context) (map[string]*Email, SessionState, State, Language, Error) {
	set := EmailSetCommand{
		AccountId: accountId,
		Update:    updates,
	}
	cmd, err := j.request(ctx, NS_MAIL, invocation(set, "0"))
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]*Email, State, Error) {
		var setResponse EmailSetResponse
		err = retrieveSet(ctx, body, set, "0", &setResponse)
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

func (j *Client) DeleteEmails(accountId string, destroy []string, ctx Context) (map[string]SetError, SessionState, State, Language, Error) {
	set := EmailSetCommand{
		AccountId: accountId,
		Destroy:   destroy,
	}
	cmd, err := j.request(ctx, NS_MAIL, invocation(set, "0"))
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]SetError, State, Error) {
		var setResponse EmailSetResponse
		err = retrieveSet(ctx, body, set, "0", &setResponse)
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

func (j *Client) SubmitEmail(accountId string, identityId string, emailId string, move *MoveMail, //NOSONAR
	ctx Context) (EmailSubmission, SessionState, State, Language, Error) {
	logger := j.logger("SubmitEmail", ctx)
	ctx = ctx.WithLogger(logger)

	update := map[string]any{
		EmailPropertyKeywords + "/" + JmapKeywordDraft: nil,  // unmark as draft
		EmailPropertyKeywords + "/" + JmapKeywordSeen:  true, // mark as seen (read)
	}
	if move != nil && move.FromMailboxId != "" && move.ToMailboxId != "" && move.FromMailboxId != move.ToMailboxId {
		update[EmailPropertyMailboxIds+"/"+move.FromMailboxId] = nil
		update[EmailPropertyMailboxIds+"/"+move.ToMailboxId] = true
	}

	id := "s0"

	submit := EmailSubmissionSetCommand{
		AccountId: accountId,
		Create: map[string]EmailSubmissionCreate{
			id: {
				IdentityId: identityId,
				EmailId:    emailId,
				Envelope:   nil,
			},
		},
		OnSuccessUpdateEmail: map[string]PatchObject{
			"#" + id: update,
		},
	}

	get := EmailSubmissionGetCommand{
		AccountId: accountId,
		Ids:       []string{"#" + id},
	}

	cmd, err := j.request(ctx, NS_MAIL_SUBMISSION,
		invocation(submit, "0"),
		invocation(get, "1"),
	)
	if err != nil {
		return EmailSubmission{}, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (EmailSubmission, State, Error) {
		var submissionResponse EmailSubmissionSetResponse
		err = retrieveSet(ctx, body, submit, "0", &submissionResponse)
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
		err = retrieveResponseMatchParameters(ctx, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return EmailSubmission{}, "", err
		}

		if len(setResponse.Updated) == 1 {
			var getResponse EmailSubmissionGetResponse
			err = retrieveGet(ctx, body, get, "1", &getResponse)
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
			err = jmapError(fmt.Errorf("failed to submit email: updated is empty"), 0) // TODO proper error handling
			return EmailSubmission{}, "", err
		}
	})
}

type emailSubmissionResult struct {
	submissions map[string]EmailSubmission
	notFound    []string
}

func (j *Client) GetEmailSubmissionStatus(accountId string, submissionIds []string, ctx Context) (map[string]EmailSubmission, []string, SessionState, State, Language, Error) {
	logger := j.logger("GetEmailSubmissionStatus", ctx)
	ctx = ctx.WithLogger(logger)

	get := EmailSubmissionGetCommand{
		AccountId: accountId,
		Ids:       submissionIds,
	}
	cmd, err := j.request(ctx, NS_MAIL_SUBMISSION, invocation(get, "0"))
	if err != nil {
		return nil, nil, "", "", "", err
	}

	result, sessionState, state, lang, err := command(j, ctx, cmd, func(body *Response) (emailSubmissionResult, State, Error) {
		var response EmailSubmissionGetResponse
		err = retrieveGet(ctx, body, get, "0", &response)
		if err != nil {
			return emailSubmissionResult{}, "", err
		}
		m := make(map[string]EmailSubmission, len(response.List))
		for _, s := range response.List {
			m[s.Id] = s
		}
		return emailSubmissionResult{submissions: m, notFound: response.NotFound}, response.State, nil
	})

	return result.submissions, result.notFound, sessionState, state, lang, err
}

func (j *Client) EmailsInThread(accountId string, threadId string,
	fetchBodies bool, maxBodyValueBytes uint,
	ctx Context) ([]Email, SessionState, State, Language, Error) { //NOSONAR
	logger := j.loggerParams("EmailsInThread", ctx, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Str("threadId", log.SafeString(threadId))
	})
	ctx = ctx.WithLogger(logger)

	thread := ThreadGetCommand{
		AccountId: accountId,
		Ids:       []string{threadId},
	}

	get := EmailGetRefCommand{
		AccountId: accountId,
		IdsRef: &ResultReference{
			ResultOf: "0",
			Name:     CommandThreadGet,
			Path:     "/list/*/emailIds",
		},
		FetchAllBodyValues: fetchBodies,
		MaxBodyValueBytes:  maxBodyValueBytes,
	}

	cmd, err := j.request(ctx, NS_MAIL,
		invocation(thread, "0"),
		invocation(get, "1"),
	)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) ([]Email, State, Error) {
		var emailsResponse EmailGetResponse
		err = retrieveGet(ctx, body, get, "1", &emailsResponse)
		if err != nil {
			return nil, "", err
		}
		return emailsResponse.List, emailsResponse.State, nil
	})
}

type EmailsSummary struct {
	Emails []Email `json:"emails"`
	Total  uint    `json:"total"`
	Limit  uint    `json:"limit"`
	Offset uint    `json:"offset"`
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

func (j *Client) QueryEmailSummaries(accountIds []string, //NOSONAR
	filter EmailFilterElement, limit uint, withThreads bool,
	ctx Context) (map[string]EmailsSummary, SessionState, State, Language, Error) {
	logger := j.logger("QueryEmailSummaries", ctx)
	ctx = ctx.WithLogger(logger)

	uniqueAccountIds := structs.Uniq(accountIds)

	factor := 2
	if withThreads {
		factor++
	}

	invocations := make([]Invocation, len(uniqueAccountIds)*factor)
	for i, accountId := range uniqueAccountIds {
		get := EmailQueryCommand{
			AccountId: accountId,
			Filter:    filter,
			Sort:      []EmailComparator{{Property: EmailPropertyReceivedAt, IsAscending: false}},
		}
		if limit > 0 {
			get.Limit = &limit
		}
		invocations[i*factor+0] = invocation(get, mcid(accountId, "0"))

		invocations[i*factor+1] = invocation(EmailGetRefCommand{
			AccountId: accountId,
			IdsRef: &ResultReference{
				Name:     CommandEmailQuery,
				Path:     "/ids/*",
				ResultOf: mcid(accountId, "0"),
			},
			Properties: EmailSummaryProperties,
		}, mcid(accountId, "1"))
		if withThreads {
			invocations[i*factor+2] = invocation(ThreadGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandEmailGet,
					Path:     "/list/*/" + EmailPropertyThreadId,
					ResultOf: mcid(accountId, "1"),
				},
			}, mcid(accountId, "2"))
		}
	}
	cmd, err := j.request(ctx, NS_MAIL, invocations...)
	if err != nil {
		return nil, "", "", "", err
	}

	return command(j, ctx, cmd, func(body *Response) (map[string]EmailsSummary, State, Error) {
		resp := map[string]EmailsSummary{}
		for _, accountId := range uniqueAccountIds {
			var queryResponse EmailQueryResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailQuery, mcid(accountId, "0"), &queryResponse)
			if err != nil {
				return nil, "", err
			}

			var response EmailGetResponse
			err = retrieveResponseMatchParameters(ctx, body, CommandEmailGet, mcid(accountId, "1"), &response)
			if err != nil {
				return nil, "", err
			}
			if len(response.NotFound) > 0 {
				// TODO what to do when there are not-found emails here? potentially nothing, they could have been deleted between query and get?
			}
			if withThreads {
				var thread ThreadGetResponse
				err = retrieveResponseMatchParameters(ctx, body, CommandThreadGet, mcid(accountId, "2"), &thread)
				if err != nil {
					return nil, "", err
				}
				setThreadSize(&thread, response.List)
			}

			resp[accountId] = EmailsSummary{
				Emails: response.List,
				Total:  queryResponse.Total,
				Limit:  queryResponse.Limit,
				Offset: queryResponse.Position,
				State:  response.State,
			}
		}
		return resp, squashStateFunc(resp, func(s EmailsSummary) State { return s.State }), nil
	})
}

type EmailSubmissionChanges = ChangesTemplate[EmailSubmission]

// Retrieve the changes in Email Submissions since a given State.
// @api:tags email,changes
func (j *Client) GetEmailSubmissionChanges(accountId string, sinceState State, maxChanges uint,
	ctx Context) (EmailSubmissionChanges, SessionState, State, Language, Error) {
	return changes(j, "GetEmailSubmissionChanges", NS_MAIL_SUBMISSION,
		func() EmailSubmissionChangesCommand {
			return EmailSubmissionChangesCommand{AccountId: accountId, SinceState: sinceState, MaxChanges: uintPtr(maxChanges)}
		},
		EmailSubmissionChangesResponse{},
		func(path string, rof string) EmailSubmissionGetRefCommand {
			return EmailSubmissionGetRefCommand{
				AccountId: accountId,
				IdsRef: &ResultReference{
					Name:     CommandEmailSubmissionChanges,
					Path:     path,
					ResultOf: rof,
				},
			}
		},
		func(resp EmailSubmissionGetResponse) []EmailSubmission { return resp.List },
		func(oldState, newState State, hasMoreChanges bool, created, updated []EmailSubmission, destroyed []string) EmailSubmissionChanges {
			return EmailSubmissionChanges{
				OldState:       oldState,
				NewState:       newState,
				HasMoreChanges: hasMoreChanges,
				Created:        created,
				Updated:        updated,
				Destroyed:      destroyed,
			}
		},
		ctx,
	)
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
