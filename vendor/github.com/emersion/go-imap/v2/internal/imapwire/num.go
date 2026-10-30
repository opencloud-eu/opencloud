package imapwire

import (
	"unsafe"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapnum"
)

type NumKind int

const (
	NumKindSeq NumKind = iota + 1
	NumKindUID
)

func seqSetFromNumSet(s imapnum.Set) imap.SeqSet {
	return *(*imap.SeqSet)(unsafe.Pointer(&s))
}

func uidSetFromNumSet(s imapnum.Set) imap.UIDSet {
	return *(*imap.UIDSet)(unsafe.Pointer(&s))
}

func NumSetKind(numSet imap.NumSet) NumKind {
	switch numSet.(type) {
	case imap.SeqSet:
		return NumKindSeq
	case imap.UIDSet:
		return NumKindUID
	default:
		panic("imap: invalid NumSet type")
	}
}

func ParseSeqSet(s string) (imap.SeqSet, error) {
	numSet, err := imapnum.ParseSet(s)
	return seqSetFromNumSet(numSet), err
}
