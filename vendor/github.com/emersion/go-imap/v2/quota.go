package imap

// QuotaResourceType is a QUOTA resource type.
//
// See RFC 9208 section 5.
type QuotaResourceType string

const (
	QuotaResourceStorage           QuotaResourceType = "STORAGE"
	QuotaResourceMessage           QuotaResourceType = "MESSAGE"
	QuotaResourceMailbox           QuotaResourceType = "MAILBOX"
	QuotaResourceAnnotationStorage QuotaResourceType = "ANNOTATION-STORAGE"
)
