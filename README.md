# Structable: Struct-Table Mapping for Go
[![Stability:
Sustained](https://masterminds.github.io/stability/sustained.svg)](https://masterminds.github.io/stability/sustained.html)

**Warning:** This is the Structable 4 development branch. For a stable
release, use version 3.1.0. Structable development happens very slowly.

This library provides basic struct-to-table mapping for Go.

It is based on the [Squirrel](https://github.com/Masterminds/squirrel) library.

## What It Does

Structable maps a struct (`Record`) to a database table via a
`structable.Recorder`. It is intended to be used as a back-end tool for
building systems like Active Record mappers.

It is designed to satisfy a CRUD-centered record management system,
filling the following contract:

```go
  type Recorder interface {
    Bind(string, Record) Recorder // link struct to table
    Interface() interface{}  // Get the struct that has been linked
    Insert() error // INSERT just one record
    Update() error // UPDATE just one record
    Delete() error // DELETE just one record
    Exists() (bool, error) // Check for just one record
    ExistsWhere(cond interface{}, args ...interface{}) (bool, error)
    Load() error  // SELECT just one record
    LoadWhere(cond interface{}, args ...interface{}) error // Alternate Load()
  }
```

Squirrel already provides the ability to perform more complicated
operations.

## How To Install It

The usual way...

```
$ glide get github.com/Masterminds/structable
$ # or...
$ go get github.com/Masterminds/structable
```

And import it via:

```
import "github.com/Masterminds/structable"
```

## How To Use It

[![GoDoc](https://godoc.org/github.com/Masterminds/structable?status.png)](https://godoc.org/github.com/Masterminds/structable)

Structable works by mapping a struct to columns in a database.

To annotate a struct, you do something like this:

```go
  type Stool struct {
    Id		 int	`stbl:"id, PRIMARY_KEY, AUTO_INCREMENT"`
    Legs	 int    `stbl:"number_of_legs"`
    Material string `stbl:"material"`
    Ignored  string // will not be stored. No tag.
  }
```

To manage instances of this struct, you do something like this:

```go
  stool := new(Stool)
  stool.Material = "Wood"
  db := getDb() // Get a sql.Db. You're on  the hook to do this part.

  // Create a new structable.Recorder and tell it to
  // bind the given struct as a row in the given table.
  r := structable.New(db, "mysql").Bind("test_table", stool)

  // This will insert the stool into the test_table.
  err := r.Insert()
```

And of course you have `Load()`, `Update()`, `Delete()` and so on.

The target use case for Structable is to use it as a backend for an
Active Record pattern. An example of this can be found in the
`structable_test.go` file

Most of Structable focuses on individual objects, but there are helpers
for listing objects:

```go
// Get a list of things that have the same type as object.
stool := new(Stool)
items, err := structable.List(stool, offset, limit)

// Customize a list of things that have the same type as object.
fn = func(object structable.Describer, sql squirrel.SelectBuilder) (squirrel.SelectBuilder, error) {
  return sql.Limit(10), nil
}
items, err := structable.ListWhere(stool, fn)
```

For example, here is a function that uses `ListWhere` to get collection
of definitions from a table described in a struct named `Table`:

```go
func (s *SchemaInfo) Tables() ([]*Table, error) {

  // Bind a new recorder. We use an empty object just to get the field
  // data for that struct.
	t := &Table{}
	st := structable.New(s.Queryer, s.Driver).Bind(t.TableName(), t)

  // We want to return no more than 10 of these.
	fn := func(d structable.Describer, q squirrel.SelectBuilder) (squirrel.SelectBuilder, error) {
		return q.Limit(10), nil
	}

  // Fetch a list of Table structs.
	items, err := structable.ListWhere(st, fn)
	if err != nil {
		return []*Table{}, err
	}

  // Because we get back a []Recorder, we need to get the original data
  // back out. We have to manually convert it back to its real type.
	tables := make([]*Table, len(items))
	for i, item := range items {
		tables[i] = item.Interface().(*Table)
	}
	return tables, nil
}
```

### Tested On

- MySQL (5.5)
- PostgreSQL (9.3, 9.4, 9.6)
- SQLite 3

## What It Does Not Do

It does not...

* Create or manage schemas.
* Guess or enforce table or column names. (You have to tell it how to
  map.)
* Provide relational mapping.
* Handle bulk operations (use Squirrel for that)

## LICENSE

This software is licensed under an MIT-style license. See LICENSE.txt
