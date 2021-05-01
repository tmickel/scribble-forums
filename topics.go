package main

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/bigfuncloud/bigfuncloud/programs/scribble-forums/auth"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gomarkdown/markdown"
	"github.com/microcosm-cc/bluemonday"
)

func newTopic(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var userId int64
	sessionUserI := ctx.Value("sessionUser")
	if sessionUserI == nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "You are not logged in!")
		return
	}
	sessionUser := sessionUserI.(auth.SessionUser)
	userId = sessionUser.Id

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "form parse error")
		return
	}

	title := r.PostForm.Get("title")
	message := r.PostForm.Get("message")
	if len(title) < 5 {
		setFlash(w, "Title must be at least 5 characters.", "danger")
		http.Redirect(w, r, "/new", http.StatusFound)
		return
	}
	if len(message) < 1 {
		setFlash(w, "Message must be at least 1 characters.", "danger")
		http.Redirect(w, r, "/new", http.StatusFound)
		return
	}

	var topicId int64
	if err := db.QueryRow(`INSERT INTO topics ("user_id", "title", "latest_post_at") VALUES ($1, $2, $3) RETURNING id`, userId, title, time.Now()).Scan(&topicId); err != nil {
		log.Printf("create topic: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not create topic: unknown query error")
		return
	}

	var postId int64
	if err := db.QueryRow(`INSERT INTO posts ("user_id", "topic_id", "message", "created_at") VALUES ($1, $2, $3, $4) RETURNING id`,
		userId,
		topicId,
		message,
		time.Now(),
	).Scan(&postId); err != nil {
		log.Printf("create post: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not create post: unknown query error")
		return
	}

	setFlash(w, "Topic posted!", "success")
	http.Redirect(w, r, fmt.Sprintf("/topic/%d", topicId), http.StatusFound)
}

type Topic struct {
	Id           int64
	Title        string
	Creator      string
	LatestPostAt string
	LatestPoster string
	PostCount    int
}

func topicsListing(db *sql.DB) []Topic {
	rows, err := db.Query(`
                SELECT topics.id, topics.title, topics.latest_post_at, creator.username, latest_poster.username, count(posts)
                FROM topics
                LEFT JOIN users as creator ON (topics.user_id = creator.id)
                LEFT JOIN users as latest_poster ON (latest_poster.id = (SELECT user_id FROM posts WHERE topic_id=topics.id ORDER BY created_at DESC LIMIT 1)) 
                INNER JOIN posts ON (posts.topic_id = topics.id)
                GROUP BY topics.id, creator.username, latest_poster.username
                ORDER BY latest_post_at DESC
        `)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		t := Topic{}
		var latestPostAt time.Time
		if err := rows.Scan(&t.Id, &t.Title, &latestPostAt, &t.Creator, &t.LatestPoster, &t.PostCount); err != nil {
			log.Printf("scan err: %v", err)
			continue
		}
		loc, err := time.LoadLocation("America/New_York")
		if err == nil {
			latestPostAt = latestPostAt.In(loc)
		}
		t.LatestPostAt = latestPostAt.Format("Mon Jan 2, 2006 3:04PM")
		topics = append(topics, t)
	}
	return topics
}

type Post struct {
	Creator   string
	CreatedAt string
	Message   template.HTML
}

func postsListing(db *sql.DB, topicId int64) []Post {
	rows, err := db.Query(`
                SELECT creator.username, posts.created_at, posts.message
                FROM posts
                LEFT JOIN users as creator ON (posts.user_id = creator.id)
                WHERE topic_id=$1
                ORDER BY created_at ASC
        `, topicId)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer rows.Close()

	var posts []Post
	for rows.Next() {
		p := Post{}
		var createdAt time.Time
		var messageMd string
		if err := rows.Scan(&p.Creator, &createdAt, &messageMd); err != nil {
			log.Printf("scan err: %v", err)
			continue
		}
		loc, err := time.LoadLocation("America/New_York")
		if err == nil {
			createdAt = createdAt.In(loc)
		}
		p.CreatedAt = createdAt.Format("Mon Jan 2, 2006 3:04PM")

		maybeUnsafe := markdown.ToHTML([]byte(messageMd), nil, nil)
		p.Message = template.HTML(bluemonday.UGCPolicy().SanitizeBytes(maybeUnsafe))

		posts = append(posts, p)
	}
	return posts
}

func topicTitle(db *sql.DB, topicId int64) string {
	var topicTitle string
	if err := db.QueryRow(`
                SELECT topics.title
                FROM topics
                WHERE id=$1
        `, topicId).Scan(&topicTitle); err != nil {
		log.Println(err)
		return ""
	}
	return topicTitle
}

func newPost(ctx context.Context, w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var userId int64
	sessionUserI := ctx.Value("sessionUser")
	if sessionUserI == nil {
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, "You are not logged in!")
		return
	}
	sessionUser := sessionUserI.(auth.SessionUser)
	userId = sessionUser.Id

	if err := r.ParseForm(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, "form parse error")
		return
	}

	topicId, err := strconv.Atoi(r.PostForm.Get("topicId"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprintf(w, "not found")
		return
	}

	res, err := db.Exec(`UPDATE topics SET "latest_post_at"=$1 WHERE id=$2`, time.Now(), topicId)
	if err != nil {
		log.Printf("update topic: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not update topic: unknown query error")
		return
	}

	affected, err := res.RowsAffected()
	if err != nil || affected != 1 {
		log.Printf("update topic: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not update topic: topic does not exist?")
		return
	}

	message := r.PostForm.Get("message")
	if len(message) < 1 {
		setFlash(w, "Message must be at least 1 characters.", "danger")
		http.Redirect(w, r, "/new", http.StatusFound)
		return
	}

	var postId int64
	if err := db.QueryRow(`INSERT INTO posts ("user_id", "topic_id", "message", "created_at") VALUES ($1, $2, $3, $4) RETURNING id`,
		userId,
		topicId,
		message,
		time.Now(),
	).Scan(&postId); err != nil {
		log.Printf("create post: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "could not create post: unknown query error")
		return
	}

	setFlash(w, "Posted!", "success")
	http.Redirect(w, r, fmt.Sprintf("/topic/%d", topicId), http.StatusFound)
}
