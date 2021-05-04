package main

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/bigfuncloud/bigfuncloud/programs/scribble-forums/auth"
)

func login(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "form parse error")
		return
	}

	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	var userId int64
	var passwordHashed string
	if err := db.QueryRow("SELECT id, password FROM users WHERE username=$1", username).Scan(&userId, &passwordHashed); err != nil {
		setFlash(w, "Incorrect username or password.", "danger")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if !auth.ComparePasswords(passwordHashed, []byte(password)) {
		setFlash(w, "Incorrect username or password.", "danger")
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	if err := auth.CreateSession(w, db, userId); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not create session")
		return
	}

	setFlash(w, "Logged in successfully.", "success")
	http.Redirect(w, r, "/", http.StatusFound)
}
