package qp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPgSQL_Placeholder(t *testing.T) {
	var res string

	res = PgsqlDriver().Placeholder(1)
	assert.Equal(t, `$1`, res)

	res = PgsqlDriver().Placeholder([]byte{'a', 'b', 'c'})
	assert.Equal(t, `$1`, res)

	res = PgsqlDriver().Placeholder([]int{})
	assert.Equal(t, ``, res)

	res = PgsqlDriver().Placeholder([]int{1, 2})
	assert.Equal(t, `$1, $2`, res)

	res = PgsqlDriver().Placeholder([]int64{1, 2, 3})
	assert.Equal(t, `$1, $2, $3`, res)

	res = PgsqlDriver().Placeholder([]string{"Tom", "Sawyer"})
	assert.Equal(t, `$1, $2`, res)

	res = PgsqlDriver().Placeholder([]interface{}{1, "Tom", true})
	assert.Equal(t, `$1, $2, $3`, res)

	res = PgsqlDriver().Placeholder([]interface{}{[]interface{}{1, "Tom", true, []byte{'a', 'b', 'c'}}})
	assert.Equal(t, `$1, $2, $3, $4`, res)

	res = PgsqlDriver().Placeholder([]interface{}{[]int{1, 2}, []int64{3, 4, 5}, 6})
	assert.Equal(t, `$1, $2, $3, $4, $5, $6`, res)
}

func BenchmarkPgSQL_Placeholder(b *testing.B) {
	var d = PgsqlDriver()
	var s = []int64{1, 2, 3}
	for i := 0; i < b.N; i++ {
		_ = d.Placeholder(s)
	}
}
