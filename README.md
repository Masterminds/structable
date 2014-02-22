# Sructable: Struct-Table Mapping for Go

This library provides basic struct-to-table mapping for Go.

It is based on the [Squirrel](https://github.com/lann/squirrel) library.

## What It Does

Structable maps a struct (`Record`) to a database table via a
`structable.Recorder`. It is intended to be used as a back-end tool for
building systems like Active Record mappers.

It is designed to satisfy a CRUD-centered record management system,
filling the following contract:

```go
type Recorder interface {
	Bind(string, Record) Recorder // link struct to table
	Insert() error // INSERT just one record
	Update() error // UPDATE just one record
	Delete() error // DELETE just one record
	Exists() (bool, error) // Check for just one record
	Load() error  // SELECT just one record
}

```

Squirrel already provides the ability to perform more complicated
operations.

## How It Does It

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
db := getDb() // You're on  the hook to do this part.

// Create a new structable.Recorder and tell it to
// bind the given struct as a row in the given table.
r := New(db).Bind("test_table", stool)

// This will insert the stool into the test_table.
err := r.Insert()
```

## What It Does Not Do

It does not...

* Create or manage schemas.
* Guess or enforce table or column names. (You have to tell it how to
  map.)
* Provide relational mapping.

## LICENSE

This software is licensed under an MIT-style license. See LICENSE.txt
