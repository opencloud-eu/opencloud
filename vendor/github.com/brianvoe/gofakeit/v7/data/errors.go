package data

var Error = map[string][]string{
	"object": {
		"argument",
		"buffer",
		"connection",
		"database",
		"header",
		"hostname",
		"method",
		"object",
		"parameter",
		"pointer",
		"port",
		"protocol",
		"request",
		"response",
		"server",
		"service",
		"signature",
		"tag",
		"undefined",
		"url",
		"uri",
		"variable",
	},
	"generic": {
		"error",
		"syntax error",
		"requested {errorobject} is unavailable",
		"failed to {hackerverb} {errorobject}",
		"expected {errorobject} is undefined",
		"[object Object]",
		"no such variable",
		"{errorobject} not initialized",
		"variable assigned before declaration",
	},
	"database": {
		"sql error",
		"database connection error",
		"table does not exist",
		"unique key constraint",
		"table migration failed",
		"bad connection",
		"destination pointer is nil",
	},
	"grpc": {
		"connection refused",
		"connection closed",
		"connection is shut down",
		"client protocol error",
	},
	"http": {
		"cross-origin-resource-policy error",
		"feature not supported",
		"trailer header without chunked transfer encoding",
		"no multipart boundary param in Content-Type",
		"request Content-Type isn't multipart/form-data",
		"header too long",
		"entity body too short",
		"missing ContentLength in HEAD response",
		"named cookie not present",
		"invalid method",
		"connection has been hijacked",
		"request method or response status code does not allow body",
		"wrote more than the declared Content-Length",
		"{httpmethod} not allowed",
	},
	"http_client": { // 400s
		"bad request",                   // 400
		"unauthorized",                  // 401
		"payment required",              // 402
		"forbidden",                     // 403
		"not found",                     // 404
		"method not allowed",            // 405
		"not acceptable",                // 406
		"proxy authentication required", // 407
		"request timeout",               // 408
		"conflict",                      // 409
		"gone",                          // 410
		"length required",               // 411
		"precondition failed",           // 412
		"payload too large",             // 413
		"URI too long",                  // 414
		"unsupported media type",        // 415
		"range not satisfiable",         // 416
		"expectation failed",            // 417
		"im a teapot",                   // 418
	},
	"http_server": { // 500s
		"internal server error",           // 500
		"not implemented",                 // 501
		"bad gateway",                     // 502
		"service unavailable",             // 503
		"gateway timeout",                 // 504
		"http version not supported",      // 505
		"variant also negotiates",         // 506
		"insufficient storage",            // 507
		"loop detected",                   // 508
		"not extended",                    // 510
		"network authentication required", // 511
	},
	"runtime": {
		"panic: runtime error: invalid memory address or nil pointer dereference",
		"address out of bounds",
		"undefined has no such property 'length'",
		"not enough arguments",
		"expected 2 arguments, got 3",
	},
	"validation": {
		"invalid format",
		"missing required field",
		"{inputname} is required",
		"{inputname} max length exceeded",
		"{inputname} must be at exactly 16 characters",
		"{inputname} must be at exactly 32 bytes",
		"failed to parse {inputname}",
		"date is in the past",
		"payment details cannot be verified",
	},
}
