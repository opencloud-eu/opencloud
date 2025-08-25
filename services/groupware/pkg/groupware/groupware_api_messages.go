package groupware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

// When the request succeeds without a "since" query parameter.
// swagger:response GetAllMessagesInMailbox200
type SwaggerGetAllMessagesInMailbox200 struct {
	// in: body
	Body struct {
		*jmap.Emails
	}
}

// When the request succeeds with a "since" query parameter.
// swagger:response GetAllMessagesInMailboxSince200
type SwaggerGetAllMessagesInMailboxSince200 struct {
	// in: body
	Body struct {
		*jmap.EmailsSince
	}
}

// swagger:route GET /accounts/{account}/mailboxes/{id}/messages messages get_all_messages_in_mailbox
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
//		200: GetAllMessagesInMailbox200
//	 200: GetAllMessagesInMailboxSince200
//		400: ErrorResponse400
//		404: ErrorResponse404
//		500: ErrorResponse500
func (g *Groupware) GetAllMessagesInMailbox(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)
	since := r.Header.Get(HeaderSince)

	if since != "" {
		// ... then it's a completely different operation
		maxChanges := uint(0)
		g.respond(w, r, func(req Request) Response {
			if mailboxId == "" {
				errorId := req.errorId()
				msg := fmt.Sprintf("Missing required mailbox ID path parameter '%v'", UriParamMailboxId)
				return errorResponse(apiError(errorId, ErrorInvalidRequestParameter,
					withDetail(msg),
					withSource(&ErrorSource{Parameter: UriParamMailboxId}),
				))
			}
			logger := log.From(req.logger.With().Str(HeaderSince, since))

			emails, sessionState, jerr := g.jmap.GetEmailsInMailboxSince(req.GetAccountId(), req.session, req.ctx, logger, mailboxId, since, true, g.maxBodyValueBytes, maxChanges)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(emails, sessionState, emails.State)
		})
	} else {
		g.respond(w, r, func(req Request) Response {
			l := req.logger.With()
			if mailboxId == "" {
				errorId := req.errorId()
				msg := fmt.Sprintf("Missing required mailbox ID path parameter '%v'", UriParamMailboxId)
				return errorResponse(apiError(errorId, ErrorInvalidRequestParameter,
					withDetail(msg),
					withSource(&ErrorSource{Parameter: UriParamMailboxId}),
				))
			}
			offset, ok, err := req.parseUNumericParam(QueryParamOffset, 0)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Uint(QueryParamOffset, offset)
			}

			limit, ok, err := req.parseUNumericParam(QueryParamLimit, g.defaultEmailLimit)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Uint(QueryParamLimit, limit)
			}

			logger := log.From(l)

			emails, sessionState, jerr := g.jmap.GetAllEmails(req.GetAccountId(), req.session, req.ctx, logger, mailboxId, offset, limit, true, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(emails, sessionState, emails.State)
		})
	}
}

func (g *Groupware) GetMessagesById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, UriParamMessageId)
	g.respond(w, r, func(req Request) Response {
		ids := strings.Split(id, ",")
		if len(ids) < 1 {
			errorId := req.errorId()
			msg := fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamMessageId, log.SafeString(id), "empty list of mail ids")
			return errorResponse(apiError(errorId, ErrorInvalidRequestParameter,
				withDetail(msg),
				withSource(&ErrorSource{Parameter: UriParamMessageId}),
			))
		}

		logger := log.From(req.logger.With().Str("id", log.SafeString(id)))
		emails, sessionState, jerr := g.jmap.GetEmails(req.GetAccountId(), req.session, req.ctx, logger, ids, true, g.maxBodyValueBytes)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(emails, sessionState, emails.State)
	})
}

func (g *Groupware) getMessagesSince(w http.ResponseWriter, r *http.Request, since string) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(QueryParamSince, since)
		maxChanges, ok, err := req.parseUNumericParam(QueryParamMaxChanges, 0)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Uint(QueryParamMaxChanges, maxChanges)
		}
		logger := log.From(l)

		emails, sessionState, jerr := g.jmap.GetEmailsSince(req.GetAccountId(), req.session, req.ctx, logger, since, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return etagResponse(emails, sessionState, emails.State)
	})
}

type MessageSearchSnippetsResults struct {
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

type MessageSearchResults struct {
	Results    []EmailWithSnippets `json:"results"`
	Total      uint                `json:"total,omitzero"`
	Limit      uint                `json:"limit,omitzero"`
	QueryState jmap.State          `json:"queryState,omitempty"`
}

func (g *Groupware) buildFilter(req Request) (bool, jmap.EmailFilterElement, uint, uint, *log.Logger, Response) {
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

	offset, ok, err := req.parseUNumericParam(QueryParamOffset, 0)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
	}
	if ok {
		l = l.Uint(QueryParamOffset, offset)
	}

	limit, ok, err := req.parseUNumericParam(QueryParamLimit, g.defaultEmailLimit)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
	}
	if ok {
		l = l.Uint(QueryParamLimit, limit)
	}

	before, ok, err := req.parseDateParam(QueryParamSearchBefore)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
	}
	if ok {
		l = l.Time(QueryParamSearchBefore, before)
	}

	after, ok, err := req.parseDateParam(QueryParamSearchAfter)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
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

	minSize, ok, err := req.parseNumericParam(QueryParamSearchMinSize, 0)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
	}
	if ok {
		l = l.Int(QueryParamSearchMinSize, minSize)
	}

	maxSize, ok, err := req.parseNumericParam(QueryParamSearchMaxSize, 0)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
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

	return true, filter, offset, limit, logger, Response{}
}

func (g *Groupware) searchMessages(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, filter, offset, limit, logger, errResp := g.buildFilter(req)
		if !ok {
			return errResp
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

			results, sessionState, jerr := g.jmap.QueryEmailsWithSnippets(req.GetAccountId(), filter, req.session, req.ctx, logger, offset, limit, fetchBodies, g.maxBodyValueBytes)
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

			return etagResponse(MessageSearchResults{
				Results:    flattened,
				Total:      results.Total,
				Limit:      results.Limit,
				QueryState: results.QueryState,
			}, sessionState, results.QueryState)
		} else {
			results, sessionState, jerr := g.jmap.QueryEmailSnippets(req.GetAccountId(), filter, req.session, req.ctx, logger, offset, limit)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(MessageSearchSnippetsResults{
				Results:    results.Snippets,
				Total:      results.Total,
				Limit:      results.Limit,
				QueryState: results.QueryState,
			}, sessionState, results.QueryState)
		}
	})
}

func (g *Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since := q.Get(QueryParamSince)
	if since == "" {
		since = r.Header.Get(HeaderSince)
	}
	if since != "" {
		// get messages changes since a given state
		g.getMessagesSince(w, r, since)
	} else {
		// do a search
		g.searchMessages(w, r)
	}
}

type MessageCreation struct {
	MailboxIds    []string                       `json:"mailboxIds,omitempty"`
	Keywords      []string                       `json:"keywords,omitempty"`
	From          []jmap.EmailAddress            `json:"from,omitempty"`
	Subject       string                         `json:"subject,omitempty"`
	ReceivedAt    time.Time                      `json:"receivedAt,omitzero"`
	SentAt        time.Time                      `json:"sentAt,omitzero"` // huh?
	BodyStructure jmap.EmailBodyStructure        `json:"bodyStructure"`
	BodyValues    map[string]jmap.EmailBodyValue `json:"bodyValues,omitempty"`
}

func (g *Groupware) CreateMessage(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		logger := req.logger

		var body MessageCreation
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

		created, sessionState, jerr := g.jmap.CreateEmail(req.GetAccountId(), create, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(created.Email, sessionState)
	})
}

func (g *Groupware) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		messageId := chi.URLParam(r, UriParamMessageId)

		l := req.logger.With()
		l.Str(UriParamMessageId, messageId)

		logger := log.From(l)

		var body map[string]any
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		updates := map[string]jmap.EmailUpdate{
			messageId: body,
		}

		result, sessionState, jerr := g.jmap.UpdateEmails(req.GetAccountId(), updates, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		if result.Updated == nil {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Missing Email Update Response",
				"An internal API behaved unexpectedly: missing Email update response from JMAP endpoint")))
		}
		updatedEmail, ok := result.Updated[messageId]
		if !ok {
			return errorResponse(apiError(req.errorId(), ErrorApiInconsistency, withTitle("API Inconsistency: Wrong Email Update Response ID",
				"An internal API behaved unexpectedly: wrong Email update ID response from JMAP endpoint")))
		}

		return response(updatedEmail, sessionState)
	})

}

func (g *Groupware) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		messageId := chi.URLParam(r, UriParamMessageId)

		l := req.logger.With()
		l.Str(UriParamMessageId, messageId)

		logger := log.From(l)

		_, sessionState, jerr := g.jmap.DeleteEmails(req.GetAccountId(), []string{messageId}, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return noContentResponse(sessionState)
	})
}

type AboutMessageEmailsEvent struct {
	Id     string       `json:"id"`
	Source string       `json:"source"`
	Emails []jmap.Email `json:"emails"`
}

type AboutMessageResponse struct {
	Email     jmap.Email `json:"email"`
	RequestId string     `json:"requestId"`
	// IV
	// Key (AES-256)
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

func (g *Groupware) RelatedToMessage(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, UriParamMessageId)

	g.respond(w, r, func(req Request) Response {
		limit, _, err := req.parseUNumericParam(QueryParamLimit, 10) // TODO configurable default limit
		if err != nil {
			return errorResponse(err)
		}

		days, _, err := req.parseUNumericParam(QueryParamDays, 5) // TODO configurable default days
		if err != nil {
			return errorResponse(err)
		}

		reqId := req.GetRequestId()
		accountId := req.GetAccountId()
		logger := log.From(req.logger.With().Str(logEmailId, log.SafeString(id)))
		emails, sessionState, jerr := g.jmap.GetEmails(accountId, req.session, req.ctx, logger, []string{id}, true, g.maxBodyValueBytes)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}
		if len(emails.Emails) < 1 {
			logger.Trace().Msg("failed to find any emails matching id") // the id is already in the log field
			return notFoundResponse(sessionState)
		}
		email := emails.Emails[0]

		beacon := email.ReceivedAt // TODO configurable: either relative to when the email was received, or relative to now
		//beacon := time.Now()
		filter := relatedEmails(email, beacon, days)

		// bgctx, _ := context.WithTimeout(context.Background(), time.Duration(30)*time.Second) // TODO configurable
		bgctx := context.Background()

		g.job(logger, RelationTypeSameSender, func(jobId uint64, l *log.Logger) {
			results, _, jerr := g.jmap.QueryEmails(accountId, filter, req.session, bgctx, l, 0, limit, false, g.maxBodyValueBytes)
			if jerr != nil {
				l.Error().Err(jerr).Msgf("failed to query %v emails", RelationTypeSameSender)
			} else {
				related := filterEmails(results.Emails, email)
				l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameSender, len(related))
				if len(related) > 0 {
					req.push(RelationEntityEmail, AboutMessageEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameSender})
				}
			}
		})

		g.job(logger, RelationTypeSameThread, func(jobId uint64, l *log.Logger) {
			emails, _, jerr := g.jmap.EmailsInThread(accountId, email.ThreadId, req.session, bgctx, l, false, g.maxBodyValueBytes)
			if jerr != nil {
				l.Error().Err(jerr).Msgf("failed to list %v emails", RelationTypeSameThread)
			} else {
				related := filterEmails(emails, email)
				l.Trace().Msgf("'%v' found %v other emails", RelationTypeSameThread, len(related))
				if len(related) > 0 {
					req.push(RelationEntityEmail, AboutMessageEmailsEvent{Id: reqId, Emails: related, Source: RelationTypeSameThread})
				}
			}
		})

		return etagResponse(AboutMessageResponse{
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
