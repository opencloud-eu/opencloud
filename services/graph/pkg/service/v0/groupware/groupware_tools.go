package groupware

import (
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/opencloud-eu/opencloud/pkg/jmap"
)

func pickInbox(folders []jmap.Mailbox) string {
	for _, folder := range folders {
		if folder.Role == "inbox" {
			return folder.Id
		}
	}
	return ""
}

func mapContentType(jmap string) string {
	switch jmap {
	case "text/html":
		return "html"
	case "text/plain":
		return "text"
	default:
		return jmap
	}
}

func foldBody(email jmap.Email) *ItemBody {
	if email.BodyValues != nil {
		if len(email.HtmlBody) > 0 {
			pick := email.HtmlBody[0]
			content, ok := email.BodyValues[pick.PartId]
			if ok {
				return &ItemBody{Content: content.Value, ContentType: mapContentType(pick.Type)}
			}
		}
		if len(email.TextBody) > 0 {
			pick := email.TextBody[0]
			content, ok := email.BodyValues[pick.PartId]
			if ok {
				return &ItemBody{Content: content.Value, ContentType: mapContentType(pick.Type)}
			}
		}
	}
	return nil
}

func firstOf[T any](ary []T) T {
	if len(ary) > 0 {
		return ary[0]
	}
	var nothing T
	return nothing
}

func emailAddress(j jmap.EmailAddress) EmailAddress {
	return EmailAddress{Address: j.Email, Name: j.Name}
}

func emailAddresses(j []jmap.EmailAddress) []EmailAddress {
	result := make([]EmailAddress, len(j))
	for i := 0; i < len(j); i++ {
		result[i] = emailAddress(j[i])
	}
	return result
}

func hasKeyword(j jmap.Email, kw string) bool {
	value, ok := j.Keywords[kw]
	if ok {
		return value
	}
	return false
}

func categories(j jmap.Email) []string {
	categories := []string{}
	for k, v := range j.Keywords {
		if v && !strings.HasPrefix(k, jmap.JmapKeywordPrefix) {
			categories = append(categories, k)
		}
	}
	return categories
}

/*
func toEdmBinary(value int) string {
	return fmt.Sprintf("%X", value)
}
*/

// https://learn.microsoft.com/en-us/graph/api/resources/message?view=graph-rest-1.0
func message(email jmap.Email, state string) Message {
	body := foldBody(email)
	importance := "" // omit "normal" as it is expected to be the default
	if hasKeyword(email, jmap.JmapKeywordFlagged) {
		importance = "high"
	}

	mailboxId := ""
	for k, v := range email.MailboxIds {
		if v {
			// TODO how to map JMAP short identifiers (e.g. 'a') to something uniquely addressable for the clients?
			// e.g. do we need to include tenant/sharding/cluster information?
			mailboxId = k
			break
		}
	}

	// TODO how to map JMAP short identifiers (e.g. 'a') to something uniquely addressable for the clients?
	// e.g. do we need to include tenant/sharding/cluster information?
	id := email.Id
	// for this one too:
	messageId := firstOf(email.MessageId)
	// as well as this one:
	threadId := email.ThreadId

	categories := categories(email)

	var from *EmailAddress = nil
	if len(email.From) > 0 {
		e := emailAddress(email.From[0])
		from = &e
	}

	// TODO how to map JMAP state to an OData Etag?
	etag := state

	weblink, err := url.JoinPath("/groupware/mail", id)
	if err != nil {
		weblink = ""
	}

	return Message{
		Etag:              etag,
		Id:                id,
		Subject:           email.Subject,
		CreatedDateTime:   email.ReceivedAt,
		ReceivedDateTime:  email.ReceivedAt,
		SentDateTime:      email.SentAt,
		HasAttachments:    email.HasAttachments,
		InternetMessageId: messageId,
		BodyPreview:       email.Preview,
		Body:              body,
		From:              from,
		ToRecipients:      emailAddresses(email.To),
		CcRecipients:      emailAddresses(email.Cc),
		BccRecipients:     emailAddresses(email.Bcc),
		ReplyTo:           emailAddresses(email.ReplyTo),
		IsRead:            hasKeyword(email, jmap.JmapKeywordSeen),
		IsDraft:           hasKeyword(email, jmap.JmapKeywordDraft),
		Importance:        importance,
		ParentFolderId:    mailboxId,
		Categories:        categories,
		ConversationId:    threadId,
		WebLink:           weblink,
		// ConversationIndex: toEdmBinary(email.ThreadIndex),
	} // TODO more email fields
}

func parseNumericParam(r *http.Request, param string, defaultValue int) (int, bool, error) {
	str := r.URL.Query().Get(param)
	if str == "" {
		return defaultValue, false, nil
	}

	value, err := strconv.ParseInt(str, 10, 0)
	if err != nil {
		return defaultValue, false, nil
	}
	return int(value), true, nil
}
