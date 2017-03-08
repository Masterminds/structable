package structable

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Masterminds/squirrel"
)

type Stool struct {
	Id       int     `stbl:"id,PRIMARY_KEY,AUTO_INCREMENT"`
	Id2      int     `stbl:"id_two,    PRIMARY_KEY      "`
	Legs     int     `stbl:"number_of_legs"`
	Material string  `stbl:"material"`
	Color    *string `stbl:"color"`
	Ignored  string  // will not be stored.
}

func newStool() *Stool {
	stool := new(Stool)

	stool.Id = 1
	stool.Id2 = 2
	stool.Legs = 3
	stool.Material = "Stainless Steel"
	stool.Ignored = "Boo"

	return stool
}

type ActRec struct {
	Id       int    `stbl:"id,SERIAL,PRIMARY_KEY"`
	Name     string `stbl:"name"`
	recorder Recorder
}

func NewActRec(db *DBStub) *ActRec {
	a := new(ActRec)

	a.recorder = New(db, "mysql").Bind("my_table", a)

	return a
}

func (a *ActRec) Exists() bool {

	ok, err := a.recorder.Exists()

	return err == nil && ok
}

func TestBind(t *testing.T) {
	store := new(DbRecorder)

	stool := newStool()
	store.Bind("test_table", stool)

	if store.table != "test_table" {
		t.Errorf("Failed to get table name.")
	}

	if len(store.fields) != 5 {
		t.Errorf("Expected 5 fields, got %d: %+v", len(store.fields), store.fields)
	}

	keyCount := 0
	for _, f := range store.fields {
		if f.isKey {
			keyCount++
		}
	}

	if keyCount != 2 {
		t.Errorf("Expected two keys.")
	}

	if len(store.Key()) != 2 {
		t.Errorf("Wrong number of keys.")
	}
}

func TestLoad(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	//db, builder := squirrelFixture()

	r := New(db, "mysql").Bind("test_table", stool)

	if err := r.Load(); err != nil {
		t.Errorf("Error running query: %s", err)
	}

	expect := "SELECT number_of_legs, material, color FROM test_table WHERE id = ? AND id_two = ?"
	if db.LastQueryRowSql != expect {
		t.Errorf("Unexpected SQL: %s", db.LastQueryRowSql)
	}

	expectargs := []interface{}{1, 2}
	got := db.LastQueryRowArgs
	for i, exp := range expectargs {
		if exp != got[i] {
			t.Errorf("Surprise! %v doesn't equal %v", exp, got[i])
		}
	}
}

func TestLoadWhere(t *testing.T) {
	stool := newStool()
	db := &DBStub{}

	r := New(db, "mysql").Bind("test_table", stool)

	if err := r.LoadWhere("number_of_legs = ?", 3); err != nil {
		t.Errorf("Error running query: %s", err)
	}

	if len(db.LastQueryRowArgs) != 1 {
		t.Errorf("Expected exactly one where arg.")
	}

	expect := "SELECT .* FROM test_table WHERE number_of_legs = ?"
	if ok, err := regexp.MatchString(expect, db.LastQueryRowSql); err != nil {
		t.Errorf("Failed to run regexp: %s", err)
	} else if !ok {
		t.Errorf("%s did not match pattern %s", db.LastQueryRowSql, expect)
	}

}

func TestList(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	//db, builder := squirrelFixture()

	r := New(db, "mysql").Bind("test_table", stool)

	if _, err := List(r, 10, 0); err != nil {
		t.Errorf("Error running query: %s", err)
	}

	expect := "SELECT number_of_legs, material, color FROM test_table LIMIT 10 OFFSET 0"
	if db.LastQuerySql != expect {
		t.Errorf("Unexpected SQL: %q\nGot %q", expect, db.LastQuerySql)
	}
}

func TestListWhere_Error(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	r := New(db, "mysql").Bind("test_table", stool)

	fn := func(d Describer, q squirrel.SelectBuilder) (squirrel.SelectBuilder, error) {
		return q, errors.New("intentional failure")
	}

	if _, err := ListWhere(r, fn); err == nil {
		t.Error("Expected failed WhereFunc to fail query")
	}
}

func TestInsert(t *testing.T) {
	stool := newStool()
	db := new(DBStub)

	rec := New(db, "mysql").Bind("test_table", stool)

	if err := rec.Insert(); err != nil {
		t.Errorf("Failed insert: %s", err)
	}

	expect := "INSERT INTO test_table (id_two,number_of_legs,material) VALUES (?,?,?)"
	if db.LastExecSql != expect {
		t.Errorf("Expected '%s', got '%s'", expect, db.LastExecSql)
	}

	expectargs := []interface{}{stool.Id2, stool.Legs, stool.Material}
	gotargs := db.LastExecArgs

	for i := range expectargs {
		if expectargs[i] != gotargs[i] {
			t.Errorf("Expected %v, got %v", expectargs[i], gotargs[i])
		}
	}
}

func TestUpdate(t *testing.T) {
	stool := newStool()
	db := new(DBStub)

	rec := New(db, "mysql").Bind("test_table", stool)

	// with nil pointer field
	if err := rec.Update(); err != nil {
		t.Errorf("Update error: %s", err)
	}

	if !strings.Contains(db.LastExecSql, "number_of_legs = ") {
		t.Error("Expected 'number_of_legs' in query")
	}
	if !strings.Contains(db.LastExecSql, "material = ") {
		t.Error("Expected 'material' in query")
	}

	eargs := []interface{}{3, "Stainless Steel", 1, 2}
	gotargs := db.LastExecArgs
	for _, exp := range eargs {
		found := false
		for _, arg := range gotargs {
			if arg == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Could not find %v in %v", exp, gotargs)
		}
	}

	// with allocated pointer
	blue := "Blue"
	stool.Color = &blue

	if err := rec.Update(); err != nil {
		t.Errorf("Update error: %s", err)
	}

	if !strings.Contains(db.LastExecSql, "number_of_legs = ") {
		t.Error("Expected 'number_of_legs' in query")
	}
	if !strings.Contains(db.LastExecSql, "material = ") {
		t.Error("Expected 'material' in query")
	}
	if !strings.Contains(db.LastExecSql, "color = ") {
		t.Error("Expected 'color' in query")
	}

	eargs = []interface{}{3, "Stainless Steel", &blue, 1, 2}
	gotargs = db.LastExecArgs
	for _, exp := range eargs {
		found := false
		for _, arg := range gotargs {
			if arg == exp {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Could not find %v in %v", exp, gotargs)
		}
	}
}

func TestDelete(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	r := New(db, "mysql").Bind("test_table", stool)

	if err := r.Delete(); err != nil {
		t.Errorf("Failed to delete: %s", err)
	}

	expect := "DELETE FROM test_table WHERE .* AND .*"
	if ok, _ := regexp.MatchString(expect, db.LastExecSql); !ok {
		t.Errorf("Unexpect query: %s", db.LastExecSql)
	}
	if got := db.LastExecArgs[0].(int); got != 1 {
		t.Errorf("Expected 1, got %d", got)
	}
}

func TestExists(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	r := New(db, "mysql").Bind("test_table", stool)

	_, err := r.Exists()
	if err != nil {
		t.Errorf("Error calling Exists: %s", err)
	}

	expect := "SELECT COUNT(*) > 0 FROM test_table WHERE id = ? AND id_two = ?"
	if db.LastQueryRowSql != expect {
		t.Errorf("Unexpected SQL: expected %q, got %q", expect, db.LastQueryRowSql)
	}
}

func TestActiveRecord(t *testing.T) {
	db := &DBStub{}
	a := NewActRec(db)
	a.Id = 999

	if a.Exists() {
		t.Errorf("Expected record to be absent.")
	}
}

func squirrelFixture() (*DBStub, squirrel.StatementBuilderType) {

	db := &DBStub{}
	//cache := squirrel.NewStmtCacher(db)
	return db, squirrel.StatementBuilder.RunWith(db)

}

// FIXTURES
type DBStub struct {
	err error

	LastPrepareSql string
	PrepareCount   int

	LastExecSql  string
	LastExecArgs []interface{}

	LastQuerySql  string
	LastQueryArgs []interface{}

	LastQueryRowSql  string
	LastQueryRowArgs []interface{}
}

var StubError = fmt.Errorf("this is a stub; this is only a stub")

func (s *DBStub) Prepare(query string) (*sql.Stmt, error) {
	s.LastPrepareSql = query
	s.PrepareCount++
	return nil, nil
}

func (s *DBStub) Exec(query string, args ...interface{}) (sql.Result, error) {
	s.LastExecSql = query
	s.LastExecArgs = args
	return &ResultStub{id: 1, affectedRows: 1}, nil
}

func (s *DBStub) Query(query string, args ...interface{}) (*sql.Rows, error) {
	s.LastQuerySql = query
	s.LastQueryArgs = args
	return nil, nil
}

func (s *DBStub) QueryRow(query string, args ...interface{}) squirrel.RowScanner {
	s.LastQueryRowSql = query
	s.LastQueryRowArgs = args
	return &squirrel.Row{RowScanner: &RowStub{}}
}

func (s *DBStub) Begin() (*sql.Tx, error) {
	return nil, nil
}

type RowStub struct {
	Scanned bool
}

func (r *RowStub) Scan(_ ...interface{}) error {
	r.Scanned = true
	return nil
}

type ResultStub struct {
	id, affectedRows int64
}

func (r *ResultStub) LastInsertId() (int64, error) {
	return r.id, nil
}
func (r *ResultStub) RowsAffected() (int64, error) {
	return r.affectedRows, nil
}
