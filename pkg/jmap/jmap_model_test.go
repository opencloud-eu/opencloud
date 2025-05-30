package jmap_test

import (
	"encoding/json"
	"testing"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
	"github.com/stretchr/testify/require"
)

func TestRequestSerialization(t *testing.T) {
	require := require.New(t)

	request, err := jmap.NewRequest(
		jmap.NewInvocation(jmap.EmailGet, map[string]any{
			"accountId":  "j",
			"queryState": "aaa",
			"ids":        []string{"a", "b"},
			"total":      1,
		}, "0"),
	)
	require.NoError(err)

	require.Len(request.MethodCalls, 1)
	require.Equal("0", request.MethodCalls[0].Tag)

	requestAsJson, err := json.Marshal(request)
	require.NoError(err)
	require.Equal(`{"using":["urn:ietf:params:jmap:core","urn:ietf:params:jmap:mail"],"methodCalls":[["Email/get",{"accountId":"j","ids":["a","b"],"queryState":"aaa","total":1},"0"]]}`, string(requestAsJson))
}

const mails2 = `{"methodResponses":[
   ["Email/query",{
     "accountId":"j",
     "queryState":"sqcakzewfqdk7oay",
     "canCalculateChanges":true,
     "position":0,
     "ids":["fmaaaabh"],
     "total":1
   }, "0"],
   ["Email/get", {
     "accountId":"j",
     "state":"sqcakzewfqdk7oay",
     "list":[
       {
         "threadId":"bl",
         "id":"fmaaaabh"
       }
     ],
     "notFound":[]
   }, "1"],
   ["Thread/get",{
     "accountId":"j",
     "state":"sqcakzewfqdk7oay",
     "list":[
       {
         "id":"bl",
         "emailIds":["fmaaaabh"]
       }
     ],
     "notFound":[]
   }, "2"],
   ["Email/get",{
     "accountId":"j",
     "state":"sqcakzewfqdk7oay",
     "list":[
       {
         "threadId":"bl",
         "mailboxIds":{"a":true},
         "keywords":{},
         "hasAttachment":false,
         "from":[
           {"name":"current generally", "email":"current.generally@example.com"}
         ],
         "subject":"eros auctor proin",
         "receivedAt":"2025-04-30T09:47:44Z",
         "size":15423,
         "preview":"Lorem ipsum dolor sit amet consectetur adipiscing elit sed urna tristique himenaeos eu a mattis laoreet aliquet enim. Magnis est facilisis nibh nisl vitae nisi mauris nostra velit donec erat pellentesque sagittis ligula turpis suscipit ultricies. Morbi ...",
         "id":"fmaaaabh"
       }
     ],
     "notFound":[]
   }, "3"]
   ], "sessionState":"3e25b2a0"
 }`

func TestResponseDeserialization(t *testing.T) {
	require := require.New(t)

	var response jmap.Response
	err := json.Unmarshal([]byte(mails2), &response)
	require.NoError(err)

	t.Log(response)

	require.Len(response.MethodResponses, 4)
	require.Nil(response.CreatedIds)
	require.Equal("3e25b2a0", response.SessionState)
	require.Equal(jmap.EmailQuery, response.MethodResponses[0].Command)
	require.Equal(map[string]any{
		"accountId":           "j",
		"queryState":          "sqcakzewfqdk7oay",
		"canCalculateChanges": true,
		"position":            0.0,
		"ids":                 []any{"fmaaaabh"},
		"total":               1.0,
	}, response.MethodResponses[0].Parameters)

	require.Equal("0", response.MethodResponses[0].Tag)
	require.Equal(jmap.EmailGet, response.MethodResponses[1].Command)
	require.Equal("1", response.MethodResponses[1].Tag)
	require.Equal(jmap.ThreadGet, response.MethodResponses[2].Command)
	require.Equal("2", response.MethodResponses[2].Tag)
	require.Equal(jmap.EmailGet, response.MethodResponses[3].Command)
	require.Equal("3", response.MethodResponses[3].Tag)

}
