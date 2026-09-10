package hierarchy

import (
	"bytes"
	"strconv"

	"github.com/blevesearch/bleve/v2/analysis"
	"github.com/blevesearch/bleve/v2/registry"
)

// emits every prefix up to a level: "./a/b" -> ".", "./a", "./a/b" with
// delimiter "/", one level per byte without. tag_depth prepends "<depth>/".
const Name = "hierarchy"

type Tokenizer struct {
	delimiter []byte
	tagDepth  bool
}

func (t *Tokenizer) Tokenize(input []byte) analysis.TokenStream {
	if len(input) == 0 {
		return nil
	}
	var out analysis.TokenStream
	emit := func(depth, end int) {
		term := input[:end]
		if t.tagDepth {
			term = strconv.AppendInt(make([]byte, 0, end+4), int64(depth), 10)
			term = append(term, '/')
			term = append(term, input[:end]...)
		}
		out = append(out, &analysis.Token{
			Term:     term,
			Position: depth,
			Start:    0,
			End:      end,
			Type:     analysis.AlphaNumeric,
		})
	}

	if len(t.delimiter) == 0 {
		for i := range input {
			emit(i+1, i+1)
		}
		return out
	}

	depth := 0
	for start := 0; start <= len(input); {
		i := bytes.Index(input[start:], t.delimiter)
		if i < 0 {
			if start < len(input) {
				depth++
				emit(depth, len(input))
			}
			break
		}
		if i > 0 {
			depth++
			emit(depth, start+i)
		}
		start += i + len(t.delimiter)
	}
	return out
}

func Constructor(config map[string]interface{}, _ *registry.Cache) (analysis.Tokenizer, error) {
	t := &Tokenizer{}
	if d, ok := config["delimiter"].(string); ok {
		t.delimiter = []byte(d)
	}
	if v, ok := config["tag_depth"].(bool); ok {
		t.tagDepth = v
	}
	return t, nil
}

func init() {
	if err := registry.RegisterTokenizer(Name, Constructor); err != nil {
		panic(err)
	}
}
