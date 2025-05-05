package jmap

type WellKnownJmap struct {
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
type JmapFolders struct {
	Folders []JmapFolder
	state   string
}

type JmapCommandResponse struct {
	MethodResponses [][]any `json:"methodResponses"`
	SessionState    string  `json:"sessionState"`
}

type Emails struct {
	Emails []Email
	State  string
}
