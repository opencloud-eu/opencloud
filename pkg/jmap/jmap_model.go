package jmap

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

const (
	JmapCore = "urn:ietf:params:jmap:core"
	JmapMail = "urn:ietf:params:jmap:mail"
)

type WellKnownResponse struct {
	Username        string            `json:"username"`
	ApiUrl          string            `json:"apiUrl"`
	PrimaryAccounts map[string]string `json:"primaryAccounts"`
}

type JmapFolder struct {
	Id            string
	Name          string
	Role          string
	TotalEmails   int
	UnreadEmails  int
	TotalThreads  int
	UnreadThreads int
}
type Folders struct {
	Folders []JmapFolder
	state   string
}

type JmapCommandResponse struct {
	MethodResponses [][]any `json:"methodResponses"`
	SessionState    string  `json:"sessionState"`
}

type Email struct {
	Id             string
	MessageId      string
	BlobId         string
	ThreadId       string
	Size           int
	From           string
	Subject        string
	HasAttachments bool
	Received       time.Time
	Preview        string
	Bodies         map[string]string
}

type Emails struct {
	Emails []Email
	State  string
}

type Command string

const (
	EmailGet   Command = "Email/get"
	EmailQuery Command = "Email/query"
	ThreadGet  Command = "Thread/get"
)

type Invocation struct {
	Command    Command
	Parameters any
	Tag        string
}

func (i *Invocation) MarshalJSON() ([]byte, error) {
	arr := []any{string(i.Command), i.Parameters, i.Tag}
	return json.Marshal(arr)
}
func strarr(value any) ([]string, error) {
	switch v := value.(type) {
	case []string:
		return v, nil
	case int:
		return []string{strconv.FormatInt(int64(v), 10)}, nil
	case float32:
		return []string{strconv.FormatFloat(float64(v), 'f', -1, 32)}, nil
	case float64:
		return []string{strconv.FormatFloat(v, 'f', -1, 64)}, nil
	case string:
		return []string{v}, nil
	default:
		return nil, fmt.Errorf("unsupported string array type")
	}
}
func (i *Invocation) UnmarshalJSON(bs []byte) error {
	arr := []any{}
	json.Unmarshal(bs, &arr)
	i.Command = Command(arr[0].(string))
	payload := arr[1].(map[string]any)
	switch i.Command {
	case EmailQuery:
		ids, err := strarr(payload["ids"])
		if err != nil {
			return err
		}
		i.Parameters = EmailQueryResponse{
			AccountId:           payload["accountId"].(string),
			QueryState:          payload["queryState"].(string),
			CanCalculateChanges: payload["canCalculateChanges"].(bool),
			Position:            payload["position"].(int),
			Ids:                 ids,
			Total:               payload["total"].(int),
		}
	default:
		return &json.UnsupportedTypeError{}
	}
	i.Tag = arr[2].(string)
	return nil
}

func NewInvocation(command Command, parameters map[string]any, tag string) Invocation {
	return Invocation{
		Command:    command,
		Parameters: parameters,
		Tag:        tag,
	}
}

type Request struct {
	Using       []string          `json:"using"`
	MethodCalls []Invocation      `json:"methodCalls"`
	CreatedIds  map[string]string `json:"createdIds,omitempty"`
}

func NewRequest(methodCalls ...Invocation) (Request, error) {
	return Request{
		Using:       []string{JmapCore, JmapMail},
		MethodCalls: methodCalls,
		CreatedIds:  nil,
	}, nil
}

// TODO: NewRequestWithIds

type Response struct {
	MethodResponses []Invocation      `json:"methodResponses"`
	CreatedIds      map[string]string `json:"createdIds,omitempty"`
	SessionState    string            `json:"sessionState"`
}

type EmailQueryResponse struct {
	AccountId           string   `json:"accountId"`
	QueryState          string   `json:"queryState"`
	CanCalculateChanges bool     `json:"canCalculateChanges"`
	Position            int      `json:"position"`
	Ids                 []string `json:"ids"`
	Total               int      `json:"total"`
}
type Thread struct {
	ThreadId string `json:"threadId"`
	Id       string `json:"id"`
}
type EmailGetResponse struct {
	AccountId string   `json:"accountId"`
	State     string   `json:"state"`
	List      []Thread `json:"list"`
	NotFound  []any    `json:"notFound"`
}
