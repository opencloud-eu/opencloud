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

func (s *StalwartTest) findInbox(t *testing.T, accountId string) (string, string) {
	require := require.New(t)
	respByAccountId, sessionState, _, _, err := s.client.GetAllMailboxes([]string{accountId}, s.session, s.ctx, s.logger, "")
	require.NoError(err)
	require.Equal(s.session.State, sessionState)
	require.Len(respByAccountId, 1)
	require.Contains(respByAccountId, accountId)
	resp := respByAccountId[accountId]

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

func TestEmails(t *testing.T) {
	if skip(t) {
		return
	}

	count := 15 + rand.Intn(20)

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	accountId := s.session.PrimaryAccounts.Mail

	inboxId, inboxFolder := s.findInbox(t, accountId)

	var threads int = 0
	var mails []filledMail = nil
	{
		mails, threads, err = s.fillEmailsWithImap(inboxFolder, count)
		require.NoError(err)
	}
	mailsByMessageId := structs.Index(mails, func(mail filledMail) string { return mail.messageId })

	{
		{
			resp, sessionState, _, _, err := s.client.GetAllIdentities(accountId, s.session, s.ctx, s.logger, "")
			require.NoError(err)
			require.Equal(s.session.State, sessionState)
			require.Len(resp, 1)
			require.Equal(s.userEmail, resp[0].Email)
			require.Equal(s.userPersonName, resp[0].Name)
		}

		{
			respByAccountId, sessionState, _, _, err := s.client.GetAllMailboxes([]string{accountId}, s.session, s.ctx, s.logger, "")
			require.NoError(err)
			require.Equal(s.session.State, sessionState)
			require.Len(respByAccountId, 1)
			require.Contains(respByAccountId, accountId)
			resp := respByAccountId[accountId]
			mailboxesUnreadByRole := map[string]int{}
			for _, m := range resp {
				if m.Role != "" {
					mailboxesUnreadByRole[m.Role] = m.UnreadEmails
				}
			}
			require.LessOrEqual(mailboxesUnreadByRole["inbox"], count)
		}

		{
			resp, sessionState, _, _, err := s.client.GetAllEmailsInMailbox(accountId, s.session, s.ctx, s.logger, "", inboxId, 0, 0, true, false, 0, true)
			require.NoError(err)
			require.Equal(s.session.State, sessionState)

			require.Equalf(threads, len(resp.Emails), "the number of collapsed emails in the inbox is expected to be %v, but is actually %v", threads, len(resp.Emails))
			for _, e := range resp.Emails {
				require.Len(e.MessageId, 1)
				expectation, ok := mailsByMessageId[e.MessageId[0]]
				require.True(ok)
				matchEmail(t, e, expectation, false)
			}
		}

		{
			resp, sessionState, _, _, err := s.client.GetAllEmailsInMailbox(accountId, s.session, s.ctx, s.logger, "", inboxId, 0, 0, false, false, 0, true)
			require.NoError(err)
			require.Equal(s.session.State, sessionState)

			require.Equalf(count, len(resp.Emails), "the number of emails in the inbox is expected to be %v, but is actually %v", count, len(resp.Emails))
			for _, e := range resp.Emails {
				require.Len(e.MessageId, 1)
				expectation, ok := mailsByMessageId[e.MessageId[0]]
				require.True(ok)
				matchEmail(t, e, expectation, false)
			}
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
