package svc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"

	"github.com/go-chi/render"
	revaMetadata "github.com/opencloud-eu/reva/v2/pkg/storage/utils/metadata"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/storage/metadata"
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
	}
)

var (
	// ErrNoBytes is returned when no bytes are found
	ErrNoBytes = errors.New("no bytes")

	// ErrInvalidContentType is returned when the content type is invalid
	ErrInvalidContentType = errors.New("invalid content type")

	// ErrMissingArgument is returned when a required argument is missing
	ErrMissingArgument = errors.New("required argument is missing")

	// ProfilePhotoPath is a function that returns the path for the profile photo of a user
	ProfilePhotoPath = func(id string) string {
		return path.Join(id, "profile_photo")
	}
)

// UsersUserProfilePhotoService is the implementation of the UsersUserProfilePhotoProvider interface
type UsersUserProfilePhotoService struct {
	storage *metadata.Deep
}

// NewUsersUserProfilePhotoService creates a new UsersUserProfilePhotoService
func NewUsersUserProfilePhotoService(storage revaMetadata.Storage) (UsersUserProfilePhotoService, error) {
	deepStorage, err := metadata.NewDeepStorage(storage)
	if err != nil {
		return UsersUserProfilePhotoService{}, fmt.Errorf("could not create deep storage: %w", err)
	}

	return UsersUserProfilePhotoService{
		storage: deepStorage,
	}, nil
}

// GetPhoto retrieves the requested photo
func (s UsersUserProfilePhotoService) GetPhoto(ctx context.Context, id string) ([]byte, error) {
	return s.storage.SimpleDownload(ctx, ProfilePhotoPath(id))
}

// DeletePhoto deletes the requested photo
func (s UsersUserProfilePhotoService) DeletePhoto(ctx context.Context, id string) error {
	return s.storage.Delete(ctx, ProfilePhotoPath(id))
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

	return s.storage.SimpleUpload(ctx, ProfilePhotoPath(id), photo)
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
