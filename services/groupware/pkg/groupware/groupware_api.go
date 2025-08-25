package groupware

const (
	Version = "0.0.1"
)

const (
	CapMail_1 = "mail:1"
)

var Capabilities = []string{
	CapMail_1,
}

const (
	RelationEntityEmail    = "email"
	RelationTypeSameThread = "same-thread"
	RelationTypeSameSender = "same-sender"
)
