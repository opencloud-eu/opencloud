package groupware

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/microcosm-cc/bluemonday"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/opencloud-eu/opencloud/services/groupware/pkg/metrics"
)

// When the request succeeds without a "since" query parameter.
// swagger:response GetAllEmailsInMailbox200
type SwaggerGetAllEmailsInMailbox200 struct {
	// in: body
	Body struct {
		*jmap.Emails
	}
}

// When the request succeeds with a "since" query parameter.
// swagger:response GetAllEmailsInMailboxSince200
type SwaggerGetAllEmailsInMailboxSince200 struct {
	// in: body
	Body struct {
		*jmap.MailboxChanges
	}
}

// swagger:route GET /groupware/accounts/{account}/mailboxes/{mailbox}/emails email get_all_emails_in_mailbox
// Get all the emails in a mailbox.
//
// Retrieve the list of all the emails that are in a given mailbox.
//
// The mailbox must be specified by its id, as part of the request URL path.
//
// A limit and an offset may be specified using the query parameters 'limit' and 'offset',
// respectively.
//
// When the query parameter 'since' or the 'if-none-match' header is specified, then the
// request behaves differently, performing a changes query to determine what has changed in
// that mailbox since a given state identifier.
//
// responses:
//
//		200: GetAllEmailsInMailbox200
//	 200: GetAllEmailsInMailboxSince200
//		400: ErrorResponse400
//		404: ErrorResponse404
//		500: ErrorResponse500
func (g *Groupware) GetAllEmailsInMailbox(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)
	since := r.Header.Get(HeaderSince)

	if since != "" {
		// ... then it's a completely different operation
		maxChanges := uint(0)
		g.respond(w, r, func(req Request) Response {
			if mailboxId == "" {
				return req.parameterErrorResponse(UriParamMailboxId, fmt.Sprintf("Missing required mailbox ID path parameter '%v'", UriParamMailboxId))
			}

			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return errorResponse(err)
			}

			logger := log.From(req.logger.With().Str(HeaderSince, log.SafeString(since)).Str(logAccountId, log.SafeString(accountId)))

			changes, sessionState, lang, jerr := g.jmap.GetMailboxChanges(accountId, req.session, req.ctx, logger, req.language(), mailboxId, since, true, g.maxBodyValueBytes, maxChanges)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(changes, sessionState, changes.State, lang)
		})
	} else {
		g.respond(w, r, func(req Request) Response {
			l := req.logger.With()
			if mailboxId == "" {
				return req.parameterErrorResponse(UriParamMailboxId, fmt.Sprintf("Missing required mailbox ID path parameter '%v'", UriParamMailboxId))
			}
			offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Uint(QueryParamOffset, offset)
			}

			limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaultEmailLimit)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Uint(QueryParamLimit, limit)
			}

			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return errorResponse(err)
			}
			l = l.Str(logAccountId, accountId)

			logger := log.From(l)

			emails, sessionState, lang, jerr := g.jmap.GetAllEmailsInMailbox(accountId, req.session, req.ctx, logger, req.language(), mailboxId, offset, limit, false, true, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			sanitized, err := req.sanitizeEmails(emails.Emails)
			if err != nil {
				return errorResponseWithSessionState(err, sessionState)
			}

			safe := jmap.Emails{
				Emails: sanitized,
				Total:  emails.Total,
				Limit:  emails.Limit,
				Offset: emails.Offset,
				State:  emails.State,
			}

			return etagResponse(safe, sessionState, emails.State, lang)
		})
	}
}

func (g *Groupware) GetEmailsById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, UriParamEmailId)
	ids := strings.Split(id, ",")

	accept := r.Header.Get("Accept")
	if accept == "message/rfc822" {
		g.stream(w, r, func(req Request, w http.ResponseWriter) *Error {
			if len(ids) != 1 {
				return req.parameterError(UriParamEmailId, fmt.Sprintf("when the Accept header is set to '%s', the API only supports serving a single email id", accept))
			}

			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return err
			}

			_, ok, err := req.parseBoolParam(QueryParamMarkAsSeen, false)
			if err != nil {
				return err
			}
			if ok {
				return req.parameterError(QueryParamMarkAsSeen, fmt.Sprintf("when the Accept header is set to '%s', the API does not support setting %s", accept, QueryParamMarkAsSeen))
			}

			logger := log.From(req.logger.With().Str(logAccountId, log.SafeString(accountId)).Str("id", log.SafeString(id)).Str("accept", log.SafeString(accept)))

			blobId, _, _, jerr := g.jmap.GetEmailBlobId(accountId, req.session, req.ctx, logger, req.language(), id)
			if jerr != nil {
				return req.apiErrorFromJmap(req.observeJmapError(jerr))
			}
			if blobId == "" {
				return nil
			} else {
				name := blobId + ".eml"
				typ := accept
				accountId, gwerr := req.GetAccountIdForBlob()
				if gwerr != nil {
					return gwerr
				}
				return req.serveBlob(blobId, name, typ, logger, accountId, w)
			}
		})
	} else {
		g.respond(w, r, func(req Request) Response {
			if len(ids) < 1 {
				return req.parameterErrorResponse(UriParamEmailId, fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamEmailId, log.SafeString(id), "empty list of mail ids"))
			}

			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return errorResponse(err)
			}
			l := req.logger.With().Str(logAccountId, log.SafeString(accountId))

			markAsSeen, ok, err := req.parseBoolParam(QueryParamMarkAsSeen, false)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Bool(QueryParamMarkAsSeen, markAsSeen)
			}

			if len(ids) == 1 {
				logger := log.From(l.Str("id", log.SafeString(id)))

				emails, sessionState, lang, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, req.language(), ids, true, g.maxBodyValueBytes, markAsSeen, true)
				if jerr != nil {
					return req.errorResponseFromJmap(jerr)
				}
				if len(emails.Emails) < 1 {
					return notFoundResponse(sessionState)
				} else {
					sanitized, err := req.sanitizeEmail(emails.Emails[0])
					if err != nil {
						return errorResponseWithSessionState(err, sessionState)
					}
					return etagResponse(sanitized, sessionState, emails.State, lang)
				}
			} else {
				logger := log.From(l.Array("ids", log.SafeStringArray(ids)))

				emails, sessionState, lang, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, req.language(), ids, true, g.maxBodyValueBytes, markAsSeen, false)
				if jerr != nil {
					return req.errorResponseFromJmap(jerr)
				}
				if len(emails.Emails) < 1 {
					return notFoundResponse(sessionState)
				} else {
					sanitized, err := req.sanitizeEmails(emails.Emails)
					if err != nil {
						return errorResponseWithSessionState(err, sessionState)
					}
					return etagResponse(sanitized, sessionState, emails.State, lang)
				}
			}
		})
	}
}

func (g *Groupware) GetEmailAttachments(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, UriParamEmailId)

	contextAppender := func(l zerolog.Context) zerolog.Context { return l }
	q := r.URL.Query()
	var attachmentSelector func(jmap.EmailBodyPart) bool = nil
	if q.Has(QueryParamPartId) {
		partId := q.Get(QueryParamPartId)
		attachmentSelector = func(part jmap.EmailBodyPart) bool { return part.PartId == partId }
		contextAppender = func(l zerolog.Context) zerolog.Context { return l.Str(QueryParamPartId, log.SafeString(partId)) }
	}
	if q.Has(QueryParamAttachmentName) {
		name := q.Get(QueryParamAttachmentName)
		attachmentSelector = func(part jmap.EmailBodyPart) bool { return part.Name == name }
		contextAppender = func(l zerolog.Context) zerolog.Context { return l.Str(QueryParamAttachmentName, log.SafeString(name)) }
	}
	if q.Has(QueryParamAttachmentBlobId) {
		blobId := q.Get(QueryParamAttachmentBlobId)
		attachmentSelector = func(part jmap.EmailBodyPart) bool { return part.BlobId == blobId }
		contextAppender = func(l zerolog.Context) zerolog.Context {
			return l.Str(QueryParamAttachmentBlobId, log.SafeString(blobId))
		}
	}

	if attachmentSelector == nil {
		g.respond(w, r, func(req Request) Response {
			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return errorResponse(err)
			}
			l := req.logger.With().Str(logAccountId, log.SafeString(accountId))
			logger := log.From(l)
			emails, sessionState, lang, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, req.language(), []string{id}, false, 0, false, false)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}
			if len(emails.Emails) < 1 {
				return notFoundResponse(sessionState)
			}
			email, err := req.sanitizeEmail(emails.Emails[0])
			if err != nil {
				return errorResponseWithSessionState(err, sessionState)
			}
			return etagResponse(email.Attachments, sessionState, emails.State, lang)
		})
	} else {
		g.stream(w, r, func(req Request, w http.ResponseWriter) *Error {
			mailAccountId, gwerr := req.GetAccountIdForMail()
			if gwerr != nil {
				return gwerr
			}
			blobAccountId, gwerr := req.GetAccountIdForBlob()
			if gwerr != nil {
				return gwerr
			}

			l := req.logger.With().Str(logAccountId, log.SafeString(mailAccountId)).Str(logBlobAccountId, log.SafeString(blobAccountId))
			l = contextAppender(l)
			logger := log.From(l)

			emails, _, lang, jerr := g.jmap.GetEmails(mailAccountId, req.session, req.ctx, logger, req.language(), []string{id}, false, 0, false, false)
			if jerr != nil {
				return req.apiErrorFromJmap(req.observeJmapError(jerr))
			}
			if len(emails.Emails) < 1 {
				return nil
			}

			email, err := req.sanitizeEmail(emails.Emails[0])
			if err != nil {
				return err
			}
			var attachment *jmap.EmailBodyPart = nil
			for _, part := range email.Attachments {
				if attachmentSelector(part) {
					attachment = &part
					break
				}
			}
			if attachment == nil {
				return nil
			}

			blob, lang, jerr := g.jmap.DownloadBlobStream(blobAccountId, attachment.BlobId, attachment.Name, attachment.Type, req.session, req.ctx, logger, req.language())
			if blob != nil && blob.Body != nil {
				defer func(Body io.ReadCloser) {
					err := Body.Close()
					if err != nil {
						logger.Error().Err(err).Msg("failed to close response body")
					}
				}(blob.Body)
			}
			if jerr != nil {
				return req.apiErrorFromJmap(jerr)
			}
			if blob == nil {
				w.WriteHeader(http.StatusNotFound)
				return nil
			}

			if blob.Type != "" {
				w.Header().Add("Content-Type", blob.Type)
			}
			if blob.CacheControl != "" {
				w.Header().Add("Cache-Control", blob.CacheControl)
			}
			if blob.ContentDisposition != "" {
				w.Header().Add("Content-Disposition", blob.ContentDisposition)
			}
			if blob.Size >= 0 {
				w.Header().Add("Content-Size", strconv.Itoa(blob.Size))
			}
			if lang != "" {
				w.Header().Add("Content-Language", string(lang))
			}
			_, cerr := io.Copy(w, blob.Body)
			if cerr != nil {
				return req.observedParameterError(ErrorStreamingResponse)
			}

			return nil
		})
	}
}

func (g *Groupware) getEmailsSince(w http.ResponseWriter, r *http.Request, since string) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(QueryParamSince, log.SafeString(since))
		maxChanges, ok, err := req.parseUIntParam(QueryParamMaxChanges, 0)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}
		l = l.Str(logAccountId, log.SafeString(accountId))

		logger := log.From(l)

		changes, sessionState, lang, jerr := g.jmap.GetEmailsSince(accountId, req.session, req.ctx, logger, req.language(), since, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(changes, sessionState, changes.State, lang)
	})
}

type EmailSearchSnippetsResults struct {
	Results    []Snippet  `json:"results,omitempty"`
	Total      uint       `json:"total,omitzero"`
	Limit      uint       `json:"limit,omitzero"`
	QueryState jmap.State `json:"queryState,omitempty"`
}

type EmailWithSnippets struct {
	AccountId string `json:"accountId,omitempty"`
	jmap.Email
	Snippets []SnippetWithoutEmailId `json:"snippets,omitempty"`
}

type Snippet struct {
	AccountId string `json:"accountId,omitempty"`
	jmap.SearchSnippetWithMeta
}

type SnippetWithoutEmailId struct {
	Subject string `json:"subject,omitempty"`
	Preview string `json:"preview,omitempty"`
}

type EmailWithSnippetsSearchResults struct {
	Results    []EmailWithSnippets `json:"results"`
	Total      uint                `json:"total,omitzero"`
	Limit      uint                `json:"limit,omitzero"`
	QueryState jmap.State          `json:"queryState,omitempty"`
}

type EmailSearchResults struct {
	Results    []jmap.Email `json:"results"`
	Total      uint         `json:"total,omitzero"`
	Limit      uint         `json:"limit,omitzero"`
	QueryState jmap.State   `json:"queryState,omitempty"`
}

func (g *Groupware) buildFilter(req Request) (bool, jmap.EmailFilterElement, bool, uint, uint, *log.Logger, *Error) {
	q := req.r.URL.Query()
	mailboxId := q.Get(QueryParamMailboxId)
	notInMailboxIds := q[QueryParamNotInMailboxId]
	text := q.Get(QueryParamSearchText)
	from := q.Get(QueryParamSearchFrom)
	to := q.Get(QueryParamSearchTo)
	cc := q.Get(QueryParamSearchCc)
	bcc := q.Get(QueryParamSearchBcc)
	subject := q.Get(QueryParamSearchSubject)
	body := q.Get(QueryParamSearchBody)
	keywords := q[QueryParamSearchKeyword]
	messageId := q.Get(QueryParamSearchMessageId)

	snippets := false

	l := req.logger.With()

	offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Uint(QueryParamOffset, offset)
	}

	limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaultEmailLimit)
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Uint(QueryParamLimit, limit)
	}

	before, ok, err := req.parseDateParam(QueryParamSearchBefore)
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Time(QueryParamSearchBefore, before)
	}

	after, ok, err := req.parseDateParam(QueryParamSearchAfter)
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Time(QueryParamSearchAfter, after)
	}

	if mailboxId != "" {
		l = l.Str(QueryParamMailboxId, log.SafeString(mailboxId))
	}
	if len(notInMailboxIds) > 0 {
		l = l.Array(QueryParamNotInMailboxId, log.SafeStringArray(notInMailboxIds))
	}
	if text != "" {
		l = l.Str(QueryParamSearchText, log.SafeString(text))
	}
	if from != "" {
		l = l.Str(QueryParamSearchFrom, log.SafeString(from))
	}
	if to != "" {
		l = l.Str(QueryParamSearchTo, log.SafeString(to))
	}
	if cc != "" {
		l = l.Str(QueryParamSearchCc, log.SafeString(cc))
	}
	if bcc != "" {
		l = l.Str(QueryParamSearchBcc, log.SafeString(bcc))
	}
	if subject != "" {
		l = l.Str(QueryParamSearchSubject, log.SafeString(subject))
	}
	if body != "" {
		l = l.Str(QueryParamSearchBody, log.SafeString(body))
	}
	if messageId != "" {
		l = l.Str(QueryParamSearchMessageId, log.SafeString(messageId))
	}

	minSize, ok, err := req.parseIntParam(QueryParamSearchMinSize, 0)
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Int(QueryParamSearchMinSize, minSize)
	}

	maxSize, ok, err := req.parseIntParam(QueryParamSearchMaxSize, 0)
	if err != nil {
		return false, nil, snippets, 0, 0, nil, err
	}
	if ok {
		l = l.Int(QueryParamSearchMaxSize, maxSize)
	}

	logger := log.From(l)

	var filter jmap.EmailFilterElement

	firstFilter := jmap.EmailFilterCondition{
		Text:               text,
		InMailbox:          mailboxId,
		InMailboxOtherThan: notInMailboxIds,
		From:               from,
		To:                 to,
		Cc:                 cc,
		Bcc:                bcc,
		Subject:            subject,
		Body:               body,
		Before:             before,
		After:              after,
		MinSize:            minSize,
		MaxSize:            maxSize,
		Header:             []string{},
	}
	if messageId != "" {
		// The array MUST contain either one or two elements.
		// The first element is the name of the header field to match against.
		// The second (optional) element is the text to look for in the header field value.
		// If not supplied, the message matches simply if it has a header field of the given name.
		firstFilter.Header = []string{"Message-ID", messageId}
	}
	filter = &firstFilter

	if text != "" || subject != "" || body != "" {
		snippets = true
	}

	if len(keywords) > 0 {
		firstFilter.HasKeyword = keywords[0]
		if len(keywords) > 1 {
			firstFilter.HasKeyword = keywords[0]
			filters := make([]jmap.EmailFilterElement, len(keywords)-1)
			for i, keyword := range keywords[1:] {
				filters[i] = jmap.EmailFilterCondition{HasKeyword: keyword}
			}
			filter = &jmap.EmailFilterOperator{
				Operator:   jmap.And,
				Conditions: filters,
			}
		}
	}

	return true, filter, snippets, offset, limit, logger, nil
}

func (g *Groupware) searchEmails(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, filter, makesSnippets, offset, limit, logger, err := g.buildFilter(req)
		if !ok {
			return errorResponse(err)
		}

		if !filter.IsNotEmpty() {
			filter = nil
		}

		fetchEmails, ok, err := req.parseBoolParam(QueryParamSearchFetchEmails, false)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			logger = log.From(logger.With().Bool(QueryParamSearchFetchEmails, fetchEmails))
		}

		if fetchEmails {
			fetchBodies, ok, err := req.parseBoolParam(QueryParamSearchFetchBodies, false)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				logger = log.From(logger.With().Bool(QueryParamSearchFetchBodies, fetchBodies))
			}

			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return errorResponse(err)
			}
			logger = log.From(logger.With().Str(logAccountId, log.SafeString(accountId)))

			resultsByAccount, sessionState, lang, jerr := g.jmap.QueryEmailsWithSnippets([]string{accountId}, filter, req.session, req.ctx, logger, req.language(), offset, limit, fetchBodies, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			if results, ok := resultsByAccount[accountId]; ok {
				flattened := make([]EmailWithSnippets, len(results.Results))
				for i, result := range results.Results {
					var snippets []SnippetWithoutEmailId
					if makesSnippets {
						snippets := make([]SnippetWithoutEmailId, len(result.Snippets))
						for j, snippet := range result.Snippets {
							snippets[j] = SnippetWithoutEmailId{
								Subject: snippet.Subject,
								Preview: snippet.Preview,
							}
						}
					} else {
						snippets = nil
					}
					sanitized, err := req.sanitizeEmail(result.Email)
					if err != nil {
						return errorResponseWithSessionState(err, sessionState)
					}
					flattened[i] = EmailWithSnippets{
						// AccountId: accountId,
						Email:    sanitized,
						Snippets: snippets,
					}
				}

				return etagResponse(EmailWithSnippetsSearchResults{
					Results:    flattened,
					Total:      results.Total,
					Limit:      results.Limit,
					QueryState: results.QueryState,
				}, sessionState, results.QueryState, lang)
			} else {
				return notFoundResponse(sessionState)
			}
		} else {
			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return errorResponse(err)
			}
			logger = log.From(logger.With().Str(logAccountId, log.SafeString(accountId)))

			resultsByAccountId, sessionState, lang, jerr := g.jmap.QueryEmailSnippets([]string{accountId}, filter, req.session, req.ctx, logger, req.language(), offset, limit)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			if results, ok := resultsByAccountId[accountId]; ok {
				return etagResponse(EmailSearchSnippetsResults{
					Results:    structs.Map(results.Snippets, func(s jmap.SearchSnippetWithMeta) Snippet { return Snippet{SearchSnippetWithMeta: s} }),
					Total:      results.Total,
					Limit:      results.Limit,
					QueryState: results.QueryState,
				}, sessionState, results.QueryState, lang)
			} else {
				return notFoundResponse(sessionState)
			}
		}
	})
}

func (g *Groupware) GetEmails(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since := q.Get(QueryParamSince)
	if since == "" {
		since = r.Header.Get(HeaderSince)
	}
	if since != "" {
		// get email changes since a given state
		g.getEmailsSince(w, r, since)
	} else {
		// do a search
		g.searchEmails(w, r)
	}
}

func (g *Groupware) GetEmailsForAllAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, filter, makesSnippets, offset, limit, logger, err := g.buildFilter(req)
		if !ok {
			return errorResponse(err)
		}

		if !filter.IsNotEmpty() {
			filter = nil
		}

		fetchEmails, ok, err := req.parseBoolParam(QueryParamSearchFetchEmails, false)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			logger = log.From(logger.With().Bool(QueryParamSearchFetchEmails, fetchEmails))
		}

		allAccountIds := structs.Keys(req.session.Accounts) // TODO(pbleser-oc) do we need a limit for a maximum amount of accounts to query at once?
		logger = log.From(logger.With().Array(logAccountId, log.SafeStringArray(allAccountIds)))

		if fetchEmails {
			fetchBodies, ok, err := req.parseBoolParam(QueryParamSearchFetchBodies, false)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				logger = log.From(logger.With().Bool(QueryParamSearchFetchBodies, fetchBodies))
			}

			if makesSnippets {
				resultsByAccountId, sessionState, lang, jerr := g.jmap.QueryEmailsWithSnippets(allAccountIds, filter, req.session, req.ctx, logger, req.language(), offset, limit, fetchBodies, g.maxBodyValueBytes)
				if jerr != nil {
					return req.errorResponseFromJmap(jerr)
				}

				flattenedByAccountId := make(map[string][]EmailWithSnippets, len(resultsByAccountId))
				total := 0
				var totalOverAllAccounts uint = 0
				for accountId, results := range resultsByAccountId {
					totalOverAllAccounts += results.Total
					flattened := make([]EmailWithSnippets, len(results.Results))
					for i, result := range results.Results {
						snippets := structs.MapN(result.Snippets, func(s jmap.SearchSnippet) *SnippetWithoutEmailId {
							if s.Subject != "" || s.Preview != "" {
								return &SnippetWithoutEmailId{
									Subject: s.Subject,
									Preview: s.Preview,
								}
							} else {
								return nil
							}
						})

						sanitized, err := req.sanitizeEmail(result.Email)
						if err != nil {
							return errorResponseWithSessionState(err, sessionState)
						}
						flattened[i] = EmailWithSnippets{
							AccountId: accountId,
							Email:     sanitized,
							Snippets:  snippets,
						}
					}
					flattenedByAccountId[accountId] = flattened
					total += len(flattened)
				}

				flattened := make([]EmailWithSnippets, total)
				{
					i := 0
					for _, list := range flattenedByAccountId {
						for _, e := range list {
							flattened[i] = e
							i++
						}
					}
				}

				slices.SortFunc(flattened, func(a, b EmailWithSnippets) int { return a.ReceivedAt.Compare(b.ReceivedAt) })
				squashedQueryState := squashQueryState(resultsByAccountId, func(e jmap.EmailQueryWithSnippetsResult) jmap.State { return e.QueryState })

				// TODO offset and limit over the aggregated results by account

				return etagResponse(EmailWithSnippetsSearchResults{
					Results:    flattened,
					Total:      totalOverAllAccounts,
					Limit:      limit,
					QueryState: squashedQueryState,
				}, sessionState, squashedQueryState, lang)
			} else {
				resultsByAccountId, sessionState, lang, jerr := g.jmap.QueryEmails(allAccountIds, filter, req.session, req.ctx, logger, req.language(), offset, limit, fetchBodies, g.maxBodyValueBytes)
				if jerr != nil {
					return req.errorResponseFromJmap(jerr)
				}

				total := 0
				var totalOverAllAccounts uint = 0
				for _, results := range resultsByAccountId {
					totalOverAllAccounts += results.Total
					total += len(results.Emails)
				}

				flattened := make([]jmap.Email, total)
				{
					i := 0
					for _, list := range resultsByAccountId {
						for _, e := range list.Emails {
							sanitized, err := req.sanitizeEmail(e)
							if err != nil {
								return errorResponseWithSessionState(err, sessionState)
							}
							flattened[i] = sanitized
							i++
						}
					}
				}

				slices.SortFunc(flattened, func(a, b jmap.Email) int { return a.ReceivedAt.Compare(b.ReceivedAt) })
				squashedQueryState := squashQueryState(resultsByAccountId, func(e jmap.EmailQueryResult) jmap.State { return e.QueryState })

				// TODO offset and limit over the aggregated results by account

				return etagResponse(EmailSearchResults{
					Results:    flattened,
					Total:      totalOverAllAccounts,
					Limit:      limit,
					QueryState: squashedQueryState,
				}, sessionState, squashedQueryState, lang)
			}
		} else {
			if makesSnippets {
				resultsByAccountId, sessionState, lang, jerr := g.jmap.QueryEmailSnippets(allAccountIds, filter, req.session, req.ctx, logger, req.language(), offset, limit)
				if jerr != nil {
					return req.errorResponseFromJmap(jerr)
				}

				var totalOverAllAccounts uint = 0
				total := 0
				for _, results := range resultsByAccountId {
					totalOverAllAccounts += results.Total
					total += len(results.Snippets)
				}

				flattened := make([]Snippet, total)
				{
					i := 0
					for accountId, results := range resultsByAccountId {
						for _, result := range results.Snippets {
							flattened[i] = Snippet{
								AccountId:             accountId,
								SearchSnippetWithMeta: result,
							}
						}
					}
				}

				slices.SortFunc(flattened, func(a, b Snippet) int { return a.ReceivedAt.Compare(b.ReceivedAt) })

				// TODO offset and limit over the aggregated results by account

				squashedQueryState := squashQueryState(resultsByAccountId, func(e jmap.EmailSnippetQueryResult) jmap.State { return e.QueryState })

				return etagResponse(EmailSearchSnippetsResults{
					Results:    flattened,
					Total:      totalOverAllAccounts,
					Limit:      limit,
					QueryState: squashedQueryState,
				}, sessionState, squashedQueryState, lang)
			} else {
				// TODO implement search without email bodies (only retrieve a few chosen properties?) + without snippets
				return notImplementesResponse()
			}
		}
	})
}

/*
type EmailCreation struct {
	MailboxIds    []string                       `json:"mailboxIds,omitempty"`
	Keywords      []string                       `json:"keywords,omitempty"`
	From          []jmap.EmailAddress            `json:"from,omitempty"`
	Subject       string                         `json:"subject,omitempty"`
	ReceivedAt    time.Time                      `json:"receivedAt,omitzero"`
	SentAt        time.Time                      `json:"sentAt,omitzero"` // huh?
	BodyStructure jmap.EmailBodyStructure        `json:"bodyStructure"`
	BodyValues    map[string]jmap.EmailBodyValue `json:"bodyValues,omitempty"`
}
*/

func (g *Groupware) CreateEmail(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		logger := req.logger

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		logger = log.From(logger.With().Str(logAccountId, log.SafeString(accountId)))

		var body jmap.Email
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		create := jmap.EmailCreate{
			MailboxIds:    body.MailboxIds,
			Keywords:      body.Keywords,
			From:          body.From,
			Subject:       body.Subject,
			ReceivedAt:    body.ReceivedAt,
			SentAt:        body.SentAt,
			BodyStructure: body.BodyStructure,
			BodyValues:    body.BodyValues,
		}

		created, sessionState, lang, jerr := g.jmap.CreateEmail(accountId, create, "", req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(created.Email, sessionState, lang)
	})
}

func (g *Groupware) ReplaceEmail(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		logger := req.logger

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}

		replaceId := chi.URLParam(r, UriParamEmailId)

		logger = log.From(logger.With().Str(logAccountId, log.SafeString(accountId)))

		var body jmap.Email
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		create := jmap.EmailCreate{
			MailboxIds:    body.MailboxIds,
			Keywords:      body.Keywords,
			From:          body.From,
			Subject:       body.Subject,
			ReceivedAt:    body.ReceivedAt,
			SentAt:        body.SentAt,
			BodyStructure: body.BodyStructure,
			BodyValues:    body.BodyValues,
		}

		created, sessionState, lang, jerr := g.jmap.CreateEmail(accountId, create, replaceId, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(created.Email, sessionState, lang)
	})
}

// swagger:parameters update_email
type SwaggerUpdateEmailBody struct {
	// List of identifiers of emails to delete.
	// in: body
	// example: ["caen3iujoo8u", "aec8phaetaiz", "bohna0me"]
	Body map[string]string
}

func (g *Groupware) UpdateEmail(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		emailId := chi.URLParam(r, UriParamEmailId)

		l := req.logger.With()
		l.Str(UriParamEmailId, log.SafeString(emailId))

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		l.Str(logAccountId, accountId)

		logger := log.From(l)

		var body map[string]any
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		updates := map[string]jmap.EmailUpdate{
			emailId: body,
		}

		result, sessionState, lang, jerr := g.jmap.UpdateEmails(accountId, updates, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if result.Updated == nil {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response",
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint")))
		}
		updatedEmail, ok := result.Updated[emailId]
		if !ok {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID",
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint")))
		}

		return response(updatedEmail, sessionState, lang)
	})
}

type emailKeywordUpdates struct {
	Add    []string `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

func (e emailKeywordUpdates) IsEmpty() bool {
	return len(e.Add) == 0 && len(e.Remove) == 0
}

func (g *Groupware) UpdateEmailKeywords(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		emailId := chi.URLParam(r, UriParamEmailId)

		l := req.logger.With()
		l.Str(UriParamEmailId, log.SafeString(emailId))

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		l.Str(logAccountId, accountId)

		logger := log.From(l)

		var body emailKeywordUpdates
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		if body.IsEmpty() {
			return noContentResponse(req.session.State)
		}

		patch := jmap.EmailUpdate{}
		for _, keyword := range body.Add {
			patch["keywords/"+keyword] = true
		}
		for _, keyword := range body.Remove {
			patch["keywords/"+keyword] = nil
		}
		patches := map[string]jmap.EmailUpdate{
			emailId: patch,
		}

		result, sessionState, lang, jerr := g.jmap.UpdateEmails(accountId, patches, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if result.Updated == nil {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response",
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint")))
		}
		updatedEmail, ok := result.Updated[emailId]
		if !ok {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID",
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint")))
		}

		return response(updatedEmail, sessionState, lang)
	})
}

// swagger:route POST /groupware/accounts/{account}/emails/{emailid}/keywords email add_email_keywords
// Add keywords to an email by its unique identifier.
//
// responses:
//
//	204: Success204
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) AddEmailKeywords(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		emailId := chi.URLParam(r, UriParamEmailId)

		l := req.logger.With()
		l.Str(UriParamEmailId, log.SafeString(emailId))

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		l.Str(logAccountId, accountId)

		logger := log.From(l)

		var body []string
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		if len(body) < 1 {
			return noContentResponse(req.session.State)
		}

		patch := jmap.EmailUpdate{}
		for _, keyword := range body {
			patch["keywords/"+keyword] = true
		}
		patches := map[string]jmap.EmailUpdate{
			emailId: patch,
		}

		result, sessionState, lang, jerr := g.jmap.UpdateEmails(accountId, patches, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if result.Updated == nil {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response",
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint")))
		}
		updatedEmail, ok := result.Updated[emailId]
		if !ok {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID",
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint")))
		}

		if updatedEmail == nil {
			return noContentResponseWithEtag(sessionState, result.State)
		} else {
			return response(updatedEmail, sessionState, lang)
		}
	})
}

// swagger:route DELETE /groupware/accounts/{account}/emails/{emailid}/keywords email remove_email_keywords
// Remove keywords of an email by its unique identifier.
//
// responses:
//
//	204: Success204
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) RemoveEmailKeywords(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		emailId := chi.URLParam(r, UriParamEmailId)

		l := req.logger.With()
		l.Str(UriParamEmailId, log.SafeString(emailId))

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		l.Str(logAccountId, accountId)

		logger := log.From(l)

		var body []string
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		if len(body) < 1 {
			return noContentResponse(req.session.State)
		}

		patch := jmap.EmailUpdate{}
		for _, keyword := range body {
			patch["keywords/"+keyword] = nil
		}
		patches := map[string]jmap.EmailUpdate{
			emailId: patch,
		}

		result, sessionState, lang, jerr := g.jmap.UpdateEmails(accountId, patches, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if result.Updated == nil {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response",
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint")))
		}
		updatedEmail, ok := result.Updated[emailId]
		if !ok {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID",
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint")))
		}

		if updatedEmail == nil {
			return noContentResponseWithEtag(sessionState, result.State)
		} else {
			return response(updatedEmail, sessionState, lang)
		}
	})
}

// swagger:route DELETE /groupware/accounts/{account}/emails/{emailid} email delete_email
// Delete an email by its unique identifier.
//
// responses:
//
//	204: Success204
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) DeleteEmail(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		emailId := chi.URLParam(r, UriParamEmailId)

		l := req.logger.With()
		l.Str(UriParamEmailId, emailId)

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		l.Str(logAccountId, accountId)

		logger := log.From(l)

		resp, sessionState, _, jerr := g.jmap.DeleteEmails(accountId, []string{emailId}, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		for _, e := range resp.NotDestroyed {
			desc := e.Description
			if desc != "" {
				return errorResponseWithSessionState(apiError(
					req.errorId(),
					ErrorFailedToDeleteEmail,
					withDetail(e.Description),
				), sessionState)
			} else {
				return errorResponseWithSessionState(apiError(
					req.errorId(),
					ErrorFailedToDeleteEmail,
				), sessionState)
			}
		}
		return noContentResponseWithEtag(sessionState, resp.State)
	})
}

// swagger:parameters delete_emails
type SwaggerDeleteEmailsBody struct {
	// List of identifiers of emails to delete.
	// in: body
	// example: ["caen3iujoo8u", "aec8phaetaiz", "bohna0me"]
	Body []string
}

// swagger:route DELETE /groupware/accounts/{account}/emails email delete_emails
// Delete a set of emails by their unique identifiers.
//
// The identifiers of the emails to delete are specified as part of the request
// body, as an array of strings.
//
// responses:
//
//	204: Success204
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) DeleteEmails(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		var emailIds []string
		err := req.body(&emailIds)
		if err != nil {
			return errorResponse(err)
		}

		l := req.logger.With()
		l.Array("emailIds", log.SafeStringArray(emailIds))

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		l.Str(logAccountId, accountId)

		logger := log.From(l)

		resp, sessionState, _, jerr := g.jmap.DeleteEmails(accountId, emailIds, req.session, req.ctx, logger, req.language())
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if len(resp.NotDestroyed) > 0 {
			meta := make(map[string]any, len(resp.NotDestroyed))
			for emailId, e := range resp.NotDestroyed {
				meta[emailId] = e.Description
			}
			return errorResponseWithSessionState(apiError(
				req.errorId(),
				ErrorFailedToDeleteEmail,
				withMeta(meta),
			), sessionState)
		}
		return noContentResponseWithEtag(sessionState, resp.State)
	})
}

type AboutEmailsEvent struct {
	Id       string        `json:"id"`
	Source   string        `json:"source"`
	Emails   []jmap.Email  `json:"emails"`
	Language jmap.Language `json:"lang"`
}

type AboutEmailResponse struct {
	Email     jmap.Email    `json:"email"`
	RequestId string        `json:"requestId"`
	Language  jmap.Language `json:"lang"`
}

func relatedEmailsFilter(email jmap.Email, beacon time.Time, days uint) jmap.EmailFilterElement {
	filters := []jmap.EmailFilterElement{}
	for _, from := range email.From {
		if from.Email != "" {
			filters = append(filters, jmap.EmailFilterCondition{From: from.Email})
		}
	}
	for _, sender := range email.Sender {
		if sender.Email != "" {
			filters = append(filters, jmap.EmailFilterCondition{From: sender.Email})
		}
	}

	timeFilter := jmap.EmailFilterCondition{
		Before: beacon.Add(time.Duration(days) * time.Hour * 24),
		After:  beacon.Add(time.Duration(-days) * time.Hour * 24),
	}

	var filter jmap.EmailFilterElement
	if len(filters) > 0 {
		filter = jmap.EmailFilterOperator{
			Operator: jmap.And,
			Conditions: []jmap.EmailFilterElement{
				timeFilter,
				jmap.EmailFilterOperator{
					Operator:   jmap.Or,
					Conditions: filters,
				},
			},
		}
	} else {
		filter = timeFilter
	}

	return filter
}

func (g *Groupware) RelatedToEmail(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, UriParamEmailId)

	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(logEmailId, log.SafeString(id))

		limit, ok, err := req.parseUIntParam(QueryParamLimit, 10) // TODO configurable default limit
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint("limit", limit)
		}

		days, ok, err := req.parseUIntParam(QueryParamDays, 5) // TODO configurable default days
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint("days", days)
		}

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		l = l.Str(logAccountId, log.SafeString(accountId))

		logger := log.From(l)

		reqId := req.GetRequestId()
		getEmailsBefore := time.Now()
		emails, sessionState, lang, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, req.language(), []string{id}, true, g.maxBodyValueBytes, false, false)
		getEmailsDuration := time.Since(getEmailsBefore)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if len(emails.Emails) < 1 {
			req.observe(g.metrics.EmailByIdDuration.WithLabelValues(req.session.JmapEndpoint, metrics.Values.Result.NotFound), getEmailsDuration.Seconds())
			logger.Trace().Msg("failed to find any emails matching id") // the id is already in the log field
			return notFoundResponse(sessionState)
		} else {
			req.observe(g.metrics.EmailByIdDuration.WithLabelValues(req.session.JmapEndpoint, metrics.Values.Result.Found), getEmailsDuration.Seconds())
		}

		email := emails.Emails[0]

		beacon := email.ReceivedAt // TODO configurable: either relative to when the email was received, or relative to now
		//beacon := time.Now()
		filter := relatedEmailsFilter(email, beacon, days)

		// bgctx, _ := context.WithTimeout(context.Background(), time.Duration(30)*time.Second) // TODO configurable
		bgctx := context.Background()

		g.job(logger, RelationTypeSameSender, func(jobId uint64, l *log.Logger) {
			before := time.Now()
			resultsByAccountId, _, lang, jerr := g.jmap.QueryEmails([]string{accountId}, filter, req.session, bgctx, l, req.language(), 0, limit, false, g.maxBodyValueBytes)
			if results, ok := resultsByAccountId[accountId]; ok {
				duration := time.Since(before)
				if jerr != nil {
					req.observeJmapError(jerr)
					l.Error().Err(jerr).Msgf("failed to query %v emails", RelationTypeSameSender)
				} else {
					req.observe(g.metrics.EmailSameSenderDuration.WithLabelValues(req.session.JmapEndpoint), duration.Seconds())
					related, err := req.sanitizeEmails(filterEmails(results.Emails, email))
					if err == nil {
						l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameSender, len(related))
						if len(related) > 0 {
							req.push(RelationEntityEmail, AboutEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameSender, Language: lang})
						}
					}
				}
			}
		})

		g.job(logger, RelationTypeSameThread, func(jobId uint64, l *log.Logger) {
			before := time.Now()
			emails, _, _, jerr := g.jmap.EmailsInThread(accountId, email.ThreadId, req.session, bgctx, l, req.language(), false, g.maxBodyValueBytes)
			duration := time.Since(before)
			if jerr != nil {
				req.observeJmapError(jerr)
				l.Error().Err(jerr).Msgf("failed to list %v emails", RelationTypeSameThread)
			} else {
				req.observe(g.metrics.EmailSameThreadDuration.WithLabelValues(req.session.JmapEndpoint), duration.Seconds())
				related, err := req.sanitizeEmails(filterEmails(emails, email))
				if err == nil {
					l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameThread, len(related))
					if len(related) > 0 {
						req.push(RelationEntityEmail, AboutEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameThread, Language: lang})
					}
				}
			}
		})

		sanitized, err := req.sanitizeEmail(email)
		if err != nil {
			return errorResponseWithSessionState(err, sessionState)
		}
		return etagResponse(AboutEmailResponse{
			Email:     sanitized,
			RequestId: reqId,
		}, sessionState, emails.State, lang)
	})
}

type EmailSummary struct {
	// The id of the account this Email summary pertains to.
	// required: true
	// example: $accountId
	AccountId string `json:"accountId,omitempty"`

	// The id of the Email object.
	//
	// Note that this is the JMAP object id, NOT the Message-ID header field value of the message [RFC5322].
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	//
	// required: true
	// example: $emailId
	Id string `json:"id,omitempty"`

	// The id of the Thread to which this Email belongs.
	//
	// example: $threadId
	ThreadId string `json:"threadId,omitempty"`

	// The number of emails in the thread, including this one.
	ThreadSize int `json:"threadSize,omitzero"`

	// The set of Mailbox ids this Email belongs to.
	//
	// An Email in the mail store MUST belong to one or more Mailboxes at all times (until it is destroyed).
	// The set is represented as an object, with each key being a Mailbox id.
	//
	// The value for each key in the object MUST be true.
	//
	// example: $mailboxIds
	MailboxIds map[string]bool `json:"mailboxIds,omitempty"`

	// A set of keywords that apply to the Email.
	//
	// The set is represented as an object, with the keys being the keywords.
	//
	// The value for each key in the object MUST be true.
	//
	// Keywords are shared with IMAP.
	//
	// The six system keywords from IMAP get special treatment.
	//
	// The following four keywords have their first character changed from \ in IMAP to $ in JMAP and have particular semantic meaning:
	//
	//   - $draft: The Email is a draft the user is composing.
	//   - $seen: The Email has been read.
	//   - $flagged: The Email has been flagged for urgent/special attention.
	//   - $answered: The Email has been replied to.
	//
	// The IMAP \Recent keyword is not exposed via JMAP. The IMAP \Deleted keyword is also not present: IMAP uses a delete+expunge model,
	// which JMAP does not. Any message with the \Deleted keyword MUST NOT be visible via JMAP (and so are not counted in the
	// “totalEmails”, “unreadEmails”, “totalThreads”, and “unreadThreads” Mailbox properties).
	//
	// Users may add arbitrary keywords to an Email.
	// For compatibility with IMAP, a keyword is a case-insensitive string of 1–255 characters in the ASCII subset
	// %x21–%x7e (excludes control chars and space), and it MUST NOT include any of these characters:
	//
	//    ( ) { ] % * " \
	//
	// Because JSON is case sensitive, servers MUST return keywords in lowercase.
	//
	// The [IMAP and JMAP Keywords] registry as established in [RFC5788] assigns semantic meaning to some other
	// keywords in common use.
	//
	// New keywords may be established here in the future. In particular, note:
	//
	//   - $forwarded: The Email has been forwarded.
	//   - $phishing: The Email is highly likely to be phishing.
	//     Clients SHOULD warn users to take care when viewing this Email and disable links and attachments.
	//   - $junk: The Email is definitely spam.
	//     Clients SHOULD set this flag when users report spam to help train automated spam-detection systems.
	//   - $notjunk: The Email is definitely not spam.
	//     Clients SHOULD set this flag when users indicate an Email is legitimate, to help train automated spam-detection systems.
	//
	// [IMAP and JMAP Keywords]: https://www.iana.org/assignments/imap-jmap-keywords/
	// [RFC5788]: https://www.rfc-editor.org/rfc/rfc5788.html
	//
	// example: $emailKeywords
	Keywords map[string]bool `json:"keywords,omitempty"`

	// The size, in octets, of the raw data for the message [RFC5322]
	// (as referenced by the blobId, i.e., the number of octets in the file the user would download).
	//
	// [RFC5322]: https://www.rfc-editor.org/rfc/rfc5322.html
	Size int `json:"size"`

	// The date the Email was received by the message store.
	//
	// This is the internal date in IMAP [RFC3501].
	//
	// [RFC3501]: https://www.rfc-editor.org/rfc/rfc3501.html
	//
	// example: $emailReceivedAt
	ReceivedAt time.Time `json:"receivedAt,omitzero"`

	// The value is identical to the value of header:Sender:asAddresses.
	// example: $emailSenders
	Sender []jmap.EmailAddress `json:"sender,omitempty"`

	// The value is identical to the value of header:From:asAddresses.
	// example: $emailFroms
	From []jmap.EmailAddress `json:"from,omitempty"`

	// The value is identical to the value of header:To:asAddresses.
	// example: $emailTos
	To []jmap.EmailAddress `json:"to,omitempty"`

	// The value is identical to the value of header:Cc:asAddresses.
	// example: $emailCCs
	Cc []jmap.EmailAddress `json:"cc,omitempty"`

	// The value is identical to the value of header:Bcc:asAddresses.
	// example: $emailBCCs
	Bcc []jmap.EmailAddress `json:"bcc,omitempty"`

	// The value is identical to the value of header:Subject:asText.
	// example: $emailSubject
	Subject string `json:"subject,omitempty"`

	// The value is identical to the value of header:Date:asDate.
	// example: $emailSentAt
	SentAt time.Time `json:"sentAt,omitzero"`

	// This is true if there are one or more parts in the message that a client UI should offer as downloadable.
	//
	// A server SHOULD set hasAttachment to true if the attachments list contains at least one item that
	// does not have Content-Disposition: inline.
	//
	// The server MAY ignore parts in this list that are processed automatically in some way or are referenced
	// as embedded images in one of the text/html parts of the message.
	//
	// The server MAY set hasAttachment based on implementation-defined or site-configurable heuristics.
	// example: true
	HasAttachment bool `json:"hasAttachment,omitempty"`

	// A list, traversing depth-first, of all parts in bodyStructure.
	//
	// They must satisfy either of the following conditions:
	//
	//   - not of type multipart/* and not included in textBody or htmlBody
	//   - of type image/*, audio/*, or video/* and not in both textBody and htmlBody
	//
	// None of these parts include subParts, including message/* types.
	//
	// Attached messages may be fetched using the Email/parse method and the blobId.
	//
	// Note that a text/html body part HTML may reference image parts in attachments by using cid:
	// links to reference the Content-Id, as defined in [RFC2392], or by referencing the Content-Location.
	//
	// [RFC2392]: https://www.rfc-editor.org/rfc/rfc2392.html
	//
	// example: $emailAttachments
	Attachments []jmap.EmailBodyPart `json:"attachments,omitempty"`

	// A plaintext fragment of the message body.
	//
	// This is intended to be shown as a preview line when listing messages in the mail store and may be truncated
	// when shown.
	//
	// The server may choose which part of the message to include in the preview; skipping quoted sections and
	// salutations and collapsing white space can result in a more useful preview.
	//
	// This MUST NOT be more than 256 characters in length.
	//
	// As this is derived from the message content by the server, and the algorithm for doing so could change over
	// time, fetching this for an Email a second time MAY return a different result.
	// However, the previous value is not considered incorrect, and the change SHOULD NOT cause the Email object
	// to be considered as changed by the server.
	//
	// example: $emailPreview
	Preview string `json:"preview,omitempty"`
}

func summarizeEmail(accountId string, email jmap.Email) EmailSummary {
	return EmailSummary{
		AccountId:     accountId,
		Id:            email.Id,
		ThreadId:      email.ThreadId,
		ThreadSize:    email.ThreadSize,
		MailboxIds:    email.MailboxIds,
		Keywords:      email.Keywords,
		Size:          email.Size,
		ReceivedAt:    email.ReceivedAt,
		Sender:        email.Sender,
		From:          email.From,
		To:            email.To,
		Cc:            email.Cc,
		Bcc:           email.Bcc,
		Subject:       email.Subject,
		SentAt:        email.SentAt,
		HasAttachment: email.HasAttachment,
		Attachments:   email.Attachments,
		Preview:       email.Preview,
	}
}

type emailWithAccountId struct {
	accountId string
	email     jmap.Email
}

// When the request succeeds.
// swagger:response GetLatestEmailsSummaryForAllAccounts200
type SwaggerGetLatestEmailsSummaryForAllAccounts200 struct {
	// in: body
	Body []EmailSummary
}

// swagger:parameters get_latest_emails_summary_for_all_accounts
type SwaggerGetLatestEmailsSummaryForAllAccountsParams struct {
	// The maximum amount of email summaries to return.
	// in: query
	// example: 10
	// default: 10
	Limit uint `json:"limit"`

	// Whether to include emails that have already been seen (read) or not.
	// in: query
	// example: true
	// default: false
	Seen bool `json:"seen"`

	// Whether to include emails that have been flagged as junk or phishing.
	// in: query
	// example: false
	// default: false
	Undesirable bool `json:"undesirable"`
}

// swagger:route GET /groupware/accounts/all/emails/latest/summary email get_latest_emails_summary_for_all_accounts
// Get a summary of the latest emails across all the mailboxes, across all of a user's accounts.
//
// Retrieves summaries of the latest emails of a user, in all accounts, across all mailboxes.
//
// The number of total summaries to retrieve is specified using the query parameter `limit`.
//
// The following additional query parameters may be specified to further filter the emails to summarize:
//
// !- `seen`: when `true`, emails that have already been seen (read) will be included as well (default is to only include emails that have not been read yet)
// !- `undesirable`: when `true`, emails that are flagged as spam or phishing will also be summarized (default is to ignore those)
//
// responses:
//
//	200: GetLatestEmailsSummaryForAllAccounts200
//	400: ErrorResponse400
//	404: ErrorResponse404
//	500: ErrorResponse500
func (g *Groupware) GetLatestEmailsSummaryForAllAccounts(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With()
		limit, ok, err := req.parseUIntParam(QueryParamLimit, 10) // TODO from configuration
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamLimit, limit)
		}

		seen, ok, err := req.parseBoolParam(QueryParamSeen, false)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Bool(QueryParamSeen, seen)
		}

		undesirable, ok, err := req.parseBoolParam(QueryParamUndesirable, false)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Bool(QueryParamUndesirable, undesirable)
		}

		var filter jmap.EmailFilterElement = nil // all emails, read and unread
		{
			notKeywords := []string{}
			if !seen {
				notKeywords = append(notKeywords, jmap.JmapKeywordSeen)
			}
			if undesirable {
				notKeywords = append(notKeywords, jmap.JmapKeywordJunk, jmap.JmapKeywordPhishing)
			}
			filter = filterFromNotKeywords(notKeywords)
		}

		allAccountIds := structs.Keys(req.session.Accounts) // TODO(pbleser-oc) do we need a limit for a maximum amount of accounts to query at once?
		l.Array(logAccountId, log.SafeStringArray(allAccountIds))

		logger := log.From(l)

		emailsSummariesByAccount, sessionState, lang, jerr := g.jmap.QueryEmailSummaries(allAccountIds, req.session, req.ctx, logger, req.language(), filter, limit, true)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		// sort in memory to respect the overall limit
		total := uint(0)
		for _, emails := range emailsSummariesByAccount {
			total += uint(max(len(emails.Emails), 0))
		}
		all := make([]emailWithAccountId, total)
		i := uint(0)
		for accountId, emails := range emailsSummariesByAccount {
			for _, email := range emails.Emails {
				all[i] = emailWithAccountId{accountId: accountId, email: email}
				i++
			}
		}

		slices.SortFunc(all, func(a, b emailWithAccountId) int { return -(a.email.ReceivedAt.Compare(b.email.ReceivedAt)) })

		summaries := make([]EmailSummary, min(limit, total))
		for i = 0; i < limit && i < total; i++ {
			summaries[i] = summarizeEmail(all[i].accountId, all[i].email)
		}

		return response(summaries, sessionState, lang)
	})
}

func filterEmails(all []jmap.Email, skip jmap.Email) []jmap.Email {
	filtered := all[:0]
	for _, email := range all {
		if skip.Id != email.Id {
			filtered = append(filtered, email)
		}
	}
	return filtered
}

func filterFromNotKeywords(keywords []string) jmap.EmailFilterElement {
	switch len(keywords) {
	case 0:
		return nil
	case 1:
		return jmap.EmailFilterCondition{NotKeyword: keywords[0]}
	default:
		conditions := make([]jmap.EmailFilterElement, len(keywords))
		for i, keyword := range keywords {
			conditions[i] = jmap.EmailFilterCondition{NotKeyword: keyword}
		}
		return jmap.EmailFilterOperator{Operator: jmap.And, Conditions: conditions}
	}
}

func squashQueryState[V any](all map[string]V, mapper func(V) jmap.State) jmap.State {
	n := len(all)
	if n == 0 {
		return jmap.State("")
	}
	if n == 1 {
		for _, v := range all {
			return mapper(v)
		}
	}

	parts := make([]string, n)
	sortedKeys := make([]string, n)
	i := 0
	for k := range all {
		sortedKeys[i] = k
		i++
	}
	slices.Sort(sortedKeys)
	for i, k := range sortedKeys {
		if v, ok := all[k]; ok {
			parts[i] = k + ":" + string(mapper(v))
		} else {
			parts[i] = k + ":"
		}
	}
	return jmap.State(strings.Join(parts, ","))
}

var sanitizationPolicy *bluemonday.Policy = bluemonday.UGCPolicy()

var sanitizableMediaTypes = []string{
	"text/html",
	"text/xhtml",
}

func (req *Request) sanitizeEmail(source jmap.Email) (jmap.Email, *Error) {
	if !req.g.sanitize {
		return source, nil
	}
	memory := map[string]int{}
	for _, ref := range []*[]jmap.EmailBodyPart{&source.HtmlBody, &source.TextBody} {
		newBody := make([]jmap.EmailBodyPart, len(*ref))
		for i, p := range *ref {
			t, _, err := mime.ParseMediaType(p.Type)
			if err != nil {
				msg := fmt.Sprintf("failed to parse the mime type '%s'", p.Type)
				req.logger.Error().Str("type", log.SafeString(p.Type)).Msg(msg)
				return source, req.apiError(&ErrorFailedToSanitizeEmail, withDetail(msg))
			}
			if slices.Contains(sanitizableMediaTypes, t) {
				if already, done := memory[p.PartId]; !done {
					if part, ok := source.BodyValues[p.PartId]; ok {
						safe := sanitizationPolicy.Sanitize(part.Value)
						part.Value = safe
						source.BodyValues[p.PartId] = part
						newLen := len(safe)
						memory[p.PartId] = newLen
						p.Size = newLen
					}
				} else {
					p.Size = already
				}
			}
			newBody[i] = p
		}
		*ref = newBody
	}

	// we could post-process attachments as well:
	/*
		for _, part := range source.Attachments {
			if part.Type == "" {
				part.Type = "application/octet-stream"
			}
			if part.Name == "" {
				part.Name = "unknown"
			}
		}
	*/

	return source, nil
}

func (req *Request) sanitizeEmails(source []jmap.Email) ([]jmap.Email, *Error) {
	if !req.g.sanitize {
		return source, nil
	}
	result := make([]jmap.Email, len(source))
	for i, email := range source {
		sanitized, gwerr := req.sanitizeEmail(email)
		if gwerr != nil {
			return nil, gwerr
		}
		result[i] = sanitized
	}
	return result, nil
}
