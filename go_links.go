package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
)

type linkResp struct {
	Link string `json:"link"`
}

type errResp struct {
	Error string `json:"error"`
}

var db *sql.DB

func main() {
	var err error

	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "3000"
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", http.StripPrefix("/", fs))

	http.HandleFunc("/api/", handler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func handler(w http.ResponseWriter, r *http.Request) {
	path := strings.Split(r.URL.Path, "/")[2:]

	if path[0] == "new" && len(path) > 1 && len(path[1]) > 0 {
		var link string

		if strings.HasPrefix(path[1], "http://") {
			link = path[1]
		} else {
			link = "http://" + path[1]
		}

		var id int
		err := db.QueryRow("INSERT INTO links(link) values($1) RETURNING id", link).Scan(&id)
		if err != nil {
			log.Println(err)
		}

		resp := linkResp{fmt.Sprintf("%s/%d", r.Host, id)}
		js, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	} else if len(path[0]) > 0 {
		id, err := strconv.Atoi(path[0])
		if err != nil {
			log.Println("Incorrect id")
			return
		}
		var link string
		err = db.QueryRow("SELECT link FROM links WHERE id = $1", id).Scan(&link)
		switch {
		case err == sql.ErrNoRows:
			http.Error(w, "404 not found", http.StatusNotFound)
		case err != nil:
			http.Error(w, "500 internal server eror", http.StatusInternalServerError)
		default:
			http.Redirect(w, r, link, 301)
		}
	} else {
		http.Redirect(w, r, "/", 301)
	}
}
