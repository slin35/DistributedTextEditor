package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"

	//	"time"
	socketio "github.com/googollee/go-socket.io"
)

// Text Body - full text currently in Quill editor
type TextBody struct {
	Ops []Operation
}

// Text History - a history of changes
type TextHistory struct {
	Ops []OperData
}

type Operation struct {
	Insert string
	Delete int
	Retain int
}

type OperData struct {
	Position int
	Type     string
	Data     string
}

type Page struct {
	Body string
}

// Storing array of < position identifier, character >
type Doc struct {
	Body []Character
}

type Identifier struct {
	pos  int
	site int
}

type Character struct {
	position []Identifier
	lamport  int
	char     string
}

var currentText string = ""
var currentID int = 0
var userTextDir = make(map[int]string)

var doc Doc

// can refactor away, currently working..
func loadPage() (*Page, error) {
	return &Page{Body: currentText}, nil
}

func initBody() {
	beg := Character{lamport: -1,
		char: ""}
	beg.position[0] = Identifier{pos: 0, site: -1}

	end := Character{lamport: -1,
		char: ""}
	// this has to be changed to math.MaxInt32 if run on a 32 bit system
	end.position[0] = Identifier{pos: int(math.MaxInt64), site: -1}

	doc.Body[0] = beg
	doc.Body[1] = end
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
			return idComp
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
	if i1.pos < i2.pos {
		return -1
	} else if i1.pos > i2.pos {
		return 1
	} else {
		if i1.site < i2.site {
			return -1
		} else if i1.site > i2.site {
			return 1
		} else {
			return 0
		}
	}
}

func fromIdentifierList(identifiers []Identifier) []int {
	returnArr := make([]int, len(identifiers))
	for _, ident := range identifiers {
		returnArr = append(returnArr, ident.pos)
	}
	return returnArr
}

// Arrays are representations of floats, subtract the floats
func subtractGreaterThan(n1 []int, n2 []int) []int {
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

func add(n1 []int, n2 []int) []int {
	var carry = 0
	diff := make([]int, maxArrayLength(n1, n2))
	for i := len(diff) - 1; i >= 0; i-- {
		var sum = eleExists(n1, i) + eleExists(n2, i) + carry
		carry = int(math.Floor(float64(sum) / 256))
		diff[i] = sum % 256
	}
	if carry != 0 {
		log.Fatal("Adding two positions results in a greater than 1 pos, this can't  be done.")
	}
	return diff
}

func increment(n1 []int, delta []int) []int {
	var firstNonzero = -1
	for i, num := range delta {
		if num > 0 {
			firstNonzero = i
		}
	}
	var inc = append(delta[0:firstNonzero], []int{0, 1}...)
	var v1 = add(n1, inc)
	var v2 []int
	if v1[len(v1)-1] == 0 {
		v2 = add(v1, inc)
	} else {
		v2 = v1
	}
	return v2
}

func toIdentifierList(n []int, before []Identifier, after []Identifier, site int) []Identifier {
	var returnArr []Identifier
	for i, num := range n {
		if i == len(n)-1 {
			returnArr[i] = Identifier{num, site}
		} else if i < len(before) && num == before[i].pos {
			returnArr[i] = Identifier{num, before[i].site}
		} else if i < len(after) && num == after[i].pos {
			returnArr[i] = Identifier{num, after[i].site}
		} else {
			returnArr[i] = Identifier{num, site}
		}
	}
	return returnArr
}

func generatePositionBetween(pos1 []Identifier, pos2 []Identifier, site int) []Identifier {
	var head1 Identifier
	var head2 Identifier
	if len(pos1) == 0 {
		head1 = Identifier{0, site}
	} else {
		head1 = pos1[0]
	}
	if len(pos2) == 0 {
		head2 = Identifier{int(^uint(0) >> 1), site} // max_int
	} else {
		head2 = pos2[0]
	}
	if head1.pos != head2.pos {
		var n1 = fromIdentifierList(pos1)
		var n2 = fromIdentifierList(pos2)
		var delta = subtractGreaterThan(n2, n1)

		var next = increment(n1, delta)
		return toIdentifierList(next, pos1, pos2, site)
	} else {
		if head1.site < head2.site {
			sliced := pos1[1:]
			recurPos := generatePositionBetween(sliced, []Identifier{}, site)
			return append([]Identifier{head1}, recurPos...)
		} else if head1.site == head2.site {
			sliced1 := pos1[1:]
			sliced2 := pos2[1:]
			recurPos := generatePositionBetween(sliced1, sliced2, site)
			return append([]Identifier{head1}, recurPos...)
		} else {
			log.Fatal("Cannot generate position at given site: ", site)
		}
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

func main() {
	/*	http.HandleFunc("/", homeHandler)
		/	http.HandleFunc("/editor", editorHandler)
		/	http.HandleFunc("/save", saveHandler)
		/	log.Fatal(http.ListenAndServe(":8080", nil)) */
	var curContent = ""
	/*	var curContent = map[string]interface{}{
		"ops": map[string]interface{}{},
	} */

	server := socketio.NewServer(nil)

	server.OnConnect("/", func(s socketio.Conn) error {
		fmt.Println("connected user with id ", s.ID())

		s.Emit("initContent", curContent)

		s.Join("bcast")
		return nil
	})

	server.OnDisconnect("/", func(s socketio.Conn, reason string) {
		fmt.Println("disconnected user with id", s.ID(), " because: ", reason)
	})

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
			server.BroadcastToRoom("", "bcast", "toAll", content)
			curContent = content
		}

	})

	server.OnEvent("/", "Delta", func(s socketio.Conn, delta string) {
		d := []byte(delta)
		var oneHistory TextBody
		if err := json.Unmarshal(d, &oneHistory); err != nil {
			panic(err)
		}
		fmt.Print("RECEIVED DELTA: ")
		fmt.Println(oneHistory)

		// var op Operation
		// for i := 0; i < len(data); i++ {
		// 	if ret, exists := data[i]["retain"]; exists {
		// 		op.Retain = ret.(int)
		// 	}
		// 	if del, exists := data[i]["delete"]; exists {
		// 		op.Delete = del.(int)
		// 	}
		// 	if ins, exists := data[i]["retain"]; exists {
		// 		op.Insert = ins.(string)
		// 	}

		// }

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
