package internal

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

const (
	DateTimeLayout = "_2-Jan-2006 15:04:05 -0700"
	DateLayout     = "2-Jan-2006"
)

const FlagRecent imap.Flag = "\\Recent" // removed in IMAP4rev2

func DecodeDateTime(dec *imapwire.Decoder) (time.Time, error) {
	var s string
	if !dec.Quoted(&s) {
		return time.Time{}, nil
	}
	t, err := time.Parse(DateTimeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("in date-time: %v", err) // TODO: use imapwire.DecodeExpectError?
	}
	return t, err
}

func ExpectDateTime(dec *imapwire.Decoder) (time.Time, error) {
	t, err := DecodeDateTime(dec)
	if err != nil {
		return t, err
	}
	if !dec.Expect(!t.IsZero(), "date-time") {
		return t, dec.Err()
	}
	return t, nil
}

func ExpectDate(dec *imapwire.Decoder) (time.Time, error) {
	var s string
	if !dec.ExpectAString(&s) {
		return time.Time{}, dec.Err()
	}
	t, err := time.Parse(DateLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("in date: %v", err) // use imapwire.DecodeExpectError?
	}
	return t, nil
}

func ExpectFlagList(dec *imapwire.Decoder) ([]imap.Flag, error) {
	var flags []imap.Flag
	err := dec.ExpectList(func() error {
		// Some servers start the list with a space, so we need to skip it
		// https://github.com/emersion/go-imap/pull/633
		dec.SP()

		flag, err := ExpectFlag(dec)
		if err != nil {
			return err
		}
		flags = append(flags, flag)
		return nil
	})
	return flags, err
}

func ExpectFlag(dec *imapwire.Decoder) (imap.Flag, error) {
	isSystem := dec.Special('\\')
	if isSystem && dec.Special('*') {
		return imap.FlagWildcard, nil // flag-perm
	}
	var name string
	if !dec.ExpectAtom(&name) {
		return "", fmt.Errorf("in flag: %w", dec.Err())
	}
	if isSystem {
		name = "\\" + name
	}
	return canonicalFlag(name), nil
}

func ExpectMailboxAttrList(dec *imapwire.Decoder) ([]imap.MailboxAttr, error) {
	var attrs []imap.MailboxAttr
	err := dec.ExpectList(func() error {
		attr, err := ExpectMailboxAttr(dec)
		if err != nil {
			return err
		}
		attrs = append(attrs, attr)
		return nil
	})
	return attrs, err
}

func ExpectMailboxAttr(dec *imapwire.Decoder) (imap.MailboxAttr, error) {
	flag, err := ExpectFlag(dec)
	return canonicalMailboxAttr(string(flag)), err
}

var (
	canonOnce        sync.Once
	canonFlag        map[string]imap.Flag
	canonMailboxAttr map[string]imap.MailboxAttr
)

func canonInit() {
	flags := []imap.Flag{
		imap.FlagSeen,
		imap.FlagAnswered,
		imap.FlagFlagged,
		imap.FlagDeleted,
		imap.FlagDraft,
		imap.FlagForwarded,
		imap.FlagMDNSent,
		imap.FlagJunk,
		imap.FlagNotJunk,
		imap.FlagPhishing,
		imap.FlagImportant,
	}
	mailboxAttrs := []imap.MailboxAttr{
		imap.MailboxAttrNonExistent,
		imap.MailboxAttrNoInferiors,
		imap.MailboxAttrNoSelect,
		imap.MailboxAttrHasChildren,
		imap.MailboxAttrHasNoChildren,
		imap.MailboxAttrMarked,
		imap.MailboxAttrUnmarked,
		imap.MailboxAttrSubscribed,
		imap.MailboxAttrRemote,
		imap.MailboxAttrAll,
		imap.MailboxAttrArchive,
		imap.MailboxAttrDrafts,
		imap.MailboxAttrFlagged,
		imap.MailboxAttrJunk,
		imap.MailboxAttrSent,
		imap.MailboxAttrTrash,
		imap.MailboxAttrImportant,
	}

	canonFlag = make(map[string]imap.Flag)
	for _, flag := range flags {
		canonFlag[strings.ToLower(string(flag))] = flag
	}

	canonMailboxAttr = make(map[string]imap.MailboxAttr)
	for _, attr := range mailboxAttrs {
		canonMailboxAttr[strings.ToLower(string(attr))] = attr
	}
}

func canonicalFlag(s string) imap.Flag {
	canonOnce.Do(canonInit)
	if flag, ok := canonFlag[strings.ToLower(s)]; ok {
		return flag
	}
	return imap.Flag(s)
}

func canonicalMailboxAttr(s string) imap.MailboxAttr {
	canonOnce.Do(canonInit)
	if attr, ok := canonMailboxAttr[strings.ToLower(s)]; ok {
		return attr
	}
	return imap.MailboxAttr(s)
}
