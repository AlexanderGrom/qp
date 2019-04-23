package qp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMySQL_Placeholder(t *testing.T) {
	var res string

	res = MysqlDriver().Placeholder(1)
	assert.Equal(t, `?`, res)

	res = MysqlDriver().Placeholder([]int{1, 2})
	assert.Equal(t, `?, ?`, res)

	res = MysqlDriver().Placeholder([]int64{1, 2, 3})
	assert.Equal(t, `?, ?, ?`, res)

	res = MysqlDriver().Placeholder([]string{"Tom", "Sawyer"})
	assert.Equal(t, `?, ?`, res)

	res = MysqlDriver().Placeholder([]interface{}{1, "Tom", true})
	assert.Equal(t, `?, ?, ?`, res)

	res = MysqlDriver().Placeholder([]interface{}{[]int{1, 2}, []int{3, 4, 5}, 6})
	assert.Equal(t, `?, ?, ?, ?, ?, ?`, res)
}

func BenchmarkMySQL_Placeholder(b *testing.B) {
	var d = MysqlDriver()
	var s = []int64{1, 2, 3}
	for i := 0; i < b.N; i++ {
		_ = d.Placeholder(s)
	}
}
