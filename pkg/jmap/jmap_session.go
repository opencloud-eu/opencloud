package jmap

import (
	"fmt"
	"net/url"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type SessionEventListener interface {
	OnSessionOutdated(session *Session)
}

// Cached user related information
//
// This information is typically retrieved once (or at least for a certain period of time) from the
// JMAP well-known endpoint of Stalwart and then kept in cache to avoid the performance cost of
// retrieving it over and over again.
//
// This is really only needed due to the Graph API limitations, since ideally, the account ID should
// be passed as a request parameter by the UI, in order to support a user having multiple accounts.
//
// Keeping track of the JMAP URL might be useful though, in case of Stalwart sharding strategies making
// use of that, by providing different URLs for JMAP on a per-user basis, and that is not something
// we would want to query before every single JMAP request. On the other hand, that then also creates
// a risk of going out-of-sync, e.g. if a node is down and the user is reassigned to a different node.
// There might be webhooks to subscribe to in Stalwart to be notified of such situations, in which case
// the Session needs to be removed from the cache.
//
// The Username is only here for convenience, it could just as well be passed as a separate parameter
// instead of being part of the Session, since the username is always part of the request (typically in
// the authentication token payload.)
type Session struct {
	// The name of the user to use to authenticate against Stalwart
	Username string

	// The base URL to use for JMAP operations towards Stalwart
	JmapUrl url.URL

	// The upload URL template
	UploadUrlTemplate string

	// The upload URL template
	DownloadUrlTemplate string

	// TODO
	DefaultMailAccountId string

	SessionResponse
}

// Create a new Session from a SessionResponse.
func newSession(sessionResponse SessionResponse) (Session, Error) {
	username := sessionResponse.Username
	if username == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide a username")}
	}
	mailAccountId := sessionResponse.PrimaryAccounts.Mail
	if mailAccountId == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide a primary mail account")}
	}
	apiStr := sessionResponse.ApiUrl
	if apiStr == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide an API URL")}
	}
	apiUrl, err := url.Parse(apiStr)
	if err != nil {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response provides an invalid API URL")}
	}
	uploadUrl := sessionResponse.UploadUrl
	if uploadUrl == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide an upload URL")}
	}
	downloadUrl := sessionResponse.DownloadUrl
	if downloadUrl == "" {
		return Session{}, SimpleError{code: JmapErrorInvalidSessionResponse, err: fmt.Errorf("JMAP session response does not provide an download URL")}
	}

	return Session{
		Username:             username,
		DefaultMailAccountId: mailAccountId,
		JmapUrl:              *apiUrl,
		UploadUrlTemplate:    uploadUrl,
		DownloadUrlTemplate:  downloadUrl,
		SessionResponse:      sessionResponse,
	}, nil
}

func (s *Session) MailAccountId(accountId string) string {
	if accountId != "" && accountId != defaultAccountId {
		return accountId
	}
	// TODO(pbleser-oc) handle case where there is no default mail account
	return s.DefaultMailAccountId
}

func (s *Session) BlobAccountId(accountId string) string {
	if accountId != "" && accountId != defaultAccountId {
		return accountId
	}
	// TODO(pbleser-oc) handle case where there is no default blob account
	return s.PrimaryAccounts.Blob
}

func (s *Session) SubmissionAccountId(accountId string) string {
	if accountId != "" && accountId != defaultAccountId {
		return accountId
	}
	// TODO(pbleser-oc) handle case where there is no default submission account
	return s.PrimaryAccounts.Submission
}

// Create a new log.Logger that is decorated with fields containing information about the Session.
func (s Session) DecorateLogger(l log.Logger) log.Logger {
	return log.Logger{Logger: l.With().
		Str(logUsername, s.Username).
		Str(logApiUrl, s.ApiUrl).
		Str(logSessionState, s.State).
		Logger()}
}
