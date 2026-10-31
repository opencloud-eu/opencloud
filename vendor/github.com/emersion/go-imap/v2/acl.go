package imap

import (
	"fmt"
	"strings"
)

// IMAP4 ACL extension (RFC 2086)

// Right describes a set of operations controlled by the IMAP ACL extension.
type Right byte

const (
	// Standard rights
	RightLookup     = Right('l') // mailbox is visible to LIST/LSUB commands
	RightRead       = Right('r') // SELECT the mailbox, perform CHECK, FETCH, PARTIAL, SEARCH, COPY from mailbox
	RightSeen       = Right('s') // keep seen/unseen information across sessions (STORE SEEN flag)
	RightWrite      = Right('w') // STORE flags other than SEEN and DELETED
	RightInsert     = Right('i') // perform APPEND, COPY into mailbox
	RightPost       = Right('p') // send mail to submission address for mailbox, not enforced by IMAP4 itself
	RightCreate     = Right('c') // CREATE new sub-mailboxes in any implementation-defined hierarchy
	RightDelete     = Right('d') // STORE DELETED flag, perform EXPUNGE
	RightAdminister = Right('a') // perform SETACL
)

// RightSetAll contains all standard rights.
var RightSetAll = RightSet("lrswipcda")

// RightsIdentifier is an ACL identifier.
type RightsIdentifier string

// RightsIdentifierAnyone is the universal identity (matches everyone).
const RightsIdentifierAnyone = RightsIdentifier("anyone")

// NewRightsIdentifierUsername returns a rights identifier referring to a
// username, checking for reserved values.
func NewRightsIdentifierUsername(username string) (RightsIdentifier, error) {
	if username == string(RightsIdentifierAnyone) || strings.HasPrefix(username, "-") {
		return "", fmt.Errorf("imap: reserved rights identifier")
	}
	return RightsIdentifier(username), nil
}

// RightModification indicates how to mutate a right set.
type RightModification byte

const (
	RightModificationReplace = RightModification(0)
	RightModificationAdd     = RightModification('+')
	RightModificationRemove  = RightModification('-')
)

// A RightSet is a set of rights.
type RightSet []Right

// String returns a string representation of the right set.
func (r RightSet) String() string {
	return string(r)
}

// Add returns a new right set containing rights from both sets.
func (r RightSet) Add(rights RightSet) RightSet {
	newRights := make(RightSet, len(r), len(r)+len(rights))
	copy(newRights, r)

	for _, right := range rights {
		if !strings.ContainsRune(string(r), rune(right)) {
			newRights = append(newRights, right)
		}
	}

	return newRights
}

// Remove returns a new right set containing all rights in r except these in
// the provided set.
func (r RightSet) Remove(rights RightSet) RightSet {
	newRights := make(RightSet, 0, len(r))

	for _, right := range r {
		if !strings.ContainsRune(string(rights), rune(right)) {
			newRights = append(newRights, right)
		}
	}

	return newRights
}

// Equal returns true if both right sets contain exactly the same rights.
func (rs1 RightSet) Equal(rs2 RightSet) bool {
	for _, r := range rs1 {
		if !strings.ContainsRune(string(rs2), rune(r)) {
			return false
		}
	}

	for _, r := range rs2 {
		if !strings.ContainsRune(string(rs1), rune(r)) {
			return false
		}
	}

	return true
}
