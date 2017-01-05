// +build sqlite

package structable

import (
	"database/sql"
	"log"
	"testing"

	"github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

type Movie struct {
	Recorder
	builder squirrel.StatementBuilderType

	Id     int64   `stbl:"id,PRIMARY_KEY,AUTO_INCREMENT"`
	Title  string  `stbl:"title"`
	Genre  *string `stbl:"genre"`
	Budget float64 `stbl:"budget"`
}

func (l *Movie) equals(other *Movie) bool {
	return l.Id == other.Id &&
		l.Title == other.Title &&
		l.Budget == other.Budget &&
		CompareStringPtr(l.Genre, other.Genre)
}

func CompareStringPtr(s1, s2 *string) bool {
	switch {
	case s1 == nil && s2 == nil:
		return true
	case s1 != nil && s2 == nil || s1 == nil && s2 != nil:
		return false
	default:
		return *s1 == *s2
	}
}

func stringPtr(s string) *string {
	return &s
}

func (m *Movie) loadFromSql(Id int64, db *sql.DB) error {

	if m.Genre == nil {
		m.Genre = new(string)
	}
	err := db.QueryRow("SELECT title, genre, budget FROM movies WHERE id=?", Id).Scan(
		&m.Title,
		m.Genre,
		&m.Budget)
	if err == nil {
		m.Id = Id
	}
	return err
}

func TestStructWithPointerInsert(t *testing.T) {

	db := getMoviesDb()

	m := &Movie{
		Id:     -1,
		Title:  "2001: A Space Odyssey",
		Genre:  stringPtr("Science-Fiction"),
		Budget: 1500000}
	m.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("movies", m)

	if err := m.Insert(); err != nil {
		t.Fatalf("Failed Insert: %s", err)
	}

	msql := new(Movie)
	msql.loadFromSql(m.Id, db)

	if *msql.Genre != "Science-Fiction" {
		t.Fatal("Insert should dereference allocated pointers")
	}

	m.Genre = nil
	if err := m.Insert(); err != nil {
		t.Fatalf("Failed Insert: %s", err)
	}

	msql.loadFromSql(m.Id, db)

	if *msql.Genre != "unclassifiable" {
		t.Fatal("Insert should ignore nil pointers")
	}
}

func TestStructWithPointerLoad(t *testing.T) {

	db := getMoviesDb()

	msql := &Movie{}
	if res, err := db.Exec("INSERT INTO movies (title, genre, budget) VALUES ('2001: A Space Odyssey', 'Science-Fiction', 1500000)"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if msql.Id, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	m := &Movie{Id: msql.Id, Genre: new(string)}
	m.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("movies", m)
	if err := m.Load(); err != nil {
		t.Fatalf("Failed Load: %s", err)
	}

	if !CompareStringPtr(m.Genre, stringPtr("Science-Fiction")) {
		t.Fatal("Load should load pointer fields")
	}

	m.Genre = nil
	m.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("movies", m)
	if err := m.Load(); err != nil {
		t.Fatalf("Failed Load: %s", err)
	}

	if !CompareStringPtr(m.Genre, stringPtr("Science-Fiction")) {
		t.Fatal("Load should instantiate nil pointers")
	}
}

func TestStructWithPointerLoadWhere(t *testing.T) {

	db := getMoviesDb()

	msql := &Movie{}
	if res, err := db.Exec("INSERT INTO movies (title, genre, budget) VALUES ('2001: A Space Odyssey', 'Science-Fiction', 1500000)"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if msql.Id, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	m := &Movie{}
	m.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("movies", m)
	if err := m.LoadWhere("budget = ?", 1500000); err != nil {
		t.Fatalf("Failed LoadWhere: %s", err)
	}

	if !CompareStringPtr(m.Genre, stringPtr("Science-Fiction")) {
		t.Fatal("LoadWhere should load pointer fields")
	}

	m.Genre = nil
	m.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("movies", m)
	if err := m.LoadWhere("budget = ?", 1500000); err != nil {
		t.Fatalf("Failed LoadWhere: %s", err)
	}

	if !CompareStringPtr(m.Genre, stringPtr("Science-Fiction")) {
		t.Fatal("LoadWhere should instantiate nil pointers")
	}
}

func TestStructWithPointerUpdate(t *testing.T) {

	db := getMoviesDb()

	var lastId int64
	if res, err := db.Exec("INSERT INTO movies (title, genre, budget) VALUES ('2001: A Space Odyssey', 'Science-Fiction', 1500000)"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if lastId, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	m := &Movie{
		Id:     lastId,
		Title:  "The Usual Suspects",
		Genre:  nil,
		Budget: 6000000}

	m.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("movies", m)
	if err := m.Update(); err != nil {
		t.Fatalf("Failed Update: %s", err)
	}

	msql := new(Movie)
	msql.loadFromSql(lastId, db)

	if !CompareStringPtr(msql.Genre, stringPtr("Science-Fiction")) {
		t.Fatal("Update should ignore nil pointers")
	}

	m.Genre = stringPtr("Crime Thriller")
	m.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("movies", m)
	if err := m.Update(); err != nil {
		t.Fatalf("Failed Update: %s", err)
	}

	msql.loadFromSql(lastId, db)
	if !CompareStringPtr(msql.Genre, stringPtr("Crime Thriller")) {
		t.Log("msql.Genre: %v\n", *msql.Genre)
		t.Fatal("Update should ignore nil pointers")
	}
}

func getMoviesDb() *sql.DB {

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Couldn't Open database: %s\n", err)
	}

	stmt := `
	CREATE TABLE movies (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		title STRING,
		genre STRING DEFAULT('unclassifiable'),
		budget REAL
	);
	DELETE FROM movies;
	`

	_, err = db.Exec(stmt)
	if err != nil {
		log.Fatalf("Couldn't Exec query \"%q\": %s\n", err, stmt)
		return nil
	}
	return db
}
