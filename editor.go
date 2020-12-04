package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
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
var userTextDir = make(map[int]string)

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
		userTextDir[currentID] = ""
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
	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/editor", editorHandler)
	http.HandleFunc("/save", saveHandler)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
