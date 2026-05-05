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
	// Driver defines placeholder generation for a database driver.
	Driver interface {
		Placeholder(x interface{}) string
	}

	// Formatter formats query fragments and collects query parameters.
	Formatter interface {
		String() string
		Params() []interface{}
		Format(format string, params ...interface{}) Formatter
		Driver(driver Driver) Formatter
		Jumper(jumper string) Formatter
	}

	// formatter implements the Formatter interface.
	formatter struct {
		format []string
		params [][]interface{}
		driver Driver
		jumper string
		master bool
	}
)

// DefaultDriver sets the default driver.
func DefaultDriver(name string) {
	var ok bool
	if driver, ok = drivers[name]; !ok {
		panic("qp: driver '" + name + "' not found")
	}
}

// RegisterDriver registers a new driver.
func RegisterDriver(name string, driver func() Driver) {
	drivers[name] = driver
}

// New returns a new empty formatter.
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

// Format formats according to a format specifier and returns the SQL query string.
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

// String returns the query string.
func (f *formatter) String() string {
	defer f.m()
	var (
		b strings.Builder

		verbLabel  int
		verbStart  int
		verbRecord bool
		verbSpread bool

		reset = func() {
			verbStart = 0
			verbLabel = 0
			verbSpread = false
			verbRecord = false
		}
	)
	b.Grow(f.formatsCap())
	for n, format := range f.format {
		var labels map[int]int
		if n > 0 {
			b.WriteString(f.jumper)
		}
		var j int
		for i, p := 0, 0; i < len(format); i++ {
			switch {
			case format[i] == '%':
				if verbRecord = !verbRecord; verbRecord {
					verbStart = i
					break
				}
				if verbSpread || verbLabel > 0 {
					b.WriteString(format[j : i+1])
				} else {
					b.WriteString(format[j : verbStart+1])
				}
				j = i + 1
				reset()
			case format[i] == '+' && verbRecord:
				if verbLabel == 0 {
					verbSpread = true
				}
			case format[i] == '[' && verbRecord:
				if verbLabel, i = f.parseVerbIndex(format, i); verbLabel > 0 {
					verbSpread = false
					break
				}
				reset()
			case format[i] == 's' && verbRecord:
				var idx = p
				if verbLabel > 0 {
					labels, idx = f.bindVerbIndex(labels, verbLabel, p)
				}
				f.paramsCheck(n, idx)
				b.WriteString(format[j:verbStart])
				f.writeString(&b, n, idx, verbSpread)
				if verbSpread {
					p = len(f.params[n])
				} else {
					p = advanceParam(p, idx)
				}
				j = i + 1
				reset()
			case format[i] == 'p' && verbRecord:
				var idx = p
				if verbLabel > 0 {
					labels, idx = f.bindVerbIndex(labels, verbLabel, p)
				}
				f.paramsCheck(n, idx)
				b.WriteString(format[j:verbStart])
				f.writePlaceholder(&b, n, idx, verbSpread)
				if verbSpread {
					p = len(f.params[n])
				} else {
					p = advanceParam(p, idx)
				}
				j = i + 1
				reset()
			default:
				reset()
			}
		}
		b.WriteString(format[j:])
	}
	return b.String()
}

// Params returns the query parameters.
func (f *formatter) Params() []interface{} {
	var (
		params = make([]interface{}, 0, f.paramsCap())

		verbLabel  int
		verbSpread bool
		verbRecord bool

		reset = func() {
			verbLabel = 0
			verbSpread = false
			verbRecord = false
		}
	)
	for n, format := range f.format {
		var labels map[int]int
		for i, p := 0, 0; i < len(format); i++ {
			switch {
			case format[i] == '%':
				if verbRecord = !verbRecord; !verbRecord {
					reset()
				}
			case format[i] == '[' && verbRecord:
				if verbLabel, i = f.parseVerbIndex(format, i); verbLabel > 0 {
					verbSpread = false
					break
				}
				reset()
			case format[i] == '+' && verbRecord:
				if verbLabel == 0 {
					verbSpread = true
				}
			case format[i] == 's' && verbRecord:
				var idx = p
				if verbLabel > 0 {
					labels, idx = f.bindVerbIndex(labels, verbLabel, p)
				}
				f.paramsCheck(n, idx)
				if verbSpread {
					params = filters(params, f.params[n][p:]...)
					p = len(f.params[n])
				} else {
					params = filters(params, f.params[n][idx])
					p = advanceParam(p, idx)
				}
				reset()
			case format[i] == 'p' && verbRecord:
				var idx = p
				if verbLabel > 0 {
					labels, idx = f.bindVerbIndex(labels, verbLabel, p)
				}
				f.paramsCheck(n, idx)
				if verbSpread {
					params = insert(params, f.params[n][p:]...)
					p = len(f.params[n])
				} else {
					params = insert(params, f.params[n][idx])
					p = advanceParam(p, idx)
				}
				reset()
			default:
				reset()
			}
		}
	}
	return params
}

// Format appends another formatted query fragment to the formatter.
func (f *formatter) Format(format string, params ...interface{}) Formatter {
	f.params = append(f.params, params)
	f.format = append(f.format, format)
	return f
}

// Driver sets the driver.
func (f *formatter) Driver(driver Driver) Formatter {
	f.driver = driver
	f.master = true
	return f
}

// Jumper sets the string used to concatenate fragments.
// For example: " AND ", " OR ", ", "
func (f *formatter) Jumper(jumper string) Formatter {
	f.jumper = jumper
	return f
}

// writeString renders a parameter for the %s verb.
func (f *formatter) writeString(b *strings.Builder, n, p int, s bool) {
	switch s {
	case true:
		f.toString(b, f.params[n][p:])
	default:
		f.toString(b, f.params[n][p])
	}
}

// writePlaceholder renders a parameter for the %p verb.
func (f *formatter) writePlaceholder(b *strings.Builder, n, p int, s bool) {
	switch s {
	case true:
		b.WriteString(f.d().Placeholder(f.params[n][p:]))
	default:
		b.WriteString(f.d().Placeholder(f.params[n][p]))
	}
}

// d returns the active driver.
func (f *formatter) d() Driver {
	if f.driver == nil {
		f.driver = driver()
	}
	return f.driver
}

// m resets the driver unless the formatter uses a custom driver.
func (f *formatter) m() {
	if !f.master {
		f.driver = driver()
	}
}

// parseVerbIndex parses an indexed verb label and returns the label and the closing bracket index.
func (f *formatter) parseVerbIndex(format string, i int) (int, int) {
	var j = i + 1
	for i = i + 1; i < len(format) && format[i] >= '0' && format[i] <= '9'; {
		i = i + 1
	}
	if i >= len(format) || format[i] != ']' {
		return 0, i
	}
	return atoi(format[j:i]), i
}

// bindVerbIndex binds a verb label to the current parameter index.
func (f *formatter) bindVerbIndex(labels map[int]int, label, p int) (map[int]int, int) {
	if labels == nil {
		labels = map[int]int{}
	}
	if p, ok := labels[label]; ok {
		return labels, p
	}
	labels[label] = p
	return labels, p
}

// paramsCheck verifies that a parameter index exists.
func (f *formatter) paramsCheck(n, p int) {
	if p >= len(f.params[n]) {
		panic("qp: parameter not found")
	}
}

// valueStringCap estimates the string size for a value.
func (f *formatter) valueStringCap(x interface{}) int {
	if x, ok := x.(Formatter); ok {
		return x.Driver(f.d()).(*formatter).formatsCap()
	}
	return count(x) * 4
}

// valueParamsCap estimates the parameter count for a value.
func (f *formatter) valueParamsCap(x interface{}) int {
	if x, ok := x.(Formatter); ok {
		return x.(*formatter).paramsCap()
	}
	return count(x)
}

// formatsCap calculates the capacity needed for the query string.
func (f *formatter) formatsCap() int {
	var cap int
	for n := range f.format {
		cap += len(f.format[n])
		for _, v := range f.params[n] {
			cap += f.valueStringCap(v)
		}
	}
	cap += len(f.jumper) * max(len(f.format)-1, 0)
	return cap
}

// paramsCap calculates the capacity needed for the parameters slice.
func (f *formatter) paramsCap() int {
	var cap int
	for _, pp := range f.params {
		for _, v := range pp {
			cap += f.valueParamsCap(v)
		}
	}
	return cap
}

// toString converts a value to a string.
func (f *formatter) toString(b *strings.Builder, x interface{}) {
	var buf [64]byte
	switch x := x.(type) {
	case string:
		b.WriteString(x)
	case Formatter:
		b.WriteString(x.Driver(f.d()).String())
	case fmt.Stringer:
		b.WriteString(x.String())
	case int:
		b.Write(strconv.AppendInt(buf[:0], int64(x), 10))
	case int8:
		b.Write(strconv.AppendInt(buf[:0], int64(x), 10))
	case int16:
		b.Write(strconv.AppendInt(buf[:0], int64(x), 10))
	case int32:
		b.Write(strconv.AppendInt(buf[:0], int64(x), 10))
	case int64:
		b.Write(strconv.AppendInt(buf[:0], x, 10))
	case uint:
		b.Write(strconv.AppendUint(buf[:0], uint64(x), 10))
	case uint8:
		b.Write(strconv.AppendUint(buf[:0], uint64(x), 10))
	case uint16:
		b.Write(strconv.AppendUint(buf[:0], uint64(x), 10))
	case uint32:
		b.Write(strconv.AppendUint(buf[:0], uint64(x), 10))
	case uint64:
		b.Write(strconv.AppendUint(buf[:0], x, 10))
	case float32:
		b.Write(strconv.AppendFloat(buf[:0], float64(x), 'f', 6, 32))
	case float64:
		b.Write(strconv.AppendFloat(buf[:0], x, 'f', 6, 64))
	case []byte:
		b.Write(x)
	case []rune:
		b.WriteString(string(x))
	case []int:
		intsToString(b, x)
	case []int64:
		intsToString(b, x)
	case []string:
		stringsToString(b, x)
	case []interface{}:
		f.strings(b, x)
	case nil:
	default:
		b.WriteString(fmt.Sprint(x))
	}
}

// strings converts []interface{} to a string.
// For example: []interface{}{"name", "surname", []string{"age"}} => "name, surname, age"
func (f *formatter) strings(b *strings.Builder, x []interface{}) {
	var n int
	if n = len(x); n == 0 {
		return
	}
	f.toString(b, x[0])
	for i := 1; i < n; i++ {
		b.WriteString(", ")
		f.toString(b, x[i])
	}
}
