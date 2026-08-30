package svc

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/go-chi/render"
	"github.com/gobwas/glob"
	"github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/graph/pkg/errorcode"
)

type (
	// UsersUserProfilePhotoProvider is the interface that defines the methods for the user profile photo service
	UsersUserProfilePhotoProvider interface {
		// GetPhoto retrieves the requested photo
		GetPhoto(ctx context.Context, id string) ([]byte, error)

		// UpdatePhoto retrieves the requested photo
		UpdatePhoto(ctx context.Context, id string, r io.Reader) error

		// DeletePhoto deletes the requested photo
		DeletePhoto(ctx context.Context, id string) error

		// SyncPhotoFromURL downloads a profile picture from the given URL
		// and stores it for the specified user. The URL is validated against
		// the configured allowlist.
		SyncPhotoFromURL(ctx context.Context, userID, rawURL string) error
	}
)

var (
	// ErrNoBytes is returned when no bytes are found
	ErrNoBytes = errors.New("no bytes")

	// ErrInvalidContentType is returned when the content type is invalid
	ErrInvalidContentType = errors.New("invalid content type")

	// ErrMissingArgument is returned when a required argument is missing
	ErrMissingArgument = errors.New("required argument is missing")
)

// UsersUserProfilePhotoService is the implementation of the UsersUserProfilePhotoProvider interface
type UsersUserProfilePhotoService struct {
	storage      metadata.Storage
	httpClient   HTTPClient
	urlAllowlist []string
	oidcIssuer   string
}

// UserProfilePhotoOption configures a UsersUserProfilePhotoService.
type UserProfilePhotoOption func(*UsersUserProfilePhotoService)

// WithProfilePictureSync configures the service to sync profile pictures
// from OIDC claims. The HTTP client is used to download images, the
// allowlist restricts which URLs are accepted, and oidcIssuer is used
// to derive the default allowlist when urlAllowlist is empty.
func WithProfilePictureSync(httpClient HTTPClient, urlAllowlist []string, oidcIssuer string) UserProfilePhotoOption {
	return func(s *UsersUserProfilePhotoService) {
		s.httpClient = httpClient
		s.urlAllowlist = urlAllowlist
		s.oidcIssuer = oidcIssuer
	}
}

// NewUsersUserProfilePhotoService creates a new UsersUserProfilePhotoService
func NewUsersUserProfilePhotoService(storage metadata.Storage, opts ...UserProfilePhotoOption) (UsersUserProfilePhotoService, error) {
	svc := UsersUserProfilePhotoService{
		storage: storage,
	}
	for _, opt := range opts {
		opt(&svc)
	}
	return svc, nil
}

// GetPhoto retrieves the requested photo
func (s UsersUserProfilePhotoService) GetPhoto(ctx context.Context, id string) ([]byte, error) {
	return s.storage.SimpleDownload(ctx, id)
}

// DeletePhoto deletes the requested photo
func (s UsersUserProfilePhotoService) DeletePhoto(ctx context.Context, id string) error {
	return s.storage.Delete(ctx, id)
}

// UpdatePhoto updates the requested photo
func (s UsersUserProfilePhotoService) UpdatePhoto(ctx context.Context, id string, r io.Reader) error {
	if id == "" {
		return fmt.Errorf("%w: %s", ErrMissingArgument, "id")
	}

	photo, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	if len(photo) == 0 {
		return ErrNoBytes
	}

	contentType := http.DetectContentType(photo)
	if !strings.HasPrefix(contentType, "image/") {
		return fmt.Errorf("%w: %s", ErrInvalidContentType, contentType)
	}

	return s.storage.SimpleUpload(ctx, id, photo)
}

// maxProfilePhotoBytes limits the size of downloaded profile pictures.
const maxProfilePhotoBytes = 10 << 20 // 10 MiB

// SyncPhotoFromURL downloads a profile picture from the given URL and
// stores it for the specified user. The URL is validated against the
// configured allowlist before downloading.
func (s UsersUserProfilePhotoService) SyncPhotoFromURL(ctx context.Context, userID, rawURL string) error {
	if userID == "" {
		return fmt.Errorf("%w: %s", ErrMissingArgument, "userID")
	}
	if !s.isProfilePictureURLAllowed(rawURL) {
		return fmt.Errorf("profile picture URL not allowed: %s", rawURL)
	}

	data, err := s.fetchProfilePicture(ctx, rawURL)
	if err != nil {
		return err
	}

	return s.UpdatePhoto(ctx, userID, bytes.NewReader(data))
}

func (s UsersUserProfilePhotoService) fetchProfilePicture(ctx context.Context, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "image/*")

	resp, err := s.httpClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("profile picture request returned %s", resp.Status)
	}

	limited := io.LimitReader(resp.Body, int64(maxProfilePhotoBytes)+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, errors.New("profile picture response was empty")
	}
	if len(data) > maxProfilePhotoBytes {
		return nil, fmt.Errorf("profile picture exceeds %d bytes", maxProfilePhotoBytes)
	}
	contentType := http.DetectContentType(data)
	if !strings.HasPrefix(contentType, "image/") {
		return nil, fmt.Errorf("unsupported profile picture content type: %s", contentType)
	}

	return data, nil
}

func (s UsersUserProfilePhotoService) isProfilePictureURLAllowed(rawURL string) bool {
	parsedURL, err := url.Parse(rawURL)
	if err != nil || parsedURL.Host == "" {
		return false
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return false
	}

	for _, pattern := range s.profilePictureAllowlistPatterns() {
		if pattern == "*" {
			return true
		}
		if urlPatternMatches(pattern, parsedURL) {
			return true
		}
	}
	return false
}

func (s UsersUserProfilePhotoService) profilePictureAllowlistPatterns() []string {
	if len(s.urlAllowlist) > 0 {
		return s.urlAllowlist
	}
	issuerURL, err := url.Parse(s.oidcIssuer)
	if err != nil || issuerURL.Host == "" {
		return nil
	}
	return []string{fmt.Sprintf("%s://%s", issuerURL.Scheme, issuerURL.Host)}
}

func urlPatternMatches(pattern string, target *url.URL) bool {
	if target == nil {
		return false
	}
	parsedPattern, err := url.Parse(pattern)
	if err == nil && parsedPattern.Host != "" {
		if parsedPattern.Scheme != "" && !strings.EqualFold(parsedPattern.Scheme, target.Scheme) {
			return false
		}
		hostMatcher, err := glob.Compile(strings.ToLower(parsedPattern.Host))
		if err != nil {
			return false
		}
		if !hostMatcher.Match(strings.ToLower(target.Host)) {
			return false
		}
		if parsedPattern.Path == "" || parsedPattern.Path == "/" {
			return true
		}
		if strings.HasSuffix(parsedPattern.Path, "*") {
			prefix := strings.TrimSuffix(parsedPattern.Path, "*")
			return strings.HasPrefix(target.Path, prefix)
		}
		return path.Clean(parsedPattern.Path) == path.Clean(target.Path)
	}

	hostMatcher, err := glob.Compile(strings.ToLower(pattern))
	if err != nil {
		return false
	}
	return hostMatcher.Match(strings.ToLower(target.Host))
}

// UsersUserProfilePhotoApi contains all photo related api endpoints
type UsersUserProfilePhotoApi struct {
	logger                       log.Logger
	usersUserProfilePhotoService UsersUserProfilePhotoProvider
}

// NewUsersUserProfilePhotoApi creates a new UsersUserProfilePhotoApi
func NewUsersUserProfilePhotoApi(usersUserProfilePhotoService UsersUserProfilePhotoProvider, logger log.Logger) (UsersUserProfilePhotoApi, error) {
	return UsersUserProfilePhotoApi{
		logger:                       log.Logger{Logger: logger.With().Str("graph api", "UsersUserProfilePhotoApi").Logger()},
		usersUserProfilePhotoService: usersUserProfilePhotoService,
	}, nil
}

// GetProfilePhoto creates a handler which renders the corresponding photo
func (api UsersUserProfilePhotoApi) GetProfilePhoto(h HTTPDataHandler[string]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, ok := h(w, r)
		if !ok {
			return
		}

		photo, err := api.usersUserProfilePhotoService.GetPhoto(r.Context(), v)
		if err != nil {
			api.logger.Debug().Err(err)
			errorcode.GeneralException.Render(w, r, http.StatusNotFound, "failed to get photo")
			return
		}

		render.Status(r, http.StatusOK)
		_, _ = w.Write(photo)
	}
}

// UpsertProfilePhoto creates a handler which updates or creates the corresponding photo
func (api UsersUserProfilePhotoApi) UpsertProfilePhoto(h HTTPDataHandler[string]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, ok := h(w, r)
		if !ok {
			return
		}

		if err := api.usersUserProfilePhotoService.UpdatePhoto(r.Context(), v, r.Body); err != nil {
			api.logger.Debug().Err(err)
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "failed to update photo")
			return
		}
		defer func() {
			_ = r.Body.Close()
		}()

		render.Status(r, http.StatusOK)
	}
}

// DeleteProfilePhoto creates a handler which deletes the corresponding photo
func (api UsersUserProfilePhotoApi) DeleteProfilePhoto(h HTTPDataHandler[string]) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		v, ok := h(w, r)
		if !ok {
			return
		}

		if err := api.usersUserProfilePhotoService.DeletePhoto(r.Context(), v); err != nil {
			api.logger.Debug().Err(err)
			errorcode.GeneralException.Render(w, r, http.StatusInternalServerError, "failed to delete photo")
			return
		}

		render.Status(r, http.StatusOK)
	}
}
