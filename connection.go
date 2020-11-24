package main

import "database/sql"

//check any error
func errorCheck(err error) {
	if err != nil {
		panic(err.Error())
	}
}

//check database current connection
func pingDb(db *sql.DB) error {
	err := db.Ping()
	return err
}

//initializing database connection
func initDb() *sql.DB {
	db, e := sql.Open("mysql", "root:password@tcp(127.0.0.1)/pymnt_db")
	errorCheck(e)
	return db
}
