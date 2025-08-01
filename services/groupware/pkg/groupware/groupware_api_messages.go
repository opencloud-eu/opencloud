package groupware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

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
			if mailboxId == "" {
				errorId := req.errorId()
				msg := fmt.Sprintf("Missing required mailbox ID path parameter '%v'", UriParamMailboxId)
				return nil, "", apiError(errorId, ErrorInvalidRequestParameter,
					withDetail(msg),
					withSource(&ErrorSource{Parameter: UriParamMailboxId}),
				)
			}
			page, ok, err := req.parseNumericParam(QueryParamPage, -1)
			if err != nil {
				return nil, "", err
			}
			logger := req.logger
			if ok {
				logger = &log.Logger{Logger: logger.With().Int(QueryParamPage, page).Logger()}
			}

			size, ok, err := req.parseNumericParam(QueryParamSize, -1)
			if err != nil {
				return nil, "", err
			}
			if ok {
				logger = &log.Logger{Logger: logger.With().Int(QueryParamSize, size).Logger()}
			}

			offset := page * size
			limit := size
			if limit < 0 {
				limit = g.defaultEmailLimit
			}

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

func (g Groupware) GetMessageUpdates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	since := q.Get(QueryParamSince)
	if since == "" {
		since = r.Header.Get("If-None-Match")
	}
	maxChanges := -1
	g.respond(w, r, func(req Request) (any, string, *Error) {
		logger := &log.Logger{Logger: req.logger.With().Str(HeaderSince, since).Logger()}

		emails, jerr := g.jmap.GetEmailsSince(req.GetAccountId(), req.session, req.ctx, logger, since, true, g.maxBodyValueBytes, maxChanges)
		if jerr != nil {
			return nil, "", req.apiErrorFromJmap(jerr)
		}

		return emails, emails.State, nil
	})
}
