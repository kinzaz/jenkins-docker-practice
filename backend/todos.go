package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type todoAPI struct {
	db *pgxpool.Pool
}

type todo struct {
	ID    int64  `json:"id"`
	Title string `json:"title"`
	Done  bool   `json:"done"`
}

func respond(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("write response: %v", err)
	}
}

func readJSON(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	if decoder.Decode(value) != nil || decoder.Decode(new(any)) != io.EOF {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return false
	}
	return true
}

func todoID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		http.Error(w, "invalid todo id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func dbError(w http.ResponseWriter, err error) {
	if errors.Is(err, pgx.ErrNoRows) {
		http.Error(w, "todo not found", http.StatusNotFound)
		return
	}
	log.Printf("database operation failed: %v", err)
	http.Error(w, "database operation failed", http.StatusInternalServerError)
}

func (api todoAPI) health(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := api.db.Ping(ctx); err != nil {
		respond(w, http.StatusServiceUnavailable, map[string]string{"status": "database unavailable"})
		return
	}
	respond(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (api todoAPI) list(w http.ResponseWriter, r *http.Request) {
	rows, err := api.db.Query(r.Context(), "SELECT id, title, done FROM todos ORDER BY id DESC")
	if err != nil {
		dbError(w, err)
		return
	}
	items, err := pgx.CollectRows(rows, pgx.RowToStructByPos[todo])
	if err != nil {
		dbError(w, err)
		return
	}
	respond(w, http.StatusOK, items)
}

func (api todoAPI) create(w http.ResponseWriter, r *http.Request) {
	var input struct{ Title string }
	if !readJSON(w, r, &input) {
		return
	}
	title := strings.TrimSpace(input.Title)
	if title == "" || utf8.RuneCountInString(title) > 200 || strings.ContainsRune(title, 0) {
		http.Error(w, "title must contain 1 to 200 valid characters", http.StatusBadRequest)
		return
	}
	var item todo
	err := api.db.QueryRow(r.Context(), "INSERT INTO todos (title) VALUES ($1) RETURNING id, title, done", title).
		Scan(&item.ID, &item.Title, &item.Done)
	if err != nil {
		dbError(w, err)
		return
	}
	respond(w, http.StatusCreated, item)
}

func (api todoAPI) update(w http.ResponseWriter, r *http.Request) {
	id, ok := todoID(w, r)
	if !ok {
		return
	}
	var input struct{ Done *bool }
	if !readJSON(w, r, &input) {
		return
	}
	if input.Done == nil {
		http.Error(w, "done is required", http.StatusBadRequest)
		return
	}
	var item todo
	err := api.db.QueryRow(r.Context(), "UPDATE todos SET done = $1 WHERE id = $2 RETURNING id, title, done", *input.Done, id).
		Scan(&item.ID, &item.Title, &item.Done)
	if err != nil {
		dbError(w, err)
		return
	}
	respond(w, http.StatusOK, item)
}

func (api todoAPI) remove(w http.ResponseWriter, r *http.Request) {
	id, ok := todoID(w, r)
	if !ok {
		return
	}
	result, err := api.db.Exec(r.Context(), "DELETE FROM todos WHERE id = $1", id)
	if err != nil {
		dbError(w, err)
		return
	}
	if result.RowsAffected() == 0 {
		dbError(w, pgx.ErrNoRows)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
