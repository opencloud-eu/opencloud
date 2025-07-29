package groupware

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
)

func (g Groupware) Route(r chi.Router) {
	r.Get("/", g.Index)
	r.Get("/mailboxes", g.GetMailboxes) // ?name=&role=&subcribed=
	r.Get("/mailbox/{id}", g.GetMailboxById)
	r.Get("/{mailbox}/messages", g.GetMessages)
	r.Get("/identity", g.GetIdentity)
	r.Get("/vacation", g.GetVacation)
}

type IndexResponse struct {
	AccountId string
}

func (IndexResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func (g Groupware) Index(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *ApiError) {
		return IndexResponse{AccountId: req.session.AccountId}, "", nil
	})
}

func (g Groupware) GetIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *ApiError) {
		res, err := g.jmap.GetIdentity(req.session, req.ctx, req.logger)
		return res, res.State, apiErrorFromJmap(err)
	})
}

func (g Groupware) GetVacation(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(req Request) (any, string, *ApiError) {
		res, err := g.jmap.GetVacationResponse(req.session, req.ctx, req.logger)
		return res, res.State, apiErrorFromJmap(err)
	})
}

func (g Groupware) GetMailboxById(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, "mailbox")
	if mailboxId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	g.respond(w, r, func(req Request) (any, string, *ApiError) {
		res, err := g.jmap.GetMailbox(req.session, req.ctx, req.logger, []string{mailboxId})
		if err != nil {
			return res, "", apiErrorFromJmap(err)
		}

		if len(res.List) == 1 {
			return res.List[0], res.State, apiErrorFromJmap(err)
		} else {
			return nil, res.State, apiErrorFromJmap(err)
		}
	})
}

func (g Groupware) GetMailboxes(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var filter jmap.MailboxFilterCondition

	hasCriteria := false
	name := q.Get("name")
	if name != "" {
		filter.Name = name
		hasCriteria = true
	}
	role := q.Get("role")
	if role != "" {
		filter.Role = role
		hasCriteria = true
	}
	subscribed := q.Get("subscribed")
	if subscribed != "" {
		b, err := strconv.ParseBool(subscribed)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		filter.IsSubscribed = &b
		hasCriteria = true
	}

	g.respond(w, r, func(req Request) (any, string, *ApiError) {
		if hasCriteria {
			mailboxes, err := g.jmap.SearchMailboxes(req.session, req.ctx, req.logger, filter)
			if err != nil {
				return nil, "", apiErrorFromJmap(err)
			}
			return mailboxes.Mailboxes, mailboxes.State, nil
		} else {
			mailboxes, err := g.jmap.GetAllMailboxes(req.session, req.ctx, req.logger)
			if err != nil {
				return nil, "", apiErrorFromJmap(err)
			}
			return mailboxes.List, mailboxes.State, nil
		}
	})
}

func (g Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, "mailbox")
	g.respond(w, r, func(req Request) (any, string, *ApiError) {
		page, ok, _ := ParseNumericParam(r, "page", -1)
		logger := req.logger
		if ok {
			logger = &log.Logger{Logger: logger.With().Int("page", page).Logger()}
		}
		size, ok, _ := ParseNumericParam(r, "size", -1)
		if ok {
			logger = &log.Logger{Logger: logger.With().Int("size", size).Logger()}
		}

		offset := page * size
		limit := size
		if limit < 0 {
			limit = g.defaultEmailLimit
		}

		emails, err := g.jmap.GetEmails(req.session, req.ctx, logger, mailboxId, offset, limit, true, g.maxBodyValueBytes)
		if err != nil {
			return nil, "", apiErrorFromJmap(err)
		}

		return emails, emails.State, nil
	})
}
