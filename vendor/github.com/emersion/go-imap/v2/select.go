package imap

// SelectOptions contains options for the SELECT or EXAMINE command.
type SelectOptions struct {
	ReadOnly  bool
	CondStore bool // requires CONDSTORE
}

// SelectData is the data returned by a SELECT command.
//
// In the old RFC 2060, PermanentFlags, UIDNext and UIDValidity are optional.
type SelectData struct {
	// Flags defined for this mailbox
	Flags []Flag
	// Flags that the client can change permanently
	PermanentFlags []Flag
	// Number of messages in this mailbox (aka. "EXISTS")
	NumMessages uint32
	UIDNext     UID
	UIDValidity uint32

	List *ListData // requires IMAP4rev2

	HighestModSeq uint64 // requires CONDSTORE
}
