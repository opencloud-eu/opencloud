package svc

import (
	"context"
	"crypto/tls"
	"net/http"
	"strconv"
	"time"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/config"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"

	"github.com/go-chi/render"

	"github.com/jellydator/ttlcache/v3"
)

const (
	logFolderId = "folder-id"
	logQuery    = "query"
)

type Groupware struct {
	logger            *log.Logger
	jmapClient        jmap.Client
	sessionCache      *ttlcache.Cache[string, jmap.Session]
	usernameProvider  jmap.HttpJmapUsernameProvider // we also need it for ourselves for now
	defaultEmailLimit int
	maxBodyValueBytes int
}

type ItemBody struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"` // text|html
}

type Message struct {
	Id                string    `json:"id"`
	CreatedDateTime   time.Time `json:"createdDateTime"`
	ReceivedDateTime  time.Time `json:"receivedDateTime"`
	HasAttachments    bool      `json:"hasAttachments"`
	InternetMessageId string    `json:"InternetMessageId"`
	Subject           string    `json:"subject"`
	BodyPreview       string    `json:"bodyPreview"`
	Body              ItemBody  `json:"body"`
}

func NewGroupware(logger *log.Logger, config *config.Config) *Groupware {
	baseUrl := config.Mail.BaseUrl
	jmapUrl := config.Mail.JmapUrl
	masterUsername := config.Mail.Master.Username
	masterPassword := config.Mail.Master.Password
	defaultEmailLimit := config.Mail.DefaultEmailLimit
	maxBodyValueBytes := config.Mail.MaxBodyValueBytes

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

	jmapClient := jmap.NewClient(api, api)

	loader := ttlcache.LoaderFunc[string, jmap.Session](
		func(c *ttlcache.Cache[string, jmap.Session], key string) *ttlcache.Item[string, jmap.Session] {
			jmapContext, err := jmapClient.FetchSession(key, logger)
			if err != nil {
				logger.Error().Err(err).Str("username", key).Msg("failed to retrieve well-known")
				return nil
			}
			item := c.Set(key, jmapContext, config.Mail.SessionCacheTTL)
			return item
		},
	)

	sessionCache := ttlcache.New(
		ttlcache.WithTTL[string, jmap.Session](
			config.Mail.SessionCacheTTL,
		),
		ttlcache.WithDisableTouchOnHit[string, jmap.Session](),
		ttlcache.WithLoader(loader),
	)
	go sessionCache.Start()

	return &Groupware{
		logger:            logger,
		jmapClient:        jmapClient,
		sessionCache:      sessionCache,
		usernameProvider:  jmapUsernameProvider,
		defaultEmailLimit: defaultEmailLimit,
		maxBodyValueBytes: maxBodyValueBytes,
	}
}

func pickInbox(folders jmap.Folders) string {
	for _, folder := range folders.Folders {
		if folder.Role == "inbox" {
			return folder.Id
		}
	}
	return ""
}

func (g Groupware) session(ctx context.Context, logger *log.Logger) (jmap.Session, error) {
	username, err := g.usernameProvider.GetUsername(ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve username")
		return jmap.Session{}, err
	}

	item := g.sessionCache.Get(username)
	return item.Value(), nil
}

func (g Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	logger := g.logger.SubloggerWithRequestID(ctx)

	session, err := g.session(ctx, &logger)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("failed to determine JMAP session")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	ctx = session.DecorateSession(ctx)
	logger = session.DecorateLogger(logger)

	offset, ok, _ := parseNumericParam(r, "$skip", 0)
	if ok {
		logger = log.Logger{Logger: logger.With().Int("$skip", offset).Logger()}
	}
	limit, ok, _ := parseNumericParam(r, "$top", g.defaultEmailLimit)
	if ok {
		logger = log.Logger{Logger: logger.With().Int("$top", limit).Logger()}
	}

	logger.Debug().Msg("fetching folders")
	folders, err := g.jmapClient.GetMailboxes(session, ctx, &logger)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("could not retrieve mailboxes")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	inboxId := pickInbox(folders)
	// TODO handle not found
	logger = log.Logger{Logger: logger.With().Str(logFolderId, inboxId).Logger()}

	logger.Debug().Msg("fetching emails from inbox")
	emails, err := g.jmapClient.GetEmails(session, ctx, &logger, inboxId, offset, limit, true, g.maxBodyValueBytes)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("could not retrieve emails from inbox")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	messages := make([]Message, 0, len(emails.Emails))
	for _, email := range emails.Emails {
		message := message(email, logger)
		messages = append(messages, message)
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, messages)
}

// https://learn.microsoft.com/en-us/graph/api/resources/message?view=graph-rest-1.0:w
func message(email jmap.Email, logger log.Logger) Message {
	var body ItemBody
	switch len(email.Bodies) {
	case 0:
		logger.Info().Msgf("zero bodies: %v", email)
	case 1:
		logger.Info().Msg("1 body")
		for mime, content := range email.Bodies {
			body = ItemBody{Content: content, ContentType: mime}
			logger.Debug().Msgf("one body: %v", mime)
		}
	default:
		content, ok := email.Bodies["text/html"]
		if ok {
			body = ItemBody{Content: content, ContentType: "text/html"}
			logger.Info().Msgf("%v bodies: picked text/html", len(email.Bodies))
		} else {
			content, ok = email.Bodies["text/plain"]
			if ok {
				body = ItemBody{Content: content, ContentType: "text/plain"}
				logger.Info().Msgf("%v bodies: picked text/plain", len(email.Bodies))
			} else {
				logger.Info().Msgf("%v bodies: neither text/html nor text/plain", len(email.Bodies))
				for mime, content := range email.Bodies {
					body = ItemBody{Content: content, ContentType: mime}
					logger.Info().Msgf("%v bodies: picked first: %v", len(email.Bodies), mime)
					break
				}
			}
		}
	}

	return Message{
		Id:                email.Id,
		Subject:           email.Subject,
		CreatedDateTime:   email.Received,
		ReceivedDateTime:  email.Received,
		HasAttachments:    email.HasAttachments,
		InternetMessageId: email.MessageId,
		BodyPreview:       email.Preview,
		Body:              body,
	} // TODO more email fields
}

func parseNumericParam(r *http.Request, param string, defaultValue int) (int, bool, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	value, err := strconv.ParseInt(str, 10, 0)
	if err != nil {
		return defaultValue, false, nil
	}
	return int(value), true, nil
}
