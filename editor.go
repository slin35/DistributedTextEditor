package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/rpc"
	"strconv"
	"time"
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
	Delete string
	Retain string
}

type OperData struct {
	Position int
	Type     string
	Data     string
}

type Page struct {
	Body string
}

var currentText string = ""
var currentID int = 0
var userTextDir = make(map[int]*Doc)













// can refactor away, currently working..
func loadPage() (*Page, error) {
	return &Page{Body: currentText}, nil
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	_, err := r.Cookie("uid")
	if err != nil {
		http.SetCookie(w, &http.Cookie{
			Name:    "uid",
			Value:   strconv.Itoa(currentID),
			Expires: time.Now().Add(999999 * time.Second),
		})
		// userTextDir[currentID].currentText = ""
		
		doc := Doc{"", currentID}
		userTextDir[doc.Site] = &doc

		currentID++
		fmt.Println("Setting current client uid to : ", currentID)
	}
	http.Redirect(w, r, "/editor", http.StatusFound)
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



	client, err := rpc.DialHTTP("tcp", "localhost:8080")
	if err != nil {
		log.Fatal("CONNECTION ERROR ", err)
	}



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

	fmt.Println("TEST")
	if len(currentBody.Ops) != 0 {
		fmt.Printf("Insert: %s", currentBody.Ops[0].Insert)
		for i, s := range currentHistory.Ops {
			fmt.Println(i, s)
		}

		// --- workaround for reset page after form submit
		currentText = currentBody.Ops[0].Insert
		// userTextDir[currUID].currentText = currentText

		var reply Doc
		newData := DocEdit{currentText, currUID}
		
		err := client.Call("DocModifier.EditDoc", newData, &reply)
		if err != nil {
			log.Fatal("ISSUE CHANGING DOC ", err)
		}
		fmt.Println("FINISHED WITH CLIENT CALL")
		fmt.Println(reply.CurrentText)


	} else {
		currentText = userTextDir[currUID].CurrentText
	}

	// redirect back to edit page
	http.Redirect(w, r, "/editor", http.StatusFound)
}

func main() {
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/editor", editorHandler)
	http.HandleFunc("/save", saveHandler)

	createServer()


	// log.Fatal(http.ListenAndServe(":8080", nil))
}
