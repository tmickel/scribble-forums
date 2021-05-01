package auth

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(pwd string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func ComparePasswords(hashedPwd string, plainPwd []byte) bool {
	byteHash := []byte(hashedPwd)
	err := bcrypt.CompareHashAndPassword(byteHash, plainPwd)
	if err != nil {
		return false
	}
	return true
}

type SessionUser struct {
	Id       int64
	Username string
}

func ProcessSession(ctx context.Context, r *http.Request, db *sql.DB) context.Context {
	sessionId, err := r.Cookie("session_id")
	if err != nil {
		// no session set
		return ctx
	}

	var sessionUser SessionUser
	if err := db.QueryRow("SELECT id, username FROM users WHERE id=(SELECT user_id FROM sessions WHERE id=$1)", sessionId.Value).Scan(&sessionUser.Id, &sessionUser.Username); err != nil {
		// invalid session
		log.Println(err)
		return ctx
	}

	return context.WithValue(ctx, "sessionUser", sessionUser)
}

func CreateSession(w http.ResponseWriter, db *sql.DB, userId int64) error {
	sessionId := uuid.Must(uuid.NewRandom()).String()
	if _, err := db.Exec("INSERT INTO sessions (id, user_id) VALUES ($1, $2)", sessionId, userId); err != nil {
		return fmt.Errorf("could not create session")
	}
	http.SetCookie(w, &http.Cookie{Name: "session_id",
		Value:    sessionId,
		Expires:  time.Now().Add(365 * 24 * time.Hour),
		Secure:   true,
		HttpOnly: true,
	})
	return nil
}

func DeleteSession(w http.ResponseWriter, db *sql.DB, sessionId string) error {
	http.SetCookie(w, &http.Cookie{
		Name:     "session_id",
		Value:    "",
		Expires:  time.Now(),
		Secure:   true,
		HttpOnly: true,
	})
	if _, err := db.Exec("DELETE FROM sessions WHERE id=$1", sessionId); err != nil {
		return fmt.Errorf("could not delete session")
	}
	return nil
}
