package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var version = "development"

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		cancel()
		log.Fatal("invalid database configuration")
	}
	err = db.Ping(ctx)
	cancel()
	if err != nil {
		log.Fatalf("database connection failed: %v", err)
	}
	defer db.Close()

	addr := os.Getenv("HTTP_ADDR")
	if addr == "" {
		addr = "127.0.0.1:8000"
	}
	api := todoAPI{db: db}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/version", func(w http.ResponseWriter, r *http.Request) {
		respond(w, http.StatusOK, map[string]string{
			"version": version,
		})
	})
	mux.HandleFunc("GET /api/health", api.health)
	mux.HandleFunc("GET /api/todos", api.list)
	mux.HandleFunc("POST /api/todos", api.create)
	mux.HandleFunc("PATCH /api/todos/{id}", api.update)
	mux.HandleFunc("DELETE /api/todos/{id}", api.remove)

	server := &http.Server{Addr: addr, Handler: mux, ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second}
	log.Printf("listening on %s", addr)
	log.Fatal(server.ListenAndServe())
}
