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
	"strings"
	"sync"
	"time"

	"github.com/Thor-x86/nullable"
	"github.com/gempir/go-twitch-irc/v2"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/imdario/mergo"
	_ "github.com/mattn/go-sqlite3"
)

type STConfig struct {
	Port     int      `json:"port"`
	Channels []string `json:"channels"`
}

const defaultPort = 42069
const defaultChannel = "kewliomzx"

var db *sql.DB

var wsListMutex = &sync.Mutex{}
var wsWriteMutex = &sync.Mutex{}
var dbAccessMutex = &sync.Mutex{}

// -------------============== TWITCHWS CLASS

type TwitchWS struct {
	msg  chan TwitchWSMsg
	conn *websocket.Conn
	id   int
	open bool
}

var openWS []TwitchWS
var wsId int

type TwitchWSMsg struct {
	MsgType     TwitchWSMsgType    `json:"msgType"`
	Id          string             `json:"id"`
	DisplayName string             `json:"displayName"`
	DisplayCol  string             `json:"displayCol"`
	Channel     string             `json:"channel"`
	Message     string             `json:"msg"`
	Time        int64              `json:"time"`
	Emotes      []TwitchWSMsgEmote `json:"emotes"`
}

type TwitchWSMsgType int

const (
	msgTypeUnknown TwitchWSMsgType = iota
	msgTypeMessage
	msgTypeAction
	msgTypeDelete
)

type TwitchWSMsgEmote struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

func (ws TwitchWS) wsWriter() {
	for {
		msg := <-ws.msg
		if !ws.open {
			fmt.Println("Stopping wsWriter")
			return
		}

		outMsg, err := json.Marshal(msg)
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
			ws.msg <- TwitchWSMsg{}
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

// -------------=========== UNIVERSAL API FUNCTIONS
func outputApiError(w http.ResponseWriter, errMsg string, errCode int) {
	w.WriteHeader(errCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"err": errMsg,
	})
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
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
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
		outputApiError(w, fmt.Sprintf("Invalid ID: %q", err), http.StatusBadRequest)
		return
	}

	stmt := `SELECT * FROM lists WHERE listId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
	} else {
		var list STList
		if err := row.Scan(&list.Id, &list.Name); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				outputApiError(w, fmt.Sprintf("List ID not found: %d", id), http.StatusNotFound)
			} else {
				outputApiError(w, fmt.Sprintf("Error during exec: %q", err), http.StatusInternalServerError)
			}
		} else {
			json.NewEncoder(w).Encode(list)
		}
	}
}

func createNewList(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: createNewList\n")

	if reqBody, err := ioutil.ReadAll(r.Body); err != nil {
		fmt.Printf("err: %v\n", err)
		outputApiError(w, fmt.Sprintf("Invalid data: %q", err), http.StatusBadRequest)
	} else {
		var list STList
		err := json.Unmarshal(reqBody, &list)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			outputApiError(w, fmt.Sprintf("Could not parse JSON: %q", err), http.StatusBadRequest)
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
			outputApiError(w, fmt.Sprintf("Error preparing query: %q", err), http.StatusInternalServerError)
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
		outputApiError(w, fmt.Sprintf("Invalid ID: %q", err), http.StatusBadRequest)
		return
	}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		outputApiError(w, fmt.Sprintf("Invalid data: %q", err), http.StatusBadRequest)
		return
	}

	stmt := `SELECT * FROM lists WHERE listId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
	} else {
		var listRetrieve STList
		if err := row.Scan(&listRetrieve.Id, &listRetrieve.Name); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				outputApiError(w, fmt.Sprintf("List ID not found: %d", id), http.StatusNotFound)
			} else {
				outputApiError(w, fmt.Sprintf("Error during exec: %q", err), http.StatusInternalServerError)
			}
		} else {
			var listUpdate STList
			err := json.Unmarshal(reqBody, &listUpdate)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				outputApiError(w, fmt.Sprintf("Could not parse JSON: %q", err), http.StatusBadRequest)
				return
			}

			listUpdate.Id = listRetrieve.Id
			err = mergo.Merge(&listRetrieve, listUpdate, mergo.WithOverride)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				outputApiError(w, fmt.Sprintf("Error merging data: %q", err), http.StatusInternalServerError)
				return
			}

			stmt := `
				UPDATE lists
				SET listName = ?
				WHERE listId = ?
			`

			if result, err := db.Exec(stmt, listRetrieve.Name, listRetrieve.Id); err != nil {
				fmt.Printf("err: %v\n", err)
				outputApiError(w, fmt.Sprintf("Error preparing query: %q", err), http.StatusInternalServerError)
			} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
				outputApiError(w, fmt.Sprintf("List ID not found: %d", id), http.StatusNotFound)
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
		outputApiError(w, fmt.Sprintf("Invalid ID: %q", err), http.StatusBadRequest)
		return
	}

	stmt := `
		DELETE FROM lists
		WHERE listId = ?
	`

	if result, err := db.Exec(stmt, id); err != nil {
		fmt.Printf("%q: during query %s\n", err, stmt)
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
		return
	} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
		outputApiError(w, fmt.Sprintf("List ID not found: %d", id), http.StatusNotFound)
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
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
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
		outputApiError(w, fmt.Sprintf("Invalid ID: %q", err), http.StatusBadRequest)
		return
	}

	stmt := `SELECT * FROM games WHERE gameId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
	} else {
		var game STGame
		var activeDisplayName string
		if err := row.Scan(&game.Id, &game.ListId, &game.Name, &game.DisplayName, &game.Description,
			&game.Weight, &game.Status, &activeDisplayName); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				outputApiError(w, fmt.Sprintf("Game ID not found: %d", id), http.StatusNotFound)
			} else {
				outputApiError(w, fmt.Sprintf("Error during exec: %q", err), http.StatusInternalServerError)
			}
		} else {
			json.NewEncoder(w).Encode(game)
		}
	}
}

func createNewGame(w http.ResponseWriter, r *http.Request) {
	fmt.Printf("Endpoint hit: createNewGame\n")

	if reqBody, err := ioutil.ReadAll(r.Body); err != nil {
		fmt.Printf("err: %v\n", err)
		outputApiError(w, fmt.Sprintf("Invalid data: %q", err), http.StatusBadRequest)
	} else {
		var game STGame
		err := json.Unmarshal(reqBody, &game)
		if err != nil {
			fmt.Printf("err: %v\n", err)
			outputApiError(w, fmt.Sprintf("Could not parse JSON: %q", err), http.StatusBadRequest)
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
			outputApiError(w, fmt.Sprintf("Error preparing query: %q", err), http.StatusInternalServerError)
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
		outputApiError(w, fmt.Sprintf("Invalid ID: %q", err), http.StatusBadRequest)
		return
	}
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("err: %v\n", err)
		outputApiError(w, fmt.Sprintf("Invalid data: %q", err), http.StatusBadRequest)
		return
	}

	stmt := `SELECT * FROM games WHERE gameId = ?`

	dbAccessMutex.Lock()
	defer dbAccessMutex.Unlock()

	if row := db.QueryRow(stmt, id); row.Err() != nil {
		err := row.Err()
		fmt.Printf("%q: during query %s\n", err, stmt)
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
	} else {
		var gameRetrieve STGame
		var activeDisplayName string
		if err := row.Scan(&gameRetrieve.Id, &gameRetrieve.ListId, &gameRetrieve.Name,
			&gameRetrieve.DisplayName, &gameRetrieve.Description, &gameRetrieve.Weight,
			&gameRetrieve.Weight, &activeDisplayName); err != nil {
			fmt.Printf("%q: during exec %s\n", err, stmt)
			if err == sql.ErrNoRows {
				outputApiError(w, fmt.Sprintf("Game ID not found: %d", id), http.StatusNotFound)
			} else {
				outputApiError(w, fmt.Sprintf("Error during exec: %q", err), http.StatusInternalServerError)
			}
		} else {
			var gameUpdate STGame
			err := json.Unmarshal(reqBody, &gameUpdate)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				outputApiError(w, fmt.Sprintf("Could not parse JSON: %q", err), http.StatusBadRequest)
				return
			}

			gameUpdate.Id = gameRetrieve.Id
			err = mergo.Merge(&gameRetrieve, gameUpdate, mergo.WithOverride)
			if err != nil {
				fmt.Printf("err: %v\n", err)
				outputApiError(w, fmt.Sprintf("Error merging data: %q", err), http.StatusInternalServerError)
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
				outputApiError(w, fmt.Sprintf("Error preparing query: %q", err), http.StatusInternalServerError)
			} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
				outputApiError(w, fmt.Sprintf("Game ID not found: %d", id), http.StatusNotFound)
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
		outputApiError(w, fmt.Sprintf("Invalid ID: %q", err), http.StatusBadRequest)
		return
	}

	stmt := `
		DELETE FROM games
		WHERE gameId = ?
	`

	if result, err := db.Exec(stmt, id); err != nil {
		fmt.Printf("%q: during query %s\n", err, stmt)
		outputApiError(w, fmt.Sprintf("Error during query: %q", err), http.StatusInternalServerError)
		return
	} else if rowsAff, _ := result.RowsAffected(); rowsAff == 0 {
		outputApiError(w, fmt.Sprintf("Game ID not found: %d", id), http.StatusNotFound)
		return
	} else {
		w.WriteHeader(http.StatusNoContent)
	}
}

// -------------=========== MAIN CODE

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

	CREATE TABLE IF NOT EXISTS games (
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

func handleReqs(port int) {
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
		} else if strings.HasSuffix(r.RequestURI, "/stconfig.json") {
			fmt.Println("Client requested config")
			http.ServeFile(w, r, "./stconfig.json")
		} else {
			fmt.Println("req not found " + r.RequestURI + " - serving index instead")
			http.ServeFile(w, r, "./build/index.html")
		}
	})
	fmt.Println("Server is go on port " + strconv.Itoa(port))
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(port), router))
}

func twitchHandler(twitchchat chan TwitchWSMsg, channels []string) {
	client := twitch.NewAnonymousClient()

	//defer client.Disconnect()

	client.OnPrivateMessage(func(msg twitch.PrivateMessage) {
		fmt.Printf("[%s] %s: %s\n", msg.Channel, msg.User.DisplayName, msg.Message)
		if msg.Bits > 0 {
			fmt.Printf("%s has given %d bit(s) to %s\n", msg.User.DisplayName, msg.Bits, msg.Channel)
		}

		var outEmotes []TwitchWSMsgEmote
		for _, inEmote := range msg.Emotes {
			outEmotes = append(outEmotes, TwitchWSMsgEmote{
				Name: inEmote.Name,
				Id:   inEmote.ID,
			})
		}

		twitchchat <- TwitchWSMsg{
			MsgType:     msgTypeMessage,
			Id:          msg.ID,
			DisplayName: msg.User.DisplayName,
			DisplayCol:  msg.User.Color,
			Channel:     msg.Channel,
			Time:        msg.Time.Unix(),
			Message:     msg.Message,
			Emotes:      outEmotes,
		}
	})

	client.OnClearMessage(func(msg twitch.ClearMessage) {
		fmt.Printf("[%s] delete %s\n", msg.Channel, msg.TargetMsgID)
		twitchchat <- TwitchWSMsg{
			MsgType: msgTypeMessage,
			Id:      msg.TargetMsgID,
			Channel: msg.Channel,
			Message: msg.Message,
		}
	})

	client.Join(channels...)

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

func twitchTransmitter(msg chan TwitchWSMsg) {
	for {
		msgIn := <-msg
		sentTo := 0
		wsListMutex.Lock()
		for _, ws := range openWS {
			if !ws.open {
				continue
			}
			if ws.msg == nil {
				ws.msg = make(chan TwitchWSMsg)
				go ws.wsWriter()
			}
			ws.msg <- msgIn
			sentTo++
		}
		fmt.Printf("Sent to %d/%d client(s)\n", sentTo, len(openWS))
		wsListMutex.Unlock()
	}
}

func readConfig() STConfig {
	defaultConfig := STConfig{
		Port:     defaultPort,
		Channels: []string{defaultChannel},
	}

	file, err := os.ReadFile("./stconfig.json")
	if err != nil {
		fmt.Println("Failed to read stconfig.json")
		return defaultConfig
	}

	var config STConfig
	err = json.Unmarshal(file, &config)
	if err != nil {
		fmt.Println("Failed to parse stconfig.json")
		return defaultConfig
	}

	if config.Port == 0 {
		config.Port = defaultPort
	}

	if len(config.Channels) == 0 {
		config.Channels = []string{defaultChannel}
	}

	return config
}

func main() {
	fmt.Println("Starting server")
	config := readConfig()

	twitchchat := make(chan TwitchWSMsg)

	var err error
	db, err = sql.Open("sqlite3", "./shuffletron.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	initDb()

	go twitchHandler(twitchchat, config.Channels)
	go twitchTransmitter(twitchchat)
	handleReqs(config.Port)
}
