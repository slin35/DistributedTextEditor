package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	socketio "github.com/googollee/go-socket.io"
)

// Storing array of < position identifier, character >
type Doc struct {
	Body []Character
}

type Identifier struct {
	Pos  int // position index
	Site int // client id
}

type Character struct {
	Position []Identifier
	Lamport  int
	Char     string
}

type OpItem struct {
	Ops []Item
}

type Item struct {
	Type      string
	Character string
	Position  []Identifier
}

var currentText string = ""
var currentID int = 0
var userTextDir = make(map[int]string)

// this has to be changed to math.MaxInt32 if run on a 32 bit system
var maxInt = math.MaxInt64

var doc Doc
var clientIds = make([]string, 0)

// Initialize the starter CRDT document body.
func initBody() {
	doc.Body = make([]Character, 1)
	beg := Character{Position: make([]Identifier, 1),
		Lamport: -1,
		Char:    ""}
	beg.Position[0] = Identifier{Pos: 1, Site: -1}
	doc.Body[0] = beg
	fmt.Println(doc.Body)
}

// get min of two ints.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Compare positions arrs.
func comparePosition(id1 []Identifier, id2 []Identifier) int {
	for i := 0; i < min(len(id1), len(id2)); i++ {
		idComp := compareIdentifier(id1[i], id2[i])
		if idComp != 0 {
			return idComp // 0 is eq, -1 is leq, 1 is geq
		}
	}

	if len(id1) < len(id2) {
		return -1
	} else if len(id1) > len(id2) {
		return 1
	} else {
		return 0
	}
}

// Compare identifier structs
func compareIdentifier(i1 Identifier, i2 Identifier) int {
	if i1.Pos < i2.Pos {
		return -1
	} else if i1.Pos > i2.Pos {
		return 1
	} else {
		if i1.Site < i2.Site {
			return -1
		} else if i1.Site > i2.Site {
			return 1
		} else {
			return 0
		}
	}
}

// Search for a specific Position arr (CRDT's UIDs) in our Doc body.
func searchPosition(prevPosition []Identifier) int {
	for i := 0; i < len(doc.Body); i++ {
		if comparePosition(doc.Body[i].Position, prevPosition) == 0 {
			return i
		}
	}
	return -1
}

func main() {
	var uidPos = 1
	initBody()

	server := socketio.NewServer(nil)

	// When a client connects to the server, give them current text, current CRDT
	//   and help them join necessary broadcasts.
	server.OnConnect("/", func(s socketio.Conn) error {
		fmt.Println("connected user with id ", s.ID())

		s.Emit("crdtTransfer", doc)
		s.Emit("initID", s.ID())
		clientIds = append(clientIds, []string{s.ID()}...)

		s.Join("bcast")
		s.Join(s.ID()) // Joins room with its own ID in order for server to send client specific messages
		return nil
	})

	// Disconnected clients get their ids removed.
	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("disconnected user with id", s.ID(), " because: ", reason)
		for i, v := range clientIds { // delete from clientIDs upon disconnect
			if v == s.ID() {
				clientIds = append(clientIds[:i], clientIds[i+1:]...)
				break
			}
		}
	})

	// When a Operation event is sent to Server from local clients
	server.OnEvent("/", "Operation", func(s socketio.Conn, opItem string) {
		// opItem - either a delete or insert or delete and insert
		// apply operation to server CRDT
		fmt.Println("received operation from :", s.ID())
		fmt.Println("Received Operation for: ", opItem)
		/*
			Received JSON:
			{
				"ops" : [
						{
							"Type" : "Insert",
							"Character" : "A",
							"Position" : []Position <- the previous position
						},
						{
							"Type" : "Delete",
							"Position" : []Position <- uid of deleted character
						}
					]
			}
		*/

		// broadcast to room -> use an arr of client IDs and send only to non-writer client.
		a := []byte(opItem)
		var anOpItem OpItem
		if err := json.Unmarshal(a, &anOpItem); err != nil {
			panic(err)
		}

		// Handle Insert operations first
		for i := 0; i < len(anOpItem.Ops); i++ {
			if anOpItem.Ops[i].Type == "Insert" {
				fmt.Println("Inserting")
				// Find the previous character's Position arr and Site.
				prevPosition := anOpItem.Ops[i].Position
				characterIdx := searchPosition(prevPosition)
				prevPos := prevPosition[len(prevPosition)-1].Pos
				prevSite := prevPosition[len(prevPosition)-1].Site
				tempID, _ := strconv.Atoi(s.ID())

				fmt.Println("found char indx:", characterIdx)
				var curPosition []Identifier
				// If the previous character is the user's, starting char, or another user's.
				if prevSite == tempID {
					fmt.Println("eq sites, prevUIDPos = ", uidPos)
					uidPos++
					prevPosition[len(prevPosition)-1].Pos = uidPos
					curPosition = prevPosition
				} else if prevSite == -1 {
					fmt.Println("prevsite is -1, prev UIDPos = ", uidPos)
					uidPos++
					curPosition = append(prevPosition, Identifier{Pos: uidPos, Site: tempID})
				} else {
					fmt.Println("prev site is not eq")
					uidPos = 1
					curPosition = append(prevPosition, Identifier{Pos: uidPos, Site: tempID})
				}
				fmt.Println("prevPos and Site: ", prevPos, prevSite)
				var newChar = Character{Position: curPosition,
					Lamport: -1,
					Char:    anOpItem.Ops[i].Character}
				fmt.Println("New character: ", newChar)

				// Inserting the new character into the document body (server crdt)
				var temp = make([]Character, len(doc.Body[characterIdx+1:]))
				copy(temp, doc.Body[characterIdx+1:])
				doc.Body = append(doc.Body[:characterIdx+1], []Character{newChar}...)
				doc.Body = append(doc.Body, temp...)
			}
		}

		// Handle delete operations second
		for i := 0; i < len(anOpItem.Ops); i++ {
			if anOpItem.Ops[i].Type == "Delete" {
				fmt.Println("Deleting")
				prevPosition := anOpItem.Ops[i].Position
				characterIdx := searchPosition(prevPosition)

				// remove from document body.
				doc.Body = append(doc.Body[:characterIdx], doc.Body[characterIdx+1:]...)
			}
		}
		out, err := json.Marshal(doc)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("After operation: ", string(out))

		// Broadcast new CRDT to all non-writer clients.
		for i := 0; i < len(clientIds); i++ {
			fmt.Println("Is ", clientIds[i], " equal to ", s.ID())
			if clientIds[i] != s.ID() {
				fmt.Println("Broadcasting to: ", clientIds[i])
				server.BroadcastToRoom("", clientIds[i], "crdtTransfer", string(out))
			}
		}

	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)

	http.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "index.html")
	})

	log.Println("serving at localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
