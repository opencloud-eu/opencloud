package textutil

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/gookit/goutil/maputil"
	"github.com/gookit/goutil/reflects"
	"github.com/gookit/goutil/strutil"
)

// type SimpleAnyFunc func(args ...any) any

// StrVarRenderer implements like shell vars renderer
// 简单的实现类似 php, kotlin, shell 插值变量渲染，表达式解析处理。
//
//  - var format: $var_name, ${some_var}, ${top.sub_var}
//  - func call: ${func($var_name, 'const string')}
type StrVarRenderer struct {
	// global variables
	vars map[string]any
	// fallback func for var not exists
	getter FallbackFn
	// funcMap map[string]any TODO use any, add reflect value to rfs
	funcMap map[string]func(...any) any
	// var func map. refer the text/template TODO
	//
	// Func allow return 1 or 2 values, if return 2 values, the second value is error.
	rfs map[string]*reflects.FuncX
}

// NewStrVarRenderer create a new StrVarRenderer
func NewStrVarRenderer() *StrVarRenderer {
	return &StrVarRenderer{
		vars: make(map[string]any),
		// funcMap: make(map[string]any),
		funcMap: make(map[string]func(...any) any),
	}
}

// SetVars set variables
func (r *StrVarRenderer) SetVars(vars map[string]any) *StrVarRenderer {
	for k, v := range vars {
		r.vars[k] = v
	}
	return r
}

// SetVar set a variable
func (r *StrVarRenderer) SetVar(name string, value any) *StrVarRenderer {
	r.vars[name] = value
	return r
}

// SetFuncMap set function map
func (r *StrVarRenderer) SetFuncMap(funcMap map[string]func(...any) any) *StrVarRenderer {
	for k, v := range funcMap {
		r.funcMap[k] = v
	}
	return r
}

// SetFunc set a function
func (r *StrVarRenderer) SetFunc(name string, fn func(...any) any) *StrVarRenderer {
	r.funcMap[name] = fn
	return r
}

// SetGetter set variable getter
func (r *StrVarRenderer) SetGetter(getter FallbackFn) *StrVarRenderer {
	r.getter = getter
	return r
}

var (
	// 处理 $var_name 格式
	// - 允许：$1..$N 这样的变量
	// - 也支持 $@, $* 变量
	reS = regexp.MustCompile(`\$(\w[a-zA-Z0-9_]*|[@|*])`)
	// 处理 ${var_name} ${top.sub} 格式
	reQ = regexp.MustCompile(`\$\{([a-zA-Z][a-zA-Z0-9_.]*)\}`)
	// 处理 ${func(...)} 格式
	reFn = regexp.MustCompile(`\$\{([a-zA-Z][a-zA-Z0-9_]*)\(([^}]*)\)\}`)
)

// Render rendering input string with variables
func (r *StrVarRenderer) Render(input string, vars map[string]any) string {
	vars = maputil.Merge1level(r.vars, vars)
	data := maputil.Map(vars)

	// 处理 $var_name 格式
	input = r.replaceVars(input, reS, data)

	if strings.Contains(input, "${") {
		// 处理 ${var.name} 格式
		input = r.replaceVars(input, reQ, data)
		// 处理 ${func(...)} 格式
		input = r.handleFuncCalls(input, reFn, data)
	}

	return input
}

func (r *StrVarRenderer) handleFuncCalls(input string, re *regexp.Regexp, vars maputil.Map) string {
	return re.ReplaceAllStringFunc(input, func(matched string) string {
		submatch := re.FindStringSubmatch(matched)
		funcName := submatch[1]

		// 解析参数 并 调用函数
		if fn, ok := r.funcMap[funcName]; ok {
			args := r.parseArgs(submatch[2], vars)
			result := fn(args...)
			return fmt.Sprint(result)
		}

		return matched
	})
}

func (r *StrVarRenderer) parseArgs(argsStr string, data maputil.Map) []any {
	if argsStr == "" {
		return []any{}
	}

	// 简单参数解析，按逗号分割
	args := strings.Split(argsStr, ",")
	results := make([]any, len(args))

	for i, arg := range args {
		arg = strings.TrimSpace(arg)
		// is var name
		if arg[0] == '$' {
			results[i] = data.Get(arg[1:])
			continue
		}

		// 常量值：去掉引号
		last := len(arg) - 1
		// strconv.Unquote( arg)
		if (arg[0] == '"' && arg[last] == '"') || (arg[0] == '\'' && arg[last] == '\'') {
			results[i] = arg[1 : len(arg)-1]
		} else if arg == "true" || arg == "false" {
			results[i] = strutil.SafeBool(arg)
		} else if strutil.IsInt(arg) {
			results[i] = strutil.SafeInt(arg)
		} else {
			results[i] = arg
		}
	}

	return results
}

func (r *StrVarRenderer) replaceVars(input string, re *regexp.Regexp, data maputil.Map) string {
	return re.ReplaceAllStringFunc(input, func(matched string) string {
		var varName string

		// format: ${var_name} 提取变量名
		if strings.HasPrefix(matched, "${") {
			varName = matched[2 : len(matched)-1]
		} else {
			// format: $var_name
			varName = matched[1:]
		}

		// 从 vars map 获取值，支持嵌套变量名
		if val, ok := data.GetByPath(varName); ok {
			return fmt.Sprint(val)
		}

		// fallback: 使用 getter fn 获取值
		if r.getter != nil {
			if val, ok := r.getter(varName); ok {
				return fmt.Sprint(val)
			}
		}

		return matched
	})
}
