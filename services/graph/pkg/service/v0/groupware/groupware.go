package groupware

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
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
	io.Closer
}

var _ io.Closer = Groupware{}

type ItemBody struct {
	Content     string `json:"content"`
	ContentType string `json:"contentType"` // text|html
}

type EmailAddress struct {
	Address string `json:"address"`
	Name    string `json:"name"`
}

type Messages struct {
	Context string    `json:"@odata.context,omitempty"`
	Value   []Message `json:"value"`
}

type Message struct {
	Etag              string         `json:"@odata.etag,omitempty"`
	Id                string         `json:"id,omitempty"`
	CreatedDateTime   time.Time      `json:"createdDateTime,omitzero"`
	ReceivedDateTime  time.Time      `json:"receivedDateTime,omitzero"`
	SentDateTime      time.Time      `json:"sentDateTime,omitzero"`
	HasAttachments    bool           `json:"hasAttachments,omitempty"`
	InternetMessageId string         `json:"internetMessageId,omitempty"`
	Subject           string         `json:"subject,omitempty"`
	BodyPreview       string         `json:"bodyPreview,omitempty"`
	Body              *ItemBody      `json:"body,omitempty"`
	From              *EmailAddress  `json:"from,omitempty"`
	ToRecipients      []EmailAddress `json:"toRecipients,omitempty"`
	CcRecipients      []EmailAddress `json:"ccRecipients,omitempty"`
	BccRecipients     []EmailAddress `json:"bccRecipients,omitempty"`
	ReplyTo           []EmailAddress `json:"replyTo,omitempty"`
	IsRead            bool           `json:"isRead,omitempty"`
	IsDraft           bool           `json:"isDraft,omitempty"`
	Importance        string         `json:"importance,omitempty"`
	ParentFolderId    string         `json:"parentFolderId,omitempty"`
	Categories        []string       `json:"categories,omitempty"`
	ConversationId    string         `json:"conversationId,omitempty"`
	WebLink           string         `json:"webLink,omitempty"`
	// ConversationIndex string         `json:"conversationIndex"`
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

	sessionCache := ttlcache.New(
		ttlcache.WithTTL[string, jmap.Session](
			config.Mail.SessionCacheTTL,
		),
		ttlcache.WithDisableTouchOnHit[string, jmap.Session](),
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

func (g Groupware) Close() error {
	g.sessionCache.Stop()
	return nil
}

func (g Groupware) session(req *http.Request, ctx context.Context, logger *log.Logger) (jmap.Session, bool, error) {
	username, err := g.usernameProvider.GetUsername(req, ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve username")
		return jmap.Session{}, false, err
	}

	fetchErrRef := atomic.Value{}
	item, _ := g.sessionCache.GetOrSetFunc(username, func() jmap.Session {
		jmapContext, err := g.jmapClient.FetchSession(username, logger)
		if err != nil {
			fetchErrRef.Store(err)
			logger.Error().Err(err).Str("username", username).Msg("failed to retrieve well-known")
			return jmap.Session{}
		}
		return jmapContext
	})
	p := fetchErrRef.Load()
	if p != nil {
		err = p.(error)
		return jmap.Session{}, false, err
	}
	if item != nil {
		return item.Value(), true, nil
	} else {
		return jmap.Session{}, false, nil
	}
}

func (g Groupware) withSession(w http.ResponseWriter, r *http.Request, handler func(r *http.Request, ctx context.Context, logger log.Logger, session *jmap.Session) (any, error)) {
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)
	session, ok, err := g.session(r, ctx, &logger)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg("failed to determine JMAP session")
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}
	if !ok {
		// no session = authentication failed
		logger.Warn().Err(err).Interface(logQuery, r.URL.Query()).Msg("could not authenticate")
		errorcode.AccessDenied.Render(w, r, http.StatusForbidden, "failed to authenticate")
		return
	}
	logger = session.DecorateLogger(logger)

	response, err := handler(r, ctx, logger, &session)
	if err != nil {
		logger.Error().Err(err).Interface(logQuery, r.URL.Query()).Msg(err.Error())
		errorcode.ServiceNotAvailable.Render(w, r, http.StatusInternalServerError, err.Error())
		return
	}

	render.Status(r, http.StatusOK)
	render.JSON(w, r, response)
}

func (g Groupware) GetIdentity(w http.ResponseWriter, r *http.Request) {
	g.withSession(w, r, func(r *http.Request, ctx context.Context, logger log.Logger, session *jmap.Session) (any, error) {
		return g.jmapClient.GetIdentity(session, ctx, &logger)
	})
}

func (g Groupware) GetVacation(w http.ResponseWriter, r *http.Request) {
	g.withSession(w, r, func(r *http.Request, ctx context.Context, logger log.Logger, session *jmap.Session) (any, error) {
		return g.jmapClient.GetVacationResponse(session, ctx, &logger)
	})
}

func (g Groupware) GetMessages(w http.ResponseWriter, r *http.Request) {
	g.withSession(w, r, func(r *http.Request, ctx context.Context, logger log.Logger, session *jmap.Session) (any, error) {
		offset, ok, _ := parseNumericParam(r, "$skip", 0)
		if ok {
			logger = log.Logger{Logger: logger.With().Int("$skip", offset).Logger()}
		}
		limit, ok, _ := parseNumericParam(r, "$top", g.defaultEmailLimit)
		if ok {
			logger = log.Logger{Logger: logger.With().Int("$top", limit).Logger()}
		}

		mailboxGetResponse, err := g.jmapClient.GetAllMailboxes(session, ctx, &logger)
		if err != nil {
			return nil, err
		}

		inboxId := pickInbox(mailboxGetResponse.List)
		if inboxId == "" {
			return nil, fmt.Errorf("failed to find an inbox folder")
		}
		logger = log.Logger{Logger: logger.With().Str(logFolderId, inboxId).Logger()}

		emails, err := g.jmapClient.GetEmails(session, ctx, &logger, inboxId, offset, limit, true, g.maxBodyValueBytes)
		if err != nil {
			return nil, err
		}

		messages := make([]Message, 0, len(emails.Emails))
		for _, email := range emails.Emails {
			message := message(email, emails.State)
			messages = append(messages, message)
		}

		odataContext := *r.URL
		odataContext.Path = fmt.Sprintf("/graph/v1.0/$metadata#users('%s')/mailFolders('%s')/messages()", session.Username, inboxId)
		return Messages{Context: odataContext.String(), Value: messages}, nil
	})
}
