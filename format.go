package qp

import (
	"fmt"
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
//		var values = qp.New().Jumper(", ")
//		values.Format("(%+p)", 1, "Tom", 12)
//		values.Format("(%+p)", 2, "Huckleberry", 13)
//
//		var query = qp.Format("INSERT INTO users (id, name, age) VALUES %s", values)
//		_ = query.String() // INSERT INTO users (id, name, age) VALUES ($1, $2, $3), ($4, $5, $6)
//		_ = query.Params() // [1, "Tom", 12, 2, "Huckleberry", 13]
func New() Formatter {
	return &formatter{
		format: []string{},
		params: [][]interface{}{},
		driver: driver(),
		jumper: " AND ",
	}
}

// Format formats according to a format specifier and returns the sql query string
//		var query = qp.Format("SELECT id FROM table WHERE name = %p LIMIT %p OFFSET %p", "Tom", 10, 0)
//		_ = query.String() // SELECT id FROM table WHERE name = $1 LIMIT $2 OFFSET $3
//		_ = query.Params() // ["Tom", 10, 0]
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
	var (
		b strings.Builder
		p int
		i int
		j int
		l int
		r bool
		s bool
	)
	for n, format := range f.format {
		if n > 0 {
			b.WriteString(f.jumper)
		}
		for i, j, p = 0, 0, 0; i < len(format); i++ {
			switch {
			case format[i] == '%':
				if l, r = btoi(!s), !r; !r {
					b.WriteString(format[j : i-l+1])
					j = i + 1
					s = false
				}
			case format[i] == '+' && r:
				s = true
				l = l + 1
			case format[i] == 's' && r:
				if p >= len(f.params[n]) {
					panic("qp: parameter not found")
				}
				b.WriteString(format[j : i-l])
				b.WriteString(f.s(n, p, s))
				if s {
					p = len(f.params[n])
				}
				p = p + 1
				j = i + 1
				r = false
				s = false
			case format[i] == 'p' && r:
				if p >= len(f.params[n]) {
					panic("qp: parameter not found")
				}
				b.WriteString(format[j : i-l])
				b.WriteString(f.p(n, p, s))
				if s {
					p = len(f.params[n])
				}
				p = p + 1
				j = i + 1
				r = false
				s = false
			default:
				r = false
				s = false
			}
		}
		b.WriteString(format[j:])
	}
	return b.String()
}

// Params returns parameters for query
func (f *formatter) Params() []interface{} {
	var (
		params = make([]interface{}, 0, len(f.params))
		record = false
		spread = false
	)
	for n, format := range f.format {
		for i, p := 0, 0; i < len(format); i++ {
			switch {
			case format[i] == '%':
				if record = !record; !record {
					spread = false
				}
			case format[i] == '+' && record:
				spread = true
			case format[i] == 's' && record:
				if p >= len(f.params[n]) {
					panic("qp: parameter not found")
				}
				if spread {
					params = filters(params, f.params[n][p:]...)
					p = len(f.params[n])
				} else {
					params = filters(params, f.params[n][p])
					p = p + 1
				}
				record = false
				spread = false
			case format[i] == 'p' && record:
				if p >= len(f.params[n]) {
					panic("qp: parameter not found")
				}
				if spread {
					params = insert(params, f.params[n][p:]...)
					p = len(f.params[n])
				} else {
					params = insert(params, f.params[n][p])
					p = p + 1
				}
				record = false
				spread = false
			default:
				record = false
				spread = false
			}
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

func (f *formatter) s(n, p int, s bool) string {
	switch s {
	case true:
		return f.toString(f.params[n][p:])
	default:
		return f.toString(f.params[n][p])
	}
}

func (f *formatter) p(n, p int, s bool) string {
	switch s {
	case true:
		return f.d().Placeholder(f.params[n][p:])
	default:
		return f.d().Placeholder(f.params[n][p])
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
