package jmap

import (
	"context"
	"encoding/base64"
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
	Total  int     `json:"total,omitzero"`
	Limit  int     `json:"limit,omitzero"`
	Offset int     `json:"offset,omitzero"`
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
		Filter:          &EmailFilterCondition{InMailbox: mailboxId},
		Sort:            []Sort{{Property: emailSortByReceivedAt, IsAscending: false}},
		CollapseThreads: true,
		CalculateTotal:  true,
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
		IdRef:              &ResultReference{Name: EmailQuery, Path: "/ids/*", ResultOf: "0"},
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
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(body, EmailQuery, "0", &queryResponse)
		if err != nil {
			return Emails{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}
		var getResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "1", &getResponse)
		if err != nil {
			return Emails{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		return Emails{
			Emails: getResponse.List,
			State:  body.SessionState,
			Total:  queryResponse.Total,
			Limit:  queryResponse.Limit,
			Offset: queryResponse.Position,
		}, nil
	})
}

type EmailsSince struct {
	Destroyed      []string `json:"destroyed,omitzero"`
	HasMoreChanges bool     `json:"hasMoreChanges,omitzero"`
	NewState       string   `json:"newState"`
	Created        []Email  `json:"created,omitempty"`
	Updated        []Email  `json:"updated,omitempty"`
	State          string   `json:"state,omitempty"`
}

func (j *Client) GetEmailsInMailboxSince(accountId string, session *Session, ctx context.Context, logger *log.Logger, mailboxId string, since string, fetchBodies bool, maxBodyValueBytes int, maxChanges int) (EmailsSince, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.loggerParams(aid, "GetEmailsInMailboxSince", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Str(logSince, since)
	})

	changes := MailboxChangesCommand{
		AccountId:  aid,
		SinceState: since,
	}
	if maxChanges >= 0 {
		changes.MaxChanges = maxChanges
	}

	getCreated := EmailGetRefCommand{
		AccountId:          aid,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &ResultReference{Name: MailboxChanges, Path: "/created", ResultOf: "0"},
	}
	if maxBodyValueBytes >= 0 {
		getCreated.MaxBodyValueBytes = maxBodyValueBytes
	}
	getUpdated := EmailGetRefCommand{
		AccountId:          aid,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &ResultReference{Name: MailboxChanges, Path: "/updated", ResultOf: "0"},
	}
	if maxBodyValueBytes >= 0 {
		getUpdated.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := request(
		invocation(MailboxChanges, changes, "0"),
		invocation(EmailGet, getCreated, "1"),
		invocation(EmailGet, getUpdated, "2"),
	)
	if err != nil {
		return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (EmailsSince, Error) {
		var mailboxResponse MailboxChangesResponse
		err = retrieveResponseMatchParameters(body, MailboxChanges, "0", &mailboxResponse)
		if err != nil {
			return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var createdResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "1", &createdResponse)
		if err != nil {
			return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var updatedResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "2", &updatedResponse)
		if err != nil {
			return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		return EmailsSince{
			Destroyed:      mailboxResponse.Destroyed,
			HasMoreChanges: mailboxResponse.HasMoreChanges,
			NewState:       mailboxResponse.NewState,
			Created:        createdResponse.List,
			Updated:        createdResponse.List,
			State:          body.SessionState,
		}, nil
	})
}

func (j *Client) GetEmailsSince(accountId string, session *Session, ctx context.Context, logger *log.Logger, since string, fetchBodies bool, maxBodyValueBytes int, maxChanges int) (EmailsSince, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.loggerParams(aid, "GetEmailsSince", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies).Str(logSince, since)
	})

	changes := EmailChangesCommand{
		AccountId:  aid,
		SinceState: since,
	}
	if maxChanges >= 0 {
		changes.MaxChanges = maxChanges
	}

	getCreated := EmailGetRefCommand{
		AccountId:          aid,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &ResultReference{Name: EmailChanges, Path: "/created", ResultOf: "0"},
	}
	if maxBodyValueBytes >= 0 {
		getCreated.MaxBodyValueBytes = maxBodyValueBytes
	}
	getUpdated := EmailGetRefCommand{
		AccountId:          aid,
		FetchAllBodyValues: fetchBodies,
		IdRef:              &ResultReference{Name: EmailChanges, Path: "/updated", ResultOf: "0"},
	}
	if maxBodyValueBytes >= 0 {
		getUpdated.MaxBodyValueBytes = maxBodyValueBytes
	}

	cmd, err := request(
		invocation(EmailChanges, changes, "0"),
		invocation(EmailGet, getCreated, "1"),
		invocation(EmailGet, getUpdated, "2"),
	)
	if err != nil {
		return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (EmailsSince, Error) {
		var changesResponse EmailChangesResponse
		err = retrieveResponseMatchParameters(body, EmailChanges, "0", &changesResponse)
		if err != nil {
			return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var createdResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "1", &createdResponse)
		if err != nil {
			return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var updatedResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "2", &updatedResponse)
		if err != nil {
			return EmailsSince{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		return EmailsSince{
			Destroyed:      changesResponse.Destroyed,
			HasMoreChanges: changesResponse.HasMoreChanges,
			NewState:       changesResponse.NewState,
			Created:        createdResponse.List,
			Updated:        createdResponse.List,
			State:          body.SessionState,
		}, nil
	})
}

type EmailSnippetQueryResult struct {
	Snippets     []SearchSnippet `json:"snippets,omitempty"`
	QueryState   string          `json:"queryState"`
	Total        int             `json:"total"`
	Limit        int             `json:"limit,omitzero"`
	Position     int             `json:"position,omitzero"`
	SessionState string          `json:"sessionState,omitempty"`
}

func (j *Client) QueryEmailSnippets(accountId string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, offset int, limit int) (EmailSnippetQueryResult, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.loggerParams(aid, "QueryEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Int(logLimit, limit).Int(logOffset, offset)
	})

	query := EmailQueryCommand{
		AccountId:       aid,
		Filter:          filter,
		Sort:            []Sort{{Property: emailSortByReceivedAt, IsAscending: false}},
		CollapseThreads: true,
		CalculateTotal:  true,
	}
	if offset >= 0 {
		query.Position = offset
	}
	if limit >= 0 {
		query.Limit = limit
	}

	snippet := SearchSnippetRefCommand{
		AccountId: aid,
		Filter:    filter,
		EmailIdRef: &ResultReference{
			ResultOf: "0",
			Name:     EmailQuery,
			Path:     "/ids/*",
		},
	}

	cmd, err := request(
		invocation(EmailQuery, query, "0"),
		invocation(SearchSnippetGet, snippet, "1"),
	)

	if err != nil {
		return EmailSnippetQueryResult{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (EmailSnippetQueryResult, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(body, EmailQuery, "0", &queryResponse)
		if err != nil {
			return EmailSnippetQueryResult{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var snippetResponse SearchSnippetGetResponse
		err = retrieveResponseMatchParameters(body, SearchSnippetGet, "1", &snippetResponse)
		if err != nil {
			return EmailSnippetQueryResult{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		return EmailSnippetQueryResult{
			Snippets:     snippetResponse.List,
			QueryState:   queryResponse.QueryState,
			Total:        queryResponse.Total,
			Limit:        queryResponse.Limit,
			Position:     queryResponse.Position,
			SessionState: body.SessionState,
		}, nil
	})

}

type EmailWithSnippets struct {
	Email    Email           `json:"email"`
	Snippets []SearchSnippet `json:"snippets,omitempty"`
}

type EmailQueryResult struct {
	Results      []EmailWithSnippets `json:"results"`
	QueryState   string              `json:"queryState"`
	Total        int                 `json:"total"`
	Limit        int                 `json:"limit,omitzero"`
	Position     int                 `json:"position,omitzero"`
	SessionState string              `json:"sessionState,omitempty"`
}

func (j *Client) QueryEmails(accountId string, filter EmailFilterElement, session *Session, ctx context.Context, logger *log.Logger, offset int, limit int, fetchBodies bool, maxBodyValueBytes int) (EmailQueryResult, Error) {
	aid := session.MailAccountId(accountId)
	logger = j.loggerParams(aid, "QueryEmails", session, logger, func(z zerolog.Context) zerolog.Context {
		return z.Bool(logFetchBodies, fetchBodies)
	})

	query := EmailQueryCommand{
		AccountId:       aid,
		Filter:          filter,
		Sort:            []Sort{{Property: emailSortByReceivedAt, IsAscending: false}},
		CollapseThreads: true,
		CalculateTotal:  true,
	}
	if offset >= 0 {
		query.Position = offset
	}
	if limit >= 0 {
		query.Limit = limit
	}

	snippet := SearchSnippetRefCommand{
		AccountId: aid,
		Filter:    filter,
		EmailIdRef: &ResultReference{
			ResultOf: "0",
			Name:     EmailQuery,
			Path:     "/ids/*",
		},
	}

	mails := EmailGetRefCommand{
		AccountId: aid,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     EmailQuery,
			Path:     "/ids/*",
		},
		FetchAllBodyValues: fetchBodies,
		MaxBodyValueBytes:  maxBodyValueBytes,
	}

	cmd, err := request(
		invocation(EmailQuery, query, "0"),
		invocation(SearchSnippetGet, snippet, "1"),
		invocation(EmailGet, mails, "2"),
	)

	if err != nil {
		return EmailQueryResult{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (EmailQueryResult, Error) {
		var queryResponse EmailQueryResponse
		err = retrieveResponseMatchParameters(body, EmailQuery, "0", &queryResponse)
		if err != nil {
			return EmailQueryResult{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var snippetResponse SearchSnippetGetResponse
		err = retrieveResponseMatchParameters(body, SearchSnippetGet, "1", &snippetResponse)
		if err != nil {
			return EmailQueryResult{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var emailsResponse EmailGetResponse
		err = retrieveResponseMatchParameters(body, EmailGet, "2", &emailsResponse)
		if err != nil {
			return EmailQueryResult{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
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

		return EmailQueryResult{
			Results:      results,
			QueryState:   queryResponse.QueryState,
			Total:        queryResponse.Total,
			Limit:        queryResponse.Limit,
			Position:     queryResponse.Position,
			SessionState: body.SessionState,
		}, nil
	})

}

type UploadedEmail struct {
	Id     string `json:"id"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Sha512 string `json:"sha:512"`
}

func (j *Client) ImportEmail(accountId string, session *Session, ctx context.Context, logger *log.Logger, data []byte) (UploadedEmail, Error) {
	aid := session.MailAccountId(accountId)

	encoded := base64.StdEncoding.EncodeToString(data)

	upload := BlobUploadCommand{
		AccountId: aid,
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
		AccountId: aid,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     BlobUpload,
			Path:     "/ids",
		},
		Properties: []string{BlobPropertyDigestSha512},
	}

	cmd, err := request(
		invocation(BlobUpload, upload, "0"),
		invocation(BlobGet, getHash, "1"),
	)
	if err != nil {
		return UploadedEmail{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (UploadedEmail, Error) {
		var uploadResponse BlobUploadResponse
		err = retrieveResponseMatchParameters(body, BlobUpload, "0", &uploadResponse)
		if err != nil {
			return UploadedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var getResponse BlobGetResponse
		err = retrieveResponseMatchParameters(body, BlobGet, "1", &getResponse)
		if err != nil {
			return UploadedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		if len(uploadResponse.Created) != 1 {
			return UploadedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}
		upload, ok := uploadResponse.Created["0"]
		if !ok {
			return UploadedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		if len(getResponse.List) != 1 {
			return UploadedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
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
	Email Email  `json:"email"`
	State string `json:"state"`
}

func (j *Client) CreateEmail(accountId string, email EmailCreate, session *Session, ctx context.Context, logger *log.Logger) (CreatedEmail, Error) {
	aid := session.MailAccountId(accountId)

	cmd, err := request(
		invocation(EmailSubmissionSet, EmailSetCommand{
			AccountId: aid,
			Create: map[string]EmailCreate{
				"c": email,
			},
		}, "0"),
	)
	if err != nil {
		return CreatedEmail{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (CreatedEmail, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(body, EmailSet, "0", &setResponse)
		if err != nil {
			return CreatedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		if len(setResponse.NotCreated) > 0 {
			// error occured
			// TODO(pbleser-oc) handle submission errors
		}

		created, ok := setResponse.Created["c"]
		if !ok {
			// failed to create?
			// TODO(pbleser-oc) handle email creation failure
		}

		return CreatedEmail{
			Email: created,
			State: setResponse.NewState,
		}, nil
	})
}

type UpdatedEmails struct {
	Updated map[string]Email `json:"email"`
	State   string           `json:"state"`
}

// The Email/set method encompasses:
//   - Changing the keywords of an Email (e.g., unread/flagged status)
//   - Adding/removing an Email to/from Mailboxes (moving a message)
//   - Deleting Emails
//
// To create drafts, use the CreateEmail function instead.
//
// To delete mails, use the DeleteEmails function instead.
func (j *Client) UpdateEmails(accountId string, updates map[string]EmailUpdate, session *Session, ctx context.Context, logger *log.Logger) (UpdatedEmails, Error) {
	aid := session.MailAccountId(accountId)

	cmd, err := request(
		invocation(EmailSet, EmailSetCommand{
			AccountId: aid,
			Update:    updates,
		}, "0"),
	)
	if err != nil {
		return UpdatedEmails{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (UpdatedEmails, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(body, EmailSet, "0", &setResponse)
		if err != nil {
			return UpdatedEmails{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
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
	State string `json:"state"`
}

func (j *Client) DeleteEmails(accountId string, destroy []string, session *Session, ctx context.Context, logger *log.Logger) (DeletedEmails, Error) {
	aid := session.MailAccountId(accountId)

	cmd, err := request(
		invocation(EmailSet, EmailSetCommand{
			AccountId: aid,
			Destroy:   destroy,
		}, "0"),
	)
	if err != nil {
		return DeletedEmails{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (DeletedEmails, Error) {
		var setResponse EmailSetResponse
		err = retrieveResponseMatchParameters(body, EmailSet, "0", &setResponse)
		if err != nil {
			return DeletedEmails{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
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
	State      string                    `json:"state"`
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

func (j *Client) SubmitEmail(accountId string, identityId string, emailId string, session *Session, ctx context.Context, logger *log.Logger, data []byte) (SubmittedEmail, Error) {
	aid := session.SubmissionAccountId(accountId)

	set := EmailSubmissionSetCommand{
		AccountId: aid,
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
		AccountId: aid,
		IdRef: &ResultReference{
			ResultOf: "0",
			Name:     EmailSubmissionSet,
			Path:     "/created/s0/id",
		},
	}

	cmd, err := request(
		invocation(EmailSubmissionSet, set, "0"),
		invocation(EmailSubmissionGet, get, "1"),
	)
	if err != nil {
		return SubmittedEmail{}, SimpleError{code: JmapErrorInvalidJmapRequestPayload, err: err}
	}

	return command(j.api, logger, ctx, session, j.onSessionOutdated, cmd, func(body *Response) (SubmittedEmail, Error) {
		var submissionResponse EmailSubmissionSetResponse
		err = retrieveResponseMatchParameters(body, EmailSubmissionSet, "0", &submissionResponse)
		if err != nil {
			return SubmittedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
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
		err = retrieveResponseMatchParameters(body, EmailSet, "0", &setResponse)
		if err != nil {
			return SubmittedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
		}

		var getResponse EmailSubmissionGetResponse
		err = retrieveResponseMatchParameters(body, EmailSubmissionGet, "1", &getResponse)
		if err != nil {
			return SubmittedEmail{}, SimpleError{code: JmapErrorInvalidJmapResponsePayload, err: err}
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
