package jmap

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"maps"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/http/httputil"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhillyerd/enmime/v2"
	"github.com/test-go/testify/require"
	"github.com/tidwall/pretty"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/brianvoe/gofakeit/v7"
	pw "github.com/sethvargo/go-password/password"

	"github.com/opencloud-eu/opencloud/pkg/jscalendar"
	"github.com/opencloud-eu/opencloud/pkg/jscontact"
	clog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"

	"github.com/go-crypt/crypt/algorithm/shacrypt"

	"github.com/ProtonMail/go-crypto/openpgp"
)

var (
	domains = [...]string{"earth.gov", "mars.mil", "opa.org", "acme.com"}
	people  = [...]string{
		"Camina Drummer",
		"Amos Burton",
		"James Holden",
		"Anderson Dawes",
		"Naomi Nagata",
		"Klaes Ashford",
		"Fred Johnson",
		"Chrisjen Avasarala",
		"Bobby Draper",
	}
)

const (
	stalwartImage  = "ghcr.io/stalwartlabs/stalwart:v0.14.0-alpine"
	httpPort       = "8080"
	imapsPort      = "993"
	configTemplate = `
authentication.fallback-admin.secret = "$6$4qPYDVhaUHkKcY7s$bB6qhcukb9oFNYRIvaDZgbwxrMa2RvF5dumCjkBFdX19lSNqrgKltf3aPrFMuQQKkZpK2YNuQ83hB1B3NiWzj."
authentication.fallback-admin.user = "mailadmin"
authentication.master.secret = "{{.masterpassword}}"
authentication.master.user = "{{.masterusername}}"
directory.memory.principals.0000.class = "admin"
directory.memory.principals.0000.description = "Superuser"
directory.memory.principals.0000.email.0000 = "admin@example.org"
directory.memory.principals.0000.name = "admin"
directory.memory.principals.0000.secret = "secret"
directory.memory.principals.0001.class = "individual"
directory.memory.principals.0001.description = "{{.description}}"
directory.memory.principals.0001.email.0000 = "{{.email}}"
directory.memory.principals.0001.name = "{{.username}}"
directory.memory.principals.0001.secret = "{{.password}}"
directory.memory.principals.0001.storage.directory = "memory"
directory.memory.type = "memory"
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
storage.directory = "memory"
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
`
)

func htmlJoin(parts []string) []string {
	var result []string
	for i := range parts {
		result = append(result, fmt.Sprintf("<p>%v</p>", parts[i]))
	}
	return result
}

var paraSplitter = regexp.MustCompile("[\r\n]+")
var emailSplitter = regexp.MustCompile("(.+)@(.+)$")

func htmlFormat(body string, msg enmime.MailBuilder) enmime.MailBuilder {
	return msg.HTML([]byte(strings.Join(htmlJoin(paraSplitter.Split(body, -1)), "\n")))
}

func textFormat(body string, msg enmime.MailBuilder) enmime.MailBuilder {
	return msg.Text([]byte(body))
}

func bothFormat(body string, msg enmime.MailBuilder) enmime.MailBuilder {
	msg = htmlFormat(body, msg)
	msg = textFormat(body, msg)
	return msg
}

var formats = []func(string, enmime.MailBuilder) enmime.MailBuilder{
	htmlFormat,
	textFormat,
	bothFormat,
}

type sender struct {
	first  string
	last   string
	from   string
	sender string
}

func (s sender) inject(b enmime.MailBuilder) enmime.MailBuilder {
	return b.From(s.first+" "+s.last, s.from).Header("Sender", s.sender)
}

type senderGenerator struct {
	senders []sender
}

func newSenderGenerator(numSenders int) senderGenerator {
	senders := make([]sender, numSenders)
	for i := range numSenders {
		person := gofakeit.Person()
		senders[i] = sender{
			first:  person.FirstName,
			last:   person.LastName,
			from:   person.Contact.Email,
			sender: person.FirstName + " " + person.LastName + "<" + person.Contact.Email + ">",
		}
	}
	return senderGenerator{
		senders: senders,
	}
}

func (s senderGenerator) nextSender() *sender {
	if len(s.senders) < 1 {
		panic("failed to determine a sender to use")
	} else {
		return &s.senders[rand.Intn(len(s.senders))]
	}
}

func fakeFilename(extension string) string {
	return strings.ReplaceAll(gofakeit.Product().Name, " ", "_") + extension
}

func mailboxId(role string, mailboxes []Mailbox) string {
	for _, m := range mailboxes {
		if m.Role == role {
			return m.Id
		}
	}
	return ""
}

func skip(t *testing.T) bool {
	if os.Getenv("CI") == "woodpecker" {
		t.Skip("Skipping tests because CI==wookpecker")
		return true
	}
	if os.Getenv("CI_SYSTEM_NAME") == "woodpecker" {
		t.Skip("Skipping tests because CI_SYSTEM_NAME==wookpecker")
		return true
	}
	if os.Getenv("USE_TESTCONTAINERS") == "false" {
		t.Skip("Skipping tests because USE_TESTCONTAINERS==false")
		return true
	}
	return false
}

type StalwartTest struct {
	t              *testing.T
	ip             string
	imapPort       int
	container      *testcontainers.DockerContainer
	ctx            context.Context
	cancelCtx      context.CancelFunc
	client         *Client
	session        *Session
	username       string
	password       string
	logger         *clog.Logger
	userPersonName string
	userEmail      string

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

func newStalwartTest(t *testing.T) (*StalwartTest, error) {
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

	usernameSuffix, err := pw.Generate(8, 2, 0, true, true)
	if err != nil {
		return nil, err
	}
	username := "user_" + usernameSuffix

	password, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	if err != nil {
		return nil, err
	}

	hostname := "localhost"

	userPersonName := people[rand.Intn(len(people))]
	var userEmail string
	{
		domain := domains[rand.Intn(len(domains))]
		userEmail = strings.Join(strings.Split(cases.Lower(language.English).String(userPersonName), " "), ".") + "@" + domain
	}

	configBuf := bytes.NewBufferString("")
	template.Must(template.New("").Parse(configTemplate)).Execute(configBuf, map[string]any{
		"hostname":       hostname,
		"password":       password,
		"username":       username,
		"description":    userPersonName,
		"email":          userEmail,
		"masterusername": masterUsername,
		"masterpassword": masterPasswordHash,
		"httpPort":       httpPort,
		"imapsPort":      imapsPort,
	})
	config := configBuf.String()
	configReader := strings.NewReader(config)

	container, err := testcontainers.Run(
		ctx,
		stalwartImage,
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

	loggerImpl := clog.NewLogger()
	logger := &loggerImpl
	var j Client
	var session *Session
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
		jmapBaseUrl := url.URL{
			Scheme: "http",
			Host:   ip + ":" + jmapPort.Port(),
			Path:   "/",
		}

		sessionUrl := jmapBaseUrl.JoinPath(".well-known", "jmap")

		api := NewHttpJmapClient(
			&jh,
			masterUsername,
			masterPassword,
			nullHttpJmapApiClientEventListener{},
		)

		wscf, err := NewHttpWsClientFactory(wsd, masterUsername, masterPassword, logger)
		if err != nil {
			return nil, err
		}

		j = NewClient(api, api, api, wscf)
		s, err := j.FetchSession(sessionUrl, username, logger)
		if err != nil {
			return nil, err
		}
		// we have to overwrite the hostname in JMAP URL because the container
		// will know its name to be a random Docker container identifier, or
		// "localhost" as we defined the hostname in the Stalwart configuration,
		// and we also need to overwrite the port number as its not mapped
		s.JmapUrl.Host = jmapBaseUrl.Host
		s.WebsocketUrl.Host = jmapBaseUrl.Host
		//s.JmapEndpoint = jmapBaseUrl.Host
		s.ApiUrl, err = replaceHost(s.ApiUrl, jmapBaseUrl.Host)
		require.NoError(t, err)
		s.DownloadUrl, err = replaceHost(s.DownloadUrl, jmapBaseUrl.Host)
		require.NoError(t, err)
		s.UploadUrl, err = replaceHost(s.UploadUrl, jmapBaseUrl.Host)
		require.NoError(t, err)
		s.EventSourceUrl, err = replaceHost(s.EventSourceUrl, jmapBaseUrl.Host)
		require.NoError(t, err)
		session = &s
	}

	require.NotNil(t, session.Capabilities.Mail)
	require.NotNil(t, session.Capabilities.Calendars)
	require.NotNil(t, session.Capabilities.Contacts)

	success = true
	return &StalwartTest{
		t:              t,
		ip:             ip,
		imapPort:       imapPort.Int(),
		container:      container,
		ctx:            ctx,
		cancelCtx:      cancel,
		client:         &j,
		session:        session,
		username:       username,
		password:       password,
		logger:         logger,
		userPersonName: userPersonName,
		userEmail:      userEmail,
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

type filledAttachment struct {
	name        string
	size        int
	mimeType    string
	disposition string
}

type filledMail struct {
	uid         int
	attachments []filledAttachment
	subject     string
	testId      string
	messageId   string
	keywords    []string
}

var allKeywords = map[string]imap.Flag{
	JmapKeywordAnswered:  imap.FlagAnswered,
	JmapKeywordDraft:     imap.FlagDraft,
	JmapKeywordFlagged:   imap.FlagFlagged,
	JmapKeywordForwarded: imap.FlagForwarded,
	JmapKeywordJunk:      imap.FlagJunk,
	JmapKeywordMdnSent:   imap.FlagMDNSent,
	JmapKeywordNotJunk:   imap.FlagNotJunk,
	JmapKeywordPhishing:  imap.FlagPhishing,
	JmapKeywordSeen:      imap.FlagSeen,
}

/*
func pickOneRandomlyFromMap[K comparable, V any](m map[K]V) (K, V) {
	l := rand.Intn(len(m))
	i := 0
	for k, v := range m {
		if i == l {
			return k, v
		}
		i++
	}
	panic("map is empty")
}
*/

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

func (s *StalwartTest) fillEmailsWithImap(folder string, count int) ([]filledMail, int, error) {
	to := fmt.Sprintf("%s <%s>", s.userPersonName, s.userEmail)
	ccEvery := 2
	bccEvery := 3
	attachmentEvery := 2
	senders := max(count/4, 1)
	maxThreadSize := 6
	maxAttachments := 4

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	c, err := imapclient.DialTLS(net.JoinHostPort(s.ip, strconv.Itoa(s.imapPort)), &imapclient.Options{TLSConfig: tlsConfig})
	if err != nil {
		return nil, 0, err
	}

	defer func(imap *imapclient.Client) {
		err := imap.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(c)

	if err = c.Login(s.username, s.password).Wait(); err != nil {
		return nil, 0, err
	}

	if _, err = c.Select(folder, &imap.SelectOptions{ReadOnly: false}).Wait(); err != nil {
		return nil, 0, err
	}

	if ids, err := c.Search(&imap.SearchCriteria{}, nil).Wait(); err != nil {
		return nil, 0, err
	} else {
		if len(ids.AllSeqNums()) > 0 {
			storeFlags := imap.StoreFlags{
				Op:     imap.StoreFlagsAdd,
				Flags:  []imap.Flag{imap.FlagDeleted},
				Silent: true,
			}
			if err = c.Store(ids.All, &storeFlags, nil).Close(); err != nil {
				return nil, 0, err
			}
			if err = c.Expunge().Close(); err != nil {
				return nil, 0, err
			}
			log.Printf("🗑️ deleted %d messages in %s", len(ids.AllSeqNums()), folder)
		} else {
			log.Printf("ℹ️ did not delete any messages, %s is empty", folder)
		}
	}

	address, err := mail.ParseAddress(to)
	if err != nil {
		return nil, 0, err
	}
	displayName := address.Name

	addressParts := emailSplitter.FindAllStringSubmatch(address.Address, 3)
	if len(addressParts) != 1 {
		return nil, 0, fmt.Errorf("address does not have one part: '%v' -> %v", address.Address, addressParts)
	}
	if len(addressParts[0]) != 3 {
		return nil, 0, fmt.Errorf("first address part does not have a size of 3: '%v'", addressParts[0])
	}

	domain := addressParts[0][2]

	toName := displayName
	toAddress := fmt.Sprintf("%s@%s", s.username, domain)
	ccName1 := "Team Lead"
	ccAddress1 := fmt.Sprintf("lead@%s", domain)
	ccName2 := "Coworker"
	ccAddress2 := fmt.Sprintf("coworker@%s", domain)
	bccName := "HR"
	bccAddress := fmt.Sprintf("corporate@%s", domain)

	sg := newSenderGenerator(senders)
	thread := 0
	mails := make([]filledMail, count)
	for i := 0; i < count; thread++ {
		threadMessageId := fmt.Sprintf("%d.%d@%s", time.Now().Unix(), 1000000+rand.Intn(8999999), domain)
		threadSubject := strings.Trim(gofakeit.SentenceSimple(), ".") // remove the . at the end, looks weird
		threadSize := 1 + rand.Intn(maxThreadSize)
		lastMessageId := ""
		lastSubject := ""
		for t := 0; i < count && t < threadSize; t++ {
			sender := sg.nextSender()

			format := formats[i%len(formats)]
			text := gofakeit.Paragraph(2+rand.Intn(9), 1+rand.Intn(4), 1+rand.Intn(32), "\n")

			msg := sender.inject(enmime.Builder().To(toName, toAddress))

			messageId := ""
			if lastMessageId == "" {
				// start a new thread
				msg = msg.Header("Message-ID", threadMessageId).Subject(threadSubject)
				lastMessageId = threadMessageId
				lastSubject = threadSubject
				messageId = threadMessageId
			} else {
				// we're continuing a thread
				messageId = fmt.Sprintf("%d.%d@%s", time.Now().Unix(), 1000000+rand.Intn(8999999), domain)
				inReplyTo := ""
				subject := ""
				switch rand.Intn(2) {
				case 0:
					// reply to first post in thread
					subject = "Re: " + threadSubject
					inReplyTo = threadMessageId
				default:
					// reply to last addition to thread
					subject = "Re: " + lastSubject
					inReplyTo = lastMessageId
				}
				msg = msg.Header("Message-ID", messageId).Header("In-Reply-To", inReplyTo).Subject(subject)
				lastMessageId = messageId
				lastSubject = subject
			}

			if i%ccEvery == 0 {
				msg = msg.CCAddrs([]mail.Address{{Name: ccName1, Address: ccAddress1}, {Name: ccName2, Address: ccAddress2}})
			}
			if i%bccEvery == 0 {
				msg = msg.BCC(bccName, bccAddress)
			}

			numAttachments := 0
			attachments := []filledAttachment{}
			if maxAttachments > 0 && i%attachmentEvery == 0 {
				numAttachments = rand.Intn(maxAttachments)
				for a := range numAttachments {
					switch rand.Intn(2) {
					case 0:
						filename := fakeFilename(".txt")
						attachment := gofakeit.Paragraph(2+rand.Intn(4), 1+rand.Intn(4), 1+rand.Intn(32), "\n")
						data := []byte(attachment)
						msg = msg.AddAttachment(data, "text/plain", filename)
						attachments = append(attachments, filledAttachment{
							name:        filename,
							size:        len(data),
							mimeType:    "text/plain",
							disposition: "attachment",
						})
					default:
						filename := ""
						mimetype := ""
						var image []byte = nil
						switch rand.Intn(2) {
						case 0:
							filename = fakeFilename(".png")
							mimetype = "image/png"
							image = gofakeit.ImagePng(512, 512)
						default:
							filename = fakeFilename(".jpg")
							mimetype = "image/jpeg"
							image = gofakeit.ImageJpeg(400, 200)
						}
						disposition := ""
						switch rand.Intn(2) {
						case 0:
							msg = msg.AddAttachment(image, mimetype, filename)
							disposition = "attachment"
						default:
							msg = msg.AddInline(image, mimetype, filename, "c"+strconv.Itoa(a))
							disposition = "inline"
						}
						attachments = append(attachments, filledAttachment{
							name:        filename,
							size:        len(image),
							mimeType:    mimetype,
							disposition: disposition,
						})
					}
				}
			}

			msg = format(text, msg)

			flags := []imap.Flag{}
			keywords := pickRandomlyFromMap(allKeywords, 0, len(allKeywords))
			for _, f := range keywords {
				flags = append(flags, f)
			}

			buf := new(bytes.Buffer)
			part, _ := msg.Build()
			part.Encode(buf)
			mail := buf.String()

			var options *imap.AppendOptions = nil
			if len(flags) > 0 {
				options = &imap.AppendOptions{Flags: flags}
			}

			size := int64(len(mail))
			appendCmd := c.Append(folder, size, options)
			if _, err := appendCmd.Write([]byte(mail)); err != nil {
				return nil, 0, err
			}
			if err := appendCmd.Close(); err != nil {
				return nil, 0, err
			}
			if appendData, err := appendCmd.Wait(); err != nil {
				return nil, 0, err
			} else {
				attachmentStr := ""
				if numAttachments > 0 {
					attachmentStr = " " + strings.Repeat("📎", numAttachments)
				}
				log.Printf("➕ appended %v/%v [in thread %v] uid=%v%s", i+1, count, thread+1, appendData.UID, attachmentStr)

				mails[i] = filledMail{
					uid:         int(appendData.UID),
					attachments: attachments,
					subject:     msg.GetSubject(),
					messageId:   messageId,
					keywords:    slices.Collect(maps.Keys(keywords)),
				}
			}

			i++
		}
	}

	listCmd := c.List("", "%", &imap.ListOptions{
		ReturnStatus: &imap.StatusOptions{
			NumMessages: true,
			NumUnseen:   true,
		},
	})
	countMap := map[string]int{}
	for {
		mbox := listCmd.Next()
		if mbox == nil {
			break
		}
		countMap[mbox.Mailbox] = int(*mbox.Status.NumMessages)
	}

	inboxCount := -1
	for f, i := range countMap {
		if strings.Compare(strings.ToLower(f), strings.ToLower(folder)) == 0 {
			inboxCount = i
			break
		}
	}
	if err = listCmd.Close(); err != nil {
		return nil, 0, err
	}
	if inboxCount == -1 {
		return nil, 0, fmt.Errorf("failed to find folder '%v' via IMAP", folder)
	}
	if count != inboxCount {
		return nil, 0, fmt.Errorf("wrong number of emails in the inbox after filling, expecting %v, has %v", count, inboxCount)
	}

	return mails, thread, nil
}

var productName = "jmaptest"

func (s *StalwartTest) fillContacts(
	t *testing.T,
	count uint,
) (string, string, map[string]jscontact.ContactCard, error) {
	require := require.New(t)
	c, err := NewTestJmapClient(s.session, s.username, s.password, true, true)
	require.NoError(err)
	defer c.Close()

	printer := func(s string) { log.Println(s) }

	accountId := c.session.PrimaryAccounts.Contacts
	require.NotEmpty(accountId, "no primary account for contacts in session")

	addressbookId := ""
	{
		addressBooksById, err := testObjectsById(c, accountId, AddressBookType, JmapContacts)
		require.NoError(err)

		for id, addressbook := range addressBooksById {
			if isDefault, ok := addressbook["isDefault"]; ok {
				if isDefault.(bool) {
					addressbookId = id
					break
				}
			}
		}
	}
	require.NotEmpty(addressbookId)

	filled := map[string]jscontact.ContactCard{}
	for i := range count {
		person := gofakeit.Person()
		nameMap, nameObj := createName(person)
		contact := map[string]any{
			"@type":          "Card",
			"version":        "1.0",
			"addressBookIds": toBoolMap([]string{addressbookId}),
			"prodId":         productName,
			"language":       pickLanguage(),
			"kind":           "individual",
			"name":           nameMap,
		}
		card := jscontact.ContactCard{
			//Type:           jscontact.ContactCardType,
			Version:        "1.0",
			AddressBookIds: toBoolMap([]string{addressbookId}),
			ProdId:         productName,
			Language:       contact["language"].(string),
			Kind:           jscontact.ContactCardKindIndividual,
			Name:           &nameObj,
		}

		if rand.Intn(3) < 1 {
			nicknameMap, nicknameObj := createNickName(person)
			id := id()
			contact["nicknames"] = map[string]map[string]any{id: nicknameMap}
			card.Nicknames = map[string]jscontact.Nickname{id: nicknameObj}
		}

		{
			emailMaps := map[string]map[string]any{}
			emailObjs := map[string]jscontact.EmailAddress{}
			emailId := id()
			emailMap, emailObj := createEmail(person, 10)
			emailMaps[emailId] = emailMap
			emailObjs[emailId] = emailObj

			for i := range rand.Intn(3) {
				id := id()
				m, o := createSecondaryEmail(gofakeit.Email(), i*100)
				emailMaps[id] = m
				emailObjs[id] = o
			}
			if len(emailMaps) > 0 {
				contact["emails"] = emailMaps
				card.Emails = emailObjs
			}
		}
		if err := propmap(contact, "phones", &card.Phones, 0, 2, func(i int, id string) (map[string]any, jscontact.Phone, error) {
			num := person.Contact.Phone
			if i > 0 {
				num = gofakeit.Phone()
			}
			var mapFeatures map[string]bool = nil
			var objFeatures map[jscontact.PhoneFeature]bool = nil
			if rand.Intn(3) < 2 {
				mapFeatures = toBoolMapS("mobile", "voice", "video", "text")
				objFeatures = toBoolMapS(jscontact.PhoneFeatureMobile, jscontact.PhoneFeatureVoice, jscontact.PhoneFeatureVideo, jscontact.PhoneFeatureText)
			} else {
				mapFeatures = toBoolMapS("voice", "main-number")
				objFeatures = toBoolMapS(jscontact.PhoneFeatureVoice, jscontact.PhoneFeatureMainNumber)
			}
			mapContexts := map[string]bool{}
			objContexts := map[jscontact.PhoneContext]bool{}
			mapContexts["work"] = true
			objContexts[jscontact.PhoneContextWork] = true
			if rand.Intn(2) < 1 {
				mapContexts["private"] = true
				objContexts[jscontact.PhoneContextPrivate] = true
			}
			tel := "tel:" + "+1" + num
			return map[string]any{
					"@type":    "Phone",
					"number":   tel,
					"features": mapFeatures,
					"contexts": mapContexts,
				}, jscontact.Phone{
					//Type:     jscontact.PhoneType,
					Number:   tel,
					Features: objFeatures,
					Contexts: objContexts,
				}, nil
		}); err != nil {
			return "", "", nil, err
		}
		if err := propmap(contact, "addresses", &card.Addresses, 1, 2, func(i int, id string) (map[string]any, jscontact.Address, error) {
			var source *gofakeit.AddressInfo
			if i == 0 {
				source = person.Address
			} else {
				source = gofakeit.Address()
			}
			mComps := []map[string]string{}
			oComps := []jscontact.AddressComponent{}
			m := streetNumberRegex.FindAllStringSubmatch(source.Street, -1)
			if m != nil {
				mComps = append(mComps, map[string]string{"kind": "name", "value": m[0][2]})
				mComps = append(mComps, map[string]string{"kind": "number", "value": m[0][1]})
				oComps = append(oComps, jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindName, Value: m[0][2]})
				oComps = append(oComps, jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindNumber, Value: m[0][1]})
			} else {
				mComps = append(mComps, map[string]string{"kind": "name", "value": source.Street})
				oComps = append(oComps, jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindName, Value: source.Street})
			}
			mComps = append(mComps,
				map[string]string{"kind": "locality", "value": source.City},
				map[string]string{"kind": "country", "value": source.Country},
				map[string]string{"kind": "region", "value": source.State},
				map[string]string{"kind": "postcode", "value": source.Zip},
			)
			oComps = append(oComps,
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindLocality, Value: source.City},
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindCountry, Value: source.Country},
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindRegion, Value: source.State},
				jscontact.AddressComponent{ /*Type: jscontact.AddressComponentType,*/ Kind: jscontact.AddressComponentKindPostcode, Value: source.Zip},
			)
			tz := pickRandom(timezones...)
			return map[string]any{
					"@type":            "Address",
					"components":       mComps,
					"defaultSeparator": ", ",
					"isOrdered":        true,
					"timeZone":         tz,
				}, jscontact.Address{
					//Type:             jscontact.AddressType,
					Components:       oComps,
					DefaultSeparator: ", ",
					IsOrdered:        true,
					TimeZone:         tz,
				}, nil
		}); err != nil {
			return "", "", nil, err
		}
		if err := propmap(contact, "onlineServices", &card.OnlineServices, 0, 2, func(i int, id string) (map[string]any, jscontact.OnlineService, error) {
			switch rand.Intn(3) {
			case 0:
				return map[string]any{
						"@type":   "OnlineService",
						"service": "Mastodon",
						"user":    "@" + person.Contact.Email,
						"uri":     "https://mastodon.example.com/@" + strings.ToLower(person.FirstName),
					}, jscontact.OnlineService{
						//Type:    jscontact.OnlineServiceType,
						Service: "Mastodon",
						User:    "@" + person.Contact.Email,
						Uri:     "https://mastodon.example.com/@" + strings.ToLower(person.FirstName),
					}, nil
			case 1:
				return map[string]any{
						"@type": "OnlineService",
						"uri":   "xmpp:" + person.Contact.Email,
					}, jscontact.OnlineService{
						//Type: jscontact.OnlineServiceType,
						Uri: "xmpp:" + person.Contact.Email,
					}, nil
			default:
				return map[string]any{
						"@type":   "OnlineService",
						"service": "Discord",
						"user":    person.Contact.Email,
						"uri":     "https://discord.example.com/user/" + person.Contact.Email,
					}, jscontact.OnlineService{
						//Type: jscontact.OnlineServiceType,
						Service: "Discord",
						User:    person.Contact.Email,
						Uri:     "https://discord.example.com/user/" + person.Contact.Email,
					}, nil
			}
		}); err != nil {
			return "", "", nil, err
		}

		if err := propmap(contact, "preferredLanguages", &card.PreferredLanguages, 0, 2, func(i int, id string) (map[string]any, jscontact.LanguagePref, error) {
			lang := pickRandom("en", "fr", "de", "es", "it")
			contexts := pickRandoms1("work", "private")
			return map[string]any{
					"@type":    "LanguagePref",
					"language": lang,
					"contexts": toBoolMap(contexts),
					"pref":     i + 1,
				}, jscontact.LanguagePref{
					// Type:     jscontact.LanguagePrefType,
					Language: lang,
					Contexts: toBoolMap(structs.Map(contexts, func(s string) jscontact.LanguagePrefContext { return jscontact.LanguagePrefContext(s) })),
					Pref:     uint(i + 1),
				}, nil
		}); err != nil {
			return "", "", nil, err
		}

		{
			organizationMaps := map[string]map[string]any{}
			organizationObjs := map[string]jscontact.Organization{}
			titleMaps := map[string]map[string]any{}
			titleObjs := map[string]jscontact.Title{}
			for range rand.Intn(2) {
				orgId := id()
				titleId := id()
				organizationMaps[orgId] = map[string]any{
					"@type":    "Organization",
					"name":     person.Job.Company,
					"contexts": toBoolMapS("work"),
				}
				organizationObjs[orgId] = jscontact.Organization{
					// Type:     jscontact.OrganizationType,
					Name:     person.Job.Company,
					Contexts: toBoolMapS(jscontact.OrganizationContextWork),
				}
				titleMaps[titleId] = map[string]any{
					"@type":          "Title",
					"kind":           "title",
					"name":           person.Job.Title,
					"organizationId": orgId,
				}
				titleObjs[titleId] = jscontact.Title{
					// Type:           jscontact.TitleType,
					Kind:           jscontact.TitleKindTitle,
					Name:           person.Job.Title,
					OrganizationId: orgId,
				}
			}
			if len(organizationMaps) > 0 {
				contact["organizations"] = organizationMaps
				contact["titles"] = titleMaps
				card.Organizations = organizationObjs
				card.Titles = titleObjs
			}
		}

		if err := propmap(contact, "cryptoKeys", &card.CryptoKeys, 0, 1, func(i int, id string) (map[string]any, jscontact.CryptoKey, error) {
			entity, err := openpgp.NewEntity(person.FirstName+" "+person.LastName, "test", person.Contact.Email, nil)
			if err != nil {
				return nil, jscontact.CryptoKey{}, err
			}
			var b bytes.Buffer
			err = entity.PrimaryKey.Serialize(&b)
			if err != nil {
				return nil, jscontact.CryptoKey{}, err
			}
			encoded := base64.RawStdEncoding.EncodeToString(b.Bytes())
			return map[string]any{
					"@type": "CryptoKey",
					"uri":   "data:application/pgp-keys;base64," + encoded,
				}, jscontact.CryptoKey{
					// Type: jscontact.CryptoKeyType,
					Uri: "data:application/pgp-keys;base64," + encoded,
				}, nil
		}); err != nil {
			return "", "", nil, err
		}

		if err := propmap(contact, "media", &card.Media, 0, 1, func(i int, id string) (map[string]any, jscontact.Media, error) {
			if rand.Intn(2) < 1 {
				img := gofakeit.ImageJpeg(128, 128)
				blob, err := c.uploadBlob(accountId, img, "image/jpeg")
				if err != nil {
					return nil, jscontact.Media{}, err
				}
				return map[string]any{
						"@type":    "Media",
						"kind":     "photo",
						"blobId":   blob.BlobId,
						"contexts": toBoolMapS("private"),
					}, jscontact.Media{
						// Type: jscontact.MediaType,
						Kind:      jscontact.MediaKindPhoto,
						BlobId:    blob.BlobId,
						MediaType: blob.Type,
						Contexts:  toBoolMapS(jscontact.MediaContextPrivate),
					}, nil

			} else {
				uri := picsum(128, 128)
				return map[string]any{
						"@type":    "Media",
						"kind":     "photo",
						"uri":      uri,
						"contexts": toBoolMapS("work"),
					}, jscontact.Media{
						// Type: jscontact.MediaType,
						Kind:     jscontact.MediaKindPhoto,
						Uri:      uri,
						Contexts: toBoolMapS(jscontact.MediaContextWork),
					}, nil
			}
		}); err != nil {
			return "", "", nil, err
		}
		if err := propmap(contact, "links", &card.Links, 0, 1, func(i int, id string) (map[string]any, jscontact.Link, error) {
			return map[string]any{
					"@type": "Link",
					"kind":  "contact",
					"uri":   "mailto:" + person.Contact.Email,
					"pref":  (i + 1) * 10,
				}, jscontact.Link{
					// Type: jscontact.LinkType,
					Kind: jscontact.LinkKindContact,
					Uri:  "mailto:" + person.Contact.Email,
					Pref: uint((i + 1) * 10),
				}, nil
		}); err != nil {
			return "", "", nil, err
		}

		uid, err := s.CreateContact(c, accountId, contact)
		if err != nil {
			return "", "", nil, err
		}
		filled[uid] = card
		printer(fmt.Sprintf("🧑🏻 created %*s/%v uid=%v", int(math.Log10(float64(count))+1), strconv.Itoa(int(i+1)), count, uid))
	}
	return accountId, addressbookId, filled, nil
}

func (s *StalwartTest) CreateContact(j *TestJmapClient, accountId string, contact map[string]any) (string, error) {
	body := map[string]any{
		"using": []string{JmapCore, JmapContacts},
		"methodCalls": []any{
			[]any{
				ContactCardType + "/set",
				map[string]any{
					"accountId": accountId,
					"create": map[string]any{
						"c": contact,
					},
				},
				"0",
			},
		},
	}
	return testCreate(j, "c", ContactCardType, body)
}

var streetNumberRegex = regexp.MustCompile(`^(\d+)\s+(.+)$`)

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

func (j *TestJmapClient) uploadBlob(accountId string, data []byte, mimetype string) (uploadedBlob, error) {
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

func testCommand[T any](j *TestJmapClient, body map[string]any, closure func([]any) (T, error)) (T, error) {
	var zero T

	payload, err := json.Marshal(body)
	if err != nil {
		return zero, err
	}
	req, err := http.NewRequest(http.MethodPost, j.u.String(), bytes.NewReader(payload))
	if err != nil {
		return zero, err
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
		return zero, err
	}
	defer resp.Body.Close()
	var response []byte = nil
	if j.trace {
		if b, err := httputil.DumpResponse(resp, false); err == nil {
			response, err = io.ReadAll(resp.Body)
			if err != nil {
				return zero, err
			}
			p := pretty.Pretty(response)
			if j.color {
				p = pretty.Color(p, nil)
			}
			log.Printf("<== %s%s\n", b, p)
		}
	}
	if resp.StatusCode >= 300 {
		return zero, fmt.Errorf("JMAP command HTTP response status is %s", resp.Status)
	}
	if response == nil {
		response, err = io.ReadAll(resp.Body)
		if err != nil {
			return zero, err
		}
	}

	r := map[string]any{}
	err = json.Unmarshal(response, &r)
	if err != nil {
		return zero, err
	}

	methodResponses := r["methodResponses"].([]any)
	return closure(methodResponses)
}

func testCreate(j *TestJmapClient, id string, objectType ObjectType, body map[string]any) (string, error) {
	return testCommand(j, body, func(methodResponses []any) (string, error) {
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
	})
}

func testObjectsById(j *TestJmapClient, accountId string, objectType ObjectType, scope string) (map[string]map[string]any, error) {
	m := map[string]map[string]any{}
	{
		body := map[string]any{
			"using": []string{JmapCore, scope},
			"methodCalls": []any{
				[]any{
					objectType + "/get",
					map[string]any{
						"accountId": accountId,
					},
					"0",
				},
			},
		}
		result, err := testCommand(j, body, func(methodResponses []any) ([]any, error) {
			z := methodResponses[0].([]any)
			f := z[1].(map[string]any)
			if list, ok := f["list"]; ok {
				return list.([]any), nil
			} else {
				return nil, fmt.Errorf("methodResponse[1] has no 'list' attribute: %v", f)
			}
		})
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
		// Type: jscontact.NameType,
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
		// Type:  jscontact.NameComponentType,
		Kind:  jscontact.NameComponentKindGiven,
		Value: person.FirstName,
	}
	mComps[1] = map[string]string{
		"kind":  "surname",
		"value": person.LastName,
	}
	oComps[1] = jscontact.NameComponent{
		// Type:  jscontact.NameComponentType,
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
			// Type:     jscontact.NicknameType,
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
			// Type:     jscontact.EmailAddressType,
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
			// Type:     jscontact.EmailAddressType,
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
}

var rooms = []jscalendar.Location{
	{
		Type:          "Location",
		Name:          "office-upstairs",
		Description:   "Office meeting room upstairs",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice),
		Coordinates:   "geo:52.5335389,13.4103296",
		Links: map[string]jscalendar.Link{
			id(): {Href: "https://www.heinlein-support.de/"},
		},
	},
	{
		Type:          "Location",
		Name:          "office-nue",
		Description:   "",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice),
		Coordinates:   "geo:49.4723337,11.1042282",
		Links: map[string]jscalendar.Link{
			id(): {Href: "https://www.workandpepper.de/"},
		},
	},
	{
		Type:          "Location",
		Name:          "Meetingraum Prenzlauer Berg",
		Description:   "This is a Hero Space with great reviews, fast response-time and good quality service",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice, jscalendar.LocationTypeOptionPublic),
		Coordinates:   "geo:52.554222,13.4142387",
		Links: map[string]jscalendar.Link{
			id(): {Href: "https://www.spacebase.com/en/venue/meeting-room-prenzlauer-be-11499/"},
		},
	},
	{
		Type:          "Location",
		Name:          "Meetingraum LIANE 1",
		Description:   "Ecofriendly Bright Urban Jungle",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice, jscalendar.LocationTypeOptionLibrary),
		Coordinates:   "geo:52.4854301,13.4224763",
		Links: map[string]jscalendar.Link{
			id(): {Href: "https://www.spacebase.com/en/venue/rent-a-jungle-8372/"},
		},
	},
	{
		Type:          "Location",
		Name:          "Dark Horse",
		Description:   "Collaboration and event spaces from the authors of the Workspace and Digital Innovation Playbooks.",
		LocationTypes: toBoolMapS(jscalendar.LocationTypeOptionOffice),
		Coordinates:   "geo:52.4942254,13.4346015",
		Links: map[string]jscalendar.Link{
			id(): {Href: "https://www.spacebase.com/en/event-venue/workshop-white-space-2667/"},
		},
	},
}

var virtualRooms = []jscalendar.VirtualLocation{
	{
		Type:        "VirtualLocation",
		Name:        "opentalk",
		Description: "the main room in our opentalk instance",
		Uri:         "https://meet.opentalk.eu/fake/room/" + gofakeit.UUID(),
		Features: toBoolMapS(
			jscalendar.VirtualLocationFeatureAudio,
			jscalendar.VirtualLocationFeatureChat,
			jscalendar.VirtualLocationFeatureVideo,
			jscalendar.VirtualLocationFeatureScreen,
		),
	},
}

func createLocation() (string, jscalendar.Location) {
	locationId := id()
	room := rooms[rand.Intn(len(rooms))]
	return locationId, room
}

func createVirtualLocation() (string, jscalendar.VirtualLocation) {
	locationId := id()
	return locationId, virtualRooms[rand.Intn(len(virtualRooms))]
}

var ChairRoles = toBoolMapS("attendee", "chair", "owner")
var RegularRoles = toBoolMapS("attendee")

func createParticipants(locationId string, virtualLocationid string) (map[string]map[string]any, string) {
	n := 1 + rand.Intn(4)
	participants := map[string]map[string]any{}
	organizerId, organizerEmail, organizer := createParticipant(0, pickRandom(locationId, virtualLocationid), "", "")
	participants[organizerId] = organizer
	for i := 1; i < n; i++ {
		id, _, participant := createParticipant(i, pickRandom(locationId, virtualLocationid), organizerId, organizerEmail)
		participants[id] = participant
	}
	return participants, organizerEmail
}

func createParticipant(i int, locationId string, organizerEmail string, organizerId string) (string, string, map[string]any) {
	participantId := id()
	person := gofakeit.Person()
	roles := RegularRoles
	if i == 0 {
		roles = ChairRoles
	}
	status := "accepted"
	if i != 0 {
		status = pickRandom("needs-action", "accepted", "declined", "tentative") //, delegated + set "delegatedTo"
	}
	statusComment := ""
	if rand.Intn(5) >= 3 {
		statusComment = gofakeit.HipsterSentence(1 + rand.Intn(5))
	}
	if i == 0 {
		organizerEmail = person.Contact.Email
		organizerId = participantId
	}
	m := map[string]any{
		"@type":       "Participant",
		"name":        person.FirstName + " " + person.LastName,
		"email":       person.Contact.Email,
		"description": person.Job.Title,
		"sendTo": map[string]string{
			"imip": "mailto:" + person.Contact.Email,
		},
		"kind":                 "individual",
		"roles":                roles,
		"locationId":           locationId,
		"language":             pickLanguage(),
		"participationStatus":  status,
		"participationComment": statusComment,
		"expectReply":          true,
		"scheduleAgent":        "server",
		"scheduleSequence":     1,
		"scheduleStatus":       []string{"1.0"},
		"scheduleUpdated":      "2025-10-01T1:59:12Z",
		"sentBy":               organizerEmail,
		"invitedBy":            organizerId,
		"scheduleId":           "mailto:" + person.Contact.Email,
	}

	links := map[string]map[string]any{}
	for range rand.Intn(3) {
		links[id()] = map[string]any{
			"@type":       "Link",
			"href":        "https://picsum.photos/id/" + strconv.Itoa(1+rand.Intn(200)) + "/200/300",
			"contentType": "image/jpeg",
			"rel":         "icon",
			"display":     "badge",
			"title":       person.FirstName + "'s Cake Day pick",
		}
	}
	if len(links) > 0 {
		m["links"] = links
	}

	return participantId, person.Contact.Email, m
}

var Keywords = []string{
	"office",
	"important",
	"sales",
	"coordination",
	"decision",
}

func keywords() map[string]bool {
	return toBoolMap(pickRandoms(Keywords...))
}

var Categories = []string{
	"secret",
	"internal",
}

func categories() map[string]bool {
	return toBoolMap(pickRandoms(Categories...))
}

func propmap[T any](container map[string]any, name string, cardProperty *map[string]T, min int, max int, generator func(int, string) (map[string]any, T, error)) error {
	n := min + rand.Intn(max-min+1)
	if n < 1 {
		return nil
	}

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

func picsum(w, h int) string {
	return fmt.Sprintf("https://picsum.photos/id/%d/%d/%d", 1+rand.Intn(200), h, w)
}

func orNilMap[K comparable, V any](m map[K]V) map[K]V {
	if len(m) < 1 {
		return nil
	} else {
		return m
	}
}

func orNilSlice[E any](s []E) []E {
	if len(s) < 1 {
		return nil
	} else {
		return s
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

func pickRandoms[T any](s ...T) []T {
	n := rand.Intn(len(s))
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
