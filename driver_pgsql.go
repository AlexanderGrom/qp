package qp

import (
	"strconv"
	"unsafe"
)

var pgsqlPreAlloc [128]string

type pgsqlDriver struct {
	placeholders int
}

var _ Driver = (*pgsqlDriver)(nil)

func init() {
	RegisterDriver("postgres", PgsqlDriver)
	pgsqlPreAllocate()
}

// PgsqlDriver returns a Driver for PostgreSQL.
func PgsqlDriver() Driver {
	return &pgsqlDriver{}
}

// placeholder increments and returns the next placeholder number.
func (d *pgsqlDriver) placeholder() int {
	d.placeholders++
	return d.placeholders
}

// next returns the next placeholder string.
func (d *pgsqlDriver) next() string {
	var p = d.placeholder()
	if p < len(pgsqlPreAlloc) {
		return pgsqlPreAlloc[p]
	}
	return "$" + strconv.Itoa(p)
}

// Placeholder returns one or more PostgreSQL placeholders.
func (d *pgsqlDriver) Placeholder(x interface{}) string {
	var n int
	switch n = count(x); n {
	case 0:
		return ""
	case 1:
		return d.next()
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

func pgsqlPreAllocate() {
	for i := 1; i < len(pgsqlPreAlloc); i++ {
		pgsqlPreAlloc[i] = "$" + strconv.Itoa(i)
	}
}
