package defaults

// MaxTagLength is the default maximum length of a single tag, in characters.
// The graph service rejects longer tags, and the frontend service announces the
// limit to clients as the graph.tags.max_tag_length capability: both read
// OC_MAX_TAG_LENGTH, and both must default to the same value, so that clients
// are told the limit the server actually enforces.
const MaxTagLength = 100
