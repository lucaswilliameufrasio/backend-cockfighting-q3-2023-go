package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ---------------------------------------------------------------------------
// Domain
// ---------------------------------------------------------------------------

type Person struct {
	ID         string    `json:"id"`
	Apelido    string    `json:"apelido"`
	Nome       string    `json:"nome"`
	Nascimento string    `json:"nascimento"`
	Stack      *[]string `json:"stack"`
}

type CreatePersonRequest struct {
	Apelido    string   `json:"apelido"`
	Nome       string   `json:"nome"`
	Nascimento string   `json:"nascimento"`
	Stack      []string `json:"stack,omitempty"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func jsonResponse(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func errorResponse(w http.ResponseWriter, status int, msg string) {
	jsonResponse(w, status, map[string]string{"message": msg})
}

func isDateValid(date string) bool {
	if len(date) != 10 || date[4] != '-' || date[7] != '-' {
		return false
	}
	y, m, d := 0, 0, 0
	if _, err := fmt.Sscanf(date, "%d-%d-%d", &y, &m, &d); err != nil {
		return false
	}
	if y < 1800 || y > 9999 || m < 1 || m > 12 || d < 1 || d > 31 {
		return false
	}
	if m == 2 {
		leap := (y%4 == 0 && y%100 != 0) || y%400 == 0
		if leap {
			return d <= 29
		}
		return d <= 28
	}
	if m == 4 || m == 6 || m == 9 || m == 11 {
		return d <= 30
	}
	return true
}

func toPGArray(v []string) string {
	if len(v) == 0 {
		return "{}"
	}
	parts := make([]string, len(v))
	for i, s := range v {
		parts[i] = `"` + s + `"`
	}
	return "{" + strings.Join(parts, ",") + "}"
}

// ---------------------------------------------------------------------------
// Server
// ---------------------------------------------------------------------------

type Server struct {
	db *pgxpool.Pool
}

func NewServer(db *pgxpool.Pool) *Server {
	return &Server{db: db}
}

func (s *Server) healthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	var result int
	err := s.db.QueryRow(ctx, "SELECT 1").Scan(&result)
	if err != nil {
		log.Printf("health check db error: %v", err)
		w.WriteHeader(http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (s *Server) createPerson(w http.ResponseWriter, r *http.Request) {
	var req CreatePersonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		errorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	if req.Apelido == "" || req.Nome == "" || !isDateValid(req.Nascimento) {
		errorResponse(w, http.StatusUnprocessableEntity, "Invalid fields")
		return
	}
	if len(req.Apelido) > 32 || len(req.Nome) > 100 {
		errorResponse(w, http.StatusUnprocessableEntity, "Field too long")
		return
	}
	for _, s := range req.Stack {
		if len(s) > 32 {
			errorResponse(w, http.StatusUnprocessableEntity, "Stack item too long")
			return
		}
	}

	id := uuid.Must(uuid.NewV7()).String()
	stackStr := toPGArray(req.Stack)

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var insertedID string
	err := s.db.QueryRow(ctx,
		`INSERT INTO people (id, nickname, name, birth_date, stack)
		 VALUES ($1, $2, $3, TO_DATE($4, 'YYYY-MM-DD'), $5::varchar[])
		 ON CONFLICT (nickname) DO NOTHING
		 RETURNING id`,
		id, req.Apelido, req.Nome, req.Nascimento, stackStr,
	).Scan(&insertedID)

	if err != nil {
		if err.Error() == "no rows in result set" {
			errorResponse(w, http.StatusUnprocessableEntity, "Conflict")
			return
		}
		log.Printf("insert error: %v", err)
		errorResponse(w, http.StatusServiceUnavailable, "DB error")
		return
	}

	w.Header().Set("Location", "/pessoas/"+insertedID)
	jsonResponse(w, http.StatusCreated, map[string]string{"id": insertedID})
}

func (s *Server) getPersonByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/pessoas/")
	if len(id) != 36 {
		errorResponse(w, http.StatusNotFound, "Not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var p Person
	err := s.db.QueryRow(ctx,
		`SELECT id, nickname, name, birth_date::text, COALESCE(stack, '{}') FROM people WHERE id = $1`,
		id,
	).Scan(&p.ID, &p.Apelido, &p.Nome, &p.Nascimento, &p.Stack)
	if p.Stack != nil && len(*p.Stack) == 0 {
		p.Stack = nil
	}

	if err != nil {
		if err.Error() == "no rows in result set" {
			errorResponse(w, http.StatusNotFound, "Not found")
			return
		}
		log.Printf("query error: %v", err)
		errorResponse(w, http.StatusServiceUnavailable, "DB error")
		return
	}

	jsonResponse(w, http.StatusOK, p)
}

func (s *Server) searchPeople(w http.ResponseWriter, r *http.Request) {
	term := r.URL.Query().Get("t")
	if term == "" {
		errorResponse(w, http.StatusBadRequest, "Missing term")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	pattern := "%" + term + "%"
	rows, err := s.db.Query(ctx,
		`SELECT id, nickname, name, birth_date::text, COALESCE(stack, '{}') FROM people
		 WHERE searchable ILIKE $1 LIMIT 50`,
		pattern,
	)
	if err != nil {
		log.Printf("search error: %v", err)
		errorResponse(w, http.StatusServiceUnavailable, "DB error")
		return
	}
	defer rows.Close()

	people := []Person{}
	for rows.Next() {
		var p Person
		if err := rows.Scan(&p.ID, &p.Apelido, &p.Nome, &p.Nascimento, &p.Stack); err != nil {
			log.Printf("scan error: %v", err)
			continue
		}
		if p.Stack != nil && len(*p.Stack) == 0 {
			p.Stack = nil
		}
		people = append(people, p)
	}

	jsonResponse(w, http.StatusOK, people)
}

func (s *Server) countPeople(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	var count int
	err := s.db.QueryRow(ctx, "SELECT COUNT(*) FROM people").Scan(&count)
	if err != nil {
		log.Printf("count error: %v", err)
		errorResponse(w, http.StatusServiceUnavailable, "DB error")
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(strconv.Itoa(count)))
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dsn := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s?sslmode=disable",
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "fight"),
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_NAME", "fight"),
	)

	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		log.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Close()

	s := NewServer(pool)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health-check", s.healthCheck)
	mux.HandleFunc("POST /pessoas", s.createPerson)
	mux.HandleFunc("GET /pessoas/{id}", s.getPersonByID)
	mux.HandleFunc("GET /pessoas", s.searchPeople)
	mux.HandleFunc("GET /contagem-pessoas", s.countPeople)

	addr := "0.0.0.0:" + port
	log.Printf("Listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
