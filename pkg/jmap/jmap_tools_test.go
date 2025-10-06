package jmap

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDeserializeMailboxGetResponse(t *testing.T) {
	require := require.New(t)
	jsonBytes, _, jmapErr := serveTestFile(t, "mailboxes1.json")
	require.NoError(jmapErr)
	var data Response
	err := json.Unmarshal(jsonBytes, &data)
	require.NoError(err)
	require.Empty(data.CreatedIds)
	require.Equal("3e25b2a0", data.SessionState)
	require.Len(data.MethodResponses, 1)
	resp := data.MethodResponses[0]
	require.Equal(CommandMailboxGet, resp.Command)
	require.Equal("0", resp.Tag)
	require.IsType(MailboxGetResponse{}, resp.Parameters)
	mgr := resp.Parameters.(MailboxGetResponse)
	require.Equal("cs", mgr.AccountId)
	require.Len(mgr.List, 5)
	require.Equal("n", mgr.State)
	require.Empty(mgr.NotFound)
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

		require.True(folder.MyRights.MayReadItems)
		require.True(folder.MyRights.MayAddItems)
		require.True(folder.MyRights.MayRemoveItems)
		require.True(folder.MyRights.MaySetSeen)
		require.True(folder.MyRights.MaySetKeywords)
		require.True(folder.MyRights.MayCreateChild)
		require.True(folder.MyRights.MayRename)
		require.True(folder.MyRights.MayDelete)
		require.True(folder.MyRights.MaySubmit)
	}
}

func TestDeserializeEmailGetResponse(t *testing.T) {
	require := require.New(t)
	jsonBytes, _, jmapErr := serveTestFile(t, "mails1.json")
	require.NoError(jmapErr)
	var data Response
	err := json.Unmarshal(jsonBytes, &data)
	require.NoError(err)
	require.Empty(data.CreatedIds)
	require.Equal("3e25b2a0", data.SessionState)
	require.Len(data.MethodResponses, 2)
	resp := data.MethodResponses[1]
	require.Equal(CommandEmailGet, resp.Command)
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

func TestUnmarshallingUnknown(t *testing.T) {
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
}

func TestMarshallingUnknown(t *testing.T) {
	require := require.New(t)

	source := EmailCreate{
		Subject: "aaa",
		BodyStructure: EmailBodyStructure{
			Type:   "a",
			PartId: "b",
			Other: map[string]any{
				"header:x": "yz",
				"header:a": "bc",
			},
		},
	}

	result, err := json.Marshal(source)
	require.NoError(err)
	require.Equal(`{"subject":"aaa","bodyStructure":{"header:a":"bc","header:x":"yz","partId":"b","type":"a"}}`, string(result))
}

func TestUnmarshallingError(t *testing.T) {
	require := require.New(t)

	responseBody := `{"methodResponses":[["error",{"type":"forbidden","description":"You do not have access to account a"},"a:0"]],"sessionState":"3e25b2a0"}`
	var response Response
	err := json.Unmarshal([]byte(responseBody), &response)
	require.NoError(err)
	require.Len(response.MethodResponses, 1)
	require.Equal(ErrorCommand, response.MethodResponses[0].Command)
	require.Equal("a:0", response.MethodResponses[0].Tag)
	require.IsType(ErrorResponse{}, response.MethodResponses[0].Parameters)
	er, _ := response.MethodResponses[0].Parameters.(ErrorResponse)
	require.Equal("forbidden", er.Type)
	require.Equal("You do not have access to account a", er.Description)
}
