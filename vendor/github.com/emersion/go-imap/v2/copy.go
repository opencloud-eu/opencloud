package imap

// CopyData is the data returned by a COPY command.
type CopyData struct {
	// requires UIDPLUS or IMAP4rev2
	UIDValidity uint32
	SourceUIDs  UIDSet
	DestUIDs    UIDSet
}
