package jmap

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

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

func TestSquashKeyedStates(t *testing.T) {
	require := require.New(t)

	result := squashKeyedStates(map[string]State{
		"a": "aaa",
		"b": "bbb",
		"c": "ccc",
	})
	require.Equal("a:aaa,b:bbb,c:ccc", string(result))
}

func TestInvocationMarshalling(t *testing.T) {
	tag := strconv.Itoa(1000 + rand.Intn(1000))
	accountId := fmt.Sprintf("a%d", 100+rand.Intn(100))
	inv := invocation(IdentityGetCommand{AccountId: AccountId(accountId), Ids: []string{"x", "y", "z"}}, tag)
	b, err := json.Marshal(inv)
	require.NoError(t, err)
	require.Equal(t, fmt.Sprintf(`{"Command":"Identity/get","Parameters":{"accountId":"%s","ids":["x","y","z"]},"Tag":"%s"}`, accountId, tag), string(b))
}
