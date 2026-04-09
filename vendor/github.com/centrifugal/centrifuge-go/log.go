package centrifuge

// LogHandler handles log entries - i.e. writes into correct destination if necessary.
type LogHandler func(LogEntry)

// LogLevel describes the chosen log level.
type LogLevel int

func (l LogLevel) String() string {
	switch l {
	case LogLevelNone:
		return "NONE"
	case LogLevelTrace:
		return "TRACE"
	case LogLevelDebug:
		return "DEBUG"
	default:
		return "UNKNOWN"
	}
}

const (
	// LogLevelNone means no logging.
	LogLevelNone LogLevel = iota
	// LogLevelTrace turns on trace logs - should only be used during development. This
	// log level shows all client-server communication (including token content) and
	// affects performance significantly.
	LogLevelTrace
	// LogLevelDebug turns on debug logs - it's generally too much for production in normal
	// conditions but can help when developing and investigating problems in production.
	LogLevelDebug
)

// LogEntry describes a log entry.
type LogEntry struct {
	Level   LogLevel
	Message string
	Fields  map[string]string
}
