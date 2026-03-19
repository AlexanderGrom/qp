package qp

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

var (
	driver  = PgsqlDriver
	drivers = map[string]func() Driver{}
)

type (
	// Driver interface
	Driver interface {
		Placeholder(x interface{}) string
	}

	// Formatter interface
	Formatter interface {
		String() string
		Params() []interface{}
		Format(format string, params ...interface{}) Formatter
		Driver(driver Driver) Formatter
		Jumper(jumper string) Formatter
	}

	// Formatter implements a Formatter interface
	formatter struct {
		format []string
		params [][]interface{}
		driver Driver
		jumper string
		master bool
	}

	formatToken struct {
		end      int
		verb     byte
		spread   bool
		explicit bool
		ref      int
		literal  string
		valid    bool
	}
)

// DefaultDriver sets a default driver
func DefaultDriver(name string) {
	var ok bool
	if driver, ok = drivers[name]; !ok {
		panic("qp: driver '" + name + "' not found")
	}
}

// RegisterDriver registers a new driver
func RegisterDriver(name string, driver func() Driver) {
	drivers[name] = driver
}

// New returns a new empty formatter
//
//	var values = qp.New().Jumper(", ")
//	values.Format("(%+p)", 1, "Tom", 12)
//	values.Format("(%+p)", 2, "Huckleberry", 13)
//
//	var query = qp.Format("INSERT INTO users (id, name, age) VALUES %s", values)
//	_ = query.String() // INSERT INTO users (id, name, age) VALUES ($1, $2, $3), ($4, $5, $6)
//	_ = query.Params() // [1, "Tom", 12, 2, "Huckleberry", 13]
func New() Formatter {
	return &formatter{
		format: []string{},
		params: [][]interface{}{},
		driver: driver(),
		jumper: " AND ",
	}
}

// Format formats according to a format specifier and returns the sql query string
//
//	var query = qp.Format("SELECT id FROM table WHERE name = %p LIMIT %p OFFSET %p", "Tom", 10, 0)
//	_ = query.String() // SELECT id FROM table WHERE name = $1 LIMIT $2 OFFSET $3
//	_ = query.Params() // ["Tom", 10, 0]
func Format(format string, params ...interface{}) Formatter {
	return &formatter{
		format: []string{format},
		params: [][]interface{}{params},
		driver: driver(),
		jumper: " AND ",
	}
}

// String returns a query string
func (f *formatter) String() string {
	defer f.m()
	var b strings.Builder
	for n, format := range f.format {
		if n > 0 {
			b.WriteString(f.jumper)
		}
		var (
			j                   int
			p                   int
			stringsByParam      = map[int]string{}
			placeholdersByParam = map[int]string{}
		)
		for i := 0; i < len(format); i++ {
			if format[i] != '%' {
				continue
			}
			token := parseFormatToken(format, i)
			if !token.valid && token.literal == "" {
				continue
			}
			b.WriteString(format[j:i])
			if token.valid {
				var idx = f.paramIndex(n, p, token)
				switch token.verb {
				case 's':
					b.WriteString(f.s(n, idx, token.spread, stringsByParam))
				case 'p':
					b.WriteString(f.p(n, idx, token.spread, placeholdersByParam))
				}
				p = f.nextParamIndex(n, idx, token)
			} else {
				b.WriteString(token.literal)
			}
			i = token.end
			j = token.end + 1
		}
		b.WriteString(format[j:])
	}
	return b.String()
}

// Params returns parameters for query
func (f *formatter) Params() []interface{} {
	var original = f.d()
	f.driver = cloneDriver(original)
	defer func() {
		if f.master {
			f.driver = original
			return
		}
		f.driver = driver()
	}()
	var params = make([]interface{}, 0, len(f.params))
	for n, format := range f.format {
		var (
			p                   int
			usedStrings         = map[int]bool{}
			usedPlaceholders    = map[int]bool{}
			stringsByParam      = map[int]string{}
			placeholdersByParam = map[int]string{}
		)
		for i := 0; i < len(format); i++ {
			if format[i] != '%' {
				continue
			}
			token := parseFormatToken(format, i)
			if !token.valid && token.literal == "" {
				continue
			}
			if token.valid {
				var idx = f.paramIndex(n, p, token)
				switch token.verb {
				case 's':
					params = f.appendStringParams(params, n, idx, token.spread, usedStrings, stringsByParam)
				case 'p':
					params = f.appendPlaceholderParams(params, n, idx, token.spread, usedPlaceholders, placeholdersByParam)
				}
				p = f.nextParamIndex(n, idx, token)
			}
			i = token.end
		}
	}
	return params
}

// Format formats according to a format specifier and returns the sql query string
func (f *formatter) Format(format string, params ...interface{}) Formatter {
	f.params = append(f.params, params)
	f.format = append(f.format, format)
	return f
}

// Driver sets a Driver
func (f *formatter) Driver(driver Driver) Formatter {
	f.driver = driver
	f.master = true
	return f
}

// Jumper sets a string concatenator
// For example: " AND ", " OR ", ", "
func (f *formatter) Jumper(jumper string) Formatter {
	f.jumper = jumper
	return f
}

func (f *formatter) s(n, p int, s bool, cache map[int]string) string {
	switch s {
	case true:
		return f.stringsAt(n, p, cache)
	default:
		return f.stringAt(n, p, cache)
	}
}

func (f *formatter) p(n, p int, s bool, cache map[int]string) string {
	switch s {
	case true:
		return f.placeholdersAt(n, p, cache)
	default:
		return f.placeholderAt(n, p, cache)
	}
}

func (f *formatter) d() Driver {
	if f.driver == nil {
		f.driver = driver()
	}
	return f.driver
}

func (f *formatter) m() {
	if !f.master {
		f.driver = driver()
	}
}

func (f *formatter) stringAt(n, p int, cache map[int]string) string {
	if s, ok := cache[p]; ok {
		return s
	}
	s := f.toString(f.params[n][p])
	cache[p] = s
	return s
}

func (f *formatter) stringsAt(n, start int, cache map[int]string) string {
	if start >= len(f.params[n]) {
		return ""
	}
	var b strings.Builder
	b.WriteString(f.stringAt(n, start, cache))
	for i := start + 1; i < len(f.params[n]); i++ {
		b.WriteString(", ")
		b.WriteString(f.stringAt(n, i, cache))
	}
	return b.String()
}

func (f *formatter) placeholderAt(n, p int, cache map[int]string) string {
	if s, ok := cache[p]; ok {
		return s
	}
	s := f.d().Placeholder(f.params[n][p])
	cache[p] = s
	return s
}

func (f *formatter) placeholdersAt(n, start int, cache map[int]string) string {
	if start >= len(f.params[n]) {
		return ""
	}
	var b strings.Builder
	b.WriteString(f.placeholderAt(n, start, cache))
	for i := start + 1; i < len(f.params[n]); i++ {
		b.WriteString(", ")
		b.WriteString(f.placeholderAt(n, i, cache))
	}
	return b.String()
}

func (f *formatter) appendStringParams(params []interface{}, n, idx int, spread bool, used map[int]bool, cache map[int]string) []interface{} {
	if spread {
		for i := idx; i < len(f.params[n]); i++ {
			if used[i] {
				continue
			}
			params = f.appendFilters(params, f.params[n][i])
			if f.reuseStringParam(n, i, cache) {
				used[i] = true
			}
		}
		return params
	}
	if used[idx] {
		return params
	}
	params = f.appendFilters(params, f.params[n][idx])
	if f.reuseStringParam(n, idx, cache) {
		used[idx] = true
	}
	return params
}

func (f *formatter) appendPlaceholderParams(params []interface{}, n, idx int, spread bool, used map[int]bool, cache map[int]string) []interface{} {
	if spread {
		for i := idx; i < len(f.params[n]); i++ {
			if used[i] {
				continue
			}
			params = insert(params, f.params[n][i])
			if f.reusePlaceholderParam(n, i, cache) {
				used[i] = true
			}
		}
		return params
	}
	if used[idx] {
		return params
	}
	params = insert(params, f.params[n][idx])
	if f.reusePlaceholderParam(n, idx, cache) {
		used[idx] = true
	}
	return params
}

func (f *formatter) appendFilters(params []interface{}, args ...interface{}) []interface{} {
	for _, x := range args {
		switch x := x.(type) {
		case Formatter:
			params = append(params, x.Driver(f.d()).Params()...)
		case []interface{}:
			params = f.appendFilters(params, x...)
		}
	}
	return params
}

func (f *formatter) paramIndex(n, p int, token formatToken) int {
	if token.explicit {
		p = token.ref
	}
	if p >= len(f.params[n]) {
		panic("qp: parameter not found")
	}
	return p
}

func (f *formatter) reuseStringParam(n, p int, cache map[int]string) bool {
	return bindsParams(f.params[n][p]) && isReusableBinding(f.stringAt(n, p, cache))
}

func (f *formatter) reusePlaceholderParam(n, p int, cache map[int]string) bool {
	return isReusableBinding(f.placeholderAt(n, p, cache))
}

func isReusableBinding(s string) bool {
	return len(s) > 0 && !hasAnonymousPlaceholder(s)
}

func cloneDriver(d Driver) Driver {
	if d == nil {
		return nil
	}
	var v = reflect.ValueOf(d)
	if v.Kind() != reflect.Ptr || v.IsNil() {
		return d
	}
	var clone = reflect.New(v.Elem().Type())
	clone.Elem().Set(v.Elem())
	if driver, ok := clone.Interface().(Driver); ok {
		return driver
	}
	return d
}

func hasAnonymousPlaceholder(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] != '?' {
			continue
		}
		if i+1 < len(s) {
			switch {
			case s[i+1] >= '0' && s[i+1] <= '9':
				continue
			case s[i+1] >= 'a' && s[i+1] <= 'z':
				continue
			case s[i+1] >= 'A' && s[i+1] <= 'Z':
				continue
			case s[i+1] == '_':
				continue
			}
		}
		return true
	}
	return false
}

func parseFormatToken(format string, start int) formatToken {
	var token = formatToken{end: start}
	if start >= len(format) || format[start] != '%' {
		return token
	}

	i := start + 1
	if i >= len(format) {
		token.literal = "%"
		return token
	}
	if format[i] == '%' {
		token.end = i
		token.literal = "%"
		return token
	}

	var ref int
	if format[i] == '[' {
		token.explicit = true
		i = i + 1
		var j = i
		for ; i < len(format) && format[i] >= '0' && format[i] <= '9'; i++ {
			ref = ref*10 + int(format[i]-'0')
		}
		switch {
		case j == i:
			token.end = i
			token.literal = format[start : i+1]
			return token
		case i >= len(format):
			token.end = i - 1
			token.literal = format[start:i]
			return token
		case format[i] != ']':
			token.end = i
			token.literal = format[start : i+1]
			return token
		default:
			i = i + 1
		}
	}
	for ; i < len(format) && format[i] == '+'; i++ {
		token.spread = true
	}

	switch {
	case i >= len(format):
		token.end = i - 1
		token.literal = format[start:i]
		return token
	case format[i] != 's' && format[i] != 'p':
		token.end = i
		token.literal = format[start : i+1]
		return token
	case token.explicit && (token.spread || ref == 0):
		token.end = i
		token.literal = format[start : i+1]
		return token
	default:
		token.end = i
		token.verb = format[i]
		token.ref = ref - 1
		token.valid = true
		return token
	}
}

func (f *formatter) nextParamIndex(n, idx int, token formatToken) int {
	if token.spread {
		return len(f.params[n])
	}
	return idx + 1
}

func bindsParams(x interface{}) bool {
	switch x := x.(type) {
	case Formatter:
		return true
	case []interface{}:
		for _, y := range x {
			if bindsParams(y) {
				return true
			}
		}
	}
	return false
}

// ToString converts an interface to string
func (f *formatter) toString(x interface{}) string {
	switch x := x.(type) {
	case string:
		return x
	case Formatter:
		return x.Driver(f.d()).String()
	case fmt.Stringer:
		return x.String()
	case int:
		return strconv.FormatInt(int64(x), 10)
	case int8:
		return strconv.FormatInt(int64(x), 10)
	case int16:
		return strconv.FormatInt(int64(x), 10)
	case int32:
		return strconv.FormatInt(int64(x), 10)
	case int64:
		return strconv.FormatInt(int64(x), 10)
	case uint:
		return strconv.FormatUint(uint64(x), 10)
	case uint8:
		return strconv.FormatUint(uint64(x), 10)
	case uint16:
		return strconv.FormatUint(uint64(x), 10)
	case uint32:
		return strconv.FormatUint(uint64(x), 10)
	case uint64:
		return strconv.FormatUint(uint64(x), 10)
	case float32:
		return strconv.FormatFloat(float64(x), 'f', 6, 32)
	case float64:
		return strconv.FormatFloat(x, 'f', 6, 64)
	case []byte:
		return string(x)
	case []rune:
		return string(x)
	case []int:
		return intsToString(x)
	case []int64:
		return int64sToString(x)
	case []string:
		return stringsToString(x)
	case []interface{}:
		return f.strings(x)
	case nil:
		return ""
	default:
		return fmt.Sprint(x)
	}
}

// strings converts []interface to string
// For example: []interface{"name", "surname", []string{"age"}} => "name, surname, age"
func (f *formatter) strings(x []interface{}) string {
	var n int
	if n = len(x); n == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(f.toString(x[0]))
	for i := 1; i < n; i++ {
		b.WriteString(", ")
		b.WriteString(f.toString(x[i]))
	}
	return b.String()
}
