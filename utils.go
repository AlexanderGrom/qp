package qp

import (
	"strconv"
	"strings"
)

// intWeight returns the number of digits in an int.
func intWeight(x int) int {
	var p = 10
	for i := 1; i < 19; i++ {
		if x < p {
			return i
		}
		p *= 10
	}
	return 19
}

// intsToString converts []int or []int64 to a string.
// For example: []int{1, 2, 3, 4} => "1, 2, 3, 4"
func intsToString[T int | int64](b *strings.Builder, x []T) {
	var n int
	if n = len(x); n == 0 {
		return
	}
	var buf [32]byte
	b.Write(strconv.AppendInt(buf[:0], int64(x[0]), 10))
	for i := 1; i < n; i++ {
		b.WriteString(", ")
		b.Write(strconv.AppendInt(buf[:0], int64(x[i]), 10))
	}
}

// stringsToString converts []string to a string.
// For example: []string{"name", "surname", "age"} => "name, surname, age"
func stringsToString(b *strings.Builder, x []string) {
	var n int
	if n = len(x); n == 0 {
		return
	}
	b.WriteString(x[0])
	for i := 1; i < n; i++ {
		b.WriteString(", ")
		b.WriteString(x[i])
	}
}

// atoi converts a string to an int.
func atoi(s string) int {
	var num, err = strconv.Atoi(s)
	return ternary(err == nil, num, 0)
}

// advanceParam verifies index and returns the next index.
func advanceParam(p, idx int) int {
	if idx == p {
		return p + 1
	}
	return p
}

// count returns the number of placeholders needed for a value.
func count(x interface{}) int {
	switch x := x.(type) {
	case []int:
		return len(x)
	case []int64:
		return len(x)
	case []string:
		return len(x)
	case []interface{}:
		var n int
		for _, x := range x {
			n += count(x)
		}
		return n
	default:
		return 1
	}
}

// filters appends only Formatter-derived params to params.
func filters(params []interface{}, args ...interface{}) []interface{} {
	for _, x := range args {
		switch x := x.(type) {
		case Formatter:
			params = append(params, x.Params()...)
		case []interface{}:
			params = filters(params, x...)
		}
	}
	return params
}

// insert appends values to params, flattening supported slice types.
func insert(params []interface{}, args ...interface{}) []interface{} {
	for _, x := range args {
		switch x := x.(type) {
		case []int:
			for _, x := range x {
				params = append(params, x)
			}
		case []int64:
			for _, x := range x {
				params = append(params, x)
			}
		case []string:
			for _, x := range x {
				params = append(params, x)
			}
		case []interface{}:
			params = insert(params, x...)
		default:
			params = append(params, x)
		}
	}
	return params
}

// ternary returns "a" if "t" is true, and "b" otherwise.
func ternary[T any](t bool, a T, b T) T {
	if t {
		return a
	}
	return b
}
