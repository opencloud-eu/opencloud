package jmap

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/gorilla/websocket"
	"github.com/jhillyerd/enmime/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/brianvoe/gofakeit/v7"
	pw "github.com/sethvargo/go-password/password"

	clog "github.com/opencloud-eu/opencloud/pkg/log"
	"github.com/opencloud-eu/opencloud/pkg/structs"

	"github.com/go-crypt/crypt/algorithm/shacrypt"
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
	stalwartImage  = "ghcr.io/stalwartlabs/stalwart:v0.13.4-alpine"
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

func newSenderGenerator(domain string, numSenders int) senderGenerator {
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
		session = &s
	}

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
}

func (s *StalwartTest) fill(folder string, count int) ([]filledMail, int, error) {
	to := fmt.Sprintf("%s <%s>", s.userPersonName, s.userEmail)
	ccEvery := 2
	bccEvery := 3
	attachmentEvery := 2
	seenEvery := 3
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

	sg := newSenderGenerator(domain, senders)
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

			buf := new(bytes.Buffer)
			part, _ := msg.Build()
			part.Encode(buf)
			mail := buf.String()

			var flags *imap.AppendOptions = nil
			if i%seenEvery == 0 {
				flags = &imap.AppendOptions{Flags: []imap.Flag{imap.FlagSeen}}
			}

			size := int64(len(mail))
			appendCmd := c.Append(folder, size, flags)
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

func TestEmails(t *testing.T) {
	if skip(t) {
		return
	}

	count := 25

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	accountId := s.session.PrimaryAccounts.Mail

	var inboxFolder string
	var inboxId string
	{
		respByAccountId, sessionState, _, err := s.client.GetAllMailboxes([]string{accountId}, s.session, s.ctx, s.logger, "")
		require.NoError(err)
		require.Equal(s.session.State, sessionState)
		require.Len(respByAccountId, 1)
		require.Contains(respByAccountId, accountId)
		resp := respByAccountId[accountId]

		mailboxesNameByRole := map[string]string{}
		mailboxesUnreadByRole := map[string]int{}
		for _, m := range resp.Mailboxes {
			if m.Role != "" {
				mailboxesNameByRole[m.Role] = m.Name
				mailboxesUnreadByRole[m.Role] = m.UnreadEmails
			}
		}
		require.Contains(mailboxesNameByRole, "inbox")
		require.Contains(mailboxesUnreadByRole, "inbox")
		require.Zero(mailboxesUnreadByRole["inbox"])

		inboxId = mailboxId("inbox", resp.Mailboxes)
		require.NotEmpty(inboxId)
		inboxFolder = mailboxesNameByRole["inbox"]
		require.NotEmpty(inboxFolder)
	}

	var threads int = 0
	var mails []filledMail = nil
	{
		mails, threads, err = s.fill(inboxFolder, count)
		require.NoError(err)
	}
	mailsByMessageId := structs.Index(mails, func(mail filledMail) string { return mail.messageId })

	{
		{
			resp, sessionState, _, err := s.client.GetIdentity(accountId, s.session, s.ctx, s.logger, "")
			require.NoError(err)
			require.Equal(s.session.State, sessionState)
			require.Len(resp.Identities, 1)
			require.Equal(s.userEmail, resp.Identities[0].Email)
			require.Equal(s.userPersonName, resp.Identities[0].Name)
		}

		{
			respByAccountId, sessionState, _, err := s.client.GetAllMailboxes([]string{accountId}, s.session, s.ctx, s.logger, "")
			require.NoError(err)
			require.Equal(s.session.State, sessionState)
			require.Len(respByAccountId, 1)
			require.Contains(respByAccountId, accountId)
			resp := respByAccountId[accountId]
			mailboxesUnreadByRole := map[string]int{}
			for _, m := range resp.Mailboxes {
				if m.Role != "" {
					mailboxesUnreadByRole[m.Role] = m.UnreadEmails
				}
			}
			require.LessOrEqual(mailboxesUnreadByRole["inbox"], count)
		}

		{
			resp, sessionState, _, err := s.client.GetAllEmailsInMailbox(accountId, s.session, s.ctx, s.logger, "", inboxId, 0, 0, true, false, 0)
			require.NoError(err)
			require.Equal(s.session.State, sessionState)

			require.Equalf(threads, len(resp.Emails), "the number of collapsed emails in the inbox is expected to be %v, but is actually %v", threads, len(resp.Emails))
			for _, e := range resp.Emails {
				require.Len(e.MessageId, 1)
				expectation, ok := mailsByMessageId[e.MessageId[0]]
				require.True(ok)
				require.Empty(e.BodyValues)
				require.Equal(expectation.subject, e.Subject)
				matchAttachments(t, e, expectation.attachments)
				require.NotEmpty(e.Preview)
			}
		}

		{
			resp, sessionState, _, err := s.client.GetAllEmailsInMailbox(accountId, s.session, s.ctx, s.logger, "", inboxId, 0, 0, false, false, 0)
			require.NoError(err)
			require.Equal(s.session.State, sessionState)

			require.Equalf(count, len(resp.Emails), "the number of emails in the inbox is expected to be %v, but is actually %v", count, len(resp.Emails))
			for _, e := range resp.Emails {
				require.Len(e.MessageId, 1)
				expectation, ok := mailsByMessageId[e.MessageId[0]]
				require.True(ok)
				require.Empty(e.BodyValues)
				require.Equal(expectation.subject, e.Subject)
				matchAttachments(t, e, expectation.attachments)
				require.NotEmpty(e.Preview)
			}
		}
	}
}

func matchAttachments(t *testing.T, email Email, expected []filledAttachment) {
	require := require.New(t)

	list := make([]filledAttachment, len(expected))
	copy(list, expected)

	require.Len(email.Attachments, len(expected))
	for _, a := range email.Attachments {
		// find a match in 'expected'
		found := false
		for j, e := range list {
			if a.Name == e.name {
				found = true
				// found a match, we are assuming that the filenames are unique
				require.Equal(e.name, a.Name)
				require.Equal(e.mimeType, a.Type)
				require.Equal(e.size, a.Size)
				require.Equal(e.disposition, a.Disposition)

				list[j] = list[len(list)-1]
				list = list[:len(list)-1]
				break
			}
		}
		require.True(found)
	}
}
