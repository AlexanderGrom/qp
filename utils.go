package qp

import (
	"strconv"
	"strings"
	"unsafe"
)

// IntWeight returns number of digits in an int
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

// IntsToString converts []int to string
// For example: []int{1,2,3,4} => "1, 2, 3, 4"
func intsToString(x []int) string {
	var n int
	if n = len(x); n == 0 {
		return ""
	}
	var b = make([]byte, 0, n*3-2)
	b = strconv.AppendInt(b, int64(x[0]), 10)
	for i := 1; i < n; i++ {
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, int64(x[i]), 10)
	}
	return *(*string)(unsafe.Pointer(&b))
}

// IntsToString converts []int64 to string
// For example: []int64{1,2,3,4} => "1, 2, 3, 4"
func int64sToString(x []int64) string {
	var n int
	if n = len(x); n == 0 {
		return ""
	}
	var b = make([]byte, 0, n*3-2)
	b = strconv.AppendInt(b, x[0], 10)
	for i := 1; i < n; i++ {
		b = append(b, ',', ' ')
		b = strconv.AppendInt(b, x[i], 10)
	}
	return *(*string)(unsafe.Pointer(&b))
}

// StringsToString converts []string to string
// For example: []string{"name", "surname", "age"} => "name, surname, age"
func stringsToString(x []string) string {
	var n int
	if n = len(x); n == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(x[0])
	for i := 1; i < n; i++ {
		b.WriteString(", ")
		b.WriteString(x[i])
	}
	return b.String()
}

// The btoi a helper function converts bool to int
func btoi(b bool) int {
	switch b {
	case true:
		return 1
	default:
		return 0
	}
}

// The filters a helper function returns count params
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

// The filters a helper function filters and appends only Formatter elements to the end of a slice params
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

// The insert a helper function appends elements to the end of a slice params
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
