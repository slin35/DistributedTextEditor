package main

import (
	"net/http"
	"net/rpc"
	"log"
	"net"
	"fmt"
)


type DocModifier int

type DocEdit struct {
	NewText string
	Index   int
}

type Doc struct {
	CurrentText string
	Site        int
}


func (d *DocModifier) EditDoc(docData DocEdit, reply *Doc) error {
	fmt.Println("TEST EDIT")
	userTextDir[docData.Index].CurrentText = docData.NewText
	*reply = *userTextDir[docData.Index]

	return nil
}


func createServer() {
	docMod := new(DocModifier)

	err := rpc.Register(docMod)
	if err != nil {
		log.Fatal("FAILED TO REGISTER DOC CONTROLLER ", err)
	}

	rpc.HandleHTTP()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal("TCP LISTENING ERROR ", err)
	}

	err = http.Serve(listener, nil)
	if err != nil {
		log.Fatal("ERROR SERVING ", err)
	}
}
