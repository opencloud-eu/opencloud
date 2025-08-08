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

// When the request contains invalid parameters.
// swagger:response ErrorResponse400
type SwaggerErrorResponse400 struct {
	// in: body
	Body struct {
		*ErrorResponse
	}
}

// When the requested object does not exist.
// swagger:response ErrorResponse404
type SwaggerErrorResponse404 struct {
}

// When the server was unable to complete the request.
// swagger:response ErrorResponse500
type SwaggerErrorResponse500 struct {
	// in: body
	Body struct {
		*ErrorResponse
	}
}

// swagger:parameters vacation mailboxes
type SwaggerAccountParams struct {
	// The identifier of the account.
	// in: path
	Account string `json:"account"`
}
