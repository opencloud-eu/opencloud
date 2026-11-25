package assets

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"syscall"

	"golang.org/x/net/html"

	"github.com/opencloud-eu/opencloud/pkg/log"
)

type fileServer struct {
	fsys   http.FileSystem
	logger log.Logger
}

// FileServer defines the http handler for the embedded files
func FileServer(logger log.Logger, fsys fs.FS) http.Handler {
	return &fileServer{fsys: http.FS(fsys), logger: logger}
}

// errNoFile reports that a path can't be served as a file: it doesn't exist,
// it is a directory, or the filesystem can't have a file with this name.
var errNoFile = errors.New("no file")

func (f *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	name := path.Clean(path.Join("/", r.URL.Path))

	served := name
	file, info, err := f.open(served)
	if errors.Is(err, errNoFile) {
		// paths that aren't files are routes of the web app, which index.html handles.
		// not every fs contains an index.html: without one, the path is not found.
		served = "/index.html"
		file, info, err = f.open(served)
		if errors.Is(err, errNoFile) {
			logger := f.logger.SubloggerWithRequestID(r.Context())
			logger.Debug().Str("path", name).Msg("neither the path nor /index.html is a file")
			http.NotFound(w, r)
			return
		}
	}
	if err != nil {
		f.serverError(w, r, name, served, err, "could not open the file")
		return
	}
	defer file.Close()

	buf := new(bytes.Buffer)

	switch info.Name() {
	case "index.html", "oidc-callback.html", "oidc-silent-redirect.html":
		w.Header().Del("Expires")
		w.Header().Set("Cache-Control", "no-cache")
		if err := withBase(buf, file, "/"); err != nil {
			f.serverError(w, r, name, served, err, "could not read the file")
			return
		}
	default:
		if _, err := buf.ReadFrom(file); err != nil {
			f.serverError(w, r, name, served, err, "could not read the file")
			return
		}
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(info.Name())))
	_, _ = w.Write(buf.Bytes())
}

// open returns the file with the given name. It returns errNoFile if there is
// no file with this name, and any other error if the file couldn't be opened.
func (f *fileServer) open(name string) (http.File, fs.FileInfo, error) {
	file, err := f.fsys.Open(name)
	switch {
	case isNoFile(err):
		return nil, nil, errNoFile
	case err != nil:
		return nil, nil, err
	}

	info, err := file.Stat()
	switch {
	case err != nil:
		_ = file.Close()
		return nil, nil, err
	case info.IsDir():
		_ = file.Close()
		return nil, nil, errNoFile
	}

	return file, info, nil
}

// isNoFile reports whether err means that there is no file with the requested
// name, either because it doesn't exist or because the filesystem can't have
// such a name, e.g. one that is too long or contains a NUL byte.
func isNoFile(err error) bool {
	return errors.Is(err, fs.ErrNotExist) ||
		errors.Is(err, fs.ErrInvalid) ||
		errors.Is(err, syscall.ENAMETOOLONG) ||
		errors.Is(err, syscall.EINVAL) // e.g. a NUL byte in the name
}

// serverError logs err with the requested path and the file that failed, which
// is /index.html for a web app route, and answers with a 500 that must not be
// cached, so that a passing error doesn't stick in browsers and proxies.
func (f *fileServer) serverError(w http.ResponseWriter, r *http.Request, name, filename string, err error, msg string) {
	logger := f.logger.SubloggerWithRequestID(r.Context())
	logger.Error().Err(err).Str("path", name).Str("filename", filename).Msg(msg)

	w.Header().Del("Expires")
	w.Header().Set("Cache-Control", "no-store")
	http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
}

func withBase(w io.Writer, r io.Reader, base string) error {
	doc, err := html.Parse(r)
	if err != nil {
		return err
	}
	var parse func(*html.Node)
	parse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "head" {
			n.InsertBefore(&html.Node{
				Type: html.ElementNode,
				Data: "base",
				Attr: []html.Attribute{
					{
						Key: "href",
						Val: base,
					},
				},
			}, n.FirstChild)

			return
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			parse(c)
		}
	}
	parse(doc)

	return html.Render(w, doc)
}
