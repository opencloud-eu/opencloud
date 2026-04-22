package apps_test

import (
	"io/fs"
	"testing"
	"testing/fstest"

	"github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/web/pkg/apps"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

func TestApplication_ToExternal(t *testing.T) {
	g := gomega.NewWithT(t)
	app := apps.Application{
		ID:         "app",
		Entrypoint: "entrypoint.js",
		Config: map[string]any{
			"foo": "bar",
		},
	}

	externalApp := app.ToExternal("path")

	g.Expect(externalApp.ID).To(gomega.Equal("app"))
	g.Expect(externalApp.Path).To(gomega.Equal("path/entrypoint.js"))
	g.Expect(externalApp.Config).To(gomega.Equal(app.Config))
}

func TestBuild(t *testing.T) {
	g := gomega.NewWithT(t)
	dir := &fstest.MapFile{
		Mode: fs.ModeDir,
	}

	{
		_, err := apps.Build(fstest.MapFS{
			"app": &fstest.MapFile{},
		}, "app", config.App{})
		g.Expect(err).To(gomega.MatchError(apps.ErrInvalidApp))
	}

	{
		_, err := apps.Build(fstest.MapFS{
			"app": dir,
		}, "app", config.App{})
		g.Expect(err).To(gomega.MatchError(apps.ErrMissingManifest))
	}

	{
		_, err := apps.Build(fstest.MapFS{
			"app":               dir,
			"app/manifest.json": dir,
		}, "app", config.App{})
		g.Expect(err).To(gomega.MatchError(apps.ErrInvalidManifest))
	}

	{
		_, err := apps.Build(fstest.MapFS{
			"app": dir,
			"app/manifest.json": &fstest.MapFile{
				Data: []byte("{}"),
			},
		}, "app", config.App{})
		g.Expect(err).To(gomega.MatchError(apps.ErrInvalidManifest))

	}

	{
		_, err := apps.Build(fstest.MapFS{
			"app":               dir,
			"app/entrypoint.js": &fstest.MapFile{},
			"app/manifest.json": &fstest.MapFile{
				Data: []byte(`{"id":"app", "entrypoint":"entrypoint.js"}`),
			},
		}, "app", config.App{})
		g.Expect(err).ToNot(gomega.HaveOccurred())
	}

	{
		_, err := apps.Build(fstest.MapFS{
			"app":               dir,
			"app/entrypoint.js": dir,
			"app/manifest.json": &fstest.MapFile{
				Data: []byte(`{"id":"app", "entrypoint":"entrypoint.js"}`),
			},
		}, "app", config.App{})
		g.Expect(err).To(gomega.MatchError(apps.ErrEntrypointDoesNotExist))
	}

	{
		_, err := apps.Build(fstest.MapFS{
			"app": dir,
			"app/manifest.json": &fstest.MapFile{
				Data: []byte(`{"id":"app", "entrypoint":"entrypoint.js"}`),
			},
		}, "app", config.App{})
		g.Expect(err).To(gomega.MatchError(apps.ErrEntrypointDoesNotExist))
	}

	{
		application, err := apps.Build(fstest.MapFS{
			"app":               dir,
			"app/entrypoint.js": &fstest.MapFile{},
			"app/manifest.json": &fstest.MapFile{
				Data: []byte(`{"id":"app", "entrypoint":"entrypoint.js", "config": {"k1": "1", "k2": "2", "k3": "3"}}`),
			},
			"app/config.json": &fstest.MapFile{
				Data: []byte(`{"config": {"k2": "overwritten-from-config.json", "injected_from_config_json": "11"}}`),
			},
		}, "app", config.App{Config: map[string]any{"k2": "overwritten-from-apps.yaml", "k3": "overwritten-from-apps.yaml", "injected_from_apps_yaml": "22"}})
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(application.Entrypoint).To(gomega.Equal("app/entrypoint.js"))
		g.Expect(application.Config).To(gomega.Equal(map[string]any{
			"k1": "1", "k2": "overwritten-from-config.json", "k3": "overwritten-from-apps.yaml", "injected_from_config_json": "11", "injected_from_apps_yaml": "22",
		}))
	}
}

func TestList(t *testing.T) {
	g := gomega.NewWithT(t)

	applications := apps.List(log.NopLogger(), map[string]config.App{})
	g.Expect(applications).To(gomega.BeEmpty())

	applications = apps.List(log.NopLogger(), map[string]config.App{}, nil)
	g.Expect(applications).To(gomega.BeEmpty())

	applications = apps.List(log.NopLogger(), map[string]config.App{}, fstest.MapFS{})
	g.Expect(applications).To(gomega.BeEmpty())

	dir := &fstest.MapFile{
		Mode: fs.ModeDir,
	}

	applications = apps.List(log.NopLogger(), map[string]config.App{
		"app": {
			Disabled: true,
		},
	}, fstest.MapFS{
		"app": dir,
	})
	g.Expect(applications).To(gomega.BeEmpty())

	applications = apps.List(log.NopLogger(), map[string]config.App{
		"app": {},
	}, fstest.MapFS{
		"app": dir,
	})
	g.Expect(applications).To(gomega.BeEmpty())

	applications = apps.List(log.NopLogger(), map[string]config.App{
		"app-3": {
			Config: map[string]any{
				"foo": "local conf 1",
				"bar": "local conf 2",
			},
		},
	}, fstest.MapFS{
		"app-1":               dir,
		"app-1/entrypoint.js": &fstest.MapFile{},
		"app-1/manifest.json": &fstest.MapFile{
			Data: []byte(`{"id":"app-1", "entrypoint":"entrypoint.js", "config": {"foo": "fs1"}}`),
		},
		"app-2":               dir,
		"app-2/entrypoint.js": &fstest.MapFile{},
		"app-2/manifest.json": &fstest.MapFile{
			Data: []byte(`{"id":"app-2", "entrypoint":"entrypoint.js", "config": {"foo": "fs1"}}`),
		},
	}, fstest.MapFS{
		"app-1":               dir,
		"app-1/entrypoint.js": &fstest.MapFile{},
		"app-1/manifest.json": &fstest.MapFile{
			Data: []byte(`{"id":"app-1", "entrypoint":"entrypoint.js", "config": {"foo": "fs2"}}`),
		},
		"app-3":               dir,
		"app-3/entrypoint.js": &fstest.MapFile{},
		"app-3/manifest.json": &fstest.MapFile{
			Data: []byte(`{"id":"app-3", "entrypoint":"entrypoint.js", "config": {"foo": "fs2"}}`),
		},
	})
	g.Expect(len(applications)).To(gomega.Equal(3))

	for _, application := range applications {
		switch {
		case application.Entrypoint == "app-1/entrypoint.js":
			g.Expect(application.Config["foo"]).To(gomega.Equal("fs2"))
		case application.Entrypoint == "app-2/entrypoint.js":
			g.Expect(application.Config["foo"]).To(gomega.Equal("fs1"))
		case application.Entrypoint == "app-3/entrypoint.js":
			g.Expect(application.Config["foo"]).To(gomega.Equal("local conf 1"))
			g.Expect(application.Config["bar"]).To(gomega.Equal("local conf 2"))
		default:
			t.Fatalf("unexpected application %s", application.Entrypoint)
		}
	}

}
