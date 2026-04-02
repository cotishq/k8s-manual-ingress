package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"
)

type User struct{
	ID 		int 	`json:"id"`
	Name    string 	`json:"name"`
}

type CreateUserRequest struct{
	Name 	string	`json:"name"`
}

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@db.default.svc.cluster.local:5432/postgres?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	if err := initDB(db); err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/users", usersHandler(db))

	log.Println("server running on port 3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}

func initDB(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			name VARCHAR(255) NOT NULL
		)
	`)
	return err
}

func usersHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.Method {
		case http.MethodGet:
			rows, err := db.Query("SELECT id, name FROM users ORDER BY id")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			defer rows.Close()

			var users []User
			for rows.Next() {
				var u User
				if err := rows.Scan(&u.ID, &u.Name); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				users = append(users, u)
			}

			if err := json.NewEncoder(w).Encode(users); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		case http.MethodPost:
			var req CreateUserRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			var user User
			err := db.QueryRow(
				"INSERT INTO users (name) VALUES ($1) RETURNING id, name",
				req.Name,
			).Scan(&user.ID, &user.Name)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			w.WriteHeader(http.StatusOK)
			if err := json.NewEncoder(w).Encode(user); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}