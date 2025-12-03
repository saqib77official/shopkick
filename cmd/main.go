package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Suggestion struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`
	Suggestion string    `json:"suggestion"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

const schema = `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS suggestions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    suggestion TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_suggestions_created_at ON suggestions (created_at DESC);
`

func main() {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "/tmp/suggestions.db"
	}

	db, err := initDB(dbPath)
	if err != nil {
		log.Fatalf("init db (%s): %v", dbPath, err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.Handle("/api/suggestions", suggestionsCollection(db))
	mux.Handle("/api/suggestions/", suggestionItem(db))
	mux.Handle("/", http.FileServer(http.Dir("./static")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s (db: %s)\n", port, dbPath)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

func initDB(path string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func suggestionsCollection(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handleList(db, w, r)
		case http.MethodPost:
			handleCreate(db, w, r)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func suggestionItem(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		idStr := strings.TrimPrefix(r.URL.Path, "/api/suggestions/")
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil || id <= 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		switch r.Method {
		case http.MethodPut, http.MethodPatch:
			handleUpdate(db, w, r, id)
		case http.MethodDelete:
			handleDelete(db, w, r, id)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func handleList(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query(`SELECT id, name, suggestion, created_at, updated_at FROM suggestions ORDER BY created_at DESC`)
	if err != nil {
		http.Error(w, "query error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var out []Suggestion
	for rows.Next() {
		var s Suggestion
		if err := rows.Scan(&s.ID, &s.Name, &s.Suggestion, &s.CreatedAt, &s.UpdatedAt); err != nil {
			http.Error(w, "scan error", http.StatusInternalServerError)
			return
		}
		out = append(out, s)
	}
	respondJSON(w, out)
}

func handleCreate(db *sql.DB, w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name       string `json:"name"`
		Suggestion string `json:"suggestion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Suggestion = strings.TrimSpace(in.Suggestion)
	if in.Name == "" {
		in.Name = "Anonymous"
	}
	if in.Suggestion == "" {
		http.Error(w, "suggestion required", http.StatusBadRequest)
		return
	}

	res, err := db.Exec(`INSERT INTO suggestions (name, suggestion) VALUES (?, ?)`, in.Name, in.Suggestion)
	if err != nil {
		http.Error(w, "insert error", http.StatusInternalServerError)
		return
	}
	id, _ := res.LastInsertId()

	var s Suggestion
	if err := db.QueryRow(`SELECT id, name, suggestion, created_at, updated_at FROM suggestions WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Suggestion, &s.CreatedAt, &s.UpdatedAt); err != nil {
		http.Error(w, "fetch error", http.StatusInternalServerError)
		return
	}
	respondJSON(w, s)
}

func handleUpdate(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	var in struct {
		Name       *string `json:"name"`
		Suggestion *string `json:"suggestion"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Build update fields
	fields := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if in.Name != nil {
		*in.Name = strings.TrimSpace(*in.Name)
		if *in.Name == "" {
			*in.Name = "Anonymous"
		}
		fields = append(fields, "name = ?")
		args = append(args, *in.Name)
	}
	if in.Suggestion != nil {
		*in.Suggestion = strings.TrimSpace(*in.Suggestion)
		if *in.Suggestion == "" {
			http.Error(w, "suggestion required", http.StatusBadRequest)
			return
		}
		fields = append(fields, "suggestion = ?")
		args = append(args, *in.Suggestion)
	}
	if len(fields) == 0 {
		http.Error(w, "no fields to update", http.StatusBadRequest)
		return
	}
	fields = append(fields, "updated_at = CURRENT_TIMESTAMP")
	query := "UPDATE suggestions SET " + strings.Join(fields, ", ") + " WHERE id = ?"
	args = append(args, id)

	if _, err := db.Exec(query, args...); err != nil {
		http.Error(w, "update error", http.StatusInternalServerError)
		return
	}

	var s Suggestion
	if err := db.QueryRow(`SELECT id, name, suggestion, created_at, updated_at FROM suggestions WHERE id = ?`, id).
		Scan(&s.ID, &s.Name, &s.Suggestion, &s.CreatedAt, &s.UpdatedAt); err != nil {
		http.Error(w, "fetch error", http.StatusInternalServerError)
		return
	}
	respondJSON(w, s)
}

func handleDelete(db *sql.DB, w http.ResponseWriter, r *http.Request, id int64) {
	if _, err := db.Exec(`DELETE FROM suggestions WHERE id = ?`, id); err != nil {
		http.Error(w, "delete error", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func respondJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(true)
	if err := enc.Encode(v); err != nil {
		http.Error(w, "encode error", http.StatusInternalServerError)
	}
}
