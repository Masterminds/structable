// +build sqlite

package structable

import (
	"database/sql"
	"log"
	"testing"
	"time"

	"github.com/Masterminds/squirrel"
	_ "github.com/mattn/go-sqlite3"
)

type Language struct {
	Recorder
	builder squirrel.StatementBuilderType

	Id        int64     `stbl:"id,PRIMARY_KEY,AUTO_INCREMENT"`
	Name      string    `stbl:"name"`
	Version   string    `stbl:"version"`
	DtRelease time.Time `stbl:"dt_release"`
}

func (l *Language) equals(other *Language) bool {
	return l.Id == other.Id &&
		l.Name == other.Name &&
		l.Version == other.Version &&
		l.DtRelease.Equal(other.DtRelease)
}

func (l *Language) loadFromSql(Id int64, db *sql.DB) error {

	err := db.QueryRow("SELECT name, version, dt_release FROM languages WHERE id=?", Id).Scan(
		&l.Name,
		&l.Version,
		&l.DtRelease)
	if err == nil {
		l.Id = Id
	}
	return err
}

func TestPlainStructInsert(t *testing.T) {

	db := getLanguagesDb()

	l := &Language{
		Id:        -1,
		Name:      "Go",
		Version:   "1.3",
		DtRelease: time.Date(2014, time.June, 18, 0, 0, 0, 0, time.UTC)}
	l.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("languages", l)

	if err := l.Insert(); err != nil {
		t.Fatalf("Failed Insert: %s", err)
	}

	lsql := new(Language)
	lsql.loadFromSql(l.Id, db)
	if !l.equals(lsql) {
		t.Fatal("Loaded and inserted objects should be equivalent")
	}
}

func TestPlainStructLoad(t *testing.T) {

	db := getLanguagesDb()

	lsql := &Language{}
	if res, err := db.Exec("INSERT INTO languages (name, version, dt_release) VALUES ('Scala', '2.11.7', '2015-06-23')"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if lsql.Id, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	l := &Language{Id: lsql.Id}
	l.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("languages", l)
	if err := l.Load(); err != nil {
		t.Fatalf("Failed Load: %s", err)
	}

	lsql.Name = "Scala"
	lsql.Version = "2.11.7"
	lsql.DtRelease = time.Date(2015, time.June, 23, 0, 0, 0, 0, time.UTC)
	if !l.equals(lsql) {
		t.Fatal("Loaded and inserted objects should be equivalent")
	}
}

func TestPlainStructLoadWhere(t *testing.T) {

	db := getLanguagesDb()

	var lastId int64
	if res, err := db.Exec("INSERT INTO languages (name, version, dt_release) VALUES ('Scala', '2.11.7', '2015-06-23')"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if lastId, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	lsql := &Language{
		Id:        -1,
		Name:      "Scala",
		Version:   "2.11.7",
		DtRelease: time.Date(2015, time.June, 23, 0, 0, 0, 0, time.UTC)}

	l := &Language{Id: lastId}
	l.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("languages", l)
	if err := l.LoadWhere("version = ?", "2.11.7"); err != nil {
		t.Fatalf("Failed LoadWhere: %s", err)
	}

	lsql.Id = lastId
	if !l.equals(lsql) {
		t.Fatal("Loaded and inserted objects should be equivalent")
	}
}

func TestPlainStructUpdate(t *testing.T) {

	db := getLanguagesDb()

	var lastId int64
	if res, err := db.Exec("INSERT INTO languages (name, version, dt_release) VALUES ('Scala', '2.11.7', '2015-06-23')"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if lastId, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	l := &Language{
		Id:        lastId,
		Name:      "Go",
		Version:   "1.4",
		DtRelease: time.Date(2014, time.June, 18, 0, 0, 0, 0, time.UTC)}

	l.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("languages", l)
	if err := l.Update(); err != nil {
		t.Fatalf("Failed Update: %s", err)
	}

	lsql := new(Language)
	lsql.loadFromSql(lastId, db)
	if !l.equals(lsql) {
		t.Fatal("Loaded and updated objects should be equivalent")
	}
}

func TestPlainStructDelete(t *testing.T) {

	db := getLanguagesDb()

	var lastId int64
	if res, err := db.Exec("INSERT INTO languages (name, version, dt_release) VALUES ('Scala', '2.11.7', '2015-06-23')"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if lastId, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	l := &Language{Id: lastId}
	l.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("languages", l)
	if err := l.Delete(); err != nil {
		t.Fatalf("Failed Delete: %s", err)
	}

	var count int64
	if err := db.QueryRow("SELECT COUNT(*) from languages;").Scan(&count); err != nil {
		t.Fatalf("Error executing query: %s", err)
	}
	if count != 0 {
		t.Fatalf("Database should count no rows, instead it has got: %v", count)
	}
}

func TestPlainStructExists(t *testing.T) {

	db := getLanguagesDb()

	l := &Language{Id: 1}
	l.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("languages", l)

	if exists, err := l.Exists(); err != nil {
		t.Fatalf("Failed Exists: %s", err)
	} else if exists {
		t.Fatal("Exists should return false")
	}

	var lastId int64
	if res, err := db.Exec("INSERT INTO languages (name, version, dt_release) VALUES ('Scala', '2.11.7', '2015-06-23')"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	} else if lastId, err = res.LastInsertId(); err != nil {
		t.Fatalf("Sqlite LastInsertId failed: %s", err)
	}

	l.Id = lastId
	if exists, err := l.Exists(); err != nil {
		t.Fatalf("Failed Exists: %s", err)
	} else if !exists {
		t.Fatal("Exists should return true")
	}
}

func TestPlainStructExistsWhere(t *testing.T) {

	db := getLanguagesDb()

	if _, err := db.Exec("INSERT INTO languages (name, version, dt_release) VALUES ('Scala', '2.11.7', '2015-06-23')"); err != nil {
		t.Fatalf("Sqlite Exec failed: %s", err)
	}

	l := &Language{}
	l.Recorder = New(squirrel.NewStmtCacheProxy(db), "mysql").Bind("languages", l)

	if exists, err := l.ExistsWhere("Name = ?", "Go"); err != nil {
		t.Fatalf("Failed ExistsWhere: %s", err)
	} else if exists {
		t.Fatal("ExistsWhere should return false")
	}

	if exists, err := l.ExistsWhere("Name = ?", "Scala"); err != nil {
		t.Fatalf("Failed Exists: %s", err)
	} else if !exists {
		t.Fatal("Exists should return true")
	}
}

func getLanguagesDb() *sql.DB {

	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		log.Fatalf("Couldn't Open database: %s\n", err)
	}

	stmt := `
	CREATE TABLE languages (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name STRING,
		version STRING,
		dt_release TIMESTAMP DEFAULT('1789-07-14 12:00:00.000')
	);
	DELETE FROM languages;
	`
	_, err = db.Exec(stmt)
	if err != nil {
		log.Fatalf("Couldn't Exec query \"%q\": %s\n", err, stmt)
		return nil
	}
	return db
}
