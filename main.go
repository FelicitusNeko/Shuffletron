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
	"sync"
	"time"

	"github.com/gempir/go-twitch-irc/v2"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	_ "github.com/mattn/go-sqlite3"
)

const port = 42069

var wsListMutex = &sync.Mutex{}
var wsWriteMutex = &sync.Mutex{}

// -------------============== TWITCHWS CLASS

type TwitchWS struct {
	msg  chan twitch.PrivateMessage
	conn *websocket.Conn
	open bool
}

func (ws TwitchWS) wsWriter() {
	for {
		msg := <-ws.msg
		if !ws.open {
			fmt.Println("Stopping wsWriter")
			return
		}

		outMsg, err := json.Marshal(TwitchWSMsg{
			Id:          msg.ID,
			DisplayName: msg.User.DisplayName,
			DisplayCol:  msg.User.Color,
			Channel:     msg.Channel,
			Time:        msg.Time.Unix(),
			Message:     msg.Message,
		})
		if err != nil {
			log.Println("Error in marshall op:", err)
		}

		wsWriteMutex.Lock()
		if err := ws.conn.WriteMessage(
			websocket.TextMessage, []byte(outMsg),
		); err != nil {
			log.Println("Error in WS write op:", err)
			wsWriteMutex.Unlock()
			break
		} else {
			wsWriteMutex.Unlock()
		}
	}
}

// define a wsReader which will listen for
// new messages being sent to our WebSocket
// endpoint
func (ws TwitchWS) wsReader() {
	defer func() {
		fmt.Println("Attempting to terminate WS goroutines")
		ws.open = false
		wsListMutex.Lock()
		fmt.Println("Now delisting this WS")
		delisted := false
		for x, delWS := range openWS {
			if !delWS.open {
				fmt.Println("Found WS to delist")
				delisted = true
				openWS = append(openWS[:x], openWS[x+1:]...)
			} else {
				fmt.Println("Open WS stays open")
			}
		}
		if !delisted {
			fmt.Println("Didn't find WS to delist")
		}
		wsListMutex.Unlock()
		if ws.msg != nil {
			ws.msg <- twitch.PrivateMessage{}
		}
	}()

	for {
		// read in a message
		messageType, p, err := ws.conn.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		// print out that message for clarity
		fmt.Println(string(p))

		wsWriteMutex.Lock()
		if err := ws.conn.WriteMessage(messageType, p); err != nil {
			log.Println("Error in WS read op:", err)
			wsWriteMutex.Unlock()
			break
		} else {
			wsWriteMutex.Unlock()
		}

	}
}

var openWS []TwitchWS

// -------------=========== MAIN CODE

type Article struct {
	Id      string `json:"id"`
	Title   string `json:"Title"`
	Desc    string `json:"desc"`
	Content string `json:"content"`
}

type TwitchWSMsg struct {
	Id          string `json:"id"`
	DisplayName string `json:"displayName"`
	DisplayCol  string `json:"displayCol"`
	Channel     string `json:"channel"`
	Message     string `json:"msg"`
	Time        int64  `json:"time"`
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

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	// TODO: look more into CORS
	upgrader.CheckOrigin = func(r *http.Request) bool { return true }
	log.Println("WS connection request")

	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
	}

	// helpful log statement to show connections
	log.Println("Client Connected")

	newWS := TwitchWS{nil, ws, true}
	go newWS.wsReader()
	wsListMutex.Lock()
	openWS = append(openWS, newWS)
	wsListMutex.Unlock()
}

func handleReqs( /*twitchchat chan twitch.PrivateMessage*/ ) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/ws", wsEndpoint)

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

func twitchTransmitter(msg chan twitch.PrivateMessage) {
	for {
		msgIn := <-msg
		sentTo := 0
		wsListMutex.Lock()
		for _, ws := range openWS {
			if !ws.open {
				continue
			}
			if ws.msg == nil {
				ws.msg = make(chan twitch.PrivateMessage)
				go ws.wsWriter()
			}
			ws.msg <- msgIn
			sentTo++
		}
		fmt.Printf("Sent to %d/%d client(s)\n", sentTo, len(openWS))
		wsListMutex.Unlock()
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
	go twitchTransmitter(twitchchat)
	handleReqs()
}
