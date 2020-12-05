package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"math"
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

func comparePosition(id1 []Identifier, id2 []Identifier) int {
	for i := 0; i < min(len(id1), len(id2)); i++ {
		idComp := compareIdentifier(id1[i], id2[i])
		if (idComp != 0) {
			return idComp
		}
	}

	if (len(id1) < len(id2)) {
		return -1
	} else if (len(id1) > len(id2)) {
		return 1
	} else {
		return 0
	}
}

func compareIdentifier(i1 Identifier, i2 Identifier) int {
	if (i1.pos < i2.pos) {
		return -1
	} else if (i1.pos > i2.pos) {
		return 1
	} else {
		if (i1.site < i2.site) {
			return -1
		} else if (i1.site > i2.site) {
			return 1
		} else {
			return 0
		}
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
/*	_, err := r.Cookie("uid")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:    "uid",
			Value:   strconv.Itoa(currentID),
			Expires: time.Now().Add(999999 * time.Second),
		})
		userTextDir[currentID] = ""
		currentID++
		fmt.Println("Setting current client uid to : ", currentID)
	}
	http.Redirect(w, r, "/editor", http.StatusFound)
*/
	p, err := loadPage()
	if err != nil {
		fmt.Println("Error loading page")
	}
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, p)
}

func editorHandler(w http.ResponseWriter, r *http.Request) {
	// title := r.URL.Path[len("/edit/"):]

	p, err := loadPage()
	if err != nil {
		fmt.Println("Error loading page")
	}
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, p)
}

// Handler to deal with "/save" endpoint which is when save button is clicked
func saveHandler(w http.ResponseWriter, r *http.Request) {
	var currentBody TextBody
	var currentHistory TextHistory

	cookie, err := r.Cookie("uid")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(cookie)
	currUID, uidErr := strconv.Atoi(cookie.Value)
	if uidErr != nil {
		log.Fatal(uidErr)
	} 

	// get form value for body and history
	body := r.FormValue("body")
	history := r.FormValue("history")
	// --- string print of incoming json
	// fmt.Println("Body: ", body)
	// fmt.Println("History: ", history)

	// unmarshall into TextBody or TextHistory
	json.Unmarshal([]byte(body), &currentBody)
	json.Unmarshal([]byte(history), &currentHistory)

	// --- sample print of unmarshelled json
	for _, ele := range currentBody.Ops {
		fmt.Println(ele)
	}
	if len(currentBody.Ops) != 0 {
		fmt.Printf("Insert: %s", currentBody.Ops[0].Insert)
		for i, s := range currentHistory.Ops {
			fmt.Println(i, s)
		}

		// --- workaround for reset page after form submit
		currentText = currentBody.Ops[0].Insert
		userTextDir[currUID] = currentText
	} else {
		currentText = userTextDir[currUID]
	}

	// redirect back to edit page
	http.Redirect(w, r, "/editor", http.StatusFound)
}


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
		fmt.Println("disconnected user with id", s.ID())
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
			fmt.Println(s.ID())
			fmt.Println(content)
			server.BroadcastToRoom("", "bcast", "toAll", content)
			curContent = content
		}

		
	})

	server.OnEvent("/", "Delta", func(s socketio.Conn, delta string) {
		d := []byte(delta)
		var dlta map[string][]map[string]interface{}
		var data []map[string]interface{}
		if err := json.Unmarshal(d, &dlta); err != nil {
			panic(err)
		}
		fmt.Print("DATA: ")
		fmt.Println(data)
		data = dlta["ops"]

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
