package textutil

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"text/template"

	"github.com/gookit/goutil/arrutil"
	"github.com/gookit/goutil/fsutil"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/structs"
	"github.com/gookit/goutil/strutil"
	"github.com/gookit/goutil/x/basefn"
)

// LTemplateOptFn lite template option func
type LTemplateOptFn func(opt *LiteTemplateOpt)

// LiteTemplateOpt template options for LiteTemplate
type LiteTemplateOpt struct {
	// func name alias map. eg: {"up_first": "upFirst"}
	nameMp structs.Aliases
	Funcs  template.FuncMap

	Left, Right string

	ParseDef bool
	ParseEnv bool
}

// SetVarFmt custom sets the variable format in template
func (o *LiteTemplateOpt) SetVarFmt(varFmt string) {
	o.Left, o.Right = strutil.TrimCut(varFmt, ",")
}

// LiteTemplate implement a simple text template engine.
//
//   - support parse template vars
//   - support access multi-level map field. eg: {{ user.name }}
//   - support parse default value
//   - support parse env vars
//   - support custom pipeline func handle. eg: {{ name | upper }} {{ name | def:guest }}
//
// NOTE: not support control flow, eg: if/else/for/with
type LiteTemplate struct {
	LiteTemplateOpt
	vr VarReplacer
	// template func map. refer the text/template
	//
	// Func allow return 1 or 2 values, if return 2 values, the second value is error.
	fxs map[string]*reflects.FuncX
}

// NewLiteTemplate instance
func NewLiteTemplate(opFns ...LTemplateOptFn) *LiteTemplate {
	st := &LiteTemplate{
		fxs: make(map[string]*reflects.FuncX),
		// with default options
		LiteTemplateOpt: LiteTemplateOpt{
			Left:     "{{",
			Right:    "}}",
			ParseDef: true,
			ParseEnv: true,
		},
	}

	st.vr.RenderFn = st.renderVars
	for _, fn := range opFns {
		fn(&st.LiteTemplateOpt)
	}

	st.Init()
	return st
}

// Init LiteTemplate
func (t *LiteTemplate) Init() {
	if t.vr.init {
		return
	}

	// init var replacer
	t.vr.init = true
	t.initReplacer(&t.vr)

	// add built-in funcs
	t.AddFuncs(builtInFuncs)
	t.nameMp.AddAliasMap(map[string]string{
		"up_first": "upFirst",
		"lc_first": "lcFirst",
		"def":      "default",
	})

	// add custom funcs
	if len(t.Funcs) > 0 {
		t.AddFuncs(t.Funcs)
	}
}

func (t *LiteTemplate) initReplacer(vr *VarReplacer) {
	vr.flatSubs = true
	vr.parseDef = t.ParseDef
	vr.parseEnv = t.ParseEnv
	vr.Left, vr.Right = t.Left, t.Right
	basefn.PanicIf(vr.Right == "", "var format right chars is required")

	vr.lLen, vr.rLen = len(vr.Left), len(vr.Right)
	rightLast := string(vr.Right[vr.rLen-1]) // 排除匹配，防止匹配到类似 "{} adb ddf {var}"

	// eg: \{(?s:([^\}]+?))\}
	// (?s:...) - 让 "." 匹配换行
	// (?s:(.+?)) - 第二个 "?" 非贪婪匹配
	pattern := regexp.QuoteMeta(vr.Left) + `(?s:([^` + regexp.QuoteMeta(rightLast) + `]+?))` + regexp.QuoteMeta(vr.Right)
	vr.varReg = regexp.MustCompile(pattern)
}

// AddFuncs add custom template functions
func (t *LiteTemplate) AddFuncs(fns map[string]any) {
	for name, fn := range fns {
		t.fxs[name] = reflects.NewFunc(fn)
	}
}

// RenderString render template string with vars
func (t *LiteTemplate) RenderString(s string, vars map[string]any) string {
	return t.vr.Replace(s, vars)
}

// RenderFile render template file with vars
func (t *LiteTemplate) RenderFile(filePath string, vars map[string]any) (string, error) {
	// read file contents
	s, err := fsutil.ReadStringOrErr(filePath)
	if err != nil {
		return "", err
	}

	return t.vr.Replace(s, vars), nil
}

// RenderWrite render template string with vars, and write result to writer
func (t *LiteTemplate) RenderWrite(wr io.Writer, s string, vars map[string]any) error {
	s = t.vr.Replace(s, vars)
	_, err := io.WriteString(wr, s)
	return err
}

func (t *LiteTemplate) renderVars(s string, varMap map[string]string) string {
	return t.vr.varReg.ReplaceAllStringFunc(s, func(sub string) string {
		// var name or pipe expression.
		name := strings.TrimSpace(sub[t.vr.lLen : len(sub)-t.vr.rLen])
		name = strings.TrimLeft(name, "$.")

		var defVal string
		var pipes []string
		if strings.ContainsRune(name, '|') {
			pipes = strutil.Split(name, "|")
			// compatible default value. eg: {{ name | inhere }}
			if len(pipes) == 2 && !strings.ContainsRune(pipes[1], ':') && !t.isFunc(pipes[1]) {
				name, defVal = pipes[0], pipes[1]
				pipes = nil // clear pipes
			} else { // collect pipe functions
				name, pipes = pipes[0], pipes[1:]
			}
		}

		if val, ok := varMap[name]; ok {
			if len(pipes) > 0 {
				var err error
				val, err = t.applyPipes(val, pipes)
				if err != nil {
					return fmt.Sprintf("Render var %q error: %v", name, err)
				}
			}
			return val
		}

		// var not found
		if len(defVal) > 0 {
			return defVal
		}

		if t.vr.NotFound != nil {
			if val, ok := t.vr.NotFound(name); ok {
				return val
			}
		}

		// check is default func. eg: {{ name | def:guest }}
		if len(pipes) == 1 && strings.ContainsRune(pipes[0], ':') {
			fName, argVal := strutil.TrimCut(pipes[0], ":")
			if t.isDefaultFunc(fName) {
				return argVal
			}
		}

		t.vr.missVars = append(t.vr.missVars, name)
		return sub
	})
}

func (t *LiteTemplate) applyPipes(val any, pipes []string) (string, error) {
	var err error

	// pipe expr: "trim|upper|substr:1,2"
	// =>
	// pipes: ["trim", "upper", "substr:1,2"]
	for _, name := range pipes {
		args := []any{val}

		// has custom args. eg: "substr:1,2"
		if strings.ContainsRune(name, ':') {
			var argStr string
			name, argStr = strutil.TrimCut(name, ":")

			if otherArgs := parseArgStr(argStr); len(otherArgs) > 0 {
				args = append(args, otherArgs...)
			}
		}

		name = t.nameMp.ResolveAlias(name)

		// call pipe func
		if fx, ok := t.fxs[name]; ok {
			val, err = fx.Call2(args...)
			if err != nil {
				return "", err
			}
		} else {
			return "", fmt.Errorf("template func %q not found", name)
		}
	}

	return strutil.ToString(val)
}

func (t *LiteTemplate) isFunc(name string) bool {
	_, ok := t.fxs[name]
	if !ok {
		// check name alias
		return t.nameMp.HasAlias(name)
	}
	return ok
}

func (t *LiteTemplate) isDefaultFunc(name string) bool {
	return name == "default" || name == "def"
}

var stdTpl = NewLiteTemplate()

// RenderFile render template file with vars
func RenderFile(filePath string, vars map[string]any) (string, error) {
	return stdTpl.RenderFile(filePath, vars)
}

// RenderString render str template string or file.
func RenderString(input string, data map[string]any) string {
	return stdTpl.RenderString(input, data)
}

// RenderWrite render template string with vars, and write result to writer
func RenderWrite(wr io.Writer, s string, vars map[string]any) error {
	return stdTpl.RenderWrite(wr, s, vars)
}

func parseArgStr(argStr string) (ss []any) {
	if argStr == "" { // no arg
		return
	}

	if len(argStr) == 1 { // one char
		return []any{argStr}
	}
	return arrutil.StringsToAnys(strutil.Split(argStr, ","))
}
