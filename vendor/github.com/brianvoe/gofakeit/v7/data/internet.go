package data

// Internet consists of various internet information
var Internet = map[string][]string{
	"browser":             {"firefox", "chrome", "internetExplorer", "opera", "safari"},
	"domain_suffix":       {"com", "biz", "info", "name", "net", "org", "io"},
	"http_method":         {"HEAD", "GET", "POST", "PUT", "PATCH", "DELETE"},
	"http_version":        {"HTTP/1.0", "HTTP/1.1", "HTTP/2.0"},
	"http_status_simple":  {"200", "301", "302", "400", "404", "500"},
	"http_status_general": {"100", "200", "201", "203", "204", "205", "301", "302", "304", "400", "401", "403", "404", "405", "406", "416", "500", "501", "502", "503", "504"},
}
