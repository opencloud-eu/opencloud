package assets

import (
	"bytes"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"path/filepath"

	"golang.org/x/net/html"
)

type fileServer struct {
	fsys http.FileSystem
}

// FileServer defines the http handler for the embedded files
func FileServer(fsys fs.FS) http.Handler {
	return &fileServer{http.FS(fsys)}
}

func (f *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	file, info, ok := f.open(path.Clean(path.Join("/", r.URL.Path)))
	if !ok {
		// paths that aren't files are routes of the web app, which index.html handles.
		// not every fs contains an index.html: without one, the path is not found.
		file, info, ok = f.open("/index.html")
		if !ok {
			http.NotFound(w, r)
			return
		}
	}
	defer file.Close()

	buf := new(bytes.Buffer)

	switch info.Name() {
	case "index.html", "oidc-callback.html", "oidc-silent-redirect.html":
		w.Header().Del("Expires")
		w.Header().Set("Cache-Control", "no-cache")
		if err := withBase(buf, file, "/"); err != nil {
			http.NotFound(w, r)
			return
		}
	default:
		if _, err := buf.ReadFrom(file); err != nil {
			http.NotFound(w, r)
			return
		}
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(info.Name())))
	_, _ = w.Write(buf.Bytes())
}

// open returns the file with the given name if it exists and isn't a directory.
func (f *fileServer) open(name string) (http.File, fs.FileInfo, bool) {
	file, err := f.fsys.Open(name)
	if err != nil {
		return nil, nil, false
	}

	info, err := file.Stat()
	if err != nil || info.IsDir() {
		_ = file.Close()
		return nil, nil, false
	}

	return file, info, true
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
