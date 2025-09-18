package groupware

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
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

			logger := log.From(req.logger.With().Str(HeaderSince, since).Str(logAccountId, accountId))

			emails, sessionState, jerr := g.jmap.GetMailboxChanges(accountId, req.session, req.ctx, logger, mailboxId, since, true, g.maxBodyValueBytes, maxChanges)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(emails, sessionState, emails.State)
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

			emails, sessionState, jerr := g.jmap.GetAllEmailsInMailbox(accountId, req.session, req.ctx, logger, mailboxId, offset, limit, true, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(emails, sessionState, emails.State)
		})
	}
}

func (g *Groupware) GetEmailsById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, UriParamEmailId)
	g.respond(w, r, func(req Request) Response {
		ids := strings.Split(id, ",")
		if len(ids) < 1 {
			return req.parameterErrorResponse(UriParamEmailId, fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamEmailId, log.SafeString(id), "empty list of mail ids"))
		}

		accountId, err := req.GetAccountIdForMail()
		if err != nil {
			return errorResponse(err)
		}

		l := req.logger.With().Str(logAccountId, log.SafeString(accountId))
		if len(ids) == 1 {
			logger := log.From(l.Str("id", log.SafeString(id)))
			emails, sessionState, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, ids, true, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}
			if len(emails.Emails) < 1 {
				return notFoundResponse(sessionState)
			} else {
				return etagResponse(emails.Emails[0], sessionState, emails.State)
			}
		} else {
			logger := log.From(l.Array("ids", log.SafeStringArray(ids)))
			emails, sessionState, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, ids, true, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}
			if len(emails.Emails) < 1 {
				return notFoundResponse(sessionState)
			} else {
				return etagResponse(emails.Emails, sessionState, emails.State)
			}
		}
	})
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
			l := req.logger.With().Str(logAccountId, accountId)
			logger := log.From(l)
			emails, sessionState, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, []string{id}, false, 0)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}
			if len(emails.Emails) < 1 {
				return notFoundResponse(sessionState)
			}
			email := emails.Emails[0]
			return etagResponse(email.Attachments, sessionState, emails.State)
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

			l := req.logger.With().Str(logAccountId, mailAccountId).Str(logBlobAccountId, log.SafeString(blobAccountId))
			l = contextAppender(l)
			logger := log.From(l)

			emails, _, jerr := g.jmap.GetEmails(mailAccountId, req.session, req.ctx, logger, []string{id}, false, 0)
			if jerr != nil {
				return req.apiErrorFromJmap(req.observeJmapError(jerr))
			}
			if len(emails.Emails) < 1 {
				return nil
			}

			email := emails.Emails[0]
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

			blob, jerr := g.jmap.DownloadBlobStream(blobAccountId, attachment.BlobId, attachment.Name, attachment.Type, req.session, req.ctx, logger)
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

			_, err := io.Copy(w, blob.Body)
			if err != nil {
				return req.observedParameterError(ErrorStreamingResponse)
			}

			return nil
		})
	}
}

func (g *Groupware) getEmailsSince(w http.ResponseWriter, r *http.Request, since string) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(QueryParamSince, since)
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

		emails, sessionState, jerr := g.jmap.GetEmailsSince(accountId, req.session, req.ctx, logger, since, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(emails, sessionState, emails.State)
	})
}

type EmailSearchSnippetsResults struct {
	Results    []jmap.SearchSnippet `json:"results,omitempty"`
	Total      uint                 `json:"total,omitzero"`
	Limit      uint                 `json:"limit,omitzero"`
	QueryState jmap.State           `json:"queryState,omitempty"`
}

type EmailWithSnippets struct {
	jmap.Email
	Snippets []SnippetWithoutEmailId `json:"snippets,omitempty"`
}

type SnippetWithoutEmailId struct {
	Subject string `json:"subject,omitempty"`
	Preview string `json:"preview,omitempty"`
}

type EmailSearchResults struct {
	Results    []EmailWithSnippets `json:"results"`
	Total      uint                `json:"total,omitzero"`
	Limit      uint                `json:"limit,omitzero"`
	QueryState jmap.State          `json:"queryState,omitempty"`
}

func (g *Groupware) buildFilter(req Request) (bool, jmap.EmailFilterElement, uint, uint, *log.Logger, *Error) {
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

	l := req.logger.With()

	offset, ok, err := req.parseUIntParam(QueryParamOffset, 0)
	if err != nil {
		return false, nil, 0, 0, nil, err
	}
	if ok {
		l = l.Uint(QueryParamOffset, offset)
	}

	limit, ok, err := req.parseUIntParam(QueryParamLimit, g.defaultEmailLimit)
	if err != nil {
		return false, nil, 0, 0, nil, err
	}
	if ok {
		l = l.Uint(QueryParamLimit, limit)
	}

	before, ok, err := req.parseDateParam(QueryParamSearchBefore)
	if err != nil {
		return false, nil, 0, 0, nil, err
	}
	if ok {
		l = l.Time(QueryParamSearchBefore, before)
	}

	after, ok, err := req.parseDateParam(QueryParamSearchAfter)
	if err != nil {
		return false, nil, 0, 0, nil, err
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

	minSize, ok, err := req.parseIntParam(QueryParamSearchMinSize, 0)
	if err != nil {
		return false, nil, 0, 0, nil, err
	}
	if ok {
		l = l.Int(QueryParamSearchMinSize, minSize)
	}

	maxSize, ok, err := req.parseIntParam(QueryParamSearchMaxSize, 0)
	if err != nil {
		return false, nil, 0, 0, nil, err
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
	}
	filter = &firstFilter

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

	return true, filter, offset, limit, logger, nil
}

func (g *Groupware) searchEmails(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, filter, offset, limit, logger, err := g.buildFilter(req)
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
			logger = log.From(logger.With().Str(logAccountId, accountId))

			results, sessionState, jerr := g.jmap.QueryEmailsWithSnippets(accountId, filter, req.session, req.ctx, logger, offset, limit, fetchBodies, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			flattened := make([]EmailWithSnippets, len(results.Results))
			for i, result := range results.Results {
				snippets := make([]SnippetWithoutEmailId, len(result.Snippets))
				for j, snippet := range result.Snippets {
					snippets[j] = SnippetWithoutEmailId{
						Subject: snippet.Subject,
						Preview: snippet.Preview,
					}
				}
				flattened[i] = EmailWithSnippets{
					Email:    result.Email,
					Snippets: snippets,
				}
			}

			return etagResponse(EmailSearchResults{
				Results:    flattened,
				Total:      results.Total,
				Limit:      results.Limit,
				QueryState: results.QueryState,
			}, sessionState, results.QueryState)
		} else {
			accountId, err := req.GetAccountIdForMail()
			if err != nil {
				return errorResponse(err)
			}
			logger = log.From(logger.With().Str(logAccountId, accountId))

			results, sessionState, jerr := g.jmap.QueryEmailSnippets(accountId, filter, req.session, req.ctx, logger, offset, limit)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(EmailSearchSnippetsResults{
				Results:    results.Snippets,
				Total:      results.Total,
				Limit:      results.Limit,
				QueryState: results.QueryState,
			}, sessionState, results.QueryState)
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

func (g *Groupware) CreateEmail(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		logger := req.logger

		accountId, gwerr := req.GetAccountIdForMail()
		if gwerr != nil {
			return errorResponse(gwerr)
		}
		logger = log.From(logger.With().Str(logAccountId, accountId))

		var body EmailCreation
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		mailboxIdsMap := map[string]bool{}
		for _, mailboxId := range body.MailboxIds {
			mailboxIdsMap[mailboxId] = true
		}

		keywordsMap := map[string]bool{}
		for _, keyword := range body.Keywords {
			keywordsMap[keyword] = true
		}

		create := jmap.EmailCreate{
			MailboxIds:    mailboxIdsMap,
			Keywords:      keywordsMap,
			From:          body.From,
			Subject:       body.Subject,
			ReceivedAt:    body.ReceivedAt,
			SentAt:        body.SentAt,
			BodyStructure: body.BodyStructure,
			BodyValues:    body.BodyValues,
		}

		created, sessionState, jerr := g.jmap.CreateEmail(accountId, create, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(created.Email, sessionState)
	})
}

func (g *Groupware) UpdateEmail(w http.ResponseWriter, r *http.Request) {
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

		var body map[string]any
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		updates := map[string]jmap.EmailUpdate{
			emailId: body,
		}

		result, sessionState, jerr := g.jmap.UpdateEmails(accountId, updates, req.session, req.ctx, logger)
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

		return response(updatedEmail, sessionState)
	})

}

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

		_, sessionState, jerr := g.jmap.DeleteEmails(accountId, []string{emailId}, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return noContentResponse(sessionState)
	})
}

type AboutEmailsEvent struct {
	Id     string       `json:"id"`
	Source string       `json:"source"`
	Emails []jmap.Email `json:"emails"`
}

type AboutEmailResponse struct {
	Email     jmap.Email `json:"email"`
	RequestId string     `json:"requestId"`
}

func relatedEmails(email jmap.Email, beacon time.Time, days uint) jmap.EmailFilterElement {
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
		l = l.Str(logAccountId, accountId)

		logger := log.From(l)

		reqId := req.GetRequestId()
		getEmailsBefore := time.Now()
		emails, sessionState, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, []string{id}, true, g.maxBodyValueBytes)
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
		filter := relatedEmails(email, beacon, days)

		// bgctx, _ := context.WithTimeout(context.Background(), time.Duration(30)*time.Second) // TODO configurable
		bgctx := context.Background()

		g.job(logger, RelationTypeSameSender, func(jobId uint64, l *log.Logger) {
			before := time.Now()
			results, _, jerr := g.jmap.QueryEmails(accountId, filter, req.session, bgctx, l, 0, limit, false, g.maxBodyValueBytes)
			duration := time.Since(before)
			if jerr != nil {
				req.observeJmapError(jerr)
				l.Error().Err(jerr).Msgf("failed to query %v emails", RelationTypeSameSender)
			} else {
				req.observe(g.metrics.EmailSameSenderDuration.WithLabelValues(req.session.JmapEndpoint), duration.Seconds())
				related := filterEmails(results.Emails, email)
				l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameSender, len(related))
				if len(related) > 0 {
					req.push(RelationEntityEmail, AboutEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameSender})
				}
			}
		})

		g.job(logger, RelationTypeSameThread, func(jobId uint64, l *log.Logger) {
			before := time.Now()
			emails, _, jerr := g.jmap.EmailsInThread(accountId, email.ThreadId, req.session, bgctx, l, false, g.maxBodyValueBytes)
			duration := time.Since(before)
			if jerr != nil {
				req.observeJmapError(jerr)
				l.Error().Err(jerr).Msgf("failed to list %v emails", RelationTypeSameThread)
			} else {
				req.observe(g.metrics.EmailSameThreadDuration.WithLabelValues(req.session.JmapEndpoint), duration.Seconds())
				related := filterEmails(emails, email)
				l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameThread, len(related))
				if len(related) > 0 {
					req.push(RelationEntityEmail, AboutEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameThread})
				}
			}
		})

		return etagResponse(AboutEmailResponse{
			Email:     email,
			RequestId: reqId,
		}, sessionState, emails.State)
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
