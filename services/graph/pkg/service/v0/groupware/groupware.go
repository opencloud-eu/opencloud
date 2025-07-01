package groupware

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
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

func pickInbox(folders []jmap.Mailbox) string {
	for _, folder := range folders {
		if folder.Role == "inbox" {
			return folder.Id
		}
	}
	return ""
}

func (g Groupware) session(ctx context.Context, logger *log.Logger) (jmap.Session, bool, error) {
	username, err := g.usernameProvider.GetUsername(ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("failed to retrieve username")
		return jmap.Session{}, false, err
	}

	item := g.sessionCache.Get(username)
	if item != nil {
		return item.Value(), true, nil
	} else {
		return jmap.Session{}, false, nil
	}
}

func (g Groupware) withSession(w http.ResponseWriter, r *http.Request, handler func(r *http.Request, ctx context.Context, logger log.Logger, session *jmap.Session) (any, error)) {
	ctx := r.Context()
	logger := g.logger.SubloggerWithRequestID(ctx)
	session, ok, err := g.session(ctx, &logger)
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
		return g.jmapClient.GetVacation(session, ctx, &logger)
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

		mailboxGetResponse, err := g.jmapClient.GetMailboxes(session, ctx, &logger)
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

func mapContentType(jmap string) string {
	switch jmap {
	case "text/html":
		return "html"
	case "text/plain":
		return "text"
	default:
		return jmap
	}
}

func foldBody(email jmap.Email) *ItemBody {
	if email.BodyValues != nil {
		if len(email.HtmlBody) > 0 {
			pick := email.HtmlBody[0]
			content, ok := email.BodyValues[pick.PartId]
			if ok {
				return &ItemBody{Content: content.Value, ContentType: mapContentType(pick.Type)}
			}
		}
		if len(email.TextBody) > 0 {
			pick := email.TextBody[0]
			content, ok := email.BodyValues[pick.PartId]
			if ok {
				return &ItemBody{Content: content.Value, ContentType: mapContentType(pick.Type)}
			}
		}
	}
	return nil
}

func firstOf[T any](ary []T) T {
	if len(ary) > 0 {
		return ary[0]
	}
	var nothing T
	return nothing
}

func emailAddress(j jmap.EmailAddress) EmailAddress {
	return EmailAddress{Address: j.Email, Name: j.Name}
}

func emailAddresses(j []jmap.EmailAddress) []EmailAddress {
	result := make([]EmailAddress, len(j))
	for i := 0; i < len(j); i++ {
		result[i] = emailAddress(j[i])
	}
	return result
}

func hasKeyword(j jmap.Email, kw string) bool {
	value, ok := j.Keywords[kw]
	if ok {
		return value
	}
	return false
}

func categories(j jmap.Email) []string {
	categories := []string{}
	for k, v := range j.Keywords {
		if v && !strings.HasPrefix(k, jmap.JmapKeywordPrefix) {
			categories = append(categories, k)
		}
	}
	return categories
}

/*
func toEdmBinary(value int) string {
	return fmt.Sprintf("%X", value)
}
*/

// https://learn.microsoft.com/en-us/graph/api/resources/message?view=graph-rest-1.0
func message(email jmap.Email, state string) Message {
	body := foldBody(email)
	importance := "" // omit "normal" as it is expected to be the default
	if hasKeyword(email, jmap.JmapKeywordFlagged) {
		importance = "high"
	}

	mailboxId := ""
	for k, v := range email.MailboxIds {
		if v {
			// TODO how to map JMAP short identifiers (e.g. 'a') to something uniquely addressable for the clients?
			// e.g. do we need to include tenant/sharding/cluster information?
			mailboxId = k
			break
		}
	}

	// TODO how to map JMAP short identifiers (e.g. 'a') to something uniquely addressable for the clients?
	// e.g. do we need to include tenant/sharding/cluster information?
	id := email.Id
	// for this one too:
	messageId := firstOf(email.MessageId)
	// as well as this one:
	threadId := email.ThreadId

	categories := categories(email)

	var from *EmailAddress = nil
	if len(email.From) > 0 {
		e := emailAddress(email.From[0])
		from = &e
	}

	// TODO how to map JMAP state to an OData Etag?
	etag := state

	weblink, err := url.JoinPath("/groupware/mail", id)
	if err != nil {
		weblink = ""
	}

	return Message{
		Etag:              etag,
		Id:                id,
		Subject:           email.Subject,
		CreatedDateTime:   email.ReceivedAt,
		ReceivedDateTime:  email.ReceivedAt,
		SentDateTime:      email.SentAt,
		HasAttachments:    email.HasAttachments,
		InternetMessageId: messageId,
		BodyPreview:       email.Preview,
		Body:              body,
		From:              from,
		ToRecipients:      emailAddresses(email.To),
		CcRecipients:      emailAddresses(email.Cc),
		BccRecipients:     emailAddresses(email.Bcc),
		ReplyTo:           emailAddresses(email.ReplyTo),
		IsRead:            hasKeyword(email, jmap.JmapKeywordSeen),
		IsDraft:           hasKeyword(email, jmap.JmapKeywordDraft),
		Importance:        importance,
		ParentFolderId:    mailboxId,
		Categories:        categories,
		ConversationId:    threadId,
		WebLink:           weblink,
		// ConversationIndex: toEdmBinary(email.ThreadIndex),
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
