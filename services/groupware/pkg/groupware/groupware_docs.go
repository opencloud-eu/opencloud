// OpenCloud Groupware API
//
// Documentation for the OpenCloud Groupware API
//
//	Schemes: https
//	BasePath: /groupware
//	Version: 1.0.0
//	Host:
//
//	Consumes:
//	- application/json
//
//	Produces:
//	- application/json
//
//	Security:
//	- bearer
//
// swagger:meta
package groupware

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
