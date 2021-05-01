package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bigfuncloud/bigfuncloud/programs/scribble-forums/auth"
	"html/template"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

func run() error {
	db, err := sql.Open("postgres", os.Getenv("PG_DSN"))
	if err != nil {
		return err
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { serveRequest(w, r, db) })
	http.ListenAndServe(":80", nil)

	return nil
}

type TemplateData struct {
	LoggedIn   bool
	Username   string
	FlashText  string
	FlashStyle string

	// xxx: ugly - data for index
	TopicsListing []Topic

	// xxx: ugly - data for topic
	TopicId      int64
	TopicTitle   string
	PostsListing []Post
}

func serveRequest(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// static
	if r.URL.Path == "/paper.min.css" {
		http.ServeFile(w, r, "./static/paper.min.css")
		return
	}

	if r.URL.Path == "/jdenticon.min.js" {
		http.ServeFile(w, r, "./static/jdenticon.min.js")
		return
	}

	ctx := auth.ProcessSession(r.Context(), r, db)

	// forms
	if r.Method == "POST" {
		processForm(ctx, w, r, db)
		return
	}

	// templates
	tpl, err := template.ParseGlob("./templates/*.gohtml")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, fmt.Sprintf("template parse error: %v", err))
		return
	}

	templateData := TemplateData{}

	name := strings.TrimPrefix(r.URL.Path, "/")
	if name == "" {
		name = "index"
	}
	if strings.HasPrefix(name, "topic") {
		name = "topic"
	}

	if tpl.Lookup(name) == nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "not found")
		return
	}

	if name == "index" {
		templateData.TopicsListing = topicsListing(db)
	}

	if name == "topic" {
		topicIdStr := strings.TrimPrefix(r.URL.Path, "/topic/")
		topicId, err := strconv.Atoi(topicIdStr)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "not found")
			return
		}
		posts := postsListing(db, int64(topicId))

		templateData.TopicId = int64(topicId)
		templateData.PostsListing = posts
		templateData.TopicTitle = topicTitle(db, int64(topicId))
	}

	sessionUserI := ctx.Value("sessionUser")
	if sessionUserI != nil {
		sessionUser := sessionUserI.(auth.SessionUser)
		templateData.LoggedIn = true
		templateData.Username = sessionUser.Username
	}

	if flashText, err := r.Cookie("flash_text"); err == nil {
		templateData.FlashText = flashText.Value
	}

	if flashStyle, err := r.Cookie("flash_style"); err == nil {
		templateData.FlashStyle = flashStyle.Value
	} else {
		templateData.FlashStyle = "primary"
	}
	clearFlash(w)

	if err := tpl.ExecuteTemplate(w, name, templateData); err != nil {
		fmt.Fprint(w, fmt.Sprintf("template error: %v", err))
	}
}

func processForm(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	if r.URL.Path == "/register" {
		register(w, r, db)
		return
	}
	if r.URL.Path == "/login" {
		login(w, r, db)
		return
	}
	if r.URL.Path == "/logout" {
		logout(w, r, db)
		return
	}
	if r.URL.Path == "/new" {
		newTopic(ctx, w, r, db)
		return
	}
	if r.URL.Path == "/new-post" {
		newPost(ctx, w, r, db)
		return
	}
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
