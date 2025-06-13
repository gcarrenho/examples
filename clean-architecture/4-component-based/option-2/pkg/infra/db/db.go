package db

import (
	"database/sql"
	"log"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/lib/pq"
)

func InitMySQL() *sql.DB {
	db, err := sql.Open("mysql", "user:password@tcp(localhost:3306)/orders")
	if err != nil {
		log.Fatalf("failed to connect to MySQL: %v", err)
	}
	return db
}

func InitPostgres() *sql.DB {
	db, err := sql.Open("postgres", "postgres://user:password@localhost:5432/payments?sslmode=disable")
	if err != nil {
		log.Fatalf("failed to connect to PostgreSQL: %v", err)
	}
	return db
}
