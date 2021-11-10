package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func homepage(w http.ResponseWriter, r *http.Request) {
	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	var member Member
	err = json.Unmarshal(reqBody, &member)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		log.Print("Err Unmarshal")
		return
	}

	//insert member to database

	//respnse 200 to writer

}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("v1/auth/register", homepage).Methods("POST")

	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

func main() {
	handleRequests()
	fmt.Println("hello")
}

type Member struct {
	Username string
	Name     string
	Lastname string
	Mobile   string
	Email    string
	Password string
}
