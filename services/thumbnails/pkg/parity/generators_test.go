package parity

import (
	"bytes"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/services/thumbnails/internal/imagortest"
	"github.com/opencloud-eu/opencloud/services/thumbnails/pkg/config/defaults"
	svc "github.com/opencloud-eu/opencloud/services/thumbnails/pkg/service/http/v0"
)

var generatorNames = generatorList()

func generatorList() []string {
	names := []string{"builtin"}
	for _, v := range imagortest.Variants() {
		names = append(names, v.Name)
	}

	return names
}

// unavailable carries the skip reason when there is nothing to talk to.
type testGenerator struct {
	name        string
	baseURL     string
	unavailable string

	// overrideKeys are the keys this generator answers to in a case's
	// generatorOverrides, most specific first.
	overrideKeys []string
}

// call this from a BeforeAll, the cleanup it defers has to outlive the specs.
func newGenerators() []testGenerator {
	GinkgoHelper()

	generators := []testGenerator{newBuiltin()}
	for _, v := range imagortest.Variants() {
		generators = append(generators, newImagor(v))
	}

	return generators
}

func newBuiltin() testGenerator {
	GinkgoHelper()

	cfg := defaults.DefaultConfig()
	defaults.EnsureDefaults(cfg)

	srv := httptest.NewServer(svc.NewService(svc.Logger(log.NopLogger()), svc.Config(cfg)))
	DeferCleanup(srv.Close)

	return testGenerator{name: "builtin", baseURL: srv.URL, overrideKeys: []string{builtinBuild, "builtin"}}
}

// an override keyed "imagor" holds for every variant, they are the same binary
// with different loaders.
func newImagor(v imagortest.Variant) testGenerator {
	name := v.Name

	g := testGenerator{name: name, overrideKeys: []string{v.Key, "imagor"}}
	if url, ok := imagorURLs[name]; ok {
		g.baseURL = url

		return g
	}

	g.unavailable = imagorErrors[name]
	if g.unavailable == "" {
		g.unavailable = "no " + name + " address"
	}

	return g
}

// the parts of a response both sides have to agree on, read from the response
// alone.
type answer struct {
	status int
	width  int
	height int
	format string
	frames int
	alpha  bool

	// the response body, for a case that probes more than the header says
	data []byte

	// part of the answer only when it disagrees with the format the body is in
	contentType string
}

// typeForFormat is the Content-Type a body of that format is served as.
var typeForFormat = map[string]string{
	"png":  "image/png",
	"jpeg": "image/jpeg",
	"gif":  "image/gif",
	"webp": "image/webp",
	"tiff": "image/tiff",
}

func (a answer) String() string {
	if a.status != http.StatusOK {
		return fmt.Sprintf("%d", a.status)
	}
	if a.format == "non-image" {
		return "200 non-image"
	}
	out := fmt.Sprintf("200 %dx%d %s", a.width, a.height, a.format)
	if a.frames > 1 {
		out += fmt.Sprintf(" frames=%d", a.frames)
	}
	if a.alpha {
		out += " alpha"
	}
	if want, known := typeForFormat[a.format]; known && a.contentType != want {
		out += fmt.Sprintf(" content-type=%s", a.contentType)
	}
	return out
}

func (g testGenerator) requestRaw(method, path string, body []byte, contentType string) answer {
	GinkgoHelper()

	req, err := http.NewRequest(method, g.baseURL+path, bytes.NewReader(body))
	Expect(err).ToNot(HaveOccurred())
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	return g.do(req)
}

// partType is the multipart part's Content-Type, which imagor validates.
func (g testGenerator) request(path string, img []byte, filename, partType string) answer {
	GinkgoHelper()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="image"; filename="`+filename+`"`)
	if partType != "" {
		h.Set("Content-Type", partType)
	}
	part, err := writer.CreatePart(h)
	Expect(err).ToNot(HaveOccurred())
	_, err = part.Write(img)
	Expect(err).ToNot(HaveOccurred())
	Expect(writer.Close()).To(Succeed())

	req, err := http.NewRequest(http.MethodPost, g.baseURL+path, &body)
	Expect(err).ToNot(HaveOccurred())
	req.Header.Set("Content-Type", writer.FormDataContentType())

	return g.do(req)
}

func (g testGenerator) do(req *http.Request) answer {
	GinkgoHelper()

	rsp, err := http.DefaultClient.Do(req)
	Expect(err).ToNot(HaveOccurred())
	defer rsp.Body.Close()

	out := answer{status: rsp.StatusCode}
	data, err := io.ReadAll(rsp.Body)
	Expect(err).ToNot(HaveOccurred())
	if rsp.StatusCode != http.StatusOK {
		return out
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		// a 200 that is not an image is an answer too, not a reason to stop
		out.format = "non-image"
		return out
	}
	out.width, out.height, out.format = cfg.Width, cfg.Height, format
	out.frames = frameCount(data, format)
	out.alpha = hasAlpha(data)
	out.data = data
	out.contentType = rsp.Header.Get("Content-Type")

	return out
}
