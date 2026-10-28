`envdecode` is a Go package for populating structs from environment
variables. It's basically a fork of https://github.com/joeshaw/envdecode,
but changed to support multiple environment variables (precedence).

`envdecode` uses struct tags to map environment variables to fields,
allowing you to use any names you want for environment variables.
`envdecode` will recurse into nested structs, including pointers to
nested structs, but it will not allocate new pointers to structs.

## API

Full API docs are available on
[godoc.org](https://godoc.org/github.com/opencloud-eu/opencloud/pkg/config/envdecode).

Define a struct with `env` struct tags:

```go
type Config struct {
    Hostname  string `env:"SERVER_HOSTNAME,default=localhost"`
    Port      uint16 `env:"HTTP_PORT;SERVER_PORT,default=8080"`

    AWS struct {
        ID        string   `env:"AWS_ACCESS_KEY_ID"`
        Secret    string   `env:"AWS_SECRET_ACCESS_KEY,required"`
        SnsTopics []string `env:"AWS_SNS_TOPICS"`
    }

    Timeout time.Duration `env:"TIMEOUT,default=1m,strict"`
}
```

Fields _must be exported_ (i.e. begin with a capital letter) in order
for `envdecode` to work with them. An error will be returned if a
struct with no exported fields is decoded (including one that contains
no `env` tags at all).
Default values may be provided by appending ",default=value" to the
struct tag. Required values may be marked by appending ",required" to the
struct tag. Strict values may be marked by appending ",strict" which will
return an error on Decode if there is an error while parsing.
In addition to any defined environment variable the same environment variable
suffixed with `_FILE` can be used with the following semantics: instead of
taking its value directly as the configuration parameter it is interpreted as a
path to a file whose content (modulo trailing newlines) is interpreted as the
value for the configuration parameter. E.g. instead of setting the access key
directly with `AWS_SECRET_ACCESS_KEY` securely store the access key in a file and
provide the path to the file with `AWS_SECRET_ACCESS_KEY_FILE`. Note:
environment variables not suffixed with `_FILE` have precendence.

Then call `envdecode.Decode`:

```go
var cfg Config
err := envdecode.Decode(&cfg)
```

If you want all fields to act `strict`, you may use `envdecode.StrictDecode`:

```go
var cfg Config
err := envdecode.StrictDecode(&cfg)
```

All parse errors will fail fast and return an error in this mode.

## Supported types

- Structs (and pointer to structs)
- Slices of defined types below, separated by semicolon
- `bool`
- `float32`, `float64`
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `string`
- `time.Duration`, using the [`time.ParseDuration()` format](http://golang.org/pkg/time/#ParseDuration)
- `*url.URL`, using [`url.Parse()`](https://godoc.org/net/url#Parse)
- Types those implement a `Decoder` interface

## Custom `Decoder`

If you want a field to be decoded with custom behavior, you may implement the interface `Decoder` for the filed type.

```go
type Config struct {
  IPAddr IP `env:"IP_ADDR"`
}

type IP net.IP

// Decode implements the interface `envdecode.Decoder`
func (i *IP) Decode(repl string) error {
  *i = net.ParseIP(repl)
  return nil
}
```

`Decoder` is the interface implemented by an object that can decode an environment variable string representation of itself.
