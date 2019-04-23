package qp

import (
	"unsafe"
)

type mysqlDriver struct{}

var _ Driver = (*mysqlDriver)(nil)

func init() {
	RegisterDriver("mysql", MysqlDriver)
}

// MysqlDriver returns a specific Driver for mysql
func MysqlDriver() Driver {
	return &mysqlDriver{}
}

// Placeholder returns n count placeholders
func (d *mysqlDriver) Placeholder(x interface{}) string {
	var n int
	if n = count(x); n == 1 {
		return "?"
	}

	var (
		p = ", ?"
		b = make([]byte, len(p)*n)
		w = copy(b, p)
	)

	for w < len(b) {
		copy(b[w:], b[:w])
		w *= 2
	}
	if len(b) >= len(p) {
		b = b[2:]
	}

	return *(*string)(unsafe.Pointer(&b))
}
