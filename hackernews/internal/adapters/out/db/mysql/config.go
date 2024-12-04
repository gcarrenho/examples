package mysql

import (
	"database/sql"
	"fmt"
)

/*
	func NewMySQLClient(dsn string) (*sql.DB, error) {
		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return nil, err
		}
		return db, nil
	}
*/
type MysqlConf struct {
	dbName     string // The database that you want to access
	dbUser     string // The database account that you want to access
	dbPassword string // The password for the database account you want to access
	dbHost     string // The endpoint of the DB instance that you want to access
	dbPort     int    // The port number used for connecting to your DB instance
}

func NewMySqlConf(user, password string) MysqlConf {
	return MysqlConf{
		dbName:     "hackernews", //os.Getenv("DBNAME"),
		dbUser:     "root",       //user,
		dbPassword: "redificil",  //password,
		dbHost:     "localhost",  //os.Getenv("DBHOST"),
		dbPort:     3306,
	}
}

func (mysql MysqlConf) NewMySQLClient() (*sql.DB, error) {
	// clientFoundRows=true is required for UPDATEs to return the number of matching rows instead of the number of rows changed
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=true&clientFoundRows=true",
		mysql.dbUser, mysql.dbPassword, mysql.dbHost, mysql.dbPort, mysql.dbName)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	err = db.Ping()
	if err != nil {
		return nil, err
	}

	return db, nil
}
