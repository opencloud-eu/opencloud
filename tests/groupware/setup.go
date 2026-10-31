package main

import (
	"bytes"
	crand "crypto/rand"
	"crypto/sha1"
	"crypto/tls"
	"encoding/base64"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"net/http"
	"net/mail"
	"net/url"

	petname "github.com/dustinkirkland/golang-petname"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/go-faker/faker/v4"
	"github.com/go-ldap/ldap/v3"
	"github.com/jhillyerd/enmime"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/loremipsum.v1"
)

var usersToKeep = []string{"lynn", "alan", "mary", "margaret"}

const displayNameMark = "$generated"

func enabled(value string) bool {
	value = strings.ToLower(value)
	return value == "true" || value == "on" || value == "1"
}

func config(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	} else {
		return defaultValue
	}
}

func iconfig(log *zerolog.Logger, key string, defaultValue int) int {
	value, ok := os.LookupEnv(key)
	if ok {
		result, err := strconv.Atoi(value)
		if err != nil {
			log.Fatal().Msgf("invalid value for %v is not numeric: '%v'", key, value)
			panic(err)
		} else {
			return result
		}
	} else {
		return defaultValue
	}
}

func hashPassword(clear string, saltSize int) string {
	salt := make([]byte, saltSize)
	crand.Read(salt)
	sha := sha1.New()
	sha.Write([]byte(clear))
	sha.Write([]byte(salt))
	digest := sha.Sum(nil)
	combined := append(digest, salt...)
	return "{SSHA}" + base64.StdEncoding.EncodeToString(combined)
}

const passwordCharset = "abcdefghijklmnopqrstuvwxyz" + "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789" + "0123456789"

var seededRand *rand.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))

func randomPassword() string {
	length := 8 + rand.Intn(32)
	b := make([]byte, length)
	for i := range b {
		b[i] = passwordCharset[seededRand.Intn(len(passwordCharset))]
	}
	return string(b)
}

func htmlJoin(parts []string) []string {
	var result []string
	for i := range parts {
		result = append(result, fmt.Sprintf("<p>%v</p>", parts[i]))
	}
	return result
}

var paraSplitter = regexp.MustCompile("[\r\n]+")

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

func fill(i *imapclient.Client, folder string, count int, uid string, clearPassword string, displayName string, domain string, ccEvery int, bccEvery int) {
	err := i.Login(uid, clearPassword).Wait()
	if err != nil {
		panic(err)
	}

	selectOptions := &imap.SelectOptions{ReadOnly: false}
	_, err = i.Select(folder, selectOptions).Wait()
	if err != nil {
		panic(err)
	}

	toName := displayName
	toAddress := fmt.Sprintf("%s@%s", uid, domain)
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
		if _, err := appendCmd.Write([]byte(mail)); err != nil {
			log.Error().Err(err).Str("uid", uid).Msg("imap: failed to append message")
		}
		if err := appendCmd.Close(); err != nil {
			log.Error().Err(err).Str("uid", uid).Msg("imap: failed to close append command")
		}
		if _, err := appendCmd.Wait(); err != nil {
			log.Error().Err(err).Str("uid", uid).Msg("imap: append command failed")
		}
	}

	if err = i.Logout().Wait(); err != nil {
		panic(err)
	}
}

type User struct {
	uid      string
	password string
}

type PrincipalRoles []string

func (r PrincipalRoles) MarshalZerologArray(a *zerolog.Array) {
	for _, role := range r {
		a.Str(role)
	}
}

type Principal struct {
	Id          int            `json:"id,omitempty"`
	Type        string         `json:"type,omitempty"`
	Emails      []string       `json:"emails,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Roles       PrincipalRoles `json:"roles,omitempty"`
	Secrets     []string       `json:"secrets,omitempty"`
}

type Principals struct {
	Data struct {
		Items []Principal `json:"items,omitempty"`
	} `json:"data,omitzero"`
	Total int `json:"total,omitempty"`
}

type StalwartOAuthRequest struct {
	Type        string `json:"type"`
	ClientId    string `json:"client_id"`
	RedirectUri string `json:"redirect_uri"`
	Nonce       string `json:"nonce"`
}

func activateUsersInStalwart(_ *zerolog.Logger, baseurl string, users []User) []User {
	var h *http.Client
	{
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		h = &http.Client{Transport: tr}
	}
	u, err := url.Parse(baseurl)
	if err != nil {
		panic(err)
	}
	u.Path = path.Join(u.Path, "api", "oauth")

	activated := []User{}
	for _, user := range users {
		oauth := StalwartOAuthRequest{Type: "code", ClientId: "groupware", RedirectUri: "stalwart://auth", Nonce: "aaa"}
		body, err := json.Marshal(oauth)
		if err != nil {
			panic(err)
		}
		req, err := http.NewRequest("POST", u.String(), bytes.NewReader(body))
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.SetBasicAuth(user.uid, user.password)
		resp, err := h.Do(req)
		if err != nil {
			panic(err)
		}
		defer func(r *http.Response) {
			r.Body.Close()
		}(resp)
		if resp.StatusCode == 200 {
			activated = append(activated, user)
		} else {
			panic(fmt.Errorf("the Stalwart API response is not 200 but %v %v", resp.StatusCode, resp.Status))
		}
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
	}
	return activated
}

func cleanStalwart(log *zerolog.Logger, baseurl string, adminUsername string, adminPassword string) []Principal {
	var h *http.Client
	{
		tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
		h = &http.Client{Transport: tr}
	}

	var principals Principals
	{
		u, err := url.Parse(baseurl)
		if err != nil {
			panic(err)
		}
		u.Path = path.Join(u.Path, "api", "principal")
		req, err := http.NewRequest("GET", u.String(), nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.SetBasicAuth(adminUsername, adminPassword)
		resp, err := h.Do(req)
		if err != nil {
			panic(err)
		}
		defer func(r *http.Response) {
			r.Body.Close()
		}(resp)
		if resp.StatusCode != 200 {
			panic(fmt.Errorf("the Stalwart API response is not 200 but %v %v", resp.StatusCode, resp.Status))
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		err = json.Unmarshal(body, &principals)
		if err != nil {
			panic(err)
		}
	}

	deleted := []Principal{}
	for _, principal := range principals.Data.Items {
		if principal.Type != "individual" {
			log.Debug().Str("name", principal.Name).Str("type", principal.Type).Msgf("stalwart: preserving principal: type is not '%v'", "individual")
			continue
		}
		if !slices.Contains(principal.Roles, "user") {
			log.Debug().Str("name", principal.Name).Array("roles", principal.Roles).Msgf("stalwart: preserving principal: does not have the role '%v'", "user")
			continue
		}
		if slices.Contains(usersToKeep, principal.Name) {
			log.Debug().Str("name", principal.Name).Msg("stalwart: preserving principal: is a user to keep")
			continue
		}
		if !strings.HasSuffix(principal.Description, displayNameMark) {
			log.Debug().Str("name", principal.Name).Str("description", principal.Description).Msgf("stalwart: preserving principal: does not have the description suffix '%v'", displayNameMark)
			continue
		}
		log.Debug().Str("name", principal.Name).Msg("stalwart: will delete principal")

		u, err := url.Parse(baseurl)
		if err != nil {
			panic(err)
		}
		// the documentation states "principal_id" but it only works with the principal's name attribute
		u.Path = path.Join(u.Path, "api", "principal", principal.Name) // strconv.Itoa(principal.Id))

		req, err := http.NewRequest("DELETE", u.String(), nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.SetBasicAuth(adminUsername, adminPassword)
		resp, err := h.Do(req)
		if err != nil {
			panic(err)
		}
		defer func(r *http.Response) {
			r.Body.Close()
		}(resp)
		if resp.StatusCode != 200 {
			panic(fmt.Errorf("the Stalwart API response is not 200 but %v %v", resp.StatusCode, resp.Status))
		}
		_, err = io.ReadAll(resp.Body)
		if err != nil {
			panic(err)
		}
		deleted = append(deleted, principal)
	}
	return deleted
}

func main() {
	log := zerolog.New(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.TimeOnly}).With().Timestamp().Logger()

	fillImapInbox := enabled(config("FILL_IMAP", "true"))
	imapHost := config("FILL_IMAP_HOST", "localhost:636")
	ccEvery := iconfig(&log, "FILL_IMAP_CC_EVERY", 3)
	bccEvery := iconfig(&log, "FILL_IMAP_BCC_EVERY", 2)
	folder := config("FILL_IMAP_FOLDER", "Inbox")
	imapCount := iconfig(&log, "FILL_IMAP_COUNT", 10)
	domain := config("DOMAIN", "example.org")
	baseDN := config("BASE_DN", "ou=users,dc=opencloud,dc=eu")
	ldapUrl := config("LDAP_URL", "ldaps://localhost:636")
	bindDN := config("BIND_DN", "cn=admin,dc=opencloud,dc=eu")
	bindPassword := config("BIND_PASSWORD", "admin")
	userPassword := config("USER_PASSWORD", "")
	usersFile := config("USERS_FILE", "")
	count := iconfig(&log, "COUNT", 10)
	cleanup := enabled(config("CLEANUP", "true"))
	cleanupLdap := enabled(config("CLEANUP_LDAP", strconv.FormatBool(cleanup)))
	cleanupStalwart := enabled(config("CLEANUP_STALWART", strconv.FormatBool(cleanup)))
	stalwartBaseUrl := config("STALWART_URL", "https://stalwart.opencloud.test")
	stalwartAdminUser := config("STALWART_ADMIN_USER", "mailadmin")
	stalwartAdminPassword := config("STALWART_ADMIN_PASSWORD", "admin")
	activateStalwart := enabled(config("ACTIVATE_STALWART", "true"))
	saltSize := iconfig(&log, "SALT_SIZE", 16)

	l, err := ldap.DialURL(ldapUrl, ldap.DialWithTLSConfig(&tls.Config{InsecureSkipVerify: true}))
	if err != nil {
		log.Fatal().Err(err).Str("url", ldapUrl).Msg("failed to connect to LDAP server")
		panic(err)
	}
	err = l.Bind(bindDN, bindPassword)
	if err != nil {
		log.Fatal().Err(err).Str("url", ldapUrl).Str("bindDN", bindDN).Msg("failed to authenticate to LDAP server")
		panic(err)
	}

	var i *imapclient.Client
	if fillImapInbox {
		i, err := imapclient.DialTLS(imapHost, &imapclient.Options{TLSConfig: &tls.Config{InsecureSkipVerify: true}})
		if err != nil {
			log.Fatal().Err(err).Str("host", imapHost).Msg("failed to connect to IMAP server")
			panic(err)
		}
		defer func(imap *imapclient.Client) {
			err := imap.Close()
			if err != nil {
				log.Warn().Err(err).Msg("failed to close IMAP connection")
			}
		}(i)
	} else {
		i = nil
	}

	if cleanupStalwart {
		deleted := cleanStalwart(&log, stalwartBaseUrl, stalwartAdminUser, stalwartAdminPassword)
		log.Info().Msgf("deleted %v principals from Stalwart", len(deleted))
	}

	if cleanupLdap {
		deleted := []string{}
		{
			llog := log.With().Str("url", ldapUrl).Logger()
			llog.Debug().Msg("ldap: cleaning up LDAP")
			filter := fmt.Sprintf("(&(objectClass=inetOrgPerson)(description=%v))", ldap.EscapeFilter(displayNameMark))
			existing, err := l.Search(ldap.NewSearchRequest(
				baseDN,
				ldap.ScopeSingleLevel,
				ldap.NeverDerefAliases,
				0, 0, false,
				filter,
				[]string{"uid"},
				[]ldap.Control{},
			))
			if err != nil {
				llog.Fatal().Err(err).Str("filter", filter).Msg("ldap: failed to perform search query")
				panic(err)
			}

			for _, entry := range existing.Entries {
				uid := entry.GetAttributeValue("uid")
				if slices.Contains(usersToKeep, uid) {
					llog.Debug().Str("uid", uid).Msg("ldap: preserving user: in list of users to keep")
					continue
				}
				err = l.Del(ldap.NewDelRequest(entry.DN, []ldap.Control{}))
				if err != nil {
					llog.Fatal().Err(err).Msg("ldap: failed to delete entry")
					panic(err)
				}
				deleted = append(deleted, uid)
				llog.Debug().Str("dn", entry.DN).Msg("ldap: deleted user entry")
			}
		}
		log.Info().Msgf("ldap: deleted %v user entries", len(deleted))
	}

	created := []User{}
	{
		var flog zerolog.Logger
		if usersFile != "" {
			flog = log.With().Str("filename", usersFile).Logger()
		} else {
			flog = log
		}
		llog := log.With().Str("url", ldapUrl).Logger()

		var d io.Writer
		{
			if usersFile != "" {
				f, err := os.OpenFile(usersFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
				if err != nil {
					flog.Fatal().Err(err).Msg("failed to open/create users output CSV file")
					panic(err)
				}
				defer f.Close()
				d = f
			} else {
				d = os.Stdout
			}
		}
		w := csv.NewWriter(d)
		w.Comma = ';'
		w.UseCRLF = false
		err = w.Write([]string{"name", "password", "mail"})
		if err != nil {
			flog.Fatal().Err(err).Msg("failed to open/create users output CSV file")
			panic(err)
		}
		for range count {
			cn := strings.ToLower(faker.Username())
			uid := cn
			gn := faker.FirstName()
			sn := faker.LastName()
			mailAddress := fmt.Sprintf("%s@%s", uid, domain)
			dn := fmt.Sprintf("uid=%s,%s", uid, baseDN)
			displayName := fmt.Sprintf("%s %s %s", gn, sn, displayNameMark)
			description := displayNameMark
			var clearPassword string
			if userPassword != "" {
				clearPassword = userPassword
			} else {
				clearPassword = randomPassword()
			}
			hashedPassword := hashPassword(clearPassword, saltSize)
			err = l.Add(&ldap.AddRequest{
				DN: dn,
				Attributes: []ldap.Attribute{
					{Type: "objectClass", Vals: []string{"inetOrgPerson", "organizationalPerson", "person", "top"}},
					{Type: "cn", Vals: []string{cn}},
					{Type: "sn", Vals: []string{sn}},
					{Type: "givenName", Vals: []string{gn}},
					{Type: "mail", Vals: []string{mailAddress}},
					{Type: "displayName", Vals: []string{displayName}},
					{Type: "description", Vals: []string{description}},
					{Type: "userPassword", Vals: []string{hashedPassword}},
				},
			})
			if err != nil {
				llog.Fatal().Err(err).Str("uid", uid).Msg("failed to add entry")
				panic(err)
			}
			err = w.Write([]string{uid, clearPassword, mailAddress})
			if err != nil {
				flog.Fatal().Err(err).Str("uid", uid).Msg("failed to write entry to CSV")
				panic(err)
			}

			if i != nil && imapCount > 0 {
				fill(i, folder, imapCount, uid, clearPassword, displayName, domain, ccEvery, bccEvery)
			}
			created = append(created, User{uid: uid, password: clearPassword})
		}
		w.Flush()
		if err := w.Error(); err != nil {
			flog.Fatal().Err(err).Msg("failed to flush CSV")
			panic(err)
		}

		{
			zev := log.Info()
			if usersFile != "" {
				zev = zev.Str("filename", usersFile)
			}
			zev.Msgf("ldap: added %v users", len(created))
		}
	}

	if activateStalwart && len(created) > 0 {
		activated := activateUsersInStalwart(&log, stalwartBaseUrl, created)
		log.Info().Msgf("stalwart: activated %v users", len(activated))
	}
}
