package svc

import (
	"context"
	"crypto/tls"
	"net/http"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"

	"github.com/go-chi/render"

	"github.com/jellydator/ttlcache/v3"
)

type Groupware struct {
	logger           *log.Logger
	jmapClient       jmap.JmapClient
	contextCache     *ttlcache.Cache[string, jmap.JmapContext]
	usernameProvider jmap.HttpJmapUsernameProvider // we also need it for ourselves for now
}

type Message struct {
	Id                string    `json:"id"`
	CreatedDateTime   time.Time `json:"createdDateTime"`
	ReceivedDateTime  time.Time `json:"receivedDateTime"`
	HasAttachments    bool      `json:"hasAttachments"`
	InternetMessageId string    `json:"InternetMessageId"`
	Subject           string    `json:"subject"`
}

func NewGroupware(logger *log.Logger, config *config.Config) *Groupware {
	baseUrl := config.Mail.BaseUrl
	jmapUrl := config.Mail.JmapUrl
	masterUsername := config.Mail.Master.Username
	masterPassword := config.Mail.Master.Password

	tr := http.DefaultTransport.(*http.Transport).Clone()
	tr.ResponseHeaderTimeout = time.Duration(config.Mail.Timeout)
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	tr.TLSClientConfig = tlsConfig
	c := *http.DefaultClient
	c.Transport = tr

	jmapUsernameProvider := jmap.NewRevaContextHttpJmapUsernameProvider()

	api := jmap.NewHttpJmapApiClient(
		baseUrl,
		jmapUrl,
		&c,
		jmapUsernameProvider,
		masterUsername,
		masterPassword,
	)

	jmapClient := jmap.NewJmapClient(api, api)

	loader := ttlcache.LoaderFunc[string, jmap.JmapContext](
		func(c *ttlcache.Cache[string, jmap.JmapContext], key string) *ttlcache.Item[string, jmap.JmapContext] {
			jmapContext, err := jmapClient.FetchJmapContext(key, logger)
			if err != nil {
				logger.Error().Err(err).Str("username", key).Msg("failed to retrieve well-known")
				return nil
			}
			item := c.Set(key, jmapContext, config.Mail.ContextCacheTTL)
			return item
		},
	)

	contextCache := ttlcache.New(
		ttlcache.WithTTL[string, jmap.JmapContext](
			config.Mail.ContextCacheTTL,
		),
		ttlcache.WithDisableTouchOnHit[string, jmap.JmapContext](),
		ttlcache.WithLoader(loader),
	)
	go contextCache.Start()

	return &Groupware{
		logger:           logger,
		jmapClient:       jmapClient,
		contextCache:     contextCache,
		usernameProvider: jmapUsernameProvider,
	}
}

func pickInbox(folders jmap.JmapFolders) string {
	for _, folder := range folders.Folders {
		if folder.Role == "inbox" {
			return folder.Id
		}
	}
	return ""
}

func (g Groupware) context(ctx context.Context, logger *log.Logger) (jmap.JmapContext, error) {
	username, err := g.usernameProvider.GetUsername(ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve username")
		return jmap.JmapContext{}, err
	}

	item := g.contextCache.Get(username)
	return item.Value(), nil
}

func (g Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := g.logger.SubloggerWithRequestID(ctx)

	jmapContext, err := g.context(ctx, &logger)
	if err != nil {
		logger.Error().Err(err).Interface("query", r.URL.Query()).Msg("failed to determine Jmap context")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	ctx = context.WithValue(ctx, jmap.ContextAccountId, jmapContext.AccountId)

	logger.Debug().Msg("fetching folders")
	folders, err := g.jmapClient.GetMailboxes(jmapContext, ctx, &logger)
	if err != nil {
		logger.Error().Err(err).Interface("query", r.URL.Query()).Msg("could not retrieve mailboxes")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	inboxId := pickInbox(folders)
	logger.Debug().Str("mailboxId", inboxId).Msg("fetching emails from inbox")
	emails, err := g.jmapClient.EmailQuery(jmapContext, ctx, &logger, inboxId)
	if err != nil {
		logger.Error().Err(err).Interface("query", r.URL.Query()).Msg("could not retrieve emails from inbox")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	messages := make([]Message, 0, len(emails.Emails))
	for _, email := range emails.Emails {
		message := Message{Id: "todo", Subject: email.Subject} // TODO more email fields
		messages = append(messages, message)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, messages)
}
