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
	myRouter := mux.NewRouter().StrictSlash(true)
	fmt.Println("test0")
	myRouter.HandleFunc("/register", register).Methods("POST")

	fmt.Println("test1")
	log.Fatal(http.ListenAndServe(":4200", myRouter))
}

func register(w http.ResponseWriter, r *http.Request) {
	fmt.Println("test3")
	reqBody, _ := ioutil.ReadAll(r.Body)
	var member Member
	json.Unmarshal(reqBody, &member)
	fmt.Println("test4")
}

func handleRequests() {
	http.HandleFunc("v1/auth", homepage)
	log.Fatal(http.ListenAndServe(":4200", nil))
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
