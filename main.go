package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

const port = 42069

type ChannelTray struct {
	twitchmsg chan twitch.PrivateMessage
}

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

// We'll need to define an Upgrader
// this will require a Read and Write buffer size
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

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

func wsEndpoint(w http.ResponseWriter, r *http.Request, twitchchat chan twitch.PrivateMessage) {
	// TODO: look more into CORS
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	log.Println("WS connection request")

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	// helpful log statement to show connections
	log.Println("Client Connected")
	/*err = ws.WriteMessage(websocket.TextMessage, []byte("Hi Client!"))
	if err != nil {
		log.Println(err)
	}*/

	go wsWriter(ws, twitchchat)
	wsReader(ws)
}

func wsWriter(conn *websocket.Conn, twitchchat chan twitch.PrivateMessage) {
	for {
		msg := <-twitchchat

		if err := conn.WriteMessage(
			websocket.TextMessage, []byte(msg.User.DisplayName+": "+msg.Message),
		); err != nil {
			log.Println(err)
			break
		}
	}
}

// define a wsReader which will listen for
// new messages being sent to our WebSocket
// endpoint
func wsReader(conn *websocket.Conn) {
	for {
		// read in a message
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		// print out that message for clarity
		fmt.Println(string(p))

		if err := conn.WriteMessage(messageType, p); err != nil {
			log.Println(err)
			break
		}

	}
}

func handleReqs(twitchchat chan twitch.PrivateMessage) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		wsEndpoint(w, r, twitchchat)
	})

	router.HandleFunc("/articles", createNewArticle).Methods("POST")
	router.HandleFunc("/articles", returnAllArticles)
	router.HandleFunc("/articles/{id}", deleteArticle).Methods("DELETE")
	router.HandleFunc("/articles/{id}", updateArticle).Methods("PUT")
	router.HandleFunc("/articles/{id}", returnSingleArticle)

	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat("./build/" + r.URL.Path[1:]); err == nil {
			fmt.Println("sending " + r.RequestURI)
			http.ServeFile(w, r, "./build/"+r.URL.Path[1:])
		} else {
			fmt.Println("req not found " + r.RequestURI)
			http.ServeFile(w, r, "./build/index.html")
		}
	})
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), router))

}

func twitchHandler(twitchchat chan twitch.PrivateMessage) {
	client := twitch.NewAnonymousClient()

	//defer client.Disconnect()

	client.OnPrivateMessage(func(msg twitch.PrivateMessage) {
		fmt.Printf("[%s] %s: %s\n", msg.Channel, msg.User.DisplayName, msg.Message)
		if msg.Bits > 0 {
			fmt.Printf("%s has given %d bit(s) to %s", msg.User.DisplayName, msg.Bits, msg.Channel)
		}
		twitchchat <- msg
	})

	client.Join("kewliomzx")

	for {
		err := client.Connect()
		if err == nil {
			break
		} else {
			fmt.Println("Disconnected from Twitch: " + err.Error())
			time.Sleep(5000)
		}
	}
}

func main() {
	fmt.Println("Starting server")
	twitchchat := make(chan twitch.PrivateMessage)

	Articles = []Article{
		{Id: "1", Title: "Hello", Desc: "Article Description", Content: "Article Content"},
		{Id: "2", Title: "Hello 2", Desc: "Article Description", Content: "Article Content"},
	}

	//initDb()
	go twitchHandler(twitchchat)
	handleReqs(twitchchat)
}
