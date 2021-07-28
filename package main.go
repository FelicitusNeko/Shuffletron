package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

type Article struct {
	Id      string `json:"id"`
	Title   string `json:"Title"`
	Desc    string `json:"desc"`
	Content string `json:"content"`
}

// let's declare a global Articles array
// that we can then populate in our main function
// to simulate a database
var Articles []Article

func initDb() {
	db, err := sql.Open("sqlite3", "./shuffletron.sqlite3")

	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	sqlStmt := `
  CREATE TABLE IF NOT EXISTS lists (
    listId INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    listName VARCHAR NOT NULL
  );

  CREATE TABLE IF NOT EXISTS games (
    gameId INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    listId INTEGER NOT NULL REFERENCES lists(listId),
    gameName TEXT NOT NULL,
    displayName TEXT DEFAULT NULL,
    description TEXT DEFAULT NULL,
    weight INTEGER NOT NULL DEFAULT 1,
    status INTEGER NOT NULL DEFAULT 0,
    activeDisplayName TEXT GENERATED ALWAYS AS (IFNULL(displayName, gameName)) VIRTUAL
  );
  `

	_, err = db.Exec(sqlStmt)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
		return
	}
}

func returnAllArticles(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: returnAllArticles\n")
	json.NewEncoder(w).Encode(Articles)
}

func returnSingleArticle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: returnSingleArticle\n")
	vars := mux.Vars(r)
	key := vars["id"]

	for _, article := range Articles {
		if article.Id == key {
			json.NewEncoder(w).Encode(article)
		}
	}
}

func createNewArticle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: createNewArticle\n")

	reqBody, _ := ioutil.ReadAll(r.Body)
	var article Article
	json.Unmarshal(reqBody, &article)

	Articles = append(Articles, article)

	json.NewEncoder(w).Encode(article)
}

func deleteArticle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: deleteArticle\n")

	vars := mux.Vars(r)
	id := vars["id"]

	for index, article := range Articles {
		if article.Id == id {
			Articles = append(Articles[:index], Articles[index+1:]...)
		}
	}

}

func updateArticle(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: updateArticle\n")

	vars := mux.Vars(r)
	id := vars["id"]

	for index, article := range Articles {
		if article.Id == id {
			reqBody, _ := ioutil.ReadAll(r.Body)
			var updatedArticle Article
			json.Unmarshal(reqBody, &updatedArticle)
			Articles[index] = updatedArticle
			json.NewEncoder(w).Encode(updatedArticle)
		}
	}
}

func handleReqs() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.Handle("/", http.FileServer(http.Dir("./serve")))
	myRouter.HandleFunc("/articles", createNewArticle).Methods("POST")
	myRouter.HandleFunc("/articles", returnAllArticles)
	myRouter.HandleFunc("/articles/{id}", deleteArticle).Methods("DELETE")
	myRouter.HandleFunc("/articles/{id}", updateArticle).Methods("PUT")
	myRouter.HandleFunc("/articles/{id}", returnSingleArticle)
	log.Fatal(http.ListenAndServe(":8081", myRouter))
}

func main() {
	fmt.Println("Starting server")

	Articles = []Article{
		{Id: "1", Title: "Hello", Desc: "Article Description", Content: "Article Content"},
		{Id: "2", Title: "Hello 2", Desc: "Article Description", Content: "Article Content"},
	}

	//initDb()
	handleReqs()

}
