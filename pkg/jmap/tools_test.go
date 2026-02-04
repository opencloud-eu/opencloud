package jmap

import (
	"encoding/json"
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
