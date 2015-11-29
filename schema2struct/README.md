# schema2struct: Create definitions from the database

This program is a proof of concept for creating Structable structs by
inspecting a database and generating closely matching structs.

Currently this only works on Postgres, though there is no reason it
could not be ported to support other databases.

It works by querying the INFORMATION_SCHEMA tables to learn about what
tables are present and what columns they stored. It then attempts to
render structs that point to those tables.

If you are interested in contributing to moving this beyond proof of
concept, feel free to issue PRs against the codebase.

## Usage

Install using `make install`. This will put `schema2struct` on your
`$PATH`.

In the package where you want to create the structs, add an annotation
to one of the Go files:

```go
//go:generate schema2struct -f schemata.go
```

The above annotation will instruct `go generate` to run `schema2struct`
and generate a file called `schemata.go`.

Finally, run `go generate` in that  package's directory:

```
$ cd model
$ go generate
```

The result should be a `schemata.go` source file.
