package groupware

import (
	"fmt"
	"net/http"
	"strings"

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
		g.respond(w, r, func(req Request) (any, string, *Error) {
			if mailboxId == "" {
				errorId := req.errorId()
				msg := fmt.Sprintf("Missing required mailbox ID path parameter '%v'", UriParamMailboxId)
				return nil, "", apiError(errorId, ErrorInvalidRequestParameter,
					withDetail(msg),
					withSource(&ErrorSource{Parameter: UriParamMailboxId}),
				)
			}
			logger := &log.Logger{Logger: req.logger.With().Str(HeaderSince, since).Logger()}

			emails, jerr := g.jmap.GetEmailsInMailboxSince(req.GetAccountId(), req.session, req.ctx, logger, mailboxId, since, true, g.maxBodyValueBytes, maxChanges)
			if jerr != nil {
				return nil, "", req.apiErrorFromJmap(jerr)
			}

			return emails, emails.State, nil
		})
	} else {
		g.respond(w, r, func(req Request) (any, string, *Error) {
			l := req.logger.With()
			if mailboxId == "" {
				errorId := req.errorId()
				msg := fmt.Sprintf("Missing required mailbox ID path parameter '%v'", UriParamMailboxId)
				return nil, "", apiError(errorId, ErrorInvalidRequestParameter,
					withDetail(msg),
					withSource(&ErrorSource{Parameter: UriParamMailboxId}),
				)
			}
			offset, ok, err := req.parseNumericParam(QueryParamOffset, 0)
			if err != nil {
				return nil, "", err
			}
			if ok {
				l = l.Int(QueryParamOffset, offset)
			}

			limit, ok, err := req.parseNumericParam(QueryParamLimit, g.defaultEmailLimit)
			if err != nil {
				return nil, "", err
			}
			if ok {
				l = l.Int(QueryParamLimit, limit)
			}

			logger := &log.Logger{Logger: l.Logger()}

			emails, jerr := g.jmap.GetAllEmails(req.GetAccountId(), req.session, req.ctx, logger, mailboxId, offset, limit, true, g.maxBodyValueBytes)
			if jerr != nil {
				return nil, "", req.apiErrorFromJmap(jerr)
			}

			return emails, emails.State, nil
		})
	}
}

func (g Groupware) GetMessagesById(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, UriParamMessagesId)
	g.respond(w, r, func(req Request) (any, string, *Error) {
		ids := strings.Split(id, ",")
		if len(ids) < 1 {
			errorId := req.errorId()
			msg := fmt.Sprintf("Invalid value for path parameter '%v': '%s': %s", UriParamMessagesId, logstr(id), "empty list of mail ids")
			return nil, "", apiError(errorId, ErrorInvalidRequestParameter,
				withDetail(msg),
				withSource(&ErrorSource{Parameter: UriParamMessagesId}),
			)
		}

		logger := &log.Logger{Logger: req.logger.With().Str("id", logstr(id)).Logger()}
		emails, jerr := g.jmap.GetEmails(req.GetAccountId(), req.session, req.ctx, logger, ids, true, g.maxBodyValueBytes)
		if jerr != nil {
			return nil, "", req.apiErrorFromJmap(jerr)
		}

		return emails, emails.State, nil
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
		g.respond(w, r, func(req Request) (any, string, *Error) {
			logger := &log.Logger{Logger: req.logger.With().Str(HeaderSince, since).Logger()}

			emails, jerr := g.jmap.GetEmailsSince(req.GetAccountId(), req.session, req.ctx, logger, since, true, g.maxBodyValueBytes, maxChanges)
			if jerr != nil {
				return nil, "", req.apiErrorFromJmap(jerr)
			}

			return emails, emails.State, nil
		})
	} else {
		// do a search
		g.respond(w, r, func(req Request) (any, string, *Error) {
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
				return nil, "", err
			}
			if ok {
				l = l.Int(QueryParamOffset, offset)
			}

			limit, ok, err := req.parseNumericParam(QueryParamLimit, g.defaultEmailLimit)
			if err != nil {
				return nil, "", err
			}
			if ok {
				l = l.Int(QueryParamLimit, limit)
			}

			before, ok, err := req.parseDateParam(QueryParamSearchBefore)
			if err != nil {
				return nil, "", err
			}
			if ok {
				l = l.Time(QueryParamSearchBefore, before)
			}

			after, ok, err := req.parseDateParam(QueryParamSearchAfter)
			if err != nil {
				return nil, "", err
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
				return nil, "", err
			}
			if ok {
				l = l.Int(QueryParamSearchMinSize, minSize)
			}

			maxSize, ok, err := req.parseNumericParam(QueryParamSearchMaxSize, 0)
			if err != nil {
				return nil, "", err
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
				return nil, "", req.apiErrorFromJmap(jerr)
			}

			return emails, emails.QueryState, nil
		})
	}
}
