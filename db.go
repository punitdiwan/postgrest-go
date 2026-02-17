package main

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func initDB() {
	var err error
	dbCfg, err := pgxpool.ParseConfig("postgres://postgres:postgres@localhost:5432/postgres")
	if err != nil {
		log.Fatalf("Unable to parse DATABASE_URL: %v", err)
	}
	log.Println("Connect to database:", dbCfg.ConnConfig.Host)

	dbCfg.MaxConns = 25
	dbCfg.MinConns = 5
	dbCfg.MaxConnLifetime = time.Hour
	dbCfg.HealthCheckPeriod = time.Minute

	DB, err = pgxpool.NewWithConfig(context.Background(), dbCfg)
	if err != nil {
		log.Fatalf("Unable to create database connection pool: %v", err)
	}
}
