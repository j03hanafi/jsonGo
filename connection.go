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
	db, e := sql.Open("mysql", "user:user@123@tcp(157.245.55.51)/pymnt_db")
	errorCheck(e)
	return db
}
