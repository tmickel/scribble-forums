package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"github.com/bigfuncloud/bigfuncloud/programs/scribble-forums/auth"

	"github.com/lib/pq"
)

func register(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "form parse error")
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	if len(username) < 3 {
		setFlash(w, "Username must be at least 3 characters.", "danger")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	if len(password) < 5 {
		setFlash(w, "Password must be at least 5 characters.", "danger")
		http.Redirect(w, r, "/register", http.StatusFound)
		return
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not register: password hash failure")
		return
	}

	var userId int64
	if err := db.QueryRow(`INSERT INTO users ("username", "password") VALUES ($1, $2) RETURNING id`, username, hash).Scan(&userId); err != nil {
		if driverErr, ok := err.(*pq.Error); ok {
			if driverErr.Constraint == "username" {
				setFlash(w, "Username already exists. Go back and try again.", "danger")
				http.Redirect(w, r, "/register", http.StatusFound)
				return
			}
		}
		log.Printf("register: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not register: unknown query error")
		return
	}

	if err := auth.CreateSession(w, db, userId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not create session")
		return
	}

	setFlash(w, "Thanks for registering. You're now logged in.", "success")
	http.Redirect(w, r, "/", http.StatusFound)
}
