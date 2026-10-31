package imapclient

import (
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/internal"
	"github.com/emersion/go-imap/v2/internal/imapwire"
)

func returnSearchOptions(options *imap.SearchOptions) []string {
	if options == nil {
		return nil
	}

	m := map[string]bool{
		"MIN":   options.ReturnMin,
		"MAX":   options.ReturnMax,
		"ALL":   options.ReturnAll,
		"COUNT": options.ReturnCount,
	}

	var l []string
	for k, ret := range m {
		if ret {
			l = append(l, k)
		}
	}
	return l
}

func (c *Client) search(numKind imapwire.NumKind, criteria *imap.SearchCriteria, options *imap.SearchOptions) *SearchCommand {
	// The IMAP4rev2 SEARCH charset defaults to UTF-8. When UTF8=ACCEPT is
	// enabled, specifying any CHARSET is invalid. For IMAP4rev1 the default is
	// undefined and only US-ASCII support is required. What's more, some
	// servers completely reject the CHARSET keyword. So, let's check if we
	// actually have UTF-8 strings in the search criteria before using that.
	// TODO: there might be a benefit in specifying CHARSET UTF-8 for IMAP4rev1
	// servers even if we only send ASCII characters: the server then must
	// decode encoded headers and Content-Transfer-Encoding before matching the
	// criteria.
	var charset string
	if !c.Caps().Has(imap.CapIMAP4rev2) && !c.enabled.Has(imap.CapUTF8Accept) && !searchCriteriaIsASCII(criteria) {
		charset = "UTF-8"
	}

	var all imap.NumSet
	switch numKind {
	case imapwire.NumKindSeq:
		all = imap.SeqSet(nil)
	case imapwire.NumKindUID:
		all = imap.UIDSet(nil)
	}

	cmd := &SearchCommand{}
	cmd.data.All = all
	enc := c.beginCommand(uidCmdName("SEARCH", numKind), cmd)
	if returnOpts := returnSearchOptions(options); len(returnOpts) > 0 {
		enc.SP().Atom("RETURN").SP().List(len(returnOpts), func(i int) {
			enc.Atom(returnOpts[i])
		})
	}
	enc.SP()
	if charset != "" {
		enc.Atom("CHARSET").SP().Atom(charset).SP()
	}
	writeSearchKey(enc.Encoder, criteria)
	enc.end()
	return cmd
}

// Search sends a SEARCH command.
func (c *Client) Search(criteria *imap.SearchCriteria, options *imap.SearchOptions) *SearchCommand {
	return c.search(imapwire.NumKindSeq, criteria, options)
}

// UIDSearch sends a UID SEARCH command.
func (c *Client) UIDSearch(criteria *imap.SearchCriteria, options *imap.SearchOptions) *SearchCommand {
	return c.search(imapwire.NumKindUID, criteria, options)
}

func (c *Client) handleSearch() error {
	cmd := findPendingCmdByType[*SearchCommand](c)
	for c.dec.SP() {
		if c.dec.Special('(') {
			var name string
			if !c.dec.ExpectAtom(&name) || !c.dec.ExpectSP() {
				return c.dec.Err()
			} else if strings.ToUpper(name) != "MODSEQ" {
				return fmt.Errorf("in search-sort-mod-seq: expected %q, got %q", "MODSEQ", name)
			}
			var modSeq uint64
			if !c.dec.ExpectModSeq(&modSeq) || !c.dec.ExpectSpecial(')') {
				return c.dec.Err()
			}
			if cmd != nil {
				cmd.data.ModSeq = modSeq
			}
			break
		}

		var num uint32
		if !c.dec.ExpectNumber(&num) {
			return c.dec.Err()
		}
		if cmd != nil {
			switch all := cmd.data.All.(type) {
			case imap.SeqSet:
				all.AddNum(num)
				cmd.data.All = all
			case imap.UIDSet:
				all.AddNum(imap.UID(num))
				cmd.data.All = all
			}
		}
	}
	return nil
}

func (c *Client) handleESearch() error {
	if !c.dec.ExpectSP() {
		return c.dec.Err()
	}
	tag, data, err := readESearchResponse(c.dec)
	if err != nil {
		return err
	}
	cmd := c.findPendingCmdFunc(func(anyCmd command) bool {
		cmd, ok := anyCmd.(*SearchCommand)
		if !ok {
			return false
		}
		if tag != "" {
			return cmd.tag == tag
		} else {
			return true
		}
	})
	if cmd != nil {
		cmd := cmd.(*SearchCommand)
		cmd.data = *data
	}
	return nil
}

// SearchCommand is a SEARCH command.
type SearchCommand struct {
	commandBase
	data imap.SearchData
}

func (cmd *SearchCommand) Wait() (*imap.SearchData, error) {
	return &cmd.data, cmd.wait()
}

func writeSearchKey(enc *imapwire.Encoder, criteria *imap.SearchCriteria) {
	firstItem := true
	encodeItem := func() *imapwire.Encoder {
		if !firstItem {
			enc.SP()
		}
		firstItem = false
		return enc
	}

	for _, seqSet := range criteria.SeqNum {
		encodeItem().NumSet(seqSet)
	}
	for _, uidSet := range criteria.UID {
		encodeItem().Atom("UID").SP().NumSet(uidSet)
	}

	if !criteria.Since.IsZero() && !criteria.Before.IsZero() && criteria.Before.Sub(criteria.Since) == 24*time.Hour {
		encodeItem().Atom("ON").SP().String(criteria.Since.Format(internal.DateLayout))
	} else {
		if !criteria.Since.IsZero() {
			encodeItem().Atom("SINCE").SP().String(criteria.Since.Format(internal.DateLayout))
		}
		if !criteria.Before.IsZero() {
			encodeItem().Atom("BEFORE").SP().String(criteria.Before.Format(internal.DateLayout))
		}
	}
	if !criteria.SentSince.IsZero() && !criteria.SentBefore.IsZero() && criteria.SentBefore.Sub(criteria.SentSince) == 24*time.Hour {
		encodeItem().Atom("SENTON").SP().String(criteria.SentSince.Format(internal.DateLayout))
	} else {
		if !criteria.SentSince.IsZero() {
			encodeItem().Atom("SENTSINCE").SP().String(criteria.SentSince.Format(internal.DateLayout))
		}
		if !criteria.SentBefore.IsZero() {
			encodeItem().Atom("SENTBEFORE").SP().String(criteria.SentBefore.Format(internal.DateLayout))
		}
	}

	for _, kv := range criteria.Header {
		switch k := strings.ToUpper(kv.Key); k {
		case "BCC", "CC", "FROM", "SUBJECT", "TO":
			encodeItem().Atom(k)
		default:
			encodeItem().Atom("HEADER").SP().String(kv.Key)
		}
		enc.SP().String(kv.Value)
	}

	for _, s := range criteria.Body {
		encodeItem().Atom("BODY").SP().String(s)
	}
	for _, s := range criteria.Text {
		encodeItem().Atom("TEXT").SP().String(s)
	}

	for _, flag := range criteria.Flag {
		if k := flagSearchKey(flag); k != "" {
			encodeItem().Atom(k)
		} else {
			encodeItem().Atom("KEYWORD").SP().Flag(flag)
		}
	}
	for _, flag := range criteria.NotFlag {
		if k := flagSearchKey(flag); k != "" {
			encodeItem().Atom("UN" + k)
		} else {
			encodeItem().Atom("UNKEYWORD").SP().Flag(flag)
		}
	}

	if criteria.Larger > 0 {
		encodeItem().Atom("LARGER").SP().Number64(criteria.Larger)
	}
	if criteria.Smaller > 0 {
		encodeItem().Atom("SMALLER").SP().Number64(criteria.Smaller)
	}

	if modSeq := criteria.ModSeq; modSeq != nil {
		encodeItem().Atom("MODSEQ")
		if modSeq.MetadataName != "" && modSeq.MetadataType != "" {
			enc.SP().Quoted(modSeq.MetadataName).SP().Atom(string(modSeq.MetadataType))
		}
		enc.SP()
		if modSeq.ModSeq != 0 {
			enc.ModSeq(modSeq.ModSeq)
		} else {
			enc.Atom("0")
		}
	}

	for _, not := range criteria.Not {
		encodeItem().Atom("NOT").SP()
		enc.Special('(')
		writeSearchKey(enc, &not)
		enc.Special(')')
	}
	for _, or := range criteria.Or {
		encodeItem().Atom("OR").SP()
		enc.Special('(')
		writeSearchKey(enc, &or[0])
		enc.Special(')')
		enc.SP()
		enc.Special('(')
		writeSearchKey(enc, &or[1])
		enc.Special(')')
	}

	if firstItem {
		enc.Atom("ALL")
	}
}

func flagSearchKey(flag imap.Flag) string {
	switch flag {
	case imap.FlagAnswered, imap.FlagDeleted, imap.FlagDraft, imap.FlagFlagged, imap.FlagSeen:
		return strings.ToUpper(strings.TrimPrefix(string(flag), "\\"))
	default:
		return ""
	}
}

func readESearchResponse(dec *imapwire.Decoder) (tag string, data *imap.SearchData, err error) {
	data = &imap.SearchData{}
	if dec.Special('(') { // search-correlator
		var correlator string
		if !dec.ExpectAtom(&correlator) || !dec.ExpectSP() || !dec.ExpectAString(&tag) || !dec.ExpectSpecial(')') {
			return "", nil, dec.Err()
		}
		if correlator != "TAG" {
			return "", nil, fmt.Errorf("in search-correlator: name must be TAG, but got %q", correlator)
		}
	}

	var name string
	if !dec.SP() {
		return tag, data, nil
	} else if !dec.ExpectAtom(&name) {
		return "", nil, dec.Err()
	}
	data.UID = name == "UID"

	if data.UID {
		if !dec.SP() {
			return tag, data, nil
		} else if !dec.ExpectAtom(&name) {
			return "", nil, dec.Err()
		}
	}

	for {
		if !dec.ExpectSP() {
			return "", nil, dec.Err()
		}

		switch strings.ToUpper(name) {
		case "MIN":
			var num uint32
			if !dec.ExpectNumber(&num) {
				return "", nil, dec.Err()
			}
			data.Min = num
		case "MAX":
			var num uint32
			if !dec.ExpectNumber(&num) {
				return "", nil, dec.Err()
			}
			data.Max = num
		case "ALL":
			numKind := imapwire.NumKindSeq
			if data.UID {
				numKind = imapwire.NumKindUID
			}
			if !dec.ExpectNumSet(numKind, &data.All) {
				return "", nil, dec.Err()
			}
			if data.All.Dynamic() {
				return "", nil, fmt.Errorf("imapclient: server returned a dynamic ALL number set in SEARCH response")
			}
		case "COUNT":
			var num uint32
			if !dec.ExpectNumber(&num) {
				return "", nil, dec.Err()
			}
			data.Count = num
		case "MODSEQ":
			var modSeq uint64
			if !dec.ExpectModSeq(&modSeq) {
				return "", nil, dec.Err()
			}
			data.ModSeq = modSeq
		default:
			if !dec.DiscardValue() {
				return "", nil, dec.Err()
			}
		}

		if !dec.SP() {
			break
		} else if !dec.ExpectAtom(&name) {
			return "", nil, dec.Err()
		}
	}

	return tag, data, nil
}

func searchCriteriaIsASCII(criteria *imap.SearchCriteria) bool {
	for _, kv := range criteria.Header {
		if !isASCII(kv.Key) || !isASCII(kv.Value) {
			return false
		}
	}
	for _, s := range criteria.Body {
		if !isASCII(s) {
			return false
		}
	}
	for _, s := range criteria.Text {
		if !isASCII(s) {
			return false
		}
	}
	for _, not := range criteria.Not {
		if !searchCriteriaIsASCII(&not) {
			return false
		}
	}
	for _, or := range criteria.Or {
		if !searchCriteriaIsASCII(&or[0]) || !searchCriteriaIsASCII(&or[1]) {
			return false
		}
	}
	return true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}
