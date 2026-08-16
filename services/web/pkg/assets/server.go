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
	root string
}

// FileServer defines the http handler for the embedded files. root is the
// subpath OpenCloud is deployed under (e.g. from OC_URL / WEB_HTTP_ROOT),
// templated into the served HTML's <base href> so relative asset/API
// references resolve under the deployment prefix instead of the domain root.
func FileServer(fsys fs.FS, root string) http.Handler {
	return &fileServer{fsys: http.FS(fsys), root: root}
}

func (f *fileServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	uPath := path.Clean(path.Join("/", r.URL.Path))
	r.URL.Path = uPath

	tryIndex := func() {
		r.URL.Path = "/index.html"

		// not every fs contains a file named index.html,
		// therefore, we need to check if the file exists and stop the recursion if it doesn't
		file, err := f.fsys.Open(r.URL.Path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()

		f.ServeHTTP(w, r)
	}

	asset, err := f.fsys.Open(uPath)
	if err != nil {
		tryIndex()
		return
	}
	defer asset.Close()

	s, _ := asset.Stat()
	if s.IsDir() {
		tryIndex()
		return
	}

	w.Header().Set("Content-Type", mime.TypeByExtension(filepath.Ext(s.Name())))

	buf := new(bytes.Buffer)

	switch s.Name() {
	case "index.html", "oidc-callback.html", "oidc-silent-redirect.html":
		w.Header().Del("Expires")
		w.Header().Set("Cache-Control", "no-cache")
		_ = withBase(buf, asset, f.root+"/")
	default:
		_, _ = buf.ReadFrom(asset)
	}

	_, _ = w.Write(buf.Bytes())
}

func withBase(w io.Writer, r io.Reader, base string) error {
	doc, _ := html.Parse(r)
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
