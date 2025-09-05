package jmap

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/mail"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"text/template"
	"time"

	"github.com/jhillyerd/enmime/v2"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	petname "github.com/dustinkirkland/golang-petname"
	pw "github.com/sethvargo/go-password/password"
	"gopkg.in/loremipsum.v1"

	clog "github.com/opencloud-eu/opencloud/pkg/log"

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
	stalwartImage  = "ghcr.io/stalwartlabs/stalwart:v0.13.2-alpine"
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

func fill(require *require.Assertions, i *imapclient.Client, folder string, to string, count int, ccEvery int, bccEvery int) {
	address, err := mail.ParseAddress(to)
	require.NoError(err)
	displayName := address.Name

	addressParts := emailSplitter.FindAllStringSubmatch(address.Address, 3)
	require.Len(addressParts, 1)
	require.Len(addressParts[0], 3)
	domain := addressParts[0][2]

	toName := displayName
	toAddress := to
	ccName1 := "Team Lead"
	ccAddress1 := fmt.Sprintf("lead@%s", domain)
	ccName2 := "Coworker"
	ccAddress2 := fmt.Sprintf("coworker@%s", domain)
	bccName := "HR"
	bccAddress := fmt.Sprintf("corporate@%s", domain)
	titler := cases.Title(language.English, cases.NoLower)

	loremIpsumGenerator := loremipsum.New()
	for n := range count {
		first := petname.Adjective()
		last := petname.Adverb()
		messageId := fmt.Sprintf("%d.%d@%s", time.Now().Unix(), 1000000+rand.Intn(8999999), domain)

		format := formats[n%len(formats)]

		text := loremIpsumGenerator.Paragraphs(2 + rand.Intn(9))
		from := fmt.Sprintf("%s.%s@%s", strings.ToLower(first), strings.ToLower(last), domain)
		sender := fmt.Sprintf("%s %s <%s.%s@%s>", titler.String(first), titler.String(last), strings.ToLower(first), strings.ToLower(last), domain)

		msg := enmime.Builder().
			From(titler.String(first)+" "+titler.String(last), from).
			Subject(titler.String(loremIpsumGenerator.Words(3+rand.Intn(7)))).
			Header("Message-ID", messageId).
			Header("Sender", sender).
			To(toName, toAddress)

		if n%ccEvery == 0 {
			msg = msg.CCAddrs([]mail.Address{{Name: ccName1, Address: ccAddress1}, {Name: ccName2, Address: ccAddress2}})
		}
		if n%bccEvery == 0 {
			msg = msg.BCC(bccName, bccAddress)
		}

		msg = format(text, msg)

		buf := new(bytes.Buffer)
		part, _ := msg.Build()
		part.Encode(buf)
		mail := buf.String()

		size := int64(len(mail))
		appendCmd := i.Append(folder, size, nil)
		_, err := appendCmd.Write([]byte(mail))
		require.NoError(err)
		err = appendCmd.Close()
		require.NoError(err)
		_, err = appendCmd.Wait()
		require.NoError(err)
	}
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

func TestWithStalwart(t *testing.T) {
	if skip(t) {
		return
	}
	require := require.New(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// A master user name different from "master" does not seem to work as of the current Stalwart version
	//masterUsernameSuffix, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	//require.NoError(err)
	masterUsername := "master" //"master_" + masterUsernameSuffix

	masterPassword, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	require.NoError(err)
	masterPasswordHash := ""
	{
		hasher, err := shacrypt.New(shacrypt.WithSHA512(), shacrypt.WithIterations(shacrypt.IterationsDefaultOmitted))
		require.NoError(err)

		digest, err := hasher.Hash(masterPassword)
		require.NoError(err)
		masterPasswordHash = digest.Encode()
	}

	usernameSuffix, err := pw.Generate(8, 2, 0, true, true)
	require.NoError(err)
	username := "user_" + usernameSuffix

	password, err := pw.Generate(4+rand.Intn(28), 2, 0, false, true)
	require.NoError(err)

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

	defer func() {
		testcontainers.CleanupContainer(t, container)
	}()
	require.NoError(err)

	ip, err := container.Host(ctx)
	require.NoError(err)

	port, err := container.MappedPort(ctx, "993")
	require.NoError(err)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	count := 5

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

		jmapPort, err := container.MappedPort(ctx, httpPort)
		require.NoError(err)
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

		j = NewClient(api, api, api)
		s, err := j.FetchSession(sessionUrl, username, logger)
		require.NoError(err)
		// we have to overwrite the hostname in JMAP URL because the container
		// will know its name to be a random Docker container identifier, or
		// "localhost" as we defined the hostname in the Stalwart configuration,
		// and we also need to overwrite the port number as its not mapped
		s.JmapUrl.Host = jmapBaseUrl.Host
		session = &s
	}

	accountId := session.PrimaryAccounts.Mail

	var inboxFolder string
	var inboxId string
	{
		resp, sessionState, err := j.GetAllMailboxes(accountId, session, ctx, logger)
		require.NoError(err)
		require.Equal(session.State, sessionState)
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

	{
		c, err := imapclient.DialTLS(net.JoinHostPort(ip, port.Port()), &imapclient.Options{TLSConfig: tlsConfig})
		require.NoError(err)

		defer func(imap *imapclient.Client) {
			err := imap.Close()
			if err != nil {
				log.Fatal(err)
			}
		}(c)

		err = c.Login(username, password).Wait()
		require.NoError(err)

		_, err = c.Select(inboxFolder, nil).Wait()
		require.NoError(err)

		fill(require, c, inboxFolder, fmt.Sprintf("%s <%s>", userPersonName, userEmail), count, 2, 3)

		listCmd := c.List("", "%", &imap.ListOptions{
			ReturnStatus: &imap.StatusOptions{
				NumMessages: true,
				NumUnseen:   true,
			},
		})
		countMap := make(map[string]int)
		for {
			mbox := listCmd.Next()
			if mbox == nil {
				break
			}
			countMap[mbox.Mailbox] = int(*mbox.Status.NumMessages)
		}

		inboxCount := -1
		for f, i := range countMap {
			if strings.Compare(strings.ToLower(f), strings.ToLower(inboxFolder)) == 0 {
				inboxCount = i
				break
			}
		}
		if inboxCount == -1 {
			require.FailNowf("huh", "failed to find folder '%v' via IMAP", inboxFolder)
		}
		require.Equal(count, inboxCount)

		err = listCmd.Close()
		require.NoError(err)
	}

	{
		{
			resp, sessionState, err := j.GetIdentity(accountId, session, ctx, logger)
			require.NoError(err)
			require.Equal(session.State, sessionState)
			require.Len(resp.Identities, 1)
			require.Equal(userEmail, resp.Identities[0].Email)
			require.Equal(userPersonName, resp.Identities[0].Name)
		}

		{
			resp, sessionState, err := j.GetAllMailboxes(accountId, session, ctx, logger)
			require.NoError(err)
			require.Equal(session.State, sessionState)
			mailboxesUnreadByRole := map[string]int{}
			for _, m := range resp.Mailboxes {
				if m.Role != "" {
					mailboxesUnreadByRole[m.Role] = m.UnreadEmails
				}
			}
			require.Equal(count, mailboxesUnreadByRole["inbox"])
		}

		{
			resp, sessionState, err := j.GetAllEmailsInMailbox(accountId, session, ctx, logger, inboxId, 0, 0, false, 0)
			require.NoError(err)
			require.Equal(session.State, sessionState)

			require.Len(resp.Emails, count)
			for _, e := range resp.Emails {
				require.Empty(e.BodyValues)
				require.False(e.HasAttachment)
				require.NotEmpty(e.Subject)
				require.NotEmpty(e.MessageId)
				require.NotEmpty(e.Preview)
			}
		}
	}
}
