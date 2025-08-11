package groupware

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

func (g Groupware) GetAllMessagesInMailbox(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, UriParamMailboxId)
	since := r.Header.Get(HeaderSince)

	if since != "" {
		// ... then it's a completely different operation
		maxChanges := -1
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

			emails, jerr := g.jmap.GetEmailsInMailboxSince(req.GetAccountId(), req.session, req.ctx, logger, mailboxId, since, true, g.maxBodyValueBytes, maxChanges)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return response(emails, emails.State)
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
			offset, ok, err := req.parseNumericParam(QueryParamOffset, 0)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Int(QueryParamOffset, offset)
			}

			limit, ok, err := req.parseNumericParam(QueryParamLimit, g.defaultEmailLimit)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Int(QueryParamLimit, limit)
			}

			logger := log.From(l)

			emails, jerr := g.jmap.GetAllEmails(req.GetAccountId(), req.session, req.ctx, logger, mailboxId, offset, limit, true, g.maxBodyValueBytes)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return response(emails, emails.State)
		})
	}
}

func (g Groupware) GetMessagesById(w http.ResponseWriter, r *http.Request) {
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
		emails, jerr := g.jmap.GetEmails(req.GetAccountId(), req.session, req.ctx, logger, ids, true, g.maxBodyValueBytes)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(emails, emails.State)
	})
}

func (g Groupware) getMessagesSince(w http.ResponseWriter, r *http.Request, since string) {
	g.respond(w, r, func(req Request) Response {
		l := req.logger.With().Str(QueryParamSince, since)
		maxChanges, ok, err := req.parseNumericParam(QueryParamMaxChanges, -1)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			l = l.Int(QueryParamMaxChanges, maxChanges)
		}
		logger := log.From(l)

		emails, jerr := g.jmap.GetEmailsSince(req.GetAccountId(), req.session, req.ctx, logger, since, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(emails, emails.State)
	})
}

type MessageSearchSnippetsResults struct {
	Results    []jmap.SearchSnippet `json:"results,omitempty"`
	Total      int                  `json:"total,omitzero"`
	Limit      int                  `json:"limit,omitzero"`
	QueryState string               `json:"queryState,omitempty"`
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
	Total      int                 `json:"total,omitzero"`
	Limit      int                 `json:"limit,omitzero"`
	QueryState string              `json:"queryState,omitempty"`
}

func (g Groupware) buildQuery(req Request) (bool, jmap.EmailFilterElement, int, int, *log.Logger, Response) {
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

	offset, ok, err := req.parseNumericParam(QueryParamOffset, 0)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
	}
	if ok {
		l = l.Int(QueryParamOffset, offset)
	}

	limit, ok, err := req.parseNumericParam(QueryParamLimit, g.defaultEmailLimit)
	if err != nil {
		return false, nil, 0, 0, nil, errorResponse(err)
	}
	if ok {
		l = l.Int(QueryParamLimit, limit)
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

func (g Groupware) searchMessages(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		ok, filter, offset, limit, logger, errResp := g.buildQuery(req)
		if !ok {
			return errResp
		}

		var empty jmap.EmailFilterElement

		if filter == empty {
			errorId := req.errorId()
			msg := "Invalid search request has no criteria"
			return errorResponse(apiError(errorId, ErrorInvalidUserRequest, withDetail(msg)))
		}

		fetchEmails, ok, err := req.parseBoolParam(QueryParamSearchFetchEmails, false)
		if err != nil {
			return errorResponse(err)
		}
		if ok {
			logger = &log.Logger{Logger: logger.With().Bool(QueryParamSearchFetchEmails, fetchEmails).Logger()}
		}

		if fetchEmails {
			fetchBodies, ok, err := req.parseBoolParam(QueryParamSearchFetchBodies, false)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				logger = &log.Logger{Logger: logger.With().Bool(QueryParamSearchFetchBodies, fetchBodies).Logger()}
			}

			results, jerr := g.jmap.QueryEmails(req.GetAccountId(), filter, req.session, req.ctx, logger, offset, limit, fetchBodies, g.maxBodyValueBytes)
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
			}, results.SessionState, results.QueryState)
		} else {
			results, jerr := g.jmap.QueryEmailSnippets(req.GetAccountId(), filter, req.session, req.ctx, logger, offset, limit)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(MessageSearchSnippetsResults{
				Results:    results.Snippets,
				Total:      results.Total,
				Limit:      results.Limit,
				QueryState: results.QueryState,
			}, results.SessionState, results.QueryState)
		}
	})
}

func (g Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
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

func (g Groupware) CreateMessage(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		messageId := chi.URLParam(r, UriParamMessageId)

		l := req.logger.With()
		l.Str(UriParamMessageId, messageId)
		logger := log.From(l)

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

		created, jerr := g.jmap.CreateEmail(req.GetAccountId(), create, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(created.Email, created.State)
	})
}

func (g Groupware) UpdateMessage(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		messageId := chi.URLParam(r, UriParamMessageId)

		l := req.logger.With()
		l.Str(UriParamMessageId, messageId)

		logger := &log.Logger{Logger: l.Logger()}

		var body map[string]any
		err := req.body(&body)
		if err != nil {
			return errorResponse(err)
		}

		updates := map[string]jmap.EmailUpdate{
			messageId: body,
		}

		result, jerr := g.jmap.UpdateEmails(req.GetAccountId(), updates, req.session, req.ctx, logger)
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

		return response(updatedEmail, result.State)
	})

}

func (g Groupware) DeleteMessage(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) Response {
		messageId := chi.URLParam(r, UriParamMessageId)

		l := req.logger.With()
		l.Str(UriParamMessageId, messageId)

		logger := &log.Logger{Logger: l.Logger()}

		deleted, jerr := g.jmap.DeleteEmails(req.GetAccountId(), []string{messageId}, req.session, req.ctx, logger)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return noContentResponse(deleted.State)
	})
}
