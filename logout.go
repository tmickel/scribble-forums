package main

import (
	"database/sql"
	"log"
	"net/http"
	"github.com/bigfuncloud/bigfuncloud/programs/scribble-forums/auth"
)

func logout(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	sessionId, err := r.Cookie("session_id")
	if err != nil {
		http.Redirect(w, r, "/", http.StatusFound)
		return
	}
	if err := auth.DeleteSession(w, db, sessionId.Value); err != nil {
		log.Printf("deleteSession: %v", err)
	}
	setFlash(w, "Logged out successfully.", "success")
	http.Redirect(w, r, "/", http.StatusFound)
}
