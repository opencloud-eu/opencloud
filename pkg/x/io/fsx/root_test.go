package fsx_test

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/onsi/gomega"
	"github.com/rs/zerolog"
	"github.com/spf13/afero"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/x/io/fsx"
)

// rootFixture is a directory with symlinks of all kinds, next to a directory
// outside of it:
//
//	<base>/outside/secret.txt
//	<base>/outside/dir/secret.txt
//	<base>/root/...
type rootFixture struct {
	base    string
	root    string
	outside string
}

func newRootFixture(t *testing.T) rootFixture {
	t.Helper()
	g := gomega.NewWithT(t)

	// resolve the temp dir itself: on macOS, it is behind a symlink (/var -> /private/var)
	base, err := filepath.EvalSymlinks(t.TempDir())
	g.Expect(err).ToNot(gomega.HaveOccurred())
	f := rootFixture{
		base:    base,
		root:    filepath.Join(base, "root"),
		outside: filepath.Join(base, "outside"),
	}

	write := func(name, content string) {
		g.Expect(os.MkdirAll(filepath.Dir(name), 0o755)).To(gomega.Succeed())
		g.Expect(os.WriteFile(name, []byte(content), 0o644)).To(gomega.Succeed())
	}
	link := func(target, name string) {
		g.Expect(os.MkdirAll(filepath.Dir(filepath.Join(f.root, name)), 0o755)).To(gomega.Succeed())
		g.Expect(os.Symlink(target, filepath.Join(f.root, name))).To(gomega.Succeed())
	}

	write(filepath.Join(f.outside, "secret.txt"), "outside content")
	write(filepath.Join(f.outside, "dir", "secret.txt"), "outside content")

	write(filepath.Join(f.root, "file.txt"), "file content")
	write(filepath.Join(f.root, "sub", "index.html"), "sub index content")
	// a theme mounted from a Kubernetes ConfigMap: file -> ..data/file -> ..<timestamp>/file
	write(filepath.Join(f.root, "theme", "..2026_11_27_00_00_00.000000000", "theme.json"), "theme content")
	link("..2026_11_27_00_00_00.000000000", "theme/..data")
	link("..data/theme.json", "theme/theme.json")

	link("file.txt", "link-inside.txt")
	link("sub", "link-dir-inside")
	link("../outside/secret.txt", "link-relative-outside.txt")
	link(filepath.Join(f.outside, "secret.txt"), "link-absolute-outside.txt")
	link(filepath.Join(f.root, "file.txt"), "link-absolute-inside.txt")
	link("../root/file.txt", "link-outside-and-back.txt")
	link("../outside/dir", "link-dir-outside")
	link("missing.txt", "link-dangling.txt")
	link("../outside/new.txt", "link-dangling-outside.txt")
	link("link-loop-b", "link-loop-a")
	link("link-loop-a", "link-loop-b")

	return f
}

// expectNotExist checks that err reports a missing file in the way afero's
// CopyOnWriteFs (the base of fsx.FallbackFS) recognizes: it compares the error
// in the *fs.PathError with == instead of errors.Is.
func expectNotExist(g *gomega.WithT, err error) {
	g.Expect(err).To(gomega.HaveOccurred())
	g.Expect(errors.Is(err, fs.ErrNotExist)).To(gomega.BeTrue(), "expected a not exist error, got %v", err)
	var pathErr *fs.PathError
	g.Expect(errors.As(err, &pathErr)).To(gomega.BeTrue(), "expected a *fs.PathError, got %T", err)
	g.Expect(pathErr.Err == fs.ErrNotExist || pathErr.Err == syscall.ENOENT).To(gomega.BeTrue(), "expected fs.ErrNotExist or ENOENT in the *fs.PathError, got %v", pathErr.Err)
}

func readAll(fSys afero.Fs, name string) (string, error) {
	f, err := fSys.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	b, err := io.ReadAll(f)
	return string(b), err
}

func TestNewRootFs_Read(t *testing.T) {
	f := newRootFixture(t)
	fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
	gomega.NewWithT(t).Expect(err).ToNot(gomega.HaveOccurred())

	for _, tt := range []struct {
		name      string
		path      string
		content   string // expected content, if the file can be read
		notExist  bool   // expected to be reported as missing
		expectErr error  // any other expected error
	}{
		{name: "regular file", path: "file.txt", content: "file content"},
		{name: "leading slash", path: "/file.txt", content: "file content"},
		{name: "double slashes", path: "//sub//index.html", content: "sub index content"},
		{name: "symlink inside", path: "link-inside.txt", content: "file content"},
		{name: "configmap symlink chain inside", path: "theme/theme.json", content: "theme content"},
		{name: "file in a symlinked directory inside", path: "link-dir-inside/index.html", content: "sub index content"},
		{name: "relative symlink outside", path: "link-relative-outside.txt", notExist: true},
		{name: "absolute symlink outside", path: "link-absolute-outside.txt", notExist: true},
		// os.Root refuses all absolute symlinks, also those that point inside
		{name: "absolute symlink inside", path: "link-absolute-inside.txt", notExist: true},
		{name: "symlink chain through outside and back", path: "link-outside-and-back.txt", notExist: true},
		{name: "file in a symlinked directory outside", path: "link-dir-outside/secret.txt", notExist: true},
		{name: "parent directory", path: "../outside/secret.txt", notExist: true},
		{name: "parent directory after a leading slash", path: "/../outside/secret.txt", notExist: true},
		{name: "missing file", path: "missing.txt", notExist: true},
		{name: "dangling symlink", path: "link-dangling.txt", notExist: true},
		{name: "dangling symlink outside", path: "link-dangling-outside.txt", notExist: true},
		{name: "symlink loop", path: "link-loop-a", expectErr: syscall.ELOOP},
		{name: "name too long", path: strings.Repeat("a", 300), expectErr: syscall.ENAMETOOLONG},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := gomega.NewWithT(t)

			content, err := readAll(fSys, tt.path)
			_, statErr := fSys.Stat(tt.path)
			switch {
			case tt.notExist:
				expectNotExist(g, err)
				expectNotExist(g, statErr)
			case tt.expectErr != nil:
				g.Expect(err).To(gomega.MatchError(tt.expectErr))
				g.Expect(statErr).To(gomega.MatchError(tt.expectErr))
			default:
				g.Expect(err).ToNot(gomega.HaveOccurred())
				g.Expect(statErr).ToNot(gomega.HaveOccurred())
				g.Expect(content).To(gomega.Equal(tt.content))
			}
		})
	}
}

func TestNewRootFs_ReadDir(t *testing.T) {
	g := gomega.NewWithT(t)
	f := newRootFixture(t)
	fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
	g.Expect(err).ToNot(gomega.HaveOccurred())

	entries, err := fs.ReadDir(fSys.IOFS(), "link-dir-inside")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(entries).To(gomega.HaveLen(1))
	g.Expect(entries[0].Name()).To(gomega.Equal("index.html"))

	_, err = fs.ReadDir(fSys.IOFS(), "link-dir-outside")
	expectNotExist(g, err)
}

func TestNewRootFs_RootIsSymlink(t *testing.T) {
	g := gomega.NewWithT(t)
	f := newRootFixture(t)
	mount := filepath.Join(f.base, "mount")
	g.Expect(os.Symlink(f.root, mount)).To(gomega.Succeed())

	fSys, err := fsx.NewRootFs(mount, log.NopLogger())
	g.Expect(err).ToNot(gomega.HaveOccurred())

	content, err := readAll(fSys, "theme/theme.json")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(content).To(gomega.Equal("theme content"))

	_, err = readAll(fSys, "link-relative-outside.txt")
	expectNotExist(g, err)

	// the symlink is followed again on each access, e.g. after a new release was linked
	next := filepath.Join(f.base, "next")
	g.Expect(os.MkdirAll(next, 0o755)).To(gomega.Succeed())
	g.Expect(os.WriteFile(filepath.Join(next, "file.txt"), []byte("next content"), 0o644)).To(gomega.Succeed())
	g.Expect(os.Remove(mount)).To(gomega.Succeed())
	g.Expect(os.Symlink(next, mount)).To(gomega.Succeed())

	content, err = readAll(fSys, "file.txt")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(content).To(gomega.Equal("next content"))
}

func TestNewRootFs_MissingRoot(t *testing.T) {
	g := gomega.NewWithT(t)
	root := filepath.Join(t.TempDir(), "themes")

	// a missing directory is no error: the embedded assets are used instead
	fSys, err := fsx.NewRootFs(root, log.NopLogger())
	g.Expect(err).ToNot(gomega.HaveOccurred())

	_, err = readAll(fSys, "file.txt")
	expectNotExist(g, err)
	_, err = fSys.Stat("file.txt")
	expectNotExist(g, err)

	// writing creates the directory, as the logo upload (afero.WriteReader) did before
	g.Expect(afero.WriteReader(fSys, "_branding/logo.png", strings.NewReader("logo"))).To(gomega.Succeed())
	b, err := fs.ReadFile(os.DirFS(root), "_branding/logo.png")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(string(b)).To(gomega.Equal("logo"))
}

func TestNewRootFs_UnusableRoot(t *testing.T) {
	t.Run("not a directory", func(t *testing.T) {
		g := gomega.NewWithT(t)
		root := filepath.Join(t.TempDir(), "file")
		g.Expect(os.WriteFile(root, []byte("x"), 0o644)).To(gomega.Succeed())

		// os.OpenRoot reports this with an error of its own, not with syscall.ENOTDIR
		_, err := fsx.NewRootFs(root, log.NopLogger())
		g.Expect(err).To(gomega.HaveOccurred())
		g.Expect(err.Error()).To(gomega.ContainSubstring("not a directory"))
		g.Expect(err.Error()).To(gomega.ContainSubstring(root))
	})

	t.Run("not readable", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can open directories without permission")
		}
		g := gomega.NewWithT(t)
		root := filepath.Join(t.TempDir(), "locked")
		g.Expect(os.Mkdir(root, 0o000)).To(gomega.Succeed())
		t.Cleanup(func() { _ = os.Chmod(root, 0o755) })

		_, err := fsx.NewRootFs(root, log.NopLogger())
		g.Expect(err).To(gomega.MatchError(fs.ErrPermission))
		g.Expect(err.Error()).To(gomega.ContainSubstring(root))
	})
}

func TestNewRootFs_Write(t *testing.T) {
	readOutside := func(g *gomega.WithT, f rootFixture, name string) string {
		b, err := fs.ReadFile(os.DirFS(f.outside), name)
		g.Expect(err).ToNot(gomega.HaveOccurred())
		return string(b)
	}
	expectNoOutside := func(g *gomega.WithT, f rootFixture, name string) {
		_, err := os.Lstat(filepath.Join(f.outside, name))
		g.Expect(errors.Is(err, fs.ErrNotExist)).To(gomega.BeTrue(), "%s must not exist outside", name)
	}

	t.Run("regular file", func(t *testing.T) {
		g := gomega.NewWithT(t)
		f := newRootFixture(t)
		fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(afero.WriteReader(fSys, "new/logo.png", strings.NewReader("logo"))).To(gomega.Succeed())
		b, err := fs.ReadFile(os.DirFS(f.root), "new/logo.png")
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(string(b)).To(gomega.Equal("logo"))
	})

	t.Run("through a symlink inside", func(t *testing.T) {
		g := gomega.NewWithT(t)
		f := newRootFixture(t)
		fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(afero.WriteFile(fSys, "link-inside.txt", []byte("changed"), 0o644)).To(gomega.Succeed())
		b, err := fs.ReadFile(os.DirFS(f.root), "file.txt")
		g.Expect(err).ToNot(gomega.HaveOccurred())
		g.Expect(string(b)).To(gomega.Equal("changed"))
	})

	t.Run("through a symlink to a file outside", func(t *testing.T) {
		g := gomega.NewWithT(t)
		f := newRootFixture(t)
		fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(afero.WriteFile(fSys, "link-relative-outside.txt", []byte("changed"), 0o644)).ToNot(gomega.Succeed())
		g.Expect(afero.WriteFile(fSys, "link-absolute-outside.txt", []byte("changed"), 0o644)).ToNot(gomega.Succeed())
		g.Expect(readOutside(g, f, "secret.txt")).To(gomega.Equal("outside content"))
	})

	t.Run("through a dangling symlink to outside", func(t *testing.T) {
		g := gomega.NewWithT(t)
		f := newRootFixture(t)
		fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(afero.WriteFile(fSys, "link-dangling-outside.txt", []byte("new"), 0o644)).ToNot(gomega.Succeed())
		expectNoOutside(g, f, "new.txt")
	})

	t.Run("into a symlinked directory outside", func(t *testing.T) {
		g := gomega.NewWithT(t)
		f := newRootFixture(t)
		fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(afero.WriteFile(fSys, "link-dir-outside/new.txt", []byte("new"), 0o644)).ToNot(gomega.Succeed())
		g.Expect(fSys.MkdirAll("link-dir-outside/newdir", 0o755)).ToNot(gomega.Succeed())
		expectNoOutside(g, f, filepath.Join("dir", "new.txt"))
		expectNoOutside(g, f, filepath.Join("dir", "newdir"))
	})

	t.Run("parent directory", func(t *testing.T) {
		g := gomega.NewWithT(t)
		f := newRootFixture(t)
		fSys, err := fsx.NewRootFs(f.root, log.NopLogger())
		g.Expect(err).ToNot(gomega.HaveOccurred())

		g.Expect(afero.WriteFile(fSys, "../outside/new.txt", []byte("new"), 0o644)).ToNot(gomega.Succeed())
		g.Expect(fSys.MkdirAll("../outside/newdir", 0o755)).ToNot(gomega.Succeed())
		expectNoOutside(g, f, "new.txt")
		expectNoOutside(g, f, "newdir")
	})
}

func TestNewRootFs_FallbackOnEscape(t *testing.T) {
	g := gomega.NewWithT(t)
	f := newRootFixture(t)
	rootFS, err := fsx.NewRootFs(f.root, log.NopLogger())
	g.Expect(err).ToNot(gomega.HaveOccurred())

	embedded := fsx.NewMemMapFs()
	g.Expect(afero.WriteFile(embedded, "link-relative-outside.txt", []byte("embedded content"), 0o644)).To(gomega.Succeed())

	// a symlink out of the directory is treated like a missing file: the embedded file is used
	content, err := readAll(fsx.NewFallbackFS(rootFS, embedded), "link-relative-outside.txt")
	g.Expect(err).ToNot(gomega.HaveOccurred())
	g.Expect(content).To(gomega.Equal("embedded content"))
}

func TestNewRootFs_LogsEscape(t *testing.T) {
	// pkg/log raises the global level (go-micro logger), the logger of the test filters on its own
	previousLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(previousLevel) })

	g := gomega.NewWithT(t)
	f := newRootFixture(t)
	buf := &bytes.Buffer{}
	fSys, err := fsx.NewRootFs(f.root, log.Logger{Logger: zerolog.New(buf).Level(zerolog.DebugLevel)})
	g.Expect(err).ToNot(gomega.HaveOccurred())

	_, _ = readAll(fSys, "missing.txt")
	_, _ = readAll(fSys, "link-inside.txt")
	g.Expect(buf.String()).To(gomega.BeEmpty(), "only escapes are logged")

	_, _ = readAll(fSys, "link-relative-outside.txt")
	g.Expect(buf.String()).To(gomega.ContainSubstring(`"level":"warn"`))
	g.Expect(buf.String()).To(gomega.ContainSubstring(`"path":"link-relative-outside.txt"`))
	// the target outside of the directory is never logged
	g.Expect(buf.String()).ToNot(gomega.ContainSubstring("outside/secret.txt"))
	g.Expect(buf.String()).ToNot(gomega.ContainSubstring(f.outside))
}

func TestNewRootFs_WarnsOncePerPath(t *testing.T) {
	// pkg/log raises the global level (go-micro logger), the logger of the test filters on its own
	previousLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.TraceLevel)
	t.Cleanup(func() { zerolog.SetGlobalLevel(previousLevel) })

	newFs := func(t *testing.T) (*fsx.BaseFS, *bytes.Buffer) {
		f := newRootFixture(t)
		buf := &bytes.Buffer{}
		fSys, err := fsx.NewRootFs(f.root, log.Logger{Logger: zerolog.New(buf).Level(zerolog.DebugLevel)})
		gomega.NewWithT(t).Expect(err).ToNot(gomega.HaveOccurred())
		return fSys, buf
	}
	warnings := func(buf *bytes.Buffer) int { return strings.Count(buf.String(), `"level":"warn"`) }

	t.Run("the same path is warned about once", func(t *testing.T) {
		g := gomega.NewWithT(t)
		fSys, buf := newFs(t)

		for range 3 {
			_, _ = readAll(fSys, "link-relative-outside.txt")
			_, _ = fSys.Stat("link-relative-outside.txt")
		}
		g.Expect(warnings(buf)).To(gomega.Equal(1))

		_, _ = readAll(fSys, "link-absolute-outside.txt")
		g.Expect(warnings(buf)).To(gomega.Equal(2), "another path is warned about too")
	})

	t.Run("the number of warned paths is limited", func(t *testing.T) {
		g := gomega.NewWithT(t)
		fSys, buf := newFs(t)

		// any name below a symlinked directory outside escapes: clients can make up as many as they want
		for i := range fsx.MaxWarnedPaths + 10 {
			_, _ = fSys.Stat(fmt.Sprintf("link-dir-outside/%d.txt", i))
		}
		g.Expect(warnings(buf)).To(gomega.Equal(fsx.MaxWarnedPaths))
		// the rest is still logged, at debug level
		g.Expect(strings.Count(buf.String(), `"level":"debug"`)).To(gomega.Equal(10))
	})
}
