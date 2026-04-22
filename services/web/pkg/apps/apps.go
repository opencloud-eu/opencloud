package apps

import (
	"encoding/json"
	"errors"
	"io/fs"
	"path"

	"dario.cat/mergo"
	"github.com/go-playground/validator/v10"
	"golang.org/x/exp/maps"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/path/filepathx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

var (
	// ErrInvalidApp is the error when an app is invalid
	ErrInvalidApp = errors.New("invalid app")

	// ErrMissingManifest is the error when the manifest is missing
	ErrMissingManifest = errors.New("missing manifest")

	// ErrInvalidManifest is the error when the manifest is invalid
	ErrInvalidManifest = errors.New("invalid manifest")

	// ErrEntrypointDoesNotExist is the error when the entrypoint does not exist or is not a file
	ErrEntrypointDoesNotExist = errors.New("entrypoint does not exist")

	validate = validator.New(validator.WithRequiredStructEnabled())
)

const (
	// _manifest is the name of the manifest file for an application
	_manifest = "manifest.json"

	// _config contains the dedicated app configuration
	_config = "config.json"
)

// Application contains the metadata of an application
type Application struct {
	// ID is the unique identifier of the application
	ID string `json:"-"`

	// Disabled is a flag to disable the application
	Disabled bool `json:"disabled,omitempty"`

	// Entrypoint is the entrypoint of the application within the bundle
	Entrypoint string `json:"entrypoint" validate:"required"`

	// Config contains the application-specific configuration
	Config map[string]any `json:"config,omitempty"`
}

// ToExternal converts an Application to an ExternalApp configuration
func (a Application) ToExternal(entrypoint string) config.ExternalApp {
	return config.ExternalApp{
		ID:     a.ID,
		Path:   filepathx.JailJoin(entrypoint, a.Entrypoint),
		Config: a.Config,
	}
}

// List returns a list of applications from the given filesystems,
// individual filesystems are searched for applications, and the list is merged.
// Last finding gets priority in case of conflicts, so the order of the filesystems is important.
func List(logger log.Logger, data map[string]config.App, fSystems ...fs.FS) []Application {
	registry := map[string]Application{}

	for _, fSystem := range fSystems {
		if fSystem == nil {
			continue
		}

		entries, err := fs.ReadDir(fSystem, ".")
		if err != nil {
			// skip non-directory listings, every app needs to be contained inside a directory
			continue
		}

		for _, entry := range entries {
			var appData config.App
			name := entry.Name()

			// configuration for the application is optional, if it is not present, the default configuration is used
			if data, ok := data[name]; ok {
				appData = data
			}

			application, err := build(fSystem, name, appData)
			if err != nil {
				// if app creation fails, log the error and continue with the next app
				logger.Debug().Err(err).Str("path", entry.Name()).Msg("failed to load application")
				continue
			}

			if application.Disabled {
				// if the app is disabled, skip it
				continue
			}

			// everything is fine, add the application to the list of applications
			registry[name] = application
		}
	}

	return maps.Values(registry)
}

func build(fSystem fs.FS, id string, globalConfig config.App) (Application, error) {
	// skip non-directory listings, every app needs to be contained inside a directory
	entry, err := fs.Stat(fSystem, id)
	if err != nil || !entry.IsDir() {
		return Application{}, ErrInvalidApp
	}

	var application Application
	// build the application
	{
		r, err := fSystem.Open(path.Join(id, _manifest))
		if err != nil {
			return Application{}, errors.Join(err, ErrMissingManifest)
		}
		defer r.Close()

		if json.NewDecoder(r).Decode(&application) != nil {
			return Application{}, errors.Join(err, ErrInvalidManifest)
		}

		if err := validate.Struct(application); err != nil {
			return Application{}, errors.Join(err, ErrInvalidManifest)
		}
	}

	var localConfig config.App
	// build the local configuration
	{
		r, err := fSystem.Open(path.Join(id, _config))
		if err == nil {
			defer r.Close()
		}

		// as soon as we have a local configuration, we expect it to be valid
		if err == nil && json.NewDecoder(r).Decode(&localConfig) != nil {
			return Application{}, errors.Join(err, ErrInvalidManifest)
		}
	}

	// apply overloads the application with the source configuration,
	// not all fields are considered secure, therefore, the allowed values are hand-picked
	overloadApplication := func(source config.App) error {
		// overload the configuration,
		// configuration options are considered secure and can be overloaded
		err = mergo.Merge(&application.Config, source.Config, mergo.WithOverride)
		if err != nil {
			return err
		}

		// overload the disabled state, consider it secure and allow overloading
		application.Disabled = source.Disabled

		return nil
	}

	// overload the application configuration from the manifest with the local and global configuration
	// priority is local.config > global.config > manifest.config
	{
		// overload the default configuration with the global application-specific configuration,
		// that configuration has priority, and failing is fine here
		_ = overloadApplication(globalConfig)

		// overload the default configuration with the local application-specific configuration,
		// that configuration has priority, and failing is fine here
		_ = overloadApplication(localConfig)
	}

	// the entrypoint is jailed to the app directory
	application.Entrypoint = filepathx.JailJoin(id, application.Entrypoint)
	info, err := fs.Stat(fSystem, application.Entrypoint)
	switch {
	case err != nil:
		return Application{}, errors.Join(err, ErrEntrypointDoesNotExist)
	case info.IsDir():
		return Application{}, ErrEntrypointDoesNotExist
	}

	application.ID = id

	return application, nil
}
