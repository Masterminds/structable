package main

import (
	"github.com/Masterminds/squirrel"
	"github.com/Masterminds/structable"
	_ "github.com/lib/pq"

	"database/sql"
	"fmt"
)

// For convenience, we declare the table name as a constant.
const UserTable = "users"

// This is our struct. Notice that we make this a structable.Recorder.
type User struct {
	structable.Recorder
	builder squirrel.StatementBuilderType

	Id    int    `stbl:"id,PRIMARY_KEY,SERIAL"`
	Name  string `stbl:"name"`
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

// LoadByName is a custom loader.
//
// The Load() method on a Recorder loads by ID. This allows us to load by
// a different field -- Name.
func (u *User) LoadByName() error {
	return u.Recorder.LoadWhere("name = ? order by id desc", u.Name)
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
	again.Name = "Matt"

	// Load using our custom loader.
	if err := again.LoadByName(); err != nil {
		panic(err.Error())
	}
	fmt.Printf("User by name has ID %d and email %s\n", again.Id, again.Email)

	again.Email = "Masterminds@example.com"
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
