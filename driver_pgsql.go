package qp

import (
	"strconv"
	"unsafe"
)

type pgsqlDriver struct {
	placeholders int
}

var _ Driver = (*pgsqlDriver)(nil)

func init() {
	RegisterDriver("postgres", PgsqlDriver)
}

// PgsqlDriver returns a specific Driver for postgresql
func PgsqlDriver() Driver {
	return &pgsqlDriver{}
}

// Placeholder returns n count placeholders
func (d *pgsqlDriver) placeholder() int {
	d.placeholders++
	return d.placeholders
}

// Placeholder returns string of placeholders
func (d *pgsqlDriver) Placeholder(x interface{}) string {
	var n int
	if n = count(x); n == 1 {
		return "$" + strconv.Itoa(d.placeholder())
	}

	var (
		sep = ", "
		cap = len(sep)*(n-1) + n
	)
	for i := 1; i <= n; i++ {
		cap += intWeight(d.placeholders + i)
	}

	var b = make([]byte, 0, cap)
	b = append(b, '$')
	b = strconv.AppendInt(b, int64(d.placeholder()), 10)
	for i := 1; i < n; i++ {
		b = append(b, ',', ' ', '$')
		b = strconv.AppendInt(b, int64(d.placeholder()), 10)
	}

	return *(*string)(unsafe.Pointer(&b))
}
