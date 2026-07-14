package imap

// NamespaceData is the data returned by the NAMESPACE command.
type NamespaceData struct {
	Personal []NamespaceDescriptor
	Other    []NamespaceDescriptor
	Shared   []NamespaceDescriptor
}

// NamespaceDescriptor describes a namespace.
type NamespaceDescriptor struct {
	Prefix string
	Delim  rune
}
