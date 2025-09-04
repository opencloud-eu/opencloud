package imap

import (
	"unsafe"

	"github.com/emersion/go-imap/v2/internal/imapnum"
)

// NumSet is a set of numbers identifying messages. NumSet is either a SeqSet
// or a UIDSet.
type NumSet interface {
	// String returns the IMAP representation of the message number set.
	String() string
	// Dynamic returns true if the set contains "*" or "n:*" ranges or if the
	// set represents the special SEARCHRES marker.
	Dynamic() bool

	numSet() imapnum.Set
}

var (
	_ NumSet = SeqSet(nil)
	_ NumSet = UIDSet(nil)
)

// SeqSet is a set of message sequence numbers.
type SeqSet []SeqRange

// SeqSetNum returns a new SeqSet containing the specified sequence numbers.
func SeqSetNum(nums ...uint32) SeqSet {
	var s SeqSet
	s.AddNum(nums...)
	return s
}

func (s *SeqSet) numSetPtr() *imapnum.Set {
	return (*imapnum.Set)(unsafe.Pointer(s))
}

func (s SeqSet) numSet() imapnum.Set {
	return *s.numSetPtr()
}

func (s SeqSet) String() string {
	return s.numSet().String()
}

func (s SeqSet) Dynamic() bool {
	return s.numSet().Dynamic()
}

// Contains returns true if the non-zero sequence number num is contained in
// the set.
func (s *SeqSet) Contains(num uint32) bool {
	return s.numSet().Contains(num)
}

// Nums returns a slice of all sequence numbers contained in the set.
func (s *SeqSet) Nums() ([]uint32, bool) {
	return s.numSet().Nums()
}

// AddNum inserts new sequence numbers into the set. The value 0 represents "*".
func (s *SeqSet) AddNum(nums ...uint32) {
	s.numSetPtr().AddNum(nums...)
}

// AddRange inserts a new range into the set.
func (s *SeqSet) AddRange(start, stop uint32) {
	s.numSetPtr().AddRange(start, stop)
}

// AddSet inserts all sequence numbers from other into s.
func (s *SeqSet) AddSet(other SeqSet) {
	s.numSetPtr().AddSet(other.numSet())
}

// SeqRange is a range of message sequence numbers.
type SeqRange struct {
	Start, Stop uint32
}

// UIDSet is a set of message UIDs.
type UIDSet []UIDRange

// UIDSetNum returns a new UIDSet containing the specified UIDs.
func UIDSetNum(uids ...UID) UIDSet {
	var s UIDSet
	s.AddNum(uids...)
	return s
}

func (s *UIDSet) numSetPtr() *imapnum.Set {
	return (*imapnum.Set)(unsafe.Pointer(s))
}

func (s UIDSet) numSet() imapnum.Set {
	return *s.numSetPtr()
}

func (s UIDSet) String() string {
	if IsSearchRes(s) {
		return "$"
	}
	return s.numSet().String()
}

func (s UIDSet) Dynamic() bool {
	return s.numSet().Dynamic() || IsSearchRes(s)
}

// Contains returns true if the non-zero UID uid is contained in the set.
func (s UIDSet) Contains(uid UID) bool {
	return s.numSet().Contains(uint32(uid))
}

// Nums returns a slice of all UIDs contained in the set.
func (s UIDSet) Nums() ([]UID, bool) {
	nums, ok := s.numSet().Nums()
	return uidListFromNumList(nums), ok
}

// AddNum inserts new UIDs into the set. The value 0 represents "*".
func (s *UIDSet) AddNum(uids ...UID) {
	s.numSetPtr().AddNum(numListFromUIDList(uids)...)
}

// AddRange inserts a new range into the set.
func (s *UIDSet) AddRange(start, stop UID) {
	s.numSetPtr().AddRange(uint32(start), uint32(stop))
}

// AddSet inserts all UIDs from other into s.
func (s *UIDSet) AddSet(other UIDSet) {
	s.numSetPtr().AddSet(other.numSet())
}

// UIDRange is a range of message UIDs.
type UIDRange struct {
	Start, Stop UID
}

func numListFromUIDList(uids []UID) []uint32 {
	return *(*[]uint32)(unsafe.Pointer(&uids))
}

func uidListFromNumList(nums []uint32) []UID {
	return *(*[]UID)(unsafe.Pointer(&nums))
}
