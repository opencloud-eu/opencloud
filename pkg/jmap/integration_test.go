package jmap

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/require"

	"github.com/gorilla/websocket"
	"github.com/tidwall/pretty"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/brianvoe/gofakeit/v7"
	pw "github.com/sethvargo/go-password/password"

	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	clog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"

	"github.com/go-crypt/crypt/algorithm/shacrypt"
)

const (
	EnableTypes = false

	// Wireshark = "/usr/bin/wireshark"
	Wireshark = ""
)

type User struct {
	name        string
	description string
	email       string
	password    string
}

func userpassword() string {
	password, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	if err != nil {
		panic(err)
	}
	return password
}

var (
	domains = [...]string{"earth.gov", "mars.mil", "opa.org"}
	users   = [...]User{
		{"cdrummer", "Camina Drummer", "camina.drummer@opa.org", userpassword()},
		{"aburton", "Amos Burton", "amos.burton@earth.gov", userpassword()},
		{"jholden", "James Holden", "james.holden@earth.gov", userpassword()},
		{"adawes", "Anderson Dawes", "anderson.dawes@opa.org", userpassword()},
		{"nnagata", "Naomi Nagata", "naomi.nagata@opa.org", userpassword()},
		{"kashford", "Klaes Ashford", "klaes.ashford@opa.org", userpassword()},
		{"fjohnson", "Fred Johnson", "fred.johnson@opa.org", userpassword()},
		{"cavasarala", "Chrisjen Avasarala}", "chrissy@earth.gov", userpassword()},
		{"bdraper", "Roberta Draper", "bobby@mars.mil", userpassword()},
	}
)

const (
	stalwartImage  = "ghcr.io/stalwartlabs/stalwart:v0.15.5-alpine"
	httpPort       = "8080"
	imapsPort      = "993"
	configTemplate = `
authentication.fallback-admin.secret = "secret"
authentication.fallback-admin.user = "mailadmin"
authentication.master.secret = "{{.masterpassword}}"
authentication.master.user = "{{.masterusername}}"
directory.test.bind.auth.method = "default"
directory.test.cache.size = 1048576
directory.test.cache.ttl.negative = "10m"
directory.test.cache.ttl.positive = "1h"
directory.test.store = "rocksdb"
directory.test.type = "internal"
metrics.prometheus.enable = false
server.listener.http.bind = "[::]:{{.httpPort}}"
server.listener.http.protocol = "http"
server.listener.imaptls.bind = "[::]:{{.imapsPort}}"
server.listener.imaptls.protocol = "imap"
server.listener.imaptls.tls.implicit = true
server.hostname = "{{.hostname}}"
server.max-connections = 8192
server.socket.backlog = 1024
server.socket.nodelay = true
server.socket.reuse-addr = true
server.socket.reuse-port = true
storage.blob = "rocksdb"
storage.data = "rocksdb"
storage.directory = "test"
storage.fts = "rocksdb"
storage.lookup = "rocksdb"
store.rocksdb.compression = "lz4"
store.rocksdb.path = "/opt/stalwart/data"
store.rocksdb.type = "rocksdb"
tracer.log.ansi = false
tracer.log.buffered = false
tracer.log.enable = true
tracer.log.level = "trace"
tracer.log.lossy = false
tracer.log.multiline = false
tracer.log.type = "stdout"
sharing.allow-directory-query = {{.dirquery}}
auth.dkim.sign = false
auth.dkim.verify = "disable"
auth.spf.verify.ehlo = "disable"
auth.spf.verify.mail-from = "disable"
auth.arc.verify = "disable"
auth.arc.seal = false
auth.dmarc.verify = "disable"
auth.iprev.verify = "disable"
`
)

func skip(t *testing.T) bool {
	if os.Getenv("CI") == "woodpecker" {
		t.Skip("Skipping tests because CI==woodpecker")
		return true
	}
	if os.Getenv("CI_SYSTEM_NAME") == "woodpecker" {
		t.Skip("Skipping tests because CI_SYSTEM_NAME==woodpecker")
		return true
	}
	if os.Getenv("USE_TESTCONTAINERS") == "false" {
		t.Skip("Skipping tests because USE_TESTCONTAINERS==false")
		return true
	}
	return false
}

type StalwartTest struct {
	t           *testing.T
	ip          string
	imapPort    int
	container   *testcontainers.DockerContainer
	ctx         context.Context
	cancelCtx   context.CancelFunc
	client      *Client
	logger      *clog.Logger
	jmapBaseUrl *url.URL
	sessionUrl  *url.URL

	io.Closer
}

func (s *StalwartTest) Close() error {
	if s.container != nil {
		var c testcontainers.Container = s.container
		testcontainers.CleanupContainer(s.t, c)
	}
	if s.cancelCtx != nil {
		s.cancelCtx()
	}
	return nil
}

func (s *StalwartTest) Session(username string) *Session {
	session, jerr := s.client.FetchSession(s.ctx, s.sessionUrl, username, s.logger)
	require.NoError(s.t, jerr)
	require.NotNil(s.t, session.Capabilities.Mail)
	require.NotNil(s.t, session.Capabilities.Calendars)
	require.NotNil(s.t, session.Capabilities.Contacts)

	// we have to overwrite the hostname in JMAP URL because the container
	// will know its name to be a random Docker container identifier, or
	// "localhost" as we defined the hostname in the Stalwart configuration,
	// and we also need to overwrite the port number as its not mapped
	session.JmapUrl.Host = s.jmapBaseUrl.Host
	session.WebsocketUrl.Host = s.jmapBaseUrl.Host
	var err error
	session.ApiUrl, err = replaceHost(session.ApiUrl, s.jmapBaseUrl.Host)
	require.NoError(s.t, err)
	session.DownloadUrl, err = replaceHost(session.DownloadUrl, s.jmapBaseUrl.Host)
	require.NoError(s.t, err)
	session.UploadUrl, err = replaceHost(session.UploadUrl, s.jmapBaseUrl.Host)
	require.NoError(s.t, err)
	session.EventSourceUrl, err = replaceHost(session.EventSourceUrl, s.jmapBaseUrl.Host)
	require.NoError(s.t, err)

	return &session
}

type stalwartTestLogConsumer struct{}

func (lc *stalwartTestLogConsumer) Accept(l testcontainers.Log) {
	fmt.Print("STALWART: " + string(l.Content))
}

func withDirectoryQueries(allowDirectoryQueries bool) func(map[string]any) {
	return func(m map[string]any) {
		m["dirquery"] = strconv.FormatBool(allowDirectoryQueries)
	}
}

func newStalwartTest(t *testing.T, options ...func(map[string]any)) (*StalwartTest, error) { //NOSONAR
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	var _ context.CancelFunc = cancel // ignore context leak warning: it is passed in the struct and called in Close()

	// A master user name different from "master" does not seem to work as of the current Stalwart version
	//masterUsernameSuffix, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	//require.NoError(err)
	masterUsername := "master" //"master_" + masterUsernameSuffix

	masterPassword, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	if err != nil {
		return nil, err
	}
	masterPasswordHash := ""
	{
		hasher, err := shacrypt.New(shacrypt.WithSHA512(), shacrypt.WithIterations(shacrypt.IterationsDefaultOmitted))
		if err != nil {
			return nil, err
		}

		digest, err := hasher.Hash(masterPassword)
		if err != nil {
			return nil, err
		}
		masterPasswordHash = digest.Encode()
	}

	hostname := "localhost"

	settings := map[string]any{
		"hostname":       hostname,
		"masterusername": masterUsername,
		"masterpassword": masterPasswordHash,
		"httpPort":       httpPort,
		"imapsPort":      imapsPort,
		"dirquery":       "false",
	}
	for _, option := range options {
		option(settings)
	}

	configBuf := bytes.NewBufferString("")
	template.Must(template.New("").Parse(configTemplate)).Execute(configBuf, settings)
	config := configBuf.String()
	configReader := strings.NewReader(config)

	container, err := testcontainers.Run(
		ctx,
		stalwartImage,
		testcontainers.WithLogConsumers(&stalwartTestLogConsumer{}),
		testcontainers.WithExposedPorts(httpPort+"/tcp", imapsPort+"/tcp"),
		testcontainers.WithFiles(testcontainers.ContainerFile{
			Reader:            configReader,
			ContainerFilePath: "/opt/stalwart/etc/config.toml",
			FileMode:          0o700,
		}),
		testcontainers.WithWaitStrategyAndDeadline(
			30*time.Second,
			wait.ForLog(`Network listener started (network.listen-start) listenerId = "imaptls"`),
			wait.ForLog(`Network listener started (network.listen-start) listenerId = "http"`),
		),
	)

	success := false
	defer func() {
		if !success {
			testcontainers.CleanupContainer(t, container)
		}
	}()

	ip, err := container.Host(ctx)
	if err != nil {
		return nil, err
	}

	imapPort, err := container.MappedPort(ctx, "993")
	if err != nil {
		return nil, err
	}

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	loggerImpl := clog.NewLogger(clog.Level("trace"))
	logger := &loggerImpl
	var j Client
	var jmapBaseUrl *url.URL
	var sessionUrl *url.URL
	{
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.ResponseHeaderTimeout = time.Duration(30 * time.Second)
		tr.TLSClientConfig = tlsConfig
		jh := *http.DefaultClient
		jh.Transport = tr

		wsd := &websocket.Dialer{
			TLSClientConfig:  tlsConfig,
			HandshakeTimeout: time.Duration(10) * time.Second,
		}

		jmapPort, err := container.MappedPort(ctx, httpPort)
		if err != nil {
			return nil, err
		}
		jmapBaseUrl = &url.URL{
			Scheme: "http",
			Host:   ip + ":" + jmapPort.Port(),
			Path:   "/",
		}
		sessionUrl = jmapBaseUrl.JoinPath(".well-known", "jmap")

		if Wireshark != "" {
			fmt.Printf("\x1b[45;37;1m Starting Wireshark on port %v \x1b[0m\n", jmapPort)
			attr := os.ProcAttr{
				Dir:   ".",
				Env:   os.Environ(),
				Files: []*os.File{os.Stdin, os.Stdout, os.Stderr},
			}
			cmd := []string{Wireshark, "-pkSl", "-i", "lo", "-f", fmt.Sprintf("port %d", jmapPort.Int()), "-Y", "http||websocket"}
			process, err := os.StartProcess(Wireshark, cmd, &attr)
			require.NoError(t, err)
			err = process.Release()
			require.NoError(t, err)

			time.Sleep(10 * time.Second)
		}

		eventListener := nullHttpJmapApiClientEventListener{}

		auth := NewMasterAuthHttpJmapClientAuthenticator(masterUsername, masterPassword)

		api := NewHttpJmapClient(&jh, auth, eventListener)

		wscf, err := NewHttpWsClientFactory(wsd, auth, logger, eventListener)
		if err != nil {
			return nil, err
		}

		j = NewClient(api, api, api, wscf)
	}

	// provision some things using Stalwart's Management API
	{
		var h http.Client
		{
			tr := http.DefaultTransport.(*http.Transport).Clone()
			tr.ResponseHeaderTimeout = time.Duration(30 * time.Second)
			tr.TLSClientConfig = tlsConfig
			h = *http.DefaultClient
			h.Transport = tr
		}

		apiPort, err := container.MappedPort(ctx, httpPort)
		require.NoError(t, err)

		url := fmt.Sprintf("http://%s:%d/api/principal", ip, apiPort.Int())

		for _, domain := range domains {
			fmt.Printf("Creating domain '%v'\n", domain)
			bb, err := json.Marshal(map[string]any{
				"type":        "domain",
				"name":        domain,
				"description": domain,
			})
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bb))
			require.NoError(t, err)
			req.SetBasicAuth("mailadmin", "secret")
			resp, err := h.Do(req)
			require.NoError(t, err)
			require.Equal(t, "200 OK", resp.Status) //NOSONAR
		}

		for _, user := range users {
			fmt.Printf("Creating individual '%v'\n", user.name)
			bb, err := json.Marshal(map[string]any{
				"type":        "individual",
				"name":        user.name,
				"description": user.description,
				"emails":      user.email,
				"roles":       []string{"user"},
				"secrets":     user.password,
				"quota":       20000000000,
			})
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(bb))
			require.NoError(t, err)
			req.SetBasicAuth("mailadmin", "secret")
			resp, err := h.Do(req)
			require.NoError(t, err)
			require.Equal(t, "200 OK", resp.Status)

			// fetch the user once with the superadmin credentials to "activate" it,
			// it is unclear why that is needed, but without that, we get errors back
			// that we are not allowed to access that resource
			{
				req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
				require.NoError(t, err)
				req.SetBasicAuth("mailadmin", "secret")
				resp, err := h.Do(req)
				require.NoError(t, err)
				require.Equal(t, "200 OK", resp.Status)
			}
		}

		{
			require.NoError(t, err)
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
			require.NoError(t, err)
			req.SetBasicAuth("mailadmin", "secret")
			resp, err := h.Do(req)
			require.NoError(t, err)
			require.Equal(t, "200 OK", resp.Status)
			var list struct {
				Data struct {
					Total int `json:"total"`
					Items []struct {
						Type   string   `json:"type"`
						Id     int      `json:"id"`
						Name   string   `json:"name"`
						Emails []string `json:"emails"`
						Roles  []string `json:"roles"`
					} `json:"items"`
				} `json:"data"`
			}
			bb, err := io.ReadAll(resp.Body)
			require.NoError(t, err)
			defer resp.Body.Close()
			err = json.Unmarshal(bb, &list)
			require.NoError(t, err)
			individuals := []struct {
				Id     int
				Name   string
				Emails []string
				Roles  []string
			}{}
			for _, p := range list.Data.Items {
				if p.Type == "individual" {
					individuals = append(individuals, struct {
						Id     int
						Name   string
						Emails []string
						Roles  []string
					}{p.Id, p.Name, p.Emails, p.Roles})
				}
			}

			require.Equal(t, len(users), len(individuals))
		}

		{
			// check whether we can fetch a session for the provisioned users
			for _, user := range users {
				session, err := j.FetchSession(ctx, sessionUrl, user.name, logger)
				require.NoError(t, err, "failed to retrieve JMAP session for newly created principal '%s'", user.name)
				require.Equal(t, user.name, session.Username)
			}
		}
	}

	success = true
	return &StalwartTest{
		t:           t,
		ip:          ip,
		imapPort:    int(imapPort.Num()),
		container:   container,
		ctx:         ctx,
		cancelCtx:   cancel,
		client:      &j,
		logger:      logger,
		jmapBaseUrl: jmapBaseUrl,
		sessionUrl:  sessionUrl,
	}, nil
}

var urlHostRegex = regexp.MustCompile(`^(https?://)(.+?)/(.+)$`)

func replaceHost(u string, host string) (string, error) {
	if m := urlHostRegex.FindAllStringSubmatch(u, -1); m != nil {
		return fmt.Sprintf("%s%s/%s", m[0][1], host, m[0][3]), nil
	} else {
		return "", fmt.Errorf("'%v' does not match '%v'", u, urlHostRegex)
	}
}

func pickRandomlyFromMap[K comparable, V any](m map[K]V, min int, max int) map[K]V {
	if min < 0 || max < 0 {
		panic("min and max must be >= 0")
	}
	l := len(m)
	if min > l || max > l {
		panic(fmt.Sprintf("min and max must be <= %d", l))
	}
	n := min + rand.Intn(max-min+1)
	if n == l {
		return m
	}
	// let's use a deep copy so we can remove elements as we pick them
	c := make(map[K]V, l)
	maps.Copy(c, m)
	// r will hold the results
	r := make(map[K]V, n)
	for range n {
		pick := rand.Intn(len(c))
		j := 0
		for k, v := range m {
			if j == pick {
				delete(c, k)
				r[k] = v
				break
			}
			j++
		}
	}
	return r
}

var productName = "jmaptest"

type TestJmapClient struct {
	h        *http.Client
	username string
	password string
	session  *Session
	u        *url.URL
	trace    bool
	color    bool
}

func NewTestJmapClient(session *Session, username string, password string, trace bool, color bool) (*TestJmapClient, error) {
	httpTransport := http.DefaultTransport.(*http.Transport).Clone()
	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	httpTransport.TLSClientConfig = tlsConfig
	h := http.DefaultClient
	h.Transport = httpTransport

	u, err := url.Parse(session.ApiUrl)
	if err != nil {
		return nil, err
	}

	return &TestJmapClient{
		h:        h,
		trace:    trace,
		color:    color,
		username: username,
		password: password,
		session:  session,
		u:        u,
	}, nil
}

func (j *TestJmapClient) Close() error {
	return nil
}

type uploadedBlob struct {
	BlobId string `json:"blobId"`
	Size   int    `json:"size"`
	Type   string `json:"type"`
	Sha512 string `json:"sha:512"`
}

func (j *TestJmapClient) uploadBlob(accountId string, data []byte, mimetype string) (uploadedBlob, error) { //NOSONAR
	uploadUrl := strings.ReplaceAll(j.session.UploadUrl, "{accountId}", accountId)
	req, err := http.NewRequest(http.MethodPost, uploadUrl, bytes.NewReader(data))
	if err != nil {
		return uploadedBlob{}, err
	}
	req.Header.Add("Content-Type", mimetype)
	req.SetBasicAuth(j.username, j.password)
	res, err := j.h.Do(req)
	if err != nil {
		return uploadedBlob{}, err
	}
	defer res.Body.Close()
	var response []byte = nil
	if j.trace {
		if b, err := httputil.DumpResponse(res, false); err == nil {
			response, err = io.ReadAll(res.Body)
			if err != nil {
				return uploadedBlob{}, err
			}
			p := pretty.Pretty(response)
			if j.color {
				p = pretty.Color(p, nil)
			}
			log.Printf("<== %s%s\n", b, p)
		}
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return uploadedBlob{}, fmt.Errorf("blob uploading to '%v': status is %s", uploadUrl, res.Status)
	}
	if response == nil {
		response, err = io.ReadAll(res.Body)
		if err != nil {
			return uploadedBlob{}, err
		}
	}

	var result uploadedBlob
	err = json.Unmarshal(response, &result)
	if err != nil {
		return uploadedBlob{}, err
	}

	return result, nil
}

func (j *TestJmapClient) command(body map[string]any) ([]any, error) { //NOSONAR
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, j.u.String(), bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	if j.trace {
		if b, err := httputil.DumpRequestOut(req, false); err == nil {
			p := pretty.Pretty(payload)
			if j.color {
				p = pretty.Color(p, nil)
			}
			log.Printf("==> %s%s\n", b, p)
		}
	}

	req.SetBasicAuth(j.username, j.password)
	resp, err := j.h.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var response []byte = nil
	if j.trace {
		if b, err := httputil.DumpResponse(resp, false); err == nil {
			response, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, err
			}
			p := pretty.Pretty(response)
			if j.color {
				p = pretty.Color(p, nil)
			}
			log.Printf("<== %s%s\n", b, p)
		}
	}
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("JMAP command HTTP response status is %s", resp.Status)
	}
	if response == nil {
		response, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
	}

	r := map[string]any{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return nil, err
	}

	return r["methodResponses"].([]any), nil
}

type Commander[T any] struct {
	j       *TestJmapClient
	closure func([]any) (T, error)
}

func newCommander[T any](j *TestJmapClient, closure func([]any) (T, error)) Commander[T] {
	return Commander[T]{j: j, closure: closure}
}

func (c Commander[T]) command(body map[string]any) (T, error) {
	var zero T
	methodResponses, err := c.j.command(body)
	if err != nil {
		return zero, err
	}
	return c.closure(methodResponses)
}

func (j *TestJmapClient) create(id string, objectType ObjectType, body map[string]any) (string, error) {
	return newCommander(j, func(methodResponses []any) (string, error) {
		z := methodResponses[0].([]any)
		f := z[1].(map[string]any)
		if x, ok := f["created"]; ok {
			created := x.(map[string]any)
			if c, ok := created[id].(map[string]any); ok {
				return c["id"].(string), nil
			} else {
				return "", fmt.Errorf("failed to create %v", objectType)
			}
		} else {
			if ncx, ok := f["notCreated"]; ok {
				nc := ncx.(map[string]any)
				c := nc[id].(map[string]any)
				return "", fmt.Errorf("failed to create %v: %v", objectType, c["description"])
			} else {
				return "", fmt.Errorf("failed to create %v", objectType)
			}
		}
	}).command(body)
}

func (j *TestJmapClient) create1(accountId string, objectType ObjectType, obj map[string]any) (string, error) {
	body := map[string]any{
		"using": structs.Map(objectType.Namespaces, func(n JmapNamespace) string { return string(n) }),
		"methodCalls": []any{
			[]any{
				objectType.Name + "/set",
				map[string]any{
					"accountId": accountId,
					"create": map[string]any{
						"c": obj,
					},
				},
				"0",
			},
		},
	}
	return j.create("c", objectType, body)
}

func (j *TestJmapClient) objectsById(accountId string, objectType ObjectType) (map[string]map[string]any, error) {
	m := map[string]map[string]any{}
	{
		body := map[string]any{
			"using": structs.Map(objectType.Namespaces, func(n JmapNamespace) string { return string(n) }),
			"methodCalls": []any{
				[]any{
					objectType.Name + "/get",
					map[string]any{
						"accountId": accountId,
					},
					"0",
				},
			},
		}
		result, err := newCommander(j, func(methodResponses []any) ([]any, error) {
			z := methodResponses[0].([]any)
			f := z[1].(map[string]any)
			if list, ok := f["list"]; ok {
				return list.([]any), nil
			} else {
				return nil, fmt.Errorf("methodResponse[1] has no 'list' attribute: %v", f)
			}
		}).command(body)
		if err != nil {
			return nil, err
		}
		for _, a := range result {
			obj := a.(map[string]any)
			id := obj["id"].(string)
			m[id] = obj
		}
	}
	return m, nil
}

func createName(person *gofakeit.PersonInfo) (map[string]any, jscontact.Name) {
	o := jscontact.Name{
		Type: jscontact.NameType,
	}
	m := map[string]any{
		"@type": "Name",
	}
	mComps := make([]map[string]string, 2)
	oComps := make([]jscontact.NameComponent, 2)
	mComps[0] = map[string]string{
		"kind":  "given",
		"value": person.FirstName,
	}
	oComps[0] = jscontact.NameComponent{
		Type:  jscontact.NameComponentType,
		Kind:  jscontact.NameComponentKindGiven,
		Value: person.FirstName,
	}
	mComps[1] = map[string]string{
		"kind":  "surname",
		"value": person.LastName,
	}
	oComps[1] = jscontact.NameComponent{
		Type:  jscontact.NameComponentType,
		Kind:  jscontact.NameComponentKindSurname,
		Value: person.LastName,
	}
	m["components"] = mComps
	o.Components = oComps
	m["isOrdered"] = true
	o.IsOrdered = true
	m["defaultSeparator"] = " "
	o.DefaultSeparator = " "
	full := fmt.Sprintf("%s %s", person.FirstName, person.LastName)
	m["full"] = full
	o.Full = full
	return m, o
}

func createNickName(_ *gofakeit.PersonInfo) (map[string]any, jscontact.Nickname) {
	name := gofakeit.PetName()
	contexts := pickRandoms(jscontact.NicknameContextPrivate, jscontact.NicknameContextWork)
	return map[string]any{
			"@type":    "Nickname",
			"name":     name,
			"contexts": toBoolMap(structs.Map(contexts, func(s jscontact.NicknameContext) string { return string(s) })),
		}, jscontact.Nickname{
			Type:     jscontact.NicknameType,
			Name:     name,
			Contexts: orNilMap(toBoolMap(contexts)),
		}
}

func createEmail(person *gofakeit.PersonInfo, pref int) (map[string]any, jscontact.EmailAddress) {
	email := person.Contact.Email
	contexts := pickRandoms1(jscontact.EmailAddressContextWork, jscontact.EmailAddressContextPrivate)
	label := strings.ToLower(person.FirstName)
	return map[string]any{
			"@type":    "EmailAddress",
			"address":  email,
			"contexts": toBoolMap(structs.Map(contexts, func(s jscontact.EmailAddressContext) string { return string(s) })),
			"label":    label,
			"pref":     pref,
		}, jscontact.EmailAddress{
			Type:     jscontact.EmailAddressType,
			Address:  email,
			Contexts: orNilMap(toBoolMap(contexts)),
			Label:    label,
			Pref:     uint(pref),
		}
}

func createSecondaryEmail(email string, pref int) (map[string]any, jscontact.EmailAddress) {
	contexts := pickRandoms(jscontact.EmailAddressContextWork, jscontact.EmailAddressContextPrivate)
	return map[string]any{
			"@type":    "EmailAddress",
			"address":  email,
			"contexts": toBoolMap(structs.Map(contexts, func(s jscontact.EmailAddressContext) string { return string(s) })),
			"pref":     pref,
		}, jscontact.EmailAddress{
			Type:     jscontact.EmailAddressType,
			Address:  email,
			Contexts: orNilMap(toBoolMap(contexts)),
			Pref:     uint(pref),
		}
}

var idFirstLetters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
var idOtherLetters = append(idFirstLetters, []rune("0123456789")...)

func id() string {
	n := 4 + rand.Intn(12-4+1)
	b := make([]rune, n)
	b[0] = idFirstLetters[rand.Intn(len(idFirstLetters))]
	for i := 1; i < n; i++ {
		b[i] = idOtherLetters[rand.Intn(len(idOtherLetters))]
	}
	return string(b)
}

func toHtml(text string) string {
	return "<!DOCTYPE html>\n<html>\n<body>\n" + strings.Join(htmlJoin(paraSplitter.Split(text, -1)), "\n") + "</body>\n</html>"
}

func htmlJoin(parts []string) []string {
	var result []string
	for i := range parts {
		result = append(result, fmt.Sprintf("<p>%v</p>", parts[i]))
	}
	return result
}

var paraSplitter = regexp.MustCompile("[\r\n]+")

var timezones = []string{
	"America/Adak",
	"America/Anchorage",
	"America/Chicago",
	"America/Denver",
	"America/Detroit",
	"America/Indiana/Knox",
	"America/Kentucky/Louisville",
	"America/Los_Angeles",
	"America/New_York",
	"Europe/Brussels",
	"Europe/Berlin",
	"Europe/Paris",
}

// https://www.w3.org/TR/css-color-3/#html4
var basicColors = []string{
	"black",
	"silver",
	"gray",
	"white",
	"maroon",
	"red",
	"purple",
	"fuchsia",
	"green",
	"lime",
	"olive",
	"yellow",
	"navy",
	"blue",
	"teal",
	"aqua",
}

/*
// https://www.w3.org/TR/SVG11/types.html#ColorKeywords
var extendedColors = []string{
	"aliceblue",
	"antiquewhite",
	"aqua",
	"aquamarine",
	"azure",
	"beige",
	"bisque",
	"black",
	"blanchedalmond",
	"blue",
	"blueviolet",
	"brown",
	"burlywood",
	"cadetblue",
	"chartreuse",
	"chocolate",
	"coral",
	"cornflowerblue",
	"cornsilk",
	"crimson",
	"cyan",
	"darkblue",
	"darkcyan",
	"darkgoldenrod",
	"darkgray",
	"darkgreen",
	"darkgrey",
	"darkkhaki",
	"darkmagenta",
	"darkolivegreen",
	"darkorange",
	"darkorchid",
	"darkred",
	"darksalmon",
	"darkseagreen",
	"darkslateblue",
	"darkslategray",
	"darkslategrey",
	"darkturquoise",
	"darkviolet",
	"deeppink",
	"deepskyblue",
	"dimgray",
	"dimgrey",
	"dodgerblue",
	"firebrick",
	"floralwhite",
	"forestgreen",
	"fuchsia",
	"gainsboro",
	"ghostwhite",
	"gold",
	"goldenrod",
	"gray",
	"grey",
	"green",
	"greenyellow",
	"honeydew",
	"hotpink",
	"indianred",
	"indigo",
	"ivory",
	"khaki",
	"lavender",
	"lavenderblush",
	"lawngreen",
	"lemonchiffon",
	"lightblue",
	"lightcoral",
	"lightcyan",
	"lightgoldenrodyellow",
	"lightgray",
	"lightgreen",
	"lightgrey",
	"lightpink",
	"lightsalmon",
	"lightseagreen",
	"lightskyblue",
	"lightslategray",
	"lightslategrey",
	"lightsteelblue",
	"lightyellow",
	"lime",
	"limegreen",
	"linen",
	"magenta",
	"maroon",
	"mediumaquamarine",
	"mediumblue",
	"mediumorchid",
	"mediumpurple",
	"mediumseagreen",
	"mediumslateblue",
	"mediumspringgreen",
	"mediumturquoise",
	"mediumvioletred",
	"midnightblue",
	"mintcream",
	"mistyrose",
	"moccasin",
	"navajowhite",
	"navy",
	"oldlace",
	"olive",
	"olivedrab",
	"orange",
	"orangered",
	"orchid",
	"palegoldenrod",
	"palegreen",
	"paleturquoise",
	"palevioletred",
	"papayawhip",
	"peachpuff",
	"peru",
	"pink",
	"plum",
	"powderblue",
	"purple",
	"red",
	"rosybrown",
	"royalblue",
	"saddlebrown",
	"salmon",
	"sandybrown",
	"seagreen",
	"seashell",
	"sienna",
	"silver",
	"skyblue",
	"slateblue",
	"slategray",
	"slategrey",
	"snow",
	"springgreen",
	"steelblue",
	"tan",
	"teal",
	"thistle",
	"tomato",
	"turquoise",
	"violet",
	"wheat",
	"white",
	"whitesmoke",
	"yellow",
	"yellowgreen",
}
*/

func propmap[T any](enabled bool, min int, max int, container map[string]any, name string, cardProperty *map[string]T, generator func(int, string) (map[string]any, T, error)) error {
	if !enabled {
		return nil
	}
	n := min + rand.Intn(max-min+1)

	m := make(map[string]map[string]any, n)
	o := make(map[string]T, n)
	for i := range n {
		id := id()
		itemForMap, itemForCard, err := generator(i, id)
		if err != nil {
			return err
		}
		if itemForMap != nil {
			m[id] = itemForMap
			o[id] = itemForCard
		}
	}
	if len(m) > 0 {
		container[name] = m
		*cardProperty = o
	}
	return nil
}

func externalImageUri() string {
	return fmt.Sprintf("https://picsum.photos/id/%d/%d/%d", 1+rand.Intn(200), 200, 300)
}

func orNilMap[K comparable, V any](m map[K]V) map[K]V {
	if len(m) < 1 {
		return nil
	} else {
		return m
	}
}

func toBoolMap[K comparable](s []K) map[K]bool {
	m := make(map[K]bool, len(s))
	for _, e := range s {
		m[e] = true
	}
	return m
}

func toBoolMapS[K comparable](s ...K) map[K]bool {
	m := make(map[K]bool, len(s))
	for _, e := range s {
		m[e] = true
	}
	return m
}

func pickRandom[T any](s ...T) T {
	return s[rand.Intn(len(s))]
}

func pickUser() User {
	return users[rand.Intn(len(users))]
}

func pickRandoms[T any](s ...T) []T {
	return pickRandomN[T](rand.Intn(len(s)), s...)
}

func pickRandomN[T any](n int, s ...T) []T {
	if n == 0 {
		return []T{}
	}
	result := make([]T, n)
	o := make([]T, len(s))
	copy(o, s)
	for i := range n {
		p := rand.Intn(len(o))
		result[i] = slices.Delete(o, p, p)[0]
	}
	return result
}

func pickRandoms1[T any](s ...T) []T {
	n := 1 + rand.Intn(len(s)-1)
	result := make([]T, n)
	o := make([]T, len(s))
	copy(o, s)
	for i := range n {
		p := rand.Intn(len(o))
		result[i] = slices.Delete(o, p, p)[0]
	}
	return result
}

func pickLanguage() string {
	return pickRandom("en-US", "en-GB", "en-AU")
}

func pickLocale() string {
	return pickRandom("en", "fr", "de")
}

func allBoxesAreTicked[S any](t *testing.T, s S, exceptions ...string) {
	v := reflect.ValueOf(s)
	typ := v.Type()
	tname := typ.Name()
	for i := range v.NumField() {
		name := typ.Field(i).Name
		if slices.Contains(exceptions, name) {
			log.Printf("%s[🍒] %s\n", tname, name)
			continue
		}
		value := v.Field(i).Bool()
		if value {
			log.Printf("%s[✅] %s\n", tname, name)
		} else {
			log.Printf("%s[❌] %s\n", tname, name)
		}
		require.True(t, value, "should be true: %v", name)
	}
}

func deepEqual[T any](t *testing.T, expected, actual T) {
	diff := ""
	if EnableTypes {
		diff = cmp.Diff(expected, actual)
	} else {
		diff = cmp.Diff(expected, actual, cmp.FilterPath(func(p cmp.Path) bool {
			switch sf := p.Last().(type) {
			case cmp.StructField:
				return sf.String() == ".Type"
			}
			return false
		}, cmp.Ignore()))
	}
	require.Empty(t, diff)
}
