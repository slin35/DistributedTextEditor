package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	//	"time"
	socketio "github.com/googollee/go-socket.io"
)

// Text Body - full text currently in Quill editor
type TextBody struct {
	Ops []Operation
}

type Operation struct {
	Insert string
	Delete int
	Retain int
}

type Page struct {
	Body string
}

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

// can refactor away, currently working..
func loadPage() (*Page, error) {
	return &Page{Body: currentText}, nil
}

func initBody() {
	doc.Body = make([]Character, 1)
	beg := Character{Position: make([]Identifier, 1),
		Lamport: -1,
		Char:    ""}
	beg.Position[0] = Identifier{Pos: 1, Site: -1}

	// end := Character{Position: make([]Identifier, 1),
	// 	Lamport: -1,
	// 	Char:    ""}
	// // this has to be changed to math.MaxInt32 if run on a 32 bit system
	// end.Position[0] = Identifier{Pos: int(maxInt), Site: -1}

	doc.Body[0] = beg
	// doc.Body[1] = end // not sure if we need an end.
	fmt.Println(doc.Body)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxArrayLength(n1 []int, n2 []int) int {
	if len(n1) < len(n2) {
		return len(n2)
	}
	return len(n1)
}

func eleExists(n1 []int, index int) int {
	if len(n1) > index {
		return 0
	}
	return n1[index]
}

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

func identifierToList(identifiers []Identifier) []int {
	returnArr := make([]int, len(identifiers))
	for _, ident := range identifiers {
		returnArr = append(returnArr, ident.Pos)
	}
	return returnArr
}

// Arrays are representations of floats, subtract the floats
func floatSubtract(n1 []int, n2 []int) []int {
	var carry = 0 // carry over in subtraction
	diff := make([]int, maxArrayLength(n1, n2))
	for i := len(diff) - 1; i >= 0; i-- {
		var d1 int
		var d2 int
		d1 = eleExists(n1, i) - carry
		d2 = eleExists(n2, i)
		if d1 < d2 {
			carry = 1
			diff[i] = d1 + 256 - d2
		} else {
			carry = 0
			diff[i] = d1 - d2
		}
	}
	return diff
}

func floatAdd(n1 []int, n2 []int) []int {
	var carry = 0
	diff := make([]int, maxArrayLength(n1, n2))
	for i := len(diff) - 1; i >= 0; i-- {
		var sum = eleExists(n1, i) + eleExists(n2, i) + carry
		carry = int(math.Floor(float64(sum) / 256))
		diff[i] = sum % 256
	}
	if carry != 0 {
		log.Fatal("Adding two positions results in a greater than 1 pos, this can't be done.")
	}
	return diff
}

func floatIncrement(n1 []int, delta []int) []int {
	var firstNonzero = -1
	for i, num := range delta {
		if num > 0 {
			firstNonzero = i
		}
	}
	var inc = append(delta[0:firstNonzero], []int{0, 1}...)
	var v1 = floatAdd(n1, inc)
	var v2 []int
	if v1[len(v1)-1] == 0 {
		v2 = floatAdd(v1, inc)
	} else {
		v2 = v1
	}
	return v2
}

func listToIdentifier(n []int, before []Identifier, after []Identifier, site int) []Identifier {
	var returnArr []Identifier
	for i, num := range n {
		if i == len(n)-1 {
			returnArr[i] = Identifier{num, site}
		} else if i < len(before) && num == before[i].Pos {
			returnArr[i] = Identifier{num, before[i].Site}
		} else if i < len(after) && num == after[i].Pos {
			returnArr[i] = Identifier{num, after[i].Site}
		} else {
			returnArr[i] = Identifier{num, site}
		}
	}
	return returnArr
}

func genPos(pos1 []Identifier, pos2 []Identifier, site int) []Identifier {
	var head1 Identifier
	var head2 Identifier
	if len(pos1) == 0 {
		head1 = Identifier{0, site}
	} else {
		head1 = pos1[0]
	}
	if len(pos2) == 0 {
		head2 = Identifier{maxInt, site}
	} else {
		head2 = pos2[0]
	}
	if head1.Pos != head2.Pos {
		var n1 = identifierToList(pos1)
		var n2 = identifierToList(pos2)
		var delta = floatSubtract(n2, n1)

		var next = floatIncrement(n1, delta)
		return listToIdentifier(next, pos1, pos2, site)
	}
	if head1.Site < head2.Site {
		sliced := pos1[1:]
		recurPos := genPos(sliced, []Identifier{}, site)
		return append([]Identifier{head1}, recurPos...)
	} else if head1.Site == head2.Site {
		sliced1 := pos1[1:]
		sliced2 := pos2[1:]
		recurPos := genPos(sliced1, sliced2, site)
		return append([]Identifier{head1}, recurPos...)
	} else {
		log.Fatal("Cannot generate position at given site : ", site)
	}

	return nil
}

// func homeHandler(w http.ResponseWriter, r *http.Request) {
// 	/*	_, err := r.Cookie("uid")
// 		if err != nil {
// 			http.SetCookie(w, &http.Cookie{
// 				Name:    "uid",
// 				Value:   strconv.Itoa(currentID),
// 				Expires: time.Now().Add(999999 * time.Second),
// 			})
// 			userTextDir[currentID] = ""
// 			currentID++
// 			fmt.Println("Setting current client uid to : ", currentID)
// 		}
// 		http.Redirect(w, r, "/editor", http.StatusFound)
// 	*/
// 	p, err := loadPage()
// 	if err != nil {
// 		fmt.Println("Error loading page")
// 	}
// 	t, _ := template.ParseFiles("index.html")
// 	t.Execute(w, p)
// }

// func editorHandler(w http.ResponseWriter, r *http.Request) {
// 	// title := r.URL.Path[len("/edit/"):]

// 	p, err := loadPage()
// 	if err != nil {
// 		fmt.Println("Error loading page")
// 	}
// 	t, _ := template.ParseFiles("index.html")
// 	t.Execute(w, p)
// }

// // Handler to deal with "/save" endpoint which is when save button is clicked
// func saveHandler(w http.ResponseWriter, r *http.Request) {
// 	var currentBody TextBody
// 	var currentHistory TextHistory

// 	cookie, err := r.Cookie("uid")
// 	if err != nil {
// 		log.Fatal(err)
// 	}
// 	fmt.Println(cookie)
// 	currUID, uidErr := strconv.Atoi(cookie.Value)
// 	if uidErr != nil {
// 		log.Fatal(uidErr)
// 	}

// 	// get form value for body and history
// 	body := r.FormValue("body")
// 	history := r.FormValue("history")
// 	// --- string print of incoming json
// 	// fmt.Println("Body: ", body)
// 	// fmt.Println("History: ", history)

// 	// unmarshall into TextBody or TextHistory
// 	json.Unmarshal([]byte(body), &currentBody)
// 	json.Unmarshal([]byte(history), &currentHistory)

// 	// --- sample print of unmarshelled json
// 	for _, ele := range currentBody.Ops {
// 		fmt.Println(ele)
// 	}
// 	if len(currentBody.Ops) != 0 {
// 		fmt.Printf("Insert: %s", currentBody.Ops[0].Insert)
// 		for i, s := range currentHistory.Ops {
// 			fmt.Println(i, s)
// 		}

// 		// --- workaround for reset page after form submit
// 		currentText = currentBody.Ops[0].Insert
// 		userTextDir[currUID] = currentText
// 	} else {
// 		currentText = userTextDir[currUID]
// 	}

// 	// redirect back to edit page
// 	http.Redirect(w, r, "/editor", http.StatusFound)
// }

func searchPosition(prevPosition []Identifier) int {
	for i := 0; i < len(doc.Body); i++ {
		if comparePosition(doc.Body[i].Position, prevPosition) == 0 {
			return i
		}
	}
	return -1
}

func main() {
	/*	http.HandleFunc("/", homeHandler)
		/	http.HandleFunc("/editor", editorHandler)
		/	http.HandleFunc("/save", saveHandler)
		/	log.Fatal(http.ListenAndServe(":8080", nil)) */
	var curContent = ""
	var uidPos = 1
	/*	var curContent = map[string]interface{}{
		"ops": map[string]interface{}{},
	} */
	initBody()

	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		fmt.Println("connected user with id ", s.ID())

		s.Emit("initContent", curContent)
		s.Emit("crdtTransfer", doc)
		s.Emit("initID", s.ID())
		clientIds = append(clientIds, []string{s.ID()}...)

		s.Join("bcast")
		s.Join(s.ID()) // Joins room with its own ID in order for server to send client specific messages
		return nil
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("disconnected user with id", s.ID(), " because: ", reason)
		for i, v := range clientIds { // delete from clientIDs upon disconnect
			if v == s.ID() {
				clientIds = append(clientIds[:i], clientIds[i+1:]...)
				break
			}
		}
	})

	// Server receives a new string
	server.OnEvent("/", "Content", func(s socketio.Conn, content string) {
		//	curContent += "/" + content

		/*		in := []byte(content)
				var raw map[string]interface{}
				if err := json.Unmarshal(in, &raw); err != nil {
					panic(err)
				}

				out, _ := json.Marshal(raw)

				fmt.Println(string(out))

		*/
		/*	fmt.Println("getting content ", curContent, "from user with id ", s.ID())
			s.Emit("fromServer", curContent); */
		if curContent == "" {
			curContent = content
		}
		if curContent != content {
			fmt.Println("Receiving content from : ", s.ID())
			fmt.Println("curContent: ", curContent)
			fmt.Println("content: ", content)
			server.BroadcastToRoom("", s.ID(), "dirMsg", "HI")
			server.BroadcastToRoom("", "bcast", "toAll", content)
			curContent = content
		}

	})

	// TODO:
	// When a Operation event is sent
	server.OnEvent("/", "Operation", func(s socketio.Conn, opItem string) {
		// opItem - either a delete or insert or delete and insert
		// apply operation to server CRDT
		fmt.Println("Received Operation for: ", opItem)
		/*
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
		// there is a data structure, clientIds which is an array of IDs, just do
		// server.BroadcastToRoom("", clientIds[i], "dirMsg", "HI")
		a := []byte(opItem)
		var anOpItem OpItem
		if err := json.Unmarshal(a, &anOpItem); err != nil {
			panic(err)
		}

		for i := 0; i < len(anOpItem.Ops); i++ {
			if anOpItem.Ops[i].Type == "Insert" {
				fmt.Println("Inserting")
				prevPosition := anOpItem.Ops[i].Position
				characterIdx := searchPosition(prevPosition)
				prevPos := prevPosition[len(prevPosition)-1].Pos
				prevSite := prevPosition[len(prevPosition)-1].Site
				tempID, _ := strconv.Atoi(s.ID())
				/*
					if(prevPosition[prevPosition.length -1].Site = myID) { // last character was mine
						uidPos+=1
						prevPosition[prevPosition.length -1].Pos = uidPos
					} else {
						uidPos = 1
						prevPosition.push({  // insert @ (retain+1) index (+1 for "" in beginning)
						...{"Pos" : uidPos},
						...{"Site" : myID}
						})
					}
				*/
				fmt.Println("found char indx:", characterIdx)
				var curPosition []Identifier
				if prevSite == tempID {
					uidPos++
					prevPosition[len(prevPosition)-1].Pos = uidPos
					curPosition = prevPosition
				} else if prevSite == -1 {
					uidPos++
					curPosition = append(prevPosition, Identifier{Pos: uidPos, Site: tempID})
				} else {
					uidPos = 1
					curPosition = append(prevPosition, Identifier{Pos: uidPos, Site: tempID})
				}
				// curPosition := append(prevPosition, Identifier{Pos: prevPos + 1, Site: tempID})
				fmt.Println("prevPos and Site: ", prevPos, prevSite)
				var newChar = Character{Position: curPosition,
					Lamport: -1,
					Char:    anOpItem.Ops[i].Character}
				fmt.Println("New character: ", newChar)
				/*
					arr1 = append(arr1, 0) // Making space for the new element
					copy(arr1[3:], arr1[2:]) // Shifting elements
					arr1[2] = 99 // Copying/inserting the value
				*/
				var temp = make([]Character, len(doc.Body[characterIdx+1:]))
				copy(temp, doc.Body[characterIdx+1:])
				doc.Body = append(doc.Body[:characterIdx+1], []Character{newChar}...)
				doc.Body = append(doc.Body, temp...)
			}
		}

		for i := 0; i < len(anOpItem.Ops); i++ {
			if anOpItem.Ops[i].Type == "Delete" {
				fmt.Println("Deleting")
				prevPosition := anOpItem.Ops[i].Position
				characterIdx := searchPosition(prevPosition)

				doc.Body = append(doc.Body[:characterIdx], doc.Body[characterIdx+1:]...)
			}
		}
		out, err := json.Marshal(doc)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("After operation: ", string(out))

		for i := 0; i < len(clientIds); i++ {
			fmt.Println("Is ", clientIds[i], " equal to ", s.ID())
			if clientIds[i] != s.ID() {
				fmt.Println("Broadcasting to: ", clientIds[i])
				server.BroadcastToRoom("", clientIds[i], "crdtTransfer", string(out))
			}
		}

	})

	// Server receives a new sequence action (insert, delete, or insert and delete)
	server.OnEvent("/", "Delta", func(s socketio.Conn, delta string) {
		d := []byte(delta)
		var oneHistory TextBody
		if err := json.Unmarshal(d, &oneHistory); err != nil {
			panic(err)
		}

		// TODO: UPDATE BODY TEXT HERE (CRDT):
		fmt.Println("RECEIVED DELTA FROM ", s.ID(), " : ")
		for _, ele := range oneHistory.Ops {
			fmt.Println("Insert: ", ele.Insert)
			fmt.Println("Delete: ", ele.Delete)
			fmt.Println("Retain: ", ele.Retain)
			fmt.Println()
		}
		// fmt.Println(oneHistory)

		// fmt.Println(op)

	})

	go server.Serve()
	defer server.Close()

	http.Handle("/socket.io/", server)

	//	http.HandleFunc("/", homeHandler)
	//	http.HandleFunc("/editor", editorHandler)
	//	http.HandleFunc("/save", saveHandler)
	http.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		http.ServeFile(response, request, "index.html")
	})

	log.Println("serving at localhost:8080")
	log.Fatal(http.ListenAndServe(":8080", nil))

}
