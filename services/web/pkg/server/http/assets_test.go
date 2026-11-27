package http

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/onsi/gomega"
	"github.com/spf13/afero"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
	"github.com/opencloud-eu/opencloud/services/web/pkg/config"
)

// newAssetDirs creates the three asset directories, each with a regular file
// and a symlink to a file outside of it.
func newAssetDirs(t *testing.T) (config.Asset, string) {
	t.Helper()
	g := gomega.NewWithT(t)
	// resolve the temp dir itself: on macOS, it is behind a symlink (/var -> /private/var)
	base, err := filepath.EvalSymlinks(t.TempDir())
	g.Expect(err).ToNot(gomega.HaveOccurred())

	outside := filepath.Join(base, "outside")
	g.Expect(os.MkdirAll(outside, 0o755)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(outside, "secret.txt"), []byte("outside content"), 0o644)).To(gomega.Succeed())

	cfg := config.Asset{
		AppsPath:   filepath.Join(base, "apps"),
		CorePath:   filepath.Join(base, "core"),
		ThemesPath: filepath.Join(base, "themes"),
	}
	for _, dir := range []string{cfg.AppsPath, cfg.CorePath, cfg.ThemesPath} {
		g.Expect(os.MkdirAll(dir, 0o755)).To(gomega.Succeed())
		g.Expect(os.WriteFile(filepath.Join(dir, "file.txt"), []byte("inside content"), 0o644)).To(gomega.Succeed())
		g.Expect(os.Symlink("../outside/secret.txt", filepath.Join(dir, "leak.txt"))).To(gomega.Succeed())
	}
	return cfg, outside
}

func TestNewAssetFSs(t *testing.T) {
	cfg, outside := newAssetDirs(t)
	assetFSs, err := newAssetFSs(cfg, log.NopLogger())
	gomega.NewWithT(t).Expect(err).ToNot(gomega.HaveOccurred())

	for name, fSys := range map[string]*fsx.FallbackFS{
		"apps":   assetFSs.apps,
		"core":   assetFSs.core,
		"themes": assetFSs.themes,
	} {
		t.Run(name+" serves files inside", func(t *testing.T) {
			g := gomega.NewWithT(t)
			b, err := fs.ReadFile(fSys.IOFS(), "file.txt")
			g.Expect(err).ToNot(gomega.HaveOccurred())
			g.Expect(string(b)).To(gomega.Equal("inside content"))
		})

		t.Run(name+" doesn't follow a symlink out of the directory", func(t *testing.T) {
			g := gomega.NewWithT(t)
			b, err := fs.ReadFile(fSys.IOFS(), "leak.txt")
			g.Expect(err).To(gomega.MatchError(fs.ErrNotExist))
			g.Expect(string(b)).ToNot(gomega.ContainSubstring("outside content"))
		})
	}

	t.Run("apps is read-only", func(t *testing.T) {
		g := gomega.NewWithT(t)
		g.Expect(afero.WriteFile(assetFSs.apps, "new.txt", []byte("new"), 0o644)).ToNot(gomega.Succeed())
		_, err := os.Lstat(filepath.Join(cfg.AppsPath, "new.txt"))
		g.Expect(err).To(gomega.MatchError(fs.ErrNotExist))
	})

	t.Run("themes is writable, but not through a symlink out of it", func(t *testing.T) {
		g := gomega.NewWithT(t)
		g.Expect(afero.WriteFile(assetFSs.themes, "new.txt", []byte("new"), 0o644)).To(gomega.Succeed())
		b, err := fs.ReadFile(os.DirFS(cfg.ThemesPath), "new.txt")
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(string(b)).To(gomega.Equal("new"))

		g.Expect(afero.WriteFile(assetFSs.themes, "leak.txt", []byte("new"), 0o644)).ToNot(gomega.Succeed())
		b, err = fs.ReadFile(os.DirFS(outside), "secret.txt")
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(string(b)).To(gomega.Equal("outside content"))
	})
}

func TestNewAssetFSs_UnusableDirectory(t *testing.T) {
	for _, tt := range []struct {
		name string
		set  func(cfg *config.Asset, path string)
	}{
		{name: "apps", set: func(cfg *config.Asset, path string) { cfg.AppsPath = path }},
		{name: "core", set: func(cfg *config.Asset, path string) { cfg.CorePath = path }},
		{name: "themes", set: func(cfg *config.Asset, path string) { cfg.ThemesPath = path }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)
			cfg, _ := newAssetDirs(t)
			file := filepath.Join(t.TempDir(), "file")
			g.Expect(os.WriteFile(file, []byte("x"), 0o644)).To(gomega.Succeed())
			tt.set(&cfg, file)

			_, err := newAssetFSs(cfg, log.NopLogger())
			g.Expect(err).To(gomega.HaveOccurred())
			g.Expect(err.Error()).To(gomega.ContainSubstring(tt.name + " asset directory"))
			g.Expect(err.Error()).To(gomega.ContainSubstring(file))
		})
	}
}
