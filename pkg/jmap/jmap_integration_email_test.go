package jmap

import (
	"maps"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/opencloud-eu/opencloud/pkg/structs"
)

func TestEmails(t *testing.T) {
	if skip(t) {
		return
	}

	count := 15 + rand.IntN(20)

	require := require.New(t)

	s, err := newStalwartTest(t)
	require.NoError(err)
	defer s.Close()

	accountId := s.session.PrimaryAccounts.Mail

	var inboxFolder string
	var inboxId string
	{
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

		inboxId = mailboxId("inbox", resp)
		require.NotEmpty(inboxId)
		inboxFolder = mailboxesNameByRole["inbox"]
		require.NotEmpty(inboxFolder)
	}

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
