package structable

import (
	"github.com/lann/squirrel"
	"reflect"
	"strings"
	"fmt"
)

const SquirrelrmTag = "sqrl"

/* ActiveRecord describes a struct that can be stored.

Example:
	type Stool struct {
		Id 		 int 	`sqrl:"id PRIMARY_KEY AUTO_INCREMENT"`
		Legs 	 int    `sqrl:"number_of_legs"`
		Material string `sqrl:"material"`
		Ignored  string // will not be stored.
	}


*/
type ActiveRecord interface {}

type Field struct {
	name, column string
	// Is a primary key
	isKey bool
	// Is an auto increment
	isAuto bool
}

type Recorder interface {
	// Bind a tble name to an active record.
	Bind(string, ActiveRecord) Recorder
	Insert() error
	Update() error
	Delete() error
	Has() (bool, error)
	Load() error
	Key() []string
}

// Implements the Recorder interface, and stores data in a DB.
type DbRecorder struct {
	builder *squirrel.StatementBuilderType
	db squirrel.DBProxy
	table string
	fields []*Field
	key []*Field
	record ActiveRecord
}

func NewDbRecorder(db squirrel.DBProxy/*builder *squirrel.StatementBuilderType*/) *DbRecorder {
	b := squirrel.StatementBuilder.RunWith(db)
	r := new(DbRecorder)
	r.builder = &b
	r.db = db

	return r
}

// Bind binds this particular instance to a particular record.
func (s *DbRecorder) Bind(tableName string, ar ActiveRecord) Recorder {
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

func (s *DbRecorder) Load() error {
	whereParts := s.whereIds()

	q := s.builder.Select(s.colList(false)...).From(s.table).Where(whereParts)
	err := q.QueryRow().Scan(s.fieldReferences(false)...)

	return err
}

func (s *DbRecorder) Has() (bool, error) {
	has := 0
	whereParts := s.whereIds()

	q := s.builder.Select("COUNT(*)").From(s.table).Where(whereParts)
	err := q.QueryRow().Scan(has)

	return (has > 0), err
}

func (s *DbRecorder) Delete() error {
	// XXX: Change this when Squirrel has a Delete().
	wheres := s.whereIds()
	where := make([]string, 0, len(wheres))
	vals := make([]interface{}, 0, len(wheres))
	for k, v := range wheres {
		where = append(where, fmt.Sprintf("%s = ?", k))
		vals = append(vals, v)
	}
	sql := fmt.Sprintf("DELETE FROM %s WHERE %s", s.table, strings.Join(where, " AND "))
	_, err := s.db.Exec(sql, vals...)
	return err
}

func (s *DbRecorder) Insert() error {
	cols, vals := s.insertFields()
	q := s.builder.Insert(s.table).Columns(cols...).Values(vals...)

	ret, err := q.Exec()

	for _, f := range s.fields {
		if f.isAuto {
			ar := reflect.Indirect(reflect.ValueOf(s.record))
			field := ar.FieldByName(f.name)
			id, _ := ret.LastInsertId()
			if !field.CanSet() {
				return fmt.Errorf("Could not set %s to returned value", f.name)
			}
			field.SetInt(id)
		}
			
	
	}

	return err
}

func (s *DbRecorder) Update() error {
	return nil
}

// colList gets a list of column names. If withKeys is false, columns that are
// designated as primary keys will not be returned in this list.
func (s *DbRecorder) colList(withKeys bool) []string {
	names := make([]string, 0, len(s.fields))

	for _, f := range s.fields {
		if !withKeys && f.isKey {
			continue
		}
		names = append(names, f.column)
	}

	return names
}

func (s *DbRecorder) fieldReferences(withKeys bool) []interface{} {
	refs := make([]interface{}, 0, len(s.fields))

	ar := reflect.Indirect(reflect.ValueOf(s.record))
	for _, f := range s.fields {
		if !withKeys && f.isKey {
			continue
		}

		//ref := reflect.ValueOf(ar.FieldByName(f.name)).Addr().Interface()
		ref := reflect.Indirect(ar.FieldByName(f.name))
		if ref.IsValid() {
			refs = append(refs, ref.Interface())
		} else { // Should never hit this part.
			var skip interface{}
			refs = append(refs, &skip)
		}

	}

	return refs 
}

func (s *DbRecorder) insertFields() (columns []string, values []interface{}) {
	/*numFields := len(s.fields)
	columns = make([]string, numFields)
	values = make([]interface{}, numFields)
	*/
	ar := reflect.Indirect(reflect.ValueOf(s.record))

	for _, field := range s.fields {
		// Serial fields are automatically set, so we don't everride, lest
		// we an invalid/duplicate key value.
		if field.isAuto {
			continue
		}

		// Get the value of the field we are going to store.
		//v := reflect.Indirect(reflect.ValueOf(ar.FieldByName(field.name))).Interface()
		v := ar.FieldByName(field.name).Interface()
		
		columns = append(columns, field.column)
		values = append(values, v)
	}

	return
}

// whereIds gets a list of names and a list of values for all columns marked as primary
// keys.
func (s *DbRecorder) whereIds() map[string]interface{} { // ([]string, []interface{}) {
	clause := make(map[string]interface{}, len(s.key))

	ar := reflect.Indirect(reflect.ValueOf(s.record))

	for _, f := range s.key {
		clause[f.column] = ar.FieldByName(f.name).Interface()
		//fmt.Printf("Where parts: %V", clause[f.column])
	}


	return clause
}

// scanFields extracts the tags from all of the fields on a struct.
func (s *DbRecorder) scanFields(ar ActiveRecord) {
	v := reflect.Indirect(reflect.ValueOf(ar))
	t := v.Type()
	count := t.NumField()
	keys := make([]*Field, 0, 2)

	for i := 0; i < count; i++ {
		f := t.Field(i)
		// Skip fields with no tag.
		if len(f.Tag) == 0 {
			continue
		}
		sqtag := f.Tag.Get("sqrl")
		if len(sqtag) == 0 {
			continue
		}

		parts := s.parseTag(f.Name, sqtag)
		field := new(Field)
		field.name = f.Name
		field.column = parts[0]
		for _, part := range parts[1:] {
			switch part {
			case "PRIMARY_KEY":
				field.isKey = true
				keys = append(keys, field)
			case "AUTO_INCREMENT":
				field.isAuto = true
			}
		}
		s.fields = append(s.fields, field)
		s.key = keys
	}
	
}

// Parse the contents of a sqrl tag.
func (s *DbRecorder) parseTag(fieldName, tag string) []string {
	parts := strings.Split(tag, " ")
	if len(parts) == 0 {
		return []string{fieldName}
	}
	return parts
}

