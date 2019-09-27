package qp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatter_Format(t *testing.T) {
	b := Format("name = %p", "Tom").Format("subname = %p", "Sawyer")
	q := Format(
		"SELECT id FROM table WHERE status = %p AND %s ORDER BY %s %s LIMIT %p OFFSET %p",
		"active", b, "id", "desc", 10, 0,
	)
	assert.Equal(t,
		`SELECT id FROM table WHERE status = $1 AND name = $2 AND subname = $3 ORDER BY id desc LIMIT $4 OFFSET $5`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", "Tom", "Sawyer", 10, 0},
		q.Params(),
	)
}

func TestFormatter_Spread(t *testing.T) {
	b := Format("id IN (%+p)", 1, 2, 3).Format("subid IN (%p)", []int{1, 2, 3}).Jumper(" OR ")
	q := Format(
		"SELECT id FROM table WHERE status = %p AND (%s)",
		"active", b,
	)
	assert.Equal(t,
		`SELECT id FROM table WHERE status = $1 AND (id IN ($2, $3, $4) OR subid IN ($5, $6, $7))`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", 1, 2, 3, 1, 2, 3},
		q.Params(),
	)
}

func TestFormatter_DefaultDriver(t *testing.T) {
	DefaultDriver("mysql")
	b := Format("name = %p", "Tom")
	q := Format(
		"SELECT id FROM table WHERE status = %p AND %s LIMIT %p, OFFSET %p",
		"active", b, 10, 0,
	)
	assert.Equal(t,
		"SELECT id FROM table WHERE status = ? AND name = ? LIMIT ?, OFFSET ?",
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", "Tom", 10, 0},
		q.Params(),
	)
	DefaultDriver("postgres")
}

func TestFormatter_Driver(t *testing.T) {
	b := Format("name = %p", "Tom")
	q := Format(
		"SELECT id FROM table WHERE status = %p AND %s LIMIT %p, OFFSET %p",
		"active", b, 10, 0,
	).Driver(MysqlDriver())
	assert.Equal(t,
		"SELECT id FROM table WHERE status = ? AND name = ? LIMIT ?, OFFSET ?",
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", "Tom", 10, 0},
		q.Params(),
	)
}

func TestFormatter_SelectWhereIn1(t *testing.T) {
	b := Format("id IN (%p)", []int{1, 2, 3})
	q := Format(
		"SELECT name FROM table WHERE status = %p AND %s LIMIT %p",
		"active", b, 10,
	)
	assert.Equal(t,
		`SELECT name FROM table WHERE status = $1 AND id IN ($2, $3, $4) LIMIT $5`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", 1, 2, 3, 10},
		q.Params(),
	)
}

func TestFormatter_SelectWhereIn2(t *testing.T) {
	b := Format("id IN (%+p)", 1, 2, 3)
	q := Format(
		"SELECT name FROM table WHERE status = %p AND %s LIMIT %p",
		"active", b, 10,
	)
	assert.Equal(t,
		`SELECT name FROM table WHERE status = $1 AND id IN ($2, $3, $4) LIMIT $5`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", 1, 2, 3, 10},
		q.Params(),
	)
}

func TestFormatter_Insert(t *testing.T) {
	c := []string{"id", "name", "age"}
	b := Format("(%+p)", 1, "Tom", 12).Format("(%p)", []interface{}{2, "Huck", 13}).Jumper(", ")
	q := Format(
		"INSERT INTO table (%s) VALUES %s", c, b,
	)
	assert.Equal(t,
		`INSERT INTO table (id, name, age) VALUES ($1, $2, $3), ($4, $5, $6)`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{1, "Tom", 12, 2, "Huck", 13},
		q.Params(),
	)
}

func TestFormatter_NewInsert(t *testing.T) {
	d := [][]interface{}{
		{1, "Tom", 12},
		{2, "Huckleberry", 13},
	}
	v := New().Jumper(", ")
	for _, p := range d {
		v.Format("(%p)", p)
	}
	q := Format(
		"INSERT INTO users (id, name, age) VALUES %s", v,
	)
	assert.Equal(t,
		`INSERT INTO users (id, name, age) VALUES ($1, $2, $3), ($4, $5, $6)`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{1, "Tom", 12, 2, "Huckleberry", 13},
		q.Params(),
	)
}

func TestFormatter_Update1(t *testing.T) {
	q := Format(
		"UPDATE table SET (name, age) = (%p) WHERE id = %p",
		[]interface{}{"Tom", 12}, 1,
	)
	assert.Equal(t,
		`UPDATE table SET (name, age) = ($1, $2) WHERE id = $3`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"Tom", 12, 1},
		q.Params(),
	)
}

func TestFormatter_Update2(t *testing.T) {
	b := Format("%+p", "Tom", 12)
	q := Format(
		"UPDATE table SET (name, age) = (%s) WHERE id = %p",
		b, 1,
	)
	assert.Equal(t,
		`UPDATE table SET (name, age) = ($1, $2) WHERE id = $3`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"Tom", 12, 1},
		q.Params(),
	)
}

func TestFormatter_Update3(t *testing.T) {
	b := Format("%+s", "name", "age")
	q := Format(
		"UPDATE table SET (%s) = (%p, %p) WHERE id = %p",
		b, "Tom", 12, 1,
	)
	assert.Equal(t,
		`UPDATE table SET (name, age) = ($1, $2) WHERE id = $3`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"Tom", 12, 1},
		q.Params(),
	)
}

func TestFormatter_Stringer1(t *testing.T) {
	b1 := Format("%p, %p", "Tom", 12)
	b2 := Format("%s", b1)
	q := Format(
		"TEST %s %p",
		b2, 1,
	)
	assert.Equal(t,
		`TEST $1, $2 $3`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"Tom", 12, 1},
		q.Params(),
	)
}

func TestFormatter_Stringer2(t *testing.T) {
	b1 := Format("%p, %p", "Tom", 12)
	b2 := Format("%s", []interface{}{b1})
	q := Format(
		"TEST %s %p",
		b2, 1,
	)
	assert.Equal(t,
		`TEST $1, $2 $3`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"Tom", 12, 1},
		q.Params(),
	)
}

func TestFormatter_Idempotency(t *testing.T) {
	b := Format("id IN (%+p)", 1, 2, 3)
	q := Format(
		"SELECT name FROM table WHERE status = %p AND %s LIMIT %p",
		"active", b, 10,
	)
	assert.Equal(t,
		`SELECT name FROM table WHERE status = $1 AND id IN ($2, $3, $4) LIMIT $5`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", 1, 2, 3, 10},
		q.Params(),
	)
	assert.Equal(t,
		`SELECT name FROM table WHERE status = $1 AND id IN ($2, $3, $4) LIMIT $5`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"active", 1, 2, 3, 10},
		q.Params(),
	)
	assert.Equal(t,
		[]interface{}{"active", 1, 2, 3, 10},
		q.Params(),
	)
	assert.Equal(t,
		`SELECT name FROM table WHERE status = $1 AND id IN ($2, $3, $4) LIMIT $5`,
		q.String(),
	)
}

func TestFormatter_Verbs(t *testing.T) {
	b := Format("(%+p)", 1, 2, 3).Format("(%p)", []int{4, 5, 6}).Jumper(", ")
	q := Format(
		"%s, %s, %s, %p, %p",
		"status", []string{"active", "passive"}, b, 789, []string{"one", "two"},
	)
	assert.Equal(t,
		`status, active, passive, ($1, $2, $3), ($4, $5, $6), $7, $8, $9`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{1, 2, 3, 4, 5, 6, 789, "one", "two"},
		q.Params(),
	)
}

func TestFormatter_Verbs2(t *testing.T) {
	b := Format("%p", 1).
		Format("%+p", []int{2, 3, 4}).
		Format("%s", []int{5, 6, 7}).
		Format("%+s", 8, 9).
		Jumper(", ")
	q := Format(
		"%s", b,
	)
	assert.Equal(t,
		`$1, $2, $3, $4, 5, 6, 7, 8, 9`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{1, 2, 3, 4},
		q.Params(),
	)
}

func TestFormatter_Verbs3(t *testing.T) {
	q := Format(
		"%p", []interface{}{1, 2, 3, "four", []int{5, 6}},
	)
	assert.Equal(t,
		`$1, $2, $3, $4, $5, $6`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{1, 2, 3, "four", 5, 6},
		q.Params(),
	)
}

func TestFormatter_Verbs4(t *testing.T) {
	q := Format(
		"%p, %s, %%s, %%, %%+s, %+%s, %%p, %+%p, %+++s, %%p",
		"one", "two", 1, 2, 3,
	)
	assert.Equal(t,
		`$1, two, %s, %, %+s, %+%s, %p, %+%p, 1, 2, 3, %p`,
		q.String(),
	)
	assert.Equal(t,
		[]interface{}{"one"},
		q.Params(),
	)
}

func TestFormatter_Verbs5(t *testing.T) {
	var q Formatter

	q = Format("%%s%s", "one")
	assert.Equal(t,
		`%sone`,
		q.String(),
	)

	q = Format("%++%s%s", "one")
	assert.Equal(t,
		`%++%sone`,
		q.String(),
	)
}

func TestUtils_toString(t *testing.T) {
	var testCases = []struct {
		name   string
		input  interface{}
		output string
	}{
		{
			name:   "case_string",
			input:  "string",
			output: "string",
		}, {
			name:   "case_stringer",
			input:  Format("string"),
			output: "string",
		}, {
			name:   "case_int8",
			input:  int8(123),
			output: "123",
		}, {
			name:   "case_int16",
			input:  int16(123),
			output: "123",
		}, {
			name:   "case_int32",
			input:  int32(123456),
			output: "123456",
		}, {
			name:   "case_int64",
			input:  int64(123456),
			output: "123456",
		}, {
			name:   "case_uint",
			input:  uint(123456),
			output: "123456",
		}, {
			name:   "case_uint8",
			input:  uint8(123),
			output: "123",
		}, {
			name:   "case_uint16",
			input:  uint16(123),
			output: "123",
		}, {
			name:   "case_uint32",
			input:  uint32(123456),
			output: "123456",
		}, {
			name:   "case_uint64",
			input:  uint64(123456),
			output: "123456",
		}, {
			name:   "case_bytes",
			input:  []byte("string"),
			output: "string",
		}, {
			name:   "case_runes",
			input:  []rune("string"),
			output: "string",
		}, {
			name:   "case_ints",
			input:  []int{1, 2, 3, 4, 5, 6},
			output: "1, 2, 3, 4, 5, 6",
		}, {
			name:   "case_ints64",
			input:  []int64{1, 2, 3, 4, 5, 6},
			output: "1, 2, 3, 4, 5, 6",
		}, {
			name:   "case_strings",
			input:  []string{"s", "t", "r", "i", "n", "g"},
			output: "s, t, r, i, n, g",
		}, {
			name:   "case_interfaces",
			input:  []interface{}{1, "s", []int{2, 3}, []interface{}{4, "t", []int64{5, 6}}},
			output: "1, s, 2, 3, 4, t, 5, 6",
		}, {
			name:   "case_nil",
			input:  nil,
			output: "",
		},
	}

	var format = new(formatter)
	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			var output = format.toString(tt.input)
			assert.Equal(t, tt.output, output)
		})
	}
}

func BenchmarkBuilder_FormatString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var b = Format("name = %p", "Tom").
			Format("age = %p", []int64{18, 21, 30})

		_ = Format(`SELECT id FROM table WHERE %s LIMIT %p`, b, 10).String()
	}
}

func BenchmarkBuilder_FormatParams(b *testing.B) {
	for i := 0; i < b.N; i++ {
		var b = Format("name = %p", "Tom").
			Format("age = %p", []int64{18, 21, 30})

		_ = Format(`SELECT id FROM table WHERE %s LIMIT %p`, b, 10).Params()
	}
}
