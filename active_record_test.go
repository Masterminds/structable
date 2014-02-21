package squirrelrm

import (
	"testing"
	"fmt"
	"database/sql"

	"github.com/lann/squirrel"
)

type Stool struct {
	Id		 int	`sqrl:"id PRIMARY_KEY"`
	Id2 	 int 	`sqrl:"id_two PRIMARY_KEY"`
	Legs	 int    `sqrl:"number_of_legs"`
	Material string `sqrl:"material"`
	Ignored  string // will not be stored.
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

func TestBind(t *testing.T) {
	store := new(DbRecorder)

	stool := newStool()
	store.Bind("test_table", stool)

	if store.table != "test_table" {
		t.Errorf("Failed to get table name.")
	}

	if len(store.fields) != 4 {
		t.Errorf("Expected 4 fields, got %d: %V", len(store.fields), store.fields)
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

	/*
	sql := store.loadSql().ToString()
	expect := "SELECT number_of_legs, material FROM test_table WHERE id = ? AND id_two = ?"
	if sql != expect {
		t.Errorf("Got SQL '%s'", sql)
	}
	*/

}

func TestLoad(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	//db, builder := squirrelFixture()

	r := NewDbRecorder(db).Bind("test_table", stool)

	if err := r.Load(); err != nil {
		t.Errorf("Error running query: %s", err)
	}

	expect := "SELECT number_of_legs, material FROM test_table WHERE id = ? AND id_two = ?"
	if db.LastQueryRowSql != expect {
		t.Errorf("Unexpected SQL: %s", db.LastQueryRowSql)
	}

	if db.LastQueryRowArgs[0].(int) != 1 {
		t.Errorf("Expected 1")
	}

}

func TestDelete(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	r := NewDbRecorder(db).Bind("test_table", stool)

	if err := r.Delete(); err != nil {
		t.Errorf("Failed to delete: %s", err)
	}

	expect := "DELETE FROM test_table WHERE id = ? AND id_two = ?"
	if db.LastExecSql != expect {
		t.Errorf("Unexpect query: %s", db.LastExecSql)
	}
	if db.LastExecArgs[0].(int) != 1 {
		t.Errorf("Expected 1")
	}
}

func TestHas(t *testing.T) {
	stool := newStool()
	db := &DBStub{}
	r := NewDbRecorder(db).Bind("test_table", stool)

	_, err := r.Has()
	if err != nil {
		t.Errorf("Error calling Has: %s", err)
	}

	expect := "SELECT COUNT(*) FROM test_table WHERE id = ? AND id_two = ?"
	if db.LastQueryRowSql != expect {
		t.Errorf("Unexpected SQL: %s", db.LastQueryRowSql)
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
	return nil, nil
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

type RowStub struct {
	Scanned bool
}

func (r *RowStub) Scan(_ ...interface{}) error {
	r.Scanned = true
	return nil
}
