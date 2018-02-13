package sqle_test

import (
	"database/sql"
	"fmt"

	"github.com/src-d/go-mysql-server"
	"github.com/src-d/go-mysql-server/mem"
	gitqlsql "github.com/src-d/go-mysql-server/sql"
)

func Example() {
	// Create a test memory database and register it to the default engine.
	sqle.DefaultEngine.AddDatabase(createTestDatabase())

	// Open a sql connection with the default engine.
	conn, err := sql.Open(sqle.DriverName, "")
	checkIfError(err)

	// Prepare a query.
	stmt, err := conn.Prepare(`SELECT name, count(*) FROM mytable
	WHERE name = 'John Doe'
	GROUP BY name`)
	checkIfError(err)

	// Get result rows.
	rows, err := stmt.Query()
	checkIfError(err)

	// Iterate results and print them.
	for {
		if !rows.Next() {
			break
		}

		name := ""
		count := int64(0)
		err := rows.Scan(&name, &count)
		checkIfError(err)

		fmt.Println(name, count)
	}
	checkIfError(rows.Err())

	// Output: John Doe 2
}

func checkIfError(err error) {
	if err != nil {
		panic(err)
	}
}

func createTestDatabase() *mem.Database {
	db := mem.NewDatabase("test")
	table := mem.NewTable("mytable", gitqlsql.Schema{
		{Name: "name", Type: gitqlsql.String},
		{Name: "email", Type: gitqlsql.String},
	})
	db.AddTable("mytable", table)
	table.Insert(gitqlsql.NewRow("John Doe", "john@doe.com"))
	table.Insert(gitqlsql.NewRow("John Doe", "johnalt@doe.com"))
	table.Insert(gitqlsql.NewRow("Jane Doe", "jane@doe.com"))
	table.Insert(gitqlsql.NewRow("Evil Bob", "evilbob@gmail.com"))
	return db
}
