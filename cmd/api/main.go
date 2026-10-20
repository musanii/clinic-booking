package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"time"

	"clinic-booking/internal/httpapi"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		log.Fatal("DATABASE_URL is required")
	}

	ctx, cancel := context.WithTimeout(
		context.Background(),
		5*time.Second,
	)
	defer cancel()

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("create database pool: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("connect to database: %v", err)
	}

	api := httpapi.New(pool)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           api.Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Println("Clinic booking API listening on http://localhost:8080")

	if err := server.ListenAndServe(); err != nil &&
		!errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("HTTP server: %v", err)
	}
}
