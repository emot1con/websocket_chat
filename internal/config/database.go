// postgres.go
package config

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type PostgresConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func Connect() (*sql.DB, error) {
	DBPort, err := strconv.Atoi(os.Getenv("DB_PORT"))
	if err != nil {
		return nil, fmt.Errorf("invalid DB_PORT: %v", err)
	}
	dbConfig := PostgresConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     DBPort,
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  "disable",
	}

	dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.Password, dbConfig.DBName, dbConfig.SSLMode,
	)

	log.Println("connecting to database...")
	counter := 1
	for {
		db, err := sql.Open("pgx", dsn)
		if counter > 5 {
			return nil, fmt.Errorf("failed connect to database: %v", err)
		}

		if err != nil {
			log.Printf("retrying connect database (%v/5)", counter)
			counter++
			time.Sleep(2 * time.Second)
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := db.PingContext(ctx); err != nil {
			return nil, err
		}
		log.Println("database connected")
		return db, nil
	}
}
