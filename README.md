# qp
[![Build Status](https://github.com/AlexanderGrom/qp/workflows/tests/badge.svg)](https://github.com/AlexanderGrom/qp/actions?workflow=tests) [![Go Report Card](https://goreportcard.com/badge/github.com/alexandergrom/qp)](https://goreportcard.com/report/github.com/alexandergrom/qp) [![GoDoc](https://godoc.org/github.com/alexandergrom/qp?status.svg)](https://godoc.org/github.com/alexandergrom/qp)

Package qp is a simple query formatter.

## Get the package
```bash
$ go get -u github.com/alexandergrom/qp
```

## Use

```go
var b = qp.
    Format("name = %p", "Tom").
    Format("age IN (%+p)", 18, 21, 30)

var q = qp.Query("SELECT id FROM table WHERE %s LIMIT %p", b, 10)
_ = q.String() // SELECT id FROM table WHERE name = $1 AND age IN ($2, $3, $4) LIMIT $5
_ = q.Params() // ["Tom", 18, 21, 30, 10]
```

## The verbs
```
%s		convert to string
%p		convert to one placeholder or slice placeholders
```

## The modifiers
```
+		capture all parameters
```

## Examples
```go
qp.Format("name: %s", "Tom Sawyer").String() // name: Tom Sawyer
qp.Format("name: %p", "Tom Sawyer").String() // name: $1
```

### Slice processing ([]int, []int64, []string, []interface{})
```go
qp.Format("fields: %s", []string{"id", "name", "age"}).String() // fields: id, name, age
qp.Format("params: %p", []string{"id", "name", "age"}).String() // params: $1, $2, $3

qp.Format("ints: %s", []int64{1, 2, 3}).String() // ints: 1, 2, 3
qp.Format("ints: %p", []int64{1, 2, 3}).String() // ints: $1, $2, $3
```

### Modifiers
```go
qp.Format("fields: %+s", "id", "name", "age").String() // fields: id, name, age
qp.Format("params: %+p", "id", "name", "age").String() // fields: $1, $2, $3
```

### Some more complicated examples
```go
qp.Format("params: %p", []interface{}{1, 2, 3, "four", []int{5, 6}}).String() // params: $1, $2, $3, $4, $5, $6
qp.Format("params: %+p", []int64{1, 2, 3}, 4).String() // params: $1, $2, $3, $4
```

### Other examples
```go
ids := []int64{1, 2, 3, 4, 5, 6}
query := qp.Format("SELECT name FROM users WHERE id IN (%p) LIMIT %p", ids, 10)
q := query.String() // SELECT name FROM users WHERE id IN ($1, $2, $3, $4, $5, $6) LIMIT $7
p := query.Params() // [1, 2, 3, 4, 5, 6, 10]
```

### Nested format
```go
filter := qp.
    Format("name = %p", "Tom").
    Format("age = %p", 12)
query := qp.Format("SELECT name FROM users WHERE %s LIMIT %p", filter, 10)
q := query.String() // SELECT name FROM users WHERE name = $1 AND age = $2 LIMIT $3
p := query.Params() // ["Tom", 12, 10]
```

### Update
```go
fields := []string{"name", "age"}
params := []interface{}{"Tom", 12}
query := qp.Format("UPDATE users SET (%s) = (%p) WHERE id = %p", fields, params, 1)
q := query.String() // UPDATE users SET (name, age) = ($1, $2) WHERE id = $3
p := query.Params() // ["Tom", 12, 1]
```

### Insert
```go
values := qp.
    Format("(%+p)", 1, "Tom", 12).
    Format("(%+p)", 2, "Huckleberry", 13).
    Jumper(", ")
query := qp.Format("INSERT INTO users (id, name, age) VALUES %s", values)
q := query.String() // INSERT INTO users (id, name, age) VALUES ($1, $2, $3), ($4, $5, $6)
p := query.Params() // [1, "Tom", 12, 2, "Huckleberry", 13]
```

### Insert
```go
values := qp.New().Jumper(", ")
for _, d := range data {
	values.Format("(%p)", d)
}
query := qp.Format("INSERT INTO users (id, name, age) VALUES %s", values)
q := query.String() // INSERT INTO users (id, name, age) VALUES ($1, $2, $3), ($4, $5, $6)
p := query.Params() // [1, "Tom", 12, 2, "Huckleberry", 13]
```

### Filter
```go
type (
    CarFilter struct {
        Mark      string
        Model     string
        Color     []int
        Price     []int
        Limit     int
        Offset    int
    }

    Car struct {
        Mark      string
        Model     string
        Color     int
        Price     int
        CreatedAt time.Time
        UpdatedAt time.Time
    }

    CarRepository struct {
        db *sql.DB
    }
)

func (r *CarRepository) GetByFilter(filter CarFilter) (_ []*Car, err error) {
    var builder = qp.Format("1=1")

    if len(filter.Mark) > 0 {
        builder.Format("mark = %p", filter.Mark)
    }

    if len(filter.Model) > 0 {
        builder.Format("model = %p", filter.Model)
    }

    if len(filter.Color) > 0 {
        builder.Format("color IN (%p)", colors)
    }

    if len(filter.Price) == 1 {
        builder.Format("price <= %p", filter.Price[0])
    } else if len(filter.Price) == 2 {
        builder.
            Format("price >= %p", filter.Price[0]).
            Format("price <= %p", filter.Price[1])
    }

    var query = qp.Format(`
        SELECT mark, model, color, price, created_at, updated_at
        FROM cars
        WHERE %s
        LIMIT %p
        OFFSET %p
    `, builder, filter.Limit, filter.Offset)

    var rows *sql.Rows
    if rows, err = r.db.Query(query.String(), query.Params()...); err != nil {
        return nil, err
    }
    defer rows.Close()

    var cars = make([]*Car, 0, filter.Limit)
    for rows.Next() {
        var car = new(Car)

        if err = rows.Scan(&car.Mark, &car.Model, &car.Color, &car.Price, &car.CreatedAt, &car.UpdatedAt); err != nil {
            return nil, err
        }

        cars = append(cars, car)
    }

    return cars, rows.Err()
}
```
