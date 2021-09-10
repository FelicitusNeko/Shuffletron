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

	"github.com/Thor-x86/nullable"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/imdario/mergo"
	_ "github.com/mattn/go-sqlite3"
)

const port = 42069

var db *sql.DB

var wsListMutex = &sync.Mutex{}
var wsWriteMutex = &sync.Mutex{}
var dbAccessMutex = &sync.Mutex{}

// -------------============== TWITCHWS CLASS

type TwitchWS struct {
	msg  chan twitch.PrivateMessage
	conn *websocket.Conn
	id   int
	open bool
}

var openWS []TwitchWS
var wsId int

func (ws TwitchWS) wsWriter() {
	for {
		msg := <-ws.msg
		if !ws.open {
			fmt.Println("Stopping wsWriter")
			return
		}

		var outEmotes []TwitchWSMsgEmote
		for _, inEmote := range msg.Emotes {
			outEmotes = append(outEmotes, TwitchWSMsgEmote{
				Name: inEmote.Name,
				Id:   inEmote.ID,
			})
		}

		outMsg, err := json.Marshal(TwitchWSMsg{
			Id:          msg.ID,
			DisplayName: msg.User.DisplayName,
			DisplayCol:  msg.User.Color,
			Channel:     msg.Channel,
			Time:        msg.Time.Unix(),
			Message:     msg.Message,
			Emotes:      outEmotes,
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
			if delWS.id == ws.id {
				fmt.Println("Found WS to delist")
				delisted = true
				openWS = append(openWS[:x], openWS[x+1:]...)
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

// -------------=========== LISTS ENDPOINTS
type STList struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

func returnAllLists(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: returnAllLists\n")

	stmt := `SELECT * FROM lists`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	rows, err := db.Query(stmt)
	if err != nil {
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
		return
	}
	defer rows.Close()

	var lists []STList
	for rows.Next() {
		var list STList
		if err := rows.Scan(&list.Id, &list.Name); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
		}
		lists = append(lists, list)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("%q: after exec %s\n", err, stmt)
	}

	json.NewEncoder(w).Encode(lists)
}

func returnSingleList(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: returnSingleList\n")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// TODO: output error data "Invalid Id"
		return
	}

	stmt := `SELECT * FROM lists WHERE listId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
	} else {
		var list STList
		if err := row.Scan(&list.Id, &list.Name); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			// TODO: add error output
		} else {
			json.NewEncoder(w).Encode(list)
		}
	}
}

func createNewList(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: createNewList\n")

	if reqBody, err := ioutil.ReadAll(r.Body); err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		// TODO: add error output
	} else {
		var list STList
		err := json.Unmarshal(reqBody, &list)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			// TODO: add error output
			return
		}

		dbAccessMutex.Lock()
		defer dbAccessMutex.Unlock()

		stmt := `
			INSERT INTO lists (listName)
			VALUES (?)
		`

		result, err := db.Exec(stmt, list.Name)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			// TODO: add error ouptut
		} else {
			list.Id, _ = result.LastInsertId()
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(list)
		}
	}
}

func updateList(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: updateList\n")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// TODO: output error data "Invalid Id"
		return
	}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		// TODO: add error output
		return
	}

	stmt := `SELECT * FROM lists WHERE listId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
	} else {
		var listRetrieve STList
		if err := row.Scan(&listRetrieve.Id, &listRetrieve.Name); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			// TODO: add error output
		} else {
			var listUpdate STList
			err := json.Unmarshal(reqBody, &listUpdate)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				// TODO: add error output
				return
			}

			listUpdate.Id = listRetrieve.Id
			err = mergo.Merge(&listRetrieve, listUpdate, mergo.WithOverride)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				// TODO: add error output
				return
			}

			stmt := `
				UPDATE lists
				SET listName = ?
				WHERE listId = ?
			`

			if result, err := db.Exec(stmt, listRetrieve.Name, listRetrieve.Id); err != nil {
				fmt.Printf("err: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				// TODO: add error output
			} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
				w.WriteHeader(http.StatusNotFound)
				// TODO: add error output
				return
			} else {
				json.NewEncoder(w).Encode(listRetrieve)
			}
		}
	}
}

func deleteList(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: deleteList\n")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// TODO: output error data "Invalid Id"
		return
	}

	stmt := `
		DELETE FROM lists
		WHERE listId = ?
	`

	if result, err := db.Exec(stmt, id); err != nil {
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
		return
	} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
		w.WriteHeader(http.StatusNotFound)
		// TODO: add error output
		return
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// -------------=========== GAMES ENDPOINTS
type STGame struct {
	Id          int64           `json:"id"`
	ListId      int64           `json:"listId"`
	Name        string          `json:"name"`
	DisplayName nullable.String `json:"displayName"`
	Description nullable.String `json:"description"`
	Weight      nullable.Int    `json:"weight"`
	Status      nullable.Int    `json:"status"`
}

func returnAllGames(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: returnAllGames\n")

	stmt := `SELECT * FROM games`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	rows, err := db.Query(stmt)
	if err != nil {
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
		return
	}
	defer rows.Close()

	var games []STGame
	var activeDisplayName string
	for rows.Next() {
		var game STGame
		if err := rows.Scan(&game.Id, &game.ListId, &game.Name, &game.DisplayName, &game.Description,
			&game.Weight, &game.Status, &activeDisplayName); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
		}
		games = append(games, game)
	}

	if err = rows.Err(); err != nil {
		fmt.Printf("%q: after exec %s\n", err, stmt)
	}

	json.NewEncoder(w).Encode(games)
}

func returnSingleGame(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: returnSingleGame\n")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// TODO: output error data "Invalid Id"
		return
	}

	stmt := `SELECT * FROM games WHERE gameId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
	} else {
		var game STGame
		var activeDisplayName string
		if err := row.Scan(&game.Id, &game.ListId, &game.Name, &game.DisplayName, &game.Description,
			&game.Weight, &game.Status, &activeDisplayName); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			// TODO: add error output
		} else {
			json.NewEncoder(w).Encode(game)
		}
	}
}

func createNewGame(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: createNewGame\n")

	if reqBody, err := ioutil.ReadAll(r.Body); err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		// TODO: add error output
	} else {
		var game STGame
		err := json.Unmarshal(reqBody, &game)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			w.WriteHeader(http.StatusBadRequest)
			// TODO: add error output
			return
		}

		stmt := `
			INSERT INTO games (listId, gameName, displayName, description, weight, status)
			VALUES (?, ?, ?, ?, ?, ?)
		`

		if game.Weight.Get() == nil {
			weightDefault := 1
			game.Weight.Set(&weightDefault)
		}

		if game.Status.Get() == nil {
			statusDefault := 0
			game.Status.Set(&statusDefault)
		}

		dbAccessMutex.Lock()
		defer dbAccessMutex.Unlock()

		result, err := db.Exec(stmt, game.ListId, game.Name, game.DisplayName, game.Description,
			game.Weight, game.Status)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			// TODO: add error ouptut
		} else {
			game.Id, _ = result.LastInsertId()
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(game)
		}
	}
}

func updateGame(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: updateList\n")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// TODO: output error data "Invalid Id"
		return
	}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		w.WriteHeader(http.StatusBadRequest)
		// TODO: add error output
		return
	}

	stmt := `SELECT * FROM games WHERE gameId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
	} else {
		var gameRetrieve STGame
		var activeDisplayName string
		if err := row.Scan(&gameRetrieve.Id, &gameRetrieve.ListId, &gameRetrieve.Name,
			&gameRetrieve.DisplayName, &gameRetrieve.Description, &gameRetrieve.Weight,
			&gameRetrieve.Weight, &activeDisplayName); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				w.WriteHeader(http.StatusNotFound)
			} else {
				w.WriteHeader(http.StatusInternalServerError)
			}
			// TODO: add error output
		} else {
			var gameUpdate STGame
			err := json.Unmarshal(reqBody, &gameUpdate)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				w.WriteHeader(http.StatusBadRequest)
				// TODO: add error output
				return
			}

			gameUpdate.Id = gameRetrieve.Id
			err = mergo.Merge(&gameRetrieve, gameUpdate, mergo.WithOverride)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				// TODO: add error output
				return
			}

			stmt := `
				UPDATE games
				SET listId = ?,
					gameName = ?,
					displayName = ?,
					description = ?,
					weight = ?,
					status = ?
				WHERE gameId = ?
			`

			if result, err := db.Exec(stmt, gameRetrieve.ListId, gameRetrieve.Name, gameRetrieve.DisplayName,
				gameRetrieve.Description, gameRetrieve.Weight, gameRetrieve.Status,
				gameRetrieve.Id); err != nil {

				fmt.Printf("err: %v\n", err)
				w.WriteHeader(http.StatusInternalServerError)
				// TODO: add error output
			} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
				w.WriteHeader(http.StatusNotFound)
				// TODO: add error output
				return
			} else {
				json.NewEncoder(w).Encode(gameRetrieve)
			}
		}
	}
}

func deleteGame(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: deleteGame\n")
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		// TODO: output error data "Invalid Id"
		return
	}

	stmt := `
		DELETE FROM games
		WHERE gameId = ?
	`

	if result, err := db.Exec(stmt, id); err != nil {
		fmt.Printf("%q: during query %s\n", err, stmt)
		w.WriteHeader(http.StatusInternalServerError)
		// TODO: add error output
		return
	} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
		w.WriteHeader(http.StatusNotFound)
		// TODO: add error output
		return
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// -------------=========== MAIN CODE

type TwitchWSMsg struct {
	Id          string             `json:"id"`
	DisplayName string             `json:"displayName"`
	DisplayCol  string             `json:"displayCol"`
	Channel     string             `json:"channel"`
	Message     string             `json:"msg"`
	Time        int64              `json:"time"`
	Emotes      []TwitchWSMsgEmote `json:"emotes"`
}

type TwitchWSMsgEmote struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

// We'll need to define an Upgrader
// this will require a Read and Write buffer size
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func initDb() {
	sqlStmt := `
  CREATE TABLE IF NOT EXISTS lists (
    listId INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    listName VARCHAR NOT NULL
  );

	CREATE TABLE games (
    gameId INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT,
    listId INTEGER NOT NULL,
    gameName TEXT NOT NULL,
    displayName TEXT DEFAULT NULL,
    description TEXT DEFAULT NULL,
    weight INTEGER NOT NULL DEFAULT 1,
    status INTEGER NOT NULL DEFAULT 0,
    activeDisplayName TEXT GENERATED ALWAYS AS (IFNULL(displayName, gameName)) VIRTUAL,
		FOREIGN KEY (listId) REFERENCES lists(listId) ON UPDATE CASCADE ON DELETE CASCADE
  );
  `

	dbAccessMutex.Lock()
	_, err := db.Exec(sqlStmt)
	if err != nil {
		log.Panicf("%q: %s\n", err, sqlStmt)
		return
	}
	dbAccessMutex.Unlock()
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

	newWS := TwitchWS{nil, ws, wsId, true}
	wsId++
	go newWS.wsReader()
	wsListMutex.Lock()
	openWS = append(openWS, newWS)
	wsListMutex.Unlock()
}

func handleReqs( /*twitchchat chan twitch.PrivateMessage*/ ) {
	router := mux.NewRouter().StrictSlash(true)
	router.HandleFunc("/ws", wsEndpoint)

	router.HandleFunc("/lists", createNewList).Methods("POST")
	router.HandleFunc("/lists", returnAllLists)
	router.HandleFunc("/lists/{id}", deleteList).Methods("DELETE")
	router.HandleFunc("/lists/{id}", updateList).Methods("PUT")
	router.HandleFunc("/lists/{id}", returnSingleList)

	router.HandleFunc("/games", createNewGame).Methods("POST")
	router.HandleFunc("/games", returnAllGames)
	router.HandleFunc("/games/{id}", deleteGame).Methods("DELETE")
	router.HandleFunc("/games/{id}", updateGame).Methods("PUT")
	router.HandleFunc("/games/{id}", returnSingleGame)

	router.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := os.Stat("./build/" + r.URL.Path[1:]); err == nil {
			fmt.Println("sending " + r.RequestURI)
			http.ServeFile(w, r, "./build/"+r.URL.Path[1:])
		} else {
			fmt.Println("req not found " + r.RequestURI)
			http.ServeFile(w, r, "./build/index.html")
		}
	})
	fmt.Println("Server is go")
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), router))
}

func twitchHandler(twitchchat chan twitch.PrivateMessage) {
	client := twitch.NewAnonymousClient()

	//defer client.Disconnect()

	client.OnPrivateMessage(func(msg twitch.PrivateMessage) {
		fmt.Printf("[%s] %s: %s\n", msg.Channel, msg.User.DisplayName, msg.Message)
		if msg.Bits > 0 {
			fmt.Printf("%s has given %d bit(s) to %s\n", msg.User.DisplayName, msg.Bits, msg.Channel)
		}
		twitchchat <- msg
	})

	client.OnClearMessage(func(msg twitch.ClearMessage) {
		fmt.Printf("[%s] delete %s\n", msg.Channel, msg.TargetMsgID)
		twitchchat <- twitch.PrivateMessage{
			ID:      msg.TargetMsgID,
			Time:    time.Unix(-1, -1),
			Channel: msg.Channel,
		}
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

	var err error
	db, err = sql.Open("sqlite3", "./shuffletron.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	initDb()

	go twitchHandler(twitchchat)
	go twitchTransmitter(twitchchat)
	handleReqs()
}
