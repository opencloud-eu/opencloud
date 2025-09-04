package imap

// ListOptions contains options for the LIST command.
type ListOptions struct {
	SelectSubscribed     bool
	SelectRemote         bool
	SelectRecursiveMatch bool // requires SelectSubscribed to be set
	SelectSpecialUse     bool // requires SPECIAL-USE

	ReturnSubscribed bool
	ReturnChildren   bool
	ReturnStatus     *StatusOptions // requires IMAP4rev2 or LIST-STATUS
	ReturnSpecialUse bool           // requires SPECIAL-USE
}

// ListData is the mailbox data returned by a LIST command.
type ListData struct {
	Attrs   []MailboxAttr
	Delim   rune
	Mailbox string

	// Extended data
	ChildInfo *ListDataChildInfo
	OldName   string
	Status    *StatusData
}

type ListDataChildInfo struct {
	Subscribed bool
}
