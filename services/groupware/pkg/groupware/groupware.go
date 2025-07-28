package groupware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"

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
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)

	session, ok, err := g.session(r, ctx, &logger)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if ok {
		//logger = session.DecorateLogger(logger)
		_ = render.Render(w, r, IndexResponse{AccountId: session.AccountId})
	} else {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

}

func (g Groupware) GetIdentity(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, *ApiError) {
		res, err := g.jmap.GetIdentity(session, ctx, logger)
		return res, res.State, apiErrorFromJmap(err)
	})
}

func (g Groupware) GetVacation(w http.ResponseWriter, r *http.Request) {
	g.respond(w, r, func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, *ApiError) {
		res, err := g.jmap.GetVacationResponse(session, ctx, logger)
		return res, res.State, apiErrorFromJmap(err)
	})
}

func (g Groupware) GetMailboxById(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, "mailbox")
	if mailboxId == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	g.respond(w, r, func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, *ApiError) {
		res, err := g.jmap.GetMailbox(session, ctx, logger, []string{mailboxId})
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

	if hasCriteria {
		g.respond(w, r, func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, *ApiError) {
			mailboxes, err := g.jmap.SearchMailboxes(session, ctx, logger, filter)
			if err != nil {
				return nil, "", apiErrorFromJmap(err)
			}
			return mailboxes.Mailboxes, mailboxes.State, nil
		})
	} else {
		g.respond(w, r, func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, *ApiError) {
			mailboxes, err := g.jmap.GetAllMailboxes(session, ctx, logger)
			if err != nil {
				return nil, "", apiErrorFromJmap(err)
			}
			return mailboxes.List, mailboxes.State, nil
		})
	}
}

func (g Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
	mailboxId := chi.URLParam(r, "mailbox")
	g.respond(w, r, func(r *http.Request, ctx context.Context, logger *log.Logger, session *jmap.Session) (any, string, *ApiError) {
		page, ok, _ := ParseNumericParam(r, "page", -1)
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

		emails, err := g.jmap.GetEmails(session, ctx, logger, mailboxId, offset, limit, true, g.maxBodyValueBytes)
		if err != nil {
			return nil, "", apiErrorFromJmap(err)
		}

		return emails, emails.State, nil
	})
}
