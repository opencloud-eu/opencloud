package jmap

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/rs/zerolog"
)

const (
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

type Emails struct {
	Emails []Email `json:"emails,omitempty"`
	Total  uint    `json:"total,omitzero"`
	Limit  uint    `json:"limit,omitzero"`
	Offset uint    `json:"offset,omitzero"`
	State  State   `json:"state,omitempty"`
}

// Retrieve specific Emails by their id.
func (j *Client) GetEmails(accountId string, session *Session, ctx context.Context, logger *log.Logger, ids []string, fetchBodies bool, maxBodyValueBytes uint) (Emails, SessionState, Error) {
	logger = j.logger("GetEmails", session, logger)

	get := EmailGetCommand{AccountId: accountId, Ids: ids, FetchAllBodyValues: fetchBodies}
	if maxBodyValueBytes > 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := j.request(session, logger, invocation(CommandEmailGet, get, "0"))
	if err != nil {
		logger.Error().Err(err).Send()
		return Emails{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}
	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Emails, Error) {
		var response EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "0", &response)
		if err != nil {
			return Emails{}, err
		}
		return Emails{Emails: response.List, State: response.State}, nil
	})
}

// Retrieve all the Emails in a given Mailbox by its id.
func (j *Client) GetAllEmailsInMailbox(accountId string, session *Session, ctx context.Context, logger *log.Logger, mailboxId string, offset uint, limit uint, fetchBodies bool, maxBodyValueBytes uint) (Emails, SessionState, Error) {
	logger = j.loggerParams("GetAllEmailsInMailbox", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Uint(logOffset, offset).Uint(logLimit, limit)
	})

	query := EmailQueryCommand{
		AccountId:       accountId,
		Filter:          &EmailFilterCondition{InMailbox: mailboxId},
		Sort:            []EmailComparator{{Property: emailSortByReceivedAt, IsAscending: false}},
		CollapseThreads: true,
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
		IdRef:              &ResultReference{Name: CommandEmailQuery, Path: "/ids/*", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		get.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailQuery, query, "0"),
		invocation(CommandEmailGet, get, "1"),
	)
	if err != nil {
		return Emails{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (Emails, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, "0", &queryResponse)
		if err != nil {
			return Emails{}, err
		}
		var getResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &getResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return Emails{}, err
		}

		return Emails{
			Emails: getResponse.List,
			Total:  queryResponse.Total,
			Limit:  queryResponse.Limit,
			Offset: queryResponse.Position,
			State:  getResponse.State,
		}, nil
	})
}

// Get all the Emails that have been created, updated or deleted since a given state.
func (j *Client) GetEmailsSince(accountId string, session *Session, ctx context.Context, logger *log.Logger, sinceState string, fetchBodies bool, maxBodyValueBytes uint, maxChanges uint) (MailboxChanges, SessionState, Error) {
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
		IdRef:              &ResultReference{Name: CommandEmailChanges, Path: "/created", ResultOf: "0"},
	}
	if maxBodyValueBytes > 0 {
		getCreated.MaxBodyValueBytes = maxBodyValueBytes
	}
	getUpdated := EmailGetRefCommand{
		AccountId:          accountId,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &ResultReference{Name: CommandEmailChanges, Path: "/updated", ResultOf: "0"},
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
		return MailboxChanges{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (MailboxChanges, Error) {
		var changesResponse EmailChangesResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailChanges, "0", &changesResponse)
		if err != nil {
			return MailboxChanges{}, err
		}

		var createdResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &createdResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, err
		}

		var updatedResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "2", &updatedResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return MailboxChanges{}, err
		}

		return MailboxChanges{
			Destroyed:      changesResponse.Destroyed,
			HasMoreChanges: changesResponse.HasMoreChanges,
			NewState:       changesResponse.NewState,
			Created:        createdResponse.List,
			Updated:        createdResponse.List,
			State:          updatedResponse.State,
		}, nil
	})
}

type EmailSnippetQueryResult struct {
	Snippets   []SearchSnippet `json:"snippets,omitempty"`
	Total      uint            `json:"total"`
	Limit      uint            `json:"limit,omitzero"`
	Position   uint            `json:"position,omitzero"`
	QueryState State           `json:"queryState"`
}

func (j *Client) QueryEmailSnippets(accountId string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, offset uint, limit uint) (EmailSnippetQueryResult, SessionState, Error) {
	logger = j.loggerParams("QueryEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Uint(logLimit, limit).Uint(logOffset, offset)
	})

	query := EmailQueryCommand{
		AccountId:       accountId,
		Filter:          filter,
		Sort:            []EmailComparator{{Property: emailSortByReceivedAt, IsAscending: false}},
		CollapseThreads: true,
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
			ResultOf: "0",
			Name:     CommandEmailQuery,
			Path:     "/ids/*",
		},
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailQuery, query, "0"),
		invocation(CommandSearchSnippetGet, snippet, "1"),
	)
	if err != nil {
		return EmailSnippetQueryResult{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (EmailSnippetQueryResult, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, "0", &queryResponse)
		if err != nil {
			return EmailSnippetQueryResult{}, err
		}

		var snippetResponse SearchSnippetGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandSearchSnippetGet, "1", &snippetResponse)
		if err != nil {
			return EmailSnippetQueryResult{}, err
		}

		return EmailSnippetQueryResult{
			Snippets:   snippetResponse.List,
			Total:      queryResponse.Total,
			Limit:      queryResponse.Limit,
			Position:   queryResponse.Position,
			QueryState: queryResponse.QueryState,
		}, nil
	})

}

type EmailQueryResult struct {
	Emails     []Email `json:"emails"`
	Total      uint    `json:"total"`
	Limit      uint    `json:"limit,omitzero"`
	Position   uint    `json:"position,omitzero"`
	QueryState State   `json:"queryState"`
}

func (j *Client) QueryEmails(accountId string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, offset uint, limit uint, fetchBodies bool, maxBodyValueBytes uint) (EmailQueryResult, SessionState, Error) {
	logger = j.loggerParams("QueryEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies)
	})

	query := EmailQueryCommand{
		AccountId:       accountId,
		Filter:          filter,
		Sort:            []EmailComparator{{Property: emailSortByReceivedAt, IsAscending: false}},
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
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     CommandEmailQuery,
			Path:     "/ids/*",
		},
		FetchAllBodyValues: fetchBodies,
		MaxBodyValueBytes:  maxBodyValueBytes,
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailQuery, query, "0"),
		invocation(CommandEmailGet, mails, "1"),
	)
	if err != nil {
		return EmailQueryResult{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (EmailQueryResult, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, "0", &queryResponse)
		if err != nil {
			return EmailQueryResult{}, err
		}

		var emailsResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &emailsResponse)
		if err != nil {
			return EmailQueryResult{}, err
		}

		return EmailQueryResult{
			Emails:     emailsResponse.List,
			Total:      queryResponse.Total,
			Limit:      queryResponse.Limit,
			Position:   queryResponse.Position,
			QueryState: queryResponse.QueryState,
		}, nil
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

func (j *Client) QueryEmailsWithSnippets(accountId string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, offset uint, limit uint, fetchBodies bool, maxBodyValueBytes uint) (EmailQueryWithSnippetsResult, SessionState, Error) {
	logger = j.loggerParams("QueryEmailsWithSnippets", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies)
	})

	query := EmailQueryCommand{
		AccountId:       accountId,
		Filter:          filter,
		Sort:            []EmailComparator{{Property: emailSortByReceivedAt, IsAscending: false}},
		CollapseThreads: true,
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
			ResultOf: "0",
			Name:     CommandEmailQuery,
			Path:     "/ids/*",
		},
	}

	mails := EmailGetRefCommand{
		AccountId: accountId,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     CommandEmailQuery,
			Path:     "/ids/*",
		},
		FetchAllBodyValues: fetchBodies,
		MaxBodyValueBytes:  maxBodyValueBytes,
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailQuery, query, "0"),
		invocation(CommandSearchSnippetGet, snippet, "1"),
		invocation(CommandEmailGet, mails, "2"),
	)

	if err != nil {
		logger.Error().Err(err).Send()
		return EmailQueryWithSnippetsResult{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (EmailQueryWithSnippetsResult, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailQuery, "0", &queryResponse)
		if err != nil {
			return EmailQueryWithSnippetsResult{}, err
		}

		var snippetResponse SearchSnippetGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandSearchSnippetGet, "1", &snippetResponse)
		if err != nil {
			return EmailQueryWithSnippetsResult{}, err
		}

		var emailsResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "2", &emailsResponse)
		if err != nil {
			return EmailQueryWithSnippetsResult{}, err
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

		return EmailQueryWithSnippetsResult{
			Results:    results,
			Total:      queryResponse.Total,
			Limit:      queryResponse.Limit,
			Position:   queryResponse.Position,
			QueryState: queryResponse.QueryState,
		}, nil
	})

}

type UploadedEmail struct {
	Id     string `json:"id"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Sha512 string `json:"sha:512"`
}

func (j *Client) ImportEmail(accountId string, session *Session, ctx context.Context, logger *log.Logger, data []byte) (UploadedEmail, SessionState, Error) {
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
		return UploadedEmail{}, "", simpleError(err, JmapErrorInvalidJmapRequestPayload)
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (UploadedEmail, Error) {
		var uploadResponse BlobUploadResponse
		err = retrieveResponseMatchParameters(logger, body, CommandBlobUpload, "0", &uploadResponse)
		if err != nil {
			return UploadedEmail{}, err
		}

		var getResponse BlobGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandBlobGet, "1", &getResponse)
		if err != nil {
			logger.Error().Err(err).Send()
			return UploadedEmail{}, err
		}

		if len(uploadResponse.Created) != 1 {
			logger.Error().Msgf("%T.Created has %v elements instead of 1", uploadResponse, len(uploadResponse.Created))
			return UploadedEmail{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		upload, ok := uploadResponse.Created["0"]
		if !ok {
			logger.Error().Msgf("%T.Created has no element '0'", uploadResponse)
			return UploadedEmail{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}

		if len(getResponse.List) != 1 {
			logger.Error().Msgf("%T.List has %v elements instead of 1", getResponse, len(getResponse.List))
			return UploadedEmail{}, simpleError(err, JmapErrorInvalidJmapResponsePayload)
		}
		get := getResponse.List[0]

		return UploadedEmail{
			Id:     upload.Id,
			Size:   upload.Size,
			Type:   upload.Type,
			Sha512: get.DigestSha512,
		}, nil
	})

}

type CreatedEmail struct {
	Email Email `json:"email"`
	State State `json:"state"`
}

func (j *Client) CreateEmail(accountId string, email EmailCreate, session *Session, ctx context.Context, logger *log.Logger) (CreatedEmail, SessionState, Error) {
	cmd, err := j.request(session, logger,
		invocation(CommandEmailSubmissionSet, EmailSetCommand{
			AccountId: accountId,
			Create: map[string]EmailCreate{
				"c": email,
			},
		}, "0"),
	)
	if err != nil {
		return CreatedEmail{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (CreatedEmail, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return CreatedEmail{}, err
		}

		if len(setResponse.NotCreated) > 0 {
			// error occured
			// TODO(pbleser-oc) handle submission errors
		}

		setErr, notok := setResponse.NotCreated["c"]
		if notok {
			logger.Error().Msgf("%T.NotCreated returned an error %v", setResponse, setErr)
			return CreatedEmail{}, setErrorError(setErr, EmailType)
		}

		created, ok := setResponse.Created["c"]
		if !ok {
			berr := fmt.Errorf("failed to find %s in %s response", string(EmailType), string(CommandEmailSet))
			logger.Error().Err(berr)
			return CreatedEmail{}, simpleError(berr, JmapErrorInvalidJmapResponsePayload)
		}

		return CreatedEmail{
			Email: created,
			State: setResponse.NewState,
		}, nil
	})
}

type UpdatedEmails struct {
	Updated map[string]Email `json:"email"`
	State   State            `json:"state"`
}

// The Email/set method encompasses:
//   - Changing the keywords of an Email (e.g., unread/flagged status)
//   - Adding/removing an Email to/from Mailboxes (moving a message)
//   - Deleting Emails
//
// To create drafts, use the CreateEmail function instead.
//
// To delete mails, use the DeleteEmails function instead.
func (j *Client) UpdateEmails(accountId string, updates map[string]EmailUpdate, session *Session, ctx context.Context, logger *log.Logger) (UpdatedEmails, SessionState, Error) {
	cmd, err := j.request(session, logger,
		invocation(CommandEmailSet, EmailSetCommand{
			AccountId: accountId,
			Update:    updates,
		}, "0"),
	)
	if err != nil {
		return UpdatedEmails{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (UpdatedEmails, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return UpdatedEmails{}, err
		}
		if len(setResponse.NotUpdated) != len(updates) {
			// error occured
			// TODO(pbleser-oc) handle submission errors
		}
		return UpdatedEmails{
			Updated: setResponse.Updated,
			State:   setResponse.NewState,
		}, nil
	})
}

type DeletedEmails struct {
	State State `json:"state"`
}

func (j *Client) DeleteEmails(accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger) (DeletedEmails, SessionState, Error) {
	cmd, err := j.request(session, logger,
		invocation(CommandEmailSet, EmailSetCommand{
			AccountId: accountId,
			Destroy:   destroy,
		}, "0"),
	)
	if err != nil {
		return DeletedEmails{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (DeletedEmails, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSet, "0", &setResponse)
		if err != nil {
			return DeletedEmails{}, err
		}
		if len(setResponse.NotDestroyed) != len(destroy) {
			// error occured
			// TODO(pbleser-oc) handle submission errors
		}
		return DeletedEmails{State: setResponse.NewState}, nil
	})
}

type SubmittedEmail struct {
	Id         string                    `json:"id"`
	State      State                     `json:"state"`
	SendAt     time.Time                 `json:"sendAt,omitzero"`
	ThreadId   string                    `json:"threadId,omitempty"`
	UndoStatus EmailSubmissionUndoStatus `json:"undoStatus,omitempty"`
	Envelope   Envelope                  `json:"envelope,omitempty"`

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

func (j *Client) SubmitEmail(accountId string, identityId string, emailId string, session *Session, ctx context.Context, logger *log.Logger, data []byte) (SubmittedEmail, SessionState, Error) {
	set := EmailSubmissionSetCommand{
		AccountId: accountId,
		Create: map[string]EmailSubmissionCreate{
			"s0": {
				IdentityId: identityId,
				EmailId:    emailId,
			},
		},
		OnSuccessUpdateEmail: map[string]PatchObject{
			"#s0": {
				"keywords/" + JmapKeywordDraft: nil,
			},
		},
	}

	get := EmailSubmissionGetRefCommand{
		AccountId: accountId,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     CommandEmailSubmissionSet,
			Path:     "/created/s0/id",
		},
	}

	cmd, err := j.request(session, logger,
		invocation(CommandEmailSubmissionSet, set, "0"),
		invocation(CommandEmailSubmissionGet, get, "1"),
	)
	if err != nil {
		return SubmittedEmail{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (SubmittedEmail, Error) {
		var submissionResponse EmailSubmissionSetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSubmissionSet, "0", &submissionResponse)
		if err != nil {
			return SubmittedEmail{}, err
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
			return SubmittedEmail{}, err
		}

		var getResponse EmailSubmissionGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailSubmissionGet, "1", &getResponse)
		if err != nil {
			return SubmittedEmail{}, err
		}

		if len(getResponse.List) != 1 {
			// for some reason (error?)...
			// TODO(pbleser-oc) handle absence of emailsubmission
		}

		submission := getResponse.List[0]

		return SubmittedEmail{
			Id:         submission.Id,
			State:      setResponse.NewState,
			SendAt:     submission.SendAt,
			ThreadId:   submission.ThreadId,
			UndoStatus: submission.UndoStatus,
			Envelope:   *submission.Envelope,
			DsnBlobIds: submission.DsnBlobIds,
			MdnBlobIds: submission.MdnBlobIds,
		}, nil
	})
}

func (j *Client) EmailsInThread(accountId string, threadId string, session *Session, ctx context.Context, logger *log.Logger, fetchBodies bool, maxBodyValueBytes uint) ([]Email, SessionState, Error) {
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
			IdRef: &ResultReference{
				ResultOf: "0",
				Name:     CommandThreadGet,
				Path:     "/list/*/emailIds",
			},
			FetchAllBodyValues: fetchBodies,
			MaxBodyValueBytes:  maxBodyValueBytes,
		}, "1"),
	)
	if err != nil {
		return []Email{}, "", err
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) ([]Email, Error) {
		var emailsResponse EmailGetResponse
		err = retrieveResponseMatchParameters(logger, body, CommandEmailGet, "1", &emailsResponse)
		if err != nil {
			return []Email{}, err
		}
		return emailsResponse.List, nil
	})

}
