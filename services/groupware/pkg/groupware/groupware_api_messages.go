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

func (g Groupware) GetAllMessages(w http.ResponseWriter, r *http.Request) {
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
			logger := &log.Logger{Logger: req.logger.With().Str(HeaderSince, since).Logger()}

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

			logger := &log.Logger{Logger: l.Logger()}

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
			msg := fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamMessageId, logstr(id), "empty list of mail ids")
			return errorResponse(apiError(errorId, ErrorInvalidRequestParameter,
				withDetail(msg),
				withSource(&ErrorSource{Parameter: UriParamMessageId}),
			))
		}

		logger := &log.Logger{Logger: req.logger.With().Str("id", logstr(id)).Logger()}
		emails, jerr := g.jmap.GetEmails(req.GetAccountId(), req.session, req.ctx, logger, ids, true, g.maxBodyValueBytes)
		if jerr != nil {
			return req.errorResponseFromJmap(jerr)
		}

		return response(emails, emails.State)
	})
}

func (g Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since := q.Get(QueryParamSince)
	if since == "" {
		since = r.Header.Get("If-None-Match")
	}
	if since != "" {
		// get messages changes since a given state
		maxChanges := -1
		g.respond(w, r, func(req Request) Response {
			logger := &log.Logger{Logger: req.logger.With().Str(HeaderSince, since).Logger()}

			emails, jerr := g.jmap.GetEmailsSince(req.GetAccountId(), req.session, req.ctx, logger, since, true, g.maxBodyValueBytes, maxChanges)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return response(emails, emails.State)
		})
	} else {
		// do a search
		g.respond(w, r, func(req Request) Response {
			mailboxId := q.Get(QueryParamMailboxId)
			notInMailboxIds := q[QueryParamNotInMailboxId]
			text := q.Get(QueryParamSearchText)
			from := q.Get(QueryParamSearchFrom)
			to := q.Get(QueryParamSearchTo)
			cc := q.Get(QueryParamSearchCc)
			bcc := q.Get(QueryParamSearchBcc)
			subject := q.Get(QueryParamSearchSubject)
			body := q.Get(QueryParamSearchBody)

			l := req.logger.With()

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

			before, ok, err := req.parseDateParam(QueryParamSearchBefore)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Time(QueryParamSearchBefore, before)
			}

			after, ok, err := req.parseDateParam(QueryParamSearchAfter)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Time(QueryParamSearchAfter, after)
			}

			if mailboxId != "" {
				l = l.Str(QueryParamMailboxId, logstr(mailboxId))
			}
			if len(notInMailboxIds) > 0 {
				l = l.Array(QueryParamNotInMailboxId, logstrarray(notInMailboxIds))
			}
			if text != "" {
				l = l.Str(QueryParamSearchText, logstr(text))
			}
			if from != "" {
				l = l.Str(QueryParamSearchFrom, logstr(from))
			}
			if to != "" {
				l = l.Str(QueryParamSearchTo, logstr(to))
			}
			if cc != "" {
				l = l.Str(QueryParamSearchCc, logstr(cc))
			}
			if bcc != "" {
				l = l.Str(QueryParamSearchBcc, logstr(bcc))
			}
			if subject != "" {
				l = l.Str(QueryParamSearchSubject, logstr(subject))
			}
			if body != "" {
				l = l.Str(QueryParamSearchBody, logstr(body))
			}

			minSize, ok, err := req.parseNumericParam(QueryParamSearchMinSize, 0)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Int(QueryParamSearchMinSize, minSize)
			}

			maxSize, ok, err := req.parseNumericParam(QueryParamSearchMaxSize, 0)
			if err != nil {
				return errorResponse(err)
			}
			if ok {
				l = l.Int(QueryParamSearchMaxSize, maxSize)
			}

			logger := &log.Logger{Logger: l.Logger()}

			filter := jmap.EmailFilterCondition{
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
				//HasKeyword: "",
			}

			emails, jerr := g.jmap.QueryEmails(req.GetAccountId(), &filter, req.session, req.ctx, logger, offset, limit, false, 0)
			if jerr != nil {
				return req.errorResponseFromJmap(jerr)
			}

			return etagResponse(emails, emails.SessionState, emails.QueryState)
		})
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
		logger := &log.Logger{Logger: l.Logger()}

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
			// TODO(pbleser-oc) handle missing update response
		}
		updatedEmail, ok := result.Updated[messageId]
		if !ok {
			// TODO(pbleser-oc) handle missing update response
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
