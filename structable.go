/* Structable is a struct-to-table mapper for databases.

Structable makes a loose distinction between a Record (a description of the
data to be stored) and a Recorder (the thing that does the storing). A
Record is a simple annotated struct that describes the properties of an
object.

Structable provides the Recorder (an interface usually backed by a *DbRecorder).
The Recorder is capable of doing the following:

	- Bind: Attach the Recorder to a Record
	- Load: Load a Record from a database
	- Insert: Create a new Record
	- Update: Change one or more fields on a Record
	- Delete: Destroy a record in the database
	- Has: Determine whether a given Record exists in a database
	- LoadWhere: Load a record where certain conditions obtain.

Structable is pragmatic in the sense that it allows ActiveRecord-like extension
of the Record object to allow business logic. A Record does not *have* to be
a simple data-only struct. It can have methods -- even methods that operate
on the database.

Importantly, Structable does not do any relation management. There is no
magic to convert structs, arrays, or maps to references to other tables.
(If you want that, you may prefer GORM or GORP.) The preferred method of
handling relations is to attach additional methods to the Record struct.

Structable uses Squirrel for statement building, and you may also use
Squirrel for working with your data.

Basic Usage

The following example is taken from the `example/users.go` file.


	package main

	import (
		"github.com/Masterminds/squirrel"
		"github.com/Masterminds/structable"
		_ "github.com/lib/pq"

		"database/sql"
		"fmt"
	)

	// This is our struct. Notice that we make this a structable.Recorder.
	type User struct {
		structable.Recorder
		builder squirrel.StatementBuilderType

		Id int `stbl:"id,PRIMARY_KEY,SERIAL"`
		Name string `stbl:"name"`
		Email string `stbl:"email"`
	}

	// NewUser creates a new Structable wrapper for a user.
	//
	// Of particular importance, watch how we intialize the Recorder.
	func NewUser(db squirrel.DBProxyBeginner, dbFlavor string) *User {
		u := new(User)
		u.Recorder = structable.New(db, dbFlavor).Bind(UserTable, u)
		return u
	}

	func main() {

		// Boilerplate DB setup.
		// First, we need to know the database driver.
		driver := "postgres"
		// Second, we need a database connection.
		con, _ := sql.Open(driver, "dbname=structable_test sslmode=disable")
		// Third, we wrap in a prepared statement cache for better performance.
		cache := squirrel.NewStmtCacheProxy(con)

		// Create an empty new user and give it some properties.
		user := NewUser(cache, driver)
		user.Name = "Matt"
		user.Email = "matt@example.com"

		// Insert this as a new record.
		if err := user.Insert(); err != nil {
			panic(err.Error())
		}
		fmt.Printf("Initial insert has ID %d, name %s, and email %s\n", user.Id, user.Name, user.Email)

		// Now create another empty User and set the user's Name.
		again := NewUser(cache, driver)
		again.Id = user.Id

		// Load a duplicate copy of our user. This loads by the value of
		// again.Id
		again.Load()

		again.Email = "technosophos@example.com"
		if err := again.Update(); err != nil {
			panic(err.Error())
		}
		fmt.Printf("Updated user has ID %d and email %s\n", again.Id, again.Email)

		// Delete using the built-in Deleter. (delete by Id.)
		if err := again.Delete(); err != nil {
			panic(err.Error())
		}
		fmt.Printf("Deleted user %d\n", again.Id)
	}

The above pattern closely binds the Recorder to the Record. Essentially, in
this usage Structable works like an ActiveRecord.

It is also possible to emulate a DAO-type model and use the Recorder as a data
access object and the Record as the data description object. An example of this
method can be found in the `example/fence.go` code.

The Stbl Tag

The `stbl` tag is of the form:

	stbl:"field_name [,PRIMARY_KEY[,AUTO_INCREMENT]]"

The field name is passed verbatim to the database. So `fieldName` will go to the database as `fieldName`.
Structable is not at all opinionated about how you name your tables or fields. Some databases are, though, so
you may need to be careful about your own naming conventions.

`PRIMARY_KEY` tells Structable that this field is (one of the pieces of) the primary key. Aliases: 'PRIMARY KEY'

`AUTO_INCREMENT` tells Structable that this field is created by the database, and should never
be assigned during an Insert(). Aliases: SERIAL, AUTO INCREMENT

Limitations

Things Structable doesn't do (by design)

	- Guess table or column names. You must specify these.
	- Handle relations between tables.
	- Manage the schema.
	- Transform complex struct fields into simple ones (that is, serialize fields).

However, Squirrel can ease many of these tasks.

*/
package structable

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/Masterminds/squirrel"
)

// 'stbl' is the main tag used for annotating Structable Records.
const StructableTag = "stbl"

/* Record describes a struct that can be stored.

Example:

	type Stool struct {
		Id 		 int 	`stbl:"id PRIMARY_KEY AUTO_INCREMENT"`
		Legs 	 int    `stbl:"number_of_legs"`
		Material string `stbl:"material"`
		Ignored  string // will not be stored.
	}

The above links the Stool record to a database table that has a primary
key (with auto-incrementing values) called 'id', an int field named
'number_of_legs', and a 'material' field that is a VARCHAR or TEXT (depending
on the database implementation).

*/
type Record interface{}

// Internal representation of a field on a database table, and its
// relation to a struct field.
type field struct {
	// name = Struct field name
	// column = table column name
	name, column string
	// Is a primary key
	isKey bool
	// Is an auto increment
	isAuto bool
}

// A Recorder is responsible for managing the persistence of a Record.
// A Recorder is bound to a struct, which it then examines for fields
// that should be stored in the database. From that point on, a recorder
// can manage the persistent lifecycle of the record.
type Recorder interface {
	// Bind this Recorder to a table and to a Record.
	//
	// The table name is used verbatim. DO NOT TRUST USER-SUPPLIED VALUES.
	//
	// The struct is examined for tags, and those tags are parsed and used to determine
	// details about each field.
	Bind(string, Record) Recorder

	// Interface provides a way of fetching the record from the Recorder.
	//
	// A record is bound to a Recorder via Bind, and retrieved from a Recorder
	// via Interface().
	//
	// This is conceptually similar to reflect.Value.Interface().
	Interface() interface{}

	Loader
	Haecceity
	Saver
	Describer

	// This returns the column names used for the primary key.
	//Key() []string
}

type Loader interface {
	// Loads the entire Record using the value of the PRIMARY_KEY(s)
	// This will only fetch columns that are mapped on the bound Record. But you can think of it
	// as doing something like this:
	//
	// 	SELECT * FROM bound_table WHERE id=? LIMIT 1
	//
	// And then mapping the result to the currently bound Record.
	Load() error
	// Load by a WHERE-like clause. See Squirrel's Where(pred, args)
	LoadWhere(interface{}, ...interface{}) error
}

type Saver interface {
	// Insert inserts the bound Record into the bound table.
	Insert() error

	// Update updates all of the fields on the bound Record based on the PRIMARY_KEY fields.
	//
	// Essentially, it does something like this:
	// 	UPDATE bound_table SET every=?, field=?, but=?, keys=? WHERE primary_key=?
	Update() error

	// Deletes a Record based on its PRIMARY_KEY(s).
	Delete() error
}

// Haecceity indicates whether a thing exists.
//
// Actually, it is responsible for testing whether a thing exists, and is
// what we think it is.
type Haecceity interface {
	// Exists verifies that a thing exists and is of this type.
	// This uses the PRIMARY_KEY to verify that a record exists.
	Exists() (bool, error)
	// ExistsWhere verifies that a thing exists and is of the expected type.
	// It takes a WHERE clause, and it needs to gaurantee that at least one
	// record matches. It need not assure that *only* one item exists.
	ExistsWhere(interface{}, ...interface{}) (bool, error)
}

// Describer is a structable object that can describe its table structure.
type Describer interface {
	// Columns gets the columns on this table.
	Columns(bool) []string
	// FieldReferences gets references to the fields on this object.
	FieldReferences(bool) []interface{}
	// WhereIds returns a map of ID fields to (current) ID values.
	//
	// This is useful to quickly generate where clauses.
	WhereIds() map[string]interface{}

	// TableName returns the table name.
	TableName() string
	// Builder returns the builder
	Builder() *squirrel.StatementBuilderType
	// DB returns a DB-like handle.
	DB() squirrel.DBProxyBeginner

	Driver() string

	Init(d squirrel.DBProxyBeginner, flavor string)
}

// List returns a list of objects of the given kind.
//
// This runs a Select of the given kind, and returns the results.
func List(d Recorder, limit, offset uint64) ([]Recorder, error) {
	fn := func(desc Describer, query squirrel.SelectBuilder) (squirrel.SelectBuilder, error) {
		return query.Limit(limit).Offset(offset), nil
	}

	return ListWhere(d, fn)
}

// WhereFunc modifies a basic select operation to add conditions.
//
// Technically, conditions are not limited to adding where clauses. It will receive
// a select statement with the 'SELECT ... FROM tablename' portion composed already.
type WhereFunc func(desc Describer, query squirrel.SelectBuilder) (squirrel.SelectBuilder, error)

// ListWhere takes a Recorder and a query modifying function and executes a query.
//
// The WhereFunc will be given a SELECT d.Colsumns() FROM d.TableName() statement,
// and may modify it. Note that while joining is supported, changing the column
// list will have unpredictable side effects. It is advised that joins be done
// using Squirrel instead.
//
// This will return a list of Recorder objects, where the underlying type
// of each matches the underlying type of the passed-in 'd' Recorder.
func ListWhere(d Recorder, fn WhereFunc) ([]Recorder, error) {
	var tn string = d.TableName()
	var cols []string = d.Columns(false)
	buf := []Recorder{}

	// Base query
	q := d.Builder().Select(cols...).From(tn)

	// Allow the fn to modify our query
	var err error
	q, err = fn(d, q)
	if err != nil {
		return buf, err
	}

	rows, err := q.Query()
	if err != nil || rows == nil {
		return buf, err
	}
	defer rows.Close()

	v := reflect.Indirect(reflect.ValueOf(d))
	t := v.Type()
	for rows.Next() {
		nv := reflect.New(t)

		// Bind an empty base object. Basically, we fetch the object out of
		// the DbRecorder, and then construct an empty one.
		rec := reflect.New(reflect.Indirect(reflect.ValueOf(d.(*DbRecorder).record)).Type())
		nv.Interface().(Recorder).Bind(d.TableName(), rec.Interface())

		s := nv.Interface().(Recorder)
		s.Init(d.DB(), d.Driver())
		dest := s.FieldReferences(true)
		rows.Scan(dest...)
		buf = append(buf, s)
	}

	return buf, rows.Err()
}

// Implements the Recorder interface, and stores data in a DB.
type DbRecorder struct {
	builder *squirrel.StatementBuilderType
	db      squirrel.DBProxyBeginner
	table   string
	fields  []*field
	key     []*field
	record  Record
	flavor  string
}

func (d *DbRecorder) Interface() interface{} {
	return d.record
}

// New creates a new DbRecorder.
//
// (The squirrel.DBProxy interface defines the functions normal for a database connection
// or a prepared statement cache.)
func New(db squirrel.DBProxyBeginner, flavor string) *DbRecorder {
	d := new(DbRecorder)
	d.Init(db, flavor)
	return d
}

// Init initializes a DbRecorder
func (d *DbRecorder) Init(db squirrel.DBProxyBeginner, flavor string) {
	b := squirrel.StatementBuilder.RunWith(db)
	if flavor == "postgres" {
		b = b.PlaceholderFormat(squirrel.Dollar)
	}

	d.builder = &b
	d.db = db
	d.flavor = flavor
}

// TableName returns the table name of this recorder.
func (s *DbRecorder) TableName() string {
	return s.table
}

// DB returns the database (DBProxyBeginner) for this recorder.
func (s *DbRecorder) DB() squirrel.DBProxyBeginner {
	return s.db
}

// Builder returns the statement builder for this recorder.
func (s *DbRecorder) Builder() *squirrel.StatementBuilderType {
	return s.builder
}

// Driver returns the string name of the driver.
func (s *DbRecorder) Driver() string {
	return s.flavor
}

// Bind binds a DbRecorder to a Record.
//
// This takes a given structable.Record and binds it to the recorder. That means
// that the recorder will track all changes to the Record.
//
// The table name tells the recorder which database table to link this record
// to. All storage operations will use that table.
func (s *DbRecorder) Bind(tableName string, ar Record) Recorder {

	// "To be is to be the value of a bound variable." - W. O. Quine

	// Get the table name
	s.table = tableName

	// Get the fields
	s.scanFields(ar)

	s.record = ar

	return Recorder(s)
}

// Key gets the string names of the fields used as primary key.
func (s *DbRecorder) Key() []string {
	key := make([]string, len(s.key))

	for i, f := range s.key {
		key[i] = f.column
	}

	return key
}

// Load selects the record from the database and loads the values into the bound Record.
//
// Load uses the table's PRIMARY KEY(s) as the sole criterion for matching a
// record. Essentially, it is akin to `SELECT * FROM table WHERE primary_key = ?`.
//
// This modifies the Record in-place. Other than the primary key fields, any
// other field will be overwritten by the value retrieved from the database.
func (s *DbRecorder) Load() error {
	whereParts := s.WhereIds()
	dest := s.FieldReferences(false)

	q := s.builder.Select(s.colList(false, false)...).From(s.table).Where(whereParts)
	err := q.QueryRow().Scan(dest...)

	return err
}

// LoadWhere loads an object based on a WHERE clause.
//
// This can be used to define alternate loaders:
//
// 	func (s *MyStructable) LoadUuid(uuid string) error {
// 		return s.LoadWhere("uuid = ?", uuid)
// 	}
//
// This functions similarly to Load, but with the notable difference that
// it loads the entire object (it does not skip keys used to do the lookup).
func (s *DbRecorder) LoadWhere(pred interface{}, args ...interface{}) error {
	dest := s.FieldReferences(true)

	q := s.builder.Select(s.colList(true, true)...).From(s.table).Where(pred, args...)
	err := q.QueryRow().Scan(dest...)

	return err
}

// Exists returns `true` if and only if there is at least one record that matches the primary keys for this Record.
//
// If the primary key on the Record has no value, this will look for records with no value (or the default
// value).
func (s *DbRecorder) Exists() (bool, error) {
	has := false
	whereParts := s.WhereIds()

	q := s.builder.Select("COUNT(*) > 0").From(s.table).Where(whereParts)
	err := q.QueryRow().Scan(&has)

	return has, err
}

// ExistsWhere returns `true` if and only if there is at least one record that matches one (or multiple) conditions.
//
// Conditions are expressed in the form of predicates and expected values
// that together build a WHERE clause. See Squirrel's Where(pred, args)
func (s *DbRecorder) ExistsWhere(pred interface{}, args ...interface{}) (bool, error) {
	has := false

	q := s.builder.Select("COUNT(*) > 0").From(s.table).Where(pred, args...)
	err := q.QueryRow().Scan(&has)

	return has, err
}

// Delete deletes the record from the underlying table.
//
// The fields on the present record will remain set, but not saved in the database.
func (s *DbRecorder) Delete() error {
	wheres := s.WhereIds()
	q := s.builder.Delete(s.table).Where(wheres)
	_, err := q.Exec()
	return err
}

// Insert puts a new record into the database.
//
// This operation is particularly sensitive to DB differences in cases where AUTO_INCREMENT is set
// on a member of the Record.
func (s *DbRecorder) Insert() error {
	switch s.flavor {
	case "postgres":
		return s.insertPg()
	default:
		return s.insertStd()
	}
}

// Insert and assume that LastInsertId() returns something.
func (s *DbRecorder) insertStd() error {

	cols, vals := s.colValLists(true, false)

	q := s.builder.Insert(s.table).Columns(cols...).Values(vals...)

	ret, err := q.Exec()
	if err != nil {
		return err
	}

	for _, f := range s.fields {
		if f.isAuto {
			ar := reflect.Indirect(reflect.ValueOf(s.record))
			field := ar.FieldByName(f.name)

			id, err := ret.LastInsertId()
			if err != nil {
				return fmt.Errorf("Could not get last insert ID. Did you set the db flavor? %s", err)
			}

			if !field.CanSet() {
				return fmt.Errorf("Could not set %s to returned value", f.name)
			}
			field.SetInt(id)
		}
	}

	return err
}

// insertPg runs a postgres-specific INSERT. Unlike the default (MySQL) driver,
// this actually refreshes ALL of the fields on the Record object. We do this
// because it is trivially easy in Postgres.
func (s *DbRecorder) insertPg() error {
	cols, vals := s.colValLists(true, false)
	dest := s.FieldReferences(true)
	q := s.builder.Insert(s.table).Columns(cols...).Values(vals...).
		Suffix("RETURNING " + strings.Join(s.colList(true, false), ","))

	sql, vals, err := q.ToSql()
	if err != nil {
		return err
	}

	return s.db.QueryRow(sql, vals...).Scan(dest...)
}

// Update updates the values on an existing entry.
//
// This updates records where the Record's primary keys match the record in the
// database. Essentially, it runs `UPDATE table SET names=values WHERE id=?`
//
// If no entry is found, update will NOT create (INSERT) a new record.
func (s *DbRecorder) Update() error {
	whereParts := s.WhereIds()
	updates := s.updateFields()
	q := s.builder.Update(s.table).SetMap(updates).Where(whereParts)
	_, err := q.Exec()
	return err
}

// Columns returns the names of the columns on this table.
//
// If includeKeys is false, the columns that are marked as keys are omitted
// from the returned list.
func (s *DbRecorder) Columns(includeKeys bool) []string {
	return s.colList(includeKeys, false)
}

// colList gets a list of column names. If withKeys is false, columns that are
// designated as primary keys will not be returned in this list.
// If omitNil is true, a column represented by pointer will be omitted if this
// pointer is nil in current record
func (s *DbRecorder) colList(withKeys bool, omitNil bool) []string {
	names := make([]string, 0, len(s.fields))

	var ar reflect.Value
	if omitNil {
		ar = reflect.Indirect(reflect.ValueOf(s.record))
	}

	for _, field := range s.fields {
		if !withKeys && field.isKey {
			continue
		}
		if omitNil {
			f := ar.FieldByName(field.name)
			if f.Kind() == reflect.Ptr && f.IsNil() {
				continue
			}
		}
		names = append(names, field.column)
	}

	return names
}

// FieldReferences returns a list of references to fields on this object.
//
// If withKeys is true, fields that compose the primary key will also be
// included. Otherwise, only non-primary key fields will be included.
//
// This is used for processing SQL results:
//
//	dest := s.FieldReferences(false)
//	q := s.builder.Select(s.Columns(false)...).From(s.table)
//	err := q.QueryRow().Scan(dest...)
func (s *DbRecorder) FieldReferences(withKeys bool) []interface{} {
	refs := make([]interface{}, 0, len(s.fields))

	ar := reflect.Indirect(reflect.ValueOf(s.record))
	for _, field := range s.fields {
		if !withKeys && field.isKey {
			continue
		}

		fv := ar.FieldByName(field.name)
		var ref reflect.Value
		if fv.Kind() != reflect.Ptr {
			// we want the address of field
			ref = fv.Addr()
		} else {
			// we already have an address
			ref = fv
			if fv.IsNil() {
				// allocate a new element of same type
				fv.Set(reflect.New(fv.Type().Elem()))
			}
		}
		refs = append(refs, ref.Interface())
	}

	return refs
}

// colValLists returns 2 lists, the column names and values.
// If withKeys is false, columns and values of fields designated as primary keys
// will not be included in those lists. Also, if withAutos is false, the returned
// lists will not include fields designated as auto-increment.
func (s *DbRecorder) colValLists(withKeys, withAutos bool) (columns []string, values []interface{}) {
	ar := reflect.Indirect(reflect.ValueOf(s.record))

	for _, field := range s.fields {

		switch {
		case !withKeys && field.isKey:
			continue
		case !withAutos && field.isAuto:
			continue
		}

		// Get the value of the field we are going to store.
		f := ar.FieldByName(field.name)
		var v reflect.Value
		if f.Kind() == reflect.Ptr {
			if f.IsNil() {
				// nothing to store
				continue
			}
			// no indirection: the field is already a reference to its value
			v = f
		} else {
			// get the value pointed to by the field
			v = reflect.Indirect(f)
		}

		values = append(values, v.Interface())
		columns = append(columns, field.column)
	}

	return
}

// updateFields produces fields to go into SetMap for an update.
// This will NOT update PRIMARY_KEY fields.
func (s *DbRecorder) updateFields() map[string]interface{} {
	update := map[string]interface{}{}
	cols, vals := s.colValLists(false, true)
	for i, col := range cols {
		update[col] = vals[i]
	}
	return update
}

// WhereIds gets a list of names and a list of values for all columns marked as primary
// keys.
func (s *DbRecorder) WhereIds() map[string]interface{} {
	clause := make(map[string]interface{}, len(s.key))

	ar := reflect.Indirect(reflect.ValueOf(s.record))

	for _, f := range s.key {
		clause[f.column] = ar.FieldByName(f.name).Interface()
	}

	return clause
}

// scanFields extracts the tags from all of the fields on a struct.
func (s *DbRecorder) scanFields(ar Record) {
	v := reflect.Indirect(reflect.ValueOf(ar))
	t := v.Type()
	count := t.NumField()
	keys := make([]*field, 0, 2)

	for i := 0; i < count; i++ {
		f := t.Field(i)
		// Skip fields with no tag.
		if len(f.Tag) == 0 {
			continue
		}
		sqtag := f.Tag.Get("stbl")
		if len(sqtag) == 0 {
			continue
		}

		parts := s.parseTag(f.Name, sqtag)
		field := new(field)
		field.name = f.Name
		field.column = parts[0]
		for _, part := range parts[1:] {
			part = strings.TrimSpace(part)
			switch part {
			case "PRIMARY_KEY", "PRIMARY KEY":
				field.isKey = true
				keys = append(keys, field)
			case "AUTO_INCREMENT", "SERIAL", "AUTO INCREMENT":
				field.isAuto = true
			}
		}
		s.fields = append(s.fields, field)
		s.key = keys
	}
}

// parseTag parses the contents of a stbl tag.
func (s *DbRecorder) parseTag(fieldName, tag string) []string {
	parts := strings.Split(tag, ",")
	if len(parts) == 0 {
		return []string{fieldName}
	}
	return parts
}
