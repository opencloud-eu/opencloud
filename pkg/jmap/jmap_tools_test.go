package jmap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeserializeMailboxGetResponse(t *testing.T) {
	require := require.New(t)
	jsonBytes, err := serveTestFile(t, "mailboxes1.json")
	require.NoError(err)
	var data Response
	err = json.Unmarshal(jsonBytes, &data)
	require.NoError(err)
	require.Empty(data.CreatedIds)
	require.Equal("3e25b2a0", data.SessionState)
	require.Len(data.MethodResponses, 1)
	resp := data.MethodResponses[0]
	require.Equal(MailboxGet, resp.Command)
	require.Equal("0", resp.Tag)
	require.IsType(MailboxGetResponse{}, resp.Parameters)
	mgr := resp.Parameters.(MailboxGetResponse)
	require.Equal("cs", mgr.AccountId)
	require.Len(mgr.List, 5)
	require.Equal("n", mgr.State)
	require.Empty(mgr.NotFound)
	var rights = []string{"mayReadItems", "mayAddItems", "mayRemoveItems", "maySetSeen", "maySetKeywords", "mayCreateChild", "mayRename", "mayDelete", "maySubmit"}
	var folders = []struct {
		id     string
		name   string
		role   string
		total  int
		unread int
	}{
		{"a", "Inbox", "inbox", 10, 8},
		{"b", "Deleted Items", "trash", 20, 0},
		{"c", "Junk Mail", "junk", 0, 0},
		{"d", "Drafts", "drafts", 0, 0},
		{"e", "Sent Items", "sent", 0, 0},
	}
	for i, expected := range folders {
		folder := mgr.List[i]
		require.Equal(expected.id, folder.Id)
		require.Equal(expected.name, folder.Name)
		require.Equal(expected.role, folder.Role)
		require.Equal(expected.total, folder.TotalEmails)
		require.Equal(expected.total, folder.TotalThreads)
		require.Equal(expected.unread, folder.UnreadEmails)
		require.Equal(expected.unread, folder.UnreadThreads)
		require.Empty(folder.ParentId)
		require.Zero(folder.SortOrder)
		require.True(folder.IsSubscribed)

		for _, right := range rights {
			require.Contains(folder.MyRights, right)
			require.True(folder.MyRights[right])
		}
	}
}

func TestDeserializeEmailGetResponse(t *testing.T) {
	require := require.New(t)
	jsonBytes, err := serveTestFile(t, "mails1.json")
	require.NoError(err)
	var data Response
	err = json.Unmarshal(jsonBytes, &data)
	require.NoError(err)
	require.Empty(data.CreatedIds)
	require.Equal("3e25b2a0", data.SessionState)
	require.Len(data.MethodResponses, 2)
	resp := data.MethodResponses[1]
	require.Equal(EmailGet, resp.Command)
	require.Equal("1", resp.Tag)
	require.IsType(EmailGetResponse{}, resp.Parameters)
	egr := resp.Parameters.(EmailGetResponse)
	require.Equal("d", egr.AccountId)
	require.Len(egr.List, 3)
	require.Equal("suqmq", egr.State)
	require.Empty(egr.NotFound)
	email := egr.List[0]
	require.Equal("moyaaaddw", email.Id)
	require.Equal("cbejozsk1fgcviw7thwzsvtgmf1ep0a3izjoimj02jmtsunpeuwmsaya1yma", email.BlobId)
}

func TestUnknown(t *testing.T) {
	require := require.New(t)

	const text = `{
	"subject": "aaa",
	"bodyStructure": {
	  "type": "a",
	  "partId": "b",
	  "header:x": "yz",
	  "header:a": "bc"
	}
	}`

	var target EmailCreate
	err := json.Unmarshal([]byte(text), &target)

	require.NoError(err)
	require.Equal("aaa", target.Subject)
	bs := target.BodyStructure
	require.Equal("a", bs.Type)
	require.Equal("b", bs.PartId)
	require.Contains(bs.Other, "header:x")
	require.Equal(bs.Other["header:x"], "yz")
	require.Contains(bs.Other, "header:a")
	require.Equal(bs.Other["header:a"], "bc")

	result, err := json.Marshal(target)
	require.NoError(err)
	require.Equal(`{"subject":"aaa","bodyStructure":{"type":"a","partId":"b","header:a":"bc","header:x":"yz"}}`, string(result))
}
