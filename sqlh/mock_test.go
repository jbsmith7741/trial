package sqlh

import (
	"database/sql"
	"fmt"
	"testing"
)

func ExampleOpen() {
	db, _ := sql.Open(Mock, "test://123")
	fmt.Printf("%T", db)
	// Output: *sql.DB
}

func TestStmt_Query(t *testing.T) {
	db, _ := NewMock("")
	_, err := db.Query("Select * from table where column = $1 and column = $2", 1, 2)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.Query("Select * from table where column = ? and column = ?", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
}
