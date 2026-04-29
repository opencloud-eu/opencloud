package jmap

import (
	"maps"
	"math/rand"
	"slices"
	"strings"
	"testing"

	"bytes"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/mail"
	"regexp"
	"strconv"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jhillyerd/enmime/v2"
	"github.com/opencloud-eu/opencloud/pkg/structs"
	"github.com/stretchr/testify/require"
)

func TestEmails(t *testing.T) {
	if skip(t) {
		return
	}

	count := 15 + rand.Intn(20)

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	user := pickUser()
	session := s.Session(user.name)
	ctx := s.Context(session)

	accountId := session.PrimaryAccounts.Mail

	inboxId, inboxFolder := s.findInbox(t, accountId, ctx)

	var threads int = 0
	var mails []filledMail = nil
	{
		mails, threads, err = s.fillEmailsWithImap(inboxFolder, count, false, user)
		require.NoError(err)
	}
	mailsByMessageId := structs.Index(mails, func(mail filledMail) string { return mail.messageId })

	{
		{
			result, err := s.client.GetIdentities(accountId, []string{}, ctx)
			require.NoError(err)
			require.Equal(session.State, result.GetSessionState())
			require.Len(result.Payload.List, 1)
			require.Equal(user.email, result.Payload.List[0].Email)
			require.Equal(user.description, result.Payload.List[0].Name)
		}

		{
			result, err := s.client.GetAllMailboxes([]string{accountId}, ctx)
			require.NoError(err)
			require.Equal(session.State, result.GetSessionState())
			require.Len(result.Payload, 1)
			require.Contains(result.Payload, accountId)
			resp := result.Payload[accountId]
			mailboxesUnreadByRole := map[string]int{}
			for _, m := range resp {
				if m.Role != "" {
					mailboxesUnreadByRole[m.Role] = m.UnreadEmails
				}
			}
			require.LessOrEqual(mailboxesUnreadByRole["inbox"], count)
		}

		{
			result, err := s.client.GetAllEmailsInMailbox(accountId, inboxId, 0, "", nil, nil, true, false, 0, true, ctx)
			require.NoError(err)
			require.Equal(session.State, result.GetSessionState())

			require.Equalf(threads, len(result.Payload.Results), "the number of collapsed emails in the inbox is expected to be %v, but is actually %v", threads, len(result.Payload.Results))
			for _, e := range result.Payload.Results {
				require.Len(e.MessageId, 1)
				expectation, ok := mailsByMessageId[e.MessageId[0]]
				require.True(ok)
				matchEmail(t, e, expectation, false)
			}
		}

		{
			result, err := s.client.GetAllEmailsInMailbox(accountId, inboxId, 0, "", nil, nil, false, false, 0, true, ctx)
			require.NoError(err)
			require.Equal(session.State, result.GetSessionState())

			require.Equalf(count, len(result.Payload.Results), "the number of emails in the inbox is expected to be %v, but is actually %v", count, len(result.Payload.Results))
			for _, e := range result.Payload.Results {
				require.Len(e.MessageId, 1)
				expectation, ok := mailsByMessageId[e.MessageId[0]]
				require.True(ok)
				matchEmail(t, e, expectation, false)
			}
		}
	}
}

func TestSendingEmails(t *testing.T) {
	if skip(t) {
		return
	}

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	from := pickUser()
	session := s.Session(from.name)
	ctx := s.Context(session)
	accountId := session.PrimaryAccounts.Mail

	var to User
	{
		others := structs.Filter(users[:], func(u User) bool { return u.name != from.name })
		to = others[rand.Intn(len(others))]
	}
	toSession := s.Session(to.name)
	toAccountId := toSession.PrimaryAccounts.Mail

	var cc User
	{
		others := structs.Filter(users[:], func(u User) bool { return u.name != from.name && u.name != to.name })
		cc = others[rand.Intn(len(others))]
	}
	ccSession := s.Session(cc.name)
	ccAccountId := ccSession.PrimaryAccounts.Mail

	var mailboxPerRole map[string]Mailbox
	{
		result, err := s.client.GetAllMailboxes([]string{accountId}, ctx)
		require.NoError(err)
		mailboxPerRole = structs.Index(result.Payload[accountId], func(m Mailbox) string { return m.Role })
		require.Contains(mailboxPerRole, JmapMailboxRoleInbox)
		require.Contains(mailboxPerRole, JmapMailboxRoleDrafts)
		require.Contains(mailboxPerRole, JmapMailboxRoleSent)
		require.Contains(mailboxPerRole, JmapMailboxRoleTrash)
	}
	{
		roles := []string{JmapMailboxRoleDrafts, JmapMailboxRoleSent, JmapMailboxRoleInbox}
		result, err := s.client.SearchMailboxIdsPerRole([]string{accountId}, roles, ctx)
		require.NoError(err)
		require.Contains(result.Payload, accountId)
		a := result.Payload[accountId]
		for _, role := range roles {
			require.Contains(a, role)
		}
	}

	// let's ensure that the recipients have zero emails in their mailboxes before we send them any
	for _, u := range []struct {
		accountId string
		session   *Session
	}{{toAccountId, toSession}, {ccAccountId, ccSession}} {
		uctx := Context{
			Session:        u.session,
			Context:        ctx.Context,
			Logger:         ctx.Logger,
			AcceptLanguage: ctx.AcceptLanguage,
		}
		result, err := s.client.GetAllMailboxes([]string{u.accountId}, uctx)
		require.NoError(err)
		for _, mailbox := range result.Payload[u.accountId] {
			require.Equal(0, mailbox.TotalEmails)
		}
	}

	subject := fmt.Sprintf("Test Subject %d", 10000+rand.Intn(90000))
	fromName := fmt.Sprintf("%s (test %d)", from.name, 1000+rand.Intn(9000))
	sender := EmailAddress{Email: from.email, Name: from.description}

	{
		var identity Identity
		{
			result, err := s.client.GetIdentities(accountId, []string{}, ctx)
			require.NoError(err)
			require.NotEmpty(result.Payload.List)
			identity = result.Payload.List[0]
		}

		create := EmailChange{
			Keywords:   toBoolMapS("test"),
			Subject:    subject,
			MailboxIds: toBoolMapS(mailboxPerRole[JmapMailboxRoleDrafts].Id),
		}
		var created *Email
		{
			result, err := s.client.CreateEmail(accountId, create, "", ctx)
			require.NoError(err)
			created = result.Payload
			require.NotEmpty(created.Id)
		}

		{
			result, err := s.client.GetEmails(accountId, []string{created.Id}, true, 0, false, false, ctx)
			require.NoError(err)
			require.Len(result.Payload.List, 1)
			require.Empty(result.Payload.NotFound)
			email := result.Payload.List[0]
			require.Equal(created.Id, email.Id)
			require.Len(email.MailboxIds, 1)
			require.Contains(email.MailboxIds, mailboxPerRole[JmapMailboxRoleDrafts].Id)
		}

		update := EmailChange{
			From:       []EmailAddress{{Name: fromName, Email: from.email}},
			To:         []EmailAddress{{Name: to.description, Email: to.email}},
			Cc:         []EmailAddress{{Name: cc.description, Email: cc.email}},
			Sender:     []EmailAddress{sender},
			Keywords:   toBoolMapS("test"),
			Subject:    subject,
			MailboxIds: toBoolMapS(mailboxPerRole[JmapMailboxRoleDrafts].Id),
		}
		var updated *Email
		{
			result, err := s.client.CreateEmail(accountId, update, created.Id, ctx)
			require.NoError(err)
			updated = result.Payload
			require.NotNil(updated)
			require.NotEmpty(updated.Id)
			require.NotEqual(created.Id, updated.Id)
		}

		var updatedMailboxId string
		{
			result, err := s.client.GetEmails(accountId, []string{created.Id, updated.Id}, true, 0, false, false, ctx)
			require.NoError(err)
			require.Len(result.Payload.List, 1)
			require.Len(result.Payload.NotFound, 1)
			email := result.Payload.List[0]
			require.Equal(updated.Id, email.Id)
			require.Len(email.MailboxIds, 1)
			require.Contains(email.MailboxIds, mailboxPerRole[JmapMailboxRoleDrafts].Id)
			require.Equal(result.Payload.NotFound[0], created.Id)
			var ok bool
			updatedMailboxId, ok = firstKey(email.MailboxIds)
			require.True(ok)
		}

		move := MoveMail{
			FromMailboxId: updatedMailboxId,
			ToMailboxId:   mailboxPerRole[JmapMailboxRoleSent].Id,
		}

		var sub EmailSubmission
		{
			result, err := s.client.SubmitEmail(accountId, identity.Id, updated.Id, &move, ctx)
			require.NoError(err)
			sub = result.Payload
			require.NotEmpty(sub.Id)
			require.NotEmpty(sub.ThreadId)
			require.Equal(updated.Id, sub.EmailId)
			require.Equal(identity.Id, sub.IdentityId)
			require.Equal(sub.UndoStatus, UndoStatusPending) // this *might* be fragile: if the server is fast enough, would we get "final" here?
			require.Empty(sub.DsnBlobIds)
			require.Empty(sub.MdnBlobIds)
			require.Equal(from.email, sub.Envelope.MailFrom.Email)
			require.Nil(sub.Envelope.MailFrom.Parameters)
			require.Len(sub.Envelope.RcptTo, 2)
			require.Contains(sub.Envelope.RcptTo, Address{Email: to.email})
			require.Contains(sub.Envelope.RcptTo, Address{Email: cc.email})
			require.NotZero(sub.SendAt)
			require.Len(sub.DeliveryStatus, 2)
			require.Contains(sub.DeliveryStatus, to.email)
			require.Contains(sub.DeliveryStatus, cc.email)
		}

		a := 0
		maxAttempts := 3
		delivery := sub.DeliveryStatus[to.email].Delivered

		for delivery != DeliveredYes {
			require.NotEqual(DeliveredNo, delivery)
			a++
			if a >= maxAttempts {
				break
			}
			time.Sleep(1 * time.Second)

			result, err := s.client.GetEmailSubmissionStatus(accountId, []string{sub.Id}, ctx)
			require.NoError(err)
			require.Empty(result.Payload.NotFound)
			submittedIds := structs.Map(result.Payload.List, func(s EmailSubmission) string { return s.Id })
			require.Contains(submittedIds, sub.Id)
			subs := structs.Index(result.Payload.List, func(s EmailSubmission) string { return s.Id })
			delivery = subs[sub.Id].DeliveryStatus[to.email].Delivered
		}

		require.Contains([]DeliveryStatusDelivered{DeliveredYes, DeliveredUnknown}, delivery)

		for _, r := range []struct {
			user      User
			accountId string
			session   *Session
		}{{to, toAccountId, toSession}, {cc, ccAccountId, ccSession}} {
			rctx := Context{
				Session:        r.session,
				Context:        ctx.Context,
				Logger:         ctx.Logger,
				AcceptLanguage: ctx.AcceptLanguage,
			}
			inboxId := ""
			{
				result, err := s.client.GetAllMailboxes([]string{r.accountId}, rctx)
				require.NoError(err)
				for _, mailbox := range result.Payload[r.accountId] {
					if mailbox.Role == JmapMailboxRoleInbox {
						inboxId = mailbox.Id
						require.Equal(1, mailbox.TotalEmails)
					}
				}
				require.NotEmpty(inboxId, "failed to find the Mailbox with the 'inbox' role for %v", r.user.name)
			}

			result, err := s.client.QueryEmails([]string{r.accountId}, EmailFilterCondition{InMailbox: inboxId}, 0, 0, true, 0, rctx)
			require.NoError(err)
			require.Contains(result.Payload, r.accountId)
			require.Len(result.Payload[r.accountId].Emails, 1)
			received := result.Payload[r.accountId].Emails[0]
			require.Len(received.From, 1)
			require.Equal(from.email, received.From[0].Email)
			require.Equal(fromName, received.From[0].Name)
			require.Len(received.Sender, 1)
			require.Equal(from.email, received.Sender[0].Email)
			require.Equal(from.description, received.Sender[0].Name)
			require.Len(received.To, 1)
			require.Equal(to.email, received.To[0].Email)
			require.Equal(to.description, received.To[0].Name)
			require.Len(received.Cc, 1)
			require.Equal(cc.email, received.Cc[0].Email)
			require.Equal(cc.description, received.Cc[0].Name)
			require.Equal(subject, received.Subject)
		}
	}
}

func matchEmail(t *testing.T, actual Email, expected filledMail, hasBodies bool) {
	require := require.New(t)
	require.Len(actual.MessageId, 1)
	require.Equal(expected.messageId, actual.MessageId[0])
	require.Equal(expected.subject, actual.Subject)
	require.NotEmpty(actual.Preview)
	if hasBodies {
		require.Len(actual.TextBody, 1)
		textBody := actual.TextBody[0]
		partId := textBody.PartId
		require.Contains(actual.BodyValues, partId)
		content := actual.BodyValues[partId].Value
		require.True(strings.Contains(content, actual.Preview), "text body contains preview")
	} else {
		require.Empty(actual.BodyValues)
	}
	require.ElementsMatch(slices.Collect(maps.Keys(actual.Keywords)), expected.keywords)

	{
		list := make([]filledAttachment, len(actual.Attachments))
		for i, a := range actual.Attachments {
			list[i] = filledAttachment{
				name:        a.Name,
				size:        a.Size,
				mimeType:    a.Type,
				disposition: a.Disposition,
			}
			require.NotEmpty(a.BlobId)
			require.NotEmpty(a.PartId)
		}

		require.ElementsMatch(list, expected.attachments)
	}
}

func (s *StalwartTest) findInbox(t *testing.T, accountId string, ctx Context) (string, string) {
	require := require.New(t)
	result, err := s.client.GetAllMailboxes([]string{accountId}, ctx)
	require.NoError(err)
	require.Equal(ctx.Session.State, result.GetSessionState())
	require.Len(result.Payload, 1)
	require.Contains(result.Payload, accountId)
	resp := result.Payload[accountId]

	mailboxesNameByRole := map[string]string{}
	mailboxesUnreadByRole := map[string]int{}
	for _, m := range resp {
		if m.Role != "" {
			mailboxesNameByRole[m.Role] = m.Name
			mailboxesUnreadByRole[m.Role] = m.UnreadEmails
		}
	}
	require.Contains(mailboxesNameByRole, "inbox")
	require.Contains(mailboxesUnreadByRole, "inbox")
	require.Zero(mailboxesUnreadByRole["inbox"])

	inboxId := mailboxId("inbox", resp)
	require.NotEmpty(inboxId)
	inboxFolder := mailboxesNameByRole["inbox"]
	require.NotEmpty(inboxFolder)
	return inboxId, inboxFolder
}

var emailSplitter = regexp.MustCompile("(.+)@(.+)$")

func htmlFormat(body string, msg enmime.MailBuilder) enmime.MailBuilder {
	return msg.HTML([]byte(toHtml(body)))
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

func (s *StalwartTest) fillEmailsWithImap(folder string, count int, empty bool, user User) ([]filledMail, int, error) { //NOSONAR
	to := fmt.Sprintf("%s <%s>", user.description, user.email)
	ccEvery := 2
	bccEvery := 3
	attachmentEvery := 2
	senders := max(count/4, 1)
	maxThreadSize := 6
	maxAttachments := 4

	tlsConfig := &tls.Config{InsecureSkipVerify: true}

	c, err := imapclient.DialTLS(net.JoinHostPort(s.ip, strconv.FormatUint(uint64(s.imapPort), 10)), &imapclient.Options{TLSConfig: tlsConfig})
	if err != nil {
		return nil, 0, err
	}

	defer func(imap *imapclient.Client) {
		err := imap.Close()
		if err != nil {
			log.Fatal(err)
		}
	}(c)

	if err = c.Login(user.name, user.password).Wait(); err != nil {
		return nil, 0, err
	}

	if _, err = c.Select(folder, &imap.SelectOptions{ReadOnly: false}).Wait(); err != nil {
		return nil, 0, err
	}

	if empty {
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
	toAddress := fmt.Sprintf("%s@%s", user.name, domain)
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
	if empty && count != inboxCount {
		return nil, 0, fmt.Errorf("wrong number of emails in the inbox after filling, expecting %v, has %v", count, inboxCount)
	}

	return mails, thread, nil
}
